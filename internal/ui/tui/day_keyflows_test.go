package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

func newDayViewTestModel(t *testing.T) Model {
	t.Helper()

	cfg := model.DefaultAppConfig()
	m := New(cfg, "", "test", nil, nil)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)
	m.ui.Current = stateDayView
	m.day.Record = DayRecord{
		Date: "2026-03-30",
		Entries: []WorkEntry{
			{ID: "e1", Task: "Initial task", DurationMin: 30},
		},
	}
	m.day.Selection.DayTab = 0
	m.day.Selection.Pane = 0
	m.day.Selection.EntryIdx = 0
	m.day.Viewport.SetContent(m.renderDayContent())
	return m
}

func dayKeyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func TestDayViewEditWithSelectedEntryOpensWorkForm(t *testing.T) {
	m := newDayViewTestModel(t)

	updated, _ := m.handleDayViewKey(dayKeyMsg(m.context.Config.Keybinds.Day.Edit))
	got := updated.(Model)

	if got.ui.Current != stateWorkForm {
		t.Fatalf("expected stateWorkForm, got %v", got.ui.Current)
	}
	if got.workForm.EditEntryIdx != 0 {
		t.Fatalf("expected edit index 0, got %d", got.workForm.EditEntryIdx)
	}
	if got.workForm.TaskInput.Value() != "Initial task" {
		t.Fatalf("expected task input to preload selected entry task, got %q", got.workForm.TaskInput.Value())
	}
}

func TestDayViewEditWithoutSelectionOpensNotesEditor(t *testing.T) {
	m := newDayViewTestModel(t)
	m.day.Selection.EntryIdx = -1

	updated, _ := m.handleDayViewKey(dayKeyMsg(m.context.Config.Keybinds.Day.Edit))
	got := updated.(Model)

	if got.ui.Current != stateNotesEditor {
		t.Fatalf("expected stateNotesEditor, got %v", got.ui.Current)
	}
}

func TestDayViewDeleteWithSelectedEntryOpensConfirmDelete(t *testing.T) {
	m := newDayViewTestModel(t)

	updated, _ := m.handleDayViewKey(dayKeyMsg(m.context.Config.Keybinds.Day.Delete))
	got := updated.(Model)

	if got.ui.Current != stateConfirmDelete {
		t.Fatalf("expected stateConfirmDelete, got %v", got.ui.Current)
	}
	if got.delete.Day {
		t.Fatalf("expected entry delete confirmation, got day delete")
	}
	if got.delete.Idx != 0 {
		t.Fatalf("expected delete index 0, got %d", got.delete.Idx)
	}
	if got.delete.PrevState != stateDayView {
		t.Fatalf("expected previous state day view, got %v", got.delete.PrevState)
	}
}

func TestDayViewTodoOverviewKeyFromSummaryFocusesTodoPane(t *testing.T) {
	m := newDayViewTestModel(t)
	m.day.Selection.DayTab = 1
	m.day.Selection.Pane = 0

	updated, _ := m.handleDayViewKey(dayKeyMsg(m.context.Config.Keybinds.Day.TodoOverview))
	got := updated.(Model)

	if got.day.Selection.DayTab != 0 {
		t.Fatalf("expected to switch to work-log tab, got tab %d", got.day.Selection.DayTab)
	}
	if got.day.Selection.Pane != 1 {
		t.Fatalf("expected todo pane focus, got pane %d", got.day.Selection.Pane)
	}
}

func TestDayViewEscStopsRunningClockAndReturnsToList(t *testing.T) {
	m := newDayViewTestModel(t)
	m.clock.Running = true
	m.clock.Task = "Active task"
	m.clock.Project = "Project A"

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)

	if got.ui.Current != stateList {
		t.Fatalf("expected stateList, got %v", got.ui.Current)
	}
	if got.clock.Running {
		t.Fatalf("expected clock to stop")
	}
	if got.clock.Task != "" || got.clock.Project != "" {
		t.Fatalf("expected clock metadata to be cleared, got task=%q project=%q", got.clock.Task, got.clock.Project)
	}
	if got.status.Message != "⏱ Clock stopped" {
		t.Fatalf("expected stop-clock status message, got %q", got.status.Message)
	}
	if got.status.IsError {
		t.Fatalf("expected non-error status when stopping clock")
	}
}

