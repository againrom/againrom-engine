# The loaded attack cycle and retreat rows of k321

## Result

In the first game a retreating unit wider than one cell now computes its flee cell from its
stored position record, the top-left cell of its footprint, and no longer from
its footprint centre. Ten divergence rows that leaned on Unknowns of the
loaded attack cycle, a pickup's two passes and the flee and bearing geometry
now cite the claims that answer them: one row is closed, one carries a code
change, eight are narrowed to what the claims leave open.

## Authority

Pin k322. The claims are read with `go run ./tools/claim <ID>`.

- `MOVE-174` (High for the getters, the picker's reads and the step bodies):
  the flee picker `R0104` reads the actor's raw position record,
  `cell * 256 + sub` per axis, without the size centre, and reads nothing from
  the mover.
- `MOVE-175` (High): the centre-bearing helper returns one of eight facings for
  every delta of two 16-bit centres. `MOVE-176` (High): the step-facing helper
  classifies signs only and returns `(facing & 7) << 5` at a zero delta.
- `AI-632` (High for the stores, Medium for the writer sets): at a stop on a
  loaded strike `actor+0x136` holds 0, so the executor's progress clear does
  not run. `AI-638` (High for the 34 read bodies): the external order writers
  store no progress, counter, phase, completion or countdown.
- `AI-637` (High for the read bodies, Medium for the composition): a cycle
  ends with its installed facing.
- `AI-634` (High for the arm and the lookup): the action 2 arm runs once per
  pickup; the lookup of an emptied cell returns 0 and the arm does nothing.
- `AI-635` (High for the census and the reset): group order 4 is installed only
  after a per-member reset of `ord+0x50` to 0.
- `AI-633` (High for the census, Medium per origin): the stores of
  `Player+0x28`; the join routine is Unknown.
- `AI-636` (High for the bodies, Medium for the window composition): no read
  route clears an attacker's victim pointers when the victim dies.

## As built

`fleeCell` (`pkg/sim/withdraw.go`) reads the actor through
`savedProjectileTargetPoint`, the raw position record: `0x80` on both axes
standing on a cell, the retained SAV motion, or the paid stride steps between
cells. The listed hostiles are still read at their footprint centres
(`moverFinePoint`); no claim states which getter the helpers' hostile sums
read. Automatic withdrawal and explicit Retreat share `fleeCell`. For a unit of
side 1 the two readings are equal, so only units wider than one cell change
their flee cell. A second-game World keeps the centre reading: its order
machine takes second-game evidence only, and `MOVE-174` is first-game
evidence.

No other production code changes. The rest of the story pins what the engine
already does against the new claims: `headingOf` against `MOVE-175`'s table
for every centre delta, `stepHeading` against `MOVE-176`'s table and every
facing byte, a latched cycle's facing against `AI-637`, a second take on an
emptied cell against `AI-634`, and the saved player Move's completion word
against `AI-635`.

## Rows

- DIV-348 code changed and narrowed: the actor is read raw
  (`TestFleeCellReadsTheActorAtItsRawPositionRecord`,
  `TestReleaseWideMoverRetreatsFromItsRawPosition` on EN and RU); the bearing
  readers match `MOVE-175` and `MOVE-176`
  (`TestHeadingOfMatchesTheCentreBearingTableForEveryDelta`,
  `TestStepHeadingMatchesTheStepFacingTable`). Open: the hostile sums' getter
  and the step record of an unrated transit.
- DIV-1509 narrowed: the facing at the end of a held cycle is the loaded one
  (`TestACycleEndsWithTheFacingItWasLoadedWith`), `AI-637`'s installed facing at
  Medium. Open: the five unread virtual calls and the executor tail's turn.
- DIV-1563 narrowed: `AI-638` confirms that the four handlers store no progress.
  Open: the original's saved order record holding the pair.
- DIV-1577 narrowed: `AI-638` covers Defend, Patrol, the Swarm walk-on, the
  script group commands and structure use. Open: the pending-order and
  progress split, and the saved record.
- DIV-1581 narrowed: `AI-632` settles the strike at a stop; `AI-633` gives
  `Player+0x28` per origin. Open: the join routine, the participant's order-5
  no-target behaviour and the native heal branch.
- DIV-1582 narrowed: `AI-632`, `AI-638` and `AI-633`. Open: `AI-354`'s common
  tail during a running cycle and the join routine.
- DIV-1607 narrowed: `AI-636` states that the victim pointers survive a death.
  Open: active-writer continuation and terminal routes other than action 0x10.
- DIV-1663 closed: `AI-634` answers both Unknowns
  (`TestHoldStrikeReplacesQueuedPickupAndRelease`,
  `TestHoldStrikeReplacesQueuedCast`,
  `TestASecondTakeOnTheEmptiedCellTransfersNothing`).
- DIV-1664 narrowed: `AI-635` places the reset before order 4
  (`TestPlayerMoveWritesTheSavedCompletionWordZeroWithOrderFour`). Open: the
  saved completion word 1 against the original's 0, and a route into a group
  already at order 4.
- DIV-2018 narrowed: `AI-636` against this build's immediate clear. Open:
  whether the strike refuses a victim the death pass removed from the world
  list.

## Proof

- Focused tests in `pkg/sim` and `pkg/game` named under Rows. The flee test
  carries the loss control: the footprint-centre reading gives another cell
  for sides 2 and 3, and a second-game World still gives it.
- Census: every shipped mission from a fresh start, 1500 ticks, World hash and
  a per-entity hash, on `2be3b46b` and on this story's head, with a lane
  instrument that is not committed. EN 28 and RU 28 missions, ROM2 EN 46 and
  ROM2 RU 46 maps: the hashes compared at tick 1500 are equal on every root.
- Review comparison, EN, base `804b4114` against this story, every tick: 149
  SAVs (91 enter a mission) for 1200 ticks and 28 fresh missions for 3000
  ticks. One flee cell changes: on mission 131 entity 147 (type 71, side 3,
  Wimpy 63) picks (11,15) instead of (12,18) at tick 1319 while held in its
  loaded attack cycle; the World hash differs on ticks 1319 to 1353 and is
  equal again from tick 1354. Start and end hashes are equal on every SAV.
  The changelog carries one `[ROM1]` line.
- Original-save LOAD: the milestone-2 acceptance and
  `TestReleaseAttackOrderReleasesAnOriginalHoldMember` on EN.

## Open debt

- DIV-2018 and DIV-1607: the original keeps an attacker's victim pointers
  through a death and, at Medium, keeps striking until the next AI slot; this
  build clears both at once. A code change waits for the unlinked-target
  question.
- DIV-1631 is answered by `MOVE-175` and pinned by
  `TestHeadingOfMatchesTheCentreBearingTableForEveryDelta`; it is outside this
  story's rows and stays open for the seat.
