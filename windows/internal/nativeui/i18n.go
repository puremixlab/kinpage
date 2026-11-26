package nativeui

import (
	"fmt"
	"strings"
	"time"
)

// Locale identifies a supported UI language in nativeui.
type Locale string

const (
	LocaleZhCN Locale = "zh-CN"
	LocaleZhTW Locale = "zh-TW"
	LocaleEn   Locale = "en"
)

// SupportedLocales returns all supported UI locales in display order.
func SupportedLocales() []Locale {
	return []Locale{LocaleZhCN, LocaleZhTW, LocaleEn}
}

// NormalizeLocale maps raw locale tags to a supported Locale.
// Unknown or missing languages fall back to English (LocaleEn).
func NormalizeLocale(raw string) Locale {
	norm := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(raw, "_", "-")))
	switch {
	case norm == "zh-cn" || norm == "zh-hans" || norm == "zh-sg" || norm == "zh":
		return LocaleZhCN
	case norm == "zh-tw" || norm == "zh-hant" || norm == "zh-hk" || norm == "zh-mo":
		return LocaleZhTW
	case norm == "en" || strings.HasPrefix(norm, "en-"):
		return LocaleEn
	default:
		return LocaleEn
	}
}

// LocaleDisplayName returns the native display name of a locale.
func LocaleDisplayName(l Locale) string {
	switch l {
	case LocaleZhCN:
		return "简体中文"
	case LocaleZhTW:
		return "繁體中文"
	default:
		return "English"
	}
}

func localeCatalog(l Locale) map[string]string {
	switch l {
	case LocaleZhCN:
		return i18nZhCN
	case LocaleZhTW:
		return i18nZhTW
	case LocaleEn:
		return i18nEn
	default:
		return i18nEn
	}
}

func translate(l Locale, key string) string {
	if dict := localeCatalog(l); dict != nil {
		if v, ok := dict[key]; ok && v != "" {
			return v
		}
	}
	if v, ok := i18nEn[key]; ok && v != "" {
		return v
	}
	if v, ok := i18nZhCN[key]; ok && v != "" {
		return v
	}
	return key
}

// Locale returns the active UI locale.
func (a *App) Locale() Locale {
	if a.locale == "" {
		return LocaleZhCN
	}
	return a.locale
}

// SetLocale switches the active UI locale (falling back to en for unsupported values)
// and refreshes dynamic editor labels.
func (a *App) SetLocale(l Locale) {
	a.locale = NormalizeLocale(string(l))
	a.syncEditorLocale()
	a.maybeSaveLayoutPrefs()
}

func (a *App) syncEditorLocale() {
	if a.editorState != nil {
		a.editorState.Placeholder = a.t("editor.placeholder")
		a.editorState.Label = a.t("editor.label")
	}
}

func (a *App) t(key string) string {
	return translate(a.Locale(), key)
}

func (a *App) tf(key string, args ...any) string {
	return fmt.Sprintf(a.t(key), args...)
}

func (a *App) fallbackTitle(s string) string {
	if strings.TrimSpace(s) == "" {
		return a.t("common.untitled")
	}
	return s
}

func (a *App) starTypeLabel(id int) string {
	for _, opt := range allStarTypes {
		if opt.id == id {
			return a.t(opt.key)
		}
	}
	return a.t("star.default")
}

func (a *App) formatRelativeDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	now := time.Now().Local()
	lt := t.Local()
	ny, nm, nd := now.Date()
	ly, lm, ld := lt.Date()
	if ny == ly && nm == lm && nd == ld {
		return lt.Format("15:04")
	}
	yesterday := now.AddDate(0, 0, -1)
	yy, ym, yd := yesterday.Date()
	if yy == ly && ym == lm && yd == ld {
		return a.t("reader.yesterday")
	}
	if ny == ly {
		return a.formatShortDate(lt)
	}
	return lt.Format("2006/01/02")
}

func (a *App) formatShortDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	lt := t.Local()
	if a.Locale() == LocaleEn {
		return lt.Format("Jan 2")
	}
	return fmt.Sprintf("%d月%d日", int(lt.Month()), lt.Day())
}

func (a *App) formatReaderDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	lt := t.Local()
	if a.Locale() == LocaleEn {
		return lt.Format("Jan 2, 2006 15:04")
	}
	return lt.Format("2006年1月2日 15:04")
}

func (a *App) formatFeedDate(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	for _, layout := range []string{
		time.RFC3339,
		time.RFC1123Z,
		time.RFC1123,
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006/01/02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return a.formatRelativeDate(t)
		}
	}
	return s
}

func (a *App) formatMastheadDate(t time.Time) string {
	if a.Locale() == LocaleEn {
		return t.Format("Mon, Jan 2")
	}
	wdKey := fmt.Sprintf("dashboard.weekday%d", int(t.Weekday()))
	return fmt.Sprintf("%d月%d日 · %s", int(t.Month()), t.Day(), a.t(wdKey))
}

func (a *App) formatWordCount(n int) string {
	if n <= 0 {
		return a.t("dashboard.noContent")
	}
	if n >= 1000 {
		return a.tf("dashboard.wordCountLarge", n/1000, n%1000)
	}
	return a.tf("dashboard.wordCount", n)
}
