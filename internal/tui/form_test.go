package tui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
	tui_form "github.com/sockheadrps/llmctl/internal/tui/form"
)

func TestFormLabelsIncludeExtendedProfileOptions(t *testing.T) {
	contains := func(labels []string, want string) bool {
		for _, label := range labels {
			if label == want {
				return true
			}
		}
		return false
	}

	for _, want := range []string{"Host", "Batch Size", "Parallel Slots", "Reasoning"} {
		if !contains(formLabels, want) {
			t.Fatalf("expected form labels to include %q", want)
		}
	}
}

func TestFieldDescriptionForKnownParam(t *testing.T) {
	if got := formFieldDescription(fieldHost); got == "" {
		t.Fatal("expected a description for host")
	}
	if got := formFieldDescription(fieldReasoning); got == "" {
		t.Fatal("expected a description for reasoning")
	}
}

func testEditFormModel(t *testing.T) Model {
	t.Helper()
	m := Model{
		cfg: &config.Config{Models: map[string]models.Model{
			"model": {
				Name: "Model",
				Profiles: map[string]models.Profile{
					"profile": {Name: "Profile", Port: 8080, FlashAttn: true},
				},
			},
		}},
	}
	next, _ := m.openEditForm("model", "profile")
	return next.(Model)
}

func TestEditFormEscExitsImmediatelyWhenUnchanged(t *testing.T) {
	m := testEditFormModel(t)

	next, _ := m.updateForm(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(Model)

	if got.screen != screenMain {
		t.Fatalf("expected unchanged edit form esc to return to main, got screen %v", got.screen)
	}
}

func TestEditFormEscPromptsWhenDirty(t *testing.T) {
	m := testEditFormModel(t)
	m.form.fields[fieldAlias].input.SetValue("changed")

	next, _ := m.updateForm(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(Model)

	if got.screen != screenFormExitConfirm {
		t.Fatalf("expected dirty edit form esc to open exit confirm, got screen %v", got.screen)
	}
	if got.formExit.selected != formExitDiscard {
		t.Fatalf("expected discard option selected by default, got %v", got.formExit.selected)
	}
}

func TestFormExitDiscardReturnsToMain(t *testing.T) {
	m := testEditFormModel(t)
	m.screen = screenFormExitConfirm
	m.formExit.selected = formExitDiscard

	next, _ := m.updateFormExit(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(Model)

	if got.screen != screenMain {
		t.Fatalf("expected discard to return to main, got screen %v", got.screen)
	}
}

func TestFormPaneHeightLeavesRoomForHotkeys(t *testing.T) {
	m := Model{height: 40}
	if got := tui_form.FormPaneHeight(m.height); got != 34 {
		t.Fatalf("expected pane inner height 34 for a 40-line terminal, got %d", got)
	}
	if got := tui_form.FormVisibleRows(tui_form.FormPaneHeight(m.height)); got != 33 {
		t.Fatalf("expected visible rows to leave room for in-pane scroll hint, got %d", got)
	}
}

func TestFormPaneWidthsShrinkParametersBeforeDetails(t *testing.T) {
	m := Model{width: 82}
	left, details := tui_form.FormPaneWidths(m.width)
	if left+details > 78 {
		t.Fatalf("expected panes to fit available width, got left %d details %d", left, details)
	}
	if details < formMinDetailsWidth {
		t.Fatalf("expected details width to keep minimum %d, got %d", formMinDetailsWidth, details)
	}
	if left < formMinLeftWidth {
		t.Fatalf("expected left width to keep minimum %d, got %d", formMinLeftWidth, left)
	}
}

func TestFormDescriptionWrapsToInnerPaneWidth(t *testing.T) {
	m := Model{width: 82}
	_, details := tui_form.FormPaneWidths(m.width)
	lines := m.formDescriptionLines(details)
	for _, line := range lines {
		if len(line) > tui_form.FormDescriptionTextWidth(details) {
			t.Fatalf("expected %q to fit inner details width %d", line, tui_form.FormDescriptionTextWidth(details))
		}
	}
}

func TestTruncateTextPreventsDetailTitleWrapping(t *testing.T) {
	got := tui_form.TruncateText("Extra Args (space-separated)", 12)
	if len(got) != 12 {
		t.Fatalf("expected truncated title to match width, got %q len %d", got, len(got))
	}
}

func TestEditFormViewDoesNotExceedViewportWithLongParameterValue(t *testing.T) {
	m := testEditFormModel(t)
	m.width = 82
	m.height = 24
	m.form.focus = fieldExtraArgs
	m.form.fields[fieldExtraArgs].input.Focus()
	m.form.fields[fieldExtraArgs].input.SetValue(strings.Repeat("--very-long-argument ", 12))

	got := m.viewForm()
	if lines := strings.Count(got, "\n") + 1; lines > m.height {
		t.Fatalf("expected form view to stay within %d lines, got %d", m.height, lines)
	}
}

func TestDescriptionWindowKeepsFixedLineCount(t *testing.T) {
	got := tui_form.DescriptionWindow([]string{"one", "two", "three", "four"}, 3, 1)
	want := []string{"two", "three", "four"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}

	got = tui_form.DescriptionWindow([]string{"one"}, 3, 0)
	want = []string{"one", "", ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected padded fixed window %v, got %v", want, got)
	}
}

func TestDescriptionScrollAdvancesDownThenBackUp(t *testing.T) {
	f := formState{}
	f.descScroll, f.descDir, f.descPause = tui_form.AdvanceDescriptionScroll(f.descScroll, f.descDir, f.descPause, 5, 3, scrollPauseTicks)
	if f.descScroll != 1 || f.descDir != 1 {
		t.Fatalf("expected scroll 1 dir 1, got scroll %d dir %d", f.descScroll, f.descDir)
	}

	f.descScroll, f.descDir, f.descPause = tui_form.AdvanceDescriptionScroll(f.descScroll, f.descDir, f.descPause, 5, 3, scrollPauseTicks)
	if f.descScroll != 2 || f.descDir != -1 || f.descPause != scrollPauseTicks {
		t.Fatalf("expected scroll 2 dir -1 pause %d at bottom, got scroll %d dir %d pause %d", scrollPauseTicks, f.descScroll, f.descDir, f.descPause)
	}

	f.descScroll, f.descDir, f.descPause = tui_form.AdvanceDescriptionScroll(f.descScroll, f.descDir, f.descPause, 5, 3, scrollPauseTicks)
	if f.descScroll != 2 || f.descDir != -1 || f.descPause != scrollPauseTicks-1 {
		t.Fatalf("expected bottom pause to hold, got scroll %d dir %d pause %d", f.descScroll, f.descDir, f.descPause)
	}

	f.descScroll, f.descDir, f.descPause = tui_form.AdvanceDescriptionScroll(f.descScroll, f.descDir, f.descPause, 5, 3, scrollPauseTicks)
	f.descScroll, f.descDir, f.descPause = tui_form.AdvanceDescriptionScroll(f.descScroll, f.descDir, f.descPause, 5, 3, scrollPauseTicks)
	if f.descScroll != 1 || f.descDir != -1 || f.descPause != 0 {
		t.Fatalf("expected scroll 1 dir -1 after bottom pause, got scroll %d dir %d pause %d", f.descScroll, f.descDir, f.descPause)
	}

	f.descScroll, f.descDir, f.descPause = tui_form.AdvanceDescriptionScroll(f.descScroll, f.descDir, f.descPause, 5, 3, scrollPauseTicks)
	if f.descScroll != 0 || f.descDir != 1 || f.descPause != scrollPauseTicks {
		t.Fatalf("expected scroll 0 dir 1 pause %d at top, got scroll %d dir %d pause %d", scrollPauseTicks, f.descScroll, f.descDir, f.descPause)
	}
}

func TestDescriptionScrollResetWaitsBeforeMoving(t *testing.T) {
	f := formState{descScroll: 2, descDir: -1}
	f.descScroll, f.descDir, f.descPause = tui_form.ResetDescriptionScroll(scrollPauseTicks)

	f.descScroll, f.descDir, f.descPause = tui_form.AdvanceDescriptionScroll(f.descScroll, f.descDir, f.descPause, 5, 3, scrollPauseTicks)
	if f.descScroll != 0 || f.descDir != 1 || f.descPause != scrollPauseTicks-1 {
		t.Fatalf("expected reset scroll to wait before moving, got scroll %d dir %d pause %d", f.descScroll, f.descDir, f.descPause)
	}
}

func TestAdvanceAutoScrollResetsWhenContentFits(t *testing.T) {
	offset, dir, pause := advanceAutoScroll(3, -1, 1, 2, 5)
	if offset != 0 || dir != 1 || pause != 0 {
		t.Fatalf("expected reset scroll state, got offset %d dir %d pause %d", offset, dir, pause)
	}
}

func TestScrollTickIntervalIsFasterThanRefreshTick(t *testing.T) {
	if scrollTickInterval != 500*time.Millisecond {
		t.Fatalf("expected 500ms scroll interval, got %s", scrollTickInterval)
	}
	if scrollTickInterval >= 2*time.Second {
		t.Fatalf("expected scroll tick to be faster than refresh tick, got %s", scrollTickInterval)
	}
	if scrollPauseTicks != 2 {
		t.Fatalf("expected two 500ms ticks for a 1s pause, got %d", scrollPauseTicks)
	}
}

func TestDescriptionScrollTopPauseThenRepeats(t *testing.T) {
	f := formState{descScroll: 1, descDir: -1}
	f.descScroll, f.descDir, f.descPause = tui_form.AdvanceDescriptionScroll(f.descScroll, f.descDir, f.descPause, 5, 3, scrollPauseTicks)
	if f.descScroll != 0 || f.descDir != 1 || f.descPause != scrollPauseTicks {
		t.Fatalf("expected top pause, got scroll %d dir %d pause %d", f.descScroll, f.descDir, f.descPause)
	}
	f.descScroll, f.descDir, f.descPause = tui_form.AdvanceDescriptionScroll(f.descScroll, f.descDir, f.descPause, 5, 3, scrollPauseTicks)
	f.descScroll, f.descDir, f.descPause = tui_form.AdvanceDescriptionScroll(f.descScroll, f.descDir, f.descPause, 5, 3, scrollPauseTicks)
	f.descScroll, f.descDir, f.descPause = tui_form.AdvanceDescriptionScroll(f.descScroll, f.descDir, f.descPause, 5, 3, scrollPauseTicks)
	if f.descScroll != 1 || f.descDir != 1 || f.descPause != 0 {
		t.Fatalf("expected repeat downward after top pause, got scroll %d dir %d pause %d", f.descScroll, f.descDir, f.descPause)
	}
}

// --- Flag override tests ---

func testNewFormModel(t *testing.T) Model {
	t.Helper()
	m := Model{
		cfg: &config.Config{Models: map[string]models.Model{
			"model": {
				Name:     "Model",
				Profiles: map[string]models.Profile{},
			},
		}},
	}
	next, _ := m.openForm("model", nil)
	return next.(Model)
}

func TestFieldDefaultFlagReturnsCorrectFlags(t *testing.T) {
	cases := []struct {
		field int
		want  string
	}{
		{fieldPort, "--port"},
		{fieldCtxSize, "--ctx-size"},
		{fieldTemp, "--temp"},
		{fieldGPULayers, "--n-gpu-layers"},
		{fieldNotes, ""},
		{fieldExtraArgs, ""},
		{fieldKey, ""},
	}
	for _, tc := range cases {
		if got := fieldDefaultFlag(tc.field); got != tc.want {
			t.Errorf("fieldDefaultFlag(%d) = %q, want %q", tc.field, got, tc.want)
		}
	}
	// Flash Attention toggle sits at len(formLabels) in the form focus index.
	if got := tui_form.FocusedFlag(len(formLabels), len(formLabels)); got != "--flash-attn" {
		t.Errorf("focusedFlag for flash attn toggle = %q, want --flash-attn", got)
	}
}

func TestFlashAttentionIsRenderedAndNavigable(t *testing.T) {
	m := testNewFormModel(t)
	m.width = 100
	m.height = 60

	got := m.viewForm()
	if !strings.Contains(got, "Flash Attention") {
		t.Fatal("expected Flash Attention row to be rendered in the form")
	}

	order := tui_form.FormNavOrder(len(m.form.fields))
	found := false
	for _, idx := range order {
		if idx == len(m.form.fields) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected Flash Attention toggle to be present in navigation order")
	}
}

func TestRightArrowEntersFlagFocusOnFieldWithFlag(t *testing.T) {
	m := testNewFormModel(t)
	m.form.focus = fieldPort
	if m.form.focus < len(m.form.fields) {
		m.form.fields[m.form.focus].input.Focus()
	}

	next, _ := m.updateForm(tea.KeyMsg{Type: tea.KeyRight})
	got := next.(Model)

	if !got.form.flagFocus {
		t.Fatal("expected flagFocus=true after right arrow on a field with a flag")
	}
}

func TestRightArrowDoesNotEnterFlagFocusOnFieldWithNoFlag(t *testing.T) {
	m := testNewFormModel(t)
	m.form.focus = fieldNotes
	m.form.fields[m.form.focus].input.Focus()

	next, _ := m.updateForm(tea.KeyMsg{Type: tea.KeyRight})
	got := next.(Model)

	if got.form.flagFocus {
		t.Fatal("expected no flagFocus after right arrow on Notes (no CLI flag)")
	}
}

func TestEnterBeginsEditingOnRpcEnabledField(t *testing.T) {
	m := testNewFormModel(t)
	m.form.focus = fieldRPCEnabled

	next, _ := m.updateForm(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(Model)

	if got.form.navigating {
		t.Fatal("expected enter on RPC Enabled to switch into edit mode")
	}
}

func TestCPUOnlyKeepsGpuLayersVisibleInView(t *testing.T) {
	m := testNewFormModel(t)
	m.width = 100
	m.height = 60
	m.form.cpuOnly = true

	got := m.viewForm()
	if !strings.Contains(got, "GPU Layers") {
		t.Fatal("expected GPU Layers row to remain visible when CPU Only is enabled")
	}
}

func TestNavigateFromRepeatLastNToCPUOnly(t *testing.T) {
	m := testNewFormModel(t)
	m.form.focus = fieldRepeatLastN

	next, _ := m.updateForm(tea.KeyMsg{Type: tea.KeyDown})
	got := next.(Model)

	if got.form.focus != fieldGPULayers {
		t.Fatalf("expected focus to move to GPU Layers, got %d", got.form.focus)
	}
}

func TestLeftArrowExitsFlagFocusAndCommitsOverride(t *testing.T) {
	m := testNewFormModel(t)
	m.form.focus = fieldPort
	m.form.flagFocus = true
	m.form.flagInput.Focus()
	m.form.flagInput.SetValue("--p")

	next, _ := m.updateForm(tea.KeyMsg{Type: tea.KeyLeft})
	got := next.(Model)

	if got.form.flagFocus {
		t.Fatal("expected flagFocus=false after left arrow")
	}
	if override, ok := got.form.flagOverrides["--port"]; !ok || override != "--p" {
		t.Fatalf("expected override --port→--p committed, got flagOverrides=%v", got.form.flagOverrides)
	}
}

func TestEnterInFlagFocusCommitsAndExits(t *testing.T) {
	m := testNewFormModel(t)
	m.form.focus = fieldCtxSize
	m.form.flagFocus = true
	m.form.flagInput.Focus()
	m.form.flagInput.SetValue("--c")

	next, _ := m.updateForm(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(Model)

	if got.form.flagFocus {
		t.Fatal("expected flagFocus=false after enter in flag focus")
	}
	if override, ok := got.form.flagOverrides["--ctx-size"]; !ok || override != "--c" {
		t.Fatalf("expected override --ctx-size→--c, got %v", got.form.flagOverrides)
	}
}

func TestFlagOverrideRestoredToDefaultWhenCleared(t *testing.T) {
	m := testNewFormModel(t)
	m.form.focus = fieldPort
	m.form.flagOverrides["--port"] = "--p"
	m.form.flagFocus = true
	m.form.flagInput.SetValue("--port") // user typed back the default

	next, _ := m.updateForm(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(Model)

	if _, ok := got.form.flagOverrides["--port"]; ok {
		t.Fatal("expected default-value override to be removed from map")
	}
}

func TestFlagOverrideMakesDirty(t *testing.T) {
	m := testEditFormModel(t)
	values := make([]string, len(m.form.fields))
	for i := range m.form.fields {
		values[i] = m.form.fields[i].input.Value()
	}
	if tui_form.Dirty(
		m.form.flash, m.form.initialFlash,
		m.form.cpuOnly, m.form.initialCPUOnly,
		m.form.mlock, m.form.initialMLock,
		m.form.rpcClientLayers, m.form.initialRPCClientLayers,
		values, m.form.initial,
		m.form.flagOverrides, m.form.initialFlagOverrides,
	) {
		t.Fatal("expected clean form initially")
	}
	m.form.flagOverrides["--port"] = "--p"
	if !tui_form.Dirty(
		m.form.flash, m.form.initialFlash,
		m.form.cpuOnly, m.form.initialCPUOnly,
		m.form.mlock, m.form.initialMLock,
		m.form.rpcClientLayers, m.form.initialRPCClientLayers,
		values, m.form.initial,
		m.form.flagOverrides, m.form.initialFlagOverrides,
	) {
		t.Fatal("expected dirty form when flag override added")
	}
}

func TestSyncFlagInputShowsCurrentOverrideOrDefault(t *testing.T) {
	m := testNewFormModel(t)
	m.form.focus = fieldGPULayers
	m.form.flagInput = tui_form.SyncFlagInput(m.form.flagInput, m.form.focus, len(formLabels), m.form.flagOverrides)
	if got := m.form.flagInput.Value(); got != "--n-gpu-layers" {
		t.Fatalf("expected default flag --n-gpu-layers, got %q", got)
	}

	m.form.flagOverrides["--n-gpu-layers"] = "--ngl"
	m.form.flagInput = tui_form.SyncFlagInput(m.form.flagInput, m.form.focus, len(formLabels), m.form.flagOverrides)
	if got := m.form.flagInput.Value(); got != "--ngl" {
		t.Fatalf("expected override --ngl, got %q", got)
	}
}

func TestParseProfileArgsMapsExportedFlags(t *testing.T) {
	got, extra := tui_form.ParseProfileArgs(`--port 8123 --host 0.0.0.0 --flash-attn on --no-mmap --reasoning auto --cache-type-k q8_0 --verbose --threads 8`, len(formLabels), fieldDefaultFlag, len(formLabels), fieldExtraArgs)

	if got[fieldPort] != "8123" {
		t.Fatalf("expected port 8123, got %q", got[fieldPort])
	}
	if got[fieldHost] != "0.0.0.0" {
		t.Fatalf("expected host 0.0.0.0, got %q", got[fieldHost])
	}
	if got[len(formLabels)] != "on" {
		t.Fatalf("expected flash attn on, got %q", got[len(formLabels)])
	}
	if got[fieldMMap] != "false" {
		t.Fatalf("expected mmap false, got %q", got[fieldMMap])
	}
	if got[fieldReasoning] != "auto" {
		t.Fatalf("expected reasoning auto, got %q", got[fieldReasoning])
	}
	if got[fieldCacheK] != "q8_0" {
		t.Fatalf("expected cache type k q8_0, got %q", got[fieldCacheK])
	}
	if got[fieldExtraArgs] != "--verbose --threads 8" {
		t.Fatalf("expected extra args preserved, got %q", got[fieldExtraArgs])
	}
	if !reflect.DeepEqual(extra, []string{"--verbose", "--threads", "8"}) {
		t.Fatalf("expected extra token list preserved, got %v", extra)
	}
}

func TestParseProfileArgsSkipsBinaryAndModelSource(t *testing.T) {
	got, extra := tui_form.ParseProfileArgs(`llama-server --model /models/llama.gguf -hf ignored/repo --port 8123 --verbose`, len(formLabels), fieldDefaultFlag, len(formLabels), fieldExtraArgs)

	if got[fieldPort] != "8123" {
		t.Fatalf("expected port 8123, got %q", got[fieldPort])
	}
	if got[fieldExtraArgs] != "--verbose" {
		t.Fatalf("expected only real extra args, got %q", got[fieldExtraArgs])
	}
	if !reflect.DeepEqual(extra, []string{"--verbose"}) {
		t.Fatalf("expected model source tokens to be ignored, got %v", extra)
	}
}

func TestImportModalAppliesArgsToForm(t *testing.T) {
	m := testNewFormModel(t)
	m.form.importEditing = true
	m.form.importErr = ""
	m.form.importInput = tui_form.OpenImportModal(tui_form.BuildImportInput())
	m.form.importInput.SetValue(`--port 8123 --host 0.0.0.0 --flash-attn off --cache-type-v q4_0 --verbose`)

	next, _ := m.updateForm(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(Model)

	if got.form.importEditing {
		t.Fatal("expected import modal to close after applying args")
	}
	if got.form.fields[fieldPort].input.Value() != "8123" {
		t.Fatalf("expected imported port 8123, got %q", got.form.fields[fieldPort].input.Value())
	}
	if got.form.fields[fieldHost].input.Value() != "0.0.0.0" {
		t.Fatalf("expected imported host 0.0.0.0, got %q", got.form.fields[fieldHost].input.Value())
	}
	if got.form.flash {
		t.Fatal("expected flash attn to be off after import")
	}
	if got.form.fields[fieldCacheV].input.Value() != "q4_0" {
		t.Fatalf("expected imported cache type v q4_0, got %q", got.form.fields[fieldCacheV].input.Value())
	}
	if got.form.fields[fieldExtraArgs].input.Value() != "--verbose" {
		t.Fatalf("expected imported extra args, got %q", got.form.fields[fieldExtraArgs].input.Value())
	}
}

func TestImportShortcutOpensModal(t *testing.T) {
	m := testNewFormModel(t)

	next, _ := m.updateForm(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	got := next.(Model)

	if !got.form.importEditing {
		t.Fatal("expected x to open import modal")
	}
}

func TestConfirmModalCanOpenExportArgs(t *testing.T) {
	m := Model{
		cfg: &config.Config{Models: map[string]models.Model{
			"model": {
				Name: "Model",
				Profiles: map[string]models.Profile{
					"profile": {Name: "Profile", Port: 8080},
				},
			},
		}},
	}
	next, _ := m.openConfirm(row{kind: rowProfile, modelKey: "model", profileKey: "profile", label: "Profile"})
	got := next.(Model)

	next, _ = got.updateConfirm(tea.KeyMsg{Type: tea.KeyRight})
	got = next.(Model)
	next, _ = got.updateConfirm(tea.KeyMsg{Type: tea.KeyRight})
	got = next.(Model)
	next, _ = got.updateConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	got = next.(Model)

	if got.screen != screenExportArgs {
		t.Fatalf("expected confirm modal export action to open export args screen, got %v", got.screen)
	}
	if got.exportArgs.label != "Profile" {
		t.Fatalf("expected export popup label to be set, got %q", got.exportArgs.label)
	}
}
