package core

import (
	"encoding/json"

	"github.com/sagernet/sing-box/option"
)

// SingBoxConfig - эта структура должна быть публичной (с большой буквы),
// чтобы её видел файл tunnel.go
type SingBoxConfig struct {
	ProxyAddress string
	ProxyPort    int
	MTU          uint32
}

func (c *SingBoxConfig) BuildOptions() (*option.Options, error) {
	// DNS
	dnsOptions := &option.DNSOptions{
		Servers: []option.DNSServerOptions{
			{
				Tag:     "google",
				Address: "8.8.8.8",
			},
		},
		Rules: []option.DNSRule{
			{
				Type: "default",
				DefaultOptions: option.DNSDefaultRule{
					Server:   "google",
					Outbound: "proxy",
				},
			},
		},
	}

	// Inbounds
	inbounds := []option.Inbound{
		{
			Type: "tun",
			Tag:  "tun-in",
			Options: &option.TunInboundOptions{
				InterfaceName: "utun",
				MTU:           c.MTU,
				Inet4Address: []option.ListenAddress{
					option.ParseListenAddress("172.19.0.1/30"),
				},
				AutoRoute:   true,
				StrictRoute: true,
				Stack:       "system",
			},
		},
		{
			Type: "mixed",
			Tag:  "mixed-in",
			Options: &option.HTTPInboundOptions{
				ListenOptions: option.ListenOptions{
					Listen: option.ParseListenAddress("127.0.0.1"),
					Port:   2080,
				},
			},
		},
	}

	// Outbounds
	outbounds := []option.Outbound{
		{
			Type: "direct",
			Tag:  "direct",
		},
		{
			Type: "dns",
			Tag:  "dns-out",
		},
		{
			Type: "socks",
			Tag:  "proxy",
			Options: &option.SocksOutboundOptions{
				ServerOptions: option.ServerOptions{
					Server:     c.ProxyAddress,
					ServerPort: uint16(c.ProxyPort),
				},
			},
		},
	}

	return &option.Options{
		DNS:       dnsOptions,
		Inbounds:  inbounds,
		Outbounds: outbounds,
	}, nil
}

func (c *SingBoxConfig) MarshalJSON() (string, error) {
	opts, err := c.BuildOptions()
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(opts)
	return string(b), err
}
