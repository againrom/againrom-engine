# The game profile switch finished

## Intent

No production code outside `pkg/base` and the campaign service chooses by
game. `ProfileDebt` in `internal/archtest/gameprofile.go` is empty, and
`base.Edition` holds no bool that picks one of two code bodies. Nothing a
player sees or a save holds changes. No CHANGELOG line: nothing is
player-visible. Base: public main `a2bd04f6` (release 0.110.0); the knowledge
pin stays.

## Authority

Owner decision 4 on the architecture audit `pipeline/reviews/arch-audit-34ae6dac.md`,
section 4: the audience flag and the save's game checks become campaign-service
answers; each bool field becomes the datum its two bodies differ by. No ROM1 or
ROM2 behaviour changes, so no claim is cited and no divergence row is added.

## As built

The kind is "game profile": one `base.Edition` per game and one campaign
service per game, picked by `campaignOf`. The story adds no profile type and no
per-game method outside the campaign service.

### Edition fields

Removed: the bools `NewGameInTown`, `TownDifficulty`, `FreshPlayers`,
`SecondMaps`, `SecondScripts`, `SecondTable`, `SecondMissionText`,
`SecondMenu`, `SecondUnitKeys`, `SecondSpellArms`, `StartupCutscenes`; the
identity field `CutsceneArchive`; the string `MissionTipText`.

| New field | First game | Second game | Replaces |
|---|---|---|---|
| `Cutscenes map[string]CutsceneArchive` | `video4` and `video8` to `Allods/video4.res` and `Allods/video8.res`, logos of `video8` from `video4` | both routes to `video.res`, named `video` | `CutsceneArchive`, the archive name tests in `cutscene.go` |
| `DefaultCutscenes string` | empty: no bank | `video4` | `CutsceneArchive != ""` in `FrontEnd.App` |
| `StartupCutscenes []string` | the three logos, then `intro/01`..`intro/99` | none | the bool `StartupCutscenes` and the list in `ui.PlayStartupCutscenes` |
| `MissionTip func(mission, n int) (string, bool)` | `main/text/battle/m%d/tips%02d.txt` | no tip | `MissionTipText` and its empty test |
| `ModFamily []string` | `rom1` | none | the mods' `"rom1"` prefix rule |

`base.AppliesTo(id)` is a base id and the family words of the edition of the
profile carrying that id. `pkg/mod` now imports `pkg/base` (one DAG edge in
`internal/archtest/dag.go`).

### Campaign-service answers

| Answer | First game | Second game | Replaces |
|---|---|---|---|
| `files() *gameFiles` | `firstGameFiles` | `secondGameFiles` | `SecondMaps`, `SecondScripts`, `SecondTable`, `SecondMissionText`, `SecondMenu`, `SecondUnitKeys`, `SecondSpellArms`, `FreshPlayers` |
| `eventTags() eventTags` | `firstEventTags` | `secondEventTags` | `EventAudience.SecondGame` |
| `newGameLimit(p)` | the mission line | the town line | `NewGameInTown` in `BaseLines` |
| `captureDifficulty(f, s, onMap)` | none | the session's own off the map | `TownDifficulty` in `Snapshot` |
| `recordDifficulty(s, record)` | none | a town save's difficulty | `TownDifficulty` in `projectCurrentSession` |
| `ownsGame(g) error` | refuses a game other than the first | refuses a game other than the second | `base.SameGame` and `Game.Known` |
| `censusRefusal() error` | refuses | admits | `SecondMaps` in `SecondGameCensus` |

`gameFiles` (`pkg/game/gamefiles.go`) holds the readers as function values:
the definition table layout (`databin.ROM1Layout`, `ROM2Layout`), the map
decoder (`alm.Open`, `alm.OpenROM2`), the script compiler, the menu loader, the
briefing and event text readers, and the three values the definition table
carries for the placement decoder.

### pkg/mapload

`Table.Game` is gone. `Table.Edition *base.Edition` carries the edition for the
caller; the package reads none of it. The decoder reads three values the
caller sets (`readUnder` in `pkg/game/table.go`):

- `UnitKeys`: `ClassUnitKeys` or `ServerUnitKeys`, each with its `resolve` and
  `cheatPlacement` body; nil is `ClassUnitKeys`.
