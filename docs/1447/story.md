# The group order gates the members' own states

## Result

In a first-game World the per-actor machine runs under Group order 0 alone.
Hold Position, the player's Stand Ground command through a new command Group,
and the script's Stand Ground store actor state 0xc for every member, on the
entity and in its SAV order. No arm reads that state while the Group stands at
order 3. The player's attack order puts a member at 0xc alone into a fresh
order-0 Group, whether an original SAV or this engine left it there, so a unit
on Hold Position that is ordered to attack leaves Hold. The second game's
dispatch and stores are unchanged.

## Authority

Owner direction: the AI rows D1 and D2 of the known-defects list as one story,
limited to what the pinned claims (k320) settle; the rest waits for research.
The second game's order machine is its own and takes second-game evidence
only.

- `AI-ORDER-010` (High for the table): order 0 hands control to the members'
  own states; a group under order 1 or 3 never evaluates `actor+0x50`. Orders 2,
  4, 5, 0x11 and 0xff run their own arms; 6 to 0x10 do nothing.
- `AI-POST-095` (High): `callto:R0008` = 2/2/0, and the live call lies inside
  arm 0. The per-actor machine is therefore entered under order 0 alone, which
  settles orders 2, 4, 5, 0x11 and 0xff as well as 1 and 3. Order 0xff runs a
  per-member routine of its own, which the engine already maps to the
  acquisition (`savedAcquire`).
- `AI-351` (High): both order 3 setters store `actor+0x50 = 0xc`, pending 0,
  `ord+0x14` the reach, `ord+0x38`, `ord+0x50`, `ord+0x60`, mover `+0x7c` and
  `actor+0x54` zero, and the post word.
- `AI-370` (High): the script's Stand Ground arm stands each member through
  the stop routine with the same stores and state 0xc, then writes order 3.

## As built

The dispatch gate. `actorLayerDecides` (`pkg/sim/guardarm.go`) reads, in a
first-game World, the order of the registry Group that holds the actor, the
record every first-game Group command writes; a second-game World reads its
native record as before. `actorPass` (`pkg/sim/actor.go`) applies it before
the dispatch in a first-game World, so every arm is entered on the same
condition. In a first-game World `actorPass` reaches only authored patrollers;
the other states are dispatched by `savedActorDispatch`, which `savedGroupPass`
already called under order 0 alone. The observable change is the patrol arm:
a script `Group Command: Guard` (or any later order) against a Group on patrol
leaves its members in the patrol state, and they no longer walk the ring or
scan for a fight. The `orderRoam` skip stays for the second game; in a first
game the gate covers it. A second-game World keeps the gate on the guard arm
alone.

The state store. `newSavedCommandGroup` stores 0xc on the entity and the order
when it builds a Stand Ground Group (Hold); `standScriptedMembers` stores 0xc in
a first-game World (script Stand Ground). `syncSavedStandGround`, shared by
both, also clears `ord+0x38` and `ord+0x60` and mover `+0x7c`, which the engine
otherwise only holds as retained SAV bytes. `actor+0x54` is not cleared: the
engine rebuilds the action word from the current cycle at SAVE (DIV-1582).

The attack order. The player's attack order puts an attacker in an acquire
state into a fresh order-0 Group (DIV-1440), as `AI-CMD-033` states for every
player order and `AI-CMD-054` for order `0x19`. A Hold or Stand Ground member
holds 0xc, so it is released like any other 0xc member; no condition separates
a member an original SAV loaded at 0xc from one this engine's setter stored.
Its Group mates stay in the order-3 Group. A unit on Hold Position now leaves
Hold when ordered to attack; before this story Hold stored 0xb, which the
attack order did not release. A member at 0xb (a SAV an earlier build wrote)
keeps its Group as before.

Old saves. A SAV an earlier build wrote holds 0xb for a Hold or Stand Ground
member. It loads unchanged. Under order 3 no arm reads 0xb or 0xc. If the Group
later returns to order 0 with the state unchanged, a 0xb member runs the guard
arm and a 0xc member runs the acquisition arm, each deterministically. No
original SAV loads differently.

CHANGELOG: one `[ROM1]` line for the attack order out of Hold. The patrol arm
ran under a nonzero order on no shipped mission (census below), and a Hold or
Stand Ground member that is given no order behaves the same with 0xb and 0xc
under order 3.

