package app

import (
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"freewave/internal/migu"
)

// Run starts the desktop application and blocks until it quits.
func Run() error {
	mygo.App.SetName("MiguWave")
	if !mygo.App.RequestSingleInstanceLock() {
		return nil
	}

	go func() {
		_, _ = cleanAudioCache(audioCacheDir(), time.Now(), audioCacheMaxBytes)
	}()

	saved := loadSaved()
	mygo.Theme.SetSource(saved.Theme)
	a := &app{
		volume:         saved.Volume,
		theme:          saved.Theme,
		opaqueWindow:   saved.OpaqueWindow,
		closeAction:    saved.CloseAction,
		favorites:      saved.Favorites,
		recent:         saved.Recent,
		savedPlaylists: saved.SavedPlaylists,
		localPlaylists: saved.LocalPlaylists,
		page:           pageHome,
		miguToken:      migu.LoadToken(),
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
	if saved.Playback != nil {
		a.pl.restore(*saved.Playback)
	}

	mygo.App.OnSecondInstance(func([]string, string) {
		a.showMainWindow()
	})
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		a.handleAppActivate(hasVisibleWindows)
	})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		a.quitting = true
		a.save()
	})
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) {
		a.stopMenuBarAnimation()
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
			Vibrancy:      mygo.VibrancyUnderWindow,
			StateKey:      "main",
			Content:       ui.View(a.view),
			Hidden:        openAtLogin,
		})
		if a.opaqueWindow {
			win.SetVibrancy(mygo.VibrancyNone)
		}
		a.attachWindow(win)
		a.start()
		a.setupMenuBar()
		if openAtLogin {
			a.setDockVisible(false)
		} else {
			win.Show()
		}
	})
	return mygo.App.Run()
}
