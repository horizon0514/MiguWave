package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"

	"freewave/internal/migu"
)

// miguWebLoginURL is the desktop login page shown inside the app. It is the
// same passport page the browser uses, so account, SMS and SIM login all work
// without the app ever seeing a password.
const miguWebLoginURL = "https://passport.migu.cn/login?sourceid=220029&apptype=2&forceAuthn=true&isPassive=false&display=pc&callbackURL=PostToken&hideRegister=1&hideForgetPass=1"

// startBrowserLogin shows migu's own login page in an embedded web view and
// watches the login session for the pacmtoken cookie.
func (a *app) startBrowserLogin() {
	ctx, gen := a.beginMiguAuth()
	a.authNotice = ""
	if a.loginViewAlive() {
		a.authPhase = miguAuthBrowserWaiting
		a.refresh()
		go a.watchLoginCookies(ctx, gen, a.loginView, a.loginSession, a.loginNav)
		return
	}
	a.authPhase = miguAuthBrowserLoading
	a.refresh()

	a.post(func() {
		if a.authGen != gen {
			return
		}
		view, session, err := newLoginWebView(a.win)
		if ctx.Err() != nil {
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
		nav := make(chan string, 16)
		a.loginView = view
		a.loginSession = session
		a.loginNav = nav
		page := view.Page()
		page.SetAudioMuted(true)
		page.OnWillNavigate(func(*mygo.NavigateEvent) {
			// The page is untrusted, but the login flow must follow migu's
			// cross-origin redirects. Attaching this handler opts out of
			// MyGo's initial-origin guard; the page still cannot call Go
			// methods or open windows.
		})
		page.OnDidNavigate(func(raw string) {
			notifyLoginNavigation(nav, raw)
			a.refresh()
		})
		page.OnDOMReady(func() {
			notifyLoginNavigation(nav, page.URL())
		})
		page.OnDidFinishLoad(func() {
			notifyLoginNavigation(nav, page.URL())
		})
		page.OnDidFailLoad(func(loadErr *mygo.LoadError) {
			if loadErr == nil || loadErr.Code == -999 {
				return
			}
			a.post(func() {
				if a.loginView != view {
					return
				}
				a.authNotice = "咪咕登录页加载失败：" + friendlyError(loadErr)
				a.refresh()
			})
		})
		if err := page.LoadURL(miguWebLoginURL); err != nil {
			view.Destroy()
			a.closeLoginView()
			a.authPhase = miguAuthError
			a.authNotice = "无法打开咪咕登录页：" + friendlyError(err)
			a.refresh()
			return
		}
		a.authPhase = miguAuthBrowserWaiting
		a.refresh()
		go a.watchLoginCookies(ctx, gen, view, session, nav)
	})
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

func notifyLoginNavigation(nav chan string, raw string) {
	raw = strings.TrimSpace(raw)
	if nav == nil || raw == "" || raw == "about:blank" {
		return
	}
	select {
	case nav <- raw:
	default:
	}
}

func (a *app) watchLoginCookies(ctx context.Context, gen int, view *mygo.WebView, session *mygo.Session, nav <-chan string) {
	if view == nil && session == nil {
		return
	}
	ticker := time.NewTicker(1200 * time.Millisecond)
	defer ticker.Stop()
	failures := 0

	check := func(resolve func() (string, error)) bool {
		token, err := resolve()
		if ctx.Err() != nil {
			return true
		}
		if token == "" {
			if err != nil && !errors.Is(err, errMiguTokenStale) {
				failures++
				if failures >= 8 {
					a.post(func() {
						if !a.finishMiguAuth(gen) {
							return
						}
						a.authPhase = miguAuthError
						a.authNotice = "登录成功，但无法取得咪咕令牌：" + friendlyError(err)
						a.refresh()
					})
					return true
				}
			}
			return false
		}
		failures = 0
		a.post(func() {
			if !a.finishMiguAuth(gen) {
				return
			}
			a.connectMiguToken(token, err)
		})
		return true
	}

	for {
		select {
		case <-ctx.Done():
			return
		case raw := <-nav:
			if check(func() (string, error) {
				return resolveMiguBrowserTokenAtURL(ctx, view, session, raw)
			}) {
				return
			}
		case <-ticker.C:
			if check(func() (string, error) {
				return resolveMiguBrowserToken(ctx, view, session)
			}) {
				return
			}
		}
	}
}

type miguBrowserToken struct {
	value      string
	loginToken bool
}

// resolveMiguBrowserToken looks in every place the passport page may keep the
// login result: the cookie jar, localStorage and the current URL. A direct
// pacmtoken is validated; a short-lived passport token is exchanged through
// token-validate first.
func resolveMiguBrowserToken(ctx context.Context, view *mygo.WebView, session *mygo.Session) (string, error) {
	return resolveMiguBrowserTokens(ctx, session, collectMiguBrowserTokens(view, session))
}

func resolveMiguBrowserTokenAtURL(ctx context.Context, view *mygo.WebView, session *mygo.Session, raw string) (string, error) {
	return resolveMiguBrowserTokens(ctx, session, collectMiguBrowserTokens(view, session, raw))
}

func resolveMiguBrowserTokens(ctx context.Context, session *mygo.Session, candidates []miguBrowserToken) (string, error) {
	jar := miguBrowserCookieJar(session)
	var lastErr error
	var networkToken string
	var networkErr error
	for _, candidate := range candidates {
		if candidate.loginToken {
			exchangeCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			token, err := migu.ExchangeLoginToken(exchangeCtx, nil, jar, candidate.value)
			cancel()
			if err == nil {
				return token, nil
			}
			// Some passport callbacks put a direct pacmtoken in the token
			// parameter. If it cannot be exchanged, validate it as-is.
			checkCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			checkErr := migu.CheckToken(checkCtx, candidate.value)
			cancel()
			if checkErr == nil {
				return candidate.value, nil
			}
			if !errors.Is(checkErr, errMiguTokenStale) {
				if networkToken == "" {
					networkToken = candidate.value
					networkErr = checkErr
				}
			}
			lastErr = err
			continue
		}

		checkCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		err := migu.CheckToken(checkCtx, candidate.value)
		cancel()
		if err == nil {
			return candidate.value, nil
		}
		if errors.Is(err, errMiguTokenStale) {
			// Some builds put the short-lived passport token in localStorage.
			exchangeCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			token, exchangeErr := migu.ExchangeLoginToken(exchangeCtx, nil, jar, candidate.value)
			cancel()
			if exchangeErr == nil {
				return token, nil
			}
			lastErr = nil
			continue
		}
		// The token exists but validation failed, usually because the network
		// is unavailable. Keep looking for a token that does validate.
		if networkToken == "" {
			networkToken = candidate.value
			networkErr = err
		}
	}
	if networkToken != "" {
		return networkToken, networkErr
	}
	return "", lastErr
}

// miguBrowserCookieJar copies the webview session's cookies so the
// token-validate request looks like it comes from the page that just logged
// in.
func miguBrowserCookieJar(session *mygo.Session) http.CookieJar {
	jar, _ := cookiejar.New(nil)
	if session == nil {
		return jar
	}
	cookies, err := session.Cookies.List(mygo.CookieFilter{})
	if err != nil {
		return jar
	}
	for _, cookie := range cookies {
		host := strings.TrimPrefix(strings.ToLower(cookie.Domain), ".")
		if host == "" {
			continue
		}
		path := cookie.Path
		if path == "" {
			path = "/"
		}
		target := &url.URL{Scheme: "https", Host: host, Path: path}
		jar.SetCookies(target, []*http.Cookie{{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Domain: cookie.Domain,
			Path:   path,
		}})
	}
	return jar
}

func collectMiguBrowserTokens(view *mygo.WebView, session *mygo.Session, extraURLs ...string) []miguBrowserToken {
	var candidates []miguBrowserToken
	seen := map[string]bool{}
	add := func(value string, loginToken bool) {
		value = migu.NormalizeToken(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		candidates = append(candidates, miguBrowserToken{value: value, loginToken: loginToken})
	}

	for _, raw := range extraURLs {
		addMiguURLTokens(add, raw)
	}
	var href string
	if view != nil && !view.IsDestroyed() {
		href = view.Page().URL()
		if strings.TrimSpace(href) == "" || href == "about:blank" {
			href = evalLoginPage(view, "location.href")
		}
		addMiguURLTokens(add, href)
	}
	for _, token := range loginCookieTokens(session) {
		add(token, false)
	}
	if view != nil && !view.IsDestroyed() {
		collectMiguStorageTokens(view, add)
	}
	return candidates
}

func addMiguURLTokens(add func(string, bool), raw string) {
	if strings.TrimSpace(raw) == "" {
		return
	}
	add(migu.TokenParamFromURL(raw, "pacmtoken"), false)
	add(migu.LoginTokenFromURL(raw), true)
}

const miguLoginStorageScript = `(() => {
  try {
    const out = [];
    for (const [storeName, store] of [["localStorage", localStorage], ["sessionStorage", sessionStorage]]) {
      for (let i = 0; i < store.length; i++) {
        const key = store.key(i) || "";
        const value = store.getItem(key);
        if (value) out.push({store: storeName, key: key, value: String(value)});
      }
    }
    return JSON.stringify(out);
  } catch (e) {
    return "[]";
  }
})()`

type miguStorageEntry struct {
	Store string `json:"store"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

func collectMiguStorageTokens(view *mygo.WebView, add func(string, bool)) {
	raw := evalLoginPage(view, miguLoginStorageScript)
	if strings.TrimSpace(raw) == "" {
		return
	}
	var entries []miguStorageEntry
	if json.Unmarshal([]byte(raw), &entries) != nil {
		return
	}
	for _, entry := range entries {
		collectMiguStorageValue(entry.Key, entry.Value, add)
	}
}

func collectMiguStorageValue(key, value string, add func(string, bool)) {
	lowerKey := strings.ToLower(strings.TrimSpace(key))
	if isMiguPacmTokenKey(lowerKey) {
		add(value, false)
	} else if isMiguLoginTokenKey(lowerKey) {
		add(value, true)
	}

	var decoded any
	if json.Unmarshal([]byte(value), &decoded) == nil {
		collectMiguTokenValues("", decoded, add)
	}

	add(migu.TokenParamFromURL(value, "pacmtoken"), false)
	add(migu.LoginTokenFromURL(value), true)
}

func collectMiguTokenValues(key string, value any, add func(string, bool)) {
	lowerKey := strings.ToLower(strings.TrimSpace(key))
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for childKey := range typed {
			keys = append(keys, childKey)
		}
		sort.Strings(keys)
		for _, childKey := range keys {
			child := typed[childKey]
			collectMiguTokenValues(childKey, child, add)
		}
	case []any:
		for _, child := range typed {
			collectMiguTokenValues(key, child, add)
		}
	case string:
		switch {
		case isMiguPacmTokenKey(lowerKey):
			add(typed, false)
		case isMiguLoginTokenKey(lowerKey):
			add(typed, true)
		default:
			add(migu.TokenParamFromURL(typed, "pacmtoken"), false)
			add(migu.LoginTokenFromURL(typed), true)
		}
	}
}

func isMiguPacmTokenKey(key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(key, "pacmtoken") ||
		strings.Contains(key, "pacm_token") ||
		strings.Contains(key, "pacm-token")
}

func isMiguLoginTokenKey(key string) bool {
	key = strings.ToLower(key)
	switch key {
	case "token", "utoken", "migutoken", "migu_token", "ptoken", "login_token", "logintoken", "passport_token", "passporttoken":
		return true
	}
	return (strings.Contains(key, "migu") || strings.Contains(key, "mg_auth")) &&
		strings.Contains(key, "token")
}

func evalLoginPage(view *mygo.WebView, script string) string {
	if view == nil || view.IsDestroyed() {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	value, err := view.Page().EvalContext(ctx, script)
	if err != nil {
		return ""
	}
	text, _ := value.(string)
	return text
}

// loginCookieToken reads the pacmtoken the login page set. Cookies are
// HTTP-only, so this reads the session store rather than page script.
func loginCookieToken(session *mygo.Session) string {
	tokens := loginCookieTokens(session)
	if len(tokens) == 0 {
		return ""
	}
	return tokens[0]
}

func loginCookieTokens(session *mygo.Session) []string {
	if session == nil {
		return nil
	}
	cookies, err := session.Cookies.List(mygo.CookieFilter{Domain: "migu.cn"})
	if err != nil {
		return nil
	}
	var tokens []string
	seen := map[string]bool{}
	for _, cookie := range cookies {
		if !strings.EqualFold(cookie.Name, "pacmtoken") {
			continue
		}
		if token := migu.NormalizeToken(cookie.Value); token != "" && !seen[token] {
			seen[token] = true
			tokens = append(tokens, token)
		}
	}
	return tokens
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
	a.loginNav = nil
}

// reloadLoginView recreates the page when the first load failed.
func (a *app) reloadLoginView() {
	a.closeLoginView()
	a.startBrowserLogin()
}

// connectMiguToken finishes either login flow: the token is written to the
// local token file and the modal switches to its connected state.
func (a *app) connectMiguToken(token string, validationErr error) {
	token = migu.NormalizeToken(token)
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
	a.miguProfile = nil
	a.miguProfileErr = ""
	a.miguProfileLoading = false
	a.closeLoginView()
	switch {
	case validationErr != nil:
		a.authNotice = "已登录咪咕账号，但暂时无法完成在线校验：" + friendlyError(validationErr)
	case migu.SaveToken(token) != nil:
		a.authNotice = "已连接。令牌本次可用，但保存登录令牌失败。"
	default:
		a.authNotice = "已连接，账号有权限的曲目会使用你的咪咕登录状态。"
	}
	a.toast = "咪咕账号已连接"
	a.refresh()
	a.loadMiguProfile(token)
}
