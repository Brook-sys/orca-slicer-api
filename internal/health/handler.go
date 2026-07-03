package health

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

type Handler struct {
	DataPath         string
	OrcaSlicerPath   string
	OrcaProfilesPath string
	OrcaVersion      string
	GenerateImage    bool
	ResolveProfiles  bool
	SanitizeProfiles bool
}

type Response struct {
	Status                 string `json:"status"`
	DataPath               string `json:"dataPath"`
	DataPathExists         bool   `json:"dataPathExists"`
	DataPathWritable       bool   `json:"dataPathWritable"`
	OrcaSlicerPath         string `json:"orcaSlicerPath"`
	OrcaSlicerConfigured   bool   `json:"orcaSlicerConfigured"`
	OrcaSlicerExists       bool   `json:"orcaSlicerExists"`
	OrcaSlicerExecutable   bool   `json:"orcaSlicerExecutable"`
	OrcaProfilesPath       string `json:"orcaProfilesPath"`
	OrcaProfilesPathExists bool   `json:"orcaProfilesPathExists"`
	OrcaVersion            string `json:"orcaVersion"`
	GenerateImage          bool   `json:"generateImage"`
	ResolveProfiles        bool   `json:"resolveProfiles"`
	SanitizeProfiles       bool   `json:"sanitizeProfiles"`
}

func (h Handler) Check(w http.ResponseWriter, r *http.Request) {
	response := Response{
		DataPath:               h.DataPath,
		DataPathExists:         pathExists(h.DataPath),
		DataPathWritable:       pathWritable(h.DataPath),
		OrcaSlicerPath:         h.OrcaSlicerPath,
		OrcaSlicerConfigured:   h.OrcaSlicerPath != "",
		OrcaSlicerExists:       h.OrcaSlicerPath != "" && pathExists(h.OrcaSlicerPath),
		OrcaSlicerExecutable:   h.OrcaSlicerPath != "" && executable(h.OrcaSlicerPath),
		OrcaProfilesPath:       h.OrcaProfilesPath,
		OrcaProfilesPathExists: h.OrcaProfilesPath != "" && pathExists(h.OrcaProfilesPath),
		OrcaVersion:            h.OrcaVersion,
		GenerateImage:          h.GenerateImage,
		ResolveProfiles:        h.ResolveProfiles,
		SanitizeProfiles:       h.SanitizeProfiles,
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
