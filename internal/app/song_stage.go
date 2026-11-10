package app

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"freewave/internal/lyrics"
)

const (
	stageArmRestAngle = float32(68)
	stageArmPlayAngle = float32(99)
)

var stageArmMotion = ui.Animation{Spring: ui.SpringOptions{
	Duration: 720 * time.Millisecond,
	Bounce:   0.055,
}}

type stageDiscGeometry struct {
	cx, cy float32
	rad    float32
}

func stageDiscGeometryFor(r ui.Rect) stageDiscGeometry {
	diameter := min(r.W*0.78, r.H*0.58)
	return stageDiscGeometry{
		cx:  r.X + r.W/2,
		cy:  r.Y + r.H*0.405,
		rad: diameter / 2,
	}
}

func (a *app) songStage(c *ui.Context, p songColors, detail miguSongDetail, active int) {
	current, next := lyrics.StagePair(detail.Lyrics, active)
	reduceMotion := c.Preferences().ReduceMotion
	cover := firstNonEmpty(detail.Cover, detail.Track.Cover)
	same := a.pl != nil && a.pl.hasCurrent && a.pl.current.ID == detail.Track.ID
	playing := same && a.pl.playing
	w, h := c.Size()
	stageH := max(float32(0), h-58)
	padX := max(float32(34), min(float32(64), w*0.045))
	dockH := max(float32(124), min(float32(152), stageH*0.19))
	a.requestArt(cover)
	stage := ui.Box(c).FillWidth().Grow(1).MinHeight(0).Clip()
	armTarget := stageArmRestAngle
	if playing {
		armTarget = stageArmPlayAngle
	}
	armAngle := stage.AnimateTo("stage-tonearm", armTarget, stageArmMotion)
	stage.Draw(func(pt *ui.Painter, r ui.Rect) {
		a.drawSongStage(pt, r, p, detail.Track, reduceMotion, a.artBitmap(cover))
		drawStageTonearm(pt, r, p.tint, armAngle, playing)
	})
	stage.Children(func() {
		a.songStageIdentity(c, detail.Track, padX, w)
		if same && a.pl.err != "" {
			ui.Row(c).Absolute().Top(18).Left(padX).Right(padX).Justify(ui.Center).
				PassThrough().Children(func() {
				a.songPlaybackNotice(c, detail.Track)
			})
		}
		if current >= 0 {
			lyricWidth := min(float32(720), w*0.56)
			ui.Column(c).Absolute().Left(padX).Bottom(dockH + 16).Width(lyricWidth).
				AlignItems(ui.Start).Children(func() {
				a.songStageLyric(c, detail.Lyrics[current], current, next, detail.Lyrics)
				if next >= 0 && stageH >= 700 {
					ui.Text(c, detail.Lyrics[next].Text).FontSize(15).FontWeight(540).
						TextColor(songWhite.Alpha(0.3)).SingleLine().Margin(7, 0, 0, 0)
				}
			})
		}
		a.songStageControls(c, p, detail.Track, padX, dockH)
	})
}

func (a *app) songStageIdentity(c *ui.Context, track Track, padX, width float32) {
	meta := joinMeta(displayArtist(track.Artist), track.Album)
	ui.Column(c).Absolute().Left(padX).Top(26).Width(min(float32(390), width*0.34)).
		Gap(4).AlignItems(ui.Start).Children(func() {
		ui.Text(c, displayTitle(track.Title)).FontSize(23).FontWeight(760).
			LetterSpacing(-0.5).SingleLine().TextColor(songWhite.Alpha(0.94))
		if meta != "" {
			ui.Text(c, meta).FontSize(11.5).FontWeight(520).SingleLine().
				TextColor(songWhite.Alpha(0.4))
		}
	})
}

