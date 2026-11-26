package nativeui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

type fakeSpeaker struct {
	speaking bool
	lastText string
	stopped  int
}

func (s *fakeSpeaker) Speak(text string) error {
	s.speaking = true
	s.lastText = text
	return nil
}

func (s *fakeSpeaker) Stop() {
	s.speaking = false
	s.stopped++
}

func (s *fakeSpeaker) Speaking() bool { return s.speaking }

func TestNativeUITTSSpeakAndTemplateBrowserPreview(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{{ID: "s1", Name: "朗读站", Created: now}},
		articles: []store.Article{{
			ID: "a1", SiteID: "s1", Title: "原生语音测试", Content: "今天天气不错",
			Created: now, Modified: now,
		}},
	}
	speaker := &fakeSpeaker{}
	openedPath := ""

	app := New(backend)
	app.speaker = speaker
	app.openExternal = func(p string) error {
		openedPath = p
		return nil
	}

	tt := ui.NewTester(app.View, 1280, 900)
	tt.Frame()

	if err := tt.Click("朗读站"); err != nil {
		t.Fatalf("failed to select site: %v", err)
	}
	if err := tt.Click("更多文章操作"); err != nil {
		t.Fatalf("failed to open reader more menu: %v", err)
	}
	if err := tt.ChooseMenuItem("朗读"); err != nil {
		t.Fatalf("failed to click speak: %v", err)
	}
	if !speaker.speaking || !strings.Contains(speaker.lastText, "今天天气不错") {
		t.Fatalf("expected speaker active with article text, got %+v", speaker)
	}
	if err := tt.Click("更多文章操作"); err != nil {
		t.Fatalf("failed to open reader more menu: %v", err)
	}
	if err := tt.ChooseMenuItem("停止朗读"); err != nil {
		t.Fatalf("failed to click stop speak: %v", err)
	}
	if speaker.speaking {
		t.Fatal("expected speaker stopped")
	}

	// Template preview opens generated HTML via external browser hook.
	if err := tt.Click("站点设置"); err != nil {
		t.Fatalf("failed to open site settings: %v", err)
	}
	if err := tt.Click("模板"); err != nil {
		t.Fatalf("failed to switch to templates tab: %v", err)
	}
	if err := tt.Click("预览"); err != nil {
		t.Fatalf("failed to click template preview: %v", err)
	}
	if openedPath == "" {
		t.Fatal("expected template preview HTML path to be opened externally")
	}
	data, err := os.ReadFile(openedPath)
	if err != nil || !strings.Contains(string(data), "<h1>plain</h1>") {
		t.Fatalf("expected rendered preview HTML at %q, err=%v", openedPath, err)
	}
}
