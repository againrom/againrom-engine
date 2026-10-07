# Spec — lighting the sprite layer

## Problem and current behaviour

The renderer shades terrain and leaves sprites raw.

A terrain pixel is attenuated per channel by a level in `[0,95]`, bilinearly interpolated from the
four corner levels of the cell it lies in, with a sky tint added per channel before the multiply.
Both front-ends apply it: the raster tool through the lit compositors, the window as a per-vertex
colour scale. Each front-end also carries an unshaded diagnostic that turns that attenuation off,
and the window additionally draws terrain unshaded whenever the map's altitude grid was not valid
at construction, because the level grid is built from those altitudes.

Sprites take no such step. One blit resolves a frame pixel's index in **that frame's own palette**
and writes the resulting colour at full opacity; transparency is structural, the palette entry's
own alpha is never read, and the frame's rectangle is clipped against the destination once. That
blit is the whole sprite path — static objects in the raster tool and in the window, unit sprites
in the window — and the standalone image a window texture is built from is the same blit onto a
transparent canvas. Nothing else touches a sprite pixel: no tint, no fade, no team colour, no
dimming, no selection recolour, no shadow pass. The window caches one texture per frame under the
frame's own pointer for the life of the session.

So a sprite is painted at full palette brightness onto ground that has been attenuated, at every
sun setting and in both front-ends, and units and objects read as cut-outs pasted onto a lit world.

## Functional requirements

- **FR-1 — the sprite ramp.** A sprite frame's pixels MUST be resolved through a **16-row** shading
  ramp over **that frame's own 256-entry palette**. Row `L` maps a palette entry's channel `ch` and
  the matching channel `t` of **the sun's own sky tint** — the same tint the terrain transform adds
  — to `clamp(((ch + t) * (16 − L) * 2) / 16, 0, 255)`, the division truncating toward zero. The sum
  `ch + t` MUST NOT be clamped before the multiply; the single clamp is the one written. Rows run
  from 0, gain 2.0, to 15, gain 0.125; **row 8 is gain 1.0 and MUST reproduce the raw palette
  exactly** wherever the tint is zero. A shaded colour MUST depend on nothing but that palette entry,
  the tint and the row — never on another sprite's palette and never on the terrain's.
- **FR-2 — one row per rendered frame, taken from the sun's ambient byte.** Every sprite in one
  rendered frame MUST be drawn at the same row, `clamp(ambient >> 2, 0, 15)`, where `ambient` is the
  ambient byte of the same sun the terrain level grid is built from. The row MUST NOT be derived
  from altitude, from the terrain's per-vertex level, or from which cell a sprite stands on, and
  static objects and unit sprites MUST take the same row.
- **FR-3 — one rule for every sprite.** A sprite's row and ramp MUST NOT vary with its class, its
  owner, the sheet it came from or the layer it is drawn in: two frames carrying equal palettes MUST
  shade equal indices to equal colours.
- **FR-4 — coverage is unchanged.** Shading MUST change only the *colour* of pixels the unshaded
  blit already paints. At every row, a transparent frame pixel MUST remain a no-op leaving the bytes
  beneath it exactly as they were; every painted pixel MUST be written fully opaque; the palette
  entry's own alpha MUST NOT be read; and the clip and the no-op for a frame wholly off the
  destination MUST be those already in force. The refusals stay exactly these four and gain no
  fifth: a nil destination, a nil frame, a non-positive dimension, and a pixel slice shorter than
  the frame's own `Width × Height`.
- **FR-5 — both front-ends, one result.** The raster tool's static-object layer and the window's
  static-object and unit-sprite layers MUST all be lit. For one frame, one row and one tint the
  pixels the raster receives and the pixels the window's texture carries MUST be identical.
- **FR-6 — the unshaded diagnostic covers sprites, and altitude does not gate them.** Where a
  front-end's unshaded switch is on, sprites MUST be drawn at raw palette values, byte for byte as
  they are drawn today; where it is off, they MUST be lit, and turning it off again MUST restore lit
  pixels rather than the raw ones. Sprite lighting MUST NOT be gated on the presence or validity of
  altitude data: a map drawn flat and with unshaded terrain for want of altitudes MUST still light
  its sprites.
- **FR-7 — the sun's parameters reach sprites through no switch of their own.** Where a front-end
  lets the ambient byte be chosen, that choice MUST move the sprite row by FR-2, with no second flag
  or setter selecting a sprite row directly; where a front-end fixes the sun, sprites MUST take that
  fixed sun's row.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | a palette spanning the channel range, zero tint | shaded at each of the 16 rows | every channel equals FR-1's expression computed independently in the test, the cases including at least one truncating division and one result clamped at 255 |
