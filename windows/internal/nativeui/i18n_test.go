package nativeui

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

var printfVerbRe = regexp.MustCompile(`%(?:\.\d+)?(?:\d+)?[sdvf]`)

func TestI18nCatalogParityAndPlaceholders(t *testing.T) {
	catalogs := map[Locale]map[string]string{
		LocaleZhCN: i18nZhCN,
		LocaleZhTW: i18nZhTW,
		LocaleEn:   i18nEn,
	}

	// Verify every key in i18nEn exists in all catalogs and vice versa.
	allKeys := map[string]bool{}
	for _, cat := range catalogs {
		for k := range cat {
			allKeys[k] = true
		}
	}

	sortedKeys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		sortedKeys = append(sortedKeys, k)
	}
	slices.Sort(sortedKeys)

	for _, k := range sortedKeys {
		enVal, okEn := i18nEn[k]
		if !okEn || enVal == "" {
			t.Errorf("key %q missing or empty in en catalog", k)
		}
		enVerbs := printfVerbRe.FindAllString(enVal, -1)

		for loc, cat := range catalogs {
			val, ok := cat[k]
			if !ok {
				t.Errorf("key %q missing in %s catalog", k, loc)
				continue
			}
			if val == "" {
				t.Errorf("key %q is empty in %s catalog", k, loc)
				continue
			}
			verbs := printfVerbRe.FindAllString(val, -1)
			if !slices.Equal(verbs, enVerbs) {
				t.Errorf("key %q placeholder mismatch in %s: got %v, en has %v", k, loc, verbs, enVerbs)
			}
		}
	}
}

func TestNormalizeLocaleFallbackToEn(t *testing.T) {
	cases := []struct {
		raw  string
		want Locale
	}{
		{"zh-CN", LocaleZhCN},
		{"zh_cn", LocaleZhCN},
		{"zh-Hans", LocaleZhCN},
		{"zh", LocaleZhCN},
		{"zh-TW", LocaleZhTW},
		{"zh-Hant", LocaleZhTW},
		{"zh-HK", LocaleZhTW},
		{"en", LocaleEn},
		{"en-US", LocaleEn},
		{"en_GB", LocaleEn},
		{"", LocaleEn},
		{"fr-FR", LocaleEn},
		{"ja", LocaleEn},
		{"invalid", LocaleEn},
	}
	for _, tc := range cases {
		if got := NormalizeLocale(tc.raw); got != tc.want {
			t.Errorf("NormalizeLocale(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestTranslateFallbackToEn(t *testing.T) {
	// Unsupported locale falls back to English.
	if got := translate(Locale("de-DE"), "nav.overview"); got != "Dashboard" {
		t.Fatalf("translate unsupported locale = %q, want %q", got, "Dashboard")
	}
	// Missing key in target locale falls back to English.
	orig := i18nZhCN["nav.overview"]
	delete(i18nZhCN, "nav.overview")
	defer func() { i18nZhCN["nav.overview"] = orig }()
	if got := translate(LocaleZhCN, "nav.overview"); got != "Dashboard" {
		t.Fatalf("translate missing key in zh-CN = %q, want %q", got, "Dashboard")
	}
}

func TestLocalePersistenceAndFallback(t *testing.T) {
	dir := t.TempDir()
	prefsFile := filepath.Join(dir, "nativeui-layout.json")

	app1 := New(&fakeBackend{})
	app1.EnablePrefsPersistence(prefsFile)
	if app1.Locale() != LocaleZhCN {
		t.Fatalf("default locale = %q, want %q", app1.Locale(), LocaleZhCN)
	}
	app1.SetLocale(LocaleEn)
	if app1.Locale() != LocaleEn {
		t.Fatalf("after SetLocale(en) = %q, want %q", app1.Locale(), LocaleEn)
	}

	// Reload into a fresh App and verify persistence.
	app2 := New(&fakeBackend{})
	app2.EnablePrefsPersistence(prefsFile)
	if app2.Locale() != LocaleEn {
		t.Fatalf("reloaded locale = %q, want %q", app2.Locale(), LocaleEn)
	}

	// Unsupported locale in prefs file falls back to en.
	if err := os.WriteFile(prefsFile, []byte(`{"locale":"ja-JP"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	app3 := New(&fakeBackend{})
	app3.EnablePrefsPersistence(prefsFile)
	if app3.Locale() != LocaleEn {
		t.Fatalf("unsupported prefs locale = %q, want %q", app3.Locale(), LocaleEn)
	}
}

func TestLiveLocaleSwitchRendersTranslatedUI(t *testing.T) {
	fb := &fakeBackend{}
	app := New(fb)

	tester := ui.NewTester(app.View, 1200, 800)
	tester.Frame()
	if !tester.HasText("写作台") || !tester.HasText("我的站点") {
		t.Fatalf("expected zh-CN sidebar labels by default")
	}

	app.SetLocale(LocaleEn)
	tester.Frame()
	if !tester.HasText("Dashboard") || !tester.HasText("My Sites") {
		t.Fatalf("expected en sidebar labels after switching to en")
	}
	if tester.HasText("写作台") {
		t.Fatalf("did not expect zh-CN label after switching to en")
	}

	app.SetLocale(LocaleZhTW)
	tester.Frame()
	if !tester.HasText("寫作台") || !tester.HasText("我的站點") {
		t.Fatalf("expected zh-TW sidebar labels after switching to zh-TW")
	}

	// Verify inline language switch from the bottom-left Settings popover.
	if err := tester.Click("設定"); err != nil {
		t.Fatalf("failed to open settings popover in zh-TW: %v", err)
	}
	if err := tester.Click("English"); err != nil {
		t.Fatalf("failed to switch language to English in settings popover: %v", err)
	}
	if app.Locale() != LocaleEn || !tester.HasText("Dashboard") {
		t.Fatalf("expected English locale after popover click, got %q", app.Locale())
	}
}
