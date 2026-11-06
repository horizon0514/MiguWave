package app

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"freewave/internal/lyrics"
	"freewave/internal/migu"
)

var (
	songWhite = ui.RGB(255, 255, 255)
	songInk   = ui.Hex("#171214")
)

type songPane int

const (
	songPaneLyrics songPane = iota
	songPaneStage
	songPaneQueue
)

var songSpring = ui.Animation{Spring: ui.SpringOptions{
	Duration: 560 * time.Millisecond,
	Bounce:   0.12,
}}

type songLayout struct {
	sideX  float32
	left   float32
	cover  float32
	gap    float32
	rightX float32
}

func songLayoutFor(w, h float32) songLayout {
	cover := min(float32(392), min(w*0.34, h-390))
	cover = max(float32(270), cover)
	left := max(float32(310), cover)
	gap := float32(68)
	if w < 1180 {
		gap = 60
	}
	side := max(float32(56), (w-1180)/2)
	return songLayout{
		sideX:  side,
		left:   left,
		cover:  cover,
		gap:    gap,
		rightX: side + left + gap,
	}
}

func (a *app) openSong(track Track, queue []Track, index int) {
	if strings.TrimSpace(track.ID) == "" {
		return
	}
	if index < 0 || index >= len(queue) || queue[index].ID != track.ID {
		queue = []Track{track}
		index = 0
	}
	if a.page != pageSong {
		a.songFrom = a.page
	}
	a.page = pageSong
	a.songTrack = track
	a.songQueue = append([]Track(nil), queue...)
	a.songIndex = index
	a.songErr = ""
	a.lyricActive = -2
	a.lyricAuto = false
	a.lyricScroll = ui.ScrollState{}
	a.lyricGlide = lyricGlide{}
	a.lyricTrack = ""
	a.lyricAt = -2
	if a.songCache == nil {
		a.songCache = map[string]miguSongDetail{}
	}
	if detail, ok := a.songCache[track.ID]; ok {
		a.songDetail = detail
		a.songTrack = detail.Track
		a.songLoading = false
		a.refresh()
		return
	}
	a.songDetail = miguSongDetail{Track: track, Album: track.Album, Cover: track.Cover}
	a.loadSongDetail(track)
}

func (a *app) loadSongDetail(track Track) {
	a.songGen++
	gen := a.songGen
	token := a.miguToken
	a.songLoading = true
	a.songErr = ""
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		detail, err := migu.FetchSongDetail(ctx, token, track)
		a.post(func() {
			if a.songGen != gen || a.songTrack.ID != track.ID {
				return
			}
			a.songLoading = false
			if err != nil {
				a.songErr = friendlyError(err)
				return
			}
			a.songDetail = detail
			a.songTrack = detail.Track
			if a.songCache == nil {
				a.songCache = map[string]miguSongDetail{}
			}
			a.songCache[track.ID] = detail
			if a.songIndex >= 0 && a.songIndex < len(a.songQueue) &&
				a.songQueue[a.songIndex].ID == track.ID {
				a.songQueue[a.songIndex] = detail.Track
			}
			a.lyricActive = -2
			a.lyricScroll = ui.ScrollState{}
			a.lyricGlide = lyricGlide{}
			a.lyricTrack = ""
			a.lyricAt = -2
		})
	}()
}

func (a *app) retrySongDetail() {
	if a.songCache != nil {
		delete(a.songCache, a.songTrack.ID)
	}
	a.loadSongDetail(a.songTrack)
}

func (a *app) followPlayerTrack(track Track) {
	if a.page != pageSong || strings.TrimSpace(track.ID) == "" {
		return
	}
	if a.songTrack.ID == track.ID && a.songDetail.Track.ID == track.ID {
		return
	}
	a.songTrack = track
	a.songErr = ""
	a.lyricActive = -2
	a.lyricAuto = false
	a.lyricScroll = ui.ScrollState{}
	a.lyricGlide = lyricGlide{}
	a.lyricTrack = ""
	a.lyricAt = -2
	if a.pl != nil {
		a.songQueue = append([]Track(nil), a.pl.queue...)
		a.songIndex = a.pl.at
	}
	if detail, ok := a.songCache[track.ID]; ok {
		a.songDetail = detail
		a.songTrack = detail.Track
		a.songLoading = false
		a.refresh()
		return
	}
	a.songDetail = miguSongDetail{Track: track, Album: track.Album, Cover: track.Cover}
	a.loadSongDetail(track)
}

func (a *app) jumpToLyric(at time.Duration) {
	if a.pl == nil {
		return
	}
	if !a.pl.hasCurrent || a.pl.current.ID != a.songDetail.Track.ID {
		if len(a.songQueue) == 0 {
			return
		}
		a.playTracks(a.songQueue, a.songIndex)
	}
	a.pl.seek(at)
}

func (a *app) songBackTarget() pageKind {
	if a.songFrom == pageSong {
		return pageHome
	}
	return a.songFrom
}

