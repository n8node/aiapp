package password

import "testing"

func TestValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in      string
		wantErr bool
	}{
		{in: "ValidPass10", wantErr: false},
		{in: "short1A", wantErr: true},
		{in: "nouppercase1", wantErr: true},
		{in: "NOLOWERCASE1", wantErr: true},
		{in: "NoDigitsHere", wantErr: true},
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

func TestHashCompare(t *testing.T) {
	t.Parallel()
	hash, err := Hash("ValidPass10")
	if err != nil {
		t.Fatal(err)
	}
	if err := Compare(hash, "ValidPass10"); err != nil {
		t.Fatal(err)
	}
	if err := Compare(hash, "wrong-password"); err == nil {
		t.Fatal("expected mismatch")
	}
}
