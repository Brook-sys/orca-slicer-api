package slicer

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

type ProfileAlias struct {
	Category string `json:"category"`
	From     string `json:"from"`
	To       string `json:"to"`
}

func (s *Service) ListAliases() ([]ProfileAlias, error) {
	return loadAliases(s.DataPath)
}

func (s *Service) SaveAlias(alias ProfileAlias) ([]ProfileAlias, error) {
	if strings.TrimSpace(alias.Category) == "" || strings.TrimSpace(alias.From) == "" || strings.TrimSpace(alias.To) == "" {
		return nil, httpx.NewError(http.StatusBadRequest, "category, from and to are required")
	}

	aliases, err := loadAliases(s.DataPath)
	if err != nil {
		return nil, err
	}

	updated := false
	for index, item := range aliases {
		if aliasMatches(item, alias.Category, alias.From) {
			aliases[index] = alias
			updated = true
			break
		}
	}
	if !updated {
		aliases = append(aliases, alias)
	}

	if err := saveAliases(s.DataPath, aliases); err != nil {
		return nil, err
	}
	return aliases, nil
}

func (s *Service) DeleteAlias(category string, from string) ([]ProfileAlias, error) {
	aliases, err := loadAliases(s.DataPath)
	if err != nil {
		return nil, err
	}

	filtered := make([]ProfileAlias, 0, len(aliases))
	deleted := false
	for _, item := range aliases {
		if aliasMatches(item, category, from) {
			deleted = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !deleted {
		return nil, httpx.NewError(http.StatusNotFound, "Profile alias not found")
	}
	if err := saveAliases(s.DataPath, filtered); err != nil {
		return nil, err
	}
	return filtered, nil
}

func profileAliasTargets(dataPath string, category string, name string) []string {
	seen := map[string]bool{}
	targets := make([]string, 0)
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := normalizeProfileName(value)
		if seen[key] {
			return
		}
		seen[key] = true
		targets = append(targets, value)
	}

	add(name)
	add(applyKnownProfileAliases(name))

	if strings.TrimSpace(dataPath) != "" {
		aliases, err := loadAliases(dataPath)
		if err == nil {
			for _, alias := range aliases {
				if aliasMatches(alias, category, name) {
					add(alias.To)
				}
			}
		}
	}

	return targets
}

func loadAliases(dataPath string) ([]ProfileAlias, error) {
	data, err := os.ReadFile(aliasPath(dataPath))
	if err != nil {
		if os.IsNotExist(err) {
			return []ProfileAlias{}, nil
		}
		return nil, err
	}
	var aliases []ProfileAlias
	if err := json.Unmarshal(data, &aliases); err != nil {
		return nil, httpx.NewError(http.StatusBadRequest, "profile-aliases.json is invalid")
	}
	return aliases, nil
}

func saveAliases(dataPath string, aliases []ProfileAlias) error {
	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(aliases, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(aliasPath(dataPath), data, 0o644)
}

func aliasPath(dataPath string) string {
	return filepath.Join(dataPath, "profile-aliases.json")
}

func aliasMatches(alias ProfileAlias, category string, from string) bool {
	return strings.EqualFold(alias.Category, category) && profileNameMatches(alias.From, from)
}

func applyKnownProfileAliases(value string) string {
	value = strings.ReplaceAll(value, "Neptune4", "N4")
	value = strings.ReplaceAll(value, "Neptune 4", "N4")
	value = strings.ReplaceAll(value, "Neptune-4", "N4")
	return value
}
