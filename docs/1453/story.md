# The refused walk, the struck hook and the activity list after k325

## Intent

Settle the AI pursuit remainder against knowledge k325: what a refused group
move, guard walk or escort walk does, which attacker the struck hook
remembers, how the activity list lives across LOAD, and whether a fight keeps
an attack move's stored destination. Change the engine only where a High
clause on a route the shipped data or a player action reaches disagrees.

## Authority

Knowledge pin k325 (`9550396b`).

- `AI-648` (High): Move and Swarm members walk at command state 1, Guard at
  0xb, Defend escorts at 8, script Follow at 0x11.
- `AI-649` (High for the tail, `R0004` and the order 4 arm; Medium across
  passes): on a refused route the executor tail stands state 1 down to 0xc,
  advances a patrol at 0xa, and sends every other state to `R0004`, the reach
  pick; the order 4 arm re-issues the walk to every member whose `ord+0x50` is
  0, whatever `grpAI+0x44` holds.
- `AI-650` (High / Medium): the struck hook's three callers and the attacker
  each passes; area spells 0x13 and 0xc run no hook; a poison tick and Drain
  Life lower health without it in the bodies read (Medium).
- `AI-651` (High / Medium): no store of `ord+0x0a` lies in acquisition,
  engage, reacquisition, strike install or the hook.
- `AI-652` (High / Medium): the writers of `ord+0x50` and `grpAI+0x44`; SAVE
  and LOAD copy order, group and mover raw.
- `AI-653`, `AI-654` (High / Medium): the activity rebuild walks the global
  on-map list; the AI slot rebuilds before its group gate while `+0xb388` is
  0, which the constructor leaves and Serialize does not carry.
- `MOVE-177` (High / Medium): the executor tail is the only reader of
  `mover+0x98`; the step starts are its only setters on a mover base.
- Amended: `AI-MOVE-023`, `AI-519`, `AI-384`, `AI-RETAL-056`, `AI-523`,
  `AI-546`, `AI-ACTIVITY-324`, `AI-LIST-003`, `MOVE-171`, `AI-ROUTE-045`.

## As built

- A first-game walk with no victim answers its refusal by its walk state
  (`answerRefusedWalk`, `refusedWalkTakesPick`, `pkg/sim/combat.go`). A walk
  to a sack, a guard's walk to its post, an escort closing on its subject and
  every other state take the hostile nearest within reach, the pick a refused
  pursuit already took. A member of a Move, Swarm, Swarm 2 or Roam Group, an
  actor at state 1, a patrol and a withdrawal take none.
- A refused first-game walk is a far search that finds no route or the
  sixteenth stalled near search (`pkg/sim/step.go`). Before, the stall count
  ended such a walk with no answer.
- The registry Move arm sends an idle member whose latch is 0 to its stored
  cell again whatever the rate byte holds (`savedGroupMove`,
  `pkg/sim/savedgroupsai.go`). A member that holds a walk, a victim or a
  pending order inside a Group with no rate byte keeps it, as before.
- The second game is unchanged.

## Proof

- `pkg/sim/refusedwalk_test.go`:
  `TestARefusedGuardWalkHomeTakesTheHostileWithinReach`,
  `TestARefusedEscortCloseTakesTheHostileWithinReach` (each with a
  past-reach control that takes nothing) and
  `TestARefusedMoveMemberTakesNoPickAndIsSentAgain`. All three fail on the
  production files of `2d6e7589` (loss control run in the lane).
- Census, every shipped mission, fresh start, 1500 ticks, World hash every 50
  ticks and per-entity hashes at the end: EN and RU 28 missions, ROM2 EN and
  RU 46 missions, byte-identical between `2d6e7589` and the story head. No
  mission moved: no fresh start refuses a walk with no victim in 1500 ticks.
- No SAV byte moves on the census population; the changed fields are the
  ordinary victim, destination and acquisition fields.

## Rows

| row | outcome |
|---|---|
| DIV-1520 | code changed: guard, escort and other refused walks take the reach pick; a refused Move member is sent again; remaining: no state 1 stand-down for Move members |
| DIV-1316 | narrowed: `MOVE-177` answers the readers of `mover+0x98`; the stall refusal of a first-game walk is answered |
| DIV-1555 | narrowed: the non-attack routes are implemented under DIV-1520 |
| DIV-1247 | narrowed: `AI-650` routes cited; Drain Life, poison, area 0x13/0xc and the non-damaging point spell stated as differences; owner direction kept |
| DIV-1517 | narrowed: `AI-651` keeps the destination through a fight; `AI-649` removes the rate-byte gate; the native form's missing fields remain |
| DIV-351 | narrowed: `AI-653`, `AI-654` give the list and the post-LOAD order; body membership and the LOAD clear remain |
| DIV-3267 | narrowed: `MOVE-177` limits the setters to the step starts |
| DIV-1022, DIV-2894, DIV-1599, DIV-789, DIV-790 | unchanged: the `AI-MOVE-023` amendment touches only the rate-byte gate, which DIV-1520 carries |
| DIV-3191 | unchanged: `AI-519` amendment concerns the refused-route tail, not the automatic cast moment |
| DIV-2556 | unchanged: the `AI-384` amendment concerns readers of `mover+0x98` |
| DIV-1519, DIV-1523, DIV-1737, DIV-007 | unchanged: they rest on "struck issues no order", which the `AI-RETAL-056` amendment keeps |
| DIV-2896 | unchanged: `+0x48` setters are not amended |
| DIV-1569 | unchanged: the `AI-ROUTE-045` amendment concerns the group re-issue, not the budget |

## Open debt

- A Move member is not stored at command state 1, so its first refusal does
  not stand it down to 0xc with its post moved, and a later refusal takes no
  pick (DIV-1520; `AI-519` Medium for the order of passes).
- Which pass runs first after a refusal in play is Unknown (`AI-649`).
- The point path's `+0x41` gate on the struck hook is not mapped (DIV-1247).
