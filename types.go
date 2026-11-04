package main

import (
	"fmt"
	"time"
)

// Track is one song from the Migu catalogue.
type Track struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Artist       string  `json:"artist"`
	Album        string  `json:"album"`
	Cover        string  `json:"cover,omitempty"`
	URL          string  `json:"url,omitempty"`
	DurationSec  float64 `json:"durationSec,omitempty"`
	Source       string  `json:"source,omitempty"`
	CopyrightID  string  `json:"copyrightId,omitempty"`
	ResourceType string  `json:"resourceType,omitempty"`
	VIP          bool    `json:"vip,omitempty"`
	PlayCount    string  `json:"playCount,omitempty"`
}

func (t Track) Duration() time.Duration {
	return time.Duration(t.DurationSec * float64(time.Second))
}

func (t Track) Subtitle() string {
	if t.Artist != "" {
		return t.Artist
	}
	return "未知艺人"
}

// SourceLabel is the platform name shown on track cards.
func (t Track) SourceLabel() string {
	if t.Source == "migu" {
		return "咪咕音乐"
	}
	return ""
}

// NeedsResolve reports whether playback first has to exchange a track id for
// a short-lived stream URL.
func (t Track) NeedsResolve() bool {
	return t.Source == "migu" && t.URL == ""
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
