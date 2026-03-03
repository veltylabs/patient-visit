← [Stage 3](PLAN_STAGE_3_ADAPTER.md) | [Master Plan](PLAN.md)

# Stage 4 — Tests & Documentation

**Goal:** Add FHIR adapter tests, update existing tests for new fields, and update all affected documentation.

---

## 4.1 New file: `tests/fhir_adapter_test.go`

```go
//go:build !wasm

package clinicalencounter_test

import (
	"testing"

	clinicalencounter "github.com/veltylabs/clinical-encounter"
)

func TestFHIRAdapter_StatusMapping(t *testing.T) {
	cases := []struct {
		internal string
		fhir     string
	}{
		{clinicalencounter.StatusCreated, "planned"},
		{clinicalencounter.StatusArrived, "arrived"},
		{clinicalencounter.StatusTriaged, "triaged"},
		{clinicalencounter.StatusInProgress, "in-progress"},
		{clinicalencounter.StatusCompleted, "finished"},
		{clinicalencounter.StatusCancelled, "cancelled"},
	}
	for _, tc := range cases {
		h := &clinicalencounter.MedicalHistory{Status: tc.internal}
		enc := clinicalencounter.ToFHIREncounter(h)
		if enc.Status != tc.fhir {
			t.Errorf("status %q: want FHIR %q, got %q", tc.internal, tc.fhir, enc.Status)
		}
	}
}

func TestFHIRAdapter_Encounter_ResourceType(t *testing.T) {
	h := &clinicalencounter.MedicalHistory{ID: "enc1", PatientID: "p1", DoctorID: "d1"}
	enc := clinicalencounter.ToFHIREncounter(h)
	if enc.ResourceType != "Encounter" {
		t.Errorf("want ResourceType 'Encounter', got %q", enc.ResourceType)
	}
	if enc.Subject.Reference != "Patient/p1" {
		t.Errorf("want subject 'Patient/p1', got %q", enc.Subject.Reference)
	}
	if enc.Participant[0].Individual.Reference != "Practitioner/d1" {
		t.Errorf("want participant 'Practitioner/d1', got %q", enc.Participant[0].Individual.Reference)
	}
}

func TestFHIRAdapter_Encounter_Period(t *testing.T) {
	mod, _ := setupTestModule(t)

	visit, _ := mod.CreateVisit(clinicalencounter.CreateVisitArgs{
		PatientID: "p1", DoctorID: "d1", Reason: "checkup",
		PatientNameSnapshot: "John", PatientRutSnapshot: "123",
		DoctorNameSnapshot: "Dr. Smith", AttentionAt: 1600000000,
	})

	// Before StartVisit: Period must be nil
	enc := clinicalencounter.ToFHIREncounter(visit)
	if enc.Period != nil {
		t.Error("expected nil Period before StartVisit")
	}

	// Walk FSM to in_progress
	visit, _ = mod.MarkArrived(clinicalencounter.MarkArrivedArgs{ID: visit.ID})
	visit, _ = mod.MarkTriaged(clinicalencounter.MarkTriagedArgs{ID: visit.ID})
	visit, _ = mod.StartVisit(clinicalencounter.StartVisitArgs{ID: visit.ID})

	if visit.StartedAt == 0 {
		t.Fatal("StartedAt must be set after StartVisit")
	}

	enc = clinicalencounter.ToFHIREncounter(visit)
	if enc.Period == nil {
		t.Fatal("expected Period after StartVisit")
	}
	if enc.Period.Start == "" {
		t.Error("Period.Start must not be empty after StartVisit")
	}
	if enc.Period.End != "" {
		t.Error("Period.End must be empty before CompleteVisit")
	}

	// Complete visit
	visit, _ = mod.CompleteVisit(clinicalencounter.CompleteVisitArgs{
		ID: visit.ID, Diagnostic: "Checkup OK", Cie10Code: "Z00",
	})

	if visit.FinishedAt == 0 {
		t.Fatal("FinishedAt must be set after CompleteVisit")
	}

	enc = clinicalencounter.ToFHIREncounter(visit)
	if enc.Period.End == "" {
		t.Error("Period.End must not be empty after CompleteVisit")
	}
}

func TestFHIRAdapter_Encounter_Diagnosis(t *testing.T) {
	mod, _ := setupTestModule(t)

	visit, _ := mod.CreateVisit(clinicalencounter.CreateVisitArgs{
		PatientID: "p1", DoctorID: "d1", Reason: "flu",
		PatientNameSnapshot: "Ana", PatientRutSnapshot: "456",
		DoctorNameSnapshot: "Dr. House", AttentionAt: 1600000000,
	})

	// Walk FSM to completed with CIE-10 code
	visit, _ = mod.MarkArrived(clinicalencounter.MarkArrivedArgs{ID: visit.ID})
	visit, _ = mod.MarkTriaged(clinicalencounter.MarkTriagedArgs{ID: visit.ID})
	visit, _ = mod.StartVisit(clinicalencounter.StartVisitArgs{ID: visit.ID})
	visit, _ = mod.CompleteVisit(clinicalencounter.CompleteVisitArgs{
		ID: visit.ID, Diagnostic: "Common cold", Cie10Code: "J00",
	})

	enc := clinicalencounter.ToFHIREncounter(visit)

	if len(enc.Diagnosis) == 0 {
		t.Fatal("expected at least one Diagnosis entry")
	}
	if enc.Diagnosis[0].Use.Coding[0].Code != "J00" {
		t.Errorf("want CIE-10 code 'J00', got %q", enc.Diagnosis[0].Use.Coding[0].Code)
	}
	if enc.Diagnosis[0].Use.Coding[0].System != "http://hl7.org/fhir/sid/icd-10" {
		t.Errorf("unexpected CIE-10 system URI")
	}
}

func TestFHIRAdapter_Observation_LOINC_UCUM(t *testing.T) {
	mod, _ := setupTestModule(t)

	// Create measurement type with LOINC + UCUM
	mt, _ := mod.CreateMeasurementType(clinicalencounter.CreateMeasurementTypeArgs{
		Name:        "Blood Pressure",
		DefaultUnit: "mmHg",
		LoincCode:   "85354-9",
		UcumUnit:    "mm[Hg]",
	})

	// Setup a visit in arrived state to add measurement
	visit, _ := mod.CreateVisit(clinicalencounter.CreateVisitArgs{
		PatientID: "p1", DoctorID: "d1", Reason: "bp check",
		PatientNameSnapshot: "Bob", PatientRutSnapshot: "789",
		DoctorNameSnapshot: "Dr. Who", AttentionAt: 1600000000,
	})
	visit, _ = mod.MarkArrived(clinicalencounter.MarkArrivedArgs{ID: visit.ID})

	measurement, _ := mod.AddMeasurement(clinicalencounter.AddMeasurementArgs{
		MedicalHistoryID:  visit.ID,
		MeasuredByStaffID: "nurse1",
		MeasurementTypeID: mt.ID,
		Value:             120,
		Unit:              "mmHg",
		MeasuredAt:        1600000100,
	})

	obs := clinicalencounter.ToFHIRObservation(measurement, mt, "p1")

	if obs.ResourceType != "Observation" {
		t.Errorf("want ResourceType 'Observation', got %q", obs.ResourceType)
	}
	if obs.Status != "final" {
		t.Errorf("want status 'final', got %q", obs.Status)
	}
	if len(obs.Code.Coding) == 0 {
		t.Fatal("expected LOINC coding")
	}
	if obs.Code.Coding[0].System != "http://loinc.org" {
		t.Errorf("want LOINC system, got %q", obs.Code.Coding[0].System)
	}
	if obs.Code.Coding[0].Code != "85354-9" {
		t.Errorf("want LOINC code '85354-9', got %q", obs.Code.Coding[0].Code)
	}
	if obs.ValueQuantity.System != "http://unitsofmeasure.org" {
		t.Errorf("want UCUM system, got %q", obs.ValueQuantity.System)
	}
	if obs.ValueQuantity.Code != "mm[Hg]" {
		t.Errorf("want UCUM code 'mm[Hg]', got %q", obs.ValueQuantity.Code)
	}
}

func TestFHIRAdapter_Observation_NoLOINC(t *testing.T) {
	mt := &clinicalencounter.MeasurementType{
		ID:   "mt1",
		Name: "Temperature",
	}
	m := &clinicalencounter.ClinicalMeasurement{
		ID:               "obs1",
		MedicalHistoryID: "enc1",
		MeasuredByStaffID: "nurse1",
		Value:            37.2,
		Unit:             "°C",
		MeasuredAt:       1600000000,
	}
	obs := clinicalencounter.ToFHIRObservation(m, mt, "p1")

	if len(obs.Code.Coding) != 0 {
		t.Error("expected no LOINC coding when LoincCode is empty")
	}
	if obs.Code.Text != "Temperature" {
		t.Errorf("want Code.Text 'Temperature', got %q", obs.Code.Text)
	}
	if obs.ValueQuantity.System != "" {
		t.Error("expected no UCUM system when UcumUnit is empty")
	}
}

func TestFHIRAdapter_CancelVisit_Period(t *testing.T) {
	mod, _ := setupTestModule(t)

	visit, _ := mod.CreateVisit(clinicalencounter.CreateVisitArgs{
		PatientID: "p1", DoctorID: "d1", Reason: "r",
		PatientNameSnapshot: "X", PatientRutSnapshot: "0",
		DoctorNameSnapshot: "Dr. Y", AttentionAt: 1600000000,
	})

	visit, _ = mod.MarkArrived(clinicalencounter.MarkArrivedArgs{ID: visit.ID})
	visit, _ = mod.MarkTriaged(clinicalencounter.MarkTriagedArgs{ID: visit.ID})
	visit, _ = mod.StartVisit(clinicalencounter.StartVisitArgs{ID: visit.ID})
	visit, _ = mod.CancelVisit(clinicalencounter.CancelVisitArgs{ID: visit.ID, Reason: "no show"})

	if visit.FinishedAt == 0 {
		t.Error("FinishedAt must be set after CancelVisit")
	}

	enc := clinicalencounter.ToFHIREncounter(visit)
	if enc.Status != "cancelled" {
		t.Errorf("want status 'cancelled', got %q", enc.Status)
	}
	if enc.Period == nil || enc.Period.End == "" {
		t.Error("Period.End must be set after CancelVisit")
	}
}
```

