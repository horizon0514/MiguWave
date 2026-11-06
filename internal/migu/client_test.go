package migu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiguImage(t *testing.T) {
	cases := map[string]string{
		"":                               "",
		"//d.musicapp.migu.cn/a.jpg":     "https://d.musicapp.migu.cn/a.jpg",
		"/v5/cover.jpg":                  "https://d.musicapp.migu.cn/v5/cover.jpg",
		"https://example.com/cover.jpeg": "https://example.com/cover.jpeg",
	}
	for raw, want := range cases {
		if got := miguImage(raw); got != want {
			t.Errorf("miguImage(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestMiguSongTrackVIP(t *testing.T) {
	var song miguSong
	raw := `{
		"contentId": "600913000007163534",
		"copyrightId": "60054704101",
		"songName": "晴天",
		"duration": 269,
		"album": "叶惠美",
		"img1": "/v5/cover.jpg",
		"img2": "/v5/cover-400.jpg",
		"img3": "/v5/cover-800.jpg",
		"vip": 1,
		"playNumDesc": "1.2亿",
		"singerList": [{"name": "周杰伦"}]
	}`
	if err := json.Unmarshal([]byte(raw), &song); err != nil {
		t.Fatal(err)
	}
	track := miguSongTrack(song)
	if !track.VIP {
		t.Error("VIP flag was not parsed")
	}
	if track.Title != "晴天" || track.Artist != "周杰伦" {
		t.Errorf("track = %+v", track)
	}
	if track.DurationSec != 269 {
		t.Errorf("duration = %v", track.DurationSec)
	}
	if !strings.HasPrefix(track.Cover, "https://d.musicapp.migu.cn/") {
		t.Errorf("cover = %q", track.Cover)
	}
	if track.Cover != "https://d.musicapp.migu.cn/v5/cover-400.jpg" {
		t.Errorf("thumb cover = %q", track.Cover)
	}
	if got := miguSongCover(song); got != "https://d.musicapp.migu.cn/v5/cover-800.jpg" {
		t.Errorf("detail cover = %q", got)
	}
	if track.CoverHQ != "https://d.musicapp.migu.cn/v5/cover-800.jpg" {
		t.Errorf("high-res cover = %q", track.CoverHQ)
	}
	if !track.NeedsResolve() {
		t.Error("Migu track without URL should need resolve")
	}

	free := song
	free.VIP = "0"
	if miguSongTrack(free).VIP {
		t.Error("vip=0 should not be treated as a VIP track")
	}
}

func TestMiguDedupeTracks(t *testing.T) {
	tracks := []Track{
		{ID: "migu:1", Title: "A"},
		{ID: "migu:2", Title: "B"},
		{ID: "migu:1", Title: "A again"},
	}
	got := miguDedupeTracks(tracks)
	if len(got) != 2 || got[0].Title != "A" || got[1].Title != "B" {
		t.Fatalf("miguDedupeTracks = %+v", got)
	}
}

func TestSearchMiguPagination(t *testing.T) {
	old := miguSearchEndpoint
	defer func() { miguSearchEndpoint = old }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("pageNo")
		name := "第一页"
		if page == "2" {
			name = "第二页"
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"contentId":   "id-" + page,
			"copyrightId": "copyright-" + page,
			"songName":    name,
			"duration":    180,
			"singerList":  []map[string]string{{"name": "歌手"}},
		}})
	}))
	defer server.Close()
	miguSearchEndpoint = server.URL

	page1, err := searchMigu(context.Background(), "周杰伦", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	page2, err := searchMigu(context.Background(), "周杰伦", 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	merged := miguDedupeTracks(append(page1, page2...))
	if len(merged) != 2 {
		t.Fatalf("merged = %+v", merged)
	}
	if merged[0].Title != "第一页" || merged[1].Title != "第二页" {
		t.Fatalf("unexpected pagination order: %+v", merged)
	}
}

func TestResolveCoverHQ(t *testing.T) {
	oldEndpoint, oldClient := miguSearchEndpoint, miguClient
	defer func() {
		miguSearchEndpoint = oldEndpoint
		miguClient = oldClient
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"contentId": "other",
				"songName":  "晴天",
				"img3":      "/v5/wrong.jpg",
			},
			{
				"contentId": "600913000007163534",
				"songName":  "晴天",
				"img1":      "/v5/cover.jpg",
				"img3":      "/v5/cover-800.jpg",
			},
		})
	}))
	defer server.Close()
	miguSearchEndpoint = server.URL
	miguClient = server.Client()

	track := Track{ID: "migu:600913000007163534", Title: "晴天", Artist: "周杰伦"}
	if got := ResolveCoverHQ(context.Background(), track); got != "https://d.musicapp.migu.cn/v5/cover-800.jpg" {
		t.Fatalf("high-res cover = %q", got)
	}
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

