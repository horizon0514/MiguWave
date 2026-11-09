package migu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	miguWebBase = "https://d.musicapp.migu.cn"
	miguUA      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0 Safari/537.36"
)

var (
	miguSearchEndpoint = "https://app.u.nf.migu.cn/pc/resource/song/item/search/v1.0"
	miguRankEndpoint   = "https://app.u.nf.migu.cn/pc/bmw/rank/rank-info/v1.0"
	miguHomeEndpoint   = "https://app.u.nf.migu.cn/pc/bmw/page-data/music-index-show/v2.0"
	miguHotEndpoint    = "https://app.u.nf.migu.cn/pc/bmw/hot-search/hot-search/v1.0"
	miguListenEndpoint = "https://app.c.nf.migu.cn/MIGUM3.0/strategy/pc/listen/v1.0"
)

var miguClient = &http.Client{Timeout: 35 * time.Second}

var errMiguVIPRequired = errors.New("需要咪咕会员授权")

type miguRankInfo struct {
	ID     string
	Title  string
	Note   string
	Accent string
}

var miguRankCatalog = []miguRankInfo{
	{ID: "83176390", Title: "国风热歌榜", Note: "每周更新", Accent: "#ff2667"},
	{ID: "75959118", Title: "音乐风向榜", Note: "每小时更新", Accent: "#ff8a5c"},
	{ID: "27553319", Title: "尖叫新歌榜", Note: "实时新歌", Accent: "#ff4f9a"},
	{ID: "27186466", Title: "尖叫热歌榜", Note: "实时热歌", Accent: "#ffa52f"},
	{ID: "27553408", Title: "尖叫原创榜", Note: "原创力量", Accent: "#b129cc"},
}

type miguRankSummary struct {
	ID          string
	Title       string
	Subtitle    string
	Description string
	Cover       string
	Tracks      []Track
}

type miguRank struct {
	ID          string
	Title       string
	Description string
	UpdateTime  string
	Cover       string
	Tracks      []Track
}

