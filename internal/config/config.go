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
	UseXvfb          bool
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
		UseXvfb:          os.Getenv("USE_XVFB") == "true" || os.Getenv("USE_XVFB") == "1",
	}
}
