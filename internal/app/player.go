package app

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	_ "github.com/egoist/mygo/audio/mp3"

	"freewave/internal/migu"
)

type repeatMode int

const (
	repeatOff repeatMode = iota
	repeatAll
	repeatOne
)

type playbackFailureKind int

const (
	playbackFailureNone playbackFailureKind = iota
	playbackFailureUnknown
	playbackFailureVIP
	playbackFailureAuth
)

const (
	seekFadeDelay       = 25 * time.Millisecond
	crossfadeDuration   = 1200 * time.Millisecond
	crossfadeStartLead  = crossfadeDuration + 150*time.Millisecond
	crossfadeFrameDelay = 16 * time.Millisecond
)

var streamClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
	},
}

// player owns the queue and the one audio output. Its methods run on the UI
// thread; downloads and connections happen on goroutines and post back.
type player struct {
	app *app

	queue      []Track
	at         int
	current    Track
	hasCurrent bool
	out        *audio.Player
	playing    bool
	loading    bool
	progress   float64
	err        string
	errKind    playbackFailureKind
	volume     float64
	gen        int
	cancel     context.CancelFunc
	shuffle    bool
	repeat     repeatMode

	duration time.Duration
	npKey    string
	pending  time.Duration

	seekGen    int
	seekTarget time.Duration
	seeking    bool

	prefetchMu   sync.Mutex
	prefetch     *prefetchJob
	cacheRootDir string

	mixMu         sync.Mutex
	voices        []*audioVoice
	fadeGen       int
	autoFadeTimer *time.Timer
}

type prefetchJob struct {
	track  Track
	key    string
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}

type audioVoice struct {
	out  *audio.Player
	gain float64
}

type fadingVoice struct {
	voice     *audioVoice
	startGain float64
}

func newPlayer(a *app) *player {
	p := &player{app: a, at: -1, volume: a.volume}
	audio.SetLatency(audio.LatencyPlayback)
	mygo.NowPlaying.OnCommand(func(cmd mygo.MediaCommand, pos time.Duration) {
		p.app.post(func() {
			switch cmd {
			case mygo.MediaPlay:
				if !p.playing {
					p.toggle()
				}
			case mygo.MediaPause:
				if p.playing {
					p.toggle()
				}
			case mygo.MediaStop:
				p.stop()
			case mygo.MediaNext:
				p.next()
			case mygo.MediaPrevious:
				p.prev()
			case mygo.MediaSeek:
				p.seek(pos)
			}
		})
	})
	return p
}

func (p *player) setQueue(tracks []Track) {
	p.queue = append([]Track(nil), tracks...)
}

func (p *player) playQueue(tracks []Track, start int) {
	if len(tracks) == 0 {
		return
	}
	if start < 0 || start >= len(tracks) {
		start = 0
	}
	p.queue = append([]Track(nil), tracks...)
	p.at = start
	p.load()
}

func (p *player) load() {
	p.loadAt(0)
}

func (p *player) loadAt(pos time.Duration) {
	if p.at < 0 || p.at >= len(p.queue) {
		return
	}
	p.cancelAutoFade()
	p.gen++
	gen := p.gen
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	// Keep the current voice alive until the replacement is ready, then
	// crossfade the two instead of cutting to silence.
	p.out = nil
	p.current = p.queue[p.at]
	p.hasCurrent = true
	p.seeking = false
	p.cancelPrefetchExcept(p.current)
	if p.app != nil {
		p.app.followPlayerTrack(p.current)
	}
	p.playing = false
	p.loading = true
	p.pending = max(0, pos)
	p.progress = 0
	p.err = ""
	p.errKind = playbackFailureNone
	p.duration = p.current.Duration()
	p.npKey = ""
	p.app.resetStageAudio(p.current.ID)
	p.app.refresh()
	p.syncNowPlaying(nil)

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	go p.openTrack(ctx, gen, p.current)
}

