# Provenance — 0058 hit test lift

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 (which height a placement is lifted by) | `TERR-SPR-039` | High |
| FR-1 (the sign: a raised cell is drawn up the screen) | `TERR-SPR-038` | High |
| FR-7 (the lattice rides the placement surface, not the terrain mesh) | `TERR-SPR-039` | High |

`TERR-SPR-039` supplies the surface a *placement* stands on, and it is not the surface the terrain is
drawn on. The engine builds a second grid — the mean of a cell's four corner heights, divided by four
truncating toward zero — and every sprite path reads that one; no sprite path reads the terrain's own
per-corner grid. The row also measures how far apart the two are: over 136 291 interior cells of
three maps they agree on 9.26 % of sloped cells, and where they differ `mean − corner` runs
**−73 … +55** rows, with |Δ| > 4 on 30.55 %. That measurement is what decides FR-7's surface.

`TERR-SPR-038` fixes the sign: `dstY = row*32 + 16 − anchorY − alt`, the altitude **subtracted**, so
higher ground moves a placement up the screen. Our `AnchorHeight` and the negation the marker path
applies are that rule already; the defect was never in the lift, it was that only one side of the
seam took it.

**What is deliberately *not* in this table.** `TERR-GEOM-036` (d) — the engine's cell picker
`R0377` bounding a cell by an exact lerp of the projected mesh, "the lerp is what the engine
hit-tests against" — was cited here for FR-1 in the first draft and is withdrawn. It is High, and it
is about the **ground** pick (spec FR-10, out of scope), and it resolves through the **corner mesh**,
which is the surface FR-7 argues against. Read as backing for FR-1 it would point the pick at the
terrain's mesh rather than the placement's. It is worth knowing and it is not evidence for this
contract.

## Ours by choice

| What the spec fixes | Why it is ours |
|---|---|
| FR-1 — a unit's hit target is its **cell's footprint**, not its sprite's rectangle | Nothing published says how the original resolves a click or a drag to a *unit* — see *Open*. The owner ruled by reporting the bug, and the cell footprint is the rectangle the selection rim already draws, so the pick and its own feedback become one thing. A sprite rectangle would need a rule for an entity that draws no sprite and a depth rule where two bodies overlap; neither exists. |
| FR-4 — the lowest id wins where two hit targets hold one point | The rule 0028 FR-7 already settled for two units on one cell, carried unchanged to two cells whose lifted footprints overlap on a cliff. |
| FR-7 — the lattice is a **diagnostic overlay**, not a reconstruction | Nothing published mentions a cell lattice drawn over the original's map; the ledgers were swept for one. It asserts nothing about the game. |
| FR-7 — the lattice rides the **placement** surface, and its size floor | The one place the two decoded surfaces force a choice; `TERR-SPR-039`'s numbers decide it, and `plan.md` DD-8 states the cost. The floor is legibility and cost, ours entirely. |
| FR-7, FR-8 — the glyph, its thickness, its colour and its key register | Diagnostic design, in the register the five existing glyph colours were authored in. |

## Open

**How the original resolves a click or a box drag to a unit is not published.** Stated positively:
the ledgers were swept and carry no row. FR-1 is therefore AUTHORED — a verdict, not a hold.

**And the nearest positive evidence points the other way, which is worth saying out loud.**
`TERR-SPR-041` records the object path's occlusion test feeding `pick(x, dstY)` and
`pick(x, dstY + frameHeight)` into that same cell picker — the row range of a **sprite rectangle**,
not of a cell. That is a cull and not a mouse hit test, so it settles nothing about picking; but it
is the only published place where a screen rectangle stands for a unit, and it is the sprite's. FR-1
authors against it deliberately, on the owner's rule and on the two structural reasons above. If a
later story opens the question, this is where it starts.

`TERR-SPR-041` also records a "second placement model" building a unit's screen rectangle at
`L10286…L10287`. That is **not** a hit test either: `TERR-SPR-067` reads the same expression as
the unit body's own draw destination.

One more follows and does not block this story: the **destination** of a move order still resolves
through the flat lattice (spec FR-10), which is the same defect class in the ground pick that
`TERR-GEOM-036` (d) would settle if it were opened.

## Removed

`TERR-GEOM-036` (d) as backing for FR-1 — see the note under *Backing*. Nothing else carried into
this story was dropped, and no spec statement lost its evidence.
