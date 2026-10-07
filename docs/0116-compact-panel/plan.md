# Plan — 0116-compact-panel

## Shape

Three tasks, in this order, because the first is the instrument the last two are judged with.

1. The measuring tool and the stated-set reader — no layout change at all, so the numbers it prints
   are the **baseline**.
2. The composition: a row may carry a second cell.
3. The shipped layout: the same twenty-one fields in twelve rows.

## DD-1 — a row's second cell is a POINTER to a small type, not two more fields

`PanelRow` gains `Right *PanelCell`, where `PanelCell` is `{Field PanelField; Label string}`.

Not `Field2 PanelField` beside `Label2 string`: `PanelField`'s zero value is `PanelFieldName`, so a
row that named no second field would silently name the **name** field, and every existing literal in
this repo — the panel's, the readout's, the tests' — would have to be edited to say "none". A nil
pointer has no such middle state, and `PanelLayout` is already non-comparable (it holds a slice), so
a pointer inside it costs nothing that was being relied on.

`PanelCell` is a new type rather than a second `PanelRow`, because `At` — a placed layout's own
offset — belongs to the row and not to a cell inside it.

## DD-2 — the resolution seam keeps its shape: `panelItem` gains a second cell

`panelItem` becomes `{label, value string; right bool; rightLabel, rightValue string; at
image.Point}`. `panelItems` resolves the left cell, then the right one if the row names one, and
**drops the row only when neither resolved**.

This is the seam `0060` split the composition at, and it stays split: everything below it is
geometry over `panelItem`s and knows nothing of fields. The readout builds `panelItem`s with no
right cell and is therefore unmoved — which is FR-6, obtained by construction rather than by a
guard.

## DD-3 — a right cell with no left cell SLIDES LEFT, at resolution time

FR-3 is applied in `panelItems`, by moving the right cell's label and value into the left cell's and
clearing the right. Not in the drawing, and not in the layout author's head.

The alternative was to require every authored pairing to put an always-present field on the left,
and to leave a hole when that was got wrong. That is a rule the type cannot enforce, that fails
silently, and that would have constrained which values may share a line for a reason that has
nothing to do with reading the panel. One statement in the resolver removes the whole class.

## DD-4 — the right column's origin is ONE number, computed over the drawn rows

`layoutLines` walks its items twice. The first pass measures every left cell and takes the widest;
the right column's x is that plus `PanelLayout.ColumnGap`. The second pass places.

Two passes rather than one because FR-2 is a property of the whole box: a per-row right origin is
what makes a two-column list look like a ransom note. Computing it over **drawn** rows — the items
the resolver already dropped — is what keeps P-1 true without a correction step.

**Only a row that carries a right cell contributes to that width.** A full-width row is not in the
grid, so it does not place the grid; it still widens the *box*, through `panelBox`, which is where a
long name has always been paid for.

*This paragraph replaces what it first said,* which was that a full-width row contributes like any
other because excluding it would make the column jump and including it "costs at most a few pixels
of unused gap". It costs the whole of that row's width. The instrument measured it: the party
panel's widest row on mission 10 is a weapon name at 235 pixels, which pushed the second column to
249 and the box to **363 × 229** against **255 × 337** before — a 32 % cut in height and none at all
in area. Corrected, the same panel is **268 × 229**. The jump the first version was avoiding is
real but small, and it is a jump in a column origin rather than in the width of the box.

The correction landed after T2 and T3, in its own untrailered commit, because it is a defect in this
plan rather than a task nobody did. It is the story's own instrument catching the story's own
design error, which is the argument for building the instrument first.

## DD-5 — the row's measured width covers both cells

`panelLine` gains `rightX`, `rightLabel`, `rightValue`, and its `width` becomes
`max(leftWidth, rightX + rightWidth)`. `panelBox`'s formula is untouched — it already takes
`at.X + width` — so the fitted box covers the second column by the same arithmetic that covered the
first, and FR-4 needs no guard.

## DD-6 — `ColumnGap` is a layout field, not a constant

It is the substitution point's own rule: everything the panel looks like is a value of
`PanelLayout`. A zero `ColumnGap` is legal and means the two columns touch, which is the author's
business.

## DD-7 — the shipped pairing, and why each pair

