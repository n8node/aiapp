package service

import "testing"

func TestParseHTTPProxyURLPercentInPassword(t *testing.T) {
	raw := "http://root:ryO3N%OkP7%G@5.35.83.120:3128"
	u, err := parseHTTPProxyURL(raw)
	if err != nil {
		t.Fatalf("parseHTTPProxyURL: %v", err)
	}
	if u.Host != "5.35.83.120:3128" {
		t.Fatalf("host = %q", u.Host)
	}
	user := u.User.Username()
	pass, ok := u.User.Password()
	if !ok || user != "root" || pass != "ryO3N%OkP7%G" {
		t.Fatalf("credentials = %q / %q (ok=%v)", user, pass, ok)
	}
}

func TestParseHTTPProxyURLWithoutAuth(t *testing.T) {
	u, err := parseHTTPProxyURL("http://127.0.0.1:3128")
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "127.0.0.1:3128" || u.User != nil {
		t.Fatalf("unexpected url: %+v", u)
	}
}

func TestHTTPClientForProxyPercentInPassword(t *testing.T) {
	_, err := httpClientForProxy(nil, "http://root:ryO3N%OkP7%G@5.35.83.120:3128")
	if err != nil {
		t.Fatalf("httpClientForProxy: %v", err)
	}
}

func TestMaskProxyURLForErrorHidesCredentials(t *testing.T) {
	got := maskProxyURLForError("http://tgproxy:secret@5.35.83.120:3128")
	if got != "5.35.83.120:3128 (authenticated)" {
		t.Fatalf("got %q", got)
	}
}

func TestProxyOrderPutsActiveFirst(t *testing.T) {
	got := proxyOrder("http://b:3128", []string{"http://a:3128", "http://b:3128"})
	if len(got) != 2 || got[0] != "http://b:3128" || got[1] != "http://a:3128" {
		t.Fatalf("got %v", got)
	}
}
