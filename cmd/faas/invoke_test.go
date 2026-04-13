package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunInvokeNotFound(t *testing.T) {
	setupCmdEnv(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runInvoke(cmd, []string{"ghost"})
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if !strings.Contains(err.Error(), `function "ghost" not found`) {
		t.Errorf("expected 'function ghost not found', got: %v", err)
	}
}

func TestInvokeURLBadHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	prevHeader := invokeHeader
	invokeHeader = []string{"NoColonHere"}
	t.Cleanup(func() { invokeHeader = prevHeader })

	var out strings.Builder
	err := invokeURL(context.Background(), srv.URL, "POST", nil, &out)
	if err == nil || !strings.Contains(err.Error(), "bad header") {
		t.Errorf("expected 'bad header' error, got: %v", err)
	}
}

func TestInvokeURLNonJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("plain old text\n"))
	}))
	defer srv.Close()

	var out strings.Builder
	err := invokeURL(context.Background(), srv.URL, "GET", nil, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "plain old text") {
		t.Errorf("expected raw text in output, got: %q", out.String())
	}
}

// TestInvokeURLTruncatesLargeResponse verifies the size cap added in the
// audit fix: bodies over maxInvokeResponseBytes get truncated and a warning
// is printed.
func TestInvokeURLTruncatesLargeResponse(t *testing.T) {
	huge := bytes.Repeat([]byte("A"), int(maxInvokeResponseBytes+1024))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(huge)
	}))
	defer srv.Close()

	var out strings.Builder
	err := invokeURL(context.Background(), srv.URL, "GET", nil, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "truncated") {
		t.Errorf("expected truncation warning in output, got prefix: %q", out.String()[:200])
	}
}

// TestRunInvokeSendsFileBodyWithContentLength is the regression test for the
// audit fix: passing -d @file used to stream the *os.File body directly,
// which caused Go's http client to use chunked Transfer-Encoding. The
// minimal language wrappers all size the body from Content-Length, so they
// saw 0 bytes. Reading the file into a bytes.Reader fixes both.
func TestRunInvokeSendsFileBodyWithContentLength(t *testing.T) {
	setupCmdEnv(t)

	var gotBody string
	var gotLen int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLen = r.ContentLength
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	srvURL, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(srvURL.Port())
	if err != nil {
		t.Fatal(err)
	}
	seedFunction(t, "fileinvoke", "id", "img", port)

	tmp := t.TempDir()
	payload := filepath.Join(tmp, "p.json")
	if err := os.WriteFile(payload, []byte(`{"hello":"world"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	prevData := invokeData
	prevMethod := invokeMethod
	prevPath := invokePath
	prevHeader := invokeHeader
	invokeData = "@" + payload
	invokeMethod = "POST"
	invokePath = "/"
	invokeHeader = nil
	t.Cleanup(func() {
		invokeData = prevData
		invokeMethod = prevMethod
		invokePath = prevPath
		invokeHeader = prevHeader
	})

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := runInvoke(cmd, []string{"fileinvoke"}); err != nil {
		t.Fatalf("runInvoke failed: %v", err)
	}
	if gotBody != `{"hello":"world"}` {
		t.Errorf("body: got %q, want %q", gotBody, `{"hello":"world"}`)
	}
	if gotLen != int64(len(`{"hello":"world"}`)) {
		t.Errorf("Content-Length: got %d, want %d", gotLen, len(`{"hello":"world"}`))
	}
}

func TestRunInvokeFileMissing(t *testing.T) {
	setupCmdEnv(t)
	seedFunction(t, "missingfile", "id", "img", 1)

	prev := invokeData
	invokeData = "@/no/such/file/here.json"
	t.Cleanup(func() { invokeData = prev })

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runInvoke(cmd, []string{"missingfile"})
	if err == nil || !strings.Contains(err.Error(), "reading data file") {
		t.Errorf("expected 'reading data file' error, got: %v", err)
	}
}

func TestInvokePOSTsBodyAndPrintsResponse(t *testing.T) {
	var gotBody string
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	var out strings.Builder
	err := invokeURL(context.Background(), srv.URL, "POST", strings.NewReader(`{"x":1}`), &out)
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method: got %q, want POST", gotMethod)
	}
	if gotBody != `{"x":1}` {
		t.Errorf("body: got %q, want {\"x\":1}", gotBody)
	}
	if !strings.Contains(out.String(), `"ok": true`) {
		t.Errorf("expected pretty-printed JSON in output, got: %q", out.String())
	}
}
