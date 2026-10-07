# Story 1285: AI pursuit refusal

## Intent

An attacker whose route to its victim is refused answers as the original does:
it takes the nearest hostile within reach, or idles with its victim kept. The
mission 121 Horisontal Bridge troll no longer stands on the deck for 31 ticks
before the refusal, and a hero ordered onto a creature it cannot reach strikes
the hostile beside it.

## Authority

Pin k120. `AI-327`, `AI-328`, `AI-329` (EXP-0373), `AI-ROUTE-045`, `AI-335`,
`MOVE-073`, `AI-GUARD-007`, `AI-PURSUE-040`, `TRIG-TARGETID-032`, plus the
claims each divergence row cites. B1 holds: no question below carries an
expected answer.

## As-built behaviour

- A unit attacker holding a unit victim between attack cycles is refused at a
  far search that comes back empty and, for an AI-owned attacker only, at a
  near search aimed at the last cell of its route that comes back empty. A
  human participant's attacker takes the 16-tick stall count and one whole-map
  search that finds no cell within reach, so an ally passing in a one-cell
  corridor cannot end a player's attack order (`pkg/sim/step.go`).
- A refusal calls `answerRefusedPursuit` (`pkg/sim/combat.go`): the hostile
  nearest within whole-cell Chebyshev reach becomes the acquisition order, the
  previous victim not excluded. With none in reach, or for a human mage of reach
  below two, the order goes idle and keeps its victim (`PursuitIdle`).
- Check opcode 9 reads the victim id only for a pathing pursuit; an acquisition
  turn and an idle order read 0 (`pkg/sim/script.go`).
- Rows rewritten: DIV-1316, DIV-1520, DIV-1440, DIV-1517, DIV-244; DIV-1555 edited
  for the attack route. DIV-1247 and DIV-351 are unchanged.

## Rows verified, not changed

- DIV-1247: no claim names the remaining spell call sites of the attacker-cell
  stamp.
- DIV-351: the claims leave the activity-list lifecycle open.

## Proof

- `pkg/sim/refusedreacquire_test.go`: nearest pick, no-hostile and past-reach
  controls, human stall-count wait at the victim-aimed search, AI refusal at
  that search in a corridor, a human attack order surviving an ally that steps
  aside in a one-cell corridor (fails with `step.go` at the previous revision),
  stall-count wait for a far victim, terrain-unreachable victim, short-reach
  mage veto.
- `pkg/sim/scriptchecktargetorder_test.go`: check 9 reads 0 for acquisition and
  idle, the id for order 5.
- `TestReleaseMission121BridgeTrollKeepsItsVictimWhenItsRouteIsRefused`: refused
  5 ticks after the stop; control with no pair.
- `TestReleaseAnOrderOntoAnUnreachableCreatureTakesTheHostileBesideTheHero`
  (mission 10, creature 14 ordered through the pointer): pick, first blow,
  SAVE and cold LOAD equal, successor ticks equal; no-hostile control idles and
  saves.
- `TestReleaseCheckTargetIDReadsOrderFiveOnly`: three installed check 9 nodes
  (111:10, 140:9, 140:12), one read by a trigger pair; 4000 ordinary ticks reach
  neither state for their subjects.
- Loss controls: with `step.go`, `combat.go` and `script.go` at the previous
  revision every new sim group fails, the troll stands 31 ticks before refusal,
  and the mission 10 hero takes no pick and no idle order.
  The corridor case fails with only the human branch of `step.go` reverted: the
  attacker idles at the refusal and the victim keeps 20 health.

## Open debt

- What is `actor+0x50` of a unit walking a player Move or pickup when
  `mover+0x98` is consumed?
- What does the state-3 arm do on the tick after the unit named by `ord+0xc`
  has left the world list or reached -10 health?
- What does the original's pursuit do with every approach cell occupied, and on
  the tick after a waypoint-aimed search comes back empty?
- Which other spell call sites run the attacker-cell stamp hook, and which
  per-actor consumer reads the stamp (DIV-1247)?
- What are the activity-list population lifecycle, the session override and the
  scheduling after LOAD (DIV-351)?
- Where are the per-member destination, arrival latch and group rate byte
  stored in the world form (DIV-1517)?
- Non-attack refused walks (Move, pickup, group move, escort) still end silently.
