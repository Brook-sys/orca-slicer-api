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

func ResolveProfile(dataPath string, orcaProfilesPath string, category string, name string, overrides map[string]any) (ResolveProfileResponse, error) {
	if overrides == nil {
		overrides = map[string]any{}
	}
	base, err := resolveProfileInheritance(dataPath, orcaProfilesPath, category, name, map[string]bool{})
	if err != nil {
		return ResolveProfileResponse{}, err
	}

	warnings := missingKeys(base, overrides, "")
	resolved := merge(copyMap(base), overrides)
	delete(resolved, "inherits")
	return ResolveProfileResponse{Category: category, Name: name, Resolved: resolved, Warnings: warnings}, nil
}

func writeResolvedProfile(dataPath string, orcaProfilesPath string, category string, name string, overrides map[string]any, outputPath string) error {
	resolved, err := ResolveProfile(dataPath, orcaProfilesPath, category, name, overrides)
	if err != nil {
		return err
	}
	return writeProfile(outputPath, resolved.Resolved)
}

func writeProfile(outputPath string, profile map[string]any) error {
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0o644)
}

func resolveProfileInheritance(dataPath string, orcaProfilesPath string, category string, name string, seen map[string]bool) (map[string]any, error) {
	key := category + ":" + name
	if seen[key] {
		return nil, httpx.NewError(http.StatusBadRequest, fmt.Sprintf("circular inherits detected for %s profile %q", category, name))
	}
	seen[key] = true

	profile, err := loadProfileByName(dataPath, orcaProfilesPath, category, name)
	if err != nil {
		return nil, err
	}

	inherits, _ := profile["inherits"].(string)
	inherits = strings.TrimSpace(inherits)
	if inherits == "" {
		return profile, nil
	}

	parent, err := resolveProfileInheritance(dataPath, orcaProfilesPath, category, inherits, seen)
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

func loadProfileByName(dataPath string, orcaProfilesPath string, category string, name string) (map[string]any, error) {
	categoryPath := filepath.Join(dataPath, category)
	targets := profileAliasTargets(dataPath, category, name)
	for _, target := range targets {
		candidates := []string{
			filepath.Join(categoryPath, target+".json"),
			filepath.Join(categoryPath, sanitizeProfileName(target)+".json"),
		}

		for _, path := range candidates {
			profile, err := readProfileFile(path, category, target)
			if err == nil {
				return profile, nil
			}
			if !os.IsNotExist(err) {
				return nil, err
			}
		}
	}

	entries, err := os.ReadDir(categoryPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || strings.HasSuffix(entry.Name(), ".source.json") {
			continue
		}
		profile, err := readProfileFile(filepath.Join(categoryPath, entry.Name()), category, strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			return nil, err
		}
		for _, target := range targets {
			if profileMatches(profile, strings.TrimSuffix(entry.Name(), ".json"), target) {
				return profile, nil
			}
		}
	}

	for _, target := range targets {
		if profile, err := loadBuiltInProfileByName(orcaProfilesPath, category, target); err == nil {
			return profile, nil
		} else if !isNotFoundHTTPError(err) {
			return nil, err
		}
	}

	return nil, httpx.NewError(http.StatusNotFound, fmt.Sprintf("%s profile %q not found", category, name))
}

func loadBuiltInProfileByName(orcaProfilesPath string, category string, name string) (map[string]any, error) {
	if strings.TrimSpace(orcaProfilesPath) == "" {
		return nil, httpx.NewError(http.StatusNotFound, fmt.Sprintf("built-in %s profile %q not found", category, name))
	}

	baseDirs := append(builtInCategoryDirs(orcaProfilesPath, category), orcaProfilesPath)
	targets := profileAliasTargets("", category, name)
	for _, dir := range baseDirs {
		for _, target := range targets {
			candidates := []string{
				filepath.Join(dir, target+".json"),
				filepath.Join(dir, sanitizeProfileName(target)+".json"),
			}
			for _, path := range candidates {
				profile, err := readProfileFile(path, category, target)
				if err == nil {
					return profile, nil
				}
				if !os.IsNotExist(err) {
					return nil, err
				}
			}
		}
	}

	for _, dir := range baseDirs {
		profile, err := findProfileRecursive(dir, category, name)
		if err == nil {
			return profile, nil
		}
		if !isNotFoundHTTPError(err) {
			return nil, err
		}
	}

	return nil, httpx.NewError(http.StatusNotFound, fmt.Sprintf("built-in %s profile %q not found", category, name))
}

func findProfileRecursive(root string, category string, name string) (map[string]any, error) {
	targets := profileAliasTargets("", category, name)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, httpx.NewError(http.StatusNotFound, fmt.Sprintf("built-in %s profile %q not found", category, name))
		}
		return nil, err
	}

	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			profile, err := findProfileRecursive(path, category, name)
			if err == nil {
				return profile, nil
			}
			if !isNotFoundHTTPError(err) {
				return nil, err
			}
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		profile, err := readProfileFile(path, category, strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			return nil, err
		}
		for _, target := range targets {
			if profileMatches(profile, strings.TrimSuffix(entry.Name(), ".json"), target) {
				return profile, nil
			}
		}
	}

	return nil, httpx.NewError(http.StatusNotFound, fmt.Sprintf("built-in %s profile %q not found", category, name))
}

