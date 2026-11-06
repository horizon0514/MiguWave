package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

func pageScroll(c *ui.Context, key string, gap float32, build func()) {
	ui.Scroll(c).Key("scroll-" + key).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).FillWidth().Padding(0, 42, 40, 42).Gap(gap).Children(build)
	})
}

func (a *app) homePage(c *ui.Context, p palette) {
	pageScroll(c, "home", 30, func() {
		a.hero(c, p)

		searches := a.hotSearches
		if len(searches) == 0 {
			searches = []string{"周杰伦", "林俊杰", "邓紫棋", "五月天", "陈奕迅", "薛之谦"}
		}
		ui.Row(c).Gap(8).Wrap().Children(func() {
			ui.Text(c, "大家都在搜").FontSize(11).FontWeight(650).TextColor(p.faint).
				Margin(0, 0, 8, 0).AlignSelf(ui.Center)
			for _, word := range searches {
				word := word
				a.chip(c, word, false, func() {
					a.searchInput = word
					a.runSearch(word)
				})
			}
		})

		a.homeQuickTiles(c, p)
		a.homePlaylists(c, p)

		ui.Column(c).Gap(16).Children(func() {
			a.sectionTitle(c, p, "新歌速递", "MIGU MUSIC")
			switch {
			case len(a.home.NewSongs) > 0:
				ui.Grid(c).Columns(a.columns(c, 168)).GapX(18).GapY(28).Children(func() {
					for i := range a.home.NewSongs {
						a.songCard(c, p, a.home.NewSongs[i], a.home.NewSongs, i, "new")
					}
				})
			case a.homeLoading:
				a.loadingPanel(c, p, "正在从咪咕取回今天的新歌…", 164)
			case a.homeErr != "":
				a.noticePanel(c, p, a.homeErr, "重新加载", a.loadMiguHome)
			default:
				a.emptyPanel(c, p, iconMusic, "新歌还在路上", "稍后刷新，咪咕曲库会自动回来。", "", nil)
			}
		})

		ui.Column(c).Gap(16).Children(func() {
			ui.Row(c).AlignItems(ui.End).Children(func() {
				a.sectionTitle(c, p, "热门榜单", "100 首一榜，听个够")
				ui.Spacer(c)
				a.miniLink(c, p, "查看全部", func() { a.navTo(pageRank) })
			})
			ui.Grid(c).Columns(min(3, a.columns(c, 280))).GapX(16).GapY(16).Children(func() {
				ranks := a.home.Ranks
				if len(ranks) == 0 {
					for _, info := range miguRankCatalog {
						ranks = append(ranks, miguRankSummary{
							ID:       info.ID,
							Title:    info.Title,
							Subtitle: info.Note,
						})
					}
				}
				for _, rank := range ranks {
					rank := rank
					a.homeRankCard(c, p, rank)
				}
			})
		})

		ui.Row(c).FillWidth().Padding(15, 18).Radius(14).Background(p.panel).
			Border(1, p.line).AlignItems(ui.Center).Gap(12).Children(func() {
			ui.Box(c).Size(34, 34).Radius(10).Background(p.accent.Alpha(0.12)).Center().Children(func() {
				ui.Icon(c, iconSpark).Size(17, 17).TextColor(p.accent)
			})
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				ui.Text(c, "登录后播放账号有权限的曲目").FontSize(12).FontWeight(600)
				ui.Text(c, "打开左下角账号入口，用咪咕音乐 App 扫码登录。").
					FontSize(11).TextColor(p.muted)
			})
			if a.miguToken == "" {
				a.ghostButton(c, "连接账号", nil, a.openAccount)
			} else {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(6, 6).Radius(999).Background(ui.Hex("#25b86b"))
					ui.Text(c, "咪咕账号已连接").FontSize(11.5).FontWeight(600).TextColor(p.muted)
				})
			}
		})
	})
}

func (a *app) hero(c *ui.Context, p palette) {
	track, tracks, index := a.heroTrack()
	card := ui.Box(c).FillWidth().Height(286).Radius(20).Clip().
		Gradient(p.accent, p.accent2, 32).Shadow(0, 18, 40, -18, p.accent.Alpha(0.45))
	card.Children(func() {
		ui.Box(c).Fill().Draw(func(pt *ui.Painter, r ui.Rect) {
			pt.Fill(ui.Rect{X: r.X + r.W*0.57, Y: r.Y - 90, W: 260, H: 260},
				ui.RGBA(255, 255, 255, 0.08), 999)
			pt.Fill(ui.Rect{X: r.X + r.W*0.78, Y: r.Y + 130, W: 190, H: 190},
				ui.RGBA(255, 255, 255, 0.07), 999)
		})
		ui.Row(c).Fill().Padding(30, 32, 26, 38).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(10).Children(func() {
				ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(24, 24).Radius(8).Background(ui.RGBA(255, 255, 255, 0.18)).
						Center().Children(func() {
						ui.Icon(c, iconMusic).Size(12, 12).TextColor(p.onAccent)
					})
					ui.Text(c, "MIGU MUSIC · 今日推荐").FontSize(10.5).FontWeight(750).
						LetterSpacing(1.4).TextColor(ui.RGBA(255, 255, 255, 0.78))
				})
				title := "把今天交给音乐"
				subtitle := "咪咕热歌、新歌与会员曲库，都在这里。"
				meta := ""
				if track != nil {
					title = track.Title
					subtitle = track.Artist
					if track.Album != "" {
						meta = track.Album
					}
					if track.VIP {
						meta = firstNonEmpty(joinMeta(meta, "VIP 曲目"), "VIP 曲目")
					}
				}
				ui.Text(c, title).FontSize(34).FontWeight(780).LetterSpacing(-0.9).
					TextColor(p.onAccent).MaxLines(2).LineHeight(1.08)
				if subtitle != "" {
					ui.Text(c, subtitle).FontSize(14).FontWeight(520).
						TextColor(ui.RGBA(255, 255, 255, 0.82)).SingleLine()
				}
				if meta != "" {
					ui.Text(c, meta).FontSize(11.5).TextColor(ui.RGBA(255, 255, 255, 0.62)).SingleLine()
				}
				ui.Row(c).Gap(9).Margin(8, 0, 0, 0).Children(func() {
					a.heroButton(c, "立即播放", iconPlay, func() {
						if track != nil {
							a.playTracks(tracks, index)
						}
					})
					a.heroGhostButton(c, "搜索咪咕曲库", iconSearchIcon, func() {
						a.focusSearch = true
						a.refresh()
					})
				})
			})
			ui.Box(c).WidthPercent(30).FillHeight().Center().Children(func() {
				cover := ui.Box(c).Size(208, 208).Radius(18).Clip().
					Shadow(0, 20, 38, -10, ui.RGBA(71, 9, 31, 0.38))
				cover.Children(func() {
					title := "MiguWave"
					raw, low := "", ""
					if track != nil {
						title = track.Title
						raw = a.hqCover(*track)
						low = track.Cover
					}
					a.coverPair(c, raw, low, title, 18)
				})
				cover.Rotate(-2.5)
			})
		})
	})
}

