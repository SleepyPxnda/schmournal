package journal

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		// Standard Go duration strings pass through time.ParseDuration.
		{"1h30m", 90 * time.Minute, false},
		{"45m", 45 * time.Minute, false},
		{"2h", 2 * time.Hour, false},
		// Space-separated variants.
		{"1h 30m", 90 * time.Minute, false},
		{"1h 30min", 90 * time.Minute, false},
		// Float hours.
		{"1.5h", 90 * time.Minute, false},
		// Bare minutes with "min" suffix.
		{"90min", 90 * time.Minute, false},
		{"90m", 90 * time.Minute, false},
		// Case-insensitive.
		{"2H", 2 * time.Hour, false},
		{"45M", 45 * time.Minute, false},
		// Leading/trailing whitespace is trimmed.
		{"  30m  ", 30 * time.Minute, false},
		// Error cases.
		{"", 0, true},
		{"—", 0, true},
		{"abc", 0, true},
		// Zero is accepted by time.ParseDuration but represents no time logged.
		{"0m", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseDuration(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDuration(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{90 * time.Minute, "1h 30m"},
		{2 * time.Hour, "2h"},
		{45 * time.Minute, "45m"},
		{0, "0m"},
		{30 * time.Minute, "30m"},
		{120 * time.Minute, "2h"},
		{150 * time.Minute, "2h 30m"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatDuration(tt.input)
			if got != tt.want {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseDurationRoundTrip(t *testing.T) {
	// FormatDuration output should always be re-parseable.
	durations := []time.Duration{
		15 * time.Minute,
		30 * time.Minute,
		45 * time.Minute,
		60 * time.Minute,
		90 * time.Minute,
		120 * time.Minute,
		150 * time.Minute,
	}
	for _, d := range durations {
		s := FormatDuration(d)
		got, err := ParseDuration(s)
		if err != nil {
			t.Errorf("ParseDuration(FormatDuration(%v)) = error: %v", d, err)
			continue
		}
		if got != d {
			t.Errorf("ParseDuration(FormatDuration(%v)) = %v, want %v", d, got, d)
		}
	}
}
