package nativeui

import (
	"fmt"
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"windows/internal/store"
)

const (
	badgeTrayHeight = float32(76)
	badgeTileSize   = float32(46)
)

type badgeVisualSpec struct {
	defaultTilt float32
	defaultPos  ui.Point
	textTopPad  float32
	shortTag    string
	lightSVG    *ui.SVG
	darkSVG     *ui.SVG
	lockedLight *ui.SVG
	lockedDark  *ui.SVG
	lightInk    ui.Color
	darkInk     ui.Color
	lightAccent ui.Color
	darkAccent  ui.Color
}

var (
	// 1. Scalloped flower seal (Total Active Days) — muted slate-lavender enamel.
	svgBadgeFlowerLight = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><defs><linearGradient id="gfl" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="#f3f1fa"/><stop offset="100%" stop-color="#e2dff2"/></linearGradient></defs><path d="M28 4.2C30.4 4.2 32.1 6.1 34.3 6.8C36.6 7.5 39.1 6.7 41 8.1C42.9 9.5 43.2 12.1 44.6 14C46 15.9 48.6 16.7 49.3 19C50 21.2 48.8 23.6 48.8 26C48.8 28.4 50 30.8 49.3 33C48.6 35.3 46 36.1 44.6 38C43.2 39.9 42.9 42.5 41 43.9C39.1 45.3 36.6 44.5 34.3 45.2C32.1 45.9 30.4 47.8 28 47.8C25.6 47.8 23.9 45.9 21.7 45.2C19.4 44.5 16.9 45.3 15 43.9C13.1 42.5 12.8 39.9 11.4 38C10 36.1 7.4 35.3 6.7 33C6 30.8 7.2 28.4 7.2 26C7.2 23.6 6 21.2 6.7 19C7.4 16.7 10 15.9 11.4 14C12.8 12.1 13.1 9.5 15 8.1C16.9 6.7 19.4 7.5 21.7 6.8C23.9 6.1 25.6 4.2 28 4.2Z" transform="translate(0 3.8)" fill="#0f172a" opacity="0.09"/><path d="M28 4.2C30.4 4.2 32.1 6.1 34.3 6.8C36.6 7.5 39.1 6.7 41 8.1C42.9 9.5 43.2 12.1 44.6 14C46 15.9 48.6 16.7 49.3 19C50 21.2 48.8 23.6 48.8 26C48.8 28.4 50 30.8 49.3 33C48.6 35.3 46 36.1 44.6 38C43.2 39.9 42.9 42.5 41 43.9C39.1 45.3 36.6 44.5 34.3 45.2C32.1 45.9 30.4 47.8 28 47.8C25.6 47.8 23.9 45.9 21.7 45.2C19.4 44.5 16.9 45.3 15 43.9C13.1 42.5 12.8 39.9 11.4 38C10 36.1 7.4 35.3 6.7 33C6 30.8 7.2 28.4 7.2 26C7.2 23.6 6 21.2 6.7 19C7.4 16.7 10 15.9 11.4 14C12.8 12.1 13.1 9.5 15 8.1C16.9 6.7 19.4 7.5 21.7 6.8C23.9 6.1 25.6 4.2 28 4.2Z" transform="translate(0 2)" fill="url(#gfl)" stroke="#ffffff" stroke-width="2" stroke-linejoin="round"/><circle cx="28" cy="28" r="16.5" fill="none" stroke="#c3bce2" stroke-width="0.9" stroke-dasharray="2 2"/></svg>`))
	svgBadgeFlowerDark  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><defs><linearGradient id="gfd" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="#312f44"/><stop offset="100%" stop-color="#242333"/></linearGradient></defs><path d="M28 4.2C30.4 4.2 32.1 6.1 34.3 6.8C36.6 7.5 39.1 6.7 41 8.1C42.9 9.5 43.2 12.1 44.6 14C46 15.9 48.6 16.7 49.3 19C50 21.2 48.8 23.6 48.8 26C48.8 28.4 50 30.8 49.3 33C48.6 35.3 46 36.1 44.6 38C43.2 39.9 42.9 42.5 41 43.9C39.1 45.3 36.6 44.5 34.3 45.2C32.1 45.9 30.4 47.8 28 47.8C25.6 47.8 23.9 45.9 21.7 45.2C19.4 44.5 16.9 45.3 15 43.9C13.1 42.5 12.8 39.9 11.4 38C10 36.1 7.4 35.3 6.7 33C6 30.8 7.2 28.4 7.2 26C7.2 23.6 6 21.2 6.7 19C7.4 16.7 10 15.9 11.4 14C12.8 12.1 13.1 9.5 15 8.1C16.9 6.7 19.4 7.5 21.7 6.8C23.9 6.1 25.6 4.2 28 4.2Z" transform="translate(0 4)" fill="#000000" opacity="0.34"/><path d="M28 4.2C30.4 4.2 32.1 6.1 34.3 6.8C36.6 7.5 39.1 6.7 41 8.1C42.9 9.5 43.2 12.1 44.6 14C46 15.9 48.6 16.7 49.3 19C50 21.2 48.8 23.6 48.8 26C48.8 28.4 50 30.8 49.3 33C48.6 35.3 46 36.1 44.6 38C43.2 39.9 42.9 42.5 41 43.9C39.1 45.3 36.6 44.5 34.3 45.2C32.1 45.9 30.4 47.8 28 47.8C25.6 47.8 23.9 45.9 21.7 45.2C19.4 44.5 16.9 45.3 15 43.9C13.1 42.5 12.8 39.9 11.4 38C10 36.1 7.4 35.3 6.7 33C6 30.8 7.2 28.4 7.2 26C7.2 23.6 6 21.2 6.7 19C7.4 16.7 10 15.9 11.4 14C12.8 12.1 13.1 9.5 15 8.1C16.9 6.7 19.4 7.5 21.7 6.8C23.9 6.1 25.6 4.2 28 4.2Z" transform="translate(0 2)" fill="url(#gfd)" stroke="rgba(255,255,255,0.18)" stroke-width="1.5" stroke-linejoin="round"/><circle cx="28" cy="28" r="16.5" fill="none" stroke="#5c5686" stroke-width="0.9" stroke-dasharray="2 2"/></svg>`))

	// 2. Soft wavy crest (Writing Streak) — muted sage-celadon enamel.
	svgBadgeStreakLight = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><defs><linearGradient id="gsl" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="#edf6f1"/><stop offset="100%" stop-color="#d9ebe1"/></linearGradient></defs><path d="M28 5.5L34.5 8.2L41.5 9.6L45.1 15.6L49.5 21.2L48.7 28L49.5 34.8L45.1 40.4L41.5 46.4L34.5 47.8L28 50.5L21.5 47.8L14.5 46.4L10.9 40.4L6.5 34.8L7.3 28L6.5 21.2L10.9 15.6L14.5 9.6L21.5 8.2Z" transform="translate(0 1.8)" fill="#0f172a" opacity="0.09"/><path d="M28 5.5L34.5 8.2L41.5 9.6L45.1 15.6L49.5 21.2L48.7 28L49.5 34.8L45.1 40.4L41.5 46.4L34.5 47.8L28 50.5L21.5 47.8L14.5 46.4L10.9 40.4L6.5 34.8L7.3 28L6.5 21.2L10.9 15.6L14.5 9.6L21.5 8.2Z" fill="url(#gsl)" stroke="#ffffff" stroke-width="2" stroke-linejoin="round"/><circle cx="28" cy="28" r="16.2" fill="none" stroke="#a9ccb9" stroke-width="0.9"/></svg>`))
	svgBadgeStreakDark  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><defs><linearGradient id="gsd" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="#25362e"/><stop offset="100%" stop-color="#1d2a24"/></linearGradient></defs><path d="M28 5.5L34.5 8.2L41.5 9.6L45.1 15.6L49.5 21.2L48.7 28L49.5 34.8L45.1 40.4L41.5 46.4L34.5 47.8L28 50.5L21.5 47.8L14.5 46.4L10.9 40.4L6.5 34.8L7.3 28L6.5 21.2L10.9 15.6L14.5 9.6L21.5 8.2Z" transform="translate(0 2)" fill="#000000" opacity="0.34"/><path d="M28 5.5L34.5 8.2L41.5 9.6L45.1 15.6L49.5 21.2L48.7 28L49.5 34.8L45.1 40.4L41.5 46.4L34.5 47.8L28 50.5L21.5 47.8L14.5 46.4L10.9 40.4L6.5 34.8L7.3 28L6.5 21.2L10.9 15.6L14.5 9.6L21.5 8.2Z" fill="url(#gsd)" stroke="rgba(255,255,255,0.18)" stroke-width="1.5" stroke-linejoin="round"/><circle cx="28" cy="28" r="16.2" fill="none" stroke="#456e59" stroke-width="0.9"/></svg>`))

	// 3. Rounded triangle stamp (Published Articles) — muted dusty-rose enamel.
	svgBadgeTriLight = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><defs><linearGradient id="gtl" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="#faf0f2"/><stop offset="100%" stop-color="#eedde1"/></linearGradient></defs><path d="M24.2 9.4C25.9 6.4 30.1 6.4 31.8 9.4L48.5 38.6C50.2 41.6 48.1 45.3 44.6 45.3H11.4C7.9 45.3 5.8 41.6 7.5 38.6Z" transform="translate(0 1.8)" fill="#0f172a" opacity="0.09"/><path d="M24.2 9.4C25.9 6.4 30.1 6.4 31.8 9.4L48.5 38.6C50.2 41.6 48.1 45.3 44.6 45.3H11.4C7.9 45.3 5.8 41.6 7.5 38.6Z" fill="url(#gtl)" stroke="#ffffff" stroke-width="2" stroke-linejoin="round"/><path d="M25.6 13.6C26.7 11.7 29.3 11.7 30.4 13.6L43.9 37.3C45 39.2 43.6 41.5 41.5 41.5H14.5C12.4 41.5 11 39.2 12.1 37.3Z" fill="none" stroke="#d8b5bf" stroke-width="0.9"/></svg>`))
	svgBadgeTriDark  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><defs><linearGradient id="gtd" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="#3a2a30"/><stop offset="100%" stop-color="#2c2025"/></linearGradient></defs><path d="M24.2 9.4C25.9 6.4 30.1 6.4 31.8 9.4L48.5 38.6C50.2 41.6 48.1 45.3 44.6 45.3H11.4C7.9 45.3 5.8 41.6 7.5 38.6Z" transform="translate(0 2)" fill="#000000" opacity="0.34"/><path d="M24.2 9.4C25.9 6.4 30.1 6.4 31.8 9.4L48.5 38.6C50.2 41.6 48.1 45.3 44.6 45.3H11.4C7.9 45.3 5.8 41.6 7.5 38.6Z" fill="url(#gtd)" stroke="rgba(255,255,255,0.18)" stroke-width="1.5" stroke-linejoin="round"/><path d="M25.6 13.6C26.7 11.7 29.3 11.7 30.4 13.6L43.9 37.3C45 39.2 43.6 41.5 41.5 41.5H14.5C12.4 41.5 11 39.2 12.1 37.3Z" fill="none" stroke="#6f4c58" stroke-width="0.9"/></svg>`))

	// 4. Brushed pewter medallion coin (Published Planet Sites).
	svgBadgeCoinLight = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><defs><linearGradient id="gcl" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="#f6f7f9"/><stop offset="52%" stop-color="#e8ebf0"/><stop offset="100%" stop-color="#dfe3eb"/></linearGradient></defs><circle cx="28" cy="29.8" r="21.5" fill="#0f172a" opacity="0.09"/><circle cx="28" cy="28" r="21.5" fill="url(#gcl)" stroke="#ffffff" stroke-width="2"/><circle cx="28" cy="28" r="17.2" fill="none" stroke="#c7cdd9" stroke-width="0.9"/><circle cx="28" cy="28" r="14.2" fill="none" stroke="#d8dde6" stroke-width="0.7" stroke-dasharray="1.5 2"/></svg>`))
	svgBadgeCoinDark  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><defs><linearGradient id="gcd" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="#363942"/><stop offset="100%" stop-color="#262830"/></linearGradient></defs><circle cx="28" cy="30" r="21.5" fill="#000000" opacity="0.34"/><circle cx="28" cy="28" r="21.5" fill="url(#gcd)" stroke="rgba(255,255,255,0.18)" stroke-width="1.5"/><circle cx="28" cy="28" r="17.2" fill="none" stroke="#585e6e" stroke-width="0.9"/><circle cx="28" cy="28" r="14.2" fill="none" stroke="#454a57" stroke-width="0.7" stroke-dasharray="1.5 2"/></svg>`))

	// Shape-matched frosted silhouettes for locked badges in the Achievement Wall.
	svgBadgeFlowerLockedLight = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><path d="M28 4.2C30.4 4.2 32.1 6.1 34.3 6.8C36.6 7.5 39.1 6.7 41 8.1C42.9 9.5 43.2 12.1 44.6 14C46 15.9 48.6 16.7 49.3 19C50 21.2 48.8 23.6 48.8 26C48.8 28.4 50 30.8 49.3 33C48.6 35.3 46 36.1 44.6 38C43.2 39.9 42.9 42.5 41 43.9C39.1 45.3 36.6 44.5 34.3 45.2C32.1 45.9 30.4 47.8 28 47.8C25.6 47.8 23.9 45.9 21.7 45.2C19.4 44.5 16.9 45.3 15 43.9C13.1 42.5 12.8 39.9 11.4 38C10 36.1 7.4 35.3 6.7 33C6 30.8 7.2 28.4 7.2 26C7.2 23.6 6 21.2 6.7 19C7.4 16.7 10 15.9 11.4 14C12.8 12.1 13.1 9.5 15 8.1C16.9 6.7 19.4 7.5 21.7 6.8C23.9 6.1 25.6 4.2 28 4.2Z" transform="translate(0 2)" fill="#f3f4f6" stroke="#d4d8e1" stroke-width="1.2" stroke-dasharray="3 2.5" stroke-linejoin="round"/></svg>`))
	svgBadgeFlowerLockedDark  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><path d="M28 4.2C30.4 4.2 32.1 6.1 34.3 6.8C36.6 7.5 39.1 6.7 41 8.1C42.9 9.5 43.2 12.1 44.6 14C46 15.9 48.6 16.7 49.3 19C50 21.2 48.8 23.6 48.8 26C48.8 28.4 50 30.8 49.3 33C48.6 35.3 46 36.1 44.6 38C43.2 39.9 42.9 42.5 41 43.9C39.1 45.3 36.6 44.5 34.3 45.2C32.1 45.9 30.4 47.8 28 47.8C25.6 47.8 23.9 45.9 21.7 45.2C19.4 44.5 16.9 45.3 15 43.9C13.1 42.5 12.8 39.9 11.4 38C10 36.1 7.4 35.3 6.7 33C6 30.8 7.2 28.4 7.2 26C7.2 23.6 6 21.2 6.7 19C7.4 16.7 10 15.9 11.4 14C12.8 12.1 13.1 9.5 15 8.1C16.9 6.7 19.4 7.5 21.7 6.8C23.9 6.1 25.6 4.2 28 4.2Z" transform="translate(0 2)" fill="#23252a" stroke="rgba(255,255,255,0.14)" stroke-width="1.2" stroke-dasharray="3 2.5" stroke-linejoin="round"/></svg>`))

	svgBadgeStreakLockedLight = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><path d="M28 5.5L34.5 8.2L41.5 9.6L45.1 15.6L49.5 21.2L48.7 28L49.5 34.8L45.1 40.4L41.5 46.4L34.5 47.8L28 50.5L21.5 47.8L14.5 46.4L10.9 40.4L6.5 34.8L7.3 28L6.5 21.2L10.9 15.6L14.5 9.6L21.5 8.2Z" fill="#f3f4f6" stroke="#d4d8e1" stroke-width="1.2" stroke-dasharray="3 2.5" stroke-linejoin="round"/></svg>`))
	svgBadgeStreakLockedDark  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><path d="M28 5.5L34.5 8.2L41.5 9.6L45.1 15.6L49.5 21.2L48.7 28L49.5 34.8L45.1 40.4L41.5 46.4L34.5 47.8L28 50.5L21.5 47.8L14.5 46.4L10.9 40.4L6.5 34.8L7.3 28L6.5 21.2L10.9 15.6L14.5 9.6L21.5 8.2Z" fill="#23252a" stroke="rgba(255,255,255,0.14)" stroke-width="1.2" stroke-dasharray="3 2.5" stroke-linejoin="round"/></svg>`))

	svgBadgeTriLockedLight = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><path d="M24.2 9.4C25.9 6.4 30.1 6.4 31.8 9.4L48.5 38.6C50.2 41.6 48.1 45.3 44.6 45.3H11.4C7.9 45.3 5.8 41.6 7.5 38.6Z" fill="#f3f4f6" stroke="#d4d8e1" stroke-width="1.2" stroke-dasharray="3 2.5" stroke-linejoin="round"/></svg>`))
	svgBadgeTriLockedDark  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><path d="M24.2 9.4C25.9 6.4 30.1 6.4 31.8 9.4L48.5 38.6C50.2 41.6 48.1 45.3 44.6 45.3H11.4C7.9 45.3 5.8 41.6 7.5 38.6Z" fill="#23252a" stroke="rgba(255,255,255,0.14)" stroke-width="1.2" stroke-dasharray="3 2.5" stroke-linejoin="round"/></svg>`))

	svgBadgeCoinLockedLight = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><circle cx="28" cy="28" r="21" fill="#f3f4f6" stroke="#d4d8e1" stroke-width="1.2" stroke-dasharray="3 2.5"/><circle cx="28" cy="28" r="15.5" fill="none" stroke="#e2e5ec" stroke-width="0.8"/></svg>`))
	svgBadgeCoinLockedDark  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="56" height="56" viewBox="0 0 56 56"><circle cx="28" cy="28" r="21" fill="#23252a" stroke="rgba(255,255,255,0.14)" stroke-width="1.2" stroke-dasharray="3 2.5"/><circle cx="28" cy="28" r="15.5" fill="none" stroke="rgba(255,255,255,0.06)" stroke-width="0.8"/></svg>`))
)

