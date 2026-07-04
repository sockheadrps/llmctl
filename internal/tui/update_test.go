package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
)

func TestModelCursorTargetsUseTopLevelUntilProfileMode(t *testing.T) {
	rows := []row{
		{kind: rowModel, modelKey: "alpha", label: "Alpha"},
		{kind: rowProfile, modelKey: "alpha", profileKey: "default", label: "Default"},
		{kind: rowAddProfile, modelKey: "alpha", label: "+ New Profile"},
		{kind: rowModel, modelKey: "beta", label: "Beta"},
		{kind: rowProfile, modelKey: "beta", profileKey: "default", label: "Default"},
		{kind: rowAddModel, label: "+ Add Model"},
	}

	got := modelCursorTargets(rows, false, "alpha")
	want := []int{0, 3, 5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestModelCursorTargetsUseExpandedProfilesInProfileMode(t *testing.T) {
	rows := []row{
		{kind: rowModel, modelKey: "alpha", label: "Alpha"},
		{kind: rowProfile, modelKey: "alpha", profileKey: "default", label: "Default"},
		{kind: rowAddProfile, modelKey: "alpha", label: "+ New Profile"},
		{kind: rowModel, modelKey: "beta", label: "Beta"},
		{kind: rowProfile, modelKey: "beta", profileKey: "default", label: "Default"},
		{kind: rowAddModel, label: "+ Add Model"},
	}

	got := modelCursorTargets(rows, true, "alpha")
	want := []int{1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestMoveModelsCursorBrowsesModelsUntilEntered(t *testing.T) {
	m := Model{
		cfg: &config.Config{Models: map[string]models.Model{
			"alpha": {Name: "Alpha", Profiles: map[string]models.Profile{"default": {Name: "Default"}}},
			"beta":  {Name: "Beta", Profiles: map[string]models.Profile{"default": {Name: "Default"}}},
		}},
		focus:            focusLeft,
		leftMode:         modeModels,
		expandedModelKey: "alpha",
	}
	m.rebuildRows()
	m.cursor = indexOfModelRow(m.rows, "alpha")

	next, _ := m.moveModelsCursor(1)
	got := next.(Model)

	if got.rows[got.cursor].kind != rowModel || got.rows[got.cursor].modelKey != "beta" {
		t.Fatalf("expected cursor to move to next model, got row %+v", got.rows[got.cursor])
	}
	if got.expandedModelKey != "beta" {
		t.Fatalf("expected focused model to expand for preview, got %q", got.expandedModelKey)
	}
	if got.modelProfilesMode {
		t.Fatal("expected model browsing to stay out of profile mode")
	}
}

func TestEnterModelEnablesProfileNavigationToAddProfile(t *testing.T) {
	m := Model{
		cfg: &config.Config{Models: map[string]models.Model{
			"alpha": {Name: "Alpha", Profiles: map[string]models.Profile{"default": {Name: "Default"}}},
		}},
		focus:            focusLeft,
		leftMode:         modeModels,
		expandedModelKey: "alpha",
	}
	m.rebuildRows()
	m.cursor = indexOfModelRow(m.rows, "alpha")

	next, _ := m.enterModel("alpha")
	entered := next.(Model)
	if !entered.modelProfilesMode {
		t.Fatal("expected enterModel to switch into profile mode")
	}

	next, _ = entered.moveModelsCursor(1)
	got := next.(Model)

	if got.cursor < 0 || got.cursor >= len(got.rows) {
		t.Fatalf("cursor out of range: %d", got.cursor)
	}
	if got.rows[got.cursor].kind != rowAddProfile {
		t.Fatalf("expected cursor on + New Profile row, got kind %v row %+v", got.rows[got.cursor].kind, got.rows[got.cursor])
	}
}

func TestRenderModelsTreeDoesNotShowFocusedRowOnTabs(t *testing.T) {
	m := Model{
		cfg: &config.Config{Models: map[string]models.Model{
			"alpha": {Name: "Alpha", Profiles: map[string]models.Profile{"default": {Name: "Default"}}},
		}, ModelsDirs: []string{"."}},
		focus:            focusTabs,
		leftMode:         modeModels,
		expandedModelKey: "alpha",
		cursor:           0,
	}
	m.rebuildRows()

	got := m.renderModelsTree(40)
	if strings.Contains(got, "> ") {
		t.Fatalf("expected no row cursor while tabs are focused, got %q", got)
	}
}

func TestRenderModelsTreeUnderlinesParentInProfileMode(t *testing.T) {
	m := Model{
		cfg: &config.Config{Models: map[string]models.Model{
			"alpha": {Name: "Alpha", Profiles: map[string]models.Profile{"default": {Name: "Default"}}},
		}, ModelsDirs: []string{"."}},
		focus:             focusLeft,
		leftMode:          modeModels,
		expandedModelKey:  "alpha",
		modelProfilesMode: true,
		cursor:            1,
	}
	m.rebuildRows()

	style := m.modelRowStyle(row{kind: rowModel, modelKey: "alpha", label: "Alpha"}, false)
	if !style.GetUnderline() {
		t.Fatal("expected active parent model style to be underlined")
	}
	got := m.renderModelsTree(40)
	if !strings.Contains(got, "> ") {
		t.Fatalf("expected profile row to keep cursor focus, got %q", got)
	}
}

func TestRenderRecentsListDoesNotShowFocusedRowOnTabs(t *testing.T) {
	m := Model{
		focus:        focusTabs,
		leftMode:     modeRecents,
		recentCursor: 0,
		recentRows: []row{
			{kind: rowProfile, modelKey: "alpha", profileKey: "default", label: "Alpha / Default"},
		},
	}

	got := m.renderRecentsList(40)
	if strings.Contains(got, "> ") {
		t.Fatalf("expected no recent row cursor while tabs are focused, got %q", got)
	}
}

func TestRenderRecentsListTruncatesLongRows(t *testing.T) {
	width := 28
	m := Model{
		focus:        focusLeft,
		leftMode:     modeRecents,
		recentCursor: 0,
		recentRows: []row{
			{kind: rowProfile, modelKey: "alpha", profileKey: "default", label: strings.Repeat("LongName", 12)},
		},
	}

	got := m.renderRecentsList(width)
	for _, line := range strings.Split(strings.TrimRight(got, "\n"), "\n") {
		if lipgloss.Width(line) > formRowTextWidth(width) {
			t.Fatalf("expected recent row to fit width %d, got %d in %q", formRowTextWidth(width), lipgloss.Width(line), line)
		}
	}
}

func TestRenderSettingsListDoesNotShowFocusedRowOnTabs(t *testing.T) {
	m := Model{
		focus:          focusTabs,
		leftMode:       modeSettings,
		settingsCursor: 0,
	}

	got := m.renderSettingsList(40)
	if strings.Contains(got, "> ") {
		t.Fatalf("expected no settings row cursor while tabs are focused, got %q", got)
	}
}

func TestEnterModelsPaneExpandsFirstModel(t *testing.T) {
	m := Model{
		cfg: &config.Config{Models: map[string]models.Model{
			"alpha": {Name: "Alpha", Profiles: map[string]models.Profile{"default": {Name: "Default"}}},
		}},
	}

	m.enterModelsPane()

	if m.expandedModelKey != "alpha" {
		t.Fatalf("expected first model to be expanded, got %q", m.expandedModelKey)
	}
	if m.cursor != 0 {
		t.Fatalf("expected cursor to land on the first model row, got %d", m.cursor)
	}
}

func TestNavigateUpFromFirstModelCollapsesExpanded(t *testing.T) {
	m := Model{
		cfg: &config.Config{Models: map[string]models.Model{
			"alpha": {Name: "Alpha", Profiles: map[string]models.Profile{"default": {Name: "Default"}}},
		}},
		focus:            focusLeft,
		leftMode:         modeModels,
		expandedModelKey: "alpha",
	}
	m.rebuildRows()
	m.cursor = indexOfModelRow(m.rows, "alpha")

	// Up from the first model row should move focus to tabs and collapse.
	next, _ := m.moveModelsCursor(-1)
	got := next.(Model)

	if got.focus != focusTabs {
		t.Fatalf("expected focusTabs after up from first model, got %v", got.focus)
	}
	if got.expandedModelKey != "" {
		t.Fatalf("expected expandedModelKey cleared, got %q", got.expandedModelKey)
	}
	if got.modelProfilesMode {
		t.Fatal("expected modelProfilesMode cleared")
	}
}

func TestFocusLeftFromModelsCollapsesExpanded(t *testing.T) {
	m := Model{
		cfg: &config.Config{Models: map[string]models.Model{
			"alpha": {Name: "Alpha", Profiles: map[string]models.Profile{"default": {Name: "Default"}}},
		}},
		focus:            focusLeft,
		leftMode:         modeModels,
		expandedModelKey: "alpha",
	}
	m.rebuildRows()
	m.cursor = indexOfModelRow(m.rows, "alpha")

	next, _ := m.moveFocusLeft()
	got := next.(Model)

	if got.focus != focusTabs {
		t.Fatalf("expected focusTabs after left from models pane, got %v", got.focus)
	}
	if got.expandedModelKey != "" {
		t.Fatalf("expected expandedModelKey cleared, got %q", got.expandedModelKey)
	}
}

func TestProfileListWrapsFromAddProfileToFirst(t *testing.T) {
	m := Model{
		cfg: &config.Config{Models: map[string]models.Model{
			"alpha": {Name: "Alpha", Profiles: map[string]models.Profile{
				"default": {Name: "Default"},
				"fast":    {Name: "Fast"},
			}},
		}},
		focus:             focusLeft,
		leftMode:          modeModels,
		expandedModelKey:  "alpha",
		modelProfilesMode: true,
	}
	m.rebuildRows()

	// Find the + New Profile row and set cursor there.
	addIdx := -1
	for i, r := range m.rows {
		if r.kind == rowAddProfile && r.modelKey == "alpha" {
			addIdx = i
			break
		}
	}
	if addIdx < 0 {
		t.Fatal("could not find + New Profile row")
	}
	m.cursor = addIdx

	// Arrow down from + New Profile should wrap to the first profile.
	next, _ := m.moveModelsCursor(1)
	got := next.(Model)

	if got.rows[got.cursor].kind != rowProfile {
		t.Fatalf("expected wrap to first rowProfile, got kind %v", got.rows[got.cursor].kind)
	}
	if got.rows[got.cursor].modelKey != "alpha" {
		t.Fatalf("expected profile in alpha, got %q", got.rows[got.cursor].modelKey)
	}
}
