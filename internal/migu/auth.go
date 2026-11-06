package migu

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	miguLoginPageURL     = "https://passport.migu.cn/login?sourceid=204040&apptype=2&forceAuthn=true&isPassive=false&display=wap&relayState=&logintype=3"
	miguQRCreateURL      = "https://passport.migu.cn/api/qrcWeb/qrcLogin?sourceid=204040"
	miguQRQueryURL       = "https://passport.migu.cn/api/qrcWeb/qrcquery"
	miguTokenValidateURL = "https://c.musicapp.migu.cn/user/h5/token-validate/v3.0"
	miguTokenCheckURL    = "https://app.u.nf.migu.cn/pc/user/home-page/trust-uid/v1.0"
	errMiguTokenStale    = errors.New("咪咕登录状态已失效")
	miguInputTagRE       = regexp.MustCompile(`(?is)<input\b[^>]*>`)
	miguInputAttrRE      = regexp.MustCompile(`(?is)\b(id|value)=["']([^"']*)["']`)
	miguAuthHTTPClient   = &http.Client{Timeout: 25 * time.Second}
)

type miguQRState int

const (
	miguQRWaiting miguQRState = iota
	miguQRScanned
	miguQRExpired
	miguQRConfirmed
)

type miguQRCode struct {
	Image     []byte
	SessionID string
}

type miguQRPoll struct {
	State       miguQRState
	Token       string
	RedirectURL string
}

type miguAuthSession struct {
	client      *http.Client
	jar         http.CookieJar
	callbackURL string
	relayState  string
}

type miguQRResponse struct {
	Status  flexInt `json:"status"`
	Message string  `json:"message"`
	Result  struct {
		QRCURL      string `json:"qrcUrl"`
		SessionID   string `json:"qrc_sessionid"`
		Token       string `json:"token"`
		RedirectURL string `json:"redirectURL"`
	} `json:"result"`
}

type flexInt int

func (n *flexInt) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*n = 0
		return nil
	}
	data = bytes.Trim(data, `"`)
	var value int
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*n = flexInt(value)
	return nil
}

func newMiguAuthSession() (*miguAuthSession, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := *miguAuthHTTPClient
	client.Jar = jar
	return &miguAuthSession{client: &client, jar: jar}, nil
}

func (s *miguAuthSession) browserGet(ctx context.Context, endpoint, accept string) ([]byte, *http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", miguUA)
	req.Header.Set("Accept", accept)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Referer", miguLoginPageURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	res, err := s.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, res, fmt.Errorf("咪咕登录服务返回 %s", res.Status)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, res, err
	}
	return body, res, nil
}

func (s *miguAuthSession) startQR(ctx context.Context) (miguQRCode, error) {
	loginBody, _, err := s.browserGet(ctx, miguLoginPageURL, "text/html,application/xhtml+xml")
	if err != nil {
		return miguQRCode{}, err
	}
	hidden := miguParseHiddenValues(loginBody)
	s.callbackURL = hidden["callbackURL"]
	s.relayState = hidden["relayState"]

	body, _, err := s.browserGet(ctx, miguQRCreateURL, "application/json, text/plain, */*")
	if err != nil {
		return miguQRCode{}, err
	}
	var response miguQRResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return miguQRCode{}, fmt.Errorf("解析咪咕二维码响应失败：%w", err)
	}
	if response.Status != 2000 {
		message := strings.TrimSpace(response.Message)
		if message == "" {
			message = "咪咕没有返回二维码"
		}
		return miguQRCode{}, errors.New(message)
	}
	image, err := miguDecodeQRImage(response.Result.QRCURL)
	if err != nil {
		return miguQRCode{}, err
	}
	if strings.TrimSpace(response.Result.SessionID) == "" {
		return miguQRCode{}, errors.New("咪咕二维码缺少会话编号")
	}
	return miguQRCode{
		Image:     image,
		SessionID: response.Result.SessionID,
	}, nil
}

func (s *miguAuthSession) pollQR(ctx context.Context, sessionID string) (miguQRPoll, error) {
	query := url.Values{}
	query.Set("qrc_sessionid", sessionID)
	body, _, err := s.browserGet(ctx, miguQRQueryURL+"?"+query.Encode(), "application/json, text/plain, */*")
	if err != nil {
		return miguQRPoll{}, err
	}
	var response miguQRResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return miguQRPoll{}, fmt.Errorf("解析咪咕扫码状态失败：%w", err)
	}
	switch int(response.Status) {
	case 4074:
		return miguQRPoll{State: miguQRWaiting}, nil
	case 4073:
		return miguQRPoll{State: miguQRScanned}, nil
	case 4072:
		return miguQRPoll{State: miguQRExpired}, nil
	case 2000:
		if strings.TrimSpace(response.Result.Token) == "" {
			return miguQRPoll{}, errors.New("咪咕登录成功但没有返回令牌")
		}
		return miguQRPoll{
			State:       miguQRConfirmed,
			Token:       response.Result.Token,
			RedirectURL: response.Result.RedirectURL,
		}, nil
	default:
		message := strings.TrimSpace(response.Message)
		if message == "" {
			message = fmt.Sprintf("未知扫码状态 %d", response.Status)
		}
		return miguQRPoll{}, errors.New(message)
	}
}

