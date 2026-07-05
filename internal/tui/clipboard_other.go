//go:build !windows

package tui

import "github.com/atotto/clipboard"

func writeClipboard(text string) error {
	return clipboard.WriteAll(text)
}
