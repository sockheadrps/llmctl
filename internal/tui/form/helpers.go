package form

import (
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/util"
)

// BuildFlagInput creates the small text input used to edit override flags.
func BuildFlagInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 64
	ti.Width = 22
	return ti
}

// BuildImportInput creates the import-args text input used in the form modal.
func BuildImportInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "paste CLI args here..."
	ti.CharLimit = 1024
	ti.Width = 40
	return ti
}

// CopyStringMap returns a shallow copy of a string map.
func CopyStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// FormNavOrder returns the Tab/arrow navigation sequence for the form.
// The layer-split slider (len(fields)+3) is inserted immediately after
// FieldGPULayers so it's reachable without scrolling past all other fields.
func FormNavOrder(numFields int) []int {
	order := make([]int, 0, numFields+5)
	for i := 0; i <= FieldGPULayers; i++ {
		order = append(order, i)
	}
	order = append(order, numFields+3)
	for i := FieldGPULayers + 1; i < numFields; i++ {
		order = append(order, i)
	}
	order = append(order, numFields+0)
	order = append(order, numFields+1)
	order = append(order, numFields+2)
	order = append(order, numFields+4)
	return order
}

// SuggestPort returns the first available port after the highest configured
// profile port.
func SuggestPort(cfg *config.Config) int {
	maxPort := 8079
	for _, mdl := range cfg.Models {
		for _, p := range mdl.Profiles {
			if p.Port > maxPort {
				maxPort = p.Port
			}
		}
	}
	start := maxPort + 1
	if free, err := util.FindFreePort(start); err == nil {
		return free
	}
	return start
}

// IntOrEmpty formats 0 as a blank string.
func IntOrEmpty(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// IntPtrOrEmpty formats nil as a blank string.
func IntPtrOrEmpty(n *int) string {
	if n == nil {
		return ""
	}
	return strconv.Itoa(*n)
}

// FloatPtrOrEmpty formats nil as a blank string.
func FloatPtrOrEmpty(f *float64) string {
	if f == nil {
		return ""
	}
	return strconv.FormatFloat(*f, 'f', -1, 64)
}

// BoolPtrOrEmpty formats nil as a blank string.
func BoolPtrOrEmpty(b *bool) string {
	if b == nil {
		return ""
	}
	if *b {
		return "true"
	}
	return "false"
}
