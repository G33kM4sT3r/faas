package runtime

import (
	"context"
	"errors"
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

func TestCheckPortAvailableCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := CheckPortAvailable(ctx, 1)
	if err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
