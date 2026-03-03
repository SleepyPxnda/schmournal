package journal

import (
	"strings"
	"testing"
	"time"
)

func TestWorkTotals(t *testing.T) {
	rec := DayRecord{
		Entries: []WorkEntry{
			{DurationMin: 60, IsBreak: false},
			{DurationMin: 30, IsBreak: false},
			{DurationMin: 15, IsBreak: true},
		},
	}
	work, breaks, total := rec.WorkTotals()

	if work != 90*time.Minute {
		t.Errorf("work = %v, want 90m", work)
	}
	if breaks != 15*time.Minute {
		t.Errorf("breaks = %v, want 15m", breaks)
	}
	if total != 105*time.Minute {
		t.Errorf("total = %v, want 105m", total)
	}
}

func TestWorkTotalsEmpty(t *testing.T) {
	rec := DayRecord{}
	work, breaks, total := rec.WorkTotals()
	if work != 0 || breaks != 0 || total != 0 {
		t.Errorf("empty record: got work=%v breaks=%v total=%v, want all 0", work, breaks, total)
	}
}

func TestWorkTotalsOnlyBreaks(t *testing.T) {
	rec := DayRecord{
		Entries: []WorkEntry{
			{DurationMin: 30, IsBreak: true},
		},
	}
	work, breaks, total := rec.WorkTotals()
	if work != 0 {
		t.Errorf("work = %v, want 0", work)
	}
	if breaks != 30*time.Minute {
		t.Errorf("breaks = %v, want 30m", breaks)
	}
	if total != 30*time.Minute {
		t.Errorf("total = %v, want 30m", total)
	}
}

func TestDayDuration(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		end       string
		wantDur   time.Duration
		wantValid bool
	}{
		{"normal day", "09:00", "17:00", 8 * time.Hour, true},
		{"half day", "09:00", "13:00", 4 * time.Hour, true},
		{"missing start", "", "17:00", 0, false},
		{"missing end", "09:00", "", 0, false},
		{"both missing", "", "", 0, false},
		{"end before start", "17:00", "09:00", 0, false},
		{"same time", "09:00", "09:00", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := DayRecord{StartTime: tt.start, EndTime: tt.end}
			dur, valid := rec.DayDuration()
			if valid != tt.wantValid {
				t.Errorf("DayDuration() valid = %v, want %v", valid, tt.wantValid)
			}
			if valid && dur != tt.wantDur {
				t.Errorf("DayDuration() = %v, want %v", dur, tt.wantDur)
			}
		})
	}
}

func TestSummary(t *testing.T) {
	t.Run("no entries", func(t *testing.T) {
		rec := DayRecord{}
		if got := rec.Summary(); got != "No entries" {
			t.Errorf("Summary() = %q, want %q", got, "No entries")
		}
	})

	t.Run("single work entry", func(t *testing.T) {
		rec := DayRecord{
			Entries: []WorkEntry{
				{Task: "coding", DurationMin: 60, IsBreak: false},
			},
		}
		got := rec.Summary()
		if got == "" {
			t.Error("Summary() returned empty string for single entry")
		}
		// Should mention "1 entry" (singular).
		if !strings.Contains(got, "1 entry") {
			t.Errorf("Summary() = %q, expected to contain %q", got, "1 entry")
		}
	})

	t.Run("multiple entries with breaks", func(t *testing.T) {
		rec := DayRecord{
			Entries: []WorkEntry{
				{Task: "coding", DurationMin: 60, IsBreak: false},
				{Task: "review", DurationMin: 30, IsBreak: false},
				{Task: "lunch", DurationMin: 30, IsBreak: true},
			},
		}
		got := rec.Summary()
		if !strings.Contains(got, "3 entries") {
			t.Errorf("Summary() = %q, expected to contain %q", got, "3 entries")
		}
		if !strings.Contains(got, "Work:") {
			t.Errorf("Summary() = %q, expected to contain %q", got, "Work:")
		}
		if !strings.Contains(got, "Breaks:") {
			t.Errorf("Summary() = %q, expected to contain %q", got, "Breaks:")
		}
	})
}

func TestParseDate(t *testing.T) {
	rec := DayRecord{Date: "2024-01-15"}
	got, err := rec.ParseDate()
	if err != nil {
		t.Fatalf("ParseDate() error: %v", err)
	}
	want := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("ParseDate() = %v, want %v", got, want)
	}
}

func TestParseDateInvalid(t *testing.T) {
	rec := DayRecord{Date: "not-a-date"}
	if _, err := rec.ParseDate(); err == nil {
		t.Error("ParseDate() expected error for invalid date, got nil")
	}
}
