# Tasks — 0056-unit-speed

Legend: each entry names what it builds, the contract ids it answers, and how it is
witnessed. Trailer `SDD-Task: 0056-unit-speed/T<n>`.

## T1 — the rate law, as arithmetic with no world

Add `pkg/sim/rate.go`: `rateOf(d Domain, speed int32, costSrc, costDst, hSrc, hDst uint8)
int32` and `transitOf(v int32, diagonal bool) int32`, with `diagonalStep` beside them and
one named constant per term of **FR-1** and **FR-2** — the multiplier, the slope clamp and
shift, the zero-mean substitution, the rate floor and ceiling, the sub-cell grid, and the
diagonal's numerator and denominator.

Carry **DD-1**: no world in the signature, the planes' bytes as `uint8`. Carry **DD-2**:
the diagonal is `(v·707)/1000`. Carry **DD-3**: a step truncating to zero is taken as one.
Do the intermediate arithmetic wide enough that a customised speed cannot wrap.

Tests, in `pkg/sim`, using no world: the worked example; the three domains at one speed
over three cost means and a slope; the clamps, the byte wrap and the zero-mean arm; the
diagonal against `math.Float64frombits(0x3FE69FBE76C8B439)` for every `v` in `0…999`; the
straight transit enumerated over `v ∈ [1,63]` for its 27 distinct values and its 12-wide
class at 5; and the zero-step case. A test may name a float — the determinism scan reads
production sources only — and must not import outside the standard library.

Witnesses AC-1, AC-2, AC-3, AC-4 · SC-1, SC-2, SC-3.

## T2 — the three fields and the byte form

`Entity.Speed int32`, `.Transit uint16`, `.TransitTotal uint16` in `pkg/sim/world.go`, with
the doc each field's meaning needs and the `rated` predicate of **DD-5** beside them. Byte
form to version **7**, record 43 bytes, the three appended at the record's tail in that
order (**DD-6**).

**FR-4**'s refusals in the constructor AND in the decoder, so neither can produce what the
other will not read: a `TransitTotal` above the law's maximum of 256; an owed count at or
past its own transit's length; a nonzero either on a unit that is not alive — which the
constructor normalises, as it already normalises such a unit's order, and the decoder
refuses. Update the field-set pin in `nostate_test.go`.

Tests: a mid-transit world round-trips and re-hashes; each refused shape fails by name;
each of the three fields moves the digest on its own; a world naming none of them is
unchanged in behaviour.

Witnesses AC-6 · SC-6.

## T3 — the transit gate

In `pkg/sim/step.go`, per **DD-4**: split the combined liveness/target test, put
`if e.Transit > 0 { e.Transit--; continue }` after the liveness test and **before** the
target test, and freeze the rate at the step itself — from the two cells the step joins,
with `rateOf` passed zero cost and height bytes, writing `TransitTotal = t` and
`Transit = t-1`. Only a **rated** mover writes them (**FR-3**, **FR-5**'s other half is
T4). Clear both where a felled unit's order is cleared.

Say in the doc comment what the zeros are: absent planes, not omitted terms, and the law's
own value at them. Say that a mover owing a transit increments no stall count, because it
makes no attempt.

Tests: one cell per transit and no search between; two speeds arriving in the predicted
ratio, ticks-to-arrive printed; an unrated mover at a cell a tick; an order arriving
mid-transit taking effect at the transit's end; a mover felled mid-transit; the whole
pre-existing simulation suite unchanged (**FR-7**).

Witnesses AC-5 · SC-4, SC-5, SC-9.

## T4 — a placement's speed

In `pkg/mapload/fromalm.go`, per **DD-8**: take `Speed` off the same resolution the health
and the domain already come from, in the same statement, and add `DefaultSpeed` beside
`SpawnHP` — the base constructor's own value, taken from `data.UnitDefaults()` rather than
written out a second time — for a placement that resolves to no units entry (**FR-5**).

Tests: a units placement takes its entry's column; one resolving to nothing and every
placement of a table-free world take the default; no entity a map builds is unrated.

Witnesses AC-7 · SC-7.

## T5 — the drawn stride

Per **DD-7**: `recordCells` in `pkg/game/world.go` skips an entity that owes transit ticks,
so the recorded previous cell stays the cell the transit began at; the two transit numbers
cross the seam on `ui.MapEntity` as `Transit` and `TransitSpan`; and `entityShift` in
`pkg/ui/overlay.go` takes its fraction over the transit — `left = Transit·period +
(period − clamp(elapsed))`, denominator `TransitSpan·period` — computed in 64-bit, and
identical to today's when the span is 0 or 1 (**FR-6**).

Tests: a whole transit driven through the seam, the step vector and the facing constant
throughout and the entity classified as walking; the displacement at several phases running
from the left cell to the entered one across the transit; an unrated mover and a viewer
told no phase drawing what they drew.

Witnesses AC-8 · SC-8.

## T6 — the measurement

Per **DD-9**: `cmd/classdump`'s `-databin` per-placement line reports the entity's speed
beside the health and domain it already reports, and the straight and diagonal transit its
own rate yields — read off the world, not recomputed. Numbers only; no class content.

Witnesses SC-10 (developer-run, both roots), SC-11 (manual), SC-12.

## Traceability

FR-1 → T1 · FR-2 → T1 · FR-3 → T3 · FR-4 → T2 · FR-5 → T3, T4 · FR-6 → T5 · FR-7 → T3
DD-1 → T1 · DD-2 → T1 · DD-3 → T1 · DD-4 → T3 · DD-5 → T2 · DD-6 → T2 · DD-7 → T5 ·
DD-8 → T4 · DD-9 → T6
