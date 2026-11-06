package app

import (
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestSongPageRendersLyrics(t *testing.T) {
	track := sampleMiguTracks()[1]
	a := testApp()
	a.page = pageSong
	a.songTrack = track
	a.songQueue = sampleMiguTracks()
	a.songIndex = 1
	a.songDetail = miguSongDetail{
		Track:       track,
		Album:       track.Album,
		Formats:     []string{"PQ", "HQ"},
		SongID:      "123456",
		LyricTitle:  track.Title,
		LyricArtist: track.Artist,
		LyricBy:     "测试",
		Lyrics: []lyricLine{
			{At: 29 * time.Second, Text: "故事的小黄花", Timed: true},
			{At: 32 * time.Second, Text: "从出生那年就飘着", Timed: true},
		},
	}
	a.songCache[track.ID] = a.songDetail

	tt := ui.NewTester(a.view, 1240, 820)
	for _, want := range []string{"歌词", "律动", "待播清单", track.Title, track.Artist, "故事的小黄花", "从出生那年就飘着"} {
		if !tt.HasText(want) {
			t.Fatalf("song page does not show %q: %q", want, tt.Texts())
		}
	}
	for _, unwanted := range []string{"歌曲信息", "PQ · HQ", "内容 ID", "歌曲 ID"} {
		if tt.HasText(unwanted) {
			t.Fatalf("song page still shows technical field %q: %q", unwanted, tt.Texts())
		}
	}
	for _, hidden := range []string{"浏览", "资料库", "我的收藏"} {
		if tt.HasText(hidden) {
			t.Fatalf("song page still shows app navigation %q: %q", hidden, tt.Texts())
		}
	}
}

func TestSongStageRendersCurrentAndNext(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("AppData", home)
	} else {
		t.Setenv("HOME", home)
	}
	track := sampleMiguTracks()[1]
	a := testApp()
	a.page = pageSong
	a.songPane = songPaneStage
	a.songTrack = track
	a.songQueue = sampleMiguTracks()
	a.songIndex = 1
	a.songDetail = miguSongDetail{
		Track: track,
		Album: track.Album,
		Lyrics: []lyricLine{
			{At: 29 * time.Second, Text: "微凉的晨露 沾湿黑礼服", Timed: true},
			{At: 34 * time.Second, Text: "石板路有雾 父在低诉", Timed: true},
			{At: 40 * time.Second, Text: "无奈的觉悟 只能更残酷", Timed: true},
		},
	}
	a.songCache[track.ID] = a.songDetail

	tt := ui.NewTester(a.view, 1240, 820)
	for _, want := range []string{
		"微凉的晨露 沾湿黑礼服", "石板路有雾 父在低诉",
	} {
		if !tt.HasText(want) {
			t.Fatalf("stage does not show %q: %q", want, tt.Texts())
		}
	}
	if tt.HasText("无奈的觉悟 只能更残酷") {
		t.Fatalf("stage shows a third line: %q", tt.Texts())
	}
}

func TestSongPageFollowsPlayerTrack(t *testing.T) {
	a := testApp()
	tracks := sampleMiguTracks()
	a.page = pageSong
	a.songTrack = tracks[0]
	a.songDetail = miguSongDetail{Track: tracks[0], Cover: tracks[0].Cover}
	a.songQueue = append([]Track(nil), tracks...)
	a.songIndex = 0
	a.pl.queue = append([]Track(nil), tracks...)
	a.pl.at = 1
	a.songCache[tracks[1].ID] = miguSongDetail{
		Track: tracks[1],
		Cover: "https://d.musicapp.migu.cn/v5/cover-800.jpg",
	}

	a.followPlayerTrack(tracks[1])

	if a.songTrack.ID != tracks[1].ID || a.songDetail.Track.ID != tracks[1].ID {
		t.Fatalf("song page did not follow player: track=%+v detail=%+v", a.songTrack, a.songDetail)
	}
	if a.songIndex != 1 || len(a.songQueue) != len(tracks) {
		t.Fatalf("queue = %d/%d", a.songIndex, len(a.songQueue))
	}
	if a.songDetail.Cover != "https://d.musicapp.migu.cn/v5/cover-800.jpg" {
		t.Fatalf("cached detail was not applied: %+v", a.songDetail)
	}
}