func (a *app) heroTrack() (*Track, []Track, int) {
	if len(a.home.NewSongs) > 0 {
		return &a.home.NewSongs[0], a.home.NewSongs, 0
	}
	for _, rank := range a.home.Ranks {
		if len(rank.Tracks) > 0 {
			track := rank.Tracks[0]
			return &track, rank.Tracks, 0
		}
	}
	return nil, nil, 0
}

func (a *app) heroButton(c *ui.Context, label string, ic *ui.SVG, onClick func()) {
	b := ui.ButtonBase(c).Key("hero-" + label).Height(38).PaddingX(19).Radius(999).
		Gap(8).AlignItems(ui.Center).FocusRing(false).Label(label).
		Background(ui.Hex("#ffffff")).TextColor(ui.Hex("#ff2667"))
	if b.Hovered() {
		b.Background(ui.Hex("#fff0f5"))
	}
	b.Children(func() {
		if ic != nil {
			ui.Icon(c, ic).Size(15, 15).TextColor(ui.Hex("#ff2667"))
		}
		ui.Text(c, label).FontSize(12.5).FontWeight(700).TextColor(ui.Hex("#ff2667"))
	})
	if b.Clicked() {
		onClick()
	}
}

func (a *app) heroGhostButton(c *ui.Context, label string, ic *ui.SVG, onClick func()) {
	b := ui.ButtonBase(c).Key("hero-ghost-"+label).Height(38).PaddingX(17).Radius(999).
		Gap(8).AlignItems(ui.Center).FocusRing(false).Label(label).
		Background(ui.RGBA(255, 255, 255, 0.12)).Border(1, ui.RGBA(255, 255, 255, 0.25))
	if b.Hovered() {
		b.Background(ui.RGBA(255, 255, 255, 0.18))
	}
	b.Children(func() {
		if ic != nil {
			ui.Icon(c, ic).Size(15, 15).TextColor(ui.Hex("#ffffff"))
		}
		ui.Text(c, label).FontSize(12.5).FontWeight(600).TextColor(ui.Hex("#ffffff"))
	})
	if b.Clicked() {
		onClick()
	}
}

func (a *app) homeRankCard(c *ui.Context, p palette, rank miguRankSummary) {
	card := ui.ButtonBase(c).Key("home-rank-"+rank.ID).Height(184).Radius(16).
		Padding(14).FocusRing(false).Label(rank.Title).Background(p.panel).Border(1, p.line)
	if card.Hovered() {
		card.Background(p.panelAlt)
	}
	card.Children(func() {
		ui.Row(c).Fill().Gap(14).Children(func() {
			ui.Box(c).Size(126, 126).Radius(13).Clip().Children(func() {
				a.cover(c, rank.Cover, rank.Title, 13)
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(6).Children(func() {
				ui.Text(c, rank.Title).FontSize(16).FontWeight(720).SingleLine()
				note := rank.Subtitle
				if note == "" {
					note = rankNote(rank.ID)
				}
				if note != "" {
					ui.Text(c, note).FontSize(10.5).FontWeight(650).TextColor(p.accent)
				}
				for i, track := range rank.Tracks {
					if i >= 3 {
						break
					}
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Textf(c, "%d", i+1).FontSize(10).FontWeight(700).
							TextColor(p.faint).Width(10)
						ui.Text(c, track.Title).FontSize(11.5).TextColor(p.muted).SingleLine()
					})
				}
				if len(rank.Tracks) == 0 {
					ui.Text(c, "打开榜单，加载 100 首热歌").FontSize(11.5).
						TextColor(p.faint).MaxLines(2)
				}
			})
		})
	})
	if card.Clicked() {
		a.page = pageRank
		a.loadRank(rank.ID)
	}
}

func (a *app) rankPage(c *ui.Context, p palette) {
	pageScroll(c, "rank-"+a.rankID, 24, func() {
		ui.Row(c).AlignItems(ui.End).DragWindow().Children(func() {
			ui.Column(c).Gap(4).Children(func() {
				ui.Text(c, "排行榜").FontSize(32).FontWeight(760).LetterSpacing(-0.7)
				ui.Text(c, "咪咕音乐榜单，每个榜单最多 100 首").FontSize(13).TextColor(p.muted)
			})
			ui.Spacer(c)
			if len(a.rank.Tracks) > 0 {
				a.primaryButton(c, "播放全部", iconPlay, func() { a.playTracks(a.rank.Tracks, 0) })
			}
		})

		ui.Row(c).Gap(8).Wrap().Children(func() {
			for _, info := range miguRankCatalog {
				info := info
				a.chip(c, info.Title, a.rankID == info.ID, func() {
					if a.rankID != info.ID {
						a.loadRank(info.ID)
					}
				})
			}
		})

		switch {
		case a.rankLoading && len(a.rank.Tracks) == 0:
			a.loadingPanel(c, p, "正在打开 "+rankTitle(a.rankID)+"…", 210)
		case a.rankErr != "" && len(a.rank.Tracks) == 0:
			a.noticePanel(c, p, a.rankErr, "重试", func() { a.loadRank(a.rankID) })
		default:
			a.rankHero(c, p)
			if len(a.rank.Tracks) > 0 {
				ui.Column(c).Gap(8).Children(func() {
					a.sectionTitle(c, p, "完整榜单", fmt.Sprintf("%d 首 · %s", len(a.rank.Tracks), formatClock(tracksDuration(a.rank.Tracks))))
					ui.Column(c).Gap(2).Children(func() {
						for i := range a.rank.Tracks {
							a.songRow(c, p, a.rank.Tracks[i], a.rank.Tracks, i, "rank")
						}
					})
				})
			}
		}
	})
}

