//go:build windows

package tui

import (
	"os/exec"
	"strings"
)

func writeClipboard(text string) error {
	cmd := exec.Command("cmd", "/c", "clip")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
