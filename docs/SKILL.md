# SKILL: clinical-encounter module

## Overview
The `clinical-encounter` module handles medical consultation lifecycles, clinical measurements, and triage.

## Core Constraints & Rules
- **Snapshot Pattern:** Never perform cross-module database/API lookups to fetch external entities (like Patients or Doctors). Clients and external callers MUST pass snapshot counterpart fields (e.g., `patient_name_snapshot`, `doctor_specialty_snapshot`) when creating or updating visit records.
- **Strict FSM Workflow:** `MedicalHistory.Status` changes MUST NOT be done arbitrarily. They MUST respect the state machine: `created -> arrived -> triaged -> in_progress -> completed | cancelled`.
- **Event Publishing:** FSM transitions trigger domain events (like `EventPatientArrived`) via an injected `EventPublisher` interface. Passing `nil` for the publisher disables event broadcasting safely.
- **Time Representation:** Use Unix timestamps (`int64`) for all temporal fields (e.g., `AttentionAt`, `UpdatedAt`, `MeasuredAt`) rather than Go standard `time.Time` structs.
- **Frontend Compatibility:** Strictly rely on `tinywasm` polyfills within the logic code (`tinywasm/fmt`, `tinywasm/time`, `tinywasm/unixid`) to ensure WebAssembly limits are adhered to.

## Primary Entities
- `MedicalHistory`: Represents a patient's visit. Holds state, snapshot data, and acts as the parent for measurements and details.
- `MeasurementType`: Master dictionary for measurement definitions (temperature, height, etc).
- `ClinicalMeasurement`: A recorded vital sign or clinical measurement tied to a specific `MedicalHistory`.
- `HistoryDetail`: Used to register actions, diagnosis details, or specific catalog items tied to the visit.

## FHIR Mapping Examples
```go
// Translate to FHIR R4 Encounter JSON
enc := clinicalencounter.ToFHIREncounter(visit)

// Translate to FHIR R4 Observation JSON
obs := clinicalencounter.ToFHIRObservation(measurement, measurementType, patientID)
```
