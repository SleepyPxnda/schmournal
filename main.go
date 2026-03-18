package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sleepypxnda/schmournal/config"
	"github.com/sleepypxnda/schmournal/journal"
	"github.com/sleepypxnda/schmournal/ui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("schmournal", version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: could not load config:", err)
	}

	// Determine the active workspace and apply its settings.
	state, err := config.LoadState()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: could not load state:", err)
	}
	activeWorkspace := resolveActiveWorkspace(cfg, state.ActiveWorkspace)

	storagePath := cfg.StoragePath
	if activeWorkspace != "" {
		for _, ws := range cfg.Workspaces {
			if ws.Name == activeWorkspace {
				if ws.StoragePath != "" {
					storagePath = ws.StoragePath
				}
				break
			}
		}
	}

	if err := journal.SetStoragePath(storagePath); err != nil {
		fmt.Fprintln(os.Stderr, "Warning: invalid storage_path in config:", err)
	}

	p := tea.NewProgram(
		ui.New(cfg, activeWorkspace, version),
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
func resolveActiveWorkspace(cfg config.Config, saved string) string {
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

