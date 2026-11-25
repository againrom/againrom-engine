# A second-game Outpost brings its group back and script action 31 sets its countdown

## Result

On a second-game map that sets up Outposts, every computer group now has an Outpost that watches it. When the
whole group has fallen, the Outpost counts down and then brings the group back as it stood at the mission start;
in the Russian build the members come back around the Outpost, in the English build on their start cells.
A map script can now set that countdown, which before did nothing. A saved game stores no Outpost, and the
Outposts of a loaded game bring no group back. Campaign maps hold no Outpost and play as before.

## Authority

- R2-ENGINE-509, R2-ENGINE-1816: the type-10 pass runs at the map build when type-0 dword 0x84 is above 1.
- R2-ENGINE-510, R2-ASSET-120, R2-ENGINE-511: one Outpost per computer group; entries; the Russian spread.
- R2-ENGINE-1817, R2-ENGINE-1818: slot 5 per container pass, the countdown and the re-creation.
- R2-ENGINE-1701, R2-ENGINE-1820: action 31 writes the countdown of every matching Outpost of every owner.
- R2-SESSION-502, R2-SESSION-503: no Outpost field in the SAV; the state a load builds.
- R2-ASSET-130: every root map opens the gate, every campaign map keeps it shut.
- DIV-5103 narrowed; DIV-5184 (return cell); DIV-5185 (loaded Outpost).

## Rules

1. When a second-game map's word 0x84 is above 1, the start builds one Outpost per computer group with members.
2. When the gate is shut or the game is the first, no Outpost is built and the World's form is unchanged.
3. When no member stands below decay stage 3, a countdown V starts and the group returns on pass V+1, not before.
4. When the group still lists a member as the countdown ends, nobody returns.
5. When the build keeps the spread, members return within the Outpost's radius around its centre.
6. When action 31 runs, every Outpost whose group is p0 takes countdown p1, whatever its owner.
7. When a game is saved, no Outpost field is written; a LOAD holds each Outpost empty at group 0, doing nothing.
8. When the installed maps are read, no campaign map holds an entry; every root map does and builds Outposts.

## Code

| kind | its one builder | change |
|---|---|---|
| map record decoder | `pkg/formats/alm/rom2.go` Map.OutpostEntries | type-10 records, beside StructureActions |
| type-10 pass input | `pkg/mapload/outpost.go` Outposts | gate, spread, entries; called by startMissionScripted |
| per-build data | `pkg/base/base.go` Profile, `pkg/game/table.go` LoadDefinitionsOf | OutpostSpread for the Russian build |
| game profile row | `pkg/sim/gameprofile.go` newGameProfile | the second game selects Outposts |
| Outpost, its pass and form | `pkg/sim/outpost.go` DeclareOutposts | new kind; pass in runNativeTickPhase; form section 146 |
| actor and group re-creation | `pkg/sim/createactor.go` createActor, `pkg/sim/carriers.go` constructedGroup | ungrouped profile |
| retained actor copy | `pkg/sim/passivebody.go` passiveHoldings | pulled out of retainPassiveBody; templates use it |
| script, editor and LOAD rows | `pkg/sim/rom2script.go`, `pkg/scriptprofile/ops.go`, `pkg/game/originalfilesource.go` | action 31 |

## Proof

| test | rules |
|---|---|
| `TestOutpostTypeTenPassBuildsOneOutpostPerComputerGroup`, `TestMissionStartBuildsTheMapsOutposts` | 1 |
| `TestOutpostsDecodeTheTypeTenEntries` | 1 |
| `TestOutpostShutGateAndFirstGameBuildNone` | 2 |
| `TestOutpostGroupComesBackOnPassCountdownPlusOne`, `TestOutpostCountdownStartsAtDecayStageThree` | 3 |
| `TestOutpostWaitsWhileTheGroupListsAMember` | 4 |
| `TestOutpostWithSpreadPlacesAroundItsCentre` | 5 |
| `TestOutpostActionThirtyOneSetsMatchingCountdowns`, `TestSecondGameAction1IncrementsRegister902AndGapsRemain` | 6 |
| `TestSecondMissionOutpostSavesNothingAndLoadsEmpty`, `TestOutpostLoadedStateHoldsNoMember` | 7 |
| `TestReleaseSecondOutpostCensusAndRootMapPass` | 8 |

Release tests at the release, EN and RU: `TestReleaseSecondOutpostCensusAndRootMapPass`.
Evidence: `review/story1769-rom2-outpost/RESULT.md`.

## Open debt

- The cell a member takes at radius 0 is Unknown; the start cell stands in, then the nearest free cell (DIV-5184).
- A loaded Outpost's countdown, owner and regrouping are Unknown; it stays inert (DIV-5185).
- The +0xb8 action, the Inn offer groups, the per-member callee and the scale key are not carried (DIV-5103).
- The Outpost pass runs after the world pass in the same tick; the original's order of the two is Unknown.
- No shipped campaign map opens the gate; root maps reach the pass only through authored maps and the census.
