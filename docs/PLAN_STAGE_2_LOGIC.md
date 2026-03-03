← [Stage 1](PLAN_STAGE_1_SCHEMA.md) | Next → [Stage 3](PLAN_STAGE_3_ADAPTER.md)

# Stage 2 — Business Logic Updates

**Goal:** Wire the new schema fields into FSM transitions and the MeasurementType creation flow. Three functions currently use `applyTransition()` and must be inlined to set extra fields before `db.Update()`.

---

## 2.1 `mcp_visit_status.go` — Inline `StartVisit`, `CompleteVisit`, `CancelVisit`

### `StartVisitArgs` — no change

### `StartVisit()` — replace `applyTransition` delegation with inline transition:

```go
func (m *Module) StartVisit(args StartVisitArgs) (*MedicalHistory, error) {
	if args.ID == "" {
		return nil, fmt.Err("missing", "id")
	}
	visit, err := getVisitByID(m.db, args.ID)
	if err != nil {
		return nil, err
	}
	next, ok := visitTransitions[visit.Status]["start_visit"]
	if !ok {
		return nil, fmt.Err("invalid", "transition", visit.Status, "->", "start_visit")
	}
	visit.Status = next
	visit.StartedAt = time.Now()
	visit.UpdatedAt = time.Now()
	return visit, m.db.Update(visit)
}
```

---

### `CompleteVisitArgs` — extend with outcome fields:

```go
type CompleteVisitArgs struct {
	ID           string `json:"id"`
	Diagnostic   string `json:"diagnostic,omitempty"`
	Prescription string `json:"prescription,omitempty"`
	Cie10Code    string `json:"cie10_code,omitempty"`
}
```

### `CompleteVisit()` — replace `applyTransition` delegation:

```go
func (m *Module) CompleteVisit(args CompleteVisitArgs) (*MedicalHistory, error) {
	if args.ID == "" {
		return nil, fmt.Err("missing", "id")
	}
	visit, err := getVisitByID(m.db, args.ID)
	if err != nil {
		return nil, err
	}
	next, ok := visitTransitions[visit.Status]["complete"]
	if !ok {
		return nil, fmt.Err("invalid", "transition", visit.Status, "->", "complete")
	}
	visit.Status = next
	visit.FinishedAt = time.Now()
	visit.UpdatedAt = time.Now()
	if args.Diagnostic != "" {
		visit.Diagnostic = args.Diagnostic
	}
	if args.Prescription != "" {
		visit.Prescription = args.Prescription
	}
	if args.Cie10Code != "" {
		visit.Cie10Code = args.Cie10Code
	}
	if err := m.db.Update(visit); err != nil {
		return nil, err
	}
	m.publish(EventVisitCompleted, map[string]any{
		"visit_id":   visit.ID,
		"patient_id": visit.PatientID,
		"doctor_id":  visit.DoctorID,
	})
	return visit, nil
}
```

---

### `CancelVisit()` — replace `applyTransition` delegation:

```go
func (m *Module) CancelVisit(args CancelVisitArgs) (*MedicalHistory, error) {
	if args.ID == "" {
		return nil, fmt.Err("missing", "id")
	}
	visit, err := getVisitByID(m.db, args.ID)
	if err != nil {
		return nil, err
	}
	next, ok := visitTransitions[visit.Status]["cancel"]
	if !ok {
		return nil, fmt.Err("invalid", "transition", visit.Status, "->", "cancel")
	}
	visit.Status = next
	visit.FinishedAt = time.Now()
	visit.UpdatedAt = time.Now()
	if err := m.db.Update(visit); err != nil {
		return nil, err
	}
	m.publish(EventVisitCancelled, map[string]any{
		"visit_id": visit.ID,
		"reason":   args.Reason,
	})
	return visit, nil
}
```

> **Note:** `MarkTriaged` still uses `applyTransition` — no changes needed there.

---

## 2.2 `mcp_measurement_type.go` — Add `LoincCode` and `UcumUnit`

### `CreateMeasurementTypeArgs` — extend with optional coding fields:

```go
type CreateMeasurementTypeArgs struct {
	Name        string  `json:"name"`
	DefaultUnit string  `json:"default_unit"`
	MinNormal   float64 `json:"min_normal,omitempty"`
	MaxNormal   float64 `json:"max_normal,omitempty"`
	LoincCode   string  `json:"loinc_code,omitempty"`
	UcumUnit    string  `json:"ucum_unit,omitempty"`
}
```

### `CreateMeasurementType()` — add fields to record construction:

```go
record := &MeasurementType{
	ID:          m.uid.GetNewID(),
	Name:        args.Name,
	DefaultUnit: args.DefaultUnit,
	MinNormal:   args.MinNormal,
	MaxNormal:   args.MaxNormal,
	IsActive:    true,
	LoincCode:   args.LoincCode,
	UcumUnit:    args.UcumUnit,
}
```

---

## Verification for Stage 2

```bash
gotest
```

All existing tests must still pass. The `TestVisitStatusFSM_HappyPath` and `TestVisitStatusFSM_Cancel` tests will continue working because `CompleteVisitArgs{ID: visit.ID}` is still valid (new fields are optional).

← [Stage 1](PLAN_STAGE_1_SCHEMA.md) | Next → [Stage 3](PLAN_STAGE_3_ADAPTER.md)
