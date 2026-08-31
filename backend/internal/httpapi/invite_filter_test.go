package httpapi

import (
	"net/url"
	"testing"
	"time"
)

func TestParseDayBoundsMoscow(t *testing.T) {
	start, err := parseDayStart("2026-08-31")
	if err != nil || start == nil {
		t.Fatalf("start: %v %v", start, err)
	}
	end, err := parseDayEnd("2026-08-31")
	if err != nil || end == nil {
		t.Fatalf("end: %v %v", end, err)
	}
	if !end.Equal(start.Add(24 * time.Hour)) {
		t.Fatalf("end should be next day, got %s %s", start, end)
	}
	empty, err := parseDayStart("")
	if err != nil || empty != nil {
		t.Fatalf("empty should be nil: %v %v", empty, err)
	}
	if _, err := parseDayStart("31.08.2026"); err == nil {
		t.Fatal("expected invalid date")
	}
}

func TestParseInviteListFilter(t *testing.T) {
	f, err := parseInviteListFilter(url.Values{
		"used_email":   []string{"  anna@rigintel.ai "},
		"created_from": []string{"2026-08-01"},
		"limit":        []string{"200"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.UsedEmail != "anna@rigintel.ai" {
		t.Fatalf("email: %q", f.UsedEmail)
	}
	if f.CreatedFrom == nil || f.Limit != 200 {
		t.Fatalf("filter: %+v", f)
	}
}
