package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *app) backLink(c *ui.Context, p palette, label string, onClick func()) {
	b := ui.ButtonBase(c).Key("back-" + label).Height(32).PaddingX(9).Radius(999).
		Gap(6).AlignItems(ui.Center).FocusRing(false).Label(label)
	if b.Hovered() {
		b.Background(p.hover)
	}
	b.Children(func() {
		ui.Icon(c, iconBack).Size(15, 15).TextColor(p.muted)
		ui.Text(c, label).FontSize(12).FontWeight(600).TextColor(p.muted)
	})
	if b.Clicked() {
		onClick()
	}
}

func (a *app) playlistsPage(c *ui.Context, p palette) {
	pageScroll(c, "playlists", 28, func() {
		ui.Row(c).AlignItems(ui.End).DragWindow().Children(func() {
			ui.Column(c).Gap(4).Children(func() {
				ui.Text(c, "歌单广场").FontSize(32).FontWeight(760).LetterSpacing(-0.7)
				ui.Text(c, "编辑推荐、歌单百宝箱、深度歌手和每日上新，全部来自咪咕曲库").
					FontSize(13).TextColor(p.muted)
			})
			ui.Spacer(c)
			if a.playlistSquareLoading && len(a.playlistSections) > 0 {
				ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c).FontSize(14).TextColor(p.accent)
					ui.Text(c, "更新中").FontSize(11).TextColor(p.faint)
				})
			}
		})

		switch {
		case a.playlistSquareLoading && len(a.playlistSections) == 0:
			a.loadingPanel(c, p, "正在打开咪咕歌单广场…", 220)
		case a.playlistSquareErr != "" && len(a.playlistSections) == 0:
			a.noticePanel(c, p, a.playlistSquareErr, "重新加载", a.loadPlaylistSquare)
		default:
			for _, section := range a.playlistSections {
				section := section
				ui.Column(c).Gap(14).Children(func() {
					a.sectionTitle(c, p, section.Title, fmt.Sprintf("%d 个歌单", len(section.Items)))
					ui.Grid(c).Columns(a.columns(c, 178)).GapX(18).GapY(24).Children(func() {
						for _, playlist := range section.Items {
							a.playlistCard(c, p, playlist, pagePlaylists, "square")
						}
					})
				})
			}
			if len(a.playlistSections) == 0 {
				a.emptyPanel(c, p, iconPlaylist, "歌单广场暂时是空的",
					"稍后重新加载，或者先去排行榜听几首。", "重新加载", a.loadPlaylistSquare)
			}
		}
	})
}

func (a *app) playlistCard(c *ui.Context, p palette, playlist miguPlaylist, from pageKind, scope string) {
	card := ui.ButtonBase(c).Key(scope + "-playlist-" + playlist.ID).Column().Gap(8).
		AlignItems(ui.Stretch).FocusRing(false).Label(playlist.Title)
	hovered := card.Hovered()
	card.Children(func() {
		art := ui.Box(c).AspectRatio(1).Radius(14).Clip()
		if hovered {
			art.Shadow(0, 16, 32, -9, p.shadow)
		} else {
			art.Shadow(0, 4, 12, -6, p.shadow)
		}
		art.Children(func() {
			cover, low := playlist.Cover, ""
			if playlist.CoverHQ != "" {
				cover, low = playlist.CoverHQ, playlist.Cover
			}
			a.coverPair(c, cover, low, playlist.Title, 14)
			if playlist.PlayCount != "" {
				ui.Row(c).Absolute().Left(9).Bottom(8).Padding(3, 7).Radius(999).
					Background(ui.RGBA(18, 8, 13, 0.42)).Gap(4).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconPlay).Size(10, 10).TextColor(ui.RGBA(255, 255, 255, 0.82))
					ui.Text(c, playlist.PlayCount).FontSize(9.5).FontWeight(600).
						TextColor(ui.RGBA(255, 255, 255, 0.9)).SingleLine()
				})
			}
			if a.isPlaylistSaved(playlist.ID) {
				ui.Box(c).Absolute().Top(8).Right(8).Size(24, 24).Radius(999).Center().
					Background(ui.RGBA(18, 8, 13, 0.48)).Children(func() {
					ui.Icon(c, iconHeartFill).Size(12, 12).TextColor(ui.Hex("#ff6b9b"))
				})
			}
			if hovered {
				a.cardPlayButton(c, p, scope+"-playlist-play-"+playlist.ID, func() {
					a.openPlaylist(playlist, from)
				})
			}
		})
		ui.Text(c, playlist.Title).FontSize(13).FontWeight(590).MaxLines(2).LineHeight(1.28)
		meta := firstNonEmpty(playlist.Owner, playlist.Description)
		if meta == "" {
			meta = "咪咕音乐歌单"
		}
		ui.Text(c, meta).FontSize(10.5).TextColor(p.faint).SingleLine().Margin(-3, 0, 0, 0)
	})
	if card.Clicked() {
		a.openPlaylist(playlist, from)
	}
}

