# Story 1300: burst area, Prismatic Spray and area cost remainders

## Intent

Adopt k135 (EXP-0453) and k136 (EXP-0452) for the known defects C5 (shot and
burst remainder), D5 (Prismatic Spray remainder) and D6 (area cost remainder),
change the engine where a High part of a claim contradicts it, and state the
rest as divergence rows.

## Authority

Pin k136 (`759d7e1a`). B1 holds: the open questions of stories 1288 and 1291 are
answered by these claims; none is restated here.

- C5: `ANIM-121`, `ANIM-122`, `ANIM-123`, `ANIM-124`, `MAGIC-245`,
  `MAGIC-246`, `MAGIC-247`, `SAV-1156`, `SAV-1157`; amended `SAV-1153` to
  `SAV-1155`, `ANIM-110`, `ANIM-114`, `ANIM-115`.
- D5: `MAGIC-241` (Medium), `MAGIC-242`, `MAGIC-243`; amended `MAGIC-235`.
- D6: `MOVE-093` to `MOVE-096`, `AI-387`; amended `MOVE-088`.

## As-built behaviour

- Fire_Ball transport target point (`pkg/sim/spelldelivery.go`). An actor wider
  than one cell is aimed at its footprint centre less one sub-unit per axis,
  in sixteen bits (`MAGIC-245`). The countdown of a cast at a cell and of a
  one-cell actor is unchanged.
- Heading helper (`pkg/sim/reacquisitiondirection.go`). `headingBetween`
  measures both footprint centres on the fine grid, the sub-cell bytes
  included, and returns the axis when one magnitude exceeds twice the other and
  the diagonal of the two signs otherwise (`MAGIC-242`, `AI-361`). The
  Prismatic Spray turn term and facing, the reacquisition winner and the
  creature step cell use it. It replaces a 22.5 degree octant rule on
  whole-cell anchor deltas; the two agree on the 81 published deltas and
  differ for ratios between 0.414 and 0.5 and for wide or off-centre actors.
- No change: the unit shot build tick and offset agree with `SAV-1157`; the
  blast cells are -Radius to +Radius with Radius the spell's parameter 9
  (`MAGIC-246`); the burst record has no size field and 22 segments
  (`ANIM-123`); the rider lowers hit points on its own tick, as client arm 0x73
  applies them at dispatch (`ANIM-124`); the Ballista and Catapult carry a
  Fire_Ball weapon spell (`MAGIC-247`); the crossing tick follows `MOVE-094`;
  a felled mover is not stepped (`MOVE-096`).
- Spell 14 cast by a wrapper-built caster: the engine keeps selecting the
  primary alone and completes the cast. `MAGIC-241` predicts a fault; none is
  reproduced (`DIV-2026`).

## Ledger

Rows updated: `DIV-944`, `DIV-1272`, `DIV-1631`,
`DIV-1843`, `DIV-1876`, `DIV-1877`, `DIV-1878`, `DIV-1879`, `DIV-2026`,
`DIV-2027`, `DIV-2028`, `DIV-2029`. `DIV-939` and `DIV-1842` are untouched (the latter an owner decision).
No new row; `DIV-2097` to `DIV-2104` are returned unused.

## Proof

- `pkg/sim/fireballaim_test.go`: a two-cell aim at speed 300 and 380 counts 2
  and 3 where the cell centre counts 3 and 2, on the cast and weapon-rider
  routes, with SAVE and cold LOAD keeping the countdown. Fails on the parent.
- `pkg/sim/headingfine_test.go`: 7:3 and exactly 2:1 headings, a fine offset,
  footprint centres of a two-cell candidate, and a Prismatic Spray whose
  secondary choice turns on the 2:1 heading. Loss control: the replaced octant
  rule fails the first and last.
- `pkg/game/unitshot_test.go` `TestAUnitShotOffsetFollowsChargeShootDelayAndDistance`:
  damage tick minus record tick for ShootDelay 2 and 6, d = 1 to 6, against the
  `SAV-1157` arithmetic. Passes on the parent: it witnesses agreement.
- `pkg/game/siegeweaponspell_release_test.go` (EN and RU): the hired Catapult
  and Ballista resolve a worn Fire_Ball weapon, powers 70 and 40 on both roots.
- Existing witnesses stay green: the burst, siege rider and cold LOAD tests of
  story 1288, the area-cost crossing tests and the Prismatic Spray tests.

The milestone census population is unchanged: no simulation timing or script
changed except reacquisition and creature-cast headings, which move only for
ratios between 0.414 and 0.5 or wide or off-centre actors.

## Open debt

- `DIV-1877`: one to two ticks of the burst record are still unexplained;
  `ANIM-121` removes a delivery delay as the cause on the local path.
- `DIV-2028`: a crossing into a cell a live unit holds (`AI-387`, High) is still
  refused here. The occupancy plane does not represent two living actors on
  one cell, and no preserved save is known to hold such a transit.
- `DIV-2027`, `DIV-2029`: the felled mover's stored cell and the claim-bit
  clear need the original's slot table.
- `DIV-1631`: the strictness of the 2:1 comparison comes from the published
  deltas.