func (a *app) rankHero(c *ui.Context, p palette) {
	rank := a.rank
	card := ui.Box(c).FillWidth().Height(226).Radius(18).Clip().
		Gradient(p.accent, p.accent2, 24).Shadow(0, 14, 34, -16, p.accent.Alpha(0.42))
	card.Children(func() {
		ui.Row(c).Fill().Padding(24, 28).Gap(24).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(156, 156).Radius(14).Clip().Children(func() {
				a.cover(c, rank.Cover, rank.Title, 14)
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(7).Children(func() {
				ui.Text(c, "MIGU CHART").FontSize(10).FontWeight(760).LetterSpacing(1.4).
					TextColor(ui.RGBA(255, 255, 255, 0.7))
				title := rank.Title
				if title == "" {
					title = rankTitle(a.rankID)
				}
				ui.Text(c, title).FontSize(28).FontWeight(760).LetterSpacing(-0.5).
					TextColor(p.onAccent).SingleLine()
				desc := strings.TrimSpace(rank.Description)
				if desc != "" {
					ui.Text(c, desc).FontSize(12).TextColor(ui.RGBA(255, 255, 255, 0.8)).
						MaxLines(2).LineHeight(1.5).MaxWidth(660)
				}
				meta := ""
				if rank.UpdateTime != "" {
					meta = "更新于 " + rank.UpdateTime
				}
				if len(rank.Tracks) > 0 {
					meta = joinMeta(meta, fmt.Sprintf("%d 首", len(rank.Tracks)))
				}
				if meta != "" {
					ui.Text(c, meta).FontSize(11).TextColor(ui.RGBA(255, 255, 255, 0.62))
				}
			})
		})
	})
}

func (a *app) searchPage(c *ui.Context, p palette) {
	pageScroll(c, "search", 24, func() {
		ui.Row(c).Height(34).AlignItems(ui.Center).DragWindow().Children(func() {
			a.backLink(c, p, searchBackLabel(a.searchFrom), func() {
				a.navTo(a.searchBackTarget())
			})
		})
		a.pageHeader(c, p, "搜索咪咕曲库", "免费曲目直接播放，登录后使用账号有权限的曲库")
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			field := ui.SearchField(c.Key("page-search"), &a.searchInput).Width(460).
				Placeholder("歌名、歌手、专辑").Label("搜索咪咕音乐")
			if field.Submitted() {
				a.runSearch(a.searchInput)
			}
			a.primaryButton(c, "搜索", iconSearchIcon, func() { a.runSearch(a.searchInput) })
		})

		if a.searchTerm == "" {
			searches := a.hotSearches
			if len(searches) == 0 {
				searches = []string{"周杰伦", "晴天", "林俊杰", "邓紫棋", "陈奕迅", "五月天", "薛之谦", "王菲"}
			}
			ui.Column(c).Gap(14).Children(func() {
				a.sectionTitle(c, p, "热门搜索", "MIGU")
				ui.Row(c).Gap(8).Wrap().Children(func() {
					for _, word := range searches {
						word := word
						a.chip(c, word, false, func() {
							a.searchInput = word
							a.runSearch(word)
						})
					}
				})
			})
			a.emptyPanel(c, p, iconSearchIcon, "找一首咪咕上的歌",
				"输入周杰伦、晴天、粤语、摇滚，结果会直接出现在这里。", "", nil)
			return
		}

		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Textf(c, "“%s”", a.searchTerm).FontSize(21).FontWeight(720).SingleLine()
			ui.Spacer(c)
			if len(a.searchTracks) > 0 {
				ui.Textf(c, "已加载 %d 首", len(a.searchTracks)).FontSize(11.5).TextColor(p.faint)
			}
		})

		switch {
		case a.searchLoading:
			a.loadingPanel(c, p, "正在搜索咪咕曲库…", 180)
		case a.searchErr != "" && len(a.searchTracks) == 0:
			a.emptyPanel(c, p, iconSearchIcon, "没有找到结果", a.searchErr, "换个关键词", func() {
				a.searchInput = ""
			})
		case len(a.searchTracks) > 0:
			ui.Grid(c).Columns(a.columns(c, 168)).GapX(18).GapY(28).Children(func() {
				for i := range a.searchTracks {
					a.songCard(c, p, a.searchTracks[i], a.searchTracks, i, "search")
				}
			})
			if a.searchHasMore || a.searchLoadingMore {
				ui.Row(c).Justify(ui.Center).Margin(4, 0, 10, 0).Children(func() {
					if a.searchLoadingMore {
						ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Spinner(c).FontSize(16).TextColor(p.accent)
							ui.Text(c, "继续加载…").FontSize(12).TextColor(p.muted)
						})
					} else {
						a.ghostButton(c, "加载更多", iconArrow, a.loadMoreSearch)
					}
				})
			} else {
				ui.Row(c).Justify(ui.Center).Margin(2, 0, 12, 0).Children(func() {
					ui.Text(c, "已经到底了").FontSize(11.5).TextColor(p.faint)
				})
			}
		}
	})
}

func searchBackLabel(from pageKind) string {
	switch from {
	case pageFavorites:
		return "返回我的收藏"
	case pageRecent:
		return "返回最近播放"
	case pageRank:
		return "返回排行榜"
	case pagePlaylists, pagePlaylistDetail:
		return "返回歌单广场"
	case pageSavedPlaylists:
		return "返回收藏歌单"
	case pageSingers, pageSingerDetail:
		return "返回歌手"
	case pageRadios, pageRadioDetail:
		return "返回电台"
	case pageLocalPlaylists, pageLocalPlaylistDetail:
		return "返回自建歌单"
	default:
		return "返回发现"
	}
}

func (a *app) libraryPage(c *ui.Context, p palette, title, subtitle string, tracks []Track, ic *ui.SVG) {
	pageScroll(c, title, 24, func() {
		a.pageHeader(c, p, title, subtitle)
		if len(tracks) == 0 {
			body := "收藏的咪咕歌曲会出现在这里。"
			if title == "最近播放" {
				body = "播放过的歌曲会按时间出现在这里。"
			}
			a.emptyPanel(c, p, ic, "这里还是空的", body, "去发现", func() { a.navTo(pageHome) })
			return
		}
		a.librarySummary(c, p, tracks, ic)
		ui.Column(c).Gap(12).Children(func() {
			a.sectionTitle(c, p, "歌曲", fmt.Sprintf("%d 首 · %s", len(tracks), formatClock(tracksDuration(tracks))))
			ui.Column(c).Gap(2).Children(func() {
				for i := range tracks {
					a.songRow(c, p, tracks[i], tracks, i, "library")
				}
			})
		})
	})
}

