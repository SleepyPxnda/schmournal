package ui

import "github.com/charmbracelet/lipgloss"

// ── Catppuccin Mocha palette ──────────────────────────────────────────────────
const (
	cRosewater = "#f5e0dc"
	cFlamingo  = "#f2cdcd"
	cPink      = "#f5c2e7"
	cMauve     = "#cba6f7"
	cRed       = "#f38ba8"
	cMaroon    = "#eba0ac"
	cPeach     = "#fab387"
	cYellow    = "#f9e2af"
	cGreen     = "#a6e3a1"
	cTeal      = "#94e2d5"
	cSky       = "#89dceb"
	cSapphire  = "#74c7ec"
	cBlue      = "#89b4fa"
	cLavender  = "#b4befe"
	cText      = "#cdd6f4"
	cSubtext1  = "#bac2de"
	cSubtext0  = "#a6adc8"
	cOverlay2  = "#9399b2"
	cOverlay1  = "#7f849c"
	cOverlay0  = "#6c7086"
	cSurface2  = "#585b70"
	cSurface1  = "#45475a"
	cSurface0  = "#313244"
	cBase      = "#1e1e2e"
	cMantle    = "#181825"
	cCrust     = "#11111b"
)

// ── Chrome styles ─────────────────────────────────────────────────────────────

var (
	headerTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cMauve)).
				Bold(true).
				Padding(0, 1)

	headerSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cSubtext0)).
				Padding(0, 1)

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cSurface2))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cOverlay0))

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cBlue))

	statusSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cGreen))

	statusErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cRed))

	confirmBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cSurface2)).
			Padding(1, 4)

	confirmTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cText)).
				Bold(true)

	confirmYesStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cGreen)).
			Bold(true)

	confirmNoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cRed)).
			Bold(true)

	editorBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(cSurface2)).
				Padding(0, 1)

	// ── Work log form styles ─────────────────────────────────────────────────

	formBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cMauve)).
			Padding(1, 3)

	formLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cSubtext1)).
			Bold(true)

	formHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cOverlay0))

	formActiveInputStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(lipgloss.Color(cMauve)).
				Padding(0, 1)

	formInactiveInputStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(lipgloss.Color(cSurface2)).
				Padding(0, 1)

	workLogBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cBase)).
				Background(lipgloss.Color(cPeach)).
				Bold(true).
				Padding(0, 1)

	breakLogBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cBase)).
				Background(lipgloss.Color(cTeal)).
				Bold(true).
				Padding(0, 1)
)

// ── List delegate styles ──────────────────────────────────────────────────────

var (
	listNormalTitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cText)).
			Padding(0, 0, 0, 2)

	listNormalDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cSubtext0)).
			Padding(0, 0, 0, 2)

	listSelectedTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cMauve)).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(lipgloss.Color(cMauve)).
				Padding(0, 0, 0, 1).
				Bold(true)

	listSelectedDesc = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cLavender)).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(lipgloss.Color(cMauve)).
				Padding(0, 0, 0, 1)

	listDimmedTitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cOverlay1)).
			Padding(0, 0, 0, 2)

	listDimmedDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cOverlay0)).
			Padding(0, 0, 0, 2)

	listFilterMatch = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cYellow))

	listNonWorkDayTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cMaroon)).
				Padding(0, 0, 0, 2)

	listNonWorkDayDesc = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cMaroon)).
				Padding(0, 0, 0, 2)

	listNonWorkDaySelectedTitle = lipgloss.NewStyle().
					Foreground(lipgloss.Color(cMaroon)).
					Border(lipgloss.NormalBorder(), false, false, false, true).
					BorderForeground(lipgloss.Color(cMaroon)).
					Padding(0, 0, 0, 1).
					Bold(true)

	listNonWorkDaySelectedDesc = lipgloss.NewStyle().
					Foreground(lipgloss.Color(cMaroon)).
					Border(lipgloss.NormalBorder(), false, false, false, true).
					BorderForeground(lipgloss.Color(cMaroon)).
					Padding(0, 0, 0, 1)
)

// ── Day view styles ───────────────────────────────────────────────────────────
var (
	dayViewSectionStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color(cLavender)).Bold(true)
	weekNonWorkDayStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color(cMaroon)).Bold(true)
	dayViewDividerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color(cSurface2))
	dayViewLabelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(cSubtext1))
	dayViewValueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(cText)).Bold(true)
	dayViewMutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(cOverlay1))
	dayViewTotalsStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(cSubtext0))
	dayViewWarnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(cPeach)).Bold(true)
	dayViewNotesStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(cText))
	selectedEntryStyle  = lipgloss.NewStyle().Background(lipgloss.Color(cSurface0)).Foreground(lipgloss.Color(cMauve)).Bold(true)
	normalEntryStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(cText))
	breakEntryStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(cTeal))

	// clockPanelBorderStyle wraps the inline clock panel in the Work Log tab.
	// It draws a left border to visually separate it from the entries column.
	clockPanelBorderStyle = lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color(cSurface2))

	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cBase)).
			Background(lipgloss.Color(cMauve)).
			Bold(true).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cSubtext0)).
				Padding(0, 1)
)

// ── Stats panel styles ────────────────────────────────────────────────────────
var (
	statsBlockFilledStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(cGreen))
	statsBlockEmptyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(cSurface2))
	statsBlockFutureStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(cSurface1))
	statsBlockNonWorkStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(cMaroon))
	statsLabelStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color(cSubtext0))
	statsValueStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color(cText)).Bold(true)
	statsStreakStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color(cPeach)).Bold(true)

	eomBannerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cBase)).
			Background(lipgloss.Color(cRed)).
			Bold(true).
			Width(0). // set at render time
			Align(lipgloss.Center)
)