func (a *app) songPage(c *ui.Context, _ palette) {
	detail := a.songDetail
	track := detail.Track
	if track.ID == "" {
		track = a.songTrack
	}
	cover := firstNonEmpty(detail.Cover, track.Cover)
	album := firstNonEmpty(detail.Album, track.Album)
	a.requestArt(cover)

	tint := ui.Hex("#8c6657")
	if a.artTints != nil {
		if v, ok := a.artTints[cover]; ok {
			tint = v
		}
	}
	p := songPalette(tint)
	w, h := c.Size()
	layout := songLayoutFor(w, h)

	root := ui.Box(c.Key("song-page")).Fill().Clip().Background(songInk).TextColor(songWhite)
	root.Children(func() {
		a.songBackdrop(c, cover, tint)
		ui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Children(func() {
			a.songTopBar(c)
			a.songPaneTabs(c, layout)
			ui.Row(c).Grow(1).MinHeight(0).
				Padding(48, layout.sideX, 26, layout.sideX).Gap(layout.gap).
				AlignItems(ui.Stretch).Children(func() {
				a.songLeftPane(c, p, track, cover, album, layout)
				a.songRightPane(c, p, detail, h)
			})
		})
	})
}

func (a *app) songBackdrop(c *ui.Context, cover string, tint ui.Color) {
	ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough().
		Draw(func(pt *ui.Painter, r ui.Rect) {
			top, foot := songBackdropColors(tint)
			pt.FillGradient(r, ui.LinearGradient{Angle: 180, Oklab: true, Stops: []ui.GradientStop{
				{Color: top, At: 0},
				{Color: foot, At: 1},
			}}, 0)
			if a.artBackdrops != nil {
				if blurred := a.artBackdrops[cover]; blurred != nil {
					pt.ImageRegion(blurred, ui.Rect{W: 48, H: 48}, coverRect(r, 48, 48),
						ui.ImageOptions{Tint: songWhite.Alpha(0.14)})
				}
			}
			glow := tint.Mix(songWhite, 0.22)
			pt.FillRadialGradient(r, ui.RadialGradient{
				Center: ui.Point{X: 0.28, Y: 0.38},
				Radius: max(r.W, r.H) * 0.58,
				Stops: []ui.GradientStop{
					{Color: glow.Alpha(0.22), At: 0},
					{Color: glow.Alpha(0), At: 1},
				},
			}, 0)
			pt.FillGradient(r, ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
				{Color: ui.RGBA(0, 0, 0, 0), At: 0.44},
				{Color: ui.RGBA(0, 0, 0, 0.24), At: 0.82},
				{Color: ui.RGBA(0, 0, 0, 0.38), At: 1},
			}}, 0)
		})
}

func (a *app) songTopBar(c *ui.Context) {
	ui.Row(c).Height(58).FillWidth().Padding(0, 24, 0, 24).
		Justify(ui.End).AlignItems(ui.Center).DragWindow().
		Children(func() {
			back := ui.ButtonBase(c).Key("song-back").Size(38, 34).Center().Radius(10).
				FocusRing(false).Label("收起播放页").Tooltip("收起播放页")
			if back.Hovered() {
				back.Background(songWhite.Alpha(0.08))
			}
			back.Children(func() {
				ui.Icon(c, iconDown).Size(21, 21).TextColor(songWhite.Alpha(0.64))
			})
			if back.Clicked() {
				a.navTo(a.songBackTarget())
			}
		})
}

func (a *app) songPaneTabs(c *ui.Context, layout songLayout) {
	tabs := []struct {
		pane  songPane
		label string
	}{
		{songPaneLyrics, "歌词"},
		{songPaneStage, "律动"},
		{songPaneQueue, "待播清单"},
	}
	ui.Row(c).Absolute().Top(17).Left(layout.rightX).Height(32).Gap(5).
		AlignItems(ui.Center).Children(func() {
		for _, tab := range tabs {
			active := a.songPane == tab.pane
			b := ui.ButtonBase(c).Key(fmt.Sprintf("song-tab-%d", tab.pane)).
				Height(30).PaddingX(12).Radius(999).Center().FocusRing(false).
				Label(tab.label)
			bg := ui.Transparent
			fg := songWhite.Alpha(0.42)
			switch {
			case active:
				bg, fg = songWhite.Alpha(0.1), songWhite.Alpha(0.92)
			case b.Hovered():
				bg, fg = songWhite.Alpha(0.055), songWhite.Alpha(0.72)
			}
			motion := ui.Animation{Duration: 140 * time.Millisecond}
			b.Background(b.AnimateColor("bg", bg, motion))
			fg = b.AnimateColor("fg", fg, motion)
			b.Children(func() {
				ui.Text(c, tab.label).FontSize(12.5).FontWeight(610).TextColor(fg)
			})
			if b.Clicked() {
				a.songPane = tab.pane
				a.refresh()
			}
		}
	})
}