func badgeVisualFor(kind store.WritingBadgeKind) badgeVisualSpec {
	switch kind {
	case store.BadgeKindActiveDays:
		return badgeVisualSpec{
			defaultTilt: -6,
			defaultPos:  ui.Point{X: 8, Y: 10},
			shortTag:    "DAYS",
			lightSVG:    svgBadgeFlowerLight,
			darkSVG:     svgBadgeFlowerDark,
			lockedLight: svgBadgeFlowerLockedLight,
			lockedDark:  svgBadgeFlowerLockedDark,
			lightInk:    ui.Hex("#4c4375"),
			darkInk:     ui.Hex("#cac4ed"),
			lightAccent: ui.Hex("#6d62a8"),
			darkAccent:  ui.Hex("#9c93d4"),
		}
	case store.BadgeKindStreak:
		return badgeVisualSpec{
			defaultTilt: 5,
			defaultPos:  ui.Point{X: 62, Y: 20},
			shortTag:    "STREAK",
			lightSVG:    svgBadgeStreakLight,
			darkSVG:     svgBadgeStreakDark,
			lockedLight: svgBadgeStreakLockedLight,
			lockedDark:  svgBadgeStreakLockedDark,
			lightInk:    ui.Hex("#2d5e46"),
			darkInk:     ui.Hex("#a6d6be"),
			lightAccent: ui.Hex("#478567"),
			darkAccent:  ui.Hex("#78b899"),
		}
	case store.BadgeKindPublished:
		return badgeVisualSpec{
			defaultTilt: -5,
			defaultPos:  ui.Point{X: 118, Y: 9},
			textTopPad:  6,
			shortTag:    "POSTS",
			lightSVG:    svgBadgeTriLight,
			darkSVG:     svgBadgeTriDark,
			lockedLight: svgBadgeTriLockedLight,
			lockedDark:  svgBadgeTriLockedDark,
			lightInk:    ui.Hex("#7d4655"),
			darkInk:     ui.Hex("#e3b6c3"),
			lightAccent: ui.Hex("#a66375"),
			darkAccent:  ui.Hex("#c98b9d"),
		}
	default:
		return badgeVisualSpec{
			defaultTilt: 4,
			defaultPos:  ui.Point{X: 174, Y: 18},
			shortTag:    "PLANET",
			lightSVG:    svgBadgeCoinLight,
			darkSVG:     svgBadgeCoinDark,
			lockedLight: svgBadgeCoinLockedLight,
			lockedDark:  svgBadgeCoinLockedDark,
			lightInk:    ui.Hex("#454e61"),
			darkInk:     ui.Hex("#cfd6e3"),
			lightAccent: ui.Hex("#68738a"),
			darkAccent:  ui.Hex("#9aa4b8"),
		}
	}
}

