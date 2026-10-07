# Plan — one ramp per sprite, one row per frame

## Baseline

`BlitStatic` is the **only** function in the tree that writes sprite pixels, and `(*StaticFrame).RGBA`
is that same blit onto a transparent canvas. The raster tool's static loop calls the first; the
window's static pass and its unit-sprite pass both call `staticImage`, which calls the second. A
`StaticFrame` carries its own `[256]color.RGBA`, copied from the sheet it was decoded from, and the
inner loop is a palette lookup and four byte stores with no bounds test, the clip having been taken
once as rectangle arithmetic. No other operation stands between a decoded frame and the destination:
the menu path composites 24-bpp images with no palette, and the `.16a` decoder reaches no renderer.

`ShadeChannel(ch, tint, level)` is `(ch + tint) * (96 − level) / 32` clamped at 255, with `level`
held into `[0, 95]`; `ShadeRGBA` applies it per channel and forces alpha opaque. The terrain table is
deliberately not materialised — the transform is per-channel linear, so it is applied to the
already-resolved 8-bit colour. `ShadeScale` is the float form the window's per-vertex colour scale
uses.

`Light` carries `Theta`, `Ambient`, `Range` and `SkyTint`; `DefaultDaytime` has `Ambient = 0x0e` and a
zero tint. The window builds its level grid from `DefaultDaytime` inline and holds no `Light` of its
own; `Viewer.Lit()` is `v.levels != nil && !v.unshaded`, so it reports terrain shading and folds in
the altitude guard. In the raster tool the resolved `Light` is a local of the **shaded** branch
alone, so nothing under `-unshaded` can see it today.

The window's sprite textures are cached in `map[*terrain.StaticFrame]*ebiten.Image`, built lazily on
first draw so a headless run uploads nothing. A mirrored unit sprite deliberately does **not** widen
that key: mirroring is a `GeoM` reflection and the pixels are identical.

## Design decisions

### DD-1 — the ramp is one 256-entry row computed per blit, not a table and not per-pixel arithmetic

The shading step is hoisted out of the pixel loop: a blit resolves the frame's 256 palette entries
through FR-1 once, into a local row, and the loop then indexes that row exactly as it indexes the
palette today. So the inner loop is **byte-identical** in the lit and unshaded cases — the loop is
the thing this layer holds to one implementation, and a second one is what would let the raster and
the window disagree per pixel with both looking plausible.

Rejected: **a materialised `[16][256]` table per sheet** — 16 KB per frame held for the session, to
answer one row per render; and it would be the first materialised shading table in the tree, against
the standing decision to apply the transform arithmetically. Rejected: **per-pixel shading inside the
loop** — it multiplies the work by `W*H/256`, and it puts the lit/unshaded choice inside the loop,
which is exactly where a divergence between the two paths would hide (FR-1, FR-4).

### DD-2 — `…Lit` counterparts beside the existing pair; the unshaded call is the same walk

`BlitStaticLit` and `RGBALit` take the tint and the row; `BlitStatic` and `RGBA` keep their
signatures and become the same walk over the frame's raw palette. This is the naming the render tier
already uses for exactly this distinction (`Composite`/`CompositeLit`), and it keeps FR-6's "byte for
byte as they are drawn today" a property of an unchanged call rather than of a re-derived argument.

Rejected: **widening `BlitStatic`'s own signature** — every call site churns, and the unshaded
diagnostic loses its own name, so a caller wanting raw pixels has to know which row means raw.
Rejected: **a boolean beside the row** — two ways to spell "unshaded", one of which (row 8, zero
tint) is already the other's exact equal, so they could disagree only by being wrong (FR-6).

### DD-3 — the sprite ramp is written in its own 16-row arithmetic, not remapped onto the terrain's 96

`ShadeChannel(ch, tint, 4L + 32)` is provably the same integer for every input: `(96 − (4L + 32))/32`
is `(64 − 4L)/32`, `2(16 − L)/16` is the same expression, and after truncation both sides are
`⌊(ch + tint)(16 − L)/8⌋`. So the identity holds for **all sixteen rows and every channel value** —
AC-3 is satisfiable by arithmetic, not by hope. It is rejected anyway, on two grounds.
It would make AC-3 a **tautology** — the criterion exists to check two independently derived ladders
against each other, and a remap would have it check one function against itself. And it would bind
the sprite ramp's domain to the terrain's `LevelCount`, so a later correction there would silently
move every sprite's colour. The two share the clamping integer helper and nothing else.

Rejected: **a float multiplier** in the style of `ShadeScale` — the raster path is exact integer with
a truncating divide, and a float would put the two front-ends' pixels a rounding apart (FR-1, FR-5).

### DD-4 — the row is a render-time scalar derived from the sun, carried by no frame and no placement

A function from `Light` to the row, evaluated once per render and passed down. `StaticFrame` gains no
field and `StaticPlacement` gains none: a frame is shared by every placement of its class and lives
across renders, so a row stored on it would be one render's row applied to the next.

Rejected: **baking the row into the frame at sheet-cache time** — it would make the data tier take a
light, and one cached frame could then only ever be one row. Rejected: **a per-cell row grid beside
the terrain level grid** — it models per-cell light sources this story does not have, and FR-2 makes
every entry of it equal (FR-2, P-4).

### DD-5 — the window's texture cache key widens to `(frame, row)`

