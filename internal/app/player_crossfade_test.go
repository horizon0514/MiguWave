package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/audio"
)

func TestCrossfadeRampsBetweenVoices(t *testing.T) {
	samples := make([]float32, 4800*2)
	old := audio.NewSound(48000, 2, samples).NewPlayer()
	incoming := audio.NewSound(48000, 2, samples).NewPlayer()

	a := testApp()
	a.pl.volume = 0.8
	a.pl.out = incoming
	a.pl.addVoice(old, 1)
	a.pl.addVoice(incoming, 0)
	a.pl.crossfadeTo(incoming)

	deadline := time.Now().Add(crossfadeDuration + time.Second)
	for time.Now().Before(deadline) {
		if incoming.Volume() >= 0.799 && old.Volume() <= 0.001 {
			a.pl.mixMu.Lock()
			voices := append([]*audioVoice(nil), a.pl.voices...)
			a.pl.mixMu.Unlock()
			if len(voices) != 1 || voices[0].out != incoming {
				t.Fatalf("crossfade left %d voices", len(voices))
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("crossfade did not finish: old=%v incoming=%v", old.Volume(), incoming.Volume())
}

func TestAutoFadeWaitsForRepeatOne(t *testing.T) {
	samples := make([]float32, 4800*2)
	out := audio.NewSound(48000, 2, samples).NewPlayer()
	a := testApp()
	a.pl.out = out
	a.pl.queue = sampleMiguTracks()
	a.pl.at = 0
	a.pl.current = a.pl.queue[0]
	a.pl.duration = time.Minute
	a.pl.playing = true
	a.pl.repeat = repeatOne

	a.pl.scheduleAutoFade()
	if a.pl.autoFadeTimer != nil {
		t.Fatal("repeat-one track should not schedule an automatic crossfade")
	}
}
