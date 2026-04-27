package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mms/sleutel/internal/tui"
)

func main() {
	vaultPath := os.Getenv("SLEUTEL_VAULT")
	if vaultPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "cannot determine home directory:", err)
			os.Exit(1)
		}
		vaultPath = filepath.Join(home, ".sleutel", "vault.slkv")
	}

	// Allow override via first positional arg for convenience during development.
	if len(os.Args) > 1 {
		vaultPath = os.Args[1]
	}

	if _, err := os.Stat(vaultPath); err != nil {
		fmt.Fprintf(os.Stderr, "vault not found at %s\nRun 'sleutel init' to create one.\n", vaultPath)
		os.Exit(1)
	}

	p := tea.NewProgram(
		tui.NewApp(vaultPath),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
