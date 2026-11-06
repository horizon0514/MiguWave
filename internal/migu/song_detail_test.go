package migu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchMiguSongDetail(t *testing.T) {
	old := miguListenEndpoint
	defer func() { miguListenEndpoint = old }()
	oldClient, oldAPIClient := miguClient, apiClient
	defer func() {
		miguClient = oldClient
		apiClient = oldAPIClient
	}()

	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/lyrics" {
			_, _ = w.Write([]byte("[ti:晴天]\n[00:29.45]故事的小黄花\n[00:32.50]从出生那年就飘着\n"))
			return
		}
		if got := r.Header.Get("pacmtoken"); got != "token-123" {
			t.Errorf("pacmtoken header = %q", got)
		}
		query := r.URL.Query()
		for key, want := range map[string]string{
			"contentId":    "600913000007163534",
			"copyrightId":  "60054704101",
			"resourceType": "2",
			"toneFlag":     "PQ",
		} {
			if got := query.Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "000000",
			"data": map[string]any{
				"lrcUrl": server.URL + "/lyrics",
				"song": map[string]any{
					"contentId":   "600913000007163534",
					"copyrightId": "60054704101",
					"songName":    "晴天",
					"duration":    269,
					"album":       "叶惠美",
					"img1":        "/v5/cover.jpg",
					"img3":        "/v5/cover-800.jpg",
					"playNumDesc": "1.2亿",
					"showTags":    []string{"vip"},
					"songId":      "123456",
					"audioFormats": []map[string]string{
						{"formatType": "PQ"},
						{"formatType": "HQ"},
						{"formatType": "PQ"},
					},
					"singerList": []map[string]string{{"name": "周杰伦"}},
				},
			},
		})
	}))
	defer server.Close()
	miguListenEndpoint = server.URL
	miguClient = server.Client()
	apiClient = server.Client()

	detail, err := fetchMiguSongDetail(context.Background(), "token-123", sampleMiguTrack())
	if err != nil {
		t.Fatal(err)
	}
	if detail.Track.Title != "晴天" || detail.Track.Artist != "周杰伦" || !detail.Track.VIP {
		t.Fatalf("track = %+v", detail.Track)
	}
	if detail.SongID != "123456" || len(detail.Formats) != 2 {
		t.Fatalf("detail = %+v", detail)
	}
	if detail.Cover != "https://d.musicapp.migu.cn/v5/cover-800.jpg" {
		t.Fatalf("cover = %q", detail.Cover)
	}
	if detail.LyricTitle != "晴天" || len(detail.Lyrics) != 2 {
		t.Fatalf("lyrics = %+v, meta = %+v", detail.Lyrics, detail)
	}
	if !strings.HasPrefix(detail.LRCURL, server.URL) {
		t.Fatalf("lrc URL = %q", detail.LRCURL)
	}
}
