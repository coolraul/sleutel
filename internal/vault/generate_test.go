package vault

import (
	"strings"
	"testing"
)

func TestGeneratePasswordLength(t *testing.T) {
	for _, l := range []int{3, 4, 16, 32, 64} {
		pw, err := GeneratePassword(l, false)
		if err != nil {
			t.Fatalf("GeneratePassword(%d, false): %v", l, err)
		}
		if len(pw) != l {
			t.Errorf("expected length %d, got %d", l, len(pw))
		}
	}
	for _, l := range []int{4, 16, 32, 64} {
		pw, err := GeneratePassword(l, true)
		if err != nil {
			t.Fatalf("GeneratePassword(%d, true): %v", l, err)
		}
		if len(pw) != l {
			t.Errorf("expected length %d, got %d", l, len(pw))
		}
	}
}

func TestGeneratePasswordMinLength(t *testing.T) {
	cases := []struct {
		length  int
		symbols bool
		wantErr bool
	}{
		{2, false, true},
		{3, false, false},
		{3, true, true},
		{4, true, false},
	}
	for _, tc := range cases {
		_, err := GeneratePassword(tc.length, tc.symbols)
		if tc.wantErr && err == nil {
			t.Errorf("GeneratePassword(%d, %v): expected error, got nil", tc.length, tc.symbols)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("GeneratePassword(%d, %v): unexpected error: %v", tc.length, tc.symbols, err)
		}
	}
}

func TestGeneratePasswordClassCoverage(t *testing.T) {
	for i := 0; i < 50; i++ {
		pw, err := GeneratePassword(8, true)
		if err != nil {
			t.Fatalf("GeneratePassword: %v", err)
		}
		if !strings.ContainsAny(pw, lowerAlpha) {
			t.Errorf("missing lowercase: %q", pw)
		}
		if !strings.ContainsAny(pw, upperAlpha) {
			t.Errorf("missing uppercase: %q", pw)
		}
		if !strings.ContainsAny(pw, digits) {
			t.Errorf("missing digit: %q", pw)
		}
		if !strings.ContainsAny(pw, specialChars) {
			t.Errorf("missing symbol: %q", pw)
		}
	}
}

func TestGeneratePasswordClassCoverageNoSymbols(t *testing.T) {
	for i := 0; i < 50; i++ {
		pw, err := GeneratePassword(6, false)
		if err != nil {
			t.Fatalf("GeneratePassword: %v", err)
		}
		if !strings.ContainsAny(pw, lowerAlpha) {
			t.Errorf("missing lowercase: %q", pw)
		}
		if !strings.ContainsAny(pw, upperAlpha) {
			t.Errorf("missing uppercase: %q", pw)
		}
		if !strings.ContainsAny(pw, digits) {
			t.Errorf("missing digit: %q", pw)
		}
		if strings.ContainsAny(pw, specialChars) {
			t.Errorf("unexpected symbol in no-symbols password: %q", pw)
		}
	}
}

func TestGeneratePasswordUniqueness(t *testing.T) {
	pw1, _ := GeneratePassword(24, true)
	pw2, _ := GeneratePassword(24, true)
	if pw1 == pw2 {
		t.Error("two generated passwords should not be equal (extremely unlikely)")
	}
}

func TestGeneratePasswordExcludedChars(t *testing.T) {
	excluded := "\"'`\\<> "
	for i := 0; i < 100; i++ {
		pw, _ := GeneratePassword(32, true)
		if strings.ContainsAny(pw, excluded) {
			t.Errorf("password contains excluded character: %q", pw)
		}
	}
}
