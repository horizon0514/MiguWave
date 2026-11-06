//go:build !darwin && !windows

package main

// sidebarTopInset leaves room for the title bar buttons of a hidden title bar.
const sidebarTopInset float32 = 40

const (
	trayLabel         = "系统托盘"
	trayHint          = "系统托盘菜单可以控制播放，并打开完整窗口"
	appIconLabel      = "任务栏图标"
	quitShortcutLabel = "Ctrl+Q"
	closeButtonLabel  = "关闭按钮"
)
