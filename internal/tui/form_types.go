package tui

import "github.com/charmbracelet/bubbles/textinput"

// formField is one text input row in the new-profile form.
type formField struct {
	label string
	input textinput.Model
}

// formState backs the "New Profile"/"Edit Profile" screen. Focus indices
// 0..len(fields)-1 are the text fields, len(fields) is the Flash Attention
// toggle, len(fields)+1 is the CPU Only toggle, len(fields)+2 is the MLock
// toggle, and len(fields)+3 is the Save action.
//
// navigating=true is navigate mode (arrow/WASD moves between fields; Enter
// activates a field for editing). navigating=false is edit mode (keystrokes
// go to the focused textinput; Enter commits and returns to navigate mode).
type formState struct {
	modelKey             string
	editing              bool
	originalKey          string
	fields               []formField
	initial              []string
	initialFlash         bool
	flash                bool
	initialCPUOnly       bool
	cpuOnly              bool
	initialMLock         bool
	mlock                bool
	focus                int
	scroll               int
	descScroll           int
	descDir              int
	descPause            int
	err                  string
	navigating           bool
	flagFocus            bool
	flagInput            textinput.Model
	flagOverrides        map[string]string
	initialFlagOverrides map[string]string
	importEditing        bool
	importInput          textinput.Model
	importErr            string
}

// Field indices into formState.fields, matching the order of formLabels.
const (
	fieldKey = iota
	fieldHost
	fieldAlias
	fieldPort
	fieldCtxSize
	fieldTemp
	fieldTopP
	fieldTopK
	fieldMinP
	fieldPresencePenalty
	fieldRepetitionPenalty
	fieldFrequencyPenalty
	fieldSeed
	fieldBatchSize
	fieldUBatchSize
	fieldRepeatLastN
	fieldGPULayers
	fieldMMap
	fieldKVOffload
	fieldParallelSlots
	fieldContBatching
	fieldCachePrompt
	fieldCacheRAM
	fieldReasoning
	fieldReasoningBudget
	fieldReasoningFormat
	fieldCacheK
	fieldCacheV
	fieldExtraArgs
	fieldNotes
	fieldRPCEnabled
)

var formLabels = []string{
	"Key", "Host", "Alias", "Port", "Ctx Size", "Temp", "Top P", "Top K", "Min P",
	"Presence Penalty", "Repetition Penalty", "Frequency Penalty", "Seed",
	"Batch Size", "UBatch Size", "Repeat Last N", "GPU Layers", "MMap", "KV Offload",
	"Parallel Slots", "Continuous Batching", "Prompt Cache", "Cache RAM",
	"Reasoning", "Reasoning Budget", "Reasoning Format", "Cache Type K", "Cache Type V",
	"Extra Args (space-separated)", "Notes",
	"RPC Enabled",
}
