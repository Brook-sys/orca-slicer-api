package slicer

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

func WriteResult(w http.ResponseWriter, r *http.Request, result Result) error {
	if len(result.Files) != 1 || !strings.EqualFold(filepath.Ext(result.Files[0]), ".gcode") {
		return httpx.NewError(http.StatusInternalServerError, "Slice result must contain exactly one G-code file")
	}
	w.Header().Set("X-Print-Time-Seconds", formatFloat(result.Metadata.PrintTimeSeconds))
	w.Header().Set("X-Filament-Used-g", formatFloat(result.Metadata.FilamentUsedG))
	w.Header().Set("X-Filament-Used-mm", formatFloat(result.Metadata.FilamentUsedMM))
	w.Header().Set("X-Slice-Quantity", strconv.Itoa(result.Quantity))
	w.Header().Set("X-Slice-Plate", "1")
	w.Header().Set("Content-Type", "application/x-gcode")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(result.Files[0])+`"`)
	http.ServeFile(w, r, result.Files[0])
	return nil
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
