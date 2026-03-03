//go:build !wasm

package clinicalencounter

// Visit status — FHIR-aligned lifecycle
const (
    StatusCreated    = "created"     // scheduled, not yet at clinic
    StatusArrived    = "arrived"     // patient registered at reception
    StatusTriaged    = "triaged"     // nurse took vitals, waiting for doctor
    StatusInProgress = "in_progress" // doctor started the consultation
    StatusCompleted  = "completed"
    StatusCancelled  = "cancelled"
)

// Events published to EventPublisher
const (
    EventPatientArrived = "visit.patient_arrived" // → staff/reception notification
    EventPatientTriaged = "visit.patient_triaged" // → doctor notification (patient ready)
    EventVisitCompleted = "visit.completed"       // → billing, scheduling update
    EventVisitCancelled = "visit.cancelled"
)

// FSM: visitTransitions[currentStatus][action] → nextStatus
var visitTransitions = map[string]map[string]string{
    StatusCreated:    {"mark_arrived": StatusArrived, "cancel": StatusCancelled},
    StatusArrived:    {"mark_triaged": StatusTriaged, "cancel": StatusCancelled},
    StatusTriaged:    {"start_visit": StatusInProgress, "cancel": StatusCancelled},
    StatusInProgress: {"complete": StatusCompleted},
}

type MedicalHistory struct {
    ID                      string
    PatientID               string `db:"not_null"`
    DoctorID                string `db:"not_null"`
    ReservationID           string // soft ref — optional
    Status                  string `db:"not_null"` // see Status* constants
    AttentionAt             int64  `db:"not_null"` // Unix timestamp (date + time unified)
    Reason                  string `db:"not_null"`
    Diagnostic              string
    Prescription            string
    Cie10Code               string
    StartedAt               int64
    FinishedAt              int64
    PatientNameSnapshot     string `db:"not_null"`
    PatientRutSnapshot      string `db:"not_null"`
    DoctorNameSnapshot      string `db:"not_null"`
    DoctorSpecialtySnapshot string
    UpdatedAt               int64 `db:"not_null"`
}

type MeasurementType struct {
    ID          string
    Name        string `db:"not_null"`
    DefaultUnit string `db:"not_null"`
    MinNormal   float64
    MaxNormal   float64
    IsActive    bool `db:"not_null"`
    LoincCode   string
    UcumUnit    string
}

type ClinicalMeasurement struct {
    ID                string
    MedicalHistoryID  string  `db:"not_null"`
    MeasuredByStaffID string  `db:"not_null"`
    MeasurementTypeID string  `db:"not_null"`
    Value             float64 `db:"not_null"`
    Unit              string  `db:"not_null"`
    MeasuredAt        int64   `db:"not_null"`
    Notes             string
}

type HistoryDetail struct {
    ID                string
    MedicalHistoryID  string  `db:"not_null"`
    CatalogItemID     string  `db:"not_null"` // soft ref
    Quantity          int     `db:"not_null"`
    ItemNameSnapshot  string  `db:"not_null"`
    ItemCodeSnapshot  string  `db:"not_null"`
    ItemPriceSnapshot float64 `db:"not_null"`
}
