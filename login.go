package main

import (
	"context"
	"errors"
	"time"

	"github.com/egoist/mygo"
)

// miguWebLoginURL is the desktop login page shown inside the app. It is the
// same passport page the browser uses, so account, SMS and SIM login all work
// without the app ever seeing a password.
const miguWebLoginURL = "https://passport.migu.cn/login?sourceid=204040&apptype=2&forceAuthn=true&isPassive=false&display=pc&relayState=&logintype=3"

// startBrowserLogin shows migu's own login page in an embedded web view and
// watches the login session for the pacmtoken cookie.
func (a *app) startBrowserLogin() {
	ctx, gen := a.beginMiguAuth()
	a.authNotice = ""
	if a.loginViewAlive() {
		a.authPhase = miguAuthBrowserWaiting
		a.refresh()
		go a.watchLoginCookies(ctx, gen, a.loginSession)
		return
	}
	a.authPhase = miguAuthBrowserLoading
	a.refresh()

	win := a.win
	go func() {
		view, session, err := newLoginWebView(win)
		if err != nil && ctx.Err() != nil {
			return
		}
		a.post(func() {
			if !a.finishMiguAuth(gen) {
				if view != nil && !view.IsDestroyed() {
					view.Destroy()
				}
				return
			}
			if err != nil {
				a.authPhase = miguAuthError
				a.authNotice = "无法打开咪咕登录页：" + friendlyError(err)
				a.refresh()
				return
			}
			a.loginView = view
			a.loginSession = session
			view.Page().SetAudioMuted(true)
			view.Page().OnDidNavigate(func(string) { a.refresh() })
			a.authPhase = miguAuthBrowserWaiting
			a.refresh()
		})
		if err != nil {
			return
		}
		a.watchLoginCookies(ctx, gen, session)
	}()
}

// newLoginWebView creates the embedded login page. The session is
// ephemeral: the page's cookies go away with the view, and only the login
// token is kept, in the system credential store.
func newLoginWebView(win *mygo.Window) (*mygo.WebView, *mygo.Session, error) {
	if win == nil {
		return nil, nil, errors.New("主窗口尚未就绪")
	}
	session := mygo.NewEphemeralSession()
	view, err := win.NewWebView(mygo.WebViewOptions{
		URL:             miguWebLoginURL,
		BackgroundColor: "#ffffff",
		Page: mygo.PageOptions{
			Session:         session,
			Untrusted:       true,
			UserAgentSuffix: "MiguWave/0.1.0",
			SwipeNavigation: true,
			PinchZoom:       true,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	return view, session, nil
}

func (a *app) watchLoginCookies(ctx context.Context, gen int, session *mygo.Session) {
	if session == nil {
		return
	}
	ticker := time.NewTicker(1200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		token := loginCookieToken(session)
		if token == "" {
			continue
		}
		checkCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		err := checkMiguToken(checkCtx, token)
		cancel()
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, errMiguTokenStale) {
			continue
		}
		a.post(func() {
			if !a.finishMiguAuth(gen) {
				return
			}
			a.connectMiguToken(token, err)
		})
		return
	}
}

// loginCookieToken reads the pacmtoken the login page set. Cookies are
// HTTP-only, so this reads the session store rather than page script.
func loginCookieToken(session *mygo.Session) string {
	if session == nil {
		return ""
	}
	cookies, err := session.Cookies.List(mygo.CookieFilter{Name: "pacmtoken", Domain: "migu.cn"})
	if err != nil {
		return ""
	}
	for _, cookie := range cookies {
		if token := normalizeMiguToken(cookie.Value); token != "" {
			return token
		}
	}
	return ""
}

func (a *app) loginViewAlive() bool {
	return a.loginView != nil && !a.loginView.IsDestroyed()
}

func (a *app) closeLoginView() {
	if a.loginViewAlive() {
		a.loginView.Destroy()
	}
	a.loginView = nil
	a.loginSession = nil
}

// reloadLoginView recreates the page when the first load failed.
func (a *app) reloadLoginView() {
	a.closeLoginView()
	a.startBrowserLogin()
}

// connectMiguToken finishes either login flow: the token is remembered in the
// keychain and the modal switches to its connected state.
func (a *app) connectMiguToken(token string, validationErr error) {
	token = normalizeMiguToken(token)
	if token == "" {
		a.authPhase = miguAuthInvalid
		a.authNotice = "没有取得有效的咪咕登录令牌，请重试。"
		a.refresh()
		return
	}
	a.miguToken = token
	a.authPhase = miguAuthConnected
	a.authQR = nil
	a.tokenDraft = ""
	a.closeLoginView()
	switch {
	case validationErr != nil:
		a.authNotice = "已登录咪咕账号，但暂时无法完成在线校验：" + friendlyError(validationErr)
	case saveMiguToken(token) != nil:
		a.authNotice = "已连接。令牌本次可用，但保存到系统钥匙串失败。"
	default:
		a.authNotice = "已连接，账号有权限的曲目会使用你的咪咕登录状态。"
	}
	a.toast = "咪咕账号已连接"
	a.refresh()
}
