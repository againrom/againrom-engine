# Spec — terrain graphics decoder & full-map render (ROM1)

**Provenance basis.** Every game fact here is research-derived. Story 0003's `pkg/formats/alm` decodes a
map's Tiles grid to raw `u16` tile words (EXP-0020 / ALM-GRID-012). Turning a tile word into a terrain
**picture** is now fully decoded by the research (EXP-0021, claims `TERR-LOC-001`…`TERR-VER-005`):

- **Location & format (`TERR-LOC-001`)** — the terrain tile images live inside `graphics.res` (the
  `&YA1` archive, 0001) under the node prefix `terrain.3d/`, as **standard 8-bpp Windows BMP** (magic
  `BM`, 14-byte file header + 40-byte `BITMAPINFOHEADER`, 256-entry palette, `bfOffBits = 1078`). Each is
  32 px wide and is a **vertical strip of 32×32 sub-cells**.
- **Loader (`TERR-LOAD-002`)** — the tiles load into a **128-slot** array `tiles[(G-1)·16 + V]`
  (`G = 1..8`, `V = 0..15`); only `tile1/2/3` (16 each) + `tile4` (`00..03`) + `dirt.bmp` ship, so the
  rest stay absent.
- **Tile-word → graphic (`TERR-IDX-003`)** — four agreeing `rom.exe` renderers select image + sub-cell
  from a tile word `w` with identical arithmetic (below).
- **Semantics (`TERR-SEM-004`)** — `tile1/tile2` = land strips, `tile3` = animated **water**, `tile4` =
  **road**; a non-water tile with **bit 13** (impassable) is composited over `dirt.bmp` before blit.
- **Corpus check (`TERR-VER-005`)** — 0 falsification violations over 38 maps / 880 552 cells.

The BMP **container** is a public documented standard (decoded generically); the `terrain.3d` strip
layout, the slot array, and the tile-word → graphic mapping are the decoded **game** facts. The
compositor, scaling, placeholder fill, and image output are the project's own engineering. Greenfield.

## Problem / goal

`pkg/formats/alm` yields raw tile words; nothing turns them into pixels. The goal is to render a whole ALM
map's terrain to an image using the game's real terrain graphics, from the research-decoded pipeline: load
the `terrain.3d/*` tile strips out of `graphics.res`, decode each as an 8-bpp BMP, slice it into 32×32
sub-cells, and for each map cell resolve its tile word to the correct strip + sub-cell and blit it at the
cell's map position at a caller-selectable scale. Water renders **statically** here (its base cell); the
time-varying phase cycling is 0006. Height and lighting are 0007.

## What research gives (usable now)

- The Tiles grid and raw tile words via `alm` (0003): `alm.Map.Tiles []uint16`, with `TileIndex(cell) =
  cell & 0x3ff` / `Impassable(cell) = cell & 0x2000` (EXP-0020). The render fields below are a finer split
  of the same word.
- The terrain-graphics **format & location** (`TERR-LOC-001`) and the **tile-word → graphic mapping**
  (`TERR-IDX-003` / `TERR-LOAD-002` / `TERR-SEM-004`), corpus-verified (`TERR-VER-005`).

## Format definition — generic 8-bit Windows BMP (public standard)

A standard uncompressed 8-bit `BITMAPINFOHEADER` BMP — the public documented Windows Bitmap format,
which the research confirms the `terrain.3d/*` tiles use verbatim (`TERR-LOC-001`, `bfOffBits = 1078`).

| Offset | Type | Content |
|---|---|---|
| 0x00 | 2 B | `"BM"` |
| 0x0A | u32 | pixel-data offset (`1078` for these tiles) |
| 0x0E | u32 | DIB header size (40, BITMAPINFOHEADER) |
| 0x12 | i32 | width (32 for terrain tiles) |
| 0x16 | i32 | height (positive ⇒ rows stored bottom-to-top) |
| 0x1C | u16 | bits per pixel (8) |
| 0x1E | u32 | compression (0 = BI_RGB) |
| 0x0E + DIB | 256×4 B | palette, entries `B,G,R,X` (X ignored) |
| pixel offset | w×h B | 8-bit indices, each row padded to a 4-byte boundary |

## Terrain tileset (research: `TERR-*`, EXP-0021)

**Files under `terrain.3d/` in `graphics.res`** (each 32 px wide; strip height ⇒ 32×32 sub-cell count):

| File(s) | Count | W×H | sub-cells | Role |
|---|---|---|---|---|
| `dirt.bmp` | 1 | 32×128 | 4 | impassable-tile composite overlay |
| `tile1-00..15` | 16 | 32×448 | 14 | Land group (Grass / Cracked / Sand / Savanna) |
| `tile2-00..15` | 16 | 32×448 | 14 | Stones / Cracked-Stones / Flowers-Savanna / Mountain |
| `tile3-00..15` | 16 | 32×256 | 8 | Water (animated) |
| `tile4-00..03` | 4 | 32×448 | 14 | Road |

