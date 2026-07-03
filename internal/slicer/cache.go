package slicer

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
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
}

func NewResultCache(ttl time.Duration, maxEntries int) *ResultCache {
	return &ResultCache{
		entries: make(map[string]CacheEntry),
		ttl:     ttl,
		max:     maxEntries,
	}
}

func (c *ResultCache) makeModelsKey(models []ModelInput, settings Settings) string {
	h := sha256.New()
	for _, model := range models {
		h.Write(model.Data)
		h.Write([]byte(model.Filename))
	}
	return c.makeSettingsKey(h, settings)
}

func (c *ResultCache) makeKey(filename string, data []byte, settings Settings) string {
	h := sha256.New()
	h.Write(data)
	h.Write([]byte(filename))
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
	e, ok := c.entries[key]
	if !ok {
		return CacheEntry{}, false
	}
	if time.Since(e.CreatedAt) > c.ttl {
		return CacheEntry{}, false
	}
	return e, true
}

func (c *ResultCache) Put(key string, files []string, workdir string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.max {
		// Evict oldest
		var oldest string
		var oldestTime time.Time
		for k, v := range c.entries {
			if oldest == "" || v.CreatedAt.Before(oldestTime) {
				oldest = k
				oldestTime = v.CreatedAt
			}
		}
		if oldest != "" {
			delete(c.entries, oldest)
		}
	}

	c.entries[key] = CacheEntry{
		Key:       key,
		Files:     append([]string(nil), files...),
		Workdir:   workdir,
		CreatedAt: time.Now(),
	}
}
