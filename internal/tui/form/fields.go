package form

import "github.com/charmbracelet/bubbles/textinput"

// Field is one text input row in the new-profile form.
type Field struct {
	Label string
	Input textinput.Model
}

// Field indices into the form field list, matching the order of Labels.
const (
	FieldKey = iota
	FieldHost
	FieldAlias
	FieldPort
	FieldCtxSize
	FieldTemp
	FieldTopP
	FieldTopK
	FieldMinP
	FieldPresencePenalty
	FieldRepetitionPenalty
	FieldFrequencyPenalty
	FieldSeed
	FieldBatchSize
	FieldUBatchSize
	FieldRepeatLastN
	FieldGPULayers
	FieldMMap
	FieldKVOffload
	FieldParallelSlots
	FieldContBatching
	FieldCachePrompt
	FieldCacheRAM
	FieldReasoning
	FieldReasoningBudget
	FieldReasoningFormat
	FieldCacheK
	FieldCacheV
	FieldExtraArgs
	FieldNotes
	FieldRPCEnabled
)

// Labels mirrors the root form field ordering so callers can build forms and
// descriptions from the same canonical list.
var Labels = []string{
	"Key", "Host", "Alias", "Port", "Ctx Size", "Temp", "Top P", "Top K", "Min P",
	"Presence Penalty", "Repetition Penalty", "Frequency Penalty", "Seed",
	"Batch Size", "UBatch Size", "Repeat Last N", "GPU Layers", "MMap", "KV Offload",
	"Parallel Slots", "Continuous Batching", "Prompt Cache", "Cache RAM",
	"Reasoning", "Reasoning Budget", "Reasoning Format", "Cache Type K", "Cache Type V",
	"Extra Args (space-separated)", "Notes",
	"RPC Enabled",
}

// FieldDefaultFlag returns the default llama-server CLI flag for a form field
// index, or "" for fields that don't map to a single CLI flag.
func FieldDefaultFlag(idx int) string {
	switch idx {
	case FieldHost:
		return "--host"
	case FieldAlias:
		return "--alias"
	case FieldPort:
		return "--port"
	case FieldCtxSize:
		return "--ctx-size"
	case FieldTemp:
		return "--temp"
	case FieldTopP:
		return "--top-p"
	case FieldTopK:
		return "--top-k"
	case FieldMinP:
		return "--min-p"
	case FieldPresencePenalty:
		return "--presence-penalty"
	case FieldRepetitionPenalty:
		return "--repeat-penalty"
	case FieldFrequencyPenalty:
		return "--frequency-penalty"
	case FieldSeed:
		return "--seed"
	case FieldBatchSize:
		return "--batch-size"
	case FieldUBatchSize:
		return "--ubatch-size"
	case FieldRepeatLastN:
		return "--repeat-last-n"
	case FieldGPULayers:
		return "--n-gpu-layers"
	case FieldMMap:
		return "--mmap"
	case FieldKVOffload:
		return "--kv-offload"
	case FieldParallelSlots:
		return "--parallel"
	case FieldContBatching:
		return "--cont-batching"
	case FieldCachePrompt:
		return "--cache-prompt"
	case FieldCacheRAM:
		return "--cache-ram"
	case FieldReasoning:
		return "--reasoning"
	case FieldReasoningBudget:
		return "--reasoning-budget"
	case FieldReasoningFormat:
		return "--reasoning-format"
	case FieldCacheK:
		return "--cache-type-k"
	case FieldCacheV:
		return "--cache-type-v"
	}
	return ""
}

// BuildFields creates a text input row for each canonical form label.
func BuildFields(defaults []string) []Field {
	fields := make([]Field, len(Labels))
	for i, label := range Labels {
		ti := textinput.New()
		ti.Prompt = ""
		ti.Placeholder = label
		ti.SetValue(defaults[i])
		ti.CharLimit = 256
		ti.Width = 40
		fields[i] = Field{Label: label, Input: ti}
	}
	return fields
}

