package main

import (
	"context"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type pageKind int

const (
	pageHome pageKind = iota
	pageRank
	pageFavorites
	pageRecent
	pageSearch
	pageSong
	pagePlaylists
	pagePlaylistDetail
	pageSavedPlaylists
	pageSingers
	pageSingerDetail
	pageRadios
	pageRadioDetail
	pageLocalPlaylists
	pageLocalPlaylistDetail
	pageSettings
)

type app struct {
	win *mygo.Window
	pl  *player

	tray          *mygo.Tray
	menuBarPanel  *mygo.Window
	quitting      bool
	closeAction   closeAction
	launchAtLogin bool

	menuDetail        miguSongDetail
	menuDetailID      string
	menuDetailLoading bool
	menuDetailErr     string
	menuDetailGen     int

	volume         float64
	favorites      []Track
	recent         []Track
	savedPlaylists []miguPlaylist
	localPlaylists []localPlaylist

	miguToken    string
	accountOpen  bool
	tokenDraft   string
	authPhase    miguAuthPhase
	authGen      int
	authCancel   context.CancelFunc
	authQR       ui.ImageSource
	authNotice   string
	authMode     authMode
	loginView    *mygo.WebView
	loginSession *mygo.Session
	loginNav     chan string

	miguProfile        *miguProfile
	miguProfileLoading bool
	miguProfileErr     string

	page        pageKind
	toast       string
	focusSearch bool

	home        miguHomeData
	homeLoading bool
	homeErr     string
	homeGen     int

	hotSearches []string
	hotLoading  bool

	rankID      string
	rank        miguRank
	rankLoading bool
	rankErr     string
	rankGen     int

	playlistSections      []miguPlaylistSection
	playlistSquareLoaded  bool
	playlistSquareLoading bool
	playlistSquareErr     string
	playlistSquareGen     int

	playlistFrom        pageKind
	playlistID          string
	playlistDetail      miguPlaylistDetail
	playlistLoading     bool
	playlistLoadingMore bool
	playlistErr         string
	playlistGen         int

	singers        []miguSinger
	singerTab      string
	singerVisible  int
	singersLoaded  bool
	singersLoading bool
	singersErr     string
	singersGen     int

	singerDetail           miguSingerDetail
	singerLoading          bool
	singerLoadingMore      bool
	singerSongsLoadingMore bool
	singerErr              string
	singerGen              int

	radioGroups   []miguRadioGroup
	radiosLoaded  bool
	radiosLoading bool
	radiosErr     string
	radiosGen     int

	radioFrom    pageKind
	radioDetail  miguRadioDetail
	radioLoading bool
	radioErr     string
	radioGen     int

	localPlaylistID string

	localPlaylistEditorOpen      bool
	localPlaylistEditorID        string
	localPlaylistEditorName      string
	localPlaylistEditorErr       string
	localPlaylistEditorTrack     Track
	localPlaylistEditorHasTrack  bool
	localPlaylistEditorOpenAfter bool

	localPlaylistDeleteOpen bool
	localPlaylistDeleteID   string

	searchInput       string
	searchTerm        string
	searchFrom        pageKind
	searchTracks      []Track
	searchLoading     bool
	searchLoadingMore bool
	searchHasMore     bool
	searchPageNo      int
	searchErr         string
	searchGen         int

	songTrack   Track
	songDetail  miguSongDetail
	songQueue   []Track
	songIndex   int
	songFrom    pageKind
	songLoading bool
	songErr     string
	songGen     int
	songCache   map[string]miguSongDetail
	songPane    songPane

	envelope      *audioEnvelope
	envelopeTrack string

	lyricActive int
	lyricAuto   bool
	lyricScroll ui.ScrollState
	lyricGlide  lyricGlide
	lyricTrack  string
	lyricAt     int

	scrub       float64
	scrubHeld   bool
	scrubActive bool

	arts         map[string]ui.ImageSource
	artLoad      map[string]bool
	artFail      map[string]bool
	artTints     map[string]ui.Color
	artBackdrops map[string]*ui.Bitmap
	coverHQ      map[string]string
	coverHQLoad  map[string]bool
	coverHQFail  map[string]bool
}

type palette struct {
	bg, sidebar, panel, panelAlt, hover, line ui.Color
	text, muted, faint                        ui.Color
	accent, accent2, onAccent                 ui.Color
	shadow                                    ui.Color
}

func paletteFor(t *ui.Theme) palette {
	if t.Dark {
		return palette{
			bg:       ui.Hex("#151114"),
			sidebar:  ui.Hex("#1e161a"),
			panel:    ui.Hex("#23191e"),
			panelAlt: ui.Hex("#2c2026"),
			hover:    ui.RGBA(255, 255, 255, 0.06),
			line:     ui.RGBA(255, 255, 255, 0.09),
			text:     ui.Hex("#fff7fa"),
			muted:    ui.RGBA(255, 235, 242, 0.62),
			faint:    ui.RGBA(255, 235, 242, 0.34),
			accent:   ui.Hex("#ff4c80"),
			accent2:  ui.Hex("#ff9a64"),
			onAccent: ui.Hex("#ffffff"),
			shadow:   ui.RGBA(0, 0, 0, 0.5),
		}
	}
	return palette{
		bg:       ui.Hex("#fffbfc"),
		sidebar:  ui.Hex("#fff4f7"),
		panel:    ui.Hex("#fff7f9"),
		panelAlt: ui.Hex("#ffedf3"),
		hover:    ui.RGBA(255, 38, 103, 0.055),
		line:     ui.RGBA(255, 38, 103, 0.11),
		text:     ui.Hex("#24181d"),
		muted:    ui.Hex("#7d6971"),
		faint:    ui.Hex("#b49da6"),
		accent:   ui.Hex("#ff2667"),
		accent2:  ui.Hex("#ff8a5c"),
		onAccent: ui.Hex("#ffffff"),
		shadow:   ui.RGBA(76, 20, 39, 0.16),
	}
}

func (a *app) post(fn func()) {
	if a.win == nil {
		return
	}
	a.win.Update(fn)
}

func (a *app) refresh() {
	if a.win != nil {
		a.win.Invalidate()
	}
	if a.menuBarPanel != nil {
		a.menuBarPanel.Invalidate()
	}
}

func (a *app) start() {
	a.loadMiguHome()
	a.loadHotSearches()
	a.loadRadios()
	if a.miguToken != "" {
		a.loadMiguProfile(a.miguToken)
	}
}

func (a *app) loadMiguHome() {
	if a.homeLoading {
		return
	}
	a.homeGen++
	gen := a.homeGen
	a.homeLoading = true
	a.homeErr = ""
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		home, err := fetchMiguHome(ctx)
		a.post(func() {
			if a.homeGen != gen {
				return
			}
			a.homeLoading = false
			if err != nil {
				a.homeErr = friendlyError(err)
				return
			}
			a.home = home
		})
	}()
}

