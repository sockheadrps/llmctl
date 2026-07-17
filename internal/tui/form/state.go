package form

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
)

// SyncFlagInput updates the override input to match the current focused row.
func SyncFlagInput(input textinput.Model, focus, fieldCount int, flagOverrides map[string]string) textinput.Model {
	def := FocusedFlag(focus, fieldCount)
	if def == "" {
		input.SetValue("")
		return input
	}
	val := def
	if override, ok := flagOverrides[def]; ok {
		val = override
	}
	input.SetValue(val)
	return input
}

// CommitFlagInput stores the override input value for the current focused row.
func CommitFlagInput(focus int, input textinput.Model, flagOverrides map[string]string) map[string]string {
	def := FocusedFlag(focus, len(Labels))
	if def == "" {
		return flagOverrides
	}
	val := strings.TrimSpace(input.Value())
	if val == "" || val == def {
		delete(flagOverrides, def)
		return flagOverrides
	}
	if flagOverrides == nil {
		flagOverrides = make(map[string]string)
	}
	flagOverrides[def] = val
	return flagOverrides
}

// ResetDescriptionScroll resets the auto-scroll state for the form details.
func ResetDescriptionScroll(scrollPauseTicks int) (int, int, int) {
	return 0, 1, scrollPauseTicks
}

// AdvanceDescriptionScroll updates the description scroll position.
func AdvanceDescriptionScroll(descScroll, descDir, descPause, lines, visible, pauseTicks int) (int, int, int) {
	return AdvanceAutoScroll(descScroll, descDir, descPause, lines, visible, pauseTicks)
}

// Dirty reports whether the edited form differs from its initial state.
func Dirty(
	currentFlash, initialFlash bool,
	currentCPUOnly, initialCPUOnly bool,
	currentMLock, initialMLock bool,
	currentRPCLayers, initialRPCLayers int,
	currentValues, initialValues []string,
	currentFlagOverrides, initialFlagOverrides map[string]string,
) bool {
	if currentFlash != initialFlash {
		return true
	}
	if currentCPUOnly != initialCPUOnly {
		return true
	}
	if currentMLock != initialMLock {
		return true
	}
	if currentRPCLayers != initialRPCLayers {
		return true
	}
	if len(currentValues) != len(initialValues) {
		return true
	}
	for i := range currentValues {
		if currentValues[i] != initialValues[i] {
			return true
		}
	}
	if len(currentFlagOverrides) != len(initialFlagOverrides) {
		return true
	}
	for k, v := range currentFlagOverrides {
		if initialFlagOverrides[k] != v {
			return true
		}
	}
	return false
}

// EnsureVisible keeps the focus row within the visible window.
func EnsureVisible(focus, scroll, visibleRows, totalRows int) int {
	if visibleRows <= 0 {
		visibleRows = 1
	}
	if totalRows <= visibleRows {
		return 0
	}
	if focus < scroll {
		scroll = focus
	} else if focus >= scroll+visibleRows {
		scroll = focus - visibleRows + 1
	}
	maxScroll := totalRows - visibleRows
	if scroll < 0 {
		return 0
	}
	if scroll > maxScroll {
		return maxScroll
	}
	return scroll
}

// OpenImportModal initializes the import-modal text input state.
func OpenImportModal(input textinput.Model) textinput.Model {
	input.Focus()
	return input
}

// CloseImportModal clears the import-modal text input state.
func CloseImportModal(input textinput.Model) textinput.Model {
	input.Blur()
	return input
}

// BlurFields removes focus from every form field input.
func BlurFields(fields []Field) {
	for i := range fields {
		fields[i].Input.Blur()
	}
}

// FieldValues returns the current values from the form fields.
func FieldValues(fields []Field) []string {
	values := make([]string, len(fields))
	for i := range fields {
		values[i] = fields[i].Input.Value()
	}
	return values
}

