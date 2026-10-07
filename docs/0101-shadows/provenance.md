# 0101 — shadows: provenance

Research pin `e6f9ee6`. Every row below was read through `research/tools/claim` at that pin, with the
retraction state cross-read.

## The pass split — FR-4, FR-5, FR-10, FR-13

`TERR-SPR-065` (High) — the unit BODY is `vt+0x28` (`R0552`) and `vt+0x2c` (`R0553`)
is the SHADOW. The shadow routine's whole indirect-call list is the width/height getters and the
silhouette family; it issues no lit blit at all. This **amends `TERR-SPR-048`**, which published the
shadow's frame, destination, anchor and mirror as the body's, and whose two overturning rows in
`retracted.md` were read before anything else here. A story built on the superseded text builds the
two passes backwards, which is the single most expensive mistake available in this area.

`TERR-SPR-067` (High for the terms, Medium for what the two fields *mean*) — the two destinations
differ by exactly `(sunShear, -unit+0x10)`: the body's `dstY` subtracts `unit+0x10` and the shadow's
never does; the shadow's `dstX` subtracts the shear and the body's never does. The anchor is shared
and unchanged, and the frame switch is duplicated arm for arm. FR-5 is this row. That `unit+0x10` is
a height above ground is the row's own Medium, and FR-6 does not depend on it: the parameter is the
difference between the two destinations whatever the field means.

`TERR-SPR-043` (High for the split, Medium for the sheet pairing) — the object shadow anchors on
frame 0 and its body on the drawn frame; eight shipped classes are displaced by the stated amount.
FR-10 carries that displacement rather than correcting it.

`TERR-FOG-037`, amended by EXP-0088 and now **High** — the destination-recolour table is 17 rows of
`out = (in * (16 - L)) >> 4`, checked against the engine's own two table-free fast paths over the
whole 16-bit pixel space, 65536/65536 in RGB565 and RGB555. FR-13 is that law. The pixel loops of
the silhouette blits advance the source without reading it and transform the destination, which is
why FR-15 is stated as an assertion about two backgrounds rather than about pixel colour.

## The shear — FR-1, FR-2, FR-3, FR-7

`TERR-LIGHT-113` (High) — every shadow pass derives its shear from the sun angle through one
routine, and the unit BODY computes it and discards it one instruction later. Re-executed over the
day band the tangent spans `-0.5774 .. +0.5754`; with the cycle off it is fixed at
`+0.57735025728078282`. FR-2 is this row's two numbers, and D-2 records that `theta * 2 / 3`
reproduces both while no instruction of the routine was read: it is our reconstruction of the row's
own stated outputs, at **Medium**, and it is falsified by either number moving.

`TERR-SPR-066` (High) — the shear is a 16.16 per-row X slope the blitter accumulates, not a
translation; but the unit's `vt+0x1c` arm carries no slope and offsets `dstX` by `shear/2000`
instead. `TERR-SPR-048`'s surviving clause says which arm: `vt+0x1c` whenever the sun scalar is
non-zero. So FR-4's unit shadow translates and FR-11's object shadow shears, and that is a decoded
difference rather than a simplification.

`TERR-STRUCT-103` (High) — the structure's shadow is the same loop over the same frames with the
same `dstY`, sheared in X as `screenCol*32 + ftol(tan(sunAngle) * ((FullHeight - k)*32 - ShadowY))`,
and `ShadowY` is the height at which the shear is zero. FR-7 is that expression term for term, with
`k` the strip's grid row, which is the index this tree's own strip builder already walks. The row
also carries FR-9: the two `VariableSize` bridge subclasses override the shadow with a bare return.

`TERR-STRUCT-107` — a structure's drawing does not depend on its owner, so no shadow does either.

**D-1 is where the corpus stops.** `TERR-STRUCT-103` writes its shear as an addition inside a
complete expression whose zero point is named; `TERR-SPR-067` and `TERR-SPR-038` write theirs as
subtractions of a term whose scale their rows do not give. Nothing reconciles the two, and the
discriminator would be the sign of the slope argument at the blitter, which no row quotes. The
structure's sense is taken because it is the one published whole.

## The level — FR-14

`TERR-LIGHT-126` (High for the values and the schedule) — the two shroud indices move with the band:
4/2 with the cycle off and by day, 6/3 in both twilights, 8/4 at night, the object path's always
twice the unit path's. `TERR-SPR-066` identifies them as the silhouette blit's fourth argument,
indexing the recolour table. `TERR-STRUCT-103` puts the structure on the object path's global, which
is why FR-14 pairs structures with objects and not with units.

Those two fields have been in `terrain.Light` since `0100` and this story is their first reader.

## Ours, not the corpus's

FR-16 through FR-20 are a rendering design, not a decoding: the engine has no texture cache, no
camera transform and no draw-order problem of this shape. What they are held to is FR-13 — the law
the corpus does establish — which is why AC-5 compares the screen's arithmetic against the exact
integer one rather than asserting the screen path in its own terms.
