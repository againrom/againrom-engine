# One unit window in two modes, for people and animals

## Result

The map editor's unit window is one window with a normal and a detailed mode; the normal mode is the detailed one
with the doll's face and sex, the equipment and the bag left out. The character card has no heading, stands in place
of the stat field blocks, and the unit's name is typed on the card itself. The doll stands as its own column, the
equipment heading appears once, and the bag lists gold, then loot, then the carried items. An animal or monster opens
the same window as a person, and the card's text stays sharp, with no dark rim, at any window size.

## Authority

- Owner: «Подпись параметры убираем… Имя надо менять прямо в карточке стата… убрать дублирующуюся надпись
  "Снаряжение"… первой строкой Золото, затем добыча, затем предметы… "обычный" это просто РЕДУКЦИЯ "детального"».
- Owner: «экран Отряда для животных… привести к одному виду с Отрядом человека».
- Owner, on the card's name with a dark blurred rim: «а мы артефакты текстов поправили? все равно какая-то рамка».

## Rules

1. Both modes come from one builder; the normal mode's rows are the detailed mode's rows, in order, less named parts.
2. No heading stands over the card, and no heading repeats; the equipment heading stands once in the detailed mode.
3. The name is edited by one input on the card, in both modes, empty or not; no other row edits it.
4. The bag lists gold first, then loot and corpse drops, then the carried items.
5. An animal or monster opens the person's window; every field the generic window edited for it stays editable.
6. At a window scale other than 1 the card's glyphs are drawn by the text overlay, scaled with the card.
7. The detailed window keeps the width rule of the window builder; the unit window adds none.
8. A unit's typed draft survives a click on a trigger that uses the unit and Back.

## Code

| kind | its one builder | change |
|---|---|---|
| unit window | `pkg/ui/mapeditorobject.go` objectSpec native branch | `pkg/ui/mapeditorunitcard.go` unitModes: a mode drops sections and keys as data |
| plain view | `pkg/ui/mapeditorplain.go` plainViews | unit table removed; serves structures, sacks and stock only |
| unit field sections | `pkg/game/mapinspectionedit.go` unitSections | every unit kind; the bag sorted gold, loot, items |
| editor window | `pkg/ui/editorpopup.go` layoutPopup, placeWindow | a held row is no field; the card goes to a band past the room (cardPastRoom) |
| text overlay | `pkg/render/text` capture, `pkg/render/textsmooth` Explain, Erase, composite | a call carries a scale; a scaled glyph cell is a block |
| scaled card draw | `pkg/ui/editorpopup.go` card draw | the blit's glyphs go to the overlay through scaleCapture |

## Proof

| test | rules |
|---|---|
| `TestEditorUnitWindowNormalModeIsTheDetailedModeReduced` | 1 |
| `TestEditorUnitWindowCardEditsTheNameUnderNoHeading`, `TestEditorUnitWindowEditsAnEmptyNameOnTheCard`, `TestEditorViewsPlaceEveryHostField` | 2, 3 |
| `TestEditorUnitBagListsGoldThenLootThenItems` | 4 |
| `TestEditorAnimalOpensThePersonsUnitWindow` | 5 |
| `TestEditorUnitCardTextIsDrawnByTheOverlayScaled`, `TestSettleReadsAScaledGlyphByBlocks` | 6 |
| `TestEditorDetailedUnitWindowIsAsWideAsItsContent`, `TestEditorPersonWindowTakesTheSizeItsRowsNeed` | 7 |
| `TestScriptUsedByBackPreservesDraft` | 8 |

Release tests at the release: none. Evidence: `review/story1679-unit-window/RESULT.md`.

## Open debt

- `pkg/ui/screens/unitcompact.json` still describes a compact view the editor does not read.
