//go:build windows

package main

// sidebarTopInset keeps the brand row clear of the caption buttons.
const sidebarTopInset float32 = 24

const (
	trayLabel         = "系统托盘"
	trayHint          = "系统托盘会显示当前歌曲和歌词，点击图标即可展开"
	appIconLabel      = "任务栏图标"
	quitShortcutLabel = "Ctrl+Q"
	closeButtonLabel  = "关闭按钮"
)
