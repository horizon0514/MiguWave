//go:build darwin

package app

// sidebarTopInset leaves room for the traffic lights in the hidden title bar.
const sidebarTopInset float32 = 46

const (
	trayLabel         = "菜单栏"
	trayHint          = "菜单栏会显示当前歌曲和歌词，点击图标即可展开"
	appIconLabel      = "Dock 图标"
	quitShortcutLabel = "⌘Q"
	closeButtonLabel  = "红色关闭按钮"
)
