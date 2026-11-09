package migu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiguRadioKindFromTrack(t *testing.T) {
	cases := map[string]string{
		"srfm":    miguRadioKindPrivate,
		"dt_sydt": miguRadioKindAddict,
		"dt_tjbt": miguRadioKindDiscover,
		"sxdt":    "",
	}
	for track, want := range cases {
		if got := miguPersonalRadioKind(track); got != want {
			t.Errorf("miguPersonalRadioKind(%q) = %q, want %q", track, got, want)
		}
	}

	var contents []json.RawMessage
	if err := json.Unmarshal([]byte(`[
		{"view":"ZJ-MusicRadio-Item","resId":"r-1","txt":"私人FM","track":"srfm"},
		{"view":"ZJ-MusicRadio-Item","resId":"r-2","txt":"上瘾电台","track":"dt_sydt"},
		{"view":"ZJ-MusicRadio-Item","resId":"r-3","txt":"听见不同","track":"dt_tjbt"}
	]`), &contents); err != nil {
		t.Fatal(err)
	}
	items := miguRadiosFromItems(contents)
	if len(items) != 3 {
		t.Fatalf("items = %d, want 3", len(items))
	}
	for i, want := range []string{miguRadioKindPrivate, miguRadioKindAddict, miguRadioKindDiscover} {
		if items[i].Kind != want {
			t.Errorf("item %d kind = %q, want %q", i, items[i].Kind, want)
		}
		if items[i].Track == "" {
			t.Errorf("item %d track is empty", i)
		}
	}
}

func TestFetchMiguPersonalRadioPrivateFM(t *testing.T) {
	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("deviceId"); got != miguRadioDeviceID {
			t.Errorf("deviceId = %q", got)
		}
		if got := r.Header.Get("pacmtoken"); got != "test-token" {
			t.Errorf("pacmtoken = %q", got)
		}
		if got := r.Header.Get("activityId"); got != "MUSIC-WWW" {
			t.Errorf("activityId = %q", got)
		}
		if got := r.Header.Get("appId"); got != "h5" {
			t.Errorf("appId = %q", got)
		}
		if got := r.URL.Query().Get("scene"); got != "PRIVATE_FM" {
			t.Errorf("scene = %q", got)
		}
		action := r.URL.Query().Get("action")
		actions = append(actions, action)
		songs := []map[string]any{}
		switch action {
		case "1":
			songs = testSongs("A", "B", "C", "D", "E")
		case "2":
			songs = testSongs("E", "F", "G", "H", "I")
		default:
			songs = testSongs("I", "J", "K", "L", "M")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "000000",
			"info": "操作成功",
			"data": map[string]any{
				"isRecommend":  "1",
				"songItemList": songs,
			},
		})
	}))
	defer server.Close()

	oldEndpoint := miguPrivateFMEndpoint
	miguPrivateFMEndpoint = server.URL
	t.Cleanup(func() { miguPrivateFMEndpoint = oldEndpoint })

	radio := miguRadio{ID: "r-1", Title: "私人 FM", Kind: miguRadioKindPrivate}
	detail, err := fetchMiguPersonalRadio(context.Background(), "test-token", radio, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Tracks) != 8 {
		t.Fatalf("tracks = %d, want 8", len(detail.Tracks))
	}
	if detail.Total != 8 {
		t.Errorf("total = %d, want 8", detail.Total)
	}
	for i, want := range []string{"A", "B", "C", "D", "E", "F", "G", "H"} {
		if detail.Tracks[i].Title != want {
			t.Errorf("track %d = %q, want %q", i, detail.Tracks[i].Title, want)
		}
	}
	if strings.Join(actions, ",") != "1,2" {
		t.Errorf("actions = %v, want [1 2]", actions)
	}
}

func TestFetchMiguPersonalRadioRecommend(t *testing.T) {
	var radioTypes []string
	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("recommendstatus"); got != "1" {
			t.Errorf("recommendstatus = %q", got)
		}
		if got := r.Header.Get("deviceId"); got != miguRadioDeviceID {
			t.Errorf("deviceId = %q", got)
		}
		if got := r.Header.Get("pacmtoken"); got != "test-token" {
			t.Errorf("pacmtoken = %q", got)
		}
		radioTypes = append(radioTypes, r.URL.Query().Get("type"))
		action := r.URL.Query().Get("actionId")
		actions = append(actions, action)
		songs := []map[string]any{}
		if action == "1" {
			songs = testSongs("A", "B", "C", "D", "E")
		} else {
			songs = testSongs("D", "E", "F", "G", "H")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "000000",
			"info": "操作成功",
			"data": songs,
		})
	}))
	defer server.Close()

	oldEndpoint := miguRecommendRadioEndpoint
	miguRecommendRadioEndpoint = server.URL
	t.Cleanup(func() { miguRecommendRadioEndpoint = oldEndpoint })

	radio := miguRadio{ID: "r-2", Title: "上瘾电台", Kind: miguRadioKindAddict}
	detail, err := fetchMiguPersonalRadio(context.Background(), "test-token", radio, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Tracks) != 7 {
		t.Fatalf("tracks = %d, want 7", len(detail.Tracks))
	}
	if detail.Tracks[6].Title != "G" {
		t.Errorf("last track = %q, want G", detail.Tracks[6].Title)
	}
	if strings.Join(radioTypes, ",") != "2,2" {
		t.Errorf("types = %v, want [2 2]", radioTypes)
	}
	if strings.Join(actions, ",") != "1,2" {
		t.Errorf("actions = %v, want [1 2]", actions)
	}
}

func TestFetchMiguPersonalRadioRequiresToken(t *testing.T) {
	radio := miguRadio{ID: "r-1", Title: "私人 FM", Kind: miguRadioKindPrivate}
	if _, err := fetchMiguPersonalRadio(context.Background(), "", radio, 10); err == nil {
		t.Fatal("expected missing token to fail")
	}
}

func testSongs(names ...string) []map[string]any {
	songs := make([]map[string]any, 0, len(names))
	for _, name := range names {
		songs = append(songs, map[string]any{
			"contentId":  "c-" + name,
			"songName":   name,
			"singerList": []map[string]string{{"name": "歌手" + name}},
			"duration":   200,
			"img2":       "/cover/" + name + ".jpg",
		})
	}
	return songs
}
