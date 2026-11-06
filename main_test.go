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
		page:         pageHome,
		volume:       0.8,
		arts:         map[string]ui.ImageSource{},
		artLoad:      map[string]bool{},
		artFail:      map[string]bool{},
		artTints:     map[string]ui.Color{},
		artBackdrops: map[string]*ui.Bitmap{},
		songCache:    map[string]miguSongDetail{},
		coverHQ:      map[string]string{},
		coverHQLoad:  map[string]bool{},
		coverHQFail:  map[string]bool{},
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

func TestSearchInteractionSubmits(t *testing.T) {
	a := testApp()
	tt := ui.NewTester(a.view, 1240, 820)
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Frame()
	if active, _ := tt.TextInputActive(); !active {
		t.Fatal("Cmd+F did not focus the search field")
	}
	tt.Type("晴天")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.page != pageSearch {
		t.Fatalf("page is %v after submitting search", a.page)
	}
	if a.searchTerm != "晴天" || a.searchInput != "晴天" {
		t.Fatalf("search input = %q, term = %q", a.searchInput, a.searchTerm)
	}
	if a.searchFrom != pageHome {
		t.Fatalf("search origin = %v, want home", a.searchFrom)
	}
}

func TestSearchResultHasBackButtonWithoutSidebarEntry(t *testing.T) {
	a := testApp()
	a.searchTerm = "周杰伦"
	a.searchInput = "周杰伦"
	a.searchTracks = sampleMiguTracks()
	a.searchFrom = pageFavorites
	a.page = pageSearch

	tt := ui.NewTester(a.view, 1240, 820)
	if tt.HasText("搜索结果") {
		t.Fatalf("search page should not add a sidebar entry: %q", tt.Texts())
	}
	if err := tt.Click("返回我的收藏"); err != nil {
		t.Fatal(err)
	}
	if a.page != pageFavorites {
		t.Fatalf("page is %v after clicking search back", a.page)
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

func TestLocalPlaylistNavigationAndCreate(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("AppData", home)
	} else {
		t.Setenv("HOME", home)
	}
	a := testApp()
	tt := ui.NewTester(a.view, 1240, 820)
	if err := tt.Click("自建歌单"); err != nil {
		t.Fatal(err)
	}
	if a.page != pageLocalPlaylists {
		t.Fatalf("page is %v after clicking local playlists", a.page)
	}
	if !tt.HasText("还没有自建歌单") {
		t.Fatalf("empty local playlist page: %q", tt.Texts())
	}
	if err := tt.Click("新建歌单"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Type("深夜循环")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if len(a.localPlaylists) != 1 {
		t.Fatalf("local playlists = %+v", a.localPlaylists)
	}
	if got := a.localPlaylists[0].Name; got != "深夜循环" {
		t.Fatalf("playlist name = %q", got)
	}
	if !tt.HasText("深夜循环") {
		t.Fatalf("created playlist not shown: %q", tt.Texts())
	}
}

func TestLocalPlaylistCRUD(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("AppData", home)
	} else {
		t.Setenv("HOME", home)
	}
	a := testApp()
	playlist, err := a.createLocalPlaylist("  深夜   循环  ")
	if err != nil {
		t.Fatal(err)
	}
	if playlist.Name != "深夜 循环" {
		t.Fatalf("normalized name = %q", playlist.Name)
	}
	if _, err := a.createLocalPlaylist("深夜 循环"); err == nil {
		t.Fatal("duplicate playlist name should be rejected")
	}
	track := sampleMiguTrack()
	track.URL = "https://example.invalid/expired-stream"
	added, err := a.addTrackToLocalPlaylist(playlist.ID, track)
	if err != nil || !added {
		t.Fatalf("add track = %v, %v", added, err)
	}
	if got := a.localPlaylists[0].Tracks[0].URL; got != "" {
		t.Fatalf("saved playlist kept a temporary stream URL: %q", got)
	}
	added, err = a.addTrackToLocalPlaylist(playlist.ID, track)
	if err != nil || added {
		t.Fatalf("duplicate add = %v, %v", added, err)
	}
	if err := a.renameLocalPlaylist(playlist.ID, "通勤"); err != nil {
		t.Fatal(err)
	}
	if !a.removeTrackFromLocalPlaylist(playlist.ID, track.ID) {
		t.Fatal("remove track returned false")
	}
	a.save()

	saved := loadSaved()
	if len(saved.LocalPlaylists) != 1 || saved.LocalPlaylists[0].Name != "通勤" {
		t.Fatalf("saved local playlists = %+v", saved.LocalPlaylists)
	}
	if !a.deleteLocalPlaylist(playlist.ID) {
		t.Fatal("delete playlist returned false")
	}
	if len(a.localPlaylists) != 0 {
		t.Fatalf("local playlists after delete = %+v", a.localPlaylists)
	}
}

func TestLocalPlaylistCardMenu(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("AppData", home)
	} else {
		t.Setenv("HOME", home)
	}
	a := testApp()
	playlist, err := a.createLocalPlaylist("深夜循环")
	if err != nil {
		t.Fatal(err)
	}
	a.localPlaylists = append(a.localPlaylists, localPlaylist{
		ID: "local-empty", Name: "空白歌单", Tracks: []Track{},
	})
	a.page = pageLocalPlaylists
	tt := ui.NewTester(a.view, 1240, 820)
	if err := tt.Click("歌单操作"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.page != pageLocalPlaylists {
		t.Fatalf("opening the card menu navigated to %v", a.page)
	}
	for _, label := range []string{"打开歌单", "播放全部", "重命名", "删除歌单"} {
		if !tt.HasText(label) {
			t.Fatalf("card menu does not show %q: %q", label, tt.Texts())
		}
	}
	if err := tt.Click("打开歌单"); err != nil {
		t.Fatal(err)
	}
	if a.page != pageLocalPlaylistDetail || a.localPlaylistID != playlist.ID {
		t.Fatalf("opened page = %v, playlist = %q", a.page, a.localPlaylistID)
	}
}

func TestSongDetailAddsTrackToLocalPlaylist(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("AppData", home)
	} else {
		t.Setenv("HOME", home)
	}
	a := testApp()
	playlist, err := a.createLocalPlaylist("深夜循环")
	if err != nil {
		t.Fatal(err)
	}
	track := sampleMiguTrack()
	a.page = pageSong
	a.songTrack = track
	a.songQueue = []Track{track}
	a.songDetail = miguSongDetail{Track: track, Album: track.Album}

	tt := ui.NewTester(a.view, 1240, 820)
	if err := tt.Click("添加到自建歌单"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("深夜循环") {
		t.Fatalf("playlist menu does not show the local playlist: %q", tt.Texts())
	}
	if err := tt.Click("深夜循环"); err != nil {
		t.Fatal(err)
	}

	updated, index := a.localPlaylistByID(playlist.ID)
	if index < 0 || len(updated.Tracks) != 1 || updated.Tracks[0].ID != track.ID {
		t.Fatalf("playlist after adding from song detail = %+v", updated)
	}
	saved := loadSaved()
	if len(saved.LocalPlaylists) != 1 || len(saved.LocalPlaylists[0].Tracks) != 1 ||
		saved.LocalPlaylists[0].Tracks[0].ID != track.ID {
		t.Fatalf("saved playlist after adding from song detail = %+v", saved.LocalPlaylists)
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
		{"playlists", func(a *app) {
			a.page = pagePlaylists
			a.playlistSections = []miguPlaylistSection{{Title: "编辑推荐", Items: samplePlaylists()}}
		}, "编辑推荐"},
		{"playlist-detail", func(a *app) {
			a.page = pagePlaylistDetail
			a.playlistFrom = pagePlaylists
			a.playlistID = samplePlaylists()[0].ID
			a.playlistDetail = miguPlaylistDetail{
				Playlist: samplePlaylists()[0], Tracks: tracks, Page: 1, Total: 247,
			}
		}, "华语热歌精选"},
		{"saved-playlists", func(a *app) {
			a.page = pageSavedPlaylists
			a.savedPlaylists = samplePlaylists()
		}, "收藏歌单"},
		{"singers", func(a *app) {
			a.page = pageSingers
			a.singerTab = miguSingerTabs[0].ID
			a.singerVisible = 30
			a.singers = sampleSingers()
		}, "周杰伦"},
		{"singer-detail", func(a *app) {
			a.page = pageSingerDetail
			a.singerDetail = miguSingerDetail{
				Singer: sampleSingers()[0], Bio: "华语流行歌手", Tracks: tracks,
				Similar: sampleSingers()[1:], Page: 1,
			}
		}, "相似歌手"},
		{"radios", func(a *app) {
			a.page = pageRadios
			a.radioGroups = sampleRadioGroups()
		}, "私享电台"},
		{"radio-detail", func(a *app) {
			a.page = pageRadioDetail
			a.radioFrom = pageRadios
			a.radioDetail = miguRadioDetail{
				Radio: sampleRadioGroups()[1].Items[0], Tracks: tracks, Total: len(tracks),
			}
		}, "电台歌曲"},
		{"local-playlists", func(a *app) {
			a.page = pageLocalPlaylists
			a.localPlaylists = []localPlaylist{{
				ID: "local-1", Name: "深夜循环", Tracks: tracks,
			}}
		}, "自建歌单"},
		{"local-playlist-detail", func(a *app) {
			a.page = pageLocalPlaylistDetail
			a.localPlaylistID = "local-1"
			a.localPlaylists = []localPlaylist{{
				ID: "local-1", Name: "深夜循环", Tracks: tracks,
			}}
		}, "LOCAL PLAYLIST"},
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
	a.localPlaylists = []localPlaylist{{
		ID: "local-roundtrip", Name: "通勤", Tracks: sampleMiguTracks(),
	}}
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
	if len(saved.LocalPlaylists) != 1 || saved.LocalPlaylists[0].Name != "通勤" ||
		len(saved.LocalPlaylists[0].Tracks) != 2 {
		t.Errorf("local playlists = %+v", saved.LocalPlaylists)
	}
	if got := statePath(); got != filepath.Join(configDir(), "state.json") {
		t.Errorf("statePath = %q", got)
	}
	if !strings.HasPrefix(configDir(), home) {
		t.Errorf("configDir = %q, want it under %q", configDir(), home)
	}
}
