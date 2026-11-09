package app

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestSnapshots(t *testing.T) {
	if os.Getenv("MIGUWAVE_SNAPSHOTS") == "" {
		t.Skip("set MIGUWAVE_SNAPSHOTS=1 to write snapshots")
	}
	dir := "/tmp/miguwave-snapshots"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	saveView := func(name string, view func(*ui.Context), width, height int, dark bool) {
		tt := ui.NewTester(view, width, height)
		tt.SetScale(2)
		tt.SetDark(dark)
		tt.Frame()
		tt.Advance(600 * time.Millisecond)
		tt.Frame()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}
	save := func(name string, a *app, dark bool) {
		saveView(name, a.view, 1240, 820, dark)
	}

	tracks := sampleMiguTracks()

	home := testApp()
	home.home.NewSongs = tracks
	home.home.Playlists = samplePlaylists()
	home.radioGroups = sampleRadioGroups()
	home.home.Ranks = []miguRankSummary{
		{ID: "83176390", Title: "国风热歌榜", Subtitle: "每周更新", Tracks: tracks},
		{ID: "75959118", Title: "音乐风向榜", Subtitle: "每小时更新", Tracks: tracks},
		{ID: "27553319", Title: "尖叫新歌榜", Subtitle: "实时新歌", Tracks: tracks},
	}
	seedSnapshotArt(t, home)
	save("home", home, false)
	save("home-dark", home, true)

	queue := testApp()
	queue.home.NewSongs = tracks
	queue.pl.queue = append([]Track(nil), tracks...)
	queue.pl.at = 0
	queue.pl.current = tracks[0]
	queue.pl.hasCurrent = true
	queue.pl.duration = tracks[0].Duration()
	queue.playerQueueOpen = true
	seedSnapshotArt(t, queue)
	save("player-queue", queue, false)
	save("player-queue-dark", queue, true)

	rank := testApp()
	rank.page = pageRank
	rank.rank = sampleRank()
	rank.rankID = rank.rank.ID
	save("rank", rank, false)
	save("rank-dark", rank, true)

	favorites := testApp()
	favorites.page = pageFavorites
	favorites.favorites = tracks
	save("favorites", favorites, false)

	search := testApp()
	search.page = pageSearch
	search.searchTerm = "周杰伦"
	search.searchInput = "周杰伦"
	search.searchTracks = tracks
	search.searchHasMore = true
	save("search", search, false)
	save("search-dark", search, true)

	song := testApp()
	song.page = pageSong
	song.songTrack = tracks[1]
	song.songQueue = tracks
	song.songIndex = 1
	song.songDetail = miguSongDetail{
		Track:       tracks[1],
		Album:       tracks[1].Album,
		Formats:     []string{"PQ", "HQ", "SQ"},
		SongID:      "600902000006889366",
		LyricTitle:  tracks[1].Title,
		LyricArtist: tracks[1].Artist,
		LyricBy:     "Migu Music",
		Lyrics: []lyricLine{
			{At: 29 * time.Second, Text: "微凉的晨露 沾湿黑礼服", Timed: true},
			{At: 34 * time.Second, Text: "石板路有雾 父在低诉", Timed: true},
			{At: 40 * time.Second, Text: "无奈的觉悟 只能更残酷", Timed: true},
			{At: 46 * time.Second, Text: "一切都为了通往圣堂的路", Timed: true},
			{At: 52 * time.Second, Text: "吹不散的雾 隐没了意图", Timed: true},
			{At: 58 * time.Second, Text: "谁轻柔踱步 停住", Timed: true},
		},
	}
	song.songCache[tracks[1].ID] = song.songDetail
	save("song-detail", song, false)
	save("song-detail-dark", song, true)

	songError := testApp()
	songError.page = pageSong
	songError.songTrack = tracks[1]
	songError.songQueue = tracks
	songError.songIndex = 1
	songError.songDetail = song.songDetail
	songError.songCache[tracks[1].ID] = song.songDetail
	songError.pl.queue = append([]Track(nil), tracks...)
	songError.pl.at = 1
	songError.pl.current = tracks[1]
	songError.pl.hasCurrent = true
	songError.pl.err = "版权方要求，该歌曲需开通白金会员后播放"
	songError.pl.errKind = playbackFailureVIP
	seedSnapshotArt(t, songError)
	save("song-error", songError, false)

	stage := testApp()
	stage.page = pageSong
	stage.songPane = songPaneStage
	stage.songTrack = tracks[1]
	stage.songQueue = tracks
	stage.songIndex = 1
	stage.songDetail = song.songDetail
	stage.songCache[tracks[1].ID] = song.songDetail
	seedSnapshotArt(t, stage)
	save("song-stage", stage, false)

	account := testApp()
	account.accountOpen = true
	account.authPhase = miguAuthBrowserWaiting
	save("account", account, false)

	accountInvalid := testApp()
	accountInvalid.accountOpen = true
	accountInvalid.authMode = authModeBrowser
	accountInvalid.authPhase = miguAuthInvalid
	accountInvalid.authNotice = "登录已过期。正在自动重新登录，完成后会继续播放。"
	save("account-invalid", accountInvalid, false)

	accountQR := testApp()
	accountQR.accountOpen = true
	accountQR.authMode = authModeQR
	accountQR.authPhase = miguAuthQRWaiting
	accountQR.authQR = snapshotQRCode(t)
	save("account-qr", accountQR, false)

	accountScanned := testApp()
	accountScanned.accountOpen = true
	accountScanned.authMode = authModeQR
	accountScanned.authPhase = miguAuthQRScanned
	accountScanned.authQR = snapshotQRCode(t)
	save("account-qr-scanned", accountScanned, false)

	accountToken := testApp()
	accountToken.accountOpen = true
	accountToken.authMode = authModeToken
	save("account-token", accountToken, false)

	accountConnected := testApp()
	accountConnected.accountOpen = true
	accountConnected.miguToken = "snapshot-token"
	accountConnected.authPhase = miguAuthConnected
	accountConnected.authNotice = "已连接，账号有权限的曲目会使用你的咪咕登录状态。"
	save("account-connected", accountConnected, false)

	playlists := testApp()
	playlists.page = pagePlaylists
	playlists.playlistSections = []miguPlaylistSection{
		{Title: "编辑推荐", Items: samplePlaylists()},
		{Title: "歌单百宝箱", Items: samplePlaylists()},
	}
	seedSnapshotArt(t, playlists)
	save("playlists", playlists, false)
	save("playlists-dark", playlists, true)

	playlistDetail := testApp()
	playlistDetail.page = pagePlaylistDetail
	playlistDetail.playlistFrom = pagePlaylists
	playlistDetail.playlistID = samplePlaylists()[0].ID
	playlistDetail.playlistDetail = miguPlaylistDetail{
		Playlist: samplePlaylists()[0],
		Tracks:   tracks,
		Page:     1,
		Total:    247,
		HasMore:  true,
	}
	seedSnapshotArt(t, playlistDetail)
	save("playlist-detail", playlistDetail, false)

	savedPlaylists := testApp()
	savedPlaylists.page = pageSavedPlaylists
	savedPlaylists.savedPlaylists = samplePlaylists()
	seedSnapshotArt(t, savedPlaylists)
	save("saved-playlists", savedPlaylists, false)

	singers := testApp()
	singers.page = pageSingers
	singers.singerTab = miguSingerTabs[0].ID
	singers.singers = sampleSingers()
	seedSnapshotArt(t, singers)
	save("singers", singers, false)
	save("singers-dark", singers, true)

	singerDetail := testApp()
	singerDetail.page = pageSingerDetail
	singerDetail.singerDetail = miguSingerDetail{
		Singer:  sampleSingers()[0],
		Bio:     "华语流行歌手，代表作横跨二十余年，旋律与叙事都极具辨识度。",
		Tracks:  tracks,
		Similar: sampleSingers()[1:],
		Page:    1,
		HasMore: true,
	}
	seedSnapshotArt(t, singerDetail)
	save("singer-detail", singerDetail, false)

	radios := testApp()
	radios.page = pageRadios
	radios.radioGroups = sampleRadioGroups()
	seedSnapshotArt(t, radios)
	save("radios", radios, false)
	save("radios-dark", radios, true)

	radioDetail := testApp()
	radioDetail.page = pageRadioDetail
	radioDetail.radioFrom = pageRadios
	radioDetail.radioDetail = miguRadioDetail{
		Radio:  sampleRadioGroups()[1].Items[0],
		Tracks: tracks,
		Total:  len(tracks),
	}
	seedSnapshotArt(t, radioDetail)
	save("radio-detail", radioDetail, false)

	localPlaylists := testApp()
	localPlaylists.page = pageLocalPlaylists
	localPlaylists.localPlaylists = sampleLocalPlaylists()
	seedSnapshotArt(t, localPlaylists)
	save("local-playlists", localPlaylists, false)
	save("local-playlists-dark", localPlaylists, true)

	localPlaylistDetail := testApp()
	localPlaylistDetail.page = pageLocalPlaylistDetail
	localPlaylistDetail.localPlaylistID = sampleLocalPlaylists()[0].ID
	localPlaylistDetail.localPlaylists = sampleLocalPlaylists()
	seedSnapshotArt(t, localPlaylistDetail)
	save("local-playlist-detail", localPlaylistDetail, false)

	localPlaylistEditor := testApp()
	localPlaylistEditor.localPlaylistEditorOpen = true
	localPlaylistEditor.localPlaylistEditorName = "深夜循环"
	localPlaylistEditor.localPlaylistEditorTrack = tracks[0]
	localPlaylistEditor.localPlaylistEditorHasTrack = true
	save("local-playlist-editor", localPlaylistEditor, false)

	localPlaylistDelete := testApp()
	localPlaylistDelete.localPlaylistDeleteOpen = true
	localPlaylistDelete.localPlaylistDeleteID = sampleLocalPlaylists()[0].ID
	localPlaylistDelete.localPlaylists = sampleLocalPlaylists()
	save("local-playlist-delete", localPlaylistDelete, false)

	settings := testApp()
	settings.page = pageSettings
	settings.launchAtLogin = true
	save("settings", settings, false)
	save("settings-dark", settings, true)

	menuBar := testApp()
	menuBarTrack := tracks[0]
	menuBarTrack.Cover = "snap://s-1"
	menuBarTrack.CoverHQ = "snap://s-1"
	menuBar.menuDetailID = menuBarTrack.ID
	menuBar.menuDetail = miguSongDetail{
		Track: menuBarTrack,
		Cover: menuBarTrack.Cover,
		Lyrics: []lyricLine{
			{At: 0, Text: "故事的小黄花", Timed: true},
			{At: 8 * time.Second, Text: "从出生那年就飘着", Timed: true},
			{At: 16 * time.Second, Text: "童年的荡秋千", Timed: true},
		},
	}
	menuBar.pl.current = menuBarTrack
	menuBar.pl.hasCurrent = true
	menuBar.pl.playing = true
	menuBar.pl.duration = menuBarTrack.Duration()
	seedSnapshotArt(t, menuBar)
	saveView("menu-bar", menuBar.menuBarView, 420, 536, false)
	saveView("menu-bar-dark", menuBar.menuBarView, 420, 536, true)

	menuBarEmpty := testApp()
	saveView("menu-bar-empty", menuBarEmpty.menuBarView, 420, 536, true)
}

