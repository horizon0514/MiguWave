package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *app) localPlaylistsPage(c *ui.Context, p palette) {
	pageScroll(c, "local-playlists", 24, func() {
		ui.Row(c).AlignItems(ui.End).DragWindow().Children(func() {
			ui.Column(c).Gap(4).Children(func() {
				ui.Text(c, "自建歌单").FontSize(32).FontWeight(760).LetterSpacing(-0.7)
				ui.Text(c, "收藏此刻想循环的歌，只保存在这台设备，不会改动咪咕账号").
					FontSize(13).TextColor(p.muted)
			})
			ui.Spacer(c)
			a.primaryButton(c, "新建歌单", iconPlus, func() {
				a.openLocalPlaylistCreate(nil, false)
			})
		})

		if len(a.localPlaylists) == 0 {
			a.emptyPanel(c, p, iconPlaylist, "还没有自建歌单",
				"新建一个歌单，再从歌曲列表或搜索结果的加号里添加歌曲。",
				"创建第一个歌单", func() { a.openLocalPlaylistCreate(nil, false) })
			return
		}

		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Textf(c, "%d 个本地歌单", len(a.localPlaylists)).FontSize(12.5).TextColor(p.muted)
			ui.Spacer(c)
			ui.Text(c, "仅本机保存").FontSize(10.5).FontWeight(600).TextColor(p.faint)
		})
		ui.Grid(c).Columns(a.columns(c, 178)).GapX(18).GapY(24).Children(func() {
			for _, playlist := range a.localPlaylists {
				a.localPlaylistCard(c, p, playlist)
			}
		})
	})
}

func (a *app) localPlaylistCard(c *ui.Context, p palette, playlist localPlaylist) {
	card := ui.ButtonBase(c).Key("local-playlist-" + playlist.ID).Column().Gap(8).
		AlignItems(ui.Stretch).FocusRing(false).Label(playlist.Name)
	hovered := card.Hovered()
	card.Children(func() {
		art := ui.Box(c).AspectRatio(1).Radius(14).Clip()
		if hovered {
			art.Shadow(0, 16, 32, -9, p.shadow)
		} else {
			art.Shadow(0, 4, 12, -6, p.shadow)
		}
		art.Children(func() {
			a.localPlaylistCover(c, p, playlist, 14)
			if len(playlist.Tracks) > 0 {
				ui.Box(c).Absolute().Left(9).Bottom(8).Padding(3, 8).Radius(999).
					Background(ui.RGBA(18, 8, 13, 0.46)).Children(func() {
					ui.Textf(c, "%d 首", len(playlist.Tracks)).FontSize(9.5).FontWeight(600).
						TextColor(ui.RGBA(255, 255, 255, 0.92))
				})
			}
			if hovered {
				a.cardPlayButton(c, p, "local-playlist-play-"+playlist.ID, func() {
					a.playTracks(playlist.Tracks, 0)
				})
			}
			more := ui.ButtonBase(c).Key("local-playlist-more-"+playlist.ID).
				Absolute().Top(8).Right(8).Size(30, 30).Center().Radius(999).
				FocusRing(false).Label("歌单操作").Background(ui.RGBA(18, 8, 13, 0.52))
			more.Children(func() {
				ui.Icon(c, iconMore).Size(15, 15).TextColor(ui.Hex("#ffffff"))
			})
			more.PopoverMenuButton(a.localPlaylistMenu(playlist))
		})
		ui.Text(c, playlist.Name).FontSize(13).FontWeight(590).MaxLines(2).LineHeight(1.28)
		meta := "本地歌单"
		if len(playlist.Tracks) > 0 {
			meta = fmt.Sprintf("本地歌单 · %d 首 · %s", len(playlist.Tracks), formatClock(playlist.Duration()))
		}
		ui.Text(c, meta).FontSize(10.5).TextColor(p.faint).SingleLine().Margin(-3, 0, 0, 0)
	})
	if card.Clicked() {
		a.openLocalPlaylist(playlist.ID)
	}
}

