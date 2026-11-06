package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type miguAuthPhase int

const (
	miguAuthIdle miguAuthPhase = iota
	miguAuthChecking
	miguAuthQRLoading
	miguAuthQRWaiting
	miguAuthQRScanned
	miguAuthVerifying
	miguAuthConnected
	miguAuthQRExpired
	miguAuthInvalid
	miguAuthError
	miguAuthBrowserLoading
	miguAuthBrowserWaiting
)

type authMode int

const (
	authModeBrowser authMode = iota
	authModeQR
	authModeToken
)

var readClipboardText = func() string {
	return mygo.Clipboard.ReadText()
}

func (a *app) openAccount() {
	a.tokenDraft = ""
	a.authNotice = ""
	a.accountOpen = true
	if a.miguToken != "" {
		a.verifyMiguToken()
		return
	}
	a.enterAuthMode(a.authMode)
}

func (a *app) closeAccount() {
	a.accountOpen = false
	a.stopMiguAuth()
	a.closeLoginView()
	a.refresh()
}

// enterAuthMode starts the flow of the mode shown, without resetting it when
// the modal opens again.
func (a *app) enterAuthMode(mode authMode) {
	switch mode {
	case authModeQR:
		switch a.authPhase {
		case miguAuthQRLoading, miguAuthQRWaiting, miguAuthQRScanned, miguAuthVerifying:
		default:
			a.startMiguQRLogin()
		}
	case authModeToken:
		a.stopMiguAuth()
		a.authPhase = miguAuthIdle
		a.refresh()
	default:
		a.startBrowserLogin()
	}
}

func (a *app) setAuthMode(mode authMode) {
	if a.authMode == mode {
		return
	}
	a.authMode = mode
	a.authNotice = ""
	a.tokenDraft = ""
	a.stopMiguAuth()
	switch mode {
	case authModeQR:
		a.authQR = nil
		a.startMiguQRLogin()
	case authModeToken:
		a.authQR = nil
		a.authPhase = miguAuthIdle
	default:
		a.authQR = nil
		a.startBrowserLogin()
	}
	a.refresh()
}

func (a *app) beginMiguAuth() (context.Context, int) {
	if a.authCancel != nil {
		a.authCancel()
		a.authCancel = nil
	}
	a.authGen++
	gen := a.authGen
	ctx, cancel := context.WithCancel(context.Background())
	a.authCancel = cancel
	return ctx, gen
}

func (a *app) stopMiguAuth() {
	if a.authCancel != nil {
		a.authCancel()
		a.authCancel = nil
	}
	a.authGen++
}

func (a *app) finishMiguAuth(gen int) bool {
	if a.authGen != gen {
		return false
	}
	if a.authCancel != nil {
		a.authCancel()
		a.authCancel = nil
	}
	return true
}

func (a *app) startMiguQRLogin() {
	ctx, gen := a.beginMiguAuth()
	a.authPhase = miguAuthQRLoading
	a.authQR = nil
	a.authNotice = ""
	a.refresh()

	go func() {
		session, err := newMiguAuthSession()
		if err != nil {
			a.post(func() {
				if !a.finishMiguAuth(gen) {
					return
				}
				a.authPhase = miguAuthError
				a.authNotice = "无法创建登录会话：" + friendlyError(err)
			})
			return
		}

		code, err := session.startQR(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			a.post(func() {
				if !a.finishMiguAuth(gen) {
					return
				}
				a.authPhase = miguAuthError
				a.authNotice = "二维码生成失败：" + friendlyError(err)
			})
			return
		}
		image, err := ui.DecodeImage(code.Image)
		if err != nil {
			a.post(func() {
				if !a.finishMiguAuth(gen) {
					return
				}
				a.authPhase = miguAuthError
				a.authNotice = "二维码图片解析失败：" + friendlyError(err)
			})
			return
		}
		a.post(func() {
			if a.authGen != gen {
				return
			}
			a.authQR = image
			a.authPhase = miguAuthQRWaiting
			a.authNotice = ""
		})

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		failures := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			poll, err := session.pollQR(ctx, code.SessionID)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				failures++
				if failures < 3 {
					continue
				}
				a.post(func() {
					if !a.finishMiguAuth(gen) {
						return
					}
					a.authPhase = miguAuthError
					a.authNotice = "检查扫码状态失败：" + friendlyError(err)
				})
				return
			}
			failures = 0

			switch poll.State {
			case miguQRWaiting:
				a.post(func() {
					if a.authGen == gen && a.authPhase == miguAuthQRLoading {
						a.authPhase = miguAuthQRWaiting
					}
				})
			case miguQRScanned:
				a.post(func() {
					if a.authGen == gen {
						a.authPhase = miguAuthQRScanned
						a.authNotice = ""
					}
				})
			case miguQRExpired:
				a.post(func() {
					if !a.finishMiguAuth(gen) {
						return
					}
					a.authPhase = miguAuthQRExpired
					a.authNotice = "二维码已过期，刷新后再扫一次。"
				})
				return
			case miguQRConfirmed:
				a.post(func() {
					if a.authGen == gen {
						a.authPhase = miguAuthVerifying
						a.authNotice = ""
					}
				})
				token, err := session.completeLogin(ctx, poll.RedirectURL, poll.Token)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					a.post(func() {
						if !a.finishMiguAuth(gen) {
							return
						}
						a.authPhase = miguAuthError
						a.authNotice = friendlyError(err)
					})
					return
				}
				validationErr := checkMiguToken(ctx, token)
				a.post(func() {
					if !a.finishMiguAuth(gen) {
						return
					}
					if errors.Is(validationErr, errMiguTokenStale) {
						a.authPhase = miguAuthInvalid
						a.authNotice = "咪咕返回的令牌已失效，请重新扫码。"
						return
					}
					a.connectMiguToken(token, validationErr)
				})
				return
			}
		}
	}()
}

