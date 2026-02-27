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

	if err := journal.SetStoragePath(cfg.StoragePath); err != nil {
		fmt.Fprintln(os.Stderr, "Warning: invalid storage_path in config:", err)
	}

	p := tea.NewProgram(
		ui.New(cfg),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