func (a *app) localPlaylistCover(c *ui.Context, p palette, playlist localPlaylist, radius float32) ui.Element {
	covers := make([]string, 0, 3)
	for _, track := range playlist.Tracks {
		cover := firstNonEmpty(track.CoverHQ, track.Cover)
		if cover != "" {
			covers = append(covers, cover)
		}
		if len(covers) == 3 {
			break
		}
	}

	box := ui.Box(c).Fill().Radius(radius).Clip()
	if len(covers) == 0 {
		box.Gradient(p.accent, p.accent2, 35)
		box.Children(func() {
			ui.Column(c).Fill().Center().Gap(7).Children(func() {
				ui.Box(c).Size(46, 46).Radius(14).Background(ui.RGBA(255, 255, 255, 0.16)).Center().
					Children(func() {
						ui.Icon(c, iconPlaylist).Size(22, 22).TextColor(ui.Hex("#ffffff"))
					})
				title := []rune(playlist.Name)
				if len(title) > 8 {
					title = title[:8]
				}
				ui.Text(c, string(title)).FontSize(11).FontWeight(650).
					TextColor(ui.RGBA(255, 255, 255, 0.86)).SingleLine()
			})
		})
		return box
	}

	if len(covers) == 1 {
		box.Children(func() {
			a.cover(c, covers[0], playlist.Name, radius)
		})
		return box
	}

	box.Children(func() {
		if len(covers) == 2 {
			ui.Row(c).Fill().Gap(1).Children(func() {
				a.localCoverCell(c, playlist.Name, covers[0])
				a.localCoverCell(c, playlist.Name, covers[1])
			})
			return
		}
		ui.Row(c).Fill().Gap(1).Children(func() {
			ui.Box(c).Grow(2).FillHeight().Children(func() {
				a.cover(c, covers[0], playlist.Name, 0)
			})
			ui.Column(c).Grow(1).FillHeight().Gap(1).Children(func() {
				ui.Box(c).Grow(1).FillWidth().Children(func() {
					a.cover(c, covers[1], playlist.Name, 0)
				})
				ui.Box(c).Grow(1).FillWidth().Children(func() {
					a.cover(c, covers[2], playlist.Name, 0)
				})
			})
		})
	})
	return box
}

func (a *app) localCoverCell(c *ui.Context, fallback, cover string) {
	ui.Box(c).Grow(1).FillHeight().Children(func() {
		a.cover(c, cover, fallback, 0)
	})
}

