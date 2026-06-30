package health

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHealthCheckHealthy(t *testing.T) {
	dir := t.TempDir()
	orcaPath := filepath.Join(dir, "orca")
	if err := os.WriteFile(orcaPath, []byte("orca"), 0o755); err != nil {
		t.Fatal(err)
	}

	handler := Handler{DataPath: dir, OrcaSlicerPath: orcaPath}
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.Check(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Code)
	}
}

func TestHealthCheckUnhealthy(t *testing.T) {
	handler := Handler{DataPath: t.TempDir()}
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.Check(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", res.Code)
	}
}
