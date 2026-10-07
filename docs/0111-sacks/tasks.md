# Tasks — 0111-sacks

**Reading key.** `FR-n` → `spec.md` §The contract. `AC-n`, `P-n` → `spec.md` §Acceptance,
§Derived properties. `DD-n` → `plan.md` §Design decisions. `R-n` → `plan.md` §Risks. `SC-n` →
`plan.md` §Success criteria.

Tasks are ordered, and each leaves `go build ./...`, `go vet ./...` and `go test ./...` green on
its own.

---

## T1 — the band carries a fourth stream *(implementation)*

**Boundary.** The render tier learns that a sack-shaped drawable exists and where it goes in the
order; nothing supplies one. Stops before any viewer state.

**Files**

- `pkg/render/terrain/structures.go` — `MODIFY`
- `pkg/render/terrain/sack.go` — `ADD`
- `pkg/render/terrain/sack_test.go` — `ADD`
- `pkg/render/terrain/structures_test.go` — `MODIFY` (three existing call sites)
- `pkg/ui/statics.go` — `MODIFY` (the one existing call site, supplying no sacks)

**Covers** FR-5, FR-6, FR-7 · DD-2, DD-3 · R-3 · SC-3, SC-4 (AC-5, AC-6, AC-7, P-2, P-4)

**Fences.** Do not add viewer state, a setter, a layer builder or a draw arm. Do not touch
`PlaneOrder`, the decoration prefix's own pass, or the existing kinds' values. Do not move any
frozen expectation in `structures_test.go`. Do not read anything from `pkg/sim`.

**Done when:** the placement and the extended merge are tested in `pkg/render/terrain`, including
a fixture whose sacks and entities both survive into the tail drain; an empty sack stream
reproduces the existing frozen orders exactly; and the whole tree is green with no sack reaching
the merge.

---

## T2 — the window draws what it is given *(implementation)*

**Boundary.** The viewer can hold a sheet and a list of sacks, place them, exclude the ones it
cannot draw, and paint them in the band. Nothing fills either yet, so no sack appears on screen.

**Files**

- `pkg/ui/viewer.go` — `MODIFY`
- `pkg/ui/statics.go` — `MODIFY`
- a test file in `pkg/ui` — `ADD`

**Covers** FR-3 (the gating half), FR-8, FR-9, FR-10 · DD-4 (the window half), DD-7, DD-8 ·
SC-5, SC-6, SC-7 (AC-8, AC-9, AC-10, P-3)

**Fences.** Do not load a sheet, name an archive path, or read a world — the setters' callers are
T3's. Do not add a hit test, a cursor arm, a selection path or a readout row for a sack. Do not
touch the shadow pass or the flat-structure pass.

**Done when:** a viewer given frames and a sack list places and paints them through the band's own
cull, tint and light row; an ungridded cell, an out-of-range index and an empty frame set each
draw nothing and raise nothing; and the drawn output with no sacks is unchanged from T1's.

---

## T3 — the world's sacks reach the window *(implementation)*

**Boundary.** The sheet is loaded once at start-up, handed to every viewer the game itself opens,
and the world's own sack list is pushed every refresh. This is the task the player sees.

**Files**

- `pkg/game/sacks.go` — `ADD`
- `pkg/game/world.go` — `MODIFY`
- `pkg/game/frontend.go` — `MODIFY`
- a test file in `pkg/game` — `ADD`

**Covers** FR-1, FR-2, FR-3, FR-4, FR-11 · DD-1, DD-4 (the game half), DD-5, DD-6 · R-1, R-2, R-4,
R-5 · SC-1, SC-2 (AC-1, AC-2, AC-3, AC-4, AC-11, AC-12, P-1, P-5)

**Fences.** Do not change `pkg/sim`, the serialized world form or its version. Do not change the
signature of any `mapWorld` constructor, of the map-viewer constructor, or of anything
`cmd/mapview` calls. Do not read an item code's class, index or value anywhere.

**Done when:** a front end built over a synthetic archive loads the sheet's frames in order,
survives an absent, an undecodable and a palette-less one without an error, installs them on both
opener paths, and a refresh pushes one record per world sack entry, in the world's order, twice
alike.

---

## Traceability

| Requirement | Plan criterion | Task |
|---|---|---|
| FR-1 | SC-1 | T3 |
| FR-2 | SC-1 | T3 |
| FR-3 | SC-2, SC-8 | T2, T3 |
| FR-4 | SC-2 | T3 |
| FR-5 | SC-3 | T1 |
| FR-6 | SC-3 | T1 |
| FR-7 | SC-4 | T1 |
| FR-8 | SC-5 | T2 |
| FR-9 | SC-6 | T2 |
| FR-10 | SC-7 | T2 |
| FR-11 | SC-9 | T3 |
| DD-1 | SC-2 | T3 |
| DD-2 | SC-4 | T1 |
| DD-3 | SC-3 | T1 |
| DD-4 | SC-1, SC-2 | T2, T3 |
| DD-5 | SC-1 | T3 |
| DD-6 | SC-1 | T3 |
| DD-7 | SC-7 | T2 |
| DD-8 | SC-6 | T2 |

SC-8 and SC-9 are developer runs against a lawful install and carry no task: the evidence stage
runs them.
