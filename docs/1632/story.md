# A map file named as an install map replaces that map

## Intent

A mod replaces a campaign mission's map by holding the map under the mission's number, `<N>.alm` or `<N>.almx`, in its top folder or in `maps/`, with no `[[mission]]` table and no script line. A mod of `mod.toml` and `10.alm` is a valid mod. For the game it is the replacement a `kind = "replace"` table already makes.

## Authority

Owner direction on the board: «в моде хочется иметь наверное такой базовый функционал: если в корне, или где-то еще папки мода лежат карты с названиями 10.alm и тд, которые имеют counterpart в оригинале, то просто делаем подмену этих карт»; on the form: «Да, корень и maps/». The six rules of the file form are the seat's, from the owner's words, in the story brief. The mod format is Againrom's own: no ROM1 claim applies and no DIV row is used.

## As-built behaviour

Rules, as `docs/MODS.md` "Replacing a campaign mission" states them:

1. Only the top folder and `maps/` are read, through the mod tree, so a packed mod reads as its folder. A name is a mission number of 1 to 65535 with no leading zero and `.alm` or `.almx`, matched without case (`missionMapName`).
2. A file a row of `data/missions.toml` names as its `map` (compared by `namefold.ModTree`) is that row's map and declares nothing else; a guest row's map stays a guest map.
3. A row of the same mod and number beside an unnamed numbered file, and two files of one number (top and `maps/`, or `.alm` and `.almx`), are refused naming both files.
4. A file whose number has no counterpart replaces and refuses nothing, also when its map or page folder would be refused; the launch and `-check` print one `warning:` line naming the file and the reason.
5. One file serves both languages: the path holds no `{lang}`.
6. `-check` prints one line per replaced mission, for the file form and for the row form: `againrom: mission 10 replaced by mod "m": map 10.alm, named by its file name`, or `..., declared at data/missions.toml:4`.

The map is read as a row's map: `Mission.ReadFiles`, which also reads a page folder beside the map (`10.text/`). Without one the install's pages, title and briefing stay. The mod enters a save's mod mark when it holds a numbered file, decided before the install is asked; a mod whose only numbered file has no counterpart is therefore marked too. The mark's content digest covers the map, as it covers every file.

Kinds and builders:

- Mission declarations of a mod: the missions kind in `pkg/mod/kinds.go`. `ParseMissions` reads the rows unchanged; the kind gains a `named` reader, `readMissionMaps` (`pkg/mod/missionmaps.go`), that `pkg/modrt` runs for every mod after its script, beside the readers of the kinds no script loads (`readBesideManifest`, through `DataKind.ReadNamed`). The file form's declarations wait in an unexported field of `MissionData` with their place in load order.
- The counterpart test: `FrontEnd.campaignMap`, the first two tests of `checkReplacing` made one function (`installedCampaignMap`): a mission of the install's campaign whose map the install holds at `MissionMap(n)`. `checkReplacing` asks it; no second map-name rule.
- Admission: `MissionData.AdmitMaps` (rules 3 and 4) runs first in `setModMissions`, the one apply step of the missions kind; the kept declarations join the rows at their load-order place and go through `MergeMissions`, `checkReplacing`, `checkModMaps` and the serving of the row form unchanged. The launch, `-check`, the editor's Publish admission and the test-play admission all reach it through `SetModData`.
- `-check` lines: `MissionData.Replacements`, carried by the new `ModDataReport.Applied` and printed by `cmd/againrom` after the kind summaries. The missions kind has no `summary` function: the row form had no `-check` line before this story, and the file form's replacements are known only after the install's counterpart test.
- The packer: `mod.Pack` carries every folder and file, so nothing changed.
- A refusal or warning of a declaration with no line names only the file (`modItemFileError`, the map rule warnings of `checkModMaps`).

Second-game base: no branch on the game. The rom2-en and rom2-ru installs hold no `scenario/scenario.reg` (`docs/HOTFIXES.md`, hotfix sav-offer-chapter), so the campaign there lists no mission. A `replace` row is refused there, "number N is no mission of this install's campaign"; a numbered file gets the same reason from the same test and, by rule 4, comes back as a warning and replaces nothing. Not run on a second-game install in this lane.

The map editor (`pkg/game/mapeditor*.go`, `pkg/game/modmapsave.go`, unchanged): W lists the tables of `data/missions.toml` only; an implicit replacement is a file of the mod it does not list. Opening the mod reports no issue, the complete copy and Save keep the file byte for byte, and the loader reads it from the saved mod (`TestTheEditorKeepsAMapNamedByItsNumberAsAFileOfTheMod`). Publish and test play admit it through `SetModData`, so a map error in it refuses them as it refuses the launch. A mod holding no `data/missions.toml` is refused by the editor's mod open, "declares no missions", as before.

## Proof

Synthetic input only. `pkg/mod`: `TestTheFileFormReadsNumberedMapsInTheTopAndTheMapsFolder` (rule 1), `TestAMapARowNamesIsThatRowsMap` (rule 2), `TestTwoDeclarationsOfOneReplacedMissionAreRefusedNamingBoth` (rule 3), `TestAFileWithoutACounterpartReplacesAndRefusesNothing` (rule 4), `TestOneMapFileServesBothLanguages` (rule 5), `TestReplacementsNameEachReplacedMission` (rule 6), `TestAPackedModsMapsAreItsFolders` (a packed mod equals its folder) and `TestAFileAndALaterModsRowOfOneMissionMergeInLoadOrder` (the order rule between two mods). `pkg/modrt`: `TestAModOfAManifestAndANumberedMapReplacesThatMission` (a manifest and a map is a valid, marked mod). `pkg/game`: `TestAMapNamedAsACampaignMissionReplacesIt` (top and `maps/`: mission 20 served from the mod's map, the mark, the unmarked-save refusal and the digest), `TestAMapNamedAsNoCampaignMapWarnsAndReplacesNothing` and `TestTheEditorKeepsAMapNamedByItsNumberAsAFileOfTheMod`.

`TestReleaseModReplacesACampaignMission` gains the subtest `file`: a mod of the manifest and the editor-saved map as `20.alm` plays mission 20 from that map through the ordinary route, SAVE and cold LOAD keep it, a LOAD without the mod and an unmarked save under it are refused. A named run (`AGAINROM_RELEASE_ONLY`) of the candidate passed both subtests, `row` and `file`, on the EN root and on the RU root; it is not a release pass. `TestReleasePackedModPlaysAsItsFolder` and the `cmd/againrom` `-check` release tests did not run here.

## Open debt

- The editor's mod open refuses a mod without `data/missions.toml`, and its mission list and its Save into the mod do not name an implicit replacement: Save of `scenario/10.alm` into such a mod answers "no mission of mod ... names this map". The sites are `MapInspector.SetMod` and `editorMod.entryFor` in `pkg/game/modmapsave.go`, kept out of this story for the editor's mod open route.
- A mod whose numbered files all lack a counterpart is still in the save's mod mark.
- Rule 3 is refused at the launch's apply step, before any game opens, not when `pkg/modrt` reads the mod: rule 4 needs the install.