func (s *miguAuthSession) completeLogin(ctx context.Context, redirectURL, token string) (string, error) {
	token = normalizeMiguToken(token)
	var exchangeErr error
	if token != "" {
		pacToken, err := s.exchangeLoginToken(ctx, token)
		if err == nil {
			return pacToken, nil
		}
		exchangeErr = err
	}

	if strings.TrimSpace(redirectURL) != "" {
		target, err := miguQRRedirectTarget(redirectURL, token, s.callbackURL, s.relayState)
		if err == nil {
			body, res, err := s.browserGet(ctx, target, "text/html,application/xhtml+xml")
			if err == nil {
				if token := miguTokenFromJar(s.jar, res.Request.URL); token != "" {
					return token, nil
				}
				for _, raw := range miguTokenURLsFromBody(body) {
					target, err := url.Parse(raw)
					if err == nil {
						if token := miguTokenFromJar(s.jar, target); token != "" {
							return token, nil
						}
					}
				}
			}
		}
	}
	if exchangeErr != nil {
		return "", fmt.Errorf("登录成功，但没有换取到 pacmtoken：%w", exchangeErr)
	}
	return "", errors.New("登录成功，但没有取得 pacmtoken；请展开高级区域手动导入")
}

// exchangeLoginToken turns the short-lived token returned by the passport QR
// flow into the pacmtoken used by the music APIs. The music web SDK does
// this through token-validate as well; the passport flow does not set the
// pacmtoken cookie itself.
func (s *miguAuthSession) exchangeLoginToken(ctx context.Context, loginToken string) (string, error) {
	return exchangeMiguLoginToken(ctx, s.client, s.jar, loginToken)
}

func exchangeMiguLoginToken(ctx context.Context, client *http.Client, jar http.CookieJar, loginToken string) (string, error) {
	loginToken = normalizeMiguToken(loginToken)
	if loginToken == "" {
		return "", errors.New("缺少咪咕登录令牌")
	}
	if client == nil {
		if jar == nil {
			jar, _ = cookiejar.New(nil)
		}
		c := *miguAuthHTTPClient
		c.Jar = jar
		client = &c
	} else if jar == nil {
		jar = client.Jar
	}
	values := url.Values{}
	values.Set("token", loginToken)
	values.Set("type", "2")
	values.Set("sourceId", "220029")
	endpoint := miguTokenValidateURL + "?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", miguUA)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Origin", "https://music.migu.cn")
	req.Header.Set("Referer", "https://music.migu.cn/")
	req.Header.Set("deviceId", miguDeviceID())

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return "", err
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("咪咕登录服务返回 %s", res.Status)
	}

	var response struct {
		Code string `json:"code"`
		Info string `json:"info"`
		Data struct {
			PacMToken string `json:"pacmtoken"`
			PacToken  string `json:"pacToken"`
		} `json:"data"`
	}
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &response); err != nil {
			return "", fmt.Errorf("解析咪咕登录令牌失败：%w", err)
		}
	}
	if response.Code != "000000" {
		if info := strings.TrimSpace(response.Info); info != "" {
			return "", errors.New(info)
		}
		if response.Code == "" {
			return "", errors.New("咪咕登录令牌校验没有返回结果")
		}
		return "", fmt.Errorf("咪咕登录令牌校验返回 %q", response.Code)
	}

	if token := normalizeMiguToken(res.Header.Get("pacmtoken")); token != "" {
		return token, nil
	}
	if target, err := url.Parse("https://c.musicapp.migu.cn/"); err == nil {
		if token := miguTokenFromJar(jar, target); token != "" {
			return token, nil
		}
	}
	if token := normalizeMiguToken(firstNonEmpty(response.Data.PacMToken, response.Data.PacToken)); token != "" {
		return token, nil
	}
	return "", errors.New("咪咕登录成功，但响应里没有 pacmtoken")
}

func miguDeviceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("freewave-%d", time.Now().UnixNano())
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	s := hex.EncodeToString(b[:])
	return s[0:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:32]
}

// miguLoginTokenFromURL finds the passport token appended to the login
// redirect URL. The music SDK accepts token, utoken, miguToken and ptoken.
func miguLoginTokenFromURL(rawURL string) string {
	return miguTokenParamFromURL(rawURL, "token", "utoken", "migutoken", "ptoken")
}

