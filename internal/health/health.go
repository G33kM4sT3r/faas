// Package health provides health check polling for deployed functions.
package health

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Options configures health check behavior.
type Options struct {
	Interval time.Duration // polling interval (default: 500ms)
	Timeout  time.Duration // total timeout (default: 30s)
}

// WaitForHealthy polls the given URL until it returns 200 or times out.
func WaitForHealthy(ctx context.Context, url string, opts Options) error {
	if opts.Interval == 0 {
		opts.Interval = 500 * time.Millisecond
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	client := &http.Client{Timeout: opts.Interval}
	deadline := time.After(opts.Timeout)
	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("health check cancelled: %w", ctx.Err())
		case <-deadline:
			return fmt.Errorf("health check timed out after %s", opts.Timeout)
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
			if err != nil {
				continue
			}
			// gosec G704: URL is the local container's health endpoint
			// (http://localhost:<port>/health), constructed by the caller from
			// runtime-resolved state — not user web input. No SSRF surface.
			resp, err := client.Do(req) //nolint:gosec // local health probe, not external SSRF

			if err != nil {
				continue
			}
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
	}
}
