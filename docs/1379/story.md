# One game profile

## Intent

The game a session runs is chosen once, by the profile `pkg/base` detects
from the install. What differs between the two games is either a field of
the profile's edition or a method of one campaign service with a first-game
and a second-game implementation, picked from the edition. Nothing a player
sees, hears or saves changes. A third profile or a mod profile is an edition
value and, at most, one more service implementation; it adds no arm to shared
code.

Base: `af6c6a09` (game 0.105.0), reconciled with main `6f05571d`. Knowledge
pin: k208.

## Authority

- Owner: one game profile, or at least one service; one builder per kind;
  customisation is a design axis (G2).
- Seat decision: one profile value resolved at detection; per-game data on
  the profile; one campaign service for behaviour that is not data; no game
  comparison outside the profile package and the service, held by an
  `internal/archtest` ratchet; the town description read through the
  profile.
- Architecture audit on `af6c6a09`, row 2. No ROM1 or ROM2 fact is asserted;
  every moved branch keeps its claim and divergence citations.

## As built

### The profile is resolved once

- `OpenArchives` detects the base when it opens a root and keeps the match on
  `Archives.Base`; `Archives.Game` reads it and no longer re-detects.
- `OpenCutscenes(root, archive)` no longer detects the base.
  `FrontEnd.OpenCutscenes(archive)` applies the profile's cutscene archive;
  `cmd/againrom` calls it.
- The zero profile is the first game, as `Profile.GameOf` already read it.

### Edition fields

`pkg/base/edition.go`. `Game.Edition()` and `Profile.Edition()` select it.

| field | first game | second game | sites it replaced |
|---|---|---|---|
| `SaveTag` | empty | `rom2` | `Snapshot` (`resume.go`) |
| `Campaign` | chapters | destinations | the service picker `campaignOf` |
| `Town` | `rom1` | empty | `rom1Town` global: `tips.go`, `townscreen.go` (square view tip), `townservices.go` (bind), `townsquare.go` (square view, music track), `installshare.go` (square art) |
| `NewGameInTown` | no | yes | `BaseLines` (`base.go`) |
| `TownDifficulty` | no | yes | `Snapshot`, `projectCurrentSession` |
| `CompanionObjectiveMission` | 40 | 0 | `StartMissionFrom` companion VIP rule |
| `Cheats` | yes | no | `CheatItem`, `CheatActor` (`pkg/mapload`), `chatCommand`, `debugLetter` |
| `FreshPlayers` | yes | no | `initializeCurrentPlayers` (`pkg/mapload`) |
| `SecondMaps` | no | yes | `StartMission`, `LoadMapViewerFor`, `SecondGameCensus`, `cmd/terraintool` |
| `SecondScripts` | no | yes | `StartMissionFrom`, `restoreCurrentScriptBindings` |
| `SecondTable` | no | yes | `databinLayout` |
| `SecondMissionText` | no | yes | `MissionObjectivesFor`, `ReadEventTextFor`, `reportWordsFor` |
| `SecondMenu` | no | yes | `NewFrontEnd` menu art |
| `SecondUnitKeys` | no | yes | `mapload.Resolve` |
| `SecondSpellArms` | no | yes | `mapload.SpellRules` |
| `CutsceneArchive` | empty | `video` | `OpenCutscenes`, `FrontEnd.App` |
| `StartupCutscenes` | yes | no | `FrontEnd.App` |

`Game.Known`, `Game.Normal` and `SameGame` replace the identity checks of
`validateCurrentSession`, `ExportCurrentSave` and `validateAuthoredGame`.

### The campaign service

