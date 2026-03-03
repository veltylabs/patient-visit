# PLAN: patient-visit module

## Development Rules

- **First Setup:** Run `go install github.com/tinywasm/devflow/cmd/gotest@latest` explicitly before attempting to test.
- **No external assertion libraries** — standard `testing` only.
- **Tests:** No IO / side-effects. MUST use basic structs to mock external interfaces like `EventPublisher`.
- **No global state** — all dependencies injected via `New(db, pub)`.
- **Max 500 lines per file** — split by domain if exceeded.
- **Build tags:** Code is isomorphic by default (no tags). Add `//go:build !wasm` **ONLY** to server-side specific files (e.g. database setup, HTTP routes).
- **Use `tinywasm` replacements** for frontend compatibility: `tinywasm/fmt` (not `fmt`/`strings`/`strconv`/`errors`), `tinywasm/time` (not `time`), and `tinywasm/json` (not `encoding/json`), `tinywasm/unixid` for all PKs.
- **Tests use `tinywasm/sqlite` in-memory** — no real DB required.
- **Run tests** with `gotest`; **publish** with `gopush 'message'`.
- **Documentation first** — update docs before touching source.

---

> **Context:** See the project's [README.md](../../README.md) or [ARCHITECTURE.md](ARCHITECTURE.md) for global domain scope.

---

## Architectural Decisions

| Decision | Resolution |
|---|---|
| DB driver | Agnostic — `New(db, pub)` accepts any driver (sqlite for tests, postgres in prod) |
| API style | Domain-oriented MCP tools (~16 handlers) |
| measurement_type | Fully manageable globally (create / list / toggle) |
| Snapshots | Caller provides them as input args (no cross-module calls) |
| Pagination | `limit` (default 20) + `offset` (default 0) on list handlers |
| Visit status | FSM: `created→arrived→triaged→in_progress→completed\|cancelled` (FHIR-aligned) |
| Timestamp | Single `AttentionAt int64` (Unix) — no separate date+time fields |
| Audit | `UpdatedAt int64` on `MedicalHistory` for state change tracking |
| Notifications | `EventPublisher` interface injected — nil-safe (no-op when nil) |

---

## Visit Flow

See [`diagrams/sequence_visit_flow.md`](diagrams/sequence_visit_flow.md)

---

## Execution Stages

| Stage | File | Contents |
|---|---|---|
| 1 | [PLAN_STAGE_1_MODELS.md](PLAN_STAGE_1_MODELS.md) | Domain models, status constants, FSM map |
| 2 | [PLAN_STAGE_2_ORM.md](PLAN_STAGE_2_ORM.md) | ORM code generation (`model_db.go`) |
| 3 | [PLAN_STAGE_3_CORE.md](PLAN_STAGE_3_CORE.md) | Module struct, `New()`, `EventPublisher` |
| 4 | [PLAN_STAGE_4_VISIT.md](PLAN_STAGE_4_VISIT.md) | Visit CRUD + `ListVisitsByDoctor` |
| 5 | [PLAN_STAGE_5_FSM.md](PLAN_STAGE_5_FSM.md) | Visit status FSM handlers |
| 6 | [PLAN_STAGE_6_MEASUREMENT_TYPE.md](PLAN_STAGE_6_MEASUREMENT_TYPE.md) | MeasurementType handlers |
| 7 | [PLAN_STAGE_7_MEASUREMENT.md](PLAN_STAGE_7_MEASUREMENT.md) | ClinicalMeasurement handlers |
| 8 | [PLAN_STAGE_8_DETAIL.md](PLAN_STAGE_8_DETAIL.md) | HistoryDetail handlers |
| 9 | [PLAN_STAGE_9_TESTS.md](PLAN_STAGE_9_TESTS.md) | Test setup + coverage matrix |
| 10 | [PLAN_STAGE_10_PUBLISH.md](PLAN_STAGE_10_PUBLISH.md) | go.mod + publish |