func (a *app) librarySummary(c *ui.Context, p palette, tracks []Track, ic *ui.SVG) {
	card := ui.Box(c).FillWidth().Height(94).Radius(16).Clip().Border(1, p.line)
	card.Background(p.panel)
	card.Children(func() {
		ui.Row(c).Fill().Padding(16, 19).Gap(14).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(54, 54).Radius(14).Background(p.accent.Alpha(0.12)).Children(func() {
				ui.Box(c).Fill().Center().Children(func() {
					ui.Icon(c, ic).Size(22, 22).TextColor(p.accent)
				})
			})
			ui.Column(c).Grow(1).Gap(3).Children(func() {
				ui.Textf(c, "%d 首歌曲", len(tracks)).FontSize(15).FontWeight(650)
				ui.Textf(c, "总时长 %s", formatClock(tracksDuration(tracks))).
					FontSize(12).TextColor(p.muted)
			})
			a.primaryButton(c, "播放全部", iconPlay, func() { a.playTracks(tracks, 0) })
		})
	})
}

func (a *app) songCard(c *ui.Context, p palette, track Track, tracks []Track, index int, scope string) {
	card := ui.ButtonBase(c).Key(scope + "-song-" + track.ID).Column().Gap(9).
		AlignItems(ui.Stretch).FocusRing(false).Label(track.Title)
	hovered := card.Hovered()
	card.Children(func() {
		art := ui.Box(c).AspectRatio(1).Radius(13).Clip()
		if hovered {
			art.Shadow(0, 14, 30, -8, p.shadow)
		} else {
			art.Shadow(0, 3, 10, -5, p.shadow)
		}
		art.Children(func() {
			cover, low := track.Cover, ""
			if scope == "new" {
				cover = a.hqCover(track)
				low = track.Cover
			}
			a.coverPair(c, cover, low, track.Title, 13)
			if track.VIP {
				ui.Box(c).Absolute().Top(9).Left(9).Padding(4, 7).Radius(999).
					Background(ui.RGBA(255, 38, 103, 0.92)).Children(func() {
					ui.Text(c, "VIP").FontSize(8).FontWeight(780).LetterSpacing(0.7).
						TextColor(ui.Hex("#ffffff"))
				})
			}
			if hovered {
				a.cardPlayButton(c, p, scope+"-play-"+track.ID, func() {
					a.playTracks(tracks, index)
				})
				a.songPlaylistOverlayButton(c, scope+"-add-"+track.ID, track)
				detail := ui.ButtonBase(c).Key(scope+"-info-"+track.ID).Absolute().Top(8).Right(8).
					Size(30, 30).Center().Radius(999).FocusRing(false).Label("查看歌词与详情").
					Background(ui.RGBA(18, 8, 13, 0.55))
				detail.Children(func() {
					ui.Icon(c, iconLyrics).Size(15, 15).TextColor(ui.Hex("#ffffff"))
				})
				if detail.Clicked() {
					a.openSong(track, tracks, index)
				}
			}
		})
		ui.Text(c, track.Title).FontSize(13).FontWeight(580).MaxLines(2).LineHeight(1.24)
		ui.Text(c, track.Artist).FontSize(11.5).TextColor(p.muted).SingleLine().Margin(-3, 0, 0, 0)
		meta := songMeta(track)
		if meta != "" {
			ui.Text(c, meta).FontSize(10.5).TextColor(p.faint).SingleLine().Margin(-3, 0, 0, 0)
		}
	})
	if card.Clicked() {
		a.playTracks(tracks, index)
	}
}

func (a *app) songRow(c *ui.Context, p palette, track Track, tracks []Track, index int, scope string) {
	playing := a.pl.hasCurrent && a.pl.current.ID == track.ID
	row := ui.ButtonBase(c).Key(scope + "-row-" + track.ID).Height(60).PaddingX(10).Gap(12).
		Radius(10).AlignItems(ui.Center).FocusRing(false).Label(track.Title)
	bg := ui.Transparent
	switch {
	case playing:
		bg = p.accent.Alpha(0.095)
	case row.Hovered():
		bg = p.hover
	}
	row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 100 * time.Millisecond}))
	row.Children(func() {
		ui.Box(c).Width(26).Center().Children(func() {
			switch {
			case playing:
				a.playingBars(c, p, a.pl.playing)
			case row.Hovered():
				ui.Icon(c, iconPlay).Size(12, 12).TextColor(p.text)
			default:
				ui.Textf(c, "%d", index+1).FontSize(11.5).FontFeatures("tnum").TextColor(p.faint)
			}
		})
		ui.Box(c).Size(42, 42).Radius(8).Clip().Children(func() {
			a.cover(c, track.Cover, track.Title, 8)
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
			ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
				title := p.text
				if playing {
					title = p.accent
				}
				ui.Text(c, track.Title).FontSize(13.5).FontWeight(570).SingleLine().TextColor(title)
				if track.VIP {
					ui.Box(c).Padding(3, 6).Radius(5).Background(p.accent.Alpha(0.12)).Children(func() {
						ui.Text(c, "VIP").FontSize(8).FontWeight(750).TextColor(p.accent)
					})
				}
			})
			meta := track.Artist
			if track.Album != "" {
				meta = joinMeta(meta, track.Album)
			}
			if meta != "" {
				ui.Text(c, meta).FontSize(11.5).TextColor(p.muted).SingleLine()
			}
		})
		if track.PlayCount != "" {
			ui.Text(c, track.PlayCount).FontSize(10.5).TextColor(p.faint).Width(74).
				TextAlign(ui.End).SingleLine()
		}
		if row.Hovered() {
			b := a.iconButton(c, scope+"-info-"+track.ID, iconLyrics, "查看歌词与详情", false)
			if b.Clicked() {
				a.openSong(track, tracks, index)
			}
			a.songPlaylistButton(c, scope+"-add-"+track.ID, track)
			if scope == "local" {
				a.removeFromLocalPlaylistButton(c, scope, track)
			}
		}
		fav := a.isFavorite(track.ID)
		if fav || row.Hovered() {
			ic := iconHeart
			if fav {
				ic = iconHeartFill
			}
			b := a.iconButton(c, scope+"-fav-"+track.ID, ic, favoriteLabel(fav), fav)
			if b.Clicked() {
				a.toggleFavorite(track)
			}
		}
		if track.DurationSec > 0 {
			ui.Text(c, formatClock(track.Duration())).FontSize(11.5).FontFeatures("tnum").
				TextColor(p.faint).Width(42).TextAlign(ui.End)
		}
	})
	if row.Clicked() {
		a.playTracks(tracks, index)
	}
}

