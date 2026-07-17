package tui

import (
	"fmt"
	"os"
	"sync"
	"time"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var debugTimingEnabled = sync.OnceValue(func() bool {
	v := os.Getenv("LLMCTL_DEBUG_TIMINGS")
	return v != "" && v != "0" && !strings.EqualFold(v, "false")
})

func debugTimingf(format string, args ...any) {
	if !debugTimingEnabled() {
		return
	}
	fmt.Fprintf(os.Stderr, "[llmctl timing] "+format+"\n", args...)
}

func timedCmd(name string, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		start := time.Now()
		msg := cmd()
		debugTimingf("%s took %s", name, time.Since(start))
		return msg
	}
}
