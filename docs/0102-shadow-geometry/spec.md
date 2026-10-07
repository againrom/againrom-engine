# 0102 — shadow geometry

A shadow leans. Every caster's silhouette is sheared about the bottom edge of the rectangle it is
drawn into, and each caster moves that pivot to the row where its own silhouette meets the ground.
The tree currently shears one caster of three, gives every unit the same displacement, and lets the
lean pass through an exact zero twice a day; this story replaces those rules with one.

## Scope

In: the sun's shear angle, the three casters' shadow placements, the shear each submits, and the
cull and the transform that carry it to the screen. Out: the recolour law, the silhouette mask, the
blend, the pass order and the placement lists it reads — all unchanged, and none restated here. Out
also: any toggle or quality setting, any change to simulation state, any second sun.

## P — properties

- **P-1** Nothing here enters the byte form, the digest or `pkg/sim`. A shadow is a parameter of a
  drawing, and this story needs no `formatVersion`.
- **P-2** The angle is the one `Light.Theta` already carries. No second sun is computed.
- **P-3** Every function of the model is pure and total over its inputs — any `float64` angle, any
  frame size, any grid row, negative included — and returns without panicking on all of them.
- **P-4** No shadow rule tests a class field for a sentinel value. A class is suppressed, or not, by
  the arithmetic alone.

## FR — functional requirements

### The lean

- **FR-1** `ShadowAngle(theta)` is the angle the tangent is taken of: `theta` clamped away from zero
  to a magnitude of at least `0.05`, then multiplied by `2/3`. The clamp takes the sign of `theta`,
  and a `theta` of exactly `0` takes the positive one. `ShadowAngle` therefore never returns `0`,
  and its smallest magnitude is `0.05*2/3`.
- **FR-2** `ShadowSlope(theta)` is `tan(ShadowAngle(theta))`. It is the only place `math.Tan` is
  called in the package. At `Theta = DefaultTheta` it is `0.57735025728078282` to within `1e-15`;
  over a full day its magnitude runs from `0.033345685` at the dead band's edge to `0.577350` at the
  band ends, and it is **never zero and never changes sign within either half-day**.
  `ShadowShear16(theta)` is `int(ShadowSlope(theta) * 65536)`, truncated toward zero.
- **FR-3** `ShadowRowOffset(slope, pivotRow, row)` is `int(slope * float64(pivotRow-row))`: the
  horizontal offset of one image row of a sheared silhouette, `0` at `row == pivotRow` and growing
  with height above it. A positive slope puts a row above the pivot toward `+X`.
- **FR-4** `ShadowPivotRow(f)` is the frame's own `Height`, `0` for a nil frame. **Every** caster's
  shear pivots there — at the bottom edge of the rectangle the silhouette is drawn into — and no
  caster passes any other pivot. A caster's own placement is what moves the pivot to the row it
  wants; the shear itself has one rule.

### The unit

- **FR-5** `ShadowPivotShift(theta, frameH, anchorY)` is
  `int(ShadowSlope(theta) * float64(2*(frameH/2) - anchorY))` — a **pixel** count, truncated toward
  zero, and not the fixed-point slope. It is the term that moves the blit's pivot from the frame's
  bottom edge to the anchor row, and it depends on the drawn frame's height and on the anchor, so
  two units drawn at different frame sizes take different shifts at one instant.
- **FR-6** `UnitShadowPlace(body, theta, airLift)` returns the shadow's placement from the body's:
  same cell, same frame, same anchor, same mirror bit, top-left displaced by
  `(-ShadowPivotShift(theta, body.Frame.Height, body.Anchor.Y), +airLift)`. `airLift` is the height
  the body is drawn above the plane its shadow lies in; every caller in this tree passes `0`.
- **FR-7** Composed with FR-3 at FR-4's pivot, a unit's shadow satisfies
  `X(r) = body.TopLeft.X + slope*(anchorY - r)` to within 2 px at every row, for every frame height
  and anchor: the shear vanishes at the **anchor row**, not at the frame's bottom. The residue is
  the two truncations and does not grow with the frame height.

### The structure

- **FR-8** A structure's shadow is the same strip, the same frame and the same `dstY` as its body,
  displaced in X by `StructureShadowShift(theta, fullHeight, gridRow, shadowY)` =
  `int(ShadowSlope(theta) * float64((fullHeight-gridRow)*CellSize - shadowY))`, **and sheared as
  well**, by FR-3 at FR-4's pivot. The two terms compose; neither is an alternative to the other.
- **FR-9** With both terms, a structure's shadow column is a function of world `y` alone: the strip
  index cancels, and the whole footprint shears about the single world row
  `(anchorRow + TileHeight)*CellSize + CellSize - shadowY - lift - originY`. Consecutive strips
  therefore join: the column at a strip's last row and the column one row further down in the next
  strip differ by less than 1 px, which is the difference of two truncations.
- **FR-10** A `VariableSize` class casts no shadow. `StructureShadowPlace` reports `false` for one,
  as it does for a placement with no class, no frame, or a non-positive `TileWidth`.
- **FR-11** No rule reads `ShadowY` as anything but a length. A class whose `ShadowY` is far larger
  than its own art displaces its shadow far outside its own footprint and is suppressed by that
  displacement alone: at every minute of the cycle, `|StructureShadowShift|` for such a class
  exceeds 300 world pixels at every strip, and exceeds 650 where `ShadowY` is 20000.

### The object

- **FR-12** An object's shadow anchors on **frame 0** while its body anchors on the drawn frame, so
  a class whose drawn frame differs in size from frame 0 has its shadow displaced from its body. Its
  silhouette is nevertheless the **drawn** frame's, and so is its shear pivot (FR-4).
