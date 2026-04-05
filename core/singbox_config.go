package core

import (
	"encoding/json"
	"github.com/sagernet/sing-box/option"
)

// Config - основная структура для параметров конфигурации
type Config struct {
	ProxyAddress string
	ProxyPort    int
	DNSServer    string
	IPv6         bool
	MTU          uint32
}

// BuildOptions собирает полную структуру option.Options для sing-box
func (c *Config) BuildOptions() (*option.Options, error) {
	// 1. Настройка DNS
	dnsOptions := &option.DNSOptions{
		Servers: []option.DNSServerOptions{
			{
				Tag:     "google",
				Address: "8.8.8.8",
			},
			{
				Tag:     "local",
				Address: "223.5.5.5",
			},
		},
		Rules: []option.DNSRule{
			{
				// ИСПРАВЛЕНИЕ ОШИБОК 155, 156: 
				// В новых версиях поля Outbound и Server находятся внутри DefaultOptions
				Type: "default",
				DefaultOptions: option.DNSDefaultRule{
					Server:   "google",
					Outbound: "proxy",
				},
			},
		},
	}

	// 2. Настройка Inbounds
	inbounds := []option.Inbound{
		{
			Type: "tun",
			Tag:  "tun-in",
			// ИСПРАВЛЕНИЕ ОШИБКИ 177, 192, 193: 
			// Вместо TunOptions используется поле Options с приведением к типу
			Options: &option.TunInboundOptions{
				InterfaceName: "utun",
				MTU:           c.MTU,
				// ИСПРАВЛЕНИЕ ОШИБОК 182, 185: 
				// Тип Listable удален. Используем слайс ListenAddress через ParseListenAddress
				Inet4Address: []option.ListenAddress{
					option.ParseListenAddress("172.19.0.1/30"),
				},
				AutoRoute:   true,
				StrictRoute: true,
				Stack:       "system",
				Sniff:       true,
			},
		},
		{
			Type: "mixed",
			Tag:  "mixed-in",
			// ИСПРАВЛЕНИЕ ОШИБКИ 206:
			// Вместо MixedOptions используем HTTPInboundOptions (который заменяет mixed)
			Options: &option.HTTPInboundOptions{
				ListenOptions: option.ListenOptions{
					// ИСПРАВЛЕНИЕ ОШИБОК 208, 226:
					// Замена ListenAddress на ParseListenAddress
					Listen: option.ParseListenAddress("127.0.0.1"),
					Port:   2080,
				},
			},
		},
	}

	// 3. Настройка Outbounds
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

	// 4. Настройка маршрутизации (Route)
	routeOptions := &option.RouteOptions{
		Rules: []option.Rule{
			{
				Type: "default",
				DefaultOptions: option.DefaultRouteRule{
					Protocol: []string{"dns"},
					Outbound: "dns-out",
				},
			},
			{
				Type: "default",
				DefaultOptions: option.DefaultRouteRule{
					Port:     []uint16{53},
					Outbound: "dns-out",
				},
			},
		},
		Final: "proxy",
	}

	return &option.Options{
		DNS:       dnsOptions,
		Inbounds:  inbounds,
		Outbounds: outbounds,
		Route:     routeOptions,
	}, nil
}

// MarshalConfig преобразует конфиг в JSON строку для передачи в sing-box
func (c *Config) MarshalConfig() (string, error) {
	opts, err := c.BuildOptions()
	if err != nil {
		return "", err
	}
	
	jsonBytes, err := json.MarshalIndent(opts, "", "  ")
	if err != nil {
		return "", err
	}
	
	return string(jsonBytes), nil
}
