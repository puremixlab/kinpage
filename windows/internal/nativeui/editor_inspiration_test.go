package nativeui

import (
	"slices"
	"testing"
	"time"

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

func TestEditorInspirationAnimatesOnTheEditingSideOfSplit(t *testing.T) {
	app := New(&fakeBackend{sites: []store.Site{{ID: "s1", Name: "山野通信"}}})
	app.startNewDraft("s1")
	tester := ui.NewTester(app.View, 1050, 700)
	tester.Frame()

	click := func(label string) {
		t.Helper()
		if err := tester.Click(label); err != nil {
			t.Fatalf("click %q: %v", label, err)
		}
	}

	click("分屏")
	click("没思路")
	body, hasBody := tester.Find(app.t("editor.label"))
	apple, hasApple := tester.Find("摘一颗苹果")
	if !hasBody || !hasApple || apple.X < body.X || apple.X+apple.W > body.X+body.W {
		t.Fatalf("apple tree should fit inside the editing pane: editor=%+v, tree=%+v", body, apple)
	}
	click("摘一颗苹果")
	tester.Advance(180 * time.Millisecond)
	if _, found := tester.Find("继续写"); found {
		t.Fatal("the prompt should wait for the apple to land")
	}
	tester.Advance(400 * time.Millisecond)
	if _, found := tester.Find(app.t("editor.inspirationOpen1")); !found {
		t.Fatal("the apple should reveal an opening prompt for a blank draft")
	}
	if app.editorBuffer.Len() != 0 || app.isEditorDirty() {
		t.Fatal("the animation should leave the draft blank")
	}

	click("再摘一颗")
	click("预览")
	tester.Advance(time.Second)
	for _, label := range []string{"没思路", "摘一颗苹果", "再摘一颗", "继续写"} {
		if _, found := tester.Find(label); found {
			t.Fatalf("%q should be hidden in Preview", label)
		}
	}

	click("分屏")
	click("没思路")
	body, hasBody = tester.Find(app.t("editor.label"))
	if !hasBody {
		t.Fatal("the editing pane should be available again")
	}
	tester.ClickAt(body.X+12, body.Y+16)
	tester.Type("雨停了。")
	if _, found := tester.Find("摘一颗苹果"); found || app.editorBuffer.String() != "雨停了。" {
		t.Fatal("clicking into the draft and typing should close the tree and edit the body")
	}
}
