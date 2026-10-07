# 1075 — shaped fog-of-war frontier

## Player result

A fog frontier no longer darkens whole 32-pixel cells as rectangles. Each
terrain quad now samples fog independently at its top-left, top-right,
bottom-left and bottom-right lattice vertices, so mixed corners reveal a
graded part of the neighbouring cell in both flat and height-displaced modes.

## Authority and behaviour

- `TERR-FOG-082` maps visible, explored and unseen to shroud levels 0, 8 and
  16; `TERR-FOG-084` fixes their gains at 1, one half and 0.
- `TERR-FOG-083` makes the projection per vertex and a mixed-corner cell a
  gradient. `TERR-FOG-037` puts that gradient over the terrain quad.
- `pkg/ui.Viewer.cornerScales` applies the four factors after terrain and spell
  lighting. Existing `drawFlat` and `drawDisplaced` carry those independent
  values through their shared four-vertex `DrawTriangles` path.

The simulation sight walk, fog plane, refresh cadence, minimap, targetability,
drawable gates, input, saves and hashed world form are unchanged. No new
research was required.

## Proof and debt

`TestFogFrontierSubmitsFourIndependentLatticeCorners` records the vertices
submitted by both production terrain loops, derives expected RGB from the
pre-fog terrain shading plus independent 1/0.5/0 factors, and pins the input
fog plane unchanged. The old one-factor implementation fails at the explored
and unseen corners. The ordinary uniform-state test remains for all three
levels.

Final evidence is one focused `pkg/ui` run, `gofmt`, one full Go test, the
no-assets guard and one paired EN/RU release gate on the exact candidate.

`DIV-515` records the remaining pixel-level difference: Ebitengine interpolates
vertex colours over two triangles, whereas ROM1 uses an integer column-span
ramp and a 16-bit LUT. The boundary topology, endpoint gains and partial-cell
reveal are implemented; byte-identical interior rounding is not claimed.
Reserved `DIV-516` through `DIV-522` were unused and return to the seat.

Allocator sweeps immediately before and after the `DIV-515` ledger edit both
reported research `2b5ab1c`, next `DIV-524`, 32 ledgers and zero missing
answers.
