package app

import (
	"bytes"
	"image/png"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

func TestShouldShowMainOnActivate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name              string
		hasVisibleWindows bool
		dockVisible       bool
		panelOpen         bool
		usedAgo           time.Duration
		neverUsed         bool
		want              bool
	}{
		{
			name:              "visible windows need no reopen",
			hasVisibleWindows: true,
			dockVisible:       true,
			want:              false,
		},
		{
			name:        "panel activation does not open the main window",
			dockVisible: true,
			panelOpen:   true,
			want:        false,
		},
		{
			name:        "recent menu bar click does not open the main window",
			dockVisible: true,
			usedAgo:     200 * time.Millisecond,
			want:        false,
		},
		{
			name:        "dock activation after the guard opens the main window",
			dockVisible: true,
			usedAgo:     time.Second,
			want:        true,
		},
		{
			name:        "dock activation without menu bar use opens the main window",
			dockVisible: true,
			neverUsed:   true,
			want:        true,
		},
		{
			name:        "activation in menu bar mode does not open the main window",
			dockVisible: false,
			neverUsed:   true,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := testApp()
			if tt.panelOpen {
				a.menuBarPanel = &mygo.Window{}
			}
			if !tt.neverUsed {
				a.menuBarUsedAt = now.Add(-tt.usedAgo)
			}
			if got := a.shouldShowMainOnActivate(tt.hasVisibleWindows, tt.dockVisible, now); got != tt.want {
				t.Fatalf("shouldShowMainOnActivate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMenuBarPlaybackIcons(t *testing.T) {
	staticIcon := menuBarIcon()
	seen := map[string]bool{}
	for i, icon := range menuBarPlaybackIcons {
		if bytes.Equal(icon, staticIcon) {
			t.Fatalf("frame %d matches the static icon", i)
		}
		if seen[string(icon)] {
			t.Fatalf("frame %d duplicates an earlier frame", i)
		}
		seen[string(icon)] = true

		img, err := png.Decode(bytes.NewReader(icon))
		if err != nil {
			t.Fatalf("frame %d is not a PNG: %v", i, err)
		}
		if got := img.Bounds().Size(); got.X != 32 || got.Y != 32 {
			t.Fatalf("frame %d is %v, want 32x32", i, got)
		}
	}
}