func (a *app) songLeftPane(c *ui.Context, p songColors, track Track, cover, album string, layout songLayout) {
	ui.Column(c).Width(layout.left).MinWidth(270).Children(func() {
		a.songArtworkCard(c, cover, p.tint, layout.cover)

		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Margin(26, 0, 12, 0).
			Children(func() {
				ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(3).Children(func() {
					ui.Text(c, displayTitle(track.Title)).FontSize(26).FontWeight(760).
						LetterSpacing(-0.5).SingleLine().TextColor(songWhite)
					ui.Text(c, displayArtist(track.Artist)).FontSize(14).FontWeight(520).
						SingleLine().TextColor(songWhite.Alpha(0.58))
					meta := joinMeta(album, durationLabel(track))
					if track.VIP {
						meta = joinMeta(meta, "VIP")
					}
					if meta != "" {
						ui.Text(c, meta).FontSize(10.5).TextColor(songWhite.Alpha(0.32)).
							SingleLine().Margin(4, 0, 0, 0)
					}
				})
				a.songPlaylistAddButton(c, track)
				a.songFavoriteButton(c, track)
			})

		ui.Box(c).Margin(15, 0, 0, 0).Children(func() {
			a.songProgress(c, p, track)
		})
		ui.Box(c).Margin(11, 0, 0, 0).Children(func() {
			a.songTransport(c, track)
		})
		ui.Row(c).Justify(ui.Center).AlignItems(ui.Center).Gap(8).Margin(9, 0, 0, 0).
			Children(func() {
				a.songVolume(c)
			})
	})
}

func (a *app) songArtworkCard(c *ui.Context, cover string, tint ui.Color, size float32) {
	wrap := ui.Box(c).Size(size, size).AlignSelf(ui.Center).Label("专辑封面")
	px, py, over := wrap.PointerPosition()
	tx, ty := float32(0), float32(0)
	if over {
		tx = min(float32(1), max(float32(-1), px/size*2-1))
		ty = min(float32(1), max(float32(-1), py/size*2-1))
	}
	reduce := c.Preferences().ReduceMotion
	if reduce {
		tx, ty = 0, 0
	}
	tilt := ui.Animation{Spring: ui.SpringOptions{Duration: 500 * time.Millisecond, Bounce: 0.18}}
	rx := wrap.AnimateTo("song-rx", -ty*5.5, tilt)
	ry := wrap.AnimateTo("song-ry", tx*5.5, tilt)
	shineTarget := float32(0)
	if over {
		shineTarget = 1
	}
	shine := wrap.AnimateTo("song-shine", shineTarget, ui.Animation{Duration: 260 * time.Millisecond})
	scale := float32(1)
	if over {
		scale = 1.012
	}

	wrap.Children(func() {
		box := ui.Box(c).Fill().Radius(18).
			Shadow(0, 30, 62, -16, ui.RGBA(0, 0, 0, 0.58)).
			RotateX(rx).RotateY(ry).Perspective(900)
		box.AnimatePose(ui.Pose{Scale: scale}, songSpring)
		box.Background(ui.RGBA(255, 255, 255, 0.07))
		box.Clip().Children(func() {
			a.songArtwork(c, cover, tint)
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough().
				Border(1, songWhite.Alpha(0.1)).Radius(18)
			if shine > 0.01 {
				ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough().
					Draw(func(pt *ui.Painter, r ui.Rect) {
						pt.FillRadialGradient(r, ui.RadialGradient{
							Center: ui.Point{X: px / size, Y: py / size},
							Radius: max(r.W, r.H) * 0.82,
							Stops: []ui.GradientStop{
								{Color: songWhite.Alpha(0.14 * shine), At: 0},
								{Color: songWhite.Alpha(0), At: 1},
							},
						}, 18)
					})
			}
		})
	})
}

func (a *app) songArtwork(c *ui.Context, cover string, tint ui.Color) ui.Element {
	if src := a.arts[cover]; src != nil {
		return ui.Image(c, src).Fill().Fit(ui.Cover)
	}
	box := ui.Box(c).Fill().Gradient(tint.Mix(songWhite, 0.16), tint.Mix(songInk, 0.12), 142)
	box.Children(func() {
		ui.Box(c).Fill().Draw(func(pt *ui.Painter, r ui.Rect) {
			size := min(r.W, r.H) * 0.2
			pt.Icon(iconMusic, ui.Rect{
				X: r.X + (r.W-size)/2,
				Y: r.Y + (r.H-size)/2,
				W: size,
				H: size,
			}, songWhite.Alpha(0.3))
		})
	})
	return box
}

