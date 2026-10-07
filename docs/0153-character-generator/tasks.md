# 0153 — character-generator tasks

Reading key: FR/AC/P refer to `spec.md`; D/R and criterion numbers refer to `plan.md`.

## T1 — Pre-create and install-backed input

**Kind:** implementation.

**Boundary:** deliver the source-backed pre-create stage and its transition into a coherent detailed
stage shell; detailed controls, Card and Doll remain outside this slice.

**Files:** ADD `pkg/formats/textinput/input.go`, `pkg/formats/textinput/input_test.go`,
`pkg/ui/chargen_page.go`, `pkg/ui/chargen_page_test.go`, `pkg/game/chargenassets.go`,
`pkg/game/chargenassets_test.go`; MODIFY `internal/archtest/dag.go`,
`internal/archtest/dag_test.go`, `pkg/ui/chargen.go`, `pkg/ui/app.go`,
`pkg/ui/chargen_test.go`, `pkg/ui/chargen_app_test.go`, `pkg/game/chargen.go`,
`pkg/game/chargen_test.go`, `pkg/game/frontend.go`, `cmd/againrom/main_test.go`.

**Upstream:** FR-1 pre-create, FR-3 class/sex and Forward, FR-5 name/Forward/pre-create Back; AC-1
pre-create, AC-6, AC-12 source resolution; D-1 through D-4, D-5 pre-create arm, D-6, D-7, D-12;
R-1, R-2; criteria 3, 10, 12, 13.

**Scope fences:** do not edit `pkg/ui/panel.go`, `pkg/game/world.go` or release integration; do not
redesign mission routing, the live panel, figure composition, save bytes or simulation state.

**Done when:** the new format leaf and source loader tests pass; criteria 3, 10, 12 and 13 pass; the
detailed shell remains buildable; `go test -trimpath ./...` passes.

## T2 — Detailed editor and confirmed-player preview

**Kind:** implementation.

**Boundary:** complete the detailed stage, live preview and generated-character continuity on top of
T1 without changing the pre-create source boundary.

**Files:** MODIFY `pkg/ui/chargen.go`, `pkg/ui/chargen_page.go`, `pkg/ui/panel.go`,
`pkg/ui/app.go`, `pkg/ui/chargen_test.go`, `pkg/ui/chargen_app_test.go`,
`pkg/ui/chargen_page_test.go`, `pkg/game/chargen.go`, `pkg/game/chargen_test.go`,
`pkg/game/chargenassets.go`, `pkg/game/chargenassets_test.go`, `pkg/game/world.go`,
`pkg/game/panelchars.go`, `pkg/game/release_integration_test.go`, `cmd/againrom/main_test.go`.

**Upstream:** FR-1 detailed, FR-2 through FR-5; AC-1 detailed, AC-2 through AC-5, AC-7 through
AC-12; P-1 through P-6; D-2 through D-5, D-8 through D-12; R-1, R-3 through R-6; criteria 1, 2,
4–9, 11.

**Scope fences:** do not add archive readers or codecs, change mission selection, alter the live
300-pixel panel layout or figure layer order, add persistent preview state, or change the sim/save
byte form. Keep T1's source-path and name-input tests intact.

**Done when:** the cited named tests cover all 20 combinations, refusal/purity, byte-identical
production-sheet parity outside the explicitly contextual `CELL` value, one source-positioned patch per control, the class column and
mask geometry, framed Card/Doll replacement and Back/Reset/Play dispatch; continuity crosses launch, save/load and campaign;
the runnable executable builds, mission 10/20 census matches baseline, and `go test -trimpath ./...`
passes.

## Traceability

| Requirement | Plan criteria | Task |
|---|---|---|
| FR-1, AC-1, AC-12 | 4, 5, 10–13 | T1, T2 |
| FR-2, AC-2, P-1 | 1, 2 | T2 |
| FR-3, AC-3, P-2 | 1, 6 | T1, T2 |
| FR-4, AC-4, AC-5, AC-11, P-3, P-6 | 4, 6, 7 | T2 |
| FR-5, AC-6 through AC-10, P-4, P-5 | 1, 3, 5, 7–9, 13 | T1, T2 |