func (a *app) playlistDetailPage(c *ui.Context, p palette) {
	detail := a.playlistDetail
	playlist := detail.Playlist
	pageScroll(c, "playlist-"+a.playlistID, 20, func() {
		ui.Row(c).Height(34).AlignItems(ui.Center).DragWindow().Children(func() {
			a.backLink(c, p, playlistBackLabel(a.playlistFrom), func() {
				a.navTo(a.playlistFrom)
			})
			ui.Spacer(c)
			if detail.Total > 0 {
				ui.Textf(c, "共 %d 首", detail.Total).FontSize(11).TextColor(p.faint)
			}
		})

		a.playlistHero(c, p, playlist, detail)

		switch {
		case a.playlistErr != "" && len(detail.Tracks) == 0:
			a.noticePanel(c, p, a.playlistErr, "重新加载", a.loadPlaylistDetail)
		case a.playlistLoading && len(detail.Tracks) == 0:
			a.loadingPanel(c, p, "正在取回歌单里的歌曲…", 180)
		case len(detail.Tracks) > 0:
			ui.Column(c).Gap(8).Children(func() {
				a.sectionTitle(c, p, "歌曲",
					fmt.Sprintf("%d 首 · %s", len(detail.Tracks), formatClock(tracksDuration(detail.Tracks))))
				ui.Column(c).Gap(2).Children(func() {
					for i := range detail.Tracks {
						a.songRow(c, p, detail.Tracks[i], detail.Tracks, i, "playlist")
					}
				})
			})
			if detail.HasMore || a.playlistLoadingMore {
				ui.Row(c).Justify(ui.Center).Margin(8, 0, 8, 0).Children(func() {
					if a.playlistLoadingMore {
						ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Spinner(c).FontSize(16).TextColor(p.accent)
							ui.Text(c, "继续加载…").FontSize(12).TextColor(p.muted)
						})
					} else {
						a.ghostButton(c, "加载更多歌曲", iconArrow, a.loadMorePlaylist)
					}
				})
			} else {
				ui.Row(c).Justify(ui.Center).Margin(4, 0, 10, 0).Children(func() {
					ui.Text(c, "已经到底了").FontSize(11.5).TextColor(p.faint)
				})
			}
		default:
			a.emptyPanel(c, p, iconMusic, "歌单里还没有歌曲", "稍后重试，或换一个歌单。", "", nil)
		}
	})
}

