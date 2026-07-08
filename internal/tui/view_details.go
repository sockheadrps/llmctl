package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderVRAMHeader renders a compact used/total VRAM bar for the "Running"
// header line. Returns "" when nvidia-smi isn't available or hasn't
// reported anything yet.
func (m Model) renderVRAMHeader() string {
	if !m.gpuAvailable || m.gpuUsage.TotalMiB <= 0 {
		return ""
	}

	const barWidth = 10
	total := float64(m.gpuUsage.TotalMiB)
	frac := float64(m.gpuUsage.UsedMiB) / total

	// RPC server VRAM segment shown at the start of the bar in amber.
	var rpcBlocks int
	if m.cfg.RPCEnabled && m.rpcServerAlive && m.rpcServerState.PID > 0 {
		if rpcMiB, ok := m.gpuByPID[m.rpcServerState.PID]; ok && rpcMiB > 0 {
			rpcBlocks = int(float64(rpcMiB) / total * barWidth)
			if rpcBlocks > barWidth {
				rpcBlocks = barWidth
			}
		}
	}

	filled := int(frac * barWidth)
	if filled > barWidth {
		filled = barWidth
	}
	llamaBlocks := filled - rpcBlocks
	if llamaBlocks < 0 {
		llamaBlocks = 0
	}
	emptyBlocks := barWidth - rpcBlocks - llamaBlocks

	llamaStyle := runningStyle
	switch {
	case frac >= 0.9:
		llamaStyle = downStyle
	case frac >= 0.7:
		llamaStyle = loadingStyle
	}

	bar := loadingStyle.Render(strings.Repeat("█", rpcBlocks)) +
		llamaStyle.Render(strings.Repeat("█", llamaBlocks)) +
		strings.Repeat("░", emptyBlocks)

	usedGB := float64(m.gpuUsage.UsedMiB) / 1024
	totalGB := total / 1024
	return fmt.Sprintf("%s %s", bar, profileStyle.Render(fmt.Sprintf("%.1f/%.1fG", usedGB, totalGB)))
}

