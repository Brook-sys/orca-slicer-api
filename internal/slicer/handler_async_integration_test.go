package slicer

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Brook-sys/orca-slicer-api/internal/jobs"
)

func TestSliceAsyncHTTPFlow(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test")
	}

	dir := t.TempDir()
	fake := filepath.Join(dir, "fake-http-async.sh")
	if err := os.WriteFile(fake, []byte(`#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--outputdir" ]; then
    shift; out="$1"
  fi
  shift
done
mkdir -p "$out"
cat > "$out/result.gcode" <<'EOF'
; total estimated time: 1m
G28
EOF
`), 0o755); err != nil {
		t.Fatal(err)
	}

	jobStore := jobs.NewStore()
	service := &Service{DataPath: dir, OrcaSlicerPath: fake, Timeout: 5 * time.Second, State: NewStateStore(dir), JobStore: jobStore}
	handler := Handler{Service: service, JobStore: jobStore}

	// 1. Submit async job
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "model.stl")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("solid test\nendsolid test\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/slice/async", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	handler.SliceAsync(w, req)

	if w.Code != 202 {
		t.Fatalf("expected 202, got %d", w.Code)
	}

	var job jobs.Job
	if err := json.Unmarshal(w.Body.Bytes(), &job); err != nil {
		t.Fatalf("failed to decode job: %v", err)
	}
	jID := job.ID

	for i := 0; i < 30; i++ {
		stats := jobStore.Stats()
		if stats["completed"] == 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if jID == "" {
		t.Fatalf("job did not complete")
	}

	// 2. Poll job status
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/jobs/"+jID, nil)
	r2.SetPathValue("id", jID)
	jobHandler := jobs.Handler{Store: jobStore}
	jobHandler.GetJob(w2, r2)

	if w2.Code != 200 {
		t.Fatalf("expected 200 from job GET, got %d", w2.Code)
	}

	// 3. Download
	w3 := httptest.NewRecorder()
	r3 := httptest.NewRequest("GET", "/jobs/"+jID+"/download", nil)
	r3.SetPathValue("id", jID)
	jobHandler.DownloadJob(w3, r3)

	if w3.Code != 200 {
		t.Fatalf("expected 200 from download, got %d", w3.Code)
	}
}