func samplePlaylists() []miguPlaylist {
	return []miguPlaylist{
		{
			ID: "pl-1", Title: "华语热歌精选", Owner: "咪咕音乐", PlayCount: "63.9万",
			Cover: "snap://pl-1", CoverHQ: "snap://pl-1", TrackCount: 247,
			Description: "把最近循环的华语好歌收在一起，通勤和深夜都适合。",
			Tags:        []string{"华语", "流行", "精选"},
		},
		{
			ID: "pl-2", Title: "温柔之声 · 治愈人心", Owner: "咪咕歌单", PlayCount: "120.4万",
			Cover: "snap://pl-2", CoverHQ: "snap://pl-2", TrackCount: 88,
		},
		{
			ID: "pl-3", Title: "经典粤语：听得见的学生时代", Owner: "咪咕歌单", PlayCount: "58.0万",
			Cover: "snap://pl-3", CoverHQ: "snap://pl-3", TrackCount: 120,
		},
		{
			ID: "pl-4", Title: "让热搜沸腾的神级 Live", Owner: "LIVE 精选", PlayCount: "42.1万",
			Cover: "snap://pl-4", CoverHQ: "snap://pl-4", TrackCount: 62,
		},
		{
			ID: "pl-5", Title: "岁月之声纪念册", Owner: "咪咕音乐", PlayCount: "100.4万",
			Cover: "snap://pl-5", CoverHQ: "snap://pl-5", TrackCount: 200,
		},
	}
}

