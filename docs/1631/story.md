# The editor opens any mod

## Intent

The map editor's mod window (W) opens any first-game mod, whatever kinds it holds: a mod folder, or a packed
`mod-<id>.res` read-only. An Overview page lists the mod's kinds, the lines `againrom -check` prints for it, its
files, its overrides, its warnings and what refuses it. The editor's lists take the mod's data the way the game
applies it: a terrain-only mod's groups paint, and the party's equipment offers the mod's items. A mod with an
enabled mission opens that mission's map as before.

## Authority

Owner direction in the story brief (mod workbench plan, story 2 of 17): first game only; no dependency on an open
map; packed is read-only, with no copy button and no writer; no load route changes and no `main.star` is written;
a read-only mod refuses every edit and save through one test at the write boundary; the packer, the mission writer
and the speaker writer stay as they are; words go through the editor's word table in EN and RU. No ROM1 claim
applies: mods are an Againrom surface. No divergence row changes.

## As-built behaviour

Opening. `MapInspector.SetMod` (`pkg/game/modmapsave.go`) reads the manifest with `mod.Load` and the mod's files
with `mod.OpenTree`, the one reader of a folder or a packed file. A packed mod skips the inside-install fence and
the ownership check and is marked `packed`. A mod without a missions file opens. `openWorkspace`
(`pkg/game/mapeditorpackage.go`) opens the first enabled mission's map for a folder mod that has one; any other
mod, and every packed mod, opens over the map in the editor. A failed open puts the previous mod back.

The mod's data. `openModContext` (`pkg/game/mapeditormodcontext.go`) loads the mod with `modrt.Load` over the
install's base and builds the editor's table and archives through `admittedFork`, which applies the data with
`FrontEnd.SetModData` over `modDataAppliers`, as a game run does. The editor's terrain list, speaker catalog,
character catalog and item names read that fork. When the game refuses the mod's missions, the fork is built
without them and the refusal is kept for the overview. `admitMissionPackage` and `AddSpeaker` use the same fork.

Read-only. `modReadOnly` is the one test of a packed mod. `SetWorkspaceField` refuses every W change through it,
except a change that opens another mod; `SaveIntoMod` refuses through it. Publish and Add speaker already refuse a
mod the editor does not own, and a packed mod is never owned. `ModTarget` names a packed mod with "packed,
read-only".

The report. `game.ModCheckLines` (`pkg/game/modcheck.go`) is the report of the mods a run loaded. `-check` prints
each line after `againrom: `; the overview shows the same lines for the editor's load.

The overview. `modOverview` (`pkg/game/mapeditormodoverview.go`) gives six sections: the kinds whose
`mod.DataKinds()` file the mod holds, the report, the files (`mod.Tree.Files`, which `Digest` now walks), the
overrides, the warnings and the refusals. An empty section reads "none". W shows the overview first for a packed
mod or one with no enabled mission. The Missions and Overview tabs on W's popup, or the W key while W is open,
turn the page; with no mod open W has no tabs and the W key works as before.

Kinds and builders:

| Kind | Builder | Removed |
| --- | --- | --- |
| Mod data in the editor | `FrontEnd.SetModData` over `modDataAppliers`, through `admittedFork` | `useModTerrain`, `useModSpeakers`, the `MergeCharacters` call in `SetMod`, the copied fork build in `admitMissionPackage` |
| Mod open (folder or packed) | `mod.Load` and `mod.OpenTree` in `SetMod` | the packed refusal and the no-missions refusal |
| Read-only write boundary | `modReadOnly` | none |
| `-check` report | `game.ModCheckLines` | the body of `modLines` in `cmd/againrom` (it now prefixes) |
| Mod file list | `mod.Tree.Files` | the walk inside `Digest` |
| W pages | `popupSpec` tabs in `modSpec`, rows as popup text rows | none |
| Words | the editor word table, EN and RU | none |

No new persisted field, no game or locale branch, no new widget. The packer (`mod.Pack`) and the mission writer
are unchanged. The speaker writer writes the same bytes; after the write `AddSpeaker` rebuilds the mod's fork
instead of laying the record over the table.

## Proof

Focused tests, synthetic or fixture inputs:

- `TestEditorOpensEveryFixtureMod`: all 18 folders of `pkg/modrt/testdata/mods` open in W's folder route; each
  overview's report equals `ModCheckLines` of the mod loaded as `-check` loads it, a refused mod shows the
  refusal, and the files list holds `mod.toml`.
- `TestModLinesKeepTheirBytesOnTheFixtures` (`cmd/againrom`): `-check`'s lines equal a frozen copy of the old
  `modLines` over the 18 fixtures alone and one set of four, on `rom1-en` and `rom1-ru`.
- `TestEditorOpensAPackedModReadOnly`: a packed copy of `snow-field` opens with its terrain groups; nine W
  changes and Save into the mod refuse with the packed word; the packed file keeps its bytes; another mod opens.
- `TestEditorPaintsTheGroupOfATerrainOnlyMod`, `TestEditorPartyEquipmentOffersAModItem`,
  `TestEditorModContextReadsTheKindsTheGameApplies`, `TestEditorShowsTheOverviewOfAModWithoutMissions`,
  `TestModWindowOpensOnTheOverviewAndItsTabsTurnPages`.
- Modes table: rows `workspace/mod-overview` and `workspace/mod-overview-fields` added and held by
  `TestMapEditorModesHoldTheModOverviewRows`; the other rows are unchanged by construction (the workspace double
  gives no overview on them).
- `TestReleaseMapEditorOwnerFrames` gains the frame `mod-overview-small`: W over a new map with `snow-field`
  open, on its overview.

Present tests changed: the no-missions case of `modmapsave_test.go` now opens and Save into the mod refuses
naming the map; `TestEditorPartyRowsPreserveTemplateAndEmptyOnSaveAndReopen` opens its mod with an install table.

Correction pass after review. W opens the mod window with no map open: the editor holds a mod host with no map
beside the map's own, and the same popup, fields and overview serve both; a mod with a mission opens that map,
any other opens on its overview, and what edits a map's own data refuses with the word `editor.mod.nomap`.
A change of the mod context drops the cached placed-person sheets, and a rebuilt mod reads its file and kind
lists again. Proof, each failing on the reviewed head: `TestEditorOpensAModWithNoMapOpen`,
`TestEditorOpensAMissionModWithNoMapOpen`, `TestModWindowOpensWithNoMapOpen`,
`TestEditorModContextDropsThePlacedArt`, `TestEditorOverviewListsTheSpeakerJustAdded`.

Release tests owed at the release, EN and RU roots of the first game: `TestReleaseMapEditorOwnerFrames`,
`TestReleaseEditorPublishedModChainSAV`, `TestReleaseMapEditorCompleteScenarioPackageSAV`,
`TestReleaseMapEditorWorkspaceRoutesAndTexts`, `TestReleasePackedModPlaysAsItsFolder`,
`TestReleaseModMissionChainPlaysAndReturns`, `TestReleaseGuestPartyBuildsCharactersAtTheirLevels`,
`TestReleaseCheckListsAModOverTheInstall`, and the editor mod release tests that open a mod through `SetMod`.

## Open debt

- A packed mod's mission maps do not open: W opens a packed mod over the map in the editor.
- The overview is text. Each kind's edit page is a later story of the plan.
- On a mod that has an overview, the W key while W is open turns the page instead of reopening W on its folder
  field. The existing W frames now show the two tabs.
