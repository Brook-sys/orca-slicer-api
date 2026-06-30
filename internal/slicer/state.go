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

type SliceDebug struct {
	StartedAt    string           `json:"startedAt,omitempty"`
	FinishedAt   string           `json:"finishedAt,omitempty"`
	Workdir      string           `json:"workdir,omitempty"`
	InputPath    string           `json:"inputPath,omitempty"`
	OutputDir    string           `json:"outputDir,omitempty"`
	Command      string           `json:"command,omitempty"`
	Args         []string         `json:"args,omitempty"`
	Settings     Settings         `json:"settings,omitempty"`
	Printer      map[string]any   `json:"printer,omitempty"`
	Preset       map[string]any   `json:"preset,omitempty"`
	Filament     map[string]any   `json:"filament,omitempty"`
	Filaments    []map[string]any `json:"filaments,omitempty"`
	Output       string           `json:"output,omitempty"`
	SlicerError  string           `json:"slicerError,omitempty"`
	ResultJSON   map[string]any   `json:"resultJson,omitempty"`
	Files        []string         `json:"files,omitempty"`
	ErrorMessage string           `json:"errorMessage,omitempty"`
}

type StateStore struct {
	path      string
	debugPath string
	mu        sync.RWMutex
}

func NewStateStore(dataPath string) *StateStore {
	return &StateStore{path: filepath.Join(dataPath, "slice-status.json"), debugPath: filepath.Join(dataPath, "slice-debug.json")}
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

func (s *StateStore) GetDebug() SliceDebug {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.debugPath)
	if err != nil {
		return SliceDebug{}
	}

	var debug SliceDebug
	if err := json.Unmarshal(data, &debug); err != nil {
		return SliceDebug{}
	}
	return debug
}

func (s *StateStore) SetDebug(debug SliceDebug) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.debugPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(debug, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.debugPath, data, 0o644)
}

func nowString() string {
	return time.Now().UTC().Format(time.RFC3339)
}
