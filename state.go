package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
)

type savedState struct {
	Volume    float64 `json:"volume"`
	Favorites []Track `json:"favorites,omitempty"`
	Recent    []Track `json:"recent,omitempty"`
}

func configDir() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "MiguWave")
}

func statePath() string {
	return filepath.Join(configDir(), "state.json")
}

// loadSaved reads favorites and recent plays, ignoring a missing file.
func loadSaved() savedState {
	s := savedState{Volume: 0.8}
	b, err := os.ReadFile(statePath())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	if s.Volume < 0 || s.Volume > 1 {
		s.Volume = 0.8
	}
	if s.Favorites == nil {
		s.Favorites = []Track{}
	}
	if s.Recent == nil {
		s.Recent = []Track{}
	}
	return s
}

// save writes the library to disk atomically enough for a desktop app.
func (a *app) save() {
	s := savedState{
		Volume:    a.volume,
		Favorites: a.favorites,
		Recent:    a.recent,
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	dir := configDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	tmp := statePath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, statePath())
}

func (a *app) isFavorite(id string) bool {
	return slices.ContainsFunc(a.favorites, func(t Track) bool { return t.ID == id })
}

// toggleFavorite adds or removes a track and returns whether it is now saved.
func (a *app) toggleFavoriteState(t Track) bool {
	for i, old := range a.favorites {
		if old.ID == t.ID {
			a.favorites = append(a.favorites[:i], a.favorites[i+1:]...)
			a.save()
			return false
		}
	}
	a.favorites = append([]Track{t}, a.favorites...)
	if len(a.favorites) > 200 {
		a.favorites = a.favorites[:200]
	}
	a.save()
	return true
}

// noteRecent remembers a played track, newest first, without duplicates.
func (a *app) noteRecent(t Track) {
	a.recent = slices.DeleteFunc(a.recent, func(old Track) bool { return old.ID == t.ID })
	a.recent = append([]Track{t}, a.recent...)
	if len(a.recent) > 30 {
		a.recent = a.recent[:30]
	}
	a.save()
}
