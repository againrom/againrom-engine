# A first-game mover's step and the flee pick as in the original

## Result

In the first game a unit that withdraws or retreats picks its flee cell from the hostiles' anchor points, so a
hostile wider than one cell is measured from its top-left cell. A unit stepping diagonally north-east or south-west
crosses into the next cell on both axes at once, as the original does, and a game saved during that step carries
that position and continues it after loading. A spell cast given while a loaded step is between cells waits for
the step to end instead of cutting it short. The second game keeps its own rules.

## Authority

- Owner direction: DIV-348 is corrected to the original where the claims state it.
- `AI-698` (High): the flee helper sums each listed entry's anchor fine point, not the size-adjusted centre.
- `MOVE-174` (High for the picker's reads and the step body): the picker reads the raw position record; the step
  subtracts one from x when the octant is 1 or 5 and both new sub bytes are 0.
- `MOVE-200` (High / Medium): a diagonal stride is `trunc(0.707 v)` for `ceil(256 / stride)` calls, 0 at v 1.
- `MOVE-199` (High / Medium): the step count is zeroed at a transit start and at centring; no order writer found.
- `SAV-1406`, `MOVE-201` (Medium): LOAD restores the step record raw and the walk continues it.

## Rules

1. In the first game `fleeCell` reads every listed hostile and the actor through one getter, the raw position record.
2. In the second game `fleeCell` reads the hostiles and the actor at their footprint centres, as before.
3. In the first game an octant 1 or 5 step takes one off x when both new sub bytes are 0; the second game does not.
4. In the first game a rate-1 diagonal strides 0 per axis for 256 calls; the second game strides 1.
5. Crossing cells, SAV writes, the native block and the flee point read one getter; a load continues by one routine.
6. In the first game a manual cast during a loaded step leaves the step to its centre; the second game ends it.

## Code

| kind | its one builder | change |
|---|---|---|
| flee pick | `pkg/sim/withdraw.go` `fleeCell` | the hostile sum and the actor read through `fleePoint` |
| the mover's step | `pkg/sim/stride.go` `strideStep` | the octant 1/5 correction, game flag `uncorrectedStride` |
| fine position getter | `pkg/sim/stride.go` `NativeStride.finePoint` | folds `strideStep`; five former formulas read it |
| stride law | `pkg/sim/stride.go` `strideAxis` | rate-1 diagonal 0; `strideFault` accepts either law |
| SAV writes | `pkg/sim/currentmotion.go` `ProjectActorMotion` | position from `finePoint`; takes the game flag |
| manual cast | `pkg/sim/manualcast.go` `beginManualCast`, `beginCastApproach` | wait for a loaded step, flag `castDropsLoadedStep` |

## Proof

| test | rules |
|---|---|
| `TestFleeCellSumsAWideHostileAtItsAnchorPoint` | 1, 2 |
| `TestFleeCellReadsTheActorAtItsRawPositionRecord` | 1, 2 |
| `TestStrideCrossingTakesTheOctantCorrection` | 3 |
| `TestStrideFinePointFollowsTheStepLaw` | 3, 4 |
| `TestNativeStride1115EveryDirectionCapturesActualRateAndLastPayment` | 4 |
| `TestASavedStepCarriesAndContinuesTheCorrectedPosition` | 3, 5 |
| `TestSavedStride1170QualifiedDiagonalKeepsNativeBoundary` | 3, 5 |
| `TestManualCastLeavesALoadedStepToItsCentre` | 6 |

Release tests at the release: `TestReleaseWideMoverRetreatsFromItsRawPosition`,
`TestReleaseAreaLayerCostRecomputesOnTheCrossingTick`, `TestReleaseSpellClickMidStepCompletesTheStep`,
`TestReleaseSpellClickMidStepSurvivesSAVE`, on EN and RU (owed, not run).

## Open debt

- After a rate-1 diagonal's zero-stride calls the engine lands the mover on the destination centre; the original's
  next walk call is not traced (`MOVE-200` Medium). DIV-348 keeps it with the unrated transit's step record.
