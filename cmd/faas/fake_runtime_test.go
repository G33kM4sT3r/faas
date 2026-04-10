package main

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/G33kM4sT3r/faas/internal/runtime"
)

// fakeRuntime is a programmable runtime.Runtime implementation for tests.
// Each Fn field overrides the corresponding method; nil means "return zero value".
type fakeRuntime struct {
	BuildFn       func(ctx context.Context, opts runtime.BuildOpts) (runtime.Image, error)
	StartFn       func(ctx context.Context, opts runtime.StartOpts) (runtime.Container, error)
	StopFn        func(ctx context.Context, id string) error
	RemoveFn      func(ctx context.Context, id string) error
	RemoveImageFn func(ctx context.Context, tag string) error
	StatusFn      func(ctx context.Context, id string) (runtime.ContainerStatus, error)
	LogsFn        func(ctx context.Context, id string, opts runtime.LogOpts) (io.ReadCloser, error)

	StopCalls   []string
	RemoveCalls []string
}

func (f *fakeRuntime) Build(ctx context.Context, opts runtime.BuildOpts) (runtime.Image, error) {
	if f.BuildFn != nil {
		return f.BuildFn(ctx, opts)
	}
	return runtime.Image{Tag: opts.Tag}, nil
}

func (f *fakeRuntime) Start(ctx context.Context, opts runtime.StartOpts) (runtime.Container, error) {
	if f.StartFn != nil {
		return f.StartFn(ctx, opts)
	}
	return runtime.Container{ID: "fake-" + opts.Name, Port: opts.Port}, nil
}

func (f *fakeRuntime) Stop(ctx context.Context, id string) error {
	f.StopCalls = append(f.StopCalls, id)
	if f.StopFn != nil {
		return f.StopFn(ctx, id)
	}
	return nil
}

func (f *fakeRuntime) Remove(ctx context.Context, id string) error {
	f.RemoveCalls = append(f.RemoveCalls, id)
	if f.RemoveFn != nil {
		return f.RemoveFn(ctx, id)
	}
	return nil
}

func (f *fakeRuntime) RemoveImage(ctx context.Context, tag string) error {
	if f.RemoveImageFn != nil {
		return f.RemoveImageFn(ctx, tag)
	}
	return nil
}

func (f *fakeRuntime) Status(ctx context.Context, id string) (runtime.ContainerStatus, error) {
	if f.StatusFn != nil {
		return f.StatusFn(ctx, id)
	}
	return runtime.ContainerStatus{Running: true}, nil
}

func (f *fakeRuntime) Logs(ctx context.Context, id string, opts runtime.LogOpts) (io.ReadCloser, error) {
	if f.LogsFn != nil {
		return f.LogsFn(ctx, id, opts)
	}
	return io.NopCloser(bytes.NewReader(nil)), nil
}

// Compile-time guarantee that fakeRuntime satisfies the runtime.Runtime interface.
var _ runtime.Runtime = (*fakeRuntime)(nil)

// withFakeRuntime replaces newRuntime for the duration of the test.
// The returned function restores the previous constructor.
func withFakeRuntime(t *testing.T, f *fakeRuntime) func() {
	t.Helper()
	prev := newRuntime
	newRuntime = func(ctx context.Context) (runtime.Runtime, error) { return f, nil }
	return func() { newRuntime = prev }
}
