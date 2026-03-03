← [Master Plan](PLAN.md) | Next → [Stage 2](PLAN_STAGE_2_LOGIC.md)

# Stage 1 — Schema Evolution

**Goal:** Add FHIR-meaningful fields to `MedicalHistory` and `MeasurementType` without breaking any existing logic.

---

## 1.1 `model.go` — Add fields to structs

### `MedicalHistory` — append after `Prescription string`:

```go
Cie10Code  string // optional CIE-10 diagnosis code (e.g., "J00")
StartedAt  int64  // Unix ts set when StartVisit() is called — FHIR Period.start
FinishedAt int64  // Unix ts set when CompleteVisit() or CancelVisit() — FHIR Period.end
```

### `MeasurementType` — append after `IsActive bool`:

```go
LoincCode string // optional LOINC code (e.g., "85354-9" = blood pressure)
UcumUnit  string // optional UCUM-standardized unit (e.g., "mm[Hg]")
```

No changes to `ClinicalMeasurement` or `HistoryDetail`.

---

## 1.2 `model_db.go` — ORM Generation

> `model_db.go` is generated automatically. Do not edit it manually.

After updating `model.go`, run the ORM compiler from your terminal to regenerate the database schema file:

```bash
ormc
```

---

## Verification for Stage 1

Run `gotest` — all existing tests must still pass. No new behavior yet; fields are nullable/optional with zero values.

← [Master Plan](PLAN.md) | Next → [Stage 2](PLAN_STAGE_2_LOGIC.md)