func (a *app) accountModal(c *ui.Context, p palette) {
	if !a.accountOpen {
		return
	}
	modal := ui.Modal(c, &a.accountOpen, func() {
		ui.Column(c).Width(624).Gap(14).Children(func() {
			ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(42, 42).Radius(12).Gradient(p.accent, p.accent2, 35).Center().Children(func() {
					ui.Icon(c, iconSpark).Size(20, 20).TextColor(p.onAccent)
				})
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					ui.Text(c, "咪咕音乐授权").FontSize(19).FontWeight(750)
					ui.Text(c, "登录咪咕账号，完整播放会员与已购曲目").FontSize(11.5).TextColor(p.muted)
				})
				if a.miguToken != "" {
					ui.Row(c).Gap(6).Padding(5, 9).Radius(999).Background(ui.RGBA(37, 184, 107, 0.12)).
						AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(5, 5).Radius(999).Background(ui.Hex("#25b86b"))
						ui.Text(c, "已连接").FontSize(10.5).FontWeight(650).TextColor(ui.Hex("#1b8e52"))
					})
				}
			})

			if a.miguToken != "" &&
				(a.authPhase == miguAuthConnected || a.authPhase == miguAuthChecking) {
				a.accountConnectedPanel(c, p)
			} else {
				a.accountModeTabs(c, p)
				switch a.authMode {
				case authModeQR:
					a.accountQRPanel(c, p)
				case authModeToken:
					a.accountTokenPanel(c, p)
				default:
					a.accountBrowserPanel(c, p)
				}

				if a.authNotice != "" {
					color := p.muted
					if a.authPhase == miguAuthError || a.authPhase == miguAuthInvalid || a.authPhase == miguAuthQRExpired {
						color = p.accent
					}
					ui.Text(c, a.authNotice).FontSize(11).TextColor(color).
						LineHeight(1.45).FillWidth()
				}

				ui.Text(c, "登录在咪咕官方页面完成；客户端不会读取账号密码，只保存咪咕返回的登录令牌。").
					FontSize(10.5).TextColor(p.faint).LineHeight(1.45)
			}

			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "令牌只保存在系统安全存储（钥匙串 / 凭据管理器），不写入普通配置文件。").
					FontSize(10).TextColor(p.faint).Grow(1)
				if a.miguToken != "" {
					a.ghostButton(c, "清除授权", nil, a.clearMiguToken)
				}
				a.ghostButton(c, "完成", nil, func() {
					a.closeAccount()
				})
			})
		})
	})
	if modal.Dismissed() {
		a.closeAccount()
	}
}

func (a *app) accountModeTabs(c *ui.Context, p palette) {
	modes := []struct {
		mode  authMode
		label string
	}{
		{authModeBrowser, "应用内登录"},
		{authModeQR, "手机扫码"},
		{authModeToken, "令牌导入"},
	}
	ui.Row(c).FillWidth().Padding(4).Radius(13).Gap(4).Background(p.panelAlt).Children(func() {
		for _, option := range modes {
			active := a.authMode == option.mode
			b := ui.ButtonBase(c).Key("auth-mode-" + option.label).Grow(1).Height(32).Radius(9).
				Center().FocusRing(false).Label(option.label)
			if active {
				b.Background(p.panel).Shadow(0, 1, 3, 0, ui.RGBA(76, 20, 39, 0.10))
			} else if b.Hovered() {
				b.Background(p.hover)
			}
			b.Children(func() {
				color := p.muted
				weight := 600
				if active {
					color = p.accent
					weight = 680
				}
				ui.Text(c, option.label).FontSize(12).FontWeight(weight).TextColor(color)
			})
			if b.Clicked() {
				a.setAuthMode(option.mode)
			}
		}
	})
}

func (a *app) accountBrowserPanel(c *ui.Context, p palette) {
	ui.Column(c).FillWidth().Gap(9).Children(func() {
		ui.Row(c).FillWidth().Height(30).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(6, 6).Radius(999).Background(ui.Hex("#25b86b"))
			ui.Text(c, "咪咕官方登录页").FontSize(11).FontWeight(620).TextColor(p.muted)
			ui.Text(c, "账号 / 短信 / SIM 登录").FontSize(10.5).TextColor(p.faint)
			ui.Spacer(c)
			if a.loginViewAlive() {
				page := a.loginView.Page()
				back := a.iconButton(c, "auth-login-back", iconBack, "返回上一页", false).
					Disabled(!page.CanGoBack())
				if back.Clicked() {
					page.GoBack()
				}
				reload := a.iconButton(c, "auth-login-reload", iconRepeat, "刷新登录页", false)
				if reload.Clicked() {
					page.Reload()
				}
			}
		})

		box := ui.Box(c).FillWidth().Height(372).Radius(14).Clip().
			Border(1, p.line).Background(ui.Hex("#ffffff"))
		box.Children(func() {
			if a.loginViewAlive() {
				ui.WebView(c, a.loginView).Fill()
				return
			}
			ui.Column(c).Fill().Center().Gap(10).Children(func() {
				if a.authPhase == miguAuthBrowserLoading || a.authPhase == miguAuthChecking {
					ui.Spinner(c).FontSize(22).TextColor(p.accent)
					label := "正在打开咪咕登录页…"
					if a.authPhase == miguAuthChecking {
						label = "正在检查咪咕登录状态…"
					}
					ui.Text(c, label).FontSize(11.5).TextColor(p.muted)
					return
				}
				ui.Icon(c, iconGlobe).Size(30, 30).TextColor(p.faint)
				if a.authPhase == miguAuthError {
					ui.Text(c, "登录页没有打开").FontSize(12.5).FontWeight(640)
					ui.Text(c, "检查网络后重试，或改用手机扫码。").FontSize(10.5).TextColor(p.muted)
					a.primaryButton(c, "重新打开登录页", iconSpark, a.reloadLoginView)
					return
				}
				ui.Text(c, "咪咕官方登录页会显示在这里").FontSize(12.5).FontWeight(640)
				ui.Text(c, "支持账号、短信与 SIM 登录，登录完成后自动连接。").
					FontSize(10.5).TextColor(p.muted)
				a.primaryButton(c, "打开咪咕登录页", iconSpark, a.reloadLoginView)
			})
		})

		ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconClock).Size(12, 12).TextColor(p.faint)
			ui.Text(c, "登录完成后会自动连接；关闭授权面板或完成登录后，网页 Cookie 立即清除。").
				FontSize(10.5).TextColor(p.faint)
			ui.Spacer(c)
			a.miniLink(c, p, "改用手机扫码", func() {
				a.setAuthMode(authModeQR)
			})
		})
	})
}

