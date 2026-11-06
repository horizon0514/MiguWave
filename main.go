package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	mygo.App.SetName("MiguWave")
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}

	go func() {
		_, _ = cleanAudioCache(audioCacheDir(), time.Now(), audioCacheMaxBytes)
	}()

	saved := loadSaved()
	a := &app{
		volume:         saved.Volume,
		closeAction:    saved.CloseAction,
		favorites:      saved.Favorites,
		recent:         saved.Recent,
		savedPlaylists: saved.SavedPlaylists,
		localPlaylists: saved.LocalPlaylists,
		page:           pageHome,
		miguToken:      loadMiguToken(),
		arts:           map[string]ui.ImageSource{},
		artLoad:        map[string]bool{},
		artFail:        map[string]bool{},
		artTints:       map[string]ui.Color{},
		artBackdrops:   map[string]*ui.Bitmap{},
		songCache:      map[string]miguSongDetail{},
		coverHQ:        map[string]string{},
		coverHQLoad:    map[string]bool{},
		coverHQFail:    map[string]bool{},
	}
	a.pl = newPlayer(a)

	mygo.App.OnSecondInstance(func([]string, string) {
		a.showMainWindow()
	})
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			a.showMainWindow()
		}
	})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { a.quitting = true })
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) {
		if a.tray != nil {
			a.tray.Destroy()
		}
	})
	mygo.App.OnWindowAllClosed(func() {})

	mygo.App.WhenReady(func() {
		a.launchAtLogin = mygo.App.OpenAtLogin()
		openAtLogin := mygo.App.WasOpenedAtLogin()
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:         "MiguWave",
			Width:         1240,
			Height:        820,
			MinWidth:      1040,
			MinHeight:     680,
			TitleBarStyle: mygo.TitleBarHiddenInset,
			StateKey:      "main",
			Content:       ui.View(a.view),
			Hidden:        openAtLogin,
		})
		a.attachWindow(win)
		a.start()
		a.setupMenuBar()
		if openAtLogin {
			a.setDockVisible(false)
		} else {
			win.Show()
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
