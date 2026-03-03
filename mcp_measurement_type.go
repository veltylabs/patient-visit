//go:build !wasm

package clinicalencounter

import (
	"github.com/tinywasm/fmt"
)

type CreateMeasurementTypeArgs struct {
	Name        string  `json:"name"`
	DefaultUnit string  `json:"default_unit"`
	MinNormal   float64 `json:"min_normal,omitempty"`
	MaxNormal   float64 `json:"max_normal,omitempty"`
	LoincCode   string  `json:"loinc_code,omitempty"`
	UcumUnit    string  `json:"ucum_unit,omitempty"`
}

func (m *Module) CreateMeasurementType(args CreateMeasurementTypeArgs) (*MeasurementType, error) {
	if args.Name == "" || args.DefaultUnit == "" {
		return nil, fmt.Err("missing", "required", "arguments")
	}

	record := &MeasurementType{
		ID:          m.uid.GetNewID(),
		Name:        args.Name,
		DefaultUnit: args.DefaultUnit,
		MinNormal:   args.MinNormal,
		MaxNormal:   args.MaxNormal,
		IsActive:    true,
		LoincCode:   args.LoincCode,
		UcumUnit:    args.UcumUnit,
	}

	if err := m.db.Create(record); err != nil {
		return nil, err
	}

	return record, nil
}

type ListMeasurementTypesArgs struct {
	IncludeInactive bool `json:"include_inactive,omitempty"`
}

func (m *Module) ListMeasurementTypes(args ListMeasurementTypesArgs) ([]*MeasurementType, error) {
	qb := m.db.Query(&MeasurementType{})

	if !args.IncludeInactive {
		qb = qb.Where(MeasurementType_.IsActive).Eq(true)
	}

	return ReadAllMeasurementType(qb)
}

type ToggleMeasurementTypeArgs struct {
	ID       string `json:"id"`
	IsActive bool   `json:"is_active"`
}

func (m *Module) ToggleMeasurementType(args ToggleMeasurementTypeArgs) (*MeasurementType, error) {
	if args.ID == "" {
		return nil, fmt.Err("missing", "id")
	}

	record := &MeasurementType{}
	qb := m.db.Query(record).Where(MeasurementType_.ID).Eq(args.ID)
	_, err := ReadOneMeasurementType(qb, record)
	if err != nil {
		return nil, fmt.Err("measurement", "type", "not", "found")
	}

	record.IsActive = args.IsActive

	if err := m.db.Update(record); err != nil {
		return nil, err
	}

	return record, nil
}
