# Withdrawal and its refused flight follow the research in both games

## Result

In the first game a wounded computer unit that was walking a Move order and finds its flight blocked now stands still
for one pass and picks a fight on the next, as the research reads it. In the second game a wounded unit and a defender
now keep an invisible enemy in mind when a group member can see invisible as far as that enemy's nearest edge, so a
wide enemy is noticed from farther. Every other row of withdrawal and Retreat already matched the engine or is an
Unknown in the ledger with the original run that settles it.

## Authority

- Owner: one order, withdrawal and AI machine per game through one builder per kind; every row built or in the ledger.
- Owner ruling (card q-escort-corpse-load): an escort whose subject is dead stands down in live play and after a load.
- `AI-929` (Medium), `AI-658`, `AI-659` (High / Medium): first game, a unit under its Wimpy bound with no Retreat order.
- `AI-945` (Medium), `AI-337`: first game, a square holding only corpses gives a live count of zero.
- `R2-ENGINE-1879` (High body, Medium meanings), `R2-ENGINE-1878`, `R2-ENGINE-1188`: second game, invisible enemies.
- `R2-ENGINE-1787` (Medium), `R2-ENGINE-2025`, `R2-ENGINE-2028`, `R2-ENGINE-2029` (RU, Medium for the absence).
- `AI-942` to `AI-950`, `AI-946`, `R2-ENGINE-2020` to `R2-ENGINE-2027`: census and message reads, no engine change.
- DIV-1555, DIV-575, DIV-3941, DIV-5050, DIV-5275, DIV-5276, DIV-5343: narrowed or written with their settling runs.

## Rules

1. First game, Medium: a wounded unit holding an unlatched Move at command state 1 is stood down on its first refusal.
2. First game, Medium: its next refusal, and the first refusal of any other unit, picks a hostile within reach.
3. First game, Medium: a unit under explicit Retreat is never stood down and takes the pick.
4. First game, Medium: a square whose only hostiles are corpses gives no flight; the unit acquires a target.
5. Second game, Medium: a withdrawal or defender list keeps an invisible enemy a group member detects by edge distance.
6. Second game, Medium: idle and support casters cast no Teleport; it stays with flight and back-off (Russian root).
7. Second game, Medium: in a terrain pocket with another reachable cell the unit walks there and stops on its own cell.

## Code

| kind | its one builder | change |
|---|---|---|
| withdrawal | `pkg/sim/withdraw.go` withdrawalArm, withdrawalActor | unchanged, matches the claims |
| refused flee answer | `pkg/sim/withdraw.go` refusedFleeAnswers | the first-game row stands an unlatched Move down |
| refused route answer | `pkg/sim/refusedroute.go` answerRefusedRoute | unchanged |
| player's Retreat | `pkg/sim/playerretreat.go` retreatOrder, armRetreat | unchanged |
| retreat continuation | `pkg/sim/retreatcontinuation.go` stepRetreatExecutor | unchanged |
| invisible enemy in a list | `pkg/sim/sight.go` detectMeasures | both games read edge distance; profile field removed |

## Proof

| test | rules |
|---|---|
| `TestFirstGameRefusedWimpyMoveStandsDownWithoutStrike` | 1, 2, 3 |
| `TestFirstGameLoneCorpseInTheSquareAcquiresInsteadOfFleeing` | 4 |
| `TestSecondGameWithdrawalAndCoverDetectWideInvisibleByEdgeDistance` | 5 |
| `TestSecondGameTeleportNotCastBySupportOrIdleRouter` | 6 |
| `TestSecondGamePocketFlightWalksToStandInAndStopsOnlyOnOwnCell` | 7 |

Release tests at the release: none.

## Open debt

- DIV-1555: the in-position gate and the original's pass order after a refusal; DIV-3941: cell slot writers.
- DIV-5275: the invisible list's identity; DIV-5276: the Russian-root-only Teleport reads; DIV-5050: byte +4 at receipt.
- DIV-5343: the original's first pass after a load with a dead escort subject. DIV-5344 to DIV-5352 unused.
