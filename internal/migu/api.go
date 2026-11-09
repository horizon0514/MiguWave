package migu

import (
	"context"
	"net/http"
	"strings"
)

type RankInfo = miguRankInfo
type RankSummary = miguRankSummary
type Rank = miguRank
type HomeData = miguHomeData
type Playlist = miguPlaylist
type PlaylistSection = miguPlaylistSection
type PlaylistDetail = miguPlaylistDetail
type Singer = miguSinger
type SingerDetail = miguSingerDetail
type SingerTab = miguSingerTab
type Radio = miguRadio
type RadioGroup = miguRadioGroup
type RadioDetail = miguRadioDetail
type Profile = miguProfile
type SongDetail = miguSongDetail
type QRState = miguQRState
type QRCode = miguQRCode
type QRPoll = miguQRPoll
type AuthSession = miguAuthSession

var (
	RankCatalog     = miguRankCatalog
	SingerTabs      = miguSingerTabs
	ErrVIPRequired  = errMiguVIPRequired
	ErrTokenStale   = errMiguTokenStale
	ErrNotMiguTrack = errNotMiguTrack
)

const (
	QRWaiting   = miguQRWaiting
	QRScanned   = miguQRScanned
	QRExpired   = miguQRExpired
	QRConfirmed = miguQRConfirmed

	UserAgent = miguUA
)

func Search(ctx context.Context, query string, page, pageSize int) ([]Track, error) {
	return searchMigu(ctx, query, page, pageSize)
}

func FetchRank(ctx context.Context, id string, limit int) (Rank, error) {
	return fetchMiguRank(ctx, id, limit)
}

func FetchHome(ctx context.Context) (HomeData, error) {
	return fetchMiguHome(ctx)
}

func FetchHotSearches(ctx context.Context) ([]string, error) {
	return fetchMiguHotSearches(ctx)
}

func ResolveURL(ctx context.Context, token string, track Track) (string, error) {
	return resolveMiguURL(ctx, token, track)
}

func NormalizeToken(raw string) string {
	return normalizeMiguToken(raw)
}

func LoadToken() string {
	return loadMiguToken()
}

func SaveToken(token string) error {
	return saveMiguToken(token)
}

func RefreshToken(ctx context.Context, token string) (string, error) {
	return refreshMiguToken(ctx, token)
}

func FetchPlaylistSquare(ctx context.Context) ([]PlaylistSection, error) {
	return fetchMiguPlaylistSquare(ctx)
}

func FetchPlaylistDetail(ctx context.Context, id string, page int) (PlaylistDetail, error) {
	return fetchMiguPlaylistDetail(ctx, id, page)
}

func FetchPlaylistSongs(ctx context.Context, id string, page int) ([]Track, int, bool, error) {
	return fetchMiguPlaylistSongs(ctx, id, page)
}

func FetchSingers(ctx context.Context, tab string) ([]Singer, error) {
	return fetchMiguSingers(ctx, tab)
}

func FetchSingerDetail(ctx context.Context, singer Singer) (SingerDetail, error) {
	return fetchMiguSingerDetail(ctx, singer)
}

func FetchSingerSongs(ctx context.Context, id string, page int) ([]Track, bool, error) {
	return fetchMiguSingerSongs(ctx, id, page)
}

func FetchRadioGroups(ctx context.Context) ([]RadioGroup, error) {
	return fetchMiguRadioGroups(ctx)
}

func FetchRadioDetail(ctx context.Context, radio Radio, maxTracks int) (RadioDetail, error) {
	return fetchMiguRadioDetail(ctx, radio, maxTracks)
}

func FetchPersonalRadio(ctx context.Context, token string, radio Radio, maxTracks int) (RadioDetail, error) {
	return fetchMiguPersonalRadio(ctx, token, radio, maxTracks)
}

func FetchSongDetail(ctx context.Context, token string, track Track) (SongDetail, error) {
	return fetchMiguSongDetail(ctx, token, track)
}

func CheckToken(ctx context.Context, token string) error {
	return checkMiguToken(ctx, token)
}

func FetchProfile(ctx context.Context, token string) (Profile, error) {
	return fetchMiguProfile(ctx, token)
}

func DedupeTracks(tracks []Track) []Track {
	return miguDedupeTracks(tracks)
}

func CleanDisplay(s string) string {
	return cleanDisplay(s)
}

func ResolveCoverHQ(ctx context.Context, track Track) string {
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
		hits, err := Search(ctx, query, 1, 20)
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

func NewAuthSession() (*AuthSession, error) {
	return newMiguAuthSession()
}

func StartQR(ctx context.Context, session *AuthSession) (QRCode, error) {
	return session.startQR(ctx)
}

func PollQR(ctx context.Context, session *AuthSession, sessionID string) (QRPoll, error) {
	return session.pollQR(ctx, sessionID)
}

func CompleteLogin(ctx context.Context, session *AuthSession, redirectURL, token string) (string, error) {
	return session.completeLogin(ctx, redirectURL, token)
}

func ExchangeLoginToken(ctx context.Context, client *http.Client, jar http.CookieJar, loginToken string) (string, error) {
	return exchangeMiguLoginToken(ctx, client, jar, loginToken)
}

func LoginTokenFromURL(rawURL string) string {
	return miguLoginTokenFromURL(rawURL)
}

func TokenParamFromURL(rawURL string, keys ...string) string {
	return miguTokenParamFromURL(rawURL, keys...)
}
