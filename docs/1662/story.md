# The mod window adds new terrain groups

## Result

The mod window has a Terrain page listing the tile groups the mod adds: number, land kind, name, minimap
colour and four strips. A modder adds, edits and removes a group and imports each strip from a picture; a
bad strip or a taken or wrong number is refused on the page with the reason. Every change is undone, redone
and saved with the map's other edits. A group with its four strips is in the brush list before the save,
and the published mod carries it and plays without a line in its script.

## Authority

- Owner card n-modwb-07: D1 to D7 on the recommended answer; D2 (b): a kind's file is read when the mod
  holds it, so the page writes no script line.
- Story 1573 rules 1 to 5: the group rows, the strip limits of the check and the brush of a mod group.

## Rules

1. The page lists every group of the mod's terrain file with its number, kind, name, minimap colour and strips.
2. The page offers the next free number from 13 to 31 and refuses a number out of that range or taken.
3. A strip is held only when the strip reader takes it; a refusal names the group, the strip and the reason.
4. A group with fewer than four strips reads incomplete, and the save refuses it with the check's text.
5. Every page edit is one draft step that Undo and Redo walk; Ctrl+S writes it through the one mod save.
6. The terrain file is rewritten by its kind's writer, keeping the source's comments and other text.
7. A whole group is in the brush list before the save and paints the map.
8. Publish packs the terrain file and strips; the packed mod passes the check and plays, its script unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| mod data kind | `pkg/mod/kinds.go` `dataKinds` | `DataKind` gains a writer; `terrain` takes `writeTerrain` beside `readTerrain` |
| mod install files | `pkg/mod/files.go` install file reader | `OverlayInstallFiles` lays draft files over the folder's |
| W page | `pkg/ui/mapeditorworkspace.go` `modPages` | a terrain page of workspace fields in scope `terrain` |
| workspace fields | `pkg/game/mapinspectionworkspace.go` | scope `terrain` routed to `mapeditormodterrain.go` |
| mod file import | `pkg/game/modassetimport.go` `importModFile` | strips go through the one import commit, checked by `ReadStrip` |
| editor mod context | `pkg/game/mapeditormodcontext.go` `openModContext` | `draftTerrain` reads the drafted rows and strips |
| field file pick | `pkg/ui/mapeditormod.go` `modPathShown` | a field with `Pick` asks the one open-file question |

## Proof

| test | rules |
|---|---|
| `pkg/mod` `TestTerrainWriterKeepsTheSource` | 6 |
| `pkg/mod` `TestInstallFileDraftsGoOverTheFolder` | 7 |
| `TestTerrainPageAddsAGroupAndRefusesWhatTheLoaderRefuses` | 1, 2, 3 |
| `TestTerrainPageImportsUndoesAndOffersTheBrushBeforeSave` | 3, 5, 7 |
| `TestTerrainPageModSavesPublishesAndPlays` | 4, 5, 6, 8 |
| `TestTerrainPageUndoPastASaveRestoresARemovedGroup` | 5 |
| `TestTerrainPageUndoPastASaveRestoresAReplacedStrip` | 5 |
| `TestTerrainPageRefusesAMinimapColourInTheLoadersWords` | 2 |

Release tests at the release: `TestReleaseModTerrainGroupPlays` on rom1-en and rom1-ru (owed, not written).

## Open debt

- A group gained by an edit is appended after the trailing blank lines of the last group.
- Remove drops the strips under `files/common/` only, not strips under a language folder.
- The W tab row asks the host for the terrain fields each time it is built.
- The page-made mod plays on the fixture front without its mission rows; its own mission waits for the
  release test above, the first to play a mod group on an install.