func builtInCategoryDirs(orcaProfilesPath string, category string) []string {
	mapped := map[string]string{
		"printers":  "machine",
		"presets":   "process",
		"filaments": "filament",
	}[category]
	if mapped == "" {
		mapped = category
	}

	return []string{
		filepath.Join(orcaProfilesPath, mapped),
		filepath.Join(orcaProfilesPath, "BBL", mapped),
		filepath.Join(orcaProfilesPath, "Elegoo", mapped),
		filepath.Join(orcaProfilesPath, "ELEGOO", mapped),
	}
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

func isNotFoundHTTPError(err error) bool {
	var httpErr *httpx.Error
	return errors.As(err, &httpErr) && httpErr.Status == http.StatusNotFound
}

func ensureCompatibleProfile(profile map[string]any, printer map[string]any) {
	for _, printerName := range profileIdentityNames(printer, []string{"name", "printer_settings_id"}) {
		profile["compatible_printers"] = appendStringValue(profile["compatible_printers"], printerName)
	}
	profile["compatible_printers_condition"] = ""
}

func profileIdentityNames(profile map[string]any, keys []string) []string {
	seen := map[string]bool{}
	names := make([]string, 0, len(keys))
	for _, key := range keys {
		value, ok := profile[key].(string)
		value = strings.TrimSpace(value)
		if !ok || value == "" || seen[value] {
			continue
		}
		seen[value] = true
		names = append(names, value)
	}
	return names
}

func appendStringValue(value any, item string) any {
	item = strings.TrimSpace(item)
	if item == "" {
		return value
	}

	switch current := value.(type) {
	case []any:
		for _, existing := range current {
			if text, ok := existing.(string); ok && text == item {
				return current
			}
		}
		return append(current, item)
	case []string:
		for _, existing := range current {
			if existing == item {
				return current
			}
		}
		return append(current, item)
	case string:
		parts := splitStringList(current)
		for _, existing := range parts {
			if existing == item {
				return current
			}
		}
		parts = append(parts, item)
		return strings.Join(parts, ";")
	default:
		return []string{item}
	}
}

func splitStringList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ';' || r == ','
	})
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			parts = append(parts, field)
		}
	}
	return parts
}

func profileMatches(profile map[string]any, filename string, target string) bool {
	candidates := []string{filename}
	for _, key := range []string{"name", "print_settings_id", "printer_settings_id", "filament_settings_id", "base_id"} {
		if value, ok := profile[key].(string); ok && value != "" {
			candidates = append(candidates, value)
		}
	}
	for _, candidate := range candidates {
		if profileNameMatches(candidate, target) {
			return true
		}
	}
	return false
}

func profileNameMatches(a string, b string) bool {
	return normalizeProfileName(a) == normalizeProfileName(b)
}

func normalizeProfileName(value string) string {
	return strings.ToLower(sanitizeProfileName(value))
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
