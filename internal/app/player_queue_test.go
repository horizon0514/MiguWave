package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestPlayerBarQueuePopover(t *testing.T) {
	a := testApp()
	tracks := sampleMiguTracks()
	a.pl.queue = append([]Track(nil), tracks...)
	a.pl.at = 0
	a.pl.current = tracks[0]
	a.pl.hasCurrent = true
	a.pl.duration = tracks[0].Duration()

	tt := ui.NewTester(a.view, 1240, 820)
	if err := tt.Click("打开待播清单"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.playerQueueOpen {
		t.Fatal("queue popover did not open")
	}
	for _, want := range []string{"待播清单", "2 首", tracks[1].Title} {
		if !tt.HasText(want) {
			t.Fatalf("queue popover does not show %q: %q", want, tt.Texts())
		}
	}
	if err := tt.Click("关闭待播清单"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.playerQueueOpen {
		t.Fatal("queue popover did not close")
	}
}

func TestSongTransportOpensQueue(t *testing.T) {
	track := sampleMiguTracks()[0]
	a := testApp()
	a.page = pageSong
	a.songTrack = track
	a.songQueue = sampleMiguTracks()
	a.songIndex = 0
	a.songDetail = miguSongDetail{Track: track}
	a.songCache[track.ID] = a.songDetail

	tt := ui.NewTester(a.view, 1240, 820)
	if err := tt.Click("查看待播清单"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.songPane != songPaneQueue {
		t.Fatalf("song pane = %d, want queue", a.songPane)
	}
	if !tt.HasText("接下来") {
		t.Fatalf("song page did not show the queue: %q", tt.Texts())
	}
}
