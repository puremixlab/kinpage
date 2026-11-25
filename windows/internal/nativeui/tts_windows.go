//go:build windows

package nativeui

import (
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

type windowsSpeaker struct {
	mu  sync.Mutex
	cmd *exec.Cmd
}

func newDefaultSpeaker() Speaker {
	return &windowsSpeaker{}
}

func (s *windowsSpeaker) Speak(text string) error {
	s.Stop()
	clean := strings.TrimSpace(text)
	if clean == "" {
		return nil
	}
	script := `Add-Type -AssemblyName System.Speech; $sp = New-Object System.Speech.Synthesis.SpeechSynthesizer; $inputText = [Console]::In.ReadToEnd(); $sp.Speak($inputText)`
	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Stdin = strings.NewReader(clean)
	if err := cmd.Start(); err != nil {
		return err
	}
	s.mu.Lock()
	s.cmd = cmd
	s.mu.Unlock()
	go func() {
		_ = cmd.Wait()
		s.mu.Lock()
		if s.cmd == cmd {
			s.cmd = nil
		}
		s.mu.Unlock()
	}()
	return nil
}

func (s *windowsSpeaker) Stop() {
	s.mu.Lock()
	cmd := s.cmd
	s.cmd = nil
	s.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func (s *windowsSpeaker) Speaking() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cmd != nil
}

func openBrowserFile(path string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	return cmd.Start()
}
