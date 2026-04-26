package vault

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	lowerAlpha = "abcdefghijklmnopqrstuvwxyz"
	upperAlpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits     = "0123456789"
	// Symbols limited to characters broadly accepted by website password fields.
	// Excluded: quotes (' " `), backslash (\), angle brackets (< >), space.
	specialChars = "!@#$%^&*()-_=+[]{}|;:,.?"
)

// GeneratePassword creates a cryptographically random password that satisfies
// common website password policies.
//
// symbols=false requires length >= 3 (one per mandatory class: lower, upper, digit).
// symbols=true  requires length >= 4 (adds one mandatory symbol).
func GeneratePassword(length int, symbols bool) (string, error) {
	minLen := 3
	if symbols {
		minLen = 4
	}
	if length < minLen {
		return "", fmt.Errorf("password length must be at least %d (symbols=%v)", minLen, symbols)
	}

	// Seed the result with one character from each required class so that
	// every generated password satisfies character-class policies.
	result := make([]byte, 0, length)
	for _, class := range mandatoryClasses(symbols) {
		ch, err := randomChar(class)
		if err != nil {
			return "", err
		}
		result = append(result, ch)
	}

	// Fill the remainder from the full charset.
	full := lowerAlpha + upperAlpha + digits
	if symbols {
		full += specialChars
	}
	for len(result) < length {
		ch, err := randomChar(full)
		if err != nil {
			return "", err
		}
		result = append(result, ch)
	}

	// Shuffle so the mandatory characters are not always at the front.
	if err := shuffleCrypto(result); err != nil {
		return "", err
	}
	return string(result), nil
}

// mandatoryClasses returns the character sets that must each contribute at
// least one character to satisfy typical website password policies.
func mandatoryClasses(symbols bool) []string {
	classes := []string{lowerAlpha, upperAlpha, digits}
	if symbols {
		classes = append(classes, specialChars)
	}
	return classes
}

// randomChar picks a single character from charset using crypto/rand.
// big.Int is used instead of a byte mask to avoid modulo bias.
func randomChar(charset string) (byte, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
	if err != nil {
		return 0, fmt.Errorf("rand: %w", err)
	}
	return charset[n.Int64()], nil
}

// shuffleCrypto performs a Fisher-Yates shuffle using crypto/rand so that
// the positions of mandatory seed characters are unpredictable.
func shuffleCrypto(b []byte) error {
	for i := len(b) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return fmt.Errorf("rand: %w", err)
		}
		j := n.Int64()
		b[i], b[j] = b[j], b[i]
	}
	return nil
}
