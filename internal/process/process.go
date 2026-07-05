// Package process builds llama-server command lines from a Profile and
// launches them as detached, logged subprocesses.
package process

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/util"
)

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// flag returns the CLI flag name to use, applying any per-profile override.
func flag(p models.Profile, def string) string {
	if p.FlagOverrides != nil {
		if override, ok := p.FlagOverrides[def]; ok && override != "" {
			return override
		}
	}
	return def
}

// BuildArgs converts a Model+Profile pair into llama-server CLI flags.
func BuildArgs(m models.Model, p models.Profile) []string {
	var args []string
	if m.IsRemote() {
		args = append(args, "-hf", m.HFRepo)
	} else {
		args = append(args, "--model", m.Path)
	}
	args = append(args, flag(p, "--port"), strconv.Itoa(p.Port))
	if p.Host != "" {
		args = append(args, flag(p, "--host"), p.Host)
	}
	if p.Alias != "" {
		args = append(args, flag(p, "--alias"), p.Alias)
	}
	if p.CtxSize > 0 {
		args = append(args, flag(p, "--ctx-size"), strconv.Itoa(p.CtxSize))
	}
	if p.BatchSize != nil {
		args = append(args, flag(p, "--batch-size"), strconv.Itoa(*p.BatchSize))
	}
	if p.UBatchSize != nil {
		args = append(args, flag(p, "--ubatch-size"), strconv.Itoa(*p.UBatchSize))
	}
	if p.Temp != nil {
		args = append(args, flag(p, "--temp"), formatFloat(*p.Temp))
	}
	if p.TopP != nil {
		args = append(args, flag(p, "--top-p"), formatFloat(*p.TopP))
	}
	if p.TopK != nil {
		args = append(args, flag(p, "--top-k"), strconv.Itoa(*p.TopK))
	}
	if p.MinP != nil {
		args = append(args, flag(p, "--min-p"), formatFloat(*p.MinP))
	}
	if p.PresencePenalty != nil {
		args = append(args, flag(p, "--presence-penalty"), formatFloat(*p.PresencePenalty))
	}
	if p.RepetitionPenalty != nil {
		args = append(args, flag(p, "--repeat-penalty"), formatFloat(*p.RepetitionPenalty))
	}
	if p.FrequencyPenalty != nil {
		args = append(args, flag(p, "--frequency-penalty"), formatFloat(*p.FrequencyPenalty))
	}
	if p.Seed != nil {
		args = append(args, flag(p, "--seed"), strconv.Itoa(*p.Seed))
	}
	if p.RepeatLastN != nil {
		args = append(args, flag(p, "--repeat-last-n"), strconv.Itoa(*p.RepeatLastN))
	}
	if p.FlashAttn {
		args = append(args, flag(p, "--flash-attn"), "on")
	}
	if p.GPULayers > 0 {
		args = append(args, flag(p, "--n-gpu-layers"), strconv.Itoa(p.GPULayers))
	}
	if p.MMap != nil {
		f := flag(p, "--mmap")
		if *p.MMap {
			args = append(args, f)
		} else {
			args = append(args, "--no-"+strings.TrimPrefix(f, "--"))
		}
	}
	if p.KVOffload != nil {
		f := flag(p, "--kv-offload")
		if *p.KVOffload {
			args = append(args, f)
		} else {
			args = append(args, "--no-"+strings.TrimPrefix(f, "--"))
		}
	}
	if p.Parallel != nil {
		args = append(args, flag(p, "--parallel"), strconv.Itoa(*p.Parallel))
	}
	if p.ContBatching != nil {
		f := flag(p, "--cont-batching")
		if *p.ContBatching {
			args = append(args, f)
		} else {
			args = append(args, "--no-"+strings.TrimPrefix(f, "--"))
		}
	}
	if p.CachePrompt != nil {
		f := flag(p, "--cache-prompt")
		if *p.CachePrompt {
			args = append(args, f)
		} else {
			args = append(args, "--no-"+strings.TrimPrefix(f, "--"))
		}
	}
	if p.CacheRAM != nil {
		args = append(args, flag(p, "--cache-ram"), strconv.Itoa(*p.CacheRAM))
	}
	if p.Reasoning != "" {
		args = append(args, flag(p, "--reasoning"), p.Reasoning)
	}
	if p.ReasoningBudget != nil {
		args = append(args, flag(p, "--reasoning-budget"), strconv.Itoa(*p.ReasoningBudget))
	}
	if p.ReasoningFormat != "" {
		args = append(args, flag(p, "--reasoning-format"), p.ReasoningFormat)
	}
	if p.CacheTypeK != "" {
		args = append(args, flag(p, "--cache-type-k"), p.CacheTypeK)
	}
	if p.CacheTypeV != "" {
		args = append(args, flag(p, "--cache-type-v"), p.CacheTypeV)
	}
	args = append(args, p.ExtraArgs...)
	return args
}

// Start launches bin (typically "llama-server") with args from the given
// profile, detached from the parent process group so it survives the CLI
// invocation exiting, with stdout/stderr redirected to logPath.
// rpcEndpoint, when non-empty, appends --rpc <endpoint> to the args.
func Start(bin string, m models.Model, p models.Profile, logPath string, rpcEndpoint string) (pid int, err error) {
	resolvedBin, err := resolveExecutable(bin)
	if err != nil {
		return 0, fmt.Errorf("start %s: %w", displayBin(bin), err)
	}

	logFile, err := os.Create(logPath)
	if err != nil {
		return 0, fmt.Errorf("create log file %s: %w", logPath, err)
	}

	args := BuildArgs(m, p)
	if rpcEndpoint != "" {
		args = append(args, "--rpc", rpcEndpoint)
	}
	cmd := exec.Command(resolvedBin, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	configureProcess(cmd)
	if m.CacheDir != "" {
		cmd.Env = append(os.Environ(), "LLAMA_CACHE="+m.CacheDir)
	}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return 0, fmt.Errorf("start %s: %w", displayBin(bin), err)
	}

	// The child now owns the log file's lifetime; the parent CLI process
	// exits shortly after Start returns.
	go cmd.Wait()

	return cmd.Process.Pid, nil
}

func resolveExecutable(bin string) (string, error) {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		bin = "llama-server"
	}

	if expanded, err := util.ExpandHome(bin); err == nil {
		bin = expanded
	}

	if resolved, err := exec.LookPath(bin); err == nil {
		return resolved, nil
	}

	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(bin), ".exe") {
		if resolved, err := exec.LookPath(bin + ".exe"); err == nil {
			return resolved, nil
		}
	}

	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	return "", fmt.Errorf("llama-server binary %q not found; set llama_server_bin in config.yaml to the full path to llama-server%s, or add it to PATH", bin, suffix)
}

func displayBin(bin string) string {
	if strings.TrimSpace(bin) == "" {
		return "llama-server"
	}
	return bin
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
