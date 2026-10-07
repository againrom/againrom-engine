# Provenance — corner-interpolated lighting in the window

`claims/retracted.md` was read first. It is why the row backing FR-4 below is read narrowly: the
claim this story leans on hardest carried High for twelve experiments while describing only half of
what its two routines do.

## Backing

| `spec.md` anchor | Claim | Confidence |
|---|---|---|
| FR-2 — a tile's interior brightness is interpolated from its four corner values, not filled flat | `TERR-LIGHT-011` (amended) | High — the LUT index arithmetic and the interpolation, instruction by instruction, in two independent blitters. Read narrowly: the row's original text described the colour half only, and the span-divided ramp is part of its correction, not of the original |
| FR-2 — the flat path interpolates too, so lighting is not a property of the sloped path | `TERR-LIGHT-011` (amended) | High — both blitters are named and both index a level dimension; they differ in the ramp's divisor, not in whether there is one |
| FR-3 — the corner values are per-**vertex** on a `Width*Height` grid, computed at runtime, never stored | `TERR-LIGHT-012` | High — three allocations and their readers in one function; the grid the blitters sample has no file read anywhere on the path |
| FR-4 — the shade is `(channel + tint) * (96 - level) / 32` per channel, integer, truncating | `TERR-LIGHT-019` | High — the per-entry transform read as integer instructions, including the trunc-toward-zero shift and both 16-bit clamps |
| FR-4 — the multiplier is above 1 over most of the range: level 64 is ×1.0, level 0 is ×3.0, higher level is darker | `TERR-LIGHT-020` | High — the row multiplier `(96−L)/32` and the unattenuated row are both named; the flat daytime level 46 → ×1.5625 follows with `TERR-LIGHT-013` |
| FR-4 — the sky tint is the only additive term, and it is `(0,0,0)` at day | `TERR-LIGHT-021`, `TERR-LIGHT-014` | High for the builder's only global reads and where they enter; High for the daytime band being untinted |
| FR-5 — the level grid, the light parameters and the far-edge clamp are 0007's, unchanged | 0007 | carried unchanged, backed there (`TERR-LIGHT-013`/`-028`/`-030`, `TERR-EDGE-024`…`-026`) |
| FR-1 — the geometry, extent, culling and mode contracts this story must not disturb | 0012, 0013 | carried unchanged, backed there |

## Ours by choice

| What the spec fixes | What the evidence actually says |
|---|---|
| The shade reaches the GPU as a **vertex colour**, one multiplier per tile corner | The engine has no GPU and no vertex colour: it walks a fixed-point accumulator down a column and indexes a baked `[96][256]` table. Nothing in the pin licenses or forbids the carrier; only the arithmetic is pinned |
| The interior is therefore **Gouraud over two triangles split TL–BR**, not bilinear over the quad | The engine's ramp is separable — a step across columns and a step down each column — which is bilinear, not triangle-split. The diagonal is our decision and it changes the interior of every non-parallelogram tile |
| The level is **not truncated to an integer row** on the window's path | The engine indexes one of 96 discrete rows, so its interior is stepped where ours is continuous. A truncation would need a fragment shader, which this story does not add |
| The window's light is `DefaultDaytime`, with no flag to move it | `TERR-LIGHT-023` establishes that the map's own stored fields are overwritten before use, so reading them is a viewer choice; `terraintool` exposes that choice as `-maplight` and this story does not extend it to the window |
| `-unshaded` exists on `mapview` and not on `againrom` | Nothing decoded says anything about an unlit mode. It is a developer diagnostic and the A/B this story is checked by, so it lives on the developer tool |
| A placeholder cell stays unlit | The engine has no placeholder: an absent tile slot is our own diagnostic fill, and shading a diagnostic would disguise it |
| Holding the primary button pans, cursor-anchored, from tick-to-tick deltas | No claim describes how a player moves the engine's scroll origin. What is decoded is where the origin may go (`CMapView`'s clamp, `TERR-EDGE-026`) and that the draw loops are viewport-relative — never which input moves it. The gesture, the button and the anchoring are ours; the clamp they obey is not |

## Open / undecoded

| What | State |
|---|---|
| The exact rounding of the engine's level ramp | `TERR-LIGHT-011`'s accumulator is seeded `+0x100`, half a `0x200` LUT row, but the claim does not give the scale it runs at, so the rounding is not statable from the pin. Recorded open by 0012 and left open here |
| What the far-edge outer ring should show | The engine never computes it (`TERR-EDGE-025`) and the allocator does not zero it, so what those bytes hold at runtime is **Unknown**. 0007's clamped read is inherited, not reopened |
| Whether shipped play runs with the day/night cycle off | Open. `TERR-LIGHT-030`'s literal is the cycle-off default, which is a labelled reference point rather than a claim about a live session |
| Whether the window's Gouraud interior and the CPU bilinear agree on a given pixel | Not measured. They provably differ on a non-parallelogram tile; the spec claims no pixel equality between them |

## Removed and why

Deleted outright, not annotated, under golden rule 4 — the baseline attributed each to
reimplementations of a related title, and a labelled hypothesis still steers the next search:

- the per-vertex light value `normal·sun·64+96` and its neutral value 144;
- the `LightScale(b) = b/144` byte-to-multiplier mapping, and its `0 → 1.0` hidden-border convention;
- the competing `atan2` height-delta shadow model with range `0..94`;
- the statement that the two sources disagree about the interior model, and the framing of
  per-vertex bilinear shading as an unproven fidelity claim — `TERR-LIGHT-011` settles it from the
  binary.

Nothing here left a hole: the level grid, the transform and the interpolation all come from claims
this repo already ships code against.
