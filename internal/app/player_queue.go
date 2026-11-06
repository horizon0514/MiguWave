package app

import (
	"time"

	"github.com/egoist/mygo/ui"
)

func (a *app) playerQueueButton(c *ui.Context, p palette) {
	enabled := a.pl != nil && len(a.pl.queue) > 0
	b := ui.ButtonBase(c).Key("player-queue").Size(32, 32).Center().Radius(999).
		FocusRing(false).Label("打开待播清单").Tooltip("待播清单")
	b.Disabled(!enabled)
	bg := ui.Transparent
	color := p.muted
	switch {
	case a.playerQueueOpen:
		bg, color = p.accent.Alpha(0.13), p.accent
	case b.Hovered() && enabled:
		bg, color = p.hover, p.text
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 100 * time.Millisecond}))
	b.Children(func() {
		ui.Icon(c, iconPlaylist).Size(16, 16).TextColor(color)
	})
	if b.Clicked() && enabled {
		a.playerQueueOpen = !a.playerQueueOpen
		if a.playerQueueOpen {
			a.playerQueueAt = -1
		}
	}

	ui.PopoverBase(c.Key("player-queue-popover"), b, &a.playerQueueOpen, func(panel ui.Element) {
		rows := 0
		if a.pl != nil {
			rows = len(a.pl.queue)
		}
		w, h := c.Size()
		height := min(float32(rows)*58+60, float32(460), max(float32(180), h-130))
		panel.AttachTo(b, ui.AnchorTopEnd, ui.AnchorBottomEnd).Margin(0, 0, 12, 0).
			Size(min(float32(348), w-28), height).Padding(8).Radius(14).
			Background(p.panelAlt).TextColor(p.text).Border(1, p.line).
			Shadow(0, 16, 44, -14, p.shadow).Label("待播清单")

		ui.Row(c).Height(42).PaddingX(6).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "待播清单").FontSize(14.5).FontWeight(700).TextColor(p.text)
			ui.Textf(c, "%d 首", rows).FontSize(11).FontFeatures("tnum").TextColor(p.faint)
			ui.Spacer(c)
			closeButton := ui.ButtonBase(c).Key("player-queue-close").Size(28, 28).
				Center().Radius(999).FocusRing(false).Label("关闭待播清单")
			if closeButton.Hovered() {
				closeButton.Background(p.hover)
			}
			closeButton.Children(func() {
				ui.Icon(c, iconX).Size(14, 14).TextColor(p.muted)
			})
			if closeButton.Clicked() {
				a.playerQueueOpen = false
			}
		})
		ui.Box(c).Height(1).FillWidth().Margin(1, 4, 5, 4).Background(p.line)
		if rows == 0 {
			ui.Box(c).Grow(1).Center().Children(func() {
				ui.Text(c, "播放队列是空的").FontSize(12.5).TextColor(p.faint)
			})
			return
		}
		a.playerQueueRows(c, p)
	})
}

func (a *app) playerQueueRows(c *ui.Context, p palette) ui.Element {
	pl := a.pl
	if pl.at >= 0 && pl.at != a.playerQueueAt {
		if a.playerQueueAt < 0 {
			a.playerQueueList.ScrollTo(pl.at, ui.Center)
		} else {
			a.playerQueueList.ScrollIntoView(pl.at)
		}
		a.playerQueueAt = pl.at
	}
	return ui.List(c.Key("player-queue-list"), &a.playerQueueList, len(pl.queue), func(index int) {
		track := pl.queue[index]
		playing := pl.hasCurrent && index == pl.at
		row := ui.ButtonBase(c).Key("player-queue-"+track.ID).Height(56).
			Padding(8, 10, 8, 10).Gap(10).Radius(10).
			Justify(ui.Start).AlignItems(ui.Center).FocusRing(false).
			Label(displayTitle(track.Title))
		bg := ui.Transparent
		switch {
		case playing:
			bg = p.accent.Alpha(0.1)
		case row.Hovered():
			bg = p.hover
		}
		row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 100 * time.Millisecond}))
		row.Children(func() {
			ui.Box(c).Width(22).Center().Children(func() {
				switch {
				case playing:
					a.playingBars(c, p, pl.playing)
				case row.Hovered():
					ui.Icon(c, iconPlay).Size(11, 11).TextColor(p.text)
				default:
					ui.Textf(c, "%d", index+1).FontSize(10.5).FontFeatures("tnum").
						TextColor(p.faint)
				}
			})
			ui.Box(c).Size(38, 38).Radius(7).Children(func() {
				a.cover(c, track.Cover, displayTitle(track.Title), 7)
			})
			ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(3).Children(func() {
				titleColor := p.text
				if playing {
					titleColor = p.accent
				}
				ui.Text(c, displayTitle(track.Title)).FontSize(13).FontWeight(590).
					SingleLine().TextColor(titleColor)
				ui.Text(c, displayArtist(track.Artist)).FontSize(11).
					SingleLine().TextColor(p.muted)
			})
			if track.DurationSec > 0 {
				ui.Text(c, formatClock(track.Duration())).FontSize(10.5).
					FontFeatures("tnum").TextColor(p.faint).Width(38).TextAlign(ui.End)
			}
		})
		if row.Clicked() {
			if playing {
				pl.toggle()
			} else {
				pl.jumpTo(index)
			}
		}
	}).Grow(1).MinHeight(0).Gap(2)
}