func sampleLocalPlaylists() []localPlaylist {
	tracks := sampleMiguTracks()
	tracks[0].Cover = "snap://pl-1"
	tracks[0].CoverHQ = "snap://pl-1"
	tracks[1].Cover = "snap://pl-2"
	tracks[1].CoverHQ = "snap://pl-2"
	second := sampleMiguTracks()
	for i := range second {
		second[i].Cover = fmt.Sprintf("snap://pl-%d", i+3)
		second[i].CoverHQ = second[i].Cover
	}
	return []localPlaylist{
		{ID: "local-1", Name: "深夜循环", Tracks: tracks},
		{ID: "local-2", Name: "通勤路上", Tracks: second},
		{ID: "local-3", Name: "还没加歌", Tracks: []Track{}},
	}
}

func sampleSingers() []miguSinger {
	return []miguSinger{
		{ID: "s-1", Name: "周杰伦", Cover: "snap://s-1", CoverHQ: "snap://s-1", Hot: "华语男歌手", Fans: "2.8亿"},
		{ID: "s-2", Name: "林俊杰", Cover: "snap://s-2", CoverHQ: "snap://s-2", Hot: "华语男歌手", Fans: "1.9亿"},
		{ID: "s-3", Name: "邓紫棋", Cover: "snap://s-3", CoverHQ: "snap://s-3", Hot: "华语女歌手", Fans: "1.5亿"},
		{ID: "s-4", Name: "陈奕迅", Cover: "snap://s-4", CoverHQ: "snap://s-4", Hot: "华语男歌手", Fans: "1.3亿"},
		{ID: "s-5", Name: "五月天", Cover: "snap://s-5", CoverHQ: "snap://s-5", Hot: "华语组合", Fans: "1.1亿"},
	}
}

