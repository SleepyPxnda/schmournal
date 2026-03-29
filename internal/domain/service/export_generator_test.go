package service

import (
	"strings"
	"testing"
	"time"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

func TestExportGenerator_GenerateMarkdown_EmptyDay(t *testing.T) {
	formatter := NewDurationFormatter()
	consolidator := NewEntryConsolidator()
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 15, 30, 0, 0, time.UTC))
	generator := NewExportGenerator(formatter, consolidator, timeProvider)

	rec := model.DayRecord{
		Date:    "2026-03-28",
		Entries: []model.WorkEntry{},
	}

	markdown := generator.GenerateMarkdown(rec)

	// Check header (Saturday is correct for 2026-03-28)
	if !strings.Contains(markdown, "# Daily Work Report — Saturday, March 28, 2026") {
		t.Errorf("missing or incorrect header, got:\n%s", markdown)
	}

	// Check empty work items
	if !strings.Contains(markdown, "_No work items logged._") {
		t.Error("missing empty work items message")
	}

	// Check empty breaks
	if !strings.Contains(markdown, "_No breaks logged._") {
		t.Error("missing empty breaks message")
	}

	// Check footer timestamp
	if !strings.Contains(markdown, "_Exported 2026-03-28 15:30_") {
		t.Error("missing or incorrect export timestamp")
	}
}

func TestExportGenerator_GenerateMarkdown_WithWorkEntries(t *testing.T) {
	formatter := NewDurationFormatter()
	consolidator := NewEntryConsolidator()
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 15, 30, 0, 0, time.UTC))
	generator := NewExportGenerator(formatter, consolidator, timeProvider)

	rec := model.DayRecord{
		Date:      "2026-03-28",
		StartTime: "09:00",
		EndTime:   "17:00",
		Entries: []model.WorkEntry{
			{ID: "1", Task: "Code Review", Project: "Backend", DurationMin: 60, IsBreak: false},
			{ID: "2", Task: "Feature Dev", Project: "Backend", DurationMin: 120, IsBreak: false},
			{ID: "3", Task: "Meeting", Project: "Frontend", DurationMin: 30, IsBreak: false},
		},
	}

	markdown := generator.GenerateMarkdown(rec)

	// Check work day times
	if !strings.Contains(markdown, "| **Start** | 09:00 |") {
		t.Error("missing start time")
	}
	if !strings.Contains(markdown, "| **End** | 17:00 |") {
		t.Error("missing end time")
	}
	if !strings.Contains(markdown, "| **Day duration** | 8h |") {
		t.Error("missing day duration")
	}

	// Check project groups
	if !strings.Contains(markdown, "### 🗂 Backend") {
		t.Error("missing Backend project section")
	}
	if !strings.Contains(markdown, "### 🗂 Frontend") {
		t.Error("missing Frontend project section")
	}

	// Check tasks
	if !strings.Contains(markdown, "| Code Review | 1h |") {
		t.Error("missing Code Review task")
	}
	if !strings.Contains(markdown, "| Feature Dev | 2h |") {
		t.Error("missing Feature Dev task")
	}
	if !strings.Contains(markdown, "| Meeting | 30m |") {
		t.Error("missing Meeting task")
	}

	// Check total
	if !strings.Contains(markdown, "**Total work: 3h 30m**") {
		t.Error("missing or incorrect total work time")
	}
}

func TestExportGenerator_GenerateMarkdown_WithBreaks(t *testing.T) {
	formatter := NewDurationFormatter()
	consolidator := NewEntryConsolidator()
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 15, 30, 0, 0, time.UTC))
	generator := NewExportGenerator(formatter, consolidator, timeProvider)

	rec := model.DayRecord{
		Date: "2026-03-28",
		Entries: []model.WorkEntry{
			{ID: "1", Task: "Work", Project: "", DurationMin: 60, IsBreak: false},
			{ID: "2", Task: "Lunch", Project: "", DurationMin: 30, IsBreak: true},
			{ID: "3", Task: "Coffee", Project: "", DurationMin: 15, IsBreak: true},
			{ID: "4", Task: "Coffee", Project: "", DurationMin: 10, IsBreak: true},
		},
	}

	markdown := generator.GenerateMarkdown(rec)

	// Check breaks section
	if !strings.Contains(markdown, "## ☕ Breaks") {
		t.Error("missing breaks section")
	}

	// Check consolidated coffee breaks (15 + 10 = 25)
	if !strings.Contains(markdown, "| Coffee | 25m |") {
		t.Error("coffee breaks not consolidated correctly")
	}
	if !strings.Contains(markdown, "| Lunch | 30m |") {
		t.Error("missing lunch break")
	}

	// Check total breaks
	if !strings.Contains(markdown, "**Total breaks: 55m**") {
		t.Error("missing or incorrect total breaks")
	}
}

