# One map editor host for both games

## Result

Nothing changes for a player or a modder. The map editor now has one host for the first and the second game's maps.
What its tools may do on a map is read from one row of the map's format, so a tool added for the second game needs
one row entry, not a second host. Every refusal the second game's editor showed stays, in the same words.

## Authority

- Owner: «Как рекомендуешь» on the architecture audit's finding N98, as one story before the second game's next editor
  story.
- Engine rule «One builder per kind»: one host, one section table, one script limit, one palette list.

## Rules

1. A second-game map takes only the terrain brush, placements, script words, undo and save; other tools refuse.
2. A second-game map offers no optional tool of the first game's editor that it did not offer before.
3. A second-game script register is held to the registers of the game's script profile, as the first game's is.
4. A second-game map saves without pages, labels or the first game's save gate; its new records sit at a cell centre.
5. Every first-game editor tool works as before.
6. No game or map editor code asks whether a map is the second game's, and no second type wraps the host.

## Code

| kind | its one builder | change |
|---|---|---|
| map edit host | `mapInspectionEdits` | the second host type and the palette wrapper are gone |
| tool capabilities of a format | `editRows` in `pkg/game/mapeditrow.go` | one row per format; 15 branches read it |
| record section table | `formatRows` in `pkg/mapedit/section.go` | the second table is the second format's entry |
| script parameter limit | `scriptAddressesVariable`, `Editor.Profile` | the literal register set is gone |
| palette list | `mapInspectionEdits.NewRecordStep` | the row names the steps the second game offers |
| optional tool seam | `ui.Seam` | 85 reads go through it; the host names the seams it offers |

## Proof

| test | rules |
|---|---|
| `TestEditRowsHoldBothFormats` | 1, 2, 5 |
| `TestHostOffersEverySeamOnTheFirstRow`, `TestSeamReadsWhatTheHostOffers` | 2, 5 |
| `TestMapEditHostIsOne` | 6 |
| `TestSecondMapFormatSurvivesCloneAndSavesUnchanged`, `TestFirstGameMapOpensForEditingWhateverTheCaller` | 1, 5 |
| `TestReleaseSecondMapOtherScriptEditsAreRefused`, `TestReleaseSecondMapPlacesMovesAndDeletes` | 1, 4 |
| `TestReleaseSecondMapParameterEditSavesAndReloads` | 3 |
| `TestReleaseSecondMapSavesEveryInstalledMapUnchanged` | 4 |

Release tests at the release: `TestReleaseFirstMapEditorStillOpensAndEdits`,
`TestReleaseSecondPalettesListTheCatalogAndNameEveryPlacement`.

## Open debt

- The row is two tables of one key: tool capabilities in the game package, the section table and script profile in the
  map editor package, which cannot name the editor's kinds.
- The map editor package still branches on the second format in its map read and its section insertion.
