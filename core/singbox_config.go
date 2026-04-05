package core

import (
	"context"
	"fmt"
	"net/netip"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/option"
)

// SingBoxMode describes how sing-box is used inside Lionheart.
type SingBoxMode int

const (
	// SingBoxModeSocks5 starts a local SOCKS5 inbound that proxies through
	// the Lionheart tunnel.  No tun interface is created.
	SingBoxModeSocks5 SingBoxMode = iota

	// SingBoxModeTUN starts a full TUN inbound (requires a VPN file descriptor
	// provided by the Android VpnService).
	SingBoxModeTUN

	// SingBoxModeMTProto starts an MTProto proxy inbound for Telegram.
	SingBoxModeMTProto
)

// SingBoxConfig holds the parameters needed to build a sing-box instance.
type SingBoxConfig struct {
	Mode SingBoxMode

	// SmartKey is the Lionheart server key (host:port|password, base64).
	SmartKey string

	// DNS is the upstream DNS server address (e.g. "1.1.1.1").
	DNS string

	// MTU for the TUN interface (TUN mode only).
	MTU int

	// TunFd is the file descriptor of the pre-opened tun device (TUN mode only,
	// passed by Android VpnService). -1 means "not provided".
	TunFd int

	// ListenAddr is the local address for the SOCKS5 / MTProto inbound.
	ListenAddr string

	// ListenPort is the local port for the SOCKS5 / MTProto inbound.
	ListenPort uint16

	// RouteMode controls which traffic is routed through the tunnel.
	// Valid values: "all", "telegram".
	RouteMode string
}

// SingBoxInstance wraps a running sing-box and exposes Start/Stop.
type SingBoxInstance struct {
	instance *box.Box
	cancel   context.CancelFunc
}

