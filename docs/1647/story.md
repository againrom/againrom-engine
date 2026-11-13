# A mod's own files follow one rule and are imported by the editor

## Result

A mod author picks a picture or a town description in the map editor and the editor copies it
into the open mod, under the folder of its kind. The file is checked at once by the same reader
the game uses, so a file the game would refuse is refused before it enters the mod. The import
is one undo step and nothing is written until Ctrl+S. The check of a mod now warns about a file
the game never reads and about a file kept outside the folder of its kind.

## Authority

- Owner: «If there are additional assets, how do we pass them through.» The owner accepted
  `assets/<kind>/` as the place of a mod's own files; a mod is made in the editor with no
  hand-written file.

## Rules

1. Every row field that names a file of the mod, and every `mod:` art key, passes one rule.
2. A refused value or a missing file fails at the row's line; dotted, over-255-byte and non-UTF-8 names are refused.
3. A mod that loaded before loads now, with the same content.
4. A named file outside `assets/<kind>/` loads and brings a warning naming the row and the place.
5. A file under `assets/` that no row names brings a warning; the packer still packs the tree.
6. A mod whose rows name its files under `assets/` shows neither warning.
7. An import puts the file at `assets/<kind>/<name>`, the name in lower case within 255 bytes.
8. An import is one undo step, writes nothing, and Ctrl+S writes it; undo takes the draft away.
9. A file the kind's reader refuses is refused at the import in the reader's words, with no draft.
10. A second import of a held name asks first, then replaces in one undo step.
11. A speaker's face is this import: record and face at `assets/speakers/<id>.png`, one undo step.
12. The import takes every sort of file the tree keeps under `assets/`, in English and Russian.

## Code

| kind | its one builder | change |
|---|---|---|
| reference to a mod's file | `pkg/mod/reference.go` `Reference` | new; replaces four older checks |
| tree description and its warnings | `pkg/mod/treedescription.go` `Describe` | two warnings added as rows |
| file import into a mod | `pkg/game/modassetimport.go` `modAssetDraft` | new; sorts are rows of `modAssets` |
| speaker add | `pkg/game/modspeakers.go` `AddSpeaker` | face goes through the import |
| project writer | `pkg/game/mapeditormod.go` `retainedPackageFiles` | removes an undone import |
| mod window keys | `pkg/game/mapeditormodcontrols.go` `setModControl` | keys `import.<kind>`, `import-replace` |
| editor words | `pkg/words/text/*/strings.toml` | five entries, EN and RU |

## Proof

| test | rules |
|---|---|
| `TestEveryRowFileFieldPassesTheReferenceRule`, `TestReferencePath` | 1, 2 |
| `TestTownArtKeyPassesTheReferenceRule` | 1, 2 |
| `TestAssetWarnings`, `TestAssetPlaces` | 4, 5, 6 |
| `pkg/modrt` `TestRowReferencesKeepFolderAndPackEquivalent`, `cmd/againrom` `TestModLinesKeepTheirBytesOnTheFixtures` | 3, 4 |
| `TestAssetWarningsReachTheCheckAndTheEditor` | 4, 5, 6 |
| `TestImportItemPictureIsOneDraftUntilSave` | 7, 8, 10 |
| `TestImportRefusesWhatTheKindsLoaderRefuses` | 9 |
| `TestSpeakerFaceIsTheImport`, `TestModSpeakerIsOneDraftEditUntilSave` | 9, 11 |
| `TestImportTakesEverySortTheTreeKeepsUnderAssets` | 12 |
| `TestImportedNameIsLowerCaseWithinTheNameBound` | 7 |

Release tests at the release: none.

## Open debt

- Items, bodies and towns have no page in the mod window; their import waits for those pages.
- The check's warnings are English rows of the report, like its other rows.
- The example mods under `pkg/modrt/testdata/mods` keep their old paths and show the warning.
