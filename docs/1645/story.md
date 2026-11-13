# A map's texts inside the map

## Result

A map maker can keep a map's briefing and message pages inside the map file itself, so one file carries the
whole map. The map editor's File menu has a switch for it, off by default, and the save window shows the same
switch; a map saved with it on is an extended map file. The game, the mod check and a packed mod read those
pages as they read a text folder, in English and Russian. With the switch off, saving writes the same files as
before. No save leaves such a map's pages in neither the map nor its folder.

## Authority

- Owner: «записывать тексты прямо в карту (с фолбеком на текстовые файлы) -- тут мб переключатель при сохранении».
- Owner: «по умолчанию выключено. если almx, то состояние переключателя можно прямо туда записывать. включение
  делает конечно же almx». The extended map format is Againrom's own; no original behaviour is involved, no DIV row.
- Owner: the switch is a row of the editor menu, reachable whatever file dialog Save As opens.

## Rules

1. One texts record of the map holds every page of both languages, and save and reopen lose none.
2. A mission reads the record's pages first, then the folders; a page the record lacks comes from a folder.
3. A map without the record reads its pages as before.
4. The switch is off by default; a map that holds the record opens with it on.
5. Switching on an original map asks the extended map question; Yes makes the map extended and its name `.almx`.
6. Saved with the switch on, the pages go into the map and no folder is written; an existing folder is left as is.
7. Saved with the switch off, the map and its folder are written as before, with the record's pages in the folder.
8. An `.alm` never holds the record: a save under an `.alm` name is refused, a save in place writes the `.almx`.
9. The mod check accepts a mod map with its pages inside and still reports a map with neither record nor folder.
10. A mod mission shows the same pages from the record as from the folder, in EN and RU, folder and packed alike.
11. A save that cannot keep the pages of a map that held the record is refused in EN and RU and writes nothing.
12. A mod's draft of such a map keeps the record or takes its pages as files; opening a mod keeps the switch.
13. A save of a map over 1 MiB with its pages inside is refused in EN and RU: the game would read no page.
14. The File menu row and the save window row are one state; switching either way marks the map changed.

## Code

| kind | its one builder | change |
|---|---|---|
| map record | `alm.frameRecord`, `alm.appendRecord` | texts record, typeId 101, a row of the family |
| map pages reader | `mod.Mission.ReadTexts` | the record's pages before the folders |
| page set for a save | `exportPageTree` | the encoder takes the folder writer's page set |
| text folder writer | `writeEditedMapPages` | no folder for a map with the record; a replaced map with it returns when the folder swap fails |
| map save bytes | `withTexts` | the record added when the switch is on; the size refusal |
| editor menu | `toolButtons` table | one switch row in File, no key |
| save window | `promptSpecBase` popup rows | one `popupSwitch` row |
| words | word book EN and RU | six `editor.save.textsinside` and `editor.menu.textsinside` words |

## Proof

| test | rules |
|---|---|
| `TestTextsRecordLayoutAndRoundTrip`, `TestTextsRecordRidesBesideTheScriptRecordAndComesOut` | 1 |
| `TestMissionPagesComeFromTheMapsTextsRecordFirst`, `TestMissionPagesFromTheRecordNeedNoFolder` | 2, 3 |
| `TestPackedModReadsTheMapsTextsRecordAsItsFolder` | 2, 10 |
| `TestTextsInsideSaveWritesThePagesIntoTheMap`, `TestTextsInsideRecordAndFolderLoseNothing` | 1, 2, 4, 6, 7 |
| `TestTextsInsideNeverRidesAnAlm` | 5, 8, 9 |
| `TestTextsInsideModMissionShowsItsPages` | 9, 10 |
| `TestMapPagesSaveAsCarriesPages`, `TestSaveInPlaceRewritesOwnTextsFolderOnly`, `TestMapPagesSourceOrder` | 3, 7 |
| `TestSwitchOffSaveNeverLosesTheMapsTexts`, `TestFailedFolderSwapReturnsTheMapWithItsTexts` | 11, 14 |
| `TestModDraftNeverLosesTheMapsTexts`, `TestOpeningAModKeepsTheSwitch` | 12, 14 |
| `TestSwitchOnSaveRefusesAMapTheGameReadsNoPageFrom`, `TestSwitchOnSaveWritesAMapTheGameReads` | 13 |
| `TestFileMenuRowSwitchesTheTextsInsideTheMap`, `TestSaveWindowSwitchPutsTheTextsInsideTheMap` | 4, 5, 14 |

Release tests at the release: `TestReleaseMapEditorLooseMapPages`, `TestReleaseMissionTextsRoundTripThroughUTF8`,
`TestReleaseInstalledMapsKeepTheirBytesThroughTheSaveForm`, `TestReleaseModMissionReadsTheMapFromTheMod`,
`TestReleaseMapEditorWorkspaceRoutesAndTexts`.

## Open debt

- `TestFailedFolderSwapReturnsTheMapWithItsTexts` fails the swap by an open page file, which holds only on Windows.
- The 1 MiB bound also refuses a loose map that large with its pages inside; such a map saves with the switch off.
