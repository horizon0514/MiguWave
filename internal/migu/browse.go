package migu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

var (
	miguPlaylistInfoEndpoint   = "https://app.c.nf.migu.cn/resource/playlist/v2.0"
	miguPlaylistSongEndpoint   = "https://app.c.nf.migu.cn/MIGUM3.0/resource/playlist/song/v2.0"
	miguPlaylistSquareEndpoint = "https://app.c.nf.migu.cn/pc/bmw/page-data/playlist-square-recommend/v1.0"
	miguSingerListEndpoint     = "https://app.u.nf.migu.cn/MIGUM3.0/bmw/singer-index/list/v1.0"
	miguSingerIndexEndpoint    = "https://app.c.nf.migu.cn/bmw/singer/index/v1.0"
	miguSingerInfoEndpoint     = "https://app.c.nf.migu.cn/bmw/singer/info/v1.1"
	miguSingerSongEndpoint     = "https://app.c.nf.migu.cn/bmw/singer/song/v1.0"
	miguRadioPageEndpoint      = "https://app.c.nf.migu.cn/pc/bmw/page-data/music-radio/v1.0"
	miguRadioSongEndpoint      = "https://app.c.nf.migu.cn/pc/column/music-radio/scene/song/v3.0"
)

const miguPlaylistPageSize = 50

type miguPlaylistSection struct {
	Title string
	Items []miguPlaylist
}

type miguPlaylistDetail struct {
	Playlist miguPlaylist
	Tracks   []Track
	Page     int
	Total    int
	HasMore  bool
}

type miguSinger struct {
	ID      string
	Name    string
	Cover   string
	CoverHQ string
	Hot     string
	Fans    string
}

type miguSingerDetail struct {
	Singer  miguSinger
	Bio     string
	Tracks  []Track
	Similar []miguSinger
	Page    int
	HasMore bool
}

type miguSingerTab struct {
	ID    string
	Label string
}

var miguSingerTabs = []miguSingerTab{
	{ID: "huayu-nan", Label: "华语男"},
	{ID: "huayu-nv", Label: "华语女"},
	{ID: "huayu-group", Label: "华语组合"},
	{ID: "oumei-nan", Label: "欧美男"},
	{ID: "oumei-nv", Label: "欧美女"},
	{ID: "oumei-group", Label: "欧美组合"},
	{ID: "rihan-nan", Label: "日韩男"},
	{ID: "rihan-nv", Label: "日韩女"},
	{ID: "rihan-group", Label: "日韩组合"},
}

type miguRadio struct {
	ID       string
	Title    string
	Subtitle string
	Cover    string
	Kind     string
}

type miguRadioGroup struct {
	Title string
	Items []miguRadio
}

type miguRadioDetail struct {
	Radio  miguRadio
	Tracks []Track
	Total  int
}