func sampleRadioGroups() []miguRadioGroup {
	return []miguRadioGroup{
		{
			Title: "私享电台",
			Items: []miguRadio{
				{ID: "r-private", Title: "私人 FM", Subtitle: "专属音乐雷达", Cover: "snap://r-1", Kind: "private"},
				{ID: "r-addict", Title: "上瘾电台", Subtitle: "一听上瘾", Cover: "snap://r-2", Kind: "private"},
				{ID: "r-different", Title: "听见不同", Subtitle: "发现惊喜", Cover: "snap://r-3", Kind: "private"},
			},
		},
		{
			Title: "场景电台",
			Items: []miguRadio{
				{ID: "r-work", Title: "上班路上", Subtitle: "给通勤一点节奏", Cover: "snap://r-4", Kind: "scene"},
				{ID: "r-night", Title: "深夜电台", Subtitle: "安静听完这一首", Cover: "snap://r-5", Kind: "scene"},
			},
		},
	}
}

func seedSnapshotArt(t *testing.T, a *app) {
	t.Helper()
	colors := []struct {
		a, b color.RGBA
	}{
		{color.RGBA{255, 112, 146, 255}, color.RGBA{255, 179, 112, 255}},
		{color.RGBA{132, 104, 255, 255}, color.RGBA{88, 190, 255, 255}},
		{color.RGBA{43, 170, 126, 255}, color.RGBA{162, 220, 117, 255}},
		{color.RGBA{240, 142, 84, 255}, color.RGBA{255, 210, 128, 255}},
		{color.RGBA{196, 92, 165, 255}, color.RGBA{255, 145, 155, 255}},
	}
	urls := []string{"snap://pl-1", "snap://pl-2", "snap://pl-3", "snap://pl-4", "snap://pl-5",
		"snap://s-1", "snap://s-2", "snap://s-3", "snap://s-4", "snap://s-5",
		"snap://r-1", "snap://r-2", "snap://r-3", "snap://r-4", "snap://r-5"}
	for i, raw := range urls {
		c := colors[i%len(colors)]
		src, img := snapshotGradient(t, c.a, c.b)
		a.arts[raw] = src
		a.artTints[raw] = coverTint(img)
		a.artBackdrops[raw] = ui.NewBitmap(coverBlur(img))
	}
}

func snapshotGradient(t *testing.T, from, to color.RGBA) (ui.ImageSource, *image.RGBA) {
	t.Helper()
	const size = 480
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		ratio := float64(y) / size
		for x := 0; x < size; x++ {
			mix := (ratio + float64(x)/size) / 2
			img.Set(x, y, color.RGBA{
				R: uint8(float64(from.R)*(1-mix) + float64(to.R)*mix),
				G: uint8(float64(from.G)*(1-mix) + float64(to.G)*mix),
				B: uint8(float64(from.B)*(1-mix) + float64(to.B)*mix),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	src, err := ui.DecodeImage(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	return src, img
}

func snapshotQRCode(t *testing.T) ui.ImageSource {
	t.Helper()
	pattern := []string{
		"11111110101",
		"10000010110",
		"10111010101",
		"10111010010",
		"10111010110",
		"10000010001",
		"11111110101",
		"00000000110",
		"11010111101",
		"00101000110",
		"11101110101",
	}
	const scale = 18
	img := image.NewRGBA(image.Rect(0, 0, len(pattern)*scale, len(pattern)*scale))
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			img.Set(x, y, color.White)
		}
	}
	for row, line := range pattern {
		for col, bit := range line {
			if bit != '1' {
				continue
			}
			for y := row * scale; y < (row+1)*scale; y++ {
				for x := col * scale; x < (col+1)*scale; x++ {
					img.Set(x, y, color.Black)
				}
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	src, err := ui.DecodeImage(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	return src
}
