package main

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
	"time"

	"github.com/hajimehoshi/go-mp3"
)

const envelopeRate = 30.0

type waveSample struct {
	Low  float32
	Mid  float32
	High float32
	Amp  float32
}

type audioEnvelope struct {
	Rate    float64
	Samples []waveSample
}

// buildAudioEnvelope decodes an MP3 file without touching the player's
// stream. It returns coarse low/mid/high energy at about 30 Hz.
func buildAudioEnvelope(path string) (*audioEnvelope, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec, err := mp3.NewDecoder(f)
	if err != nil {
		return nil, err
	}
	rate := dec.SampleRate()
	if rate <= 0 {
		return nil, errors.New("invalid sample rate")
	}
	bucketFrames := max(1, rate/int(envelopeRate))
	envelope := &audioEnvelope{Rate: envelopeRate}
	var low, mid float64
	var lowSum, midSum, highSum, ampSum float64
	var frames int
	buf := make([]byte, 32<<10)
	var carry []byte
	for {
		n, readErr := dec.Read(buf)
		if n > 0 {
			data := append(carry, buf[:n]...)
			whole := len(data) / 4 * 4
			carry = append(carry[:0], data[whole:]...)
			for i := 0; i < whole; i += 4 {
				left := int16(binary.LittleEndian.Uint16(data[i:]))
				right := int16(binary.LittleEndian.Uint16(data[i+2:]))
				sample := (float64(left) + float64(right)) / 65536
				low += (sample - low) * 0.045
				mid += (sample - mid) * 0.35
				lowBand := low
				midBand := mid - low
				highBand := sample - mid
				lowSum += lowBand * lowBand
				midSum += midBand * midBand
				highSum += highBand * highBand
				ampSum += sample * sample
				frames++
				if frames >= bucketFrames {
					envelope.Samples = append(envelope.Samples, normalizedSample(
						lowSum, midSum, highSum, ampSum, frames))
					lowSum, midSum, highSum, ampSum, frames = 0, 0, 0, 0, 0
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
			break
		}
	}
	if frames > 0 {
		envelope.Samples = append(envelope.Samples, normalizedSample(
			lowSum, midSum, highSum, ampSum, frames))
	}
	if len(envelope.Samples) < 2 {
		return nil, errors.New("audio is too short")
	}
	normalizeEnvelope(envelope.Samples)
	return envelope, nil
}

func normalizedSample(low, mid, high, amp float64, frames int) waveSample {
	n := float64(max(1, frames))
	return waveSample{
		Low:  float32(math.Sqrt(low / n)),
		Mid:  float32(math.Sqrt(mid / n)),
		High: float32(math.Sqrt(high / n)),
		Amp:  float32(math.Sqrt(amp / n)),
	}
}

func normalizeEnvelope(samples []waveSample) {
	var peak float32
	for _, sample := range samples {
		peak = max(peak, sample.Amp)
	}
	if peak <= 1e-5 {
		return
	}
	compress := func(v float32) float32 {
		v = min(1, max(0, v/peak))
		return float32(math.Pow(float64(v), 0.72))
	}
	for i := range samples {
		samples[i].Low = compress(samples[i].Low)
		samples[i].Mid = compress(samples[i].Mid)
		samples[i].High = compress(samples[i].High)
		samples[i].Amp = compress(samples[i].Amp)
	}
}

func (e *audioEnvelope) at(pos time.Duration) (low, mid, high, amp float32) {
	if e == nil || len(e.Samples) == 0 {
		return 0, 0, 0, 0
	}
	x := pos.Seconds() * e.Rate
	if x <= 0 {
		s := e.Samples[0]
		return s.Low, s.Mid, s.High, s.Amp
	}
	i := int(x)
	if i >= len(e.Samples)-1 {
		s := e.Samples[len(e.Samples)-1]
		return s.Low, s.Mid, s.High, s.Amp
	}
	f := float32(x - float64(i))
	a, b := e.Samples[i], e.Samples[i+1]
	return a.Low + (b.Low-a.Low)*f,
		a.Mid + (b.Mid-a.Mid)*f,
		a.High + (b.High-a.High)*f,
		a.Amp + (b.Amp-a.Amp)*f
}

func (a *app) resetStageAudio(trackID string) {
	a.envelope = nil
	a.envelopeTrack = trackID
}

func (a *app) loadEnvelope(gen int, trackID, path string) {
	envelope, err := buildAudioEnvelope(path)
	if err != nil {
		return
	}
	a.post(func() {
		if a.pl != nil && a.pl.gen == gen && a.envelopeTrack == trackID {
			a.envelope = envelope
			a.refresh()
		}
	})
}