---

## 4.2 Updates to existing test files

### `tests/visit_status_test.go`

In **`TestVisitStatusFSM_HappyPath`**:

After `StartVisit()` call, add:
```go
if visit.StartedAt == 0 {
    t.Error("StartedAt must be set after StartVisit")
}
```

After `CompleteVisit()` call, add:
```go
if visit.FinishedAt == 0 {
    t.Error("FinishedAt must be set after CompleteVisit")
}
if visit.StartedAt >= visit.FinishedAt {
    t.Error("StartedAt must be before FinishedAt")
}
```

In **`TestVisitStatusFSM_Cancel`**, after `CancelVisit()` success:
```go
if v1.FinishedAt == 0 {
    t.Error("FinishedAt must be set after CancelVisit")
}
```

Add new test **`TestCompleteVisit_WithDiagnostic`**:
```go
func TestCompleteVisit_WithDiagnostic(t *testing.T) {
    mod, _ := setupTestModule(t)

    visit, _ := mod.CreateVisit(clinicalencounter.CreateVisitArgs{
        PatientID: "p1", DoctorID: "d1", Reason: "checkup",
        PatientNameSnapshot: "Ana", PatientRutSnapshot: "123",
        DoctorNameSnapshot: "Dr. House", AttentionAt: 1600000000,
    })
    visit, _ = mod.MarkArrived(clinicalencounter.MarkArrivedArgs{ID: visit.ID})
    visit, _ = mod.MarkTriaged(clinicalencounter.MarkTriagedArgs{ID: visit.ID})
    visit, _ = mod.StartVisit(clinicalencounter.StartVisitArgs{ID: visit.ID})
    visit, err := mod.CompleteVisit(clinicalencounter.CompleteVisitArgs{
        ID:           visit.ID,
        Diagnostic:   "Mild flu",
        Prescription: "Rest and hydration",
        Cie10Code:    "J11",
    })
    if err != nil {
        t.Fatalf("CompleteVisit failed: %v", err)
    }
    if visit.Diagnostic != "Mild flu" {
        t.Errorf("want Diagnostic 'Mild flu', got %q", visit.Diagnostic)
    }
    if visit.Prescription != "Rest and hydration" {
        t.Errorf("want Prescription 'Rest and hydration', got %q", visit.Prescription)
    }
    if visit.Cie10Code != "J11" {
        t.Errorf("want Cie10Code 'J11', got %q", visit.Cie10Code)
    }

    // Verify persistence via re-fetch
    fetched, err := mod.GetVisit(clinicalencounter.GetVisitArgs{ID: visit.ID})
    if err != nil {
        t.Fatalf("GetVisit failed: %v", err)
    }
    if fetched.Cie10Code != "J11" {
        t.Errorf("Cie10Code not persisted: got %q", fetched.Cie10Code)
    }
}
```

