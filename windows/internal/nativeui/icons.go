package nativeui

import (
	"encoding/base64"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Exact vector icons from windows/resources/workspace/*.svg adapted with
// currentColor so ui.Icon tints them cleanly for both Light and Dark themes.
var (
	iconPlanet = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256" viewBox="0 0 256 256"><path fill="currentColor" d="M64 32H144A32 32 0 0 1 176 64V80H96A16 16 0 0 0 80 96V144H64A32 32 0 0 1 32 112V64A32 32 0 0 1 64 32Z"/><path fill="currentColor" d="M64 32H144A32 32 0 0 1 176 64V80H96A16 16 0 0 0 80 96V144H64A32 32 0 0 1 32 112V64A32 32 0 0 1 64 32Z" transform="rotate(180 128 128)"/></svg>`))

	iconGrid = ui.MustParseSVG([]byte(`<svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M3 2.25H6.75C7.16421 2.25 7.5 2.58579 7.5 3V6.75C7.5 7.16421 7.16421 7.5 6.75 7.5H3C2.58579 7.5 2.25 7.16421 2.25 6.75V3C2.25 2.58579 2.58579 2.25 3 2.25Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M11.25 2.25H15C15.4142 2.25 15.75 2.58579 15.75 3V6.75C15.75 7.16421 15.4142 7.5 15 7.5H11.25C10.8358 7.5 10.5 7.16421 10.5 6.75V3C10.5 2.58579 10.8358 2.25 11.25 2.25Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M3 10.5H6.75C7.16421 10.5 7.5 10.8358 7.5 11.25V15C7.5 15.4142 7.16421 15.75 6.75 15.75H3C2.58579 15.75 2.25 15.4142 2.25 15V11.25C2.25 10.8358 2.58579 10.5 3 10.5Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M11.25 10.5H15C15.4142 10.5 15.75 10.8358 15.75 11.25V15C15.75 15.4142 15.4142 15.75 15 15.75H11.25C10.8358 15.75 10.5 15.4142 10.5 15V11.25C10.5 10.8358 10.8358 10.5 11.25 10.5Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))

	iconClock = ui.MustParseSVG([]byte(`<svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M15.75 9C15.75 12.7279 12.7279 15.75 9 15.75C5.27208 15.75 2.25 12.7279 2.25 9C2.25 5.27208 5.27208 2.25 9 2.25C12.7279 2.25 15.75 5.27208 15.75 9Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M9 5.25V9L11.25 10.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))

	iconInbox = ui.MustParseSVG([]byte(`<svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M3 3H15V15H3V3Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M3 9.75H6.75L8.25 12H9.75L11.25 9.75H15" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))

	iconStar = ui.MustParseSVG([]byte(`<svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M9 2.25L11.1 6.525L15.75 7.2L12.375 10.5L13.2 15.15L9 12.975L4.8 15.15L5.625 10.5L2.25 7.2L6.9 6.525L9 2.25Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))

	iconGlobe = ui.MustParseSVG([]byte(`<svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><circle cx="9" cy="9" r="6.75" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><ellipse cx="9" cy="9" rx="3" ry="6.75" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M2.6 6.75H15.4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M2.6 11.25H15.4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))

	iconPlus = ui.MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M8 3.33333V12.6667M3.33333 8H12.6667" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))

	iconSettings = ui.MustParseSVG([]byte(`<svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M3 4.5H15M3 9H15M3 13.5H15" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><circle cx="6" cy="4.5" r="1.5" fill="currentColor"/><circle cx="12" cy="9" r="1.5" fill="currentColor"/><circle cx="7.5" cy="13.5" r="1.5" fill="currentColor"/></svg>`))

	iconEdit = ui.MustParseSVG([]byte(`<svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M11.25 3L15 6.75M14.625 6.375L6.75 14.25L3 15V10.5L12 1.5C13.0355 0.464466 14.7145 0.464466 15.75 1.5C16.7855 2.53553 16.7855 4.21447 15.75 5.25L14.625 6.375Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))

	iconFile = ui.MustParseSVG([]byte(`<svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M10.5 2.25H3.75V15.75H14.25V6L10.5 2.25Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M10.5 2.25V6H14.25M6 9H12M6 12H10.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))

	iconArrow = ui.MustParseSVG([]byte(`<svg width="14" height="14" viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M5.25 2.91667L9.33333 7L5.25 11.0833" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))

	iconMore = ui.MustParseSVG([]byte(`<svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><circle cx="3.75" cy="9" r="1" fill="currentColor"/><circle cx="9" cy="9" r="1" fill="currentColor"/><circle cx="14.25" cy="9" r="1" fill="currentColor"/></svg>`))

	iconSearch = ui.MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg"><circle cx="7" cy="7" r="4.5" stroke="currentColor" stroke-width="1.5"/><path d="M10.5 10.5L13.5 13.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`))

	iconSidebar = ui.MustParseSVG([]byte(`<svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M3 4.5H15M3 9H15M3 13.5H15" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`))

	iconChevronDown = ui.MustParseSVG([]byte(`<svg width="14" height="14" viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M3.5 5.25L7 8.75L10.5 5.25" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))

	// Solid macOS Smart Feed icons matching SF Symbols (sun.max.fill, circle.inset.filled, star.fill).
	iconSunSolid = ui.MustParseSVG([]byte(`<svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><circle cx="9" cy="9" r="3.6" fill="currentColor"/><path d="M9 1.6V3.5M9 14.5V16.4M1.6 9H3.5M14.5 9H16.4M3.77 3.77L5.11 5.11M12.89 12.89L14.23 14.23M14.23 3.77L12.89 5.11M5.11 12.89L3.77 14.23" stroke="currentColor" stroke-width="1.65" stroke-linecap="round"/></svg>`))

	iconCircleInsetSolid = ui.MustParseSVG([]byte(`<svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><circle cx="9" cy="9" r="6.75" stroke="currentColor" stroke-width="1.65"/><circle cx="9" cy="9" r="4.1" fill="currentColor"/></svg>`))

	iconStarSolid = ui.MustParseSVG([]byte(`<svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M9 1.9L11.2 6.35L16.1 7.06L12.55 10.52L13.39 15.4L9 13.09L4.61 15.4L5.45 10.52L1.9 7.06L6.8 6.35L9 1.9Z" fill="currentColor"/></svg>`))
)

// bitmapForDataURL decodes and caches base64 PNG strings (site avatars and
// subscription favicons) so they render natively via ui.Image.
func (a *App) bitmapForDataURL(raw string) *ui.Bitmap {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if a.bitmapCache == nil {
		a.bitmapCache = map[string]*ui.Bitmap{}
	}
	if bmp, ok := a.bitmapCache[raw]; ok {
		return bmp
	}
	payload := raw
	if idx := strings.Index(payload, ","); idx >= 0 && strings.HasPrefix(payload, "data:") {
		payload = payload[idx+1:]
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			a.bitmapCache[raw] = nil
			return nil
		}
	}
	bmp, err := ui.DecodeBitmapScaled(decoded, 64)
	if err != nil {
		a.bitmapCache[raw] = nil
		return nil
	}
	a.bitmapCache[raw] = bmp
	return bmp
}
