# 1002 — effect marks: spec

Canonicalized to as-built at the landing. This document is self-contained; claim provenance is in
`contract.md`.

## Terms

**Mark record** — one 8-byte record: `dx`, `dy`, `depth` (all signed 16-bit), a record index and a
phase (both unsigned bytes).

**Kind** — `2*spellId + 8`. It is both the dispatch key and the record index, and the record index
is the `projectiles.reg` picture id the record's art resolves through.

**Element** — one open `(actor, kind)` pair with a countdown. The countdown is a phase clock, not a
lifetime.

**Rebuild** — one advance of the client's own per-tick state. This build rebuilds once per world
tick.

**TileSize** — the `units.reg` key of that name on the actor's class, the footprint scale every mark
offset is multiplied by. Its registry default is 1, and a value below 1 is treated as 1.

## FR-1 — the record and the two passes

The unit draw walks an actor's record array twice. The first pass runs before the actor's own
sprite and draws a record only when `depth > 0`; the second runs after and draws the rest. Both
compute the same position:

```
x = actor anchor x + dx - sheet.CenterX
y = actor anchor y + dy - depth - sheet.CenterY
```

The anchor is the actor's own cell anchor in world pixels, `cell*CellSize + CellSize/2`, moved by
the actor's own displacement and lifted by the same relief terms the actor's sprite takes. The
sheet's centring halves are the registry's own Width and Height halved.

The two passes are interleaved inside the drawn content band, at the actor's own place in it, so a
mark that draws behind its actor still draws in front of whatever the actor is in front of.

The second pass runs whether or not the actor's own sprite is drawn. A mark is culled on its own
rectangle.

## FR-2 — the element list

For every `(actor, kind)` the simulation carries a lasting effect for, an element is open. It is
opened at countdown `0xffff`, decremented by one on every rebuild, and removed when the effect
leaves the simulation's set.

The record set is not stored. It is rebuilt from the element list on every rebuild.

An element whose picture names no sheet in the loaded projectile set contributes no records.

## FR-3 — the dispatch

A kind outside the closed range `0x12..0x3e` builds nothing. Inside it, seven kinds reach a builder:

| Kind | Spell | Builder |
|---|---|---|
| `0x12` `0x1c` `0x28` `0x34` | 5, 10, 16, 22 — the four Protections | FR-4 |
| `0x2c` | 18 — Shield | FR-5 |
| `0x36` | 23 — Bless | FR-6 |
| `0x3e` | 27 — Curse | FR-6 |
| `0x18` | 8 — Poison Cloud | FR-7 |

Every other kind in range builds nothing, and that includes `0x26` Invisibility and `0x30` Stone
Curse, which FR-8 covers instead. Each spell's builder is independent, so an actor carrying several
effects shows the union of their record sets.

## FR-4 — the four Protections

One record each, at `depth = 0`, record index the kind, phase `countdown mod 6`, and

| Kind | Spell | dx | dy |
|---|---|---|---|
| `0x12` | Protection from Fire | -6 | `-TileSize*32` |
| `0x1c` | Protection from Water | +6 | `-TileSize*32` |
| `0x28` | Protection from Air | 0 | `-TileSize*32 - 6` |
| `0x34` | Protection from Earth | 0 | `-TileSize*32 + 6` |

All four therefore stand `TileSize*32` above the actor anchor and 6 pixels apart on the two axes.
Four at once form a diamond of half-width 6; nothing in the builder consults the other effects.

## FR-5 — the Shield

The point set is the midpoint circle rasteriser at radius `TileSize*11`, decision term `2 - 2r`,
eight symmetric points per iteration. Each point is rotated in the figure's own plane by
`phase * 4` degrees, where `phase` is `countdown mod 90`. The rotated second coordinate is scaled
from `TileSize*11` to `TileSize*16` and becomes the point's vertical term `a`.

The envelope is `trunc(abs(phase/45 - 1) * TileSize*28)`. It is zero at phase 45 and widest at 0.

Each point appends **two** records, both at record index `0x2c` and both at the same frame:
`depth = -(envelope + a)` and `depth = envelope - a`. `dx` is the rotated first coordinate and `dy`
is zero.

The frame index is `4 - abs(phase - 45)/9`, floored at zero.

Because the vertical term is written into `depth`, FR-1's two passes split the figure: the half
above the actor draws behind its sprite and the half below in front. At `TileSize` 1 the rasteriser
yields 72 points and therefore 144 records.

## FR-6 — Bless and Curse

Five steps, four records each, twenty in all. Bless's step counter runs 89, 71, 53, 35, 17; Curse's
runs 0, 18, 36, 54, 72, so the two sweep opposite ways as the countdown falls.

For a step at angle `θ` degrees, with `c = trunc(cos(θ) * 20)` and `s = trunc(sin(θ) * 20)`, the
four records take the four sign combinations of `(±c, ±s)`: `dx = ±c`, `depth = ±s`,
`dy = -TileSize*32`, record index the kind.

The frame index is per record, `4 - step/18`, which runs 0..4 for Bless and 4..0 for Curse, so five
frames of the sheet are visible at once. The caller's `countdown mod 5` selects no frame.

The sine in `depth` splits the ring around the actor by FR-1's two passes.

## FR-7 — Poison Cloud

One record: `dx = 0`, `dy = -TileSize*32`, `depth = 0`, record index `0x18`, phase
`countdown mod 6`.

This build delivers Poison Cloud as a cell effect and never attaches it to an actor, so no element
of kind `0x18` opens and the builder is unreached in play. It is built and tested because the
dispatch owes it. DIV-076.

## FR-8 — the two by-kind arms

Invisibility (`0x26`) and Stone Curse (`0x30`) add no record. They change the actor's own draw.

- **Invisibility.** An actor carrying spell 15 is not drawn for a participant that does not detect
  it. A participant detects it when the actor is its own, or when one of its own actors stands
  within that actor's decoded `SeeInvisible` radius of it — the same test AI acquisition already
  applies. The actor leaves the pushed entity list entirely, so it is out of the picture and out of
  selection together. DIV-074.
- **Stone Curse.** An actor carrying spell 20 has its live-selection animation clock held at the
  value it had when the effect landed, and its walk cycle suppressed, until the effect expires. What
  stops such an actor moving is undecoded and is not built. DIV-073.

## FR-9 — what does not change

- No serialized byte form changes. `formatVersion` is untouched.
- The element list and the animation hold are cosmetic client state: neither is in the world, the
  byte form or the digest, and both are ruled cosmetic in `docs/0143-save-and-load/spec.md` FR-5.
- The `0154` authored `SpellFX` school ring and the `0161` Heal burst are unchanged and draw beside
  the marks.
- An actor carrying no marking effect produces exactly the band it produced before this story.

## Acceptance

- **AC-1** The four Protections build the FR-4 table at any `TileSize`, all at `depth = 0`.
- **AC-2** The Shield builds two records per rasterised point, both at record index `0x2c`, with the
  envelope zero at phase 45 and the records split between the two passes.
- **AC-3** Bless and Curse each build 20 records with frames 0..4 and 4..0 respectively, split
  between the two passes.
- **AC-4** Every kind in FR-3's "builds nothing" set builds nothing, and every kind in its table
  builds at least one record.
- **AC-5** A positive-depth record precedes the actor's sprite in the band and a non-positive one
  follows it.
- **AC-6** An attached effect opens an element at `0xffff`; a rebuild decrements it; the effect
  leaving removes it.
- **AC-7** An actor carrying no marks produces its sprite alone.
- **AC-8** `InvisibleTo` answers false for an actor with no invisibility effect and false for the
  actor's own participant.
