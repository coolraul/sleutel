//go:build linux

package clip

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func writeAll(text string) error {
	tool, args := linuxWriteTool()
	if tool == "" {
		return fmt.Errorf("no clipboard tool found (install xclip, xsel, or wl-clipboard)")
	}
	cmd := exec.Command(tool, args...)
	cmd.Stdin = bytes.NewBufferString(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", tool, err)
	}
	return nil
}

func readAll() (string, error) {
	tool, args := linuxReadTool()
	if tool == "" {
		return "", fmt.Errorf("no clipboard tool found (install xclip, xsel, or wl-clipboard)")
	}
	out, err := exec.Command(tool, args...).Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w", tool, err)
	}
	return string(out), nil
}

// linuxWriteTool returns the first available clipboard write command.
// Wayland is preferred when WAYLAND_DISPLAY is set; X11 tools are the fallback.
func linuxWriteTool() (string, []string) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if path, err := exec.LookPath("wl-copy"); err == nil {
			return path, nil
		}
	}
	if path, err := exec.LookPath("xclip"); err == nil {
		return path, []string{"-selection", "clipboard"}
	}
	if path, err := exec.LookPath("xsel"); err == nil {
		return path, []string{"--clipboard", "--input"}
	}
	return "", nil
}

func linuxReadTool() (string, []string) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if path, err := exec.LookPath("wl-paste"); err == nil {
			return path, []string{"--no-newline"}
		}
	}
	if path, err := exec.LookPath("xclip"); err == nil {
		return path, []string{"-selection", "clipboard", "-o"}
	}
	if path, err := exec.LookPath("xsel"); err == nil {
		return path, []string{"--clipboard", "--output"}
	}
	return "", nil
}