### `tests/measurement_type_test.go`

In **`TestMeasurementType`**, update the `CreateMeasurementType` call to include new fields:
```go
mt, err := mod.CreateMeasurementType(clinicalencounter.CreateMeasurementTypeArgs{
    Name:        "Blood Pressure",
    DefaultUnit: "mmHg",
    MinNormal:   80,
    MaxNormal:   120,
    LoincCode:   "85354-9",
    UcumUnit:    "mm[Hg]",
})
// ...existing assertions...
if mt.LoincCode != "85354-9" {
    t.Errorf("want LoincCode '85354-9', got %q", mt.LoincCode)
}
if mt.UcumUnit != "mm[Hg]" {
    t.Errorf("want UcumUnit 'mm[Hg]', got %q", mt.UcumUnit)
}
```

---

## 4.3 Documentation updates

### `docs/FHIR_ROADMAP.md`

Update the compliance status table / sections:
- **§3.1 (Nomenclaturas):** Mark as `partially implemented` — internal adapter provides FHIR JSON; REST API remains out of scope.
- **§3.2 (Terminologías):** Mark CIE-10, LOINC, UCUM as `implemented`. SNOMED CT remains out of scope.
- **§3.3 (Período):** Mark as `implemented` via `StartedAt`/`FinishedAt` fields.