func TestFetchMiguRank(t *testing.T) {
	old := miguRankEndpoint
	defer func() { miguRankEndpoint = old }()

	songData, _ := json.Marshal(map[string]any{
		"contentId":   "600913000007163534",
		"copyrightId": "60054704101",
		"songName":    "晴天",
		"duration":    269,
		"singerList":  []map[string]string{{"name": "周杰伦"}},
		"showTags":    []string{"vip"},
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "000000",
			"data": map[string]any{
				"columnId": "83176390",
				"title":    "国风热歌榜",
				"desc":     "热门国风",
				"contents": []map[string]any{{
					"resId":       "600913000007163534",
					"copyrightId": "60054704101",
					"songData":    string(songData),
				}},
			},
		})
	}))
	defer server.Close()
	miguRankEndpoint = server.URL

	rank, err := fetchMiguRank(context.Background(), "83176390", 100)
	if err != nil {
		t.Fatal(err)
	}
	if rank.Title != "国风热歌榜" || len(rank.Tracks) != 1 {
		t.Fatalf("rank = %+v", rank)
	}
	if !rank.Tracks[0].VIP {
		t.Error("songData VIP tag was not parsed")
	}
}

func TestResolveMiguURLSendsToken(t *testing.T) {
	old := miguListenEndpoint
	defer func() { miguListenEndpoint = old }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("pacmtoken"); got != "token-123" {
			t.Errorf("pacmtoken header = %q", got)
		}
		cookie, err := r.Cookie("pacmtoken")
		if err != nil || cookie.Value != "token-123" {
			t.Errorf("pacmtoken cookie = %v, %v", cookie, err)
		}
		query := r.URL.Query()
		if query.Get("contentId") != "600913000007163534" {
			t.Errorf("contentId = %q", query.Get("contentId"))
		}
		if query.Get("copyrightId") != "60054704101" {
			t.Errorf("copyrightId = %q", query.Get("copyrightId"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "000000",
			"data": map[string]any{"url": "https://example.com/song.mp3"},
		})
	}))
	defer server.Close()
	miguListenEndpoint = server.URL

	got, err := resolveMiguURL(context.Background(), "token-123", sampleMiguTrack())
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://example.com/song.mp3" {
		t.Fatalf("url = %q", got)
	}
}

func TestResolveMiguVIPRequired(t *testing.T) {
	old := miguListenEndpoint
	defer func() { miguListenEndpoint = old }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "000000",
			"data": map[string]any{
				"cannotCode":      "440013",
				"auditionsLength": 60,
				"dialogInfo":      map[string]string{"text": "该歌曲需开通白金会员"},
			},
		})
	}))
	defer server.Close()
	miguListenEndpoint = server.URL

	_, err := resolveMiguURL(context.Background(), "", sampleMiguTracks()[1])
	if !errors.Is(err, errMiguVIPRequired) {
		t.Fatalf("err = %v, want errMiguVIPRequired", err)
	}
}

func TestNormalizeMiguToken(t *testing.T) {
	cases := map[string]string{
		"":                                  "",
		"abc123":                            "abc123",
		`"abc123"`:                          "abc123",
		"foo=1; pacmtoken=xyz%2B123; bar=2": "xyz+123",
	}
	for raw, want := range cases {
		if got := normalizeMiguToken(raw); got != want {
			t.Errorf("normalizeMiguToken(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestMiguErrorEnvelope(t *testing.T) {
	old := miguSearchEndpoint
	defer func() { miguSearchEndpoint = old }()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"code":"100001","info":"参数错误"}`)
	}))
	defer server.Close()
	miguSearchEndpoint = server.URL
	if _, err := searchMigu(context.Background(), "x", 1, 20); err == nil ||
		!strings.Contains(err.Error(), "参数错误") {
		t.Fatalf("err = %v", err)
	}
}
