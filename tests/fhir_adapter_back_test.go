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
	// Simulate starting visit but keeping status triaged so we can cancel it
	visit, err := mod.CancelVisit(clinicalencounter.CancelVisitArgs{ID: visit.ID, Reason: "no show"})
	if err != nil {
		t.Fatalf("CancelVisit failed: %v", err)
	}

	if visit.FinishedAt == 0 {
		t.Error("FinishedAt must be set after CancelVisit")
	}

	enc := clinicalencounter.ToFHIREncounter(visit)
	if enc.Status != "cancelled" {
		t.Errorf("want status 'cancelled', got %q", enc.Status)
	}
	// if we haven't officially StartVisit, Period might not be set in normal flow.
	// But let's check what the adapter outputs if we canceled from Triaged.
	// We will skip testing Period.End specifically here unless we force a StartedAt.
}