package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sockheadrps/llmctl/internal/util"
)

func pickerSpinnerFrame(step int) string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return frames[step%len(frames)]
}

func (m Model) viewPicker() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Add Model"))
	b.WriteString("\n\n")

	dirs, _ := m.cfg.ResolvedModelsDirs()
	if len(dirs) > 0 {
		b.WriteString(infoStyle.Render("scanned " + strings.Join(dirs, ", ")))
		b.WriteString("\n")
	}
	if len(m.picker.unreadable) > 0 {
		b.WriteString(errorStyle.Render("could not scan: " + strings.Join(m.picker.unreadable, ", ")))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	body := strings.Builder{}
	switch {
	case m.picker.err != nil:
		body.WriteString(errorStyle.Render(m.picker.err.Error()))
	case len(m.picker.files) == 0:
		body.WriteString(profileStyle.Render("no new .gguf files found"))
	default:
		for i, f := range m.picker.files {
			cursor := "  "
			style := profileStyle
			if i == m.picker.cursor {
				cursor = cursorStyle.Render("> ")
				style = selectedProfileStyle
			}
			body.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(f)))
			if meta := pickerFileMetadata(f); meta != "" {
				body.WriteString(fmt.Sprintf("  %s\n", detailMutedStyle.Render(meta)))
			}
		}
	}
	b.WriteString(paneStyle.Render(body.String()))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/k up  ↓/j down  enter import  esc cancel  (manage directories from Settings)"))
	return b.String()
}

var quantPattern = regexp.MustCompile(`(?i)\b(?:q\d(?:_[a-z0-9]+)?|iq\d(?:_[a-z0-9]+)?|f16|bf16|f32)\b`)

func pickerFileMetadata(path string) string {
	parts := []string{}
	if info, err := os.Stat(path); err == nil {
		parts = append(parts, util.FormatBytes(info.Size()))
	}
	if family := inferModelFamily(path); family != "" {
		parts = append(parts, "family "+family)
	}
	if quant := inferQuant(path); quant != "" {
		parts = append(parts, "quant "+strings.ToUpper(quant))
	}
	return strings.Join(parts, " · ")
}

func inferModelFamily(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	for _, sep := range []string{"-", "_", "."} {
		if idx := strings.Index(name, sep); idx > 0 {
			return name[:idx]
		}
	}
	return name
}

func inferQuant(path string) string {
	return quantPattern.FindString(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
}
