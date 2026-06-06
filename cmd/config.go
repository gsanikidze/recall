package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds persistent recall settings
type Config struct {
	ProjectPath string `json:"project_path"`
}

// configDir returns the directory where recall stores its settings, using the
// OS-appropriate user config location (e.g. ~/Library/Application Support on
// macOS, ~/.config on Linux, %AppData% on Windows).
func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating user config dir: %w", err)
	}
	return filepath.Join(base, "recall"), nil
}

// configPath returns the full path to the config file.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// loadConfig reads the config file. found is false when no config exists yet.
func loadConfig() (cfg Config, found bool, err error) {
	path, err := configPath()
	if err != nil {
		return Config{}, false, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, false, nil
	}
	if err != nil {
		return Config{}, false, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, false, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return cfg, true, nil
}

// saveConfig writes the config file, creating the config dir if needed.
func saveConfig(cfg Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config dir %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config %s: %w", path, err)
	}
	return nil
}

// defaultProjectPath returns the OS-appropriate default storage location.
func defaultProjectPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating home dir: %w", err)
	}
	return filepath.Join(home, "recall"), nil
}
