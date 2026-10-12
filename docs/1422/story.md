# A mod map replaces a campaign mission

## Intent

A mod declares that its map replaces a first-game campaign mission by number.
When the campaign reaches that mission by ordinary play, the mission is built
from the mod's map; everything else of the campaign stays the install's. The
map editor saves into a mod, closing the editor-to-play loop without a
hand-written path. The story started at public main `0129b0a5`, pin k272; the candidate merged main `165270ed`, pin k275.

## Authority

Owner direction; no ROM1 research. Owner rulings: a mod map is an ALM map; it
plays as an additional mission (the guest mission), a replaced campaign
mission, a standalone mission or a map for another base; a mod may carry a
whole map. This story builds the replaced campaign mission. Rows: `DIV-3141`
(the replaced mission and its rules), `DIV-3142` (its SAV).

## As built

### One mission table, two row kinds

`data/missions.toml` keeps one `[[mission]]` table. `kind` is `"guest"`
(default; every guest declaration parses as before) or `"replace"`
(`mod.MissionGuest`, `mod.MissionReplace`). A replacing row takes `key`,
`number`, `map` and the optional `texts`, `title`, `briefing`; the five guest
keys are refused with file and line. `MissionData.AfterWin` and `ByNumber`
answer guests only; `Replacing(n)` answers the replacing row of mission n.

### One resolver, one start

`FrontEnd.SetModMissions` is the one resolver for both kinds. A guest row is
checked as before; a replacing row by `checkReplacing`: a number in the
campaign's `Main`, `Side` or `Offered`, an installed
`scenario/<number>.alm`, and a map `MapOpener` decodes. Both kinds are then
served by the same overlay loop on the front end's forked filesystem: map at
`scenario/<number>.alm`, texts, title and briefing under
`main/text/battle/m<number>/`. The mission starts through the one
`enterMission`, which reads the map by address; nothing in the start knows
the row kind.

### Rules

| case | rule |
|---|---|
| two mods replace one mission | the second is refused at load (`pkg/modrt`: one number, one mod) |
| a replacement and a guest of another mod | cannot collide: a guest number is never a campaign number; a guest after a replaced mission plays after its win |
| a number the install lacks | refused at start-up: no campaign mission, or no installed map |
| a SAVE inside a replaced mission | the ordinary SAV with the mod mark; LOAD without the mod refuses |
| an unmarked save of a replaced mission | refused while the mod is active, even with `-mods-accept-unmarked` (`refuseUnmarkedReplaced`) |
| a map the build refuses | refused at start-up with file and line |

### Editor save into a mod

`mapedit -mod <mod folder>` (`MapInspector.SetMod`) reads the mod's
`data/missions.toml` in the install's language. Ctrl+S (the `saveas` key)
then calls `SaveIntoMod` instead of asking for a path, and the help line reads
`Ctrl+S: save into mod <id>`. The entry for the opened map is the row whose
`number` is the installed `scenario/<n>.alm`, or whose `map` is the opened
file. With no entry the save is refused and the message stays on screen. The
write goes to a dot-named file beside the target and is renamed over it, stays
inside the mod folder and outside the install and its preserved pair
(`fenceEditorWrite`, shared with Save As), and refuses a map over the 1 MiB a
mod may load. Without `-mod`, Save As is unchanged.

## Proof

- `pkg/mod`: `TestParseMissionsReadsAReplacingMission`,
  `TestParseMissionsReplacingRefusals` (loss control: the same table without
  `kind` is a guest).
- `pkg/modrt`: `TestTwoModsCannotReplaceOneMission`.
- `pkg/game`: `TestSetModMissionsServesAReplacingMapOverTheInstalledMission`,
  `TestSetModMissionsRefusesAReplacementTheInstallDoesNotBearOut`,
  `TestAnUnmarkedSaveInsideAReplacedMissionIsRefused`,
  `TestEditorSavesIntoTheModMissionPath`,
  `TestEditorRefusesAModThatNamesNoEntryForTheMap`. The guest tests run
  unchanged.
- `pkg/ui`: `TestSaveGoesIntoTheModTheHostNames`,
  `TestSaveAsAsksForAPathWithoutAMod`.
- Release, EN and RU: `TestReleaseModReplacesACampaignMission`. The test copies
  the installed mission 20 map and texts into a temporary mod, deletes one
  interior creature through the editor and saves with Ctrl+S into the mod,
  plays mission 10 and reaches mission 20 by the ordinary route, and asserts
  the built map and World lack that unit; SAVE and a cold LOAD under the mod
  keep it; LOAD without the mod refuses; an unmarked save is refused under the
  mod with unmarked saves accepted. Loss control: the same route without the
  mod builds the installed map with the unit.

## Open debt

- The standalone mission waits for multiplayer; the map for another base
  waits for the second game's track.
- A replaced mission keeps the install's world-map marker and scenario rows;
  a mod cannot yet change them.
- The replaced mission is built and proven on the first game only. The rule
  reads the install's campaign data, so a second-game mod would pass the same
  checks; its play there is Unknown.