type miguPlaylist struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Cover       string   `json:"cover,omitempty"`
	CoverHQ     string   `json:"coverHq,omitempty"`
	Owner       string   `json:"owner,omitempty"`
	PlayCount   string   `json:"playCount,omitempty"`
	TrackCount  int      `json:"trackCount,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type miguHomeData struct {
	NewSongs  []Track
	Ranks     []miguRankSummary
	Playlists []miguPlaylist
}

type miguSong struct {
	ResourceType flexString `json:"resourceType"`
	ContentID    string     `json:"contentId"`
	CopyrightID  string     `json:"copyrightId"`
	SongID       flexString `json:"songId"`
	SongName     string     `json:"songName"`
	Duration     flexString `json:"duration"`
	Album        string     `json:"album"`
	AlbumID      flexString `json:"albumId"`
	Img1         string     `json:"img1"`
	Img2         string     `json:"img2"`
	Img3         string     `json:"img3"`
	ShowTags     []string   `json:"showTags"`
	ShowTag      []string   `json:"showTag"`
	VIP          flexString `json:"vip"`
	PlayNumDesc  string     `json:"playNumDesc"`
	AudioFormats []struct {
		FormatType string `json:"formatType"`
	} `json:"audioFormats"`
	SingerList []struct {
		Name string `json:"name"`
	} `json:"singerList"`
}

type miguFeedItem struct {
	View        string     `json:"view"`
	Action      string     `json:"action"`
	Track       string     `json:"track"`
	ResID       string     `json:"resId"`
	ResType     flexString `json:"resType"`
	CopyrightID string     `json:"copyrightId"`
	Img         string     `json:"img"`
	Img2        string     `json:"img2"`
	Img3        string     `json:"img3"`
	Txt         string     `json:"txt"`
	Txt2        string     `json:"txt2"`
	Txt3        string     `json:"txt3"`
	Txt4        string     `json:"txt4"`
	Txt5        string     `json:"txt5"`
	ShowTag     []string   `json:"showTag"`
	VIP         flexString `json:"vip"`
	SongData    string     `json:"songData"`
}

type miguFeedSection struct {
	ColumnID string            `json:"columnId"`
	View     string            `json:"view"`
	Title    string            `json:"title"`
	Desc     string            `json:"desc"`
	SubTitle string            `json:"subTitle"`
	Contents []json.RawMessage `json:"contents"`
}

type miguEnvelope struct {
	Code string          `json:"code"`
	Info string          `json:"info"`
	Data json.RawMessage `json:"data"`
}

func miguHeaders(req *http.Request, token string) {
	req.Header.Set("User-Agent", miguUA)
	req.Header.Set("Referer", "https://music.migu.cn/v5/")
	req.Header.Set("Origin", "https://music.migu.cn")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("channel", "014X031")
	req.Header.Set("subchannel", "014X031")
	req.Header.Set("platform", "H5")
	req.Header.Set("version", "6.8.8")
	req.Header.Set("ua", "Android_migu")
	req.Header.Set("appId", "1101000001")
	req.Header.Set("test", "00")
	if token != "" {
		req.Header.Set("pacmtoken", token)
		req.AddCookie(&http.Cookie{Name: "pacmtoken", Value: token})
	}
}

func miguGet(ctx context.Context, endpoint, token string) ([]byte, error) {
	return miguGetWithHeaders(ctx, endpoint, token, nil)
}

func miguGetWithHeaders(ctx context.Context, endpoint, token string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	miguHeaders(req, token)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	res, err := miguClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("咪咕接口返回 %s", res.Status)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	return body, nil
}

func miguImage(raw string) string {
	raw = strings.TrimSpace(raw)
	switch {
	case raw == "":
		return ""
	case strings.HasPrefix(raw, "//"):
		return "https:" + raw
	case strings.HasPrefix(raw, "/"):
		return miguWebBase + raw
	default:
		return raw
	}
}

func miguSongThumb(song miguSong) string {
	for _, raw := range []string{song.Img2, song.Img1, song.Img3} {
		if src := miguImage(raw); src != "" {
			return src
		}
	}
	return ""
}

func miguSongCover(song miguSong) string {
	for _, raw := range []string{song.Img3, song.Img2, song.Img1} {
		if src := miguImage(raw); src != "" {
			return src
		}
	}
	return ""
}

func miguArtist(names []string) string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
		}
	}
	return strings.Join(out, " / ")
}

func miguHasVIPTag(tags ...[]string) bool {
	for _, group := range tags {
		for _, tag := range group {
			if strings.EqualFold(strings.TrimSpace(tag), "vip") ||
				strings.EqualFold(strings.TrimSpace(tag), "tl-vip") {
				return true
			}
		}
	}
	return false
}

func miguVIPFlag(values ...flexString) bool {
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(string(value))) {
		case "", "0", "false", "no", "否":
			continue
		default:
			return true
		}
	}
	return false
}

func miguSongTrack(song miguSong) Track {
	names := make([]string, 0, len(song.SingerList))
	for _, singer := range song.SingerList {
		names = append(names, singer.Name)
	}
	resourceType := strings.TrimSpace(string(song.ResourceType))
	if resourceType == "" {
		resourceType = "2"
	}
	duration, _ := strconv.ParseFloat(strings.TrimSpace(string(song.Duration)), 64)
	vip := miguVIPFlag(song.VIP) || miguHasVIPTag(song.ShowTags, song.ShowTag)
	return Track{
		ID:           "migu:" + song.ContentID,
		Title:        cleanDisplay(song.SongName),
		Artist:       cleanDisplay(miguArtist(names)),
		Album:        cleanDisplay(song.Album),
		Cover:        miguSongThumb(song),
		CoverHQ:      miguSongCover(song),
		DurationSec:  duration,
		Source:       "migu",
		CopyrightID:  song.CopyrightID,
		ResourceType: resourceType,
		VIP:          vip,
		PlayCount:    strings.TrimSpace(song.PlayNumDesc),
	}
}

func miguFeedTrack(item miguFeedItem) Track {
	if strings.TrimSpace(item.SongData) != "" {
		var song miguSong
		if json.Unmarshal([]byte(item.SongData), &song) == nil && song.ContentID != "" {
			track := miguSongTrack(song)
			if track.Cover == "" {
				track.Cover = miguImage(item.Img)
			}
			if track.CopyrightID == "" {
				track.CopyrightID = item.CopyrightID
			}
			return track
		}
	}
	resourceType := strings.TrimSpace(string(item.ResType))
	if resourceType == "" {
		resourceType = "2"
	}
	artist := strings.TrimSpace(item.Txt2)
	artist = strings.ReplaceAll(artist, "|", " / ")
	title := strings.TrimSpace(item.Txt)
	if title == "" {
		title = "未知歌曲"
	}
	return Track{
		ID:           "migu:" + item.ResID,
		Title:        cleanDisplay(title),
		Artist:       cleanDisplay(artist),
		Album:        cleanDisplay(item.Txt3),
		Cover:        miguImage(item.Img),
		Source:       "migu",
		CopyrightID:  item.CopyrightID,
		ResourceType: resourceType,
		VIP:          miguVIPFlag(item.VIP) || miguHasVIPTag(item.ShowTag),
	}
}

func miguFeedTracks(contents []json.RawMessage) []Track {
	tracks := make([]Track, 0, len(contents))
	seen := map[string]bool{}
	for _, raw := range contents {
		var item miguFeedItem
		if json.Unmarshal(raw, &item) != nil {
			continue
		}
		if item.ResID == "" {
			continue
		}
		track := miguFeedTrack(item)
		if track.ID == "migu:" || seen[track.ID] {
			continue
		}
		seen[track.ID] = true
		tracks = append(tracks, track)
	}
	return tracks
}

func miguDedupeTracks(tracks []Track) []Track {
	out := make([]Track, 0, len(tracks))
	seen := map[string]bool{}
	for _, track := range tracks {
		if track.ID == "" || seen[track.ID] {
			continue
		}
		seen[track.ID] = true
		out = append(out, track)
	}
	return out
}

func searchMigu(ctx context.Context, query string, page, pageSize int) ([]Track, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("请输入歌名、歌手或专辑")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	values := url.Values{}
	values.Set("text", query)
	values.Set("pageNo", strconv.Itoa(page))
	values.Set("pageSize", strconv.Itoa(pageSize))
	body, err := miguGet(ctx, miguSearchEndpoint+"?"+values.Encode(), "")
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, errors.New("咪咕搜索返回了空响应")
	}
	if trimmed[0] == '{' {
		var env miguEnvelope
		if json.Unmarshal(trimmed, &env) == nil && env.Code != "" && env.Code != "000000" {
			return nil, fmt.Errorf("咪咕搜索：%s", env.Info)
		}
		return nil, errors.New("咪咕搜索返回了无法识别的数据")
	}
	var songs []miguSong
	if err := json.Unmarshal(trimmed, &songs); err != nil {
		return nil, err
	}
	tracks := make([]Track, 0, len(songs))
	for _, song := range songs {
		if song.ContentID == "" || strings.TrimSpace(song.SongName) == "" {
			continue
		}
		tracks = append(tracks, miguSongTrack(song))
	}
	return miguDedupeTracks(tracks), nil
}

func fetchMiguRank(ctx context.Context, id string, limit int) (miguRank, error) {
	if strings.TrimSpace(id) == "" {
		return miguRank{}, errors.New("缺少榜单 ID")
	}
	if limit <= 0 {
		limit = 100
	}
	values := url.Values{}
	values.Set("rankId", id)
	values.Set("pageNo", "1")
	values.Set("pageSize", strconv.Itoa(limit))
	body, err := miguGet(ctx, miguRankEndpoint+"?"+values.Encode(), "")
	if err != nil {
		return miguRank{}, err
	}
	var env miguEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return miguRank{}, err
	}
	if env.Code != "" && env.Code != "000000" {
		return miguRank{}, fmt.Errorf("咪咕榜单：%s", env.Info)
	}
	var data struct {
		ColumnID   string         `json:"columnId"`
		Title      string         `json:"title"`
		Desc       string         `json:"desc"`
		UpdateTime string         `json:"updateTime"`
		TitlePic   string         `json:"titlePic"`
		Contents   []miguFeedItem `json:"contents"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return miguRank{}, err
	}
	rank := miguRank{
		ID:          firstNonEmpty(id, data.ColumnID),
		Title:       cleanDisplay(data.Title),
		Description: cleanDescription(data.Desc),
		UpdateTime:  strings.TrimSpace(data.UpdateTime),
		Cover:       miguImage(data.TitlePic),
	}
	for _, item := range data.Contents {
		track := miguFeedTrack(item)
		if track.ID == "migu:" {
			continue
		}
		rank.Tracks = append(rank.Tracks, track)
	}
	rank.Tracks = miguDedupeTracks(rank.Tracks)
	if rank.Cover == "" && len(rank.Tracks) > 0 {
		rank.Cover = rank.Tracks[0].Cover
	}
	return rank, nil
}

