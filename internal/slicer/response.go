package slicer

import (
	"archive/zip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

func WriteResult(w http.ResponseWriter, r *http.Request, result Result) error {
	w.Header().Set("X-Print-Time-Seconds", formatFloat(result.Metadata.PrintTimeSeconds))
	w.Header().Set("X-Filament-Used-g", formatFloat(result.Metadata.FilamentUsedG))
	w.Header().Set("X-Filament-Used-mm", formatFloat(result.Metadata.FilamentUsedMM))

	if len(result.Files) == 1 {
		w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(result.Files[0])+`"`)
		http.ServeFile(w, r, result.Files[0])
		return nil
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="result.zip"`)
	archive := zip.NewWriter(w)
	defer archive.Close()

	for _, file := range result.Files {
		if err := addFile(archive, file); err != nil {
			return err
		}
	}
	return nil
}

func addFile(archive *zip.Writer, path string) error {
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := archive.Create(filepath.Base(path))
	if err != nil {
		return err
	}

	_, err = io.Copy(output, input)
	return err
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
