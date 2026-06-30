package slicer

import "testing"

func TestStateStore(t *testing.T) {
	store := NewStateStore(t.TempDir())
	state := JobState{Status: StatusProcessing, StartedAt: "now"}
	if err := store.Set(state); err != nil {
		t.Fatal(err)
	}

	got := store.Get()
	if got.Status != StatusProcessing {
		t.Fatalf("expected processing, got %s", got.Status)
	}
	if got.StartedAt != "now" {
		t.Fatalf("expected startedAt to persist")
	}
}
