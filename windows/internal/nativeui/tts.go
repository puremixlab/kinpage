package nativeui

// Speaker provides native text-to-speech playback without a browser WebView.
type Speaker interface {
	Speak(text string) error
	Stop()
	Speaking() bool
}