func (a *app) accountTokenPanel(c *ui.Context, p palette) {
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		ui.Column(c).FillWidth().Gap(5).Padding(14, 13).Radius(13).Background(p.panelAlt).Children(func() {
			ui.Text(c, "手动导入 pacmtoken").FontSize(12.5).FontWeight(660)
			ui.Text(c, "在浏览器登录 music.migu.cn，打开开发者工具 → Application → Cookies，复制 pacmtoken；也可以直接粘贴完整 Cookie。").
				FontSize(10.5).TextColor(p.muted).LineHeight(1.55)
		})
		ui.TextInput(c.Key("migu-token"), &a.tokenDraft).Password().FillWidth().
			Placeholder("pacmtoken 或完整 Cookie").Label("咪咕 pacmtoken")
		ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
			a.primaryButton(c, "验证并保存", iconSpark, a.updateMiguToken)
			a.ghostButton(c, "从剪贴板导入", nil, a.importTokenFromClipboard)
			a.ghostButton(c, "打开登录页", iconExternal, func() {
				c.OpenURL("https://music.migu.cn/")
			})
		})
		ui.Text(c, "保存前会先向咪咕验证；无效或过期的令牌不会覆盖当前的授权。").
			FontSize(10.5).TextColor(p.faint)
	})
}

func (a *app) accountConnectedPanel(c *ui.Context, p palette) {
	ui.Column(c).FillWidth().Gap(13).Children(func() {
		ui.Row(c).FillWidth().Padding(16, 15).Radius(15).
			Background(ui.RGBA(37, 184, 107, 0.09)).Border(1, ui.RGBA(37, 184, 107, 0.18)).
			Gap(12).AlignItems(ui.Center).Children(func() {
			a.accountAvatar(c, p, 48, 15)
			ui.Column(c).Grow(1).Gap(3).Children(func() {
				title := "咪咕账号已连接"
				if a.miguProfile != nil && a.miguProfile.Nickname != "" {
					title = a.miguProfile.Nickname
				}
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, title).FontSize(15).FontWeight(700).SingleLine()
					if level := a.miguProfileLevel(); level != "" {
						ui.Box(c).Padding(3, 7).Radius(999).Background(p.accent.Alpha(0.1)).Children(func() {
							ui.Text(c, level).FontSize(9.5).FontWeight(680).TextColor(p.accent).SingleLine()
						})
					}
				})
				ui.Text(c, "咪咕账号已连接").FontSize(10.5).FontWeight(650).TextColor(ui.Hex("#1b8e52"))
				if meta := a.miguProfileMeta(); meta != "" {
					ui.Text(c, meta).FontSize(10.5).TextColor(p.muted).SingleLine()
				}
				if a.miguProfile != nil && a.miguProfile.MedalName != "" {
					ui.Text(c, "勋章 · "+a.miguProfile.MedalName).FontSize(10.5).TextColor(p.faint).SingleLine()
				}
			})
		})

		switch {
		case a.miguProfile == nil && a.miguProfileLoading:
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c).FontSize(13).TextColor(p.accent)
				ui.Text(c, "正在读取账号资料…").FontSize(10.5).TextColor(p.muted)
			})
		case a.miguProfile == nil && a.miguProfileErr != "":
			ui.Text(c, "账号资料暂时无法读取："+a.miguProfileErr).FontSize(10.5).TextColor(p.faint).
				LineHeight(1.45)
		}
		if a.authNotice != "" {
			ui.Text(c, a.authNotice).FontSize(10.5).TextColor(p.muted).LineHeight(1.45)
		}

		ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
			a.primaryButton(c, "重新登录", iconSpark, func() {
				a.authPhase = miguAuthIdle
				a.authNotice = ""
				a.enterAuthMode(a.authMode)
			})
			a.ghostButton(c, "检查连接", nil, a.verifyMiguToken)
			a.ghostButton(c, "打开咪咕网页", iconExternal, func() {
				c.OpenURL("https://music.migu.cn/")
			})
		})

		ui.Text(c, "“已连接”只代表咪咕登录有效；实际可播放范围仍由账号会员状态和单曲版权决定。").
			FontSize(10.5).TextColor(p.faint).LineHeight(1.5)
	})
}

func (a *app) accountAvatar(c *ui.Context, p palette, size, radius float32) {
	box := ui.Box(c).Size(size, size).Radius(radius).Clip()
	if a.miguProfile != nil && a.miguProfile.AvatarURL != "" {
		if src := a.arts[a.miguProfile.AvatarURL]; src != nil {
			box.Background(p.panelAlt).Children(func() {
				ui.Image(c, src).Fill().Fit(ui.Cover)
			})
			return
		}
		a.requestArt(a.miguProfile.AvatarURL)
	}
	box.Gradient(p.accent, p.accent2, 35).Center().Children(func() {
		if a.miguProfile != nil && a.miguProfile.Nickname != "" {
			ui.Text(c, miguProfileInitial(a.miguProfile.Nickname)).
				FontSize(size * 0.36).FontWeight(750).TextColor(p.onAccent)
			return
		}
		ui.Icon(c, iconSpark).Size(size*0.42, size*0.42).TextColor(p.onAccent)
	})
}

func (a *app) miguProfileLevel() string {
	if a.miguProfile == nil {
		return ""
	}
	return a.miguProfile.LevelName
}

func (a *app) miguProfileMeta() string {
	if a.miguProfile == nil {
		return ""
	}
	var parts []string
	if a.miguProfile.Phone != "" {
		parts = append(parts, a.miguProfile.Phone)
	}
	if a.miguProfile.UserID != "" {
		parts = append(parts, "UID "+a.miguProfile.UserID)
	}
	return strings.Join(parts, "  ·  ")
}

func (a *app) accountQRPanel(c *ui.Context, p palette) {
	ui.Row(c).FillWidth().Gap(20).AlignItems(ui.Center).Children(func() {
		a.accountQRArtwork(c, p)
		ui.Column(c).Grow(1).MinWidth(0).Gap(8).Children(func() {
			title, body := a.accountQRStatus()
			ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
				ui.Text(c, title).FontSize(17).FontWeight(720).SingleLine()
				ui.Box(c).Padding(3, 7).Radius(999).Background(p.accent.Alpha(0.1)).Children(func() {
					ui.Text(c, a.accountPhaseLabel()).FontSize(9.5).FontWeight(680).TextColor(p.accent)
				})
			})
			ui.Text(c, body).FontSize(11.5).TextColor(p.muted).LineHeight(1.55).MaxLines(4)
			if a.authPhase == miguAuthQRWaiting {
				ui.Text(c, "咪咕音乐 App → 我的 → 扫一扫").
					FontSize(10.5).FontWeight(580).TextColor(p.faint)
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Margin(4, 0, 0, 0).Children(func() {
				switch a.authPhase {
				case miguAuthQRWaiting, miguAuthQRScanned:
					a.primaryButton(c, "刷新二维码", iconSpark, a.refreshMiguQRLogin)
					a.ghostButton(c, "取消扫码", nil, a.cancelMiguQRLogin)
					a.ghostButton(c, "从剪贴板导入", nil, a.importTokenFromClipboard)
				case miguAuthQRLoading, miguAuthChecking, miguAuthVerifying:
					a.ghostButton(c, "取消", nil, a.cancelMiguQRLogin)
				default:
					a.primaryButton(c, "生成二维码", iconSpark, a.startMiguQRLogin)
					a.ghostButton(c, "从剪贴板导入", nil, a.importTokenFromClipboard)
					a.ghostButton(c, "应用内登录", nil, func() {
						a.setAuthMode(authModeBrowser)
					})
				}
			})
		})
	})
}

