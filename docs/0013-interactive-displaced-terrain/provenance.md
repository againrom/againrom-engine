# Provenance — interactive displaced terrain

Re-read against research `03a9448`, 2026-07-30. `TERR-SPR-041`'s identification is retracted at the
**High** it held: the `vt+0x30` dispatch it named draws the HP/mana bars, and the unit body draw is
`vt+0x28` = `R0552` (`TERR-SPR-065`), which ignores arguments 1–2 and reads the third, a
light level (`TERR-SPR-067`). The Unknown this story fenced off is therefore closed. The fence
still holds: placements are out of scope here and the overlay-forces-flat rule is exactly it, so no
row below moves.

## Backing

| Spec anchor | Claim | Confidence |
|---|---|---|
| *Displaced geometry* — `V(c,r) = r*32 - h(c,r)`; the altitude is subtracted, read signed, from the map's own altitude grid | `TERR-GEOM-031` | **High** for the formula, the sign, the signedness and the grid's identity — one named instruction each, one writer. Signedness is decidable only from the instruction: no shipped altitude byte reaches `0x80`, so no corpus can separate a signed from an unsigned read |
| *Displaced geometry* — a larger altitude moves a pixel **up** the screen | `TERR-GEOM-031` | **High**, with the last link named: the destination row is bounded by two fields the engine passes as a `RECT`'s `top`/`bottom`, and "smaller row is higher" is then the Win32 device-coordinate convention, not a ROM1 instruction |
| *Camera and culling* — the horizontal tile range is unaffected; FR-5's "no altitude term reaches a destination column" | `TERR-GEOM-035` | **High** — the destination-address arithmetic carries no altitude-derived term, and both clip tests were read instruction by instruction |
| *Rendering mode selection* — that a placement's own lift is a **different** grid from the one terrain is drawn on, which is why an overlay forces flat until a later story | `TERR-GEOM-031` (amended), `TERR-SPR-039` | **High** that the two grids are different functions of the same four corners; **Medium** that the lift stands a sprite on the drawn ground (a corpus cross-check between two transcribed paths) |

## Ours by choice

Each of these is engineering this story fixes and a later one may change without contradicting any
source.

| Spec anchor | What is ours |
|---|---|
| *Rendering mode selection* | The whole rule. Which inputs make **our** renderer switch modes is a decision about our renderer; the engine has no such mode. Also ours: that the decision never reads altitude values, so an all-zero grid still displaces |
| *Displaced geometry* — world coordinates | The canvas, its `[MinV, MaxV)` extent and the translation by `MinV`. The engine has a clip rect, a scroll origin and an over-scan margin; it defines no map-sized world and has no scale |
| *Displaced geometry* — the far-edge ring | A cell owns four corners, so the last cell row and column need vertices outside a `Width*Height` grid. The engine reads them at a raw flat offset into that unpadded grid and the allocator does not zero it (`TERR-EDGE-024`). We clamp the index into the grid instead — the intent-matching treatment for a port, never a claim about what the engine draws there |
| *Displaced geometry* — tile drawing | Sampling one texture across the whole displaced quad. The engine instead selects a flat or a sloped blitter and, on the sloped path, resamples 32 source rows into each destination column's own span (`TERR-GEOM-032`…`034`). We deliberately do not transcribe that here, and claim no pixel equality with it or with the original |
| *Displaced geometry* — painter order | Row-major, later tile wins. It matches the PNG raster's order; it is our choice in both places |
| *Camera and culling* | Pan, edge-scroll, zoom, clamping and centering are the project's own UX and reproduce no documented original camera. The padding rule — over-cover freely, never under-cover — is ours |
| *Tile drawing* — placeholder fill | Our diagnostic for a source cell the archive does not carry |

## Open

Undecoded, and deliberately given no meaning here.

| Question | Claim | How this story avoids depending on it |
|---|---|---|
| Whether the outermost terrain ring ever reaches a visible pixel, or always stays inside the engine's culled over-scan margin | `TERR-EDGE-026` | Our camera clamps to our own world and reproduces no over-scan margin; nothing in the contract asserts what the engine shows there |
| What the engine's uncomputed outer brightness ring holds at runtime | `TERR-EDGE-025` | The window applies no shading at all, so the question cannot arise in this story |

## Removed

Statements the imported baseline carried that are not in this contract.

| Dropped | Why |
|---|---|
| Its research disclosure that the displacement direction and scale are unproven for this game, held as "a working hypothesis, not a proven fact" | Closed from the game itself by `TERR-GEOM-031` and `TERR-GEOM-035`. The baseline's basis was the direction supplied by two reimplementations of a related title; golden rule 4 forbids that source, and it is not carried forward in any form |
| "The per-cell single light value is applied uniformly across the whole tile, exactly as in flat mode" | Flat mode applies no lighting. There is no such behaviour to preserve and none is added |
| Drag panning, listed among the behaviours that must not drift | The viewer has none |
| "The game front-end requests both overlays, so it stays flat here" | Ours requests neither, so the same rule leaves the game displaced. The consequence is a criterion, not a caveat |
| Named types and commands from a differently-named tree | They do not exist here; the contract names behaviour rather than symbols |
