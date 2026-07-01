package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/Brook-sys/orca-slicer-api/internal/config"
	"github.com/Brook-sys/orca-slicer-api/internal/docs"
	"github.com/Brook-sys/orca-slicer-api/internal/health"
	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
	"github.com/Brook-sys/orca-slicer-api/internal/profiles"
	"github.com/Brook-sys/orca-slicer-api/internal/slicer"
)

func main() {
	cfg := config.Load()
	profileStore := profiles.NewStore(cfg.DataPath)
	if err := profileStore.EnsureDirs(); err != nil {
		slog.Error("failed to initialize data path", "error", err)
		os.Exit(1)
	}

	healthHandler := health.Handler{DataPath: cfg.DataPath, OrcaSlicerPath: cfg.OrcaSlicerPath}
	profileHandler := profiles.Handler{Store: profileStore}
	sliceService := &slicer.Service{
		DataPath:         cfg.DataPath,
		OrcaSlicerPath:   cfg.OrcaSlicerPath,
		OrcaProfilesPath: cfg.OrcaProfilesPath,
		Timeout:          cfg.SliceTimeout,
		GenerateImage:    cfg.GenerateImage,
		State:            slicer.NewStateStore(cfg.DataPath),
	}
	sliceHandler := slicer.Handler{Service: sliceService}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /openapi.json", docs.OpenAPIHandler)
	mux.HandleFunc("GET /api-docs", docs.SwaggerUIHandler)
	mux.HandleFunc("GET /health", healthHandler.Check)
	mux.HandleFunc("GET /profiles/{category}", profileHandler.List)
	mux.HandleFunc("GET /profiles/{category}/{name}", profileHandler.Get)
	mux.HandleFunc("POST /profiles/{category}/upload", profileHandler.Upload)
	mux.HandleFunc("POST /profiles/{category}/import-url", profileHandler.ImportURL)
	mux.HandleFunc("POST /profiles/{category}/{name}/update-from-source", profileHandler.UpdateFromSource)
	mux.HandleFunc("DELETE /profiles/{category}/{name}", profileHandler.Delete)
	mux.HandleFunc("GET /profile-aliases", sliceHandler.ListAliases)
	mux.HandleFunc("POST /profile-aliases", sliceHandler.SaveAlias)
	mux.HandleFunc("DELETE /profile-aliases/{category}/{from}", sliceHandler.DeleteAlias)
	mux.HandleFunc("POST /profiles/resolve", sliceHandler.ResolveProfile)
	mux.HandleFunc("GET /slice/status", sliceHandler.Status)
	mux.HandleFunc("GET /slice/debug", sliceHandler.Debug)
	mux.HandleFunc("POST /slice/resolve-profiles", sliceHandler.ResolveProfiles)
	mux.HandleFunc("POST /slice", sliceHandler.Slice)
	mux.HandleFunc("POST /slice/preview", sliceHandler.Preview)

	handler := httpx.Middleware(cfg.CORSOrigins, mux)

	slog.Info("server listening", "port", cfg.Port, "generate_image", cfg.GenerateImage)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