func (a *app) accountQRArtwork(c *ui.Context, p palette) {
	box := ui.Box(c).Size(188, 188).Radius(16).Background(ui.Hex("#ffffff")).
		Border(1, p.line).Center()
	if a.authQR != nil && (a.authPhase == miguAuthQRWaiting || a.authPhase == miguAuthQRScanned) {
		box.Padding(10).Clip().Children(func() {
			ui.Image(c, a.authQR).Fill().Fit(ui.Contain)
			if a.authPhase == miguAuthQRScanned {
				ui.Row(c).Absolute().Top(4).Right(4).Padding(4, 9).Radius(999).
					Background(ui.Hex("#25b86b")).Gap(4).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "✓").FontSize(9.5).FontWeight(750).TextColor(ui.Hex("#ffffff"))
					ui.Text(c, "已扫码").FontSize(9.5).FontWeight(650).TextColor(ui.Hex("#ffffff"))
				})
			}
		})
		return
	}
	box.Background(p.panelAlt).Children(func() {
		ui.Column(c).Gap(9).AlignItems(ui.Center).Children(func() {
			switch a.authPhase {
			case miguAuthQRLoading, miguAuthChecking, miguAuthVerifying:
				ui.Spinner(c).FontSize(24).TextColor(p.accent)
			case miguAuthQRExpired:
				ui.Icon(c, iconClock).Size(32, 32).TextColor(p.faint)
			default:
				ui.Icon(c, iconSpark).Size(32, 32).TextColor(p.accent)
			}
			ui.Text(c, a.accountPhaseShort()).FontSize(10.5).FontWeight(600).TextColor(p.muted)
		})
	})
}

func (a *app) accountQRStatus() (string, string) {
	switch a.authPhase {
	case miguAuthQRLoading:
		return "正在生成二维码", "正在连接咪咕登录服务，通常只需要几秒。"
	case miguAuthChecking:
		return "正在检查登录状态", "正在向咪咕确认当前授权是否仍然有效。"
	case miguAuthQRWaiting:
		return "请用咪咕音乐扫码", "扫码后请直接在手机上确认，桌面端会自动完成登录。"
	case miguAuthQRScanned:
		return "已扫码，请在手机上确认", "确认后会自动读取登录状态，不需要再刷新二维码。"
	case miguAuthVerifying:
		return "正在完成授权", "已经确认登录，正在取得 pacmtoken 并检查连接。"
	case miguAuthQRExpired:
		return "二维码已过期", "为了安全，登录二维码会定时失效。刷新后再扫一次即可。"
	case miguAuthInvalid:
		return "登录状态已失效", "重新扫码，或从剪贴板导入新的 pacmtoken。"
	case miguAuthError:
		return "授权没有完成", "检查网络后重试，也可以改用剪贴板或高级手动输入。"
	case miguAuthBrowserLoading:
		return "正在打开登录页", "正在载入咪咕官方登录页，通常只需要几秒。"
	case miguAuthBrowserWaiting:
		return "请在上方登录", "登录完成后会自动连接，不需要再复制令牌。"
	default:
		return "登录咪咕账号", "可以应用内登录、用咪咕 App 扫码，或导入 pacmtoken。"
	}
}

func (a *app) accountPhaseLabel() string {
	switch a.authPhase {
	case miguAuthQRLoading:
		return "准备中"
	case miguAuthChecking:
		return "校验中"
	case miguAuthQRWaiting:
		return "等待扫码"
	case miguAuthQRScanned:
		return "待确认"
	case miguAuthVerifying:
		return "登录中"
	case miguAuthQRExpired:
		return "已过期"
	case miguAuthInvalid:
		return "已失效"
	case miguAuthError:
		return "需重试"
	case miguAuthBrowserLoading:
		return "载入中"
	case miguAuthBrowserWaiting:
		return "等待登录"
	default:
		return "登录"
	}
}

func (a *app) accountPhaseShort() string {
	switch a.authPhase {
	case miguAuthQRLoading:
		return "生成二维码"
	case miguAuthChecking:
		return "检查授权"
	case miguAuthVerifying:
		return "正在登录"
	case miguAuthQRExpired:
		return "二维码过期"
	case miguAuthInvalid, miguAuthError:
		return "需要重新授权"
	case miguAuthBrowserLoading:
		return "正在载入"
	case miguAuthBrowserWaiting:
		return "请登录"
	default:
		return "登录咪咕"
	}
}

func (a *app) pageHeader(c *ui.Context, p palette, title, subtitle string) {
	ui.Row(c).AlignItems(ui.End).DragWindow().Children(func() {
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, title).FontSize(32).FontWeight(760).LetterSpacing(-0.7)
			if subtitle != "" {
				ui.Text(c, subtitle).FontSize(13).TextColor(p.muted)
			}
		})
	})
}

func (a *app) sectionTitle(c *ui.Context, p palette, title, meta string) {
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(3).Height(18).Radius(999).Background(p.accent)
		ui.Text(c, title).FontSize(19).FontWeight(720).LetterSpacing(-0.3).Margin(0, 0, 0, 10)
		ui.Spacer(c)
		if meta != "" {
			ui.Text(c, meta).FontSize(10.5).FontWeight(600).LetterSpacing(0.5).TextColor(p.faint)
		}
	})
}

func (a *app) columns(c *ui.Context, minWidth float32) int {
	w, _ := c.Size()
	usable := float32(w - sidebarWidth - 84)
	n := int(usable / (minWidth + 20))
	return min(6, max(2, n))
}

func (a *app) cover(c *ui.Context, rawURL, fallback string, radius float32) ui.Element {
	return a.coverPair(c, rawURL, "", fallback, radius)
}

