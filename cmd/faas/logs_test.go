package main

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/G33kM4sT3r/faas/internal/runtime"
)

func TestRunLogsNotFound(t *testing.T) {
	setupCmdEnv(t)
	defer withFakeRuntime(t, &fakeRuntime{})()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runLogs(cmd, []string{"missing"})
	if err == nil {
		t.Fatal("expected error for missing function")
	}
	if !strings.Contains(err.Error(), `Function "missing" not found`) {
		t.Errorf("expected not-found message, got: %v", err)
	}
}

func TestRunLogsStreams(t *testing.T) {
	setupCmdEnv(t)
	seedFunction(t, "ping", "id-ping", "img-ping", 5200)

	logLines := `{"level":"info","msg":"hello"}
{"level":"warn","msg":"watch out"}
plain text line
`
	fake := &fakeRuntime{
		LogsFn: func(ctx context.Context, id string, opts runtime.LogOpts) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewBufferString(logLines)), nil
		},
	}
	defer withFakeRuntime(t, fake)()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	logsFollow = false
	logsNoFollow = true
	logsJSON = false
	logsLevel = ""
	t.Cleanup(func() { logsFollow = true; logsNoFollow = false; logsJSON = false; logsLevel = "" })

	if err := runLogs(cmd, []string{"ping"}); err != nil {
		t.Errorf("runLogs failed: %v", err)
	}
}

func TestRunLogsLevelFilter(t *testing.T) {
	setupCmdEnv(t)
	seedFunction(t, "filtered", "id-f", "img-f", 5201)

	logLines := `{"level":"debug","msg":"noise"}
{"level":"error","msg":"boom"}
`
	fake := &fakeRuntime{
		LogsFn: func(ctx context.Context, id string, opts runtime.LogOpts) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewBufferString(logLines)), nil
		},
	}
	defer withFakeRuntime(t, fake)()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	logsLevel = "warn"
	t.Cleanup(func() { logsLevel = "" })

	if err := runLogs(cmd, []string{"filtered"}); err != nil {
		t.Errorf("runLogs --level=warn failed: %v", err)
	}
}