func fetchMiguPlaylistSquare(ctx context.Context) ([]miguPlaylistSection, error) {
	values := url.Values{}
	values.Set("templateVersion", "2")
	body, err := miguGet(ctx, miguPlaylistSquareEndpoint+"?"+values.Encode(), "")
	if err != nil {
		return nil, err
	}
	var env miguEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	if env.Code != "" && env.Code != "000000" {
		return nil, fmt.Errorf("咪咕歌单广场：%s", env.Info)
	}
	var data struct {
		Contents []miguFeedSection `json:"contents"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, err
	}
	sections := make([]miguPlaylistSection, 0, len(data.Contents))
	for _, section := range data.Contents {
		switch section.View {
		case "ZJ-SongList-Scroll":
			items := miguPlaylistsFromSection(section)
			if len(items) > 0 {
				sections = append(sections, miguPlaylistSection{
					Title: miguPlaylistSectionTitle(section.Title),
					Items: items,
				})
			}
		case "playlist_square_scene_zone":
			for _, raw := range section.Contents {
				var child miguFeedSection
				if json.Unmarshal(raw, &child) != nil || child.View != "ZJ-SongList-Scroll" {
					continue
				}
				items := miguPlaylistsFromSection(child)
				if len(items) == 0 {
					continue
				}
				title := cleanDisplay(child.Title)
				if title == "" || title == "标题" {
					title = cleanDisplay(section.Title)
				}
				sections = append(sections, miguPlaylistSection{
					Title: miguPlaylistSectionTitle(title),
					Items: items,
				})
			}
		}
	}
	return sections, nil
}

func miguPlaylistSectionTitle(title string) string {
	title = cleanDisplay(title)
	title = strings.ReplaceAll(title, "9月深度-", "深度")
	for _, suffix := range []string{"-歌单内容", "-歌单", "-内容", "-标题"} {
		title = strings.TrimSuffix(title, suffix)
	}
	title = strings.Trim(title, " -『』「」")
	if title == "" {
		return "推荐歌单"
	}
	return title
}

func fetchMiguPlaylistDetail(ctx context.Context, id string, page int) (miguPlaylistDetail, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return miguPlaylistDetail{}, errors.New("缺少歌单 ID")
	}
	if page < 1 {
		page = 1
	}
	playlist, err := fetchMiguPlaylistInfo(ctx, id)
	if err != nil {
		return miguPlaylistDetail{}, err
	}
	tracks, total, hasMore, err := fetchMiguPlaylistSongs(ctx, id, page)
	if err != nil {
		return miguPlaylistDetail{}, err
	}
	if total > 0 {
		playlist.TrackCount = total
	}
	return miguPlaylistDetail{
		Playlist: playlist,
		Tracks:   tracks,
		Page:     page,
		Total:    total,
		HasMore:  hasMore,
	}, nil
}

func fetchMiguPlaylistInfo(ctx context.Context, id string) (miguPlaylist, error) {
	values := url.Values{}
	values.Set("playlistId", id)
	body, err := miguGet(ctx, miguPlaylistInfoEndpoint+"?"+values.Encode(), "")
	if err != nil {
		return miguPlaylist{}, err
	}
	var env miguEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return miguPlaylist{}, err
	}
	if env.Code != "" && env.Code != "000000" {
		return miguPlaylist{}, fmt.Errorf("咪咕歌单：%s", env.Info)
	}
	var data struct {
		Title          string `json:"title"`
		Summary        string `json:"summary"`
		MusicNum       int    `json:"musicNum"`
		OriginalImgURL string `json:"originalImgUrl"`
		OwnerName      string `json:"ownerName"`
		ImgItem        struct {
			Img     string `json:"img"`
			ImgOri  string `json:"imgOri"`
			WebpImg string `json:"webpImg"`
		} `json:"imgItem"`
		Tags []struct {
			TagName string `json:"tagName"`
		} `json:"tags"`
		OpNumItem struct {
			PlayNumDesc string `json:"playNumDesc"`
			KeepNumDesc string `json:"keepNumDesc"`
		} `json:"opNumItem"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return miguPlaylist{}, err
	}
	tags := make([]string, 0, len(data.Tags))
	for _, tag := range data.Tags {
		if tag.TagName = cleanDisplay(tag.TagName); tag.TagName != "" {
			tags = append(tags, tag.TagName)
		}
	}
	return miguPlaylist{
		ID:          id,
		Title:       cleanDisplay(data.Title),
		Description: cleanDescription(data.Summary),
		Cover:       miguImage(firstNonEmpty(data.ImgItem.Img, data.ImgItem.WebpImg, data.OriginalImgURL)),
		CoverHQ:     miguImage(firstNonEmpty(data.OriginalImgURL, data.ImgItem.ImgOri, data.ImgItem.Img)),
		Owner:       cleanDisplay(data.OwnerName),
		PlayCount:   cleanDisplay(data.OpNumItem.PlayNumDesc),
		TrackCount:  data.MusicNum,
		Tags:        tags,
	}, nil
}

