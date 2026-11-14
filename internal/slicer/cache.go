package slicer

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type CacheEntry struct {
	Key       string
	Files     []string
	Workdir   string
	CreatedAt time.Time
}

type ResultCache struct {
	mu      sync.RWMutex
	entries map[string]CacheEntry
	ttl     time.Duration
	max     int
	baseDir string
}

func NewResultCache(ttl time.Duration, maxEntries int) *ResultCache {
	return &ResultCache{
		entries: make(map[string]CacheEntry),
		ttl:     ttl,
		max:     maxEntries,
	}
}

func NewPersistentResultCache(ttl time.Duration, maxEntries int, baseDir string) *ResultCache {
	cache := NewResultCache(ttl, maxEntries)
	cache.baseDir = baseDir
	return cache
}

func (c *ResultCache) makeModelsKey(models []ModelInput, settings Settings) string {
	h := sha256.New()
	for _, model := range models {
		h.Write(model.Data)
		h.Write([]byte(model.Filename))
	}
	return c.makeSettingsKey(h, settings)
}

func (c *ResultCache) makeSettingsKey(h hash.Hash, settings Settings) string {
	h.Write([]byte(settings.Printer + "|" + settings.Preset + "|" + settings.Filament))
	h.Write([]byte(settings.ExportType))
	if settings.EnableSupport != nil {
		if *settings.EnableSupport {
			h.Write([]byte("es1"))
		} else {
			h.Write([]byte("es0"))
		}
	}
	if settings.BrimType != nil {
		if *settings.BrimType {
			h.Write([]byte("bt1"))
		} else {
			h.Write([]byte("bt0"))
		}
	}
	if settings.PrintSequenceByObject {
		h.Write([]byte("pso1"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (c *ResultCache) Get(key string) (CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return CacheEntry{}, false
	}
	if time.Since(entry.CreatedAt) > c.ttl {
		return CacheEntry{}, false
	}
	for _, file := range entry.Files {
		if _, err := os.Stat(file); err != nil {
			return CacheEntry{}, false
		}
	}
	return entry, true
}

func (c *ResultCache) Put(key string, files []string, workdir string) {
	storedFiles := append([]string(nil), files...)
	storedWorkdir := workdir
	if c.baseDir != "" {
		cachedFiles, cachedWorkdir, err := c.copyFiles(key, files)
		if err == nil {
			storedFiles = cachedFiles
			storedWorkdir = cachedWorkdir
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.max {
		oldest := c.oldestKey()
		if oldest != "" {
			if old := c.entries[oldest]; c.baseDir != "" {
				_ = os.RemoveAll(old.Workdir)
			}
			delete(c.entries, oldest)
		}
	}

	c.entries[key] = CacheEntry{
		Key:       key,
		Files:     storedFiles,
		Workdir:   storedWorkdir,
		CreatedAt: time.Now(),
	}
}

func (c *ResultCache) oldestKey() string {
	oldest := ""
	var oldestTime time.Time
	for key, entry := range c.entries {
		if oldest == "" || entry.CreatedAt.Before(oldestTime) {
			oldest = key
			oldestTime = entry.CreatedAt
		}
	}
	return oldest
}

func (c *ResultCache) EvictOldFiles() {
	if c.baseDir == "" {
		return
	}
	entries, err := os.ReadDir(c.baseDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-c.ttl)
	for _, entry := range entries {
		if entry.IsDir() {
			info, err := entry.Info()
			if err == nil && info.ModTime().Before(cutoff) {
				_ = os.RemoveAll(filepath.Join(c.baseDir, entry.Name()))
			}
		}
	}
}

func (c *ResultCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, entry := range c.entries {
		if time.Since(entry.CreatedAt) > c.ttl {
			if c.baseDir != "" {
				_ = os.RemoveAll(entry.Workdir)
			}
			delete(c.entries, key)
		}
	}
}

func (c *ResultCache) copyFiles(key string, files []string) ([]string, string, error) {
	workdir := filepath.Join(c.baseDir, key)
	if err := os.RemoveAll(workdir); err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return nil, "", err
	}

	cachedFiles := make([]string, 0, len(files))
	for _, src := range files {
		dst := filepath.Join(workdir, filepath.Base(src))
		if err := copyFile(dst, src); err != nil {
			return nil, "", err
		}
		cachedFiles = append(cachedFiles, dst)
	}
	return cachedFiles, workdir, nil
}

func copyFile(dst string, src string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, input)
	return err
}
