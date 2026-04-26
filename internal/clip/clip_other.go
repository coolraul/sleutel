//go:build !darwin && !linux

package clip

import "fmt"

func writeAll(_ string) error {
	return fmt.Errorf("clipboard not supported on this platform")
}

func readAll() (string, error) {
	return "", fmt.Errorf("clipboard not supported on this platform")
}