func (a *App) writingBadges() []store.WritingBadge {
	return store.ComputeWritingBadges(a.activity, a.dashboard)
}

func (a *App) badgeTitle(kind store.WritingBadgeKind) string {
	switch kind {
	case store.BadgeKindActiveDays:
		return a.t("dashboard.badge.activeDays.title")
	case store.BadgeKindStreak:
		return a.t("dashboard.badge.streak.title")
	case store.BadgeKindPublished:
		return a.t("dashboard.badge.published.title")
	default:
		return a.t("dashboard.badge.planet.title")
	}
}

func (a *App) badgeStatText(b store.WritingBadge) string {
	switch b.Kind {
	case store.BadgeKindActiveDays:
		return a.tf("dashboard.badge.activeDays.stat", b.Value)
	case store.BadgeKindStreak:
		return a.tf("dashboard.badge.streak.stat", b.Value)
	case store.BadgeKindPublished:
		return a.tf("dashboard.badge.published.stat", b.Value)
	default:
		return a.tf("dashboard.badge.planet.stat", b.Value)
	}
}

func (a *App) badgeTooltip(b store.WritingBadge) string {
	title := a.badgeTitle(b.Kind)
	stat := a.badgeStatText(b)
	if b.NextGoal > 0 {
		return fmt.Sprintf("%s · %s (%d/%d) — %s", title, stat, b.Value, b.NextGoal, a.t("dashboard.badge.dragHint"))
	}
	return fmt.Sprintf("%s · %s — %s", title, stat, a.t("dashboard.badge.dragHint"))
}

