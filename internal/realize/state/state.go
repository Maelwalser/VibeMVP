package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const fileName = ".realize-progress.json"

// Store tracks which task IDs have been successfully completed.
// It is safe for concurrent use.
type Store struct {
	mu        sync.Mutex
	path      string
	completed map[string]bool
}

// Load reads the state file at <dir>/.realize-progress.json, or starts fresh
// if the file does not exist yet.
func Load(dir string) (*Store, error) {
	path := filepath.Join(dir, fileName)
	s := &Store{
		path:      path,
		completed: make(map[string]bool),
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state file %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &s.completed); err != nil {
		return nil, fmt.Errorf("parse state file %s: %w", path, err)
	}
	return s, nil
}

// IsCompleted reports whether taskID was already successfully completed in a
// prior run.
func (s *Store) IsCompleted(taskID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.completed[taskID]
}

// MarkCompleted records taskID as completed and immediately persists the state
// so a crash after this call will not re-run the task.
func (s *Store) MarkCompleted(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completed[taskID] = true
	return s.flush()
}

// CompletedCount returns the number of task IDs that were previously completed.
func (s *Store) CompletedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.completed)
}

// flush serialises the completed set to disk. Caller must hold mu.
func (s *Store) flush() error {
	data, err := json.MarshalIndent(s.completed, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("write state file %s: %w", s.path, err)
	}
	return nil
}
