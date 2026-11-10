package app

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"freewave/internal/migu"
)

type liveArt struct {
	src      ui.ImageSource
	tint     ui.Color
	backdrop *ui.Bitmap
}

func TestLiveSnapshots(t *testing.T) {
	if os.Getenv("MIGUWAVE_LIVE_SNAPSHOTS") == "" {
		t.Skip("set MIGUWAVE_LIVE_SNAPSHOTS=1 to write live snapshots")
	}
	dir := "/tmp/miguwave-live-snapshots"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	homeData, err := migu.FetchHome(ctx)
	if err != nil {
		t.Fatal(err)
	}
	rank, err := migu.FetchRank(ctx, migu.RankCatalog[0].ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	hits, err := migu.Search(ctx, "周杰伦", 1, 20)
	if err != nil {
		t.Fatal(err)
	}

	stageTrack := hits[0]
	for _, track := range hits {
		if track.Title == "晴天" {
			stageTrack = track
			break
		}
	}
	detail, err := migu.FetchSongDetail(ctx, migu.LoadToken(), stageTrack)
	if err != nil {
		t.Logf("song detail: %v", err)
		detail = miguSongDetail{Track: stageTrack, Album: stageTrack.Album, Cover: stageTrack.Cover}
	}
	if hq := migu.ResolveCoverHQ(ctx, stageTrack); hq != "" {
		detail.Cover = hq
		detail.Track.Cover = hq
	}
	if detail.Cover == "" {
		detail.Cover = stageTrack.Cover
	}
	lyricAt := liveStageLyricTime(detail.Lyrics)

	urls := []string{detail.Cover, detail.Track.Cover}
	for _, track := range homeData.NewSongs {
		urls = append(urls, track.Cover)
	}
	for _, playlist := range homeData.Playlists {
		urls = append(urls, playlist.Cover, playlist.CoverHQ)
	}
	for _, summary := range homeData.Ranks {
		urls = append(urls, summary.Cover)
		for _, track := range summary.Tracks {
			urls = append(urls, track.Cover)
		}
	}
	urls = append(urls, rank.Cover)
	for _, track := range rank.Tracks {
		urls = append(urls, track.Cover)
	}
	for _, track := range hits {
		urls = append(urls, track.Cover)
	}
	art := loadLiveArt(ctx, t, urls)

	saveView := func(name string, view func(*ui.Context), width, height int) {
		t.Helper()
		tt := ui.NewTester(view, width, height)
		tt.SetScale(2)
		tt.Frame()
		tt.Advance(900 * time.Millisecond)
		tt.Frame()
		tt.Advance(300 * time.Millisecond)
		tt.Frame()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
	save := func(name string, a *app) {
		seedLiveArt(a, art)
		saveView(name, a.view, 1240, 820)
	}

	home := testApp()
	home.home = homeData
	save("home", home)

	search := testApp()
	search.page = pageSearch
	search.searchTerm = "周杰伦"
	search.searchInput = "周杰伦"
	search.searchTracks = hits
	search.searchHasMore = true
	save("search", search)

	rankView := testApp()
	rankView.page = pageRank
	rankView.rank = rank
	rankView.rankID = rank.ID
	save("rank", rankView)

	stageIndex := 0
	for i, track := range hits {
		if track.ID == stageTrack.ID {
			stageIndex = i
			break
		}
	}
	stage := testApp()
	stage.page = pageSong
	stage.songPane = songPaneStage
	stage.songTrack = stageTrack
	stage.songQueue = hits
	stage.songIndex = stageIndex
	stage.songDetail = detail
	stage.songCache[stageTrack.ID] = detail
	stage.pl.queue = append([]Track(nil), hits...)
	stage.pl.at = stageIndex
	stage.pl.current = stageTrack
	stage.pl.hasCurrent = true
	stage.pl.playing = true
	stage.pl.duration = stageTrack.Duration()
	stage.pl.pending = lyricAt
	save("song-stage", stage)

	menu := testApp()
	menu.menuDetailID = stageTrack.ID
	menu.menuDetail = detail
	menu.pl.current = stageTrack
	menu.pl.hasCurrent = true
	menu.pl.playing = true
	menu.pl.duration = stageTrack.Duration()
	menu.pl.pending = lyricAt
	seedLiveArt(menu, art)
	saveView("menu-bar", menu.menuBarView, 420, 536)
}

func liveStageLyricTime(lines []lyricLine) time.Duration {
	for _, line := range lines {
		if strings.Contains(line.Text, "童年的荡秋千") {
			return line.At + 2*time.Second
		}
	}
	for _, line := range lines {
		text := strings.TrimSpace(line.Text)
		if line.At >= 10*time.Second && len([]rune(text)) >= 8 && !strings.Contains(text, "：") {
			return line.At + 2*time.Second
		}
	}
	return 0
}

func loadLiveArt(ctx context.Context, t *testing.T, urls []string) map[string]liveArt {
	t.Helper()
	art := map[string]liveArt{}
	for _, raw := range urls {
		if raw == "" {
			continue
		}
		if _, ok := art[raw]; ok {
			continue
		}
		data, err := fetchBytes(ctx, raw)
		if err != nil {
			t.Logf("cover %s: %v", raw, err)
			continue
		}
		src, err := ui.DecodeImage(data)
		if err != nil {
			t.Logf("cover %s: %v", raw, err)
			continue
		}
		tint, backdrop := inspectCover(data)
		art[raw] = liveArt{src: src, tint: tint, backdrop: backdrop}
	}
	return art
}

func seedLiveArt(a *app, art map[string]liveArt) {
	for raw, item := range art {
		a.arts[raw] = item.src
		if item.tint != (ui.Color{}) {
			a.artTints[raw] = item.tint
		}
		if item.backdrop != nil {
			a.artBackdrops[raw] = item.backdrop
		}
	}
}
