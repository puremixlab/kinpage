package nativeui

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
)

type layoutPrefs struct {
	SidebarWidth     float32 `json:"sidebarWidth"`
	PostsPanelWidth  float32 `json:"postsPanelWidth"`
	SidebarCollapsed bool    `json:"sidebarCollapsed"`
	PostsCollapsed   bool    `json:"postsCollapsed"`
}

// EnablePrefsPersistence loads saved split widths and panel collapse states from
// path (if present) and persists subsequent changes whenever layout state updates.
func (a *App) EnablePrefsPersistence(path string) {
	a.prefsPath = path
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		a.lastSavedPrefs = a.currentLayoutPrefs()
		return
	}
	var p layoutPrefs
	if err := json.Unmarshal(data, &p); err != nil {
		a.lastSavedPrefs = a.currentLayoutPrefs()
		return
	}
	if p.SidebarWidth >= 220 && p.SidebarWidth <= 420 {
		a.sidebarWidth = p.SidebarWidth
	}
	if p.PostsPanelWidth >= 240 && p.PostsPanelWidth <= 520 {
		a.postsPanelWidth = p.PostsPanelWidth
	}
	a.sidebarCollapsed = p.SidebarCollapsed
	a.postsCollapsed = p.PostsCollapsed
	a.lastSavedPrefs = a.currentLayoutPrefs()
}

func (a *App) currentLayoutPrefs() layoutPrefs {
	return layoutPrefs{
		SidebarWidth:     float32(math.Round(float64(a.sidebarWidth))),
		PostsPanelWidth:  float32(math.Round(float64(a.postsPanelWidth))),
		SidebarCollapsed: a.sidebarCollapsed,
		PostsCollapsed:   a.postsCollapsed,
	}
}

func (a *App) maybeSaveLayoutPrefs() {
	if a.prefsPath == "" {
		return
	}
	cur := a.currentLayoutPrefs()
	if cur == a.lastSavedPrefs {
		return
	}
	a.lastSavedPrefs = cur
	if err := os.MkdirAll(filepath.Dir(a.prefsPath), 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(cur, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(a.prefsPath, data, 0o600)
}
