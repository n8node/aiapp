package emaildomain

import "testing"

func TestAllowed(t *testing.T) {
	t.Parallel()
	domains := []string{"rigintel.ai", "rigintelpro.ru"}
	if !Allowed("erman.ai@yandex.ru", domains) {
		t.Fatal("superadmin exception")
	}
	if Allowed("other@yandex.ru", domains) {
		t.Fatal("yandex must be rejected for non-superadmin")
	}
	if !Allowed("user@rigintel.ai", domains) {
		t.Fatal("company domain")
	}
	if Allowed("user@gmail.com", domains) {
		t.Fatal("gmail")
	}
}
