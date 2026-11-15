# The map editor's unit window in its plain and detailed views

## Result

The unit window takes the size its rows need: the plain view is the narrowest window that holds its rows with the
Kind row visible, and the detailed view is nearly as wide as the editor and shows every section at once, without tabs.
The unit's doll stands at the size the game draws it. Both views set the unit's facing on the pad of eight
directions and random; the plain view stands it in the lower left corner beside the card. The detailed view sets
the unit's maximum health, the gold it drops and whether it leaves loot at all.

## Authority

- Owner: «оба эти режима должны свои окна всегда растягивать на свое НЕОБХОДИМОЕ МИНИМАЛЬНОЕ значение».
- Owner: «показывать куклу того размера как в игре»; «переключатель ориентации ... в левый нижний угол».
- Owner: «изменить МАКСИМАЛЬНОЕ количество здоровья»; «строку ЗОЛОТО»; «дропает лут / не дропает лут».
- DIV-4945: the maximum health is an Againrom field at the unit record's +0x28; ALM-TAILVER-084, ALM-UNITREAD-227.
- DIV-4946: the gold is the unit's own loot record's gold word; the loot switch is unit flag bit 8; ALM-SACK-065,
  ITEM-DEATH-012, MISSION-DROP-074, ALM-FLAGCORP-110, ALM-FLAGPATH-109.

## Rules

1. Each view's window is the size its rows need at every editor size from 640x480, in English and Russian.
2. The plain view shows the Kind row as a choice; the detailed view shows every section at once, without tabs.
3. In a room lower than the card band, the band scrolls with the rows in the window's one scroll.
4. The unit's doll is the game's figure at the scale the game fits its mission frame to a window of that size.
5. Both views hold the facing pad; each of its nine buttons sets the facing.
6. The plain view stands the pad in the lower left corner beside the card, clear of the card, at the proved sizes.
7. Maximum health is set in the detailed view, written to the map, 0 derives it, and current health stays at most it.
8. Gold is the gold of the unit's own loot record, default 0, added with that record and dropped at death.
9. The loot switch off makes the unit leave nothing at death; saves keep the gold and the switch.

## Code

| kind | its one builder | change |
|---|---|---|
| popup size | `layoutPopup`, `popupFit` | per-view sizes as data; the card band joins the one scroll |
| plain rows | `plainViews`, `plainFieldRows` | the facing row and its corner placement (`plainCorner`) as table data |
| detailed rows | `personColumns`, the detailed list | sections in columns instead of tabs; health maximum, gold and loot rows |
| facing pad | `popupRow.withChoices` (`popupDirection`) | the same pad placed beside the card |
| doll | `drawObjectPreview`, `unitFigureSize` | the game's figure at the game's scale |
| unit record field | `mapedit` field table, `mapload.withTailWords` | `hpmax` and `noloot` fields; the spawn applies them |
| unit purse | `sim.Entity.Purse`, the death sack | the purse joins the sack; SAV and the world form keep it |

## Proof

| test | rules |
|---|---|
| `TestEditorPersonWindowTakesTheSizeItsRowsNeed`, `TestEditorPersonWindowOwnerFrames` | 1, 3 |
| `TestEditorPersonDetailedViewShowsEverySectionAtOnce`, `TestMapEditorUnitBasicViewIsTheCardWithItsInputs` | 2 |
| `TestEditorPersonFigureStandsAtTheGameSize` | 4 |
| `TestEditorPersonWindowFacingPadInBothViews` | 5 |
| `TestEditorPlainViewFacingPadInTheLowerLeftCorner` | 5, 6 |
| `TestEditorUnitHealthMaxGoldAndLootReachTheMap` | 7, 8, 9 |
| `TestPlacementTailHealthMaxReplacesTheDerivedMaximum`, `TestRecomputeKeepsAnAuthoredHealthMax` | 7 |
| `TestEditorUnitGoldAddsTheUnitsOwnLootRecord`, `TestOwnedStockGoldIsTheUnitsPurse` | 8 |
| `TestAPurseDropsWithTheUnitsItemsAndOnlyThere`, `TestPlacementNoLootLeavesNothing` | 8, 9 |
| `TestAPurseInTheActorValuesLeaf`, `TestAPurseRidesItsOwnFormSection` | 9 |

Release tests at the release: the `TestReleaseMapEditor*` tests that draw the unit window, English and Russian.

## Open debt

- DIV-4945: a Human that re-derives after LOAD loses an authored maximum health; no SAV home for it yet.
- DIV-4946: no claim says whether the original pays a unit's own loot record gold or reads flag bit 8; `111.alm`
  unit 91 now drops his record's 10000 gold, where base paid none.
- A 72x218 figure at 2560x1440 or wider sends the plain view's pad back among the rows.
- Evidence and frames: `review/story1661-editor-unit-window` in the project tree.
