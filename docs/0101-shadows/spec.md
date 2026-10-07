# 0101 — shadows

Units, structures and map objects cast shadows. A shadow is a **recolouring of the destination under
a silhouette**, leaning with the sun the day/night cycle already moves.

## Scope

In: the shadow geometry for the three casters, the recolour law, and the screen pass that draws
them. Out: any toggle, option or quality setting; any change to simulation state; any second sun.

## P — properties

- **P-1** Nothing in this story enters the byte form, the digest or `pkg/sim`. A shadow is a
  parameter of a drawing.
- **P-2** The angle is the one `Light.Theta` already carries. No second angle is computed anywhere.
- **P-3** Every function of the model is pure and total over its inputs: any `float64` angle, any
  level, any frame size, negative included.
- **P-4** The screen pass paints only where a silhouette pixel is opaque. Everywhere else the
  destination is unchanged, pixel for pixel.

## FR — functional requirements

### The lean

- **FR-1** `ShadowSlope(theta)` is `tan(theta * 2 / 3)` — the 16.16 per-row X slope every shadow
  pass derives from the sun, expressed as a `float64` ratio. `ShadowShear16(theta)` is that slope in
  the engine's own fixed point, `int(ShadowSlope(theta) * 65536)`, truncated toward zero.
- **FR-2** At `Theta = DefaultTheta` (the cycle-off sun) `ShadowSlope` is `0.57735025728078282` to
  within `1e-15`, and over the daylight band it runs from `-0.577350` at the band's first minute to
  `+0.575413` at its last, passing through exactly `0` at the band's midpoint.
- **FR-3** `ShadowRowOffset(slope, pivotRow, row)` is `int(slope * float64(pivotRow-row))`: the
  horizontal offset of one image row of a sheared silhouette. It is `0` at `row == pivotRow` — the
  row at which the silhouette meets the ground — and grows with height above it.

### The unit

- **FR-4** A unit's shadow is **translated, not sheared**: the whole silhouette moves by
  `UnitShadowShift(theta)` pixels in X, which is `ShadowShear16(theta)` divided by `2000` rounded
  toward minus infinity. It spans `-19 .. +18` pixels over a day.
- **FR-5** `UnitShadowPlace(body, theta, airLift)` returns the shadow's placement from the body's:
  the same cell, the same frame, the same anchor, the same mirror bit, and a top-left displaced by
  `(+UnitShadowShift(theta), +airLift)`. So on one unit at one frame the two destinations differ in
  exactly those two terms and in nothing else — which is `TERR-SPR-067`'s `(sunShear, -unit+0x10)`
  read shadow-from-body and with D-1's sense, not a third rule.
- **FR-6** `airLift` is the height the body is drawn above the plane its shadow lies in. Every
  caller in this tree passes `0`; the parameter exists because the two destinations differ by it.

### The structure

- **FR-7** A structure's shadow is the same strip, the same frame and the same `dstY` as its body,
  displaced in X by `StructureShadowShift(theta, fullHeight, gridRow, shadowY)` =
  `int(slope * float64((fullHeight-gridRow)*CellSize - shadowY))`, where `gridRow` is the strip's
  row in the class grid counting from the top and `shadowY` the class's own `ShadowY` key.
- **FR-8** `ShadowY` is the pixel height at which the shear is zero — where the structure meets the
  ground. `terrain.StructureClass` carries it and the loader fills it from the registry.
- **FR-9** A `VariableSize` class casts **no** shadow. `StructureShadowPlace` returns `false` for
  one, as it does for a placement with no class or no frame.

### The object

- **FR-10** An object's shadow **anchors on frame 0** while its body anchors on the drawn frame, so
  a class whose drawn frame differs in size from frame 0 has its shadow displaced from its body by
  `((frameW-frame0W)/2, (frameH-frame0H)/2)`. That is the shipped behaviour and is not corrected.
- **FR-11** An object's shadow is **sheared**, per image row, by FR-3, with `pivotRow` the frame-0
  anchor's Y. `ObjectShadowPlace(p)` returns the frame-0-anchored placement and the pivot row.
- **FR-12** An object's shadow is drawn unconditionally. The engine gates it on a detail flag; this
  tree has no settings surface to move one.

### The recolour

- **FR-13** The shadow does not draw the sprite. It reads the **destination** pixel and writes it
  back darkened: `out = (in * (16 - L)) >> 4` per channel, exactly, `L` in `[0,16]`.
  `ShadowChannel(in, L)` and `ShadowRGBA(c, L)` are that law; alpha is carried unchanged.
- **FR-14** `L` is the band's own index: `Light.ShroudUnit` for a unit — 2 by day, 3 in either
  twilight, 4 at night — and `Light.ShroudObject` for a structure and for an object — 4, 6, 8. Both
  come from the schedule `SunAt` already returns; neither is recomputed.