// NextFocus returns the next focus index in the form's navigation order.
func NextFocus(currentFocus, delta, fieldCount int) int {
	order := FormNavOrder(fieldCount)
	pos := 0
	for i, v := range order {
		if v == currentFocus {
			pos = i
			break
		}
	}
	pos = ((pos+delta)%len(order) + len(order)) % len(order)
	return order[pos]
}

// ImportedArgs captures the pieces of form state derived from a pasted export
// args string.
type ImportedArgs struct {
	Values map[int]string
	Flash  bool
}

// FormInit bundles the repeated state used to initialize the form screen.
type FormInit struct {
	Fields                 []Field
	Initial                []string
	InitialFlash           bool
	Flash                  bool
	InitialCPUOnly         bool
	CPUOnly                bool
	InitialMLock           bool
	MLock                  bool
	RPCClientLayers        int
	InitialRPCClientLayers int
	FlagInput              textinput.Model
	FlagOverrides          map[string]string
	InitialFlagOverrides   map[string]string
	ImportInput            textinput.Model
}

// NewProfileFormInit builds the default state for creating a new profile.
func NewProfileFormInit(cfg *config.Config, overrides map[int]string) FormInit {
	defaults := BuildNewProfileDefaults(SuggestPort(cfg), overrides)
	return FormInit{
		Fields:               BuildFields(defaults),
		Initial:              append([]string(nil), defaults...),
		InitialFlash:         true,
		Flash:                true,
		FlagInput:            BuildFlagInput(),
		FlagOverrides:        map[string]string{},
		InitialFlagOverrides: map[string]string{},
		ImportInput:          BuildImportInput(),
	}
}

// EditProfileFormInit builds the default state for editing an existing profile.
func EditProfileFormInit(profileKey string, profile models.Profile) FormInit {
	defaults, initClientLayers := BuildEditProfileDefaults(profileKey, profile)
	return FormInit{
		Fields:                 BuildFields(defaults),
		Initial:                append([]string(nil), defaults...),
		InitialFlash:           profile.FlashAttn,
		Flash:                  profile.FlashAttn,
		InitialCPUOnly:         profile.CPUOnly,
		CPUOnly:                profile.CPUOnly,
		InitialMLock:           profile.MLock,
		MLock:                  profile.MLock,
		RPCClientLayers:        initClientLayers,
		InitialRPCClientLayers: initClientLayers,
		FlagInput:              BuildFlagInput(),
		FlagOverrides:          CopyStringMap(profile.FlagOverrides),
		InitialFlagOverrides:   CopyStringMap(profile.FlagOverrides),
		ImportInput:            BuildImportInput(),
	}
}

// ApplyImportedArgs parses a pasted export-args string into field values and
// toggle state.
func ApplyImportedArgs(
	argsStr string,
	fieldCount int,
	defaultFlag func(int) string,
	flashIndex int,
	extraArgsIndex int,
) (ImportedArgs, error) {
	values, extra := ParseProfileArgs(argsStr, fieldCount, defaultFlag, flashIndex, extraArgsIndex)
	if len(values) == 0 && len(extra) == 0 {
		return ImportedArgs{}, fmt.Errorf("no recognizable CLI args found")
	}

	res := ImportedArgs{
		Values: make(map[int]string, len(values)),
	}
	for i, v := range values {
		res.Values[i] = v
	}
	if v, ok := values[flashIndex]; ok {
		res.Flash = strings.EqualFold(v, "on") || strings.EqualFold(v, "true") || v == "1"
	}
	return res, nil
}

// AdvanceAutoScroll updates a ping-pong auto-scroll offset.
func AdvanceAutoScroll(offset, dir, pause, lines, visible, pauseTicks int) (int, int, int) {
	if dir == 0 {
		dir = 1
	}
	maxScroll := lines - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if maxScroll == 0 {
		return 0, 1, 0
	}
	if pause > 0 {
		return offset, dir, pause - 1
	}
	offset += dir
	if offset >= maxScroll {
		return maxScroll, -1, pauseTicks
	}
	if offset <= 0 {
		return 0, 1, pauseTicks
	}
	return offset, dir, 0
}
