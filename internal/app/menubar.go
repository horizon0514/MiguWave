package app

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"freewave/internal/lyrics"
	"freewave/internal/migu"
)

const menuBarActivationGrace = 800 * time.Millisecond

func (a *app) attachWindow(win *mygo.Window) {
	a.win = win
	win.OnClose(a.onMainWindowClose)
}

func (a *app) setupMenuBar() {
	tray, err := mygo.NewTray(mygo.TrayOptions{
		ID:             "player",
		Icon:           menuBarIcon(),
		IconIsTemplate: true,
		ToolTip:        "MiguWave",
	})
	if err != nil {
		log.Printf("menu bar: %v", err)
		return
	}
	a.tray = tray

	if runtime.GOOS == "linux" {
		tray.SetMenu(a.menuBarMenu())
		return
	}
	tray.OnClick(a.toggleMenuBarPanel)
	tray.OnRightClick(func() {
		a.noteMenuBarUse()
		a.closeMenuBarPanel()
		tray.SetMenu(a.menuBarMenu())
		defer tray.SetMenu(nil)
		tray.PopUpMenu()
	})
}

func (a *app) noteMenuBarUse() {
	a.menuBarUsedAt = time.Now()
}

func (a *app) shouldShowMainOnActivate(hasVisibleWindows, dockVisible bool, now time.Time) bool {
	if hasVisibleWindows || !dockVisible || a.menuBarPanel != nil {
		return false
	}
	if !a.menuBarUsedAt.IsZero() && now.Sub(a.menuBarUsedAt) < menuBarActivationGrace {
		return false
	}
	return true
}

func (a *app) handleAppActivate(hasVisibleWindows bool) {
	dockVisible := mygo.App.ActivationPolicy() == mygo.ActivationPolicyRegular
	if a.shouldShowMainOnActivate(hasVisibleWindows, dockVisible, time.Now()) {
		a.showMainWindow()
	}
}

func (a *app) menuBarMenu() *mygo.Menu {
	items := []*mygo.MenuItem{
		{Label: "打开完整窗口", Click: func(*mygo.MenuItem, *mygo.Window) { a.showMainWindow() }},
	}
	if a.pl != nil && a.pl.hasCurrent {
		items = append(items,
			mygo.Separator(),
			&mygo.MenuItem{Label: playLabel(a.pl), Click: func(*mygo.MenuItem, *mygo.Window) {
				a.pl.toggle()
				a.refresh()
			}},
			&mygo.MenuItem{Label: "上一首", Click: func(*mygo.MenuItem, *mygo.Window) { a.pl.prev() }},
			&mygo.MenuItem{Label: "下一首", Click: func(*mygo.MenuItem, *mygo.Window) { a.pl.next() }},
		)
	}
	items = append(items,
		mygo.Separator(),
		&mygo.MenuItem{Label: "设置…", Click: func(*mygo.MenuItem, *mygo.Window) { a.showSettings() }},
		&mygo.MenuItem{Label: "咪咕账号…", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.showSettings()
			a.openAccount()
		}},
		mygo.Separator(),
		&mygo.MenuItem{Role: mygo.RoleQuit},
	)
	return mygo.NewMenu(items)
}

func (a *app) toggleMenuBarPanel() {
	a.noteMenuBarUse()
	if a.menuBarPanel != nil {
		a.closeMenuBarPanel()
		return
	}
	if a.tray == nil {
		return
	}
	panel := mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:         a.tray,
		Placement:    mygo.PlacementBottom,
		Width:        420,
		Height:       536,
		Focusable:    true,
		Shadow:       true,
		Popover:      true,
		CornerRadius: 14,
		Content:      ui.View(a.menuBarView),
	})
	a.menuBarPanel = panel
	panel.OnClosed(func() {
		a.noteMenuBarUse()
		if a.menuBarPanel == panel {
			a.menuBarPanel = nil
		}
	})
}

func (a *app) closeMenuBarPanel() {
	if a.menuBarPanel == nil {
		return
	}
	panel := a.menuBarPanel
	a.menuBarPanel = nil
	panel.Close()
}

func (a *app) onMainWindowClose(e *mygo.CloseEvent) {
	if a.quitting {
		return
	}
	e.PreventDefault()
	if a.closeAction == closeActionQuit {
		mygo.App.Quit()
		return
	}
	a.hideMainWindow()
}

