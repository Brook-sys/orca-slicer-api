package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port             string
	DataPath         string
	OrcaSlicerPath   string
	OrcaProfilesPath string
	SliceTimeout     time.Duration
	CORSOrigins      string
	GenerateImage    bool
	ResolveProfiles  bool
	SanitizeProfiles bool
	OrcaVersion      string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	dataPath := os.Getenv("DATA_PATH")
	if dataPath == "" {
		dataPath = "data"
	}

	timeoutSeconds := 1800
	if value := os.Getenv("SLICE_TIMEOUT_SECONDS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			timeoutSeconds = parsed
		}
	}

	orcaProfilesPath := os.Getenv("ORCA_PROFILES_PATH")
	if orcaProfilesPath == "" {
		orcaProfilesPath = "/app/squashfs-root/resources/profiles"
	}

	return Config{
		Port:             port,
		DataPath:         dataPath,
		OrcaSlicerPath:   os.Getenv("ORCASLICER_PATH"),
		OrcaProfilesPath: orcaProfilesPath,
		SliceTimeout:     time.Duration(timeoutSeconds) * time.Second,
		CORSOrigins:      os.Getenv("CORS_ORIGINS"),
		GenerateImage:    boolEnv("GENERATE_IMAGE"),
		ResolveProfiles:  boolEnv("RESOLVE_PROFILES"),
		SanitizeProfiles: boolEnv("SANITIZE_PROFILES"),
		OrcaVersion:      os.Getenv("ORCA_VERSION"),
	}
}

func boolEnv(name string) bool {
	value := os.Getenv(name)
	return value == "true" || value == "1" || value == "on"
}
