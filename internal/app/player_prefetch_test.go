package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPrefetchNextTrackIsReusedByPlayback(t *testing.T) {
	audio := bytes.Repeat([]byte("audio-data"), 512)
	var requests atomic.Int32
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseDownload := func() { releaseOnce.Do(func() { close(release) }) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			started <- struct{}{}
		}
		<-release
		_, _ = w.Write(audio)
	}))
	defer server.Close()
	defer releaseDownload()

	a := testApp()
	a.pl.cacheRootDir = t.TempDir()
	tracks := []Track{
		{ID: "test:current", Title: "当前", Source: "test"},
		{ID: "test:next", Title: "下一首", Source: "test", URL: server.URL + "/next.mp3"},
	}
	a.pl.queue = tracks
	a.pl.at = 0
	a.pl.current = tracks[0]
	a.pl.hasCurrent = true
	a.pl.schedulePrefetch()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("prefetch did not start")
	}

	type result struct {
		file *os.File
		err  error
	}
	done := make(chan result, 1)
	go func() {
		file, err := a.pl.downloadTrack(context.Background(), 1, tracks[1])
		done <- result{file: file, err: err}
	}()

	time.Sleep(100 * time.Millisecond)
	if got := requests.Load(); got != 1 {
		t.Fatalf("playback started a duplicate download: %d requests", got)
	}
	releaseDownload()

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatal(got.err)
		}
		defer got.file.Close()
		data, err := io.ReadAll(got.file)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, audio) {
			t.Fatalf("cached audio length = %d, want %d", len(data), len(audio))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("playback did not reuse the prefetched download")
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("next track was downloaded %d times", got)
	}
}

func TestNextIndexMatchesShufflePrefetch(t *testing.T) {
	a := testApp()
	a.pl.queue = sampleMiguTracks()
	a.pl.at = 0
	a.pl.current = a.pl.queue[0]
	a.pl.shuffle = true

	first := a.pl.nextIndex()
	second := a.pl.nextIndex()
	if first < 0 || first == a.pl.at {
		t.Fatalf("shuffle next index = %d", first)
	}
	if first != second {
		t.Fatalf("shuffle next index is unstable: %d then %d", first, second)
	}

	a.pl.shuffle = false
	a.pl.at = len(a.pl.queue) - 1
	a.pl.current = a.pl.queue[a.pl.at]
	if got := a.pl.nextIndex(); got != -1 {
		t.Fatalf("repeat-off end index = %d, want -1", got)
	}
	a.pl.repeat = repeatAll
	if got := a.pl.nextIndex(); got != 0 {
		t.Fatalf("repeat-all end index = %d, want 0", got)
	}
}
