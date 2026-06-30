package health

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

type Handler struct {
	DataPath       string
	OrcaSlicerPath string
}

type Response struct {
	Status               string `json:"status"`
	DataPathExists       bool   `json:"dataPathExists"`
	DataPathWritable     bool   `json:"dataPathWritable"`
	OrcaSlicerConfigured bool   `json:"orcaSlicerConfigured"`
	OrcaSlicerExists     bool   `json:"orcaSlicerExists"`
	OrcaSlicerExecutable bool   `json:"orcaSlicerExecutable"`
}

func (h Handler) Check(w http.ResponseWriter, r *http.Request) {
	response := Response{
		DataPathExists:       pathExists(h.DataPath),
		DataPathWritable:     pathWritable(h.DataPath),
		OrcaSlicerConfigured: h.OrcaSlicerPath != "",
		OrcaSlicerExists:     h.OrcaSlicerPath != "" && pathExists(h.OrcaSlicerPath),
		OrcaSlicerExecutable: h.OrcaSlicerPath != "" && executable(h.OrcaSlicerPath),
	}

	status := http.StatusOK
	response.Status = "healthy"
	if !response.DataPathExists || !response.DataPathWritable || !response.OrcaSlicerConfigured || !response.OrcaSlicerExists || !response.OrcaSlicerExecutable {
		response.Status = "unhealthy"
		status = http.StatusServiceUnavailable
	}

	httpx.WriteJSON(w, status, response)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func pathWritable(path string) bool {
	if path == "" {
		return false
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return false
	}
	file, err := os.CreateTemp(path, ".health-*")
	if err != nil {
		return false
	}
	name := file.Name()
	_ = file.Close()
	_ = os.Remove(name)
	return true
}

func executable(path string) bool {
	info, err := os.Stat(filepath.Clean(path))
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}
