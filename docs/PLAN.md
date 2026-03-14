# Migrate to tinywasm/orm v2 API (fmt.Field)

> **Note**: Previous FHIR stage plans preserved in `PLAN_STAGE_*.md` files.

## Context

The ORM code generator (`ormc`) now produces `Schema() []fmt.Field` (from `tinywasm/fmt`) with individual bool constraint fields instead of the old `[]orm.Field` with bitmask constraints. The `Values()` method is removed; consumers use `fmt.ReadValues(schema, ptrs)` instead.

### Key API Changes

| Old (current) | New (target) |
|---|---|
| `[]orm.Field{...Constraints: orm.ConstraintPK}` | `[]fmt.Field{...PK: true}` |
| `orm.TypeText`, `orm.TypeInt64`, `orm.TypeBool` | `fmt.FieldText`, `fmt.FieldInt`, `fmt.FieldBool` |
| `m.Values() []any` | `fmt.ReadValues(m.Schema(), m.Pointers())` |
| `var MedicalHistory_ = struct{...}` | Verify `_` suffix consistency |

### Models in scope

- `MedicalHistory`
- `MeasurementType`
- `ClinicalMeasurement`
- `HistoryDetail`

### Target fmt.Field Struct (`tinywasm/fmt`)

```go
type Field struct {
    Name    string
    Type    FieldType // FieldText, FieldInt, FieldFloat, FieldBool, FieldBlob, FieldStruct
    PK      bool
    Unique  bool
    NotNull bool
    AutoInc bool
    Input   string
    JSON    string
}
```

### Generated Code per Struct (`ormc`)

- `TableName() string`, `FormName() string`
- `Schema() []fmt.Field`, `Pointers() []any`
- `T_` metadata struct with typed column constants
- `ReadOneT(qb *orm.QB, model *T)`, `ReadAllT(qb *orm.QB)`

---

## Stage 1 — Regenerate ORM Code

**File**: `model_orm.go` (auto-generated)

1. Update `ormc`: `go install github.com/tinywasm/orm/cmd/ormc@latest`
2. Run `ormc` from project root
3. Verify all 4 models generated with `fmt.Field`, bool constraints
4. Verify `_` suffix meta structs: `MedicalHistory_`, `MeasurementType_`, `ClinicalMeasurement_`, `HistoryDetail_`

---

## Stage 2 — Update Handwritten Code

**Files**: `mcp.go`, `mcp_visit.go`

1. If meta struct names changed, update all references
2. Search for `.Values()` calls → replace with `fmt.ReadValues(m.Schema(), m.Pointers())`
3. Add `"github.com/tinywasm/fmt"` import where needed

> **Note**: `.Where()`, `.Eq()`, `.Gte()`, `.Lte()`, `.OrderBy()`, `.Desc()`, `.Limit()`, `.Offset()`, `ReadAll*` — all unchanged.

---

## Stage 3 — Update Tests

**Files**: `tests/setup_test.go`

1. If tests construct `orm.Field` literals, update to `fmt.Field`
2. Run tests to verify no regressions

---

## Stage 4 — Update go.mod

1. Run `go mod tidy`

---

## Verification

```bash
gotest
```

## Linked Documents

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [SKILL.md](SKILL.md)
- [FHIR_ROADMAP.md](FHIR_ROADMAP.md)
