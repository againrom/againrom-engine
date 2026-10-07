# Tasks — animated water in the map viewer

**Reading key.** `FR-x`/`AC-x`/`P-x` → `spec.md`; `SC-x` → `plan.md` §Success criteria; `DD-x` →
`plan.md` §Design decisions; `R-x` → `plan.md` §Risks. Task kinds: **implementation** (one coherent
product change → exactly one implementation commit, trailer `SDD-Task: 0006-water-animation/T<n>`),
**developer-run verification** (agent authors, a human runs against a lawful install; no implementation
commit).

`pkg/render/terrain`, `pkg/ui` and `cmd/mapview` are already registered in the DAG allow-map and
`docs/ARCHITECTURE.md`, and this story adds no package, so **no** `dag.go`/`ARCHITECTURE.md` edit is part
of any task.

## T1 — the decoded animation model  *(implementation)*

Add the whole model as pure arithmetic, and re-express the static path as its phase-0 case so the two
cannot drift.

- Files: `pkg/render/terrain/water.go` (ADD) — `IsWaterGroup`, `WaterPhase`, `ResolveAnimated`, the
  water/cycle constants, the speed table with `TicksPerSecond`/`TickMillis` (clamping, integer division),
  and `Ticker` with `Advance` dividing rather than looping (DD1, DD3, DD4, DD5).
  `pkg/render/terrain/mapping.go` (EDIT) — `Resolve` keeps its animation-off meaning and delegates to the
  shared helper (DD2); update the doc comments that say cycling "is story 0006".
  `pkg/render/terrain/water_test.go` (ADD) — the synthetic suite.
- Covers: FR-1, FR-2, FR-3, FR-4, FR-6; AC-1…AC-9; P-1…P-5; SC-1…SC-9, SC-11; DD1–DD6.
- **Done when:** `TestWaterGroupClassification`, `TestWaterPhaseFormula`, `TestResolveAnimatedWater`,
  `TestWaterVariantReachability`, `TestResolveAnimatedPassesNonWaterThrough`, `TestWaterCycleCadence`,
  `TestAdjacentCellsRipple`, `TestStaticEqualsPhaseZero`, `TestSpeedTableAndTickMillis`,
  `TestTickerAccumulates` and `TestTickerConservesTime` pass; `go build ./...`, `go vet ./...`,
  `gofmt -l` (tracked `*.go`), `go test ./...`, `internal/archtest` and
  `bash scripts/check-no-game-assets.sh` are clean with no game install present.

## T2 — drive it from the viewer  *(implementation)*

Wire the model into the run loop and expose the game's switches.

- Files: `pkg/ui/viewer.go` (EDIT) — hold a `terrain.Ticker` and the animation flag, advance the counter
  from measured elapsed milliseconds in `Update` (the only clock read, DD4), pass world `(col,row)` and
  the counter through the draw path, and add `SetAnimated`/`SetSpeedIndex`/`Animation`; default to
  animation on at `DefaultSpeedIndex`.
  `pkg/ui/viewer_test.go` (EDIT/ADD) — animation defaults and setters, asserted without an engine context.
  `cmd/mapview/main.go` (EDIT) — `-noanimation` and `-speed <0..8>` flags, applied before the window
  opens; append the cadence to the `-check` summary.
  `cmd/mapview/main_test.go` (EDIT) — the flags reach the viewer and `-check` reports the cadence.
- Covers: FR-5; SC-10; R-2.
- **Done when:** `TestViewerAnimation`, `TestCheckReportsCadence` and `TestAnimationFlags` pass; all gates
  above stay clean with no game install present.

## T3 — AC-10 evidence against a lawful install  *(developer-run verification)*

Run `mapview` against a lawful install on a map with substantial water and record **evidence only** — no
game bytes.

- Produces: evidence in `verification.md` — that water cycles, that the ripple is diagonal and does not
  shift while panning (the scroll-stability that distinguishes the decoded world-coordinate model from a
  screen-space artefact), that `-noanimation` freezes it to the 0004 image, and that `-speed` changes the
  rate; plus the headless `-check` cadence line for every shipped map.
- Covers: AC-10; SC-12; R-3.
- **Done when:** `verification.md` records the run (or an explicit "not run" limitation with the reason);
  `bash scripts/check-no-game-assets.sh` stays clean.

## Traceability

| Requirement (spec) | Plan criterion | Task |
|---|---|---|
| FR-1 (phase, total) | SC-1, SC-2, SC-5 | T1 |
| FR-2 (animated resolve) | SC-3, SC-4 | T1 |
| FR-3 (static == phase 0) | SC-7 | T1 |
| FR-4 (cadence + accumulator) | SC-8, SC-9 | T1 |
| FR-5 (viewer drives it) | SC-10 | T2 |
| FR-6 (purity / DAG) | SC-11 | T1 |
| AC-1 | SC-1 | T1 |
| AC-2 | SC-2, SC-3 | T1 |
| AC-3 | SC-4 | T1 |
| AC-4 | SC-5 | T1 |
| AC-5 | SC-3 | T1 |
| AC-6 | SC-6 | T1 |
| AC-7 | SC-8 | T1 |
| AC-8 | SC-9 | T1 |
| AC-9 | SC-7 | T1 |
| AC-10 | SC-12 | T3 |
| P-1 (phase in 0..3 always) | SC-2 | T1 |
| P-2 (slot never escapes 0..127) | SC-3 | T1 |
| P-3 (non-water passthrough) | SC-4 | T1 |
| P-4 (accumulator conserves time) | SC-9 | T1 |
| P-5 (low 2 counter bits inert) | SC-5 | T1 |
