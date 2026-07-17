package picker

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/util"
)

// State backs the "Add Model" screen: a list of .gguf files found under the
// config's models_dirs that aren't already registered. Unreadable notes which
// configured directories couldn't be scanned.
type State struct {
	Files      []string
	Cursor     int
	Err        error
	Unreadable []string
}

// Action describes the result of a picker keypress.
type Action int

const (
	ActionNone Action = iota
	ActionBack
	ActionImport
)

// Open scans every configured models directory for importable GGUF files and
// returns the picker state.
func Open(cfg *config.Config) (State, error) {
	dirs, err := cfg.ResolvedModelsDirs()
	if err != nil {
		return State{}, err
	}

	used := make(map[string]bool, len(cfg.Models))
	for _, mdl := range cfg.Models {
		if len(mdl.Profiles) > 0 {
			used[mdl.Path] = true
		}
	}

	var files, unreadable []string
	for _, dir := range dirs {
		found, err := models.ScanGGUF(dir)
		if err != nil {
			unreadable = append(unreadable, dir)
			continue
		}
		for _, f := range found {
			if !used[f] {
				files = append(files, f)
			}
		}
	}
	sort.Strings(files)

	if len(dirs) == 0 {
		return State{Err: fmt.Errorf("no model directories configured - add one from the Model Directories screen")}, nil
	}
	return State{Files: files, Unreadable: unreadable}, nil
}

// Update handles picker navigation keys.
func (s *State) Update(msg tea.KeyMsg) Action {
	switch msg.String() {
	case "esc", "q":
		return ActionBack
	case "up", "k":
		if s.Cursor > 0 {
			s.Cursor--
		}
	case "down", "j":
		if s.Cursor < len(s.Files)-1 {
			s.Cursor++
		}
	case "enter":
		return ActionImport
	}
	return ActionNone
}

// SpinnerFrame returns a tiny loading spinner frame.
func SpinnerFrame(step int) string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return frames[step%len(frames)]
}

var quantPattern = regexp.MustCompile(`(?i)\b(?:q\d(?:_[a-z0-9]+)?|iq\d(?:_[a-z0-9]+)?|f16|bf16|f32)\b`)

// FileMetadata returns size/family/quantization details for a picker row.
func FileMetadata(path string) string {
	parts := []string{}
	if info, err := os.Stat(path); err == nil {
		parts = append(parts, util.FormatBytes(info.Size()))
	}
	if family := InferModelFamily(path); family != "" {
		parts = append(parts, "family "+family)
	}
	if quant := InferQuant(path); quant != "" {
		parts = append(parts, "quant "+strings.ToUpper(quant))
	}
	return strings.Join(parts, " · ")
}

// InferModelFamily extracts the model family name from a GGUF filename.
func InferModelFamily(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	for _, sep := range []string{"-", "_", "."} {
		if idx := strings.Index(name, sep); idx > 0 {
			return name[:idx]
		}
	}
	return name
}

// InferQuant extracts a quantization label from a GGUF filename.
func InferQuant(path string) string {
	return quantPattern.FindString(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
}