// restore brings back the queue and track from the last session without
// downloading or playing anything; the first toggle resumes at the saved spot.
func (p *player) restore(state playbackState) {
	normalized := normalizePlayback(&state)
	if normalized == nil {
		return
	}
	state = *normalized
	p.queue = append([]Track(nil), state.Queue...)
	p.at = state.Index
	p.current = p.queue[p.at]
	p.hasCurrent = true
	p.shuffle = state.Shuffle
	p.repeat = state.Repeat
	p.duration = p.current.Duration()
	p.pending = seconds(state.Position)
	p.progress = 1
	p.playing = false
	p.loading = false
	p.err = ""
	p.errKind = playbackFailureNone
	p.syncNowPlaying(nil)
}

func (p *player) openTrack(ctx context.Context, gen int, t Track) {
	file, err := p.downloadTrack(ctx, gen, t)
	if err != nil {
		p.fail(gen, err)
		return
	}
	out, err := audio.OpenStream(file)
	if err != nil {
		file.Close()
		_ = os.Remove(file.Name())
		p.fail(gen, err)
		return
	}
	go p.app.loadEnvelope(gen, t.ID, file.Name())
	p.app.post(func() {
		if p.gen != gen {
			out.Close()
			return
		}
		p.start(out, t, nil)
		p.loadArtwork(gen, t)
	})
}

func (p *player) loadArtwork(gen int, t Track) {
	if t.Cover == "" {
		return
	}
	go func() {
		art, err := p.fetchArtwork(t.Cover)
		if err != nil || len(art) == 0 {
			return
		}
		p.app.post(func() {
			if p.gen == gen && p.hasCurrent && p.current.ID == t.ID {
				p.syncNowPlaying(art)
			}
		})
	}()
}

func (p *player) start(out *audio.Player, t Track, art []byte) {
	p.out = out
	p.addVoice(out, 0)
	p.playing = true
	p.loading = false
	p.progress = 1
	p.err = ""
	p.errKind = playbackFailureNone
	if d := out.Duration(); d > 0 {
		p.duration = d
	}
	if p.pending > 0 {
		out.Seek(p.pending)
		p.pending = 0
	}
	gen := p.gen
	out.OnEnd(func() {
		p.app.post(func() { p.ended(gen) })
	})
	out.Play()
	p.crossfadeTo(out)
	p.app.noteRecent(t)
	p.syncNowPlaying(art)
	p.app.refresh()
	p.schedulePrefetch()
	p.scheduleAutoFade()
}

func (p *player) fail(gen int, err error) {
	p.app.post(func() {
		if p.gen != gen {
			return
		}
		p.cancelAutoFade()
		p.stopVoices(true)
		p.loading = false
		p.playing = false
		p.pending = 0
		p.err = friendlyError(err)
		switch {
		case errors.Is(err, errMiguVIPRequired):
			p.errKind = playbackFailureVIP
		case errors.Is(err, errMiguTokenStale):
			p.errKind = playbackFailureAuth
			p.app.handlePlaybackAuthFailure(p.current.ID)
		case p.err != "":
			p.errKind = playbackFailureUnknown
		default:
			p.errKind = playbackFailureNone
		}
		p.app.refresh()
	})
}

func friendlyError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case errors.Is(err, context.Canceled):
		return ""
	case errors.Is(err, errMiguTokenStale):
		return "登录已过期，请重新登录"
	case errors.Is(err, errMiguVIPRequired):
		if _, reason, ok := strings.Cut(msg, "："); ok {
			if reason = strings.TrimSpace(reason); reason != "" {
				reason, _, _ = strings.Cut(reason, "。")
				return reason
			}
		}
		return msg
	case strings.Contains(msg, "no such host"),
		strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "network is unreachable"):
		return "网络连接失败，请稍后重试"
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline"),
		strings.Contains(msg, "i/o timeout"):
		return "连接超时，请稍后重试"
	case strings.Contains(msg, "unknown format"):
		return "这个音频格式暂时无法播放"
	}
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "http://") ||
		strings.Contains(lower, "https://") ||
		strings.Contains(lower, "bad gateway") ||
		strings.Contains(lower, "status") ||
		strings.Contains(lower, "invalid character") ||
		strings.Contains(lower, "unexpected end") ||
		strings.Contains(lower, "parse") ||
		strings.Contains(lower, "decode") {
		return "操作没有完成，请稍后重试"
	}
	if len([]rune(msg)) > 90 {
		msg = string([]rune(msg)[:90]) + "…"
	}
	return msg
}

