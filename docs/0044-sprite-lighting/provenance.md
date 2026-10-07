# Provenance — lighting the sprite layer

Pinned at research `ce40c15` (`ce40c15d32dc5323098119b56de891d6f6741733`), frozen for the story.
`claims/retracted.md` read first: nothing in `TERR-LIGHT-059…064` is overturned, and the two rows
here that amend others — `TERR-SPR-065`, `TERR-SPR-066` — are the amendments, not their casualties.

## Backing

| `spec.md` anchor | Claim | Confidence as published | Pin |
|---|---|---|---|
| A sprite is lit at all, and the body pass is the lit one (Problem; FR-1) | `TERR-LIGHT-059`, `TERR-SPR-065` | High | `ce40c15` |
| The ramp: 16 rows, tint per channel before a truncating multiply-divide, clamped, gain `2(16 − L)/16` (FR-1) | `TERR-LIGHT-060` | High | `ce40c15` |
| The row is `ambient >> 2`, uniform over the map, objects and units reading one grid, and not the terrain's brightness (FR-2, FR-3, FR-6) | `TERR-LIGHT-061` | High for the grid / **Medium** that the byte reaches the *unit* blit on every path | `ce40c15` |
| Row 8 is the raw palette, and the sprite ladder is the terrain ladder sampled every fourth row (FR-1's row-8 clause; AC-2, AC-3) | `TERR-LIGHT-062` | High, the bit-equality measured at `L = 32…64` / **Medium** that a live session runs at this ambient and θ | `ce40c15` |
| The defect's size: 65.2 luma with no table against 104.9 at the engine's row on a shipped frame, terrain 106.1–118.3 in the same crop (Problem) | `TERR-LIGHT-062` | High for the ladder; the luma figures are that row's own owner-review measurement | `ce40c15` |
| A sheet is lit through a 16-row mode-2 table from **its own** palette, handed to the blit per sheet (FR-1, FR-3) | `TERR-LIGHT-064` | High | `ce40c15` |
| Terrain and sprite palettes are disjoint — 53 terrain tiles carry 1 palette, 1373 sheets carry 1225, and **0 of 1373** is the terrain's, the closest off by a **max per-channel delta of 201** — so no sprite may take the terrain ramp, and none is near enough to borrow it either (constraint B) | `TERR-LIGHT-064` | High | `ce40c15` |
| The transform clamps per channel and does not stop short of white (FR-1's clamp; constraints) | `TERR-LIGHT-063` | High | `ce40c15` |

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| **One rule for every sprite** — the object arm's construction (a 16-row mode-2 ramp from the sheet's own palette) applied to units too (FR-1, FR-3) | Decoded for object sheets. A unit's shading object is selected on a class field with three arms — 16 shared tables, the class's own, or one per owner — and which registry key fills that field is **Unknown**, as is the node the shared arm's 16 KB palette blob is read from (`TERR-LIGHT-064`). Two arms therefore have no palette source we hold, so the choice was never merely which arm: the selector is deferred, and nothing here asserts it |
| The row clamped into `[0,15]` (FR-2) | The grid holds a byte and the table has 16 rows; no published path bounds one to the other, and `ambient >> 2` exceeds 15 from ambient 64 up, which our own flag admits |
| The unshaded diagnostic extended to sprites (FR-6) | Ours entirely: the engine has no such switch |
| Sprite lighting ungated by altitude validity (FR-6) | The decoded row carries no relief term (`TERR-LIGHT-061`); the altitude guard is our own fallback, not extended here |
| The sky tint carried on the sprite ramp (FR-1) | Mode 2 does add a tint before the multiply (`TERR-LIGHT-060`); our sun's is `(0,0,0)` everywhere, so the parameter is kept for shape, not for a value anything produces |
| The row applied where the window's texture is built, and the cache keyed by it | Engineering; no engine fact is claimed about texture lifetime |
| No greyscale sprite (out of scope) | `TERR-LIGHT-060` names a mode-5 greyscale ramp a unit takes on one override, and what the two overrides are for is **Unknown** (`TERR-LIGHT-064`) |

## Open / undecoded

- **The unit selector.** The class field is the *length* of the class's shading-table array, so
  0 / 1 / >1 select the shared tables, the class's own, or one per owner; which registry key fills
  it — and therefore which of the 34 classes takes which arm — is **Unknown**, as is the source node
  of the 16 KB blob (`TERR-LIGHT-064`). This story assigns it no meaning.
- **Whether the level byte reaches the unit blit on every path** — `TERR-LIGHT-061`'s own Medium:
  the body routine also reads that argument slot as a frame index, forces it to 0 on one path, and a
  second dispatch pushes 0. We light units at the row objects take.
- **The two shade-object overrides**, the per-draw greyscale included (`TERR-LIGHT-060`/`064`).
- **Per-cell light stamps.** `TERR-LIGHT-061` publishes the grid's other two stages, the level being
  uniform elsewhere. We carry no lit-object list, so the uniform case is the whole of it.
- **Whether a live session runs at this ambient and θ** (`TERR-LIGHT-062`, Medium).
- **The shadow and overlay passes.** `TERR-LIGHT-059` locates both and reads neither.
- **What a later story must not inherit.** This one is render-only and reaches no hashed simulation
  state, which is why it may ship with the selector Unknown. A story carrying lighting into hashed
  state must re-open the selector rather than take "one rule for every sprite" as decoded.

## Removed from the baseline and why

- **The sprite level as the shroud level, and as a sample of the terrain brightness grid.** Both
  named and refuted by `TERR-LIGHT-061`.
- **The fifth blit argument as a second level.** Retracted as a level by `TERR-SPR-066` — it is the
  shadow's per-row X slope — so no second brightness input enters the body pass.
- **"Unit sprites unlit, as object art is."** The standing choice this story replaces; its own
  register row already recorded that the evidence had gone against it at this pin.
## Appended 2026-08-02 — pin `9ff259c`: `TERR-LIGHT-064`'s `> 1` arm is REFUTED, and not on any clause this story built on

`TERR-LIGHT-064` stands at `● active (amended)`. What `claims/retracted.md` classes **REFUTED** is
the label on the selector's third arm: `> 1` is **per-tier**, not per-owner. The subscript is
`unit+0x24 − 1` and `unit+0x24` is the actor's **`face`** — the `Data.bin` Units/Humans tier column,
identified by two routines computing the same `typeID*4 + face` into the same cache
(`PAL-FACE-005`, `PAL-TIER-004`). The *owner*-subscripted arm is the **other** branch of the same
`if`: `Palette == 0` takes one of the 16 team palettes of `human.pal` (`PAL-OWN-007`, High for the
build loop, **Medium** that the subscript is the owner's colour slot). The row named both arms and
swapped them.

**Neither clause this story cites is either arm.** The *Backing* row for FR-1/FR-3 cites the `== 1`
case — a sheet lit through a 16-row mode-2 table from **its own** palette — which is the middle arm
and untouched; the disjointness row cites a palette census, a measurement with no arm in it.
`pkg/render/terrain/spriteshade.go` cites that same census. **No shipped pixel is wrong because of
this and no code changes.**

**What moves is the *Open* item, which largely closes.** Three of the row's `Unknown` clauses are
answered: `class+0x98` is the `units.reg` key **`Palette`** (`PAL-KEY-002`); the 34-class split is
**18** at `Palette` 0 (every human/hero), **3** at 1, **13** at 4 (every monster class shipping
tiers); and the 16 KB blob is the `human.pal` node, whose 16 sub-palettes differ from sub-palette 0
on exactly 55 fixed indices (`PAL-OWN-007`). So the *Ours by choice* row for **one rule for every
sprite** keeps its ruling and loses its reason: it reads "two arms therefore have no palette source
we hold", and both sources are now held.

**The divergence is therefore no longer unquantified.** Under one rule for every sprite the 13
tiered classes draw every tier through the sheet's own palette where the engine draws
`class+0x9c[face−1]` — and `PAL-TIER-004` measures tier 1 as byte-identical to the sheet's own
palette on 13 of 13 classes, so our output is **exact for tier 1 and wrong for tiers 2-4**; the 18
human/hero classes draw with no team colour where the engine selects one of 16. Whether the sheet's
own palette coincides with any `human.pal` sub-palette is not established at this pin. Lifting
either is a story. The *Open* section's standing rule — a story carrying lighting into hashed state
must re-open the selector — is unchanged, and the selector is now decoded rather than Unknown.
