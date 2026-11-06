package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanAudioCache(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	write := func(name string, size int, age time.Duration) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
			t.Fatal(err)
		}
		mod := now.Add(-age)
		if err := os.Chtimes(path, mod, mod); err != nil {
			t.Fatal(err)
		}
		return path
	}

	old := write("old.mp3", 40, 31*24*time.Hour)
	abandoned := write("stale.mp3.123.part", 30, 25*time.Hour)
	oldest := write("oldest.mp3", 50, 2*time.Hour)
	newer := write("newer.mp3", 60, time.Hour)

	result, err := cleanAudioCache(dir, now, 100)
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed != 3 || result.Freed != 120 {
		t.Fatalf("result = %+v", result)
	}
	for _, path := range []string{old, abandoned, oldest} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("%s still exists", path)
		}
	}
	if _, err := os.Stat(newer); err != nil {
		t.Errorf("newest cache file was removed: %v", err)
	}
}

func TestCleanAudioCacheMissingDirectory(t *testing.T) {
	result, err := cleanAudioCache(filepath.Join(t.TempDir(), "missing"), time.Now(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed != 0 || result.Freed != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestPlayerCacheKeySeparatesAccounts(t *testing.T) {
	a := testApp()
	p := a.pl
	track := sampleMiguTrack()
	anonymous := p.cacheKey(track)
	a.miguToken = "first-account-token"
	first := p.cacheKey(track)
	a.miguToken = "second-account-token"
	second := p.cacheKey(track)
	if anonymous == first || first == second || anonymous == second {
		t.Fatalf("cache keys should differ by account: %q %q %q", anonymous, first, second)
	}
}
