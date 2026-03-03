# FHIR Compliance — Master Plan

**Objective:** Achieve full technical FHIR compliance for the `clinical-encounter` module by implementing the three gaps identified in [`FHIR_ROADMAP.md`](FHIR_ROADMAP.md).

**Related docs:** [`ARCHITECTURE.md`](ARCHITECTURE.md) | [`SKILL.md`](SKILL.md) | [`FHIR_ROADMAP.md`](FHIR_ROADMAP.md)

---

## Development Rules

- **Agent Setup:** Run `go install github.com/tinywasm/devflow/cmd/gotest@latest` before anything.
- **Dependencies:** No external libraries. Standard library + `tinywasm/*` polyfills only (Use `tinywasm/json` instead of standard `encoding/json`).
- **Build Tags:** All server-side files MUST carry `//go:build !wasm`.
- **Database:** `model.go` is the source of truth. DO NOT edit `model_db.go` manually; run the `ormc` code generator after changes.
- **Testing:** Split logic per rule `_back_test.go` + `//go:build !wasm`. Run via `gotest` (no args). No external assertion libraries.
- **Publishing:** Update documentation FIRST, then run `gopush 'message'`. Never `git commit/push` directly.

---

## Architecture Decision

**Hybrid: Minimal Schema Evolution + Pure Adapter Pattern**

Add domain-meaningful fields (`StartedAt`, `FinishedAt`, `Cie10Code`, `LoincCode`, `UcumUnit`) directly to the schema — these are business-valid constructs, not FHIR artifacts. A standalone FHIR adapter (`fhir_types.go` + `fhir_adapter.go`) translates internal models to FHIR R4 JSON on demand. No REST API endpoints are added; no external interfaces (`EventPublisher`, `Module.New()`) change.

---

## Stage Checklist

| # | Stage | Files Touched | Status |
|---|-------|--------------|--------|
| 1 | [Schema Evolution](PLAN_STAGE_1_SCHEMA.md) | `model.go`, `model_db.go` | pending |
| 2 | [Business Logic Updates](PLAN_STAGE_2_LOGIC.md) | `mcp_visit_status.go`, `mcp_measurement_type.go` | pending |
| 3 | [FHIR Adapter Layer](PLAN_STAGE_3_ADAPTER.md) | `fhir_types.go` (new), `fhir_adapter.go` (new) | pending |
| 4 | [Tests & Documentation](PLAN_STAGE_4_TESTS.md) | `tests/fhir_adapter_back_test.go` (new), existing tests | pending |

---

## Verification

```bash
# Run full test suite after all stages are complete
gotest

# Publish if all tests pass
gopush 'feat: FHIR compliance — CIE-10, LOINC, UCUM, Period tracking, adapter layer'
```
