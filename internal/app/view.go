package app

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	sidebarWidth      = 232
	pageTopDragHeight = 36
)

// shellTint fades the window chrome over the native vibrancy material and
// keeps it opaque where the platform draws no material.
func shellTint(c *ui.Context, color ui.Color, alpha float32) ui.Color {
	if !c.Vibrancy() {
		return color
	}
	return color.Alpha(alpha)
}

func (a *app) view(c *ui.Context) {
	theme := *c.Theme()
	p := paletteFor(&theme)
	theme.Background = p.bg
	theme.Surface = p.panelAlt
	theme.SurfaceHover = p.hover.Over(p.panelAlt)
	theme.SurfacePressed = p.line.Over(p.panelAlt)
	theme.Border = p.line
	theme.Text = p.text
	theme.TextMuted = p.muted
	theme.Accent = p.accent
	theme.AccentText = p.onAccent
	theme.AccentHover = p.accent.Mix(ui.Hex("#ffffff"), 0.14)
	theme.AccentPressed = p.accent.Mix(ui.Hex("#000000"), 0.12)
	theme.Selection = p.accent.Alpha(0.26)
	theme.Focus = p.accent.Alpha(0.55)
	theme.Scrollbar = p.muted.Alpha(0.35)
	theme.Radius = 8
	if a.page == pageSong {
		theme.Background = songInk
		theme.Surface = ui.RGBA(255, 255, 255, 0.06)
		theme.SurfaceHover = ui.RGBA(255, 255, 255, 0.1)
		theme.Border = ui.RGBA(255, 255, 255, 0.12)
		theme.Text = songWhite
		theme.TextMuted = ui.RGBA(255, 255, 255, 0.56)
		theme.Accent = songWhite
		theme.AccentText = songInk
		theme.AccentHover = ui.Hex("#f6f1f3")
		theme.AccentPressed = ui.Hex("#eadfe4")
		theme.Scrollbar = ui.RGBA(255, 255, 255, 0.18)
	}
	c.SetTheme(&theme)
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	}

	c.OnShortcut(ui.Cmd, ui.KeyF, func() {
		a.focusSearch = true
		a.refresh()
	})
	c.OnShortcut(ui.Cmd, ui.KeyLeft, func() {
		if a.pl != nil {
			a.pl.prev()
		}
	})
	c.OnShortcut(ui.Cmd, ui.KeyRight, func() {
		if a.pl != nil {
			a.pl.next()
		}
	})
	c.OnShortcut(0, ui.KeySpace, func() {
		if a.pl != nil {
			a.pl.toggle()
		}
	})
	c.OnShortcut(ui.Cmd, ui.KeyQ, mygo.App.Quit)
	c.OnShortcut(ui.Cmd, ui.KeyW, func() {
		if a.win != nil {
			a.win.Close()
		}
	})
	if a.page == pageSong {
		c.OnShortcut(0, ui.KeyEscape, func() {
			a.navTo(a.songBackTarget())
		})
	}

	rootBg := p.bg
	if c.Vibrancy() {
		rootBg = ui.Transparent
	}
	ui.Column(c).Fill().Background(rootBg).Children(func() {
		if a.page == pageSong {
			a.songPage(c, p)
			return
		}
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			a.sidebar(c, p)
			ui.Column(c).Grow(1).MinWidth(0).
				Background(shellTint(c, p.bg, 0.9)).Children(func() {
				ui.Row(c).Height(pageTopDragHeight).FillWidth().Shrink(0).DragWindow()
				a.pageContent(c, p)
			})
		})
		a.playerBar(c, p)
	})

	a.accountModal(c, p)
	a.localPlaylistModals(c, p)

	if a.toast != "" {
		c.Toast(a.toast)
		a.toast = ""
	}
	if a.pl != nil && a.pl.playing {
		a.pl.syncNowPlaying(nil)
		c.After(500 * time.Millisecond)
	}
}

