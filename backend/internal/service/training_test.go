package service

import "testing"

func TestNextTrainingStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		current, action, want string
		ok                    bool
	}{
		{current: "draft", action: "submit", want: "submitted", ok: true},
		{current: "draft", action: "cancel", want: "cancelled", ok: true},
		{current: "submitted", action: "cancel", want: "cancelled", ok: true},
		{current: "submitted", action: "approve", want: "approved", ok: true},
		{current: "submitted", action: "reject", want: "rejected", ok: true},
		{current: "approved", action: "submit", ok: false},
		{current: "draft", action: "approve", ok: false},
		{current: "rejected", action: "cancel", ok: false},
	}
	for _, tc := range cases {
		got, err := NextTrainingStatus(tc.current, tc.action)
		if tc.ok {
			if err != nil || got != tc.want {
				t.Fatalf("%s + %s: got %q %v want %q", tc.current, tc.action, got, err, tc.want)
			}
			continue
		}
		if err == nil {
			t.Fatalf("%s + %s: expected error", tc.current, tc.action)
		}
	}
}

func TestNormalizeUILocale(t *testing.T) {
	t.Parallel()
	if got := NormalizeUILocale("ru"); got != "ru" {
		t.Fatalf("ru: %q", got)
	}
	if got := NormalizeUILocale("pt-BR"); got != "pt-BR" {
		t.Fatalf("pt-BR: %q", got)
	}
	if got := NormalizeUILocale("xx"); got != "" {
		t.Fatalf("unknown: %q", got)
	}
}
