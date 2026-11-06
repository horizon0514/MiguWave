package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestMiguDecodeQRImage(t *testing.T) {
	raw := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString([]byte("qr-bytes"))
	got, err := miguDecodeQRImage(raw)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "qr-bytes" {
		t.Fatalf("decoded QR = %q", got)
	}

	if _, err := miguDecodeQRImage("https://example.com/qr.png"); err == nil {
		t.Fatal("expected unsupported QR URL to fail")
	}
}

func TestMiguParseHiddenValues(t *testing.T) {
	body := []byte(`<input id="callbackURL" value="" /><input type="hidden" value="state-1" id="relayState">`)
	got := miguParseHiddenValues(body)
	if got["callbackURL"] != "" || got["relayState"] != "state-1" {
		t.Fatalf("hidden values = %#v", got)
	}
}

func TestMiguAuthStartQRAndPoll(t *testing.T) {
	var sawLoginCookie atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "passport", Value: "ready", Path: "/"})
			_, _ = w.Write([]byte(`<input id="callbackURL" value="" /><input id="relayState" value="" />`))
		case "/qr":
			if cookie, err := r.Cookie("passport"); err == nil && cookie.Value == "ready" {
				sawLoginCookie.Store(true)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": 2000,
				"result": map[string]any{
					"qrcUrl":        "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString([]byte("qr")),
					"qrc_sessionid": "session-123",
				},
			})
		case "/poll":
			if got := r.URL.Query().Get("qrc_sessionid"); got != "session-123" {
				t.Errorf("qrc_sessionid = %q", got)
			}
			_, _ = w.Write([]byte(`{"status":4073,"message":"已扫码","result":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	oldLogin, oldCreate, oldQuery := miguLoginPageURL, miguQRCreateURL, miguQRQueryURL
	defer func() {
		miguLoginPageURL, miguQRCreateURL, miguQRQueryURL = oldLogin, oldCreate, oldQuery
	}()
	miguLoginPageURL = server.URL + "/login"
	miguQRCreateURL = server.URL + "/qr"
	miguQRQueryURL = server.URL + "/poll"

	session, err := newMiguAuthSession()
	if err != nil {
		t.Fatal(err)
	}
	code, err := session.startQR(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !sawLoginCookie.Load() {
		t.Error("QR request did not reuse the login-page cookie jar")
	}
	if string(code.Image) != "qr" || code.SessionID != "session-123" {
		t.Fatalf("QR code = %+v", code)
	}
	poll, err := session.pollQR(context.Background(), code.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if poll.State != miguQRScanned {
		t.Fatalf("poll state = %v", poll.State)
	}
}

func TestMiguQRPollStates(t *testing.T) {
	var status int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  status,
			"message": "message",
			"result": map[string]any{
				"token":       "passport-token",
				"redirectURL": "https://passport.migu.cn/redirect",
			},
		})
	}))
	defer server.Close()
	oldQuery := miguQRQueryURL
	defer func() { miguQRQueryURL = oldQuery }()
	miguQRQueryURL = server.URL

	session, err := newMiguAuthSession()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		status int
		want   miguQRState
	}{
		{4074, miguQRWaiting},
		{4073, miguQRScanned},
		{4072, miguQRExpired},
		{2000, miguQRConfirmed},
	}
	for _, tc := range cases {
		status = tc.status
		got, err := session.pollQR(context.Background(), "session")
		if err != nil {
			t.Fatalf("status %d: %v", tc.status, err)
		}
		if got.State != tc.want {
			t.Errorf("status %d: state = %v, want %v", tc.status, got.State, tc.want)
		}
		if tc.status == 2000 && got.Token != "passport-token" {
			t.Errorf("confirmed token = %q", got.Token)
		}
	}
}

func TestMiguQRRedirectTarget(t *testing.T) {
	target, err := miguQRRedirectTarget(
		"https://passport.migu.cn/redirect?foo=1",
		"passport-token",
		"",
		"state with spaces",
	)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(target)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if parsed.Host != "passport.migu.cn" || query.Get("foo") != "1" {
		t.Fatalf("redirect target = %q", target)
	}
	if query.Get("token") != "passport-token" ||
		query.Get("relayState") != "state with spaces" ||
		query.Get("qrclogin") != "1" ||
		query.Get("logintype") != "QRCSSO" {
		t.Fatalf("redirect query = %#v", query)
	}
}

func TestMiguCompleteLoginExtractsPacmtoken(t *testing.T) {
	validate := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "199999", "info": "系统繁忙，请稍后重试"})
	}))
	defer validate.Close()
	oldValidate := miguTokenValidateURL
	defer func() { miguTokenValidateURL = oldValidate }()
	miguTokenValidateURL = validate.URL

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			if got := r.URL.Query().Get("token"); got != "passport-token" {
				t.Errorf("redirect token = %q", got)
			}
			http.SetCookie(w, &http.Cookie{Name: "pacmtoken", Value: "p-ac-token", Path: "/"})
			http.Redirect(w, r, "/done", http.StatusFound)
		case "/done":
			_, _ = w.Write([]byte("ok"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	session, err := newMiguAuthSession()
	if err != nil {
		t.Fatal(err)
	}
	token, err := session.completeLogin(context.Background(), server.URL+"/redirect?foo=1", "passport-token")
	if err != nil {
		t.Fatal(err)
	}
	if token != "p-ac-token" {
		t.Fatalf("pacmtoken = %q", token)
	}
}

func TestMiguExchangeLoginToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("type"); got != "2" {
			t.Errorf("type = %q", got)
		}
		if got := r.URL.Query().Get("sourceId"); got != "220029" {
			t.Errorf("sourceId = %q", got)
		}
		if r.Header.Get("deviceId") == "" {
			t.Error("deviceId header is missing")
		}
		switch r.URL.Query().Get("token") {
		case "header-token":
			w.Header().Set("pacmtoken", "p-ac-from-header")
			_, _ = w.Write([]byte(`{"code":"000000"}`))
		case "data-token":
			_, _ = w.Write([]byte(`{"code":"000000","data":{"pacmtoken":"p-ac-from-data"}}`))
		case "bad-token":
			w.Header().Set("pacmtoken", "must-not-be-used")
			_, _ = w.Write([]byte(`{"code":"199999","info":"系统繁忙，请稍后重试"}`))
		default:
			_, _ = w.Write([]byte(`{"code":"000000"}`))
		}
	}))
	defer server.Close()
	oldValidate := miguTokenValidateURL
	defer func() { miguTokenValidateURL = oldValidate }()
	miguTokenValidateURL = server.URL

	cases := []struct {
		token string
		want  string
		err   string
	}{
		{"header-token", "p-ac-from-header", ""},
		{"data-token", "p-ac-from-data", ""},
		{"bad-token", "", "系统繁忙"},
	}
	for _, tc := range cases {
		token, err := exchangeMiguLoginToken(context.Background(), nil, nil, tc.token)
		if tc.err != "" {
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Errorf("%s: err = %v, want %q", tc.token, err, tc.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", tc.token, err)
			continue
		}
		if token != tc.want {
			t.Errorf("%s: token = %q, want %q", tc.token, token, tc.want)
		}
	}
}

func TestMiguLoginTokenFromURL(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"https://passport.migu.cn/done?token=abc", "abc"},
		{"https://passport.migu.cn/done?utoken=def-token", "def-token"},
		{"https://passport.migu.cn/done?miguToken=ghi", "ghi"},
		{"https://passport.migu.cn/done#ptoken=jkl", "jkl"},
		{"https://music.migu.cn/v3/#/login?token=hash-token", "hash-token"},
		{"https://music.migu.cn/v3/#/login?foo=1&utoken=hash-utoken", "hash-utoken"},
		{"https://passport.migu.cn/done?foo=1", ""},
	}
	for _, tc := range cases {
		if got := miguLoginTokenFromURL(tc.raw); got != tc.want {
			t.Errorf("%s: token = %q, want %q", tc.raw, got, tc.want)
		}
	}
	raw := "https://passport.migu.cn/done?" + "pacm" + "token=direct-token"
	if got := miguTokenParamFromURL(raw, "pacmtoken"); got != "direct-token" {
		t.Errorf("pacmtoken param = %q", got)
	}
}

func TestCollectMiguStorageValue(t *testing.T) {
	var candidates []miguBrowserToken
	add := func(value string, loginToken bool) {
		value = normalizeMiguToken(value)
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

func TestMiguQRPollConfirmedWithoutRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": 2000,
			"result": map[string]any{"token": "passport-token"},
		})
	}))
	defer server.Close()
	oldQuery := miguQRQueryURL
	defer func() { miguQRQueryURL = oldQuery }()
	miguQRQueryURL = server.URL

	session, err := newMiguAuthSession()
	if err != nil {
		t.Fatal(err)
	}
	poll, err := session.pollQR(context.Background(), "session")
	if err != nil {
		t.Fatal(err)
	}
	if poll.State != miguQRConfirmed || poll.Token != "passport-token" || poll.RedirectURL != "" {
		t.Fatalf("poll = %+v", poll)
	}
}

func TestCheckMiguToken(t *testing.T) {
	old := miguTokenCheckURL
	defer func() { miguTokenCheckURL = old }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("pacmtoken"); got == "" {
			t.Error("token was not sent")
		}
		code := "000000"
		if r.URL.Query().Get("invalid") == "1" {
			code = "290001"
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "info": "请先登录"})
	}))
	defer server.Close()

	miguTokenCheckURL = server.URL
	if err := checkMiguToken(context.Background(), "valid-token"); err != nil {
		t.Fatalf("valid token: %v", err)
	}
	miguTokenCheckURL = server.URL + "?invalid=1"
	if err := checkMiguToken(context.Background(), "stale-token"); !errors.Is(err, errMiguTokenStale) {
		t.Fatalf("stale token err = %v", err)
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

func TestMiguQRResponseAcceptsStringStatus(t *testing.T) {
	var response miguQRResponse
	if err := json.Unmarshal([]byte(`{"status":"4074","message":"未扫码","result":{}}`), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != 4074 || !strings.Contains(response.Message, "未扫码") {
		t.Fatalf("response = %+v", response)
	}
}
