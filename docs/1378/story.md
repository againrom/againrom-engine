# One decoder per format

## Intent

Each image and sound format the engine reads has one decoder under
`pkg/formats`. Every caller goes through that decoder, the copies are deleted
and no fallback copy is kept. A defect fixed in one decoder then reaches every
screen at once. Nothing a player sees or hears changes.

Base: `5d22d8cc` (game 0.102.0). Reconciled main: `51cccab7` (game 0.104.0,
knowledge k208).

## Authority

| Part | Authority |
|---|---|
| `.16a` pixel: palette colour at coverage (level+1)/16, premultiplied | SPR16A-031 |
| palette-less arrow sheets draw through the shared `projectiles.pal` table | PAL-PROJ-011 |
| BMP header, row order by height sign, stride, colour table, RIFF WAVE chunks | the public file formats |
| one decoder per kind | owner direction |

## Copies found and removed

| Format | Copies before | One decoder now |
|---|---|---|
| WAV | `audio.DecodeWAV` and `audio.DecodeTrackWAV`, two chunk walks | `wav.Parse` (`pkg/formats/wav`); both audio paths keep only resampling |
| BGRA palette entry | `pal.Decode`, `pal.DecodeOwnerTables`, the `spr256` and `spr16` palette loops, the 8-bit BMP loops in `render/menu` and `render/terrain`; colour re-wraps in `units.go`, `projectiles.go`, `statics.go` | `pal.Entries` (4-byte entries) and `pal.Pixels` (3-byte pixels); `Color.Opaque`, `Table.Opaque` |
| `.256` frame to RGBA | `decodeCursor256Frames`, `indexedEffectFrames`, `loadTipGemFrame`, `cmd/sprtool` | `Sprite.Table`, `Frame.Colors`, `Frame.RGBA` (`pkg/formats/spr256`) |
| `.16a` frame to RGBA | `LoadAttackPointer`, `decodeCursor16AFrames`, `loadItemIcon`, `effectFrames`, `cmd/sprtool` | `spr16.Resolve`, `FrameA.Colors`, `FrameA.RGBA` |
| BMP | `pkg/formats/bmp` (24-bit), `render/menu` 24-bit and 8-bit decoders, `terrain.DecodeBMP8`; colour-grid to RGBA in `portrait.go`, `spellicon.go`, `cmd/buttonframecheck`, `cmd/plaqueseams` | `bmp.Decode`, `bmp.DecodeRGBA`, `bmp.DecodePaletted`, `Image.RGBA`, `Image.SubRGBA` |
| `units.reg` | parsed by `LoadUnits` and again by `LoadUnitSounds` at front-end start | `loadUnitRegistry`: one read gives the unit set and the sound table; the install share carries both |
| attack cursor | read and decoded by `LoadAttackPointer` and again as the registry's attack slot | `loadCursorArt`: each slot sheet read once; the pointer is the attack slot's frame 0 |

`pkg/formats/winicon` stays apart. An icon image is a DIB without a file
header, with a doubled height carrying an AND mask, depths 1, 4, 8, 24 and 32,
and PNG entries. It shares no header or row rule with the bitmap subset
`pkg/formats/bmp` reads.

## Defects and variants

Defects, none player-visible:

- The BMP pixel-run check multiplied height by stride. On a hostile header the
  product could overflow and pass a run that is not present. Every copy had
  it. The one decoder compares by division. Test: the
  "overflowing dimensions" rows of `TestDecodeRefusesEveryOtherShape` and the
  paletted refusal tests in `pkg/formats/bmp`.
- `cmd/sprtool png16a` drew a painted pixel at alpha level*17, so a level-0
  pixel was invisible. The game draws it at coverage 1/16 (SPR16A-031). The
  tool now uses `spr16.Resolve`. Developer tool only.
- A `.256` cursor sheet without a palette was drawn opaque black, while the tip
  gem refused the same sheet. Cursor decode now refuses it. No shipped cursor
  sheet lacks a palette.

Variants kept as named options:

- `.256` table: the sheet's own palette, or a shared table passed in
  (`Frame.Colors(table)`). The palette-less arrow sheets take the shared
  `projectiles.pal` table (PAL-PROJ-011).
- BMP depth: `Decode`/`DecodeRGBA` read 24 bits, `DecodePaletted` reads 8 bits
  and keeps indices for hit masks and terrain tiles. Row order follows the sign
  of the height, the pixel run starts at the file's own offset, and only an
  8-bit file reads a colour table. These are public-format rules.

## Proof

- Decoder tests on synthetic bytes: `pkg/formats/wav` (chunk order, unknown
  chunks, 8-bit widening, channels, every refusal, truncation),
  `pkg/formats/pal/entries_test.go`, `pkg/formats/spr256/rgba_test.go` (own and
  shared table, nil sprite), `pkg/formats/spr16/rgba_test.go` (levels 0, 7 and
  15), `pkg/formats/bmp` (top-down rows, later pixel offset, a declared table
  at 24 bits, 8-bit paletted decode, opaque RGBA, sub-rectangles).
- Decode once: `pkg/game/decodeonce_test.go` counts reads of the attack sheet
  and of `units.reg` and holds each at one; a missing slot fails the registry
  and keeps the pointer.
- Architecture ratchet: `internal/archtest/decoders.go` scans every production
  file outside `pkg/formats` for a RIFF chunk walk, a blue-green-red byte
  conversion, a `pal.Color` channel copied into pixels, or a read of a decoded
  `.256` or `.16a` pixel. A synthetic source proves each shape is seen. The
  composer's files are a list that may only fall.
- Per-screen hashes: an uncommitted harness hashed every loaded screen image,
  every cursor, icon, effect and tile frame, and every sound on EN, RU and the
  ROM2 RU root, before and after. EN 112 488 lines, RU 112 371, ROM2 193 274.
  Every screen and frame line is equal except two. The harness-only
  `s256cursor` line on the six palette-less arrow sheets changed because cursor
  decode now refuses a sheet with no palette; the game draws those sheets as
  projectiles through the shared table, and that line is equal. The world-map
  route graph line differs between two runs on the same base and is not art.

## Open debt

- `LoadUnitSounds` stays as a test seam over `loadUnitClasses`.
- `terrain.DecodeBMP8` and `game.cursorPixel` stay as one-line forwarders to
  the one decoder for files the town composer owns.

## Left to the town composer

`internal/archtest.DecoderComposerDebt` lists the files that still decode
themselves: `pkg/game/shopart.go`, `pkg/game/worldmap.go`,
`cmd/schoolcheck/main.go` and `cmd/townsquarecheck/main.go`. When each moves
onto `pkg/formats`, its entry and the two forwarders leave in the same commit.