func (a *app) localPlaylistDetailPage(c *ui.Context, p palette) {
	playlist, index := a.localPlaylistByID(a.localPlaylistID)
	if index < 0 {
		a.page = pageLocalPlaylists
		a.localPlaylistsPage(c, p)
		return
	}

	pageScroll(c, "local-playlist-"+playlist.ID, 20, func() {
		ui.Row(c).Height(34).AlignItems(ui.Center).DragWindow().Children(func() {
			a.backLink(c, p, "返回自建歌单", func() { a.navTo(pageLocalPlaylists) })
			ui.Spacer(c)
			ui.Box(c).Padding(4, 9).Radius(999).Background(p.accent.Alpha(0.1)).Children(func() {
				ui.Text(c, "仅本机保存").FontSize(9.5).FontWeight(680).TextColor(p.accent)
			})
		})

		card := ui.Box(c).FillWidth().Radius(18).Clip().Background(p.panel).Border(1, p.line)
		card.Children(func() {
			ui.Row(c).FillWidth().Padding(22, 24).Gap(24).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(190, 190).Radius(16).Clip().Children(func() {
					a.localPlaylistCover(c, p, playlist, 16)
				})
				ui.Column(c).Grow(1).MinWidth(0).Gap(8).Children(func() {
					ui.Text(c, "LOCAL PLAYLIST").FontSize(9.5).FontWeight(750).LetterSpacing(1.5).
						TextColor(p.accent)
					ui.Text(c, playlist.Name).FontSize(28).FontWeight(760).LetterSpacing(-0.5).
						MaxLines(2).LineHeight(1.15)
					meta := fmt.Sprintf("本地歌单 · %d 首", len(playlist.Tracks))
					if len(playlist.Tracks) > 0 {
						meta = joinMeta(meta, formatClock(playlist.Duration()))
					}
					ui.Text(c, meta).FontSize(11.5).TextColor(p.muted).SingleLine()
					ui.Text(c, "这个歌单只写入本机 state.json，不会同步或修改咪咕账号。").
						FontSize(11.5).TextColor(p.muted).LineHeight(1.55)
					ui.Row(c).Gap(9).Wrap().Margin(5, 0, 0, 0).Children(func() {
						if len(playlist.Tracks) > 0 {
							a.primaryButton(c, "播放全部", iconPlay, func() {
								a.playTracks(playlist.Tracks, 0)
							})
						}
						a.ghostButton(c, "添加歌曲", iconPlus, func() {
							a.focusSearch = true
							a.navTo(pageSearch)
						})
						a.ghostButton(c, "重命名", nil, func() {
							a.openLocalPlaylistRename(playlist)
						})
						a.ghostButton(c, "删除歌单", nil, func() {
							a.openLocalPlaylistDelete(playlist)
						})
					})
				})
			})
		})

		if len(playlist.Tracks) == 0 {
			a.emptyPanel(c, p, iconMusic, "歌单还是空的",
				"去搜索歌曲，点歌曲旁边的加号就能加入这个歌单。", "去找歌", func() {
					a.focusSearch = true
					a.navTo(pageSearch)
				})
			return
		}

		ui.Column(c).Gap(8).Children(func() {
			a.sectionTitle(c, p, "歌曲",
				fmt.Sprintf("%d 首 · %s", len(playlist.Tracks), formatClock(playlist.Duration())))
			ui.Column(c).Gap(2).Children(func() {
				for i := range playlist.Tracks {
					a.songRow(c, p, playlist.Tracks[i], playlist.Tracks, i, "local")
				}
			})
		})
	})
}

func (a *app) openLocalPlaylist(id string) {
	if _, index := a.localPlaylistByID(id); index < 0 {
		return
	}
	a.localPlaylistID = id
	a.page = pageLocalPlaylistDetail
	a.refresh()
}

func (a *app) localPlaylistMenu(playlist localPlaylist) func(*ui.Menu) {
	return func(m *ui.Menu) {
		if m.Item("打开歌单").Chosen() {
			a.openLocalPlaylist(playlist.ID)
		}
		if m.Item("播放全部").Disabled(len(playlist.Tracks) == 0).Chosen() {
			a.playTracks(playlist.Tracks, 0)
		}
		if m.Item("重命名").Chosen() {
			a.openLocalPlaylistRename(playlist)
		}
		m.Separator()
		if m.Item("删除歌单").Chosen() {
			a.openLocalPlaylistDelete(playlist)
		}
	}
}

func (a *app) openLocalPlaylistCreate(track *Track, openAfter bool) {
	a.localPlaylistDeleteOpen = false
	a.localPlaylistEditorOpen = true
	a.localPlaylistEditorID = ""
	a.localPlaylistEditorName = ""
	a.localPlaylistEditorErr = ""
	a.localPlaylistEditorTrack = Track{}
	a.localPlaylistEditorHasTrack = track != nil && strings.TrimSpace(track.ID) != ""
	if a.localPlaylistEditorHasTrack {
		a.localPlaylistEditorTrack = *track
	}
	a.localPlaylistEditorOpenAfter = openAfter
	a.refresh()
}

func (a *app) openLocalPlaylistRename(playlist localPlaylist) {
	a.localPlaylistDeleteOpen = false
	a.localPlaylistEditorOpen = true
	a.localPlaylistEditorID = playlist.ID
	a.localPlaylistEditorName = playlist.Name
	a.localPlaylistEditorErr = ""
	a.localPlaylistEditorTrack = Track{}
	a.localPlaylistEditorHasTrack = false
	a.localPlaylistEditorOpenAfter = false
	a.refresh()
}