- **FR-15** `BlitShadow(dst, f, destX, destY, level, slope, pivotRow, mirror)` applies FR-13 to an
  RGBA destination under the frame's opaque pixels, row by row at FR-3's offsets, and is the
  reference statement of the law. It reads no source colour: two different backgrounds under one
  shadow give two different results, and one background under two different sprites of the same
  silhouette gives one.

### The screen

- **FR-16** `ShadowMask(f)` is the frame's silhouette as an image: opaque black where the frame's
  pixel is opaque, fully transparent everywhere else. It depends on the frame alone — not on the
  level, not on the angle — so one exists per frame for the life of a run.
- **FR-17** The screen path draws that mask with source alpha `ShadowAlpha(L)` = `round(255*L/16)`
  under a blend whose source factor is zero and whose destination colour factor is one minus the
  source alpha. The result is `dst * (1 - L/16)`, which is FR-13's law in the blend equation; a
  transparent mask pixel leaves the destination untouched. Destination alpha is preserved.
- **FR-18** The shear of an object's shadow reaches the screen as the draw's own transform, not as a
  second texture: the matrix carries `-slope` in its `(0,1)` element and the pivot's constant in its
  translation, so the mask cache holds one entry per frame at every hour.
- **FR-19** The shadow pass runs **once**, between the flat-structure pass and the content plane, so
  every shadow lies on the ground and under every sprite. It reads the three placement lists the
  content plane already reads and builds no fourth.
- **FR-20** A shadow is culled on the rectangle actually drawn — the sheared one for an object —
  through the package's one cull, at the camera's own transform and zoom.

## D — divergences, disclosed

- **D-1 The sense of the lean is ours.** Three published passes carry a horizontal shear term and do
  not agree in sign as written. This story takes the sense of `TERR-STRUCT-103`, the only one
  published as a complete expression with its zero point: a silhouette's head leans toward `+X` at a
  positive angle, its ground contact does not move. All three casters lean together; if the corpus
  later settles the sense the other way it is one sign in `ShadowSlope`.
- **D-2 The shear angle's routine was not read.** `theta * 2 / 3` reproduces both values
  `TERR-LIGHT-113` publishes — the cycle-off tangent to a unit in the last place and the day band's
  range to four decimals — and no instruction of the routine is quoted anywhere.
- **D-3 Shadows do not fall on sprites.** The engine interleaves an object's shadow with its own
  cell's body; ours is one pass under all art, so a shadow never darkens a sprite. Same law, one
  ordering.
- **D-4 The screen's arithmetic is the GPU's, in both axes.** FR-17 realises FR-13 in floating
  point, so a channel may differ from the exact integer law by one unit in 255; and FR-18 realises
  FR-3 as one affine transform, so a row's offset is the continuous `slope*(pivotRow-row)` where
  `ShadowRowOffset` truncates it to an integer. Both are the same admission — an exact integer law
  and a shader that cannot express it. `BlitShadow` and every assertion run the exact one.

## AC — acceptance criteria

- **AC-1** `ShadowSlope(DefaultTheta)` is within `1e-15` of `0.57735025728078282` (FR-2).
- **AC-2** Sampled across a day at the cycle's own minutes, `UnitShadowShift` is negative before the
  daylight band's midpoint, exactly `0` at it, positive after, and never leaves `-19 .. +18` (FR-2,
  FR-4). **A fixed lean fails only this criterion**, which is why it is stated on the sweep.
- **AC-3** For one unit, one frame and one `airLift != 0`, `UnitShadowPlace`'s top-left minus
  `UnitPlace`'s is exactly `(UnitShadowShift(theta), airLift)`, and the two placements agree in
  cell, frame, anchor and mirror (FR-5, FR-6).
- **AC-4** `BlitShadow` of one frame at one level over two destinations differing in one pixel gives
  two results differing in that pixel; over one destination, two frames with the same opaque mask
  and different colours give the identical result (FR-13, FR-15).
- **AC-5** `ShadowChannel(in, L)` equals `(in*(16-L))>>4` for all 256 inputs at every `L` in
  `[0,16]`, and `dst*(1-ShadowAlpha(L)/255)` rounds to within 1 of it (FR-13, FR-17, D-4).
- **AC-6** A structure strip's shadow shift matches FR-7 term for term at a non-zero `ShadowY`, is
  zero at the strip whose height equals `ShadowY`, and a `VariableSize` placement yields no shadow
  (FR-7, FR-9).
- **AC-7** An object whose drawn frame differs in size from frame 0 has its shadow's top-left
  displaced from its body's by FR-10's amount, and its pivot row is the frame-0 anchor's (FR-10,
  FR-11).
- **AC-8** The shadow pass submits one draw per drawable placement, in the order FR-19 states,
  before the first content-plane draw, each with the FR-17 blend and the FR-14 alpha for its kind;
  a `VariableSize` structure and a frameless placement submit none.
- **AC-9** `againrom -check` and `missionrun -mission 10 -census` print, on both roots, what they
  printed before this story: `lost at tick 272`, 2 movers.
