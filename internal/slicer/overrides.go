package slicer

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func writeResolvedProfile(dataPath string, category string, name string, overrides map[string]any, outputPath string) error {
	data, err := os.ReadFile(filepath.Join(dataPath, category, name+".json"))
	if err != nil {
		return err
	}

	if len(overrides) == 0 {
		return os.WriteFile(outputPath, data, 0o644)
	}

	var base map[string]any
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}

	merged := merge(base, overrides)
	resolved, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, resolved, 0o644)
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
