# 0101 — shadows: analysis

## What was not known before this story

**Which routine draws a unit.** Two published rows named the wrong vtable slot in a row, and the
second of them published the shadow's frame, destination, anchor and mirror as the body's. Everything
downstream of that row had to be re-read before a shadow could be drawn at all: a consumer holding
the superseded text would have built the shadow out of the body's rule and the body out of the
shadow's.

**Whether the lean is a skew or a shift.** The engine has two silhouette entry points and the unit
path uses both — one carrying a 16.16 per-row X slope, one carrying none and shifting the whole
destination instead. Which arm a unit takes decides whether a unit's shadow is a parallelogram or a
translated copy, and nothing but the guard on the sun scalar says.

**What the sun's angle has to do with the shear.** The stored angle and the shear are not the same
number: the shear passes through a routine before the tangent is taken. The corpus states the
routine's output at one input and the tangent's range over the day band, and nothing states the
routine.

**What the destination-recolour table does.** The shadow is not a darkened copy of the sprite: the
silhouette blits advance the source without reading it and transform the destination instead. What
the transform is was Medium until the table was reproduced.

## What was looked at

`TERR-SPR-065`, `-066`, `-067`, `TERR-SPR-038`, `-043`, `-047`, `-048` and the two `retracted.md`
rows that overturn `-048`; `TERR-LIGHT-113` and `-126`; `TERR-FOG-037` with its EXP-0088 amendment;
`TERR-STRUCT-103` and `-107`. All read through `tools/claim` at pin `e6f9ee6`.

The tree was read for its existing sun: `pkg/render/terrain/sun.go` (0092, 0100) already carries the
angle and the per-band shroud indices, and nothing read the latter before this story.

## What is still not known, and what was done about it

**The sense of the lean.** Three passes publish a horizontal shear term and they do not agree in
sign as written: the structure's is an addition inside a complete expression with its own zero
point, the unit's and the object's are subtractions of a term whose scale the row does not give.
Nothing in the corpus reconciles them. The story takes the structure's sense for all three, because
it is the only one published as a whole expression, and records the choice as a divergence — it is
one global sign and a later reading flips it in one place.

**The routine behind the shear angle.** `TERR-LIGHT-113` states the routine's output at the
cycle-off input, and the tangent's range over the day band. Two thirds of the argument reproduces
both to a unit in the last place; no instruction of the routine was read. Taken, and recorded as
Medium.

**`unit+0x10`.** The term that lifts a body off the plane its shadow lies in. Nothing in this tree
produces one — there is no flying unit — so it is a parameter of the placement and zero at every
call site today.

**The object shadow's detail gate.** The engine gates the object shadow on a flag the corpus reads
as a detail setting. This story draws the shadow unconditionally: a settings surface is not in scope
and a flag with no way to move it is not a feature.
