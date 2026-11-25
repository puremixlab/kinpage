package store

import (
	"context"
	"encoding/json"
	"os"
	"time"
)

// PreventSleep returns whether the user enabled system sleep prevention.
func (s *Store) PreventSleep() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.PreventSleep
}

// SetPreventSleep persists the prevent-sleep preference.
func (s *Store) SetPreventSleep(enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.PreventSleep == enabled {
		return nil
	}
	next := s.data
	next.PreventSleep = enabled
	return s.write(next)
}

// ReloadIfExternallyModified checks whether the backing library.json was
// modified by an external process since the last load/write and reloads it
// atomically if valid. Returns true when an external change was applied.
func (s *Store) ReloadIfExternallyModified() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, err := os.Stat(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !s.lastModTime.IsZero() && info.ModTime().Equal(s.lastModTime) && info.Size() == s.lastSize {
		return false, nil
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return false, err
	}
	var parsed data
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return false, err
	}
	if parsed.Version == 0 {
		parsed.Version = 1
	}
	s.data = parsed
	s.lastModTime = info.ModTime()
	s.lastSize = info.Size()
	return true, nil
}

// WatchExternalChanges polls library.json at the given interval and invokes
// onChange whenever an external modification is detected and loaded.
func (s *Store) WatchExternalChanges(ctx context.Context, interval time.Duration, onChange func()) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if changed, err := s.ReloadIfExternallyModified(); err == nil && changed && onChange != nil {
				onChange()
			}
		}
	}
}