func (a *app) songFavoriteButton(c *ui.Context, track Track) {
	fav := a.isFavorite(track.ID)
	label := "收藏"
	ic := iconHeart
	if fav {
		label = "已收藏"
		ic = iconHeartFill
	}
	b := ui.ButtonBase(c).Key("song-fav").Size(36, 36).Center().Radius(999).
		FocusRing(false).Label(label).Tooltip(label)
	switch {
	case fav:
		b.Background(songWhite.Alpha(0.1))
	case b.Hovered():
		b.Background(songWhite.Alpha(0.075))
	}
	b.Children(func() {
		color := songWhite.Alpha(0.62)
		if fav {
			color = songWhite
		}
		ui.Icon(c, ic).Size(18, 18).TextColor(color)
	})
	if b.Clicked() {
		a.toggleFavorite(track)
	}
}

func (a *app) songPlaylistAddButton(c *ui.Context, track Track) {
	b := ui.ButtonBase(c).Key("song-add-playlist").Size(36, 36).Center().Radius(999).
		FocusRing(false).Label("添加到自建歌单").Tooltip("添加到自建歌单")
	if b.Hovered() {
		b.Background(songWhite.Alpha(0.075))
	}
	b.Children(func() {
		ui.Icon(c, iconPlus).Size(18, 18).TextColor(songWhite.Alpha(0.62))
	})
	b.PopoverMenuButton(a.addToPlaylistMenu(track))
}

func (a *app) songProgress(c *ui.Context, p songColors, track Track) {
	same := a.pl != nil && a.pl.hasCurrent && a.pl.current.ID == track.ID
	if same {
		a.progressControl(c, p.palette)
	} else {
		ui.Box(c).Height(18).FillWidth().Draw(func(pt *ui.Painter, r ui.Rect) {
			pt.Fill(ui.Rect{X: r.X, Y: r.Y + r.H/2 - 2, W: r.W, H: 4},
				songWhite.Alpha(0.16), 2)
		})
	}
	ui.Row(c).AlignItems(ui.Center).Margin(0, 0, 2, 0).Children(func() {
		pos := time.Duration(0)
		dur := track.Duration()
		if same {
			pos = a.pl.position()
			dur = a.pl.duration
		}
		ui.Text(c, formatClock(pos)).FontSize(10.5).FontFeatures("tnum").
			TextColor(songWhite.Alpha(0.4))
		ui.Spacer(c)
		if dur <= 0 {
			ui.Text(c, "--:--").FontSize(10.5).FontFeatures("tnum").
				TextColor(songWhite.Alpha(0.4))
		} else {
			ui.Text(c, formatClock(dur)).FontSize(10.5).FontFeatures("tnum").
				TextColor(songWhite.Alpha(0.4))
		}
	})
}

func (a *app) songTransport(c *ui.Context, track Track) {
	enabled := a.pl != nil
	ui.Row(c).Justify(ui.Center).AlignItems(ui.Center).Gap(10).Children(func() {
		a.songIconButton(c, "song-shuffle", iconShuffle, "随机播放",
			enabled && a.pl.shuffle, 17, func() {
				if a.pl != nil {
					a.pl.shuffle = !a.pl.shuffle
					a.refresh()
				}
			})
		a.songIconButton(c, "song-prev", iconPrev, "上一首", false, 19, func() {
			if a.pl != nil {
				a.pl.prev()
			}
		})
		a.songPlayButton(c, track)
		a.songIconButton(c, "song-next", iconNext, "下一首", false, 19, func() {
			if a.pl != nil {
				a.pl.next()
			}
		})
		repeatActive := enabled && a.pl.repeat != repeatOff
		repeatIcon := iconRepeat
		if enabled && a.pl.repeat == repeatOne {
			repeatIcon = iconRepeatOne
		}
		a.songIconButton(c, "song-repeat", repeatIcon, repeatLabel(songRepeat(a)),
			repeatActive, 17, func() {
				if a.pl != nil {
					a.pl.repeat = (a.pl.repeat + 1) % 3
					a.refresh()
				}
			})
	})
}

func songRepeat(a *app) repeatMode {
	if a == nil || a.pl == nil {
		return repeatOff
	}
	return a.pl.repeat
}

func (a *app) songPlayButton(c *ui.Context, track Track) {
	same := a.pl != nil && a.pl.hasCurrent && a.pl.current.ID == track.ID
	loading := same && a.pl.loading
	playing := same && a.pl.playing
	label := "播放"
	ic := iconPlay
	switch {
	case loading:
		label = "载入中"
	case playing:
		label = "暂停"
		ic = iconPause
	}
	b := ui.ButtonBase(c).Key("song-play").Size(56, 56).Center().Radius(999).
		FocusRing(false).Label(label).Background(songWhite)
	if b.Hovered() {
		b.Background(ui.Hex("#f6f1f3"))
	}
	b.Children(func() {
		if loading {
			ui.Spinner(c).FontSize(18).TextColor(songInk)
		} else {
			ui.Icon(c, ic).Size(20, 20).TextColor(songInk)
		}
	})
	if b.Clicked() && !loading {
		if same {
			a.pl.toggle()
		} else if len(a.songQueue) > 0 {
			a.playTracks(a.songQueue, a.songIndex)
		}
	}
}

