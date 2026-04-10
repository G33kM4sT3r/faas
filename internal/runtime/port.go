package runtime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"
)

// CheckPortAvailable returns nil if the port is free on 127.0.0.1, an error
// otherwise. Only an ECONNREFUSED dial is treated as "free"; every other
// error (timeout, cancellation, permission denied, network unreachable) is
// propagated through %w so callers can distinguish via errors.Is.
func CheckPortAvailable(ctx context.Context, port int) error {
	dialer := net.Dialer{Timeout: 500 * time.Millisecond}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err == nil {
		_ = conn.Close()
		return fmt.Errorf("port %d is already in use", port)
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return nil
	}
	return fmt.Errorf("probing port %d: %w", port, err)
}
