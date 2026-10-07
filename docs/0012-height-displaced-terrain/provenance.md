# Provenance — height-displaced terrain geometry

`claims/retracted.md` was read first, and two rows below are narrowed because of it.

## Backing

| `spec.md` anchor | Claim | Confidence |
|---|---|---|
| FR-1 — everything the flat raster does other than place a pixel | 0004, 0006, 0007 | carried unchanged, backed there |
| FR-2 — vertex screen row `r*32 - h`, by subtraction; one writer, no negation, no second write | `TERR-GEOM-031` | High — mesh formula, sign and grid identity are one named instruction each |
| FR-2 — the altitude is **signed** at the point of use | `TERR-GEOM-031`, `TERR-LIGHT-028` (d) | High, and decidable **only** from the instruction (`MOVSX`): 0 of 880 704 shipped height bytes reach `0x80`, so no corpus could discriminate it |
| FR-2 — a larger altitude moves the pixel **up** the image | `TERR-GEOM-031` | High, with the last step marked apart by the claim itself: the row index is bounded by the `top`/`bottom` fields of a rect passed to `CopyRect`, and "`top < bottom`, y increasing downward" is an OS convention, not a ROM1 instruction. Everything before it is instruction-level |
| FR-3 — flat exactly when all four corner altitudes are equal | `TERR-GEOM-032` | High — three compares in four byte-identical renderers; the reduction from projected Y to altitudes verified over 870 198 interior cells at 0 disagreements, ruling out a threshold and an any-difference test |
| FR-4 — the flat cell is an axis-aligned 32x32 rectangle; no altitude reaches that blitter | `TERR-GEOM-033` | High |
| FR-5 — the per-column span, its inclusive/exclusive bounds, the 16.16 resample and the dropped column | `TERR-GEOM-033` | High — every step a named instruction; that it is a **stretch, not a shift** (an `IDIV` by the span height, not the constant `0x10000`) and the exclusive bound are each pinned by one |
| FR-6 — the step-table recipe, the two walks, the latch, the `T[d][d] == 32` sentinel | `TERR-GEOM-034` | High — a 14-instruction builder reproduced exactly, sentinel and monotonicity holding on all 127 rows, cross-checked against a second routine in the same binary |
| FR-5 — vertical only; destination X is the plain column counter | `TERR-GEOM-035` | High — destination-address arithmetic read instruction by instruction; no altitude-derived term reaches it |
| FR-8 — ascending paint order, later-cell ownership, and the one-row seam at deltas 63 and 127 | `TERR-GEOM-036` (a) | High — 9280 boundary columns, every mismatch **classified per column** (overdraw 9280, holes 0), not inferred from the sign |
| FR-5, P-5 — a collapsed span is the only hole the model produces | `TERR-GEOM-036` (b) | High — 3576 of 24 983 232 drawn columns, in 353 cells |
| FR-7 — the level ramp runs corner-to-corner over the drawn **span** on the sloped path, over 32 rows on the flat one | `TERR-LIGHT-011` (amended) | High for the arithmetic, read narrowly: this row carried High for twelve experiments while describing only the *colour* half of the two blitters (`retracted.md`); the span-divided ramp is part of the correction, not of the original row |
| FR-2 — the far-edge `+1` vertex has no value the engine defines | `TERR-EDGE-024`, `-025`, `-026` | High for the mechanism — unpadded `W*H` grids, flat addressing off the end of them, an uncomputed outer ring, an allocator that does not zero / **Unknown** for what those bytes hold at runtime |
| Why the geometry is the default and not a mode | `TERR-GEOM-032` | The shipped corpus is **89.72 % sloped / 10.28 % flat**, measured on the corrected grid base and independent of the withdrawn height census (`TERR-LIGHT-016`) — so this story may lean on it, and may assert nothing else about how much relief shipped maps carry |

## Ours by choice

Everything left sits at the **boundary** of the engine's model — it draws into a scrolling viewport
over a surface that already holds pixels; we draw one still image. None of it is a raster rule.

| What the spec fixes | What the evidence actually says |
|---|---|
| FR-9 — the whole-map canvas, its vertical origin, and integer scale by pixel replication | The engine has no canvas — it has a clip rect, a scroll origin and an over-scan margin, and clips whole cells horizontally but single rows vertically (`TERR-GEOM-035`, `TERR-EDGE-026`). Nothing in it defines a map-sized image, and it has no scale |
| Uncovered pixels are transparent | The engine writes over a surface that already holds something; "uncovered" is not a state it has |
| The far-edge vertex ring reads the height index clamped into the grid | `TERR-EDGE-026` names clamp-to-edge as the safe, intent-matching choice for a port — but says it of the *brightness* ring. Extending it to the geometry mesh is ours |
| The step-table recipe applies for any `d >= 1` | See *Open*: the engine's table has 127 usable rows and defines nothing past them |
| FR-11 — markers keep the un-displaced cell lattice, translated by the same vertical origin | Nothing in the pin says where a marker belongs; the engine has none. Where a *sprite* belongs is 0015's question |
| The vertical level ramp truncates | See *Open*: the engine's rounding differs and is not transcribable from the pin |
| FR-10, FR-12 — the rejection set, the no-panic guarantee and the reported summary | Project engineering carried from 0004; the engine validates none of this |

## Open — recorded, not solved

| What | Status |
|---|---|
| A height delta of 128 or more between adjacent vertices | Past the step table's last row. The corpus maximum is exactly **127** — the last row is exercised, nothing to spare — so the recipe extension cannot be checked against the engine, which is not robust there either; the shipped data merely stays inside it (`TERR-GEOM-036` (c)) |
| The drawn edge disagrees with the engine's own hit-test edge | The picker bounds the same quad by an exact lerp and differs from the drawn walk on **3665 of 4064** `(d, col)` pairs, by up to **3 rows**; reported, not explained. We draw the walk — what the engine draws (`TERR-GEOM-036` (d)) |
| The exact rounding of the level ramp | `TERR-LIGHT-011` (amended) gives the accumulator's `+0x100` seed — half a `0x200` LUT row — but not the scale it runs at, so the rounding is not statable from the pin, and is not guessed |
| Sprite and unit geometry | Three further blitters index the same step table; located, unread. If they are the sprite paths, units stand on the same warped quads. Not in this pin — 0015 owns it, and until then the overlay output of a geometric render is knowingly unregistered |

## Removed

| What | Why |
|---|---|
| The standing research item asserting the displacement direction, and the direction's basis with it | The imported baseline took the direction from consulted third-party reimplementations of a related title. Golden rule 4 forbids that source outright, so the item is not annotated or downgraded — it is deleted, and the earlier basis is carried forward in no form. The value survives only because `TERR-GEOM-031` re-derives it from the game, independently |
| The baseline's edge-sampling and source-row formulas | Our own deterministic invention, disclaimed against the original even then. The decoded rule replaces them; the two do not agree, and keeping the invention with the real one in hand would be deliberate infidelity |
| "Adds an explicit displaced PNG mode", flat by default, and its requirement that every currently valid input keep exact PNG pixels | Contradicted by the corpus share above. The contract now states what stays stable and what changes |
| Three baseline clauses that do not apply here | Its `cellpx` downscale contract (this repo scales by integer replication upward); its four-corner-light exclusion (0007 ships it — this story excludes changes to the lighting *model* instead); its ban on combining displacement with the overlays (written for an opt-in mode — as the default it would disable both) |