Sub-cell `k` occupies the pixel byte offset `8 + k·0x400` into the loaded 8-bpp image buffer (`0x400` =
32×32 @ 8 bpp). Tiles load into a 128-slot array `tiles[(G-1)·16 + V]` (`G=1..8`, `V=0..15`); absent
files (`G=5..8`, `tile4` `V≥4`) stay null (`TERR-LOAD-002`).

**Tile-word → graphic mapping** (`TERR-IDX-003`), for a tile word `w` at map cell `(col, row)`:

```
g   = (w & 0x1fff) >> 6      strip group   (bits 6..12; corpus range 0..12)
b   = (w >> 4) & 3           blend column  (bits 4..5)
sub = w & 0xf                sub-cell      (bits 0..3)

image = tiles[g*4 + b]       == tileG-VV.bmp with G = (g>>2)+1, V = (g&3)*4 + b
cell  = image sub-cell `sub` (pixel offset 8 + sub*0x400)
```

Group → file → terrain (`TERR-SEM-004`; terrain labels inherited from `ALM-TERR-015`):

```
g 0..3  tile1  Land: Grass / Cracked / Sand / Savanna
g 4..7  tile2  Stones / Cracked-Stones / Flowers-Savanna / Mountain
g 8..11 tile3  Water (animated — see below)
g 12    tile4  Road
```

- **Water (`g ∈ 8..11`, `tile3`)** — the renderer overrides the group to `8 + phase`; with animation
  disabled `phase = 0`, so static water renders `tiles[8*4 + b]` (`tile3-0b`). The stored `g` (8..11)
  feeds the animation phase — that cycling is **0006**, not this story. Water uses only 8 sub-cells
  (`sub ≤ 7`, `TERR-VER-005`).
- **Impassable, non-water (`w & 0x2000`)** — the resolved cell is composited over `dirt.bmp` sub-cell
  `(col + row*5) & 3` before blit (`TERR-SEM-004`). The composite is a **transparent-keyed overlay, not a
  blend** (`TERR-DIRT-017`): the renderer copies the terrain sub-cell, then overlays the dirt sub-cell
  byte by byte, where a **non-zero** dirt palette index replaces the terrain pixel and a **zero** index
  (the 8-bpp transparent key, `SPR256-PAL-013`) leaves the terrain showing. No arithmetic mixes the two.

## Functional requirements

- **FR-1 (BMP decode)** A pure decoder MUST accept a standard uncompressed 8-bit Windows BMP byte slice
  and return a **paletted** image — the per-pixel palette **indices** plus the palette itself
  (`B,G,R → R,G,B`, opaque) — reading rows bottom-up into top-down display order with 4-byte row padding.
  The indices MUST be retained rather than resolved to colours, because two decoded behaviours address a
  terrain pixel by index: the dirt overlay's transparent key (index 0, `TERR-DIRT-017`) and the terrain
  blitters' shading table, addressed by `(brightness level, palette index)` (`TERR-LIGHT-011`). Slicing a
  strip into sub-cells MUST preserve the indices and the palette.
- **FR-2 (BMP validation)** The decoder MUST validate the subset (magic `BM`, DIB size 40, 8 bpp,
  compression 0, palette and pixel data within bounds) and MUST reject anything else atomically — an error
  and no result, never a partial image or a panic.
- **FR-3 (tileset load)** Given `graphics.res` (via `pkg/vfs`/`pkg/formats/res`, 0001), the loader MUST
  read each `terrain.3d/tileG-VV.bmp` into slot `tiles[(G-1)*16 + V]` and `terrain.3d/dirt.bmp` into its
  own slot, decode each (FR-1), and slice each strip into its 32×32 sub-cells (14 / 8 / 4 by strip
  height). Absent files leave a null slot; a null/short slot is a recorded condition, never a crash.
- **FR-4 (tile-word → graphic — the decoded mapping)** A pure function MUST map a tile word `w` at cell
  `(col,row)` to `(strip slot, sub-cell index)` using exactly `TERR-IDX-003`: `g=(w&0x1fff)>>6`,
  `b=(w>>4)&3`, `sub=w&0xf`, slot `= g*4 + b`, sub-cell `= sub`. For water (`g∈8..11`) the group is forced
  to `8` (phase 0, static — 0006 owns the phase). It MUST NOT read the word through any other bit split.
- **FR-5 (full-map composite)** The compositor MUST place each cell's resolved 32×32 sub-cell at its map
  position `(col*32, row*32)` at a caller-selectable integer scale, applying the `dirt.bmp` composite for
  impassable non-water cells (`w & 0x2000`, dirt sub-cell `(col+row*5)&3`), and MUST report any cell whose
  strip slot is absent/short — rendering a solid placeholder fill for it, never crashing.
