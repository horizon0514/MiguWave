package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestAccountModalShowsProfile(t *testing.T) {
	for _, phase := range []miguAuthPhase{miguAuthConnected, miguAuthChecking} {
		a := testApp()
		a.accountOpen = true
		a.miguToken = "profile-test-token"
		a.authPhase = phase
		a.miguProfile = &miguProfile{
			UserID:    "91215317963",
			Nickname:  "棒棒的福斯特",
			Phone:     "183****9595",
			LevelName: "lv3",
			MedalName: "咪咕练习生",
		}

		tt := ui.NewTester(a.view, 1240, 820)
		for _, want := range []string{
			"棒棒的福斯特",
			"咪咕账号已连接",
			"183****9595  ·  UID 91215317963",
			"lv3",
			"勋章 · 咪咕练习生",
		} {
			if !tt.HasText(want) {
				t.Errorf("phase %d: account modal does not show %q", phase, want)
			}
		}
	}
}