func clampBadgePoint(pt ui.Point, maxX, maxY float32) ui.Point {
	if maxX < 0 {
		maxX = 0
	}
	if maxY < 0 {
		maxY = 0
	}
	if pt.X < 0 {
		pt.X = 0
	} else if pt.X > maxX {
		pt.X = maxX
	}
	if pt.Y < 0 {
		pt.Y = 0
	} else if pt.Y > maxY {
		pt.Y = maxY
	}
	return pt
}

func badgePointsNear(a, b ui.Point) bool {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx >= -1 && dx <= 1 && dy >= -1 && dy <= 1
}

func (a *App) ensureBadgeDragOffset(b store.WritingBadge, maxX, maxY float32) *ui.Point {
	if a.badgeDragOffsets == nil {
		a.badgeDragOffsets = make(map[string]*ui.Point, 4)
	}
	pt, ok := a.badgeDragOffsets[b.ID]
	if !ok || pt == nil {
		initial := badgeVisualFor(b.Kind).defaultPos
		if saved, hasSaved := a.badgePositions[b.ID]; hasSaved {
			initial = saved
		}
		clamped := clampBadgePoint(initial, maxX, maxY)
		pt = &clamped
		a.badgeDragOffsets[b.ID] = pt
		return pt
	}
	if a.badgeDraggingID != b.ID {
		*pt = clampBadgePoint(*pt, maxX, maxY)
	}
	return pt
}

