package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/runtime"
	"github.com/sockheadrps/llmctl/internal/statusserver"
)

func TestPickerSpinnerFrame(t *testing.T) {
	if got := pickerSpinnerFrame(0); got != "⠋" {
		t.Fatalf("expected first frame %q, got %q", "⠋", got)
	}
	if got := pickerSpinnerFrame(1); got != "⠙" {
		t.Fatalf("expected second frame %q, got %q", "⠙", got)
	}
	if got := pickerSpinnerFrame(11); got != "⠙" {
		t.Fatalf("expected looped frame %q, got %q", "⠙", got)
	}
}

func TestInactiveAddStyleDoesNotUseSelectedAccent(t *testing.T) {
	inactive := stripANSI(addStyle.Render("+ Add Model"))
	selected := stripANSI(selectedAddStyle.Render("+ Add Model"))
	if inactive != selected {
		// Strip check keeps the text stable while the color assertion below
		// checks the actual visual distinction.
		t.Fatalf("expected add styles to render same text, got %q and %q", inactive, selected)
	}
	if addStyle.GetForeground() == selectedAddStyle.GetForeground() {
		t.Fatalf("expected inactive add style not to use selected accent color")
	}
}

func TestViewPickerShowsCompletedScanStatus(t *testing.T) {
	m := Model{
		cfg: &config.Config{ModelsDirs: []string{`D:\models`}},
		picker: pickerState{
			files: []string{`D:\models\model.gguf`},
		},
	}

	got := stripANSI(m.viewPicker())
	if strings.Contains(got, "scanning") {
		t.Fatalf("expected completed picker view not to show scanning, got %q", got)
	}
	if !strings.Contains(got, `scanned D:\models`) {
		t.Fatalf("expected scanned status, got %q", got)
	}
}

func TestFormatDetailPairsUsesStackedLayoutWhenNarrow(t *testing.T) {
	got := formatDetailPairs([]detailPair{{label: "Port", value: "8080"}, {label: "Ctx Size", value: "4096"}}, 36)
	want := []string{"Port: 8080", "Ctx Size: 4096"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestFormatDetailPairsUsesTwoColumnLayoutWhenWide(t *testing.T) {
	got := formatDetailPairs([]detailPair{{label: "Port", value: "8080"}, {label: "Ctx Size", value: "4096"}}, 90)
	want := []string{"Port: 8080    Ctx Size: 4096"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestSplitPaneHeightCapsDetailsPane(t *testing.T) {
	_, details := splitPaneHeight(40, 0)
	if details > 12 {
		t.Fatalf("expected details pane to stay compact, got %d", details)
	}
}

func TestRenderDetailsWindowKeepsFixedHeight(t *testing.T) {
	m := Model{detailsScroll: 2}
	content := strings.Join([]string{
		"one",
		"two",
		"three",
		"four",
		"five",
	}, "\n")

	got := m.renderDetailsWindow(content, 20, 3)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected fixed 3-line details window, got %d lines: %q", len(lines), got)
	}
	if strings.TrimSpace(lines[0]) != "three" {
		t.Fatalf("expected scrolled window to start at third line, got %q", lines[0])
	}
}

func TestRenderDetailsWindowDoesNotPadShortContent(t *testing.T) {
	m := Model{}
	got := m.renderDetailsWindow("Recents\n\nSelect from your most recently run profiles to quickly re-run one.", 34, 12)
	lines := strings.Split(got, "\n")
	if len(lines) >= 12 {
		t.Fatalf("expected short details content not to be padded to pane height, got %d lines: %q", len(lines), got)
	}
	if strings.HasSuffix(got, "\n") {
		t.Fatalf("expected no trailing newline, got %q", got)
	}
}

func TestWrappedContentLinesFitPaneInnerWidth(t *testing.T) {
	width := 34
	lines := wrappedContentLines("Select from your most recently run profiles to quickly re-run one.", width)
	for _, line := range lines {
		if lipgloss.Width(line) > formDescriptionTextWidth(width) {
			t.Fatalf("expected line %q width %d to fit inner width %d", line, lipgloss.Width(line), formDescriptionTextWidth(width))
		}
	}
}

func TestTailFittingHeightSanitizesAndCapsWrappedLogPreview(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "run.log")
	raw := "\x1b[40m" + strings.Repeat("wrapped-output ", 40) + "\x1b[0m\r\nnext\x07"
	if err := os.WriteFile(logPath, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}

	got := tailFittingHeight(logPath, 24, 3)
	if strings.Contains(got, "\x1b") || strings.Contains(got, "\a") || strings.Contains(got, "\r") {
		t.Fatalf("expected sanitized preview, got %q", got)
	}

	lines := strings.Split(got, "\n")
	if len(lines) > 3 {
		t.Fatalf("expected at most 3 wrapped lines, got %d: %q", len(lines), got)
	}
	for _, line := range lines {
		if lipgloss.Width(line) > formDescriptionTextWidth(24) {
			t.Fatalf("expected line %q width %d to fit inner width %d", line, lipgloss.Width(line), formDescriptionTextWidth(24))
		}
	}
}

func TestRenderDetailsShowsProfileNotesBelowModelSource(t *testing.T) {
	m := Model{
		cfg: &config.Config{Models: map[string]models.Model{
			"model": {
				Name: "Model",
				Path: "model.gguf",
				Profiles: map[string]models.Profile{
					"profile": {Name: "Profile", Port: 8080, Notes: "important profile notes"},
				},
			},
		}},
		focus:    focusLeft,
		leftMode: modeModels,
		cursor:   1,
		rows: []row{
			{kind: rowModel, modelKey: "model", label: "Model"},
			{kind: rowProfile, modelKey: "model", profileKey: "profile", label: "Profile"},
		},
	}

	got := stripANSI(m.renderDetails(40))
	source := strings.Index(got, "model.gguf")
	notes := strings.Index(got, "important profile notes")
	connection := strings.Index(got, "\nProfile\n")
	if source < 0 || notes < 0 || connection < 0 {
		t.Fatalf("expected source, notes, and settings in details, got %q", got)
	}
	if !(source < notes && notes < connection) {
		t.Fatalf("expected notes between source and settings, got %q", got)
	}
	if strings.Contains(got, "Notes:") {
		t.Fatalf("expected notes not to render as a lower Notes section, got %q", got)
	}
}

func TestRenderClientStatusLinesShowsModelAndSizeOnly(t *testing.T) {
	m := Model{
		rpcServerAlive: true,
		rpcServerState: runtime.RPCServerState{PID: 42},
		gpuByPID:       map[int]int64{42: 2048},
	}
	got := stripANSI(m.renderClientStatusLines(statusserver.ClientInfo{
		Name: "client-machine",
		Running: []statusserver.RunningInfo{{
			Model:          "Model",
			Profile:        "Profile",
			ModelSizeBytes: 4 * 1024 * 1024 * 1024,
		}},
	}))

	if !strings.Contains(got, "Model / Profile") {
		t.Fatalf("expected model/profile label, got %q", got)
	}
	if !strings.Contains(got, "4.0 GB / 2.0 GB server GPU") {
		t.Fatalf("expected full/server GPU size metadata, got %q", got)
	}
	if strings.Contains(got, "client-machine") {
		t.Fatalf("expected client name to be omitted, got %q", got)
	}
}
