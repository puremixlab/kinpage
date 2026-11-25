//go:build !windows

package nativeui

import "errors"

type noopSpeaker struct{}

func newDefaultSpeaker() Speaker       { return noopSpeaker{} }
func (noopSpeaker) Speak(string) error { return errors.New("TTS is only supported on Windows") }
func (noopSpeaker) Stop()              {}
func (noopSpeaker) Speaking() bool     { return false }
func openBrowserFile(string) error     { return errors.New("browser preview is only supported on Windows") }