func (a *app) showMainWindow() {
	a.closeMenuBarPanel()
	a.setDockVisible(true)
	mygo.App.Show()
	if a.win != nil {
		a.win.Show()
	}
}

func (a *app) hideMainWindow() {
	if a.win != nil {
		a.win.Hide()
	}
	a.setDockVisible(false)
}

func (a *app) setDockVisible(visible bool) {
	if visible {
		mygo.App.Dock.Show()
		return
	}
	mygo.App.Dock.Hide()
}

func (a *app) showSettings() {
	a.showMainWindow()
	a.navTo(pageSettings)
}

func (a *app) menuBarView(c *ui.Context) {
	c.OnShortcut(ui.Cmd, ui.KeyQ, mygo.App.Quit)
	c.OnShortcut(ui.Cmd, ui.KeyW, a.closeMenuBarPanel)

	theme := *c.Theme()
	theme.Dark = true
	theme.Background = ui.Transparent
	theme.Surface = ui.RGBA(255, 255, 255, 0.08)
	theme.SurfaceHover = ui.RGBA(255, 255, 255, 0.12)
	theme.Border = ui.RGBA(255, 255, 255, 0.12)
	theme.Text = ui.RGB(255, 255, 255)
	theme.TextMuted = ui.RGBA(255, 255, 255, 0.6)
	theme.Accent = ui.RGBA(255, 255, 255, 0.92)
	theme.AccentText = songInk
	theme.Radius = 12
	c.SetTheme(&theme)

	pl := a.pl
	cover := ""
	if pl != nil && pl.hasCurrent {
		cur := pl.current
		a.ensureMenuBarDetail(cur)
		cover = firstNonEmpty(a.menuDetail.Cover, a.hqCover(cur))
		a.requestArt(cover)
	}
	tint := ui.Hex("#8f4a63")
	if a.artTints != nil && cover != "" {
		if v, ok := a.artTints[cover]; ok {
			tint = v
		}
	}
	p := songPalette(tint)

	root := ui.Box(c).Fill().Clip()
	root.Children(func() {
		a.menuBarBackdrop(c, cover, tint)
		ui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).
			Padding(14, 22, 16, 22).AlignItems(ui.Center).Children(func() {
			a.menuBarTopBar(c, p, pl)
			if pl == nil || !pl.hasCurrent {
				a.menuBarEmpty(c)
				return
			}
			a.menuBarNowPlaying(c, p, pl, cover)
		})
	})

	c.After(350 * time.Millisecond)
}

func (a *app) menuBarBackdrop(c *ui.Context, cover string, tint ui.Color) {
	ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough().
		Draw(func(pt *ui.Painter, r ui.Rect) {
			top, foot := songBackdropColors(tint)
			top = top.Mix(songInk, 0.46)
			foot = foot.Mix(songInk, 0.32)
			pt.FillGradient(r, ui.LinearGradient{Angle: 165, Oklab: true, Stops: []ui.GradientStop{
				{Color: top, At: 0},
				{Color: foot, At: 1},
			}}, 0)
			if a.artBackdrops != nil && cover != "" {
				if blurred := a.artBackdrops[cover]; blurred != nil {
					pt.ImageRegion(blurred, ui.Rect{W: 48, H: 48}, coverRect(r, 48, 48),
						ui.ImageOptions{Tint: songWhite.Alpha(0.3)})
				}
			}
			glow := tint.Mix(songWhite, 0.3)
			pt.FillRadialGradient(r, ui.RadialGradient{
				Center: ui.Point{X: 0.5, Y: 0.16},
				Radius: max(r.W, r.H) * 0.72,
				Stops: []ui.GradientStop{
					{Color: glow.Alpha(0.26), At: 0},
					{Color: glow.Alpha(0), At: 1},
				},
			}, 0)
			pt.FillGradient(r, ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
				{Color: ui.RGBA(0, 0, 0, 0.16), At: 0},
				{Color: ui.RGBA(0, 0, 0, 0.1), At: 0.34},
				{Color: ui.RGBA(0, 0, 0, 0.5), At: 0.78},
				{Color: ui.RGBA(0, 0, 0, 0.68), At: 1},
			}}, 0)
		})
}