- `SpellArms`: nil or `SecondGameSpellArms`.
- `FreshPlayers`: `SlotPlayers` or `NoFreshPlayers`; nil is `SlotPlayers`.

A table built without them keeps the first game's behaviour, as `Game` empty
did.

### Former sites

| Site | Reads now |
|---|---|
| `cmd/terraintool/main.go` `mapOpener` | `game.MapOpener(profile game)`: `files().openMap` |
| `pkg/game/base.go` `BaseLines` | `campaign().newGameLimit` |
| `pkg/game/companionreport.go` `secondGameAudience` | `campaign().eventTags()` |
| `pkg/game/currentsave.go` `ExportCurrentSave` | `campaign().ownsGame` |
| `pkg/game/currentscriptbindings.go` | `tableFiles(table).compileScript` |
| `pkg/game/currentsecondcampaign.go` `validateAuthoredGame` | `campaignOf(game).ownsGame` |
| `pkg/game/currentsession.go` `projectCurrentSession` | `recordDifficulty` |
| `pkg/game/currentsession.go` `validateCurrentSession` | `validateSession`, whose first-game body refuses another game |
| `pkg/game/cutscene.go` (3) | `Edition.Cutscenes` |
| `pkg/game/eventtext.go` `accepts`, `EventPartSpeaker` | the audience's `eventTags` |
| `pkg/game/eventtext.go` `MissionTipPath` | `Edition.MissionTip` |
| `pkg/game/frontend.go` `NewFrontEnd` | `files().loadMenu` |
| `pkg/game/frontend.go` `App` (2) | `Edition.Cutscenes`, `DefaultCutscenes`, `StartupCutscenes` |
| `pkg/game/mapload.go` `LoadMapViewerFor` | `MapOpener` |
| `pkg/game/mission.go` `StartMission`, `startMissionFromWith` | `tableFiles(t).openMap`, `compileScript` |
| `pkg/game/questobjectives.go` `MissionObjectivesFor` | `files().briefing` |
| `pkg/game/resume.go` `Snapshot` | `captureDifficulty` |
| `pkg/game/secondcampaign.go` `dialogueBody` | the screen's `tags`, given by `townScreen` |
| `pkg/game/secondcensus.go` `SecondGameCensus` | `censusRefusal` |
| `pkg/game/secondgametext.go` `ReadEventTextFor` | `files().eventText` |
| `pkg/game/table.go` `databinLayout` | `files().table` |
| `pkg/mapload/cheatfactory.go` `CheatActor` | `Table.UnitKeys` |
| `pkg/mapload/currentplayers.go` | `Table.FreshPlayers` |
| `pkg/mapload/spawn.go` `Resolve` | `Table.UnitKeys` |
| `pkg/mapload/spell.go` `SpellRules` | `Table.SpellArms` |
| `pkg/mod/order.go` `Applies` (2) | `base.AppliesTo` |
| `pkg/game/mods.go` `BaseID` fallback | `base.Undetected().ID`: the profile pkg/base gives an install it did not identify |
| `pkg/game/campaignservice.go` `tableGame` | `tableEdition`: the table's `Edition` |

The scan shapes, `ProfileAllowed` and `CheckProfile` are unchanged.

## Proof

- `internal/archtest` passes with `ProfileDebt` empty.
- No golden, trace, witness or expected value is edited. Test construction
  edits: second-game tables built with `readUnder(base.GameROM2, ...)` or the
  three mapload values instead of `Game: base.GameROM2`; `EventAudience{tags:
  secondEventTags{}}` instead of `SecondGame: true`; `tableEdition(t).Game`
  instead of `t.Game`; `SecondGameSpellArms` for `secondGameRules`; three
  release tests skip on `Campaign == base.CampaignDestinations` instead of
  `NewGameInTown`; the `ui` startup test names its movies with
  `SetStartupCutscenes`.
- Release filters, scenarios and gates: the lane return lists each with its
  count and result.

## Open debt

- `Table.Edition` still carries the edition through pkg/mapload for the
  game package's mission code, which reads its campaign and text code page
  from the mission's table.
- `CommentBytes` rose for the new code's docs (`internal/storyguard/baseline.go`).
