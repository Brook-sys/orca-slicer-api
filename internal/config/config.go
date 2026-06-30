package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port           string
	DataPath       string
	OrcaSlicerPath string
	SliceTimeout   time.Duration
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

	return Config{
		Port:           port,
		DataPath:       dataPath,
		OrcaSlicerPath: os.Getenv("ORCASLICER_PATH"),
		SliceTimeout:   time.Duration(timeoutSeconds) * time.Second,
	}
}
