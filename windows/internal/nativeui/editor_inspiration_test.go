package nativeui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

func TestEditorInspirationResumesDraft(t *testing.T) {
	const content = "雨是傍晚停的。\n巷口的路灯亮了。"
	draft := store.Draft{ID: "d1", SiteID: "s1", Title: "雨停以后", Content: content}
	app := New(&fakeBackend{sites: []store.Site{{ID: "s1", Name: "山野通信"}}, drafts: []store.Draft{draft}})
	app.openDraft(draft)
	tester := ui.NewTester(app.View, 1280, 820)
	tester.SetPreferences(ui.Preferences{ReduceMotion: true})
	tester.Frame()
	app.editorState.Select(len("雨是傍晚停的。\n"), len("雨是傍晚停的。\n巷口"))
	selection := app.editorState.Selections()
	version := app.editorBuffer.Version()

	click := func(label string) {
		t.Helper()
		if err := tester.Click(label); err != nil {
			t.Fatalf("click %q: %v", label, err)
		}
	}

	click("没思路")
	if !tester.Focused("摘一颗苹果") {
		t.Fatal("opening the tree should focus its apple")
	}
	click("摘一颗苹果")
	if _, found := tester.Find(app.t("editor.inspirationContinue1")); !found {
		t.Fatal("the first pick should offer a continuation prompt")
	}
	click("再摘一颗")
	if _, found := tester.Find(app.t("editor.inspirationContinue2")); !found {
		t.Fatal("picking again should offer a different prompt")
	}
	click("继续写")
	if !tester.Focused(app.t("editor.label")) {
		t.Fatal("Continue should return focus to the editor")
	}
	if !slices.Equal(app.editorState.Selections(), selection) {
		t.Fatal("the original selection should survive picking and continuing")
	}
	if app.editorBuffer.Version() != version || app.editorBuffer.String() != content || app.isEditorDirty() {
		t.Fatal("picking and continuing should leave the draft untouched")
	}
	if _, found := tester.Find("继续写"); found {
		t.Fatal("the prompt should close after continuing")
	}

	click("没思路")
	tester.Key(0, ui.KeyEscape)
	if _, found := tester.Find("摘一颗苹果"); found || !tester.Focused(app.t("editor.label")) {
		t.Fatal("Escape from the tree should close it and return focus to the editor")
	}
}
