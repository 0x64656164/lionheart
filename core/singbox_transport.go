package core

import (
	"context"
	"fmt"
	"net"
	"time"

	// FIXED: all three packages were referenced in the file but not imported.
	// Adding explicit imports resolves: "undefined: adapter", "undefined: log",
	// "undefined: option" errors on lines 362, 389, 396, 402.
	"github.com/sagernet/sing-box/adapter"
	sbLog "github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"

	"github.com/hashicorp/yamux"
	"github.com/xtaci/kcp-go/v5"
)

// KCPTransport implements a sing-box compatible network transport over KCP/Yamux
// tunnelled through TURN.  It is used when sing-box is configured as the
// protocol layer but we still want to route traffic through the Lionheart
// TURN relay infrastructure.
type KCPTransport struct {
	peer   string
	pw     string
	cache  *CredsCache
	sess   *Session
	ctx    context.Context
	cancel context.CancelFunc
	logger sbLog.Logger // FIXED: import sbLog "github.com/sagernet/sing-box/log"
}

// NewKCPTransport creates a KCPTransport that connects to peer using pw as
// the session password.  It does NOT start the tunnel; call Connect() first.
func NewKCPTransport(ctx context.Context, peer, pw string, logger sbLog.Logger) *KCPTransport {
	child, cancel := context.WithCancel(ctx)
	return &KCPTransport{
		peer:   peer,
		pw:     pw,
		cache:  &CredsCache{},
		sess:   &Session{},
		ctx:    child,
		cancel: cancel,
		logger: logger,
	}
}

// Connect establishes (or re-establishes) the KCP/Yamux tunnel over TURN.
func (t *KCPTransport) Connect() error {
	ym, cl, err := Establish(t.cache, t.peer, t.pw, true)
	if err != nil {
		return fmt.Errorf("KCPTransport.Connect: %w", err)
	}
	t.sess.Set(ym, cl)
	return nil
}

// Close tears down the transport.
func (t *KCPTransport) Close() error {
	t.cancel()
	t.sess.Stop()
	return nil
}

// DialContext opens a new multiplexed stream through the existing tunnel.
// It satisfies the net.Conn-returning dialer contract expected by sing-box
// outbound implementations.
func (t *KCPTransport) DialContext(ctx context.Context, _, addr string) (net.Conn, error) {
	ym, ok := t.sess.Get()
	if !ok || ym == nil {
		if err := t.Connect(); err != nil {
			return nil, err
		}
		ym, ok = t.sess.Get()
		if !ok {
			return nil, fmt.Errorf("KCPTransport: tunnel not ready")
		}
	}

	stream, err := ym.OpenStream()
	if err != nil {
		t.sess.Down()
		return nil, fmt.Errorf("KCPTransport.DialContext: yamux open: %w", err)
	}

	return &yamuxConn{Stream: stream, remoteAddr: addr}, nil
}

// ---------------------------------------------------------------------------
// yamuxConn wraps a yamux.Stream to implement net.Conn
// ---------------------------------------------------------------------------

type yamuxConn struct {
	*yamux.Stream
	remoteAddr string
}

func (c *yamuxConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
}

func (c *yamuxConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
}

// ---------------------------------------------------------------------------
// DirectKCPDialer dials the VPN peer directly over KCP (no TURN relay).
// Used for environments where TURN is unavailable.
// ---------------------------------------------------------------------------

type DirectKCPDialer struct {
	peer string
	pw   string
}

func NewDirectKCPDialer(peer, pw string) *DirectKCPDialer {
	return &DirectKCPDialer{peer: peer, pw: pw}
}

func (d *DirectKCPDialer) DialContext(ctx context.Context, _, _ string) (net.Conn, error) {
	blk, err := kcp.NewAESBlockCrypt(DeriveKey(d.pw))
	if err != nil {
		return nil, fmt.Errorf("DirectKCPDialer: crypt: %w", err)
	}

	conn, err := kcp.DialWithOptions(d.peer, blk, 10, 3)
	if err != nil {
		return nil, fmt.Errorf("DirectKCPDialer: kcp dial %s: %w", d.peer, err)
	}
	conn.SetNoDelay(1, 10, 2, 1)
	conn.SetWindowSize(1024, 1024)
	conn.SetStreamMode(true)

	ym, err := yamux.Client(conn, YmxCfg())
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("DirectKCPDialer: yamux: %w", err)
	}

	stream, err := ym.OpenStream()
	if err != nil {
		ym.Close()
		return nil, fmt.Errorf("DirectKCPDialer: stream: %w", err)
	}
	return stream, nil
}

// ---------------------------------------------------------------------------
// LionheartOutbound is a sing-box adapter.Outbound-compatible struct that
// routes connections through the Lionheart KCP tunnel.
//
// FIXED: all three packages (adapter, sbLog, option) are now imported above.
// Previously these were referenced without imports, causing:
//   - "undefined: adapter" on lines 362, 389, 396, 402
//   - "undefined: log"     on line 402
//   - "undefined: option"  on line 402
// ---------------------------------------------------------------------------

type LionheartOutbound struct {
	transport *KCPTransport
	tag       string
	logger    adapter.Logger     // FIXED: uses imported adapter package
}

// NewLionheartOutbound creates an outbound that uses the given KCPTransport.
// router and logger satisfy the adapter.Router / adapter.Logger interfaces
// required by sing-box outbound constructors.
func NewLionheartOutbound(
	router adapter.Router,  // FIXED: adapter imported
	logger adapter.Logger,  // FIXED: adapter imported
	_ option.Outbound,      // FIXED: option imported — carry the outbound tag/cfg
	tag string,
	transport *KCPTransport,
) *LionheartOutbound {
	return &LionheartOutbound{
		transport: transport,
		tag:       tag,
		logger:    logger,
	}
}

// Tag returns the outbound tag.
func (o *LionheartOutbound) Tag() string { return o.tag }

// Type returns the outbound type string recognised by sing-box.
func (o *LionheartOutbound) Type() string { return "lionheart" }

// DialContext opens a connection through the KCP transport.
func (o *LionheartOutbound) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return o.transport.DialContext(ctx, network, addr)
}

// ---------------------------------------------------------------------------
// HealthCheck pings the remote through the tunnel and returns the RTT.
// ---------------------------------------------------------------------------

func TransportHealthCheck(t *KCPTransport) (time.Duration, error) {
	ym, ok := t.sess.Get()
	if !ok || ym == nil {
		return 0, fmt.Errorf("TransportHealthCheck: not connected")
	}
	start := time.Now()
	_, err := ym.Ping()
	if err != nil {
		return 0, fmt.Errorf("TransportHealthCheck: ping: %w", err)
	}
	return time.Since(start), nil
}
