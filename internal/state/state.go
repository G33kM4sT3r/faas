// Package state manages persistent function deployment state.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// StatusBuilding indicates the function image is being built.
	StatusBuilding Status = "building"
	// StatusStarting indicates the container is starting.
	StatusStarting Status = "starting"
	// StatusHealthy indicates the function passed health checks.
	StatusHealthy Status = "healthy"
	// StatusUnhealthy indicates the function failed health checks.
	StatusUnhealthy Status = "unhealthy"
	// StatusStopped indicates the function has been stopped.
	StatusStopped Status = "stopped"
	// StatusError indicates the function encountered an error.
	StatusError Status = "error"
)

// ErrNotFound is returned when a function is not found in state.
var ErrNotFound = errors.New("function not found")

// Status represents the lifecycle status of a deployed function.
type Status string

// Function represents a deployed function's state.
type Function struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Language    string    `json:"language"`
	ContainerID string    `json:"container_id"`
	ImageID     string    `json:"image_id"`
	Port        int       `json:"port"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// Store manages persistent function state.
type Store struct {
	path string
}

type stateFile struct {
	Functions map[string]Function `json:"functions"`
}

// New creates a Store backed by the given file path.
func New(path string) *Store {
	return &Store{path: path}
}

// Get retrieves a function by name.
func (s *Store) Get(name string) (Function, error) {
	sf, err := s.load()
	if err != nil {
		return Function{}, err
	}

	fn, ok := sf.Functions[name]
	if !ok {
		return Function{}, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return fn, nil
}

// Set stores or updates a function.
func (s *Store) Set(fn *Function) error {
	sf, err := s.load()
	if err != nil {
		return err
	}

	sf.Functions[fn.Name] = *fn
	return s.save(sf)
}

// Remove deletes a function from state.
func (s *Store) Remove(name string) error {
	sf, err := s.load()
	if err != nil {
		return err
	}

	delete(sf.Functions, name)
	return s.save(sf)
}

// List returns all stored functions.
func (s *Store) List() ([]Function, error) {
	sf, err := s.load()
	if err != nil {
		return nil, err
	}

	fns := make([]Function, 0, len(sf.Functions))
	for name := range sf.Functions {
		fns = append(fns, sf.Functions[name])
	}
	return fns, nil
}

// UpdateStatus updates only the status of a function.
func (s *Store) UpdateStatus(name string, status Status) error {
	sf, err := s.load()
	if err != nil {
		return err
	}

	fn, ok := sf.Functions[name]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, name)
	}

	fn.Status = status
	sf.Functions[name] = fn
	return s.save(sf)
}

func (s *Store) load() (stateFile, error) {
	sf := stateFile{Functions: make(map[string]Function)}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return sf, nil
		}
		return sf, fmt.Errorf("reading state: %w", err)
	}

	if err := json.Unmarshal(data, &sf); err != nil {
		return sf, fmt.Errorf("parsing state: %w", err)
	}

	if sf.Functions == nil {
		sf.Functions = make(map[string]Function)
	}
	return sf, nil
}

func (s *Store) save(sf stateFile) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("renaming state: %w", err)
	}

	return nil
}