func (a *app) loadHotSearches() {
	if a.hotLoading {
		return
	}
	a.hotLoading = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		words, err := fetchMiguHotSearches(ctx)
		a.post(func() {
			a.hotLoading = false
			if err == nil && len(words) > 0 {
				a.hotSearches = words
			}
		})
	}()
}

func (a *app) loadRank(id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		id = miguRankCatalog[0].ID
	}
	a.rankID = id
	a.rankGen++
	gen := a.rankGen
	a.rankLoading = true
	a.rankErr = ""
	a.rank = miguRank{ID: id, Title: rankTitle(id)}
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		rank, err := fetchMiguRank(ctx, id, 100)
		a.post(func() {
			if a.rankGen != gen {
				return
			}
			a.rankLoading = false
			if err != nil {
				a.rankErr = friendlyError(err)
				return
			}
			a.rank = rank
		})
	}()
}

func (a *app) loadPlaylistSquare() {
	if a.playlistSquareLoading {
		return
	}
	a.playlistSquareGen++
	gen := a.playlistSquareGen
	a.playlistSquareLoading = true
	a.playlistSquareErr = ""
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		sections, err := fetchMiguPlaylistSquare(ctx)
		a.post(func() {
			if a.playlistSquareGen != gen {
				return
			}
			a.playlistSquareLoading = false
			if err != nil {
				a.playlistSquareErr = friendlyError(err)
				return
			}
			a.playlistSections = sections
			a.playlistSquareLoaded = true
			if len(sections) == 0 {
				a.playlistSquareErr = "咪咕歌单广场暂时没有返回内容"
			}
		})
	}()
}

func (a *app) openPlaylist(p miguPlaylist, from pageKind) {
	if strings.TrimSpace(p.ID) == "" {
		return
	}
	if from == pagePlaylistDetail {
		from = pagePlaylists
	}
	a.playlistFrom = from
	a.playlistID = p.ID
	a.playlistDetail = miguPlaylistDetail{Playlist: p}
	a.page = pagePlaylistDetail
	a.loadPlaylistDetail()
}

