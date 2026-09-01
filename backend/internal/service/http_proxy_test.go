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

func TestIsHuggingFaceHubHost(t *testing.T) {
	if !isHuggingFaceHubHost("huggingface.co") || !isHuggingFaceHubHost("cdn-lfs.huggingface.co") {
		t.Fatal("hub hosts should match")
	}
	if isHuggingFaceHubHost("us.aws.cdn.hf.co") || isHuggingFaceHubHost("hf.co") || isHuggingFaceHubHost("cas-bridge.xethub.hf.co") {
		t.Fatal("CDN hosts must not match hub NO_PROXY")
	}
}

func TestHostFromURL(t *testing.T) {
	if got := hostFromURL("https://US.AWS.CDN.HF.CO/path?sig=secret"); got != "us.aws.cdn.hf.co" {
		t.Fatalf("got %q", got)
	}
}