func (a *app) coverPair(c *ui.Context, primary, secondary, fallback string, radius float32) ui.Element {
	a.requestArt(primary)
	if secondary != "" && secondary != primary {
		a.requestArt(secondary)
	}
	box := ui.Box(c).Fill().Radius(radius).Clip()
	var src ui.ImageSource
	if primary != "" {
		src = a.arts[primary]
	}
	if src == nil && secondary != "" {
		src = a.arts[secondary]
	}
	if src != nil {
		box.Children(func() {
			ui.Image(c, src).Fill().Fit(ui.Cover)
		})
		return box
	}
	p := paletteFor(c.Theme())
	box.Background(p.panelAlt)
	box.Children(func() {
		ui.Box(c).Fill().Draw(func(pt *ui.Painter, r ui.Rect) {
			size := min(r.W, r.H) * 0.28
			pt.Icon(iconMusic, ui.Rect{
				X: r.X + (r.W-size)/2,
				Y: r.Y + (r.H-size)/2,
				W: size,
				H: size,
			}, p.faint)
		})
	})
	return box
}

func (a *app) cardPlayButton(c *ui.Context, p palette, key any, play func()) {
	b := ui.ButtonBase(c).Key(key).Absolute().Bottom(9).Right(9).Size(38, 38).Radius(999).
		Center().FocusRing(false).Label("播放").Background(p.accent).TextColor(p.onAccent).
		Shadow(0, 5, 14, 0, ui.RGBA(71, 9, 31, 0.3))
	if b.Hovered() {
		b.Background(p.accent.Mix(ui.RGB(255, 255, 255), 0.12))
	}
	b.Children(func() {
		ui.Icon(c, iconPlay).Size(14, 14).TextColor(p.onAccent).Margin(0, 0, 0, 1)
	})
	if b.Clicked() {
		play()
	}
}

func (a *app) chip(c *ui.Context, label string, selected bool, onClick func()) ui.Element {
	return a.chipLabeled(c, label, label, selected, onClick)
}

func (a *app) chipLabeled(c *ui.Context, label, accessible string, selected bool, onClick func()) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Key("chip-" + accessible).Height(30).PaddingX(13).Radius(999).
		AlignItems(ui.Center).FocusRing(false).Label(accessible)
	bg := t.Surface
	color := t.TextMuted
	if selected {
		bg = t.Accent
		color = t.AccentText
	} else if b.Hovered() {
		bg = t.SurfaceHover
		color = t.Text
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 100 * time.Millisecond}))
	b.Children(func() {
		ui.Text(c, label).FontSize(11.5).FontWeight(560).TextColor(color)
	})
	if b.Clicked() {
		onClick()
	}
	return b
}

func (a *app) primaryButton(c *ui.Context, label string, ic *ui.SVG, onClick func()) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Key("primary-" + label).Height(36).PaddingX(17).Radius(999).
		Gap(8).AlignItems(ui.Center).FocusRing(false).Label(label).Background(t.Accent)
	if b.Hovered() {
		b.Background(t.AccentHover)
	}
	b.Children(func() {
		if ic != nil {
			ui.Icon(c, ic).Size(15, 15).TextColor(t.AccentText)
		}
		ui.Text(c, label).FontSize(12.5).FontWeight(650).TextColor(t.AccentText)
	})
	if b.Clicked() {
		onClick()
	}
	return b
}

func (a *app) ghostButton(c *ui.Context, label string, ic *ui.SVG, onClick func()) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Key("ghost-" + label).Height(36).PaddingX(16).Radius(999).
		Gap(8).AlignItems(ui.Center).FocusRing(false).Label(label).
		Background(t.Surface)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		if ic != nil {
			ui.Icon(c, ic).Size(15, 15).TextColor(t.TextMuted)
		}
		ui.Text(c, label).FontSize(12.5).FontWeight(600)
	})
	if b.Clicked() {
		onClick()
	}
	return b
}

func (a *app) miniLink(c *ui.Context, p palette, label string, onClick func()) ui.Element {
	b := ui.ButtonBase(c).Key("link-" + label).Height(26).PaddingX(4).Gap(4).
		AlignItems(ui.Center).FocusRing(false).Label(label).Background(ui.Transparent)
	b.Children(func() {
		ui.Text(c, label).FontSize(11.5).FontWeight(600).TextColor(p.accent)
		ui.Icon(c, iconArrow).Size(12, 12).TextColor(p.accent)
	})
	if b.Clicked() {
		onClick()
	}
	return b
}

func (a *app) loadingPanel(c *ui.Context, p palette, label string, height float32) {
	ui.Box(c).FillWidth().Height(height).Radius(16).Background(p.panel).
		Border(1, p.line).Children(func() {
		ui.Column(c).Fill().Center().Gap(10).Children(func() {
			ui.Spinner(c).FontSize(21).TextColor(p.accent)
			ui.Text(c, label).FontSize(12).TextColor(p.muted)
		})
	})
}

func (a *app) noticePanel(c *ui.Context, p palette, message, action string, onClick func()) {
	ui.Row(c).FillWidth().Padding(16, 18).Radius(14).Background(p.panel).
		Border(1, p.line).AlignItems(ui.Center).Gap(12).Children(func() {
		ui.Icon(c, iconSpark).Size(18, 18).TextColor(p.accent)
		ui.Text(c, message).FontSize(12.5).TextColor(p.muted).Grow(1)
		if action != "" && onClick != nil {
			a.ghostButton(c, action, nil, onClick)
		}
	})
}

func (a *app) emptyPanel(c *ui.Context, p palette, ic *ui.SVG, title, body, action string, onClick func()) {
	ui.Box(c).FillWidth().Height(220).Radius(16).Background(p.panel).
		Border(1, p.line).Children(func() {
		ui.Column(c).Fill().Center().Gap(8).Children(func() {
			ui.Box(c).Size(46, 46).Radius(14).Background(p.accent.Alpha(0.12)).Children(func() {
				ui.Box(c).Fill().Center().Children(func() {
					ui.Icon(c, ic).Size(21, 21).TextColor(p.accent)
				})
			})
			ui.Text(c, title).FontSize(16).FontWeight(680).Margin(3, 0, 0, 0)
			ui.Text(c, body).FontSize(12).TextColor(p.muted).TextAlign(ui.Center)
			if action != "" && onClick != nil {
				a.ghostButton(c, action, iconArrow, onClick)
			}
		})
	})
}

func songMeta(track Track) string {
	parts := make([]string, 0, 3)
	if track.DurationSec > 0 {
		parts = append(parts, formatClock(track.Duration()))
	}
	if track.PlayCount != "" {
		parts = append(parts, track.PlayCount)
	}
	if track.VIP {
		parts = append(parts, "会员")
	}
	return strings.Join(parts, " · ")
}

func joinMeta(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, " · ")
}

func tracksDuration(tracks []Track) time.Duration {
	var total time.Duration
	for _, track := range tracks {
		total += track.Duration()
	}
	return total
}
