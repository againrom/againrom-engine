# Tasks — 0052-flyers-fly

**Reading key.** `FR`/`AC`/`P` → `spec.md`. `DD-x`, `R-x`, `SC-x` → `plan.md`. Every task below is
**implementation**: one coherent product change, one commit. There is no characterization task —
the table-free path's digest, byte-form length and entity set are already pinned in the tree from
before the definition table reached this package, and that pin is what AC-3 is judged by.

## T1 — a definition table a test can write

Files: `internal/synth/databin.go` ADD · `internal/synth/databin_test.go` ADD

Covers: R-5, DD-11.

Boundary: the writer and its own round-trip test only. It lands before anything requires a table,
so no commit in the history is red for want of a fixture.

Scope fences: do not touch the parser it must satisfy; do not give the writer knowledge of what any
column means — a caller says which slots carry what.

Done when: a table written by it parses, every collection and entry reads back as written, and the
suite is green.

## T2 — the column reaches the entity

Files: `pkg/mapload/spawn.go` MODIFY · `pkg/mapload/fromalm.go` MODIFY ·
`pkg/mapload/spawn_test.go` MODIFY · `pkg/mapload/fromalm_test.go` MODIFY

Covers: FR-1, FR-2, FR-3, AC-1, AC-2, AC-3, AC-7, AC-10, P-1, P-2, P-4, SC-1, SC-2, SC-3, SC-7,
SC-8, DD-1, DD-4.

Boundary: the map-loading tier alone. Nothing outside it learns the word "domain" in this commit.

Scope fences: do not export the mapping — T6 needs no export (DD-2). Do not alter the existing
pre-story pin's literals; if they move, the change is wrong, not the pin. Do not touch the block
plane or its derivation — another story owns it this cycle.

Done when: the mapping's five cases, the four arms, the pin and the two health maxima all assert,
and `go test ./pkg/mapload/...` is green.

## T3 — a flyer crosses what stops a walker

Files: `pkg/mapload/domain_test.go` ADD

Covers: FR-4, AC-4, AC-5, SC-4, SC-5.

Boundary: a world built from a synthetic map and table, ordered and stepped. It demonstrates the
simulation's existing behaviour through the join; it changes no behaviour.

Scope fences: if an assertion here fails, the defect is upstream of this story — report it, do not
edit `pkg/sim` to make the test pass. Assert the ground mover over every tick, not only the last.

Done when: both fixtures assert as AC-4 and AC-5 state, and the suite is green.

## T4 — the install must carry the table

Files: `pkg/game/archives.go` MODIFY · `pkg/game/table.go` ADD · `pkg/game/table_test.go` ADD ·
`pkg/game/archives_test.go` MODIFY · `pkg/game/address_census_test.go` MODIFY ·
`cmd/againrom/main_test.go` MODIFY · `cmd/againrom/main.go` MODIFY

Covers: FR-6, AC-6, P-3, SC-6, DD-5, DD-6, DD-12.

Boundary: opening the archive and loading the table. The front-end does not yet hold the result and
no world changes.

Scope fences: the two command files are in this commit and not a later one, because the archive
becomes required here and their fixtures go red the moment it does. Do not reach for the developer
tool's spelling of the entry path (DD-12).

Done when: the three failure shapes each fail construction naming what failed, a root carrying a
written table constructs, and `go test ./...` is green.

## T5 — the running game builds its worlds from it

Files: `pkg/game/frontend.go` MODIFY · `pkg/game/frontend_test.go` MODIFY ·
`pkg/game/world.go` MODIFY · `pkg/game/world_test.go` MODIFY

Covers: FR-5, FR-7, SC-10, DD-7, DD-8, DD-10.

Boundary: the front-end field, the load's position in construction, and the map-open path's entry
point, difficulty and error. The unit census keeps the entry point it has (DD-10).

Scope fences: do not add a setter, flag or field for the difficulty. Do not make the map-open path
recover from a refused table.

Done when: construction fills the field before the map list is built, the map open goes through the
resolving entry point and propagates its error with the map's name, and `go test ./...` is green.

## T6 — the report says which domain each placement landed in

Files: `cmd/classdump/databin.go` MODIFY · `cmd/classdump/main.go` MODIFY

Covers: FR-8, SC-9, DD-2, DD-3.

Boundary: the per-placement line and the per-map census in the one verb that already resolves
placements.

Scope fences: read the domain from the built world, never from a second resolution. Do not import
the simulation tier and do not widen the allow-map — neither is needed to print a name.

Done when: the per-placement line carries a domain name, the census prints all three counts, they
sum to the placement count, and the gate is clean.

## Traceability

| Spec | Plan criterion | Task |
|---|---|---|
| FR-1, AC-1, P-2 | SC-1, SC-2 | T2 |
| FR-2, AC-2 | SC-2 | T2 |
| FR-3, AC-3, P-4 | SC-3 | T2 |
| FR-4, AC-4, AC-5 | SC-4, SC-5 | T3 |
| FR-5, AC-7, AC-9 | SC-7, SC-10 | T2 (AC-7), T5 |
| FR-6, AC-6, P-3 | SC-6 | T4 |
| FR-7, AC-10 | SC-8 | T2 (AC-10), T5 |
| FR-8, AC-8 | SC-9 | T6 |
| P-1 | SC-1 | T2 |
| — | SC-11 | every task |
