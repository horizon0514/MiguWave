package app

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
)

const stageShaderSource = `package main

var Time float
var Amp float
var Low float
var Mid float
var High float
var Pulse float
var Tint vec3
var Glow vec3
var HasCover float

func spinQ(q vec2, a float) vec2 {
	s := sin(a)
	c := cos(a)
	return vec2(q.x*c-q.y*s, q.x*s+q.y*c)
}

func coverAt(uv vec2) vec4 {
	size := image0Size()
	return image0Sample(uv*size)
}

func Fragment(dst vec4, pos vec2) vec4 {
	size := rectSize()
	center := vec2(size.x*0.5, size.y*0.405)
	rad := min(size.x*0.39, size.y*0.29)
	q := (pos-center) / rad
	r := length(q)
	aa := 1.30 / rad
	energy := clamp(Amp*0.60+Low*0.25+Mid*0.10+High*0.05, 0.0, 1.0)
	disc := 1.0 - smoothstep(1.0-aa, 1.0+aa, r)

	outside := max(0.0, r-1.0)
	shadow := exp(-(outside*outside)/0.006) * 0.48
	bloom := exp(-pow((r-1.05)/0.20, 2.0)) * (0.010+0.018*energy+0.020*Pulse)
	outRGB := mix(Tint, Glow, 0.25) * bloom
	outA := clamp(shadow+bloom*0.60, 0.0, 1.0)
	if r > 1.30 {
		return vec4(outRGB, outA)
	}

	col := vec3(0.013, 0.012, 0.014)
	col += Tint * (0.014 * (1.0-r))

	spin := Time * 0.52
	qg := spinQ(q, -spin)
	ang := atan2(qg.y, qg.x)
	warp := 0.0025 * sin(ang+spin*0.4)
	groove := pow(0.5+0.5*cos((r+warp)*96.0*6.2831853), 18.0)
	gmask := smoothstep(0.478, 0.505, r) * (1.0-smoothstep(0.972, 0.994, r))

	theta := atan2(q.y, q.x)
	cres := pow(abs(cos(theta-0.85)), 22.0)
	cresMask := smoothstep(0.36, 0.99, r) * (1.0-smoothstep(0.984, 1.0, r))
	lobe := cres * cresMask
	sheen := lobe * (0.065+0.060*energy+0.035*Pulse)
	col += mix(vec3(1.0, 1.0, 1.0), Glow, 0.14) * sheen

	col += mix(vec3(0.80, 0.78, 0.76), Glow, 0.22) * groove * gmask * sheen * 1.40
	col += mix(vec3(0.55, 0.54, 0.53), Glow, 0.30) * gmask * 0.005

	fine := pow(0.5+0.5*cos((r+warp*2.0)*260.0*6.2831853), 34.0)
	col += mix(vec3(0.85, 0.85, 0.85), Glow, 0.25) * fine * gmask * sheen * 0.55

	labelR := 0.460
	label := 1.0 - smoothstep(labelR-aa, labelR+aa, r)
	labelEdge := exp(-pow((r-labelR)/0.006, 2.0))
	col = col*(1.0-labelEdge*0.65) + vec3(0.010, 0.010, 0.012)*(labelEdge*0.65)

	if label > 0.0 {
		lq := spinQ(q, -spin) / labelR
		luv := (lq*0.5 + vec2(0.5) - vec2(0.5)) / 1.45 + vec2(0.5)
		paper := mix(vec3(0.80, 0.72, 0.61), Tint, 0.42)
		var lc vec4
		if HasCover > 0.5 {
			lc = coverAt(luv)
		} else {
			lc = vec4(paper, 1.0)
		}
		lcol := lc.rgb + (1.0-lc.a)*paper
		col = mix(col, lcol, label)
		labelShine := (1.0-smoothstep(0.0, labelR, length(lq+vec2(0.34, 0.42)))) * 0.050
		col += vec3(1.0, 1.0, 1.0) * labelShine * label
	}

	holeR := 0.0130
	hole := 1.0 - smoothstep(holeR-aa*1.4, holeR+aa*1.4, r)
	col = mix(col, vec3(0.004, 0.004, 0.006), hole)
	holeRim := exp(-pow((r-holeR)/0.0035, 2.0))
	col += vec3(0.55, 0.52, 0.49) * holeRim * 0.55

	bevelDir := max(0.0, dot(normalize(q+vec2(0.00001, 0.0)), normalize(vec2(-0.60, -0.80))))
	bevel := exp(-pow((r-labelR)/0.0025, 2.0))
	col += mix(vec3(1.0, 1.0, 1.0), Glow, 0.20) * bevel * bevelDir * 0.18

	gloss := exp(-pow((r-0.70)/0.50, 2.0)) * lobe * (0.010+0.012*energy)
	col += mix(vec3(1.0, 1.0, 1.0), Glow, 0.20) * gloss

	edge := exp(-pow((r-0.994)/0.0030, 2.0))
	rimDir := max(0.0, dot(normalize(q+vec2(0.00001, 0.0)), normalize(vec2(-0.55, -0.83))))
	col += mix(vec3(0.95, 0.93, 0.90), Glow, 0.25) * edge * (0.10+0.30*rimDir)

	rgb := outRGB + col*disc
	alpha := clamp(outA*(1.0-disc)+disc, 0.0, 1.0)
	return vec4(min(rgb, vec3(alpha)), alpha)
}
`

