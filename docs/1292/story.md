# Creature spellbooks: aim, approach and the draw

## Intent and authority

A creature whose class has a spell draws a spell each engage pass and casts it
at the target its arm fixes, walks into range when it is out of range, and
keeps the cast armed. Authority: `MAGIC-237` (each of the 28 spell ids has one
arm: seven aim at the victim, nine at the caster, eight at the victim's cell,
ids 9 and 26 at the cell next to the caster, ids 4 and 25 write no order),
`MAGIC-238` (the cell next to the caster is one step toward the victim),
`MAGIC-239` (an out-of-range or unfaced cast walks and stays armed; admission
has no range test), `MAGIC-240` (Medium: order byte 0x60 is the retention
flag), `AI-376` (the draw runs while a cast is pending; a miss writes order
kind 5 over the retained order), `AI-377` (a zero slot id is skipped before the
draw; non-mage actors draw). Owner direction: cast per the claims; an Unknown
takes the smallest consistent rule and a DIV row.

## As built

- `pkg/sim/creaturecast.go` holds the arm table `creatureAimOf`, the step cell
  (`creatureStepCell`: the eight-way heading of the direction helper, kept in
  the map) and `creatureEngageCast`, called first by `orderAttack`.
- The draw runs on every pass, before the cast-busy test. A zero slot id makes
  no `rand` call; an all-zero block draws nothing and the default engage runs.
- A hit casts at the arm's target with a retained order. A cast refused only by
  range becomes an armed approach (`bookApproach`): the creature walks toward
  its victim as an attack pursuit, begins the cast in range and repeats it in
  place as a retained order. Another engine refusal writes no cast and the
  ordinary engage runs.
- While a cast is pending a miss ends the retained order (a winding cast
  completes once, a recovery keeps its wait); a hit rewrites the retained
  operands outside the wind-up.
- The group join pass skips a member whose selector draw already ran in the
  decision pass, so a creature draws once per pass; a member with a pending
  cast that only the join pass reaches draws there.
- SAVE writes a retained creature cast with byte 0x60 set; an armed approach is
  a kind-8 or kind-9 order at progress 0 with no actor cast state. LOAD
  rebuilds the retained record, and the armed phase persists in the sim byte
  form and the SAV `Actions` supplement.
- The creature walk is the existing attack pursuit; `armCreatureApproach` is
  its only new `attachAttack` caller.

## Hashed state

The retained draw changes the RNG stream: every creature with a spell draws
once per pass per nonzero slot, including while its cast is pending, and no
draw is made for a zero slot id.

## Proof

Sim tests in `pkg/sim/creaturecast_test.go`: arm per spell id, step cell by
heading and at the map corner, out-of-range walk and cast in range, in-place
repeat, reload of an armed approach in lockstep with the live world, draw
while pending, miss while pending, zero block, zero threshold.

Release tests on EN and RU (`pkg/game/creaturebooks_release_test.go`), each run
through ordinary routes in mission 130 with the other units off the map where
the oracle needs one victim:

- `TestReleaseCreatureSpellsAreAimedByTheirArm`: classes 73, 76, 80, 70 and 74
  cast at the aimed target; the step cell equals an independent floating-point
  bearing.
- `TestReleaseCreatureCastOutOfRangeWalksAndStaysArmed`: a draw at distance 8
  arms an approach, the cast begins at distance 7.
- `TestReleaseCreatureRetainedCastEndsAtAMissingDraw`: 1 cast in 1500 ticks
  against a bound of 6.
- `TestReleaseCreatureArmedCastSaveColdLoad`: an armed creature in the
  player's session saves a kind-8 or kind-9 order with byte 0x60 set, cold LOAD
  holds the same record, and both sessions keep one hash.
- `TestReleaseCreatureSpellSaveColdLoadCastsNext` gains the byte 0x60 check
  for a charging retained cast.

Loss control: the four new witnesses and the extended one fail on
`18316c4d` (the victim-arm case passes there, as that arm is unchanged); on
that commit a creature cast 80 times in 1500 ticks against 1 here. The
story1264 witnesses stay green and are unchanged apart from the byte 0x60
check.

## Open debt

Rows `DIV-1725`, `DIV-1727`, `DIV-1728`, `DIV-1730` (narrowed), `DIV-1729`
(narrowed), `DIV-1726` (closed), `DIV-2032` to `DIV-2035` (new). Unknowns kept:
the kind-8 distance metric, the end of the walk, a target that dies while
armed, the map-edge step cell, the reach of owner-0 actors through the
selector, and an original-runtime load of an engine-written armed order.
