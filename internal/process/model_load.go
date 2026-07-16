package process

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/sockheadrps/llmctl/internal/statusserver"
)

var modelBufferPattern = regexp.MustCompile(`(?i)(CUDA\d+|RPC[_\d]+(?:\[[^\]]+\])?|CPU_Mapped|CPU)\s+(model|KV|compute)\s+buffer size\s*=\s*([\d.]+)\s*MiB`)

type bufferAllocation struct {
	model   float64
	kv      float64
	compute float64
}

// ParseModelLoadSlices extracts per-device model VRAM slices from a llama-server log.
// It aggregates model, KV, and compute buffers for each device, and ignores CPU-only
// mapped buffers when building GPU slices.
func ParseModelLoadSlices(logPath string) ([]statusserver.GPUDeviceInfo, error) {
	if strings.TrimSpace(logPath) == "" {
		return nil, nil
	}
	file, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	summary := map[string]*bufferAllocation{}
	order := make([]string, 0, 4)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		match := modelBufferPattern.FindStringSubmatch(scanner.Text())
		if len(match) != 4 {
			continue
		}

		device := normalizeModelLoadDevice(match[1])
		if device == "" {
			continue
		}

		value, err := strconv.ParseFloat(match[3], 64)
		if err != nil || value <= 0 {
			continue
		}

		alloc := summary[device]
		if alloc == nil {
			alloc = &bufferAllocation{}
			summary[device] = alloc
			order = append(order, device)
		}

		switch strings.ToLower(match[2]) {
		case "model":
			alloc.model = value
		case "kv":
			alloc.kv = value
		case "compute":
			alloc.compute = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(order) == 0 {
		return nil, nil
	}

	result := make([]statusserver.GPUDeviceInfo, 0, len(order))
	for _, device := range order {
		alloc := summary[device]
		if alloc == nil {
			continue
		}
		total := alloc.model + alloc.kv + alloc.compute
		if total <= 0 {
			continue
		}
		result = append(result, statusserver.GPUDeviceInfo{
			Name:    device,
			UUID:    device,
			UsedMiB: int64(total + 0.5),
		})
	}

	return result, nil
}

func normalizeModelLoadDevice(device string) string {
	device = strings.TrimSpace(device)
	if device == "" {
		return ""
	}
	if idx := strings.Index(device, "["); idx >= 0 {
		device = device[:idx]
	}
	switch {
	case strings.EqualFold(device, "CPU_Mapped") || strings.EqualFold(device, "CPU"):
		return ""
	default:
		return strings.ToUpper(device)
	}
}
