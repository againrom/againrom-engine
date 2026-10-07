# Provenance — 0116-compact-panel

Research submodule pin: `87a256d`. Claims read with `go run ./tools/claim -full <ID>` from the pin.

## What research supplies

| Claim | Confidence | What it settles here |
|---|---|---|
| `UNIT-PANEL-011` | **High as a negative**; the positive question is Unknown *by construction* | The original's panel cache is addressed by a **computed index** from `actor+0x14a` on, so `EnumRefs disp:14b` returns four hits and no reader at all. Its own words: *which cached value appears at which position, and whether the display has a multi-selection mode, cannot be settled without the index's own table and the drawing loop*, and a consumer reproducing the panel therefore has the **value set** and its **arithmetic** on evidence and its **layout** on nothing. |
| `UNIT-GATE-013` | High | The panel is not lying: the same three-way difficulty arithmetic reaches the simulation actor at spawn, so the numbers this window states are the ones a blow reads. |
| `HERO-SHEET-038` | High | The character sheet prints `[base, base + spread]`. The `DAMAGE` row already composes exactly that, and this story does not touch it. |

## What is OURS by choice

**The whole layout, and this is the one story on this pipeline where that is the finding rather
than an admission.** `UNIT-PANEL-011` establishes positively that the original's arrangement cannot
be recovered from the executable without the interface layer, which was scoped out. So:

- which values share a line, and which pair with which;
- the order the rows are in;
- every label, including the abbreviations `DMG`, `HIT`, `DEF`, `ABS`, `REACT`, `PROT`, `RES`;
- the column gap, the padding, the corner, the colours, the width floor;
- the rule that a right cell with nothing beside it slides into the left position

are all authored. Nothing here is claimed to reproduce anything, and no evidence was sought for it —
`UNIT-PANEL-011` says in advance that none exists.

The **value set** is not ours and is untouched: it is `0082`'s, `0109`'s and `0113`'s, each on its
own claims, and this story adds and removes nothing from it.

## What is open

Nothing this story needs. `UNIT-PANEL-011`'s positive question — the original's own arrangement and
whether its display has a multi-selection mode — stays Unknown, and closing it needs the index
table and the drawing loop, which is a research round rather than a story.

`PanelLayout` is the substitution point: the original's arrangement, if it is ever decoded, arrives
as a different value of that type and moves no drawing code. This story widened that seam rather
than narrowing it, since a two-cell row is expressible in it and was not before.

## What was removed

Nothing. The pin is the one `0113` bumped to and is frozen for this story (SDD S-5). All three
claims above read `active` through the reader's own retraction cross-read at that pin.
