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
