# 0102 — analysis

`0101` shipped the shadow pass and four of its rules are wrong. The owner can see one of them: a
structure's shadow is a stack of rectangles that step sideways instead of one leaning silhouette,
with a visible cut every 32 world pixels. The other three are the same arithmetic read from the
wrong end, which is why they are corrected together rather than one story each.

## What we did not know when 0101 was written, and now do

`0101` had the three casters' `dstX` terms but not the **blitter's own** law, so it could not tell a
caster's relocation apart from the lean the blitter adds after it. Everything below follows from
having both.

- **The blitter shears about the bottom edge of its own rectangle.** The whole shift is pre-added
  to the destination pointer and each row walks back left, so image row `r` lands at
  `dstX + trunc(h*s/65536) - floor(r*s/65536)` and the row that sits on the `dstX` passed in is
  `r = h`. Every caster therefore leans; a caster's own term only moves the pivot.
- **A structure carries both terms at once.** The per-strip integer `0101` implements and the
  16.16 slope it leaves at zero are pushed on the same call. One strip is displaced rigidly *and*
  each of its rows again inside the blit.
- **The unit's `dstX` subtrahend is a pixel count, not the 16.16 shear.** It is
  `ftol(tan(theta) * (2*floor(frameH/2) - anchorY))`, so it depends on the drawn frame's height and
  on the anchor; `0101`'s `shear/2000` gave every unit the same shift, and `shear/2000` belongs to a
  different arm of the unit path altogether — the translated, unsheared one.
- **The angle has a dead band.** The routine clamps the sun angle away from zero before the `2/3`,
  so the smallest magnitude that reaches the tangent is non-zero and the lean never reverses
  through an exact zero.

## What that costs us on screen

The tower we reproduced — mission 10, anchor cell (18,61), `TileWidth 2, TileHeight 2,
FullHeight 3, ShadowY 40` — takes per-strip shifts of 32, 13 and -4 world pixels at the cycle-off
sun. Consecutive strips are 32*slope ~ 18.5 px apart and each strip is 32 px tall, so the cut is
exactly the lean the strip does not have. Adding it back makes the composed column a function of
world `y` alone: substituting a strip's own `dstY` into the two terms cancels `k` entirely, and the
whole footprint shears about one world row. The residue between consecutive strips is then the
difference of two `ftol` truncations, under one pixel, against 18.5 today.

## The two things we had to settle rather than assume

**The strip frame height.** `TERR-SHDW-131` leaves the structure's pivot Unknown pending it. Our
own sheets answer it: over 66 classes and 1300 frames, on both roots, every structure strip frame
is exactly 32x32, and no class mixes heights. So the pivot is `ShadowY - 32` px above the image
bottom, which is the reading `TERR-SHDW-131` names as the alternative to `TERR-STRUCT-103`'s
wording. Two classes carry no frames at all (33 and 37) and are the `VariableSize` bridges `0101`
already excludes.

**The object's anchor frame.** `TERR-SPR-043` and `TERR-SHDW-130` contradict each other on the
object path and we could not reconcile them; `provenance.md` records the reasoning and the open
question, and `spec.md` D-3 states what we do meanwhile. It changes nothing for 58 of 66 classes.

## What we deliberately did not do

We did not add a test for the fifteen classes whose `ShadowY` is 10000 or 20000. They are
suppressed in the original by displacement alone, with no branch anywhere, and the dead band is
what keeps that displacement large. A sentinel or a clamp would give fifteen classes a shadow the
original does not draw. We checked the reasoning instead of adopting it: swept
over all 1440 minutes, the smallest displacement a `ShadowY = 20000` class takes is 660 world px,
which is the figure the corpus publishes.

**The published figure does not choose between the two readings of the dead band, and we first
thought it did.** The competing reading — return `+-0.05` rather than clamp before the `2/3` —
gives 991 px only for an angle that stays inside the band. The angle does not: it steps by 0.0022
per minute, so it lands just outside the band twice a day, and the competing reading's own minimum
over a real day is 663 px against ours at 660.5. Both round to the published 660. What actually
decides it is that only our reading attains the published smallest magnitude 0.0333 rather than
approaching it from outside, and that the competing one is discontinuous in `|theta|` — it leans
*more* at noon-adjacent angles than at the band's edge, which is not what a clamp does.

The four classes at `ShadowY = 10000` are weaker: 327 world px at the same minute. That is still
far outside any structure's own footprint, but it is not obviously outside every viewport, and we
say so rather than assert that all fifteen are equally safe.
