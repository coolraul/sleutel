//go:build linux

package clip

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func writeAll(text string) error {
	tool, args, wayland := linuxWriteTool()
	if tool == "" {
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			return fmt.Errorf("no clipboard tool found (install wl-clipboard: sudo apt install wl-clipboard)")
		}
		return fmt.Errorf("no clipboard tool found (install xclip or xsel)")
	}
	cmd := exec.Command(tool, args...)
	cmd.Stdin = bytes.NewBufferString(text)
	if wayland {
		// wl-copy stays running to serve clipboard requests until displaced;
		// Start (not Run) so we don't block the caller.
		return cmd.Start()
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", tool, err)
	}
	return nil
}

func readAll() (string, error) {
	tool, args := linuxReadTool()
	if tool == "" {
		return "", fmt.Errorf("no clipboard tool found")
	}
	out, err := exec.Command(tool, args...).Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w", tool, err)
	}
	return string(out), nil
}

// linuxWriteTool returns the clipboard write command, its args, and whether it
// is a Wayland-native tool (wl-copy). On Wayland sessions, xclip/xsel are not
// used as fallback because the X11↔Wayland clipboard bridge is unreliable.
func linuxWriteTool() (path string, args []string, wayland bool) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if p, err := exec.LookPath("wl-copy"); err == nil {
			return p, nil, true
		}
		return "", nil, false
	}
	if p, err := exec.LookPath("xclip"); err == nil {
		return p, []string{"-selection", "clipboard"}, false
	}
	if p, err := exec.LookPath("xsel"); err == nil {
		return p, []string{"--clipboard", "--input"}, false
	}
	return "", nil, false
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