| **AC-2** | unit | a frame with an injective palette over a background varying on both axes | blitted at row 8 with zero tint, and blitted unshaded onto an identical background | the two destinations are byte-for-byte equal |
| **AC-3** | unit | every channel value 0..255 at zero tint | the sprite gain at each row `L` in 0..15 compared with the terrain attenuation at level `4L + 32` | the two agree on every one of the 4096 pairs |
| **AC-4** | unit | suns with ambient bytes `0x0e`, `0x00`, `0x03` and `0xff` | the sprite row taken from each | 3, 0, 0 and 15 |
| **AC-5** | unit | a frame carrying transparent pixels, an opaque pixel at index 0 and a palette entry with alpha 0, over a background varying on both axes | blitted at rows 0, 8 and 15 | at every row the pixels left untouched are exactly those the unshaded blit leaves, every painted pixel is fully opaque, and the alpha-0 entry paints its shaded colour |
| **AC-6** | unit | one frame, one row, one tint | blitted onto a raster and converted to a standalone image | the two agree pixel for pixel, transparent frame pixels reaching the image as transparency |
| **AC-7** | unit | two frames of different sizes carrying the same palette, one placed as a static object and one as a unit sprite | both drawn in one rendered frame | equal indices shade to equal colours and both are drawn at the row FR-2 gives |
| **AC-8** | unit | a raster render at ambient `0x0e`, the same render unshaded, and the same render at ambient `0x20` | the sprite pixels compared | the unshaded pixels are the raw palette's, the two lit renders differ from it and from each other, and each matches FR-1 at its own row |
| **AC-9** | unit | a window sprite drawn lit, then with the unshaded switch on, then with it off again | the pixels the window would upload for that frame in each state | lit, raw, and lit again — the third equal to the first and neither of the lit ones equal to the raw one |
| **AC-10** | unit | a nil destination, a nil frame, a zero-area frame, a frame whose pixel slice is shorter than its header claims, and rows of −1 and 99 | each blitted onto a prepared destination | nothing panics; the first four draw nothing and leave the destination byte-for-byte unchanged; the out-of-range rows draw at rows 0 and 15 |

**Error cases:** AC-10. Nothing here rejects input that is accepted today: a blit that cannot draw
still draws nothing rather than reporting, and an out-of-range row is held into range rather than
refused, so no caller gains an error path.

## Derived properties

- **P-1 (invariant)** — For any frame, row, tint and destination, the set of destination pixels
  written is exactly the set the unshaded blit writes: shading changes colours and never coverage.
- **P-2 (negative-invariant)** — For any blit the refusals reject, the destination is unchanged at
  every row, and no partial row or partial frame is written.
- **P-3 (completeness)** — A painted sprite pixel has exactly one outcome: one of the 16 rows, or
  the raw palette under the unshaded diagnostic. There is no third arm and no per-pixel exception.
- **P-4 (invariant)** — For any rendered frame, every sprite in it carries the same row, so no two
  sprites of one frame can differ in gain.

## I/O examples

The sun is the existing light: an angle, an ambient byte, a range byte and a sky tint. Only the
ambient byte reaches sprites, as `clamp(ambient >> 2, 0, 15)` — the fixed daytime sun's `0x0e`
gives row 3 and a gain of 1.625, against the 1.5625 that same sun gives flat ground. The gains
below name the rows FR-1 defines; FR-1's integer expression is the contract and these decimals
report it, so a path that reproduced the decimals but not the truncation would not satisfy it.

```
-ambient 14   (the default sun)   sprite row 3    gain 1.6250
-ambient 32                       sprite row 8    gain 1.0000   (raw palette)
-ambient 60                       sprite row 15   gain 0.1250
-unshaded                         no row          raw palette
```

No new flag, setter or config key is introduced: `-ambient` and `-unshaded` already exist and gain
this second effect, and the window's fixed sun already fixes the row.

## Constraints

The ramp a sprite is lit through is a product-visible boundary, and three shapes are credible:

| Option | What a viewer sees | Verdict |
|---|---|---|
| **A** — one ramp per sprite, built from that sprite's own palette | every sprite lit at one gain; two owners' units of one class identical; a sheet's own colours never displaced | **taken** (FR-1, FR-3) |
| **B** — one ramp shared by every sprite, built from a single palette | sprites whose sheets carry other palettes shift hue as well as brightness | rejected: sheets do not share a palette, and no sprite sheet carries the terrain's |
| **C** — a per-class or per-owner ramp, selected by a class field | two owners' units of one class differ in colour | rejected: no owner-keyed palette and no selector input exists here, so which owner took which ramp could only be invented |

Sprite gain above 1.0 saturates per channel at 255 rather than wrapping or rescaling, so bright
palette entries flatten toward white at the low rows — the same behaviour the terrain transform
already has, on art where it is more visible.

## Out of scope

- **Per-class and per-owner ramp selection** (constraint C). No class field or owner selects a
  ramp, a palette or a row, and none is carried into the render tier.
- **A luminance ramp.** Nothing here draws a sprite through a greyscale transform.
- **Sprite shadows** — a silhouette recoloured into the destination beneath a sprite. None is drawn
  today and none is added; this story lights the body pass only.
- **Overlay sheets.** A second sprite composited over a body, and the blend it would need.
- **Per-cell light sources.** Lamps and fires that brighten the cells around them, and any grid that
  would carry them: the row is uniform over the map (FR-2), so no sprite is lit differently from
  another.
- **The menu and UI art path**, which resolves no palette and takes no ramp.
- **RGB565 packing**, the day/night cycle, and any change to the terrain transform, the terrain
  level grid or the sun model itself.

## Verification mapping

Every criterion is a unit test over frames, palettes and suns built in test code; none reads a game
install, so all ten are CI-automatable. AC-3 is exhaustive over its 4096 pairs and AC-1 samples the
channel range, so P-1, P-2 and P-4 are **sampled, not proved**; P-3 is structural and is witnessed
by AC-2 and AC-8 together.

## Gate check

FR-1 → AC-1, AC-2, AC-3, AC-5, P-3 · FR-2 → AC-4, AC-7, AC-8, P-4 · FR-3 → AC-3, AC-7 ·
FR-4 → AC-5, AC-6, AC-10, P-1, P-2 · FR-5 → AC-6, AC-7, AC-8, AC-9 · FR-6 → AC-2, AC-8, AC-9, P-3 ·
FR-7 → AC-4, AC-8.