`pkg/game/campaignservice.go` declares `campaignService` and the one picker,
`campaignOf(game)`, which reads `Edition.Campaign`. A session reaches it
through `FrontEnd.campaign()` (the install's profile), a mission through its
table's game (`missionNotices.campaign()`), a decoded save through its session
record (`sessionCampaign`), and a mission entry through `missionPorts.campaign`.
Both implementations are stateless.

- `firstCampaignRules` (`campaignfirst.go`): the first game's bodies of every
  moved branch.
- `secondCampaignRules` (`campaignsecond.go`): the second game's bodies. It
  embeds the first-game rules; where the replaced code tested the second
  campaign's state for nil and fell back to first-game behaviour, the method
  does the same.

| method | replaced |
|---|---|
| `townScreen` | `TownScreen` profile test; first-game body is `chapterTownScreen` |
| `townSavePoint`, `captureMission` | `Snapshot` town save point, map animation, motion and open notice |
| `exportable`, `refuseCompleted`, `campaignProjection` | `ExportCurrentSave` second-game gate and completed-campaign refusal; the campaign record in `currentMissionDocument` and `currentCityDocument` |
| `validateSession`, `validateAuthored`, `undecodedSave`, `statesCampaign` | `validateCurrentSession`, `validateAuthoredGame`, `validateOriginalGame`, `decodeOriginalCampaign` |
| `canEnter`, `enterMission`, `seatWorld` | `admitMission`, `activateLive`, `seatWorld` |
| `selectedMarkers`, `checkTownLoad`, `arriveLoaded`, `keepsTownSurface` | `RestoreOriginal`, `restoreOriginalTown`, `installCandidate`, `resetTownSurface` |
| `beforeAdvance`, `finishWon`, `route`, `returnToTown` | the four `frontTransitions` second-campaign arms |
| `openMission`, `loadMissionText`, `scriptMessages`, `settleNotices`, `acknowledgeNotice`, `outcomeText`, `eventAudience`, `objectives` | the table test in `openMission` and the helper `missionNotices.secondGame` with its seven callers in `world.go`, `secondgamenotices.go`, `companionreport.go`, `questobjectives.go`; its two callers in `cheats.go` read `Edition.Cheats` |

The second game's script-message queue, notice settling, audience and
objectives panel stay second-game logic (`observeSecondGameMessages`,
`settleSecondGameNotices`, `secondGameAudience`, `secondGameObjectivePanel`)
and are reached only through the service. `secondCampaign.noticePending`
replaces the snapshot's inline state test.

### The ratchet

`internal/archtest/gameprofile.go` type-checks every production file outside
`pkg/base` for three shapes: a use of `base.GameROM1` or `base.GameROM2`, an
`==` or `!=` with a `base.Game` operand, and an `==` or `!=` against nil of a
field named `second` or `Second`. Allowed: `campaignfirst.go` (2 findings) and
`campaignsecond.go` (17). Debt: none.

| measure, same scan | before (`af6c6a09`) | after |
|---|---|---|
| names a game | 46 | 0 |
| compares a game | 43 | 0 |
| tests a second-campaign state for nil | 24 | 0 |
| files with a finding | 29 | 0 |

## Proof

- Profile witness (`profilewitness_release_test.go`), recorded on the
  unchanged code in `pkg/game/testdata/profilewitness/`: from the main menu, a
  new game to the first mission (the second game through its town, tavern
  talk and gates), a SAVE in the town (second game) and on the map, a LOAD of
  each in the session and a LOAD from a fresh start. Each step records the
  screen, the frame hash (the map window composed on the CPU around the
  party), the World hash and tick, and the SAV hash. EN 14 lines, RU 14
  lines, ROM2 RU 30 lines: identical after the change.
- Town square trace: identical on EN and RU.
- Release, scenario, ROM2 save and milestone-2 gates: see the lane return.

## Open debt

- The service has 29 methods. The mission-driver methods could become a
  narrower mission service when a third campaign needs one.
- `mapload.Table.Game` carries the game identity into mission code; it is set
  from the profile when the table is loaded.
- `EventAudience.SecondGame` is a per-audience flag the service sets; the
  dialogue tag reader branches on it.
- `ROM1TownDescription` and `TownTipPath` remain first-game exports for the
  town square tool and tests.
- The second game's town has no square description, so no square art is
  loaded for it; nothing on that game read it.
