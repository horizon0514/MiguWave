package main

import (
	"bytes"
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

	save := func(name string, a *app, dark bool) {
		tt := ui.NewTester(a.view, 1240, 820)
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

	tracks := sampleMiguTracks()

	home := testApp()
	home.home.NewSongs = tracks
	home.home.Ranks = []miguRankSummary{
		{ID: "83176390", Title: "国风热歌榜", Subtitle: "每周更新", Tracks: tracks},
		{ID: "75959118", Title: "音乐风向榜", Subtitle: "每小时更新", Tracks: tracks},
		{ID: "27553319", Title: "尖叫新歌榜", Subtitle: "实时新歌", Tracks: tracks},
	}
	save("home", home, false)
	save("home-dark", home, true)

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

	account := testApp()
	account.accountOpen = true
	account.authPhase = miguAuthBrowserWaiting
	save("account", account, false)

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
