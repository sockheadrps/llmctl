package form

import (
	"strconv"
	"strings"

	"github.com/sockheadrps/llmctl/internal/models"
)

// BuildNewProfileDefaults returns the base defaults for a new profile form.
func BuildNewProfileDefaults(suggestedPort int, overrides map[int]string) []string {
	defaults := make([]string, len(Labels))
	defaults[FieldKey] = ""
	defaults[FieldHost] = ""
	defaults[FieldAlias] = ""
	defaults[FieldPort] = strconv.Itoa(suggestedPort)
	defaults[FieldCtxSize] = "8192"
	defaults[FieldTemp] = "0.6"
	defaults[FieldTopP] = "0.95"
	defaults[FieldTopK] = "20"
	defaults[FieldMinP] = "0.0"
	defaults[FieldPresencePenalty] = ""
	defaults[FieldRepetitionPenalty] = ""
	defaults[FieldFrequencyPenalty] = ""
	defaults[FieldSeed] = ""
	defaults[FieldBatchSize] = ""
	defaults[FieldUBatchSize] = ""
	defaults[FieldRepeatLastN] = ""
	defaults[FieldGPULayers] = "999"
	defaults[FieldMMap] = ""
	defaults[FieldKVOffload] = ""
	defaults[FieldParallelSlots] = ""
	defaults[FieldContBatching] = ""
	defaults[FieldCachePrompt] = ""
	defaults[FieldCacheRAM] = ""
	defaults[FieldReasoning] = ""
	defaults[FieldReasoningBudget] = ""
	defaults[FieldReasoningFormat] = ""
	defaults[FieldCacheK] = ""
	defaults[FieldCacheV] = ""
	defaults[FieldExtraArgs] = ""
	defaults[FieldNotes] = ""
	defaults[FieldRPCEnabled] = ""

	for idx, val := range overrides {
		if idx >= 0 && idx < len(defaults) {
			defaults[idx] = val
		}
	}
	return defaults
}

// BuildEditProfileDefaults returns the current values for an existing profile.
func BuildEditProfileDefaults(profileKey string, p models.Profile) ([]string, int) {
	defaults := make([]string, len(Labels))
	defaults[FieldKey] = profileKey
	defaults[FieldHost] = p.Host
	defaults[FieldAlias] = p.Alias
	defaults[FieldPort] = strconv.Itoa(p.Port)
	defaults[FieldCtxSize] = IntOrEmpty(p.CtxSize)
	defaults[FieldTemp] = FloatPtrOrEmpty(p.Temp)
	defaults[FieldTopP] = FloatPtrOrEmpty(p.TopP)
	defaults[FieldTopK] = IntPtrOrEmpty(p.TopK)
	defaults[FieldMinP] = FloatPtrOrEmpty(p.MinP)
	defaults[FieldPresencePenalty] = FloatPtrOrEmpty(p.PresencePenalty)
	defaults[FieldRepetitionPenalty] = FloatPtrOrEmpty(p.RepetitionPenalty)
	defaults[FieldFrequencyPenalty] = FloatPtrOrEmpty(p.FrequencyPenalty)
	defaults[FieldSeed] = IntPtrOrEmpty(p.Seed)
	defaults[FieldBatchSize] = IntPtrOrEmpty(p.BatchSize)
	defaults[FieldUBatchSize] = IntPtrOrEmpty(p.UBatchSize)
	defaults[FieldRepeatLastN] = IntPtrOrEmpty(p.RepeatLastN)
	defaults[FieldGPULayers] = IntOrEmpty(p.GPULayers)
	defaults[FieldMMap] = BoolPtrOrEmpty(p.MMap)
	defaults[FieldKVOffload] = BoolPtrOrEmpty(p.KVOffload)
	defaults[FieldParallelSlots] = IntPtrOrEmpty(p.Parallel)
	defaults[FieldContBatching] = BoolPtrOrEmpty(p.ContBatching)
	defaults[FieldCachePrompt] = BoolPtrOrEmpty(p.CachePrompt)
	defaults[FieldCacheRAM] = IntPtrOrEmpty(p.CacheRAM)
	defaults[FieldReasoning] = p.Reasoning
	defaults[FieldReasoningBudget] = IntPtrOrEmpty(p.ReasoningBudget)
	defaults[FieldReasoningFormat] = p.ReasoningFormat
	defaults[FieldCacheK] = p.CacheTypeK
	defaults[FieldCacheV] = p.CacheTypeV
	defaults[FieldExtraArgs] = strings.Join(p.ExtraArgs, " ")
	defaults[FieldNotes] = p.Notes
	defaults[FieldRPCEnabled] = BoolPtrOrEmpty(p.RPCEnabled)

	clientLayers := 0
	if p.TensorSplit != "" {
		parts := strings.SplitN(p.TensorSplit, ",", 2)
		if n, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && n >= 0 {
			clientLayers = n
		}
	}
	return defaults, clientLayers
}