func (p *player) toggle() {
	if !p.hasCurrent || p.loading {
		return
	}
	if p.out == nil {
		p.loadAt(p.pending)
		return
	}
	if p.playing {
		p.cancelAutoFade()
		p.pauseVoices()
		p.playing = false
	} else {
		p.out.Play()
		p.playing = true
		p.scheduleAutoFade()
	}
	p.syncNowPlaying(nil)
	p.app.refresh()
}

func (p *player) retry() {
	if !p.hasCurrent || p.loading {
		return
	}
	p.loadAt(p.position())
}

func (p *player) stop() {
	p.cancelAutoFade()
	p.stopVoices(false)
	p.playing = false
	p.syncNowPlaying(nil)
	p.app.refresh()
}

func (p *player) next() {
	if len(p.queue) == 0 {
		return
	}
	if p.repeat == repeatOne && p.hasCurrent {
		if p.out != nil {
			if p.playing {
				p.seek(0)
			} else {
				p.out.Seek(0)
				p.out.Play()
				p.playing = true
				p.syncNowPlaying(nil)
				p.app.refresh()
				p.scheduleAutoFade()
			}
		} else {
			p.loadAt(p.pending)
		}
		return
	}
	next := p.nextIndex()
	if next < 0 {
		p.playing = false
		if p.out != nil {
			p.out.Pause()
		}
		p.syncNowPlaying(nil)
		p.app.refresh()
		return
	}
	p.at = next
	p.load()
}

// nextIndex is shared by playback and prefetch so shuffle picks the same track
// that will actually start next.
func (p *player) nextIndex() int {
	if len(p.queue) == 0 || p.at < 0 || p.at >= len(p.queue) {
		return -1
	}
	if p.repeat == repeatOne {
		return p.at
	}
	next := p.at + 1
	if p.shuffle && len(p.queue) > 1 {
		hash := sha1.Sum([]byte(p.current.ID))
		jitter := int(uint32(hash[0])<<8|uint32(hash[1])) % (len(p.queue) - 1)
		next = (p.at + 1 + jitter) % len(p.queue)
		if next == p.at {
			next = (next + 1) % len(p.queue)
		}
	}
	if next >= len(p.queue) {
		if p.repeat == repeatOff {
			return -1
		}
		next = 0
	}
	return next
}

func (p *player) prev() {
	if p.out != nil && p.out.Position() > 4*time.Second {
		p.seek(0)
		return
	}
	if len(p.queue) == 0 {
		return
	}
	prev := p.at - 1
	if prev < 0 {
		if p.repeat == repeatOff {
			return
		}
		prev = len(p.queue) - 1
	}
	p.at = prev
	p.load()
}

func (p *player) jumpTo(index int) {
	if index < 0 || index >= len(p.queue) {
		return
	}
	p.at = index
	p.load()
}

func (p *player) seek(pos time.Duration) {
	p.cancelAutoFade()
	if p.out == nil {
		p.pending = max(0, pos)
		p.syncNowPlaying(nil)
		p.app.refresh()
		return
	}
	pos = max(0, pos)
	out := p.out
	if !p.playing {
		p.seeking = false
		out.Seek(pos)
		p.syncNowPlaying(nil)
		p.app.refresh()
		return
	}
	// Seeking restarts the decoder at a new waveform, which clicks if it is
	// mixed next to the old one. Pause fades the old audio out first, then
	// Play fades the new position in.
	p.seekGen++
	gen := p.seekGen
	p.seekTarget = pos
	p.seeking = true
	out.Pause()
	time.AfterFunc(seekFadeDelay, func() {
		p.app.post(func() {
			if p.seekGen != gen || p.out != out {
				return
			}
			out.Seek(pos)
			if p.playing {
				out.Play()
			}
			p.seeking = false
			p.syncNowPlaying(nil)
			p.app.refresh()
			p.scheduleAutoFade()
		})
	})
	p.syncNowPlaying(nil)
	p.app.refresh()
}

func (p *player) position() time.Duration {
	if p.out == nil {
		return p.pending
	}
	if p.seeking {
		return p.seekTarget
	}
	return p.out.Position()
}

