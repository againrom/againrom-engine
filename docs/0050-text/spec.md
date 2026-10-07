# Spec — the game's own font, and the first drawn word

## Problem and current behaviour

The tree decodes a `.16` glyph atlas today — `font1.16` yields 224 grids of 4-bit values — and
cannot draw a word. Nothing knows that a character is a record, that a record's width is not how far
the pen moves, or what colour a 4-bit value is. A unit-information panel cannot be built on that,
and is why this story exists. Nothing that draws today changes.

## The font's second node

A font is **two files**. Beside the glyph atlas `<base>.16` sits `<base>.dat`, the
**advance table**: one unsigned 32-bit little-endian value per atlas record, in record order, with
no header, no trailer and no padding — 896 bytes for a 224-record atlas, 256 for a 64-record one.

`dat[g]` is how far the pen moves for glyph `g`, and it is **not** the record's width: the record is
a fixed cell and the proportional metric lives outside the sprite.

## The font

A font is a record-ordered list of **glyphs** plus one **letter spacing**. A glyph is a
`Width x Height` grid of pixels in row-major order with row 0 at the top, plus its own advance. A
pixel is a 4-bit **level** 0..15 that is meaningful only when the pixel is **painted**.

Paintedness is **structural and carried per pixel**: the atlas's own grammar leaves a cell's
background unwritten with skip and blank-row ops rather than spending a level value on it, so no
level is reserved to mean transparent and **level 0 is a painted pixel, not a hole**.

The letter spacing the engine constructs its fonts with is **2**.

## The arrangement, and the byte with no glyph

Record `k` is character `32 + k`. Read the other way, byte `c` selects record `c - 32`.

Text is a **byte string in the game's own arrangement, not UTF-8**: a string is walked one **byte**
at a time, never decoded as part of a multi-byte sequence, replaced or transcoded, so the whole
range `0x80..0xFF` reaches its own record.

A byte that selects no record — one below 32, or one whose record index is at or past the atlas's
record count — **selects record 0, the space**, and is then drawn exactly as record 0 is drawn. No
byte value can index outside the record list. Record 0 is empty in every shipped atlas, so the byte
paints nothing there and still costs the space's advance — a gap rather than a string that closes
up. Against a font holding no record at all, a byte selects nothing and moves the pen not at all.

## The placement rule

The pen starts at 0. For each byte of the string, its glyph's cell is placed with its left edge at
the pen, and the pen then advances by

```text
advance(g)  =  dat[g] + spacing                       for g != 0
advance(0)  =  dat[0] + spacing + height(0) / 2       for the space
```

where `height(0)` is record 0's own height and `/` is integer division. Glyphs are placed
**top-aligned**: every cell's top edge is the text's top edge. Cells may overlap, since an advance
is smaller than its own cell; that is the format's arithmetic and not an error.

## Measurement and drawing

**Measurement and drawing are two readings of one placement**, and neither carries pen arithmetic or
glyph selection of its own: two copies of that arithmetic drifting apart is what makes text measure
right in a test and draw wrong on screen. Three exact answers come off it.

**The advance** of a string is the pen's final position — where a following run of text starts. It
includes the spacing after the last glyph, as the engine's own layout does.

**Measuring** yields the box `(w, h)` that contains **every pixel drawing paints**, anchored at drawing's own position. `w` is the larger of the advance and one past the rightmost painted pixel —
the two differ in both directions. `h` is the font's **line height**, the tallest record it holds,
for a non-empty string, and 0 for the empty string, which measures `(0, 0)`.

**Drawing** paints the string into a destination image with the box's top-left corner at a
caller-given position in the destination's own coordinates, in a caller-given colour. For a painted
pixel of level `v`, the value written is the caller's colour with each of R, G and B scaled by
`v/15` — **truncating integer division**, so level 15 reproduces the colour exactly — and the
caller's alpha written through unscaled, keeping a premultiplied colour valid. The write
**replaces** the destination: no read of what was there, no blend. An unpainted pixel writes nothing.
Bytes draw left to right, so in an overlap the later glyph's painted pixels win. Anything
outside the destination's bounds is clipped away.

## Which font, and its two nodes

The two nodes are addressed `graphics/<base>/<base>.16` and `graphics/<base>/<base>.dat`.

**The default is `font1`** — 224 records, cell 16x15 — a default and not a lock, since the loader
takes the base name and the other two byte-control atlases (`font2`, 224 records, cell 8x10;
`font3`, 64 records, cell 8x6) load through the same call.

## Functional requirements

- **FR-1 (the advance table)** — A decoder turns the sidecar's bytes into one advance per record. It
  refuses a length that is not a whole number of 32-bit values, a table of more than **4096**
  entries, and an entry above **2048** — the atlas decoder's own record and dimension caps. The
  count is refused **before** anything is allocated for it, and an entry is compared **as read**,
  before a conversion could hide a large value. It never panics and yields no partial result.

- **FR-2 (the font model, at the tier that draws)** — The glyph, font and pixel above are plain data
  at the drawing tier, filled by a loader outside it. That tier reads no archive, opens no file and
  imports no container, format or simulation package.

- **FR-3 (the arrangement and the missing byte)** — As above, total for all 256 byte values against
  any font, including one with no records at all.

- **FR-4 (one placement)** — The pen rule above is stated once; the three answers are its only
  consumers.

- **FR-5 (measurement)** — Measuring yields the box above, and the advance is available on its own
  from the same placement, so a caller placing a second run beside a first need not recover the pen
  from the box.

- **FR-6 (drawing)** — Drawing paints as above into an in-memory RGBA image, clipped to it.