func fetchMiguPlaylistSongs(ctx context.Context, id string, page int) ([]Track, int, bool, error) {
	if page < 1 {
		page = 1
	}
	values := url.Values{}
	values.Set("playlistId", id)
	values.Set("pageNo", strconv.Itoa(page))
	values.Set("pageSize", strconv.Itoa(miguPlaylistPageSize))
	body, err := miguGet(ctx, miguPlaylistSongEndpoint+"?"+values.Encode(), "")
	if err != nil {
		return nil, 0, false, err
	}
	var env miguEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, 0, false, err
	}
	if env.Code != "" && env.Code != "000000" {
		return nil, 0, false, fmt.Errorf("咪咕歌单歌曲：%s", env.Info)
	}
	var data struct {
		SongList   []miguSong `json:"songList"`
		TotalCount int        `json:"totalCount"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, 0, false, err
	}
	tracks := make([]Track, 0, len(data.SongList))
	for _, song := range data.SongList {
		if song.ContentID == "" || strings.TrimSpace(song.SongName) == "" {
			continue
		}
		tracks = append(tracks, miguSongTrack(song))
	}
	tracks = miguDedupeTracks(tracks)
	hasMore := len(data.SongList) >= miguPlaylistPageSize
	if data.TotalCount > 0 {
		hasMore = page*miguPlaylistPageSize < data.TotalCount
	}
	return tracks, data.TotalCount, hasMore, nil
}

func fetchMiguSingers(ctx context.Context, tab string) ([]miguSinger, error) {
	tab = strings.TrimSpace(tab)
	if tab == "" {
		tab = miguSingerTabs[0].ID
	}
	values := url.Values{}
	values.Set("tab", tab)
	values.Set("templateVersion", "3")
	body, err := miguGet(ctx, miguSingerListEndpoint+"?"+values.Encode(), "")
	if err != nil {
		return nil, err
	}
	var env miguEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	if env.Code != "" && env.Code != "000000" {
		return nil, fmt.Errorf("咪咕歌手：%s", env.Info)
	}
	var data struct {
		Contents []miguFeedItem `json:"contents"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, err
	}
	singers := make([]miguSinger, 0, len(data.Contents))
	seen := map[string]bool{}
	for _, item := range data.Contents {
		singer := miguSingerFromItem(item)
		if singer.ID == "" || singer.Name == "" || seen[singer.ID] {
			continue
		}
		seen[singer.ID] = true
		singers = append(singers, singer)
	}
	return singers, nil
}

func miguSingerFromItem(item miguFeedItem) miguSinger {
	id := strings.TrimSpace(item.ResID)
	if id == "" {
		id = strings.TrimSpace(item.Txt2)
	}
	if id == "" && strings.Contains(item.Action, "id=") {
		id = item.Action[strings.LastIndex(item.Action, "id=")+3:]
		if cut := strings.IndexAny(id, "&?"); cut >= 0 {
			id = id[:cut]
		}
	}
	return miguSinger{
		ID:      id,
		Name:    cleanDisplay(item.Txt),
		Cover:   miguImage(firstNonEmpty(item.Img, item.Img2, item.Img3)),
		CoverHQ: miguImage(firstNonEmpty(item.Img3, item.Img2, item.Img)),
		Hot:     cleanDisplay(item.Txt4),
		Fans:    cleanDisplay(item.Txt5),
	}
}

