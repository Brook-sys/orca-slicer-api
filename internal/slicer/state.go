package slicer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type JobStatus string

const (
	StatusIdle       JobStatus = "idle"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusCancelled  JobStatus = "cancelled"
)

type JobState struct {
	Status       JobStatus `json:"status"`
	StartedAt    string    `json:"startedAt,omitempty"`
	FinishedAt   string    `json:"finishedAt,omitempty"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	Files        []string  `json:"files,omitempty"`
	Metadata     Metadata  `json:"metadata,omitempty"`
}

type StateStore struct {
	path string
	mu   sync.RWMutex
}

func NewStateStore(dataPath string) *StateStore {
	return &StateStore{path: filepath.Join(dataPath, "slice-status.json")}
}

func (s *StateStore) Get() JobState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		return JobState{Status: StatusIdle}
	}

	var state JobState
	if err := json.Unmarshal(data, &state); err != nil {
		return JobState{Status: StatusIdle}
	}
	if state.Status == "" {
		state.Status = StatusIdle
	}
	return state
}

func (s *StateStore) Set(state JobState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func nowString() string {
	return time.Now().UTC().Format(time.RFC3339)
}
