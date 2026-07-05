//go:build !windows

package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/aymanbagabas/go-osc52/v2"
)

func writeClipboard(text string) error {
	if runningOverSSH() {
		seq := osc52.New(text)
		if os.Getenv("TMUX") != "" {
			seq = seq.Tmux()
		}
		_, err := fmt.Fprint(os.Stderr, seq.String())
		return err
	}
	return clipboard.WriteAll(text)
}

func runningOverSSH() bool {
	return os.Getenv("SSH_CONNECTION") != "" ||
		os.Getenv("SSH_CLIENT") != "" ||
		strings.TrimSpace(os.Getenv("SSH_TTY")) != ""
}
