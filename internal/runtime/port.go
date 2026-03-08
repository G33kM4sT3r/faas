package runtime

import (
	"context"
	"fmt"
	"net"
	"time"
)

// CheckPortAvailable returns an error if the port is already in use.
func CheckPortAvailable(ctx context.Context, port int) error {
	dialer := net.Dialer{Timeout: 500 * time.Millisecond}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil // Connection failed = port is free.
	}
	_ = conn.Close()
	return fmt.Errorf("port %d is already in use", port)
}
