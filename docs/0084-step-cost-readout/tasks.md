# Tasks — the step cost, relative to the mover

Legend: **Kind** is `impl` (one coherent product change, one trailered commit). `Done when:` is the
entry's own exit condition. `SC-n` is `plan.md` §Success criteria; `R-n` is §Risks.

## T1 one rate, one site, and a read that reaches it

Kind: impl. Carries FR-1, FR-2, FR-3, FR-5, FR-6, FR-7, FR-8, DD-1, DD-3, DD-4, DD-5, DD-9, P-1, P-2, R-1.
Criteria SC-1, SC-2, SC-3, SC-4, SC-8.
Files: `pkg/sim/step.go` MODIFY, `pkg/sim/steprate_test.go` ADD.

Boundary: from a world and an ordered pair of cells to the law's rate and transit, and from an
entity id and a destination to whether there is an answer at all. Nothing here learns a cursor
exists.

Scope fence: `rate.go` is not edited — a world in one of its signatures is the shape that file
exists to refuse; `costAt`, `heightAt`, `describes`, `moverSpeed` and `rated` are read, never
changed; no state field is added and `formatVersion` is untouched; the advance's five-outcome
contract, stall count, occupancy writes and phase order stand. No existing test file is edited —
R-1's whole evidence is that they pass as they stand.

Done when: the advance computes its transit through the new function; the query returns what the
advance does for the same mover and pair, and separates the law's two arms at a mean cost other
than 8; it refuses each of DD-5's four causes; it reports the eight neighbours adjacent and a
two-cell-distant pair not; downhill beats its uphill reverse and a diagonal transit exceeds its
orthogonal twin; a world's encoded bytes and digest are unchanged; and the whole gate is clean.

## T2 the box states what a step would cost

Kind: impl. Carries FR-1, FR-2, FR-4, FR-5, FR-6, FR-9, DD-1, DD-2, DD-6, DD-7, DD-8, P-3, P-4, R-2, R-4.
Criteria SC-5, SC-6, SC-7, SC-9, SC-10.
Files: `pkg/ui/readout.go` MODIFY, `pkg/ui/viewer.go` MODIFY, `pkg/ui/stepcost_test.go` ADD.

Boundary: from a resolved cursor cell and a selected unit to two rows of text, and the seam that
carries the question out. Everything crossing is a builtin or a struct of builtins.

Scope fence: `panel.go` is not edited and no unit-panel field number, row, label or colour moves; no
readout field number below 28 moves and no existing row's label or layout position changes;
`readoutHidden` keeps its inverted storage and its zero value, and none of the three readout
visibility methods changes behaviour; the path below `panelItem` is untouched; the selected unit is
reached through the existing filter, not a second walk.

R-2's failure is a panic on the *second* composed frame, so the test that exposes it composes two.

Done when: both rows state the query's values for a selected rated unit with the cursor on another
on-map cell; each of FR-5's four causes produces its specified result, including a viewer handed no
query; a distant pair carries the marker and an adjacent one does not; changing either value alone
recomposes the box; a second composition does not panic; and the whole gate is clean.

## T3 the world answers the box

Kind: impl. Carries FR-3, FR-9, DD-10. Criteria SC-2, SC-5.
Files: `pkg/game/world.go` MODIFY, `pkg/game/stepcost_test.go` ADD, `pkg/ui/readout.go` MODIFY.

Boundary: the one statement that hands the viewer the world's own query, at the point the readout's
first push already happens — and the accessor that makes its arrival observable, which is
`ReadoutState`'s twin and carries its reason.

Scope fence: `pushReadout` keeps its four fields and its shown-only guard; the tick-0 push order is
not reordered; no entity-seam field is added; the query is installed once and not re-installed per
frame; nothing here recomputes, caches or interprets the returned numbers.

Done when: a viewer built by the tier that owns a world answers the query for a live rated entity
with the world's own figures, and a mission opened and left stopped states them on its first frame;
and the whole gate is clean.

## Traceability

| Requirement | Task |
|---|---|
| FR-1, FR-2 | T1, T2 |
| FR-3 | T1, T3 |
| FR-4 | T2 |
| FR-5, FR-6 | T1, T2 |
| FR-7, FR-8 | T1 |
| FR-9 | T2, T3 |
| P-1, P-2 | T1 |
| P-3, P-4 | T2 |
