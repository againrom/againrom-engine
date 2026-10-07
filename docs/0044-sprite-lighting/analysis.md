# Analysis — lighting the sprite layer

## Intensity & terrain

| Axis | Declaration |
| --- | --- |
| Intensity | **spec-first / static** — the profile's tier for rendering; no watcher tool exists, so it is discipline |
| Terrain — the ramp and the row a sun gives | **greenfield**: neither exists in any form |
| Terrain — the blit, the raster tool's static layer, the window's two sprite passes and its texture cache | **brownfield**: all shipped and touched here |

## What the decoded rule turned out to be, and what it is not

The sprite blit family splits in two, decided by which pointer the pixel loop reads. One pair reads
the **source** index and looks it up in a per-level row of a shading table — the lookup shape the
terrain blitters already use. The other never reads the source: it advances past it, reads the
**destination**, and indexes a second table. That pair is the shadow and shroud pass, so "we draw no
sprite shadows" names a separate pass we do not have, not an option left off the one we do.

Two prior readings are refuted at the pin — the shroud level, and a sample of the terrain brightness
grid. Either would have made the row follow relief, so the design would have needed a per-cell input
and a per-placement level; instead the level is one byte, memset over the visible window from the
sun's ambient and uniform except at stamped cells.

The ladder was the surprise. Sprite and terrain share one, the sprite's being the terrain's sampled
every fourth row, and the sprite table's neutral row is pixel-identical to no table at all — an
exact meaning for "unshaded", and a cross-check of the new transform against the shipped one.

## The unit arm: the Unknown is wider than it is advertised

The row assigning tables to drawables publishes its Unknown as *which registry key fills the class
field, and therefore which of the 34 classes takes which arm*. Read for what could be built, the gap
is wider: the arm taking the 16 shared tables needs a 16 KB palette blob whose **source node is
Unknown too**, and the class-owned and per-owner arms read palettes at a class offset the row names
but never says what fills. All three unit arms lack a palette source we hold.

So the choice was never "which arm". The only construction decoded end to end whose inputs are in
hand is the **object** sheet's: its own sixteen-row table, same mode, from its own palette. Extending
that to units is ours, and is the whole of what is ours.

Three things make it safe at this tier. The arms differ in *which palette a ramp is built from* and
not in the gain ladder, so being wrong costs hue, on units, and nothing else. The arm structure is
itself decoded at High — the class field is the length of the class's table array, which is why
zero, one and many select as they do — so the choice does not pretend the arms are absent. And the
story reaches no hashed state, so nothing inherits it silently provided it is written down.

Refused: inventing an owner-to-ramp mapping. With no owner-keyed palette in reach, "owner 1 takes
ramp 0" would be a number of ours wearing a reconstruction's appearance.

## What the six rows could and could not carry

- **The lit-family row is the strongest.** It makes "ours are unlit and the engine's are lit" a fact
  rather than an impression, and the body/shadow split comes with it.
- **The mode row is exact and complete on the arithmetic** — every operation named, the arm fixed by
  the dispatch table's own bytes. Its one unusable clause is the greyscale: a unit takes a luminance
  ramp on one override, while what the overrides are *for* is Unknown. Recordable, not actionable.
- **The level row is High on the grid and Medium on exactly the half a unit needs.** Allocation,
  memset, all three stages and the pointer's writer set are instruction-level; that the byte reaches
  the *unit* blit on every path is not — the body routine also reads that argument slot as a frame
  index, forces it to zero on one path, and a second dispatch pushes zero.
- **The ladder row's headline is its least durable part.** The 38 % deficit is one owner-review
  render of one frame, under that row's own Medium about whether a live session runs at this ambient
  and sun angle. Its durable content is the identity and the neutral-row equality — measured bit for
  bit over terrain levels 32–64, i.e. sprite rows 0 to 8 only. Rows 9–15 rest on the arithmetic,
  which reduces to one expression on both sides and is exact for every channel value, so nothing is
  lost; the *measurement* does not reach them, which is worth recording rather than smoothing over.
- **The output-range row is the weakest here, and terrain-side end to end.** It closes the terrain
  mapping's range and its white-saturation census over tile pixels and measures nothing about a
  sprite. It carries the clamp's shape and the finding that the mapping does not stop short of white
  — which is why saturation is disclosed rather than designed around.
- **The table-assignment row carries two strengths and one hole**: the object construction and the
  palette census, both High, and the unit selector, above. The census is why a shared ramp was never
  a candidate — no sheet carries the terrain palette, and the sheets do not agree among themselves.

## The two questions asked of the tree rather than assumed

**Does anything already apply a per-sprite colour transform a ramp would double with?** No, and it
was searched for rather than inferred: no colour key (transparency is structural), no team colour,
tint, fade or grey-out, no shadow pass, and the window's sprite draw sets only a filter and a
transform. The colour scale is on terrain vertices alone and the selection highlight is a separate
rim pass over the art.

**Do the object and unit paths share a blit a table can reach?** They share one, and it is the only
sprite blit in the tree. The palette rides on the frame rather than on a sheet or a class, so "the
sheet's own ramp" is a local fact at the blit — which is why the unit arm's resolution costs nothing
at the call sites.

## What we looked at

The `research/` submodule at the pin: `claims/retracted.md` first, then `claims/terrain.md` rows
`TERR-LIGHT-059`…`064` with `TERR-SPR-065`/`066`. The tree:
`pkg/render/terrain/{blit,shade,light,statics,lit,project}.go`, `pkg/ui/{viewer,statics}.go`,
`pkg/game/{statics,units}.go`, `pkg/formats/spr256`, `cmd/terraintool/main.go`, and the work items
for terrain lighting, corner interpolation, map objects and the three sprite stories.
