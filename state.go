package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type closeAction string

const (
	closeActionMenuBar closeAction = "menu"
	closeActionQuit    closeAction = "quit"
)

type savedState struct {
	Volume         float64         `json:"volume"`
	CloseAction    closeAction     `json:"closeAction,omitempty"`
	Favorites      []Track         `json:"favorites,omitempty"`
	Recent         []Track         `json:"recent,omitempty"`
	SavedPlaylists []miguPlaylist  `json:"savedPlaylists,omitempty"`
	LocalPlaylists []localPlaylist `json:"localPlaylists,omitempty"`
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
	s := savedState{Volume: 0.8, CloseAction: closeActionMenuBar}
	b, err := os.ReadFile(statePath())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	if s.Volume < 0 || s.Volume > 1 {
		s.Volume = 0.8
	}
	s.CloseAction = normalizeCloseAction(s.CloseAction)
	if s.Favorites == nil {
		s.Favorites = []Track{}
	}
	if s.Recent == nil {
		s.Recent = []Track{}
	}
	if s.SavedPlaylists == nil {
		s.SavedPlaylists = []miguPlaylist{}
	}
	if s.LocalPlaylists == nil {
		s.LocalPlaylists = []localPlaylist{}
	}
	for i := range s.LocalPlaylists {
		if s.LocalPlaylists[i].Tracks == nil {
			s.LocalPlaylists[i].Tracks = []Track{}
		}
	}
	return s
}

func normalizeCloseAction(action closeAction) closeAction {
	if action == closeActionQuit {
		return closeActionQuit
	}
	return closeActionMenuBar
}

// save writes the library to disk atomically enough for a desktop app.
func (a *app) save() {
	s := savedState{
		Volume:         a.volume,
		CloseAction:    normalizeCloseAction(a.closeAction),
		Favorites:      a.favorites,
		Recent:         a.recent,
		SavedPlaylists: a.savedPlaylists,
		LocalPlaylists: a.localPlaylists,
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

func (a *app) isPlaylistSaved(id string) bool {
	return slices.ContainsFunc(a.savedPlaylists, func(p miguPlaylist) bool { return p.ID == id })
}

func (a *app) togglePlaylistSaved(p miguPlaylist) bool {
	for i, old := range a.savedPlaylists {
		if old.ID == p.ID {
			a.savedPlaylists = append(a.savedPlaylists[:i], a.savedPlaylists[i+1:]...)
			a.save()
			return false
		}
	}
	a.savedPlaylists = append([]miguPlaylist{p}, a.savedPlaylists...)
	if len(a.savedPlaylists) > 100 {
		a.savedPlaylists = a.savedPlaylists[:100]
	}
	a.save()
	return true
}

const localPlaylistNameLimit = 40

func newLocalPlaylistID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "local-" + hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("local-%d", time.Now().UnixNano())
}

func normalizeLocalPlaylistName(raw string) string {
	name := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	runes := []rune(name)
	if len(runes) > localPlaylistNameLimit {
		name = string(runes[:localPlaylistNameLimit])
	}
	return name
}

func (a *app) localPlaylistByID(id string) (localPlaylist, int) {
	for i, playlist := range a.localPlaylists {
		if playlist.ID == id {
			return playlist, i
		}
	}
	return localPlaylist{}, -1
}

func (a *app) localPlaylistNameTaken(name, exceptID string) bool {
	for _, playlist := range a.localPlaylists {
		if playlist.ID != exceptID && strings.EqualFold(playlist.Name, name) {
			return true
		}
	}
	return false
}

func (a *app) createLocalPlaylist(rawName string) (localPlaylist, error) {
	name := normalizeLocalPlaylistName(rawName)
	if name == "" {
		return localPlaylist{}, errors.New("歌单名称不能为空")
	}
	if a.localPlaylistNameTaken(name, "") {
		return localPlaylist{}, errors.New("已经有同名歌单了")
	}
	playlist := localPlaylist{
		ID:     newLocalPlaylistID(),
		Name:   name,
		Tracks: []Track{},
	}
	a.localPlaylists = append([]localPlaylist{playlist}, a.localPlaylists...)
	a.save()
	return playlist, nil
}

func (a *app) renameLocalPlaylist(id, rawName string) error {
	name := normalizeLocalPlaylistName(rawName)
	if name == "" {
		return errors.New("歌单名称不能为空")
	}
	if a.localPlaylistNameTaken(name, id) {
		return errors.New("已经有同名歌单了")
	}
	_, index := a.localPlaylistByID(id)
	if index < 0 {
		return errors.New("歌单不存在")
	}
	a.localPlaylists[index].Name = name
	a.save()
	return nil
}

func (a *app) deleteLocalPlaylist(id string) bool {
	_, index := a.localPlaylistByID(id)
	if index < 0 {
		return false
	}
	a.localPlaylists = append(a.localPlaylists[:index], a.localPlaylists[index+1:]...)
	a.save()
	return true
}

func (a *app) addTrackToLocalPlaylist(id string, track Track) (bool, error) {
	if strings.TrimSpace(track.ID) == "" {
		return false, errors.New("这首歌暂时不能加入歌单")
	}
	_, index := a.localPlaylistByID(id)
	if index < 0 {
		return false, errors.New("歌单不存在")
	}
	if a.localPlaylists[index].Contains(track.ID) {
		return false, nil
	}
	track.URL = ""
	a.localPlaylists[index].Tracks = append(a.localPlaylists[index].Tracks, track)
	a.save()
	return true, nil
}

func (a *app) removeTrackFromLocalPlaylist(id, trackID string) bool {
	_, index := a.localPlaylistByID(id)
	if index < 0 {
		return false
	}
	tracks := a.localPlaylists[index].Tracks
	for i, track := range tracks {
		if track.ID != trackID {
			continue
		}
		a.localPlaylists[index].Tracks = append(tracks[:i], tracks[i+1:]...)
		a.save()
		return true
	}
	return false
}
