# Large-radius area spells

## Intent and authority

A mod-set Fire_Ball radius up to 255 hits exactly the cells the radius covers inside the map, draws one burst per covered area, scorches the covered cells and saves to SAV. This is step 2 of the spell mod plan (inventory `docs/1299/inventory.md` on branch `story-1299-spell-mod`, "Story split" row 2). Owner direction: MOD milestone item 4, one named example being the Fire_Ball radius. The radius key and its 0..255 domain are step 1 (`docs/1341/story.md`). Differences from ROM1 are `DIV-2334` and `DIV-2335` in `docs/divergences/mods.md`; the wrap clause of `DIV-2328` is closed by them. The game without a mod is unchanged.

## As built

Cells. A blast lands through `World.blastCellsInMap` (`pkg/sim/celleffect.go`). While `2r+1 <= 256` it is the byte producer `blastCells` exactly, including the cells that wrap past an edge of a 256-wide or 256-high map, so the installed radius of one keeps its cells, order and hashes on every map. Above that the byte walk would visit cells twice, so the walk is the square `[x-r, x+r] x [y-r, y+r]` clipped to the map, x outer and y inner, each cell once. The in-flight blast restored from a SAV uses the same walk with the radius saved in the area record (`AE48`).

Hits. The cell walk has no owner or hostility test, as in ROM1: the caster, allies and enemies on a covered cell are each hit once per covered cell they stand on (a larger body is hit once per cell and its damage is divided by its footprint, unchanged). Scorch lands on every covered in-map cell.

Bursts. Radius 0 or 1 draws one burst at the anchor, as before. A larger radius draws one burst per three-cell tile of the covered square that touches the map (`releaseFireBallBursts`, `pkg/sim/burst.go`), each centred on its tile and kept inside the covered map cells, in x-outer order. The count is at most `ceil((W+2)/3) * ceil((H+2)/3)` for a map of W by H cells. A blast landing from a loaded SAV draws the same bursts with no owner slot, as one already did.

Cost. A radius of 255 costs one walk over the map's cells and one burst per tile; a radius of 127 or less costs at most 255 by 255 cell visits. `insertProjectile` placed a record with a sort over `slices.Index`, cubic in the record count; it now moves the new record to its id's place when the stored records are already in id order, with the sort kept as the fallback (`pkg/sim/unitshot.go`). Records and ids come out identical. Measured with `go test -v -run WidestRadius ./pkg/sim` (one cast, radius 255, 256 by 256 map, 201 units): 77 s with the sort, 0.88 s after, with 7396 bursts and 65536 scorched cells.

Save. No SAV format changed. The area record already holds the radius byte, scorch is in the footer and bursts are projectile records.

## Proof

- `pkg/sim` (`blastradius_test.go`): the walk equals the byte producer up to radius 127 and the clipped definition above it, for four map shapes, seven radii and four anchors including the corners; an installed-radius blast at corners and edges of a 256 by 256 map scorches and hits the wrapped cells as the byte walk does; one burst for radius 0 and 1, tile counts for 2, 4 and 5; every covered cell lies under a burst, no burst is off the map and the count stays within the tile bound; scorch equals the covered cells exactly; the caster and an ally are hit once at radius 1 and once at radius 255 where the byte producer reached them again through wrapped coordinates, a unit outside the radius is not hit; radius 255 on 256 by 256 stays bounded; a wide blast in flight and during its bursts survives the byte form and ends on its tick.
- `pkg/game` release, EN and RU (`modradius_release_test.go`, gated): a fixture mod written inside the test sets the Fire_Ball radius to 255; a cast in mission 41 scorches all 6400 cells, hits the mage and an ally, draws more than one burst and no more than the tile bound; the SAV written while the cast is in flight holds the radius byte 255; its cold LOAD blasts onto the same cells with the same burst count; a SAV written during the bursts restores the scorch and the bursts; a game without the mod refuses it. Loss control: the unmodded install casts one burst over nine cells.
- Unmodded identity: the existing pinned world hashes, the full `go test` and the EN and RU release suites pass unchanged.

## Open debt

- The original's behaviour for a radius wider than the installed one, and the burst count it would draw, were not observed (`DIV-2334`, `DIV-2335`).
- A burst per tile centred at a clamped edge cell draws partly outside the covered cells for a tile that straddles the edge; the picture is not clipped.
- A cast near the map's largest size draws up to 7396 burst records for 23 ticks; the renderer's cost for that count was not measured.
- Not built here: target filters, ray rule, formula tables, live hooks (stories 3 to 6 of the plan). Other area spells (clouds, walls) already clip to the map.
