package slicer

import (
	"context"
	"testing"
)

func TestSliceReturnsBusyWhenLocked(t *testing.T) {
	service := &Service{}
	service.mu.Lock()
	defer service.mu.Unlock()

	_, err := service.Slice(context.Background(), "model.stl", []byte("model"), Settings{})
	if err == nil {
		t.Fatalf("expected busy error")
	}
	if err.Error() != "Slicer is busy" {
		t.Fatalf("expected busy error, got %v", err)
	}
}

func TestSliceStatusDefaultsIdle(t *testing.T) {
	service := &Service{}
	if service.Status().Status != StatusIdle {
		t.Fatalf("expected idle status")
	}
}
