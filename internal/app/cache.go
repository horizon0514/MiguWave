package app

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	audioCacheMaxBytes    int64 = 2 << 30
	audioCacheMaxAge            = 30 * 24 * time.Hour
	audioCachePartMaxAge        = 24 * time.Hour
	maxAudioDownloadBytes int64 = 300 << 20
)

type cacheCleanupResult struct {
	Removed int
	Freed   int64
}

type cacheEntry struct {
	path    string
	size    int64
	modTime time.Time
}

func audioCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "MiguWave", "audio")
}

// cleanAudioCache removes abandoned partial downloads, expires old tracks, and
// keeps the cache under maxBytes. It is intended to run at startup, before the
// player can have an open cache file.
func cleanAudioCache(dir string, now time.Time, maxBytes int64) (cacheCleanupResult, error) {
	var result cacheCleanupResult
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, err
	}

	var failures []error
	var cached []cacheEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			failures = append(failures, err)
			continue
		}
		path := filepath.Join(dir, entry.Name())
		age := now.Sub(info.ModTime())
		expired := age > audioCacheMaxAge
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".part") {
			expired = age > audioCachePartMaxAge
		}
		if expired {
			if err := os.Remove(path); err != nil {
				failures = append(failures, err)
				continue
			}
			result.Removed++
			result.Freed += info.Size()
			continue
		}
		cached = append(cached, cacheEntry{path: path, size: info.Size(), modTime: info.ModTime()})
	}

	sort.Slice(cached, func(i, j int) bool {
		return cached[i].modTime.Before(cached[j].modTime)
	})
	var total int64
	for _, entry := range cached {
		total += entry.size
	}
	for _, entry := range cached {
		if total <= maxBytes {
			break
		}
		if err := os.Remove(entry.path); err != nil {
			failures = append(failures, err)
			continue
		}
		total -= entry.size
		result.Removed++
		result.Freed += entry.size
	}
	return result, errors.Join(failures...)
}
