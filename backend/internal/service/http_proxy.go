package service

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

func noHTTPProxy(*http.Request) (*url.URL, error) {
	return nil, nil
}

func directHTTPTransport() *http.Transport {
	return &http.Transport{
		Proxy:             noHTTPProxy,
		ForceAttemptHTTP2: false,
		TLSNextProto:      map[string]func(string, *tls.Conn) http.RoundTripper{},
	}
}

func normalizeProxyURLs(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		clean := strings.TrimSpace(raw)
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}

// parseHTTPProxyURL parses http://user:pass@host:port without treating literal
// percent signs in credentials as URL escapes (net/url.Parse fails on "%Ok" etc.).
func parseHTTPProxyURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty proxy url")
	}
	if !strings.HasPrefix(strings.ToLower(raw), "http://") {
		return nil, fmt.Errorf("only http:// proxies are supported")
	}

	rest := raw[len("http://"):]
	if rest == "" {
		return nil, fmt.Errorf("missing host")
	}

	hostPart := rest
	userinfo := ""
	if at := strings.LastIndex(rest, "@"); at >= 0 {
		userinfo = rest[:at]
		hostPart = rest[at+1:]
	}
	if strings.TrimSpace(hostPart) == "" {
		return nil, fmt.Errorf("missing host")
	}

	u := &url.URL{Scheme: "http", Host: hostPart}
	if userinfo != "" {
		user, pass, hasPass := strings.Cut(userinfo, ":")
		if hasPass {
			u.User = url.UserPassword(user, pass)
		} else {
			u.User = url.User(userinfo)
		}
	}
	return u, nil
}

func maskProxyURLForError(raw string) string {
	u, err := parseHTTPProxyURL(raw)
	if err != nil || u.Host == "" {
		return "proxy"
	}
	host := u.Hostname()
	if p := u.Port(); p != "" {
		host = net.JoinHostPort(host, p)
	}
	if u.User != nil {
		return host + " (authenticated)"
	}
	return host
}

func containsProxyURL(items []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, item := range items {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}

func proxyOrder(activeURL string, urls []string) []string {
	if len(urls) == 0 {
		return nil
	}
	active := strings.TrimSpace(activeURL)
	if active == "" || !slices.Contains(urls, active) {
		return urls
	}
	out := make([]string, 0, len(urls))
	out = append(out, active)
	for _, raw := range urls {
		if raw == active {
			continue
		}
		out = append(out, raw)
	}
	return out
}

func cloneProxyURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	c := *u
	return &c
}

func directDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var dialer net.Dialer
	dialer.Timeout = 15 * time.Second
	return dialer.DialContext(ctx, network, addr)
}

func transportViaHTTPConnectProxy(proxyURL *url.URL) *http.Transport {
	proxy := cloneProxyURL(proxyURL)
	t := directHTTPTransport()
	t.DialContext = directDialContext
	t.DialTLSContext = dialTLSViaHTTPConnectProxy(proxy)
	return t
}

func dialTLSViaHTTPConnectProxy(proxyURL *url.URL) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, fmt.Errorf("unsupported network %q", network)
		}
		conn, err := connectViaHTTPProxy(ctx, proxyURL, addr)
		if err != nil {
			return nil, err
		}
		host, _, splitErr := net.SplitHostPort(addr)
		if splitErr != nil {
			conn.Close()
			return nil, splitErr
		}
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
			NextProtos: []string{"http/1.1"},
		})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			conn.Close()
			return nil, err
		}
		return tlsConn, nil
	}
}

func connectViaHTTPProxy(ctx context.Context, proxyURL *url.URL, targetAddr string) (net.Conn, error) {
	if proxyURL == nil {
		return nil, errors.New("proxy url is required")
	}
	scheme := strings.ToLower(strings.TrimSpace(proxyURL.Scheme))
	if scheme != "http" {
		return nil, fmt.Errorf("unsupported proxy scheme %q (use http://)", proxyURL.Scheme)
	}

	proxyAddr := proxyURL.Host
	if _, _, err := net.SplitHostPort(proxyAddr); err != nil {
		proxyAddr = net.JoinHostPort(proxyAddr, "80")
	}

	var dialer net.Dialer
	dialer.Timeout = 15 * time.Second
	conn, err := dialer.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("proxy dial: %w", err)
	}

	var req strings.Builder
	req.WriteString("CONNECT ")
	req.WriteString(targetAddr)
	req.WriteString(" HTTP/1.1\r\nHost: ")
	req.WriteString(targetAddr)
	req.WriteString("\r\n")
	if proxyURL.User != nil {
		user := proxyURL.User.Username()
		pass, _ := proxyURL.User.Password()
		token := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		req.WriteString("Proxy-Authorization: Basic ")
		req.WriteString(token)
		req.WriteString("\r\n")
	}
	req.WriteString("User-Agent: RigIntel-ProxyConnect/1.0\r\n")
	req.WriteString("Proxy-Connection: Keep-Alive\r\n")
	req.WriteString("\r\n")

	if _, err := conn.Write([]byte(req.String())); err != nil {
		conn.Close()
		return nil, fmt.Errorf("proxy connect write: %w", err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("proxy connect response: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		conn.Close()
		return nil, fmt.Errorf("proxy connect status %s", resp.Status)
	}
	return conn, nil
}

func httpClientForProxy(base *http.Client, proxyURL string) (*http.Client, error) {
	parsed, err := parseHTTPProxyURL(proxyURL)
	if err != nil {
		return nil, err
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("invalid proxy url %q", proxyURL)
	}
	if base == nil {
		base = &http.Client{}
	}
	return &http.Client{
		Timeout:   base.Timeout,
		Transport: transportViaHTTPConnectProxy(parsed),
	}, nil
}
