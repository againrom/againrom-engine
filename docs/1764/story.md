# Both games follow the answered fidelity checks

## Result

The first game's Fire Sacrifice, Acid Stream and Meteor Storm now hit their first cells on the step after the
cast and every third step after, as the second game's staged areas already did. A second-game mage's automatic
support reads its target setting against the relation byte, so Allies alone takes the player's own units.

A second-game Human, placed or hired, keeps the zero back-off and flee thresholds it is built with; only a
player's Retreat setting writes them. The other checks match or keep the engine under a named divergence row.

## Authority

- Owner: «Эпики надо довести до завершения»; «сделать как в оригинале» for every check of the epics «Пять
  проверок» and «Восемь проверок».
- MAGIC-463 (Medium): a staged area built in a step's actor pass is first driven by a later step's area pass,
  stage k on relative step 1 + 3k; R2-ENGINE-1789 (Medium): Acid Stream and Blizzard both attempt first on area
  update 1.
- R2-ENGINE-1793 (High bodies, Medium play), R2-ENGINE-1794: the affinity tests no low bit with Neutral, the ally
  bit with Allies, the same owner with Own; the map diagonal is 0x12; a hostile bit keeps a unit out.
- R2-ENGINE-1795 (High bodies, Medium): a Human built by the hire or the map arm reaches no threshold setter, so
  its thresholds are 0 after creation.

## Rules

1. When a staged area is inserted in either game, it paints nothing; stage k runs on the (1 + 3k)th area update.
2. When a second-game support candidate is not hostile, Neutral takes no low bit, Allies the ally bit, Own the owner.
3. When a first-game support candidate is tested, the owner, ally and Neutral order stays.
4. When a mission starts in either game, no unit's Withdraw or Wimpy is rewritten.

## Code

| kind | its one builder | change |
|---|---|---|
| area-spell stage driver | `pkg/sim/areaarms.go` areaArms, ringStageDelay | the first-stage delay moves from the second-game arm row to the staged arm row |
| support candidate test | `pkg/sim/autocastsetting.go` autoCastCandidates, supportAffinities | the target setting reads through a profile rule |
| game profile | `pkg/sim/gameprofile.go` supportAffinity | second game: the relation byte |
| unit creation path | `pkg/sim/world.go` world construction | the class threshold pass is removed |

## Proof

| test | rules |
|---|---|
| `TestFirstStagedAreaFirstStageWaitsForTheNextAreaPass` | 1 |
| `TestSecondAcidStreamFirstStageWaitsForTheNextAreaPass` | 1 |
| `TestAreaSaveRingCutsContinueWithoutReplay` | 1 |
| `TestFlightRecordsMatchTheirPinnedRuns` | 1 |
| `TestSecondSupportCandidatesFollowTheRelationByte` | 2, 3 |
| `TestTheSecondGameSupportCandidatesTakeWhatTheSideSees` | 2 |
| `TestMissionStartKeepsEveryBuiltThreshold` | 4 |

The flight witness digests `first/staged` and `second/staged` move: the staged burst's first stage hits one tick
later. Evidence: the lane's review report.

## Open debt

- DIV-5016: the second game's flee idle planner, which attacks the nearest hostile within cast or weapon range plus
  one when none stands in the Teleport list, is not modelled; the unit takes no action.
- DIV-1555: a first-game Wimpy unit holding command state 1 stands down with no strike in the original (AI-798,
  Medium); the engine reacquires within reach.
- DIV-5049, DIV-4906, DIV-5160, DIV-4629: the runs their claims name settle the own-cell stand-in, an indirect
  threshold write, the Teleport scroll's per-sub-tick values and the caster's place in its own Heal list.
