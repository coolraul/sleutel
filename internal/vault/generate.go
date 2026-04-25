package vault

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	lowerAlpha  = "abcdefghijklmnopqrstuvwxyz"
	upperAlpha  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits      = "0123456789"
	specialChars = "!@#$%^&*()-_=+[]{}|;:,.<>?"
)

// GeneratePassword creates a cryptographically random password.
// If symbols is true, special characters are included.
func GeneratePassword(length int, symbols bool) (string, error) {
	if length < 4 {
		return "", fmt.Errorf("password length must be at least 4")
	}

	charset := lowerAlpha + upperAlpha + digits
	if symbols {
		charset += specialChars
	}

	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("rand: %w", err)
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}
