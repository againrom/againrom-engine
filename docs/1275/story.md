# 1275 - remaining special tooltips

## Result

Three hover surfaces now follow the research claims at their grades: the
spellbook popup, the character generation attribute rows and the
map-selection list. The inherited control hints stay unbound because no claim
names the control that carries any text source. No simulation, save or hashed
state changed.

## Authority

Knowledge pin k118. TEXT-080, TEXT-081 (spellbook), TEXT-082 (attribute rows),
TEXT-083 (map list), TEXT-084, TEXT-085 (inherited hints), TEXT-HOVERPAINT-053
(popup layout, unchanged).

## As-built behaviour

### Spellbook popup

The mission book and the shop's Book toggle compose, over every selected actor
that knows the cell:

- the spell name and `main[117]: mana`, the mana value taken from the last
  knower;
- `main[118]` damage, summed over the knowers, only for a Damaging row and a
  nonzero maximum;
- `main[123]` range and `main[124]` duration as minimum and maximum, each
  omitted at a zero maximum; duration is ticks times 0.0625 at `%5.1f`, with no
  unit;
- one caption of the seven blocks in code order 182, 183, 184, 185, 187, 186,
  217, the last block with a value winning. The level is skill plus Mind minus
  30 clamped to 0..100. Per book cell: 182 on spells 24 and 7, 183 on 5, 16,
  10 and 22, 184 on 12, 185 on 23, 186 on 14, 217 on 18. Templates and the
  equal-pair rule follow TEXT-080. Caption 183, 185, 186 and 217 need a nonzero
  maximum; 182 and 184 always show a held value. Caption 187 and spell ids 17,
  27 and 28 are outside the book's 24 cells and never show.

Compared with the previous build: range moved after damage; range, damage and
duration omit at zero; the damage range collapses when equal; duration lost
its unit and shows one decimal; the captions are new; a multi-unit selection
folds instead of reading the first knower.

### Generator attribute rows

On the detailed page, per attribute row:

| Region | Text |
|---|---|
| plate left of the value | `main[155+i]` |
| value rectangle | `main[15+i] = value` |
| raise control | signed cost of the next point, comma-grouped |
| lower control | signed refund of the current point, comma-grouped |
| free-points box | `main[273]` |

The grouping output shape is Medium in TEXT-082. The previous build answered
the raise and lower controls with the attribute prose. The page's status line
still states the same signed text.

### Map-selection list

The engine's debug-font map list draws a size column and the `.alm` payload
`+0x70` and `+0x74` words at 300, 390 and 420 pixels from the list edge for
each row with a decoded size. Hover chooses by the cursor x alone: the row's
decoded description left of 300, then `dialogs.txt[134]`, `[135]` or `[136]`
for any row and for the empty slots of the list. `alm.Info` carries the two
words.

### Inherited hints

Not bound. TEXT-084 and TEXT-085 name constructors, a setter and text indices
by address. Population checked in the EN install: `dialogs.txt[23]` and `[75]`
read as music-unavailable messages, `[117]` as the multiplayer address prompt,
`[118]` as the map list's own title (its getter replaces the inherited one),
`patch.txt[52..54]` as "Shadows", "Dynamic lighting" and "Object animations".
A text match does not establish a receiving control (TEXT-HOVERSET-049), so no
control is bound.

## Divergences

DIV-1882 (spellbook values), DIV-1883 (attribute rows), DIV-1884 (map list),
DIV-1885 (inherited hints). DIV-1886 and DIV-1887 are unused. DIV-1270 keeps
lifecycle and layout.

## Proof

- `pkg/game/spellcaption_test.go`: every caption formula, template and
  min/max fold; `spell_test.go` fold and omission rules.
- `pkg/ui/tooltip_special_test.go`: column choice by x, empty caption, row
  columns, attribute rows with signs and grouping.
- `pkg/formats/alm/info_test.go`: `Word70` and `Word74`.
- EN and RU release witnesses, built from the installed text tables and the
  claim formulas, each with a loss control, registered in
  `internal/gatedtests/testdata/population.txt`:
  `TestReleaseSpellbookCaptionHoverIsBuiltFromTheClaim`,
  `TestReleaseGeneratorAttributeRowsHover`,
  `TestReleaseMapListHoverChoosesByColumn`. Each writes the composed popups as
  PNGs under `AGAINROM_TOOLTIP_SHOTS`, or a temporary directory when unset.
- Offscreen screenshots: `review/story1275-tooltips/{en,ru}/`.

## Open debt

- Spellbook values come from the engine's spell rows, not the original's record
  bytes; the fold's skip tests are not modelled (DIV-1882).
- Attribute rectangle positions and the enabling flag (DIV-1883).
- Map list: size relation to the record's `+0x14`/`+0x18` minus 16 and the
  description block length (DIV-1884).
- Inherited hints: the receiving control of each text source (DIV-1885).
