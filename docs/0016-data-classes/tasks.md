# Tasks — typed data classes from the graphics registries

Legend: **files** the task may change · **done when** the observable it must leave behind.
Fixtures are built with `internal/synth` — `synth.Reg`, `synth.Archive`, `synth.ALM` — never read.

## T1 — the three class types, their key tables and the node reader

**files** `pkg/data/classes.go`, `pkg/data/keys.go`, a new `pkg/data/keys_test.go`

Declare `UnitClass`, `ObjectClass` and `StructureClass`, one exported field per key of that
registry's inventory at that key's kind. Beside each, its key table and its descriptor (DD-2, DD-3):
rows `{name, kind, ptr}`, and count key, section prefix, sprite prefix, whether `[Files]` exists,
whether `Parent` is legal. Add the section-node and key-node lookup of DD-1 over `Reg.Root`.

Nothing else is exported: `Load*` is T2's, the path methods T3's.

**done when** a reflect-driven test pins each table against its struct — every exported field is one
row and every row one field, so no key can be validated but never stored or stored but never
kind-checked — the three tables' lengths are the inventory's, and the reader answers *absent*,
*present* and *wrong kind* apart on a synthetic registry.

## T2 — `own`/`eff` resolution and the three `Load*` entry points

**files** `pkg/data/load.go`, `pkg/data/doc.go` (its *will implement* is false once the loaders
exist), a new `pkg/data/inherit_test.go`, a new `pkg/data/enumerate_test.go`. Needs T1.

Stage 1 walks `0..count-1` by constructed section name, recording `own[i]` per key row and the
`ID → index` map; stage 2 fills `eff` under the two guards, copying an array when it takes one, and
the public struct is filled from `eff` afterwards (DD-2, DD-3, DD-4). `LoadUnitClasses`,
`LoadObjectClasses` and `LoadStructureClasses` return a collection holding one `[]*Class` in build
order plus the map into it, exposing `All()` — a copy of the pointer slice — and `ByID(int32)`
(FR-1, FR-2).

**done when** SC-1, SC-2, SC-3, SC-4 and SC-6 pass, each with the break its criterion names run and
reported: a zero-value guard, `""` read as a clear, an array resolved from the parent's built struct,
`0` read as no parent, and enumeration of `Root.Children`. Sprite paths are T3's and the resolved
length checks T4's; a class loads without either.

## T3 — the sprite base and the two path methods

**files** `pkg/data/sprite.go`, `pkg/data/load.go`, `pkg/data/classes.go` (the unexported `base`
field, which can only be declared with the structs), a new `pkg/data/sprite_test.go`. Needs T2.

Compute the unexported `base` once per class inside the load loop and add `SpritePath()` and
`OverlayPath()` to all three class types (DD-5). `File`'s bound and the `[Files]` entry's presence and
non-emptiness are checked there and nowhere else.

**done when** SC-7 passes over a unit, an object and a structure, one path carrying several
backslashes and mixed case, compared byte for byte; a class whose `File` is absent after inheritance
returns `""` from both rather than an error; and no exported field was added, so every exported field
is still an inventory key (FR-2).

## T4 — one fixture per malformed case, and the two that must load

**files** a new `pkg/data/validate_test.go`, and `pkg/data/load.go`, `pkg/data/keys.go`,
`pkg/data/sprite.go` for the checks those fixtures find missing. Needs T3.

Build the fixture set SC-5 enumerates, counting the `Parent` family as four and the `[Files]` faults
as two, and write whatever check a fixture reaches that the code does not yet make — the checks on
resolved values among them (DD-8). Errors take the fixed `<section>: <key>: <what>` prefix, the key
dropped where the fault has none.

**done when** SC-5 passes: each fixture yields a nil collection and an error carrying its section and
key, none panics, and the two clean pins — a class with no `ShootOffset`, one with a 4-element
`Sound` — load as values. No rejection the contract does not name is added (FR-1).

## T5 — the inventory round-trip

**files** a new `pkg/data/inventory_test.go`. Needs T3.

The pin SC-8 asks for: one class per registry carrying **every** key of its inventory at a distinct
sentinel, compared against a written-out expected struct, so a dropped key shows as a zero and a
field with no key as a surplus. Beside it the three cases that are values and not errors — no
`DescText`, a sprite entry naming nothing that ships, an unknown key (FR-1).

**done when** SC-8 passes and one break is run and reported: repointing a single key table row at
another field of the same kind must fail this test — T1's bijection cannot see that one, which is why
the round-trip is written out rather than generated.

## T6 — `cmd/classdump`, its DAG row and the architecture rows

**files** `cmd/classdump/main.go`, `internal/archtest/dag.go`, `docs/ARCHITECTURE.md`, a new
`cmd/classdump/main_test.go`. Needs T3.

All four move in one commit (DD-7): `internal/archtest` is fail-closed, so the package may not exist
for even one commit without its `allow` row, and `ARCHITECTURE.md`'s tier row and dependency row are
bound to that map. The row admits what the tool imports — the gate, not this entry, is what says
whether it is complete.

