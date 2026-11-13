# One mission layout in a mod, with `.almx`

## Intent

Every writer of a mission into a mod puts the map, its pages and its variable labels at the paths of the fixed
tree, and every reader names the same paths. A mod written in an older layout keeps working and moves into the
tree on its next save. An extended map saved into a mod is an `.almx`. W's mission page shows the row's `class`,
`sex`, `carry` and `off` keys.

## Authority

Owner direction: the fixed tree of the mod structure specification, section 1, and its story order, row P6
(review `mod-structure/SPEC.md`); section 6, option A, states the compatibility rule: old paths are read with a
`-check` warning and every writer writes only the new tree. Section 1 is the authority where it and the P6 row
differ. Its rows 14, 15 and 15a were compared with the P6 row and name the same paths: the per-language map of
row 14 with the language folder of row 15 is the `cmd/modmission` path `maps/<lang>/m<N>.text/<lang>/`. No
difference was found, so no choice between them was made.

No divergence row is added: the change touches no ROM1 behaviour.

## As-built behaviour

### The place rule

`mod.MissionPlace` (`pkg/mod/missionplace.go`) is the one path rule of a mission's files:

- `PlaceMission(stem, extended)` is `maps/<stem>.alm`, or `.almx` for a map that holds the extended record;
- `PlaceLanguageMission(lang, stem, extended)` is `maps/<lang>/<stem>.alm` or `.almx`; a row names it with
  `{lang}`;
- `MissionStem(name)` is the source map's file name without its extension: `m<N>` for digits, so a written map is
  never the numbered file that replaces a campaign mission by name, and `map` for a new or unusable name;
- `Pages(lang)` is `<stem>.text` or `<stem>.text/<lang>`; `Labels()` is `<stem>.variables`;
- `Free(held)` adds `-2`, `-3` and on to the stem while a map, page or labels file of that stem is held;
- `InTree()` tells a row map in the tree from an old path.

The tree rows of `pkg/mod/kinds.go`, the tree description and the page reader (`Mission.ReadTexts`) take their
folders from the rule. The rows name the map only; the pages lie beside it.

Callers:

| writer | path |
|---|---|
| W package save (`missionPackageFiles`, `editorPackageSpec.mission`) | `maps/<stem>.alm` or `.almx` under the open map's stem |
| W Add open map | the same, made free in the project (`Free`) |
| F5 test play | the project's paths after the move below, into the private copy |
| `AssembleModMission`, `cmd/modmission` | `maps/<lang>/m<N>.alm`, pages in `maps/<lang>/m<N>.text/<lang>/` |

### Older layouts

The older layouts are the package save's `mission/map.alm` with `mission/text/{lang}`, W Add's
`mission/map<N>/map.alm` with `mission/map<N>/text/{lang}`, and the assembler's `mission/{lang}/map.alm` with
`mission/{lang}/text`. Such a mod opens, checks, test plays and packs unchanged; the check prints one old-path
warning per key.

`mod.TreeLayout` (`pkg/mod/missionlayout.go`) runs on every save through the project writer, a copy to a new
folder included. For each row whose map or texts folder lies outside the tree:

- the map, its labels and the pages beside it move under the rule with their bytes and names;
- the pages of the texts folder, and of its parent when its tail is a language, join the pages beside the map
  unless a page the reader takes first holds the name;
- `map` and `texts` are rewritten in place by `mod.WriteMissions`; `texts` stays only for a per-language map whose
  texts were per language, and then names `maps/{lang}/<stem>.text/{lang}`;
- a row whose map holds the extended record but is named `.alm` takes `.almx`;
- a speaker record bound to a moved map names the map's new path (`mod.MoveSpeakerMaps`);
- old files no row reads any more are removed, and folders the move empties go with them.

What stays: every other file and key; a row whose new file a held file takes; rows that share a map but not a
texts folder. Each language reads the same map bytes and pages before and after. A mod already in the tree,
saved without an edit, keeps its bytes. F5 writes the moved layout only into its private copy; the source mod is
not touched.