func (p *player) setVolume(v float64) {
	p.mixMu.Lock()
	defer p.mixMu.Unlock()
	p.volume = min(1, max(0, v))
	for _, voice := range p.voices {
		voice.out.SetVolume(p.volume * voice.gain)
	}
}

func (p *player) addVoice(out *audio.Player, gain float64) *audioVoice {
	p.mixMu.Lock()
	defer p.mixMu.Unlock()
	for _, voice := range p.voices {
		if voice.out == out {
			voice.gain = gain
			voice.out.SetVolume(p.volume * gain)
			return voice
		}
	}
	voice := &audioVoice{out: out, gain: gain}
	voice.out.SetVolume(p.volume * gain)
	p.voices = append(p.voices, voice)
	return voice
}

func (p *player) crossfadeTo(out *audio.Player) {
	p.mixMu.Lock()
	p.fadeGen++
	gen := p.fadeGen
	var target *audioVoice
	var retiring []fadingVoice
	for _, voice := range p.voices {
		if voice.out == out {
			target = voice
			continue
		}
		retiring = append(retiring, fadingVoice{voice: voice, startGain: voice.gain})
	}
	if target == nil {
		target = &audioVoice{out: out}
		p.voices = append(p.voices, target)
	}
	target.gain = 0
	target.out.SetVolume(0)
	if len(retiring) == 0 {
		target.gain = 1
		target.out.SetVolume(p.volume)
		p.mixMu.Unlock()
		return
	}
	p.mixMu.Unlock()

	go p.runCrossfade(gen, target, retiring)
}

func (p *player) runCrossfade(gen int, target *audioVoice, retiring []fadingVoice) {
	ticker := time.NewTicker(crossfadeFrameDelay)
	defer ticker.Stop()
	start := time.Now()
	for range ticker.C {
		progress := min(1, max(0, float64(time.Since(start))/float64(crossfadeDuration)))
		incoming := math.Sin(progress * math.Pi / 2)
		outgoing := math.Cos(progress * math.Pi / 2)

		p.mixMu.Lock()
		if p.fadeGen != gen {
			p.mixMu.Unlock()
			return
		}
		if !p.hasVoiceLocked(target) {
			p.mixMu.Unlock()
			return
		}
		target.gain = incoming
		target.out.SetVolume(p.volume * incoming)
		for _, fade := range retiring {
			if !p.hasVoiceLocked(fade.voice) {
				continue
			}
			fade.voice.gain = fade.startGain * outgoing
			fade.voice.out.SetVolume(p.volume * fade.voice.gain)
		}
		if progress < 1 {
			p.mixMu.Unlock()
			continue
		}
		for _, fade := range retiring {
			p.removeVoiceLocked(fade.voice)
			fade.voice.out.SetVolume(0)
			fade.voice.out.Close()
		}
		target.gain = 1
		target.out.SetVolume(p.volume)
		p.mixMu.Unlock()
		return
	}
}

func (p *player) hasVoiceLocked(want *audioVoice) bool {
	for _, voice := range p.voices {
		if voice == want {
			return true
		}
	}
	return false
}

func (p *player) removeVoiceLocked(want *audioVoice) {
	for i, voice := range p.voices {
		if voice == want {
			p.voices = append(p.voices[:i], p.voices[i+1:]...)
			return
		}
	}
}

func (p *player) pauseVoices() {
	p.mixMu.Lock()
	defer p.mixMu.Unlock()
	p.fadeGen++
	for _, voice := range p.voices {
		voice.out.Pause()
	}
	p.closeRetiringVoicesLocked()
}

func (p *player) stopVoices(closeAll bool) {
	p.mixMu.Lock()
	defer p.mixMu.Unlock()
	p.fadeGen++
	for _, voice := range p.voices {
		voice.out.Stop()
	}
	if closeAll {
		for _, voice := range p.voices {
			voice.out.Close()
		}
		p.voices = nil
		return
	}
	p.closeRetiringVoicesLocked()
}

