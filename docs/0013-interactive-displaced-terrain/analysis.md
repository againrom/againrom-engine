# Analysis — interactive displaced terrain

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** |
| Terrain — the camera world model, the viewer draw loop, the single load path | **brownfield** |
| Terrain — the displaced draw path and its culling | **greenfield** |

The profile's default for rendering work is `spec-first / static`; this is a recorded step-up. The
camera's world extent stops being derivable from the tile count, and that extent is the frame every
later story drawing in world coordinates measures against (0014's corner lighting, 0015's
placements). It lives in a package both entry points reach through one constructor, and 0005
asserts shipped invariants over it. `static` because no watcher tool exists in this repo.

Both terrains coexist deliberately: the camera and the draw loop have behaviour to preserve and are
pinned before they change; the projected quad path and the padded range are new code, driven
red-first.

## What the baseline entry got wrong

The item was imported from a baseline written against a differently-named tree, and written before
this project had any game evidence for terrain geometry. Four of its statements do not survive
contact, and each fails differently.

1. **Its one research item is already closed, and closed from the game.** The entry flags the
   displacement direction and scale as unresolved, recording that two consulted reimplementations
   of a related title supplied the direction alone. Golden rule 4 forbids that source outright, so
   the question was never how confident it is — it was whether we have it from the game. We do:
   `TERR-GEOM-031` names one instruction each for the `r*32` term, for the subtraction, and for the
   signed read; `TERR-GEOM-035` rules out any altitude-derived horizontal term. The earlier basis
   is not carried forward in any form, and story 0012 already builds on both.
2. **The code it names does not exist here.** There is no `MapScene` and no `cmd/openrom`. The
   interactive viewer is `pkg/ui.Viewer`, and `cmd/mapview` and `cmd/againrom` both reach it
   through one constructor.
3. **The viewer applies no lighting at all.** The entry requires the per-cell light value to be
   applied in displaced mode "exactly as in flat mode", and lists that lighting among the behaviour
   that must not drift. Flat mode applies none — 0007's shading reaches the PNG compositor only. So
   there is no lighting behaviour here for this story to preserve, and none for it to add.
4. **The overlay coupling reaches further than the entry thinks.** It reasons that the game
   front-end requests both diagnostic overlays and therefore stays flat until placements can follow
   the terrain. Ours requests neither. The same rule therefore leaves the game itself displaced,
   which makes this story the one that puts relief on screen rather than in a PNG.

## What we looked at

- `pkg/render/camera`. The world is `Cols x Rows` tiles and both extents are that count times
  `CellSize`, so a world height that is not a multiple of 32 is currently inexpressible. It is the
  normal case for a displaced map: 0012 recorded canvas heights of 2571 and 2648 native rows for
  80x80 maps whose flat height is 2560.
- `VisibleTiles()`, which returns *exactly* the tiles intersecting the view, clipped to the world —
  an exactness 0005 pins. A displaced quad can leave its nominal row band, so the range has to
  widen without weakening what 0005 asserts for a flat map.
- `terrain.Grid` carries tile words only. The altitudes exist at the one place a viewer is
  constructed, on the decoded map.
- `terrain.Projection`. `Project`, `Vertex`, `MinV` and `CanvasHeight` are already exported and
  already carry the whole geometry, so nothing about height is written twice.

## What we did not look at, and what stays unknown

- Whether GPU sampling across a quad and the CPU per-column resample agree on a given pixel. They
  are not claimed equal; 0012 already disclaims pixel-exactness against the original, and the
  shared contract between the two paths is the vertex projection rather than the raster.
- `TERR-GEOM-031`'s last link — that a smaller destination row is higher on the screen — is the
  Win32 device-coordinate convention rather than a ROM1 instruction, and research names it as such.
  It is recorded, not resolved here.
- The engine's own scroll clamp and over-scan margin (`TERR-EDGE-026`, open). Our camera is our own
  design and does not reproduce one.
