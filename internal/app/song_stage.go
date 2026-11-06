package app

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"freewave/internal/lyrics"
)

func (a *app) songStage(c *ui.Context, p songColors, detail miguSongDetail, active int) {
	current, next := lyrics.StagePair(detail.Lyrics, active)
	stage := ui.Box(c).FillWidth().Grow(1).MinHeight(0).Radius(24).Clip().
		Background(ui.RGBA(255, 255, 255, 0.035)).
		Border(1, songWhite.Alpha(0.075))
	stage.Draw(func(pt *ui.Painter, r ui.Rect) {
		a.drawSongStage(pt, r, p, detail.Track)
	})
	stage.Children(func() {
		ui.Column(c).Fill().Padding(28, 30).Children(func() {
			ui.Box(c).Grow(1)
			if current >= 0 {
				a.songStageLyric(c, detail.Lyrics[current], current, next, detail.Lyrics)
			} else {
				ui.Text(c, displayTitle(detail.Track.Title)).FontSize(36).FontWeight(790).
					LetterSpacing(-0.8).TextColor(songWhite.Alpha(0.92)).MaxLines(2)
			}
			if next >= 0 {
				ui.Text(c, detail.Lyrics[next].Text).FontSize(16).FontWeight(550).
					TextColor(songWhite.Alpha(0.3)).SingleLine().Margin(9, 0, 0, 0)
			}
		})
	})
}

func (a *app) songStageLyric(c *ui.Context, line lyricLine, current, next int, lines []lyricLine) {
	pos := time.Duration(0)
	live := a.pl != nil && a.pl.hasCurrent && a.pl.current.ID == a.songDetail.Track.ID
	if live {
		pos = a.pl.position()
	}
	nextAt := lyrics.LineEnd(lines, current)
	if next >= 0 {
		nextAt = lines[next].At
	}
	words := lyrics.WordsFor(line, nextAt)
	if len(words) == 0 {
		ui.Text(c, line.Text).FontSize(35).FontWeight(790).LineHeight(1.22).
			LetterSpacing(-0.8).TextColor(songWhite.Alpha(0.92)).MaxLines(3)
		return
	}
	spans := make([]ui.Span, 0, len(words))
	for _, word := range words {
		color := songWhite.Alpha(0.26)
		span := ui.Span{Text: word.Text, Color: color}
		switch {
		case !live:
			color = songWhite.Alpha(0.9)
		case pos >= word.At+word.Dur && word.Dur > 0:
			color = songWhite.Alpha(0.9)
		case pos >= word.At:
			color = songWhite
		}
		span.Color = color
		spans = append(spans, span)
	}
	ui.RichText(c, spans...).FontSize(35).FontWeight(790).LineHeight(1.22).
		LetterSpacing(-0.8)
}

func (a *app) drawSongStage(pt *ui.Painter, r ui.Rect, p songColors, track Track) {
	now := pt.Now()
	pos := time.Duration(0)
	same := a.pl != nil && a.pl.hasCurrent && a.pl.current.ID == track.ID
	playing := same && a.pl.playing
	if same {
		pos = a.pl.position()
	}
	if playing {
		pt.AnimationFrame()
	}
	useEnvelope := a.envelope != nil && same
	sample := a.stageSampleAt(pos, now, track)
	pt.FillGradient(r, ui.LinearGradient{Angle: 145, Oklab: true, Stops: []ui.GradientStop{
		{Color: p.tint.Alpha(0.16), At: 0},
		{Color: songInk.Alpha(0), At: 0.68},
		{Color: ui.RGBA(255, 255, 255, 0.018), At: 1},
	}}, 24)
	drawStageWave(pt, r, p.tint, a, pos, now, sample, useEnvelope)
}

func (a *app) stageSampleAt(pos time.Duration, now time.Time, track Track) waveSample {
	if a.envelope != nil && a.pl != nil && a.pl.hasCurrent && a.pl.current.ID == track.ID {
		low, mid, high, amp := a.envelope.at(pos)
		return waveSample{Low: low, Mid: mid, High: high, Amp: amp}
	}
	phase := pos.Seconds() + float64(track.DurationSec)*0.013 + now.Sub(time.Unix(0, 0)).Seconds()*0.08
	wave := func(offset, speed float64) float32 {
		return float32((math.Sin(phase*speed+offset) + 1) * 0.5)
	}
	low := 0.24 + wave(0, 1.2)*0.36
	mid := 0.18 + wave(1.7, 1.8)*0.42
	high := 0.1 + wave(3.2, 2.5)*0.3
	return waveSample{Low: low, Mid: mid, High: high, Amp: (low + mid + high) / 3}
}

func drawStageWave(pt *ui.Painter, r ui.Rect, tint ui.Color, a *app, pos time.Duration, now time.Time, sample waveSample, useEnvelope bool) {
	const bars = 58
	gap := float32(4)
	barW := max(float32(2), (r.W-80-gap*float32(bars-1))/float32(bars))
	start := r.X + (r.W-(barW*bars+gap*float32(bars-1)))/2
	base := r.Y + r.H*0.49
	for i := 0; i < bars; i++ {
		at := pos - time.Duration(float64(bars/2-i)*72*float64(time.Millisecond))
		v := sample.Amp
		if useEnvelope {
			low, mid, high, amp := a.envelope.at(at)
			v = low*0.58 + mid*0.3 + high*0.12 + amp*0.35
		} else {
			phase := at.Seconds() + float64(now.UnixNano()%1_000_000_000)/1e9*0.2
			v = float32((math.Sin(phase*3.4+float64(i)*0.47) + 1) * 0.28)
			v += sample.Amp * 0.55
		}
		h := 8 + v*r.H*0.46
		color := tint.Mix(ui.RGB(255, 255, 255), 0.34)
		if i%7 == 0 {
			color = songWhite
		}
		pt.Fill(ui.Rect{
			X: start + float32(i)*(barW+gap),
			Y: base - h/2,
			W: barW,
			H: h,
		}, color.Alpha(0.28+v*0.48), barW/2)
	}
}
