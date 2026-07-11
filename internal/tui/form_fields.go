package tui

import "github.com/charmbracelet/bubbles/textinput"

// fieldDefaultFlag returns the default llama-server CLI flag for a form field
// index, or "" for fields that don't map to a single CLI flag.
func fieldDefaultFlag(idx int) string {
	switch idx {
	case fieldHost:
		return "--host"
	case fieldAlias:
		return "--alias"
	case fieldPort:
		return "--port"
	case fieldCtxSize:
		return "--ctx-size"
	case fieldTemp:
		return "--temp"
	case fieldTopP:
		return "--top-p"
	case fieldTopK:
		return "--top-k"
	case fieldMinP:
		return "--min-p"
	case fieldPresencePenalty:
		return "--presence-penalty"
	case fieldRepetitionPenalty:
		return "--repeat-penalty"
	case fieldFrequencyPenalty:
		return "--frequency-penalty"
	case fieldSeed:
		return "--seed"
	case fieldBatchSize:
		return "--batch-size"
	case fieldUBatchSize:
		return "--ubatch-size"
	case fieldRepeatLastN:
		return "--repeat-last-n"
	case fieldGPULayers:
		return "--n-gpu-layers"
	case fieldMMap:
		return "--mmap"
	case fieldKVOffload:
		return "--kv-offload"
	case fieldParallelSlots:
		return "--parallel"
	case fieldContBatching:
		return "--cont-batching"
	case fieldCachePrompt:
		return "--cache-prompt"
	case fieldCacheRAM:
		return "--cache-ram"
	case fieldReasoning:
		return "--reasoning"
	case fieldReasoningBudget:
		return "--reasoning-budget"
	case fieldReasoningFormat:
		return "--reasoning-format"
	case fieldCacheK:
		return "--cache-type-k"
	case fieldCacheV:
		return "--cache-type-v"
	}
	return ""
}

func buildFormFields(defaults []string) []formField {
	fields := make([]formField, len(formLabels))
	for i, label := range formLabels {
		ti := textinput.New()
		ti.Prompt = ""
		ti.Placeholder = label
		ti.SetValue(defaults[i])
		ti.CharLimit = 256
		ti.Width = 40
		fields[i] = formField{label: label, input: ti}
	}
	return fields
}

func formFieldDescription(idx int) string {
	switch idx {
	case fieldHost:
		return "Sets the network interface that llama-server listens on, such as 127.0.0.1 for local-only access or 0.0.0.0 for all interfaces."
	case fieldAlias:
		return "A friendly identifier for the profile that can help distinguish multiple profiles using the same model."
	case fieldPort:
		return "The TCP port used by the server. Each running profile should use a unique port."
	case fieldCtxSize:
		return "The model context window size. Larger values increase memory usage and allow longer conversations."
	case fieldTemp:
		return "Controls how random the generated output is. Lower values are more deterministic."
	case fieldTopP:
		return "Limits sampling to the smallest set of tokens whose cumulative probability exceeds this value."
	case fieldTopK:
		return "Restricts sampling to the most likely K tokens."
	case fieldMinP:
		return "Filters out low-probability tokens. A value of 0.0 disables this filter."
	case fieldPresencePenalty:
		return "Encourages the model to introduce new topics rather than repeating earlier ones."
	case fieldRepetitionPenalty:
		return "Discourages repeated text and can reduce loops or repetition."
	case fieldFrequencyPenalty:
		return "Penalizes tokens that have already appeared frequently to improve diversity."
	case fieldSeed:
		return "Sets the random seed for reproducible results. Use -1 for randomness."
	case fieldBatchSize:
		return "Maximum logical prompt processing batch size. Larger values can improve prompt throughput but need more memory."
	case fieldUBatchSize:
		return "Maximum physical micro-batch size. Advanced tuning option for throughput and memory tradeoffs."
	case fieldRepeatLastN:
		return "Number of recent tokens considered when applying repetition penalties."
	case fieldGPULayers:
		return "How many transformer layers to load on the GPU. Larger values usually increase performance."
	case fieldMMap:
		return "Enables memory-mapped model loading for faster startup and lower RAM use."
	case fieldKVOffload:
		return "Lets KV cache operations use the GPU. This can improve performance on supported hardware."
	case fieldParallelSlots:
		return "How many simultaneous inference slots the server should support."
	case fieldContBatching:
		return "Enables dynamic batching across multiple clients for better throughput."
	case fieldCachePrompt:
		return "Caches prompt processing to speed up repeated requests."
	case fieldCacheRAM:
		return "Maximum RAM allocated for prompt caching."
	case fieldReasoning:
		return "Turns reasoning mode on, off, or auto for compatible models."
	case fieldReasoningBudget:
		return "Sets a budget for reasoning tokens to limit thinking time and latency."
	case fieldReasoningFormat:
		return "Changes how reasoning content is returned, such as auto, none, or DeepSeek-style output."
	case fieldCacheK:
		return "The data type used for the key portion of the KV cache."
	case fieldCacheV:
		return "The data type used for the value portion of the KV cache."
	case fieldExtraArgs:
		return "Additional raw llama.cpp arguments, split by spaces, for advanced or experimental features."
	case fieldNotes:
		return "Optional notes for this profile that help you remember how it was intended to be used."
	case fieldRPCEnabled:
		return "Override the global RPC setting for this profile. Leave blank to follow the global setting; set true to force RPC on, false to force it off."
	case len(formLabels):
		return "The Flash Attention toggle enables hardware-optimized attention when supported by your build."
	case len(formLabels) + 1:
		return "Run on CPU only — forces GPU layers to 0 regardless of the gpu_layers setting."
	case len(formLabels) + 2:
		return "Pin model in RAM — prevents OS from paging weights to disk under memory pressure. Requires free RAM equal to model size."
	case len(formLabels) + 3:
		return "Split GPU layers between local and remote. ◄ ► or ← → moves one layer; shift+← → jumps five. Left (cyan) = local GPU, right (orange) = remote RPC server. Total comes from GPU Layers above."
	case len(formLabels) + 4:
		return "Save this profile to your model configuration and return to the main view."
	default:
		return "Adjust this option to change how llama-server starts for this profile."
	}
}
