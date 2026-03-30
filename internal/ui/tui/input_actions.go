package tui

import domainmodel "github.com/sleepypxnda/schmournal/internal/domain/model"

type listAction int

const (
	listActionNone listAction = iota
	listActionQuit
	listActionOpenToday
	listActionOpenDate
	listActionOpenSelected
	listActionDeleteSelected
	listActionOpenWeekView
	listActionOpenStatsView
	listActionOpenWorkspacePicker
)

func listActionForKey(key string, kb domainmodel.ListKeybinds) listAction {
	switch key {
	case kb.Quit, "esc":
		return listActionQuit
	case kb.OpenToday:
		return listActionOpenToday
	case kb.OpenDate:
		return listActionOpenDate
	case "enter":
		return listActionOpenSelected
	case kb.Delete:
		return listActionDeleteSelected
	case kb.WeekView:
		return listActionOpenWeekView
	case kb.StatsView:
		return listActionOpenStatsView
	case kb.SwitchWorkspace:
		return listActionOpenWorkspacePicker
	default:
		return listActionNone
	}
}

type workspacePickerAction int

const (
	workspacePickerActionNone workspacePickerAction = iota
	workspacePickerActionMoveDown
	workspacePickerActionMoveUp
	workspacePickerActionConfirm
	workspacePickerActionCancel
)

func workspacePickerActionForKey(key string, listQuit string) workspacePickerAction {
	switch key {
	case "j", "down":
		return workspacePickerActionMoveDown
	case "k", "up":
		return workspacePickerActionMoveUp
	case "enter":
		return workspacePickerActionConfirm
	case "esc", listQuit:
		return workspacePickerActionCancel
	default:
		return workspacePickerActionNone
	}
}

type statsAction int

const (
	statsActionNone statsAction = iota
	statsActionBack
	statsActionLeft
	statsActionRight
)

func statsActionForKey(key string, listQuit string) statsAction {
	switch key {
	case "esc", listQuit:
		return statsActionBack
	case "left":
		return statsActionLeft
	case "right":
		return statsActionRight
	default:
		return statsActionNone
	}
}

type weekViewAction int

const (
	weekViewActionNone weekViewAction = iota
	weekViewActionBack
	weekViewActionLeft
	weekViewActionRight
)

func weekViewActionForKey(key string, listQuit string) weekViewAction {
	switch key {
	case "esc", listQuit:
		return weekViewActionBack
	case "left":
		return weekViewActionLeft
	case "right":
		return weekViewActionRight
	default:
		return weekViewActionNone
	}
}

