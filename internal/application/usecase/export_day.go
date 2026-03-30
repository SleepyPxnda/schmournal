package usecase

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sleepypxnda/schmournal/internal/domain/repository"
	"github.com/sleepypxnda/schmournal/internal/domain/service"
)

// ExportDayInput contains the data needed to export a day.
type ExportDayInput struct {
	Date string // YYYY-MM-DD format
}

// ExportDayOutput contains the result of exporting a day.
type ExportDayOutput struct {
	FilePath       string
	MarkdownLength int
}

// ExportDayUseCase handles exporting a day record to Markdown format.
// This use case orchestrates:
// 1. Loading the day record
// 2. Generating Markdown
// 3. Saving the export to disk
// 4. Returning the file path
type ExportDayUseCase struct {
	dayRepo         repository.DayRecordRepository
	exportDir       string // base directory for exports
	exportGenerator *service.ExportGenerator
}

// NewExportDayUseCase creates a new ExportDayUseCase.
// exportDir is the base directory where exports will be saved (e.g., ~/.journal/exports).
func NewExportDayUseCase(
	dayRepo repository.DayRecordRepository,
	exportDir string,
	exportGenerator *service.ExportGenerator,
) *ExportDayUseCase {
	return &ExportDayUseCase{
		dayRepo:         dayRepo,
		exportDir:       exportDir,
		exportGenerator: exportGenerator,
	}
}

// Execute exports the specified day to a Markdown file.
func (uc *ExportDayUseCase) Execute(input ExportDayInput) (*ExportDayOutput, error) {
	// Validate input
	if input.Date == "" {
		return nil, fmt.Errorf("date is required")
	}

	// Load day record
	record, err := uc.dayRepo.FindByDate(input.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to load day record: %w", err)
	}

	// Check if record exists (has any data)
	if len(record.Entries) == 0 && record.Notes == "" {
		return nil, fmt.Errorf("day record is empty, nothing to export")
	}

	if uc.exportGenerator == nil {
		return nil, fmt.Errorf("export generator is required")
	}

	// Generate Markdown using domain export generator
	markdown := uc.exportGenerator.GenerateMarkdown(record)

	// Ensure export directory exists
	if err := os.MkdirAll(uc.exportDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create export directory: %w", err)
	}

	// Write to file
	filename := fmt.Sprintf("export-%s.md", input.Date)
	filePath := filepath.Join(uc.exportDir, filename)

	if err := os.WriteFile(filePath, []byte(markdown), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write export file: %w", err)
	}

	return &ExportDayOutput{
		FilePath:       filePath,
		MarkdownLength: len(markdown),
	}, nil
}
