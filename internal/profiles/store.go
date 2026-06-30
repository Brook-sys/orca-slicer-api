package profiles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

const maxProfileSize = 4_000_000

var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

var categories = map[string]bool{
	"printers":  true,
	"presets":   true,
	"filaments": true,
}

type Store struct {
	BasePath string
	Client   *http.Client
}

type ImportRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ImportResponse struct {
	Name     string `json:"name"`
	Checksum string `json:"checksum"`
}

func NewStore(basePath string) *Store {
	return &Store{
		BasePath: basePath,
		Client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (s *Store) EnsureDirs() error {
	for category := range categories {
		if err := os.MkdirAll(filepath.Join(s.BasePath, category), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func ValidateCategory(category string) error {
	if !categories[category] {
		return httpx.NewError(http.StatusBadRequest, "Invalid category")
	}
	return nil
}

func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return httpx.NewError(http.StatusBadRequest, "Name cannot be empty")
	}
	if !validName.MatchString(name) {
		return httpx.NewError(http.StatusBadRequest, "Name must contain only letters, numbers, underscore or dash")
	}
	return nil
}

func (s *Store) Save(category string, name string, data []byte) (string, error) {
	if err := ValidateCategory(category); err != nil {
		return "", err
	}
	if err := ValidateName(name); err != nil {
		return "", err
	}
	if err := validateJSON(data); err != nil {
		return "", err
	}

	dir := filepath.Join(s.BasePath, category)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	checksum := checksum(data)
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}

	return checksum, nil
}

func (s *Store) List(category string) ([]string, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(filepath.Join(s.BasePath, category))
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	items := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		items = append(items, strings.TrimSuffix(entry.Name(), ".json"))
	}
	sort.Strings(items)
	return items, nil
}

func (s *Store) Get(category string, name string) ([]byte, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, err
	}
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(s.BasePath, category, name+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, httpx.NewError(http.StatusNotFound, "Profile not found")
		}
		return nil, err
	}
	return data, nil
}

func (s *Store) Delete(category string, name string) error {
	if err := ValidateCategory(category); err != nil {
		return err
	}
	if err := ValidateName(name); err != nil {
		return err
	}

	err := os.Remove(filepath.Join(s.BasePath, category, name+".json"))
	if err != nil && os.IsNotExist(err) {
		return httpx.NewError(http.StatusNotFound, "Profile not found")
	}
	return err
}

func (s *Store) ImportURL(ctx context.Context, category string, req ImportRequest) (ImportResponse, error) {
	if err := ValidateCategory(category); err != nil {
		return ImportResponse{}, err
	}
	if err := ValidateName(req.Name); err != nil {
		return ImportResponse{}, err
	}
	parsed, err := url.Parse(req.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return ImportResponse{}, httpx.NewError(http.StatusBadRequest, "URL must be a valid HTTPS URL")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return ImportResponse{}, err
	}

	res, err := s.Client.Do(httpReq)
	if err != nil {
		return ImportResponse{}, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return ImportResponse{}, httpx.NewError(http.StatusBadRequest, fmt.Sprintf("Failed to download profile: status %d", res.StatusCode))
	}

	data, err := io.ReadAll(io.LimitReader(res.Body, maxProfileSize+1))
	if err != nil {
		return ImportResponse{}, err
	}
	if len(data) > maxProfileSize {
		return ImportResponse{}, httpx.NewError(http.StatusBadRequest, "Profile is too large")
	}

	checksum, err := s.Save(category, req.Name, data)
	if err != nil {
		return ImportResponse{}, err
	}

	return ImportResponse{Name: req.Name, Checksum: checksum}, nil
}

func validateJSON(data []byte) error {
	if len(data) == 0 {
		return httpx.NewError(http.StatusBadRequest, "Profile cannot be empty")
	}
	if len(data) > maxProfileSize {
		return httpx.NewError(http.StatusBadRequest, "Profile is too large")
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return httpx.NewError(http.StatusBadRequest, "Invalid JSON profile")
	}
	return nil
}

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