- **FR-6 (headless render)** A developer/CLI harness MUST render a chosen map's terrain to an image file
  (asset root from `-assets`/`AGAINROM_ASSETS`, never hardcoded) and print a one-line summary (map W×H,
  cell count, and count of cells that fell back to a placeholder), exiting non-zero on load failure.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a synthetic 8-bit BMP (known palette/indices, positive height) | decoded | pixels match the palette (BGR→RGB), rows in top-down order |
| AC-2 | unit | a BMP with bpp≠8, or compression≠0, or truncated palette/pixel data | decoded | error; no result |
| AC-3 | unit | a BMP with a malformed/short header | decoded | error; no result; no panic |
| AC-4 | unit | synthetic 32×448 / 32×256 / 32×128 strips | sliced | yield 14 / 8 / 4 sub-cells, each a 32×32 block at pixel offset `8 + k·0x400` |
| AC-5 | unit | tile words spanning `g=0..12`, all `b`, all `sub`, incl. water and bit-13 | mapped (FR-4) | slot `= g*4+b` (`G=(g>>2)+1`, `V=(g&3)*4+b`), sub-cell `= w&0xf`; water forced to `g=8`; land `sub≤13`, water `sub≤7` |
| AC-6 | unit | a synthetic `res` archive of `terrain.3d/*` strips + a small tile grid (incl. an absent-slot word, an impassable non-water word) | composited (FR-3/FR-5) | each cell shows its resolved sub-cell at `(col*32,row*32)·scale`; impassable non-water is dirt-composited (`(col+row*5)&3`); the absent-slot cell is a placeholder and is reported |
| AC-7 | unit | non-BMP/short strip bytes, empty archive, out-of-range tile word | loaded/mapped | recorded null slot / placeholder; no panic; no out-of-bounds read |
| AC-9 | unit | an impassable non-water cell whose dirt sub-cell is palette index 0 across half its width and a non-zero index across the other half | composited | the index-0 half leaves the terrain pixel **byte-identical**, the non-zero half shows the dirt colour **outright**, and no pixel anywhere lands between the two (which a blend would produce) |
| AC-8 | manual | a real GOG install (`graphics.res` + a `.alm`) | `-assets <dir>` render | the composite matches the game's terrain; 0 placeholder fallbacks across the corpus; W×H and cell counts recorded in verification.md — no game bytes committed |

## Derived properties

- **P-1** (negative-invariant) For any input byte sequence the BMP decoder never panics or reads out of
  bounds; malformed input yields an error and no result.
- **P-2** (invariant) A decoded BMP returns exactly `width × height` pixels; a strip slices into exactly
  `height/32` sub-cells.
- **P-3** (invariant) FR-4's mapping is a pure function of `(w, col, row)`; for every corpus-valid word the
  resolved slot is a shipped file (`g∈0..12 ⇒ G∈{1,2,3,4}`) and the sub-cell is in range for its strip
  (land `≤13`, water `≤7`) — matching `TERR-VER-005`.
- **P-4** (negative-invariant) Compositing never indexes outside the strip, palette, or output image;
  absent slots and off-range words become placeholders, not panics.

## Constraints

- Terrain graphics come from `graphics.res` via `pkg/vfs`/`pkg/formats/res` (0001); the asset root comes
  from `-assets`/`AGAINROM_ASSETS`, never hardcoded; no game data in the repo.
- Unit tests use **synthetic** BMPs, synthetic `res` archives, and synthetic tile grids only — green with
  no game install.
- The BMP decoder, strip slicer, tile-word mapping, and compositor live in the render tier; the mapping
  and geometry take plain integers / decoded images (no `formats` import), and the harness wires
  `alm.Map.Tiles` + the `res` archive.
- Water is rendered **static** (phase 0); the animation phase is 0006. Height/lighting are 0007. The
  legacy non-3d `terrain\tile*.bmp` fallback set is out of scope (the shipped path is `terrain.3d`).

## Out of scope

- Water-tile animation cycling & cadence (0006), height-displaced terrain / relief lighting (0007),
  land/water edge blending, fog of war.
- Terrain-class / passability resolution (needs `map.reg` — the mapload/data tier, `ALM-TERR-015/016`).
- The interactive windowed viewer (0005); objects/units overlays (0008/0009); the type3 object-overlay's
  own graphics (a separate render path).

## Verification mapping

AC-1…AC-7 + P-1…P-4: unit tests on the BMP decoder, strip slicer, tile-word mapping, and compositor
(synthetic bytes/archives/grids). AC-8: a developer-run render of a real GOG map, evidence (W×H, cell
count, 0 placeholder fallbacks) recorded in verification.md — no game bytes committed.