func miguTokenParamFromURL(rawURL string, keys ...string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || len(keys) == 0 {
		return ""
	}
	lowerKeys := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		lowerKeys[strings.ToLower(string(key))] = struct{}{}
	}

	// Scan the raw URL as well as its parsed query and fragment. Login
	// redirects sometimes look like /#/route?token=..., where the query is
	// nested inside the fragment and url.Query only sees the outer URL.
	for i := 0; i < len(rawURL); i++ {
		if rawURL[i] != '?' && rawURL[i] != '&' && rawURL[i] != '#' {
			continue
		}
		start := i + 1
		end := start
		for end < len(rawURL) && rawURL[end] != '&' && rawURL[end] != '#' {
			end++
		}
		key, value, ok := strings.Cut(rawURL[start:end], "=")
		if !ok {
			continue
		}
		decodedKey, err := url.QueryUnescape(key)
		if err != nil {
			decodedKey = key
		}
		if _, ok := lowerKeys[strings.ToLower(decodedKey)]; !ok {
			continue
		}
		if decodedValue, err := url.QueryUnescape(value); err == nil {
			value = decodedValue
		}
		if token := normalizeMiguToken(value); token != "" {
			return token
		}
	}

	target, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	candidates := []url.Values{target.Query()}
	if fragment := target.Fragment; fragment != "" {
		if queryAt := strings.IndexByte(fragment, '?'); queryAt >= 0 {
			if values, err := url.ParseQuery(fragment[queryAt+1:]); err == nil {
				candidates = append(candidates, values)
			}
		}
		if values, err := url.ParseQuery(fragment); err == nil {
			candidates = append(candidates, values)
		}
	}
	for _, values := range candidates {
		for key, items := range values {
			if _, ok := lowerKeys[strings.ToLower(key)]; !ok {
				continue
			}
			for _, item := range items {
				if token := normalizeMiguToken(item); token != "" {
					return token
				}
			}
		}
	}
	return ""
}

func miguQRRedirectTarget(base, token, callbackURL, relayState string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return "", errors.New("缺少咪咕登录跳转地址")
	}
	target, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	query := target.Query()
	query.Set("callbackURL", callbackURL)
	query.Set("relayState", relayState)
	query.Set("token", token)
	query.Set("qrclogin", "1")
	query.Set("logintype", "QRCSSO")
	target.RawQuery = query.Encode()
	return target.String(), nil
}

func miguParseHiddenValues(body []byte) map[string]string {
	values := map[string]string{}
	for _, tag := range miguInputTagRE.FindAll(body, -1) {
		attrs := map[string]string{}
		for _, match := range miguInputAttrRE.FindAllSubmatch(tag, -1) {
			if len(match) == 3 {
				attrs[strings.ToLower(string(match[1]))] = string(match[2])
			}
		}
		if id := attrs["id"]; id != "" {
			values[id] = attrs["value"]
		}
	}
	return values
}

func miguDecodeQRImage(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("咪咕返回了空二维码")
	}
	if !strings.HasPrefix(raw, "data:") {
		return nil, errors.New("咪咕二维码格式无法识别")
	}
	comma := strings.IndexByte(raw, ',')
	if comma < 0 {
		return nil, errors.New("咪咕二维码数据不完整")
	}
	meta, payload := raw[:comma], raw[comma+1:]
	var (
		image []byte
		err   error
	)
	if strings.Contains(meta, ";base64") {
		image, err = base64.StdEncoding.DecodeString(payload)
	} else {
		var decoded string
		decoded, err = url.QueryUnescape(payload)
		image = []byte(decoded)
	}
	if err != nil {
		return nil, fmt.Errorf("解码咪咕二维码失败：%w", err)
	}
	if len(image) == 0 {
		return nil, errors.New("咪咕返回了空二维码")
	}
	return image, nil
}

func miguTokenFromJar(jar http.CookieJar, final *url.URL) string {
	if jar == nil {
		return ""
	}
	urls := []*url.URL{final}
	for _, raw := range []string{
		"https://music.migu.cn/",
		"https://c.musicapp.migu.cn/",
		"https://passport.migu.cn/",
		"https://app.u.nf.migu.cn/",
	} {
		if target, err := url.Parse(raw); err == nil {
			urls = append(urls, target)
		}
	}
	for _, target := range urls {
		if target == nil {
			continue
		}
		for _, cookie := range jar.Cookies(target) {
			if strings.EqualFold(cookie.Name, "pacmtoken") {
				if token := normalizeMiguToken(cookie.Value); token != "" {
					return token
				}
			}
		}
	}
	return ""
}

func miguTokenURLsFromBody(body []byte) []string {
	text := string(body)
	matches := regexp.MustCompile(`https?://[^\s"'<>]+`).FindAllString(text, 12)
	for i, match := range matches {
		matches[i] = strings.TrimRight(match, "),.;")
	}
	return matches
}

func checkMiguToken(ctx context.Context, token string) error {
	token = normalizeMiguToken(token)
	if token == "" {
		return errMiguTokenStale
	}
	body, err := miguGet(ctx, miguTokenCheckURL, token)
	if err != nil {
		return err
	}
	var response miguEnvelope
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}
	switch response.Code {
	case "000000":
		return nil
	case "290001":
		return errMiguTokenStale
	default:
		if strings.TrimSpace(response.Info) != "" {
			return fmt.Errorf("咪咕登录校验：%s", response.Info)
		}
		return fmt.Errorf("咪咕登录校验返回 %q", response.Code)
	}
}
