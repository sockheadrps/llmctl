package util

import (
	"fmt"
	"net"
)

// IsPortFree reports whether a TCP port is currently free to bind on localhost.
func IsPortFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// FindFreePort returns the first free port at or after start, scanning at
// most 1000 ports.
func FindFreePort(start int) (int, error) {
	for port := start; port < start+1000; port++ {
		if IsPortFree(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port found starting at %d", start)
}