func TestExportGenerator_GenerateMarkdown_WithNotes(t *testing.T) {
	formatter := NewDurationFormatter()
	consolidator := NewEntryConsolidator()
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 15, 30, 0, 0, time.UTC))
	generator := NewExportGenerator(formatter, consolidator, timeProvider)

	rec := model.DayRecord{
		Date:    "2026-03-28",
		Entries: []model.WorkEntry{},
		Notes:   "Today was productive!\n\nAccomplished:\n- Feature X\n- Bug fix Y",
	}

	markdown := generator.GenerateMarkdown(rec)

	// Check notes section
	if !strings.Contains(markdown, "## 📝 Notes") {
		t.Error("missing notes section")
	}
	if !strings.Contains(markdown, "Today was productive!") {
		t.Error("missing notes content")
	}
	if !strings.Contains(markdown, "- Feature X") {
		t.Error("missing notes content (Feature X)")
	}
}

func TestExportGenerator_GenerateMarkdown_ProjectConsolidation(t *testing.T) {
	formatter := NewDurationFormatter()
	consolidator := NewEntryConsolidator()
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 15, 30, 0, 0, time.UTC))
	generator := NewExportGenerator(formatter, consolidator, timeProvider)

	rec := model.DayRecord{
		Date: "2026-03-28",
		Entries: []model.WorkEntry{
			{ID: "1", Task: "Review", Project: "ProjectA", DurationMin: 30, IsBreak: false},
			{ID: "2", Task: "Review", Project: "ProjectA", DurationMin: 20, IsBreak: false},
			{ID: "3", Task: "Coding", Project: "ProjectA", DurationMin: 60, IsBreak: false},
		},
	}

	markdown := generator.GenerateMarkdown(rec)

	// Check consolidated review (30 + 20 = 50)
	if !strings.Contains(markdown, "| Review | 50m |") {
		t.Error("Review tasks not consolidated correctly within ProjectA")
	}
	if !strings.Contains(markdown, "| Coding | 1h |") {
		t.Error("missing Coding task")
	}

	// Check subtotal for ProjectA (50 + 60 = 110)
	if !strings.Contains(markdown, "**Subtotal: 1h 50m**") {
		t.Error("missing or incorrect ProjectA subtotal")
	}
}

func TestExportGenerator_GenerateMarkdown_NoProjectMovedToEnd(t *testing.T) {
	formatter := NewDurationFormatter()
	consolidator := NewEntryConsolidator()
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 15, 30, 0, 0, time.UTC))
	generator := NewExportGenerator(formatter, consolidator, timeProvider)

	rec := model.DayRecord{
		Date: "2026-03-28",
		Entries: []model.WorkEntry{
			{ID: "1", Task: "Task1", Project: "", DurationMin: 30, IsBreak: false},
			{ID: "2", Task: "Task2", Project: "ProjectA", DurationMin: 60, IsBreak: false},
		},
	}

	markdown := generator.GenerateMarkdown(rec)

	// Find positions of "ProjectA" and "Other"
	posProjectA := strings.Index(markdown, "### 🗂 ProjectA")
	posOther := strings.Index(markdown, "### 🗂 Other")

	if posProjectA == -1 {
		t.Error("missing ProjectA section")
	}
	if posOther == -1 {
		t.Error("missing Other section (no project)")
	}

	// "Other" should come after "ProjectA"
	if posProjectA > posOther {
		t.Error("Other section should come after named projects")
	}
}