func (p *player) closeRetiringVoicesLocked() {
	kept := p.voices[:0]
	for _, voice := range p.voices {
		if voice.out == p.out {
			kept = append(kept, voice)
			continue
		}
		voice.out.SetVolume(0)
		voice.out.Close()
	}
	p.voices = kept
}

func (p *player) scheduleAutoFade() {
	p.cancelAutoFade()
	if p.out == nil || !p.playing || p.loading || p.repeat == repeatOne {
		return
	}
	next := p.nextIndex()
	if next < 0 || next == p.at || p.duration <= 0 {
		return
	}
	delay := p.duration - p.position() - crossfadeStartLead
	if delay < 100*time.Millisecond {
		delay = 100 * time.Millisecond
	}
	gen := p.gen
	p.autoFadeTimer = time.AfterFunc(delay, func() {
		if p.app == nil {
			return
		}
		p.app.post(func() { p.autoCrossfade(gen) })
	})
}

func (p *player) autoCrossfade(gen int) {
	p.autoFadeTimer = nil
	if p.gen != gen || !p.playing || p.loading || p.out == nil || p.repeat == repeatOne {
		return
	}
	next := p.nextIndex()
	if next < 0 || next == p.at {
		return
	}
	p.at = next
	p.load()
}

func (p *player) cancelAutoFade() {
	if p.autoFadeTimer != nil {
		p.autoFadeTimer.Stop()
		p.autoFadeTimer = nil
	}
}

func (p *player) ended(gen int) {
	if p.gen != gen || !p.playing {
		return
	}
	p.cancelAutoFade()
	if p.repeat == repeatOne {
		if p.out != nil {
			p.out.Seek(0)
			p.out.Play()
			p.syncNowPlaying(nil)
			p.scheduleAutoFade()
		}
		return
	}
	p.next()
}

func (p *player) syncNowPlaying(art []byte) {
	if p.app != nil {
		p.app.setMenuBarPlaying(p.playing)
	}
	if !p.hasCurrent {
		mygo.NowPlaying.Clear()
		return
	}
	t := p.current
	state := mygo.PlaybackPaused
	if p.playing {
		state = mygo.PlaybackPlaying
	}
	dur := p.duration
	commands := mygo.MediaPlay | mygo.MediaPause | mygo.MediaNext | mygo.MediaPrevious | mygo.MediaSeek
	key := fmt.Sprintf("%s|%d", t.ID, len(art))
	if key != p.npKey {
		p.npKey = key
		mygo.NowPlaying.Set(mygo.NowPlayingInfo{
			Title:    t.Title,
			Artist:   t.Subtitle(),
			Album:    t.Album,
			Artwork:  art,
			Duration: dur,
			Position: p.position(),
			State:    state,
			Commands: commands,
		})
		return
	}
	mygo.NowPlaying.SetPlayback(state, p.position())
}