func fetchMiguHome(ctx context.Context) (miguHomeData, error) {
	values := url.Values{}
	values.Set("templateVersion", "1")
	body, err := miguGet(ctx, miguHomeEndpoint+"?"+values.Encode(), "")
	if err != nil {
		return miguHomeData{}, err
	}
	var env miguEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return miguHomeData{}, err
	}
	if env.Code != "" && env.Code != "000000" {
		return miguHomeData{}, fmt.Errorf("咪咕首页：%s", env.Info)
	}
	var data struct {
		Contents []miguFeedSection `json:"contents"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return miguHomeData{}, err
	}
	var home miguHomeData
	for _, section := range data.Contents {
		switch section.View {
		case "ZJ-Song-Scroll":
			tracks := miguFeedTracks(section.Contents)
			if section.Title == "新歌速递" || len(home.NewSongs) == 0 {
				home.NewSongs = append(home.NewSongs, tracks...)
			}
		case "ZJ-Rank-Compound":
			home.Ranks = miguRanksFromSection(section)
		case "ZJ-SongList-Scroll":
			home.Playlists = miguPlaylistsFromSection(section)
		}
	}
	home.NewSongs = miguDedupeTracks(home.NewSongs)
	return home, nil
}

func miguRanksFromSection(section miguFeedSection) []miguRankSummary {
	ranks := make([]miguRankSummary, 0, 5)
	for _, raw := range section.Contents {
		var child miguFeedSection
		if json.Unmarshal(raw, &child) != nil || child.View != "ZJ-Song-Scroll" {
			continue
		}
		tracks := miguFeedTracks(child.Contents)
		rank := miguRankSummary{
			ID:          child.ColumnID,
			Title:       cleanDisplay(child.Title),
			Subtitle:    cleanDisplay(child.SubTitle),
			Description: cleanDescription(child.Desc),
			Tracks:      tracks,
		}
		if len(tracks) > 0 {
			rank.Cover = tracks[0].Cover
		}
		if rank.ID != "" && rank.Title != "" {
			ranks = append(ranks, rank)
		}
	}
	return ranks
}

func miguPlaylistsFromSection(section miguFeedSection) []miguPlaylist {
	out := make([]miguPlaylist, 0, len(section.Contents))
	for _, raw := range section.Contents {
		var item miguFeedItem
		if json.Unmarshal(raw, &item) != nil || item.ResID == "" || item.Txt == "" {
			continue
		}
		out = append(out, miguPlaylist{
			ID:          item.ResID,
			Title:       cleanDisplay(item.Txt),
			Description: cleanDescription(item.Txt2),
			Cover:       miguImage(firstNonEmpty(item.Img, item.Img2, item.Img3)),
			CoverHQ:     miguImage(firstNonEmpty(item.Img3, item.Img2, item.Img)),
		})
	}
	return out
}

func fetchMiguHotSearches(ctx context.Context) ([]string, error) {
	body, err := miguGet(ctx, miguHotEndpoint, "")
	if err != nil {
		return nil, err
	}
	var env miguEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	if env.Code != "" && env.Code != "000000" {
		return nil, fmt.Errorf("咪咕热搜：%s", env.Info)
	}
	var data struct {
		HotWordItemList []struct {
			Word string `json:"word"`
		} `json:"hotWordItemList"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(data.HotWordItemList))
	seen := map[string]bool{}
	for _, item := range data.HotWordItemList {
		word := strings.TrimSpace(item.Word)
		if word == "" || seen[word] {
			continue
		}
		seen[word] = true
		out = append(out, word)
	}
	return out, nil
}

