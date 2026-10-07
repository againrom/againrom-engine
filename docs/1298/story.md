# Story 1298: Attack cycle, retreat and pursuit remainders

## Intent

Adopt knowledge k134 for the retreat, loaded attack cycle and pursuit
remainders left by stories 1290 and earlier. Every new claim is Medium in the
part the engine would change, so the story moves the pin and narrows the rows;
it changes one simulation timing: the first step of a held move.

## Authority

Pin k134. `AI-381` to `AI-385`, `MOVE-090`, amended `AI-375` and `AI-376`,
plus the claims each divergence row cites. B1 holds.

## As-built behaviour

A move held behind a loaded cycle (player move, withdrawal, escort and the
other writers of DIV-1509, 1563, 1577) makes its first walk call on the cycle's
last boundary tick, 2 ticks after recovery reaches zero (`endHeldCycleForWalk`,
`pkg/sim/combat.go`; the movement pass in `pkg/sim/step.go`). The call steps
when `Entity.Facing` already equals the direction to the first node and
otherwise turns only, the step following at 3 ticks. A structure use held
behind a cycle and a pending order other than the held destination are
unchanged. Nothing new is persisted: the state at every tick boundary is
expressible in the existing byte form.

## Claims

- `AI-385`: the AI pass holding the group tail runs once per 16 ticks on both
  callers. The engine's phase-6 pass has the same period. DIV-576 updated.
- `AI-381` (Medium outcome): a stop leaves a loaded strike running unless
  progress is cleared elsewhere. The engine keeps a loaded cycle across Hold
  and the script stops, which agrees. DIV-1581 and DIV-1582 updated; the
  condition for the progress clear stays Unknown.
- `AI-382` (Medium outcome): the first pickup pass runs the whole sack
  transfer. The engine's atomic transfer on arrival agrees. DIV-1663 narrowed.
- `AI-383` (Medium effect): group orders 3 and 5 reach the routine that stores
  pending order 0 or 8 without reading the pending order. The engine leaves a
  member holding a destination alone; whether the original rewrites it depends
  on `Player+0x28`, Unknown. DIV-1581, DIV-1582 and DIV-1664 updated.
- `MOVE-090` and amended `AI-375`: the first walk call at recovery plus 2
  steps only when the facing already matches the first node. Adopted: the
  engine steps at plus 2 on a matching facing and at plus 3 otherwise. The
  engine's facing is its own byte, so the Medium remainder is the original's
  facing at the end of a cycle. DIV-1509, DIV-1563 and DIV-1577 updated and
  kept OPEN for that remainder.
- `AI-384`: thresholds read (re-search above 16, near search above 32, full
  search above static count over 3 plus 1); no stalled-tick give-up in the
  four routines read. DIV-1316 updated: the engine's human stall count and
  16-tick refusal are this build's rule.
- `AI-376` amended: cadence only; DIV rows citing it unchanged.

## Rows

Changed: DIV-576, 1316, 1509, 1563, 1577, 1581, 1582, 1663, 1664. No change:
DIV-574, 575, 350, 352, 348, 351, 1247, 1517, 1520,
1601, 1607, 2018; no k134 claim bears on them. Owner decision: DIV-575 keeps the full hostile count as divisor and DIV-350 keeps a petrified unit standing; both stay DEVIATION, ACCEPTED. DIV-2081 to DIV-2088 are
returned unused.

## Proof

- `TestAHeldMoveFirstStepFollowsFacingAtRecoveryPlusTwoOrThree`
  (`pkg/sim/walkoffset_test.go`): the archer faces its victim, east. Measured
  from the tick the phase reaches the first boundary: a destination east
  steps at 2; west, south and north step at 3. Before the change all four
  gave 3.
- `TestAHeldMoveFirstStepFollowsRecoveryByThreeTicks`: the turning case.
- `TestAHeldMoveFirstStepSurvivesSaveAndColdLoad`: a world encoded on the last
  boundary tick and decoded steps to the same cell and hash as the live world.
- Loss control: with `step.go` and `combat.go` at the parent the first and
  third tests fail (east steps at 3; the live archer has not moved).
- Release witnesses: the EN and RU release tests of
  `check-release-tests.sh`, one root at a time, including the held-move
  witnesses of DIV-1509 to DIV-1577.
- Milestone census: the population cannot change. It counts the script
  instants and checks the build cannot run (`check-milestone.sh`, missions 10
  and 20), and this change touches no script decoder.

## Open debt

- Whether `ord+0x15 > 2` and `actor+0x136` hold at a stop on a loaded strike.
- What `Player+0x28` holds per controller.
- The original's facing byte at the end of a cycle, which selects the step
  tick.
- Occupied-ring outcome, `R0453` and the readers of `mover+0x98`.
- Which AI-pass route a session mode uses.