func (p *player) fetchArtwork(rawURL string) ([]byte, error) {
	if rawURL == "" {
		return nil, errors.New("no artwork")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return fetchBytes(ctx, rawURL)
}

func (p *player) cacheDir() string {
	if p.cacheRootDir != "" {
		return p.cacheRootDir
	}
	return audioCacheDir()
}

func (p *player) cachePath(rawURL string) string {
	sum := sha1.Sum([]byte(rawURL))
	ext := ".mp3"
	if u, err := url.Parse(rawURL); err == nil {
		if e := strings.ToLower(path.Ext(u.Path)); e == ".mp3" || e == ".ogg" {
			ext = e
		}
	}
	return filepath.Join(p.cacheDir(), hex.EncodeToString(sum[:])+ext)
}

func (p *player) cacheKey(t Track) string {
	if t.URL != "" {
		return t.URL
	}
	token := p.app.miguToken
	tokenID := "anonymous"
	if token != "" {
		sum := sha256.Sum256([]byte(token))
		tokenID = hex.EncodeToString(sum[:8])
	}
	return t.Source + ":" + t.ID + ":auth=" + tokenID
}

func (p *player) downloadTrack(ctx context.Context, gen int, t Track) (*os.File, error) {
	key := p.cacheKey(t)
	if err := p.waitForPrefetch(ctx, key); err != nil {
		return nil, err
	}
	progress := func(frac float64) {
		p.app.post(func() {
			if p.gen == gen {
				p.progress = frac
				p.app.refresh()
			}
		})
	}
	return p.fetchTrackToCache(ctx, key, t, progress)
}

func (p *player) fetchTrackToCache(ctx context.Context, key string, t Track, progress func(float64)) (*os.File, error) {
	dest := p.cachePath(key)
	if fi, err := os.Stat(dest); err == nil && fi.Size() > 1024 {
		if progress != nil {
			progress(1)
		}
		return os.Open(dest)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return nil, err
	}
	if t.NeedsResolve() {
		if t.Source != "migu" {
			return nil, errors.New("暂不支持这个音源")
		}
		streamURL, err := migu.ResolveURL(ctx, p.app.miguToken, t)
		if err != nil {
			return nil, err
		}
		t.URL = streamURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", migu.UserAgent)
	req.Header.Set("Referer", "https://music.migu.cn/")
	req.Header.Set("Accept", "audio/mpeg,audio/ogg;q=0.9,*/*;q=0.5")
	res, err := streamClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: %s", res.Status)
	}
	if res.ContentLength > maxAudioDownloadBytes ||
		strings.Contains(strings.ToLower(res.Header.Get("Content-Type")), "text/html") {
		return nil, errors.New("下载地址没有返回音频")
	}
	f, err := os.CreateTemp(filepath.Dir(dest), filepath.Base(dest)+".*.part")
	if err != nil {
		return nil, err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		f.Close()
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	total := res.ContentLength
	var got int64
	buf := make([]byte, 64<<10)
	last := time.Now()
	head := make([]byte, 0, 512)
	for {
		n, readErr := res.Body.Read(buf)
		if n > 0 {
			if len(head) < cap(head) {
				head = append(head, buf[:min(n, cap(head)-len(head))]...)
				if bytes.HasPrefix(bytes.TrimSpace(head), []byte("<")) {
					return nil, errors.New("download returned a web page")
				}
			}
			if _, err := f.Write(buf[:n]); err != nil {
				return nil, err
			}
			got += int64(n)
			if got > maxAudioDownloadBytes {
				return nil, errors.New("音频文件过大，已停止下载")
			}
			if total > 0 && time.Since(last) > 120*time.Millisecond {
				last = time.Now()
				frac := float64(got) / float64(total)
				if progress != nil {
					progress(min(1, max(0, frac)))
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
		if n == 0 {
			return nil, io.ErrNoProgress
		}
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	ok = true
	if err := os.Rename(tmp, dest); err != nil {
		return nil, err
	}
	if progress != nil {
		progress(1)
	}
	return os.Open(dest)
}

func (p *player) schedulePrefetch() {
	next := p.nextIndex()
	if next < 0 || next == p.at || next >= len(p.queue) {
		return
	}
	track := p.queue[next]
	if track.Cover != "" && p.app != nil {
		p.app.requestArt(track.Cover)
	}

	key := p.cacheKey(track)
	p.prefetchMu.Lock()
	if job := p.prefetch; job != nil && job.key == key {
		select {
		case <-job.done:
			if job.err == nil {
				p.prefetchMu.Unlock()
				return
			}
		default:
			p.prefetchMu.Unlock()
			return
		}
	}
	if p.prefetch != nil {
		p.prefetch.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	job := &prefetchJob{
		track:  track,
		key:    key,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	p.prefetch = job
	p.prefetchMu.Unlock()

	go func() {
		file, err := p.fetchTrackToCache(ctx, key, track, nil)
		if file != nil {
			file.Close()
		}
		job.err = err
		close(job.done)
	}()
}

func (p *player) waitForPrefetch(ctx context.Context, key string) error {
	p.prefetchMu.Lock()
	job := p.prefetch
	p.prefetchMu.Unlock()
	if job == nil || job.key != key {
		return nil
	}
	select {
	case <-job.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *player) cancelPrefetchExcept(track Track) {
	if p.app == nil {
		return
	}
	key := p.cacheKey(track)
	p.prefetchMu.Lock()
	job := p.prefetch
	if job != nil && job.key != key {
		job.cancel()
	}
	p.prefetchMu.Unlock()
}
