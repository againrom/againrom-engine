# The first game's mission minimap

## Intent

The first game's mission minimap shows the original's picture instead of this
build's symbolic map: MiniMapData terrain under the map's light, the CrystalR
background through unexplored and half through explored fog, owner-coloured
structure and unit marks, and the white view outline. Known defect B9;
`DIV-2105`. Base: public main `8a053839`, merged with `b61fc0a6`; pin k254.

## Authority

- `MISSION-083`: terrain from tile word `t & 0x1fff` at `(t>>4, t&15)` of the
  52x14 MiniMapData.bmp, brightness factor `96 - trunc(sum of four levels/4)`,
  channel `min(255, c*factor>>5)`, the eight-cell border excluded, scale steps
  and picture origin.
- `MISSION-084`: the masked OR of fog words `i, i+1, i+W+1, i+W+2`; OR 0 copies
  the CrystalR pixel, exactly `0x8000` averages per channel, any bit 14 keeps
  terrain.
- `MISSION-085`: structure and bridge extents, a unit's one cell, the ordinary
  quad gate, a unit's corpse stage at most 1, the colour word `+0x1148` of the
  owner's shade table, null-owner output Unknown.
- `MISSION-086`: the white outline, near edge `scroll-8`, inclusive far edge,
  per-pixel clipping; the widget-edge clip untraced.
- `MISSION-087`: the paint keeps its last picture while
  `abs(counter - last) < 11`; the sender set is open.
- `MISSION-088`: EN and RU executables and the three payloads are identical.
- `MISSION-063` (with its retracted clauses), `MISSION-064`, `AI-MINIMAP-062`,
  `AI-MINIMAP-124`, `AI-MINIMAP-157` and `SHOP-FIGURE-041` place and identify
  the widget.
- Owner directions kept: fill scale (`DIV-1281`, `DIV-2994`); a press or drag
  only moves the camera (`DIV-320`); loot dots unchanged (`DIV-2954`).

## As built

`pkg/game/screens/rom1.json` names a `mission-minimap` screen whose one picture
is `graphics/interface/minimapdata.bmp`. `missionMinimapPaint`
(`pkg/game/missionart.go`) loads it through the screen-art loader and takes
entry 164 of each of the sixteen `human.pal` owner tables; the second game's
description has no such screen, so its viewer gets no paint. `SetMinimapPaint`
hands the paint to the viewer.

`pkg/ui/minimaporiginal.go` builds the picture when the paint and the crystal
frame are present. `minimapPlan` and `composeMinimapPlan` (`pkg/ui/minimap.go`)
are the one builder for both pictures; the symbolic picture is the same plan
with its own pixel rule.

- Terrain: one colour per cell, rebuilt when the level plane is replaced. The
  level is the cell's four corner levels of the terrain shade plane.
- Window: the playable cell rectangle, the map without its eight-cell border.
  The picture sits at `(72 - width>>1, 82 - height>>1)` from the crystal's
  left edge, filled on the tighter axis.
- Fog: the shifted four-word block of `MISSION-084`, from the viewer's fog
  plane (visible `0xc000`, explored `0x8000`, unseen 0). An unseen block is
  the CrystalR pixel, opaque.
- Marks: structures from the structure snapshot with the class's
  `TileWidth`/`TileHeight` (a `VariableSize` class takes the object's own
  size); units one cell. Every mark needs its quad's OR to be `0xc000` and an
  owner with a colour; a unit needs corpse stage at most 1, and another
  owner's unit its own cell visible. Structures draw first, then loot dots,
  then units. A structure's owner is its map record's owner field
  (`structureOwnerSlots`, `pkg/game/world.go`).
- Outline: white, near edges at the scaled `scroll-8`, far edges at the scaled
  `scroll-8+span` minus one, clipped per pixel to the picture.
