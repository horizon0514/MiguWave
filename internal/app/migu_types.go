package app

import "freewave/internal/migu"

type miguRankInfo = migu.RankInfo
type miguRankSummary = migu.RankSummary
type miguRank = migu.Rank
type miguHomeData = migu.HomeData
type miguPlaylist = migu.Playlist
type miguPlaylistSection = migu.PlaylistSection
type miguPlaylistDetail = migu.PlaylistDetail
type miguSinger = migu.Singer
type miguSingerDetail = migu.SingerDetail
type miguSingerTab = migu.SingerTab
type miguRadio = migu.Radio
type miguRadioGroup = migu.RadioGroup
type miguRadioDetail = migu.RadioDetail
type miguProfile = migu.Profile
type miguSongDetail = migu.SongDetail
type miguQRState = migu.QRState
type miguQRCode = migu.QRCode
type miguQRPoll = migu.QRPoll
type miguAuthSession = migu.AuthSession

const (
	miguQRWaiting   = migu.QRWaiting
	miguQRScanned   = migu.QRScanned
	miguQRExpired   = migu.QRExpired
	miguQRConfirmed = migu.QRConfirmed
)

var (
	miguRankCatalog    = migu.RankCatalog
	miguSingerTabs     = migu.SingerTabs
	errMiguVIPRequired = migu.ErrVIPRequired
	errMiguTokenStale  = migu.ErrTokenStale
)
