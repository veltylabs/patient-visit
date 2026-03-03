//go:build !wasm

package clinicalencounter

import "github.com/tinywasm/time"

// fhirStatus maps internal FSM status codes to FHIR R4 Encounter.status values.
func fhirStatus(s string) string {
	switch s {
	case StatusCreated:
		return "planned"
	case StatusArrived:
		return "arrived"
	case StatusTriaged:
		return "triaged"
	case StatusInProgress:
		return "in-progress"
	case StatusCompleted:
		return "finished"
	case StatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// fhirTime converts a Unix int64 timestamp (seconds) to an ISO 8601 string.
// Returns "" if ts == 0 (field not yet set).
func fhirTime(ts int64) string {
	if ts == 0 {
		return ""
	}

	// Convert seconds to nanoseconds for tinywasm/time
	nano := ts * 1000000000

	// Returns YYYY-MM-DDTHH:MM:SSZ in strict UTC via tinywasm/time
	// We use FormatDate and FormatTime and construct the ISO8601 string
	dateStr := time.FormatDate(nano)
	timeStr := time.FormatTime(nano)
	return dateStr + "T" + timeStr + "Z"
}

// ToFHIREncounter converts a MedicalHistory to a FHIR R4 Encounter resource.
func ToFHIREncounter(h *MedicalHistory) FHIREncounter {
	enc := FHIREncounter{
		ResourceType: "Encounter",
		ID:           h.ID,
		Status:       fhirStatus(h.Status),
		Subject:      FHIRReference{Reference: "Patient/" + h.PatientID},
		Participant: []FHIRParticipant{
			{Individual: FHIRReference{Reference: "Practitioner/" + h.DoctorID}},
		},
	}

	// Period — only populate when StartedAt has been set (StartVisit called)
	if h.StartedAt > 0 {
		enc.Period = &FHIRPeriod{
			Start: fhirTime(h.StartedAt),
			End:   fhirTime(h.FinishedAt),
		}
	}

	// Reason — free text from Reason field
	if h.Reason != "" {
		enc.ReasonCode = []FHIRCodeableConcept{
			{Text: h.Reason},
		}
	}

	// Diagnosis — CIE-10 coded when available, always include free text
	if h.Diagnostic != "" {
		diag := FHIREncounterDiagnosis{
			Condition: FHIRReference{Reference: "Condition/" + h.ID},
		}
		if h.Cie10Code != "" {
			diag.Use = FHIRCodeableConcept{
				Coding: []FHIRCoding{{
					System:  "http://hl7.org/fhir/sid/icd-10",
					Code:    h.Cie10Code,
					Display: h.Diagnostic,
				}},
				Text: h.Diagnostic,
			}
		} else {
			diag.Use = FHIRCodeableConcept{
				Text: h.Diagnostic,
			}
		}
		enc.Diagnosis = []FHIREncounterDiagnosis{diag}
	}

	return enc
}

// ToFHIRObservation converts a ClinicalMeasurement + its MeasurementType to a
// FHIR R4 Observation resource. patientID must be provided by the caller since
// ClinicalMeasurement does not store it directly.
func ToFHIRObservation(m *ClinicalMeasurement, mt *MeasurementType, patientID string) FHIRObservation {
	obs := FHIRObservation{
		ResourceType:      "Observation",
		ID:                m.ID,
		Status:            "final",
		Subject:           FHIRReference{Reference: "Patient/" + patientID},
		Encounter:         FHIRReference{Reference: "Encounter/" + m.MedicalHistoryID},
		EffectiveDateTime: fhirTime(m.MeasuredAt),
		Performer:         []FHIRReference{{Reference: "Practitioner/" + m.MeasuredByStaffID}},
		Category: []FHIRCodeableConcept{{
			Coding: []FHIRCoding{{
				System: "http://terminology.hl7.org/CodeSystem/observation-category",
				Code:   "vital-signs",
			}},
		}},
	}

	// Code — LOINC coded when available, always include display name
	obs.Code = FHIRCodeableConcept{Text: mt.Name}
	if mt.LoincCode != "" {
		obs.Code.Coding = []FHIRCoding{{
			System:  "http://loinc.org",
			Code:    mt.LoincCode,
			Display: mt.Name,
		}}
	}

	// ValueQuantity — UCUM coded when available
	qty := &FHIRQuantity{
		Value: m.Value,
		Unit:  m.Unit,
	}
	if mt.UcumUnit != "" {
		qty.System = "http://unitsofmeasure.org"
		qty.Code   = mt.UcumUnit
	}
	obs.ValueQuantity = qty

	return obs
}