func (a *app) loadPlaylistDetail() {
	id := strings.TrimSpace(a.playlistID)
	if id == "" {
		return
	}
	a.playlistGen++
	gen := a.playlistGen
	a.playlistLoading = true
	a.playlistLoadingMore = false
	a.playlistErr = ""
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		detail, err := fetchMiguPlaylistDetail(ctx, id, 1)
		a.post(func() {
			if a.playlistGen != gen || a.playlistID != id {
				return
			}
			a.playlistLoading = false
			if err != nil {
				a.playlistErr = friendlyError(err)
				return
			}
			a.playlistDetail = detail
		})
	}()
}

func (a *app) loadMorePlaylist() {
	if a.playlistLoading || a.playlistLoadingMore || !a.playlistDetail.HasMore {
		return
	}
	id := a.playlistID
	page := a.playlistDetail.Page + 1
	gen := a.playlistGen
	a.playlistLoadingMore = true
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		tracks, total, hasMore, err := fetchMiguPlaylistSongs(ctx, id, page)
		a.post(func() {
			if a.playlistGen != gen || a.playlistID != id {
				return
			}
			a.playlistLoadingMore = false
			if err != nil {
				a.toast = friendlyError(err)
				return
			}
			a.playlistDetail.Tracks = miguDedupeTracks(append(a.playlistDetail.Tracks, tracks...))
			a.playlistDetail.Page = page
			a.playlistDetail.HasMore = hasMore
			if total > 0 {
				a.playlistDetail.Total = total
			}
		})
	}()
}

func (a *app) loadSingers(tab string) {
	tab = strings.TrimSpace(tab)
	if tab == "" {
		tab = firstNonEmpty(a.singerTab, miguSingerTabs[0].ID)
	}
	if a.singerTab != tab {
		a.singers = nil
		a.singersLoaded = false
	}
	a.singerTab = tab
	a.singerVisible = 30
	a.singersGen++
	gen := a.singersGen
	a.singersLoading = true
	a.singersErr = ""
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		singers, err := fetchMiguSingers(ctx, tab)
		a.post(func() {
			if a.singersGen != gen || a.singerTab != tab {
				return
			}
			a.singersLoading = false
			if err != nil {
				a.singersErr = friendlyError(err)
				return
			}
			a.singers = singers
			a.singersLoaded = true
			if len(singers) == 0 {
				a.singersErr = "这个分类暂时没有歌手"
			}
		})
	}()
}

func (a *app) openSinger(singer miguSinger) {
	if strings.TrimSpace(singer.ID) == "" {
		return
	}
	a.singerDetail = miguSingerDetail{Singer: singer}
	a.page = pageSingerDetail
	a.loadSingerDetail()
}

func (a *app) loadSingerDetail() {
	singer := a.singerDetail.Singer
	if strings.TrimSpace(singer.ID) == "" {
		return
	}
	a.singerGen++
	gen := a.singerGen
	a.singerLoading = true
	a.singerLoadingMore = false
	a.singerErr = ""
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		detail, err := fetchMiguSingerDetail(ctx, singer)
		a.post(func() {
			if a.singerGen != gen || a.singerDetail.Singer.ID != singer.ID {
				return
			}
			a.singerLoading = false
			if err != nil {
				a.singerErr = friendlyError(err)
				return
			}
			a.singerDetail = detail
		})
	}()
}

func (a *app) loadMoreSingerSongs() {
	if a.singerLoading || a.singerLoadingMore || !a.singerDetail.HasMore {
		return
	}
	id := a.singerDetail.Singer.ID
	page := a.singerDetail.Page + 1
	gen := a.singerGen
	a.singerLoadingMore = true
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		tracks, hasMore, err := fetchMiguSingerSongs(ctx, id, page)
		a.post(func() {
			if a.singerGen != gen || a.singerDetail.Singer.ID != id {
				return
			}
			a.singerLoadingMore = false
			if err != nil {
				a.toast = friendlyError(err)
				return
			}
			a.singerDetail.Tracks = miguDedupeTracks(append(a.singerDetail.Tracks, tracks...))
			a.singerDetail.Page = page
			a.singerDetail.HasMore = hasMore
		})
	}()
}

func (a *app) loadRadios() {
	if a.radiosLoading {
		return
	}
	a.radiosGen++
	gen := a.radiosGen
	a.radiosLoading = true
	a.radiosErr = ""
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		groups, err := fetchMiguRadioGroups(ctx)
		a.post(func() {
			if a.radiosGen != gen {
				return
			}
			a.radiosLoading = false
			if err != nil {
				a.radiosErr = friendlyError(err)
				return
			}
			a.radioGroups = groups
			a.radiosLoaded = true
			if len(groups) == 0 {
				a.radiosErr = "咪咕电台暂时没有返回内容"
			}
		})
	}()
}

