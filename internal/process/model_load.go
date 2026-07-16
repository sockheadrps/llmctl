package process

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/sockheadrps/llmctl/internal/statusserver"
)

var modelBufferLineRE = regexp.MustCompile(`(?i)^\s*(CUDA|RPC)(\d+)\s+model buffer size\s*=\s*([0-9]+(?:\.[0-9]+)?)\s*MiB\s*$`)

// ParseModelLoadSlices extracts model VRAM slices from a llama-server log.
// It keeps the last seen size for each CUDA/RPC buffer label.
func ParseModelLoadSlices(logPath string) ([]statusserver.GPUDeviceInfo, error) {
	if strings.TrimSpace(logPath) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		return nil, err
	}
	return parseModelLoadSlices(string(data)), nil
}

func parseModelLoadSlices(raw string) []statusserver.GPUDeviceInfo {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	seen := make(map[string]statusserver.GPUDeviceInfo)
	order := make([]string, 0, 4)

	for _, line := range strings.Split(raw, "\n") {
		match := modelBufferLineRE.FindStringSubmatch(strings.TrimSpace(line))
		if len(match) != 4 {
			continue
		}
		label := strings.ToUpper(match[1]) + match[2]
		value, err := strconv.ParseFloat(match[3], 64)
		if err != nil {
			continue
		}
		info := statusserver.GPUDeviceInfo{
			Index:   mustAtoi(match[2]),
			UUID:    label,
			Name:    label,
			UsedMiB: int64(math.Round(value)),
		}
		if _, ok := seen[label]; !ok {
			order = append(order, label)
		}
		seen[label] = info
	}

	if len(order) == 0 {
		return nil
	}
	out := make([]statusserver.GPUDeviceInfo, 0, len(order))
	for _, label := range order {
		out = append(out, seen[label])
	}
	return out
}

func mustAtoi(v string) int {
	n, err := strconv.Atoi(v)
	if err != nil {
		return -1
	}
	return n
}

func formatModelLoadSlices(slices []statusserver.GPUDeviceInfo) string {
	parts := make([]string, 0, len(slices))
	for _, s := range slices {
		parts = append(parts, fmt.Sprintf("%s=%d", s.Name, s.UsedMiB))
	}
	return strings.Join(parts, ",")
}
