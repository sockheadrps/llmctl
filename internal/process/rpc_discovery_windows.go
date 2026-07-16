//go:build windows

package process

import (
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
		exeName = "ggml-rpc-server.exe"
	}
	if !strings.HasSuffix(strings.ToLower(exeName), ".exe") {
		exeName += ".exe"
	}

	escapedName := psSingleQuote(exeName)
	escapedHost := psSingleQuote(strings.TrimSpace(host))
	escapedPort := psSingleQuote(strconv.Itoa(port))

	script := fmt.Sprintf(`
$name = %s
$host = %s
$port = %s
Get-CimInstance Win32_Process |
	Where-Object {
		$_.Name -ieq $name -and
		$_.CommandLine -and
		$_.CommandLine -like "*-H $host*" -and
		$_.CommandLine -like "*-p $port*"
	} |
	Select-Object -First 1 -ExpandProperty ProcessId
`, escapedName, escapedHost, escapedPort)

	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return 0, false, nil
	}
	pidStr := strings.TrimSpace(string(out))
	if pidStr == "" {
		return 0, false, nil
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, false, nil
	}
	return pid, true, nil
}

func psSingleQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}
