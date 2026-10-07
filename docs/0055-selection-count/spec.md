# Spec — how many units are selected

## Problem and current behaviour

The map screen's information panel describes **one** unit: the first of the current selection. Drag
a box over four and it names one of them, states that one's health and cell, and says nothing at all
about the other three. Nothing on screen states how large the selection is, so a player cannot read
back what a drag caught, and cannot tell a selection of one from a selection of many except by
counting marks on the ground.

This story puts the number on the panel.

## What is stated, and when

**One new statement: how many units the selection presently holds.** The count is taken over the
same set every other reading of the selection takes — the entries the current snapshot still holds
and still reports alive or downed — and never over the raw selection, so an id that has died or left
the world is not counted on screen while being invisible everywhere else.

It is a value of the panel's **closed list of fields**, and it is the second field that can be
**absent**:

| Units present | The count |
|---|---|
| none | there is no panel at all, and that is unchanged |
| one | the count has **no value**, and its row is skipped |
| two or more | the count is stated, in decimal |

**A skipped row costs no space**, by the rule already in force. So a panel over a single-unit
selection is **the picture it was before this story, pixel for pixel** — not a similar picture, the
same one.

**The described unit does not change and is not replaced.** With several selected, the panel still
describes the first of them in full and states the count **beside** that description. Which unit is
described, and the order it is chosen in, are untouched.

## Where the number sits in the panel

The count is a statement about the **selection**, not about the unit — the only such value the panel
carries. It reaches the composition on the same value that carries the described unit's name, health
and cell, and that placement is load-bearing rather than convenient: what the picture is a function
of is exactly what decides when it is redrawn, so **a selection that grows or shrinks while the
described unit stays the same must redraw on the next frame**, and that follows only if the count is
part of that value.

It is drawn as **an ordinary row of the layout** — a field, a label, a colour, a position, in the row
order the layout gives. Nothing in the drawing path knows this row from the other three. A
replacement layout may relabel it, move it, reorder it or omit it entirely, and a layout that omits
it draws no count for any selection; that is the test that the row really is ordinary and not a
special case wearing a row's clothes.

## The wording is ours

**The label is AUTHORED.** No decoded source in this tree names the text the original puts on screen
beside this number, nor where that text is held, nor whether the original states it on this panel at
all. What ships is our own label in the authored layout — disclosed here and in the evidence ledger
exactly as the rest of the panel's appearance is, and replaceable at the same single value. It is
not a reconstruction and is not claimed as one.

The **number** is not authored: it is the size of the selection this build holds.

## Functional requirements

- **FR-1 (the count is a field)** — The panel's closed list of fields gains one: how many units the
  selection presently holds, formatted in decimal. It has a value when that number is **two or
  more** and no value otherwise; a field with no value is skipped by the rule already in force, so
  its row is neither drawn nor allotted space.

- **FR-2 (what is counted)** — The number is the size of the same set the described unit is drawn
  from: the selection's entries that the current snapshot holds and reports alive or downed. Ids the
  snapshot does not hold, and dead ones, are not counted.

- **FR-3 (the description is unchanged)** — Which unit the panel describes, in what order it is
  chosen, and what is stated about it are exactly as before. The count is stated in addition to that
  description and never instead of it.

- **FR-4 (single and empty selections are untouched)** — For a selection presenting one unit, the
  composed panel is byte-identical to the panel this story inherits for the same unit, layout and
  font. For one presenting none, there is no panel.

- **FR-5 (through the layout seam)** — The count is one ordinary row of the layout value: its label,
  colour, order and position are the layout's; the composition treats it as it treats every other
  row; and a layout carrying no such row draws no count for any selection. The authored layout
  carries one, and its label text is ours.

- **FR-6 (redraw)** — The composed picture is a function of the count as well as of the described
  unit, the layout, the font and the placement area. A frame in which only the count changes rebuilds
  it; a frame in which nothing changes does not.

- **FR-7 (what does not move)** — No simulation, digest, save, codec, map-loading or map-drawing
  behaviour changes. No entity gains a field, nothing inside the determinism wall is touched, and no
  package's permitted import set is widened. The selection itself is unchanged: nothing is pruned,
  added or reordered. A viewer holding no font still draws no panel and fails at nothing. No game
  bytes enter the repository.

## Acceptance criteria

Synthetic throughout: every fixture is built in test code, and no test reads a game install.

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | counts of 0, 1, 2 and 17 | asking the count field for its text | no value at 0 and at 1; `2` and `17` at the other two |
| AC-2 | a subject presenting one unit, the authored layout and a font | composing it, and composing the same subject against that layout with its count row removed | the two pictures are byte-identical |
| AC-3 | a subject presenting four units | composing it against the authored layout | the picture carries the count row's label and the number 4, and the other three rows state what they stated for the same unit before |
| AC-4 | a viewer whose selection holds five ids — one dead, one absent from the snapshot, three alive | asking for the frame to present | the count stated is 3, and the described unit is the first of those three in the selection's own order |
| AC-5 | a viewer presenting a panel over three units | a frame in which one of the three dies, then a frame in which nothing changes | the picture is rebuilt on the first and not on the second, and the count then stated is 2 |
| AC-6 | a layout whose rows omit the count, and one placing the count row at its own offset under its own label | composing each over a subject presenting four units | the first states no count anywhere; the second draws that label and that number at that offset |
| AC-7 | a viewer with nothing selected, and a viewer never given a font with several selected | asking for the frame to present | no panel in either |

## Properties

- **P-1** — The number stated is exactly the size of the set the described unit is chosen from, for
  every selection and every snapshot. It is never the raw selection's size, and never larger than the
  number of entries the snapshot holds.
- **P-2** — For every layout, font and unit, a panel over a selection presenting fewer than two
  units is identical to the panel the same inputs produced before this story. The single-unit screen
  cannot regress.
- **P-3** — The panel's appearance remains a function of the layout value alone: two viewers
  differing only in that value differ only in the panel's pixels, and one whose layout omits the
  count row is one of them.
- **P-4** — No simulation, determinism, digest or decoded-format behaviour is touched, and no entity
  field is added.

## Out of scope

- **Reconstructing the original's readout.** Its wording, its place on the screen, and whether the
  original states it on this panel at all are undecoded. What ships is authored and replaceable at
  one value.
- **Describing more than one unit** — no second subject, no summary of the others' health, no
  cycling between them. The panel still describes one unit and now says how many there are.
- **Selecting**, and what a selection is. No id is pruned, added or reordered, and no input
  behaviour changes.
- **Structures and objects**, which are not selectable.
