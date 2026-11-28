package updates

// SettingsInfo exposes the local trust configuration to the desktop settings.
type SettingsInfo struct {
	Settings       Config `json:"settings"`
	CurrentVersion string `json:"currentVersion"`
	Configured     bool   `json:"configured"`
}

// CheckInfo refers to a release selected from a verified update manifest.
type CheckInfo struct {
	Version   string `json:"version"`
	SHA256    string `json:"sha256"`
	Size      int64  `json:"size"`
	Available bool   `json:"available"`
}
