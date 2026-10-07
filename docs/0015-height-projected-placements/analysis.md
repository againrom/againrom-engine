# Analysis — height-projected placements

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-first / static** — the profile's default |
| Terrain — `pkg/ui.Viewer.Mode()` and the PNG marker offset | **brownfield** |
| Terrain — the cell height lookup itself | **greenfield** |

Brownfield for `Mode()` because 0013 FR-2 froze "an overlay forces flat" on purpose and this story
lifts it on purpose; brownfield for `cmd/terraintool`'s marker offset because 0012's own code
already names this exact gap as work it defers. Greenfield for the lookup, which is new.

## What the baseline entry got wrong

`cleandocs/0015-height-projected-placements/spec.md` was written against a tree this repo is not.
None of its named identifiers exist here (`cmd/almview`, `cmd/openrom`, `render.TerrainProjection`,
`pkg/game.MapScene`, `StructureRects`/`UnitRects`, `SampleHeight8p8`, `cellpx` as a continuous
ratio, `nativeMinY`), and several of its behavioural claims do not survive contact either.

1. Its "current behavior" is only half right, mechanized differently. The viewer
   (`pkg/ui.Viewer`, shared by `cmd/mapview` and the game front-end) does not *reject*
   displaced+overlay - `Viewer.Mode()` silently *selects flat* whenever either overlay is on
   (0013 FR-2), no error, nothing printed. `cmd/terraintool` has no refusal at all: `-objects`/
   `-units` already compose with the default projected geometry today. What it gets wrong is the
   *height*, not the combination - `DrawObjectMarkersAt`'s own doc names this story as the fix
   ("a projected image, a marker does not register with the terrain beneath it... Placing a sprite
   on the displaced surface is work item 0015's contract").
2. No code named `render.TerrainProjection`, `pkg/game.MapScene`, `StructureRects`/`UnitRects`
   exists. The mesh is `terrain.Projection` (`pkg/render/terrain/project.go`); the interactive
   scene is `pkg/ui.Viewer`, built once by `pkg/game.LoadMapViewer` for both `cmd/mapview` and the
   game front-end; the marker geometry is `terrain.ObjectMarkerRects`/`terrain.UnitMarkerRects`.
   There is no `cmd/almview`/`cmd/openrom` - the PNG tool is `cmd/terraintool`, the game is
   `cmd/againrom`.
3. The structure-vs-unit anchor split is false here. Both 0008 (objects, ALM type-4) and 0009
   (units, ALM type-6) already collapse every anchor to `terrain.AnchorCell(x,y) = (x>>8, y>>8)`
   *before* any marker rectangle is built. Neither kind keeps its sub-cell fraction; both are
   cell-snapped, both centred on the same point. There is no "structure" concept in code at all -
   0008's directory is named `0008-structures-overlay` but its content, spec and identifiers are
   all "objects" (the ALM format's own dual name for its type-4 section); no footprint rectangle
   exists either (0008 draws a fixed two-rectangle cross, nothing that could be "warped").
4. The external-vertex convention it assumes is the opposite of ours. It states `heightAt=0` on
   a right/bottom external vertex. `terrain.Projection.Altitude` instead **clamps** an out-of-grid
   index to the nearest valid vertex - 0012's own disclosed choice ("We clamp the index into the
   grid instead: the safe, intent-matching treatment"), never a read of zero.
5. `cellpx` is not the continuous ratio it assumes. `cmd/terraintool -scale` is a positive
   *integer* multiplier of the native 32-px cell, so a native offset scales to output pixels by an
   exact multiply; `pkg/ui` builds marker geometry at native scale and lets the camera's own
   (already-adopted) transform apply zoom. Neither site ever needs the `roundDiv(dy*cellpx,32)`
   rounding the baseline built for a fractional `cellpx`.
6. It claims "no reference source... independent rewrite," but research has since decoded the
   engine's own convention: `TERR-GEOM-031` (amended) and `TERR-SPR-039` read, from the
   disassembly, that ROM1 lifts a placed object by the **mean of its anchor cell's four corner
   heights**, truncated toward zero - not a bilinear sample at an arbitrary point. Since both our
   overlays already anchor at one cell, this decoded fact is exactly what this story needs, rather
   than an unclaimed, "ours by choice" invention. (2026-07-29: half of that fell — `TERR-SPR-041`'s
   identification is retracted, and the engine's unit draw ignores the cell mean, placing from the
   unit's own raw position, `TERR-SPR-048`. For unit markers the lift IS ours by choice — see
   provenance.)

## What we looked at

- `pkg/ui/viewer.go` (`Mode`, `syncWorld`, `NewViewer`) and `pkg/ui/overlay.go`
  (`SetObjects`/`SetUnits`, `overlayScreenRects`) - `Mode()`'s guard is
  `v.proj != nil && !v.showObjects && !v.showUnits`; enabling either overlay re-syncs the camera to
  the flat world height.
- `pkg/render/terrain/overlay.go` - `AnchorCell`, `ObjectMarkerRects`/`UnitMarkerRects`,
  `markerRects`, `DrawObjectMarkersAt`/`DrawUnitMarkersAt` and their `offsetY` (today one constant
  per whole image, never per cell).
- `pkg/render/terrain/project.go` - `Projection.Altitude`/`Vertex`/`WorldCorner`/`MinV`/`MaxV`; the
  far-edge clamp is read off `Altitude` itself, not reimplemented.
- `cmd/terraintool/main.go` (`anchorCells`, `unitAnchorCells`, `markerOffsetY = -render.OriginY *
  scale`) and `cmd/mapview/main.go`/`pkg/game/mapload.go`/`frontend.go` - one loader
  (`LoadMapViewer`) serves both the standalone viewer and the game; neither carries a "MapScene"
  type.
- `research/claims/terrain.md`: `TERR-GEOM-031` (amended), `TERR-SPR-039` (amended),
  `TERR-EDGE-024`..`026` - the engine's own far-edge addressing and its two separate altitude
  models (terrain mesh vs. sprite lift).
- `research/claims/alm.md`: `ALM-OBJ-019`, `ALM-UNIT-018`, both amended by EXP-0030's frame
  correction - the 8.8 encoding stands; the correction is a pure offset relabeling.
- `research/claims/retracted.md` - read first. What it takes back from `TERR-SPR-039` is the
  universal "the two altitude models disagree by construction on any sloped cell": they agree on
  12,513 of 135,129 sloped cells. The tie-break below is a different, Medium-graded half of the
  same row.

## What we did not look at, and what stays unknown

- The engine's exact tie-break for a negative corner-sum mean (`round4` vs. a bare truncating
  divide). `TERR-SPR-039` finds the shipped corpus cannot discriminate it - no height byte reaches
  `0x80` in any of the 38 maps. Not depended on: Go's plain truncating division reproduces the
  decoded instruction sequence for every value the ambiguity could affect, and the gap is
  disclosed rather than resolved.
- Whether the engine's two altitude models (terrain mesh, sprite mean) ever agree widely.
  `TERR-SPR-039`: they agree on only 9.26% of sloped cells, diverging up to 73 rows on the rest.
  Not reconciled here; this story places a diagnostic marker at its own anchor cell's height,
  nothing more.
- Real object/unit art, footprints and per-corner projection - still downstream of the class
  registries and static-data formats, unaffected by this story.
