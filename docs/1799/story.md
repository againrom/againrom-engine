# A computer Swarm group drops a victim it no longer scores, and the rest of group behaviour is pinned

## Result

A computer unit in a Swarm group that already stands on the cell the group was sent to now lets go of the enemy it was
fighting when its group stops scoring that enemy. Before, it kept fighting until the enemy died. This holds in both
games. How groups guard, patrol and pick targets already matched the research; those rules now have tests, and every
part the research leaves open is in the ledger with the original run that settles it.

## Authority

- Owner: one order, group and AI machine per game through one builder per kind; every row built or in the ledger.
- `AI-908`, `AI-910` (High / Medium): first game, a member with no target has its order byte rewritten to idle or walk.
- `R2-ENGINE-844`, `R2-ENGINE-1985` (High / Medium): second game, the same tail for a computer owner (Medium).
- `AI-905`, `AI-741`, `AI-904` (High / Medium): first game, member walk order, radius latch, Guard pass counter.
- `R2-ENGINE-1984`, `R2-ENGINE-843` (High body): second game, member walk order and the radius latch stores.
- `AI-909`, `R2-ENGINE-1992` (High): the least score wins, the first on a tie, no memory of the old target.
- DIV-5363 to DIV-5368 written; DIV-5369 to DIV-5372 unused. DIV-1523 and DIV-5268 stand.

## Rules

1. Both games, Medium: a computer Swarm member on its commanded cell with no assigned target releases its victim.
2. Both games, High / Medium: a group pass takes members head to tail through the group list, each member in full.
3. First game High / Medium, second game High: the Guard radius rolls once when the latch flips and the counter counts.
4. Both games, High: the Guard clip uses the current centre and stored radius; a member with no winner walks home.
5. Both games, High: each member takes the lowest score, the first candidate on a tie.

## Code

| kind | its one builder | change |
|---|---|---|
| group engagement | `pkg/sim/savedgroupsai.go` savedDecision | Swarm member on its cell releases its victim |
| Guard arm | `pkg/sim/guardarm.go` armGuard, postEngage | unchanged, matches the claims |
| group programme routes | `pkg/sim/grouproute.go` groupRouteProfile | unchanged, matches the claims |
| saved group state | `pkg/sim/groupwords.go` setGroupWords, latchWords | unchanged, matches the claims |
| target choice | `pkg/sim/targetchoice.go` chooseTarget | unchanged, matches the claims |

## Proof

| test | rules |
|---|---|
| `TestASwarmMemberOnItsCellDropsAVictimNothingScores` | 1 |
| `TestAGroupPassWalksItsMembersHeadToTailThroughTheGroupList` | 2 |
| `TestLiveGuardRollsItsWorkingRadiusOnceTheLatchFlips` | 3 |
| `TestSecondGuardRadiusRollsOnceAndSurvivesAColdLoad` | 3 |
| `TestLiveGuardClipsItsFirstPassAtTheBase` | 4 |
| `TestSecondGuardLeashEndsThePursuitBeyondTheStoredRadius` | 4 |
| `TestAnOffPostGuardWithNoCandidatesWalksHome` | 4 |
| `TestTargetChoiceKeepsGroupAndRepickTieRules` | 5 |

Release tests at the release: none.

## Open debt

- DIV-5363: the order of players, groups and members. DIV-5364: the second game's radius draw divisor.
- DIV-5365: a blocked Move destination. DIV-5366: the idle tail's helpers. DIV-5367: slot 0x17. DIV-5368: a second pass.