func (a *App) commitBadgePosition(b store.WritingBadge, live ui.Point, maxX, maxY float32) {
	clamped := clampBadgePoint(live, maxX, maxY)
	def := clampBadgePoint(badgeVisualFor(b.Kind).defaultPos, maxX, maxY)
	if badgePointsNear(clamped, def) {
		delete(a.badgePositions, b.ID)
		return
	}
	if a.badgePositions == nil {
		a.badgePositions = make(map[string]ui.Point, 4)
	}
	a.badgePositions[b.ID] = clamped
}

func (a *App) bringBadgeToFront(id string) {
	if idx := slices.Index(a.badgeZOrder, id); idx >= 0 {
		a.badgeZOrder = slices.Delete(a.badgeZOrder, idx, idx+1)
	}
	a.badgeZOrder = append(a.badgeZOrder, id)
}

func (a *App) badgeZIndex(id string, dragging bool) int {
	if dragging {
		return 40
	}
	if idx := slices.Index(a.badgeZOrder, id); idx >= 0 {
		return 10 + idx
	}
	return 5
}

func (a *App) hasCustomBadgePositions() bool {
	if len(a.badgePositions) == 0 {
		return false
	}
	for _, b := range a.writingBadges() {
		pt, ok := a.badgePositions[b.ID]
		if !ok {
			continue
		}
		if !badgePointsNear(pt, badgeVisualFor(b.Kind).defaultPos) {
			return true
		}
	}
	return false
}

func (a *App) resetBadgePositions() {
	a.badgePositions = nil
	a.badgeDragOffsets = nil
	a.badgeZOrder = nil
	a.badgeDraggingID = ""
	a.setStatus(a.t("dashboard.badge.resetDone"))
}

