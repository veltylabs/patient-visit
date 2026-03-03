//go:build !wasm

package clinicalencounter_test

import (
	"testing"

	clinicalencounter "github.com/veltylabs/clinical-encounter"
)

func TestVisitStatusFSM_HappyPath(t *testing.T) {
	mod, pub := setupTestModule(t)

	// Create
	visit, _ := mod.CreateVisit(clinicalencounter.CreateVisitArgs{
		PatientID:           "pat1",
		DoctorID:            "doc1",
		Reason:              "checkup",
		PatientNameSnapshot: "John",
		PatientRutSnapshot:  "123",
		DoctorNameSnapshot:  "Dr. Smith",
		AttentionAt:         1600000000,
	})

	if visit.Status != clinicalencounter.StatusCreated {
		t.Fatalf("Expected status created, got %s", visit.Status)
	}

	// Arrived
	visit, err := mod.MarkArrived(clinicalencounter.MarkArrivedArgs{ID: visit.ID, PatientNameSnapshot: "John Updated"})
	if err != nil {
		t.Fatalf("MarkArrived failed: %v", err)
	}
	if visit.Status != clinicalencounter.StatusArrived {
		t.Errorf("Expected status arrived, got %s", visit.Status)
	}
	if visit.PatientNameSnapshot != "John Updated" {
		t.Errorf("Expected patient name to be updated")
	}
	if len(pub.events) != 1 || pub.events[0] != clinicalencounter.EventPatientArrived {
		t.Errorf("Expected EventPatientArrived")
	}

	// Triaged
	visit, err = mod.MarkTriaged(clinicalencounter.MarkTriagedArgs{ID: visit.ID})
	if err != nil {
		t.Fatalf("MarkTriaged failed: %v", err)
	}
	if visit.Status != clinicalencounter.StatusTriaged {
		t.Errorf("Expected status triaged, got %s", visit.Status)
	}
	if len(pub.events) != 2 || pub.events[1] != clinicalencounter.EventPatientTriaged {
		t.Errorf("Expected EventPatientTriaged")
	}

	// In Progress
	visit, err = mod.StartVisit(clinicalencounter.StartVisitArgs{ID: visit.ID})
	if err != nil {
		t.Fatalf("StartVisit failed: %v", err)
	}
	if visit.Status != clinicalencounter.StatusInProgress {
		t.Errorf("Expected status in_progress, got %s", visit.Status)
	}
	if visit.StartedAt == 0 {
		t.Error("StartedAt must be set after StartVisit")
	}

	// Completed
	visit, err = mod.CompleteVisit(clinicalencounter.CompleteVisitArgs{ID: visit.ID})
	if err != nil {
		t.Fatalf("CompleteVisit failed: %v", err)
	}
	if visit.Status != clinicalencounter.StatusCompleted {
		t.Errorf("Expected status completed, got %s", visit.Status)
	}
	if visit.FinishedAt == 0 {
		t.Error("FinishedAt must be set after CompleteVisit")
	}
	if visit.StartedAt > visit.FinishedAt {
		t.Error("StartedAt must not be after FinishedAt")
	}
	if len(pub.events) != 3 || pub.events[2] != clinicalencounter.EventVisitCompleted {
		t.Errorf("Expected EventVisitCompleted")
	}
}

func TestVisitStatusFSM_Cancel(t *testing.T) {
	mod, pub := setupTestModule(t)

	// From Created
	v1, _ := mod.CreateVisit(clinicalencounter.CreateVisitArgs{
		PatientID: "p1", DoctorID: "d1", Reason: "r1", PatientNameSnapshot: "n1", PatientRutSnapshot: "rut1", DoctorNameSnapshot: "dn1", AttentionAt: 1600000000,
	})
	v1, err := mod.CancelVisit(clinicalencounter.CancelVisitArgs{ID: v1.ID, Reason: "no show"})
	if err != nil {
		t.Fatalf("CancelVisit from created failed: %v", err)
	}
	if v1.Status != clinicalencounter.StatusCancelled {
		t.Errorf("Expected cancelled")
	}
	if v1.FinishedAt == 0 {
		t.Error("FinishedAt must be set after CancelVisit")
	}
	if len(pub.events) != 1 || pub.events[0] != clinicalencounter.EventVisitCancelled {
		t.Errorf("Expected EventVisitCancelled")
	}

	// Invalid transition
	_, err = mod.CancelVisit(clinicalencounter.CancelVisitArgs{ID: v1.ID})
	if err == nil {
		t.Errorf("Expected error cancelling an already cancelled visit")
	}

	// Complete from Arrived should fail
	v2, _ := mod.CreateVisit(clinicalencounter.CreateVisitArgs{
		PatientID: "p1", DoctorID: "d1", Reason: "r1", PatientNameSnapshot: "n1", PatientRutSnapshot: "rut1", DoctorNameSnapshot: "dn1", AttentionAt: 1600000000,
	})
	_, _ = mod.MarkArrived(clinicalencounter.MarkArrivedArgs{ID: v2.ID})
	_, err = mod.CompleteVisit(clinicalencounter.CompleteVisitArgs{ID: v2.ID})
	if err == nil {
		t.Errorf("Expected error completing an arrived visit")
	}
}

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
