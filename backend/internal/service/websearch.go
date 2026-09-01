package service

import (
	"context"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/n8node/aiapp/internal/model"
)

const (
	webSearchHost    = "html.duckduckgo.com"
	webSearchMaxBody = 256 << 10
	webSearchMaxHits = 5
)

var (
	ddgResultRe = regexp.MustCompile(`(?i)<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	htmlTagRe   = regexp.MustCompile(`<[^>]+>`)
)

func webSearchAllowedHost(host string) bool {
	return strings.EqualFold(host, webSearchHost)
}

func publicHTTPSURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return false
	}
	if host == "metadata.google.internal" || strings.HasPrefix(host, "metadata.") {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return false
	}
	return true
}

func decodeDDGHref(href string) string {
	href = html.UnescapeString(strings.TrimSpace(href))
	if strings.HasPrefix(href, "//") {
		href = "https:" + href
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if q := u.Query().Get("uddg"); q != "" {
		decoded, err := url.QueryUnescape(q)
		if err != nil {
			return ""
		}
		if publicHTTPSURL(decoded) {
			return decoded
		}
		return ""
	}
	if publicHTTPSURL(u.String()) {
		return u.String()
	}
	return ""
}

func parseDuckDuckGoHTML(body string) []model.ChatCitation {
	matches := ddgResultRe.FindAllStringSubmatch(body, webSearchMaxHits*3)
	var out []model.ChatCitation
	seen := map[string]struct{}{}
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		dest := decodeDDGHref(m[1])
		if dest == "" {
			continue
		}
		if _, ok := seen[dest]; ok {
			continue
		}
		seen[dest] = struct{}{}
		title := strings.TrimSpace(html.UnescapeString(htmlTagRe.ReplaceAllString(m[2], "")))
		if title == "" {
			title = dest
		}
		out = append(out, model.ChatCitation{Source: "web", FileName: title, URL: dest})
		if len(out) >= webSearchMaxHits {
			break
		}
	}
	if out == nil {
		out = []model.ChatCitation{}
	}
	return out
}

func SearchWeb(ctx context.Context, query string) ([]model.ChatCitation, error) {
	q := strings.TrimSpace(query)
	if utf8CountRunes(q) < 2 {
		return []model.ChatCitation{}, nil
	}
	if utf8CountRunes(q) > 200 {
		q = string([]rune(q)[:200])
	}
	u := url.URL{Scheme: "https", Host: webSearchHost, Path: "/html/"}
	vals := url.Values{}
	vals.Set("q", q)
	u.RawQuery = vals.Encode()
	if !webSearchAllowedHost(u.Hostname()) || u.Scheme != "https" {
		return nil, ErrInvalidInput
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, ErrGatewayUnavailable
	}
	req.Header.Set("User-Agent", "RigIntelSearch/1.0")
	client := &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !webSearchAllowedHost(req.URL.Hostname()) || req.URL.Scheme != "https" {
				return http.ErrUseLastResponse
			}
			if len(via) >= 2 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return []model.ChatCitation{}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return []model.ChatCitation{}, nil
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, webSearchMaxBody))
	if err != nil {
		return []model.ChatCitation{}, nil
	}
	return parseDuckDuckGoHTML(string(raw)), nil
}
