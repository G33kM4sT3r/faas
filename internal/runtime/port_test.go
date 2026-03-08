package runtime

import (
	"context"
	"net"
	"testing"
)

func TestCheckPortAvailableOnFreePort(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	if err := CheckPortAvailable(context.Background(), port); err != nil {
		t.Errorf("port %d should be available: %v", port, err)
	}
}

func TestCheckPortAvailableOnUsedPort(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	port := l.Addr().(*net.TCPAddr).Port
	if err := CheckPortAvailable(context.Background(), port); err == nil {
		t.Errorf("port %d should be in use", port)
	}
}
