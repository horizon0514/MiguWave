package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	go func() {
		_, _ = cleanAudioCache(audioCacheDir(), time.Now(), audioCacheMaxBytes)
	}()

	saved := loadSaved()
	a := &app{
		volume:    saved.Volume,
		favorites: saved.Favorites,
		recent:    saved.Recent,
		page:      pageHome,
		miguToken: loadMiguToken(),
		arts:      map[string]ui.ImageSource{},
		artLoad:   map[string]bool{},
		artFail:   map[string]bool{},
	}
	a.pl = newPlayer(a)

	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:         "MiguWave",
			Width:         1240,
			Height:        820,
			MinWidth:      1040,
			MinHeight:     680,
			TitleBarStyle: mygo.TitleBarHiddenInset,
			StateKey:      "main",
			Content:       ui.View(a.view),
		})
		a.win = win
		a.start()
		win.Show()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