var (
	stageShader = shader.MustCompile(stageShaderSource)
	stageValues = stageShader.NewValues()

	stageTimeU  = stageShader.Uniform("Time")
	stageAmpU   = stageShader.Uniform("Amp")
	stageLowU   = stageShader.Uniform("Low")
	stageMidU   = stageShader.Uniform("Mid")
	stageHighU  = stageShader.Uniform("High")
	stageTintU  = stageShader.Uniform("Tint")
	stageGlowU  = stageShader.Uniform("Glow")
	stagePulseU = stageShader.Uniform("Pulse")
	stageHasArt = stageShader.Uniform("HasCover")
)

// drawStageShader paints the full-screen rhythm stage as a slowly turning
// vinyl record. The window falls back to drawStageVinyl without a GPU.
func drawStageShader(pt *ui.Painter, r ui.Rect, tint ui.Color, cover *ui.Bitmap, timeValue float64, sample waveSample, pulse float32) {
	hasCover := float32(0)
	if cover != nil {
		hasCover = 1
	}
	stageValues.
		Float(stageTimeU, float32(timeValue)).
		Float(stageAmpU, sample.Amp).
		Float(stageLowU, sample.Low).
		Float(stageMidU, sample.Mid).
		Float(stageHighU, sample.High).
		Color(stageTintU, tint).
		Color(stageGlowU, tint.Mix(songWhite, 0.42)).
		Float(stagePulseU, pulse).
		Float(stageHasArt, hasCover)
	stageShader.Draw(pt, r, shader.Options{
		Values: stageValues,
		Images: [4]*ui.Bitmap{cover},
		Radius: 0,
	})
}

// stageBeatPulse is how strongly the most recent beat still rings, from 0 to 1.
func stageBeatPulse(a *app, pos time.Duration, useEnvelope bool) float32 {
	if useEnvelope && a.envelope != nil && len(a.envelope.Samples) > 4 {
		envelope := a.envelope
		end := min(len(envelope.Samples)-2, int(pos.Seconds()*envelope.Rate))
		start := max(1, int((pos-1200*time.Millisecond).Seconds()*envelope.Rate))
		last := end + 1
		var pulse float32
		for i := end; i >= start; i-- {
			score := stageBeatScore(envelope.Samples, i)
			if score < 0.12 || score < stageBeatScore(envelope.Samples, i-1) || score < stageBeatScore(envelope.Samples, i+1) {
				continue
			}
			if last <= end && last-i < int(envelope.Rate*0.16) {
				continue
			}
			age := pos.Seconds() - float64(i)/envelope.Rate
			if age < 0 || age > 1.1 {
				continue
			}
			gain := min(1, max(0.2, score*4.2))
			pulse = max(pulse, gain*float32(math.Exp(-age*6.5)))
			last = i
		}
		return pulse
	}

	recent := math.Mod(pos.Seconds(), 0.72)
	if recent < 0 {
		recent += 0.72
	}
	return float32(math.Exp(-recent * 7))
}

func stageBeatScore(samples []waveSample, i int) float32 {
	if i <= 0 || i >= len(samples)-1 {
		return 0
	}
	level := func(sample waveSample) float32 {
		return sample.Low*0.68 + sample.Amp*0.25 + sample.Mid*0.07
	}
	current := level(samples[i])
	previous := level(samples[i-1])
	base := (level(samples[i-2]) + level(samples[i-3]) + level(samples[i-4])) / 3
	rise := max(0, current-previous)
	contrast := max(0, current-base)
	return rise*0.55 + contrast*0.80
}