- **FR-13** `ObjectShadowPlace(p, theta, originY)` returns the frame-0-anchored placement with
  `TopLeft.X` further displaced by `-ShadowPivotShift(theta, frame0.Height, frame0Anchor.Y)`, and
  `false` for a placement with no class, no drawn frame, or no frame 0.

### The screen

- **FR-14** All three casters submit a non-zero shear. Each entry carries the slope FR-2 returns and
  the pivot FR-4 gives for the frame it draws; no caster submits zero for either.
- **FR-15** A mirrored shadow is reflected in its source columns and **not** in its lean: the mirror
  and the shear compose in the frame's own local space, mirror first, so a mirrored unit's shadow
  leans the same way as an unmirrored one at the same instant.
- **FR-16** Every shadow is culled on the rectangle actually drawn — widened in X by the row offsets
  the shear reaches at its top and bottom rows — through the package's one cull, at the camera's own
  transform and zoom. All three casters are culled that way, not only one.

## D — divergences, disclosed

- **D-1 The sign taken at `theta == 0` is ours.** The clamp's sign at exactly zero is not published;
  we take the positive branch. It moves the lean of one in-game minute per half-day.
- **D-2 The screen's arithmetic is the GPU's.** The transform realises FR-3 as one affine map, so a
  row's offset is the continuous `slope*(pivotRow-row)` where `ShadowRowOffset` truncates it to an
  integer, and a mirrored draw reflects about the frame's width where the software walk reflects
  about `width-1`. `BlitShadow` and every assertion run the exact integer law.
- **D-3 The object's anchor frame is a corpus contradiction we did not resolve.** One published row
  puts the object shadow's anchor at frame 0 and another puts it at the drawn frame; we keep frame
  0. It changes nothing for any class whose drawn frame is frame 0's size, and at most 4 columns and
  8 rows for the few that differ.
- **D-4 The dead band's shape is ours, and the published displacement does not choose it.** Two
  readings fit what is published: the clamp applies to the angle before the `2/3`, or the clamp's
  own `+-0.05` is itself the returned value. We take the first. The published displacement range
  does **not** discriminate them — over a real day both reach the same figure to within 3 px,
  because the angle steps by 0.0022 and so takes values just outside the band twice a day. What
  decides it is that only the first *attains* the published smallest magnitude `0.0333` instead of
  merely approaching it, and that the second is discontinuous, falling as `|theta|` rises through
  the band's edge, which no clamp does. AC-1 pins the choice behaviourally: the second would give
  `0.0500` at minutes 360 and 1080.

## AC — acceptance criteria

- **AC-1** `ShadowSlope(DefaultTheta)` is within `1e-15` of `0.57735025728078282`; over all 1440
  minutes of the cycle `ShadowSlope(SunAngle(m,true))` is never `0`, its magnitude is never below
  `0.0333` and never above `0.5774`, and at minutes 360 and 1080 — where `SunAngle` is exactly `0` —
  it is `+0.033345685` to within `1e-9` (FR-1, FR-2). **Reverting the clamp fails this.**
- **AC-2** `ShadowPivotShift` differs between two frame heights at one `theta`, and between two
  anchors at one frame height; it is `0` only where `2*(frameH/2) == anchorY`; and for odd `frameH`
  it takes the value `frameH-1` gives, not `frameH` (FR-5). **A shift computed from the angle alone
  fails this.**
- **AC-3** Over a sweep of frame heights, anchors and day minutes, `UnitShadowPlace`'s top-left
  composed with FR-3 at FR-4's pivot is within 2 px of `body.TopLeft.X + slope*(anchorY-r)` at every
  row, and the maximum deviation divided by the frame height does not grow with it (FR-6, FR-7).
  **Leaving the pivot at the frame bottom fails this by tens of pixels.**
- **AC-4** For the class `TileWidth 2, TileHeight 2, FullHeight 3, ShadowY 40` at `DefaultTheta`,
  the composed column of every image row of every strip is a single straight line in world `y` to
  within 1 px, and the join between consecutive strips is under 1 px where the per-strip term alone
  leaves 18.4 (FR-8, FR-9). **Submitting zero shear fails this.**
- **AC-5** A `VariableSize` placement, a classless one, a frameless one and one with a non-positive
  `TileWidth` all yield no structure shadow (FR-10).
- **AC-6** At every minute of the cycle and every strip of every shipped `FullHeight`, a class with
  `ShadowY = 20000` displaces by more than 650 world px and one with `ShadowY = 10000` by more than
  300, and no function in the package tests `ShadowY`, `FullHeight` or any other class field against
  a constant (P-4, FR-11).
- **AC-7** An object whose drawn frame differs in size from frame 0 anchors at frame 0 and takes
  frame 0's displacement, while its silhouette and its shear pivot are the drawn frame's; a
  placement with no class, no drawn frame or no frame 0 yields none (FR-12, FR-13).
- **AC-8** The screen pass submits, for each of the three casters, an entry whose slope is
  `ShadowSlope(theta)` and whose pivot is the drawn frame's height; a mirrored entry and an
  unmirrored one at one instant put their top rows on the same side of their bottom rows; and a
  placement whose sheared rectangle leaves the view submits nothing while its unsheared rectangle
  would have been kept (FR-14, FR-15, FR-16).
- **AC-9** Every function named in this contract returns for a nil frame, a nil class, a negative
  frame size, a negative grid row and a `theta` of `1e9` and `-1e9` (P-3); and `againrom -check` and
  `missionrun -mission 10 -census` print on both roots what they printed before this story (P-1,
  P-2).
