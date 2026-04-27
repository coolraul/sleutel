package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mms/sleutel/internal/cli"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine home directory:", err)
		os.Exit(1)
	}
	defaultVault := filepath.Join(home, ".sleutel", "vault.slkv")

	root := cli.NewRootCmd(defaultVault)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
