# Every page of the mod window says what it is for behind a (?) mark

## Result

Each page of the mod window has a (?) mark beside the window's close button. On hover it says what the page
changes in the game, then the page's keys, wrapped to as many lines as it takes. The keys no longer stand as
lines in the window; the window shows only the mod summary, a status the modder must see and the build output.

A field or group with an explanation shows it behind its own (?) mark, never as lines under it.
The Overview headings say what they list, and the Missions page's map button names its act.

The Terrain page adds a ground from one PNG or BMP picture: the editor cuts it into 32 by 32 squares, brings
it to 256 colours and writes the four strips and the terrain row itself. The page lists the mod's grounds by
picture, name and the game ground units walk it as, then the game's grounds read-only; no group number,
strip or minimap colour stands on it. Delete is refused while a mod map paints the ground, naming the map.

Every list and object window opens from its home in the middle of the editor window until it is dragged or
sized; a dragged one reopens where it was. A window opened while the list window stands,
such as a player from the Players list, opens one title row and padding down and right of the list.
A long dropdown list rises over its field when it does not fit below it.

## Authority

- Owner: «every page of the mod window says what it is for, in a modder's words»; popups open in the middle.
- Owner: a ground is made from one picture; the modder never sees group numbers or strips.

## Rules

1. When a page of the mod window is shown, its title has a (?) mark whose hover says the page's purpose and keys.
2. When a field or group has an explanation, it stands behind the row's (?) mark, wrapped and whole.
3. When a mission page lists a map no mission uses, its button and caption are a verb and plain words, whole.
4. When the Overview page is shown, each heading says what it lists; errors and warnings say what they mean.
5. A list or object window opens from home in the middle, or one step off an open list; a dragged one keeps its place.
6. Nothing else changes: files, saves and undo behave as before.
7. When Add ground takes a picture of 1 to 56 squares of 32 pixels, one undo step writes its strips and row.
8. When a mod map paints a ground, its Delete is refused naming the map; otherwise Delete and undo are one step.
9. When any choice list in the editor does not fit below its field, it rises until it ends at the window's bottom.
10. When a row's (?) mark is pressed, it shows the row's help and never runs the row's action.

## Code

| kind | its one builder | change |
|---|---|---|
| popup | `pkg/ui/editorpopup.go` `layoutPopup`, `drawPopup`, `popupTipAt` | `Help` per row, `TitleHelp`, one help tip; `objectFieldRows` moves notes to `Help` |
| mod window | `pkg/ui/mapeditormod.go` `modSpec`, `pkg/ui/mapeditorworkspace.go` `modPageWords` | tab word and page help per page |
| popup place | `pkg/ui/mapeditorpopupstore.go` `keepInRoom`, `openCentred` | home popups open in the middle; `inputAt` and `selectionPress` read that place |
| dropdown | `pkg/ui/widgetdropdown.go` `Dropdown.ListBottom` | a list that passes the bottom rises |
| mod fields | `pkg/game/mapeditormodfields.go` `modFields` | field help from `editor.mod.help.*` words |
| terrain page | `pkg/game/mapeditormodterrain.go` `terrainFields`, `addGround`, `editTerrain` | one picture to four strips; BMP through `decodeModPicture` |
| 8-bit BMP writer | `pkg/formats/bmp/encode.go` `EncodePaletted` | new: the one writer of a paletted BMP |
| colour reduction | `pkg/formats/pal/reduce.go` `Reduce` | new: median cut to at most n colours |

## Proof

| test | rules |
|---|---|
| `TestPopupHelpMarkShowsItsHelpWrapped`, `TestModPagesSayWhatTheyAreForBehindTheTitleMark` | 1, 2 |
| `TestModWindowMissionsPageShowsEveryMapCaptionWhole`, `TestEditorListsAndOpensEveryMapOfTheMod` | 3 |
| `TestMapEditorModesHoldTheModOverviewRows`, `TestEditorModMapListDrawsInEachLanguage` | 3, 4 |
| `TestModWindowHomeOpensInTheMiddle`, `TestEveryPopupHomeOpensInTheMiddle` | 5 |
| `TestDraggedPopupReopensWhereItWasDragged`, `TestRussianModWindowLeavesTheMapItsInput`, `TestWindowOpenedFromAnotherOpensOffsetFromIt` | 5 |
| `TestEditorPopupPlacementTranscript`, `TestMapEditorModesHoldEveryKeyAndGesture` | 6 |
| `TestTerrainPageAddsAGroundFromOnePicture`, `TestModWindowAddGroundButtonAsksForAPicture` | 7, 8 |
| `TestTerrainPageRefusesAPictureItCannotCut`, `TestEncodePalettedReadsBackThroughTheDecoder` | 7 |
| `TestReduceBringsAPictureToAtMostTheTable`, `TestTerrainPageModSavesPublishesAndPlays` | 7 |
| `TestTerrainPageUndoPastASaveRestoresADeletedGround` | 8 |
| `TestTriggerWizardItemListsOpenWholeOnTheSmallestScreen`, `TestChoiceListRisesToTheWindowBottom` | 9 |
| `TestPressOnARowHelpMarkNeverRunsItsAction` | 10 |

Release tests at the release: `TestReleaseMapEditorWorkspaceRoutesAndTexts` in EN and RU.

## Open debt

- English fragments remain in Russian messages from the Terrain page: `errors.New("a folder")`, the picture
  decoder errors.
- Texts page rework (edit where shown, where/English/Russian list): not started.
- Items page rework (80x80 picture and base item, typed name, material and tier choices): not started.
- Field help and plain labels on the Texts, Items and Files pages; first-step-only empty pages.
