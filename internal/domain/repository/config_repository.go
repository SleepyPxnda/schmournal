package repository

import "github.com/sleepypxnda/schmournal/internal/domain/model"

// ConfigRepository defines the interface for loading and saving application configuration.
// This is a domain interface - implementations live in infrastructure layer.
//
// Design Decision: Config is an infrastructure concern (file I/O, TOML parsing),
// but the application needs to access it through an abstraction.
type ConfigRepository interface {
	// Load reads the configuration from disk.
	// Returns default config if file doesn't exist.
	// Performs automatic migration if config is outdated.
	Load() (model.AppConfig, error)

	// Save writes the configuration to disk.
	Save(cfg model.AppConfig) error

	// GetPath returns the file path of the config file.
	GetPath() (string, error)
}