func (a *app) playlistHero(c *ui.Context, p palette, playlist miguPlaylist, detail miguPlaylistDetail) {
	card := ui.Box(c).FillWidth().Radius(18).Clip().Background(p.panel).Border(1, p.line)
	card.Children(func() {
		ui.Row(c).FillWidth().Padding(22, 24).Gap(24).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(190, 190).Radius(16).Clip().Children(func() {
				cover, low := playlist.Cover, ""
				if playlist.CoverHQ != "" {
					cover, low = playlist.CoverHQ, playlist.Cover
				}
				a.coverPair(c, cover, low, playlist.Title, 16)
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(8).Children(func() {
				ui.Text(c, "MIGU PLAYLIST").FontSize(9.5).FontWeight(750).LetterSpacing(1.5).
					TextColor(p.accent)
				ui.Text(c, playlist.Title).FontSize(27).FontWeight(760).LetterSpacing(-0.5).
					MaxLines(2).LineHeight(1.15)
				meta := joinMeta(playlist.Owner, playlist.PlayCount)
				count := detail.Total
				if count <= 0 {
					count = playlist.TrackCount
				}
				if count <= 0 {
					count = len(detail.Tracks)
				}
				if count > 0 {
					meta = joinMeta(meta, fmt.Sprintf("%d 首", count))
				}
				if len(detail.Tracks) > 0 && (detail.Total <= 0 || len(detail.Tracks) >= detail.Total) {
					meta = joinMeta(meta, formatClock(tracksDuration(detail.Tracks)))
				}
				if meta != "" {
					ui.Text(c, meta).FontSize(11.5).TextColor(p.muted).SingleLine()
				}
				if playlist.Description != "" {
					ui.Text(c, playlist.Description).FontSize(11.5).TextColor(p.muted).
						MaxLines(2).LineHeight(1.55).MaxWidth(680)
				}
				if len(playlist.Tags) > 0 {
					ui.Row(c).Gap(6).Wrap().Children(func() {
						for _, tag := range playlist.Tags {
							ui.Box(c).Padding(4, 8).Radius(999).Background(p.accent.Alpha(0.08)).
								Children(func() {
									ui.Text(c, tag).FontSize(9.5).FontWeight(600).TextColor(p.accent).SingleLine()
								})
						}
					})
				}
				ui.Row(c).Gap(9).Margin(5, 0, 0, 0).Children(func() {
					if len(detail.Tracks) > 0 {
						a.primaryButton(c, "播放全部", iconPlay, func() {
							a.playTracks(detail.Tracks, 0)
						})
					}
					saved := a.isPlaylistSaved(playlist.ID)
					label := "收藏歌单"
					if saved {
						label = "取消收藏"
					}
					a.ghostButton(c, label, iconHeart, func() {
						a.togglePlaylistSaved(playlist)
						if a.isPlaylistSaved(playlist.ID) {
							a.toast = "已收藏歌单 " + playlist.Title
						} else {
							a.toast = "已取消收藏歌单"
						}
						a.refresh()
					})
				})
			})
		})
	})
}

func (a *app) savedPlaylistsPage(c *ui.Context, p palette) {
	pageScroll(c, "saved-playlists", 24, func() {
		a.pageHeader(c, p, "收藏歌单", "留在资料库里的咪咕歌单")
		if len(a.savedPlaylists) == 0 {
			a.emptyPanel(c, p, iconPlaylist, "还没有收藏歌单",
				"在歌单详情里点一下收藏，歌单就会出现在这里。", "去歌单广场", func() {
					a.navTo(pagePlaylists)
				})
			return
		}
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Textf(c, "%d 个歌单", len(a.savedPlaylists)).FontSize(12.5).TextColor(p.muted)
			ui.Spacer(c)
			a.miniLink(c, p, "继续发现", func() { a.navTo(pagePlaylists) })
		})
		ui.Grid(c).Columns(a.columns(c, 178)).GapX(18).GapY(24).Children(func() {
			for _, playlist := range a.savedPlaylists {
				a.playlistCard(c, p, playlist, pageSavedPlaylists, "saved")
			}
		})
	})
}