// viewDashboardBadgeTray renders the restrained, freely draggable micro-badge
// board to the right of the editorial date masthead, returning true when the
// floating badge board is expanded.
func (a *App) viewDashboardBadgeTray(c *ui.Context, availW float32) bool {
	badges := a.writingBadges()
	var unlocked []store.WritingBadge
	for _, b := range badges {
		if b.Unlocked {
			unlocked = append(unlocked, b)
		}
	}

	// Narrow layout, hidden state, or zero unlocked badges: show a quiet compact trophy pill.
	if a.badgesHidden || availW < 185 || len(unlocked) == 0 {
		a.badgeDraggingID = ""
		a.viewCompactBadgePill(c, len(unlocked), len(badges))
		return false
	}

	boardW := availW
	if boardW > 520 {
		boardW = 520
	}
	maxX := boardW - badgeTileSize
	maxY := badgeTrayHeight - badgeTileSize
	reduceMotion := c.Preferences().ReduceMotion

	ui.Box(c.Key("dash-badge-board")).Size(boardW, badgeTrayHeight).Children(func() {
		for _, b := range unlocked {
			badge := b
			spec := badgeVisualFor(badge.Kind)
			offset := a.ensureBadgeDragOffset(badge, maxX, maxY)
			bounds := ui.Rect{X: 0, Y: 0, W: maxX, H: maxY}

			btn := ui.ButtonBase(c.Key("dash-badge-"+badge.ID)).
				Label(a.badgeTitle(badge.Kind)).
				Size(badgeTileSize, badgeTileSize).
				Absolute().Left(0).Top(0).
				Cursor(ui.CursorGrab).
				Tooltip(a.badgeTooltip(badge)).
				Draggable(offset, ui.DraggableOptions{
					Bounds: &bounds,
				})

			dragging := btn.Dragging()
			hovered := btn.Hovered()
			tilt := spec.defaultTilt
			scale := float32(1.0)
			if dragging {
				if a.badgeDraggingID != badge.ID {
					a.badgeDraggingID = badge.ID
					a.bringBadgeToFront(badge.ID)
				}
				btn.Cursor(ui.CursorGrabbing).ZIndex(a.badgeZIndex(badge.ID, true))
				tilt = 0
				if !reduceMotion {
					scale = 1.10
				}
			} else {
				if a.badgeDraggingID == badge.ID {
					a.badgeDraggingID = ""
				}
				// Persist only settled positions so pointer moves do no disk I/O.
				a.commitBadgePosition(badge, *offset, maxX, maxY)
				if hovered && !reduceMotion {
					scale = 1.07
				}
				btn.ZIndex(a.badgeZIndex(badge.ID, false))
			}

			btn.Children(func() {
				face := ui.Box(c).Size(badgeTileSize, badgeTileSize).Rotate(tilt).Scale(scale, scale)
				face.Children(func() {
					a.viewBadgeGraphic(c, badge, spec, badgeTileSize, false)
				})
			})

			if btn.Clicked() {
				a.badgeWallOpen = true
			}
		}
	})
	return true
}

func (a *App) viewCompactBadgePill(c *ui.Context, unlockedCount, totalCount int) {
	pal := a.pal
	label := a.tf("dashboard.badge.compactLabel", unlockedCount, totalCount)
	tooltip := a.t("dashboard.badge.wallTooltip")
	if a.badgesHidden && unlockedCount > 0 {
		tooltip = a.t("dashboard.badge.showTooltip")
	}

	pill := ui.ButtonBase(c.Key("dash-badges-compact")).
		Label(label).
		Height(28).Padding(0, 10, 0, 9).Radius(14).
		Background(pal.prefsCard).Border(0.5, pal.prefsCardBorder).
		Shadow(0, 1, 3, 0, pal.prefsCardShadow).
		Center().Cursor(ui.CursorPointer).
		Tooltip(tooltip).
		Transition(rowHoverTransition)
	if pill.Hovered() {
		pill.Background(pal.hover)
	}
	pill.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
			tierKey, _, hasAny := writingTrophyRank(a.writingBadges())
			if bmp := trophyBitmapFor(tierKey); bmp != nil {
				img := ui.Image(c, bmp).Size(16, 16)
				if !hasAny {
					img.Grayscale().Opacity(0.55)
				}
			} else {
				ui.Icon(c, lucide.Trophy()).Size(12.5, 12.5).TextColor(pal.prefsMuted)
			}
			ui.Text(c, fmt.Sprintf("%d/%d", unlockedCount, totalCount)).
				Font(macPlanetFont).FontSize(11.5).FontWeight(500).
				FontFeatures("tnum").TextColor(pal.prefsSecondary).NoWrap()
		})
	})
	if pill.Clicked() {
		if a.badgesHidden && unlockedCount > 0 {
			a.badgesHidden = false
		} else {
			a.badgeWallOpen = true
		}
	}
}

