# 1003 — area overlays: where a cell effect is drawn, and on which frames

## The result

Five owner reports about how area spells are drawn (2026-08-15, and the 2026-08-16 reminder about
Wall of Fire's burning sprites) resolve to three defects in this build's draw path. After this story:

1. **A burning Wall of Fire cell plays only the full-fire frames.** It no longer cycles birth and
   fade. The frame law is `ANIM-WALLFIREFRAME-033`'s: five of the sheet's eleven frames, on a
   10-tick cycle.
2. **A staged area effect puts one short-lived burst on each cell it paints, and the bursts
   overlap.** Fire Sacrifice's two shells therefore read as an explosion spreading outward from the
   caster's own cell; Meteor Storm's 32 stages read as rocks landing at random points across the
   area at a visible rate; Acid Stream's six stages read as the cone growing away from the caster.
3. **Every spell sprite stands on its cell's centre.** It was drawn on the cell's top-left corner,
   16 px left and 16 px up of the ground point every other sprite in the renderer uses. That is the
   whole of the Acid Stream cone-origin report and the whole of the floating Wall of Earth report.

A fourth change follows from the first: the retained per-cell overlay is drawn for the four spells
that have overlay art and for no others.

**Observable result.** The game in `builds/1003-area-overlays/`, played on a lawful install. This
story moves nothing in `pipeline/check-milestone.sh`'s script-gap census; both mission figures are
recorded unchanged in `closure.md`.

## Claims

| Claim | What this story takes from it |
|---|---|
| `ANIM-WALLFIREFRAME-033` | Wall of Fire's frame law: `abs(counter/2 + screenX*screenY) mod 5 + 3`, so only frames 3..7 of the 11-frame sheet are drawn. The other three overlay arms take the record's own `Phases`. |
| `MAGIC-OVERLAYART-051` | The retained overlay's art is four baked immediates: mask bits 3, 7, 8 and 19 index projectile records 15, 23, 25 and 47. Which four of the six cloud spells have a sprite at all is an engine limit. |
| `MAGIC-OVERLAY-050` | A cloud puts a per-cell bitmask on the client, not objects: no object, no counter, **no phase** per cell. A cell cannot remember when it started burning. |
| `MAGIC-AREADRAW-049` | The three tick modes create three different drawables. A **cloud** creates no object. A **blast** creates one object at the centre, 22 ticks. A **staged** effect sends one message per accepted cell, each building one transient object of 16 ticks, 18 for `acid_stream`. |
| `MAGIC-RING-048` | The three staged cell generators, their stage counts (2, 6, 32), the three-tick stage clock, and that an accepted cell sends picture `2*spellId+9` once. Staged effects register no map layer. |
| `MAGIC-WALLFIRE-058` | `wall_of_fire` is the only overlay arm that falls through instead of exiting, so it and `wall_of_earth` can be drawn at one cell. |
| `ANIM-PROJ-026` | The projectile draw centres the art on the **object's own position** by the registry's `Width/2` and `Height/2`. |
| `ANIM-MSG-005` | An actor standing on a cell has both fine position coordinates at `0x80` — half of the 256 units a cell spans. A cell's own point is its centre. |
| `TERR-SPR-040`, `TERR-SPR-038` | The decoded sprite ground point, already built in `StaticAnchor`: `col*CellSize + CellSize/2`. |

## UNKNOWNs

- **What `screenX*screenY` denotes** in `ANIM-WALLFIREFRAME-033`'s frame law. The claim gives the
  formula and names the two `.text` immediates that produce the 5 and the 3; it does not say in
  which units the spatial term is measured. Expected divergence row.
- **Whether the meteor sprite descends.** No claim names a vertical fall for Meteor Storm's per-cell
  object. `MAGIC-RING-048` and `MAGIC-AREADRAW-049` place the object at the accepted cell and give
  it a 16-tick life; picture 51 runs the raw clock, so its sheet plays through once over that life.
  The owner's directive proceeds on his intent; the fall itself is not decoded. Expected divergence
  row.
- **Which of the two overlay draw passes is which.** `MAGIC-OVERLAYART-051` states the fourth
  argument splits the four arms into two passes — bits 3 and 19 in one, bits 7 and 8 in the other —
  and names both call sites inside the map draw. It does not say what separates the two passes on
  screen. Expected divergence row.

## Aspects

| Aspect | Applies |
|---|---|
| Data | yes — the overlay spell-to-picture table in `pkg/data` |
| Runtime state | yes — the client's transient burst list |
| Simulation | yes, narrowly — a per-tick observation of painted cells, on `castObs`' own pattern. No canonical world field, no byte-form change |
| Player input | no |
| AI | no |
| UI/HUD | yes — the ground point and the overlay frame law |
| Triggers/scripts | yes — a script-authored area cast paints through the same arm |
| Inventory/equipment | no |
| Persistence/save-load | no — nothing added crosses the byte form |
| Campaign/session | no |
| Shipped content | yes — the four overlay spells and the three staged spells are shipped rows |
| Interactions with existing mechanics | yes — the point-cast burst, the heal shower and the archer's shot mark share the ground point being corrected |

## Domains

**8 Client** (`pkg/ui`, `pkg/render/terrain`, `pkg/game`) — the draw path, which is where all five
reports live. **3 Combat & Magic** (inside `pkg/sim`) — the paint observation. **1 Assets**
(`pkg/data`) — the overlay art table.

## Out of scope

- **The cast's own burst object at the landing cell.** `spawnBurst` fires for every applied cast,
  cloud rows included. `MAGIC-AREADRAW-049` establishes that the AreaEffect *tick* routine creates
  no object for a cloud; it says nothing about what the cast dispatch does before the area record
  exists, so this build's burst is not shown to be wrong. It is also what draws the birth sprite the
  owner describes for Wall of Fire. Unchanged and untested here.
- **Draw order between the two overlay passes.** Recorded as an UNKNOWN row; all four overlays draw
  in one pass, with the content band.
- **The `view+0xa7c` hash and the terrain-type gate** of `MAGIC-WALLFIRE-058`'s add arm — a cell
  burns only where the terrain type permits. Simulation, not presentation, and the claim itself
  leaves what the type table selects open.
- **Stone Curse's mask and immobilisation** (`DIV-073`), **Lightning's shape**, **Fire Ball's tail**.
  Other stories.
- **`pkg/sim/spell.go`'s six-slot pre-check** (`DIV-064`). Untouched.

## Expected divergence rows

`DIV-077` (Meteor Storm's fall, UNKNOWN), `DIV-078` (the frame law's spatial term, UNKNOWN),
`DIV-079` (the two overlay draw passes, UNKNOWN).

`formatVersion` **55** was allocated to this story and is **not spent**: no serialized byte changes.
