# Analysis — corner-interpolated lighting in the window

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-first / static** — the profile's default for rendering |
| Terrain — the viewer's two draw paths | **brownfield** |
| Terrain — the vertex-colour shading itself | **greenfield** |

No step-up. Unlike 0013, nothing here becomes a frame a later story measures against: the change is
a per-pixel brightness, and every geometry, extent and culling contract 0012/0013 fixed is left
untouched. Brownfield for the draw paths because 0013's FR-1 froze flat mode explicitly and this
story changes it on purpose; greenfield for the shade, which is new code driven red-first.

## What the baseline entry got wrong

The item was imported from a baseline written before this project had game evidence for terrain
lighting, and before 0007 and 0012 shipped. Five of its statements do not survive contact.

1. **Both of its research items are closed, and closed from the game.** It flags as unproven (a)
   that the original interpolates a per-vertex brightness across a tile's interior and (b) the
   per-vertex light value formula. `TERR-LIGHT-011` reads the LUT index arithmetic of two
   independent blitters and finds the level **bilinearly interpolated from four per-vertex corner
   brightness bytes** — so smooth shading is fidelity here, not a taste we exercised. The value
   formula is `TERR-LIGHT-013` as amended by `TERR-LIGHT-028`/`-030`, which is what 0007 re-landed
   on.
2. **The earlier basis for both is deleted, not carried forward in any form.** The baseline attributes
   its interpolation hypothesis and its light-value formula to two reimplementations of a related
   title. Golden rule 4 forbids that source, so nothing it says appears here in any form — not as a
   hypothesis, not as a labelled guess, not as a fallback. Removed with it: the per-vertex value
   `normal·sun·64+96`, the neutral value 144, the `LightScale(b) = b/144` mapping, the `0 → 1.0`
   border convention, and the claim that the two references disagree about the model. What replaces
   them is measured: `provenance.md` names the claim per contract line.
3. **Most of what it asks for already shipped.** Its requirements for the flat CPU compositor and
   for the displaced CPU raster describe `CompositeLit` (0007) and the span-divided ramp in
   `Project` (0012), both landed and both already bilinear. What is genuinely undone is the one
   requirement it wrote for the interactive path — the window applies no lighting at all.
4. **The code it names does not exist here.** There is no `almview`, no `MapScene`, no
   `render.TerrainLight` and no `LightScale`. The compositors are `terrain.CompositeLit` and
   `terrain.Project`, the level grid is `terrain.LevelGrid`, the transform is `terrain.ShadeRGBA`
   over `terrain.InterpSpan`, and the window is `ui.Viewer`.
5. **Its interactive plan predates the two window modes.** It describes one interactive path. Since
   0013 the window renders flat **or** displaced, and either diagnostic overlay forces flat — so a
   lit window has to light both paths or the overlays would turn the light off with them.

## What we looked at

- `terrain.ShadeChannel`/`ShadeRGBA` — the transform is `(channel + tint) * (96 - level) / 32`,
  per channel, integer, truncating. It is **not** a multiplier in `[0,1]`: level 64 is ×1.0, level 46
  (flat daytime ground) is ×1.5625, level 0 is ×3.0. Anything carrying this shade to a GPU has to
  brighten, not just darken.
- `terrain.InterpSpan` — one place computes a level from four corner levels, and 0012 already gave
  it separate denominators per axis. The window needs the corner levels themselves, which is the
  unexported `levelAt` read that both compositors make.
- `terrain.DefaultDaytime` — `Theta` 0.78539815, ambient `0x0e`, range `0x20`, sky tint `(0,0,0)`.
  The tint being zero is what lets a pure multiply express the whole transform.
- `Viewer.drawFlat` submits one `DrawImage` per cell and `drawDisplaced` one `DrawTriangles`. A
  vertex colour reaches the second directly; the first has no per-corner colour to carry at all.
- `game.LoadMapViewer` is the single construction point, and it already hands the altitudes to the
  viewer (0013 DD-4). Nothing new has to be plumbed to reach the height field.
- Ebitengine's own behaviour on this machine, measured rather than assumed (a scratch probe under
  the untracked `builds/`): a vertex colour of 1.5625 over a channel of 100 yields **156**, 3.0
  saturates at 255, and a 1.0→2.0 ramp across 8 px interpolates monotonically. So the GPU can carry
  the shade, and at the flat daytime level it lands on the same byte the CPU transform does.

## What we did not look at, and what stays unknown

- **The exact rounding of the engine's level ramp.** `TERR-LIGHT-011`'s accumulator is seeded
  `+0x100`, half a LUT row, but the claim does not give the scale that accumulator runs at, so the
  rounding is not statable from the pin. 0012 recorded this as open and did not guess; this story
  does not resolve it either, and gains a second deviation of its own — a GPU ramp is continuous
  where the engine's indexes one of 96 discrete rows.
- **Whether two-triangle Gouraud and bilinear agree on a pixel.** They do not in general: bilinear
  is not affine, so splitting the quad makes the interior depend on the diagonal. The disagreement
  is bounded and disclosed rather than measured.
- **What the outer ring should show.** The engine never computes it (`TERR-EDGE-025`); 0007's
  clamped read is our defined choice, and this story inherits it unchanged rather than reopening it.
- **Whether shipped play runs with the day/night cycle off.** Open in research. The window uses the
  cycle-off daytime sun because that is the one configuration we can name a literal for.
