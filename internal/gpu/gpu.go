// Package gpu reads NVIDIA GPU VRAM usage via nvidia-smi, when available.
// It degrades silently on systems without an NVIDIA GPU or driver — callers
// should check Available() once and skip GPU display entirely if false,
// rather than shelling out on every poll.
package gpu

import (
	"bufio"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// Usage is aggregate VRAM usage across all detected GPUs, in MiB.
type Usage struct {
	UsedMiB  int64
	TotalMiB int64
}

// DeviceUsage describes one GPU or one process's load on a GPU.
type DeviceUsage struct {
	Index    int
	UUID     string
	Name     string
	UsedMiB  int64
	TotalMiB int64
}

// ProcessUsage describes how much VRAM a PID is using on one GPU.
type ProcessUsage struct {
	PID     int
	GPUUUID string
	UsedMiB int64
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

// Devices returns the VRAM usage for each detected GPU.
func Devices() ([]DeviceUsage, error) {
	out, err := exec.Command("nvidia-smi", "--query-gpu=index,uuid,name,memory.used,memory.total", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return nil, err
	}

	var devices []DeviceUsage
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ",")
		if len(parts) != 5 {
			continue
		}
		index, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		used, err2 := strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 64)
		total, err3 := strconv.ParseInt(strings.TrimSpace(parts[4]), 10, 64)
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		devices = append(devices, DeviceUsage{
			Index:    index,
			UUID:     strings.TrimSpace(parts[1]),
			Name:     strings.TrimSpace(parts[2]),
			UsedMiB:  used,
			TotalMiB: total,
		})
	}
	sort.SliceStable(devices, func(i, j int) bool {
		return devices[i].Index < devices[j].Index
	})
	return devices, nil
}

// Total returns used/total VRAM summed across all GPUs nvidia-smi reports.
func Total() (Usage, error) {
	devices, err := Devices()
	if err != nil {
		return Usage{}, err
	}

	var usage Usage
	for _, device := range devices {
		usage.UsedMiB += device.UsedMiB
		usage.TotalMiB += device.TotalMiB
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

// ByPIDDevices returns current VRAM usage keyed by PID and GPU UUID, for
// processes with an active CUDA context.
func ByPIDDevices() ([]ProcessUsage, error) {
	out, err := exec.Command("nvidia-smi", "--query-compute-apps=pid,gpu_uuid,used_memory", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return nil, err
	}

	var result []ProcessUsage
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) != 3 {
			continue
		}
		pid, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		mem, err2 := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		result = append(result, ProcessUsage{
			PID:     pid,
			GPUUUID: strings.TrimSpace(parts[1]),
			UsedMiB: mem,
		})
	}
	return result, nil
}
