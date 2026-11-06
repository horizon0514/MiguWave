package migu

import "time"

// Track is one song from the Migu catalogue.
type Track struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Artist       string  `json:"artist"`
	Album        string  `json:"album"`
	Cover        string  `json:"cover,omitempty"`
	CoverHQ      string  `json:"-"`
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
