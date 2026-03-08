package state

import (
	"errors"
	"os"
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

func TestUpdateStatus(t *testing.T) {
	dir := t.TempDir()
	store := New(filepath.Join(dir, "state.json"))

	_ = store.Set(&Function{Name: "hello", Status: StatusBuilding})

	if err := store.UpdateStatus("hello", StatusHealthy); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	fn, err := store.Get("hello")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if fn.Status != StatusHealthy {
		t.Errorf("expected healthy, got %q", fn.Status)
	}
}

func TestUpdateStatusNotFound(t *testing.T) {
	dir := t.TempDir()
	store := New(filepath.Join(dir, "state.json"))

	err := store.UpdateStatus("nonexistent", StatusHealthy)
	if err == nil {
		t.Error("expected error for nonexistent function")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestLoadCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := New(path)
	_, err := store.Get("anything")
	if err == nil {
		t.Error("expected error for corrupt JSON")
	}
}

func TestLoadNullFunctions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte(`{"functions":null}`), 0o644); err != nil {
		t.Fatal(err)
	}

	store := New(path)
	fns, err := store.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fns) != 0 {
		t.Errorf("expected empty list, got %d", len(fns))
	}
}

func TestSetMultipleAndList(t *testing.T) {
	dir := t.TempDir()
	store := New(filepath.Join(dir, "state.json"))

	for _, name := range []string{"a", "b", "c"} {
		_ = store.Set(&Function{Name: name, Language: "python", Port: 8080})
	}

	fns, _ := store.List()
	if len(fns) != 3 {
		t.Errorf("expected 3 functions, got %d", len(fns))
	}
}

func TestSetOverwrite(t *testing.T) {
	dir := t.TempDir()
	store := New(filepath.Join(dir, "state.json"))

	_ = store.Set(&Function{Name: "hello", Port: 8080})
	_ = store.Set(&Function{Name: "hello", Port: 9090})

	fn, _ := store.Get("hello")
	if fn.Port != 9090 {
		t.Errorf("expected port 9090, got %d", fn.Port)
	}
}

func TestRemoveNonexistent(t *testing.T) {
	dir := t.TempDir()
	store := New(filepath.Join(dir, "state.json"))

	err := store.Remove("nope")
	if err != nil {
		t.Errorf("remove of nonexistent should not error, got: %v", err)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "state.json")

	store := New(path)
	err := store.Set(&Function{Name: "hello"})
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Error("state file should have been created with parent dirs")
	}
}

func TestGetNotFoundError(t *testing.T) {
	dir := t.TempDir()
	store := New(filepath.Join(dir, "state.json"))

	_, err := store.Get("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListEmpty(t *testing.T) {
	dir := t.TempDir()
	store := New(filepath.Join(dir, "state.json"))

	fns, err := store.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fns) != 0 {
		t.Errorf("expected empty list, got %d", len(fns))
	}
}