func (a *app) verifyMiguToken() {
	a.verifyMiguTokenValue(a.miguToken, true)
}

func (a *app) verifyMiguTokenValue(raw string, keepOnNetworkError bool) {
	token := normalizeMiguToken(raw)
	if token == "" {
		a.startMiguQRLogin()
		return
	}
	ctx, gen := a.beginMiguAuth()
	a.authPhase = miguAuthChecking
	a.authQR = nil
	a.authNotice = "正在检查咪咕登录状态…"
	a.refresh()

	go func() {
		checkCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		err := checkMiguToken(checkCtx, token)
		a.post(func() {
			if !a.finishMiguAuth(gen) {
				return
			}
			switch {
			case err == nil:
				if a.miguToken != token {
					a.miguProfile = nil
				}
				a.miguToken = token
				a.authPhase = miguAuthConnected
				a.authQR = nil
				a.closeLoginView()
				if saveMiguToken(token) != nil {
					a.authNotice = "咪咕账号已连接，但保存到系统安全存储失败。"
				} else {
					a.authNotice = "咪咕账号已连接。"
				}
				a.loadMiguProfile(token)
			case errors.Is(err, errMiguTokenStale):
				a.authPhase = miguAuthInvalid
				a.authQR = nil
				if keepOnNetworkError {
					a.miguToken = ""
					_ = saveMiguToken("")
					if a.authMode == authModeBrowser {
						a.startBrowserLogin()
						return
					}
					a.authNotice = "原来的登录状态已失效，请重新登录。"
				} else if a.miguToken != "" {
					a.authPhase = miguAuthConnected
					a.authNotice = "这个令牌无效，已保留原来的咪咕授权。"
				} else {
					a.authNotice = "这个 pacmtoken 无效或已经过期。"
				}
			case keepOnNetworkError:
				a.authPhase = miguAuthConnected
				a.authNotice = "暂时无法连接咪咕校验服务，仍保留本地登录状态。"
			case a.miguToken != "":
				a.authPhase = miguAuthConnected
				a.authNotice = "暂时无法验证新令牌，已保留原来的咪咕授权。"
			default:
				a.authPhase = miguAuthError
				a.authNotice = "网络校验失败：" + friendlyError(err)
			}
		})
	}()
}

func (a *app) importTokenFromClipboard() {
	raw := strings.TrimSpace(readClipboardText())
	token := normalizeMiguToken(raw)
	if token == "" {
		a.authNotice = "剪贴板里没有可用的 pacmtoken。"
		a.refresh()
		return
	}
	a.verifyTokenDraft(token)
}

func (a *app) updateMiguToken() {
	token := normalizeMiguToken(a.tokenDraft)
	if token == "" {
		a.authNotice = "请粘贴 pacmtoken，或粘贴包含 pacmtoken 的完整 Cookie。"
		a.refresh()
		return
	}
	a.verifyTokenDraft(token)
}

func (a *app) verifyTokenDraft(token string) {
	a.tokenDraft = ""
	a.verifyMiguTokenValue(token, false)
	a.toast = "正在验证咪咕授权"
}

func (a *app) clearMiguToken() {
	a.stopMiguAuth()
	a.closeLoginView()
	a.miguToken = ""
	a.tokenDraft = ""
	a.authQR = nil
	a.authPhase = miguAuthIdle
	a.authNotice = ""
	a.miguProfile = nil
	a.miguProfileErr = ""
	a.miguProfileLoading = false
	if err := saveMiguToken(""); err != nil {
		a.authNotice = "已清除当前登录状态，但系统凭据删除失败：" + friendlyError(err)
	} else {
		a.authNotice = "授权已清除，现在只播放公开可用的曲目。"
	}
	a.toast = "已清除咪咕授权"
	a.refresh()
}

func (a *app) cancelMiguQRLogin() {
	a.stopMiguAuth()
	a.authQR = nil
	if a.miguToken != "" {
		a.authPhase = miguAuthConnected
		a.authNotice = "已取消本次操作，原来的咪咕授权仍然保留。"
	} else {
		a.authPhase = miguAuthIdle
		a.authNotice = "已取消扫码，可以重新生成二维码或从剪贴板导入。"
	}
	a.refresh()
}

func (a *app) refreshMiguQRLogin() {
	a.startMiguQRLogin()
}
