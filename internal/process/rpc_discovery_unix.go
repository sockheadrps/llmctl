//go:build !windows

package process

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// FindRPCServerPID returns the PID of a running ggml-rpc-server instance that
// matches the provided binary name and host/port, if one can be found.
func FindRPCServerPID(bin, host string, port int) (int, bool, error) {
	exeName := filepath.Base(strings.TrimSpace(bin))
	if exeName == "." || exeName == string(filepath.Separator) {
		exeName = "ggml-rpc-server"
	}

	out, err := exec.Command("ps", "-eo", "pid=,args=").Output()
	if err != nil {
		return 0, false, fmt.Errorf("list processes: %w", err)
	}

	hostNeedle := strings.TrimSpace(host)
	portNeedle := fmt.Sprintf("-p %d", port)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		args := strings.Join(fields[1:], " ")
		if !strings.Contains(args, exeName) {
			continue
		}
		if hostNeedle != "" && !strings.Contains(args, "-H "+hostNeedle) {
			continue
		}
		if !strings.Contains(args, portNeedle) {
			continue
		}
		return pid, true, nil
	}
	if err := scanner.Err(); err != nil {
		return 0, false, err
	}
	return 0, false, nil
}