// FieldDescription returns the help copy for a given field index.
func FieldDescription(idx int) string {
	switch idx {
	case FieldHost:
		return "Sets the network interface that llama-server listens on, such as 127.0.0.1 for local-only access or 0.0.0.0 for all interfaces."
	case FieldAlias:
		return "A friendly identifier for the profile that can help distinguish multiple profiles using the same model."
	case FieldPort:
		return "The TCP port used by the server. Each running profile should use a unique port."
	case FieldCtxSize:
		return "The model context window size. Larger values increase memory usage and allow longer conversations."
	case FieldTemp:
		return "Controls how random the generated output is. Lower values are more deterministic."
	case FieldTopP:
		return "Limits sampling to the smallest set of tokens whose cumulative probability exceeds this value."
	case FieldTopK:
		return "Restricts sampling to the most likely K tokens."
	case FieldMinP:
		return "Filters out low-probability tokens. A value of 0.0 disables this filter."
	case FieldPresencePenalty:
		return "Encourages the model to introduce new topics rather than repeating earlier ones."
	case FieldRepetitionPenalty:
		return "Discourages repeated text and can reduce loops or repetition."
	case FieldFrequencyPenalty:
		return "Penalizes tokens that have already appeared frequently to improve diversity."
	case FieldSeed:
		return "Sets the random seed for reproducible results. Use -1 for randomness."
	case FieldBatchSize:
		return "Maximum logical prompt processing batch size. Larger values can improve prompt throughput but need more memory."
	case FieldUBatchSize:
		return "Maximum physical micro-batch size. Advanced tuning option for throughput and memory tradeoffs."
	case FieldRepeatLastN:
		return "Number of recent tokens considered when applying repetition penalties."
	case FieldGPULayers:
		return "How many transformer layers to load on the GPU. Larger values usually increase performance."
	case FieldMMap:
		return "Enables memory-mapped model loading for faster startup and lower RAM use."
	case FieldKVOffload:
		return "Lets KV cache operations use the GPU. This can improve performance on supported hardware."
	case FieldParallelSlots:
		return "How many simultaneous inference slots the server should support."
	case FieldContBatching:
		return "Enables dynamic batching across multiple clients for better throughput."
	case FieldCachePrompt:
		return "Caches prompt processing to speed up repeated requests."
	case FieldCacheRAM:
		return "Maximum RAM allocated for prompt caching."
	case FieldReasoning:
		return "Turns reasoning mode on, off, or auto for compatible models."
	case FieldReasoningBudget:
		return "Sets a budget for reasoning tokens to limit thinking time and latency."
	case FieldReasoningFormat:
		return "Changes how reasoning content is returned, such as auto, none, or DeepSeek-style output."
	case FieldCacheK:
		return "The data type used for the key portion of the KV cache."
	case FieldCacheV:
		return "The data type used for the value portion of the KV cache."
	case FieldExtraArgs:
		return "Additional raw llama.cpp arguments, split by spaces, for advanced or experimental features."
	case FieldNotes:
		return "Optional notes for this profile that help you remember how it was intended to be used."
	case FieldRPCEnabled:
		return "Override the global RPC setting for this profile. Leave blank to follow the global setting; set true to force RPC on, false to force it off."
	case len(Labels):
		return "The Flash Attention toggle enables hardware-optimized attention when supported by your build."
	case len(Labels) + 1:
		return "Run on CPU only — forces GPU layers to 0 regardless of the gpu_layers setting."
	case len(Labels) + 2:
		return "Pin model in RAM — prevents OS from paging weights to disk under memory pressure. Requires free RAM equal to model size."
	case len(Labels) + 3:
		return "Split GPU layers between local and remote. ◄ ► or ← → moves one layer; shift+← → jumps five. Left (cyan) = local GPU, right (orange) = remote RPC server. Total comes from GPU Layers above."
	case len(Labels) + 4:
		return "Save this profile to your model configuration and return to the main view."
	default:
		return "Adjust this option to change how llama-server starts for this profile."
	}
}