func (a *app) songIconButton(c *ui.Context, key any, ic *ui.SVG, label string, active bool, size float32, onClick func()) {
	b := ui.ButtonBase(c).Key(key).Size(38, 38).Center().Radius(999).
		FocusRing(false).Label(label).Tooltip(label)
	if b.Hovered() {
		b.Background(songWhite.Alpha(0.075))
	}
	b.Children(func() {
		color := songWhite.Alpha(0.44)
		if active {
			color = songWhite.Alpha(0.92)
		} else if b.Hovered() {
			color = songWhite.Alpha(0.72)
		}
		ui.Icon(c, ic).Size(size, size).TextColor(color)
	})
	if b.Clicked() {
		onClick()
	}
}

func (a *app) songVolume(c *ui.Context) {
	ic := iconVolume
	if a.volume <= 0.01 {
		ic = iconVolumeOff
	}
	ui.Icon(c, ic).Size(14, 14).TextColor(songWhite.Alpha(0.38))
	vol := ui.Slider(c, &a.volume, 0, 1).Width(154).Label("音量")
	if vol.Changed() && a.pl != nil {
		a.pl.setVolume(a.volume)
		a.save()
	}
}

func (a *app) songRightPane(c *ui.Context, p songColors, detail miguSongDetail, h float32) {
	active := -1
	if !detail.PlainLyrics && len(detail.Lyrics) > 0 {
		same := a.pl != nil && a.pl.hasCurrent && a.pl.current.ID == detail.Track.ID
		if same {
			active = lyrics.IndexAt(detail.Lyrics, a.pl.position())
		} else {
			active = 0
		}
	}
	if active != a.lyricActive {
		a.lyricActive = active
	}

	ui.Column(c).Grow(1).Basis(0).MinWidth(0).AlignItems(ui.Stretch).Children(func() {
		switch a.songPane {
		case songPaneStage:
			a.songStage(c, p, detail, active)
		case songPaneQueue:
			a.songQueuePane(c, p)
		default:
			switch {
			case a.songLoading:
				a.songLyricMessage(c, true, "正在取回歌词…", "", nil)
			case a.songErr != "":
				a.songLyricMessage(c, false, a.songErr, "重新加载", a.retrySongDetail)
			case detail.LyricErr != "":
				a.songLyricMessage(c, false, detail.LyricErr, "重新加载歌词", a.retrySongDetail)
			case len(detail.Lyrics) == 0:
				a.songLyricMessage(c, false, "咪咕暂未提供这首歌的歌词", "", nil)
			case detail.PlainLyrics:
				a.plainSongLyrics(c, detail, h)
			default:
				a.songLyricLines(c, detail, active, h)
			}
		}
	})
}

func (a *app) songLyricMessage(c *ui.Context, loading bool, message, retry string, onClick func()) {
	ui.Box(c).Fill().Center().Children(func() {
		ui.Column(c).Gap(14).AlignItems(ui.Center).Children(func() {
			if loading {
				ui.Spinner(c).FontSize(22).TextColor(songWhite.Alpha(0.6))
			} else {
				ui.Icon(c, iconLyrics).Size(28, 28).TextColor(songWhite.Alpha(0.22))
			}
			ui.Text(c, message).FontSize(13).TextColor(songWhite.Alpha(0.46)).
				TextAlign(ui.Center).MaxWidth(520)
			if retry != "" && onClick != nil {
				b := ui.ButtonBase(c).Key("song-lyric-retry").Height(34).PaddingX(15).
					Radius(999).AlignItems(ui.Center).FocusRing(false).Label(retry).
					Background(songWhite.Alpha(0.08))
				if b.Hovered() {
					b.Background(songWhite.Alpha(0.14))
				}
				b.Children(func() {
					ui.Text(c, retry).FontSize(11.5).FontWeight(620).
						TextColor(songWhite.Alpha(0.7))
				})
				if b.Clicked() {
					onClick()
				}
			}
		})
	})
}

func (a *app) plainSongLyrics(c *ui.Context, detail miguSongDetail, h float32) {
	ui.Scroll(c).Key("song-plain-lyrics").FillWidth().Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).FillWidth().AlignItems(ui.Start).
			Padding(max(34, h*0.06), 10, h*0.24, 0).
			Gap(13).Children(func() {
			for _, line := range detail.Lyrics {
				ui.Text(c, line.Text).FontSize(20).FontWeight(580).LineHeight(1.65).
					TextColor(songWhite.Alpha(0.5))
			}
		})
	})
}