func (a *app) openRadio(radio miguRadio, from pageKind) {
	if strings.TrimSpace(radio.ID) == "" {
		return
	}
	if from == pageRadioDetail {
		from = pageRadios
	}
	a.radioFrom = from
	a.radioDetail = miguRadioDetail{Radio: radio}
	a.page = pageRadioDetail
	a.loadRadioDetail()
}

func (a *app) loadRadioDetail() {
	radio := a.radioDetail.Radio
	if strings.TrimSpace(radio.ID) == "" {
		return
	}
	if radio.Kind == "private" {
		a.radioLoading = false
		a.radioErr = "私人 FM 需要咪咕个性化推荐接口，桌面端暂时只接入了场景电台。"
		a.refresh()
		return
	}
	a.radioGen++
	gen := a.radioGen
	a.radioLoading = true
	a.radioErr = ""
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		detail, err := fetchMiguRadioDetail(ctx, radio, 80)
		a.post(func() {
			if a.radioGen != gen || a.radioDetail.Radio.ID != radio.ID {
				return
			}
			a.radioLoading = false
			if err != nil {
				a.radioErr = friendlyError(err)
				return
			}
			a.radioDetail = detail
			if len(detail.Tracks) == 0 {
				a.radioErr = "这个电台暂时没有返回歌曲"
			}
		})
	}()
}

func (a *app) runSearch(q string) {
	q = strings.TrimSpace(q)
	if q == "" {
		a.toast = "输入歌名、歌手或专辑后再搜索"
		a.refresh()
		return
	}
	a.searchInput = q
	a.searchTerm = q
	if a.page != pageSearch {
		a.searchFrom = a.page
	}
	a.page = pageSearch
	a.searchGen++
	gen := a.searchGen
	a.searchLoading = true
	a.searchLoadingMore = false
	a.searchErr = ""
	a.searchTracks = nil
	a.searchPageNo = 0
	a.searchHasMore = false
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		var (
			tracks  []Track
			lastErr error
			pages   int
			more    bool
		)
		for page := 1; page <= 2; page++ {
			batch, err := searchMigu(ctx, q, page, 20)
			if err != nil {
				if page == 1 {
					lastErr = err
				}
				break
			}
			tracks = miguDedupeTracks(append(tracks, batch...))
			pages = page
			more = len(batch) == 20
			if !more {
				break
			}
		}
		a.post(func() {
			if a.searchGen != gen {
				return
			}
			a.searchLoading = false
			if lastErr != nil {
				a.searchErr = friendlyError(lastErr)
				return
			}
			a.searchTracks = tracks
			a.searchPageNo = max(pages, 1)
			a.searchHasMore = more
			if len(tracks) == 0 {
				a.searchErr = "咪咕曲库里没有找到结果，换个关键词试试"
			}
		})
	}()
}

func (a *app) searchBackTarget() pageKind {
	if a.searchFrom == pageSearch || a.searchFrom == pageSong {
		return pageHome
	}
	return a.searchFrom
}

func (a *app) loadMoreSearch() {
	if a.searchLoading || a.searchLoadingMore || !a.searchHasMore || a.searchTerm == "" {
		return
	}
	term := a.searchTerm
	page := a.searchPageNo + 1
	gen := a.searchGen
	a.searchLoadingMore = true
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		batch, err := searchMigu(ctx, term, page, 20)
		a.post(func() {
			if a.searchGen != gen || a.searchTerm != term {
				return
			}
			a.searchLoadingMore = false
			if err != nil {
				a.toast = friendlyError(err)
				return
			}
			a.searchTracks = miguDedupeTracks(append(a.searchTracks, batch...))
			a.searchPageNo = page
			a.searchHasMore = len(batch) == 20
		})
	}()
}

func (a *app) navTo(p pageKind) {
	a.page = p
	switch p {
	case pageHome:
		if len(a.home.NewSongs) == 0 && !a.homeLoading {
			a.loadMiguHome()
		}
	case pageRank:
		if len(a.rank.Tracks) == 0 && !a.rankLoading {
			id := a.rankID
			if id == "" {
				id = miguRankCatalog[0].ID
			}
			a.loadRank(id)
		}
	case pagePlaylists:
		if len(a.playlistSections) == 0 && !a.playlistSquareLoading {
			a.loadPlaylistSquare()
		}
	case pageSingers:
		if len(a.singers) == 0 && !a.singersLoading {
			a.loadSingers(a.singerTab)
		}
	case pageRadios:
		if len(a.radioGroups) == 0 && !a.radiosLoading {
			a.loadRadios()
		}
	case pageSong:
		a.lyricActive = -2
	}
	a.refresh()
}

