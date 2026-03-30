package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// statsDurFieldW is the fixed column width used for rendered durations in the
// stats view (e.g. "10h30m   ") so bars and columns align across rows.
const statsDurFieldW = 9

func (m Model) viewStats() string {
	header := m.renderHeader("📊  Stats", "Overview")
	tabBar := m.renderStatsTabBar()
	footer := m.renderFooter([][2]string{
		{"←/→", "switch tab"},
		{"j/k", "scroll"},
		{"esc", "back"},
	})
	return lipgloss.JoinVertical(lipgloss.Left, header, tabBar, m.day.Viewport.View(), footer)
}

// renderStatsTabContent dispatches to the appropriate tab renderer.
func (m Model) renderStatsTabContent() string {
	switch m.stats.Tab {
	case 1:
		return m.renderStatsMonthly()
	case 2:
		return m.renderStatsYearly()
	case 3:
		return m.renderStatsAllTime()
	default:
		return m.renderStatsOverview()
	}
}

// renderStatsOverview renders the Overview tab: streaks + 16-week activity heatmap.
func (m Model) renderStatsOverview() string {
	now := time.Now()
	innerW := m.window.Width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	// Build a set of dates for which records exist.
	dated := make(map[string]bool, len(m.listState.Records))
	for _, r := range m.listState.Records {
		dated[r.Date] = true
	}

	var b strings.Builder
	b.WriteString("\n")

	// ── Streaks ───────────────────────────────────────────────────────────────
	b.WriteString("  " + dayViewSectionStyle.Render("Streaks") + "\n")
	b.WriteString("  " + div + "\n")

	// Current streak: consecutive working days going back from today.
	// Non-working days are skipped so that a weekend never breaks the count.
	// The iteration limit is derived from the oldest record so the loop is
	// always bounded by real data and never underestimates long streaks.
	currentStreak := 0
	for i := 0; i < streakIterLimit(m.listState.Records, now); i++ {
		check := now.AddDate(0, 0, -i)
		dateStr := check.Format("2006-01-02")
		if dated[dateStr] {
			currentStreak++
		} else if m.effectiveIsWorkDay(check) {
			// A working day with no entry breaks the streak.
			break
		}
		// Non-working day without an entry: continue (streak passes through).
	}

	// Longest streak: scan all dated records.
	longestStreak := 0
	if len(m.listState.Records) > 0 {
		var days []time.Time
		for _, r := range m.listState.Records {
			if t, err := r.ParseDate(); err == nil {
				days = append(days, t)
			}
		}
		sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
		if len(days) > 0 {
			run := 1
			for i := 1; i < len(days); i++ {
				// Fast path: directly adjacent days are always consecutive.
				// Otherwise check that every day in the gap is a non-working day
				// (e.g. Fri → Mon across a weekend).
				gapOK := days[i-1].AddDate(0, 0, 1).Equal(days[i])
				if !gapOK {
					gapOK = true
					for d := days[i-1].AddDate(0, 0, 1); d.Before(days[i]); d = d.AddDate(0, 0, 1) {
						if m.effectiveIsWorkDay(d) {
							gapOK = false
							break
						}
					}
				}
				if gapOK {
					run++
					if run > longestStreak {
						longestStreak = run
					}
				} else {
					if run > longestStreak {
						longestStreak = run
					}
					run = 1
				}
			}
			if run > longestStreak {
				longestStreak = run
			}
		}
	}

	totalDays := len(m.listState.Records)
	var streakLine string
	if currentStreak > 0 {
		streakLine = "  🔥 " + statsStreakStyle.Render(fmt.Sprintf("%d", currentStreak)) +
			statsLabelStyle.Render(" day streak")
	} else {
		streakLine = "  " + statsLabelStyle.Render("No active streak")
	}
	streakLine += statsLabelStyle.Render("  ·  Longest: ") +
		statsValueStyle.Render(fmt.Sprintf("%d", longestStreak)) +
		statsLabelStyle.Render(" days") +
		statsLabelStyle.Render("  ·  Total logged: ") +
		statsValueStyle.Render(fmt.Sprintf("%d", totalDays)) +
		statsLabelStyle.Render(" days")
	b.WriteString(streakLine + "\n\n")

	// ── Activity heatmap (last 16 weeks) ──────────────────────────────────────
	b.WriteString("  " + dayViewSectionStyle.Render("Activity  (last 16 weeks)") + "\n")
	b.WriteString("  " + div + "\n")

	// Find Monday of current week.
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	thisMonday := now.AddDate(0, 0, -(weekday - 1))

	const heatmapWeeks = 16
	const filled = "▓"
	const empty = "░"

	// Header row: day initials.
	dayInitials := [7]string{"M", "T", "W", "T", "F", "S", "S"}
	var headerCols []string
	headerCols = append(headerCols, "        ") // month label column width
	for _, lbl := range dayInitials {
		headerCols = append(headerCols, statsLabelStyle.Render(lbl))
	}
	b.WriteString("  " + strings.Join(headerCols, " ") + "\n")

	for w := heatmapWeeks - 1; w >= 0; w-- {
		monday := thisMonday.AddDate(0, 0, -w*7)
		// Month label for this row (show if first week or month changes).
		var monthLabel string
		if w == heatmapWeeks-1 || monday.Month() != monday.AddDate(0, 0, -7).Month() {
			monthLabel = monday.Format("Jan")
		}
		row := fmt.Sprintf("  %-6s  ", monthLabel)
		for i := 0; i < 7; i++ {
			d := monday.AddDate(0, 0, i)
			dateStr := d.Format("2006-01-02")
			var block string
			switch {
			case dated[dateStr]:
				block = statsBlockFilledStyle.Render(filled)
			case d.After(now):
				block = statsBlockFutureStyle.Render(empty)
			case !m.effectiveIsWorkDay(d):
				block = statsBlockNonWorkStyle.Render(empty)
			default:
				block = statsBlockEmptyStyle.Render(empty)
			}
			if i > 0 {
				row += " "
			}
			row += block
		}
		b.WriteString(row + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

// renderStatsMonthly renders the Monthly tab.
func (m Model) renderStatsMonthly() string {
	now := time.Now()
	innerW := m.window.Width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	monthKey := now.Format("2006-01")
	monthName := now.Format("January 2006")

	projMap := make(map[string]time.Duration)
	var totalWork time.Duration
	days := 0
	for _, r := range m.listState.Records {
		if !strings.HasPrefix(r.Date, monthKey) {
			continue
		}
		days++
		for _, e := range r.Entries {
			if e.IsBreak {
				continue
			}
			proj := e.Project
			if proj == "" {
				proj = "Other"
			}
			projMap[proj] += e.Duration()
			totalWork += e.Duration()
		}
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + dayViewSectionStyle.Render("This Month") +
		dayViewMutedStyle.Render("  "+monthName) +
		"    " +
		statsLabelStyle.Render("Total: ") + statsValueStyle.Render(formatDuration(totalWork)) +
		statsLabelStyle.Render(fmt.Sprintf("  ·  %d days", days)) +
		"\n")
	b.WriteString("  " + div + "\n")
	if len(projMap) == 0 {
		b.WriteString("  " + dayViewMutedStyle.Render("No work entries this month") + "\n")
	} else {
		b.WriteString(renderProjectBars(projMap, totalWork))
	}
	return b.String()
}

// renderStatsYearly renders the Yearly tab.
func (m Model) renderStatsYearly() string {
	now := time.Now()
	innerW := m.window.Width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	yearKey := now.Format("2006")

	projMap := make(map[string]time.Duration)
	monthlyWork := make(map[string]time.Duration)
	var totalWork time.Duration
	days := 0
	for _, r := range m.listState.Records {
		if !strings.HasPrefix(r.Date, yearKey) {
			continue
		}
		days++
		mk := r.Date[:7]
		for _, e := range r.Entries {
			if e.IsBreak {
				continue
			}
			proj := e.Project
			if proj == "" {
				proj = "Other"
			}
			projMap[proj] += e.Duration()
			totalWork += e.Duration()
			monthlyWork[mk] += e.Duration()
		}
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + dayViewSectionStyle.Render("This Year") +
		dayViewMutedStyle.Render("  "+yearKey) +
		"    " +
		statsLabelStyle.Render("Total: ") + statsValueStyle.Render(formatDuration(totalWork)) +
		statsLabelStyle.Render(fmt.Sprintf("  ·  %d days", days)) +
		"\n")
	b.WriteString("  " + div + "\n")

	if totalWork == 0 {
		b.WriteString("  " + dayViewMutedStyle.Render("No work entries this year") + "\n")
		return b.String()
	}

	// Month-by-month bar chart (relative to busiest month).
	var maxMonthWork time.Duration
	for _, d := range monthlyWork {
		if d > maxMonthWork {
			maxMonthWork = d
		}
	}
	const monthBarW = 16
	for mo := 1; mo <= int(now.Month()); mo++ {
		mk := fmt.Sprintf("%s-%02d", yearKey, mo)
		d := monthlyWork[mk]
		moLabel := time.Month(mo).String()[:3]
		pct := float64(d) / float64(maxMonthWork)
		filledCount := int(pct * monthBarW)
		bar := statsBlockFilledStyle.Render(strings.Repeat("█", filledCount)) +
			statsBlockEmptyStyle.Render(strings.Repeat("░", monthBarW-filledCount))
		durStr := fmt.Sprintf("%-*s", statsDurFieldW, formatDuration(d))
		b.WriteString("  " + statsLabelStyle.Render(moLabel+"  ") +
			bar + "  " +
			statsValueStyle.Render(durStr) + "\n")
	}
	b.WriteString("\n")
	b.WriteString("  " + div + "\n")
	b.WriteString(renderProjectBars(projMap, totalWork))
	return b.String()
}

// renderStatsAllTime renders the All-time tab.
func (m Model) renderStatsAllTime() string {
	innerW := m.window.Width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	projMap := make(map[string]time.Duration)
	var totalWork time.Duration
	for _, r := range m.listState.Records {
		for _, e := range r.Entries {
			if e.IsBreak {
				continue
			}
			proj := e.Project
			if proj == "" {
				proj = "Other"
			}
			projMap[proj] += e.Duration()
			totalWork += e.Duration()
		}
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + dayViewSectionStyle.Render("All-time Top Projects") +
		"    " +
		statsLabelStyle.Render("Total: ") + statsValueStyle.Render(formatDuration(totalWork)) +
		"\n")
	b.WriteString("  " + div + "\n")
	if len(projMap) == 0 {
		b.WriteString("  " + dayViewMutedStyle.Render("No work entries") + "\n")
	} else {
		b.WriteString(renderProjectBars(projMap, totalWork))
	}
	return b.String()
}

// renderProjectBars renders a sorted list of project durations with inline bar
// charts relative to the total work duration. All bars start at the same column.
func renderProjectBars(projMap map[string]time.Duration, total time.Duration) string {
	type ps struct {
		name string
		dur  time.Duration
	}
	var projects []ps
	for name, dur := range projMap {
		projects = append(projects, ps{name, dur})
	}
	// Sort descending by duration.
	sort.Slice(projects, func(i, j int) bool { return projects[i].dur > projects[j].dur })

	// Find the widest rendered project name so all bars start at the same column.
	maxNameW := 0
	for _, p := range projects {
		if w := lipgloss.Width(p.name); w > maxNameW {
			maxNameW = w
		}
	}

	const barW = 20
	var sb strings.Builder
	for _, p := range projects {
		pct := float64(0)
		if total > 0 {
			pct = float64(p.dur) / float64(total)
		}
		filledCount := int(pct * barW)
		bar := statsBlockFilledStyle.Render(strings.Repeat("█", filledCount)) +
			statsBlockEmptyStyle.Render(strings.Repeat("░", barW-filledCount))
		durStr := fmt.Sprintf("%-*s", statsDurFieldW, formatDuration(p.dur))
		pctStr := fmt.Sprintf("%3.0f%%", pct*100)
		// Width() on the style pads the name to a uniform column width.
		nameStr := dayViewSectionStyle.Width(maxNameW).Render(p.name)
		sb.WriteString("  " + dayViewTotalsStyle.Render(durStr) +
			"  " + nameStr +
			"  " + bar +
			"  " + statsLabelStyle.Render(pctStr) + "\n")
	}
	return sb.String()
}
