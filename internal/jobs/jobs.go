package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusSlicing   Status = "slicing"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type Job struct {
	ID        string    `json:"id"`
	Status    Status    `json:"status"`
	Progress  int       `json:"progress"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	CancelFunc  context.CancelFunc `json:"-"`
	ResultFiles []string           `json:"-"`
	Workdir     string             `json:"-"`
}

type Store struct {
	mu      sync.RWMutex
	jobs    map[string]*Job
	baseDir string
}

func NewStore() *Store {
	return &Store{jobs: make(map[string]*Job)}
}

func NewPersistentStore(baseDir string) *Store {
	s := NewStore()
	s.baseDir = baseDir
	_ = os.MkdirAll(baseDir, 0o755)
	s.loadFromDisk()
	return s
}

func randomID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405")))
	}
	return hex.EncodeToString(bytes)
}

func (s *Store) Create() (*Job, error) {
	now := time.Now()
	job := &Job{
		ID:        randomID(),
		Status:    StatusPending,
		Progress:  0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()

	s.persist(job)
	return job, nil
}

func (s *Store) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	copy := *job
	copy.ResultFiles = append([]string(nil), job.ResultFiles...)
	return &copy, true
}

func (s *Store) UpdateStatus(id string, status Status, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[id]; ok {
		job.Status = status
		job.UpdatedAt = time.Now()
		if status == StatusCompleted {
			job.Progress = 100
		}
		if err != nil {
			job.Error = err.Error()
		}
		s.persist(job)
	}
}

func (s *Store) UpdateProgress(id string, progress int) {
	if progress < 0 {
		progress = 0
	}
	if progress > 99 {
		progress = 99
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[id]; ok {
		if progress > job.Progress {
			job.Progress = progress
		}
		if job.Status == StatusPending {
			job.Status = StatusSlicing
		}
		job.UpdatedAt = time.Now()
		s.persist(job)
	}
}

func (s *Store) SetResult(id string, files []string, workdir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[id]; ok {
		job.ResultFiles = append([]string(nil), files...)
		job.Workdir = workdir
		job.UpdatedAt = time.Now()
		s.persist(job)
	}
}

func (s *Store) SetCancelFunc(id string, cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[id]; ok {
		job.CancelFunc = cancel
	}
}

func (s *Store) Cancel(id string) error {
	s.mu.RLock()
	job, ok := s.jobs[id]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("job not found")
	}
	if job.CancelFunc != nil && (job.Status == StatusPending || job.Status == StatusSlicing) {
		job.CancelFunc()
		s.UpdateStatus(id, StatusFailed, fmt.Errorf("job cancelled by user"))
	}
	return nil
}

func (s *Store) persist(job *Job) {
	if s.baseDir == "" {
		return
	}
	data, err := json.Marshal(job)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(s.baseDir, job.ID+".json"), data, 0o644)
}

func (s *Store) loadFromDisk() {
	if s.baseDir == "" {
		return
	}
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.baseDir, entry.Name()))
		if err != nil {
			continue
		}
		var job Job
		if err := json.Unmarshal(data, &job); err != nil {
			continue
		}
		if job.Status != StatusCompleted && job.Status != StatusFailed {
			job.Status = StatusFailed
			job.Error = "job lost after restart"
		}
		s.jobs[job.ID] = &job
	}
}

func (s *Store) Stats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]int{
		"total":     len(s.jobs),
		"pending":   0,
		"slicing":   0,
		"completed": 0,
		"failed":    0,
	}
	for _, job := range s.jobs {
		stats[string(job.Status)]++
	}
	return stats
}

func (s *Store) Cleanup(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	for id, job := range s.jobs {
		if (job.Status == StatusCompleted || job.Status == StatusFailed) && job.UpdatedAt.Before(cutoff) {
			if s.baseDir != "" {
				_ = os.Remove(filepath.Join(s.baseDir, id+".json"))
			}
			delete(s.jobs, id)
		}
	}
}
