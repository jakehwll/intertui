package config

import (
	"os"
	"path/filepath"
)

const (
	dirName    = ".intertui"
	configName = "config.yaml"
)

// Dir returns the intertui config directory (~/.intertui).
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, dirName), nil
}

// ConfigPath returns the default config file path (~/.intertui/config.yaml).
func ConfigPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configName), nil
}

// defaultConfigPath returns the config path when the file exists.
func defaultConfigPath() string {
	path, err := ConfigPath()
	if err != nil {
		return ""
	}
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}
