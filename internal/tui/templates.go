package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type profileTemplate struct {
	name      string
	desc      string
	overrides map[int]string
}

var profileTemplates = []profileTemplate{
	{
		name: "Blank",
		desc: "Start with default values",
	},
	{
		name: "Fast Inference",
		desc: "Low context, high parallelism, flash attention on",
		overrides: map[int]string{
			fieldCtxSize:       "2048",
			fieldParallelSlots: "4",
			fieldBatchSize:     "512",
		},
	},
	{
		name: "High Quality",
		desc: "Large context, conservative sampling",
		overrides: map[int]string{
			fieldCtxSize: "16384",
			fieldTemp:    "0.5",
			fieldTopP:    "0.9",
		},
	},
	{
		name: "Coding",
		desc: "16k context, quantized KV cache — fast generation without maxing VRAM",
		overrides: map[int]string{
			fieldCtxSize:   "16384",
			fieldCacheK:    "q8_0",
			fieldCacheV:    "q8_0",
			fieldTemp:      "0.3",
			fieldTopP:      "0.9",
			fieldTopK:      "20",
		},
	},
	{
		name: "Low VRAM",
		desc: "Small context, CPU offload, single slot",
		overrides: map[int]string{
			fieldCtxSize:       "4096",
			fieldGPULayers:     "0",
			fieldMMap:          "true",
			fieldParallelSlots: "1",
		},
	},
}

type templatePickerState struct {
	modelKey string
	cursor   int
}

func (m Model) openTemplatePicker(modelKey string) (tea.Model, tea.Cmd) {
	m.templatePicker = templatePickerState{modelKey: modelKey, cursor: 0}
	m.screen = screenProfileTemplate
	return m, nil
}

func (m Model) updateTemplatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenMain
		return m, nil
	case "up", "k":
		if m.templatePicker.cursor > 0 {
			m.templatePicker.cursor--
		}
	case "down", "j":
		if m.templatePicker.cursor < len(profileTemplates)-1 {
			m.templatePicker.cursor++
		}
	case "enter":
		t := profileTemplates[m.templatePicker.cursor]
		return m.openForm(m.templatePicker.modelKey, t.overrides)
	}
	return m, nil
}

func (m Model) viewTemplatePicker() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("New Profile — Choose a Template"))
	b.WriteString("\n\n")

	for i, t := range profileTemplates {
		cursor := "  "
		nameStyle := profileStyle
		if i == m.templatePicker.cursor {
			cursor = cursorStyle.Render("> ")
			nameStyle = selectedProfileStyle
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, nameStyle.Render(t.name)))
		b.WriteString(fmt.Sprintf("   %s\n", detailMutedStyle.Render(t.desc)))
		if i < len(profileTemplates)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/k ↓/j move · enter select · esc cancel"))
	return b.String()
}
