package slicer

import (
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

const maxSliceQuantity = 512

func normalizeSliceSettings(settings Settings) (Settings, error) {
	if settings.Quantity == 0 {
		settings.Quantity = 1
	}
	if settings.Quantity < 1 || settings.Quantity > maxSliceQuantity {
		return Settings{}, httpx.NewError(http.StatusBadRequest, fmt.Sprintf("quantity must be an integer between 1 and %d", maxSliceQuantity))
	}
	if settings.ExportType == "" {
		settings.ExportType = "gcode"
	}
	if settings.ExportType != "gcode" {
		return Settings{}, httpx.NewError(http.StatusBadRequest, "exportType must be gcode; 3MF and ZIP delivery are not supported")
	}
	if settings.Plate == "" {
		settings.Plate = "1"
	}
	if settings.Plate != "1" {
		return Settings{}, httpx.NewError(http.StatusBadRequest, "plate must be 1; additional plates are not supported")
	}
	if settings.Quantity > 1 && !settings.Arrange {
		return Settings{}, httpx.NewError(http.StatusBadRequest, "arrange must be true when quantity is greater than 1")
	}
	if math.IsNaN(settings.ArrangeSpacing) || math.IsInf(settings.ArrangeSpacing, 0) || settings.ArrangeSpacing < 0 {
		return Settings{}, httpx.NewError(http.StatusBadRequest, "arrangeSpacing must be a finite, non-negative number of millimeters")
	}
	if settings.ArrangeSpacing > 0 {
		return Settings{}, httpx.NewError(http.StatusBadRequest, "arrangeSpacing greater than 0 is not supported by the OrcaSlicer 2.5.0 CLI; omit it or use 0 for automatic spacing")
	}
	settings.ArrangeSpacing = 0
	if settings.Orient && hasExplicitRotation(settings) {
		return Settings{}, httpx.NewError(http.StatusBadRequest, "orient must be false when rotate, rotateX or rotateY is set; automatic orientation replaces explicit rotations")
	}
	for name, value := range map[string]*float64{"rotate": settings.Rotate, "rotateX": settings.RotateX, "rotateY": settings.RotateY} {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0)) {
			return Settings{}, httpx.NewError(http.StatusBadRequest, name+" must be a finite number of degrees")
		}
	}
	return settings, nil
}

func hasExplicitRotation(settings Settings) bool {
	return settings.Rotate != nil || settings.RotateX != nil || settings.RotateY != nil
}

func validateSingleSTL(models []ModelInput) error {
	if len(models) == 0 {
		return httpx.NewError(http.StatusBadRequest, "Model file is required")
	}
	if len(models) != 1 {
		return httpx.NewError(http.StatusBadRequest, "Exactly one STL file is supported per request; multiple models are not supported")
	}
	if strings.ToLower(filepath.Ext(models[0].Filename)) != ".stl" {
		return httpx.NewError(http.StatusBadRequest, "Only one STL file is supported per request")
	}
	return nil
}

func rotationArg(value *float64) string {
	return strconv.FormatFloat(*value, 'f', -1, 64)
}
