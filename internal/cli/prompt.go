package cli

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// readPassword reads a password from the terminal without echoing.
func readPassword(prompt string) ([]byte, error) {
	fmt.Fprint(os.Stderr, prompt)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // newline after silent input
	if err != nil {
		return nil, fmt.Errorf("read password: %w", err)
	}
	return pw, nil
}

// readPasswordConfirm reads a password twice and checks they match.
func readPasswordConfirm(prompt string) ([]byte, error) {
	pw, err := readPassword(prompt)
	if err != nil {
		return nil, err
	}
	confirm, err := readPassword("Confirm master password: ")
	if err != nil {
		return nil, err
	}
	if string(pw) != string(confirm) {
		return nil, fmt.Errorf("passwords do not match")
	}
	return pw, nil
}

// confirm asks a yes/no question and returns true if the user answers yes.
func confirm(prompt string) bool {
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	var resp string
	fmt.Scanln(&resp)
	return strings.ToLower(strings.TrimSpace(resp)) == "y"
}
