package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/Brook-sys/orca-slicer-api/internal/config"
	"github.com/Brook-sys/orca-slicer-api/internal/profiles"
	"github.com/Brook-sys/orca-slicer-api/internal/slicer"
)

type healthResponse struct {
	Status string `json:"status"`
}

func main() {
	cfg := config.Load()
	profileStore := profiles.NewStore(cfg.DataPath)
	if err := profileStore.EnsureDirs(); err != nil {
		slog.Error("failed to initialize data path", "error", err)
		os.Exit(1)
	}

	profileHandler := profiles.Handler{Store: profileStore}
	sliceHandler := slicer.Handler{Service: slicer.Service{
		DataPath:       cfg.DataPath,
		OrcaSlicerPath: cfg.OrcaSlicerPath,
		Timeout:        cfg.SliceTimeout,
	}}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(healthResponse{Status: "healthy"})
	})
	mux.HandleFunc("GET /profiles/{category}", profileHandler.List)
	mux.HandleFunc("GET /profiles/{category}/{name}", profileHandler.Get)
	mux.HandleFunc("POST /profiles/{category}/upload", profileHandler.Upload)
	mux.HandleFunc("POST /profiles/{category}/import-url", profileHandler.ImportURL)
	mux.HandleFunc("DELETE /profiles/{category}/{name}", profileHandler.Delete)
	mux.HandleFunc("POST /slice", sliceHandler.Slice)

	slog.Info("server listening", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