func (a *app) sidebar(c *ui.Context, p palette) {
	ui.Column(c).Width(sidebarWidth).FillHeight().Padding(0, 14, 16, 14).Gap(6).
		Background(shellTint(c, p.sidebar, 0.58)).BorderWidth(0, 0, 0, 1).
		BorderColor(p.line).Children(func() {
		// The column's gap completes the inset below this drag region.
		ui.Row(c).Height(sidebarTopInset - 6).FillWidth().DragWindow()
		ui.Row(c).Height(42).Gap(11).AlignItems(ui.Center).DragWindow().Children(func() {
			logoMark(c, p)
			ui.Column(c).Gap(1).Children(func() {
				ui.Text(c, "MiguWave").FontSize(16.5).FontWeight(720).LetterSpacing(-0.25)
				ui.Text(c, "MIGU MUSIC").FontSize(8).FontWeight(700).LetterSpacing(1.5).TextColor(p.faint)
			})
		})

		search := ui.SearchField(c.Key("side-search"), &a.searchInput).FillWidth().
			Placeholder("搜索歌曲、歌手、专辑").Label("搜索").Margin(6, 0, 10, 0)
		if a.focusSearch {
			search.Focus()
			a.focusSearch = false
		}
		if search.Submitted() {
			a.runSearch(a.searchInput)
		}

		ui.Text(c, "浏览").FontSize(11).FontWeight(620).LetterSpacing(0.6).
			TextColor(p.faint).Margin(10, 6, 4, 2)
		a.navItem(c, p, iconHome, "发现", pageHome)
		a.navItem(c, p, iconRank, "排行榜", pageRank)
		a.navItem(c, p, iconPlaylist, "歌单广场", pagePlaylists)
		a.navItem(c, p, iconMic, "歌手", pageSingers)
		a.navItem(c, p, iconRadio, "电台", pageRadios)

		ui.Text(c, "资料库").FontSize(11).FontWeight(620).LetterSpacing(0.6).
			TextColor(p.faint).Margin(10, 16, 4, 2)
		a.navItem(c, p, iconHeart, "我的收藏", pageFavorites)
		a.navItem(c, p, iconPlus, "自建歌单", pageLocalPlaylists)
		a.navItem(c, p, iconPlaylist, "收藏歌单", pageSavedPlaylists)
		a.navItem(c, p, iconClock, "最近播放", pageRecent)
		if a.page == pageSong {
			a.navItem(c, p, iconLyrics, "歌曲详情", pageSong)
		}

		ui.Spacer(c)
		a.navItem(c, p, iconSettings, "设置", pageSettings)
		ui.Text(c, "非咪咕官方客户端 · 曲库来自 music.migu.cn").FontSize(8.5).
			TextColor(p.faint).Margin(4, 2, 0, 0).LineHeight(1.4)
	})
}

func statusDot(p palette, ok bool) ui.Color {
	if ok {
		return ui.Hex("#25b86b")
	}
	return p.faint
}

func logoMark(c *ui.Context, p palette) {
	ui.Box(c).Size(32, 32).Radius(9).Gradient(p.accent, p.accent2, 35).
		Shadow(0, 4, 12, -4, p.accent.Alpha(0.35)).Draw(func(pt *ui.Painter, r ui.Rect) {
		mid := r.Y + r.H/2
		bar := float32(3)
		for i, h := range []float32{8, 16, 11, 6} {
			x := r.X + 7.5 + float32(i)*4.7
			pt.Fill(ui.Rect{X: x, Y: mid - h/2, W: bar, H: h},
				p.onAccent.Alpha(0.92), bar/2)
		}
	})
}

func (a *app) navItem(c *ui.Context, p palette, ic *ui.SVG, label string, target pageKind) {
	selected := a.navActive(target)
	b := ui.ButtonBase(c).Key("nav-" + label).FillWidth().Height(34).PaddingX(10).Radius(8).
		Gap(10).AlignItems(ui.Center).Justify(ui.Start).FocusRing(false).Label(label)
	bg := ui.Transparent
	switch {
	case selected:
		bg = p.line
	case b.Hovered():
		bg = p.hover
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 110 * time.Millisecond}))
	color := p.muted
	if selected {
		color = p.accent
	}
	b.Children(func() {
		ui.Icon(c, ic).Size(17, 17).TextColor(color)
		ui.Text(c, label).FontSize(13.5).FontWeight(550).TextColor(p.text)
	})
	if b.Clicked() {
		a.navTo(target)
	}
}

func (a *app) navActive(target pageKind) bool {
	switch target {
	case pagePlaylists:
		return a.page == pagePlaylists || a.page == pagePlaylistDetail
	case pageSingers:
		return a.page == pageSingers || a.page == pageSingerDetail
	case pageRadios:
		return a.page == pageRadios || a.page == pageRadioDetail
	case pageLocalPlaylists:
		return a.page == pageLocalPlaylists || a.page == pageLocalPlaylistDetail
	default:
		return a.page == target
	}
}

