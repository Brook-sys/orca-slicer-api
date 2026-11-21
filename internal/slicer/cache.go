package slicer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type CacheEntry struct {
	Key        string
	Files      []string
	Workdir    string
	CreatedAt  time.Time
	Quantity   int
	Statistics SliceStatistics
}

// ResultCache owns private snapshots only. Get returns a new caller-owned copy;
// eviction and replacement must never remove a result being served or a job file.
type ResultCache struct {
	mu      sync.RWMutex
	entries map[string]CacheEntry
	ttl     time.Duration
	max     int
	baseDir string
}

func NewResultCache(ttl time.Duration, maxEntries int) *ResultCache {
	return &ResultCache{entries: make(map[string]CacheEntry), ttl: ttl, max: maxEntries}
}
func NewPersistentResultCache(ttl time.Duration, maxEntries int, baseDir string) *ResultCache {
	c := NewResultCache(ttl, maxEntries)
	c.baseDir = baseDir
	return c
}

func (c *ResultCache) makeModelsKey(models []ModelInput, settings Settings) string {
	// Explicitly include uploaded profiles: Settings hides these from HTTP JSON.
	return cacheDigest(struct {
		Models    []ModelInput
		Settings  Settings
		Printer   map[string]any
		Preset    map[string]any
		Filaments []map[string]any
	}{models, settings, settings.PrinterProfile, settings.PresetProfile, settings.FilamentProfiles})
}

func cacheDigest(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	} // Unsupported values must bypass caching, not collide.
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Get transfers ownership of an independent temporary directory to the caller.
func (c *ResultCache) Get(key string) (CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok || time.Since(entry.CreatedAt) > c.ttl {
		return CacheEntry{}, false
	}
	resultFileCount := len(entry.Files)
	snapshotFiles := append([]string(nil), entry.Files...)
	if entry.Quantity > 0 {
		// Validated results include the evidence needed to repeat the complete
		// plate/quantity checks on every cache hit.
		snapshotFiles = append(snapshotFiles,
			filepath.Join(entry.Workdir, "result.json"),
			filepath.Join(entry.Workdir, "plate_1.statistics.json"),
		)
	}
	files, dir, err := copyResultFiles("", snapshotFiles)
	if err != nil {
		return CacheEntry{}, false
	}
	entry.Files, entry.Workdir = files[:resultFileCount], dir
	entry.Statistics.Objects = append([]SliceStatisticsObject(nil), entry.Statistics.Objects...)
	return entry, true
}

func (c *ResultCache) Put(key string, files []string, workdir string) {
	c.put(key, files, len(files), 0, SliceStatistics{})
}

func (c *ResultCache) PutResult(key string, result Result) {
	if len(result.Files) == 0 {
		return
	}
	outputDir := filepath.Dir(result.Files[0])
	snapshotFiles := append([]string(nil), result.Files...)
	snapshotFiles = append(snapshotFiles,
		filepath.Join(outputDir, "result.json"),
		filepath.Join(outputDir, "plate_1.statistics.json"),
	)
	c.put(key, snapshotFiles, len(result.Files), result.Quantity, result.Statistics)
}

func (c *ResultCache) put(key string, files []string, resultFileCount int, quantity int, statistics SliceStatistics) {
	if key == "" || c.max <= 0 || resultFileCount <= 0 || resultFileCount > len(files) {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	storedFiles, dir, err := copyResultFiles(c.baseDir, files)
	if err != nil {
		return
	} // Never fall back to borrowing caller-owned paths.
	if old, ok := c.entries[key]; ok {
		_ = os.RemoveAll(old.Workdir)
		delete(c.entries, key)
	}
	if len(c.entries) >= c.max {
		oldest := c.oldestKey()
		_ = os.RemoveAll(c.entries[oldest].Workdir)
		delete(c.entries, oldest)
	}
	statistics.Objects = append([]SliceStatisticsObject(nil), statistics.Objects...)
	c.entries[key] = CacheEntry{Key: key, Files: storedFiles[:resultFileCount], Workdir: dir, CreatedAt: time.Now(), Quantity: quantity, Statistics: statistics}
}

func (c *ResultCache) oldestKey() string {
	oldest := ""
	var at time.Time
	for key, entry := range c.entries {
		if oldest == "" || entry.CreatedAt.Before(at) {
			oldest, at = key, entry.CreatedAt
		}
	}
	return oldest
}

func (c *ResultCache) EvictOldFiles() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.baseDir == "" {
		return
	}
	entries, err := os.ReadDir(c.baseDir)
	if err != nil {
		return
	}
	active := make(map[string]bool)
	for _, entry := range c.entries {
		active[filepath.Clean(entry.Workdir)] = true
	}
	cutoff := time.Now().Add(-c.ttl)
	for _, entry := range entries {
		path := filepath.Join(c.baseDir, entry.Name())
		if !entry.IsDir() || active[filepath.Clean(path)] {
			continue
		}
		// Only remove directories created by this cache, never unrelated files.
		if !strings.HasPrefix(entry.Name(), "snapshot-") {
			continue
		}
		if info, err := entry.Info(); err == nil && info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(path)
		}
	}
}

func (c *ResultCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, entry := range c.entries {
		if time.Since(entry.CreatedAt) > c.ttl {
			_ = os.RemoveAll(entry.Workdir)
			delete(c.entries, key)
		}
	}
}

func copyResultFiles(baseDir string, files []string) ([]string, string, error) {
	if baseDir != "" {
		if err := os.MkdirAll(baseDir, 0o755); err != nil {
			return nil, "", err
		}
	}
	dir, err := os.MkdirTemp(baseDir, "snapshot-")
	if err != nil {
		return nil, "", err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(dir)
		}
	}()
	copied := make([]string, 0, len(files))
	for _, src := range files {
		dst := filepath.Join(dir, filepath.Base(src))
		if err := copyFile(dst, src); err != nil {
			return nil, "", err
		}
		copied = append(copied, dst)
	}
	complete = true
	return copied, dir, nil
}

func copyFile(dst, src string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