func (a *app) songLyricLines(c *ui.Context, detail miguSongDetail, active int, h float32) {
	if a.lyricTrack != detail.Track.ID {
		a.lyricTrack = detail.Track.ID
		a.lyricAt = -2
		a.lyricScroll = ui.ScrollState{}
		a.lyricGlide = lyricGlide{}
	}
	lag := a.lyricGlide.step(c, &a.lyricScroll)
	var activeLine ui.Element

	scroll := ui.Scroll(c).Key("song-lyrics").FillWidth().Grow(1).MinHeight(0).
		TrackScroll(&a.lyricScroll)
	scroll.Children(func() {
		ui.Column(c).FillWidth().AlignItems(ui.Start).Translate(0, -lag).Children(func() {
			ui.Box(c).Height(max(56, h*0.1))
			for i, line := range detail.Lyrics {
				if strings.TrimSpace(line.Text) == "" {
					ui.Box(c).Height(13)
					continue
				}
				selected := i == active
				row := ui.ButtonBase(c).
					Key(fmt.Sprintf("song-lyric-%s-%d", detail.Track.ID, i)).
					FillWidth().Padding(10, 12, 10, 12).Radius(12).
					Justify(ui.Start).AlignItems(ui.Start).FocusRing(false).Label(line.Text)
				if row.Hovered() && !selected {
					row.Background(songWhite.Alpha(0.035))
				}
				scale := float32(0.94)
				if selected {
					scale = 1
				}
				row.TransformOrigin(0, 0.5).AnimatePose(
					ui.Pose{Scale: scale},
					ui.Animation{Spring: ui.SpringOptions{
						Duration: 600 * time.Millisecond,
						Bounce:   0.08,
					}},
				)
				textColor := songWhite.Alpha(0.24)
				weight := 700
				switch {
				case selected:
					textColor, weight = songWhite, 800
				case row.Hovered():
					textColor, weight = songWhite.Alpha(0.48), 720
				}
				row.Children(func() {
					ui.Text(c, line.Text).FontSize(29).FontWeight(weight).LineHeight(1.28).
						LetterSpacing(-0.45).TextColor(textColor)
				})
				if row.Clicked() {
					a.jumpToLyric(line.At)
				}
				if selected {
					activeLine = row
				}
			}
			ui.Box(c).Height(max(100, h*0.44))
		})
	})

	if active != a.lyricAt {
		a.lyricAt = active
		if active < 0 {
			a.lyricScroll.Y = 0
			a.lyricGlide = lyricGlide{}
			return
		}
		box := scroll.Bounds()
		target := float32(0)
		if activeLine.Valid() {
			if lb := activeLine.LayoutBounds(); lb.H > 0 && box.H > 0 {
				target = lb.Y - box.Y + a.lyricScroll.Y - box.H*0.3
			} else {
				return
			}
		}
		target = min(max(0, target), a.lyricScroll.MaxY)
		a.lyricGlide.to(c, &a.lyricScroll, target)
	}
}

func (a *app) songQueuePane(c *ui.Context, p songColors) {
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Height(32).AlignItems(ui.Center).Margin(0, 0, 10, 0).Children(func() {
			ui.Text(c, "接下来").FontSize(12).FontWeight(650).LetterSpacing(0.4).
				TextColor(songWhite.Alpha(0.5))
			ui.Spacer(c)
			ui.Textf(c, "%d 首", len(a.songQueue)).FontSize(11).
				FontFeatures("tnum").TextColor(songWhite.Alpha(0.3))
		})
		if len(a.songQueue) == 0 {
			ui.Box(c).Grow(1).Center().Children(func() {
				ui.Text(c, "播放队列是空的").FontSize(13).TextColor(songWhite.Alpha(0.36))
			})
			return
		}
		ui.Scroll(c).Key("song-queue").FillWidth().Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().Gap(3).Padding(0, 4, 24, 0).Children(func() {
				for i, track := range a.songQueue {
					a.songQueueRow(c, p, track, i)
				}
			})
		})
	})
}

func (a *app) songQueueRow(c *ui.Context, p songColors, track Track, index int) {
	same := a.pl != nil && a.pl.hasCurrent && a.pl.current.ID == track.ID
	row := ui.ButtonBase(c).Key("song-queue-" + track.ID).Height(56).PaddingX(10).
		Gap(12).Radius(11).Justify(ui.Start).AlignItems(ui.Center).
		FocusRing(false).Label(track.Title)
	bg := ui.Transparent
	switch {
	case same:
		bg = songWhite.Alpha(0.075)
	case row.Hovered():
		bg = songWhite.Alpha(0.045)
	}
	row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 120 * time.Millisecond}))
	row.Children(func() {
		ui.Box(c).Width(24).Center().Children(func() {
			switch {
			case same:
				a.playingBars(c, p.palette, a.pl.playing)
			case row.Hovered():
				ui.Icon(c, iconPlay).Size(12, 12).TextColor(songWhite.Alpha(0.76))
			default:
				ui.Textf(c, "%d", index+1).FontSize(11.5).FontFeatures("tnum").
					TextColor(songWhite.Alpha(0.28))
			}
		})
		ui.Box(c).Size(40, 40).Radius(8).Clip().Children(func() {
			a.songArtwork(c, track.Cover, p.tint)
		})
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(3).Children(func() {
			color := songWhite.Alpha(0.86)
			if same {
				color = songWhite
			}
			ui.Text(c, displayTitle(track.Title)).FontSize(13.5).FontWeight(590).
				SingleLine().TextColor(color)
			ui.Text(c, displayArtist(track.Artist)).FontSize(11.5).
				SingleLine().TextColor(songWhite.Alpha(0.38))
		})
		if track.DurationSec > 0 {
			ui.Text(c, formatClock(track.Duration())).FontSize(10.5).
				FontFeatures("tnum").TextColor(songWhite.Alpha(0.28)).Width(42).
				TextAlign(ui.End)
		}
	})
	if row.Clicked() {
		if !same {
			a.playTracks(a.songQueue, index)
		}
		if a.songTrack.ID != track.ID {
			a.openSong(track, a.songQueue, index)
		}
	}
}

