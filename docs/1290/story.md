# Story 1290: Order and pursuit remainders

## Intent

Apply the six claims `AI-370` to `AI-375` to the loaded-cycle and pursuit
remainders. The script's Stand Ground stands the members of its group, and a
unit whose walk to a sack is refused takes the hostile beside it. The other
rows the claims touch are narrowed to the question the claims leave open.

## Authority

Pin k125. `AI-370` to `AI-375`, plus `AI-350`, `AI-351`, `AI-352`, `AI-354`,
`AI-356`, `AI-ROUTE-045`, `AI-327` and the claims each divergence row cites.
B1 holds: no question below carries an expected answer.

## As-built behaviour

- The script's Stand Ground, native (`cmdGroupStandGround`) and saved
  (`cmdSavedGroupOrder`), stands each living member (`standScriptedMembers`,
  `pkg/sim/standground.go`): a walk or pursuit ends, a pending release,
  pickup or uninstalled cast row is replaced, a patrol or escort state gives
  way to guard, the group rate term is cleared and a loaded cycle is kept as
  for Hold. A saved order's completion word `ord+0x50` is written 0 here and by
  Hold (`AI-351`, `AI-370`).
- A refused walk to a sack takes the hostile nearest within reach in place of
  the pick-up request (`answerRefusedPickupWalk`, `pkg/sim/combat.go`); with
  none in reach, or for a human mage of reach below two, the request stays. A
  refused player move only stands (`AI-375`, `AI-350`).
- Unchanged: story 1285's human attack order surviving an ally in a one-cell
  corridor (`TestAHumanAttackOrderSurvivesAnAllyPassingInAOneCellCorridor`).

## Rows

- DIV-1581: script route closed into `standScriptedMembers`; the stop routine's
  action-word store and the two evaluation routes stay open.
- DIV-1520: refused pick-up walk answered; group move and escort walks open.
- DIV-1664: readers of `ord+0x50` found (`AI-372`); the kept record agrees;
  blind-spot readers and the order 4 reach stay open.
- DIV-1663: tick order read (`AI-371`); open on what the first pickup pass
  transfers.
- DIV-1509, DIV-1563, DIV-1577: first walk call placed at recovery plus 2
  (`AI-375`); this build's first position change is recovery plus 3, pinned by
  `TestAHeldMoveFirstStepFollowsRecoveryByThreeTicks`; step write Unknown.
- DIV-1316: `AI-373` read into the row; the 16-tick give-up stays, its
  thresholds Unknown.
- DIV-1607 and new DIV-2018: a victim at or below -10 health ends the order at
  once here and at the first AI slot in the original.
- DIV-1601 gains the script route. DIV-1582, DIV-1555, DIV-1517 unchanged.

## Proof

- `pkg/sim/scriptedstand_test.go`: walk ended, patrol and escort left, loaded
  cycle kept, saved completion word zero; native and saved groups.
  `pkg/sim/refusedpickup_test.go`: pick taken, move control, no-hostile, past
  reach and short-reach mage controls.
- `TestReleaseScriptStandGroundStopsAGroupOnTheMove`: the installed mission 40
  group 1 Stand Ground command, run through the script pass of a controlled
  copy of the mission world; six members on a script Move stand, the control
  without the node keeps walking, SAVE and cold LOAD hold state and position.
- `TestReleaseARefusedPickupWalkTakesTheHostileBesideTheHero`: mission 20
  ground with a wall across the map and a sack beyond it; pick, SAVE through
  F2, cold LOAD, tick-for-tick equal successors, first blow.
- Loss controls: with `combat.go`, `script.go`, `savedgroupcommands.go` and
  `standground.go` at the previous revision the new sim tests fail on the
  walk, the patrol, the completion word and the pick.

## Open debt

- Does the stop routine's action-word clear end a loaded strike?
- What does the first pickup pass transfer before a setter can run?
- Does the walk routine write a position step at recovery plus 2?
- Which thresholds and which cell labels decide the occupied-ring and the
  waypoint retry in the original's pursuit?
- Which readers of `ord+0x50` lie outside the census, and is the group order 4
  arm reached after a completed pickup?
- What does the executor do with a victim whose action word is 0x10 before the
  first AI slot?
