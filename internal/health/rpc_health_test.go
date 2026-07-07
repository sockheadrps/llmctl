package health

import (
	"net"
	"os"
	"testing"
)

func TestCheckRPCServerUp(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	pid := os.Getpid()

	if got := CheckRPCServer("127.0.0.1", port, pid); got != StatusUp {
		t.Errorf("got %q, want %q", got, StatusUp)
	}
}

func TestCheckRPCServerPortUnreachable(t *testing.T) {
	// Bind then immediately close to guarantee the port is free.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	pid := os.Getpid()
	if got := CheckRPCServer("127.0.0.1", port, pid); got != StatusDown {
		t.Errorf("got %q, want %q (port closed)", got, StatusDown)
	}
}

func TestCheckRPCServerDeadPID(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	// PID 0 is never a valid live process.
	if got := CheckRPCServer("127.0.0.1", port, 0); got != StatusDown {
		t.Errorf("got %q, want %q (dead PID)", got, StatusDown)
	}
}
