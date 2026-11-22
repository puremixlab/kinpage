package main

import "windows/internal/store"

func (l *Library) Dashboard() (store.DashboardSnapshot, error) {
	s, err := l.open()
	if err != nil {
		return store.DashboardSnapshot{}, err
	}
	return s.Dashboard(), nil
}

func (l *Library) WritingActivity() ([]store.DailyActivity, error) {
	s, err := l.open()
	if err != nil {
		return nil, err
	}
	return s.WritingActivity(), nil
}
