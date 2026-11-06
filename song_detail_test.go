package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestParseLRC(t *testing.T) {
	raw := "\ufeff[ti:晴天]\r\n[ar:周杰伦]\r\n[al:叶惠美]\r\n[by:测试]\r\n[offset:500]\r\n" +
		"[00:01.00]故事的小黄花\r\n[00:28.50][00:30.00]从出生那年就飘着\r\n"
	doc := parseLRC(raw)
	if doc.Plain {
		t.Fatal("timed LRC was parsed as plain text")
	}
	if doc.Title != "晴天" || doc.Artist != "周杰伦" || doc.Album != "叶惠美" || doc.By != "测试" {
		t.Fatalf("metadata = %+v", doc)
	}
	if len(doc.Lines) != 3 {
		t.Fatalf("lines = %+v", doc.Lines)
	}
	if doc.Lines[0].At != 1500*time.Millisecond || doc.Lines[0].Text != "故事的小黄花" {
		t.Fatalf("first line = %+v", doc.Lines[0])
	}
	if doc.Lines[1].At != 29*time.Second || doc.Lines[2].At != 30500*time.Millisecond {
		t.Fatalf("multiple timestamps = %+v", doc.Lines)
	}
}

func TestParseLRCPlainText(t *testing.T) {
	doc := parseLRC("[ti:测试]\n第一行\n\n第二行 &amp; 结尾\n")
	if !doc.Plain || doc.Title != "测试" {
		t.Fatalf("doc = %+v", doc)
	}
	if len(doc.Lines) != 2 || doc.Lines[0].Text != "第一行" || doc.Lines[1].Text != "第二行 & 结尾" {
		t.Fatalf("lines = %+v", doc.Lines)
	}
}

func TestParseMRC(t *testing.T) {
	raw := "[ti:晴天]\r\n[ar:周杰伦]\r\n" +
		"[0,0]作(0,0)词(0,0)：(0,0)周(0,0)杰(0,0)伦(0,0)\r\n" +
		"[27566,2095]故(27566,299)事(27865,199)的(28064,250)小(28314,549)黄(28863,449)花(29312,349)\r\n" +
		"[30459,2494]从(30459,349)出(30808,349)生(31157,499)那(31656,300)年(31956,249)就(32205,200)飘(32405,249)着(32654,299)\r\n"
	doc := parseMRC(raw)
	if doc.Plain || doc.Title != "晴天" || doc.Artist != "周杰伦" {
		t.Fatalf("doc = %+v", doc)
	}
	if len(doc.Lines) != 3 {
		t.Fatalf("lines = %+v", doc.Lines)
	}
	line := doc.Lines[1]
	if line.At != 27566*time.Millisecond || line.Text != "故事的小黄花" {
		t.Fatalf("line = %+v", line)
	}
	if len(line.Words) != 6 {
		t.Fatalf("words = %+v", line.Words)
	}
	if line.Words[0].At != 27566*time.Millisecond || line.Words[0].Dur != 299*time.Millisecond ||
		line.Words[0].Text != "故" {
		t.Fatalf("first word = %+v", line.Words[0])
	}
}

func TestInterpolateLyricWords(t *testing.T) {
	line := lyricLine{At: 10 * time.Second, Text: "你好 world!", Timed: true}
	words := lyricWordsFor(line, 14*time.Second)
	if len(words) < 4 {
		t.Fatalf("words = %+v", words)
	}
	if words[0].Text != "你" || words[0].At != 10*time.Second {
		t.Fatalf("first word = %+v", words[0])
	}
	if last := words[len(words)-1]; last.At >= 14*time.Second {
		t.Fatalf("last word runs past line end: %+v", last)
	}
	var joined strings.Builder
	for _, word := range words {
		joined.WriteString(word.Text)
	}
	if got := joined.String(); got != "你好 world!" {
		t.Fatalf("joined = %q", got)
	}
}

func TestStageLyricPair(t *testing.T) {
	lines := []lyricLine{
		{At: 1 * time.Second, Text: "第一句", Timed: true},
		{At: 2 * time.Second, Text: "", Timed: true},
		{At: 3 * time.Second, Text: "第二句", Timed: true},
		{At: 4 * time.Second, Text: "第三句", Timed: true},
	}
	current, next := stageLyricPair(lines, 2)
	if current != 2 || next != 3 {
		t.Fatalf("pair = %d/%d", current, next)
	}
	current, next = stageLyricPair(lines, -1)
	if current != 0 || next != 2 {
		t.Fatalf("leading pair = %d/%d", current, next)
	}
}

func TestStripLyricHeader(t *testing.T) {
	title := "一半（《折月亮》影视剧片尾曲）"
	lines := []lyricLine{
		{Text: title + " - 段奥娟"},
		{Text: "作词：四季逗/潘超"},
		{Text: "作曲：宋天祥/李喆"},
		{Text: "制作人：宋天祥"},
		{Text: "总策划：王锋"},
		{Text: "「 未经著作权人许可不得翻唱翻录或使用」"},
		{At: 12 * time.Second, Text: "时间上的灰尘", Timed: true},
		{At: 18 * time.Second, Text: "每一步都很轻缓", Timed: true},
	}
	got := stripLyricHeader(lines, title, "段奥娟")
	if len(got) != 2 || got[0].Text != "时间上的灰尘" || got[1].Text != "每一步都很轻缓" {
		t.Fatalf("stripped = %+v", got)
	}

	lyrics := []lyricLine{
		{At: 29 * time.Second, Text: "故事的小黄花", Timed: true},
		{At: 32 * time.Second, Text: "从出生那年就飘着", Timed: true},
	}
	got = stripLyricHeader(lyrics, "晴天", "周杰伦")
	if len(got) != 2 || got[0].Text != "故事的小黄花" {
		t.Fatalf("plain lyrics changed = %+v", got)
	}
}

func TestStripBilingualLyricCredits(t *testing.T) {
	lines := []lyricLine{
		{Text: "月下签 - 蒲熠星"},
		{Text: "作词：蒲熠星/郑眠眠"},
		{Text: "音乐监制Music Executive Director：朴桑松@壹橙Music"},
		{Text: "和声Background Vocals：陈小同@壹橙Music"},
		{Text: "和音：杜婧荧"},
		{Text: "混音/母带后期工程师Mixing/Mastering Engineer：钟焕楷@壹橙Music"},
		{Text: "封面设计Cover Designer：敬尧"},
		{At: 17 * time.Second, Text: "推开山门月光落在脚边", Timed: true},
		{At: 21 * time.Second, Text: "她折起签文一篇折起谎言", Timed: true},
		{At: 200 * time.Second, Text: "特别鸣谢Special Thanks：蒲熠星工作室", Timed: true},
		{At: 202 * time.Second, Text: "OP(Original Publisher)：咪咕音乐", Timed: true},
	}
	got := stripLyricHeader(lines, "月下签", "蒲熠星")
	if len(got) != 2 || got[0].Text != "推开山门月光落在脚边" {
		t.Fatalf("stripped = %+v", got)
	}
}

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
