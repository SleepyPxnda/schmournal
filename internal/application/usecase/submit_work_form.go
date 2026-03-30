package usecase

import (
	"fmt"
	"strings"
	"time"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
	"github.com/sleepypxnda/schmournal/internal/domain/repository"
	"github.com/sleepypxnda/schmournal/internal/domain/service"
)

type SubmitWorkFormInput struct {
	Date       string
	Task       string
	ProjectRaw string
	Duration   string
	IsBreak    bool
	EditEntry  int // -1=new, >=0 edit existing entry index
}

type SubmitWorkFormOutput struct {
	Record           DayRecordDTO
	Label            string
	SelectedEntryIdx int
}

type SubmitWorkFormUseCase struct {
	dayRepo        repository.DayRecordRepository
	durationParser *service.DurationParser
	timeProvider   service.TimeProvider
}

func NewSubmitWorkFormUseCase(dayRepo repository.DayRecordRepository, timeProvider service.TimeProvider) *SubmitWorkFormUseCase {
	return &SubmitWorkFormUseCase{
		dayRepo:        dayRepo,
		durationParser: service.NewDurationParser(),
		timeProvider:   timeProvider,
	}
}

func (uc *SubmitWorkFormUseCase) Execute(input SubmitWorkFormInput) (*SubmitWorkFormOutput, error) {
	if input.Date == "" {
		return nil, fmt.Errorf("date is required")
	}
	task := strings.TrimSpace(input.Task)
	if task == "" {
		return nil, fmt.Errorf("task name is required")
	}
	dur, err := uc.durationParser.Parse(strings.TrimSpace(input.Duration))
	if err != nil {
		return nil, err
	}
	durationMin := int(dur.Minutes())
	if durationMin <= 0 {
		return nil, fmt.Errorf("duration must be positive")
	}
	if uc.dayRepo == nil || uc.timeProvider == nil {
		return nil, fmt.Errorf("work entry dependencies are not configured")
	}

	record, err := uc.dayRepo.FindByDate(input.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to load day record: %w", err)
	}

	editIdx := input.EditEntry
	wasSplit := false
	selectedIdx := -1
	if input.IsBreak {
		if editIdx >= 0 && editIdx < len(record.Entries) {
			entry := &record.Entries[editIdx]
			entry.Task = task
			entry.DurationMin = durationMin
			entry.IsBreak = true
			entry.Project = ""
			selectedIdx = editIdx
		} else {
			taskLower := strings.ToLower(task)
			merged := false
			for i := range record.Entries {
				if record.Entries[i].IsBreak && strings.ToLower(record.Entries[i].Task) == taskLower {
					record.Entries[i].DurationMin += durationMin
					selectedIdx = i
					merged = true
					break
				}
			}
			if !merged {
				record.Entries = append(record.Entries, model.WorkEntry{
					ID:          uc.timeProvider.GenerateID(),
					Task:        task,
					DurationMin: durationMin,
					IsBreak:     true,
				})
				selectedIdx = len(record.Entries) - 1
			}
		}
	} else {
		distributed, distributeErr := uc.distributeWorkEntries(task, strings.TrimSpace(input.ProjectRaw), dur)
		if distributeErr != nil {
			return nil, distributeErr
		}
		wasSplit = len(distributed) > 1
		if editIdx >= 0 && editIdx < len(record.Entries) {
			updated := make([]model.WorkEntry, 0, len(record.Entries)-1+len(distributed))
			updated = append(updated, record.Entries[:editIdx]...)
			updated = append(updated, distributed...)
			updated = append(updated, record.Entries[editIdx+1:]...)
			record.Entries = updated
			selectedIdx = editIdx + len(distributed) - 1
		} else {
			record.Entries = append(record.Entries, distributed...)
			selectedIdx = len(record.Entries) - 1
		}
	}

	if err := uc.dayRepo.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save day record: %w", err)
	}

	label := "✓ Work entry logged"
	if editIdx >= 0 && !wasSplit {
		label = "✓ Entry updated"
	} else if input.IsBreak {
		label = "✓ Break logged"
	} else if wasSplit {
		label = "✓ Work entries split across projects"
	}

	return &SubmitWorkFormOutput{
		Record:           mapDomainDayRecordToDTO(record),
		Label:            label,
		SelectedEntryIdx: selectedIdx,
	}, nil
}

func (uc *SubmitWorkFormUseCase) distributeWorkEntries(task, projectRaw string, durMinutes time.Duration) ([]model.WorkEntry, error) {
	rawParts := strings.Split(projectRaw, ",")
	projects := make([]string, 0, len(rawParts))
	for _, p := range rawParts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			projects = append(projects, trimmed)
		}
	}
	if len(projects) == 0 {
		projects = []string{""}
	}
	totalMin := int(durMinutes.Minutes())
	base := totalMin / len(projects)
	remainder := totalMin % len(projects)
	newEntries := make([]model.WorkEntry, 0, len(projects))
	for i, proj := range projects {
		mins := base
		if i < remainder {
			mins++
		}
		if mins == 0 {
			continue
		}
		newEntries = append(newEntries, model.WorkEntry{
			ID:          uc.timeProvider.GenerateID(),
			Task:        task,
			Project:     proj,
			DurationMin: mins,
			IsBreak:     false,
		})
	}
	if len(newEntries) == 0 {
		return nil, fmt.Errorf("duration too short to distribute across projects")
	}
	return newEntries, nil
}
