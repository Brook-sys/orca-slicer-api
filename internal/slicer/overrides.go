package slicer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

type ResolveProfileRequest struct {
	Category  string         `json:"category"`
	Name      string         `json:"name"`
	Overrides map[string]any `json:"overrides"`
}

type ResolveProfileResponse struct {
	Category string         `json:"category"`
	Name     string         `json:"name"`
	Resolved map[string]any `json:"resolved"`
	Warnings []string       `json:"warnings"`
}

type ResolveProfilesResponse struct {
	Printer  *ResolveProfileResponse `json:"printer,omitempty"`
	Preset   *ResolveProfileResponse `json:"preset,omitempty"`
	Filament *ResolveProfileResponse `json:"filament,omitempty"`
}

func ResolveProfile(dataPath string, category string, name string, overrides map[string]any) (ResolveProfileResponse, error) {
	if overrides == nil {
		overrides = map[string]any{}
	}
	data, err := os.ReadFile(filepath.Join(dataPath, category, name+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return ResolveProfileResponse{}, httpx.NewError(http.StatusNotFound, fmt.Sprintf("%s profile %q not found", category, name))
		}
		return ResolveProfileResponse{}, err
	}

	var base map[string]any
	if err := json.Unmarshal(data, &base); err != nil {
		return ResolveProfileResponse{}, httpx.NewError(http.StatusBadRequest, fmt.Sprintf("%s profile %q is invalid JSON", category, name))
	}

	warnings := missingKeys(base, overrides, "")
	resolved := merge(copyMap(base), overrides)
	return ResolveProfileResponse{Category: category, Name: name, Resolved: resolved, Warnings: warnings}, nil
}

func writeResolvedProfile(dataPath string, category string, name string, overrides map[string]any, outputPath string) error {
	resolved, err := ResolveProfile(dataPath, category, name, overrides)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(resolved.Resolved, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0o644)
}

func merge(base map[string]any, overrides map[string]any) map[string]any {
	for key, value := range overrides {
		baseChild, baseOK := base[key].(map[string]any)
		overrideChild, overrideOK := value.(map[string]any)
		if baseOK && overrideOK {
			base[key] = merge(baseChild, overrideChild)
			continue
		}
		base[key] = value
	}
	return base
}

func missingKeys(base map[string]any, overrides map[string]any, prefix string) []string {
	warnings := make([]string, 0)
	for key, value := range overrides {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		baseValue, ok := base[key]
		if !ok {
			warnings = append(warnings, "Override key does not exist in base profile: "+path)
			continue
		}
		baseChild, baseOK := baseValue.(map[string]any)
		overrideChild, overrideOK := value.(map[string]any)
		if baseOK && overrideOK {
			warnings = append(warnings, missingKeys(baseChild, overrideChild, path)...)
		}
	}
	return warnings
}

func copyMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		if child, ok := value.(map[string]any); ok {
			output[key] = copyMap(child)
			continue
		}
		output[key] = value
	}
	return output
}
