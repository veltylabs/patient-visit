← [Stage 2](PLAN_STAGE_2_LOGIC.md) | Next → [Stage 4](PLAN_STAGE_4_TESTS.md)

# Stage 3 — FHIR Adapter Layer

**Goal:** Create two new server-side files that translate internal models to FHIR R4 resources. Zero changes to existing files.

Both new files must carry `//go:build !wasm` and belong to `package clinicalencounter`.

---

## 3.1 `fhir_types.go` — FHIR R4 resource structs

Define only the subset of FHIR R4 needed to represent `Encounter` and `Observation`. No methods — pure data types.

```go
//go:build !wasm

package clinicalencounter

// FHIREncounter represents an HL7 FHIR R4 Encounter resource.
// Spec: https://hl7.org/fhir/R4/encounter.html
type FHIREncounter struct {
	ResourceType string                   `json:"resourceType"` // always "Encounter"
	ID           string                   `json:"id"`
	Status       string                   `json:"status"`
	Subject      FHIRReference            `json:"subject"`
	Participant  []FHIRParticipant        `json:"participant,omitempty"`
	Period       *FHIRPeriod              `json:"period,omitempty"`
	ReasonCode   []FHIRCodeableConcept    `json:"reasonCode,omitempty"`
	Diagnosis    []FHIREncounterDiagnosis `json:"diagnosis,omitempty"`
}

// FHIRObservation represents an HL7 FHIR R4 Observation resource.
// Spec: https://hl7.org/fhir/R4/observation.html
type FHIRObservation struct {
	ResourceType      string                `json:"resourceType"` // always "Observation"
	ID                string                `json:"id"`
	Status            string                `json:"status"` // always "final"
	Category          []FHIRCodeableConcept `json:"category"`
	Code              FHIRCodeableConcept   `json:"code"`
	Subject           FHIRReference         `json:"subject"`
	Encounter         FHIRReference         `json:"encounter"`
	EffectiveDateTime string                `json:"effectiveDateTime,omitempty"`
	Performer         []FHIRReference       `json:"performer,omitempty"`
	ValueQuantity     *FHIRQuantity         `json:"valueQuantity,omitempty"`
}

type FHIRReference struct {
	Reference string `json:"reference"`
}

type FHIRPeriod struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

type FHIRCoding struct {
	System  string `json:"system,omitempty"`
	Code    string `json:"code,omitempty"`
	Display string `json:"display,omitempty"`
}

type FHIRCodeableConcept struct {
	Coding []FHIRCoding `json:"coding,omitempty"`
	Text   string       `json:"text,omitempty"`
}

type FHIRQuantity struct {
	Value  float64 `json:"value"`
	Unit   string  `json:"unit"`
	System string  `json:"system,omitempty"`
	Code   string  `json:"code,omitempty"`
}

type FHIRParticipant struct {
	Individual FHIRReference `json:"individual"`
}

type FHIREncounterDiagnosis struct {
	Condition FHIRReference       `json:"condition"`
	Use       FHIRCodeableConcept `json:"use,omitempty"`
}
```

---

## 3.2 `fhir_adapter.go` — Translation functions

```go
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
	return time.FormatISO8601(nano)
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
```

---

## Implementation Notes

### `fhirTime` and `tinywasm/time`
The `fhirTime` helper successfully avoids `time.Time` from stdlib by using `github.com/tinywasm/time`, guaranteeing tinygo/WASM portability without dealing with raw math logic. Verify output with: Unix `0` → `"1970-01-01T00:00:00Z"`.

### `ToFHIRObservation` — `patientID` parameter
`ClinicalMeasurement` does not store `patient_id` directly; it links through `medical_history_id`. The caller must pass the patient ID (obtained from the parent `MedicalHistory`).

---

## Verification for Stage 3

```bash
gotest
```

No new tests yet — Stage 4 adds them. The build must succeed with zero compilation errors.

← [Stage 2](PLAN_STAGE_2_LOGIC.md) | Next → [Stage 4](PLAN_STAGE_4_TESTS.md)