func (a *app) singersPage(c *ui.Context, p palette) {
	pageScroll(c, "singers-"+a.singerTab, 22, func() {
		ui.Row(c).AlignItems(ui.End).DragWindow().Children(func() {
			ui.Column(c).Gap(4).Children(func() {
				ui.Text(c, "歌手").FontSize(32).FontWeight(760).LetterSpacing(-0.7)
				ui.Text(c, "按华语、欧美、日韩分类浏览咪咕歌手").
					FontSize(13).TextColor(p.muted)
			})
			ui.Spacer(c)
			if len(a.singers) > 0 {
				ui.Textf(c, "%d 位", len(a.singers)).FontSize(11.5).TextColor(p.faint)
			}
		})

		ui.Row(c).Gap(8).Wrap().Children(func() {
			for _, tab := range miguSingerTabs {
				tab := tab
				a.chip(c, tab.Label, a.singerTab == tab.ID, func() {
					if a.singerTab != tab.ID {
						a.loadSingers(tab.ID)
					}
				})
			}
		})

		switch {
		case a.singersLoading && len(a.singers) == 0:
			a.loadingPanel(c, p, "正在读取咪咕歌手…", 220)
		case a.singersErr != "" && len(a.singers) == 0:
			a.noticePanel(c, p, a.singersErr, "重新加载", func() { a.loadSingers(a.singerTab) })
		case len(a.singers) > 0:
			visible := a.singers
			limit := a.singerVisible
			if limit <= 0 {
				limit = 30
			}
			if len(visible) > limit {
				visible = visible[:limit]
			}
			ui.Grid(c).Columns(a.columns(c, 150)).GapX(16).GapY(24).Children(func() {
				for i := range visible {
					a.singerCard(c, p, visible[i])
				}
			})
			if limit < len(a.singers) {
				ui.Row(c).Justify(ui.Center).Margin(4, 0, 10, 0).Children(func() {
					a.ghostButton(c, "显示更多歌手", iconArrow, func() {
						a.singerVisible += 30
					})
				})
			}
		default:
			a.emptyPanel(c, p, iconMic, "这个分类还没有歌手",
				"换一个分类，或者稍后重新加载。", "重新加载", func() { a.loadSingers(a.singerTab) })
		}
	})
}

func (a *app) singerCard(c *ui.Context, p palette, singer miguSinger) {
	card := ui.ButtonBase(c).Key("singer-" + singer.ID).Column().Gap(8).
		AlignItems(ui.Stretch).FocusRing(false).Label(singer.Name)
	hovered := card.Hovered()
	card.Children(func() {
		art := ui.Box(c).AspectRatio(1).Radius(16).Clip()
		if hovered {
			art.Shadow(0, 14, 28, -9, p.shadow)
		}
		art.Children(func() {
			cover := firstNonEmpty(singer.CoverHQ, singer.Cover)
			a.coverPair(c, cover, singer.Cover, singer.Name, 16)
			if hovered {
				ui.Box(c).Absolute().Bottom(9).Right(9).Size(30, 30).Radius(999).Center().
					Background(p.accent).Children(func() {
					ui.Icon(c, iconArrow).Size(13, 13).TextColor(p.onAccent)
				})
			}
			if singer.Fans != "" {
				ui.Box(c).Absolute().Left(9).Bottom(9).Padding(3, 7).Radius(999).
					Background(ui.RGBA(18, 8, 13, 0.42)).Children(func() {
					ui.Text(c, singer.Fans+" 粉丝").FontSize(9.5).FontWeight(600).
						TextColor(ui.RGBA(255, 255, 255, 0.9))
				})
			}
		})
		ui.Text(c, singer.Name).FontSize(13).FontWeight(620).SingleLine()
		meta := firstNonEmpty(singer.Hot, "咪咕歌手")
		ui.Text(c, meta).FontSize(10.5).TextColor(p.faint).SingleLine().Margin(-3, 0, 0, 0)
	})
	if card.Clicked() {
		a.openSinger(singer)
	}
}

