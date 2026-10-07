# Analysis — 0116-compact-panel

## What was asked

The owner, 2026-08-09: the full statistics on the info window, the window made compact, and made
reusable for ordinary units too.

`0113` shipped the first clause: `XP`, `PROTECT` and `RESIST` reached the panel. This story is the
second and the third.

## What we did not know, and what the measurement said

**How big the window actually is.** Nobody had measured it. A row count off `AuthoredPanelLayout`'s
table is not a measurement — rows drop when a field has nothing to state, and the box is fitted to
the survivors through the *game's own* font, which no synthetic test may read (golden rule 2). So
the first thing built was the instrument: `cmd/paneldump` starts a campaign mission exactly as the
front end does and composes the panel for two real subjects out of the world it produces.

Mission 10, `scenario/10.alm`, font `font1`, line height 15 — the baseline, before any change:

| subject | rows drawn | box |
|---|---|---|
| the party's swordsman, character known in full | 18 | **255 x 337** |
| `Human ClubMan`, a unit the map placed, no character | 9 | **168 x 175** |

The window is 1024 x 768. The party panel was **44 % of the window's height**. That is the number
the word *compact* was missing.

**Whether the panel was already reusable for an ordinary unit.** It was — and that is a finding
rather than an assumption. `ui.PanelSubject` names no party type; `Char.Known` gates the six
character rows and `Combat.Known` the combat block, and the clubman above states nine rows with no
empty ones and no zeroes standing in for absent values. So the owner's third clause was already
half true: the ordinary-unit case needed to be made **good**, not built.

**What the ordinary case actually lacked** was proportion. Nine rows of one value each, in a box
whose width was the layout's 168-pixel floor, is a tall thin column stating eight numbers that
would sit comfortably on four lines.

## What this story therefore is

A layout story and only a layout story. The panel states exactly the fields it stated before — the
same twenty-one, the same values, resolved by the same `panelText` — laid out in fewer rows.

Two things were weighed and dropped, both because they cost the one thing the story is measured on:

- **A section rule** between the character block and the combat block. A drawn rule is height, and
  the block boundaries are already legible from where the paired rows change subject.
- **Aligning every row's value on one column.** There is no rule for it that is both tidy and
  stable when a row drops: the widest label would set the origin, and `SELECTED` — present only
  above one selected unit — would then move every value in the box when a second unit is picked.

The one thing worth recording that this story did **not** do: nothing was found missing from the
panel's value set, so nothing is deferred on that axis.
