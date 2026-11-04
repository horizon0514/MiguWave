package main

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func testApp() *app {
	a := &app{
		page:    pageHome,
		volume:  0.8,
		arts:    map[string]ui.ImageSource{},
		artLoad: map[string]bool{},
		artFail: map[string]bool{},
		hotSearches: []string{
			"周杰伦", "林俊杰", "邓紫棋",
		},
	}
	a.pl = &player{app: a, at: -1, volume: a.volume}
	return a
}

func sampleMiguTrack() Track {
	return Track{
		ID:           "migu:600913000007163534",
		Title:        "晴天",
		Artist:       "周杰伦",
		Album:        "叶惠美",
		DurationSec:  269,
		Source:       "migu",
		CopyrightID:  "60054704101",
		ResourceType: "2",
		PlayCount:    "1.2亿",
	}
}

func sampleMiguTracks() []Track {
	first := sampleMiguTrack()
	second := Track{
		ID:           "migu:600902000006889366",
		Title:        "以父之名",
		Artist:       "周杰伦",
		Album:        "叶惠美",
		DurationSec:  342,
		Source:       "migu",
		CopyrightID:  "60054701923",
		ResourceType: "2",
		VIP:          true,
	}
	return []Track{first, second}
}

func sampleRank() miguRank {
	return miguRank{
		ID:         "83176390",
		Title:      "国风热歌榜",
		UpdateTime: "2026-11-04",
		Tracks:     sampleMiguTracks(),
	}
}

func TestHomeView(t *testing.T) {
	a := testApp()
	a.home.NewSongs = sampleMiguTracks()
	a.home.Ranks = []miguRankSummary{{
		ID:       "83176390",
		Title:    "国风热歌榜",
		Subtitle: "每周更新",
		Tracks:   sampleMiguTracks(),
	}}

	tt := ui.NewTester(a.view, 1240, 820)
	for _, want := range []string{"MiguWave", "发现", "新歌速递", "晴天", "热门榜单"} {
		if !tt.HasText(want) {
			t.Errorf("home view does not show %q", want)
		}
	}
}

func TestNavigation(t *testing.T) {
	a := testApp()
	a.rank = sampleRank()
	a.rankID = a.rank.ID
	tt := ui.NewTester(a.view, 1240, 820)
	if err := tt.Click("排行榜"); err != nil {
		t.Fatal(err)
	}
	if a.page != pageRank {
		t.Fatalf("page is %v after clicking rank", a.page)
	}
	if !tt.HasText("国风热歌榜") {
		t.Fatalf("rank page does not show the chart: %q", tt.Texts())
	}
}

func TestLibrarySelection(t *testing.T) {
	a := testApp()
	tt := ui.NewTester(a.view, 1240, 820)
	if err := tt.Click("我的收藏"); err != nil {
		t.Fatal(err)
	}
	if a.page != pageFavorites {
		t.Fatalf("page is %v after clicking favorites", a.page)
	}
	if !tt.HasText("这里还是空的") {
		t.Fatalf("empty favorites does not show its empty state: %q", tt.Texts())
	}
}

func TestAllPagesRender(t *testing.T) {
	tracks := sampleMiguTracks()
	tests := []struct {
		name  string
		setup func(*app)
		want  string
	}{
		{"home", func(a *app) {
			a.page = pageHome
			a.home.NewSongs = tracks
		}, "新歌速递"},
		{"rank", func(a *app) {
			a.page = pageRank
			a.rank = sampleRank()
			a.rankID = a.rank.ID
		}, "完整榜单"},
		{"favorites", func(a *app) {
			a.page = pageFavorites
			a.favorites = tracks
		}, "我的收藏"},
		{"recent", func(a *app) {
			a.page = pageRecent
			a.recent = tracks
		}, "最近播放"},
		{"search", func(a *app) {
			a.page = pageSearch
			a.searchTerm = "周杰伦"
			a.searchTracks = tracks
		}, "周杰伦"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := testApp()
			tc.setup(a)
			tt := ui.NewTester(a.view, 1240, 820)
			if !tt.HasText(tc.want) {
				t.Fatalf("%s page does not show %q: %q", tc.name, tc.want, tt.Texts())
			}
			if got := tt.Image(); got == nil || got.Bounds().Empty() {
				t.Fatalf("%s page rendered no image", tc.name)
			}
		})
	}
}

func TestParsingHelpers(t *testing.T) {
	if got := formatClock(3723 * time.Second); got != "1:02:03" {
		t.Errorf("formatClock = %q", got)
	}
	if got := cleanDisplay("<em>周杰伦</em>  晴天"); got != "<em>周杰伦</em> 晴天" {
		t.Errorf("cleanDisplay = %q", got)
	}
	track := sampleMiguTrack()
	if !track.NeedsResolve() {
		t.Error("Migu track should require resolving a stream URL")
	}
	if got := track.SourceLabel(); got != "咪咕音乐" {
		t.Errorf("SourceLabel = %q", got)
	}
	vip := sampleMiguTracks()[1]
	if !vip.VIP {
		t.Error("VIP track should keep its VIP flag")
	}
}

func TestSavedStateRoundTrip(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("AppData", home)
	} else {
		t.Setenv("HOME", home)
	}
	a := testApp()
	a.volume = 0.37
	a.favorites = sampleMiguTracks()[:1]
	a.recent = sampleMiguTracks()[1:]
	a.save()

	saved := loadSaved()
	if saved.Volume != 0.37 {
		t.Errorf("volume = %v", saved.Volume)
	}
	if len(saved.Favorites) != 1 || saved.Favorites[0].Title != "晴天" {
		t.Errorf("favorites = %+v", saved.Favorites)
	}
	if len(saved.Recent) != 1 || saved.Recent[0].Title != "以父之名" {
		t.Errorf("recent = %+v", saved.Recent)
	}
	if got := statePath(); got != filepath.Join(configDir(), "state.json") {
		t.Errorf("statePath = %q", got)
	}
	if !strings.HasPrefix(configDir(), home) {
		t.Errorf("configDir = %q, want it under %q", configDir(), home)
	}
}
