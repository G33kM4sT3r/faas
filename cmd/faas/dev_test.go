package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchDebouncesCoalescesRapidWrites(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "handler.py")
	if err := os.WriteFile(f, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events := make(chan struct{}, 10)
	go func() {
		_ = watchFile(ctx, f, 100*time.Millisecond, func() {
			events <- struct{}{}
		})
	}()

	time.Sleep(50 * time.Millisecond)

	for range 5 {
		if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	select {
	case <-events:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected one debounced event, got none")
	}

	select {
	case <-events:
		t.Fatal("expected only one event, got multiple")
	case <-time.After(300 * time.Millisecond):
	}
}
