# The Units tab places units for a chosen player without a window

## Result

The map editor's Units tab shows a player switch where the hint to choose a picture stood. It lists the map's
players by number and name, starts at the player the add window would have offered first, and holds the choice while
the editor stays open on that map, following a rename or a delete of that player. A press on a unit picture, then on
the map, places one copy per press for the chosen player with no window; the group follows the add window's default.

The unit window's detailed view shows the corpse drops in the bag column and no longer shows read-only stat blocks.
It has three columns: the character card stands with its inputs in its own block under the identity, and the figure
under the equipment.

The Players window is the one screen that manages players. Each matrix row starts with the player's human/computer
switch, number, name field and a cross. The cross asks first and deletes as one edit. Add adds a player at once.
The add window and the field prompt no longer offer a player; a player chosen elsewhere opens a window that only
reads its fields and lists its users. E types the player's name in its row, Del in the Players window asks the same
question, and A adds at once. The editor's agent tools keep their commands.

## Authority

- Owner: «вместо этого текста делаем переключатель Игрока,
  которому мы ставим этого юнита»; «при клике на юнита не открывать
  больше промежуточное окно»; «блок на средней сложности убираем»;
  «добыча с трупа идет на колонку "Сумка и лут"»; «значения карты убираем»;
  «игроки должны управляться только одним вот этим экраном!».

## Rules

1. On the Units tab, a player switch lists the map's players, the current one marked, where the hint stood.
2. The switch starts at the add window's first offered owner and keeps its player across picks and renames.
3. A unit picture press, then a map press, places one unit for the switch's player and opens no window.
4. Every other tab keeps its hint and shows no player switch.
5. The unit window shows the corpse drops under the bag and no difficulty or map-value block.
6. The detailed view has three columns: the card and its inputs under the identity, the figure under the equipment.
7. Each Players matrix row holds a whole switch, number, name field and cross; the first player's cross is off.
8. The switch and the name write one edit each; the cross and Del ask first, and the answer deletes as one edit.
9. The A key and the Players window's Add add one player with no window.
10. Elsewhere a player's window only reads its fields; E there types the name in the Players window's row.
11. One player still shows its switch, number and name field, with no matrix; Del asks only in the Players window.

## Code

| kind | its one builder | change |
|---|---|---|
| choice row | `pkg/ui/editorpopup.go` popupText row with buttons | the records palette adds the player row |
| add flow | `pkg/ui/mapeditoradd.go` addRun, addKindRow, `mapeditortrigger.go` addNow | quiet run; OneClick kind |
| place palette | `pkg/ui/mapeditorplace.go` placeGridSpec, placeChoose | Units tab: no hint, quiet run, owner pick by key |
| popup layout | `pkg/ui/editorpopup.go` layoutPopup | holder rows (card, figure); grid head rows (HeadOf) |
| record window | `pkg/ui/mapeditorobject.go`, `mapeditorplain.go` plainCardRows | the held card block; a player's window only reads |
| Players window | `pkg/ui/mapeditorplayers.go` relationGridSpec | row controls (also one player); delete question |
| question box | `pkg/ui/mapeditorwindow.go` questionSpec, `mapeditormode.go` | promptDelete |
| editor words | `pkg/words/text/{en,ru}/strings.toml` | `editor.place.player`, `editor.players.delete*` |

## Proof

| test | rules |
|---|---|
| `TestMapEditorUnitsTabPlacesForTheChosenPlayer` | 1, 3, 4 |
| `TestMapEditorUnitsTabPlayerFollowsRenameAndDelete` | 2 |
| `TestEditorViewsUnitLootStandsWithTheBag` | 5 |
| `TestEditorDetailedViewHoldsTheCardUnderIdentity` | 6 |
| `TestMapEditorWindowReachesEveryField` | 6 |
| `TestMapEditorPlayersRowsHoldTheirControls` | 7, 9 |
| `TestMapEditorPlayersRowSwitchAndNameAreOneEditEach` | 8 |
| `TestMapEditorPlayersCrossAsksThenDeletes` | 8 |
| `TestMapEditorPlayersAddOpensNoAddWindow` | 9 |
| `TestMapEditorPlayerWindowElsewhereOnlyReads` | 10 |
| `TestMapEditorPlayersOnePlayerKeepsItsRowControls` | 7, 8, 11 |
| `TestMapEditorPlayerDeleteNeedsThePlayersWindow` | 8, 11 |

Release tests at the release: `TestReleaseMapEditorOverrideFrames`, `TestReleaseMapEditorUnitTierFrames`.

## Open debt

- A player's colour has no control in the Players window; only the agent tool set_field writes it.
- The editor's view goldens and mode tables were not regenerated (owner: the editor is pinned by no golden or table).
