package slicer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

const maxStatisticsBytes = 64 << 20

type SliceStatistics struct {
	Version   int                     `json:"version"`
	Generator string                  `json:"generator"`
	Objects   []SliceStatisticsObject `json:"objects"`
}

type SliceStatisticsObject struct {
	Name     string `json:"name"`
	Instance int    `json:"instance"`
}

type sliceResultManifest struct {
	ReturnCode   *int `json:"return_code"`
	PlateIndex   int  `json:"plate_index"`
	SlicedPlates []struct {
		ID int `json:"id"`
	} `json:"sliced_plates"`
}

func validateSliceOutput(outputDir string, quantity int) ([]string, SliceStatistics, error) {
	files, err := resultFiles(outputDir, "gcode")
	if err != nil {
		return nil, SliceStatistics{}, httpx.NewError(http.StatusInternalServerError, "Slice result validation failed: "+err.Error())
	}
	if len(files) == 0 {
		return nil, SliceStatistics{}, httpx.NewError(http.StatusInternalServerError, "Slice result validation failed: no G-code file was generated")
	}
	if len(files) != 1 {
		return nil, SliceStatistics{}, httpx.NewError(http.StatusUnprocessableEntity, fmt.Sprintf("requested quantity %d did not fit on one plate; reduce quantity and retry", quantity))
	}
	if filepath.Base(files[0]) != "plate_1.gcode" {
		return nil, SliceStatistics{}, httpx.NewError(http.StatusInternalServerError, "Slice result validation failed: the only G-code file is not plate_1.gcode")
	}
	if info, err := os.Stat(files[0]); err != nil || info.Size() == 0 {
		return nil, SliceStatistics{}, httpx.NewError(http.StatusInternalServerError, "Slice result validation failed: the generated G-code is missing or empty")
	}

	manifest, err := readSliceResultManifest(outputDir)
	if err != nil {
		return nil, SliceStatistics{}, err
	}
	if len(manifest.SlicedPlates) != 1 || manifest.PlateIndex != 1 {
		return nil, SliceStatistics{}, httpx.NewError(http.StatusUnprocessableEntity, fmt.Sprintf("requested quantity %d did not fit on one plate; reduce quantity and retry", quantity))
	}
	if manifest.SlicedPlates[0].ID != 1 {
		return nil, SliceStatistics{}, httpx.NewError(http.StatusInternalServerError, "Slice result validation failed: result.json does not identify plate 1")
	}

	statistics, err := readSliceStatistics(filepath.Join(outputDir, "plate_1.statistics.json"))
	if err != nil {
		return nil, SliceStatistics{}, err
	}
	if len(statistics.Objects) != quantity {
		return nil, SliceStatistics{}, httpx.NewError(http.StatusUnprocessableEntity, fmt.Sprintf("requested quantity %d, but OrcaSlicer produced %d object(s) on plate 1; reduce quantity and retry", quantity, len(statistics.Objects)))
	}
	seen := make(map[string]struct{}, len(statistics.Objects))
	for _, object := range statistics.Objects {
		if object.Name == "" || object.Instance < 0 {
			return nil, SliceStatistics{}, httpx.NewError(http.StatusInternalServerError, "Slice result validation failed: statistics contain an invalid object identity")
		}
		identity := fmt.Sprintf("%s\x00%d", object.Name, object.Instance)
		if _, exists := seen[identity]; exists {
			return nil, SliceStatistics{}, httpx.NewError(http.StatusInternalServerError, "Slice result validation failed: statistics contain duplicate object identities")
		}
		seen[identity] = struct{}{}
	}
	return files, statistics, nil
}

func readSliceResultManifest(outputDir string) (sliceResultManifest, error) {
	var manifest sliceResultManifest
	if err := readBoundedJSON(filepath.Join(outputDir, "result.json"), &manifest); err != nil {
		return manifest, httpx.NewError(http.StatusInternalServerError, "Slice result validation failed: invalid result.json: "+err.Error())
	}
	if manifest.ReturnCode == nil || *manifest.ReturnCode != 0 {
		return manifest, httpx.NewError(http.StatusInternalServerError, "Slice result validation failed: result.json does not report successful slicing")
	}
	return manifest, nil
}

func readSliceStatistics(path string) (SliceStatistics, error) {
	var statistics SliceStatistics
	if err := readBoundedJSON(path, &statistics); err != nil {
		return statistics, httpx.NewError(http.StatusInternalServerError, "Slice result validation failed: plate_1.statistics.json is required and must be valid: "+err.Error())
	}
	if statistics.Version != 1 || statistics.Generator == "" {
		return statistics, httpx.NewError(http.StatusInternalServerError, "Slice result validation failed: unsupported or incomplete G-code statistics")
	}
	return statistics, nil
}

func readBoundedJSON(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() > maxStatisticsBytes {
		return fmt.Errorf("file exceeds %d bytes", int64(maxStatisticsBytes))
	}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return fmt.Errorf("unexpected trailing JSON value")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