Thirteen rows in the table, twelve drawn for a fully-known subject with no mana and no mark, six for
a placed one. Every one of the twenty-one fields appears exactly once (FR-7).

| # | left | right | why together |
|---|---|---|---|
| 1 | `SELECTED` | — | states the selection, not the unit; present only above one |
| 2 | *name* | — | the title, unlabelled, and it can be long |
| 3 | `HP` | `MANA` | the two pools |
| 4 | `CELL` | `SPEED` | where it stands and how fast it gets elsewhere |
| 5 | `BODY` | `REACT` | the four statistics, in the chargen panel's own order |
| 6 | `MIND` | `SPIRIT` | — |
| 7 | `SKILL` | `XP` | the trained slot and what the sheet cost |
| 8 | `WEAPON` | — | a name, and it is the widest thing the panel draws |
| 9 | `DMG` | `HIT` | what a blow throws |
| 10 | `DEF` | `ABS` | what a blow meets |
| 11 | `SWING` | `ALWAYS HITS` | the cadence and the mark that skips the roll |
| 12 | `PROT` | — | five numbers; a family fills a line by itself |
| 13 | `RES` | — | — |

The four statistics keep the chargen panel's read-back order, which is the only ordering in this
layout that is not ours. `XP`, `PROT` and `RES` move out of the middle of the combat block, where
`0113` had to put them without moving anything, and into the character block they belong to.

Labels shorten to `DMG`, `HIT`, `DEF`, `ABS`, `REACT`, `PROT`, `RES`. They are ours
(`UNIT-PANEL-011`), no value changes, and the width they buy is spent on the second column.

Row 7 is the one pairing where the left cell can be absent while the right is present — a hero who
trained nothing has no `SKILL` and still has `XP`. DD-3 is what makes that a non-event.

## DD-8 — the tool starts a real mission and takes two real subjects

`cmd/paneldump` resolves the asset root from `-assets`/`AGAINROM_ASSETS` (golden rule 3), opens the
archives, loads the font and the unit registry, and starts the mission through `game.StartMission`
with `game.LoadDefinitions` — missionrun's own pairing, so the tool and the game arm one party.

It walks the started world's entities in id order and takes the **first** for which the loader knew
a character and the **first** for which it did not. Both are the mission's own; neither is a fixture.

The character comes from `game.PartyCharacters`, a thin exported wrapper over the map the front end
already builds, in a **new file** so that `pkg/game/world.go` — which another lane is editing — is
not touched. The rest of the subject is a field-for-field copy off the simulation entity, which is
what the per-frame push is; a tool deriving any of it differently would measure a window nobody
sees.

## DD-9 — the stated set is a `panelItem.String`, exported through `PanelStatement`

FR-12 gives both the tool and the tests a reading of the panel with no font in it. It is one method
on the already-resolved item plus a three-line exported walk, so "what it says" and "how big it is"
cannot drift.

## DD-10 — the compaction is asserted with LITERALS

AC-6's counts — 12 and 6 — are written as numbers in the test, not recomputed from the layout. A
test that derived the expected count from `AuthoredPanelLayout` would pass for any layout at all,
which is the defect `0113` shipped and caught. The pixel numbers stay out of the tests entirely:
they need the game's font, so they belong to the tool and to verification.md.

## Risks

- **The readout shares `layoutLines` and `composeItems`.** Its layout names no right cell, so both
  new passes are inert for it; its existing picture tests are the witness (AC-4).
- **`pkg/ui/inventory.go` is being edited by another lane.** Nothing here touches it.
- **`PanelSubject` must stay comparable.** Nothing in this story goes near it; AC-10 pins it anyway.

## Traceability

FR-1 -> DD-1, DD-2. FR-2 -> DD-4. FR-3 -> DD-3. FR-4 -> DD-5. FR-5 -> DD-2, DD-5. FR-6 -> DD-2.
FR-7 -> DD-7. FR-8 -> DD-7. FR-9 -> DD-7, DD-10. FR-10 -> DD-7, DD-10. FR-11 -> DD-7.
FR-12 -> DD-9. FR-13 -> DD-8. FR-14 -> DD-8. FR-15 -> nothing goes near it; DD-1 keeps the second
cell out of `PanelSubject` and inside `PanelLayout`, which is not comparable and never was.
FR-16 -> DD-8 adds a row for the tool only; `pkg/ui`'s own row is untouched.