func (a *app) openLocalPlaylistDelete(playlist localPlaylist) {
	a.localPlaylistEditorOpen = false
	a.localPlaylistDeleteOpen = true
	a.localPlaylistDeleteID = playlist.ID
	a.refresh()
}

func (a *app) localPlaylistModals(c *ui.Context, p palette) {
	if a.localPlaylistEditorOpen {
		a.localPlaylistEditorModal(c, p)
		return
	}
	if a.localPlaylistDeleteOpen {
		a.localPlaylistDeleteModal(c, p)
	}
}

func (a *app) localPlaylistEditorModal(c *ui.Context, p palette) {
	editing := a.localPlaylistEditorID != ""
	submit := func() {
		if editing {
			if err := a.renameLocalPlaylist(a.localPlaylistEditorID, a.localPlaylistEditorName); err != nil {
				a.localPlaylistEditorErr = err.Error()
				return
			}
			a.localPlaylistEditorOpen = false
			a.toast = "歌单已重命名"
			a.refresh()
			return
		}
		playlist, err := a.createLocalPlaylist(a.localPlaylistEditorName)
		if err != nil {
			a.localPlaylistEditorErr = err.Error()
			return
		}
		if a.localPlaylistEditorHasTrack {
			_, _ = a.addTrackToLocalPlaylist(playlist.ID, a.localPlaylistEditorTrack)
		}
		a.localPlaylistEditorOpen = false
		a.toast = "已新建歌单 " + playlist.Name
		if a.localPlaylistEditorOpenAfter {
			a.localPlaylistID = playlist.ID
			a.page = pageLocalPlaylistDetail
		}
		a.refresh()
	}

	ui.Modal(c, &a.localPlaylistEditorOpen, func() {
		ui.Column(c).Width(430).Gap(14).Children(func() {
			ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(42, 42).Radius(12).Gradient(p.accent, p.accent2, 35).Center().Children(func() {
					ui.Icon(c, iconPlaylist).Size(20, 20).TextColor(p.onAccent)
				})
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					title := "新建自建歌单"
					if editing {
						title = "重命名歌单"
					}
					ui.Text(c, title).FontSize(19).FontWeight(750)
					ui.Text(c, "歌曲和歌单只保存在这台设备").FontSize(11.5).TextColor(p.muted)
				})
			})

			input := ui.TextInput(c.Key("local-playlist-name"), &a.localPlaylistEditorName).
				FillWidth().Label("歌单名称").Placeholder("例如：深夜循环").AutoFocus()
			if input.Submitted() {
				submit()
			}

			if a.localPlaylistEditorHasTrack {
				ui.Row(c).FillWidth().Padding(10, 12).Radius(10).Background(p.panelAlt).
					Gap(9).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconPlus).Size(14, 14).TextColor(p.accent)
					ui.Textf(c, "创建后会自动加入《%s》", a.localPlaylistEditorTrack.Title).
						FontSize(11).TextColor(p.muted).Grow(1).SingleLine()
				})
			}
			if a.localPlaylistEditorErr != "" {
				ui.Text(c, a.localPlaylistEditorErr).FontSize(11).TextColor(p.accent)
			}
			ui.Text(c, "不会同步到咪咕账号，也不会上传歌单名称。").
				FontSize(10.5).TextColor(p.faint).LineHeight(1.45)
			ui.Row(c).Justify(ui.End).Gap(8).Children(func() {
				a.ghostButton(c, "取消", nil, func() {
					a.localPlaylistEditorOpen = false
					a.refresh()
				})
				label := "创建歌单"
				if editing {
					label = "保存名称"
				}
				a.primaryButton(c, label, nil, submit)
			})
		})
	})
}

