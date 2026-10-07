# Tasks — the field, the form, the two orders, and the front-end that reads them

Legend: **files** what an entry may change — a permission, not a prediction · **fences** what it
must not do · **done when** the observable it leaves behind. All nine are *implementation* entries,
in ascending order, each depending only on entries before it. SC-9's five mutants are distributed
among them, applied to production code, measured over the whole tree and reverted by the entry that
owns one; a kill is claimed only where it was run.

## T1 — the two numbers, and the three states they answer

**files** MODIFY `pkg/sim/world.go`, `pkg/sim/nostate_test.go`; ADD `pkg/sim/health_test.go`

DD-1, DD-3 (constructor half) — FR-1.

**fences** no file outside `pkg/sim` is opened and no encoder, decoder, digest or step is touched, so
this entry moves no byte form. The stall rule, the duplicate-id refusal and the targetless-residue
zeroing keep their bodies; `Entity` stays a type of integers and bools with no slice, pointer or
method that allocates. Nothing here reads a class id or a route.

**done when** AC-1 holds and SC-1 with it, the eight pairs each a case of its own with its expected
state written out by hand; the field-set pin lists the two new fields; and a world built with a
target on a unit that is not alive comes back with all three of target, stall and route cleared.

## T2 — version 5, and every pin derived afresh

**files** MODIFY `pkg/sim/binary.go`, `pkg/sim/binary_test.go`, `pkg/sim/hash_test.go`,
`pkg/sim/relaxation_test.go`, `pkg/sim/routeform_test.go`, `pkg/sim/malformed_test.go`

DD-2, DD-3 (decoder half) — FR-2.

**fences** the header, the grid section and the route section keep their layout, their order and
their length rules; the declared entity count is still checked rather than trusted, and the route
section still consumes the remainder exactly. No migration path, no compatibility reader, no field
normalised on the way in. No production file outside `binary.go` is edited.

**done when** AC-6 and AC-7 hold and SC-4 with them: the offset table is still a partition of the
whole form, `pinBytes` is re-transcribed by hand at the new length, and `pinDigest`, `rlxTick1Digest`,
`hybTick1Digest` and `rtfDigest` are each recomputed outside this tree from their own hand-written
bytes and cross-checked through the in-tree second FNV — none carried over by arithmetic. A whole
well-formed version-4 world is refused beside the 256-value first-byte sweep. Then SC-9's
version-left-at-4 mutant is applied, the whole tree run with the failing tests named, reverted, and
the tree confirmed byte-identical.

## T3 — a kind on the command, and the two arms behind it

**files** MODIFY `pkg/sim/step.go`; ADD `pkg/sim/damage_test.go`

DD-4, DD-5 — FR-3.

**fences** the move-to arm keeps its body and its zero-value kind, so no existing command literal,
schedule or queue is edited anywhere in the tree. The three phases keep their order and the tick is
still incremented exactly once; `clearOrder` is called rather than reimplemented, and neither the
stall rule nor the give-up threshold moves. No occupancy, route or move-loop change belongs here.

**done when** AC-2, AC-3 and AC-4 hold and SC-2 and SC-3 with them, the ladder asserted at each of
its eleven steps and every no-op compared by digest against a step carrying no command. Then SC-9's
clamped-damage mutant and its widened-death-test mutant are each applied, the whole tree run with
the failing tests named, reverted, and byte-identity confirmed.

## T4 — who is counted, who advances, and the replay that proves it

**files** MODIFY `pkg/sim/route.go`, `pkg/sim/step.go`; ADD `pkg/sim/downed_test.go`

DD-6 — FR-4, FR-10.

**fences** `moved`, `enterable`, `terrainOpen`, the window, the budget and both searches keep their
bodies; the count plane stays a count and gains no second plane beside it. The resolution order stays
ascending id and the four per-tick outcomes stay four. Nothing here reads the front-end or a clock.

**done when** AC-5 and AC-8 hold and SC-5 and SC-6 with them — both cells tested entered and
refused, the immobile pair driven several ticks, and a corpus walked to quiescence under a stream
mixing all three kinds with the decoded world compared at every later tick. Then SC-9's
seed-skips-downed mutant is applied, the whole tree run with the failing tests named, reverted, and
byte-identity confirmed.