// renderModelPreview shows a collapsed model's profiles as a quick preview
// — port and a couple of key settings each — so you can see what's there
// without expanding it. Enter expands the model into the tree for the full
// per-profile Details view.
func (m Model) renderModelPreview(modelKey string) string {
	var b strings.Builder

	mdl, ok := m.cfg.Models[modelKey]
	if !ok {
		return b.String()
	}

	fmt.Fprintf(&b, "%s\n\n", profileStyle.Render(modelSourceLine(mdl)))

	profileKeys := make([]string, 0, len(mdl.Profiles))
	for pk := range mdl.Profiles {
		profileKeys = append(profileKeys, pk)
	}
	sort.Strings(profileKeys)

	b.WriteString(modelStyle.Render("Profiles:"))
	b.WriteString("\n")
	for _, pk := range profileKeys {
		p := mdl.Profiles[pk]
		text := fmt.Sprintf("%-16s :%d", p.Name, p.Port)
		if p.Temp != nil {
			text += fmt.Sprintf("  temp %.2g", *p.Temp)
		}
		b.WriteString(detailMutedStyle.Render("• " + text))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter to expand and select a profile"))
	return b.String()
}

type detailPair struct {
	label string
	value string
}

func formatDetailPairs(pairs []detailPair, width int) []string {
	if width <= 0 {
		width = 36
	}

	if width < 56 {
		lines := make([]string, 0, len(pairs))
		for _, pair := range pairs {
			lines = append(lines, fmt.Sprintf("%s: %s", pair.label, pair.value))
		}
		return lines
	}

	lines := make([]string, 0, (len(pairs)+1)/2)
	for i := 0; i < len(pairs); i += 2 {
		left := fmt.Sprintf("%s: %s", pairs[i].label, pairs[i].value)
		if i+1 >= len(pairs) {
			lines = append(lines, left)
			continue
		}
		right := fmt.Sprintf("%s: %s", pairs[i+1].label, pairs[i+1].value)
		lines = append(lines, left+"    "+right)
	}
	return lines
}

// renderDetails shows the settings for the currently selected profile,
// including the backing model's on-disk file size and any notes. The
// header names whatever's actually focused instead of a static "Details"
// label, since that's more useful at a glance.
func (m Model) renderDetails(width int) string {
	var b strings.Builder

	if m.leftMode == modeNetwork {
		return m.renderNetworkDetails()
	}

	// Still at the outer tab bar — nothing's selected yet within a tab,
	// so explain what arrowing down into it will show instead of an empty
	// "(select a profile...)" placeholder that doesn't fit Recents/Settings.
	if m.focus == focusTabs {
		b.WriteString(modelStyle.Render(m.tabTitle(m.leftMode)))
		b.WriteString("\n\n")
		b.WriteString(profileStyle.Render(m.tabInstructions(m.leftMode)))
		return b.String()
	}

	r, ok := m.currentRow()
	if !ok {
		b.WriteString(modelStyle.Render("Details"))
		b.WriteString("\n\n")
		b.WriteString(profileStyle.Render("(select a profile to see details)"))
		return b.String()
	}

	if r.kind == rowModel {
		mdl := m.cfg.Models[r.modelKey]
		b.WriteString(modelStyle.Render(mdl.Name))
		b.WriteString("\n")
		b.WriteString(m.renderModelPreview(r.modelKey))
		return b.String()
	}

	if r.kind == rowSettingsCategory {
		return m.renderSettingsDetail(r.modelKey)
	}

	if r.kind != rowProfile {
		b.WriteString(modelStyle.Render("Details"))
		b.WriteString("\n\n")
		b.WriteString(profileStyle.Render("(select a profile to see details)"))
		return b.String()
	}

	mdl, ok := m.cfg.Models[r.modelKey]
	if !ok {
		return b.String()
	}
	p, ok := mdl.Profiles[r.profileKey]
	if !ok {
		return b.String()
	}

	// Keep profile details compact so the preview doesn't expand the pane
	// enough to push the UI off-screen when a model is selected.
	fmt.Fprintf(&b, "%s\n", modelStyle.Render(mdl.Name+" / "+p.Name))
	fmt.Fprintf(&b, "%s\n", detailMutedStyle.Render(modelSourceLine(mdl)))
	if p.Notes != "" {
		b.WriteString(profileStyle.Render(p.Notes))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	dash := func(s string) string {
		if s == "" {
			return "-"
		}
		return s
	}

	boolDash := func(v *bool) string {
		if v == nil {
			return "-"
		}
		if *v {
			return "true"
		}
		return "false"
	}

	sections := []struct {
		name  string
		pairs []detailPair
	}{
		{
			name: "Profile",
			pairs: []detailPair{
				{label: "Port", value: fmt.Sprint(p.Port)},
				{label: "Ctx Size", value: dash(intOrEmpty(p.CtxSize))},
			},
		},
		{
			name: "Sampling",
			pairs: []detailPair{
				{label: "Temp", value: dash(floatPtrOrEmpty(p.Temp))},
				{label: "Top P", value: dash(floatPtrOrEmpty(p.TopP))},
				{label: "Top K", value: dash(intPtrOrEmpty(p.TopK))},
				{label: "Min P", value: dash(floatPtrOrEmpty(p.MinP))},
				{label: "Presence Pen", value: dash(floatPtrOrEmpty(p.PresencePenalty))},
				{label: "Repeat Pen", value: dash(floatPtrOrEmpty(p.RepetitionPenalty))},
				{label: "Freq Pen", value: dash(floatPtrOrEmpty(p.FrequencyPenalty))},
				{label: "Seed", value: dash(intPtrOrEmpty(p.Seed))},
			},
		},
		{
			name: "Runtime",
			pairs: []detailPair{
				{label: "Flash Attn", value: fmt.Sprint(p.FlashAttn)},
				{label: "GPU Layers", value: fmt.Sprint(p.GPULayers)},
				{label: "MMap", value: boolDash(p.MMap)},
				{label: "KV Offload", value: boolDash(p.KVOffload)},
				{label: "Parallel", value: dash(intPtrOrEmpty(p.Parallel))},
				{label: "Cont Batching", value: boolDash(p.ContBatching)},
			},
		},
		{
			name: "Cache",
			pairs: []detailPair{
				{label: "Cache K", value: dash(p.CacheTypeK)},
				{label: "Cache V", value: dash(p.CacheTypeV)},
				{label: "Cache Prompt", value: boolDash(p.CachePrompt)},
				{label: "Cache RAM", value: dash(intPtrOrEmpty(p.CacheRAM))},
			},
		},
		{
			name: "Reasoning",
			pairs: []detailPair{
				{label: "Reasoning", value: dash(p.Reasoning)},
				{label: "Budget", value: dash(intPtrOrEmpty(p.ReasoningBudget))},
				{label: "Format", value: dash(p.ReasoningFormat)},
			},
		},
	}

	// Always show all pairs — dash("-") for unset so the full picture is visible.

	if width >= 70 {
		leftSections := []struct {
			name  string
			pairs []detailPair
		}{sections[0], sections[1]}
		rightSections := []struct {
			name  string
			pairs []detailPair
		}{sections[2], sections[3], sections[4]}

		columnWidth := (width - 3) / 2
		if columnWidth < 24 {
			columnWidth = width
		}

		formatSection := func(section struct {
			name  string
			pairs []detailPair
		}) string {
			var sectionBuilder strings.Builder
			sectionBuilder.WriteString(modelStyle.Render(section.name))
			sectionBuilder.WriteString("\n")
			for _, line := range formatDetailPairs(section.pairs, columnWidth) {
				sectionBuilder.WriteString(profileStyle.Render(line))
				sectionBuilder.WriteString("\n")
			}
			return sectionBuilder.String()
		}

		leftColumn := strings.Builder{}
		for _, section := range leftSections {
			if section.name != "" {
				if leftColumn.Len() > 0 {
					leftColumn.WriteString("\n")
				}
				leftColumn.WriteString(formatSection(section))
			}
		}

		rightColumn := strings.Builder{}
		for _, section := range rightSections {
			if section.name != "" {
				if rightColumn.Len() > 0 {
					rightColumn.WriteString("\n")
				}
				rightColumn.WriteString(formatSection(section))
			}
		}

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(columnWidth).Render(leftColumn.String()),
			lipgloss.NewStyle().Width(columnWidth).Render(rightColumn.String()),
		))
		b.WriteString("\n")
	} else {
		for _, section := range sections {
			if len(section.pairs) == 0 {
				continue
			}
			if section.name != "" {
				b.WriteString("\n")
				b.WriteString(modelStyle.Render(section.name))
				b.WriteString("\n")
			}
			for _, line := range formatDetailPairs(section.pairs, width) {
				b.WriteString(profileStyle.Render(line))
				b.WriteString("\n")
			}
		}
	}

	extraArgs := dash(strings.Join(p.ExtraArgs, " "))
	if extraArgs != "-" {
		b.WriteString("\n")
		b.WriteString(modelStyle.Render("Extra Args"))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Width(width).Render(profileStyle.Render(extraArgs)))
	} else {
		b.WriteString("\n")
		b.WriteString(modelStyle.Render("Extra Args"))
		b.WriteString("\n")
		b.WriteString(profileStyle.Render("-"))
	}

	return b.String()
}
