package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	store := New(path)

	fn := Function{
		Name:        "hello",
		Path:        "/tmp/hello.py",
		Language:    "python",
		ContainerID: "abc123",
		Port:        52341,
		Status:      StatusHealthy,
		CreatedAt:   time.Now().UTC().Truncate(time.Second),
	}

	if err := store.Set(&fn); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	store2 := New(path)
	got, err := store2.Get("hello")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.ContainerID != "abc123" {
		t.Errorf("expected container_id abc123, got %q", got.ContainerID)
	}
	if got.Port != 52341 {
		t.Errorf("expected port 52341, got %d", got.Port)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	store := New(filepath.Join(dir, "state.json"))

	_ = store.Set(&Function{Name: "a", Status: StatusHealthy})
	_ = store.Set(&Function{Name: "b", Status: StatusStopped})

	all, err := store.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 functions, got %d", len(all))
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	store := New(filepath.Join(dir, "state.json"))

	_ = store.Set(&Function{Name: "hello"})

	if err := store.Remove("hello"); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	_, err := store.Get("hello")
	if err == nil {
		t.Error("expected error after removal")
	}
}

func TestGetNotFound(t *testing.T) {
	dir := t.TempDir()
	store := New(filepath.Join(dir, "state.json"))

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent function")
	}
}
