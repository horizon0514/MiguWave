package main

import (
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *app) settingsPage(c *ui.Context, p palette) {
	pageScroll(c, "settings", 22, func() {
		a.pageHeader(c, p, "设置", "账号、菜单栏与桌面体验")

		ui.Column(c).FillWidth().MaxWidth(780).Gap(18).Children(func() {
			a.settingsAccountCard(c, p)
			a.settingsDesktopCard(c, p)
		})
	})
}

func (a *app) settingsAccountCard(c *ui.Context, p palette) {
	ui.Column(c).FillWidth().Padding(20).Radius(18).
		Background(p.panel).Border(1, p.line).Gap(16).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).Gap(3).Children(func() {
				ui.Text(c, "账号与授权").FontSize(16).FontWeight(710)
				ui.Text(c, "咪咕账号只用于读取有权限的曲库与会员权益").FontSize(11).TextColor(p.muted)
			})
			connected := a.miguToken != ""
			ui.Row(c).Gap(6).Padding(4, 8).Radius(999).
				Background(statusDot(p, connected).Alpha(0.13)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(5, 5).Radius(999).Background(statusDot(p, connected))
				label := "未连接"
				if connected {
					label = "已连接"
				}
				color := p.faint
				if connected {
					color = ui.Hex("#1b8e52")
				}
				ui.Text(c, label).FontSize(9.5).FontWeight(650).TextColor(color)
			})
		})

		ui.Row(c).Gap(14).AlignItems(ui.Center).Children(func() {
			a.accountAvatar(c, p, 52, 15)
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				title := "还没有连接咪咕账号"
				if a.miguToken != "" {
					title = "咪咕账号已连接"
					if a.miguProfile != nil && a.miguProfile.Nickname != "" {
						title = a.miguProfile.Nickname
					}
				}
				ui.Text(c, title).FontSize(14).FontWeight(680).SingleLine()
				subtitle := "当前只播放免费曲目"
				if a.miguToken != "" {
					subtitle = firstNonEmpty(a.miguProfileMeta(), "账号有权限的曲目可以直接播放")
				}
				ui.Text(c, subtitle).FontSize(11).TextColor(p.muted).SingleLine()
				if a.miguProfileLoading {
					ui.Text(c, "正在读取账号资料…").FontSize(10).TextColor(p.faint)
				} else if a.miguProfileErr != "" {
					ui.Text(c, "账号资料暂时无法读取").FontSize(10).TextColor(p.faint)
				}
			})
			if a.miguToken == "" {
				a.primaryButton(c, "连接咪咕账号", iconSpark, a.openAccount)
			} else {
				a.ghostButton(c, "管理账号", iconSpark, a.openAccount)
			}
		})
	})
}

func (a *app) settingsDesktopCard(c *ui.Context, p palette) {
	ui.Column(c).FillWidth().Padding(20).Radius(18).
		Background(p.panel).Border(1, p.line).Gap(16).Children(func() {
		ui.Column(c).Gap(3).Children(func() {
			ui.Text(c, "桌面与启动").FontSize(16).FontWeight(710)
			ui.Text(c, trayHint).
				FontSize(11).TextColor(p.muted)
		})

		a.settingsRow(c, p, iconClock, "开机启动",
			fmt.Sprintf("登录系统后在%s静默启动，不主动打开完整窗口。", trayLabel), func() {
				launch := ui.Switch(c.Key("launch-at-login"), &a.launchAtLogin).
					Label("开机启动")
				if launch.Changed() {
					a.applyLaunchAtLogin(a.launchAtLogin)
				}
			})

		ui.Divider(c)

		choice := 0
		if a.closeAction == closeActionQuit {
			choice = 1
		}
		a.settingsRow(c, p, iconDown, "关闭窗口时",
			fmt.Sprintf("%s默认只收起窗口；%s 始终退出应用。", closeButtonLabel, quitShortcutLabel), func() {
				seg := ui.Segmented(c, &choice, "隐藏到"+trayLabel, "退出应用").
					Width(300).Label("关闭窗口时的动作")
				if seg.Changed() {
					action := closeActionMenuBar
					if choice == 1 {
						action = closeActionQuit
					}
					a.setCloseAction(action)
				}
			})

		ui.Row(c).FillWidth().Padding(12, 11).Radius(12).
			Background(p.panelAlt).Gap(9).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconSpark).Size(14, 14).TextColor(p.accent)
			ui.Text(c, fmt.Sprintf("隐藏后音乐继续播放；重新打开完整窗口会恢复%s。", appIconLabel)).
				FontSize(10.5).TextColor(p.muted)
		})
	})
}

func (a *app) settingsRow(c *ui.Context, p palette, ic *ui.SVG, title, subtitle string, control func()) {
	ui.Row(c).FillWidth().Gap(13).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(38, 38).Radius(11).Background(p.accent.Alpha(0.1)).Center().Children(func() {
			ui.Icon(c, ic).Size(17, 17).TextColor(p.accent)
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
			ui.Text(c, title).FontSize(13).FontWeight(650)
			ui.Text(c, subtitle).FontSize(10.5).TextColor(p.muted).LineHeight(1.45)
		})
		control()
	})
}

func (a *app) setCloseAction(action closeAction) {
	action = normalizeCloseAction(action)
	if a.closeAction == action {
		return
	}
	a.closeAction = action
	a.save()
	a.refresh()
}

func (a *app) applyLaunchAtLogin(enabled bool) {
	if err := mygo.App.SetOpenAtLogin(enabled); err != nil {
		a.launchAtLogin = !enabled
		a.toast = fmt.Sprintf("开机启动设置失败：%s", friendlyError(err))
	} else if enabled {
		a.toast = "已启用开机启动"
	} else {
		a.toast = "已关闭开机启动"
	}
	a.refresh()
}