## Proof

- `TestFirstGameArmsRunUnderGroupOrderZeroAlone` (`pkg/sim/actorgate_test.go`):
  for guard, acquire, patrol, defend, follow and retreat, a member's state is
  read and its arm entered under order 0, and is never read under orders 1, 2,
  3, 4, 5, 6, 0x11 and 0xff. The instrument is the test seam `actorStateRead`,
  called at both dispatches. With the gate removed the patrol case fails under
  order 1.
- `TestScriptGuardOverAPatrolStopsThePatrolArm`: a script Guard against a
  Group a Patrol node set walking leaves the patroller on its cell for four
  decisions; the control walks. It fails without the gate.
- `TestSecondGameDispatchKeepsItsGuardOnlyGate`: a second-game patroller whose
  Group order is raised to 1 is still read and still walks.
- `TestAttackOrderReleasesAHoldMemberIntoAFreshGroup` (`pkg/sim`): Hold on
  two members stores 0xc in one order-3 Group; an attack order on one puts it
  alone into a new Group at order 0 with state 3, and the other stays in the
  Hold Group at 0xc. It fails with the exemption restored.
- `TestReleaseAttackOrderReleasesAnOriginalHoldMember` (EN): loads each
  original SAV in `AGAINROM_ORIGINAL_HOLD_ATTACK_DIR/<en|ru>`, picks the first
  party member at 0xc in an order-3 Group and the nearest hostile, gives the
  attack order and requires a new one-member Group at order 0. On `game0016.sav`
  (member 0 against hostile 27) and `game0000.sav` (member 97 against hostile
  5) the World hash at ticks 1, 601 and 901 equals base `f2886439`:
  `33e14eb2144dddba`, `d1d30a002fc05321`, `a5247b78880fc8ce` and
  `f7c7388f0239d165`, `7e81c8e12d26a9d6`, `07b1cb7b54df8fe5`. The one RU
  original SAV in that set is a town save, so RU has no input and skips.
- `TestReleaseHoldStoresStateCSAVColdProcess` (EN and RU): Hold on two
  warriors stores 0xc on each entity and order; the F2 SAV holds U50 = 0xc for
  both; a main-menu LOAD in the process and in a fresh process restores 0xc,
  and 240 ticks of the next action (blows on the creature beside the first
  warrior) match the uninterrupted session and the fresh process tick for tick.
- `TestReleaseScriptStandGroundStopsAGroupOnTheMove` (EN and RU) now requires
  0xc on every member of mission 40's Stand Ground Group and repeats the same
  SAV, cold LOAD and fresh-process check on its SAV.
- The retreat-ending tests (`TestReleaseHoldPositionEndsARetreat`,
  `TestHoldPressedDuringARetreatEndsTheRetreat`,
  `TestHoldPressedOnARetreatingAndAnIdleMemberLeavesTheIdleOneAsItWas`) pass
  with 0xc.
- Census. Instrument: a lane-only test that starts every shipped mission with
  the default party at normal difficulty, steps 1500 ticks (or to the outcome),
  counts before each tick the authored patrollers whose registry Group order is
  neither 0 nor 0x11, and writes the World digest and a per-entity digest with
  the actor state masked. EN and RU, 28 missions each: before the change 0
  patrollers ran under a nonzero order; 5 missions (40, 110, 131, 150, 151)
  changed digest; over every entity of every mission the masked digest is
  unchanged, and exactly 90 entities per root changed state, all 0xb to 0xc,
  all members of order 3 Groups. `rom2-en` and `rom2-ru`, 46 maps each: no
  digest changed.

## Open debt

- At mission start the participant's constructed order holds 0xc while its
  entities hold 0xb until the first SAVE and LOAD; neither is read under order
  3. Unchanged by this story.
- A Hold member at 0xb from a SAV an earlier build wrote keeps its order-3
  Group under the attack order and returns to 0xb when its victim is gone
  (`savedDecision`) (DIV-1440, DIV-1519).
- The attack order releases the acquire state only; the other states keep
  their Group as DIV-1440 states.
- The remaining D1 and D2 rows wait for their claims.
