package nativeui

import (
	"embed"
	"sync"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

// Embed only the 256x256 scaled copies so the desktop binary stays lean while
// keeping the original high-resolution master PNGs untouched in assets/achievements/.
//
//go:embed assets/achievements/scaled/medals/*.png assets/achievements/scaled/trophies/*.png
var achievementArtworkFS embed.FS

var (
	achievementArtOnce sync.Once
	medalBitmaps       map[store.WritingBadgeKind]*ui.Bitmap
	trophyBitmaps      map[string]*ui.Bitmap
)

func loadAchievementArtwork() {
	achievementArtOnce.Do(func() {
		medalBitmaps = make(map[store.WritingBadgeKind]*ui.Bitmap, 4)
		medalFiles := map[store.WritingBadgeKind]string{
			store.BadgeKindActiveDays: "assets/achievements/scaled/medals/active-days.png",
			store.BadgeKindStreak:     "assets/achievements/scaled/medals/streak.png",
			store.BadgeKindPublished:  "assets/achievements/scaled/medals/published.png",
			store.BadgeKindPlanet:     "assets/achievements/scaled/medals/planet.png",
		}
		for kind, path := range medalFiles {
			if raw, err := achievementArtworkFS.ReadFile(path); err == nil {
				if bmp, err := ui.DecodeBitmapScaled(raw, 256); err == nil {
					medalBitmaps[kind] = bmp
				}
			}
		}

		trophyBitmaps = make(map[string]*ui.Bitmap, 5)
		for _, tier := range []string{"bronze", "silver", "gold", "platinum", "orbit"} {
			path := "assets/achievements/scaled/trophies/" + tier + ".png"
			if raw, err := achievementArtworkFS.ReadFile(path); err == nil {
				if bmp, err := ui.DecodeBitmapScaled(raw, 256); err == nil {
					trophyBitmaps[tier] = bmp
				}
			}
		}
	})
}

func medalBitmapFor(kind store.WritingBadgeKind) *ui.Bitmap {
	loadAchievementArtwork()
	return medalBitmaps[kind]
}

func trophyBitmapFor(tier string) *ui.Bitmap {
	loadAchievementArtwork()
	return trophyBitmaps[tier]
}

func writingTrophyRank(badges []store.WritingBadge) (tierKey string, titleKey string, unlocked bool) {
	totalTier := 0
	for _, b := range badges {
		if b.Unlocked {
			unlocked = true
			totalTier += b.Tier
		}
	}
	switch {
	case totalTier >= 15:
		return "orbit", "dashboard.badge.trophy.orbit", unlocked
	case totalTier >= 11:
		return "platinum", "dashboard.badge.trophy.platinum", unlocked
	case totalTier >= 7:
		return "gold", "dashboard.badge.trophy.gold", unlocked
	case totalTier >= 4:
		return "silver", "dashboard.badge.trophy.silver", unlocked
	default:
		return "bronze", "dashboard.badge.trophy.bronze", unlocked
	}
}