func (a *app) pageContent(c *ui.Context, p palette) {
	switch a.page {
	case pageHome:
		a.homePage(c, p)
	case pageRank:
		a.rankPage(c, p)
	case pageFavorites:
		a.libraryPage(c, p, "我的收藏", "留在身边的咪咕好歌", a.favorites, iconHeart)
	case pageRecent:
		a.libraryPage(c, p, "最近播放", "回到刚刚听过的音乐", a.recent, iconClock)
	case pageSearch:
		a.searchPage(c, p)
	case pageSong:
		a.songPage(c, p)
	case pagePlaylists:
		a.playlistsPage(c, p)
	case pagePlaylistDetail:
		a.playlistDetailPage(c, p)
	case pageSavedPlaylists:
		a.savedPlaylistsPage(c, p)
	case pageSingers:
		a.singersPage(c, p)
	case pageSingerDetail:
		a.singerDetailPage(c, p)
	case pageRadios:
		a.radiosPage(c, p)
	case pageRadioDetail:
		a.radioDetailPage(c, p)
	case pageLocalPlaylists:
		a.localPlaylistsPage(c, p)
	case pageLocalPlaylistDetail:
		a.localPlaylistDetailPage(c, p)
	case pageSettings:
		a.settingsPage(c, p)
	default:
		a.homePage(c, p)
	}
}

func (a *app) playerBar(c *ui.Context, p palette) {
	pl := a.pl
	if pl == nil {
		return
	}
	cur := pl.current
	ui.Row(c).Height(88).PaddingX(22).Gap(18).Background(shellTint(c, p.bg, 0.88)).
		BorderWidth(1, 0, 0, 0).BorderColor(p.line).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Width(310).Gap(6).AlignItems(ui.Center).Children(func() {
			info := ui.ButtonBase(c).Key("player-info").Grow(1).MinWidth(0).Padding(2, 4).Radius(8).
				Gap(10).AlignItems(ui.Center).FocusRing(false).Label("查看歌曲详情")
			if info.Hovered() && pl.hasCurrent {
				info.Background(p.hover)
			}
			info.Children(func() {
				ui.Box(c).Size(50, 50).Radius(8).Children(func() {
					title := "MiguWave"
					if pl.hasCurrent {
						title = cur.Title
					}
					a.cover(c, cur.Cover, title, 8)
				})
				ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(3).Children(func() {
					title := "还没有播放"
					if pl.hasCurrent {
						title = cur.Title
					}
					titleColor := p.text
					if info.Hovered() && pl.hasCurrent {
						titleColor = p.accent
					}
					ui.Text(c, title).FontSize(13).FontWeight(600).SingleLine().TextColor(titleColor)
					sub := "从发现页挑一首咪咕歌曲"
					if pl.hasCurrent {
						sub = cur.Subtitle()
					}
					if pl.loading {
						if cur.NeedsResolve() {
							sub = "正在解析咪咕音频…"
						} else {
							sub = fmt.Sprintf("正在载入 %d%%", int(pl.progress*100))
						}
					}
					if pl.err != "" {
						sub = pl.err
					}
					ui.Text(c, sub).FontSize(11.5).TextColor(metaColor(p, pl)).SingleLine()
				})
			})
			if info.Clicked() && pl.hasCurrent {
				a.openSong(cur, pl.queue, pl.at)
			}
			if pl.hasCurrent {
				fav := a.isFavorite(cur.ID)
				ic := iconHeart
				if fav {
					ic = iconHeartFill
				}
				b := a.iconButton(c, "player-fav", ic, favoriteLabel(fav), fav)
				if b.Clicked() {
					a.toggleFavorite(cur)
				}
			}
		})

		ui.Column(c).Grow(1).MinWidth(0).Gap(4).AlignItems(ui.Stretch).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.Center).Gap(8).Children(func() {
				shuffle := a.iconButton(c, "shuffle", iconShuffle, "随机播放", pl.shuffle)
				if shuffle.Clicked() {
					pl.shuffle = !pl.shuffle
					a.refresh()
				}
				prev := a.iconButton(c, "prev", iconPrev, "上一首", false)
				if prev.Clicked() {
					pl.prev()
				}
				play := ui.ButtonBase(c).Key("play").Size(38, 38).Center().Radius(999).
					FocusRing(false).Label(playLabel(pl)).Background(p.accent)
				if play.Hovered() {
					play.Background(p.accent.Mix(ui.RGB(255, 255, 255), 0.13))
				}
				play.Children(func() {
					if pl.loading && pl.out == nil {
						ui.Spinner(c).FontSize(16).TextColor(p.onAccent)
					} else {
						ic := iconPlay
						if pl.playing {
							ic = iconPause
						}
						ui.Icon(c, ic).Size(17, 17).TextColor(p.onAccent)
					}
				})
				if play.Clicked() {
					pl.toggle()
				}
				next := a.iconButton(c, "next", iconNext, "下一首", false)
				if next.Clicked() {
					pl.next()
				}
				rep := iconRepeat
				if pl.repeat == repeatOne {
					rep = iconRepeatOne
				}
				repeat := a.iconButton(c, "repeat", rep, repeatLabel(pl.repeat), pl.repeat != repeatOff)
				if repeat.Clicked() {
					pl.repeat = (pl.repeat + 1) % 3
					a.refresh()
				}
			})
			ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
				ui.Text(c, formatClock(pl.position())).Width(38).FontSize(10).
					FontFeatures("tnum").TextColor(p.faint).TextAlign(ui.End)
				a.progressControl(c, p).Grow(1)
				label := formatClock(pl.duration)
				if !pl.hasCurrent {
					label = "--:--"
				}
				ui.Text(c, label).Width(38).FontSize(10).FontFeatures("tnum").
					TextColor(p.faint)
			})
		})

		ui.Row(c).Width(200).Justify(ui.End).Gap(7).AlignItems(ui.Center).Children(func() {
			ic := iconVolume
			if a.volume <= 0.01 {
				ic = iconVolumeOff
			}
			ui.Icon(c, ic).Size(15, 15).TextColor(p.muted)
			vol := ui.Slider(c, &a.volume, 0, 1).Width(90).Label("音量")
			if vol.Changed() {
				pl.setVolume(a.volume)
				a.save()
			}
			if pl.hasCurrent && len(pl.queue) > 1 {
				ui.Textf(c, "%d/%d", pl.at+1, len(pl.queue)).FontSize(10.5).
					FontFeatures("tnum").TextColor(p.faint)
			}
		})
	})
}

