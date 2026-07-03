package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

	CancelFunc context.CancelFunc `json:"-"`
	ResultFiles []string           `json:"-"`
	Workdir     string             `json:"-"`
}

type Store struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewStore() *Store {
	return &Store{jobs: make(map[string]*Job)}
}

func (s *Store) Create() (*Job, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}

	now := time.Now()
	job := &Job{
		ID:        hex.EncodeToString(bytes),
		Status:    StatusPending,
		Progress:  0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()

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
	}
}

func (s *Store) SetResult(id string, files []string, workdir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[id]; ok {
		job.ResultFiles = append([]string(nil), files...)
		job.Workdir = workdir
		job.UpdatedAt = time.Now()
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
