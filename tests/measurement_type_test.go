//go:build !wasm

package clinicalencounter_test

import (
	"testing"

	clinicalencounter "github.com/veltylabs/clinical-encounter"
)

func TestMeasurementType(t *testing.T) {
	mod, _ := setupTestModule(t)

	// Create OK
	mtype, err := mod.CreateMeasurementType(clinicalencounter.CreateMeasurementTypeArgs{
		Name:        "Weight",
		DefaultUnit: "kg",
		MinNormal:   50.0,
		MaxNormal:   100.0,
		LoincCode:   "85354-9",
		UcumUnit:    "mm[Hg]",
	})
	if err != nil {
		t.Fatalf("CreateMeasurementType failed: %v", err)
	}
	if mtype.Name != "Weight" || mtype.DefaultUnit != "kg" || !mtype.IsActive {
		t.Errorf("Unexpected measurement type properties")
	}
	if mtype.LoincCode != "85354-9" {
		t.Errorf("want LoincCode '85354-9', got %q", mtype.LoincCode)
	}
	if mtype.UcumUnit != "mm[Hg]" {
		t.Errorf("want UcumUnit 'mm[Hg]', got %q", mtype.UcumUnit)
	}

	// Create Missing args
	_, err = mod.CreateMeasurementType(clinicalencounter.CreateMeasurementTypeArgs{Name: "Missing Unit"})
	if err == nil {
		t.Errorf("Expected error for missing default unit")
	}

	// List Types - active only filter
	types, err := mod.ListMeasurementTypes(clinicalencounter.ListMeasurementTypesArgs{})
	if err != nil {
		t.Fatalf("ListMeasurementTypes failed: %v", err)
	}
	if len(types) != 1 {
		t.Errorf("Expected 1 active type, got %d", len(types))
	}

	// Toggle Deactivate
	updated, err := mod.ToggleMeasurementType(clinicalencounter.ToggleMeasurementTypeArgs{
		ID:       mtype.ID,
		IsActive: false,
	})
	if err != nil {
		t.Fatalf("ToggleMeasurementType failed: %v", err)
	}
	if updated.IsActive {
		t.Errorf("Expected type to be inactive")
	}

	// List Types - active only (should be 0)
	types, _ = mod.ListMeasurementTypes(clinicalencounter.ListMeasurementTypesArgs{})
	if len(types) != 0 {
		t.Errorf("Expected 0 active types, got %d", len(types))
	}

	// List Types - include inactive
	types, _ = mod.ListMeasurementTypes(clinicalencounter.ListMeasurementTypesArgs{IncludeInactive: true})
	if len(types) != 1 {
		t.Errorf("Expected 1 type including inactive, got %d", len(types))
	}

	// Toggle Activate
	mod.ToggleMeasurementType(clinicalencounter.ToggleMeasurementTypeArgs{
		ID:       mtype.ID,
		IsActive: true,
	})
	types, _ = mod.ListMeasurementTypes(clinicalencounter.ListMeasurementTypesArgs{})
	if len(types) != 1 {
		t.Errorf("Expected 1 active type after reactivation, got %d", len(types))
	}
}
