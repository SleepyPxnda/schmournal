package usecase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
	"github.com/sleepypxnda/schmournal/internal/domain/service"
)

func newTestExportGenerator() *service.ExportGenerator {
	return service.NewExportGenerator(
		service.NewDurationFormatter(),
		service.NewEntryConsolidator(),
		newTestTimeProviderAt(time.Date(2026, 3, 28, 15, 30, 0, 0, time.UTC)),
	)
}

func TestExportDay_Success(t *testing.T) {
	// Create temp export directory
	tempDir := t.TempDir()

	repo := NewMockDayRecordRepository()
	useCase := NewExportDayUseCase(repo, tempDir, newTestExportGenerator())

	// Add a record to export
	record := model.DayRecord{
		Date:      "2026-03-28",
		StartTime: "09:00",
		EndTime:   "17:00",
		Entries: []model.WorkEntry{
			{ID: "1", Task: "Work", Project: "Backend", DurationMin: 60, IsBreak: false},
		},
		Notes: "Productive day!",
	}
	_ = repo.Save(record)

	// Execute export
	input := ExportDayInput{Date: "2026-03-28"}
	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify output
	if output.FilePath == "" {
		t.Error("expected file path to be set")
	}
	if output.MarkdownLength == 0 {
		t.Error("expected markdown length > 0")
	}

	// Verify file was created
	if _, err := os.Stat(output.FilePath); os.IsNotExist(err) {
		t.Errorf("expected export file to exist at %s", output.FilePath)
	}

	// Verify file name
	expectedFilename := "export-2026-03-28.md"
	if !strings.HasSuffix(output.FilePath, expectedFilename) {
		t.Errorf("expected filename to end with %s, got %s", expectedFilename, output.FilePath)
	}

	// Verify content
	content, err := os.ReadFile(output.FilePath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}
	contentStr := string(content)

	// Check for expected sections
	if !strings.Contains(contentStr, "# Daily Work Report") {
		t.Error("expected markdown to contain header")
	}
	if !strings.Contains(contentStr, "Work") {
		t.Error("expected markdown to contain work entry")
	}
	if !strings.Contains(contentStr, "Productive day!") {
		t.Error("expected markdown to contain notes")
	}
}

func TestExportDay_EmptyRecord(t *testing.T) {
	tempDir := t.TempDir()
	repo := NewMockDayRecordRepository()
	useCase := NewExportDayUseCase(repo, tempDir, newTestExportGenerator())

	input := ExportDayInput{Date: "2026-03-28"}
	_, err := useCase.Execute(input)

	if err == nil {
		t.Error("expected error for empty record")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error message to mention 'empty', got %v", err)
	}
}

func TestExportDay_MissingDate(t *testing.T) {
	tempDir := t.TempDir()
	repo := NewMockDayRecordRepository()
	useCase := NewExportDayUseCase(repo, tempDir, newTestExportGenerator())

	input := ExportDayInput{Date: ""}
	_, err := useCase.Execute(input)

	if err == nil {
		t.Error("expected error for missing date")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected error message to mention 'required', got %v", err)
	}
}

func TestExportDay_ExportDirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()
	// Use a nested directory that doesn't exist yet
	exportDir := filepath.Join(tempDir, "nested", "exports")

	repo := NewMockDayRecordRepository()
	useCase := NewExportDayUseCase(repo, exportDir, newTestExportGenerator())

	// Add a record
	record := model.DayRecord{
		Date: "2026-03-28",
		Entries: []model.WorkEntry{
			{ID: "1", Task: "Work", DurationMin: 30, IsBreak: false},
		},
	}
	_ = repo.Save(record)

	// Execute export
	input := ExportDayInput{Date: "2026-03-28"}
	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(exportDir); os.IsNotExist(err) {
		t.Error("expected export directory to be created")
	}

	// Verify file exists
	if _, err := os.Stat(output.FilePath); os.IsNotExist(err) {
		t.Error("expected export file to exist")
	}
}
