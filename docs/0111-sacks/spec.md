# Spec — a sack on the ground is drawn

## Terms

**Ground-sack list** — the ordered list of ground sacks a simulation world holds: a cell, a purse
and the item codes carried there. It already exists; this story adds no field to it and changes
nothing about how it is built, ordered, queried or serialized.

**Sack sheet** — the one decoded sprite sheet whose frames a sack is drawn from. One sheet serves
every sack on every map.

**Drawn band** — the single back-to-front sequence of cell-anchored content the window paints
between the terrain and the diagnostic markers. Today it holds the map's structures, the map's
static objects and the world's entities.

**Ground point** — the point of a cell that a cell-anchored sprite is measured against: the cell's
centre, displaced upward in the height-displaced geometry by that cell's own height and the
render's vertical origin, and not displaced at all in the flat geometry. That displacement is the
**whole** of the difference between the two geometries, and it is applied once.

## Problem

A world already holds the ground sacks its map authored, answers the mission script's question about
whether a given one is still there, and hands the list out to any reader that asks. **Nothing draws
one.** A player walking over mission 10 sees bare ground on all four cells the map put loot on, and
the only observable consequence of that loot is a trigger silently answering a question about it.

There is no sack art in the drawn band, no sheet decoded for one, and no path by which the world's
sack list reaches the window at all.

## Scope

Sacks become visible. Nothing becomes takeable: no sack is created, merged, emptied or destroyed by
anything this story adds, and the simulation is not touched.

## The contract

### The sheet

**FR-1** The build decodes **one sack sheet** at start-up: every frame of one archive address, in
the sheet's own order, converted into the same drawable frame the map's art already uses — pixels,
palette and size. A sheet that is absent, that the decoder refuses, or that carries no usable
palette yields **no frames**; that is **not an error**, does not fail start-up and does not stop a
map or a mission opening.

**FR-2** A sack draws the **first frame** of that sheet. **Every sack draws the same frame**,
whatever cell it stands on, whatever it carries and whatever the map is.

### What is drawn

**FR-3** **One sack is drawn per entry of the world's ground-sack list**, at that entry's own cell,
and **nothing else draws one**. No art switch, no diagnostic overlay flag and no mission state gates
it: a sack is world content, not the map's art.

**FR-4** The drawn set is **rebuilt from the world within the same refresh that rebuilds the drawn
entities**, so what lies on the ground is what the world holds at that moment and cannot be a
separately maintained copy of it. It is rebuilt at no other time.

**FR-5** A sack's frame is placed so that the **frame's centre pixel lands on the cell's ground
point**, through the same anchor, the same exact-rectangle cull and the same camera transform every
other cell-anchored sprite takes.

**FR-6** The displacement in FR-5's ground point is **the sack's own cell's** — never a whole
map's, never a neighbour's, and never one offset for the set — so sacks on a slope are lifted cell
by cell.

**FR-7** A sack is **merged into the drawn band by cell row**, so a sack on a further row is covered
by what stands in front of it and a sack on a nearer row covers what stands behind it. **At an equal
row a sack draws after both map-art planes and before an entity** — so a unit standing on a sack's
own cell stands in front of it. The band's **ground-decoration prefix** — the art drawn before the
whole band whatever row it stands on — is outside that ordering, and **a sack always draws after
it**, so a sack lying on a paved courtyard lies on top of the paving.

*Folded from hotfix `8e13e8f` — see `docs/hotfix/ARCHIVE.md#8e13e8f`.* Paint order within a row is
three-way and lexicographic on (row, tie): a living unit above, the sack under it, the corpse
under the sack (owner as author, 2026-08-09). Plan `DD-2`'s equal-row tie-break is void; the
ground tier is the zero value.

**FR-8** A sack takes **the same light row and the same tint** as every other sprite in the band, so
it dims and warms with the world clock exactly as the art around it does.

**FR-9** A sack **casts no shadow** and **draws no overlay layer**. Its art is the one sheet's frame
and nothing composited over or under it.

### Absence and refusal

**FR-10** Each of these draws **nothing at all** for the sack concerned, is **not an error**, and
leaves every other drawable in the band drawn exactly as it would otherwise have been: no sheet
loaded; a sack whose selected frame index falls outside the loaded sheet; a sack whose cell the
drawing side's own grid does not contain. **None of them removes the sack from the world**, and
none is reported as a failure to the player. The third is the drawing side being **total over a
list it does not own**: no world this build constructs can hold a sack outside its own bounds, so
the case arises only if the grid drawn and the world pushed ever disagree.

**FR-11** **The simulation is unchanged.** No world state, no tick, no ordering and no serialized
byte differs because of this story, and a headless run reports exactly what it reported before.

## Acceptance