func metaColor(p palette, pl *player) ui.Color {
	if pl.err != "" {
		return ui.Hex("#ef6b6b")
	}
	return p.muted
}

func (a *app) progressControl(c *ui.Context, p palette) ui.Element {
	pl := a.pl
	if !pl.hasCurrent {
		ui.Box(c).Height(18).Draw(func(pt *ui.Painter, r ui.Rect) {
			pt.Fill(ui.Rect{X: r.X, Y: r.Y + r.H/2 - 2, W: r.W, H: 4}, p.line, 2)
		})
		return ui.Box(c)
	}
	dur := max(pl.duration.Seconds(), 1)
	if !a.scrubActive && !a.scrubHeld {
		a.scrub = pl.position().Seconds()
	}
	s := ui.SliderBase(c, &a.scrub, 0, dur).Height(18).FocusRing(false).
		Cursor(ui.CursorPointer).Label("播放进度")
	s.HandleInput(func(ev ui.InputEvent) bool {
		switch ev.Kind {
		case ui.InputPointerDown:
			a.scrubHeld = ev.Button == 0
		case ui.InputPointerUp:
			a.scrubHeld = false
		}
		return false
	})
	pressed := a.scrubHeld
	switch {
	case pressed:
		a.scrubActive = true
	case a.scrubActive:
		a.scrubActive = false
		pl.seek(seconds(min(a.scrub, max(0, dur-1))))
	}
	if s.Changed() {
		pl.seek(seconds(min(a.scrub, max(0, dur-1))))
	}
	hot := s.Hovered() || pressed
	thick := float32(4)
	if hot {
		thick = 6
	}
	frac := float32(max(0, min(1, a.scrub/dur)))
	s.Draw(func(pt *ui.Painter, r ui.Rect) {
		y := r.Y + (r.H-thick)/2
		head := r.X + r.W*frac
		pt.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: thick}, p.line, thick/2)
		pt.Fill(ui.Rect{X: r.X, Y: y, W: max(thick, head-r.X), H: thick}, p.accent, thick/2)
		if hot {
			pt.Shadow(ui.Rect{X: head, Y: r.Y + r.H/2}, 0, 0, 0, 9, 3, p.accent.Alpha(0.5))
		}
	})
	return s
}

