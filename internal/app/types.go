package app

import (
	"fmt"
	"time"

	"freewave/internal/lyrics"
	"freewave/internal/migu"
)

type Track = migu.Track
type lyricLine = lyrics.Line

// localPlaylist is a playlist kept by this app. It never changes the Migu
// account, so every mutation stays local and is restored from state.json.
type localPlaylist struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Tracks []Track `json:"tracks,omitempty"`
}

func (p localPlaylist) Duration() time.Duration {
	var total time.Duration
	for _, track := range p.Tracks {
		total += track.Duration()
	}
	return total
}

func (p localPlaylist) Contains(trackID string) bool {
	for _, track := range p.Tracks {
		if track.ID == trackID {
			return true
		}
	}
	return false
}

// formatClock formats a duration as M:SS or H:MM:SS.
func formatClock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Round(time.Second) / time.Second)
	h, m, s := total/3600, total/60%60, total%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
