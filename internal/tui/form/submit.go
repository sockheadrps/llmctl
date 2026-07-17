package form

import (
	"fmt"
	"strings"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
)

// SubmitResult is the parsed profile data produced from a form submission.
type SubmitResult struct {
	Key     string
	Profile models.Profile
}

// CommitProfileSubmission writes a submitted profile back into config.
func CommitProfileSubmission(cfg *config.Config, modelKey, originalKey string, editing bool, submission SubmitResult) {
	profileModel := cfg.Models[modelKey]
	if profileModel.Profiles == nil {
		profileModel.Profiles = map[string]models.Profile{}
	}
	if editing && submission.Key != originalKey {
		delete(profileModel.Profiles, originalKey)
	}
	profileModel.Profiles[submission.Key] = submission.Profile
	cfg.Models[modelKey] = profileModel
}

// BuildProfileSubmission parses the canonical form values into a profile
// ready to be written back into config.
func BuildProfileSubmission(
	values []string,
	editing bool,
	originalKey string,
	existingProfiles map[string]models.Profile,
	flash, cpuOnly, mlock bool,
	rpcClientLayers int,
	cfgRPCEnabled bool,
	flagOverrides map[string]string,
) (SubmitResult, error) {
	value := func(i int) string { return strings.TrimSpace(values[i]) }

	key := value(FieldKey)
	if key == "" {
		return SubmitResult{}, fmt.Errorf("key is required")
	}

	renamed := editing && key != originalKey
	if _, exists := existingProfiles[key]; exists && (!editing || renamed) {
		return SubmitResult{}, fmt.Errorf("profile %q already exists on this model", key)
	}

	port, err := ParsePort(value(FieldPort))
	if err != nil {
		return SubmitResult{}, err
	}
	ctxSize, err := ParseIntOrZero(value(FieldCtxSize))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("ctx size must be an integer")
	}
	temp, err := ParseFloatPtr(value(FieldTemp))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("temp must be a number")
	}
	topP, err := ParseFloatPtr(value(FieldTopP))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("top p must be a number")
	}
	topK, err := ParseIntPtr(value(FieldTopK))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("top k must be an integer")
	}
	minP, err := ParseFloatPtr(value(FieldMinP))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("min p must be a number")
	}
	presencePenalty, err := ParseFloatPtr(value(FieldPresencePenalty))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("presence penalty must be a number")
	}
	repetitionPenalty, err := ParseFloatPtr(value(FieldRepetitionPenalty))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("repetition penalty must be a number")
	}
	frequencyPenalty, err := ParseFloatPtr(value(FieldFrequencyPenalty))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("frequency penalty must be a number")
	}
	seed, err := ParseIntPtr(value(FieldSeed))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("seed must be an integer")
	}
	batchSize, err := ParseIntPtr(value(FieldBatchSize))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("batch size must be an integer")
	}
	ubatchSize, err := ParseIntPtr(value(FieldUBatchSize))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("ubatch size must be an integer")
	}
	repeatLastN, err := ParseIntPtr(value(FieldRepeatLastN))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("repeat last n must be an integer")
	}
	gpuLayers, err := ParseIntOrZero(value(FieldGPULayers))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("gpu layers must be an integer")
	}
	mmap, err := ParseBoolPtr(value(FieldMMap))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("mmap must be true or false")
	}
	kvOffload, err := ParseBoolPtr(value(FieldKVOffload))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("kv offload must be true or false")
	}
	parallelSlots, err := ParseIntPtr(value(FieldParallelSlots))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("parallel slots must be an integer")
	}
	contBatching, err := ParseBoolPtr(value(FieldContBatching))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("continuous batching must be true or false")
	}
	cachePrompt, err := ParseBoolPtr(value(FieldCachePrompt))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("prompt cache must be true or false")
	}
	cacheRAM, err := ParseIntPtr(value(FieldCacheRAM))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("cache ram must be an integer")
	}
	reasoning, err := ParseReasoning(value(FieldReasoning))
	if err != nil {
		return SubmitResult{}, err
	}
	reasoningBudget, err := ParseIntPtr(value(FieldReasoningBudget))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("reasoning budget must be an integer")
	}
	rpcEnabled, err := ParseBoolPtr(value(FieldRPCEnabled))
	if err != nil {
		return SubmitResult{}, fmt.Errorf("rpc enabled must be true or false")
	}

	var extraArgs []string
	if raw := value(FieldExtraArgs); raw != "" {
		extraArgs = strings.Fields(raw)
	}

	if flagOverrides == nil {
		flagOverrides = map[string]string{}
	}
	if len(flagOverrides) == 0 {
		flagOverrides = nil
	} else {
		flagOverrides = CopyStringMap(flagOverrides)
	}

	rpcActive := (rpcEnabled != nil && *rpcEnabled) || (rpcEnabled == nil && cfgRPCEnabled)
	tensorSplit := ""
	if !cpuOnly && gpuLayers > 0 && rpcActive {
		client := rpcClientLayers
		if client < 0 {
			client = 0
		}
		if client > gpuLayers {
			client = gpuLayers
		}
		tensorSplit = fmt.Sprintf("%d,%d", client, gpuLayers-client)
	}

	return SubmitResult{
		Key: key,
		Profile: models.Profile{
			Name:              key,
			Host:              value(FieldHost),
			Alias:             value(FieldAlias),
			Port:              port,
			CtxSize:           ctxSize,
			Temp:              temp,
			TopP:              topP,
			TopK:              topK,
			MinP:              minP,
			PresencePenalty:   presencePenalty,
			RepetitionPenalty: repetitionPenalty,
			FrequencyPenalty:  frequencyPenalty,
			Seed:              seed,
			BatchSize:         batchSize,
			UBatchSize:        ubatchSize,
			RepeatLastN:       repeatLastN,
			FlashAttn:         flash,
			CPUOnly:           cpuOnly,
			MLock:             mlock,
			GPULayers:         gpuLayers,
			MMap:              mmap,
			KVOffload:         kvOffload,
			Parallel:          parallelSlots,
			ContBatching:      contBatching,
			CachePrompt:       cachePrompt,
			CacheRAM:          cacheRAM,
			Reasoning:         reasoning,
			ReasoningBudget:   reasoningBudget,
			ReasoningFormat:   value(FieldReasoningFormat),
			CacheTypeK:        value(FieldCacheK),
			CacheTypeV:        value(FieldCacheV),
			ExtraArgs:         extraArgs,
			Notes:             value(FieldNotes),
			RPCEnabled:        rpcEnabled,
			TensorSplit:       tensorSplit,
			FlagOverrides:     flagOverrides,
		},
	}, nil
}
