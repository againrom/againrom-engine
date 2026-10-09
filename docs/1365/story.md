# Pursuit search

## Intent and authority

Owner report on a save: orcs on a river bank stand still instead of closing in
on a victim, although cells nearer it are free. Hotfix 0.89.4 answered it with
a stand-in, an eight-ring far search for a victim within eight cells
(DIV-2456). Owner direction: replace the stand-in with the original's
full-versus-near search cadence and picker B.

Authority, knowledge pin k201: `AI-413`..`AI-418`, `MOVE-097`..`MOVE-100`,
`AI-373`, `AI-384`, `MOVE-ALT-018`, `MOVE-ALT-019`, `MOVE-ALT-020` (narrowed by
`MOVE-098`). Medium and Unknown parts take the smallest rule and a row,
DIV-2556 to DIV-2562.

Base: public main `3ddc6ed6`, game 0.92.1.

## As built

A unit attacker outside reach of a unit victim runs one pursuit pass per tick
on its cell centre (`pkg/sim/pursuitsearch.go`). Its search record stands for
mover bytes `+0x7c`, `+0x8a`, `+0x09`, `+0x76` and `+0x8c`:

- A victim other than the recorded one resets the record: count and passes
  0xff, route end and aim at the victim's cell, no static list (`AI-413`).
- Passes above a third of the static list plus one re-search. A count above
  five runs the full search toward the victim's cell, picker A over rings 1 to
  `(D>>2)+3` (`AI-414`, `AI-394`); five or less rebuilds the route end as one
  node (`AI-415`). An empty list refuses the pursuit.
- Standing on the route end while the victim stands where the last full search
  found it forgets the victim, so the next pass searches in full (`AI-415`).
- The near search aims at the route end while five nodes or fewer remain, else
  at the head, or at the node three after it when the head is within three
  cells. A head within three cells of the cell the search started from is
  removed after a counted pass (`AI-373`, `AI-384`, `AI-417`). A held victim
  follows both actor-ID remaps with the attack target.
- The near search settles by picker B (`pkg/sim/pickerb.go`): eight rings
  around the victim, entered on the edge the 16-way bearing selects where the
  line between the fine centres crosses it, two walkers turning at the side
  ends, at most 100 steps per ring, the first strictly lowest label kept,
  walker 1 first; an occupied cell is skipped and the start cell may be
  returned (`MOVE-097`..`MOVE-100`, `MOVE-ALT-020`). The single-precision slope
  is reproduced exactly in integers.
- An empty near search aimed at the route end refuses an AI-owned attacker;
  aimed at a waypoint it retries next pass. A human participant counts stalled
  passes instead and forgets the route end, so its next pass searches in full
  toward the victim's current cell (DIV-2561, DIV-1316).
- A pass whose step only turns is not counted (DIV-2556). A refusal, a lost
  victim and a new order clear the record (DIV-2557). A group reissue at the
  held victim keeps the route (DIV-2558).

The stand-in's eight-ring far search and the nearest-to-victim settle rule are
removed; DIV-2456 and DIV-2225 are closed. The engine save form carries the
record as optional form 115; a world holding no record keeps its earlier bytes.
A SAV carries the record in the native continuation supplement; the SAV mover
bytes are neither read into the record nor written from it
(DIV-2562).

## Proof

Focused tests in `pkg/sim/pursuitsearch_test.go`: the bearing sectors, the
crossing against a double-precision line over a single slope, the wide-mover
entry that leaves the ring, the first strictly lowest label, occupied cells and
a full ring cancelling, full search then rebuild with counts 8, 5 and 1, the
band case of `AI-418` from 7 and from 12, the record round trip, and a reissue
at the held victim. `pkg/sim/pursuitbank_hotfix_test.go` (the bank test of the
stand-in) passes unchanged.

The owner's save, RU, 600 ticks, victim at (77,37): the ten orcs on column 70,
seven cells from the victim, go idle on their first pass, as the band case of
`AI-418` predicts for a full search over rings 1 to 4 that finds no cell. Orcs
75, 79, 76 and 104, eight or more cells away, take route end (72,37), walk
round the column and stand on column 72 in their attack phase.

Release drives the cadence moved, each rebuilt on its own subject:

- `TestReleaseAcquiredVictimLeavingReachTurnsWithoutWalking`: Defend acquires
  only on the actor pass, so the drive tries sixteen start delays.
- `TestReleaseALargeAICreaturesFarSearchSpendsTheNxNArm`: pursuit routes now
  end beside the hero (`MOVE-099`); the budget bound covers those routes.
- `TestReleaseCarriedPotionMissionSaveKeepsModifierWordThroughColdLoadAndExpiry`:
  both worlds heal the hero below half health until the potion expires.
- `TestReleaseAnOrderOntoAnUnreachableCreatureTakesTheHostileBesideTheHero`:
  the hero starts 27 cells from the creature, where the first full search
  finds no cell.
- Scenario 1060 (mission 40) removes the two pursuers that now reach the hero.

Merged with game 0.93.0, whose melee facing turn moves the same cadence:

- `TestReleaseOriginalGround1076GoldEffectsAndStackSurviveNativeContinuation`
  found an engine defect. An arrow fired after the loaded spell graph retired
  refused the SAVE ("spell graph carrier lost while effects remain"). Only an
  area driver needs the graph's records now; a projectile driver has its own
  (`TestRetiredSpellGraphRefusesOnlyForAnAreaDriver`).
- `TestReleaseAnAttackedGroupAtKadaganPostFightsAsAGroup`: the far member's
  full search settles beside the occupied victim cell (`AI-414`, `MOVE-099`)
  and arrives after the hero falls; the hero is healed below half health.
- `TestReleaseRescuedVillagersReplyAndHurtInThePeasantBank`: a monster group
  scores the joined villager best (`candidateCost`: distance first, the lowest
  id on a tie) and strikes it before the fight; the monsters already fighting
  the party are removed when it joins.
- `TestReleaseSkillLevelsSurviveEverySavePath`: the trainee also casts at its
  attackers on its own, so the award bound is two per tick, not per ordered
  cast.

The three adapted drives pass on EN and RU both on this branch and on main
`5006517a`.

Gates on `ff84f1ce`: scenarios EN 54 s and RU 32 s pass. The census script set
is unchanged; the mission 10 drive's escort is lost at tick 800 instead of
496, with 7 of 36 units moved and 1 fallen. M2 (EN and RU, 214 s) fails the
same 24 tests as before the change, less two. Release tests pass on EN in
663 s and RU in 1076 s. `go test -trimpath -count=1 ./...` (213 s) fails only
`TestReleaseOgreStrikesOnlyWhatItsBodyTouches`, which needs the installed
root beside its save and passes with it on EN and RU. `check-no-game-assets.sh`
passes.

## Open debt

- DIV-2556: no stored dynamic list and no `mover+0x78`; the passes per stepped
  cell are Medium.
- DIV-2559: the fine centre ignores a sub-cell position in transit; the FPU
  precision in play is Unknown.
- DIV-2562: SAV mover projection of the record.
- The owner's save: the column of orcs seven cells from the victim is refused
  under `AI-418` (Medium). That state was produced by this build before the
  change; whether the original's crowd ever reaches it is Unknown.
