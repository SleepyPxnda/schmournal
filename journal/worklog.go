package journal

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDuration parses flexible duration strings:
// "1h 30m", "1h30m", "90m", "90min", "1.5h", "2h", "45m".
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "—" {
		return 0, fmt.Errorf("empty duration")
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	var hours, mins float64
	if n, _ := fmt.Sscanf(s, "%fh %fm", &hours, &mins); n == 2 {
		return time.Duration((hours*60+mins)*float64(time.Minute)), nil
	}
	if n, _ := fmt.Sscanf(s, "%fh %fmin", &hours, &mins); n == 2 {
		return time.Duration((hours*60+mins)*float64(time.Minute)), nil
	}
	if strings.HasSuffix(s, "h") {
		if h, err := strconv.ParseFloat(strings.TrimSuffix(s, "h"), 64); err == nil {
			return time.Duration(h * float64(time.Hour)), nil
		}
	}
	s2 := strings.TrimSuffix(strings.TrimSuffix(s, "min"), "m")
	if m, err := strconv.ParseFloat(strings.TrimSpace(s2), 64); err == nil && m > 0 {
		return time.Duration(m * float64(time.Minute)), nil
	}
	return 0, fmt.Errorf("cannot parse duration %q – try: 1h 30m, 45m, 2h", s)
}

// FormatDuration converts a duration to a human-readable string like "1h 30m".
func FormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dm", m)
	}
}