FR-3's print half: read a `graphics.res` through `pkg/formats/res`, parse the three registries, load
them, print each collection. Usage and exit-code conventions follow `cmd/regtool`.

**done when** `go test ./internal/archtest/` is green with the package present, and a test over a
synthetic archive prints all three collections and exits zero. `-sweep` is T9's.

## T7 — the type-6 classifier

**files** `cmd/classdump/classify.go`, a new `cmd/classdump/classify_test.go`. Needs T6 — the package
may not exist before its DAG row does.

The pure function of DD-6: the `Flags` and `DefID` conditions evaluated independently into `direct`,
`npc`, `def` and `both`, and `ClassID` widened `int16 → int32` so a negative key stays negative. It
resolves nothing, reads nothing and prints nothing.

**done when** SC-9's classifier half passes: a table over all four buckets, over `DefID` at both `0`
and `0xcdcdcdcd`, and over a `ClassID` written `0x8001` that must read `-32767`.

## T8 — placed records in the synthetic map builder

**files** `internal/synth/synth.go`, `internal/synth/synth_test.go`

`synth.ALM` writes an all-zero type-3 grid and no type-4 or type-6 payload at all, so no test can
build a map with a placement in it and T9 has nothing to sweep. Add optional fields carrying the
type-3 codes and the type-4 and type-6 records, with the type-0 counts following them; the defaults
stay today's payloads, so no existing caller changes.

**done when** a map built with placements of all three kinds decodes back through `pkg/formats/alm`
to the records given — `Overlay`, `Objects`, and `Units` with their `ClassID`, `Flags` and `DefID` —
and `go test ./...` is green with no existing fixture edited.

## T9 — `-sweep`

**files** `cmd/classdump/main.go`, `cmd/classdump/sweep.go`, a new `cmd/classdump/sweep_test.go`.
Needs T7 and T8.

Walk `<dir>` for loose `.alm` files and for `.alm` entries inside `.res` archives — `cmd/regtool`'s
sweep is the walk precedent — resolve each source's three placement kinds against the three
collections, and print DD-6's fixed tokens: one line per source, then a totals block, then one line
per failing reference naming source, kind, index and value. The diverted buckets are counted, as is
how many of their `ClassID`s would have resolved anyway. Exit non-zero on an unresolved type-4 or
`direct` type-6 reference and on nothing else (FR-3).

**done when** SC-9's sweep half passes: a sweep whose only non-`direct` records are diverted exits
zero with them counted, one unresolvable `direct` record exits non-zero, and a type-3 code past the
registry is counted and printed without moving the exit code.

## T10 — the per-key absent-everywhere defaults of `objects.reg`

**files** `pkg/data/keys.go`, `pkg/data/load.go`, a new `pkg/data/defaults_test.go`, and
`pkg/data/inherit_test.go` for the one assertion whose expected value changes.

Add DD-9's `{key, value, noInherit}` table beside the objects key table — `0` for `InMapEditor`, `-1`
for the eleven keys the engine loads — resolve it once per load into the two arrays over row indices,
and have stage 3 write the default where `eff[k]` is nil instead of leaving the Go zero. `units.reg`
and `structures.reg` get no rows. `noInherit` is declared here and read by T11.

**done when** SC-12 passes with its break run and reported — falling through to the Go zero reads `0`,
`0`, `0` — a unit class in the same shape still reads `0`, and every default row is pinned to a
`kindInt` row of the objects key table. `TestParentZeroIsARealReference`'s "an absent `Parent`
inherits nothing" assertion reads `-1` rather than `0`: the same contract, at the value it now names.

## T11 — `File` does not inherit in `objects.reg`

**files** `pkg/data/keys.go`, `pkg/data/load.go`, `pkg/data/defaults_test.go`. Needs T10.

Mark the objects `File` row `noInherit` and have stage 2's scalar arm skip the parent for it (DD-9),
so an object class whose own section omits `File` resolves `-1` and no sprite. `units.reg`'s `File` is
untouched.

**done when** SC-13 passes: an object child with a `Parent` that sets `File` resolves `File` to `-1`
with both paths empty, where a unit child in the same shape inherits its parent's `File` and path. No
output on shipped data moves, and the fixture is the only witness there is — the evidence says that
rather than implying the install demonstrated it.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1 | DD-1, DD-2, DD-3 |
| T2 | FR-1, FR-2 | DD-2, DD-3, DD-4 |
| T3 | FR-2 | DD-5 |
| T4 | FR-1 | DD-8 |
| T5 | FR-1 | DD-2 |
| T6 | FR-3 | DD-7 |
| T7 | FR-3 | DD-6 |
| T8 | FR-3 | DD-6 |
| T9 | FR-3 | DD-6 |
| T10 | FR-1 | DD-9 |
| T11 | FR-1 | DD-9 |
