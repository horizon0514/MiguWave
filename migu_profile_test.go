package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestFetchMiguProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("pacmtoken"); got != "profile-test-token" {
			t.Errorf("pacmtoken = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "000000",
			"info": "success",
			"userInfoItem": map[string]any{
				"userId":   "91215317963",
				"nickName": "棒棒的福斯特",
				"bigIcon":  "https://example.com/avatar.jpg",
				"msisdn":   "183****9595",
				"userLevelInfo": map[string]any{
					"level":     "3",
					"levelName": "lv3",
				},
				"userMedalInfoList": []map[string]any{{"medalName": "咪咕练习生"}},
			},
		})
	}))
	defer server.Close()

	old := miguProfileURL
	defer func() { miguProfileURL = old }()
	miguProfileURL = server.URL

	profile, err := fetchMiguProfile(context.Background(), "profile-test-token")
	if err != nil {
		t.Fatal(err)
	}
	if profile.Nickname != "棒棒的福斯特" ||
		profile.UserID != "91215317963" ||
		profile.Phone != "183****9595" ||
		profile.LevelName != "lv3" ||
		profile.MedalName != "咪咕练习生" ||
		profile.AvatarURL != "https://example.com/avatar.jpg" {
		t.Fatalf("profile = %+v", profile)
	}
}

func TestFetchMiguProfileError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "290001", "info": "登录状态已失效"})
	}))
	defer server.Close()

	old := miguProfileURL
	defer func() { miguProfileURL = old }()
	miguProfileURL = server.URL

	_, err := fetchMiguProfile(context.Background(), "bad-token")
	if err == nil || !strings.Contains(err.Error(), "登录状态已失效") {
		t.Fatalf("err = %v", err)
	}
}

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
