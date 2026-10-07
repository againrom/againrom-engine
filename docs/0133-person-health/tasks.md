# tasks — 0133

Reading key: `FR`/`AC`/`P` → `spec.md`; `DD`/`R`/`SC` → `plan.md` (§Design decisions, §Risks,
§Success criteria). Both tasks below are **implementation**: one coherent product change each.

## T1 — the graph's health arm, and a definition that can answer for its own row

**Kind:** implementation.
**Boundary:** stops inside the derived-stat graph and the definition beside it. No caller of the new
accessor is changed here, and the world a map builds is byte-identical after this task.

**Files:** `pkg/data/recompute.go` MODIFY · `pkg/data/humandef.go` MODIFY ·
`pkg/data/chargenbase.go` MODIFY (doc only) · `pkg/data/recompute_test.go` MODIFY ·
`pkg/mapload/start.go` MODIFY (doc only) · `pkg/mapload/pools_test.go` MODIFY

**Covers:** FR-2, FR-4, FR-5, FR-8 · AC-2, AC-3, AC-4, AC-7, AC-11 · P-3 · DD-1, DD-2, DD-3, DD-4,
DD-5, DD-8 · R-2 · SC-2, SC-3, SC-4, SC-7, SC-9.

**Scope fences.** Do not change the mana arm's gate, its multiplier or its arithmetic. Do not remove
the column flag from the profile type or change what sets it. Do not touch the placement arm's own code,
any command under `cmd/`, or any test outside the two files named above. Two landed tests assert the
behaviour FR-4 replaces — re-pin each at the corrected number with the reason it moved; delete
neither.

**Done when:** `go build ./... && go vet ./... && go test -count=1 -trimpath ./...` is green, and
reverting only the gate change in `recompute.go` makes a named test fail.

## T2 — a placement mints the derived maximum, and the report shows it

**Kind:** implementation.
**Boundary:** the person rung of the placement arm and the definition-table report. The graph and
the accessor T1 added are read, never edited.

**Files:** `pkg/mapload/fromalm.go` MODIFY · `pkg/mapload/fromalm_test.go` MODIFY ·
`pkg/mapload/human_test.go` MODIFY · `cmd/classdump/databin.go` MODIFY ·
`cmd/classdump/databin_test.go` MODIFY

**Covers:** FR-1, FR-3, FR-6, FR-7, FR-9, FR-10, FR-11 · AC-1, AC-5, AC-6, AC-8 · P-1, P-2, P-4 ·
DD-1, DD-6, DD-7, DD-9 · R-1, R-3, R-4 · SC-1, SC-5, SC-6, SC-8, SC-10, SC-11.

**Scope fences.** Do not edit `pkg/data`. Do not touch a placement's mana pair, its periods, its
combat numbers or its worn set. Add no new command and no new flag. No shipped row name, count or
number may appear in source — every one the report prints is read off the install at run time. Five
landed tests assert the health FR-1 replaces, four in the placement package and one in the tool's
own; re-pin each with the reason it moved and delete none.

**Done when:** `go build ./... && go vet ./... && go test -count=1 -trimpath ./...` is green; the
report's per-placement line carries the owning slot and its census block prints the figures FR-10
names, both exercised by a synthetic fixture in the tool's own test.

## Traceability

| Spec | Plan | Task |
|---|---|---|
| FR-2, FR-4, FR-5, FR-8; AC-2, AC-3, AC-4, AC-7, AC-11; P-3 | DD-1…DD-5; R-2; SC-2, SC-3, SC-4, SC-7, SC-9 | T1 |
| FR-1, FR-3, FR-6, FR-7, FR-9, FR-10, FR-11; AC-1, AC-5, AC-6, AC-8; P-1, P-2, P-4 | DD-1, DD-6, DD-7, DD-9; R-1, R-3, R-4; SC-1, SC-5, SC-6, SC-8, SC-10, SC-11 | T2 |
| AC-9, AC-10 | SC-10, SC-11, SC-12 | verification stage |
