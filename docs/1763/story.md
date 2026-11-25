# The second-game map editor reads and edits traps, structure casters and type-3 bytes

## Result

On a second-game map the editor lists a trap's spell, power, cell, source and target, and a
structure caster's spell, power and scan radius. The map maker changes them, adds a trap, a
trap area or a caster bound to a structure from the palette, and removes them. A trap
refuses the teleport's event.

A cell's type-3 byte reads by its value, with what the game does with it and that its
picture is unread; the editor keeps it, and the object brush clears it.

## Authority

- R2-ENGINE-1698: a type-9 record of mode below 4 writes a six-byte cell event: event, power, source, target.
- R2-ENGINE-1776, R2-ENGINE-673: one event per cell; any event but 23 casts its spell on the cell.
- R2-ENGINE-1779 (Medium): event 23 reads no power; a teleport takes no power step.
- R2-ENGINE-1009, R2-ENGINE-1006, R2-ENGINE-1560: a mode-4 record is a caster keyed to a building: spell, power, radius.
- R2-ENGINE-1669: the caster reads none of its entries.
- R2-ENGINE-1667 (Medium), R2-ASSET-261: a nonzero type-3 byte blocks move kinds 1 and 2; its picture is Unknown.

## Rules

1. When a second-game trap is read, it lists spell, power, cell, source and target, and its label names them.
2. When a structure caster is read, it lists spell, power and scan radius.
3. When a trap or caster field changes, the save writes that byte alone; every other byte stays.
4. When a trap is given the teleport's event, the edit is refused with its reason.
5. When the palette adds a trap, a trap area or a caster bound to a structure, each saves in the claims' layout.
6. When a teleport is picked in the palette, no power step is asked.
7. When a removed trap goes, its cell event goes and the other records stay.
8. When a type-3 byte is shown, it reads as its value with its effect and stays in the file unchanged.
9. When an installed second-game map is opened and saved untouched, it saves byte for byte equal.
10. When a type-9 record is a trap or a mode-4 caster, it lists fields; this narrows rule 10 of story 1761.

## Code

| kind | its one builder | change |
|---|---|---|
| type-9 field table | `pkg/mapedit/second.go` secondEnchantmentFields | trap and caster rows as data |
| field check | `pkg/mapedit/field.go` FieldCheck | CheckCellSpell |
| new type-9 template | `pkg/mapedit/enchantment.go` SpellTemplateFields | takes the chosen spell |
| editor row | `pkg/game/mapeditrow.go` editRows | steps, scenery and trap words |
| type-9 record view | `pkg/game/mapinspectioncasts.go` inspectionEnchantment | trap word from castView |
| type-3 record view | `pkg/game/mapinspection.go` overlay records | label and detail from the row |
| cast spell choices | `pkg/game/mapinspectionchoices.go` castSpellChoices | spell catalog and teleport |
| trap area stroke | `pkg/ui/mapeditortraparea.go` trapAreaDrawn | power only when asked |

## Proof

| test | rules |
|---|---|
| `TestSecondMapCellRecordsReadWithNamedFields` | 1, 2, 8, 10 |
| `TestSecondMapCellRecordsChangeAsTheClaimsLayout` | 3, 4 |
| `TestSecondMapTrapAndCasterAddedAndRemoved` | 5, 6, 7 |
| `TestSecondMapTeleportRecordAddedChangedAndRemoved` | 6 |
| `TestReleaseSecondMapSavesEveryInstalledMapUnchanged` | 9 |

Release tests at the release: `TestReleaseSecondMapSavesEveryInstalledMapUnchanged`.
The install census is in the review report `review/story1763-rom2-editor-records/RESULT.md`.

## Open debt

- A type-3 byte moves and deletes only with the area copy seam the second game lacks (G4).
- What a type-3 value draws, and the client copy's readers, are Unknown (R2-ENGINE-1667).
- View areas and footprints (gap row 23) are not read; no claim in the pin states their layout.
- A deleted structure leaves its casters; the second game's casters do not go with their object.
- A trap's entry words beyond the cells and the caster dword's upper bytes are kept, unread.
