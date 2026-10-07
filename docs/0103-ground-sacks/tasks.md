# Tasks — 0103

Three tasks, one tier each, bottom up. T1 decodes the section and can be checked against the shipped
corpus on its own. T2 gives the simulation somewhere to put it and writes no placement. T3 adds the
two consumers, so both are witnessed through a real map and a real script.

| Task | FRs | ACs | Ps | Ds |
|---|---|---|---|---|
| T1 | FR-1, FR-2, FR-3, FR-4, FR-5, FR-6, FR-7, FR-8 | AC-1, AC-2, AC-3 | P-1..P-7 | D-4 |
| T2 | FR-9, FR-10, FR-11, FR-12, FR-13, FR-14, FR-15 | AC-4, AC-5, AC-6 | P-8..P-13 | D-1, D-2, D-8 |
| T3 | FR-16, FR-17, FR-18, FR-19, FR-20, FR-21, FR-22, FR-23 | AC-7, AC-8, SC-1, SC-2 | P-14..P-17 | D-3, D-5, D-6, D-7 |

AC-1 and AC-8 are measurements taken in the verification stage against an installed root; a task
neither asserts them nor reads an install.

## T1 — the type-8 grammar and the dump

**Files:** new `pkg/formats/alm/loot.go` and `loot_test.go`, `alm.go`, `alm_test.go`, new
`cmd/almtool/loot.go`, `cmd/almtool/main.go`.

`loot.go` mirrors `script.go` in this package: `Loot`, `LootRecord`, `LootElement`, the methods P-4
names, `ItemClass`/`ItemIndex` beside `TileIndex`, and `func (m *Map) Loot() (Loot, error)` reading
`m.LootSection.Body`, `m.Meta.Word2C` and `m.FormatVersion` and storing nothing.

In `alm.go`, rename the type and field `Markers` to `LootSection` and replace its doc comment: it is
the map's authored loot, not a marker tree, and the reading it currently states is retracted. Leave
`TileMarkers` untouched.

`almtool loot FILE` is one `case` in the existing dispatch plus one usage line.

**Landed tests that change and are meant to:** the three `Markers.Body` assertions in `alm_test.go`
— name only, never expectation.

Tests build the payload as bytes: both head widths, an absent section, an empty one, a record with
zero elements, one that overruns the payload, a payload with one byte left over, and a class-14 code
beside a class-2 one so the index widening has somewhere to fail. AC-3 is a round trip through
`OpenDocument`/`Write` with `Loot()` called in between.

## T2 — the sack, the constructor and the byte form

**Files:** new `pkg/sim/sack.go` and `sackform_test.go`, `world.go`, `binary.go`, `binary_test.go`,
and the digest pins named below.

`sack.go` holds `Sack{X, Y int32; Gold uint32; Items []uint16}`, `sackFault` (the per-sack predicate
the constructor and the decoder both call), `normaliseSacks` (merge, sort, refuse), `sackAt` and
`Sacks()`. `world.go` gains the field, `newWorld` gains the parameter, the four existing
constructors pass `nil`, and `NewLootWorld` is added beside `NewRelatedWorld`.

`binary.go`: `formatVersion` becomes **22**; the section is encoded and decoded between the groups
and the script; the offset-table comment gains its rows and the version comment a paragraph in the
voice of the ones above it.

**Landed tests that go red and are meant to:** every literal digest and every byte-form offset or
length pin in `pkg/sim/*_test.go` and `pkg/mapload/{fromalm,gridform}_test.go`. The version byte is
at offset 0, so they all move. **Re-pin from a run and record old -> new in the commit body.** A test
red for any other reason is a stop-and-report, not an edit.

## T3 — the sack query and the placement

**Files:** `pkg/sim/script.go`, new `pkg/sim/sackcheck_test.go`, `pkg/sim/script_test.go`,
`pkg/mapload/fromalm.go`, new `pkg/mapload/loot_test.go`, `pkg/mapload/script_test.go`.

Add `ScriptCheckSackAt int32 = 14` to the const block and to `scriptCheckSupported`. **The arm goes
in the first switch of `runCheck`, beside `ScriptCheckVariable`** — read P-14 before writing it; the
second switch is unreachable for a node that names no unit, and a check-14 node names none. Write no
second list of unsupported opcodes.

In `fromalm.go`, add `sacksFrom` and switch `FromALMWith` to `sim.NewLootWorld`. A decode error is
no sacks and no load failure; a stock record and an out-of-bounds cell each place nothing.

**Landed tests that go red and are meant to:** the unimplemented-check reports in
`pkg/sim/script_test.go` and `pkg/mapload/script_test.go`, which name 14 today.
**`TestTheTenthMissionIsDrivenToAWin` must not change state** — mission 10 carries no check-14 node,
so if it moves, stop and report.

Tests: a compiled node over a world with and without a sack, a trigger conditioned on one, an
out-of-bounds cell, a coordinate above 255 to witness the byte truncation, and a map fixture
carrying both record kinds asserting exactly the ground cells are placed.
