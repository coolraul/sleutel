package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	KeyLen  = 32 // AES-256
	SaltLen = 32
	NonceLen = 12 // GCM standard nonce

	// Default Argon2id parameters — tuned for ~100ms on modern hardware.
	DefaultTime    uint32 = 3
	DefaultMemory  uint32 = 65536 // 64 MiB
	DefaultThreads uint8  = 4
)

// Params holds the Argon2id key derivation parameters.
type Params struct {
	Time    uint32
	Memory  uint32
	Threads uint8
}

// DefaultParams returns the recommended Argon2id parameters.
func DefaultParams() Params {
	return Params{
		Time:    DefaultTime,
		Memory:  DefaultMemory,
		Threads: DefaultThreads,
	}
}

// DeriveKey derives a 32-byte AES key from a password and salt using Argon2id.
func DeriveKey(password, salt []byte, p Params) []byte {
	return argon2.IDKey(password, salt, p.Time, p.Memory, p.Threads, KeyLen)
}

// Encrypt encrypts plaintext using AES-256-GCM with the provided key and nonce.
// The nonce must be exactly NonceLen bytes. Returns ciphertext with the GCM tag appended.
func Encrypt(key, nonce, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	return gcm.Seal(nil, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM.
func Decrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Do not leak internal error details — wrong password looks the same as tampering.
		return nil, errors.New("decryption failed: wrong password or corrupted vault")
	}
	return plain, nil
}

// RandBytes fills a slice of n bytes from the system CSPRNG.
func RandBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, fmt.Errorf("rand read: %w", err)
	}
	return b, nil
}

// Zero overwrites a byte slice with zeros to reduce the window in which
// sensitive material sits in memory.
func Zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
