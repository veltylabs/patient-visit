# patient-visit Architecture

## 1. Domain Scope
The `patient-visit` module manages the full lifecycle of a patient's visit to a medical clinic or hospital. It tracks the scheduling, arrival, triage, and consultation states through a Finite State Machine (FSM), as well as clinical measurements (vitals) and history details.

## 2. Core Entities
- **MedicalHistory:** The central entity representing a single visit. It contains state, timestamps (`AttentionAt`, `UpdatedAt`), and snapshots of the patient/doctor data at the time of the visit.
- **MeasurementType:** Reference dictionary for types of vital measurements (e.g., Blood Pressure, Weight, Temperature).
- **ClinicalMeasurement:** The actual measurement values taken during triage or consultation. Linked to `MedicalHistory` and `MeasurementType`.
- **HistoryDetail:** Details of specific actions or catalog items applied/prescribed during the visit.

## 3. Finite State Machine (FSM)
The lifecycle of a visit strictly aligns with FHIR standards through the following statuses:
```
created ──mark_arrived──► arrived ──mark_triaged──► triaged ──start_visit──► in_progress ──complete──► completed
   │                         │                        │
   └──cancel──► cancelled ◄──┴────────────────────────┘
```
Transitions emit business events (`visit.patient_arrived`, `visit.patient_triaged`, `visit.completed`, `visit.cancelled`) through an injected `EventPublisher` interface, enabling decoupled cross-module communication (e.g., to notify receptionists, doctors, or trigger billing workflows).

## 4. Architectural Patterns
1. **Dependency Injection:** The module core (`Module` struct) relies on injected dependencies (`*orm.DB`, `EventPublisher`) initialized via `New(db, pub)`. There is no global state.
2. **Snapshotting:** The `patient-visit` module does not make cross-module HTTP/DB calls to fetch patient or doctor names. Instead, these are taken as input parameters during creation and stored as immutable "snapshots" (e.g., `PatientNameSnapshot`).
3. **Agnostic Storage & Execution:** Uses `tinywasm/orm` to abstract database interactions, making it trivially mockable with `tinywasm/sqlite` in tests and able to run on PostgreSQL in production. Isomorphic codebase logic (runs in WebAssembly frontend and Go server backend).
