# Tasks — units as static sprites

Legend: **files** the task may change · **done when** the observable it must leave behind. Every
entry is an implementation task; they land in ascending order, and every dependency of a task is
a task before it.

## T1 — `pkg/sim`: the version-2 byte form

**files** `pkg/sim/world.go` (the `Entity` field), `pkg/sim/binary.go`,
`pkg/sim/binary_test.go`, `pkg/sim/hash_test.go`, `pkg/sim/step_test.go` (the lockstep case)

DD-1.

**done when** SC-1 passes (FR-1). `step.go`, `run.go` and `world_test.go` are not edited; no new
import and no new exported identifier enters the package; the pin bytes are hand-transcribed,
never captured from the encoder.

## T2 — `pkg/mapload`: the class key

**files** `pkg/mapload/fromalm.go`, `pkg/mapload/fromalm_test.go`

DD-2.

**done when** SC-2 passes (FR-2). `schedule.go` and `schedule_test.go` are not edited; no new
import enters the package.

## T3 — the unit-art bundle: render-tier types, game loader, fixtures

**files** new `pkg/render/terrain/units.go`, `pkg/render/terrain/units_test.go`; new
`pkg/game/units.go`, `pkg/game/units_test.go`; `internal/synth/reg.go`,
`internal/synth/reg_test.go` (the `UnitsReg` builder)

DD-3, DD-4, and DD-9's builder and archive fixtures.

**done when** SC-3 and SC-4 pass (FR-3, FR-8). `statics.go` in both packages is not edited —
`sheetCache` is called, never copied; no ebiten import enters either package; `UnitPlace`
consults no marker geometry; the allow map is untouched.

## T4 — `pkg/ui`: the entity layer

**files** `pkg/ui/overlay.go`, `pkg/ui/viewer.go`, `pkg/ui/entity_overlay_test.go`;
`pkg/game/world.go`, `pkg/game/world_test.go` (the setter's call sites)

DD-5's `ui` half and DD-6, landing with the seam's game-side call sites in one buildable slice.

**done when** SC-5 passes (FR-4, FR-6's window-tier clauses). `push` stays art-less here — every
entity crosses with nil art, resolution being T5's; the three diagnostic glyph builders, their
tests and the marker family in `pkg/render/terrain` are not edited; `drawStatics` and
`staticImage` keep their signatures.

## T5 — `pkg/game`: resolution at the seam, the front-end bundle, the instrument

**files** `pkg/game/world.go`, `pkg/game/frontend.go`, `pkg/game/world_test.go`;
`cmd/againrom/main_test.go` (the install fixture's unit registry and its omission case)

DD-5's game half, DD-7, and DD-8's front-end half.

**done when** SC-6, SC-7 and SC-8 pass (FR-5, FR-6, FR-7). `LoadMapViewer`, `MarkerCells` and
the schedule are unchanged; `cmd/againrom/main.go` gains no flag and no line; the failing
`-check` case is an install-fixture layout, not a mock.

## T6 — the census harness

**files** new `pkg/game/census.go`, `pkg/game/census_test.go`; `cmd/terraintool/main.go`; new
`cmd/terraintool/units_test.go`

DD-8's census half.

**done when** SC-9 passes (FR-3). `render`'s flags, output and tests are unchanged;
`terraintool` imports nothing new; nothing census-shaped reaches `cmd/mapview` or
`cmd/againrom`.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1 | DD-1, SC-1 |
| T2 | FR-2 | DD-2, SC-2 |
| T3 | FR-3, FR-8 | DD-3, DD-4, DD-9, SC-3, SC-4 |
| T4 | FR-4, FR-6 | DD-5, DD-6, SC-5 |
| T5 | FR-5, FR-6, FR-7 | DD-5, DD-7, DD-8, SC-6, SC-7, SC-8 |
| T6 | FR-3 | DD-8, SC-9 |
