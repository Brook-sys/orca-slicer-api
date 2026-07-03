package jobs

import (
	"testing"
	"time"
)

func TestJobStoreCreateAndGet(t *testing.T) {
	store := NewStore()
	job, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != StatusPending {
		t.Fatalf("expected pending, got %s", job.Status)
	}

	fetched, ok := store.Get(job.ID)
	if !ok || fetched.ID != job.ID {
		t.Fatal("failed to get job")
	}
}

func TestJobStoreUpdateProgress(t *testing.T) {
	store := NewStore()
	job, _ := store.Create()

	store.UpdateProgress(job.ID, 50)
	fetched, _ := store.Get(job.ID)
	if fetched.Progress != 50 || fetched.Status != StatusSlicing {
		t.Fatalf("expected progress 50 and status slicing, got %d %s", fetched.Progress, fetched.Status)
	}

	// Should cap at 99
	store.UpdateProgress(job.ID, 150)
	fetched, _ = store.Get(job.ID)
	if fetched.Progress != 99 {
		t.Fatalf("expected progress capped at 99, got %d", fetched.Progress)
	}
}

func TestJobStoreUpdateStatus(t *testing.T) {
	store := NewStore()
	job, _ := store.Create()

	store.UpdateStatus(job.ID, StatusCompleted, nil)
	fetched, _ := store.Get(job.ID)
	if fetched.Status != StatusCompleted || fetched.Progress != 100 {
		t.Fatalf("expected completed and 100%%, got %s %d", fetched.Status, fetched.Progress)
	}
}

func TestJobStoreCancel(t *testing.T) {
	store := NewStore()
	job, _ := store.Create()

	cancelled := false
	store.SetCancelFunc(job.ID, func() {
		cancelled = true
	})

	if err := store.Cancel(job.ID); err != nil {
		t.Fatal(err)
	}
	if !cancelled {
		t.Fatal("cancel func not called")
	}

	fetched, _ := store.Get(job.ID)
	if fetched.Status != StatusFailed {
		t.Fatalf("expected failed status, got %s", fetched.Status)
	}
}

func TestPersistentStore(t *testing.T) {
	dir := t.TempDir()
	store := NewPersistentStore(dir)

	job, _ := store.Create()
	store.UpdateStatus(job.ID, StatusSlicing, nil)

	// Recreate store to simulate restart
	store2 := NewPersistentStore(dir)
	fetched, ok := store2.Get(job.ID)
	if !ok {
		t.Fatal("expected job to be recovered")
	}
	if fetched.Status != StatusFailed {
		t.Fatalf("expected slicing job to become failed after restart, got %s", fetched.Status)
	}

	// Test completed job recovery
	job2, _ := store.Create()
	store.UpdateStatus(job2.ID, StatusCompleted, nil)
	
	store3 := NewPersistentStore(dir)
	fetched2, _ := store3.Get(job2.ID)
	if fetched2.Status != StatusCompleted {
		t.Fatalf("expected completed job to remain completed, got %s", fetched2.Status)
	}
}

func TestStoreCleanup(t *testing.T) {
	store := NewStore()
	job, _ := store.Create()
	store.UpdateStatus(job.ID, StatusCompleted, nil)
	
	// Fast forward time for test
	store.mu.Lock()
	store.jobs[job.ID].UpdatedAt = time.Now().Add(-2 * time.Hour)
	store.mu.Unlock()

	store.Cleanup(1 * time.Hour)
	if _, ok := store.Get(job.ID); ok {
		t.Fatal("expected old completed job to be cleaned up")
	}
}
