package model

type OutboundProxySettings struct {
	ProxyEnabled   bool     `json:"proxy_enabled"`
	ProxyActiveURL string   `json:"proxy_active_url"`
	ProxyURLs      []string `json:"proxy_urls"`
}

func DefaultOutboundProxySettings() OutboundProxySettings {
	return OutboundProxySettings{
		ProxyEnabled:   false,
		ProxyActiveURL: "",
		ProxyURLs:      []string{},
	}
}

type OutboundProxyAdminView struct {
	ProxyEnabled   bool     `json:"proxy_enabled"`
	ProxyActiveURL string   `json:"proxy_active_url"`
	ProxyURLs      []string `json:"proxy_urls"`
	ActiveMasked   string   `json:"active_masked"`
}

type OutboundProxyTestResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Hub     string `json:"hub"`
	CDN     string `json:"cdn"`
}