| # | GIVEN | WHEN | THEN | Level |
|---|---|---|---|---|
| **AC-1** | an archive holding a sack sheet of several frames | the build loads its art | every frame of the sheet is available to draw with, in the sheet's own order (FR-1) | unit |
| **AC-2** | an archive with no sack sheet, or one the decoder refuses, or one with no palette | the build loads its art and a map opens | no sack is drawn, start-up succeeds, the map opens, and every other drawable is unchanged (FR-1, FR-10) | unit |
| **AC-3** | a world holding sacks at known cells and a loaded sheet | the window is refreshed | exactly one sack is drawn per list entry, at that entry's cell, in the list's order (FR-3) | unit |
| **AC-4** | a world holding sacks | two refreshes with no world change between them | the two drawn sack lists are equal (FR-4) | unit |
| **AC-5** | a loaded sheet and a sack at a cell | the sack is placed in the flat geometry | the placed frame's centre pixel is at the cell's ground point (FR-2, FR-5) | unit |
| **AC-6** | the same sack and cell | it is placed in the flat and in the displaced geometry | the two placements differ by exactly the cell's lift and the vertical origin in Y, and by zero in X (FR-6) | unit |
| **AC-7** | sacks, structures, objects and entities on known rows, including one of each sharing a row, and one ground-decoration entry on a far row | the band's order is built | the decoration entry comes first, the rest is ascending by row, and within the shared row the two art planes come first, then the sack, then the entity (FR-7) | unit |
| **AC-8** | a drawn sack and the world clock at a dark hour and at a bright one | the band is painted at each | the sack's pixels take the same light row and tint the sprite beside it takes (FR-8) | unit |
| **AC-9** | a drawn sack | the band is painted | no shadow entry is produced for it, and its painted pixels are the decoded frame's own, changed by nothing but the light row and tint of AC-8 (FR-9) | unit |
| **AC-10** | a drawing side given a sack whose cell its grid does not contain, and one whose frame index is past the sheet | the band is built | neither is drawn, no error is raised, neither is removed from what was given, and every other sack is drawn (FR-10) | unit |
| **AC-11** | a lawful install and mission 10 | the mission is opened and played | four sacks are drawn, at (20,65), (38,64), (12,50) and (10,14), and none at (36,51) (FR-3) | developer-run |
| **AC-12** | a lawful install, either root | the headless mission-10 drive is run before and after this story | it reports the same outcome at the same tick (FR-11) | developer-run |

**Error cases:** AC-2 and AC-10. Each yields the negative invariant P-3.

## Derived properties

**P-1** *(invariant)* For any world and any refresh, the drawn sack cells are **exactly** the cells
of that world's ground-sack list, as a sequence — same length, same order, no cell added, dropped or
repeated — before the placement's own off-map and frameless exclusions are applied.

**P-2** *(invariant)* For any cell, that cell's flat and displaced sack placements differ by exactly
the negated sum of the lift and the vertical origin in Y, and by **zero** in X.

**P-3** *(negative invariant)* For any absent, refused or palette-less sheet, any off-map sack cell,
and any out-of-range frame index: the world's ground-sack list is unchanged, no error reaches the
player, and the drawn band is exactly what it would have been with that sack absent.

**P-4** *(completeness)* For any two drawables of the **merged** band standing on different rows, the
one on the lower row is drawn first; for any two on the same row, the order is map art, then sack,
then entity, and **two sacks on one row keep the order the world's own list gave them**. The
ground-decoration prefix is outside the merge and precedes all of it, whatever row its members
stand on.

**P-5** *(idempotence)* Refreshing twice from an unchanged world yields equal drawn sack lists and an
equal band order.

## I/O examples

**Archive address.** The sack sheet is read from the graphics container at `backpack/sprites.256`.

**Mission 10's authored loot**, as the map decodes it — every cell below is **(column, row)**:
ground records at (20,65), (38,64), (12,50) and (10,14), with one item and no gold each, and one
non-ground record at (36,51) that is not a sack.

## Constraints

**Which of the sheet's frames a sack draws** is an external-content choice, and nothing decoded
names the rule:

| | Rule | Observable trade-off |
|---|---|---|
| **A** | The sheet's **first frame**, always | One authored assertion. Every sack looks alike; the rest of the sheet is unused until the real rule is known. **Chosen.** |
| B | A ladder on how many things the sack holds | Three authored assertions — that it is content, that it is count, and the step size. Draws the great majority of shipped sacks identically anyway, and never reaches the largest frames |
| C | A ladder on what the sack is worth | The best fit to the art, and **not implementable here**: an item's worth needs definition tables this build does not decode for three of the four classes mission 10 puts on the ground |

Being wrong costs a sack drawn at the wrong size and nothing else: no cell moves, no list changes,
and the rule is one function when the answer arrives.

**One sack per cell** is the world's own rule and this story inherits it: two drawn sacks can never
share a cell, so the band needs no tie-break between them.

## Out of scope

- **Picking a sack up**, and everything it needs: an actor container, an order, and the sack's
  destruction. Nothing here creates, merges, empties or destroys a sack.
- **Dropping**, **death leaving a sack**, and any other origin. The only sacks that exist are the
  ones the map authored, which is already true and stays true.
- **The map's non-ground loot records.** They stock an actor rather than the ground; they are not
  sacks and nothing here draws them.
- **The frame ladder**, until the selector is decoded.
- **The overlay layer and a shadow** — FR-9 states the divergence rather than deferring it.
- **Any pointer behaviour**: a sack is not hit-tested, not hovered, not selectable, not named in the
  readout and not the target of any order.
- **A drawing side opened without the game's own start-up path** — a diagnostic tool's window.
  It is handed no sheet and so draws no sack; FR-1 and FR-10 make that the ordinary frameless case
  rather than a special one.

## Verification mapping

AC-1..AC-10 and P-1..P-5 are CI-automatable and run in `go test`. AC-11 and AC-12 are developer runs
against a lawful install: AC-11 by opening mission 10 in the built binary, AC-12 by the headless
mission drive on both roots.

## Gate check

FR-1 → AC-1, AC-2. FR-2 → AC-5. FR-3 → AC-3, AC-11, P-1. FR-4 → AC-4, P-5. FR-5 → AC-5, P-1.
FR-6 → AC-6, P-2. FR-7 → AC-7, P-4. FR-8 → AC-8. FR-9 → AC-9. FR-10 → AC-2, AC-10, P-3.
FR-11 → AC-12.
