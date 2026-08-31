package bitrixurl

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var webhookPath = regexp.MustCompile(`(?i)^/rest/\d+/[A-Za-z0-9]+/?$`)

func NormalizeWebhook(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty webhook")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse webhook")
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return nil, fmt.Errorf("https required")
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "" || strings.Contains(host, "..") {
		return nil, fmt.Errorf("invalid host")
	}
	if err := AssertPublicHost(host); err != nil {
		return nil, err
	}
	if u.User != nil || u.Fragment != "" {
		return nil, fmt.Errorf("invalid webhook")
	}
	path := u.EscapedPath()
	if path == "" {
		path = u.Path
	}
	if !webhookPath.MatchString(path) {
		return nil, fmt.Errorf("webhook path")
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	out := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   path,
	}
	if u.Port() != "" && u.Port() != "443" {
		out.Host = net.JoinHostPort(host, u.Port())
	}
	return out, nil
}

func AssertPublicHost(host string) error {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return fmt.Errorf("blocked host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if BlockedIP(ip) {
			return fmt.Errorf("blocked host")
		}
		return nil
	}
	return nil
}

func AssertPublicResolved(host string) error {
	if err := AssertPublicHost(host); err != nil {
		return err
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve host")
	}
	if len(ips) == 0 {
		return fmt.Errorf("resolve host")
	}
	for _, ip := range ips {
		if BlockedIP(ip) {
			return fmt.Errorf("blocked host")
		}
	}
	return nil
}

func BlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

func MaskWebhook(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return "••••"
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 3 && strings.EqualFold(parts[0], "rest") {
		return fmt.Sprintf("https://%s/rest/%s/••••", u.Hostname(), parts[1])
	}
	return "https://" + u.Hostname() + "/rest/••••"
}

func MethodURL(base *url.URL, method string) string {
	method = strings.Trim(method, "/")
	if method != "" && !strings.HasSuffix(strings.ToLower(method), ".json") {
		method += ".json"
	}
	u := *base
	u.Path = strings.TrimSuffix(base.Path, "/") + "/" + method
	return u.String()
}