func displayTitle(title string) string {
	if strings.TrimSpace(title) == "" {
		return "未知歌曲"
	}
	return title
}

func displayArtist(artist string) string {
	if strings.TrimSpace(artist) == "" {
		return "未知艺人"
	}
	return artist
}

func durationLabel(track Track) string {
	if track.DurationSec <= 0 {
		return ""
	}
	return formatClock(track.Duration())
}

type songColors struct {
	palette
	tint ui.Color
}

func songPalette(tint ui.Color) songColors {
	return songColors{
		palette: palette{
			bg:       songInk,
			sidebar:  songInk,
			panel:    songWhite.Alpha(0.05),
			panelAlt: songWhite.Alpha(0.08),
			hover:    songWhite.Alpha(0.08),
			line:     songWhite.Alpha(0.16),
			text:     songWhite,
			muted:    songWhite.Alpha(0.56),
			faint:    songWhite.Alpha(0.3),
			accent:   songWhite.Alpha(0.88),
			accent2:  tint,
			onAccent: songInk,
			shadow:   ui.RGBA(0, 0, 0, 0.5),
		},
		tint: tint,
	}
}

func songBackdropColors(tint ui.Color) (top, foot ui.Color) {
	h, s, _ := rgbHSV(float64(tint.R)/255, float64(tint.G)/255, float64(tint.B)/255)
	if s > 0.12 {
		s = min(max(s*1.35, 0.46), 0.72)
	}
	return hsvColor(h, s, 0.35), hsvColor(h, min(s*1.02, 0.72), 0.22)
}

func hsvColor(h, s, v float64) ui.Color {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	var r, g, b float64
	switch {
	case h < 60:
		r, g = c, x
	case h < 120:
		r, g = x, c
	case h < 180:
		g, b = c, x
	case h < 240:
		g, b = x, c
	case h < 300:
		r, b = x, c
	default:
		r, b = c, x
	}
	m := v - c
	return ui.RGB(uint8((r+m)*255+0.5), uint8((g+m)*255+0.5), uint8((b+m)*255+0.5))
}

func rgbHSV(r, g, b float64) (h, s, v float64) {
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	v = mx
	if mx == 0 {
		return 0, 0, 0
	}
	s = (mx - mn) / mx
	d := mx - mn
	switch {
	case d == 0:
		h = 0
	case mx == r:
		h = math.Mod((g-b)/d, 6) * 60
	case mx == g:
		h = ((b-r)/d + 2) * 60
	default:
		h = ((r-g)/d + 4) * 60
	}
	if h < 0 {
		h += 360
	}
	return h, s, v
}

func inspectCover(data []byte) (ui.Color, *ui.Bitmap) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ui.Color{}, nil
	}
	return coverTint(img), ui.NewBitmap(coverBlur(img))
}

func coverTint(img image.Image) ui.Color {
	small := shrinkCover(img, 24)
	const buckets = 18
	var weight [buckets]float64
	var sum [buckets][3]float64
	var gray [3]float64
	for i := 0; i < len(small.Pix); i += 4 {
		r := float64(small.Pix[i]) / 255
		g := float64(small.Pix[i+1]) / 255
		b := float64(small.Pix[i+2]) / 255
		h, s, v := rgbHSV(r, g, b)
		gray[0], gray[1], gray[2] = gray[0]+r, gray[1]+g, gray[2]+b
		w := s * s * v
		if s < 0.15 || v < 0.15 {
			continue
		}
		k := int(h/360*buckets) % buckets
		weight[k] += w
		sum[k][0], sum[k][1], sum[k][2] = sum[k][0]+r*w, sum[k][1]+g*w, sum[k][2]+b*w
	}
	best := 0
	for k := range weight {
		if weight[k] > weight[best] {
			best = k
		}
	}
	if weight[best] < 2 {
		n := float64(len(small.Pix) / 4)
		return ui.RGB(uint8(gray[0]/n*255), uint8(gray[1]/n*255), uint8(gray[2]/n*255))
	}
	w := weight[best]
	return ui.RGB(uint8(sum[best][0]/w*255), uint8(sum[best][1]/w*255), uint8(sum[best][2]/w*255))
}