func (a *app) singerDetailPage(c *ui.Context, p palette) {
	detail := a.singerDetail
	singer := detail.Singer
	pageScroll(c, "singer-"+singer.ID, 22, func() {
		ui.Row(c).Height(34).AlignItems(ui.Center).DragWindow().Children(func() {
			a.backLink(c, p, "返回歌手", func() { a.navTo(pageSingers) })
			ui.Spacer(c)
		})

		a.singerHero(c, p, singer, detail)

		if len(detail.Similar) > 0 {
			ui.Column(c).Gap(14).Children(func() {
				a.sectionTitle(c, p, "相似歌手", "SIMILAR ARTISTS")
				ui.Grid(c).Columns(a.columns(c, 132)).GapX(16).GapY(20).Children(func() {
					for _, similar := range detail.Similar {
						a.singerCard(c, p, similar)
					}
				})
			})
		}

		switch {
		case a.singerErr != "" && len(detail.Tracks) == 0:
			a.noticePanel(c, p, a.singerErr, "重新加载", a.loadSingerDetail)
		case a.singerLoading && len(detail.Tracks) == 0:
			a.loadingPanel(c, p, "正在读取热门歌曲…", 180)
		case len(detail.Tracks) > 0:
			ui.Column(c).Gap(8).Children(func() {
				a.sectionTitle(c, p, "热门歌曲",
					fmt.Sprintf("%d 首 · %s", len(detail.Tracks), formatClock(tracksDuration(detail.Tracks))))
				ui.Column(c).Gap(2).Children(func() {
					for i := range detail.Tracks {
						a.songRow(c, p, detail.Tracks[i], detail.Tracks, i, "singer")
					}
				})
			})
			if detail.HasMore || a.singerLoadingMore {
				ui.Row(c).Justify(ui.Center).Margin(8, 0, 8, 0).Children(func() {
					if a.singerLoadingMore {
						ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Spinner(c).FontSize(16).TextColor(p.accent)
							ui.Text(c, "继续加载…").FontSize(12).TextColor(p.muted)
						})
					} else {
						a.ghostButton(c, "加载更多歌曲", iconArrow, a.loadMoreSingerSongs)
					}
				})
			}
		}
	})
}

func (a *app) singerHero(c *ui.Context, p palette, singer miguSinger, detail miguSingerDetail) {
	card := ui.Box(c).FillWidth().Radius(18).Clip().Background(p.panel).Border(1, p.line)
	card.Children(func() {
		ui.Row(c).FillWidth().Padding(22, 24).Gap(24).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(188, 188).Radius(20).Clip().Children(func() {
				cover := firstNonEmpty(singer.CoverHQ, singer.Cover)
				a.coverPair(c, cover, singer.Cover, singer.Name, 20)
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(8).Children(func() {
				ui.Text(c, "MIGU ARTIST").FontSize(9.5).FontWeight(750).LetterSpacing(1.5).
					TextColor(p.accent)
				ui.Text(c, singer.Name).FontSize(30).FontWeight(780).LetterSpacing(-0.6).SingleLine()
				meta := joinMeta(singer.Hot, singer.Fans)
				if meta != "" {
					ui.Text(c, meta).FontSize(11.5).TextColor(p.muted).SingleLine()
				}
				if detail.Bio != "" {
					ui.Text(c, detail.Bio).FontSize(11.5).TextColor(p.muted).
						MaxLines(4).LineHeight(1.6).MaxWidth(720)
				}
				if len(detail.Tracks) > 0 {
					ui.Row(c).Margin(5, 0, 0, 0).Children(func() {
						a.primaryButton(c, "播放全部", iconPlay, func() {
							a.playTracks(detail.Tracks, 0)
						})
					})
				}
			})
		})
	})
}

