package journal

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SyncConfig holds settings for cloud synchronisation via rclone.
type SyncConfig struct {
	// Remote is the rclone remote path, e.g. "gdrive:journal" or "s3:mybucket/journal".
	// Leave empty to disable sync.
	Remote string `json:"remote"`
	// Direction controls which way files are copied: "push", "pull", or "both" (default).
	Direction string `json:"direction"`
}

// Config holds application-level settings stored in ~/.journal/config.json.
type Config struct {
	Sync SyncConfig `json:"sync"`
}

// configPath returns the path to the config file.
func configPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// LoadConfig reads the config from ~/.journal/config.json.
// If the file does not exist, a zero-value Config (sync disabled) is returned.
func LoadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// SaveConfig writes cfg to ~/.journal/config.json.
func SaveConfig(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
