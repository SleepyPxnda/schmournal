package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sleepypxnda/schmournal/internal/application/usecase"
	"github.com/sleepypxnda/schmournal/internal/domain/model"
	"github.com/sleepypxnda/schmournal/internal/domain/repository"
	"github.com/sleepypxnda/schmournal/internal/domain/service"
	infraConfig "github.com/sleepypxnda/schmournal/internal/infrastructure/config"
	"github.com/sleepypxnda/schmournal/internal/infrastructure/persistence/json"
	infrastructuretime "github.com/sleepypxnda/schmournal/internal/infrastructure/time"
	"github.com/sleepypxnda/schmournal/internal/ui/tui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("schmournal", version)
		return
	}

	configDir, err := resolveConfigDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: could not determine config directory:", err)
		configDir = "."
	}

	configRepo, err := infraConfig.NewFileSystemConfigRepository(configDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: could not initialize config repository:", err)
	}
	stateRepo := infraConfig.NewFileSystemStateRepository(configDir)

	cfgModel, err := loadConfigModel(configRepo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: could not load config:", err)
	}

	// Determine the active workspace and apply its settings.
	state, err := stateRepo.LoadState()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: could not load state:", err)
	}
	activeWorkspace := resolveActiveWorkspace(cfgModel, state.ActiveWorkspace)

	storagePath := cfgModel.StoragePath
	if activeWorkspace != "" {
		for _, ws := range cfgModel.Workspaces {
			if ws.Name == activeWorkspace {
				if ws.StoragePath != "" {
					storagePath = ws.StoragePath
				}
				break
			}
		}
	}

	useCaseFactory := newUseCaseSetFactory(stateRepo)
	initialSet, err := useCaseFactory(storagePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: could not initialize use cases:", err)
		os.Exit(1)
	}
	useCases := tui.NewUseCases(initialSet, useCaseFactory)

	p := tea.NewProgram(
		tui.New(cfgModel, activeWorkspace, version, useCases),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// resolveActiveWorkspace returns the workspace name to use on startup.
// If the saved name is valid it is returned; otherwise the first configured
// workspace name is returned (or "" when no workspaces are defined).
func resolveActiveWorkspace(cfg model.AppConfig, saved string) string {
	if len(cfg.Workspaces) == 0 {
		return ""
	}
	for _, ws := range cfg.Workspaces {
		if ws.Name == saved {
			return saved
		}
	}
	// Fall back to the first workspace.
	return cfg.Workspaces[0].Name
}

func newUseCaseSetFactory(stateRepo repository.StateRepository) tui.UseCaseSetFactory {
	return func(storagePath string) (tui.UseCaseSet, error) {
		storageManager, err := json.NewStorageManager(storagePath)
		if err != nil {
			return tui.UseCaseSet{}, fmt.Errorf("failed to initialize storage manager: %w", err)
		}

		timeProvider := infrastructuretime.NewRealTimeProvider()
		dayRepo := json.NewFileSystemDayRecordRepository(storageManager)
		todoRepo := json.NewFileSystemTodoRepository(storageManager)
		todoOps := service.NewTodoOperations()

		return tui.UseCaseSet{
			DayRecordRepo:      dayRepo,
			TodoRepo:           todoRepo,
			StateRepo:          stateRepo,
			LoadDayRecord:      usecase.NewLoadDayRecordUseCase(dayRepo),
			LoadAllDayRecords:  usecase.NewLoadAllDayRecordsUseCase(dayRepo),
			SaveDayRecord:      usecase.NewSaveDayRecordUseCase(dayRepo),
			DeleteDayRecord:    usecase.NewDeleteDayRecordUseCase(dayRepo),
			LoadWorkspaceTodos: usecase.NewLoadWorkspaceTodosUseCase(todoRepo),
			SaveWorkspaceTodos: usecase.NewSaveWorkspaceTodosUseCase(todoRepo),
			AddWorkEntry:       usecase.NewAddWorkEntryUseCase(dayRepo, timeProvider),
			UpdateWorkEntry:    usecase.NewUpdateWorkEntryUseCase(dayRepo),
			DeleteWorkEntry:    usecase.NewDeleteWorkEntryUseCase(dayRepo),
			SubmitWorkForm:     usecase.NewSubmitWorkFormUseCase(dayRepo, timeProvider),
			SetDayTimes:        usecase.NewSetDayTimesUseCase(dayRepo),
			UpdateNotes:        usecase.NewUpdateNotesUseCase(dayRepo),
			ManageTodos:        usecase.NewManageTodosUseCase(todoRepo, todoOps),
		}, nil
	}
}

func resolveConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

func loadConfigModel(repo repository.ConfigRepository) (model.AppConfig, error) {
	if repo == nil {
		return model.DefaultAppConfig(), fmt.Errorf("config repository is not configured")
	}
	return repo.Load()
}
