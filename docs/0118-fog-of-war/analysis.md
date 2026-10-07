# Analysis — 0118 fog of war

## What we did not know going in

Whether fog is a new subsystem or a second reader of something we already have.
The brief arrived sized as the former. It is the latter, and that is the single
finding that decides this story's shape.

## What was looked at

**`pkg/sim/sight.go`, read end to end.** It is already a faithful, *corrected*
transcription of the law: the 41x41 window, both of `AI-LOS-088`'s grids built
once at package level from geometry and `k` alone, the two zone tests `j < i>>1`
and `j > 2i`, the four-mirror store order, the two trailing literal repairs to
`(+1,0)` and `(-1,0)`, an integer `isqrt` standing in for the FPU's `ftol`, the
four-term recurrence stored *before* the visibility test, and a blocking cell
left unpruned. So we do not build `AI-LOS-088`. `AI-SIGHT-093` grades the drawn
fog and the AI's vision as one algorithm written twice, agreeing in 0 of 1681
window cells at a whole-cell sight, and closes with the sentence that is this
story's whole architecture: *"A consumer needs one implementation,
parameterised."*

**What that parameterisation costs.** Three things separate the two readers and
all three are decoded:

1. the in-bounds rectangle — `TERR-FOG-118`, client 7 cells from every edge,
   server 8, and we have only the server's, read off the grid's air-block bit;
2. the seed width — `AI-SIGHT-094`, client from a `u16` in 1/256 cell, server
   from the byte above it, a 40-cell difference for a hero;
3. `k` — `TERR-FOG-088`, two sources, and ours is the constant `sightShift = 7`.

**Where visibility state could live.** Nowhere today: `groupSight` builds a
stamp per engagement decision and drops it, and `World` has no seen or explored
field. The map's own tile bits 15..14 (`TERR-TILE-079`) are not preserved at
load. So the accumulated *explored* plane is genuinely new, and the question is
whether it is simulation state. It is not — nothing in `Step` reads it, and it
is per-participant *view*. It is built and held one tier out, in `pkg/game`.

**What the seam already carries.** `SetLocalOwner`/`localOwner`
(`pkg/ui/command.go:565`, `pkg/ui/viewer.go:850`), already used to decide which
way a damage numeral is drawn. Whose view this is needed no new notion.

**Where the shroud multiplies.** `withScales` (`pkg/ui/viewer.go:2517`) already
multiplies each of a tile's four corner colours by that corner's own scale, and
both terrain paths go through it. A per-cell fog factor is a second multiply on
an existing one rather than a new pass.

**Whether the sight radius is already customisable.** It is.
`pkg/mapload/fromalm.go:584,602,619` take `def.ScanRange` from the `Data.bin`
Units, Humans and Dwellings definitions, and `pkg/mapload/start.go:309` takes
the party hero's from `Hero.Sight()`. No constant in this tree names a sight
radius, so `AI-SIGHT-092`'s customisation point is already held.

## What we chose not to look at

The minimap's click surface. `AI-MINIMAP-062` establishes a second click
dispatch covering the six `s`-prefixed cursors, but its own confidence cell says
the window's *identity* as a minimap is inferred from those art names and not
established. Building an input system on an inference, for a thing the owner
asked only to be able to *see*, is the wrong trade.
