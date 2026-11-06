package migu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"freewave/internal/lyrics"
)

type miguSongDetail struct {
	Track       Track
	Album       string
	Cover       string
	LRCURL      string
	MRCURL      string
	TRCURL      string
	Lyrics      []lyrics.Line
	PlainLyrics bool
	LyricErr    string
	LyricTitle  string
	LyricArtist string
	LyricAlbum  string
	LyricBy     string
	Tags        []string
	Formats     []string
	SongID      string
}

var errNotMiguTrack = errors.New("暂时只支持咪咕曲目")

func fetchMiguSongDetail(ctx context.Context, token string, track Track) (miguSongDetail, error) {
	if track.Source != "migu" {
		return miguSongDetail{}, errNotMiguTrack
	}
	contentID := strings.TrimPrefix(track.ID, "migu:")
	if contentID == "" {
		return miguSongDetail{}, errors.New("缺少咪咕 contentId")
	}
	if strings.TrimSpace(track.CopyrightID) == "" {
		return miguSongDetail{}, errors.New("缺少咪咕 copyrightId")
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
		return miguSongDetail{}, err
	}

	var res struct {
		Code string `json:"code"`
		Info string `json:"info"`
		Data struct {
			LRCURL     string   `json:"lrcUrl"`
			MRCURL     string   `json:"mrcUrl"`
			TRCURL     string   `json:"trcUrl"`
			CannotCode string   `json:"cannotCode"`
			Song       miguSong `json:"song"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return miguSongDetail{}, err
	}
	if res.Code != "" && res.Code != "000000" {
		return miguSongDetail{}, fmt.Errorf("咪咕歌曲详情：%s", cleanDisplay(res.Info))
	}

	detail := miguSongDetail{
		Track:  track,
		Album:  track.Album,
		Cover:  track.Cover,
		LRCURL: miguHTTPSURL(res.Data.LRCURL),
		MRCURL: miguHTTPSURL(res.Data.MRCURL),
		TRCURL: miguHTTPSURL(res.Data.TRCURL),
	}
	if info := miguSongTrack(res.Data.Song); info.ID != "" {
		detail.Track = mergeTrackInfo(track, info)
		detail.Album = firstNonEmpty(info.Album, detail.Album)
		detail.Cover = firstNonEmpty(miguSongCover(res.Data.Song), detail.Cover)
		detail.Tags = append([]string(nil), res.Data.Song.ShowTags...)
		detail.SongID = strings.TrimSpace(string(res.Data.Song.SongID))
		seen := map[string]bool{}
		for _, format := range res.Data.Song.AudioFormats {
			formatType := strings.ToUpper(strings.TrimSpace(format.FormatType))
			if formatType == "" || seen[formatType] {
				continue
			}
			seen[formatType] = true
			detail.Formats = append(detail.Formats, formatType)
		}
	}
	if detail.LRCURL != "" {
		lyricBody, err := fetchBytes(ctx, detail.LRCURL)
		if err != nil {
			detail.LyricErr = "歌词加载失败：" + err.Error()
		} else if looksLikeHTML(lyricBody) {
			detail.LyricErr = "歌词加载失败：咪咕返回了网页而不是歌词文本"
		} else {
			doc := lyrics.ParseLRC(string(lyricBody))
			detail.Lyrics = lyrics.StripHeader(doc.Lines, detail.Track.Title, detail.Track.Artist)
			detail.PlainLyrics = doc.Plain
			detail.LyricTitle = doc.Title
			detail.LyricArtist = doc.Artist
			detail.LyricAlbum = doc.Album
			detail.LyricBy = doc.By
		}
	}
	for _, rawURL := range []string{detail.MRCURL, detail.TRCURL} {
		if rawURL == "" {
			continue
		}
		body, err := fetchBytes(ctx, rawURL)
		if err != nil || looksLikeHTML(body) {
			continue
		}
		text, err := lyrics.DecryptMRC(body)
		if err != nil {
			continue
		}
		doc := lyrics.ParseMRC(text)
		lines := lyrics.StripHeader(doc.Lines, detail.Track.Title, detail.Track.Artist)
		if lyrics.WordCount(lines) == 0 {
			continue
		}
		detail.Lyrics = lines
		detail.PlainLyrics = false
		detail.LyricErr = ""
		if doc.Title != "" {
			detail.LyricTitle = doc.Title
		}
		if doc.Artist != "" {
			detail.LyricArtist = doc.Artist
		}
		if doc.Album != "" {
			detail.LyricAlbum = doc.Album
		}
		if doc.By != "" {
			detail.LyricBy = doc.By
		}
		break
	}
	return detail, nil
}

func mergeTrackInfo(base, info Track) Track {
	if info.Title != "" {
		base.Title = info.Title
	}
	if info.Artist != "" {
		base.Artist = info.Artist
	}
	if info.Album != "" {
		base.Album = info.Album
	}
	if info.Cover != "" {
		base.Cover = info.Cover
	}
	if info.DurationSec > 0 {
		base.DurationSec = info.DurationSec
	}
	if info.Source != "" {
		base.Source = info.Source
	}
	if info.CopyrightID != "" {
		base.CopyrightID = info.CopyrightID
	}
	if info.ResourceType != "" {
		base.ResourceType = info.ResourceType
	}
	if info.VIP {
		base.VIP = true
	}
	if info.PlayCount != "" {
		base.PlayCount = info.PlayCount
	}
	return base
}

func miguHTTPSURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "http://") {
		return "https://" + strings.TrimPrefix(raw, "http://")
	}
	return raw
}

func looksLikeHTML(body []byte) bool {
	head := strings.ToLower(strings.TrimSpace(string(body[:min(len(body), 256)])))
	return strings.HasPrefix(head, "<!doctype") || strings.HasPrefix(head, "<html")
}