func resolveMiguURL(ctx context.Context, token string, track Track) (string, error) {
	if track.Source != "migu" {
		return "", errors.New("不是咪咕曲目")
	}
	contentID := strings.TrimPrefix(track.ID, "migu:")
	if contentID == "" {
		return "", errors.New("缺少咪咕 contentId")
	}
	if strings.TrimSpace(track.CopyrightID) == "" {
		return "", errors.New("缺少咪咕 copyrightId")
	}
	resourceType := track.ResourceType
	if resourceType == "" {
		resourceType = "2"
	}
	values := url.Values{}
	values.Set("contentId", contentID)
	values.Set("copyrightId", track.CopyrightID)
	values.Set("resourceType", resourceType)
	values.Set("netType", "01")
	values.Set("toneFlag", "PQ")
	values.Set("scene", "")
	body, err := miguGet(ctx, miguListenEndpoint+"?"+values.Encode(), token)
	if err != nil {
		return "", err
	}
	var out struct {
		Code string `json:"code"`
		Info string `json:"info"`
		Data struct {
			URL             string `json:"url"`
			CannotCode      string `json:"cannotCode"`
			AuditionsLength int    `json:"auditionsLength"`
			FreeListenType  string `json:"freeListenType"`
			DialogInfo      struct {
				Text string `json:"text"`
			} `json:"dialogInfo"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	if out.Code != "" && out.Code != "000000" {
		if out.Code == "290001" {
			return "", fmt.Errorf("%w：请重新登录咪咕账号", errMiguTokenStale)
		}
		return "", fmt.Errorf("咪咕播放：%s", out.Info)
	}
	if strings.TrimSpace(out.Data.URL) != "" {
		return strings.TrimSpace(out.Data.URL), nil
	}
	reason := cleanDisplay(out.Data.DialogInfo.Text)
	if reason == "" {
		reason = "该曲目没有返回可播放地址"
	}
	if out.Data.CannotCode == "440013" || out.Data.AuditionsLength > 0 ||
		strings.Contains(reason, "会员") || strings.Contains(reason, "白金") {
		return "", fmt.Errorf("%w：%s。请在左下角扫码登录咪咕账号", errMiguVIPRequired, reason)
	}
	return "", errors.New(reason)
}

func normalizeMiguToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		lower := strings.ToLower(part)
		if strings.HasPrefix(lower, "pacmtoken=") {
			value := strings.TrimSpace(part[len("pacmtoken="):])
			if decoded, err := url.QueryUnescape(value); err == nil {
				value = decoded
			}
			return strings.Trim(value, `"' `)
		}
	}
	return strings.Trim(raw, `"' `)
}

// miguTokenDir resolves the directory holding the local login token.
// It is a variable so tests can point it at a temporary directory.
var miguTokenDir = func() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return "", errors.New("找不到用户配置目录")
	}
	return filepath.Join(dir, "MiguWave"), nil
}

func miguTokenPath() (string, error) {
	dir, err := miguTokenDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "auth.json"), nil
}

func loadMiguToken() string {
	for _, key := range []string{"MIGU_PACTOKEN", "FREEWAVE_MIGU_TOKEN"} {
		if token := normalizeMiguToken(os.Getenv(key)); token != "" {
			return token
		}
	}
	path, err := miguTokenPath()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return normalizeMiguToken(string(b))
}

func saveMiguToken(token string) error {
	token = normalizeMiguToken(token)
	path, err := miguTokenPath()
	if err != nil {
		return err
	}
	if token == "" {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(token+"\n"), 0o600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
