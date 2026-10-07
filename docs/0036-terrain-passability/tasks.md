# Tasks — the pass over the cells, the fixtures it invalidates, and the wiring

Legend: **files** what an entry may change — a permission, not a prediction · **fences** what it
must not do · **done when** the observable it leaves behind. All six are *implementation* entries in
ascending order, each depending only on entries before it. The fixtures move **before** the loader
does, because after it a fixture that has stopped moving is indistinguishable from a routing defect.

## T1 — the plane, the census beside it, and the six mutants

**files** ADD `pkg/mapload/passability.go`, `pkg/mapload/passability_test.go`

DD-1 (both functions), DD-2, DD-3, DD-4, DD-5, DD-6 — FR-1, FR-2, FR-3, FR-4, FR-5, FR-8.

**fences** nothing outside `pkg/mapload` is opened and `fromalm.go` is not touched, so this entry
changes no world. No registry, altitude, unit, object or trigger section is read; no cost is
computed; no terrain class is named beyond the single compare DD-3 keeps. The blend table is
transcribed from `spec.md`'s own table and from no other source and carries its claim id in the
source, and the two entry points share one classifier rather than each walking its own rule.

**done when** AC-1 to AC-6 hold and AC-10's derivation half with them, the census reports FR-8's
eleven numbers over a map whose every arm is known, and SC-1 to SC-4 hold — the 56-cell walk
asserted with its expected levels written by hand, and every byte compared whole. Then SC-8's six mutants are each applied to production code, the whole tree run with
the failing tests named, reverted, and the tree confirmed byte-identical; a survivor is reported as
a criterion that does not discriminate its own decision.

## T2 — the fixtures that walk a map-built world

**files** MODIFY `pkg/game/world_test.go` and the tests there that advance a fixture world;
`pkg/mapload/fromalm_test.go`

DD-7 — FR-4.

**fences** no production file is opened, and no behaviour changes: this entry is green before and
after for the same reasons. The unit anchors keep their fractional parts, the cell each names is
still written out by hand rather than computed from them, the altitude relief keeps its per-column
gradient, and no fixture gains a grid, a tile word or an overlay byte it did not have. The unit one
fixture deliberately places outside its extent stays outside it.

**done when** both fixture maps are at least 64 on each axis, every placed unit and every
hand-written target in the tests that advance one stands inside the interior, and the tree is green.

## T3 — the loader passes it

**files** MODIFY `pkg/mapload/fromalm.go`, `pkg/mapload/fromalm_test.go`; MODIFY any test outside
`pkg/mapload` that the derived grid turns red

DD-1 (the loader), DD-6 — FR-1, FR-6.

**fences** the id convention, the cell conversion, the seed, the spawn health and the named
canonical mode are untouched, and the signature keeps its single return and its discarded error.
`sim.NewWorld` gains no parameter and no default, and nothing under `pkg/sim` is edited: a red test
there is a finding, not a thing to fix here. The seed test's hand-built comparison world is given a
grid built in the test from the contract's words — never one taken from the derivation, which would
turn a pin into an agreement.

**done when** AC-8 and AC-10 hold and SC-6 with them — the three build paths read through their byte
forms, the map differing in every section but the three that feed a grid shown to derive the same
plane, and each degenerate shape shown to yield a world a caller can use. Any non-vacuity guard this
entry passes through is shown to observe a moved **cell**, not merely a moved field.

## T4 — the form, the digest, and the round trip

**files** ADD `pkg/mapload/gridform_test.go`

DD-1 — FR-6, FR-7.

**fences** no production file is opened. The expectation is written from the contract and never from
a call to the derivation; the digest is taken from no run of this tree; and the round trip is
compared byte for byte, a digest comparison alone being what this entry exists to distrust.

**done when** AC-7 holds and SC-5 with it: the interior compared against a literal table and the
border against a predicate written from FR-4's words, the digest recomputed outside this tree from
those same bytes and cross-checked through a second FNV written in the test, and the round trip —
taken after a tick in which a unit routed — byte-for-byte the first form.

## T5 — a unit that will not cross water

**files** ADD `pkg/mapload/routing_test.go`

FR-6.

**fences** no production file is opened and no routing rule is restated: the test drives `sim.Step`
and reads cells, and asserts nothing about which route was chosen. The channel is placed inside the
interior, so what turns the unit is the water rather than the border.

**done when** AC-9 holds and SC-7 with it, in **both** routing modes: the unit arrives through the
gap having never stood on a cell whose ground bit is set, and with the gap closed it holds its cell
and its target and gives up on the sixteenth tick.

## T6 — the instrument, and the edge it needs

**files** MODIFY `cmd/almtool/main.go`, `internal/archtest/dag.go` and its test,
`docs/ARCHITECTURE.md`, `AGENTS.md`

DD-8 — FR-8.

**fences** the existing subcommands keep their output and their arguments; no path is embedded, no
asset is written, nothing here is reachable from `go test`, and the counting is the census T1 built
rather than a second reading of the arms. Exactly one edge is added to the allow-map.

**done when** the subcommand prints FR-8's counts for a `.alm` path given on the command line, the
import check passes with the new edge and fails without it, and all **three** copies of the table —
the allow-map, `docs/ARCHITECTURE.md` and `AGENTS.md` — name the same set.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1, FR-2, FR-3, FR-4, FR-5, FR-8, AC-1, AC-2, AC-3, AC-4, AC-5, AC-6 | DD-1, DD-2, DD-3, DD-4, DD-5, DD-6, SC-1, SC-2, SC-3, SC-4, SC-8 |
| T2 | FR-4 | DD-7 |
| T3 | FR-1, FR-6, AC-8, AC-10 | DD-1, DD-6, SC-6 |
| T4 | FR-6, FR-7, AC-7 | DD-1, SC-5 |
| T5 | FR-6, AC-9 | SC-7 |
| T6 | FR-8, AC-11 | DD-8, SC-9 |
