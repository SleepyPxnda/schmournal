package usecase

import (
	"fmt"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
	"github.com/sleepypxnda/schmournal/internal/domain/repository"
)

type LoadDayRecordInput struct {
	Date string
}

type LoadDayRecordUseCase struct {
	dayRepo repository.DayRecordRepository
}

func NewLoadDayRecordUseCase(dayRepo repository.DayRecordRepository) *LoadDayRecordUseCase {
	return &LoadDayRecordUseCase{dayRepo: dayRepo}
}

func (uc *LoadDayRecordUseCase) Execute(input LoadDayRecordInput) (model.DayRecord, error) {
	if input.Date == "" {
		return model.DayRecord{}, fmt.Errorf("date is required")
	}
	record, err := uc.dayRepo.FindByDate(input.Date)
	if err != nil {
		return model.DayRecord{}, fmt.Errorf("failed to load day record: %w", err)
	}
	return record, nil
}

func (uc *LoadDayRecordUseCase) ExecuteDTO(input LoadDayRecordInput) (DayRecordDTO, error) {
	record, err := uc.Execute(input)
	if err != nil {
		return DayRecordDTO{}, err
	}
	return mapDomainDayRecordToDTO(record), nil
}

type LoadAllDayRecordsUseCase struct {
	dayRepo repository.DayRecordRepository
}

func NewLoadAllDayRecordsUseCase(dayRepo repository.DayRecordRepository) *LoadAllDayRecordsUseCase {
	return &LoadAllDayRecordsUseCase{dayRepo: dayRepo}
}

func (uc *LoadAllDayRecordsUseCase) Execute() ([]model.DayRecord, error) {
	records, err := uc.dayRepo.FindAll()
	if err != nil {
		return nil, fmt.Errorf("failed to load day records: %w", err)
	}
	return records, nil
}

func (uc *LoadAllDayRecordsUseCase) ExecuteDTO() ([]DayRecordDTO, error) {
	records, err := uc.Execute()
	if err != nil {
		return nil, err
	}
	out := make([]DayRecordDTO, len(records))
	for i, rec := range records {
		out[i] = mapDomainDayRecordToDTO(rec)
	}
	return out, nil
}

type SaveDayRecordInput struct {
	Record model.DayRecord
}

type SaveDayRecordUseCase struct {
	dayRepo repository.DayRecordRepository
}

func NewSaveDayRecordUseCase(dayRepo repository.DayRecordRepository) *SaveDayRecordUseCase {
	return &SaveDayRecordUseCase{dayRepo: dayRepo}
}

func (uc *SaveDayRecordUseCase) Execute(input SaveDayRecordInput) error {
	if input.Record.Date == "" {
		return fmt.Errorf("record date is required")
	}
	if err := uc.dayRepo.Save(input.Record); err != nil {
		return fmt.Errorf("failed to save day record: %w", err)
	}
	return nil
}

type SaveDayRecordDTOInput struct {
	Record DayRecordDTO
}

func (uc *SaveDayRecordUseCase) ExecuteDTO(input SaveDayRecordDTOInput) error {
	return uc.Execute(SaveDayRecordInput{Record: mapDayRecordDTOToDomain(input.Record)})
}

type DeleteDayRecordInput struct {
	Date string
}

type DeleteDayRecordUseCase struct {
	dayRepo repository.DayRecordRepository
}

func NewDeleteDayRecordUseCase(dayRepo repository.DayRecordRepository) *DeleteDayRecordUseCase {
	return &DeleteDayRecordUseCase{dayRepo: dayRepo}
}

func (uc *DeleteDayRecordUseCase) Execute(input DeleteDayRecordInput) error {
	if input.Date == "" {
		return fmt.Errorf("date is required")
	}
	if err := uc.dayRepo.Delete(input.Date); err != nil {
		return fmt.Errorf("failed to delete day record: %w", err)
	}
	return nil
}