func fetchMiguSingerDetail(ctx context.Context, singer miguSinger) (miguSingerDetail, error) {
	if strings.TrimSpace(singer.ID) == "" {
		return miguSingerDetail{}, errors.New("缺少歌手 ID")
	}
	if info, err := fetchMiguSingerInfo(ctx, singer.ID); err == nil {
		if singer.Name == "" {
			singer.Name = info.Name
		}
		singer.Cover = firstNonEmpty(singer.Cover, info.Cover)
		singer.CoverHQ = firstNonEmpty(info.CoverHQ, singer.Cover)
		singer.Hot = firstNonEmpty(singer.Hot, info.Hot)
		singer.Fans = firstNonEmpty(singer.Fans, info.Fans)
	}
	bio, similar, err := fetchMiguSingerOverview(ctx, singer.ID)
	if err != nil {
		return miguSingerDetail{}, err
	}
	tracks, hasMore, err := fetchMiguSingerSongs(ctx, singer.ID, 1)
	if err != nil {
		return miguSingerDetail{}, err
	}
	return miguSingerDetail{
		Singer:  singer,
		Bio:     bio,
		Tracks:  tracks,
		Similar: similar,
		Page:    1,
		HasMore: hasMore,
	}, nil
}

func fetchMiguSingerInfo(ctx context.Context, id string) (miguSinger, error) {
	values := url.Values{}
	values.Set("singerId", id)
	body, err := miguGet(ctx, miguSingerInfoEndpoint+"?"+values.Encode(), "")
	if err != nil {
		return miguSinger{}, err
	}
	var env miguEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return miguSinger{}, err
	}
	if env.Code != "" && env.Code != "000000" {
		return miguSinger{}, fmt.Errorf("咪咕歌手资料：%s", env.Info)
	}
	var data struct {
		Contents []struct {
			Contents []miguFeedItem `json:"contents"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return miguSinger{}, err
	}
	for _, section := range data.Contents {
		for _, item := range section.Contents {
			if item.View != "ZJ-SingerDetail-Item" {
				continue
			}
			return miguSinger{
				ID:      id,
				Name:    cleanDisplay(item.Txt),
				Cover:   miguImage(firstNonEmpty(item.Img, item.Img2, item.Img3)),
				CoverHQ: miguImage(firstNonEmpty(item.Img3, item.Img2, item.Img)),
				Hot:     cleanDisplay(item.Txt4),
				Fans:    cleanDisplay(item.Txt5),
			}, nil
		}
	}
	return miguSinger{}, errors.New("咪咕没有返回歌手资料")
}

func fetchMiguSingerOverview(ctx context.Context, id string) (string, []miguSinger, error) {
	values := url.Values{}
	values.Set("singerId", id)
	body, err := miguGet(ctx, miguSingerIndexEndpoint+"?"+values.Encode(), "")
	if err != nil {
		return "", nil, err
	}
	var env miguEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return "", nil, err
	}
	if env.Code != "" && env.Code != "000000" {
		return "", nil, fmt.Errorf("咪咕歌手主页：%s", env.Info)
	}
	var data struct {
		Contents []miguFeedSection `json:"contents"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return "", nil, err
	}
	var bio string
	var similar []miguSinger
	for _, section := range data.Contents {
		switch section.View {
		case "ZJ-Singer-Intro-Scroll":
			for _, raw := range section.Contents {
				var item miguFeedItem
				if json.Unmarshal(raw, &item) == nil {
					bio = cleanDescription(item.Txt2)
					if bio != "" {
						break
					}
				}
			}
		case "ZJ-Singer-Scroll":
			for _, raw := range section.Contents {
				var item miguFeedItem
				if json.Unmarshal(raw, &item) != nil {
					continue
				}
				singer := miguSingerFromItem(item)
				if singer.ID != "" && singer.Name != "" {
					similar = append(similar, singer)
				}
			}
		}
	}
	return bio, similar, nil
}

