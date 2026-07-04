// Package process builds llama-server command lines from a Profile and
// launches them as detached, logged subprocesses.
package process

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/sockheadrps/llmctl/internal/models"
)

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// BuildArgs converts a Model+Profile pair into llama-server CLI flags.
func BuildArgs(m models.Model, p models.Profile) []string {
	var args []string
	if m.IsRemote() {
		args = append(args, "-hf", m.HFRepo)
	} else {
		args = append(args, "--model", m.Path)
	}
	args = append(args, "--port", strconv.Itoa(p.Port))
	if p.Host != "" {
		args = append(args, "--host", p.Host)
	}
	if p.Alias != "" {
		args = append(args, "--alias", p.Alias)
	}
	if p.CtxSize > 0 {
		args = append(args, "--ctx-size", strconv.Itoa(p.CtxSize))
	}
	if p.BatchSize != nil {
		args = append(args, "--batch-size", strconv.Itoa(*p.BatchSize))
	}
	if p.UBatchSize != nil {
		args = append(args, "--ubatch-size", strconv.Itoa(*p.UBatchSize))
	}
	if p.Temp != nil {
		args = append(args, "--temp", formatFloat(*p.Temp))
	}
	if p.TopP != nil {
		args = append(args, "--top-p", formatFloat(*p.TopP))
	}
	if p.TopK != nil {
		args = append(args, "--top-k", strconv.Itoa(*p.TopK))
	}
	if p.MinP != nil {
		args = append(args, "--min-p", formatFloat(*p.MinP))
	}
	if p.PresencePenalty != nil {
		args = append(args, "--presence-penalty", formatFloat(*p.PresencePenalty))
	}
	if p.RepetitionPenalty != nil {
		args = append(args, "--repeat-penalty", formatFloat(*p.RepetitionPenalty))
	}
	if p.FrequencyPenalty != nil {
		args = append(args, "--frequency-penalty", formatFloat(*p.FrequencyPenalty))
	}
	if p.Seed != nil {
		args = append(args, "--seed", strconv.Itoa(*p.Seed))
	}
	if p.RepeatLastN != nil {
		args = append(args, "--repeat-last-n", strconv.Itoa(*p.RepeatLastN))
	}
	if p.FlashAttn {
		args = append(args, "--flash-attn", "on")
	}
	if p.GPULayers > 0 {
		args = append(args, "--n-gpu-layers", strconv.Itoa(p.GPULayers))
	}
	if p.MMap != nil {
		if *p.MMap {
			args = append(args, "--mmap")
		} else {
			args = append(args, "--no-mmap")
		}
	}
	if p.KVOffload != nil {
		if *p.KVOffload {
			args = append(args, "--kv-offload")
		} else {
			args = append(args, "--no-kv-offload")
		}
	}
	if p.Parallel != nil {
		args = append(args, "--parallel", strconv.Itoa(*p.Parallel))
	}
	if p.ContBatching != nil {
		if *p.ContBatching {
			args = append(args, "--cont-batching")
		} else {
			args = append(args, "--no-cont-batching")
		}
	}
	if p.CachePrompt != nil {
		if *p.CachePrompt {
			args = append(args, "--cache-prompt")
		} else {
			args = append(args, "--no-cache-prompt")
		}
	}
	if p.CacheRAM != nil {
		args = append(args, "--cache-ram", strconv.Itoa(*p.CacheRAM))
	}
	if p.Reasoning != "" {
		args = append(args, "--reasoning", p.Reasoning)
	}
	if p.ReasoningBudget != nil {
		args = append(args, "--reasoning-budget", strconv.Itoa(*p.ReasoningBudget))
	}
	if p.ReasoningFormat != "" {
		args = append(args, "--reasoning-format", p.ReasoningFormat)
	}
	if p.CacheTypeK != "" {
		args = append(args, "--cache-type-k", p.CacheTypeK)
	}
	if p.CacheTypeV != "" {
		args = append(args, "--cache-type-v", p.CacheTypeV)
	}
	args = append(args, p.ExtraArgs...)
	return args
}

// Start launches bin (typically "llama-server") with args from the given
// profile, detached from the parent process group so it survives the CLI
// invocation exiting, with stdout/stderr redirected to logPath.
func Start(bin string, m models.Model, p models.Profile, logPath string) (pid int, err error) {
	logFile, err := os.Create(logPath)
	if err != nil {
		return 0, fmt.Errorf("create log file %s: %w", logPath, err)
	}

	cmd := exec.Command(bin, BuildArgs(m, p)...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	configureProcess(cmd)
	if m.CacheDir != "" {
		cmd.Env = append(os.Environ(), "LLAMA_CACHE="+m.CacheDir)
	}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return 0, fmt.Errorf("start %s: %w", bin, err)
	}

	// The child now owns the log file's lifetime; the parent CLI process
	// exits shortly after Start returns.
	go cmd.Wait()

	return cmd.Process.Pid, nil
}

// Stop sends SIGTERM to the process group led by pid and waits for it to
// actually exit, escalating to SIGKILL if it hasn't died within a grace
// period. It only returns success once the process is confirmed gone —
// trusting SIGTERM alone let a previous instance linger, holding its GPU
// memory, while llmctl believed it had already stopped.
func Stop(pid int) error {
	if err := terminateProcess(pid); err != nil {
		return fmt.Errorf("stop pid %d: %w", pid, err)
	}

	if awaitDeath(pid, 2*time.Second) {
		return nil
	}

	if err := killProcess(pid); err != nil {
		return fmt.Errorf("kill pid %d: %w", pid, err)
	}
	if awaitDeath(pid, time.Second) {
		return nil
	}

	return fmt.Errorf("pid %d did not exit after termination", pid)
}

func awaitDeath(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !IsAlive(pid) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return !IsAlive(pid)
}

// IsAlive reports whether a process with the given PID is still running.
func IsAlive(pid int) bool {
	return isProcessAlive(pid)
}

// TailLog returns the last n non-empty lines of the file at path.
func TailLog(path string, n int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n"), nil
}
