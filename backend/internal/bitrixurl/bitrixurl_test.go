package bitrixurl

import (
	"net"
	"strings"
	"testing"
)

func TestNormalizeWebhookAcceptsBitrixPath(t *testing.T) {
	u, err := NormalizeWebhook("https://bitrix.rigintel.ai/rest/1/abcTOKEN99/")
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "bitrix.rigintel.ai" || !strings.HasSuffix(u.Path, "/") {
		t.Fatalf("got %s %s", u.Host, u.Path)
	}
}

func TestNormalizeWebhookRejects(t *testing.T) {
	bads := []string{
		"http://bitrix.rigintel.ai/rest/1/abc/",
		"https://localhost/rest/1/abc/",
		"https://127.0.0.1/rest/1/abc/",
		"https://169.254.169.254/rest/1/abc/",
		"https://bitrix.rigintel.ai/rest/",
		"https://bitrix.rigintel.ai/",
		"https://bitrix.rigintel.ai/rest/1/abc/department.get",
	}
	for _, raw := range bads {
		if _, err := NormalizeWebhook(raw); err == nil {
			t.Fatalf("expected reject: %s", raw)
		}
	}
}

func TestMaskWebhookHidesSecret(t *testing.T) {
	got := MaskWebhook("https://bitrix.rigintel.ai/rest/1/supersecret/")
	if strings.Contains(got, "supersecret") {
		t.Fatalf("leaked: %s", got)
	}
	if !strings.Contains(got, "bitrix.rigintel.ai") {
		t.Fatalf("host missing: %s", got)
	}
}

func TestMethodURLAddsJSON(t *testing.T) {
	u, err := NormalizeWebhook("https://bitrix.rigintel.ai/rest/1/abcTOKEN99/")
	if err != nil {
		t.Fatal(err)
	}
	got := MethodURL(u, "department.get")
	if !strings.HasSuffix(got, "/department.get.json") {
		t.Fatalf("got %s", got)
	}
}

func TestBlockedIP(t *testing.T) {
	if !BlockedIP(net.ParseIP("10.0.0.1")) || !BlockedIP(net.ParseIP("192.168.1.1")) {
		t.Fatal("private should block")
	}
	if BlockedIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("public should pass")
	}
}
