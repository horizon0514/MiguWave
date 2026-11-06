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
		a.closeMenuBarPanel()
		tray.SetMenu(a.menuBarMenu())
		defer tray.SetMenu(nil)
		tray.PopUpMenu()
	})
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
	if a.menuBarPanel != nil {
		a.closeMenuBarPanel()
		return
	}
	if a.tray == nil {
		return
	}
	panel := mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:      a.tray,
		Placement: mygo.PlacementBottom,
		Width:     388,
		Height:    322,
		Focusable: true,
		Shadow:    true,
		Popover:   true,
		Content:   ui.View(a.menuBarView),
	})
	a.menuBarPanel = panel
	panel.OnClosed(func() {
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
	p := paletteFor(&theme)
	theme.Background = p.bg
	theme.Surface = p.panelAlt
	theme.SurfaceHover = p.hover.Over(p.panelAlt)
	theme.Border = p.line
	theme.Text = p.text
	theme.TextMuted = p.muted
	theme.Accent = p.accent
	theme.AccentText = p.onAccent
	theme.Radius = 10
	c.SetTheme(&theme)

	pl := a.pl
	root := ui.Column(c).Fill().Padding(16).Gap(12)
	if !c.Vibrancy() {
		root.Background(p.bg)
	}
	root.Children(func() {
		if pl == nil || !pl.hasCurrent {
			a.menuBarEmpty(c, p)
			return
		}

		cur := pl.current
		a.ensureMenuBarDetail(cur)
		cover := firstNonEmpty(a.menuDetail.Cover, a.hqCover(cur))
		a.requestArt(cover)

		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			art := ui.Box(c).Size(66, 66).Radius(12).Clip()
			art.Children(func() {
				a.cover(c, cover, cur.Title, 12)
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, cur.Title).FontSize(15).FontWeight(720).SingleLine()
				ui.Text(c, cur.Subtitle()).FontSize(11.5).TextColor(p.muted).SingleLine()
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(5, 5).Radius(999).Background(statusDot(p, pl.playing))
					status := "已暂停"
					if pl.loading {
						status = "正在载入"
					} else if pl.playing {
						status = "正在播放"
					}
					ui.Text(c, status).FontSize(10).TextColor(p.faint)
				})
			})
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

		duration := max(pl.duration.Seconds(), 1)
		position := pl.position()
		frac := float64(position) / float64(duration)
		if frac < 0 {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
		ui.Progress(c, frac).FillWidth().Height(3)
		ui.Row(c).Children(func() {
			ui.Text(c, formatClock(position)).FontSize(9.5).FontFeatures("tnum").TextColor(p.faint)
			ui.Spacer(c)
			ui.Text(c, formatClock(pl.duration)).FontSize(9.5).FontFeatures("tnum").TextColor(p.faint)
		})

		a.menuBarLyrics(c, p, position)

		ui.Row(c).FillWidth().Justify(ui.Center).Gap(10).Children(func() {
			prev := a.iconButton(c, "menu-prev", iconPrev, "上一首", false)
			if prev.Clicked() {
				pl.prev()
			}
			play := ui.ButtonBase(c).Key("menu-play").Size(42, 42).Center().Radius(999).
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
			next := a.iconButton(c, "menu-next", iconNext, "下一首", false)
			if next.Clicked() {
				pl.next()
			}
		})

	})

	c.After(350 * time.Millisecond)
}

func (a *app) menuBarEmpty(c *ui.Context, p palette) {
	ui.Column(c).Fill().Center().Gap(10).Children(func() {
		ui.Box(c).Size(48, 48).Radius(15).Background(p.accent.Alpha(0.12)).Center().Children(func() {
			ui.Icon(c, iconMusic).Size(22, 22).TextColor(p.accent)
		})
		ui.Text(c, "还没有播放").FontSize(16).FontWeight(690)
		ui.Text(c, "打开完整窗口，挑一首咪咕音乐。").FontSize(11.5).TextColor(p.muted)
		a.menuBarOpenButton(c, p)
	})
}

func (a *app) menuBarLyrics(c *ui.Context, p palette, position time.Duration) {
	ui.Box(c).FillWidth().Height(112).Padding(15, 14).Radius(14).
		Background(p.panel).Border(1, p.line).Children(func() {
		switch {
		case a.menuDetailLoading:
			ui.Column(c).Fill().Center().Gap(8).Children(func() {
				ui.Spinner(c).FontSize(16).TextColor(p.accent)
				ui.Text(c, "正在取回歌词…").FontSize(10.5).TextColor(p.muted)
			})
		case len(a.menuDetail.Lyrics) == 0:
			message := "这首歌暂时没有歌词"
			if a.menuDetailErr != "" {
				message = "歌词暂时不可用"
			}
			ui.Column(c).Fill().Center().Gap(7).Children(func() {
				ui.Icon(c, iconLyrics).Size(22, 22).TextColor(p.faint)
				ui.Text(c, message).FontSize(11).TextColor(p.muted)
			})
		default:
			current, next := menuBarLyricPreview(a.menuDetail, position)
			ui.Column(c).Gap(7).Children(func() {
				ui.Text(c, current).FontSize(16).FontWeight(680).LineHeight(1.35).
					MaxLines(2)
				if next != "" {
					ui.Text(c, next).FontSize(11).TextColor(p.muted).SingleLine()
				}
			})
		}
	})
}

func (a *app) menuBarOpenButton(c *ui.Context, p palette) {
	b := ui.ButtonBase(c).Key("menu-open-main").FillWidth().Height(28).Radius(9).
		Gap(6).AlignItems(ui.Center).Justify(ui.Center).FocusRing(false).
		Label("打开完整窗口")
	if b.Hovered() {
		b.Background(p.hover.Over(p.bg))
	}
	b.Children(func() {
		ui.Icon(c, iconExternal).Size(12, 12).TextColor(p.faint)
		ui.Text(c, "打开完整窗口").FontSize(11).TextColor(p.muted)
	})
	if b.Clicked() {
		a.showMainWindow()
	}
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
	lines := detail.Lyrics
	if len(lines) == 0 {
		return "这首歌暂时没有歌词", ""
	}
	index := -1
	if detail.PlainLyrics {
		index = firstLyricLine(lines, 0)
	} else {
		index = lyrics.IndexAt(lines, position)
		if index < 0 {
			index = firstLyricLine(lines, 0)
		}
	}
	if index < 0 {
		return "这首歌暂时没有歌词", ""
	}
	next := firstLyricLine(lines, index+1)
	current := ""
	if index >= 0 && index < len(lines) {
		current = lines[index].Text
	}
	if next >= 0 {
		return current, lines[next].Text
	}
	return current, ""
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
