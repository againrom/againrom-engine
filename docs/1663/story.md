# The mod window edits the mod's texts in English and Russian

## Result

The mod window has a Texts page listing every text key of the mod with its English and Russian text side by
side. A key one language lacks is marked and counted. A modder adds, edits, renames and removes a key in either
language; a bad or taken key name is refused on the page with the reason, and a key in use shows what uses it.
Every change is undone, redone and saved with the map's other edits. A mission's title and briefing pick a key
from the list, and the starter and the game show its text in English and Russian.

## Authority

- Owner card n-modwb-08: a Texts page that shows each key with its English and Russian value, marks a key
  missing in one language, and lets a mission's title and briefing pick a key from the list.
- Owner plan of the mod workbench, decisions D1 to D7 on the recommended answer: a kind's file is read when
  the mod holds it, so the page writes no script line.
- Seat decision on used keys, accepted with the plan: rules 6 and 7.

## Rules

1. The Texts page lists every key of the mod's strings files with its English and Russian text side by side.
2. A key one language lacks is marked in its group and counted per language in the page's count line.
3. A key is added, edited, renamed and removed in either language; a new key starts in English with empty text.
4. A key name is held to the loader's key rule; a refused or taken name is refused on the page, naming it.
5. A key a mission row, item, character or screen cites shows what cites it.
6. A rename carries the mission rows citing a key; a key items, characters or screens cite is refused, naming the file.
7. A key in use cannot be removed from English; it can be removed from Russian while English holds it.
8. Every page edit is one draft step that Undo and Redo walk, also past a save; Ctrl+S writes it through the mod save.
9. Each language's strings file is rewritten by the strings kind's writer, keeping comments, other lines and the BOM.
10. A mission's title and briefing pick a key English holds, shown with its text; the saved row cites the key.
11. The starter's mission route and the mod loader read the title and briefing of a key in English and Russian.
12. Publish packs both strings files; the packed mod loads on rom1-en and rom1-ru, its script unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| mod data kind | `pkg/mod/kinds.go` `dataKinds` | `strings` takes `writeStrings` beside its reader; `WrittenKind`; `Cites` names the keys a reader asks |
| text key rule | `pkg/mod/items.go` `CheckTextKey` | the one key rule the strings parser and the page ask |
| mission row writer | `pkg/mod/missionwrite.go` `WriteMissions` | writes the keys a row cites, `TitleKey` and `BriefKey` |
| W page | `pkg/ui/mapeditorworkspace.go` `modScopePages` | one page table for the scope pages; texts page in scope `texts` |
| workspace fields | `pkg/game/mapinspectionworkspace.go` | scope `texts` routed to `mapeditormodtexts.go`; two package fields pick a key |
| text lookup | `pkg/mod` `Lookup` | the page and the draft rows read the draft's strings through it |
| editor mod document | `pkg/game/mapeditormod.go` `document` | the snapshot holds the strings files |
| mod file import | `pkg/game/modassetimport.go` `importedModFile` | a strings file made after a save leaves on an undo past it |

## Proof

| test | rules |
|---|---|
| `pkg/mod` `TestStringsWriterKeepsTheSource` | 9 |
| `pkg/mod` `TestStringsWriterRefusesTheKeyTheParserRefuses` | 4 |
| `pkg/mod` `TestMissionRowsCiteTheirTextKeys` | 10 |
| `pkg/mod` `TestKindsCiteTheTextKeysTheirReadersAsk` | 5 |
| `pkg/ui` `TestModWindowShowsTheTextsPageOfItsHost` | 1 |
| `TestTextsPageMarksAKeyMissingInOneLanguageAndRefusesABadKey` | 1, 2, 3, 4 |
| `TestTextsPageUndoesRedoesAndCarriesARenameThroughTheMissionRow` | 3, 5, 6, 8, 10 |
| `TestTextsPageRefusesWhatWouldLeaveACitationUnread` | 5, 6, 7, 10 |
| `TestTextsPageModSavesPublishesAndShowsTheTitle` | 8, 9, 10, 11, 12 |
| `TestTextsPageUndoPastASaveRestoresTheStrings` | 8 |

Release tests at the release: `TestReleaseModMissionTitleFromKey` on rom1-en and rom1-ru (owed, not written).

## Open debt

- A reader that stops before it asks a key cites nothing for the keys after its error.
- The install-side launch check of a page-made mission title waits for the release test above.
- The page lists the keys of English and Russian only; a third language waits for its locale.