func (a *app) songStageControls(c *ui.Context, p songColors, track Track, padX, dockH float32) {
	dock := ui.Box(c).Absolute().Left(0).Right(0).Bottom(0).Height(dockH).
		Padding(0, padX, 16, padX)
	dock.Draw(func(pt *ui.Painter, r ui.Rect) {
		pt.FillGradient(r, ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
			{Color: songInk.Alpha(0), At: 0},
			{Color: songInk.Alpha(0.52), At: 0.3},
			{Color: songInk.Alpha(0.88), At: 1},
		}}, 0)
	})
	dock.Children(func() {
		a.songProgress(c, p, track)
		ui.Box(c).Height(5)
		ui.Row(c).FillWidth().Grow(1).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Width(200).AlignItems(ui.Center).Gap(8).Children(func() {
				a.songVolume(c)
			})
			ui.Box(c).Grow(1).Center().Children(func() {
				a.songTransport(c, track)
			})
			ui.Row(c).Width(200).Gap(2).Justify(ui.End).AlignItems(ui.Center).Children(func() {
				a.songPlaylistAddButton(c, track)
				a.songFavoriteButton(c, track)
			})
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

func (a *app) drawSongStage(pt *ui.Painter, r ui.Rect, p songColors, track Track, reduceMotion bool, cover *ui.Bitmap) {
	now := pt.Now()
	pos := time.Duration(0)
	same := a.pl != nil && a.pl.hasCurrent && a.pl.current.ID == track.ID
	playing := same && a.pl.playing
	if same {
		pos = a.pl.position()
	}
	if playing && !reduceMotion {
		pt.AnimationFrame()
	}
	useEnvelope := a.envelope != nil && same
	sample := a.stageSampleAt(pos, now, track)
	pulse := float32(0)
	if playing && !reduceMotion {
		pulse = stageBeatPulse(a, pos, useEnvelope)
	}
	timeValue := pos.Seconds()
	if reduceMotion {
		timeValue = 0
	}
	pt.FillGradient(r, ui.LinearGradient{Angle: 145, Oklab: true, Stops: []ui.GradientStop{
		{Color: p.tint.Alpha(0.16), At: 0},
		{Color: songInk.Alpha(0), At: 0.68},
		{Color: ui.RGBA(255, 255, 255, 0.018), At: 1},
	}}, 0)
	if pt.Accelerated() {
		drawStageShader(pt, r, p.tint, cover, timeValue, sample, pulse)
	} else {
		drawStageVinyl(pt, r, p.tint, cover, timeValue, sample, pulse)
	}
}

func (a *app) artBitmap(rawURL string) *ui.Bitmap {
	if rawURL == "" || a.arts == nil {
		return nil
	}
	bitmap, _ := a.arts[rawURL].(*ui.Bitmap)
	return bitmap
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

func drawStageVinyl(pt *ui.Painter, r ui.Rect, tint ui.Color, cover *ui.Bitmap, timeValue float64, sample waveSample, pulse float32) {
	g := stageDiscGeometryFor(r)
	rad := g.rad
	cx, cy := g.cx, g.cy
	diameter := rad * 2
	disc := ui.Rect{X: cx - rad, Y: cy - rad, W: diameter, H: diameter}

	halo := rad * 1.12
	pt.FillRadialGradient(ui.Rect{X: cx - halo, Y: cy - halo + rad*0.05, W: halo * 2, H: halo * 2}, ui.RadialGradient{
		Radius: halo,
		Stops: []ui.GradientStop{
			{Color: ui.RGBA(0, 0, 0, 0.36), At: 0.82},
			{Color: ui.RGBA(0, 0, 0, 0.16), At: 0.94},
			{Color: ui.RGBA(0, 0, 0, 0), At: 1},
		},
	}, 0)
	pt.Fill(disc, ui.RGBA(7, 7, 9, 255), rad)
	pt.Clip(disc, rad, func() {
		pt.FillRadialGradient(disc, ui.RadialGradient{
			Center: ui.Point{X: 0.46, Y: 0.42},
			Radius: rad * 1.35,
			Stops: []ui.GradientStop{
				{Color: ui.RGBA(27, 25, 29, 255), At: 0},
				{Color: ui.RGBA(16, 15, 18, 255), At: 0.62},
				{Color: ui.RGBA(9, 8, 10, 255), At: 1},
			},
		}, 0)

		minor := &ui.Path{}
		for i := 0; i < 84; i++ {
			rr := rad * (0.478 + 0.492*float32(i)/83)
			minor.Circle(cx, cy, rr)
		}
		pt.StrokePath(minor, 1, songWhite.Alpha(0.014+0.02*sample.Amp))

		sheen := 0.55 + 0.55*sample.Amp + 0.25*pulse
		pt.FillGradient(ui.Rect{X: cx - rad*1.25, Y: cy - rad*1.25, W: rad * 2.5, H: rad * 2.5}, ui.LinearGradient{Angle: 128, Stops: []ui.GradientStop{
			{Color: songWhite.Alpha(0), At: 0.16},
			{Color: songWhite.Alpha(0.050 * sheen), At: 0.33},
			{Color: songWhite.Alpha(0.075 * sheen), At: 0.40},
			{Color: songWhite.Alpha(0), At: 0.46},
			{Color: songWhite.Alpha(0), At: 0.54},
			{Color: songWhite.Alpha(0.058 * sheen), At: 0.62},
			{Color: songWhite.Alpha(0.032 * sheen), At: 0.69},
			{Color: songWhite.Alpha(0), At: 0.82},
		}}, 0)

		pt.FillRadialGradient(disc, ui.RadialGradient{
			Center: ui.Point{X: 0.5, Y: 0.5},
			Radius: rad,
			Stops: []ui.GradientStop{
				{Color: tint.Alpha(0), At: 0.60},
				{Color: tint.Alpha(0.05 + 0.05*sample.Amp + 0.03*pulse), At: 0.94},
				{Color: tint.Alpha(0.018), At: 1},
			},
		}, 0)
	})

	rim := (&ui.Path{}).Circle(cx, cy, rad-0.7)
	pt.StrokePathGradient(rim, 1.3, ui.LinearGradient{Angle: 130, Stops: []ui.GradientStop{
		{Color: songWhite.Alpha(0.42), At: 0},
		{Color: songWhite.Alpha(0.13), At: 0.38},
		{Color: songWhite.Alpha(0.10), At: 0.68},
		{Color: tint.Mix(songWhite, 0.55).Alpha(0.30), At: 1},
	}})

	labelR := rad * 0.46
	label := ui.Rect{X: cx - labelR, Y: cy - labelR, W: labelR * 2, H: labelR * 2}
	paper := ui.RGBA(224, 208, 184, 255).Mix(tint, 0.35)
	if cover != nil {
		degrees := float32(math.Mod(timeValue*30, 360))
		pt.Clip(label, labelR, func() {
			pt.Transform(ui.Rotation(degrees), cx, cy, func() {
				side := labelR * 2 * 1.45
				pt.Image(cover, ui.Rect{X: cx - side/2, Y: cy - side/2, W: side, H: side}, ui.Cover)
			})
		})
	} else {
		pt.FillPath((&ui.Path{}).Circle(cx, cy, labelR), paper)
		pt.Clip(label, labelR, func() {
			pt.FillRadialGradient(label, ui.RadialGradient{
				Center: ui.Point{X: 0.38, Y: 0.34},
				Radius: labelR * 1.4,
				Stops: []ui.GradientStop{
					{Color: songWhite.Alpha(0.10), At: 0},
					{Color: songWhite.Alpha(0), At: 1},
				},
			}, 0)
		})
	}
	pt.StrokePath((&ui.Path{}).Circle(cx, cy, labelR+1.5), 3, ui.RGBA(0, 0, 0, 0.42))
	pt.StrokePath((&ui.Path{}).Circle(cx, cy, labelR), 1.1, ui.RGBA(6, 6, 8, 0.9))
	pt.StrokePathGradient((&ui.Path{}).Circle(cx, cy, labelR-0.6), 1.2, ui.LinearGradient{Angle: 135, Stops: []ui.GradientStop{
		{Color: songWhite.Alpha(0.22), At: 0},
		{Color: songWhite.Alpha(0.03), At: 0.5},
		{Color: ui.RGBA(0, 0, 0, 0.30), At: 1},
	}})

	holeR := rad * 0.0130
	pt.FillPath((&ui.Path{}).Circle(cx, cy, holeR), ui.RGBA(5, 5, 7, 255))
	pt.StrokePath((&ui.Path{}).Circle(cx, cy, holeR), 1.2, songWhite.Alpha(0.28))

	pt.Clip(disc, rad, func() {
		gloss := ui.Rect{X: cx - rad*0.9, Y: cy - rad*1.05, W: rad * 1.8, H: rad * 1.8}
		pt.FillRadialGradient(gloss, ui.RadialGradient{
			Center: ui.Point{X: 0.44, Y: 0.40},
			Radius: rad * 0.9,
			Stops: []ui.GradientStop{
				{Color: songWhite.Alpha(0), At: 0},
				{Color: songWhite.Alpha(0.018 + 0.024*sample.Amp), At: 0.58},
				{Color: songWhite.Alpha(0), At: 1},
			},
		}, 0)
	})
}

type stageArmFrame struct {
	pivot          ui.Point
	length         float32
	ux, uy, px, py float32
	lift           float32
}

func newStageArmFrame(g stageDiscGeometry, angle float32) stageArmFrame {
	theta := float64(angle) * math.Pi / 180
	ux := float32(math.Cos(theta))
	uy := float32(math.Sin(theta))
	span := stageArmPlayAngle - stageArmRestAngle
	lift := float32(0)
	if span > 0 {
		lift = min(1, max(0, (stageArmPlayAngle-angle)/span))
	}
	return stageArmFrame{
		pivot:  ui.Point{X: g.cx + g.rad*0.675, Y: g.cy - g.rad*0.804},
		length: g.rad * 1.18,
		ux:     ux,
		uy:     uy,
		px:     -uy,
		py:     ux,
		lift:   lift,
	}
}

func (f stageArmFrame) at(along, across float32) ui.Point {
	return ui.Point{
		X: f.pivot.X + f.ux*along + f.px*across,
		Y: f.pivot.Y + f.uy*along + f.py*across,
	}
}

func drawStageTonearm(pt *ui.Painter, r ui.Rect, tint ui.Color, angle float32, playing bool) {
	g := stageDiscGeometryFor(r)
	rad := g.rad
	arm := newStageArmFrame(g, angle)
	arm.length *= 1 - 0.004*arm.lift
	end := arm.length - rad*0.075
	at := arm.at

	line := func(a, b ui.Point) *ui.Path {
		return (&ui.Path{}).MoveTo(a.X, a.Y).LineTo(b.X, b.Y)
	}

	tube := &ui.Path{}
	tube.MoveTo(at(rad*0.02, 0).X, at(rad*0.02, 0).Y).
		CubeTo(at(end*0.38, -rad*0.008).X, at(end*0.38, -rad*0.008).Y,
			at(end*0.74, rad*0.018).X, at(end*0.74, rad*0.018).Y,
			at(end, rad*0.004).X, at(end, rad*0.004).Y)

	band := func(a0, a1, c0, c1 float32) *ui.Path {
		p := &ui.Path{}
		p0, p1 := at(a0, c0), at(a1, c0)
		p2, p3 := at(a1, c1), at(a0, c1)
		p.MoveTo(p0.X, p0.Y).LineTo(p1.X, p1.Y).LineTo(p2.X, p2.Y).LineTo(p3.X, p3.Y).Close()
		return p
	}

	pt.Transform(ui.Translation(rad*0.012*(1+arm.lift*1.6), rad*0.022*(1+arm.lift*1.6)), 0, 0, func() {
		pt.FillPath(band(-rad*0.30, -rad*0.14, -rad*0.033, rad*0.033), ui.RGBA(0, 0, 0, 0.20))
		pt.StrokePath(tube, rad*0.036, ui.RGBA(0, 0, 0, 0.20))
		pt.StrokePath(tube, rad*0.024, ui.RGBA(0, 0, 0, 0.24))
	})

	pt.FillPath(band(-rad*0.22, rad*0.02, -rad*0.008, rad*0.008), ui.RGBA(30, 27, 31, 255))
	pt.FillPath(band(-rad*0.20, rad*0.00, -rad*0.007, -rad*0.003), songWhite.Alpha(0.22))

	pt.FillPath(band(-rad*0.30, -rad*0.145, -rad*0.031, rad*0.031), ui.RGBA(34, 31, 36, 255))
	pt.FillPath(band(-rad*0.296, -rad*0.289, -rad*0.033, rad*0.033), ui.RGBA(118, 109, 104, 255))
	pt.FillPath(band(-rad*0.184, -rad*0.177, -rad*0.031, rad*0.031), ui.RGBA(76, 70, 72, 255))
	pt.FillPath(band(-rad*0.282, -rad*0.156, -rad*0.024, -rad*0.014), songWhite.Alpha(0.18))
	pt.FillPath(band(-rad*0.282, -rad*0.156, rad*0.018, rad*0.026), tint.Mix(songWhite, 0.4).Alpha(0.10))

	pt.StrokePath(tube, rad*0.021, ui.RGBA(26, 23, 28, 255))
	pt.StrokePath(tube, rad*0.0155, ui.RGBA(84, 78, 82, 255))

	gleam := &ui.Path{}
	gleam.MoveTo(at(rad*0.03, -rad*0.0052).X, at(rad*0.03, -rad*0.0052).Y).
		CubeTo(at(end*0.38, -rad*0.0135).X, at(end*0.38, -rad*0.0135).Y,
			at(end*0.74, rad*0.0128).X, at(end*0.74, rad*0.0128).Y,
			at(end*0.99, -rad*0.0015).X, at(end*0.99, -rad*0.0015).Y)
	pt.StrokePath(gleam, rad*0.0055, songWhite.Alpha(0.5))

	under := &ui.Path{}
	under.MoveTo(at(rad*0.03, rad*0.0062).X, at(rad*0.03, rad*0.0062).Y).
		CubeTo(at(end*0.38, rad*0.0022).X, at(end*0.38, rad*0.0022).Y,
			at(end*0.74, rad*0.0262).X, at(end*0.74, rad*0.0262).Y,
			at(end*0.99, rad*0.0135).X, at(end*0.99, rad*0.0135).Y)
	pt.StrokePath(under, rad*0.0045, tint.Mix(songWhite, 0.5).Alpha(0.16))

	head := &ui.Path{}
	head.MoveTo(at(end+rad*0.058, -rad*0.036).X, at(end+rad*0.058, -rad*0.036).Y).
		LineTo(at(end+rad*0.068, rad*0.030).X, at(end+rad*0.068, rad*0.030).Y).
		LineTo(at(end-rad*0.092, rad*0.040).X, at(end-rad*0.092, rad*0.040).Y).
		LineTo(at(end-rad*0.098, -rad*0.027).X, at(end-rad*0.098, -rad*0.027).Y).
		Close()
	pt.FillPathGradient(head, ui.LinearGradient{Angle: angle + 90, Stops: []ui.GradientStop{
		{Color: ui.RGBA(206, 199, 190, 255), At: 0},
		{Color: ui.RGBA(136, 128, 124, 255), At: 0.46},
		{Color: ui.RGBA(58, 53, 57, 255), At: 1},
	}})
	pt.StrokePath(head, max(float32(1), rad*0.0042), ui.RGBA(15, 14, 17, 0.72))
	pt.StrokePath(line(at(end-rad*0.078, -rad*0.016), at(end+rad*0.046, -rad*0.025)),
		max(float32(1), rad*0.0042), songWhite.Alpha(0.55))

	cart := &ui.Path{}
	cart.MoveTo(at(end+rad*0.030, rad*0.006).X, at(end+rad*0.030, rad*0.006).Y).
		LineTo(at(end+rad*0.036, rad*0.040).X, at(end+rad*0.036, rad*0.040).Y).
		LineTo(at(end-rad*0.052, rad*0.046).X, at(end-rad*0.052, rad*0.046).Y).
		LineTo(at(end-rad*0.055, rad*0.010).X, at(end-rad*0.055, rad*0.010).Y).
		Close()
	pt.FillPath(cart, ui.RGBA(28, 25, 29, 235))
	pt.StrokePath(cart, max(float32(1), rad*0.0035), ui.RGBA(150, 142, 136, 0.42))

	stylusTip := at(end+rad*0.014, rad*0.030)
	pt.StrokePath(line(at(end-rad*0.006, rad*0.032), stylusTip),
		max(float32(1.1), rad*0.0052), ui.RGBA(216, 208, 196, 0.72))
	pt.FillPath((&ui.Path{}).Circle(stylusTip.X, stylusTip.Y, max(float32(1.1), rad*0.0052)),
		ui.RGBA(232, 226, 214, 0.9))

	if playing {
		glow := rad * 0.075
		pt.FillRadialGradient(ui.Rect{X: stylusTip.X - glow, Y: stylusTip.Y - glow, W: glow * 2, H: glow * 2}, ui.RadialGradient{
			Stops: []ui.GradientStop{
				{Color: tint.Mix(songWhite, 0.5).Alpha(0.20), At: 0},
				{Color: tint.Alpha(0), At: 1},
			},
			Radius: glow,
		}, 999)
	}

	pivot := arm.pivot
	pt.FillPath((&ui.Path{}).Circle(pivot.X+rad*0.016, pivot.Y+rad*0.032, rad*0.070),
		ui.RGBA(0, 0, 0, 0.30))
	pt.FillPathConicGradient((&ui.Path{}).Circle(pivot.X, pivot.Y, rad*0.058), ui.ConicGradient{Angle: 135, Stops: []ui.GradientStop{
		{Color: ui.RGBA(164, 155, 148, 255), At: 0},
		{Color: ui.RGBA(58, 53, 57, 255), At: 0.30},
		{Color: ui.RGBA(120, 112, 109, 255), At: 0.60},
		{Color: ui.RGBA(20, 18, 22, 255), At: 1},
	}})
	pt.StrokePath((&ui.Path{}).Circle(pivot.X, pivot.Y, rad*0.058),
		max(float32(1), rad*0.0038), songWhite.Alpha(0.20))
	pt.FillPathConicGradient((&ui.Path{}).Circle(pivot.X, pivot.Y, rad*0.034), ui.ConicGradient{Angle: 220, Stops: []ui.GradientStop{
		{Color: ui.RGBA(102, 96, 95, 255), At: 0},
		{Color: ui.RGBA(28, 25, 29, 255), At: 0.42},
		{Color: ui.RGBA(74, 69, 72, 255), At: 0.72},
		{Color: ui.RGBA(15, 14, 17, 255), At: 1},
	}})
	pt.FillPath((&ui.Path{}).Circle(pivot.X, pivot.Y, rad*0.012),
		ui.RGBA(9, 8, 11, 255))

}