func (a *App) viewBadgeGraphic(c *ui.Context, b store.WritingBadge, spec badgeVisualSpec, size float32, forceLockedStyle bool) {
	pal := a.pal
	locked := !b.Unlocked || forceLockedStyle

	if bmp := medalBitmapFor(b.Kind); bmp != nil {
		ui.Box(c).Size(size, size).Children(func() {
			img := ui.Image(c, bmp).Absolute().Fill()
			if locked {
				img.Grayscale().Opacity(0.30)
				return
			}
			valText := fmt.Sprintf("%d", b.Value)
			if b.Value > 999 {
				valText = "999+"
			}
			pillH := float32(15)
			pillPadX := float32(4.5)
			fontSize := float32(9.5)
			if size >= 76 {
				pillH = 20
				pillPadX = 7
				fontSize = 11.5
			}
			ui.Box(c).Height(pillH).PaddingX(pillPadX).Radius(pillH/2).
				Absolute().Right(0).Bottom(0).Center().
				Background(pal.prefsCard).Border(0.5, pal.prefsCardBorder).
				Shadow(0, 1, 3, 0, pal.prefsCardShadow).Children(func() {
				ui.Text(c, valText).
					Font(macPlanetFont).FontSize(fontSize).FontWeight(700).
					FontFeatures("tnum").TextColor(pal.prefsInk).NoWrap()
			})
		})
		return
	}

	var svg *ui.SVG
	var ink ui.Color
	if locked {
		if pal.dark {
			svg = spec.lockedDark
			ink = pal.prefsMuted
		} else {
			svg = spec.lockedLight
			ink = pal.prefsMuted
		}
	} else {
		if pal.dark {
			svg = spec.darkSVG
			ink = spec.darkInk
		} else {
			svg = spec.lightSVG
			ink = spec.lightInk
		}
	}

	numSize := float32(13.5)
	tagSize := float32(6.5)
	topPad := spec.textTopPad
	if size >= 76 {
		numSize = 23
		tagSize = 9
		topPad = spec.textTopPad * 1.7
	} else if size >= 52 {
		numSize = 16
		tagSize = 7.5
		topPad = spec.textTopPad * 1.2
	}

	ui.Box(c).Size(size, size).Children(func() {
		ui.Image(c, svg).Absolute().Fill()
		ui.Column(c).Absolute().Fill().Center().Padding(topPad, 0, 0, 0).Gap(0).Children(func() {
			valText := fmt.Sprintf("%d", b.Value)
			if locked && b.Value == 0 {
				valText = "—"
			} else if b.Value > 999 {
				valText = "999+"
				numSize -= 2.5
			}
			ui.Text(c, valText).
				Font(macPlanetFont).FontSize(numSize).FontWeight(700).LetterSpacing(-0.02).
				FontFeatures("tnum").TextColor(ink).NoWrap()
			ui.Text(c, spec.shortTag).
				Font(macPlanetFont).FontSize(tagSize).FontWeight(600).LetterSpacing(0.04).
				TextColor(ink.Alpha(0.72)).NoWrap()
		})
	})
}

func badgeMaxTiers(kind store.WritingBadgeKind) int {
	if kind == store.BadgeKindPlanet {
		return 4
	}
	return 5
}

func (a *App) badgeWallTooltip(b store.WritingBadge) string {
	title := a.badgeTitle(b.Kind)
	stat := a.badgeStatText(b)
	tierText := a.t("dashboard.badge.locked")
	if b.Unlocked {
		tierText = fmt.Sprintf("Lv.%d", b.Tier)
	}
	if b.NextGoal > 0 {
		return fmt.Sprintf("%s · %s · %s · %s", title, tierText, stat, a.tf("dashboard.badge.nextGoal", b.Value, b.NextGoal))
	}
	return fmt.Sprintf("%s · %s · %s · %s", title, tierText, stat, a.t("dashboard.badge.maxTier"))
}

