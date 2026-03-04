package journal

import (
	"strings"
	"testing"
	"time"
)

// makeRec is a helper to build a DayRecord with the given entries.
func makeRec(start, end string, entries ...WorkEntry) DayRecord {
	return DayRecord{
		Date:      "2024-06-10",
		StartTime: start,
		EndTime:   end,
		Entries:   entries,
	}
}

func work(task, project string, mins int) WorkEntry {
	return WorkEntry{ID: "1", Task: task, Project: project, DurationMin: mins, IsBreak: false}
}

func brk(label string, mins int) WorkEntry {
	return WorkEntry{ID: "2", Task: label, DurationMin: mins, IsBreak: true}
}

// ── consolidateEntries ────────────────────────────────────────────────────────

func TestConsolidateEntries(t *testing.T) {
	t.Run("no duplicates unchanged", func(t *testing.T) {
		entries := []WorkEntry{
			work("TaskA", "", 30),
			work("TaskB", "", 60),
		}
		got := consolidateEntries(entries)
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
	})

	t.Run("duplicates merged", func(t *testing.T) {
		entries := []WorkEntry{
			work("TaskA", "", 30),
			work("TaskA", "", 45),
		}
		got := consolidateEntries(entries)
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].DurationMin != 75 {
			t.Errorf("DurationMin = %d, want 75", got[0].DurationMin)
		}
	})

	t.Run("case-sensitive task name", func(t *testing.T) {
		entries := []WorkEntry{
			work("taskA", "", 30),
			work("TaskA", "", 30),
		}
		got := consolidateEntries(entries)
		// Different case → not merged.
		if len(got) != 2 {
			t.Errorf("len = %d, want 2 (case-sensitive)", len(got))
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		got := consolidateEntries(nil)
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})
}

// ── ExportDay ─────────────────────────────────────────────────────────────────

func TestExportDayHeader(t *testing.T) {
	rec := makeRec("09:00", "17:00", work("Coding", "Backend", 120))
	out := ExportDay(rec)

	if !strings.Contains(out, "# Daily Work Report") {
		t.Error("export missing report header")
	}
	// Date is rendered as "Monday, January 2, 2006" for 2024-06-10.
	if !strings.Contains(out, "2024") {
		t.Error("export missing year in header")
	}
}

func TestExportDayWorkDay(t *testing.T) {
	rec := makeRec("09:00", "17:00", work("Coding", "Backend", 120))
	out := ExportDay(rec)

	if !strings.Contains(out, "09:00") {
		t.Error("export missing start time")
	}
	if !strings.Contains(out, "17:00") {
		t.Error("export missing end time")
	}
	// Day duration = 8h
	if !strings.Contains(out, "8h") {
		t.Error("export missing day duration")
	}
}

func TestExportDayWorkItems(t *testing.T) {
	rec := makeRec("", "",
		work("Feature development", "Backend", 90),
		work("Code review", "Backend", 30),
	)
	out := ExportDay(rec)

	if !strings.Contains(out, "Backend") {
		t.Error("export missing project name")
	}
	if !strings.Contains(out, "Feature development") {
		t.Error("export missing task name")
	}
	// Two work entries in same project: subtotal = 2h.
	if !strings.Contains(out, "2h") {
		t.Error("export missing 2h subtotal")
	}
}

func TestExportDayBreaks(t *testing.T) {
	rec := makeRec("", "", brk("Lunch", 60), brk("Coffee", 15))
	out := ExportDay(rec)

	if !strings.Contains(out, "Lunch") {
		t.Error("export missing lunch break")
	}
	if !strings.Contains(out, "Coffee") {
		t.Error("export missing coffee break")
	}
	if !strings.Contains(out, "1h 15m") {
		t.Error("export missing total breaks duration")
	}
}

func TestExportDayNoEntries(t *testing.T) {
	rec := makeRec("", "")
	out := ExportDay(rec)

	if !strings.Contains(out, "_No work items logged._") {
		t.Error("export missing 'no work items' placeholder")
	}
	if !strings.Contains(out, "_No breaks logged._") {
		t.Error("export missing 'no breaks' placeholder")
	}
}

func TestExportDayConsolidatesWorkEntries(t *testing.T) {
	// Two entries with the same task in the same project should be consolidated.
	rec := makeRec("", "",
		work("Bug fix", "Backend", 30),
		work("Bug fix", "Backend", 45),
	)
	out := ExportDay(rec)

	// Consolidated: one row for "Bug fix" with 1h 15m total.
	count := strings.Count(out, "Bug fix")
	if count != 1 {
		t.Errorf("'Bug fix' appears %d times, want 1 (should be consolidated)", count)
	}
	if !strings.Contains(out, "1h 15m") {
		t.Error("export missing consolidated duration 1h 15m")
	}
}

func TestExportDayNotes(t *testing.T) {
	rec := makeRec("", "")
	rec.Notes = "These are my notes."
	out := ExportDay(rec)

	if !strings.Contains(out, "📝 Notes") {
		t.Error("export missing notes section")
	}
	if !strings.Contains(out, "These are my notes.") {
		t.Error("export missing note content")
	}
}

func TestExportDayNoProjectGroupedAsOther(t *testing.T) {
	// Entries without a project should appear under "Other".
	rec := makeRec("", "", work("Standup", "", 15))
	out := ExportDay(rec)

	if !strings.Contains(out, "Other") {
		t.Error("export missing 'Other' group for entries without a project")
	}
}

func TestExportDayTimestamp(t *testing.T) {
	rec := makeRec("", "")
	out := ExportDay(rec)

	// The export footer contains the current year.
	currentYear := time.Now().Format("2006")
	if !strings.Contains(out, currentYear) {
		t.Errorf("export missing current year %s in timestamp", currentYear)
	}
}