The pixels genuinely differ between rows, which is what separates this from the mirror: a mirrored
sprite's pixels are identical and its key was deliberately left alone. With FR-2's one row per render
the key's cardinality is the number of distinct rows a session actually draws — one, or two across a
toggle — so the cache stays one entry per frame in the ordinary case.

Rejected: **dropping the cache when the row changes** — a second thing to keep true, invisible at the
call site, and it silently re-uploads every frame's texture on a toggle. Rejected: **a GPU colour
scale on the sprite draw**, mirroring what terrain does per vertex — it is a float multiply against
the raster's truncating integer divide, so the window and the raster tool would disagree per pixel;
terrain accepts that because its level is interpolated per vertex and cannot be baked, where a
sprite's row is one scalar for the whole frame (FR-5, FR-6).

### DD-6 — sprite lighting is gated on the unshaded switch alone, never on the altitude grid

`Lit()` folds two conditions together, and only one of them belongs here: a sprite's row comes from
the ambient byte and has no dependence on relief whatsoever, so a map whose altitudes were rejected
must still light its sprites. The window therefore reads `!v.unshaded` for this and leaves `Lit()`
untouched, keeping the terrain predicate's meaning intact.

Rejected: **reusing `Lit()`** — it would make sprite colour depend on altitude data that never
enters it, and the dependence would be invisible in every map that has valid altitudes (FR-6).

### DD-7 — the raster tool resolves the sun once, above the shaded/unshaded branch

The resolved `Light` moves out of the shaded arm so both arms can see it, and the statics loop takes
the row from it, or draws unshaded under the diagnostic. One resolution, so `-ambient`, `-theta`,
`-range` and `-maplight` cannot mean one thing to terrain and another to sprites, and the render
descriptor keeps naming the sun that was actually used.

Rejected: **a second flag selecting a sprite row** — FR-7 refuses it, and two switches for one
physical light is how they drift apart (FR-7).

### DD-8 — the unit pass and the static pass keep sharing one image builder

Both window passes already reach pixels through `staticImage`, and both take FR-2's one row, so the
row is applied where the texture is built and neither pass learns about lighting. This is what makes
FR-3 structural rather than a discipline: there is no place for a class or an owner to be consulted,
because the only thing the builder is handed is a frame and a row.

Rejected: **lighting the unit pass separately**, in the pass that already knows about facing and
mirroring — it is where an owner colour would later be tempting, and it would give the two passes two
ramps to keep equal (FR-3, FR-5).

## Risks

- **R-1** At the default sun the gain is above 1.0, so bright palette entries saturate: a sprite's
  highlights flatten toward white and lose detail that is present in the raw art. The terrain
  transform has done this from the start, but a sprite is small and the flattening is more visible on
  art than on ground.
- **R-2** Every sprite is lit at one gain, so if a class is in fact meant to take an owner-selected
  ramp, our render draws every owner's units alike. A later story adding owner colour must replace
  this ramp rather than compose with it, or the gain is applied twice.
- **R-3** The row is uniform, so nothing in the render can be brightened or darkened locally: a lamp,
  a fire or a shrouded cell has no way to reach a sprite, and the layer will need a per-cell input
  before it can.

## Success criteria

- **SC-1** AC-1 and AC-3 hold in full, AC-3 exhaustive over all 4096 `(row, channel)` pairs and its
  terrain side taken from the existing transform rather than recomputed beside it (FR-1).
- **SC-2** AC-2 holds byte for byte over a background varying on both axes, so a one-pixel
  displacement or a single wrong entry fails it rather than averaging out (FR-1, FR-6).
- **SC-3** AC-4 holds on all four suns, including both clamped ends (FR-2, FR-7).
- **SC-4** AC-5 and AC-10 hold in full, each refusal and each out-of-range row on a case of its own,
  with P-1 and P-2 checked by comparing the whole destination before and after rather than the
  painted rectangle alone (FR-4).
- **SC-5** AC-6 and AC-7 hold in full, AC-7's two frames differing in size so a shared palette cannot
  be confused with a shared frame (FR-3, FR-5).
- **SC-6** AC-8 holds in full over the three renders, the two lit ones pinned at their own rows
  rather than compared only for inequality (FR-5, FR-6, FR-7).
- **SC-7** AC-9 holds in full, the third state's pixels compared with the first's byte for byte —
  which is the assertion a stale texture fails and an equality-free "they differ" would not (FR-5,
  FR-6).
- **SC-8** Four mutants, each applied to production code, run over the whole tree with its failing
  tests named, and reverted: the ramp numerator's `× 2` dropped; the standalone frame image left
  unshaded while the raster blit lights; the raster tool's sprite row pinned to the default sun
  rather than the resolved one; the window's texture key narrowed back to the frame pointer. Each is
  a design decision expressed as a defect — DD-3, DD-2, DD-7 and DD-5 respectively — so a survivor
  is a criterion that does not discriminate its own decision, and is reported as one (FR-1, FR-2,
  FR-5, FR-6, FR-7).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-1, DD-2, DD-3 | SC-1, SC-2, SC-8 |
| FR-2 | DD-4 | SC-3, SC-8 |
| FR-3 | DD-8 | SC-5, SC-8 |
| FR-4 | DD-1, DD-2 | SC-4 |
| FR-5 | DD-3, DD-5, DD-8 | SC-5, SC-6, SC-7, SC-8 |
| FR-6 | DD-2, DD-5, DD-6 | SC-2, SC-6, SC-7, SC-8 |
| FR-7 | DD-7 | SC-3, SC-6, SC-8 |
