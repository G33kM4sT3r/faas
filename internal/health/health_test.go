package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWaitForHealthySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := WaitForHealthy(context.Background(), server.URL+"/health", Options{
		Interval: 100 * time.Millisecond,
		Timeout:  2 * time.Second,
	})
	if err != nil {
		t.Errorf("expected healthy, got error: %v", err)
	}
}

func TestWaitForHealthyTimeout(t *testing.T) {
	err := WaitForHealthy(context.Background(), "http://127.0.0.1:1/health", Options{
		Interval: 50 * time.Millisecond,
		Timeout:  200 * time.Millisecond,
	})
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestWaitForHealthyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := WaitForHealthy(ctx, "http://127.0.0.1:1/health", Options{
		Interval: 50 * time.Millisecond,
		Timeout:  5 * time.Second,
	})
	if err == nil {
		t.Error("expected cancellation error")
	}
}