func (a *app) radiosPage(c *ui.Context, p palette) {
	pageScroll(c, "radios", 26, func() {
		ui.Row(c).AlignItems(ui.End).DragWindow().Children(func() {
			ui.Column(c).Gap(4).Children(func() {
				ui.Text(c, "电台").FontSize(32).FontWeight(760).LetterSpacing(-0.7)
				ui.Text(c, "私人 FM、时序电台与场景电台，按此刻的心情打开").
					FontSize(13).TextColor(p.muted)
			})
			ui.Spacer(c)
			if a.radiosLoading && len(a.radioGroups) > 0 {
				ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c).FontSize(14).TextColor(p.accent)
					ui.Text(c, "更新中").FontSize(11).TextColor(p.faint)
				})
			}
		})

		switch {
		case a.radiosLoading && len(a.radioGroups) == 0:
			a.loadingPanel(c, p, "正在打开咪咕电台…", 220)
		case a.radiosErr != "" && len(a.radioGroups) == 0:
			a.noticePanel(c, p, a.radiosErr, "重新加载", a.loadRadios)
		default:
			for _, group := range a.radioGroups {
				group := group
				ui.Column(c).Gap(14).Children(func() {
					a.sectionTitle(c, p, group.Title, fmt.Sprintf("%d 个电台", len(group.Items)))
					ui.Grid(c).Columns(a.columns(c, 178)).GapX(18).GapY(24).Children(func() {
						for _, radio := range group.Items {
							a.radioCard(c, p, radio)
						}
					})
				})
			}
			if len(a.radioGroups) == 0 {
				a.emptyPanel(c, p, iconRadio, "电台暂时是空的",
					"稍后重新加载，咪咕的电台栏目会自动回来。", "重新加载", a.loadRadios)
			}
		}
	})
}

func (a *app) radioCard(c *ui.Context, p palette, radio miguRadio) {
	card := ui.ButtonBase(c).Key("radio-" + radio.ID).Column().Gap(8).
		AlignItems(ui.Stretch).FocusRing(false).Label(radio.Title)
	hovered := card.Hovered()
	card.Children(func() {
		art := ui.Box(c).AspectRatio(1).Radius(14).Clip()
		if hovered {
			art.Shadow(0, 14, 30, -8, p.shadow)
		}
		art.Children(func() {
			a.coverPair(c, radio.Cover, "", radio.Title, 14)
			ui.Box(c).Absolute().Top(9).Left(9).Padding(3, 8).Radius(999).
				Background(ui.RGBA(255, 255, 255, 0.18)).Children(func() {
				ui.Text(c, radioKindLabel(radio.Kind)).FontSize(8.5).FontWeight(680).
					TextColor(ui.RGBA(255, 255, 255, 0.92))
			})
			if hovered {
				ui.Box(c).Absolute().Bottom(9).Right(9).Size(36, 36).Radius(999).Center().
					Background(p.accent).Children(func() {
					ui.Icon(c, iconPlay).Size(13, 13).TextColor(p.onAccent)
				})
			}
		})
		ui.Text(c, radio.Title).FontSize(13).FontWeight(600).MaxLines(1)
		sub := radio.Subtitle
		if sub == "" {
			sub = "咪咕音乐电台"
		}
		ui.Text(c, sub).FontSize(10.5).TextColor(p.faint).SingleLine().Margin(-3, 0, 0, 0)
	})
	if card.Clicked() {
		a.openRadio(radio, pageRadios)
	}
}