func (a *app) menuBarTopBar(c *ui.Context, p songColors, pl *player) {
	ui.Row(c).Height(30).FillWidth().AlignItems(ui.Center).Children(func() {
		if pl != nil && pl.hasCurrent {
			status := "已暂停"
			if pl.loading {
				status = "正在载入"
			} else if pl.playing {
				status = "正在播放"
			}
			dotColor := songWhite.Alpha(0.32)
			if pl.playing {
				dotColor = p.tint.Mix(songWhite, 0.42)
			}
			ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
				dot := ui.Box(c).Size(6, 6).Radius(999).Background(dotColor)
				if pl.playing {
					dot.Shadow(0, 0, 8, 0, dotColor.Alpha(0.7))
				}
				ui.Text(c, status).FontSize(10).FontWeight(620).LetterSpacing(0.6).
					TextColor(songWhite.Alpha(0.54))
			})
		}
		ui.Spacer(c)
		ui.Row(c).Gap(2).AlignItems(ui.Center).Children(func() {
			open := a.iconButton(c, "menu-open-main", iconExternal, "打开完整窗口", false)
			if open.Clicked() {
				a.showMainWindow()
			}
			settings := a.iconButton(c, "menu-settings", iconSettings, "设置", false)
			if settings.Clicked() {
				a.showSettings()
			}
		})
	})
}

func (a *app) menuBarNowPlaying(c *ui.Context, p songColors, pl *player, cover string) {
	cur := pl.current

	ui.Box(c).Size(184, 184).Radius(20).Clip().Margin(10, 0, 0, 0).
		Shadow(0, 18, 40, -14, ui.RGBA(0, 0, 0, 0.62)).Children(func() {
		a.cover(c, cover, displayTitle(cur.Title), 20)
	})

	ui.Column(c).FillWidth().AlignItems(ui.Center).Margin(14, 0, 0, 0).Gap(3).Children(func() {
		ui.Text(c, displayTitle(cur.Title)).FillWidth().FontSize(20).FontWeight(740).
			LetterSpacing(-0.2).SingleLine().TextAlign(ui.Center)
		meta := displayArtist(cur.Artist)
		if album := strings.TrimSpace(cur.Album); album != "" {
			meta += " · " + album
		}
		ui.Text(c, meta).FillWidth().FontSize(11.5).TextColor(songWhite.Alpha(0.5)).
			SingleLine().TextAlign(ui.Center)
	})

	a.menuBarLyrics(c, pl.position())

	ui.Column(c).FillWidth().Margin(10, 0, 0, 0).Gap(3).Children(func() {
		a.progressControl(c, p.palette)
		ui.Row(c).FillWidth().Children(func() {
			ui.Text(c, formatClock(pl.position())).FontSize(10).FontFeatures("tnum").
				TextColor(songWhite.Alpha(0.4))
			ui.Spacer(c)
			label := "--:--"
			if pl.duration > 0 {
				label = formatClock(pl.duration)
			}
			ui.Text(c, label).FontSize(10).FontFeatures("tnum").
				TextColor(songWhite.Alpha(0.4))
		})
	})

	ui.Row(c).FillWidth().Height(50).Justify(ui.Center).AlignItems(ui.Center).
		Gap(18).Margin(3, 0, 0, 0).Children(func() {
		a.menuBarTransportButton(c, "menu-prev", iconPrev, "上一首", pl.prev)
		play := ui.ButtonBase(c).Key("menu-play").Size(46, 46).Center().Radius(999).
			FocusRing(false).Label(playLabel(pl)).Background(songWhite.Alpha(0.94)).
			Shadow(0, 10, 24, -8, ui.RGBA(0, 0, 0, 0.55))
		if play.Hovered() {
			play.Background(songWhite)
		}
		play.Children(func() {
			if pl.loading && pl.out == nil {
				ui.Spinner(c).FontSize(17).TextColor(songInk)
			} else {
				ic := iconPlay
				if pl.playing {
					ic = iconPause
				}
				ui.Icon(c, ic).Size(18, 18).TextColor(songInk)
			}
		})
		if play.Clicked() {
			pl.toggle()
		}
		a.menuBarTransportButton(c, "menu-next", iconNext, "下一首", pl.next)
	})
}