func fetchMiguSingerSongs(ctx context.Context, id string, page int) ([]Track, bool, error) {
	if page < 1 {
		page = 1
	}
	values := url.Values{}
	values.Set("singerId", id)
	values.Set("pageNo", strconv.Itoa(page))
	values.Set("type", "1")
	body, err := miguGet(ctx, miguSingerSongEndpoint+"?"+values.Encode(), "")
	if err != nil {
		return nil, false, err
	}
	var env miguEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, false, err
	}
	if env.Code != "" && env.Code != "000000" {
		return nil, false, fmt.Errorf("咪咕歌手歌曲：%s", env.Info)
	}
	var data struct {
		Header struct {
			NextPageURL string `json:"nextPageUrl"`
		} `json:"header"`
		Contents []struct {
			Contents []struct {
				View     string          `json:"view"`
				SongItem json.RawMessage `json:"songItem"`
			} `json:"contents"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, false, err
	}
	tracks := make([]Track, 0, 50)
	for _, section := range data.Contents {
		for _, item := range section.Contents {
			if item.View != "ZJ-Singer-Song-Item" || len(item.SongItem) == 0 {
				continue
			}
			var song miguSong
			if json.Unmarshal(item.SongItem, &song) != nil || song.ContentID == "" {
				continue
			}
			tracks = append(tracks, miguSongTrack(song))
		}
	}
	tracks = miguDedupeTracks(tracks)
	hasMore := strings.TrimSpace(data.Header.NextPageURL) != "" || len(tracks) >= 50
	return tracks, hasMore, nil
}

func fetchMiguRadioGroups(ctx context.Context) ([]miguRadioGroup, error) {
	values := url.Values{}
	values.Set("templateVersion", "1")
	body, err := miguGet(ctx, miguRadioPageEndpoint+"?"+values.Encode(), "")
	if err != nil {
		return nil, err
	}
	var env miguEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	if env.Code != "" && env.Code != "000000" {
		return nil, fmt.Errorf("咪咕电台：%s", env.Info)
	}
	var data struct {
		Contents []miguFeedSection `json:"contents"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, err
	}
	groups := make([]miguRadioGroup, 0, 8)
	for _, section := range data.Contents {
		switch section.View {
		case "ZJ-MusicRadio-Private-Scroll":
			items := miguRadiosFromItems(section.Contents)
			for i := range items {
				items[i].Kind = "private"
			}
			if len(items) > 0 {
				groups = append(groups, miguRadioGroup{
					Title: cleanDisplay(firstNonEmpty(section.Title, "私享电台")),
					Items: items,
				})
			}
		case "ZJ-MusicRadio-Timing-Scroll":
			items := miguRadiosFromItems(section.Contents)
			for i := range items {
				items[i].Kind = "scene"
			}
			if len(items) > 0 {
				groups = append(groups, miguRadioGroup{
					Title: cleanDisplay(firstNonEmpty(section.Title, "时序电台")),
					Items: items,
				})
			}
		case "ZJ-MusicRadio-Category-Scroll":
			categoryGroups := miguRadioCategoryGroups(section)
			for i := range categoryGroups {
				for j := range categoryGroups[i].Items {
					categoryGroups[i].Items[j].Kind = "scene"
				}
			}
			groups = append(groups, categoryGroups...)
		}
	}
	return groups, nil
}

func miguRadioCategoryGroups(section miguFeedSection) []miguRadioGroup {
	groups := make([]miguRadioGroup, 0, 8)
	heading := ""
	for _, raw := range section.Contents {
		var child miguFeedSection
		if json.Unmarshal(raw, &child) != nil {
			continue
		}
		switch child.View {
		case "ZJ-Title":
			for _, titleRaw := range child.Contents {
				var item miguFeedItem
				if json.Unmarshal(titleRaw, &item) == nil {
					heading = miguRadioGroupTitle(firstNonEmpty(item.Txt, child.Title))
					break
				}
			}
		case "ZJ-MusicRadio-Category-Scroll":
			items := miguRadiosFromItems(child.Contents)
			if len(items) == 0 {
				continue
			}
			title := miguRadioGroupTitle(firstNonEmpty(heading, child.Title))
			groups = append(groups, miguRadioGroup{Title: title, Items: items})
		}
	}
	return groups
}

