package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/G33kM4sT3r/faas/internal/ui"
)

// maxInvokeResponseBytes caps how much of a response body invoke will read
// into memory. 10 MiB is generous for any sane function response and prevents
// a misbehaving function from OOMing the client.
const maxInvokeResponseBytes int64 = 10 << 20

var ( //nolint:gochecknoglobals // cobra flag variables
	invokeMethod string
	invokeData   string
	invokeHeader []string
	invokePath   string
)

var invokeCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra command
	Use:   "invoke [func]",
	Short: "Invoke a deployed function over HTTP",
	Args:  cobra.ExactArgs(1),
	RunE:  runInvoke,
}

func setupInvokeFlags() {
	invokeCmd.Flags().StringVarP(&invokeMethod, "method", "X", "POST", "HTTP method")
	invokeCmd.Flags().StringVarP(&invokeData, "data", "d", "", `request body (literal, or @path/to/file.json)`)
	invokeCmd.Flags().StringSliceVarP(&invokeHeader, "header", "H", nil, "extra request header (KEY: VALUE)")
	invokeCmd.Flags().StringVar(&invokePath, "path", "/", "request path")
}

func runInvoke(cmd *cobra.Command, args []string) error {
	name := args[0]
	fn, err := store.Get(name)
	if err != nil {
		return ui.Errorf(fmt.Sprintf("function %q not found", name), "list with: faas ls")
	}

	var body io.Reader
	switch {
	case invokeData == "":
		body = nil
	case strings.HasPrefix(invokeData, "@"):
		f, err := os.Open(invokeData[1:]) //nolint:gosec // user-supplied data file is intended
		if err != nil {
			return fmt.Errorf("opening data file: %w", err)
		}
		defer func() { _ = f.Close() }()
		body = f
	default:
		body = strings.NewReader(invokeData)
	}

	url := fmt.Sprintf("http://localhost:%d%s", fn.Port, invokePath)
	return invokeURL(cmd.Context(), url, invokeMethod, body, os.Stdout)
}

// invokeURL issues the HTTP request and writes a pretty-printed response body.
// Lowercase so command-level tests can drive it against an httptest.Server.
func invokeURL(ctx context.Context, url, method string, body io.Reader, out io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	for _, h := range invokeHeader {
		k, v, ok := strings.Cut(h, ":")
		if !ok {
			return fmt.Errorf("bad header %q (want KEY: VALUE)", h)
		}
		req.Header.Set(strings.TrimSpace(k), strings.TrimSpace(v))
	}
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	limited := io.LimitReader(resp.Body, maxInvokeResponseBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	truncated := int64(len(raw)) > maxInvokeResponseBytes
	if truncated {
		raw = raw[:maxInvokeResponseBytes]
	}

	_, _ = fmt.Fprintf(out, "%s %d %s\n", ui.SymbolSuccess, resp.StatusCode, resp.Status)
	if truncated {
		_, _ = fmt.Fprintf(out, "%s response truncated at %d bytes\n", ui.SymbolWarning, maxInvokeResponseBytes)
	}

	var tmp any
	if json.Unmarshal(raw, &tmp) == nil {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		_ = enc.Encode(tmp)
		_, _ = out.Write(buf.Bytes())
		return nil
	}
	_, _ = out.Write(raw)
	if len(raw) > 0 && raw[len(raw)-1] != '\n' {
		_, _ = fmt.Fprintln(out)
	}
	return nil
}