## T5 — what a map's units are born with

**files** MODIFY `pkg/mapload/fromalm.go`, `pkg/mapload/fromalm_test.go`

DD-11 — FR-5.

**fences** the id convention, the cell conversion, the seed, the mode and the absent grid are
untouched, and no registry or class is consulted. `NewWorld` gains no default and no parameter, so
no world built any other way changes.

**done when** AC-9 holds: every unit of a map-built world carries the pair, and a hand-built world
still carries the pair it was given.

## T6 — the state crosses the seam

**files** MODIFY `pkg/ui/overlay.go`, `pkg/game/world.go`, `pkg/game/world_test.go`;
ADD `pkg/ui/life_test.go`

DD-7 — FR-6, FR-10.

**fences** `pkg/ui` names no simulation type and derives no life state of its own; the push stays a
read through the copy-handing entity read, keeps its slice-order contract and writes nothing back.
Facing, frame selection and the art lookup keep their bodies, and no drawing changes in this entry.

**done when** AC-10 holds and SC-7 with it, the snapshot compared entry by entry over an alive, a
downed and a dead unit, and the world's digest asserted unmoved across a push.

## T7 — one filter, and a corpse that cannot be picked

**files** MODIFY `pkg/ui/command.go`, `pkg/ui/overlay.go`, `pkg/ui/command_test.go`;
ADD `pkg/ui/deadselect_test.go`

DD-8 — FR-7, FR-10.

**fences** nothing prunes, sorts or otherwise writes the stored selection outside the tap and box
branches; the four outcomes stay four and the ascending order stays a property of the walk rather
than of a sort. The marquee geometry, the slop rule and the modifier latch are untouched.

**done when** AC-11's selection half holds and SC-8's with it, the set compared whole before and
after, and a dead id shown marked by nothing, ordered nothing, and still present. Then SC-9's
marks-only-filter mutant is applied, the whole tree run with the failing tests named, reverted, and
byte-identity confirmed.

## T8 — two keys, and the seam they reach the world through

**files** MODIFY `pkg/ui/app.go`, `pkg/ui/flow.go`, `pkg/ui/flow_test.go`, `pkg/game/world.go`,
`pkg/game/frontend.go`, `pkg/game/frontend_test.go`; ADD `pkg/ui/affect_test.go`

DD-9 — FR-8.

**fences** the two keys are read on the map arm alone and nowhere else; the cadence keys, the tick
call and the order path keep their bodies and their call counts, and the advance stays exactly one
per map-screen tick. Nothing here changes the selection, and no damage amount crosses the seam.

**done when** AC-11's K and L half holds, one command per marked unit in ascending id, the selection
unchanged by either key, and a map screen with no world shown to issue nothing.

## T9 — the bar over the living

**files** MODIFY `pkg/render/terrain/overlay.go`, `pkg/ui/overlay.go`,
`pkg/render/terrain/overlay_test.go`; ADD `pkg/ui/healthbar_test.go`

DD-10 — FR-9.

**fences** the existing glyph builders, colours, pass order and the selection rim are untouched, and
the bar's passes are appended without moving any pass already in the slice. The geometry is integer
throughout and allocates nothing per frame beyond its own two slices; no drawn unit is culled or
skipped by this entry that was drawn before it.

**done when** AC-12 holds and SC-8's bar half with it, the five pairs each a case of its own and the
relief offset compared as a point against the mark on the same cell.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1, AC-1 | DD-1, DD-3, SC-1 |
| T2 | FR-2, AC-6, AC-7 | DD-2, DD-3, SC-4, SC-9 |
| T3 | FR-3, AC-2, AC-3, AC-4 | DD-4, DD-5, SC-2, SC-3, SC-9 |
| T4 | FR-4, FR-10, AC-5, AC-8 | DD-6, SC-5, SC-6, SC-9 |
| T5 | FR-5, AC-9 | DD-11, SC-7 |
| T6 | FR-6, FR-10, AC-10 | DD-7, SC-7 |
| T7 | FR-7, AC-11 | DD-8, SC-8, SC-9 |
| T8 | FR-8, AC-11 | DD-9, SC-8 |
| T9 | FR-9, AC-12 | DD-10, SC-8 |
