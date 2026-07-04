// Package gpu reads NVIDIA GPU VRAM usage via nvidia-smi, when available.
// It degrades silently on systems without an NVIDIA GPU or driver — callers
// should check Available() once and skip GPU display entirely if false,
// rather than shelling out on every poll.
package gpu

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
)

// Usage is aggregate VRAM usage across all detected GPUs, in MiB.
type Usage struct {
	UsedMiB  int64
	TotalMiB int64
}

// Available reports whether nvidia-smi is on PATH.
func Available() bool {
	_, err := exec.LookPath("nvidia-smi")
	return err == nil
}

// Name returns the name of the first GPU reported by nvidia-smi, or an error
// if nvidia-smi is unavailable or returns no output.
func Name() (string, error) {
	out, err := exec.Command("nvidia-smi", "--query-gpu=name", "--format=csv,noheader").Output()
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	return name, nil
}

// Total returns used/total VRAM summed across all GPUs nvidia-smi reports.
func Total() (Usage, error) {
	out, err := exec.Command("nvidia-smi", "--query-gpu=memory.used,memory.total", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return Usage{}, err
	}

	var usage Usage
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ",")
		if len(parts) != 2 {
			continue
		}
		used, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		total, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		usage.UsedMiB += used
		usage.TotalMiB += total
	}
	return usage, nil
}

// ByPID returns current VRAM usage (MiB) keyed by PID, for processes with
// an active CUDA context.
func ByPID() (map[int]int64, error) {
	out, err := exec.Command("nvidia-smi", "--query-compute-apps=pid,used_memory", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return nil, err
	}

	result := map[int]int64{}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) != 2 {
			continue
		}
		pid, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		mem, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		result[pid] = mem
	}
	return result, nil
}
