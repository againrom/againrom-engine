# 0101 — shadows: plan

## DD-1 — the model lives in `pkg/render/terrain`, in one new file

`shadow.go`, beside `sun.go` and `light.go`. Not `pkg/sim`: the angle is a `float64` and that
package is held to no floating-point identifier by a scan over its own syntax, and a shadow is a
parameter of a drawing rather than simulation state — `0092` DD-1 settled this for the sun and
nothing here is different. Not a package of its own: every input it takes (`Light`,
`StaticPlacement`, `StructurePlacement`, `StaticFrame`) is this package's, and a shadow package
would import all of it to add three functions.

## DD-2 — the shear is one function, and the sense is one sign in it

`ShadowSlope(theta)` is FR-1, and it is the only place `tan` is called and the only place the
`2/3` appears. FR-2's two numbers are assertions about it and about nothing else. FR-3's row
offset, FR-4's unit shift and FR-7's structure shift all take the slope rather than the angle, so
D-1's sense is settled once and a later correction is one minus sign in one expression. Writing
the `2/3` at three call sites would make the divergence three edits and the check in AC-1 a
tautology about whichever one the test happened to call.

The fixed-point form `ShadowShear16` exists because FR-4's `/2000` is defined on it: the engine
divides the 16.16 integer, not the ratio, so `floor(int(tan*65536)/2000)` and `floor(tan*32.768)`
part company at the boundaries. Truncation toward zero for the `ftol`, floor for the division —
`__ftol` chops and the magic-multiply divide has no sign correction.

## DD-3 — the shadow placement is derived from the body placement, never rebuilt

`UnitShadowPlace` takes the body's `StaticPlacement` and returns a `StaticPlacement`, and FR-6's
`airLift` rides it as an argument rather than as a field nobody fills. It does not call
`StaticAnchor` a second time. FR-5 is then true **by construction**: the two placements cannot
differ in cell, frame, anchor or mirror because one is a copy of the other with two integers
added, and the test that asserts it is asserting a property of the tree rather than of two
independent copies of the anchor arithmetic. This is the same reason `0100`'s `SunAt` composes
rather than re-spells.

The object is the one caster where this does not hold, because FR-10's whole content is that the
anchor is built at a *different frame*. `ObjectShadowPlace` therefore does call `StaticAnchor`, at
frame 0's size, and the displacement AC-7 asserts is the difference between the two calls. It
hands back FR-11's pivot row from the same call, because the pivot IS that anchor's Y and a second
expression for it could drift from the placement it belongs to.

## DD-4 — the level reaches the model as an `int`, not as a `Light`

FR-13's law is one function of a channel and a level, and FR-15's `BlitShadow` is the only thing
that walks pixels with it, so `ShadowChannel`/`ShadowRGBA`/`ShadowAlpha` take `L` and not a
`Light`. The choice of *which* of a `Light`'s two indices a caster reads is one exported function,
`ShadowLevel(lt, kind)`, and nothing else in the package looks at `ShroudUnit` or `ShroudObject`.
So FR-14's pairing — structures with objects, not with units — exists in one place, which is where
it will be wrong if `TERR-STRUCT-103`'s global is ever re-read.

## DD-5 — `ShadowY` is plumbed, not re-derived

FR-8's `ShadowY` needs no decoding: `pkg/data` already has the key. `pkg/game`'s structure loader
copies it onto `terrain.StructureClass` beside the three flags it already copies. The render tier
does not learn the registry's spelling and nothing recomputes a default: an absent key resolves to
`0`, which FR-7 reads as a shear pivot at the structure's own base, and that is a legal answer
rather than a sentinel. FR-9's exclusion needs nothing plumbed at all — `VariableSize` is already
on the class.

## DD-6 — the screen path draws a mask, and the level rides the draw

Rejected: baking the level into the mask. The texture would then be keyed on `(frame, level)` and
every relight — one per twenty in-game minutes, one per twenty **real** seconds at the shipped
speed — would build a new set. Rejected too: baking the shear into the mask, which adds the angle
to the key and makes the set rebuild continuously.

Taken: the mask is the silhouette alone (FR-16), keyed on the frame pointer exactly as the sprite
texture cache is; the level rides the draw as a colour scale on alpha (FR-17) and the shear rides
it as the transform's `(0,1)` element (FR-18). One texture per frame for the life of a run, and
both the hour and the band are free.

The blend is written out rather than taken from a named constant: `BlendDestinationOut` has the
same colour factors but also multiplies the destination's **alpha** by one minus the source's,
which would eat the target's alpha on an off-screen target. Source factors zero, destination
colour factor one-minus-source-alpha, destination alpha factor one.

## DD-7 — one pass, reading the lists the content plane already reads

FR-19 says one pass and says where: `drawShadows` sits between `drawStructuresFlat` and
`drawPlane` inside `drawArt`, which is where the order between passes already lives and the only
place it is observable. It walks `staticPlacements()`, `structurePlacements()` and `entityLayer()`
— the same accessors `planeSprites` walks, memoised there already — so no fourth list exists and a
placement cannot be in the shadow pass but not in the content plane.

It does **not** walk `depthOrder`. Shadows all lie on the ground and none occludes another;
merging them into the depth order would buy an arrangement that no pixel can distinguish and would
put the shadow of a near unit above the body of a far one, which is D-3 in the other direction and
worse.

## DD-8 — `shadowDraws` returns a list, `drawShadows` paints it

The split is `planeSprites`/`drawPlane`'s, for its reason: an `*ebiten.Image`'s pixels cannot be
read back before the game starts, so a pass stated only against a concrete target is asserted by
nothing. The arrangement, the cull, the alpha and the transform are decided in a function
returning plain Go values, and AC-8 reads them. The recording target `imageTarget` already exists
and already carries the options, so the blend and the colour scale are observable without a new
seam.

## DD-9 — the cull rectangle is the sheared one

A sheared silhouette leaves its frame's rectangle. The screen pass widens the rectangle by the row
offsets at its top and bottom rows before the cull, so FR-20 holds on the pixels actually drawn.
It widens there and not in the model, because the model states the shear as a per-row offset and
only a consumer that has decided to draw the whole frame at once needs a bound over all of them.
The unit and the structure shift by whole integers and their rectangles are their frames'
translated, which is the existing cull unchanged.

## DD-10 — what is not built

No settings surface (FR-12), no byte-form version, no claim, no artifact. `pkg/sim` is not touched
at all. The two shroud fields `0100` left are read here and nowhere else.