- **FR-7 (the font reaches the drawing tier)** — One loader turns an entry source plus a base name
  into a font, at the addresses and with the letter spacing above, refusing a pair whose record and
  advance counts differ and reporting which node failed. The default base name is a named constant.
  A missing or undecodable node is an error, and so is **an atlas holding no record** — a font
  without record 0 has no space to fall back on, and returning one is the silent empty font this
  clause exists to refuse.

- **FR-8 (the instrument)** — A developer tool reports a font's census from an install and renders a
  given string to a PNG at a path the caller names. The census carries the record count, the cell
  size, the advance range, how many records carry ink, **the histogram of painted levels including
  level 0's count**, whether record 0 is blank, and **how many records paint past their own
  advance** — the three clauses no unit test can falsify. The
  string is accepted as bytes as well as text, so a byte at or above `0x80` can reach it. The asset
  root comes from a flag or the environment, never a built-in path, and no image is written unless
  one is asked for.

- **FR-9 (what does not move)** — No simulation, digest, save, codec or map behaviour changes; no
  file under the determinism wall is opened; the two shipped sprite decoders behave as before. The
  drawing tier's new package imports nothing inside the module, so it adds **no outgoing**
  intra-module edge — the edges into it from its loader and its tool are the point of it. No game
  bytes enter the repository.

## Acceptance criteria

Synthetic throughout: every fixture below is built in test code, no test reads a game install, and
a byte at or above `0x80` is written as an escape, never as literal text.

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | sidecars of 0, 1 and 64 entries, and ones of length 1, 3 and 5 | decoding | the first three yield that many advances in file order, the last three are refused with an error and no result |
| AC-2 | a sidecar over the entry cap, one holding an entry over the value cap, and one at each cap exactly | decoding | the first two are refused naming the offending quantity, the last two are accepted |
| AC-3 | fonts of 224, 64 and 0 records, and every byte value 0..255 | selecting a record for each | 32..255 select records 0..223 where they exist, 0..31 select record 0, a byte past the record count selects record 0, and a font with no records refuses no byte and selects nothing |
| AC-4 | a font with distinct per-glyph advances and spacing 2 | measuring `"AB"`, `"A B"` and the empty string | the pen sums match the rule glyph by glyph, the extra `height(0)/2` appears once per space and follows record 0's height alone, and the empty string measures `(0,0)` |
| AC-5 | the same font, and strings holding a byte with no record, below 32 and past the record count | measuring | each such byte contributes exactly the space's advance |
| AC-6 | a font whose last glyph paints ink wider than its advance | measuring, then drawing at the origin into a large image | every painted pixel lies inside the measured box, whose width is at least the pen's final position |
| AC-7 | a glyph carrying levels 0, 8 and 15, drawn in a known colour | drawing onto a background of another colour | level 15 writes the colour exactly, level 8 writes it truncated to 8/15 per channel, **level 0 writes a painted pixel** and not the background, an unpainted pixel leaves the background byte for byte, and the alpha is unscaled at every level |
| AC-8 | any font and string | drawing at a position putting part of the text before the origin, past the right edge, above and below, and into a destination whose bounds do not start at the origin | the in-bounds part is identical to the same draw into an image with room for all of it; nothing outside is touched and nothing panics |
| AC-9 | a synthetic archive holding a sound atlas and sidecar under a base name | loading that base name | a font with one glyph per record, each carrying its own sidecar advance, and the letter spacing 2 |
| AC-10 | archives whose atlas node is absent, whose sidecar node is absent, whose atlas will not decode, whose sidecar will not decode, and whose two counts differ | loading | an error in each case, naming the node or the two counts; no font is returned |
| AC-11 | a font of known cells, inks, levels and advances, record 0 blank, one record painting past its advance | the tool's census over it | every figure FR-8 names equals the fixture's own, the level histogram included |
| AC-12 | the tool with no asset root available | running it | it reports the missing root and exits non-zero, reading no built-in path |
| AC-13 | a font whose records differ in height | measuring strings and the empty one | every non-empty string measures the tallest record's height, the empty one 0; the advance is reported apart from the box, equals the pen, and differs from the width when ink overhangs |
| AC-14 | a 224-record font, and a string of bytes at and above `0x80` written as escapes | measuring it | each byte selects its own record `c-32`, so the advance is the sum of those records' and not a run of one repeated glyph |
| AC-15 | an archive whose atlas decodes to zero records and whose sidecar holds zero entries | loading it | refused with an error naming the atlas; no font is returned |

## Properties

- **P-1** — The drawing tier stays pure: no file, archive, clock, window or graphics context; given
  a font and a string it is a function of its arguments.
- **P-2** — No simulation, determinism, digest or decoded-format behaviour is touched.
- **P-3** — For every font and string, every pixel drawing paints lies inside the box measurement
  reports. Held over generated strings, bytes with no record included.
- **P-4** — No byte can make a font index outside its record list, at any font size including zero,
  and no glyph can be made to read past its own grid.
- **P-5** — The drawing tier's new package adds no outgoing intra-module edge.

## Out of scope

- **A text layout engine.** No wrapping, justification, alignment, rich text, input field or line
  breaking. A newline is a byte with no record, so it is a space like any other.
- **Where the first word appears in the running game.** The panel decides its own placement; a
  caption planted now is a placement that story has to move.
- **The palette-bearing atlases**, and the sections past a count trailer.
- **A character-set conversion.** Bytes reach records unchanged; the arrangement past ASCII is none
  of CP866, CP1251 or KOI8-R, and no conversion is invented to bridge that.
- **The absolute colour of the game's text**, and which of its ramps a string gets.
- **Caching, atlasing or texture upload.** Drawing yields pixels in an image.
