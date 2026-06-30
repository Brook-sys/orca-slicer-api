package slicer

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
	base, err := resolveProfileInheritance(dataPath, category, name, map[string]bool{})
	if err != nil {
		return ResolveProfileResponse{}, err
	}

	warnings := missingKeys(base, overrides, "")
	resolved := merge(copyMap(base), overrides)
	delete(resolved, "inherits")
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

func resolveProfileInheritance(dataPath string, category string, name string, seen map[string]bool) (map[string]any, error) {
	key := category + ":" + name
	if seen[key] {
		return nil, httpx.NewError(http.StatusBadRequest, fmt.Sprintf("circular inherits detected for %s profile %q", category, name))
	}
	seen[key] = true

	profile, err := loadProfileByName(dataPath, category, name)
	if err != nil {
		return nil, err
	}

	inherits, _ := profile["inherits"].(string)
	inherits = strings.TrimSpace(inherits)
	if inherits == "" {
		return profile, nil
	}

	parent, err := resolveProfileInheritance(dataPath, category, inherits, seen)
	if err != nil {
		var httpErr *httpx.Error
		if errors.As(err, &httpErr) && httpErr.Status == http.StatusNotFound {
			return nil, httpx.NewError(http.StatusBadRequest, fmt.Sprintf("%s profile %q inherits from %q, but parent profile was not found", category, profileDisplayName(profile, name), inherits))
		}
		return nil, err
	}

	merged := merge(parent, profile)
	delete(merged, "inherits")
	return merged, nil
}

func loadProfileByName(dataPath string, category string, name string) (map[string]any, error) {
	categoryPath := filepath.Join(dataPath, category)
	candidates := []string{
		filepath.Join(categoryPath, name+".json"),
		filepath.Join(categoryPath, sanitizeProfileName(name)+".json"),
	}

	for _, path := range candidates {
		profile, err := readProfileFile(path, category, name)
		if err == nil {
			return profile, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	entries, err := os.ReadDir(categoryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, httpx.NewError(http.StatusNotFound, fmt.Sprintf("%s profile %q not found", category, name))
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || strings.HasSuffix(entry.Name(), ".source.json") {
			continue
		}
		profile, err := readProfileFile(filepath.Join(categoryPath, entry.Name()), category, strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			return nil, err
		}
		profileName, _ := profile["name"].(string)
		if profileName == name || sanitizeProfileName(profileName) == sanitizeProfileName(name) {
			return profile, nil
		}
	}

	return nil, httpx.NewError(http.StatusNotFound, fmt.Sprintf("%s profile %q not found", category, name))
}

func readProfileFile(path string, category string, name string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var profile map[string]any
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, httpx.NewError(http.StatusBadRequest, fmt.Sprintf("%s profile %q is invalid JSON", category, name))
	}
	return profile, nil
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

func sanitizeProfileName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "@", "")
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	value = re.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	return value
}

func profileDisplayName(profile map[string]any, fallback string) string {
	if name, ok := profile["name"].(string); ok && name != "" {
		return name
	}
	return fallback
}
