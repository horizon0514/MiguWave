package app

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"freewave/internal/migu"
)

func TestCollectMiguStorageValue(t *testing.T) {
	var candidates []miguBrowserToken
	add := func(value string, loginToken bool) {
		value = migu.NormalizeToken(value)
		if value == "" {
			return
		}
		candidates = append(candidates, miguBrowserToken{value: value, loginToken: loginToken})
	}

	collectMiguStorageValue("mg_auth", `{"pacmtoken":"direct-token","token":"login-token"}`, add)
	collectMiguStorageValue("mg_auth_pacmtoken", `"quoted-token"`, add)
	collectMiguStorageValue("unrelated", `https://music.migu.cn/v3/#/login?token=url-token`, add)

	want := []miguBrowserToken{
		{value: "direct-token"},
		{value: "login-token", loginToken: true},
		{value: "quoted-token"},
		{value: "url-token", loginToken: true},
	}
	if len(candidates) != len(want) {
		t.Fatalf("candidates = %+v, want %+v", candidates, want)
	}
	for i := range want {
		if candidates[i] != want[i] {
			t.Errorf("candidate %d = %+v, want %+v", i, candidates[i], want[i])
		}
	}
}

func TestMiguAuthGenerationCancelsOldFlow(t *testing.T) {
	a := testApp()
	ctx, gen := a.beginMiguAuth()
	_, nextGen := a.beginMiguAuth()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("starting a new auth flow did not cancel the previous one")
	}
	if nextGen <= gen {
		t.Fatalf("generation did not advance: %d -> %d", gen, nextGen)
	}
	if a.finishMiguAuth(gen) {
		t.Fatal("old auth generation was accepted")
	}
	if !a.finishMiguAuth(nextGen) {
		t.Fatal("current auth generation was rejected")
	}
}

func TestAuthModeTokenStopsPendingLogin(t *testing.T) {
	a := testApp()
	a.authMode = authModeQR
	a.authPhase = miguAuthQRWaiting
	ctx, _ := a.beginMiguAuth()
	a.setAuthMode(authModeToken)
	select {
	case <-ctx.Done():
	default:
		t.Fatal("switching to token mode did not stop the login watcher")
	}
	if a.authPhase != miguAuthIdle {
		t.Fatalf("phase = %v, want idle", a.authPhase)
	}
	if token := loginCookieToken(nil); token != "" {
		t.Fatalf("nil session token = %q", token)
	}
}

func TestAccountModalOffersLoginModes(t *testing.T) {
	a := testApp()
	a.accountOpen = true
	tt := ui.NewTester(a.view, 1240, 820)
	for _, want := range []string{
		"咪咕音乐授权",
		"应用内登录",
		"手机扫码",
		"令牌导入",
		"咪咕官方登录页会显示在这里",
	} {
		if !tt.HasText(want) {
			t.Errorf("account modal does not show %q", want)
		}
	}

	a.authMode = authModeQR
	a.authPhase = miguAuthQRWaiting
	tt = ui.NewTester(a.view, 1240, 820)
	for _, want := range []string{"请用咪咕音乐扫码", "刷新二维码", "从剪贴板导入"} {
		if !tt.HasText(want) {
			t.Errorf("QR mode does not show %q", want)
		}
	}

	a.authMode = authModeToken
	a.authPhase = miguAuthIdle
	tt = ui.NewTester(a.view, 1240, 820)
	for _, want := range []string{"手动导入 pacmtoken", "验证并保存", "从剪贴板导入"} {
		if !tt.HasText(want) {
			t.Errorf("token mode does not show %q", want)
		}
	}
}