### `docs/ARCHITECTURE.md`

In the entities section, add new fields to `MedicalHistory`:
- `Cie10Code string` — optional CIE-10 code for the diagnosis
- `StartedAt int64` — Unix ts when consultation started (FHIR Period.start)
- `FinishedAt int64` — Unix ts when visit ended (FHIR Period.end)

Add new fields to `MeasurementType`:
- `LoincCode string` — optional LOINC code for the measurement type
- `UcumUnit string` — optional UCUM unit code

Add new section: **FHIR Adapter** — describe `fhir_types.go` and `fhir_adapter.go` as a translation layer.

### `docs/SKILL.md`

Add FHIR adapter usage snippet:
```go
// Translate to FHIR R4 Encounter JSON
enc := clinicalencounter.ToFHIREncounter(visit)

// Translate to FHIR R4 Observation JSON
obs := clinicalencounter.ToFHIRObservation(measurement, measurementType, patientID)
```

### `docs/diagrams/database.md`

In the `medical_history` entity, add:
```
cie10_code TEXT
started_at INT
finished_at INT
```

In the `measurement_type` entity, add:
```
loinc_code TEXT
ucum_unit TEXT
```

---

## Final Verification

```bash
# All tests must pass — existing + new FHIR adapter tests
gotest

# Publish
gopush 'feat: FHIR compliance — CIE-10, LOINC, UCUM, Period tracking, adapter layer'
```

← [Stage 3](PLAN_STAGE_3_ADAPTER.md) | [Master Plan](PLAN.md)
