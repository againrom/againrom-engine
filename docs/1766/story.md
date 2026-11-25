# The second-game map editor names its record fields and paints passability by move kind

## Result

On a second-game map every field of a unit, a structure and a sack now has a name, and a field no game reader was
found for says so. A sack item's word says that a nonzero word puts the item in its owner's hand and zero puts it in
the bag, and that the English game reads it only under one session condition. The passability view colours each cell
by the move kinds the game's route search closes there and names them on hover. Its brush closes a cell to move kind 1
or opens it, clearing the cell's object byte that would close kind 2 as well.

## Authority

- Owner: «Редактор ром2 давай доделаем»; epic «Редактор карт ROM2», gap groups G1 and G2.
- R2-ENGINE-1801, R2-ASSET-313: no reader of unit fields 0x0a and 0x20 is found (Medium).
- R2-ENGINE-513, R2-ENGINE-535, R2-ENGINE-537: unit 0x08 picks the built object; 0x10 holds Inn offer and no-drop bits.
- R2-ENGINE-1802, R2-ENGINE-1288, R2-ENGINE-1289: structure word 0x0c is read only on the Building arm (a zero test);
  the extension replaces the building row's size (Medium).
- R2-ENGINE-1803, R2-ENGINE-1328, R2-ASSET-254: a unit's sack item word nonzero to the owner's hand (Medium), zero
  to the bag; the English stage runs only at session word +0x74 = 0; a ground sack is placed with its 0x10.
- R2-ENGINE-1805, R2-ENGINE-1804, R2-ENGINE-1340, R2-ENGINE-1665, R2-ENGINE-809: value 1 closes kind 1, 5 kinds 1 and 2,
  the ring's 0x1f all three (Medium); flag 0x2000, tile class 8 and tile bits 0x200 store 1, a type-3 byte 5 after them.
- R2-ENGINE-1667: which type-3 value draws no object is unread, so value 5 is shown, not painted.

## Rules

1. When a second-game unit, structure or sack field is shown, it carries a name, not its offset alone.
2. When the claims find no reader of a field, its name says no game reader was found.
3. When a unit's sack item word is shown, it names hand, bag, Medium and the English session gate.
4. When the passability view is on, each cell's fill and hover give the move kinds its value closes.
5. When the passability brush paints a second-game cell, it sets or clears flag 0x2000 and clears the type-3 byte.
6. When a first-game map is painted or its fields shown, the brush rows and fields are as before.
7. When an installed second-game map is opened and saved untouched, the file is byte-equal.

## Code

| kind | its one builder | change |
|---|---|---|
| per-game field table | `pkg/mapedit/second.go` secondUnitFields:102, secondStructureFields:191, secondLootFields:224 | names and marks as data |
| brush stroke | `pkg/mapedit/brush.go` impassableStroke:608, PlaneRule:92 | `ClearsObjects` row field |
| second-game marks | `pkg/mapload/groundarms.go` MarkRows:88, MarkValue:153, PlaneValueRows:139 | class-8 mark, values, kinds |
| inspection document | `pkg/game/mapinspection.go` inspect:311 | values and brush rows from the edit row |
| edit row of a format | `pkg/game/mapeditrow.go` editRows:179 | `passValues`, `passBrush` |
| passability overlay | `pkg/ui/mapeditorpassability.go` passabilityQuads:280, hoverPassability:204 | fill and hover by value; fills in `pkg/ui/editorink.go` |
| brush list | `pkg/ui/mapeditorbrush.go` brushRows:268, brushCommit:768 | rows from the document |
| word tables | `pkg/words/text/{en,ru}/strings.toml` | field, value and brush words |

## Proof

| test | rules |
|---|---|
| `TestSecondFieldsNameTheirReaders` | 1, 2, 3 |
| `TestSecondEditorFieldsAndCardsForUnitStructureAndSack` | 1, 2 |
| `TestMarkValuesCloseMoveKinds` | 4 |
| `TestPassValuesFillAndNameTheMoveKinds` | 4, 6 |
| `TestSecondMapPassabilityShowsAndSetsMoveKinds` | 4, 5 |
| `TestSecondPassabilityStrokeSetsTheFlagAndClearsTheObjectByte` | 5 |
| `TestSecondMapBrushesPaintHeightsPassabilityAndClearObjects` | 5 |
| `TestImpassableStrokeChangesOnlyBitThirteenOfTheDisc`, `TestFirstEditorUnitFieldListUnchanged` | 6 |
| `TestReleaseSecondMapSavesEveryInstalledMapUnchanged` | 7 |

Release tests at the release: `TestReleaseSecondMapSavesEveryInstalledMapUnchanged`,
`TestReleaseSecondMapEditsUnitStructureAndSackFields` on the EN and RU second-game installs.
Evidence: `review/story1766-rom2-editor-fields-named/RESULT.md`.

## Open debt

- Value 5 by the brush waits for the R2-ENGINE-1667 run: which type-3 byte closes kinds 1 and 2 drawing no object.
- Tile groups 13 to 15 wait for an R2-ENGINE-1665 extension; such a word is shown without a class-8 mark.
- The play effect of fields 0x0a, 0x20 and a nonzero word 0x0c waits for a play run (R2-ENGINE-1801, 1802).
