package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

type exitGuardBackend struct {
	*fakeBackend
	failSave bool
}

func (b exitGuardBackend) SaveDraftWithTags(siteID, draftID, articleID, title, content, tags string) (store.Draft, error) {
	if b.failSave {
		return store.Draft{}, errors.New("draft save failed")
	}
	return b.fakeBackend.SaveDraftWithTags(siteID, draftID, articleID, title, content, tags)
}

func TestRequestExitProtectsUnsavedEditor(t *testing.T) {
	for _, tc := range []struct {
		name, button string
		failSave     bool
		wantExit     bool
		wantSaved    bool
	}{
		{name: "cancel", button: "取消"},
		{name: "save", button: "保存并继续", wantExit: true, wantSaved: true},
		{name: "save failure", button: "保存并继续", failSave: true},
		{name: "discard", button: "放弃更改", wantExit: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			site := store.Site{ID: "s1", Name: "写作"}
			backend := &fakeBackend{sites: []store.Site{site}}
			app := New(exitGuardBackend{fakeBackend: backend, failSave: tc.failSave})
			app.startNewDraft(site.ID)
			app.editorTitle = "正在编辑的标题"
			tt := ui.NewTester(app.View, 1280, 820)
			exited := false
			app.RequestExit(func() { exited = true })
			tt.Frame()
			if exited || !app.unsavedDialogOpen {
				t.Fatal("exit continued without asking how to handle the unsaved draft")
			}
			if err := tt.Click(tc.button); err != nil {
				t.Fatal(err)
			}
			if exited != tc.wantExit || (len(backend.drafts) == 1) != tc.wantSaved {
				t.Fatalf("exited=%v, saved drafts=%d", exited, len(backend.drafts))
			}
			if !tc.wantExit && (!app.isEditorDirty() || app.editorTitle != "正在编辑的标题") {
				t.Fatal("draft changes were lost after staying in the editor")
			}
		})
	}
}