func miguRadioGroupTitle(title string) string {
	title = cleanDisplay(title)
	for _, suffix := range []string{"-标题", "--标题", "-内容"} {
		title = strings.TrimSuffix(title, suffix)
	}
	title = strings.Trim(title, " -")
	if title == "" {
		return "分类电台"
	}
	return title
}

func miguRadiosFromItems(contents []json.RawMessage) []miguRadio {
	items := make([]miguRadio, 0, len(contents))
	seen := map[string]bool{}
	for _, raw := range contents {
		var item miguFeedItem
		if json.Unmarshal(raw, &item) != nil {
			continue
		}
		radio := miguRadio{
			ID:       strings.TrimSpace(item.ResID),
			Title:    cleanDisplay(item.Txt),
			Subtitle: cleanDisplay(item.Txt2),
			Cover:    miguImage(item.Img),
		}
		if radio.ID == "" || radio.Title == "" || seen[radio.ID] {
			continue
		}
		seen[radio.ID] = true
		items = append(items, radio)
	}
	return items
}

func fetchMiguRadioDetail(ctx context.Context, radio miguRadio, maxTracks int) (miguRadioDetail, error) {
	if strings.TrimSpace(radio.ID) == "" {
		return miguRadioDetail{}, errors.New("缺少电台 ID")
	}
	if maxTracks <= 0 {
		maxTracks = 20
	}
	detail := miguRadioDetail{Radio: radio}
	seen := map[string]bool{}
	for page := 1; page <= 8 && len(detail.Tracks) < maxTracks; page++ {
		tracks, next, err := fetchMiguRadioSongs(ctx, radio.ID, page)
		if err != nil {
			if page == 1 {
				return miguRadioDetail{}, err
			}
			break
		}
		if len(tracks) == 0 {
			break
		}
		for _, track := range tracks {
			if track.ID == "" || seen[track.ID] {
				continue
			}
			seen[track.ID] = true
			detail.Tracks = append(detail.Tracks, track)
			if len(detail.Tracks) >= maxTracks {
				break
			}
		}
		if next <= page {
			break
		}
	}
	detail.Total = len(detail.Tracks)
	return detail, nil
}

func fetchMiguRadioSongs(ctx context.Context, id string, page int) ([]Track, int, error) {
	if page < 1 {
		page = 1
	}
	values := url.Values{}
	values.Set("remain", "0")
	values.Set("pageNo", strconv.Itoa(page))
	values.Set("actionId", "1")
	values.Set("radioId", id)
	body, err := miguGet(ctx, miguRadioSongEndpoint+"?"+values.Encode(), "")
	if err != nil {
		return nil, 0, err
	}
	var env miguEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, 0, err
	}
	if env.Code != "" && env.Code != "000000" {
		return nil, 0, fmt.Errorf("咪咕电台歌曲：%s", env.Info)
	}
	var data struct {
		NextPageNo flexString `json:"nextPageNo"`
		SongItems  []miguSong `json:"songItems"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, 0, err
	}
	tracks := make([]Track, 0, len(data.SongItems))
	for _, song := range data.SongItems {
		if song.ContentID == "" || strings.TrimSpace(song.SongName) == "" {
			continue
		}
		tracks = append(tracks, miguSongTrack(song))
	}
	next, _ := strconv.Atoi(strings.TrimSpace(string(data.NextPageNo)))
	return miguDedupeTracks(tracks), next, nil
}

func formatMiguCount(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "万") || strings.Contains(raw, "亿") {
		return raw
	}
	n, err := strconv.ParseInt(strings.ReplaceAll(raw, ",", ""), 10, 64)
	if err != nil {
		return raw
	}
	switch {
	case n >= 100000000:
		return fmt.Sprintf("%.1f亿", float64(n)/100000000)
	case n >= 10000:
		return fmt.Sprintf("%.1f万", float64(n)/10000)
	default:
		return strconv.FormatInt(n, 10)
	}
}