// NewSingBoxInstance creates and starts a sing-box instance from cfg.
func NewSingBoxInstance(cfg SingBoxConfig) (*SingBoxInstance, error) {
	opts, err := buildOptions(cfg)
	if err != nil {
		return nil, fmt.Errorf("build options: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// FIXED: Do not use log.NewLogger / log.Options directly.
	// sing-box v1.11 manages its own logger internally via box.New().
	// Pass the root context and the option.Options struct only.
	instance, err := box.New(box.Options{
		Context: ctx,
		Options: opts,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("box.New: %w", err)
	}

	if err := instance.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("box.Start: %w", err)
	}

	return &SingBoxInstance{
		instance: instance,
		cancel:   cancel,
	}, nil
}

// Close stops the sing-box instance and releases resources.
func (s *SingBoxInstance) Close() error {
	s.cancel()
	return s.instance.Close()
}

// ---------------------------------------------------------------------------
// Internal option builders
// ---------------------------------------------------------------------------

func buildOptions(cfg SingBoxConfig) (option.Options, error) {
	peer, pw, err := ParseSmartKey(cfg.SmartKey)
	if err != nil {
		return option.Options{}, err
	}

	if cfg.DNS == "" {
		cfg.DNS = "1.1.1.1"
	}
	if cfg.MTU <= 0 {
		cfg.MTU = 1500
	}

	// FIXED: Use option.LogOptions (from sing-box/option), NOT log.Options
	// (from sing-box/log).  The option.LogOptions struct uses Level as a
	// plain string ("info", "debug", …); there is no log.Options.Level
	// field in sing-box v1.11 accessible to external callers.
	logOpts := &option.LogOptions{
		Disabled: false,
		Level:    "warn", // string, not log.Level — this is option.LogOptions.Level
	}

	dnsOpts := buildDNS(cfg.DNS)
	inbounds := buildInbounds(cfg)
	outbounds := buildOutbounds(peer, pw)
	route := buildRoute(cfg, outbounds)

	return option.Options{
		Log:       logOpts,
		DNS:       dnsOpts,
		Inbounds:  inbounds,
		Outbounds: outbounds,
		Route:     route,
	}, nil
}

func buildDNS(upstream string) *option.DNSOptions {
	return &option.DNSOptions{
		Servers: []option.DNSServerOptions{
			{
				Tag:     "remote",
				Address: "https://" + upstream + "/dns-query",
			},
			{
				Tag:     "local",
				Address: "local",
			},
		},
		Rules: []option.DNSRule{
			{
				DefaultOptions: option.DefaultDNSRule{
					Outbound: []string{"any"},
					Server:   "local",
				},
			},
		},
		Final: "remote",
	}
}

func buildInbounds(cfg SingBoxConfig) []option.Inbound {
	listenAddr := cfg.ListenAddr
	if listenAddr == "" {
		listenAddr = "127.0.0.1"
	}

	addr, _ := netip.ParseAddr(listenAddr)

	switch cfg.Mode {
	case SingBoxModeTUN:
		inbound := option.Inbound{
			Type: "tun",
			Tag:  "tun-in",
			TunOptions: option.TunInboundOptions{
				InterfaceName: "tun0",
				MTU:           uint32(cfg.MTU),
				AutoRoute:     true,
				StrictRoute:   false,
				Inet4Address:  option.Listable[netip.Prefix]{
					netip.MustParsePrefix("172.19.0.1/30"),
				},
				Inet6Address: option.Listable[netip.Prefix]{
					netip.MustParsePrefix("fdfe:dcba:9876::1/126"),
				},
			},
		}
		if cfg.TunFd >= 0 {
			fd := uint32(cfg.TunFd)
			inbound.TunOptions.FileDescriptor = int(fd)
			inbound.TunOptions.AutoRoute = false
		}
		return []option.Inbound{inbound}

	case SingBoxModeMTProto:
		port := cfg.ListenPort
		if port == 0 {
			port = 1080
		}
		return []option.Inbound{
			{
				Type: "mixed",
				Tag:  "mixed-in",
				MixedOptions: option.HTTPMixedInboundOptions{
					ListenOptions: option.ListenOptions{
						Listen:     (*option.ListenAddress)(&addr),
						ListenPort: port,
					},
				},
			},
		}

	default: // SingBoxModeSocks5
		port := cfg.ListenPort
		if port == 0 {
			port = 2080
		}
		return []option.Inbound{
			{
				Type: "socks",
				Tag:  "socks-in",
				SocksOptions: option.SocksInboundOptions{
					ListenOptions: option.ListenOptions{
						Listen:     (*option.ListenAddress)(&addr),
						ListenPort: port,
					},
				},
			},
		}
	}
}

func buildOutbounds(peer, pw string) []option.Outbound {
	// The Lionheart tunnel is exposed locally as a SOCKS5 proxy on a dynamic
	// port by the engine.  Here we configure sing-box to use a SOCKS5
	// outbound that talks to the engine's local proxy.
	//
	// The engine starts on 127.0.0.1:0 and writes the actual port to the
	// shared SingBoxTunnelPort variable (set by engine before calling here).
	tunnelPort := uint16(SingBoxTunnelPort)

	_ = peer // peer / pw used by the engine layer; not needed here
	_ = pw

	return []option.Outbound{
		{
			Type: "socks",
			Tag:  "proxy",
			SocksOptions: option.SocksOutboundOptions{
				ServerOptions: option.ServerOptions{
					Server:     "127.0.0.1",
					ServerPort: tunnelPort,
				},
				Version: "5",
			},
		},
		{
			Type: "direct",
			Tag:  "direct",
		},
		{
			Type: "block",
			Tag:  "block",
		},
		{
			Type: "dns",
			Tag:  "dns-out",
		},
	}
}

func buildRoute(cfg SingBoxConfig, outbounds []option.Outbound) *option.RouteOptions {
	proxyTag := outbounds[0].Tag

	var rules []option.Rule
	switch cfg.RouteMode {
	case "telegram":
		rules = BuildTelegramRules(proxyTag, "direct")
	default:
		rules = BuildBypassRules(proxyTag, "direct")
	}

	return &option.RouteOptions{
		Rules: rules,
		Final: proxyTag,
		AutoDetectInterface: true,
	}
}

// SingBoxTunnelPort is set by the engine to the actual local SOCKS5 port
// before NewSingBoxInstance is called.  Zero means "not yet assigned".
var SingBoxTunnelPort int
