package vault

import (
	"strings"
	"testing"
)

func TestGeneratePasswordLength(t *testing.T) {
	for _, l := range []int{4, 16, 32, 64} {
		pw, err := GeneratePassword(l, false)
		if err != nil {
			t.Fatalf("GeneratePassword(%d): %v", l, err)
		}
		if len(pw) != l {
			t.Errorf("expected length %d, got %d", l, len(pw))
		}
	}
}

func TestGeneratePasswordTooShort(t *testing.T) {
	_, err := GeneratePassword(3, false)
	if err == nil {
		t.Fatal("expected error for length < 4")
	}
}

func TestGeneratePasswordUniqueness(t *testing.T) {
	pw1, _ := GeneratePassword(24, true)
	pw2, _ := GeneratePassword(24, true)
	if pw1 == pw2 {
		t.Error("two generated passwords should not be equal (extremely unlikely)")
	}
}

func TestGeneratePasswordSymbols(t *testing.T) {
	found := false
	for i := 0; i < 20; i++ {
		pw, _ := GeneratePassword(32, true)
		if strings.ContainsAny(pw, specialChars) {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one special character in 20 attempts with symbols=true")
	}
}

func TestGeneratePasswordNoSymbols(t *testing.T) {
	for i := 0; i < 10; i++ {
		pw, _ := GeneratePassword(32, false)
		if strings.ContainsAny(pw, specialChars) {
			t.Errorf("found special char in no-symbols password: %q", pw)
		}
	}
}
