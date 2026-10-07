# 0101 — shadows: tasks

## T1 the shadow model

Add `pkg/render/terrain/shadow.go` with `ShadowSlope`, `ShadowShear16`, `ShadowRowOffset`,
`UnitShadowShift`, `UnitShadowPlace`, `StructureShadowShift`, `StructureShadowPlace`,
`ObjectShadowPlace`, `ShadowKind` with its two values, `ShadowLevel`, `ShadowChannel`, `ShadowRGBA`,
`ShadowAlpha`, `ShadowMask` and `BlitShadow` (FR-1..FR-16, DD-1..DD-4). `math.Tan` is called once in
the file; `2/3` and the sign D-1 settles appear once, both inside `ShadowSlope`.

`ObjectShadowPlace(p StaticPlacement, originY int) (StaticPlacement, int, bool)` returns the
frame-0-anchored placement, its pivot row and whether one was made; it needs `p.Class.Frames[0]` and
`p.Cell`, and takes the lift back out of `p` rather than being handed one — derive it from `p`'s own
`TopLeft` and its drawn-frame anchor so one placement is enough to build the other.

Add `ShadowY int` to `terrain.StructureClass` (`structures.go`) and fill it in `pkg/game`'s
structure loader from `pkg/data`'s already-decoded `ShadowY` (FR-8, DD-5).

Tests in `shadow_test.go`: AC-1; AC-2 sampled at every hour of a day through `SunAt`; AC-3 at a
non-zero `airLift`; AC-4; AC-5 over all 256 inputs and all 17 levels; AC-6; AC-7. `ShadowMask` is
opaque exactly where the frame is and transparent elsewhere.

Touch nothing under `pkg/sim` or `pkg/ui`. Add no byte-form version.

## T2 the shadows reach the screen

Add `pkg/ui/shadow.go` with `shadowDraws()` and `drawShadows(target imageTarget)` (FR-16..FR-20,
DD-6..DD-9), and call the second from `drawArt` between `drawStructuresFlat` and `drawPlane`.

`shadowDraws` walks `staticPlacements()`, `structurePlacements()` and `entityLayer()`'s sprite list
and returns one entry per drawable shadow carrying the mask's frame, the culled screen rectangle,
the alpha, the slope, the pivot row and the mirror bit. It skips a `VariableSize` structure, a
frameless placement and a culled one, and it honours `showStaticArt`/`showStructureArt` exactly as
`planeSprites` does. Entities are never gated.

The mask texture is cached on the frame pointer in a map of its own, built through
`terrain.ShadowMask`, nil until the first shadow is drawn (the `staticImage` precedent). The level
comes from `terrain.ShadowLevel(v.Sun(), kind)`; the slope from `terrain.ShadowSlope(v.Sun().Theta)`.

Tests in `shadow_test.go`: AC-8 against a recording target — the pass's draws come
before the first content-plane draw, one per drawable placement, each with FR-17's blend and FR-14's
alpha; a unit's draw carries no shear and an object's does; a `VariableSize` structure and a
frameless placement submit none; the mask cache holds one entry per frame across two hours.

Touch nothing under `pkg/sim` and no file under `pkg/render`.

## Traceability

| Task | Requirements | Decisions |
|---|---|---|
| T1 | FR-1, FR-2, FR-3, FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, FR-10, FR-11, FR-13, FR-14, FR-15, FR-16 | DD-1, DD-2, DD-3, DD-4, DD-5, DD-10 |
| T2 | FR-12, FR-14, FR-16, FR-17, FR-18, FR-19, FR-20 | DD-6, DD-7, DD-8, DD-9, DD-10 |