- Repaint: `minimapRepaint` keeps the last picture while the presentation-tick
  counter has moved fewer than 11 steps and no repaint key (box, frame, paint,
  outlined cells, reveal, loot option) changed.

`pkg/game/mapreveal_release_test.go` computed its interior window from
`image.Rect(w, h, 0, 0)`, which `image.Rect` canonicalises to the whole map;
the window now starts empty and the witness reads interior cells.

## Proof

Focused tests, `pkg/ui/minimaporiginal_test.go`, each with a loss control:

| test | rule | loss control |
|---|---|---|
| `TestOriginalMinimapTerrainIsTheLookupTimesBrightness` | lookup sample times factor | the symbolic picture does not draw the sample |
| `TestOriginalMinimapExcludesTheEightCellBorder` | first pixel is cell (8,8), no margin sample | a whole-grid picture would put cell (0,0) there |
| `TestOriginalMinimapFogCases` | OR 0, `0x8000`, bit 14 on the shifted block | symbolic explored and shifted-quad pixels differ; symbolic unseen black is not opaque |
| `TestOriginalMinimapMarkExtentsColoursAndGates` | class extents, bridge size, unit cell, colours, quad and corpse gates | the symbolic picture differs at each mark |
| `TestOriginalMinimapDownscaledMarks` | unscaled extents and one-pixel units below one pixel per cell | the pixel past the third is not structure colour |
| `TestOriginalMinimapWhiteOutlineFollowsScroll` | white edges follow the camera; clipped edges are dropped | the symbolic outline is not white |
| `TestOriginalMinimapLootDotUnchanged` | amber loot dot on the new picture | the symbolic dot is the same colour |
| `TestOriginalMinimapRepaintGate` | 10 counter steps keep the picture, the 11th repaints, a moved view repaints | the fresh composition differs from the kept one |

Release witness `TestReleaseMissionMinimapPaintsTheInstalledPicture` reads
installed MiniMapData.bmp and CrystalR.bmp itself and checks every interior
pixel of mission 10 after 241 frames of walking. EN and RU give the same
census: map 80x80, interior (8,8)-(72,72), picture 128x128 at (24,18); unseen
14,869 of 14,976 pixels, explored 894 of 900, visible 457 of 508; the rest are
120 outline, 44 mark and 0 loot pixels, 0 unexplained. Loss control, the
symbolic picture: explored 0 of 900, visible 6 of 508, 1,353 unexplained.

Hotfix release tests:
`TestReleaseMinimapRetiresAFallenMercenaryDot` now checks the body's dot
against the `MISSION-085` rule (corpse stage at most 1, a visible quad cell)
instead of the Heal floor; `TestReleaseMinimapAlwaysVisibleAfterHistoricalUIAndSAVLoad`
and `TestReleaseOwnerMinimapAlwaysVisibleOnLoadAndF9` take their terrain
palette from MiniMapData samples times factors 1 to 96 instead of the
symbolic tile colours. `TestReleaseShiftF4RevealsMapWithoutQuickSave` checks
its changed pixel against the lookup, the crystal and the fog block.

Second game: a probe composing the minimap 301 times over 309 ticks of the
first second-game mission with the party walking gave one SHA-256 over all
pictures, `c2807efa...`, on the base tree and on this branch, on rom2-en and
rom2-ru.

Install probe over the 28 maps `ArchiveMaps` lists, EN and RU: 335,168
interior cells, none outside the lookup; 951 structures, none with owner 0.

## Open debt

- `DIV-2995`: 8-bit channels and the corner-level reading of the brightness
  bytes; the native pixel format is Unknown.
- `DIV-2996`: the low-nibble shade selector; ownerless objects draw no mark.
- `DIV-2997`: enemy units need their own cell visible; draw order; picture
  clip; opaque black CrystalR pixels.
- `DIV-2998`: this build's forced repaint triggers; no fog cache.
- `DIV-2994` (accepted): fill scale instead of the four steps.
