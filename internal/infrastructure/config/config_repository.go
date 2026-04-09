package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/sleepypxnda/schmournal/internal/domain/model"
	"github.com/sleepypxnda/schmournal/internal/domain/repository"
)

// FileSystemConfigRepository implements repository.ConfigRepository
// using TOML files on the filesystem.
type FileSystemConfigRepository struct {
	configDir string // directory containing config file (e.g., ~/.config)
}

type legacyGlobalSettings struct {
	StoragePath     string   `toml:"storage_path"`
	WeeklyHoursGoal float64  `toml:"weekly_hours_goal"`
	WorkDays        []string `toml:"work_days"`
}

// NewFileSystemConfigRepository creates a new FileSystemConfigRepository.
// If configDir is empty, uses the user's home directory + ".config".
func NewFileSystemConfigRepository(configDir string) (repository.ConfigRepository, error) {
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &FileSystemConfigRepository{
		configDir: configDir,
	}, nil
}

// Load reads the configuration from disk.
// Returns default config if file doesn't exist.
func (r *FileSystemConfigRepository) Load() (model.AppConfig, error) {
	def := model.DefaultAppConfig()

	path, err := r.GetPath()
	if err != nil {
		return def, fmt.Errorf("failed to get config path: %w", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Best effort: create a default config file for the user.
		_ = r.Save(def)
		return def, nil
	}

	var cfg model.AppConfig
	md, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return def, err
	}
	var legacy legacyGlobalSettings
	if _, err := toml.DecodeFile(path, &legacy); err != nil {
		return def, err
	}

	// Apply defaults for boolean module fields that were absent in older configs.
	// We cannot rely on ValidateAndNormalize for booleans because false is a valid
	// explicit value and also the zero value; TOML metadata tells us which is which.
	if !md.IsDefined("modules", "clock_enabled") {
		cfg.Modules.ClockEnabled = true
	}
	if !md.IsDefined("modules", "todo_enabled") {
		cfg.Modules.TodoEnabled = true
	}
	applyLegacyWorkspaceFallbacks(&cfg, legacy)

	if err := cfg.ValidateAndNormalize(); err != nil {
		return def, err
	}

	// If keys are missing (newer app version), rewrite the file with a complete
	// config and keep a backup of the old file.
	if needsMigration(md) {
		_ = r.migrateConfig(path, cfg)
	}
	return cfg, nil
}

// Save writes the configuration to disk.
func (r *FileSystemConfigRepository) Save(cfg model.AppConfig) error {
	path, err := r.GetPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}
	if err := cfg.ValidateAndNormalize(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}
	return nil
}

// GetPath returns the file path of the config file.
func (r *FileSystemConfigRepository) GetPath() (string, error) {
	return filepath.Join(r.configDir, "schmournal.config"), nil
}

func needsMigration(md toml.MetaData) bool {
	if md.IsDefined("storage_path") || md.IsDefined("weekly_hours_goal") || md.IsDefined("work_days") {
		return true
	}
	for _, path := range collectTOMLPaths(reflect.TypeOf(model.AppConfig{}), nil) {
		if !md.IsDefined(path...) {
			return true
		}
	}
	return false
}

func collectTOMLPaths(t reflect.Type, prefix []string) [][]string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var paths [][]string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := strings.Split(f.Tag.Get("toml"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		// Skip slices (arrays / arrays-of-tables) because they are optional.
		if f.Type.Kind() == reflect.Slice {
			continue
		}

		path := make([]string, len(prefix), len(prefix)+1)
		copy(path, prefix)
		path = append(path, tag)

		if f.Type.Kind() == reflect.Struct {
			paths = append(paths, collectTOMLPaths(f.Type, path)...)
		} else {
			paths = append(paths, path)
		}
	}
	return paths
}

func (r *FileSystemConfigRepository) migrateConfig(path string, cfg model.AppConfig) error {
	oldPath := strings.TrimSuffix(path, ".config") + ".old.config"
	if err := os.Rename(path, oldPath); err != nil {
		return err
	}
	return r.Save(cfg)
}

func applyLegacyWorkspaceFallbacks(cfg *model.AppConfig, legacy legacyGlobalSettings) {
	def := model.DefaultWorkspaceConfig("")

	workDaysFallback := append([]string(nil), def.WorkDays...)
	if len(legacy.WorkDays) > 0 {
		workDaysFallback = append([]string(nil), legacy.WorkDays...)
	}

	weeklyGoalFallback := def.WeeklyHoursGoal
	if legacy.WeeklyHoursGoal > 0 {
		weeklyGoalFallback = legacy.WeeklyHoursGoal
	}

	storageFallback := def.StoragePath
	if legacy.StoragePath != "" {
		storageFallback = legacy.StoragePath
	}

	if len(cfg.Workspaces) == 0 {
		cfg.Workspaces = []model.WorkspaceConfig{
			{
				Name:            "default",
				StoragePath:     storageFallback,
				WeeklyHoursGoal: weeklyGoalFallback,
				WorkDays:        append([]string(nil), workDaysFallback...),
			},
		}
		return
	}

	for i := range cfg.Workspaces {
		if cfg.Workspaces[i].StoragePath == "" {
			cfg.Workspaces[i].StoragePath = storageFallback
		}
		if cfg.Workspaces[i].WeeklyHoursGoal == 0 {
			cfg.Workspaces[i].WeeklyHoursGoal = weeklyGoalFallback
		}
		if len(cfg.Workspaces[i].WorkDays) == 0 {
			cfg.Workspaces[i].WorkDays = append([]string(nil), workDaysFallback...)
		}
	}
}
