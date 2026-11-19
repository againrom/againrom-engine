# Paint terrain on a second-game map and save it

## Result

A second-game map opened in the editor can be painted with the terrain brush and saved. A map saved with no edit is the
same file, byte for byte. A saved edit reloads and shows the stroke, and the records it did not touch keep their bytes.
Every other tool stays off on a second-game map and says so in one line. The first game's map save is as before.

## Authority

- Owner: «Еще задача это редактор rom2»; one builder per kind, the first game as profile data.
- Claims R2-ASSET-231 to R2-ASSET-234: the original editor's write sequence; a value the claims grade Medium or
  Unknown is kept as loaded.

## Rules

1. A second-game map saved with no edit is the installed file, byte for byte, on every installed map.
2. A terrain stroke changes only the tile plane and the group mask word; the saved map reloads and shows it.
3. A second-game save keeps the loaded group mask bits and adds the plane's; the first game's mask is as before.
4. A second-game map refuses move, copy, delete, field edit, add and the non-terrain strokes with one line.
5. The first game's map save is byte for byte as before.

## Code

| kind | its one builder | change |
|---|---|---|
| map save form | `alm.SaveForm` | takes the format; the second game's row keeps the retained bytes |
| record extent | `alm.PayloadExtent` | the loader's and the editor's one reading of a type-0 record's length |
| group mask | `alm.SavedTileGroupMask` | format row: replace or keep the loaded bits |
| editor save | `Editor.SaveBytes` | one path for both formats; a map not written back equal is refused |
| editor host | `mapInspectionEdits.project` | a second-game map gets the narrow host `secondEdits` |
| map file write | `mapInspectionEdits.save` | the second game writes the map file alone |

## Proof

| test | rules |
|---|---|
| `TestReleaseSecondMapSavesEveryInstalledMapUnchanged` | 1 |
| `TestReleaseSecondMapTerrainEditSavesAndReloads` | 2, 4 |
| `TestSecondMapTerrainEditSavesAndReloads` | 2, 3 |
| `TestSecondMapFormatSurvivesCloneAndSavesUnchanged` | 1 |
| `TestReleaseInstalledMapsKeepTheirBytesThroughTheSaveForm` | 5 |

Release tests at the release: both second-game installs and the first-game install.

## Open debt

- Whether the original program loads an edited file is untested; no original ran.
- Object moves and deletes, and the height, object and passability planes, stay off until their written fields are
  read.
- No map is written from nothing for the second game: a new map's values are Unknown.
