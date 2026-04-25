package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key, err := RandBytes(KeyLen)
	if err != nil {
		t.Fatal(err)
	}
	nonce, err := RandBytes(NonceLen)
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte("super secret payload")

	ct, err := Encrypt(key, nonce, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	got, err := Decrypt(key, nonce, ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(got, plaintext) {
		t.Fatalf("roundtrip mismatch: got %q want %q", got, plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key, _ := RandBytes(KeyLen)
	nonce, _ := RandBytes(NonceLen)
	ct, _ := Encrypt(key, nonce, []byte("secret"))

	wrongKey, _ := RandBytes(KeyLen)
	_, err := Decrypt(wrongKey, nonce, ct)
	if err == nil {
		t.Fatal("expected error with wrong key, got nil")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key, _ := RandBytes(KeyLen)
	nonce, _ := RandBytes(NonceLen)
	ct, _ := Encrypt(key, nonce, []byte("secret"))

	ct[0] ^= 0xFF // flip bits in first byte
	_, err := Decrypt(key, nonce, ct)
	if err == nil {
		t.Fatal("expected error with tampered ciphertext, got nil")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	password := []byte("hunter2")
	salt := make([]byte, SaltLen)
	p := Params{Time: 1, Memory: 8192, Threads: 1}

	k1 := DeriveKey(password, salt, p)
	k2 := DeriveKey(password, salt, p)

	if !bytes.Equal(k1, k2) {
		t.Fatal("DeriveKey is not deterministic")
	}
}

func TestDeriveKeyDifferentPasswords(t *testing.T) {
	salt := make([]byte, SaltLen)
	p := Params{Time: 1, Memory: 8192, Threads: 1}

	k1 := DeriveKey([]byte("password1"), salt, p)
	k2 := DeriveKey([]byte("password2"), salt, p)

	if bytes.Equal(k1, k2) {
		t.Fatal("different passwords produced same key")
	}
}

func TestZero(t *testing.T) {
	b := []byte{1, 2, 3, 4, 5}
	Zero(b)
	for i, v := range b {
		if v != 0 {
			t.Fatalf("byte %d not zeroed: got %d", i, v)
		}
	}
}