func (a *app) menuBarTransportButton(c *ui.Context, key any, ic *ui.SVG, label string, onClick func()) {
	t := c.Theme()
	b := ui.ButtonBase(c).Key(key).Size(36, 36).Center().Radius(999).
		FocusRing(false).Label(label)
	color := t.TextMuted
	if b.Hovered() {
		b.Background(t.SurfaceHover)
		color = t.Text
	}
	b.Children(func() {
		ui.Icon(c, ic).Size(19, 19).TextColor(color)
	})
	if b.Clicked() {
		onClick()
	}
}

func (a *app) menuBarEmpty(c *ui.Context) {
	ui.Column(c).Fill().Center().Gap(12).Children(func() {
		ui.Box(c).Size(64, 64).Radius(20).Background(songWhite.Alpha(0.07)).
			Border(1, songWhite.Alpha(0.1)).Center().
			Shadow(0, 12, 30, -12, ui.RGBA(0, 0, 0, 0.5)).Children(func() {
			ui.Icon(c, iconMusic).Size(26, 26).TextColor(songWhite.Alpha(0.72))
		})
		ui.Text(c, "还没有播放").FontSize(17).FontWeight(700)
		ui.Text(c, "在咪咕音乐里挑一首喜欢的歌").FontSize(11.5).
			TextColor(songWhite.Alpha(0.46))
	})
}

func (a *app) menuBarLyrics(c *ui.Context, position time.Duration) {
	ui.Box(c).FillWidth().Grow(1).MinHeight(0).Children(func() {
		switch {
		case a.menuDetailLoading:
			ui.Column(c).Fill().Center().Gap(8).Children(func() {
				ui.Spinner(c).FontSize(16).TextColor(songWhite.Alpha(0.6))
				ui.Text(c, "正在取回歌词…").FontSize(10.5).TextColor(songWhite.Alpha(0.42))
			})
		case len(a.menuDetail.Lyrics) == 0:
			message := "这首歌暂时没有歌词"
			if a.menuDetailErr != "" {
				message = "歌词暂时不可用"
			}
			ui.Column(c).Fill().Center().Gap(7).Children(func() {
				ui.Icon(c, iconLyrics).Size(22, 22).TextColor(songWhite.Alpha(0.3))
				ui.Text(c, message).FontSize(11).TextColor(songWhite.Alpha(0.5))
			})
		default:
			prev, current, next := menuBarLyricWindow(a.menuDetail, position)
			if current < 0 {
				ui.Column(c).Fill().Center().Gap(7).Children(func() {
					ui.Icon(c, iconLyrics).Size(22, 22).TextColor(songWhite.Alpha(0.3))
					ui.Text(c, "这首歌暂时没有歌词").FontSize(11).
						TextColor(songWhite.Alpha(0.5))
				})
				return
			}
			lines := a.menuDetail.Lyrics
			ui.Column(c).Fill().Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				for _, index := range prev {
					ui.Text(c, lines[index].Text).FillWidth().FontSize(12).
						TextColor(songWhite.Alpha(0.26)).SingleLine().
						TextAlign(ui.Center).Margin(0, 0, 5, 0)
				}
				a.menuBarCurrentLyric(c, position, current)
				for i, index := range next {
					size := float32(12.5)
					color := songWhite.Alpha(0.4)
					if i > 0 {
						size = 11.5
						color = songWhite.Alpha(0.22)
					}
					ui.Text(c, lines[index].Text).FillWidth().FontSize(size).
						TextColor(color).SingleLine().TextAlign(ui.Center).
						Margin(6, 0, 0, 0)
				}
			})
		}
	})
}

func (a *app) menuBarCurrentLyric(c *ui.Context, position time.Duration, index int) {
	line := a.menuDetail.Lyrics[index]
	live := a.pl != nil && a.pl.hasCurrent && a.pl.current.ID == a.menuDetail.Track.ID
	if live && line.Timed && !a.menuDetail.PlainLyrics && position >= line.At {
		nextAt := lyrics.LineEnd(a.menuDetail.Lyrics, index)
		for i := index + 1; i < len(a.menuDetail.Lyrics); i++ {
			if a.menuDetail.Lyrics[i].At > line.At {
				nextAt = a.menuDetail.Lyrics[i].At
				break
			}
		}
		words := lyrics.WordsFor(line, nextAt)
		if len(words) > 0 {
			spans := make([]ui.Span, 0, len(words))
			for _, word := range words {
				color := songWhite.Alpha(0.88)
				if position >= word.At {
					color = songWhite
				}
				spans = append(spans, ui.Span{Text: word.Text, Color: color})
			}
			ui.RichText(c, spans...).FillWidth().FontSize(22).FontWeight(700).
				LineHeight(1.36).TextAlign(ui.Center).MaxLines(2).Margin(1, 0, 1, 0)
			return
		}
	}
	ui.Text(c, line.Text).FillWidth().FontSize(22).FontWeight(700).LineHeight(1.36).
		TextAlign(ui.Center).MaxLines(2).Margin(1, 0, 1, 0)
}