func (a *app) playTracks(tracks []Track, at int) {
	if len(tracks) == 0 {
		return
	}
	a.pl.playQueue(tracks, at)
}

func (a *app) requestArt(rawURL string) {
	if rawURL == "" {
		return
	}
	if a.arts == nil {
		a.arts = map[string]ui.ImageSource{}
	}
	if a.artLoad == nil {
		a.artLoad = map[string]bool{}
	}
	if a.artFail == nil {
		a.artFail = map[string]bool{}
	}
	if a.arts[rawURL] != nil || a.artLoad[rawURL] || a.artFail[rawURL] {
		return
	}
	a.artLoad[rawURL] = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		data, err := fetchBytes(ctx, rawURL)
		if err != nil {
			a.post(func() {
				delete(a.artLoad, rawURL)
				a.artFail[rawURL] = true
			})
			return
		}
		a.post(func() {
			delete(a.artLoad, rawURL)
			src, err := ui.DecodeImage(data)
			if err != nil {
				a.artFail[rawURL] = true
				return
			}
			a.arts[rawURL] = src
			if tint, backdrop := inspectCover(data); tint != (ui.Color{}) {
				if a.artTints == nil {
					a.artTints = map[string]ui.Color{}
				}
				a.artTints[rawURL] = tint
				if backdrop != nil {
					if a.artBackdrops == nil {
						a.artBackdrops = map[string]*ui.Bitmap{}
					}
					a.artBackdrops[rawURL] = backdrop
				}
			}
			a.refresh()
		})
	}()
}

var coverHQSlots = make(chan struct{}, 2)

func (a *app) hqCover(track Track) string {
	if track.CoverHQ != "" {
		return track.CoverHQ
	}
	if a.coverHQ == nil {
		a.coverHQ = map[string]string{}
	}
	if src := a.coverHQ[track.ID]; src != "" {
		return src
	}
	a.requestCoverHQ(track)
	return track.Cover
}

func (a *app) requestCoverHQ(track Track) {
	if track.ID == "" || track.CoverHQ != "" {
		return
	}
	if a.coverHQ == nil {
		a.coverHQ = map[string]string{}
	}
	if a.coverHQLoad == nil {
		a.coverHQLoad = map[string]bool{}
	}
	if a.coverHQFail == nil {
		a.coverHQFail = map[string]bool{}
	}
	if a.coverHQ[track.ID] != "" || a.coverHQLoad[track.ID] || a.coverHQFail[track.ID] {
		return
	}
	a.coverHQLoad[track.ID] = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		select {
		case coverHQSlots <- struct{}{}:
			defer func() { <-coverHQSlots }()
		case <-ctx.Done():
			a.post(func() {
				delete(a.coverHQLoad, track.ID)
				a.coverHQFail[track.ID] = true
			})
			return
		}
		src := resolveMiguCoverHQ(ctx, track)
		a.post(func() {
			delete(a.coverHQLoad, track.ID)
			if src == "" {
				a.coverHQFail[track.ID] = true
				return
			}
			a.coverHQ[track.ID] = src
			a.refresh()
		})
	}()
}

func resolveMiguCoverHQ(ctx context.Context, track Track) string {
	if track.CoverHQ != "" {
		return track.CoverHQ
	}
	queries := make([]string, 0, 2)
	if q := strings.TrimSpace(track.Title + " " + track.Artist); q != "" {
		queries = append(queries, q)
	}
	if q := strings.TrimSpace(track.Title); q != "" && (len(queries) == 0 || q != queries[0]) {
		queries = append(queries, q)
	}
	for _, query := range queries {
		hits, err := searchMigu(ctx, query, 1, 20)
		if err != nil {
			continue
		}
		for _, hit := range hits {
			if hit.ID == track.ID && hit.CoverHQ != "" {
				return hit.CoverHQ
			}
		}
	}
	return ""
}

func (a *app) toggleFavorite(t Track) {
	if a.toggleFavoriteState(t) {
		a.toast = "已收藏 " + t.Title
	} else {
		a.toast = "已取消收藏"
	}
	a.refresh()
}

func rankTitle(id string) string {
	for _, rank := range miguRankCatalog {
		if rank.ID == id {
			return rank.Title
		}
	}
	return "排行榜"
}

func rankNote(id string) string {
	for _, rank := range miguRankCatalog {
		if rank.ID == id {
			return rank.Note
		}
	}
	return ""
}
