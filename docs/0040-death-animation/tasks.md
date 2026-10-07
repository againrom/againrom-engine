# Tasks — a length, a selection, a link, a dispatch, and what a death does not move

Legend: **files** what the entry may change — a permission, not a prediction · **fences** what
it must not do · **done when** the observable it leaves behind. Every entry is an
implementation entry and they land in ascending order, each depending only on those before it.
T1 through T3 add reachable-but-unreached surface on purpose: nothing draws a death until T4,
so each of the three lands with its own witnesses and none lands red. SC-9's two mutants both
name the phase arithmetic, so both are applied by the entry that WRITES it, measured over the
whole tree, and reverted there; a mutant applied to a line an entry has not written yet kills
nothing, and a kill is claimed only where it was run.

## T1 — the length the sheet arithmetic already counted with

**files** MODIFY `pkg/data/anim.go`, `pkg/data/anim_test.go`, `pkg/render/terrain/unitanim.go`,
`pkg/game/units.go`; ADD `pkg/game/units_anim_test.go`

DD-1 — FR-1.

**fences** no base, stride, track, gate or total changes value, and no existing field is
renamed or re-derived; the clamp is the one already there and is not reapplied, widened or
copied. No selector is opened. The render tier's mirror gains a field and no derivation, and
the loader's copy stays a copy — nothing in it computes.

**done when** SC-1 holds on both sides of the mirror: the derived length over hand-written
classes including absent phases, the block-sum identity against the base of the block after
it, and the loader's copy asserted to carry the same value the data tier derived. Every
descriptor assertion already in the tree still holds unchanged.

## T2 — the death selection, and the two ticks a dying frame is held for

**files** MODIFY `pkg/render/terrain/unitanim.go`; ADD
`pkg/render/terrain/deathanim_test.go`

DD-3, DD-4 — FR-3.

**fences** the live selection is not touched: its signature, its arms, its fallback chain and
its answer at every input are unchanged, and the direction rule and the step reduction are
SHARED rather than copied. Nothing here reads a clock, a world, a life state or a sheet, and
nothing allocates. No caller is added — this entry leaves the function unreached.

**done when** SC-3, SC-4 and SC-5 hold, each index checked against this file's own arithmetic
rather than against the function's, and every answer asserted to lie inside its own direction's
slot. Then SC-9's two mutants are applied one at a time to the phase expression, the whole tree
run with the failing tests named, reverted, and byte identity confirmed.

## T3 — the class whose sheet carries the body

**files** MODIFY `pkg/render/terrain/units.go`, `pkg/game/units.go`; ADD
`pkg/game/corpse_test.go`

DD-2 — FR-2.

**fences** the resolution happens once, at load, and no lookup is added to any draw path; the
registry key is read in the tier that already reads registries and its value does not cross
into the render tier. The link is filled for every loaded class, frameless included, and no
class is dropped, skipped or errored on account of it. Nothing in the bundle's existing shape
changes meaning, and no selector is opened.

**done when** SC-2 holds over a hand-built bundle — the sibling, the self-namer, the dangling
namer and the two-hop namer — with the two-hop case asserted to stop at the class the key
names. The link is read back through the bundle a loader would hand over, not through the
helper that fills it.

## T4 — the tick a unit was first seen dead, and what the push does with it

**files** MODIFY `pkg/game/world.go`, `pkg/game/world_test.go`; ADD
`pkg/game/death_test.go`

DD-5, DD-6, DD-7, DD-8 — FR-4, FR-5, FR-6, FR-7.

**fences** `pkg/sim` is not opened and neither is `pkg/ui`: no seam field is added, renamed or
given a second meaning, and no glyph, placement, cull or texture path is touched. The new
memory is looked up by key and never ranged, written once per entity and never rewritten,
cleared or pruned. The step memory, the facing memory and the scene clock keep their existing
writers and their existing placement. The live path's inputs — the moving flag, the octant and
the effective tick — are unchanged for an entity that is alive. The one edit
permitted in the older test file is the seam fixture's memory list, which a
build that writes a memory the fixture does not hold would panic on; nothing
else there is corrected in passing.

**done when** SC-6 and SC-7 hold, the frames read off the pushed entities rather than off the
memory behind them, the corpse's art asserted to be the CORPSE class's and not the unit's, and
a snapshot built twice with no advance between seen to answer identically. The three failing
shapes are driven as worlds, not as unit calls into the selection.

## T5 — what a death does not move

**files** MODIFY `pkg/game/drawn_invariance_test.go`

FR-8 — SC-8, P-1, P-2.

**fences** no production file is opened. The headless side of every comparison is assembled
from the command stream this entry asserts and never from a second run of the driver under
test, and no test here reads a game install or a wall clock.

**done when** SC-8 holds over a stream carrying both a kill and a damage blow, the digests
compared at every tick index rather than at the end. This entry owns no mutant: it exercises no
production line an entry above does not already carry.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1, AC-1 | DD-1, SC-1 |
| T2 | FR-3, AC-3, AC-4, AC-5, P-4 | DD-3, DD-4, SC-3, SC-4, SC-5, SC-9 |
| T3 | FR-2, AC-2 | DD-2, SC-2 |
| T4 | FR-4, FR-5, FR-6, FR-7, AC-6, AC-7, AC-8, AC-9, P-2, P-3 | DD-5, DD-6, DD-7, DD-8, SC-6, SC-7 |
| T5 | FR-8, AC-10, P-1 | SC-8 |