func (a *app) radioDetailPage(c *ui.Context, p palette) {
	detail := a.radioDetail
	radio := detail.Radio
	pageScroll(c, "radio-"+radio.ID, 22, func() {
		ui.Row(c).Height(34).AlignItems(ui.Center).DragWindow().Children(func() {
			a.backLink(c, p, radioBackLabel(a.radioFrom), func() { a.navTo(a.radioFrom) })
			ui.Spacer(c)
			ui.Box(c).Padding(4, 9).Radius(999).Background(p.accent.Alpha(0.1)).Children(func() {
				ui.Text(c, radioKindLabel(radio.Kind)).FontSize(9.5).FontWeight(680).TextColor(p.accent)
			})
		})

		radioCard := ui.Box(c).FillWidth().Radius(18).Clip().Background(p.panel).Border(1, p.line)
		radioCard.Children(func() {
			ui.Row(c).FillWidth().Padding(22, 24).Gap(24).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(186, 186).Radius(16).Clip().Children(func() {
					a.coverPair(c, radio.Cover, "", radio.Title, 16)
				})
				ui.Column(c).Grow(1).MinWidth(0).Gap(8).Children(func() {
					ui.Text(c, "MIGU RADIO").FontSize(9.5).FontWeight(750).LetterSpacing(1.5).
						TextColor(p.accent)
					ui.Text(c, radio.Title).FontSize(28).FontWeight(760).LetterSpacing(-0.5).SingleLine()
					sub := firstNonEmpty(radio.Subtitle, "咪咕音乐为你准备的电台")
					ui.Text(c, sub).FontSize(12).TextColor(p.muted).MaxLines(2).LineHeight(1.5)
					if len(detail.Tracks) > 0 {
						ui.Textf(c, "%d 首 · %s", len(detail.Tracks), formatClock(tracksDuration(detail.Tracks))).
							FontSize(11).TextColor(p.faint)
						ui.Row(c).Margin(5, 0, 0, 0).Children(func() {
							a.primaryButton(c, "播放全部", iconPlay, func() {
								a.playTracks(detail.Tracks, 0)
							})
						})
					}
				})
			})
		})

		switch {
		case a.radioErr != "":
			ui.Row(c).FillWidth().Padding(18, 20).Radius(14).Background(p.panel).
				Border(1, p.line).AlignItems(ui.Center).Gap(14).Children(func() {
				ui.Box(c).Size(38, 38).Radius(12).Background(p.accent.Alpha(0.1)).Center().Children(func() {
					ui.Icon(c, iconRadio).Size(18, 18).TextColor(p.accent)
				})
				ui.Column(c).Grow(1).Gap(3).Children(func() {
					ui.Text(c, "这个电台暂时不能在桌面端播放").FontSize(12.5).FontWeight(650)
					ui.Text(c, a.radioErr).FontSize(11).TextColor(p.muted).LineHeight(1.5)
				})
				a.ghostButton(c, "打开咪咕音乐", iconExternal, func() {
					c.OpenURL("https://music.migu.cn/")
				})
			})
		case a.radioLoading && len(detail.Tracks) == 0:
			a.loadingPanel(c, p, "正在加载电台歌曲…", 180)
		case len(detail.Tracks) > 0:
			ui.Column(c).Gap(8).Children(func() {
				a.sectionTitle(c, p, "电台歌曲",
					fmt.Sprintf("%d 首 · %s", len(detail.Tracks), formatClock(tracksDuration(detail.Tracks))))
				ui.Column(c).Gap(2).Children(func() {
					for i := range detail.Tracks {
						a.songRow(c, p, detail.Tracks[i], detail.Tracks, i, "radio")
					}
				})
			})
		default:
			a.emptyPanel(c, p, iconMusic, "电台还在准备歌曲", "稍后刷新，或换一个场景电台。", "", nil)
		}
	})
}

func (a *app) homeRadioLists() (private, scene []miguRadio) {
	for _, group := range a.radioGroups {
		for _, radio := range group.Items {
			if radio.Kind == "private" {
				private = append(private, radio)
			} else if radio.Kind == "scene" {
				scene = append(scene, radio)
			}
		}
	}
	return private, scene
}

func (a *app) homeRadioByTitle(parts ...string) (miguRadio, bool) {
	private, scene := a.homeRadioLists()
	all := append(private, scene...)
	for _, part := range parts {
		for _, radio := range all {
			if strings.Contains(radio.Title, part) {
				return radio, true
			}
		}
	}
	return miguRadio{}, false
}

