# Footprint centre, step order and temporary-caster spray

## Intent and authority

Adopt the footprint claims of the k126 pin for three known defects: the position a wide mover is measured at (the retreat geometry of `DIV-348`), the order of a cell crossing and the release at an actor's teardown (`DIV-1843`, `DIV-1844`), and the group of the caster a script's Prismatic Spray is cast by (`DIV-1272`).

Authority: `MOVE-087`, `MOVE-088`, `MOVE-089`, `MAGIC-235` (Medium) and the amended `MAGIC-SPRAY-135`, with `HERO-DEATH-026` for the moment of the teardown.

## As built

- `moverFinePoint` returns a mover's centre on the 8.8 grid: the retained, paid or standing point plus (n-1)*128 per axis for a footprint of side n. The retreat flee cell reads the actor and every listed hostile at their centres. Range and edge gap already read `P + (n-1)*128` (`strikeDistance`, `edgeDistance`), so nothing changes there.
- An imported crossing in a saved-plane world no longer stops at a taken slot. The release ends at the first cell with no record or no held slot. The occupy builds each entered cell's trigger caster before the slot test, and a taken slot ends it with no store, no recompute and no coverage issue; the cells after it are neither entered nor given a caster. A slot held by a current mover or a reservation still refuses the crossing before the step (`DIV-2028`).
- A body's teardown recomputes every cell of its footprint (`releaseAtTeardown`, called where `decayPass` passes the bones stage or removes a body that never reached it). A layered cell under a removed body returns to the byte its layers give.
- A script-cast Prismatic Spray selects the primary alone. `MAGIC-235` states why: the temporary caster has no owner and no group. The original's unguarded read predicts a fault; the engine completes the cast (`DIV-2026`).
- Domain outside 1..3: the engine has three domains and refuses an undefined one at construction, so the unconditional recompute has no reachable case.

## Ledger

- `DIV-348`: wide bodies closed; the unrated transit and the bearing readers (`DIV-1631`) remain.
- `DIV-1843`: narrowed to the native stride's slot skips, the second step routine and the rows below. `DIV-1844`: unchanged; `MOVE-088` finds no deferral, which the row already assumes.
- `DIV-1272`: the temporary-caster group is decoded (Medium); the fault is `DIV-2026`.
- New: `DIV-2026` (spray fault), `DIV-2027` (mover felled in transit), `DIV-2028` (crossing onto a held cell), `DIV-2029` (teardown slot walk). `DIV-2030` and `DIV-2031` are returned unused.

## Proof

- `pkg/sim/footprintstep_test.go`: flee cells for sides 1 to 3 worked in 8.8 arithmetic, a wide hostile, the refused entry of a 2 by 2 crossing (slots, costs, casts, issue, SAVE and cold LOAD continuation), the teardown recompute at rest and for a mover felled in transit, and the temporary-caster spray against a mage's spray.
- `TestReleaseWideMoverRetreatsFromItsFootprintCentre` and `TestReleaseTeardownRecomputesTheLayeredCellUnderABody` (`pkg/game`, EN and RU roots): mission 20's terrain, cost plane and spell rules, fixture actors, ordinary ticks, with SAVE and cold LOAD continuation. Both fail on the parent commit.
- Loss controls: reverting the six source files fails the flee, teardown and refused-entry tests; the spray test passes on both because the selection did not change.
- Existing area-cost and Prismatic Spray witnesses stay green.

## Open debt

- Installed witness for the refused entry: it needs a loaded original save with a stale slot under a crossing; none of the preserved saves is known to hold one, so the proof is the fixture above.
- Question for research: where does the second step routine release, claim and occupy, and in what order?
- Question for research: which tick of a transit calls the release and occupy recompute for a native stride whose slot is empty or taken?
- Question for research: after a teardown whose release ended early on an empty slot, what do the claim bits of the later cells read in play?
- Question for research: what does a script cast of spell 14 by a wrapper-built caster do when run?