func (a *App) viewBadgeWallModal(c *ui.Context) {
	if !a.badgeWallOpen {
		return
	}
	pal := a.pal
	badges := a.writingBadges()
	unlockedCount := 0
	for _, b := range badges {
		if b.Unlocked {
			unlockedCount++
		}
	}
	reduceMotion := c.Preferences().ReduceMotion
	_, hoverBorder, topHighlight := letterpressSurfaceColors(pal)

	macSheet(c, &a.badgeWallOpen, macSheetProps{
		Title:        a.t("dashboard.badge.wallTitle"),
		Width:        488,
		HideFooter:   true,
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		trophyTier, trophyTitleKey, hasAnyBadge := writingTrophyRank(badges)
		// Compact visual header: 3D rank trophy + colored unlock pips + numeric fraction (all details in tooltip).
		ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(10).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(10).Tooltip(a.t("dashboard.badge.wallSubtitle")).Children(func() {
				if bmp := trophyBitmapFor(trophyTier); bmp != nil {
					img := ui.Image(c, bmp).Size(40, 40)
					if !hasAnyBadge {
						img.Grayscale().Opacity(0.42)
					}
				}
				ui.Column(c).Gap(4).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Text(c, a.t(trophyTitleKey)).
							Font(macPlanetFont).FontSize(13).FontWeight(600).
							TextColor(pal.prefsInk).NoWrap()
						ui.Box(c).Padding(1, 6, 1, 6).Radius(4).Background(pal.hover).Children(func() {
							ui.Text(c, fmt.Sprintf("%d / %d", unlockedCount, len(badges))).
								Font(macPlanetFont).FontSize(11).FontWeight(600).
								FontFeatures("tnum").TextColor(pal.prefsSecondary).NoWrap()
						})
					})
					ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
						for _, b := range badges {
							spec := badgeVisualFor(b.Kind)
							dotColor := pal.prefsMuted.Alpha(0.25)
							if b.Unlocked {
								dotColor = spec.lightAccent
								if pal.dark {
									dotColor = spec.darkAccent
								}
							}
							ui.Box(c).Size(6.5, 6.5).Radius(3.25).Background(dotColor)
						}
					})
				})
			})

			ui.Row(c).AlignItems(ui.Center).Gap(6).Shrink(0).Children(func() {
				toggleLabel := a.t("dashboard.badge.hideOnDesk")
				if a.badgesHidden {
					toggleLabel = a.t("dashboard.badge.showOnDesk")
				}
				if macSecondaryButton(c, "badge-wall-toggle", toggleLabel, pal) {
					a.badgesHidden = !a.badgesHidden
				}
				if a.hasCustomBadgePositions() {
					if macSecondaryButton(c, "badge-wall-reset", a.t("dashboard.badge.resetLayout"), pal) {
						a.resetBadgePositions()
					}
				}
			})
		})

		// 2x2 Visual Medal Showcase Pedestals (graphics-first, text tucked into tooltips).
		ui.Column(c).AlignSelf(ui.Stretch).Gap(12).Children(func() {
			for row := 0; row < len(badges); row += 2 {
				ui.Row(c).AlignSelf(ui.Stretch).Gap(12).Children(func() {
					for col := 0; col < 2; col++ {
						idx := row + col
						if idx >= len(badges) {
							ui.Box(c).Grow(1).MinWidth(0)
							continue
						}
						b := badges[idx]
						spec := badgeVisualFor(b.Kind)
						accent := spec.lightAccent
						if pal.dark {
							accent = spec.darkAccent
						}

						progress := float32(1)
						if b.NextGoal > 0 {
							progress = float32(b.Value) / float32(b.NextGoal)
							if progress < 0 {
								progress = 0
							} else if progress > 1 {
								progress = 1
							}
						}

						card := ui.Column(c.Key("badge-wall-card-"+b.ID)).
							Grow(1).MinWidth(0).Height(178).Radius(14).Clip().
							Background(pal.prefsCard).Border(0.5, pal.prefsCardBorder).
							Shadow(0, 3, 10, 0, pal.prefsCardShadow).
							Tooltip(a.badgeWallTooltip(b))

						hovered := card.Hovered()
						if hovered {
							card.Border(0.5, hoverBorder)
						}

						// Rich ambient pedestal backdrop: soft radial halo + subtle showcase ring.
						glowAlpha := float32(0.16)
						if pal.dark {
							glowAlpha = 0.22
						}
						if hovered {
							glowAlpha += 0.06
						}
						card.Draw(func(p *ui.Painter, r ui.Rect) {
							// Subtle top specular hairline.
							p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: 0.5}, topHighlight, 0)
							cx := r.X + r.W/2
							cy := r.Y + 66
							if b.Unlocked {
								p.FillRadialGradient(r, ui.RadialGradient{
									Center: ui.Point{X: 0.5, Y: 0.37},
									Radius: 96,
									Stops: []ui.GradientStop{
										{At: 0, Color: accent.Alpha(glowAlpha)},
										{At: 0.58, Color: accent.Alpha(glowAlpha * 0.28)},
										{At: 1, Color: ui.Transparent},
									},
								}, 14)
								ring := ui.Rect{X: cx - 50, Y: cy - 50, W: 100, H: 100}
								p.Stroke(ring, accent.Alpha(0.16), 50, 0.75)
							} else {
								ring := ui.Rect{X: cx - 48, Y: cy - 48, W: 96, H: 96}
								p.StrokeDashed(ring, pal.prefsMuted.Alpha(0.16), 48, 0.75)
							}
						})

						card.Children(func() {
							ui.Column(c).Fill().Padding(18, 16, 14, 16).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
								// Hero die-cut medal (86x86) with gentle tilt and hover lift.
								medal := ui.Box(c).Size(86, 86).Rotate(spec.defaultTilt * 0.55)
								if hovered && !reduceMotion {
									medal.Translate(0, -3).Scale(1.06, 1.06)
								}
								medal.Children(func() {
									a.viewBadgeGraphic(c, b, spec, 86, !b.Unlocked)
									if !b.Unlocked {
										ui.Box(c).Size(20, 20).Radius(10).Absolute().Right(2).Bottom(2).Center().
											Background(pal.prefsCard).Border(0.5, pal.prefsCardBorder).Children(func() {
											ui.Icon(c, lucide.Lock()).Size(10.5, 10.5).TextColor(pal.prefsMuted)
										})
									}
								})

								// Visual tier pips + short title + hairline progress bar.
								ui.Column(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Gap(6).Children(func() {
									totalTiers := badgeMaxTiers(b.Kind)
									ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
										for t := 1; t <= totalTiers; t++ {
											pipColor := pal.prefsMuted.Alpha(0.22)
											if t <= b.Tier {
												pipColor = accent
											}
											ui.Box(c).Size(5.5, 5.5).Radius(2.75).Background(pipColor)
										}
									})

									ui.Text(c, a.badgeTitle(b.Kind)).
										Font(macPlanetFont).FontSize(12.5).FontWeight(600).
										TextColor(pal.prefsInk).NoWrap().Ellipsis("…")

									ui.Row(c).Height(3).Width(104).Radius(1.5).Background(pal.hover).Clip().Children(func() {
										barColor := pal.prefsMuted.Alpha(0.32)
										if b.Unlocked {
											barColor = accent
										}
										if progress > 0 {
											ui.Box(c).Height(3).Grow(progress).Radius(1.5).Background(barColor)
										}
										if progress < 1 {
											ui.Box(c).Height(3).Grow(1 - progress)
										}
									})
								})
							})
						})
					}
				})
			}
		})
	})
}
