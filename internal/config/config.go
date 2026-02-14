package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds application configuration.
type Config struct {
	RepoPath string
}

// Default returns the default configuration.
func Default() Config {
	return Config{}
}

// FileView controls how files are displayed in the working copy panel.
type FileView string

const (
	FileViewTree FileView = "tree"
	FileViewList FileView = "list"
)

// Settings holds user preferences persisted to disk.
type Settings struct {
	FileView FileView `json:"fileView"`
}

var configDir = filepath.Join(os.Getenv("HOME"), ".moron")
var configPath = filepath.Join(configDir, "settings.json")

// DefaultSettings returns the default configuration.
func DefaultSettings() Settings {
	return Settings{
		FileView: FileViewTree,
	}
}

// LoadSettings reads settings from ~/.moron/settings.json.
func LoadSettings() Settings {
	s := DefaultSettings()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	if s.FileView != FileViewTree && s.FileView != FileViewList {
		s.FileView = FileViewTree
	}
	return s
}

// SaveSettings writes settings to ~/.moron/settings.json.
func SaveSettings(s Settings) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}
