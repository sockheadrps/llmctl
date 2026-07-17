package tui

import (
	"github.com/charmbracelet/bubbles/textinput"

	tui_form "github.com/sockheadrps/llmctl/internal/tui/form"
)

// formField is one text input row in the new-profile form.
type formField struct {
	label string
	input textinput.Model
}

// formState backs the "New Profile"/"Edit Profile" screen. Focus indices
// 0..len(fields)-1 are the text fields, len(fields) is the Flash Attention
// toggle, len(fields)+1 is the CPU Only toggle, len(fields)+2 is the MLock
// toggle, len(fields)+3 is the Tensor Split slider, and len(fields)+4 is Save.
//
// navigating=true is navigate mode (arrow/WASD moves between fields; Enter
// activates a field for editing). navigating=false is edit mode (keystrokes
// go to the focused textinput; Enter commits and returns to navigate mode).
type formState struct {
	modelKey               string
	editing                bool
	originalKey            string
	fields                 []formField
	initial                []string
	initialFlash           bool
	flash                  bool
	initialCPUOnly         bool
	cpuOnly                bool
	initialMLock           bool
	mlock                  bool
	rpcClientLayers        int
	initialRPCClientLayers int
	focus                  int
	scroll                 int
	descScroll             int
	descDir                int
	descPause              int
	err                    string
	navigating             bool
	flagFocus              bool
	flagInput              textinput.Model
	flagOverrides          map[string]string
	initialFlagOverrides   map[string]string
	importEditing          bool
	importInput            textinput.Model
	importErr              string
}

// Field indices into formState.fields, matching the order of formLabels.
const (
	fieldKey               = tui_form.FieldKey
	fieldHost              = tui_form.FieldHost
	fieldAlias             = tui_form.FieldAlias
	fieldPort              = tui_form.FieldPort
	fieldCtxSize           = tui_form.FieldCtxSize
	fieldTemp              = tui_form.FieldTemp
	fieldTopP              = tui_form.FieldTopP
	fieldTopK              = tui_form.FieldTopK
	fieldMinP              = tui_form.FieldMinP
	fieldPresencePenalty   = tui_form.FieldPresencePenalty
	fieldRepetitionPenalty = tui_form.FieldRepetitionPenalty
	fieldFrequencyPenalty  = tui_form.FieldFrequencyPenalty
	fieldSeed              = tui_form.FieldSeed
	fieldBatchSize         = tui_form.FieldBatchSize
	fieldUBatchSize        = tui_form.FieldUBatchSize
	fieldRepeatLastN       = tui_form.FieldRepeatLastN
	fieldGPULayers         = tui_form.FieldGPULayers
	fieldMMap              = tui_form.FieldMMap
	fieldKVOffload         = tui_form.FieldKVOffload
	fieldParallelSlots     = tui_form.FieldParallelSlots
	fieldContBatching      = tui_form.FieldContBatching
	fieldCachePrompt       = tui_form.FieldCachePrompt
	fieldCacheRAM          = tui_form.FieldCacheRAM
	fieldReasoning         = tui_form.FieldReasoning
	fieldReasoningBudget   = tui_form.FieldReasoningBudget
	fieldReasoningFormat   = tui_form.FieldReasoningFormat
	fieldCacheK            = tui_form.FieldCacheK
	fieldCacheV            = tui_form.FieldCacheV
	fieldExtraArgs         = tui_form.FieldExtraArgs
	fieldNotes             = tui_form.FieldNotes
	fieldRPCEnabled        = tui_form.FieldRPCEnabled
)

var formLabels = tui_form.Labels
