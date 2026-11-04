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
)

type app struct {
	win *mygo.Window
	pl  *player

	volume    float64
	favorites []Track
	recent    []Track

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

	searchInput       string
	searchTerm        string
	searchTracks      []Track
	searchLoading     bool
	searchLoadingMore bool
	searchHasMore     bool
	searchPageNo      int
	searchErr         string
	searchGen         int

	scrub       float64
	scrubHeld   bool
	scrubActive bool

	arts    map[string]ui.ImageSource
	artLoad map[string]bool
	artFail map[string]bool
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
}

func (a *app) start() {
	a.loadMiguHome()
	a.loadHotSearches()
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

func (a *app) runSearch(q string) {
	q = strings.TrimSpace(q)
	if q == "" {
		a.toast = "输入歌名、歌手或专辑后再搜索"
		a.refresh()
		return
	}
	a.searchInput = q
	a.searchTerm = q
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
			a.refresh()
		})
	}()
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
