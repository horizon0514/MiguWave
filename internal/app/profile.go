package app

import (
	"context"
	"strings"
	"time"

	"freewave/internal/migu"
)

func miguProfileInitial(name string) string {
	for _, r := range strings.TrimSpace(name) {
		return string(r)
	}
	return "咪"
}

func (a *app) loadMiguProfile(token string) {
	token = migu.NormalizeToken(token)
	if token == "" {
		a.miguProfile = nil
		a.miguProfileErr = ""
		a.miguProfileLoading = false
		return
	}
	a.miguProfileLoading = true
	a.miguProfileErr = ""
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		profile, err := migu.FetchProfile(ctx, token)
		a.post(func() {
			if migu.NormalizeToken(a.miguToken) != token {
				return
			}
			a.miguProfileLoading = false
			if err != nil {
				a.miguProfileErr = friendlyError(err)
				a.refresh()
				return
			}
			a.miguProfile = &profile
			a.miguProfileErr = ""
			if profile.AvatarURL != "" {
				a.requestArt(profile.AvatarURL)
			}
			a.refresh()
		})
	}()
}
