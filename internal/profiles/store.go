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

type ProfileInfo struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Checksum  string `json:"checksum"`
	UpdatedAt string `json:"updatedAt"`
	SourceURL string `json:"sourceUrl,omitempty"`
}

type ImportRequest struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Overwrite bool   `json:"overwrite"`
}

type ImportResponse struct {
	Name      string `json:"name"`
	Checksum  string `json:"checksum"`
	SourceURL string `json:"sourceUrl,omitempty"`
	Updated   bool   `json:"updated"`
}

type sourceFile struct {
	URL       string `json:"url"`
	Checksum  string `json:"checksum"`
	UpdatedAt string `json:"updatedAt"`
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
	return s.save(category, name, data, true)
}

func (s *Store) List(category string) ([]ProfileInfo, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(filepath.Join(s.BasePath, category))
	if err != nil {
		if os.IsNotExist(err) {
			return []ProfileInfo{}, nil
		}
		return nil, err
	}

	items := make([]ProfileInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || strings.HasSuffix(entry.Name(), ".source.json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		info, err := s.Info(category, name)
		if err != nil {
			return nil, err
		}
		items = append(items, info)
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (s *Store) Info(category string, name string) (ProfileInfo, error) {
	if err := ValidateCategory(category); err != nil {
		return ProfileInfo{}, err
	}
	if err := ValidateName(name); err != nil {
		return ProfileInfo{}, err
	}

	path := profilePath(s.BasePath, category, name)
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ProfileInfo{}, httpx.NewError(http.StatusNotFound, "Profile not found")
		}
		return ProfileInfo{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ProfileInfo{}, err
	}

	info := ProfileInfo{
		Name:      name,
		Size:      stat.Size(),
		Checksum:  checksum(data),
		UpdatedAt: stat.ModTime().UTC().Format(time.RFC3339),
	}
	if source, err := s.readSource(category, name); err == nil {
		info.SourceURL = source.URL
	}
	return info, nil
}

func (s *Store) Get(category string, name string) ([]byte, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, err
	}
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(profilePath(s.BasePath, category, name))
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

	err := os.Remove(profilePath(s.BasePath, category, name))
	if err != nil && os.IsNotExist(err) {
		return httpx.NewError(http.StatusNotFound, "Profile not found")
	}
	if err != nil {
		return err
	}
	_ = os.Remove(sourcePath(s.BasePath, category, name))
	return nil
}

func (s *Store) ImportURL(ctx context.Context, category string, req ImportRequest) (ImportResponse, error) {
	if err := ValidateCategory(category); err != nil {
		return ImportResponse{}, err
	}
	if err := ValidateName(req.Name); err != nil {
		return ImportResponse{}, err
	}
	if !req.Overwrite {
		if _, err := os.Stat(profilePath(s.BasePath, category, req.Name)); err == nil {
			return ImportResponse{}, httpx.NewError(http.StatusConflict, "Profile already exists")
		}
	}

	data, err := s.download(ctx, req.URL)
	if err != nil {
		return ImportResponse{}, err
	}

	checksum, err := s.save(category, req.Name, data, true)
	if err != nil {
		return ImportResponse{}, err
	}
	if err := s.writeSource(category, req.Name, sourceFile{URL: req.URL, Checksum: checksum, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}); err != nil {
		return ImportResponse{}, err
	}

	return ImportResponse{Name: req.Name, Checksum: checksum, SourceURL: req.URL, Updated: true}, nil
}

func (s *Store) UpdateFromSource(ctx context.Context, category string, name string) (ImportResponse, error) {
	if err := ValidateCategory(category); err != nil {
		return ImportResponse{}, err
	}
	if err := ValidateName(name); err != nil {
		return ImportResponse{}, err
	}
	if _, err := os.Stat(profilePath(s.BasePath, category, name)); err != nil {
		if os.IsNotExist(err) {
			return ImportResponse{}, httpx.NewError(http.StatusNotFound, "Profile not found")
		}
		return ImportResponse{}, err
	}

	source, err := s.readSource(category, name)
	if err != nil {
		return ImportResponse{}, httpx.NewError(http.StatusBadRequest, "Profile has no source URL")
	}

	data, err := s.download(ctx, source.URL)
	if err != nil {
		return ImportResponse{}, err
	}
	newChecksum := checksum(data)
	if newChecksum == source.Checksum {
		return ImportResponse{Name: name, Checksum: newChecksum, SourceURL: source.URL, Updated: false}, nil
	}

	if _, err := s.save(category, name, data, true); err != nil {
		return ImportResponse{}, err
	}
	if err := s.writeSource(category, name, sourceFile{URL: source.URL, Checksum: newChecksum, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}); err != nil {
		return ImportResponse{}, err
	}

	return ImportResponse{Name: name, Checksum: newChecksum, SourceURL: source.URL, Updated: true}, nil
}

func (s *Store) save(category string, name string, data []byte, overwrite bool) (string, error) {
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

	path := profilePath(s.BasePath, category, name)
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return "", httpx.NewError(http.StatusConflict, "Profile already exists")
		}
	}

	checksum := checksum(data)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}

	return checksum, nil
}

func (s *Store) download(ctx context.Context, rawURL string) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, httpx.NewError(http.StatusBadRequest, "URL must be a valid HTTPS URL")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	res, err := s.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, httpx.NewError(http.StatusBadRequest, fmt.Sprintf("Failed to download profile: status %d", res.StatusCode))
	}

	data, err := io.ReadAll(io.LimitReader(res.Body, maxProfileSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxProfileSize {
		return nil, httpx.NewError(http.StatusBadRequest, "Profile is too large")
	}
	if err := validateJSON(data); err != nil {
		return nil, err
	}
	return data, nil
}

func (s *Store) readSource(category string, name string) (sourceFile, error) {
	data, err := os.ReadFile(sourcePath(s.BasePath, category, name))
	if err != nil {
		return sourceFile{}, err
	}
	var source sourceFile
	if err := json.Unmarshal(data, &source); err != nil {
		return sourceFile{}, err
	}
	return source, nil
}

func (s *Store) writeSource(category string, name string, source sourceFile) error {
	data, err := json.MarshalIndent(source, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sourcePath(s.BasePath, category, name), data, 0o644)
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

func profilePath(basePath string, category string, name string) string {
	return filepath.Join(basePath, category, name+".json")
}

func sourcePath(basePath string, category string, name string) string {
	return filepath.Join(basePath, category, name+".source.json")
}

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