The move changes the mod's file paths and so its content digest, as any edit through W does: a game save made
with the mod before its first W save names the old digest. Opening, checking, playing and packing an old-layout
mod move nothing.

Undo and redo across a moving save: the save rebases every project state that undo and redo hold through the
same `mod.LayoutChange` it wrote (`rebaseHistory` in `pkg/game/mapeditormod.go`). A row that declares a moved
map and texts takes the new place (`LayoutChange.Rebase`), the table source and the rows it reads as are the
saved ones, drafts and the speakers draft follow the moves, and an opened map that moved opens at its new path. The
state's own edits stay. An undone or redone state saved again therefore writes the same files and reads the same
map and pages in each language; no `maps/map-2.alm` and no old file is left. A save that moves nothing leaves the
history as it is, so undo is unchanged for a mod already in the tree.

The old-path warning of `describeMissionPages` names the pages folder of the place `MissionPlace` gives the map:
its own for a map in the tree, the one `TreeLayout` moves it to otherwise.

### `.almx`

W saves an extended map into a mod as `maps/<stem>.almx`. The loader admits it, F5 plays it, Publish packs it,
and the packed mod reads the same bytes as its folder. `CheckModMap` is unchanged; the `extended-name` warning
still names a handwritten `.alm` that holds the record, until its next save renames it.

### W mission page

Below the route fields of a guest row:

- `class` (Any, Fighter, Mage) and `sex` (Any, Male, Female) are choice fields. A change is a draft change with
  undo and redo; the project writer writes it through `mod.WriteMissions`. Any writes no key, so an absent key
  stays absent.
- `carry` is a read-only line, "joins at the next town", on a companion row; a party row carries none. The
  loader admits only `carry = "join"`, so there is nothing to choose.
- `off` is a read-only line for every row. An off row holds only `number` and `off`; the project stages the mod
  alone, where an off row switches off nothing and is refused, so the page does not set it.

The words are in the word book in EN and RU (`editor.mod.class`, `sex`, `anyhero`, `carry`, `carryjoin`, `off`,
`offyes`, `offno`). No key or gesture is added.

No SAV byte, simulation state or game without a mod changes.

## Proof

- `pkg/mod`: the place rule, `TreeLayout` with read equality per language, `MoveSpeakerMaps`,
  `TestOldTextsWarningNamesThePlaceRulePages`.
- `pkg/game`: `TestAnOlderLayoutModPlaysAndItsNextSaveWritesTheTree` (the three older forms: check, pack, F5,
  save, exact moves, rewritten table and speaker record, read equality), `TestAnExtendedMapSavesIntoAModAsAlmx`,
  `TestATreeLayoutModSavedWithoutAnEditKeepsItsBytes`,
  `TestUndoAndRedoAcrossAMigratingSaveKeepWhatTheMissionReads` (undo and redo across the moving save, save
  again, F5: same files, same map and pages in EN and RU), `TestEditorMissionPageEditsClassAndSexAndShowsCarryAndOff`,
  `TestEditorNewGuestMissionShowsItsCarryLine`, and the caller fixtures moved to the tree layout.
- `-check` lines changed on purpose: the `guest-chain` and `reniesta` fixtures of `pkg/modrt/testdata/mods` name
  `maps/{lang}/m10.alm`, so their refusal paths and mission and mark digests in `merged-data.txt` change and their
  six old-path warnings in `cmd/againrom/modlines_test.go` are gone; the mission text record names `texts/en/`.
- Release tests, not run here: `TestReleaseMapEditorCompleteScenarioPackageSAV`,
  `TestReleaseEditorPublishedModChainSAV`, `TestReleasePackedModPlaysAsItsFolder`, and the release tests whose
  source this story moved to the tree layout (listed in the lane's result).

## Open debt

- `TestReleaseMapEditorCompleteScenarioPackageSAV` saves a plain `.alm`; an `.almx` mission on rom1-en and rom1-ru
  is proven headless only.
- `off` and `carry` are shown, not edited (reasons above). Editing `off` needs an owner decision on staging a mod
  with its lower layers.
- `cmd/againrom/testplay_release_test.go` keeps the old layout on purpose, as the compatibility witness on an
  install.
