# Tasks — a popup takes everything onto itself

**Reading key.** `FR`/`AC`/`P` → `spec.md`; `DD-x` → `plan.md` §Design decisions; `R-x` → §Risks;
criterion numbers → §Success criteria.

---

## T1 — the one answer, and the map arm's gate

**Kind:** implementation.

**Boundary:** the predicate and every input the map arm resolves. It stops before the camera, before
either clock and before the draw method.

**Files:** `pkg/ui/popup.go` `ADD` · `pkg/ui/flow.go` `MODIFY` · `pkg/ui/app.go` `MODIFY` ·
`pkg/ui/popup_test.go` `ADD` · `pkg/ui/notice_test.go` `MODIFY`

**Covers:** FR-1, FR-2, FR-8 · AC-1, AC-2, AC-8, AC-10 · P-4 · DD-1, DD-2, DD-3, DD-8, DD-9 · R-1,
R-2 · criteria 1, 8, 9

**Scope fences:** do not gate the cadence call, the world advance, or the viewer's step — each must
still run on every frame a popup is open on. Do not add a condition to any individual order path.
Do not touch the draw method, the camera, or either clock. Leave every `NoticeOpen` test that serves
the popup's own three dismissal inputs exactly where it is: those are the notice's, not the popup's.

**Done when:** with a popup open, a frame carrying a press, a drag, a release, a secondary press,
both blow keys and both diagnostic keys changes no selection and reaches neither the order seam nor
the blow seam nor either diagnostic, while the same frame still advances the world once and still
declares the stop; the shipped assertion that every other map-screen input reaches the map is
amended to the three that do; and the package builds and tests green.

---

## T2 — the viewer holds still

**Kind:** implementation.

**Boundary:** the camera, the gesture latches and the ambient clock, all inside the viewer's own
step. It stops before the draw method.

**Files:** `pkg/ui/viewer.go` `MODIFY` · `pkg/ui/popup_test.go` `MODIFY`

**Covers:** FR-3, FR-4, FR-5, FR-7 · AC-3, AC-4, AC-5, AC-5a, AC-7, AC-9 · P-1, P-2 · DD-4, DD-5,
DD-6, DD-9 · R-2, R-4 · criteria 2, 3, 4, 4a, 6, 7

**Scope fences:** the step must still be called and must still take its clock baseline and its cursor
memo — do not gate the call, gate inside it. Do not add a stop to the ticker type or to its setter's
signature. Do not touch the far side's suspension or anything under `pkg/game`. Correct the setter's
doc paragraph that names the superseded behaviour as a decision; do not delete it.

**Done when:** criteria 2, 3, 4, 4a, 6 and 7 each have their named test passing, every one of them
driving the popup-open case against a popup-closed control on the same drive; and the package builds
and tests green.

---

## T3 — the dim covers everything but the popup

**Kind:** implementation.

**Boundary:** where the dim stands in the composed frame, and what raises it.

**Files:** `pkg/ui/viewer.go` `MODIFY` · `pkg/ui/notice.go` `MODIFY` ·
`pkg/ui/backdrop_test.go` `MODIFY` · `pkg/ui/popup_test.go` `MODIFY`

**Covers:** FR-1, FR-6, FR-7 · AC-6, AC-7, AC-9 · P-1 · DD-1, DD-7, DD-10 · R-3 · criteria 5, 6, 7

**Scope fences:** do not change the dim's colour, its alpha, its rectangle, the seam that supplies it
or the frames it appears on. Do not add a second composition statement. Do not exempt the readout.
Amend the shipped order assertion rather than removing it.

**Done when:** the order assertion reads the amended contract and passes — the dim after the map
picture, after the unit panel and after the readout, before the popup — a frame with no popup and a
frame under a fully transparent value are each unchanged, and the package builds and tests green.

---

## T4 — the runnable build

**Kind:** manual runbook.

**Boundary:** `builds/0077-popup-modality/` only. It commits nothing.

**Files:** none tracked.

**Covers:** R-5 · criterion 10

**Done when:** the directory holds this branch's binary and a `README.md` giving the exact
invocation against a lawful install through `-assets`/`AGAINROM_ASSETS`, and naming the four things
the product's author asked to see.

---

## Traceability

| Spec | Plan criterion | Task |
|---|---|---|
| FR-1 | 1, 2, 5, 6 | T1, T3 |
| FR-2, AC-1, AC-2 | 1 | T1 |
| FR-3, AC-3, P-2 | 2 | T2 |
| FR-4, AC-4 | 3 | T2 |
| FR-5, AC-5, AC-5a | 4, 4a | T2 |
| FR-6, AC-6 | 5 | T3 |
| FR-7, AC-7, P-1 | 6 | T2, T3 |
| FR-8, AC-10, P-4 | 8 | T1 |
| AC-8 | 9 | T1 |
| AC-9 | 7 | T2, T3 |
| P-3 | — | none of T1–T3 touches `pkg/sim` or anything under it |
| R-5 | 10 | T4 |
