# 0102 — plan

Two files change and no package moves: `pkg/render/terrain/shadow.go` holds the model, `pkg/ui/shadow.go`
the screen pass. Nothing else in the tree calls either. No new package, no new type, no `formatVersion`
(spec P-1), and no loader change — `StructureClass.ShadowY` is already carried.

## DD-1 — the shear has one pivot, and it is a frame's own height

The pivot is not a per-caster quantity. `ShadowPivotRow(f) = f.Height` is the whole of spec FR-4, and
every submission passes it to spec FR-3's `ShadowRowOffset`. That is what lets the three casters differ only in their `TopLeft.X`, which
in turn is what makes spec FR-7 and FR-9 checkable identities rather than three separate derivations.

`ShadowRowOffset` keeps its `pivotRow` parameter — it is the general shear primitive and the tests need
to pass a pivot the caster would not — but no production call site computes one of its own. Reverting
`ShadowPivotRow` to a constant `0` must fail AC-3 and AC-4; that is the point of naming it.

## DD-2 — the dead band lives in `ShadowAngle`, above `ShadowSlope`

`ShadowSlope` becomes `math.Tan(ShadowAngle(theta))` and `ShadowAngle` carries the clamp and the `2/3`.
Splitting them is what makes spec AC-1's "reverting the clamp fails this" a one-line revert: delete the
clamp from `ShadowAngle` and the sweep goes through zero at minute 360. Folding the clamp into
`ShadowSlope` would leave the `2/3` and the clamp in one expression, where a reviewer cannot tell which
of the two the test is pinning.

The clamp is written as a magnitude test and a signed replacement, not as `math.Copysign` on a clamped
absolute value: `theta` of exactly `0` has no sign to copy and spec D-1's choice would then be a
property of `math.Copysign` rather than a line anyone can find.

## DD-3 — the pixel shift is one function, shared by two casters

`ShadowPivotShift(theta, frameH, anchorY)` is spec FR-5 and both the unit and the object call it. The
object calls it with **frame 0's** height and frame 0's anchor Y, the unit with the drawn frame's — so
spec D-3, if it is ever resolved the other way, is one argument at one call site.

Written as `2*(frameH/2)` and not as `frameH &^ 1`: the doubled floor-halving is the shape the published
term has, and it is what makes odd frame heights take `frameH-1` (AC-2). Go's `/` truncates toward zero,
which differs from a floor for negative `frameH`; a negative frame height is not a frame and P-3 asks
only that the function return, so the truncating form stands and the test asserts totality, not a value.

`floorDiv` and `UnitShadowShift` are **deleted**. Nothing else in the tree calls either; leaving a
`UnitShadowShift` that no longer places anything is the second copy of a rule that goes stale.

## DD-4 — the three placements keep their shapes; only their X term changes

- `UnitShadowPlace(body, theta, airLift) StaticPlacement` — signature unchanged. `+UnitShadowShift`
  becomes `-ShadowPivotShift(theta, body.Frame.Height, body.Anchor.Y)`. A nil `body.Frame` must not
  panic (P-3), so the shift is taken from a zero height in that case rather than dereferenced blind.
- `ObjectShadowPlace(p, theta, originY) (StaticPlacement, bool)` — **loses its `int` return** and
  **gains `theta`** (spec FR-13). The pivot row it used to hand back was the frame-0 anchor's Y; under DD-1 the pivot
  is the drawn frame's height, which the caller already has from the returned placement's own `Frame`,
  so returning it would be a second copy. The frame-0 re-anchor itself is unchanged (spec FR-12).
- `StructureShadowPlace(p, theta) (StructurePlacement, bool)` — signature and guards unchanged, spec
  FR-10's four refusals included. Only `pkg/ui` changes, by submitting a shear for it.

## DD-5 — the mirror composes inside the local matrix, not around it (spec FR-15)

Today's mirror branch is `Scale(-zoom, zoom)` then `Translate(d.X+d.W, d.Y)`, applied *after* the shear
element. That flips the lean, because scaling X by `-1` negates the `(0,1)` element with it. Since a
unit's slope was always `0` this was unreachable; under spec FR-14 it is reached on every unit.

Build the local map in one place instead, before any zoom:

```
op.GeoM.SetElement(0, 1, -slope)          // x -> x - slope*y
if mirror {
    op.GeoM.SetElement(0, 0, -1)          // x -> -x
    op.GeoM.SetElement(0, 2, w)           // ... + w, the body path's own convention
}
op.GeoM.Scale(zoom, zoom)
op.GeoM.Translate(d.X, d.Y)
```

so the composite is `x -> (w - x) - slope*y`, which is the software walk's `w-1-col` read on the source
plus the same lean (spec D-2 owns the one-pixel difference). `Scale` and `Translate` are left as the two
composing calls they already are; only the local part is set element-wise, because ebiten's `GeoM` has
no "pre-multiply a shear" call and `Concat`'s order is one more thing a reader would have to be right
about.

`d.W` is then no longer read by the mirror branch and `screenRect.W` stays only for the plain scale.

## DD-6 — one submission shape for all three casters in `shadowDraws`

All three arms converge on the same five lines: resolve the placement, take `pivotRow` from the drawn
frame, cull on `shadowWiden(rect, slope, pivotRow)` — spec FR-16's one cull, now reached by all three —, compute `originX = float64(rect.Min.X) +
slope*float64(pivotRow)` — the world X of local `(0,0)`, row 0's own offset already in it — and submit.
The object arm already has that shape; the structure and unit arms are brought to it.

`shadowWiden` is unchanged. It widens by `ShadowRowOffset` at rows `0` and `Dy()-1`, which under DD-1's
pivot are `slope*h` and `slope`, and those are the extremes because the offset is monotone in the row.
Its comment about being reached only by the object arm goes.

The `alpha` for structures and units is unchanged: `ShadowLevel` at `ShadowObject` and `ShadowUnit`.

## DD-7 — what proves each of the four laws, and what reverting it breaks

Each law gets one assertion that a one-line revert turns red. This is the story's reason for existing,
so the tests are named by the revert they catch rather than by the function they call.

| Law | Revert | What goes red |
|---|---|---|
| structure shears too (FR-8, FR-9) | submit `Slope: 0` for structures | AC-4: the composed column stops being a line; the join jumps to 18.4 px |
| the unit shift is a pixel term (FR-5, FR-6) | make the shift depend on `theta` alone | AC-2 and AC-3: two frame heights give one shift; the anchor-row residue blows past 2 px |
| the dead band (FR-1, FR-2) | drop the clamp from `ShadowAngle` | AC-1: minute 360 gives slope `0` |
| both signs are right (FR-8 vs FR-6) | negate either caster's term | AC-3 or AC-4 |

AC-6's second half — that no function tests a class field against a constant, which is what makes spec
FR-11's suppression arithmetic rather than a branch — is a source assertion, not a numeric one: the sweep proves the displacement, and the reading of the source proves there is no
branch behind it. Write it as a scan of `shadow.go` for a comparison against `ShadowY`, in the shape
`internal/archtest`'s own source scan already uses, or as a review note in `verification.md` if a scan
over one file is more machinery than the claim is worth. Do not add a sentinel test to make it pass.

## DD-8 — what is out

No change to the recolour, the mask, the blend, the cache, the pass order, `BlitShadow`'s body beyond
its callers' new pivot, or any placement list. No new `cmd/` tool: the strip-frame-height measurement is
already made and recorded in `analysis.md`, and a permanent tool for a one-off census is a fourth copy.
