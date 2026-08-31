package password

import "testing"

func TestValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in      string
		wantErr bool
	}{
		{in: "ValidPass10!", wantErr: false},
		{in: "Aa1!aaaa", wantErr: false},
		{in: "ValidPass10", wantErr: true},
		{in: "sh1A!", wantErr: true},
		{in: "nouppercase1!", wantErr: true},
		{in: "NOLOWERCASE1!", wantErr: true},
		{in: "NoDigitsHere!", wantErr: true},
		{in: "NoSpecial12", wantErr: true},
	}
	for _, tt := range tests {
		err := Validate(tt.in)
		if tt.wantErr && err == nil {
			t.Fatalf("%q: want error", tt.in)
		}
		if !tt.wantErr && err != nil {
			t.Fatalf("%q: %v", tt.in, err)
		}
	}
}

func TestValidateSeed(t *testing.T) {
	t.Parallel()
	if err := ValidateSeed("ValidPass10"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSeed("short"); err == nil {
		t.Fatal("expected error")
	}
}

func TestHashCompare(t *testing.T) {
	t.Parallel()
	hash, err := Hash("ValidPass10!")
	if err != nil {
		t.Fatal(err)
	}
	if err := Compare(hash, "ValidPass10!"); err != nil {
		t.Fatal(err)
	}
	if err := Compare(hash, "wrong-password"); err == nil {
		t.Fatal("expected mismatch")
	}
}
