# Tasks — a domain in the bytes, a plane per layer, and a rest test at three sites

Legend: **files** what the entry may change — a permission, not a prediction · **fences** what it
must not do · **done when** the observable it leaves behind. Entries land in ascending order, each
depending only on those before it. The mutants named in the plan's success criteria are applied to
production code one at a time, the whole tree run, the failing tests named, and each reverted by the
entry that owns it; a kill is claimed only where it was run.

## T1 — the domain, and the bytes that carry it

**files** MODIFY `pkg/sim/world.go`, `pkg/sim/binary.go`, `pkg/sim/route.go`, every pin and call
site the widened record and the domain argument reach in `pkg/sim`, and `pkg/mapload/gridform_test.go`,
whose digest and form length the version moves; ADD `pkg/sim/domain_test.go`

DD-1, DD-2, DD-12, DD-13, DD-15 — FR-1, FR-2, FR-7.

**fences** occupancy is still one plane and no mover's contention changes: this entry gives the
domain a terrain term and a byte, and nothing else. `Step` is not touched. No pin is deleted — each
moves and says what it now asserts — and the recomputed digests are derived from the new bytes, not
read back out of the encoder.

**done when** nine answers are recorded for three domains against the three derivable grid bytes;
the form's version, total length and record width are checked against numbers written out by hand;
a world holding each domain round-trips and one carrying a byte outside them is refused with the
receiving world unchanged field for field; and a flyer's stored route across a blocks-ground cell
reads back where a ground mover's is refused. Then SC-8's first and eighth mutants — the ghost's
mask set to bit 0, and every domain held to bit 0 on route decode — are applied one at a time, the
whole tree run with the failing tests named, and each reverted.

## T2 — two planes, and which one a mover contends on

**files** MODIFY `pkg/sim/route.go`, `pkg/sim/step.go`, `pkg/sim/enterable_test.go`,
`pkg/sim/domain_test.go`

DD-3 — FR-3.

**fences** every flyer is still counted whatever it is doing, so no interpenetration exists yet and
the soft rule is not started early. A ground mover's answers are unchanged at every cell: the
ground layer holds exactly what the single plane held, plus ghosts, and there are none in any
fixture this entry does not add.

**done when** a flyer and a ground unit ordered onto one cell end **on** it, a ghost and a ground
unit end distinct, and a ground pair end distinct too; the predicate is asked directly at a cell
each domain stands on, so the six answers are recorded and not inferred; and the whole suite is
green with no landed expectation edited.

## T3 — a flyer is counted only at rest, and flies blind

**files** MODIFY `pkg/sim/route.go`, `pkg/sim/step.go`, `pkg/sim/domain_test.go`

DD-4, DD-5, DD-6, DD-8 — FR-4, FR-5. The who-is-counted predicate lands here and both subtraction
sites are put behind it in the same entry, so no site is left guessing who the seed put in a plane.

**fences** the far call site keeps its relation and the near call site's window, budget rule and
settle rule are untouched — only which relation an air mover is handed moves. `clearFelled` keeps
the scratch-free clearing call, and no clearing site gains a nil-checked parameter. Nothing here
decides where an order may end: a flyer may still rest on a peer, and the criterion that says
otherwise is the next entry's.

**done when** two moving flyers occupy one cell at some tick and both still arrive, with no detour;
a flyer crosses a resting flyer's cell on its straight line and lands exactly on a free target; and
the same two layouts in ground units get nowhere at all. Then SC-8's second and third mutants — the
air seed made unconditional, and the near search handed the unit relation for air — are applied one
at a time, the whole tree run with the failing tests named, and each reverted. The two bookkeeping
mutants are the next entry's: until something READS the air plane a corrupted count changes no
outcome.

## T4 — where an order may end

**files** MODIFY `pkg/sim/route.go`, `pkg/sim/optimised.go`, `pkg/sim/step.go`,
`pkg/sim/giveup_test.go`, `pkg/sim/domain_test.go`, `pkg/sim/approach_test.go`

DD-7, DD-9, DD-10, DD-11, DD-16 — FR-6.

**fences** the picker's ring order, strict accept and growth bound are not touched, and neither is
the near search under any domain: the predicate this entry adds answers true for every mover but a
flyer, so a ground mover's budget, break test and settle are the expressions they were. No new field
records that a mover is resting, and no search gains a picker it did not have.

**done when** two flyers ordered onto one cell end distinct, including where they would arrive on
the same tick; one ordered onto a resting flyer's cell settles beside it and its own target names
the substitute from the first tick; the same fixture under the mode that does not settle ends the
order in the first tick with no per-tick re-search; a flyer re-ordered onto its own cell ends counted
there and a settle onto the mover's own cell stays reachable; and the disclosed case is asserted as
the overlap it is. Then SC-8's remaining five mutants — the two the entry before this one could not
observe, and this entry's own three — are applied one at a time, the whole tree run with the failing
tests named, and each reverted.

## Traceability

T1 → FR-1, FR-2, FR-7, DD-1, DD-2, DD-12, DD-13, DD-15 · T2 → FR-3, DD-3 · T3 → FR-4, FR-5, DD-4,
DD-5, DD-6, DD-8 · T4 → FR-6, DD-7, DD-9, DD-10, DD-11, DD-16

FR-8 and DD-14 are a fence on every entry above and the deliverable of none: no entry may change a
production file outside `pkg/sim`, and the one test file outside it that moves is named where it
moves. FR-9 is likewise a standing condition rather than a slice — every entry
leaves the tick integer-only — and SC-1 through SC-8 are measured over the finished tree by the
evidence stage.