func seconds(v float64) time.Duration {
	return time.Duration(v * float64(time.Second))
}

func (a *app) trackRow(c *ui.Context, p palette, tr Track, tracks []Track, i int) {
	pl := a.pl
	playing := pl.hasCurrent && pl.current.ID == tr.ID
	row := ui.ButtonBase(c).Key("track-" + tr.ID).Height(54).PaddingX(11).Gap(12).
		Radius(10).AlignItems(ui.Center).FocusRing(false).Label(tr.Title)
	bg := ui.Transparent
	switch {
	case playing:
		bg = p.accent.Alpha(0.1)
	case row.Hovered():
		bg = p.hover
	}
	row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 100 * time.Millisecond}))
	row.Children(func() {
		ui.Box(c).Width(28).Center().Children(func() {
			switch {
			case playing:
				a.playingBars(c, p, pl.playing)
			case row.Hovered():
				ui.Icon(c, iconPlay).Size(12, 12).TextColor(p.text)
			default:
				ui.Textf(c, "%d", i+1).FontSize(12).FontFeatures("tnum").TextColor(p.faint)
			}
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
			title := p.text
			if playing {
				title = p.accent
			}
			ui.Text(c, tr.Title).FontSize(13.5).FontWeight(560).SingleLine().TextColor(title)
			artist := tr.Artist
			if artist == "" {
				artist = tr.Album
			}
			if artist != "" {
				ui.Text(c, artist).FontSize(11.5).TextColor(p.muted).SingleLine()
			}
		})
		if row.Hovered() {
			b := a.iconButton(c, "track-info-"+tr.ID, iconLyrics, "查看歌词与详情", false)
			if b.Clicked() {
				a.openSong(tr, tracks, i)
			}
		}
		fav := a.isFavorite(tr.ID)
		if fav || row.Hovered() {
			ic := iconHeart
			if fav {
				ic = iconHeartFill
			}
			b := a.iconButton(c, "track-fav-"+tr.ID, ic, favoriteLabel(fav), fav)
			if b.Clicked() {
				a.toggleFavorite(tr)
			}
		}
		if tr.DurationSec > 0 {
			ui.Text(c, formatClock(tr.Duration())).FontSize(11.5).FontFeatures("tnum").
				TextColor(p.faint).Width(42).TextAlign(ui.End)
		}
	})
	if row.Clicked() {
		a.playTracks(tracks, i)
	}
}

func (a *app) playingBars(c *ui.Context, p palette, playing bool) {
	ui.Box(c).Size(16, 15).Draw(func(pt *ui.Painter, r ui.Rect) {
		now := float64(c.Now().UnixMilli()) / 1000
		for i := 0; i < 3; i++ {
			v := 0.35
			if playing {
				v = 0.35 + 0.6*((math.Sin(now*6+float64(i)*1.9)+1)/2)
			}
			h := 3 + float32(v)*9
			pt.Fill(ui.Rect{X: r.X + 1 + float32(i)*5, Y: r.Y + r.H - h, W: 3, H: h},
				p.accent, 1.5)
		}
		if playing {
			pt.After(140 * time.Millisecond)
		}
	})
}

func (a *app) iconButton(c *ui.Context, key any, ic *ui.SVG, label string, active bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Key(key).Size(32, 32).Center().Radius(999).
		FocusRing(false).Label(label)
	bg := ui.Transparent
	color := t.TextMuted
	switch {
	case active:
		color = t.Accent
	case b.Hovered():
		bg = t.SurfaceHover
		color = t.Text
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 100 * time.Millisecond}))
	b.Children(func() {
		ui.Icon(c, ic).Size(16, 16).TextColor(color)
	})
	return b
}

func favoriteLabel(fav bool) string {
	if fav {
		return "取消收藏"
	}
	return "收藏"
}

func playLabel(pl *player) string {
	if pl.loading {
		return "正在载入"
	}
	if pl.playing {
		return "暂停"
	}
	return "播放"
}

func repeatLabel(m repeatMode) string {
	switch m {
	case repeatAll:
		return "列表循环"
	case repeatOne:
		return "单曲循环"
	default:
		return "不循环"
	}
}
