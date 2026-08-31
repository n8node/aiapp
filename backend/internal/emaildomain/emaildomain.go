package emaildomain

import (
	"encoding/json"
	"strings"
)

const ReservedSuperadminEmail = "erman.ai@yandex.ru"

func Normalize(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func DomainOf(email string) string {
	email = Normalize(email)
	_, domain, ok := strings.Cut(email, "@")
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(domain))
}

func ParseList(raw string) []string {
	var domains []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &domains); err != nil {
		return nil
	}
	out := make([]string, 0, len(domains))
	seen := map[string]struct{}{}
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		d = strings.TrimPrefix(d, "@")
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	return out
}

func EncodeList(domains []string) string {
	clean := ParseList(mustJSON(domains))
	b, _ := json.Marshal(clean)
	return string(b)
}

func mustJSON(domains []string) string {
	b, _ := json.Marshal(domains)
	return string(b)
}

func Allowed(email string, domains []string) bool {
	email = Normalize(email)
	if email == ReservedSuperadminEmail {
		return true
	}
	domain := DomainOf(email)
	if domain == "" {
		return false
	}
	for _, d := range domains {
		if domain == strings.ToLower(strings.TrimSpace(d)) {
			return true
		}
	}
	return false
}

func IsReservedSuperadmin(email string) bool {
	return Normalize(email) == ReservedSuperadminEmail
}