func (a *app) ensureMenuBarDetail(track Track) {
	if track.ID == "" || a.menuDetailID == track.ID {
		return
	}
	a.menuDetailID = track.ID
	a.menuDetailErr = ""
	if detail, ok := a.songCache[track.ID]; ok {
		a.menuDetail = detail
		a.menuDetailLoading = false
		return
	}
	a.menuDetail = miguSongDetail{Track: track, Album: track.Album, Cover: track.Cover}
	if track.Source != "migu" {
		a.menuDetailLoading = false
		return
	}
	a.menuDetailGen++
	gen := a.menuDetailGen
	token := a.miguToken
	a.menuDetailLoading = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		detail, err := migu.FetchSongDetail(ctx, token, track)
		a.post(func() {
			if a.menuDetailGen != gen || a.menuDetailID != track.ID {
				return
			}
			a.menuDetailLoading = false
			if err != nil {
				a.menuDetailErr = friendlyError(err)
				a.refresh()
				return
			}
			a.menuDetail = detail
			if a.songCache == nil {
				a.songCache = map[string]miguSongDetail{}
			}
			a.songCache[track.ID] = detail
			a.refresh()
		})
	}()
}

func menuBarLyricPreview(detail miguSongDetail, position time.Duration) (string, string) {
	_, current, next := menuBarLyricWindow(detail, position)
	if current < 0 {
		return "这首歌暂时没有歌词", ""
	}
	if len(next) > 0 {
		return detail.Lyrics[current].Text, detail.Lyrics[next[0]].Text
	}
	return detail.Lyrics[current].Text, ""
}

func menuBarLyricWindow(detail miguSongDetail, position time.Duration) (prev []int, current int, next []int) {
	lines := detail.Lyrics
	current = -1
	if len(lines) == 0 {
		return nil, -1, nil
	}
	if detail.PlainLyrics {
		current = firstLyricLine(lines, 0)
	} else {
		current = lyrics.IndexAt(lines, position)
	}
	if current < 0 {
		current = firstLyricLine(lines, 0)
	} else if strings.TrimSpace(lines[current].Text) == "" {
		if index := firstLyricLine(lines, current); index >= 0 {
			current = index
		}
	}
	if current < 0 {
		return nil, -1, nil
	}
	for i := current - 1; i >= 0 && len(prev) < 2; i-- {
		if strings.TrimSpace(lines[i].Text) != "" {
			prev = append(prev, i)
		}
	}
	for i, j := 0, len(prev)-1; i < j; i, j = i+1, j-1 {
		prev[i], prev[j] = prev[j], prev[i]
	}
	for i := current + 1; i < len(lines) && len(next) < 2; i++ {
		if strings.TrimSpace(lines[i].Text) != "" {
			next = append(next, i)
		}
	}
	return prev, current, next
}

func firstLyricLine(lines []lyricLine, start int) int {
	for i := max(start, 0); i < len(lines); i++ {
		if strings.TrimSpace(lines[i].Text) != "" {
			return i
		}
	}
	return -1
}

func menuBarIcon() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	ink := color.NRGBA{A: 255}
	for _, bar := range []image.Rectangle{
		image.Rect(5, 12, 9, 23),
		image.Rect(11, 6, 15, 27),
		image.Rect(17, 9, 21, 24),
		image.Rect(23, 14, 27, 21),
	} {
		for y := bar.Min.Y; y < bar.Max.Y; y++ {
			for x := bar.Min.X; x < bar.Max.X; x++ {
				img.Set(x, y, ink)
			}
		}
	}
	var out bytes.Buffer
	_ = png.Encode(&out, img)
	return out.Bytes()
}
