package main

import "github.com/egoist/mygo/ui"

func icon(svg string) *ui.SVG {
	return ui.MustParseSVG([]byte(svg))
}

var (
	iconHome       = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="m3 10l9-7l9 7v10a1 1 0 0 1-1 1h-5v-7H9v7H4a1 1 0 0 1-1-1Z"/></svg>`)
	iconRadio      = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="M5 16a7 7 0 0 1 14 0M8.5 16a3.5 3.5 0 0 1 7 0M12 16h.01M4.5 6.5l15 11"/></svg>`)
	iconGlobe      = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="M12 21a9 9 0 1 0 0-18a9 9 0 0 0 0 18Zm0-18c2.4 2.4 3.6 5.4 3.6 9S14.4 18.6 12 21m0-18C9.6 5.4 8.4 8.4 8.4 12s1.2 6.6 3.6 9M3.5 9h17m-17 6h17"/></svg>`)
	iconHeart      = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="M20.8 4.6a5.5 5.5 0 0 0-7.8 0L12 5.7l-1.1-1.1a5.5 5.5 0 0 0-7.8 7.8l1.1 1.1L12 21l7.8-7.5l1.1-1.1a5.5 5.5 0 0 0-.1-7.8Z"/></svg>`)
	iconHeartFill  = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="currentColor" d="M20.8 4.6a5.5 5.5 0 0 0-7.8 0L12 5.7l-1.1-1.1a5.5 5.5 0 0 0-7.8 7.8l1.1 1.1L12 21l7.8-7.5l1.1-1.1a5.5 5.5 0 0 0-.1-7.8Z"/></svg>`)
	iconClock      = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="M12 8v4l3 2m-3-11a9 9 0 1 0 0 18a9 9 0 0 0 0-18Z"/></svg>`)
	iconMusic      = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="M9 18V5l12-2v13M9 18a3 3 0 1 1-6 0a3 3 0 0 1 6 0Zm12-2a3 3 0 1 1-6 0a3 3 0 0 1 6 0Z"/></svg>`)
	iconPlay       = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="currentColor" d="M8 5.6v12.8c0 .8.9 1.3 1.6.9l10-6.4a1 1 0 0 0 0-1.8l-10-6.4A1 1 0 0 0 8 5.6Z"/></svg>`)
	iconPause      = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="currentColor" d="M7 4h3v16H7a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1Zm7 0h3a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1h-3Z"/></svg>`)
	iconPrev       = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="currentColor" d="M6 5a1 1 0 0 1 2 0v5.2l8.7-5.6A1 1 0 0 1 18 5.5v13a1 1 0 0 1-1.3.9L8 13.8V19a1 1 0 0 1-2 0Z"/></svg>`)
	iconNext       = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="currentColor" d="M18 5a1 1 0 0 0-2 0v5.2L7.3 4.6A1 1 0 0 0 6 5.5v13a1 1 0 0 0 1.3.9L16 13.8V19a1 1 0 0 0 2 0Z"/></svg>`)
	iconShuffle    = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="M16 3h5v5M4 20L21 3M21 16v5h-5M15 15l6 6M4 4l5 5"/></svg>`)
	iconRepeat     = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="m17 2l4 4l-4 4M3 11V9a3 3 0 0 1 3-3h15M7 22l-4-4l4-4m14-1v2a3 3 0 0 1-3 3H3"/></svg>`)
	iconRepeatOne  = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="m17 2l4 4l-4 4M3 11V9a3 3 0 0 1 3-3h15M7 22l-4-4l4-4m14-1v2a3 3 0 0 1-3 3H3m8-7h1v4"/></svg>`)
	iconVolume     = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="M11 5L6 9H2v6h4l5 4Zm4.5 3.5a5 5 0 0 1 0 7M18.5 6a9 9 0 0 1 0 12"/></svg>`)
	iconVolumeOff  = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="M11 5L6 9H2v6h4l5 4Zm6 4l4 4m0-4l-4 4"/></svg>`)
	iconBack       = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="m15 18l-6-6l6-6"/></svg>`)
	iconSearchIcon = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-width="1.8" d="M14.5 14.5L20 20m-4-10a6 6 0 1 1-12 0a6 6 0 0 1 12 0Z"/></svg>`)
	iconSpark      = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.7" d="M12 3l1.3 4.2a6 6 0 0 0 3.5 3.5L21 12l-4.2 1.3a6 6 0 0 0-3.5 3.5L12 21l-1.3-4.2a6 6 0 0 0-3.5-3.5L3 12l4.2-1.3a6 6 0 0 0 3.5-3.5Z"/></svg>`)
	iconArrow      = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="M5 12h14m-5-5l5 5l-5 5"/></svg>`)
	iconExternal   = icon(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="M14 4h6v6m0-6L10 14M19 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h5"/></svg>`)
)
