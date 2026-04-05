package core

import (
	"github.com/sagernet/sing-box/option"
)

// PrivateIPCIDR is the list of RFC-private address ranges used in routing rules.
var PrivateIPCIDR = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"255.255.255.255/32",
	"::/128",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
	"ff00::/8",
}

// TelegramIPCIDR is the list of Telegram server address ranges.
var TelegramIPCIDR = []string{
	"91.108.4.0/22",
	"91.108.8.0/22",
	"91.108.12.0/22",
	"91.108.16.0/22",
	"91.108.20.0/22",
	"91.108.56.0/22",
	"91.108.56.0/23",
	"109.239.140.0/24",
	"149.154.160.0/20",
	"149.154.164.0/22",
	"2001:67c:4e8::/48",
	"2001:b28:f23c::/48",
	"2001:b28:f23d::/48",
	"2001:b28:f23f::/48",
	"2001:b28:f242::/48",
}

// MTProtoBypassDomains lists domains that should bypass the proxy for MTProto mode.
var MTProtoBypassDomains = []string{
	"pushwoosh.com",
}

// CommonProxyPorts contains well-known proxy/web ports.
// FIXED: was []int{} — must be []uint16 to match option.DefaultRouteRule.Port type.
var CommonProxyPorts = []uint16{
	80, 443, 8080, 8443, 1080, 1081, 3128,
}

// DNSPorts is the list of DNS ports used for DNS routing rules.
// FIXED: was []int{} — must be []uint16.
var DNSPorts = []uint16{53}

// BuildBypassRules returns routing rules that send LAN/private traffic direct
// and everything else through the proxy outbound.
func BuildBypassRules(proxyOutbound, directOutbound string) []option.Rule {
	return []option.Rule{
		// DNS goes direct to avoid loops inside sing-box
		{
			DefaultOptions: option.DefaultRule{
				Port:     []uint16{53},
				Outbound: directOutbound,
			},
		},
		// Private ranges go direct
		{
			DefaultOptions: option.DefaultRule{
				IPCIDR:   PrivateIPCIDR,
				Outbound: directOutbound,
			},
		},
		// Everything else through proxy
		{
			DefaultOptions: option.DefaultRule{
				Outbound: proxyOutbound,
			},
		},
	}
}

// BuildTelegramRules returns routing rules that route Telegram IPs through the
// proxy outbound and everything else direct.
func BuildTelegramRules(proxyOutbound, directOutbound string) []option.Rule {
	return []option.Rule{
		// DNS direct
		{
			DefaultOptions: option.DefaultRule{
				Port:     []uint16{53},
				Outbound: directOutbound,
			},
		},
		// Telegram IPs → proxy
		{
			DefaultOptions: option.DefaultRule{
				IPCIDR:   TelegramIPCIDR,
				Outbound: proxyOutbound,
			},
		},
		// Everything else direct
		{
			DefaultOptions: option.DefaultRule{
				Outbound: directOutbound,
			},
		},
	}
}

// BuildSplitRules builds routing rules for split-tunnel mode: apps in the
// allowList go through the proxy; everything else is direct.
func BuildSplitRules(proxyOutbound, directOutbound string, allowedPorts []uint16) []option.Rule {
	rules := []option.Rule{
		// DNS direct
		{
			DefaultOptions: option.DefaultRule{
				Port:     DNSPorts,
				Outbound: directOutbound,
			},
		},
	}

	if len(allowedPorts) > 0 {
		rules = append(rules, option.Rule{
			DefaultOptions: option.DefaultRule{
				Port:     allowedPorts,
				Outbound: proxyOutbound,
			},
		})
	}

	// Default: direct
	rules = append(rules, option.Rule{
		DefaultOptions: option.DefaultRule{
			Outbound: directOutbound,
		},
	})

	return rules
}