func (a *app) localPlaylistDeleteModal(c *ui.Context, p palette) {
	playlist, index := a.localPlaylistByID(a.localPlaylistDeleteID)
	if index < 0 {
		a.localPlaylistDeleteOpen = false
		return
	}
	ui.Modal(c, &a.localPlaylistDeleteOpen, func() {
		ui.Column(c).Width(410).Gap(12).Children(func() {
			ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(42, 42).Radius(12).Background(p.accent.Alpha(0.12)).Center().Children(func() {
					ui.Icon(c, iconMinus).Size(20, 20).TextColor(p.accent)
				})
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					ui.Text(c, "删除自建歌单？").FontSize(18).FontWeight(750)
					ui.Text(c, playlist.Name).FontSize(12).TextColor(p.muted).SingleLine()
				})
			})
			ui.Text(c, "只会删除本机保存的歌单，不会删除咪咕歌曲，也不会影响咪咕账号。").
				FontSize(11.5).TextColor(p.muted).LineHeight(1.55)
			ui.Row(c).Justify(ui.End).Gap(8).Margin(3, 0, 0, 0).Children(func() {
				a.ghostButton(c, "保留歌单", nil, func() {
					a.localPlaylistDeleteOpen = false
					a.refresh()
				})
				a.primaryButton(c, "确认删除", nil, func() {
					wasOpen := a.page == pageLocalPlaylistDetail && a.localPlaylistID == playlist.ID
					a.deleteLocalPlaylist(playlist.ID)
					a.localPlaylistDeleteOpen = false
					if wasOpen {
						a.page = pageLocalPlaylists
						a.localPlaylistID = ""
					}
					a.toast = "已删除歌单 " + playlist.Name
					a.refresh()
				})
			})
		})
	})
}

func (a *app) addToPlaylistMenu(track Track) func(*ui.Menu) {
	return func(m *ui.Menu) {
		if len(a.localPlaylists) == 0 {
			if m.Item("新建歌单…").Chosen() {
				a.openLocalPlaylistCreate(&track, false)
			}
			return
		}
		for _, playlist := range a.localPlaylists {
			playlist := playlist
			checked := playlist.Contains(track.ID)
			if m.Item(playlist.Name).Checked(checked).Chosen() {
				if checked {
					a.removeTrackFromLocalPlaylist(playlist.ID, track.ID)
					a.toast = "已从「" + playlist.Name + "」移除"
				} else {
					if _, err := a.addTrackToLocalPlaylist(playlist.ID, track); err != nil {
						a.toast = err.Error()
					} else {
						a.toast = "已加入「" + playlist.Name + "」"
					}
				}
				a.refresh()
				return
			}
		}
		m.Separator()
		if m.Item("新建歌单…").Chosen() {
			a.openLocalPlaylistCreate(&track, false)
		}
	}
}

func (a *app) songPlaylistButton(c *ui.Context, key any, track Track) ui.Element {
	b := a.iconButton(c, key, iconPlus, "添加到自建歌单", false)
	b.PopoverMenuButton(a.addToPlaylistMenu(track))
	return b
}

func (a *app) songPlaylistOverlayButton(c *ui.Context, key any, track Track) ui.Element {
	b := ui.ButtonBase(c).Key(key).Absolute().Top(8).Right(44).
		Size(30, 30).Center().Radius(999).FocusRing(false).
		Label("添加到自建歌单").Background(ui.RGBA(18, 8, 13, 0.55))
	b.Children(func() {
		ui.Icon(c, iconPlus).Size(15, 15).TextColor(ui.Hex("#ffffff"))
	})
	b.PopoverMenuButton(a.addToPlaylistMenu(track))
	return b
}

func (a *app) removeFromLocalPlaylistButton(c *ui.Context, scope string, track Track) ui.Element {
	b := a.iconButton(c, scope+"-remove-"+track.ID, iconMinus, "从自建歌单移除", false)
	if b.Clicked() {
		if a.removeTrackFromLocalPlaylist(a.localPlaylistID, track.ID) {
			a.toast = "已从歌单移除 " + track.Title
			a.refresh()
		}
	}
	return b
}