func (a *app) quickTile(c *ui.Context, p palette, title, subtitle string, ic *ui.SVG,
	cover string, tint ui.Color, onClick func()) {
	card := ui.ButtonBase(c).Key("quick-"+title).Column().Grow(1).Height(126).Radius(16).Clip().
		FocusRing(false).Label(title).Background(p.panel).Border(1, p.line)
	if card.Hovered() {
		card.Background(p.panelAlt)
	}
	card.Children(func() {
		ui.Box(c).Fill().Draw(func(pt *ui.Painter, r ui.Rect) {
			pt.FillGradient(r, ui.LinearGradient{Angle: 135, Stops: []ui.GradientStop{
				{Color: tint.Alpha(0.22), At: 0},
				{Color: ui.Transparent, At: 0.62},
			}}, 16)
			pt.Fill(ui.Rect{X: r.X + r.W*0.62, Y: r.Y - r.H*0.46, W: r.H * 1.35, H: r.H * 1.35},
				tint.Alpha(0.12), 999)
		})
		ui.Row(c).Fill().Padding(15, 16).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(5).AlignItems(ui.Start).Children(func() {
				ui.Box(c).Size(30, 30).Radius(10).Background(tint.Alpha(0.16)).Center().Children(func() {
					ui.Icon(c, ic).Size(15, 15).TextColor(tint)
				})
				ui.Text(c, title).FontSize(14).FontWeight(680).SingleLine().Margin(5, 0, 0, 0)
				ui.Text(c, subtitle).FontSize(10.5).TextColor(p.muted).SingleLine()
			})
			if cover != "" {
				ui.Box(c).Size(64, 64).Radius(12).Clip().Rotate(3).
					Shadow(0, 10, 22, -6, p.shadow).Children(func() {
					a.cover(c, cover, title, 12)
				})
			}
		})
	})
	if card.Clicked() {
		onClick()
	}
}

func (a *app) homeQuickTiles(c *ui.Context, p palette) {
	tracks := a.home.NewSongs
	cover := ""
	if len(tracks) > 0 {
		cover = a.hqCover(tracks[0])
	}
	ui.Row(c).Gap(14).Children(func() {
		a.quickTile(c, p, "今日推荐", "每日 50 首 · 从新歌开始", iconSpark,
			cover, p.accent, func() {
				if len(tracks) > 0 {
					a.playTracks(tracks, 0)
				} else {
					a.navTo(pageHome)
					a.loadMiguHome()
				}
			})
		for _, item := range []struct {
			title string
			parts []string
			ic    *ui.SVG
			tint  ui.Color
		}{
			{"私人 FM", []string{"私人", "FM"}, iconRadio, ui.Hex("#5bbd8a")},
			{"上瘾电台", []string{"上瘾"}, iconRadio, ui.Hex("#b58aef")},
			{"听见不同", []string{"听见", "不同"}, iconSpark, ui.Hex("#f4a26c")},
		} {
			radio, ok := a.homeRadioByTitle(item.parts...)
			item := item
			title := item.title
			subtitle := "打开咪咕电台"
			radioCover := ""
			if ok {
				title = radio.Title
				subtitle = firstNonEmpty(radio.Subtitle, item.title)
				radioCover = radio.Cover
			}
			a.quickTile(c, p, title, subtitle, item.ic, radioCover, item.tint, func() {
				if ok {
					a.openRadio(radio, pageHome)
				} else {
					a.navTo(pageRadios)
				}
			})
		}
	})
}

func (a *app) homePlaylists(c *ui.Context, p palette) {
	if len(a.home.Playlists) == 0 {
		return
	}
	ui.Column(c).Gap(14).Children(func() {
		ui.Row(c).AlignItems(ui.End).Children(func() {
			a.sectionTitle(c, p, "推荐歌单", "MIGU PLAYLIST")
			ui.Spacer(c)
			a.miniLink(c, p, "逛歌单广场", func() { a.navTo(pagePlaylists) })
		})
		ui.Grid(c).Columns(a.columns(c, 178)).GapX(18).GapY(24).Children(func() {
			for _, playlist := range a.home.Playlists {
				a.playlistCard(c, p, playlist, pageHome, "home")
			}
		})
	})
}

func playlistBackLabel(from pageKind) string {
	switch from {
	case pageSavedPlaylists:
		return "返回收藏歌单"
	case pageHome:
		return "返回发现"
	default:
		return "返回歌单广场"
	}
}

func radioBackLabel(from pageKind) string {
	if from == pageHome {
		return "返回发现"
	}
	return "返回电台"
}

func radioKindLabel(kind string) string {
	if kind == "private" {
		return "私人 FM"
	}
	if kind == "podcast" {
		return "播客"
	}
	return "场景电台"
}
