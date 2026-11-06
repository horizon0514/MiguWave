package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func liveContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	if os.Getenv("MIGU_LIVE_TEST") == "" {
		t.Skip("set MIGU_LIVE_TEST=1 to call the live Migu service")
	}
	return context.WithTimeout(context.Background(), 30*time.Second)
}

func TestLiveMiguReadOnly(t *testing.T) {
	ctx, cancel := liveContext(t)
	defer cancel()

	home, err := fetchMiguHome(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(home.NewSongs) == 0 || len(home.Ranks) == 0 || len(home.Playlists) == 0 {
		t.Fatalf("home is missing songs, ranks, or playlists: %+v", home)
	}
	hq := resolveMiguCoverHQ(ctx, home.NewSongs[0])
	if hq == "" || hq == home.NewSongs[0].Cover {
		t.Fatalf("no high-res cover for %q: %q", home.NewSongs[0].Title, hq)
	}

	tracks, err := searchMigu(ctx, "周杰伦", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) == 0 {
		t.Fatal("search returned no tracks")
	}

	rank, err := fetchMiguRank(ctx, miguRankCatalog[0].ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(rank.Tracks) == 0 {
		t.Fatal("rank returned no tracks")
	}

	var free *Track
	for i := range tracks {
		if !tracks[i].VIP {
			free = &tracks[i]
			break
		}
	}
	if free == nil {
		t.Skip("no non-VIP search result was available for playback resolution")
	}
	streamURL, err := resolveMiguURL(ctx, "", *free)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(streamURL, "https://") {
		t.Fatalf("stream URL = %q", streamURL)
	}
}

func TestLiveMiguQRStart(t *testing.T) {
	ctx, cancel := liveContext(t)
	defer cancel()

	session, err := newMiguAuthSession()
	if err != nil {
		t.Fatal(err)
	}
	code, err := session.startQR(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(code.Image) == 0 || strings.TrimSpace(code.SessionID) == "" {
		t.Fatalf("incomplete QR response: image=%d session=%q", len(code.Image), code.SessionID)
	}
	poll, err := session.pollQR(ctx, code.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if poll.State != miguQRWaiting {
		t.Fatalf("fresh QR state = %v, want waiting", poll.State)
	}
}

func TestLiveMiguBrowse(t *testing.T) {
	ctx, cancel := liveContext(t)
	defer cancel()

	sections, err := fetchMiguPlaylistSquare(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) == 0 || len(sections[0].Items) == 0 {
		t.Fatalf("playlist square is empty: %+v", sections)
	}
	detail, err := fetchMiguPlaylistDetail(ctx, sections[0].Items[0].ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Tracks) == 0 {
		t.Fatalf("playlist %q returned no tracks", detail.Playlist.Title)
	}

	singers, err := fetchMiguSingers(ctx, miguSingerTabs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(singers) == 0 {
		t.Fatal("singer list is empty")
	}
	singerDetail, err := fetchMiguSingerDetail(ctx, singers[0])
	if err != nil {
		t.Fatal(err)
	}
	if singerDetail.Singer.Name == "" || len(singerDetail.Tracks) == 0 {
		t.Fatalf("singer detail is incomplete: %+v", singerDetail)
	}

	groups, err := fetchMiguRadioGroups(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var scene *miguRadio
	for i := range groups {
		for j := range groups[i].Items {
			if groups[i].Items[j].Kind == "scene" {
				scene = &groups[i].Items[j]
				break
			}
		}
		if scene != nil {
			break
		}
	}
	if scene == nil {
		t.Fatalf("no scene radio was returned: %+v", groups)
	}
	radioDetail, err := fetchMiguRadioDetail(ctx, *scene, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(radioDetail.Tracks) == 0 {
		t.Fatalf("scene radio %q returned no tracks", scene.Title)
	}
}
