# Who a unit strikes, heals or alarms follows the research in both games

## Result

In the first game, a computer unit whose route is refused with no enemy in reach now idles in guard state and turns in
place when alarmed or on the rare idle draw. In both games, a computer mage skips flying allies in its area heal and
buff list while a wounded flying caster still heals itself, and a Darkness cloud reaching its own caster runs the same
blow reaction as on any other occupant. A defender guarding an ally now engages a closer downed enemy ahead of a farther
living enemy. In the second game, downing a neutral unit at zero health no longer triggers the dead-state flip.

## Authority

- Owner: one order, withdrawal and AI machine per game through one builder per kind; every row built or tracked.
- `AI-864` (High stores, Medium divisor), `AI-884` (High turn window): first game, refused reacquisition of a computer
  unit stores order state 0xb, leaves target untouched, and runs an idle turn on alarm or draw below 205 of 32768.
- `AI-866` (High lookup, Medium domain-3/caster), `AI-923`, `AI-943`, `AI-944`: first game, Heal list reads cell slot +4
  (domains 1 and 2), skipping domain-3 actors; the second game follows it, no claim reading it (`DIV-4629`).
- `AI-867`, `AI-829` (High loop/tail, Medium later stages); `R2-ENGINE-985`, `R2-ENGINE-1898` (High RU/EN body, Medium
  self-hook): Darkness and area blow loops pass the area caster with no caster-equals-occupant exclusion.
- `AI-872` (High bodies, Medium composed), `R2-ENGINE-1878`, `R2-ENGINE-1944` (High RU bodies, Medium units/EN): cover
  ranking loads no health or life test for units still in a cell record.
- `R2-ENGINE-1943` (High RU immediate stores, Medium overall/EN): second game, state 0x10 is stored at HP -10 or below
  and death finish; downed units at HP 0 or below hold state 0.
- `AI-862`, `AI-863`, `AI-865`, `AI-868..871`, `R2-ENGINE-1940..1942`, `R2-ENGINE-1944`: confirmed matches; unknowns in
  DIV rows.

## Rules

1. ROM1 (`AI-864` High/Medium, `AI-884` High): refused reacquisition writes 0/0xb and turns AI (alarm or <205/32768).
2. Both games (`AI-866` High/Medium; ROM2 `R2-ENGINE-2003` Medium): computer Heal/buff list skips domain 3; self heals.
3. Both games (`AI-867` High/Medium; ROM2 `R2-ENGINE-985`/`1898` Medium): Darkness runs its blow hook on the caster too.
4. ROM2 (`R2-ENGINE-1943` High RU / Medium overall): area-hit state 0x10 needs HP <= -10, not downed HP 0.
5. Both games (`AI-872` High/Medium; ROM2 `R2-ENGINE-1878`/`1944` High/Medium): cover ranks downed 0..-9 with living.
6. Both games: targeting, support, Darkness self-blow and cover changes keep identical replay and round-trip hashes.

## Code

| kind | its one builder | change |
|---|---|---|
| refused route answer | `pkg/sim/refusedroute.go` `answerRefusedRoute` / `answerRefusedNoneByRow` | write `ord+8` 0/0xb and call `stepIdleTurn` for first-game computer units |
| computer support Heal list | `pkg/sim/autocastsetting.go` `supportCast` / `areaListOrder` + `pkg/sim/gameprofile.go` | filter domain-3 actors by `supportCastPolicy.groundSlotOnly` |
| area blow hook | `pkg/sim/celleffect.go` `applyAreaCells` + `pkg/sim/savedcontinuation.go` `applySavedAreaPayload` | call `answerAreaBlow` without excluding the caster |
| area-hit reverse hostility | `pkg/sim/world.go` `flipOnBlow` / `reverseHostileOnAreaGate` | read state 0x10 as `!OrdinaryTargetable()` (`HP <= -10`) |
| defender cover | `pkg/sim/escort.go` `coverEngage` + `pkg/sim/targetchoice.go` `chooseTarget` | remove `LifeDecaying` split before `chooseTarget(..., targetCover)` |

## Proof

| test | rules |
|---|---|
| `TestFirstGameRefusedReacquisitionStoresOrderStateAndTurnsComputerUnit` | 1 |
| `TestComputerHealListExcludesDomainThreeActorsInBothGames` | 2 |
| `TestDarknessPulseRunsBlowHookOnCasterInBothGames` | 3 |
| `TestSecondGameAreaHitStateSixteenRequiresTerminalHealth` | 4 |
| `TestSecondGameAreaHitFlipsReverseInDeadState` | 4 |
| `TestCoverEngageRanksDownedHostileWithoutHealthSplitInBothGames` | 5 |
| `TestTargetingAdoptionReplayHashAndBinaryRoundTrip` | 6 |

Release tests at the release: `TestReleaseMission121BridgeTrollKeepsItsVictimWhenItsRouteIsRefused`,
`TestReleasePrismaticSpraySecondariesFollowTheGroupsSightAndRank`,
`TestReleaseAnOrderOntoAnUnreachableCreatureTakesTheHostileBesideTheHero`.

## Open debt

- First-game unknowns: `DIV-1272`, `DIV-1316`, `DIV-1520`, `DIV-1600`, `DIV-4629`, `DIV-5139`, `DIV-5266`, `DIV-5333`,
  `DIV-5334`.
- Second-game unknowns: `DIV-4671`, `DIV-4767`, `DIV-4842`, `DIV-4843`, `DIV-4909`, `DIV-4912`, `DIV-4913`, `DIV-4917`,
  `DIV-5096`, `DIV-5135`, `DIV-5136`, `DIV-5215`, `DIV-5273..5276`, `DIV-5335`, `DIV-5336`.
- Folding `joinEngagedGroup` and `coverEngage` rankers into `chooseTarget` belongs to later story 9 of the cut.