func shrinkCover(img image.Image, n int) *image.RGBA {
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := range n {
		for x := range n {
			x0, x1 := b.Min.X+x*b.Dx()/n, b.Min.X+(x+1)*b.Dx()/n
			y0, y1 := b.Min.Y+y*b.Dy()/n, b.Min.Y+(y+1)*b.Dy()/n
			var r, g, bl, cnt uint32
			for yy := y0; yy < max(y1, y0+1); yy += max(1, (y1-y0)/4) {
				for xx := x0; xx < max(x1, x0+1); xx += max(1, (x1-x0)/4) {
					cr, cg, cb, _ := img.At(xx, yy).RGBA()
					r, g, bl, cnt = r+cr>>8, g+cg>>8, bl+cb>>8, cnt+1
				}
			}
			out.SetRGBA(x, y, color.RGBA{uint8(r / cnt), uint8(g / cnt), uint8(bl / cnt), 255})
		}
	}
	return out
}

func coverBlur(img image.Image) *image.RGBA {
	const n = 48
	small := shrinkCover(img, n)
	px := make([][3]float64, n*n)
	for i := range px {
		r := float64(small.Pix[4*i])
		g := float64(small.Pix[4*i+1])
		b := float64(small.Pix[4*i+2])
		l := 0.3*r + 0.59*g + 0.11*b
		px[i] = [3]float64{l + (r-l)*1.5, l + (g-l)*1.5, l + (b-l)*1.5}
	}
	tmp := make([][3]float64, n*n)
	for range 3 {
		boxBlurCover(px, tmp, n, 5)
	}
	out := image.NewRGBA(image.Rect(0, 0, n, n))
	for i, p := range px {
		for c := range 3 {
			out.Pix[4*i+c] = uint8(math.Max(0, math.Min(255, p[c])))
		}
		out.Pix[4*i+3] = 255
	}
	return out
}

func boxBlurCover(px, tmp [][3]float64, n, r int) {
	at := func(i int) int { return max(0, min(n-1, i)) }
	for y := range n {
		for x := range n {
			var s [3]float64
			for k := -r; k <= r; k++ {
				p := px[y*n+at(x+k)]
				s[0], s[1], s[2] = s[0]+p[0], s[1]+p[1], s[2]+p[2]
			}
			d := float64(2*r + 1)
			tmp[y*n+x] = [3]float64{s[0] / d, s[1] / d, s[2] / d}
		}
	}
	for y := range n {
		for x := range n {
			var s [3]float64
			for k := -r; k <= r; k++ {
				p := tmp[at(y+k)*n+x]
				s[0], s[1], s[2] = s[0]+p[0], s[1]+p[1], s[2]+p[2]
			}
			d := float64(2*r + 1)
			px[y*n+x] = [3]float64{s[0] / d, s[1] / d, s[2] / d}
		}
	}
}

func coverRect(r ui.Rect, w, h float32) ui.Rect {
	s := max(r.W/w, r.H/h)
	return ui.Rect{
		X: r.X + (r.W-w*s)/2,
		Y: r.Y + (r.H-h*s)/2,
		W: w * s,
		H: h * s,
	}
}

type lyricGlide struct {
	on      bool
	a, b    float64
	start   time.Time
	target  float32
	pending bool
	set     float32
}

const lyricOmega = 10

func (g *lyricGlide) at(now time.Time) (lag, v float64) {
	if !g.on {
		return 0, 0
	}
	t := now.Sub(g.start).Seconds()
	e := math.Exp(-lyricOmega * t)
	return (g.a + g.b*t) * e, (g.b - lyricOmega*(g.a+g.b*t)) * e
}

func (g *lyricGlide) to(c *ui.Context, s *ui.ScrollState, target float32) {
	now := c.Now()
	lag, v := g.at(now)
	shown := float64(s.Y) + lag
	g.target, g.pending = target, true
	g.on = !c.Preferences().ReduceMotion
	g.start, g.a = now, shown-float64(target)
	g.b = v + lyricOmega*g.a
	c.AnimationFrame()
}

func (g *lyricGlide) step(c *ui.Context, s *ui.ScrollState) float32 {
	if g.pending {
		s.Y, g.set, g.pending = g.target, g.target, false
	}
	if !g.on {
		return 0
	}
	lag, v := g.at(c.Now())
	if s.Y != g.set {
		s.Y, g.on = max(0, min(s.Y+float32(lag), s.MaxY)), false
		return 0
	}
	if math.Abs(lag) < 0.5 && math.Abs(v) < 20 {
		g.on = false
		return 0
	}
	c.AnimationFrame()
	return float32(lag)
}
