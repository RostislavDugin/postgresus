package telegram_notifier

type ProxyScheme string

const (
	ProxySchemeHTTP    ProxyScheme = "http"
	ProxySchemeHTTPS   ProxyScheme = "https"
	ProxySchemeSOCKS5  ProxyScheme = "socks5"
	ProxySchemeSOCKS5H ProxyScheme = "socks5h"
)

func (s ProxyScheme) IsAllowed() bool {
	return allowedProxySchemes[s]
}

var allowedProxySchemes = map[ProxyScheme]bool{
	ProxySchemeHTTP:    true,
	ProxySchemeHTTPS:   true,
	ProxySchemeSOCKS5:  true,
	ProxySchemeSOCKS5H: true,
}
