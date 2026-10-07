# Spec — the map's static-object layer

## Problem and current behaviour

`pkg/formats/alm` decodes ALM section id 3 into `Map.Overlay`, a row-major `Width*Height` byte grid,
and `pkg/data` loads the 82 object classes of `objects/objects.reg` — each with its `ID`, `Index`,
`Width`/`Height`, `CenterX`/`CenterY` and `SpritePath()`. Nothing joins the two, so no renderer draws
a map's trees, bushes, stones or fences. `cmd/terraintool render` composes terrain to a PNG
(height-displaced by default, flat under `-flat`, shaded or `-unshaded`); `pkg/ui.Viewer` — behind
`cmd/mapview` and the game front-end `cmd/againrom` — composes the same terrain live, panned, clamped
and zoomed. Both draw diagnostic markers on placed structures (`-objects`, type-4) and units
(`-units`, type-6), and both show bare ground where the original shows a forest.

This story renders the missing layer in **both** renderers off one placement definition. `-objects`
and `-units` keep the meanings 0008 and 0009 gave them and are not repurposed; the type-3 art takes
the new name `-statics` and its own diagnostic marker `-staticmarkers`.

## Object placement contract

For a map with object grid `O` (`O[row*Width + col]`) and the loaded object classes:

1. **Byte to class.** `O = 0` is *no object*. `O = b` names the class whose `ID` is `b - 1`. A byte
   naming no loaded class contributes nothing.
2. **Class to sprite frame.** The sheet is the `.256` entry the class's `SpritePath()` names inside
   `graphics.res`, decoded with the palette that sheet carries; the drawn frame is the one the
   class's `Index` selects. Transparency is **structural** — carried by the decoded frame per pixel,
   never by a reserved palette index — so an opaque pixel may hold index 0. A class whose sheet is
   absent, undecodable or palette-less, or whose `Index` falls outside it, contributes nothing.
3. **Ground point.** A cell's ground point is its centre `(col*32 + 16, row*32 + 16)`, raised by
   `lift + originY`. Both are `0` in the flat geometry; in the height-displaced geometry `lift` is
   the cell's four-corner mean altitude (`Projection.AnchorHeight(col, row)`, the lookup and sign the
   diagnostic markers already use) and `originY` the render's native vertical origin
   (`Render.OriginY`).
4. **Sprite anchor.** The sprite's top-left in native pixels is

   ```
   anchorX = (CenterX - Width/2)  + frameW/2
   anchorY = (CenterY - Height/2) + frameH/2
   destX   = col*32 + 16 - anchorX
   destY   = row*32 + 16 - anchorY - lift - originY
   ```

   `Width`/`Height` are the class's canvas, `CenterX`/`CenterY` its anchor pixel, `frameW`/`frameH`
   the size of the **frame being drawn** — not the sheet's first frame, not the canvas. Every `/2`
   truncates toward zero. Frame pixel `(CenterX, CenterY)` is the one that lands on the ground point.
5. **Blit.** Every frame pixel that is not transparent is written at `(destX + px, destY + py)` as
   the fully opaque colour its index selects in the **sheet's** palette, clipped to the destination;
   a transparent pixel leaves what is beneath untouched. There is no alpha blending and no terrain
   shading on object pixels.
6. **Draw order.** Objects are drawn after all terrain, water and shading and before every
   diagnostic marker, in row-major order (`row` ascending, then `col`) — so where two sprites
   overlap, the later cell's owns the overlap.

Native pixels are world pixels: one cell is 32 world pixels in both renderers.

## Functional requirements

- **FR-1** `pkg/data`'s object collection MUST expose a total, pure lookup from an object byte to a
  class per *Byte to class*, and MUST NOT mutate class state.
- **FR-2** The render tier MUST expose a pure integer function returning a sprite's top-left per
  *Sprite anchor*, its inputs being the cell, the class's `Width`/`Height` and `CenterX`/`CenterY`,
  the **drawn frame's** width and height, `lift` and `originY` — so no caller can obtain an anchor
  without supplying the canvas and the frame size.
- **FR-3** A pure, GPU-free function MUST turn a map's object grid, the loaded classes and the
  chosen geometry into the **placement list**: one entry per cell resolving to a drawable frame, in
  row-major order, each carrying its cell, its frame and that frame's world top-left and size. Both
  renderers MUST place from this one list, so they cannot diverge.
- **FR-4** The render tier MUST expose a blit per *Blit*: transparent pixels see-through, opaque
  pixels through the sheet's palette, clipped on every edge with no read or write outside either
  image and no error.
- **FR-5** `terraintool render -statics` MUST blit the placement list onto the composed raster in
  both geometries, after terrain and before every marker, and MUST report the number of placements
  drawn in its summary whenever the flag is given — on the flag alone, never on the count. Without
  the flag every byte of its image and summary MUST be what it produced before this story.
- **FR-6** `-staticmarkers` MUST draw a diagnostic marker on every resolving type-3 cell, positioned
  from the **cell alone** through the existing marker geometry — no class field, no frame size, and
  not through the FR-2 anchor — so the marker and the art are two independent derivations of one
  ground point and their disagreement is visible. It MUST mark every cell that resolves to a
  drawable frame, in a colour distinct from the `-objects` and `-units` markers, and MUST draw
  beneath neither. Either flag MUST be usable without the other, and the same marker MUST be
  reachable in `cmd/againrom` through the switch it already has (FR-11): the instrument belongs in
  the binary the layer is played in, not only in the developer tools.
- **FR-7** Object art is full-resolution and the raster path does not resample it, so `terraintool`
  MUST refuse `-statics` at any `-scale` but 1, with an error naming the conflict and no output file
  created, truncated or modified. `-statics` MUST otherwise combine with `-flat`, `-unshaded`,
  `-objects`, `-units` and `-staticmarkers`.
- **FR-8** A GPU-free loader MUST decode the object classes and their drawn frames into an in-memory
  bundle, skipping what *Class to sprite frame* excludes, so it is safe under `-check` and in tests.
  `pkg/ui.Viewer` MUST accept such a bundle optionally: with one it builds the placement list once,
  at construction, purely and without a GPU, from its own geometry; without one it behaves exactly as
  before — no object layer, no summary change.
- **FR-9** `Viewer.Draw` MUST draw the object layer between the terrain and every diagnostic marker,
  each **visible** placement at `screen = (world - camera) * Zoom` with its native art scaled by
  `Zoom` and sampled nearest, in placement order. Object textures MUST reach the GPU lazily on first
  draw — never at construction, never under `-check`.
- **FR-10** Visibility MUST be decided by a pure function of the placement list and the camera, using
  each placement's **exact world rectangle** — so a tall sprite whose ground cell is below the view
  but whose crown enters it is kept — preserving order among the survivors and adding, dropping or
  duplicating nothing else.
- **FR-11** `cmd/mapview` MUST gain `-statics` and `-staticmarkers`, both default off, reporting the
  placement count under `-check` exactly when `-statics` is given and leaving its summary otherwise
  unchanged. `cmd/againrom` MUST load the bundle once at startup and draw objects in every map it
  opens — game content, not a diagnostic — and MUST gain neither flag: its existing marker switch
  MUST cover the type-3 cross beside the structure and unit crosses it already draws — one question,
  three glyphs — and MUST keep its shipped default. Both MUST load the bundle headlessly under
  `-check`, failing there if `objects/objects.reg` cannot be read.

  *Folded from hotfix `2ad4c29` — see `docs/hotfix/ARCHIVE.md#2ad4c29`.* The diagnostic marker ships
  OFF in the game and `-markers` brings it back; `cmd/mapview` is a developer's window and its
  defaults do not move.
- **FR-12** The lookup, the anchor, the placement list, the blit and the visibility test MUST be
  testable headlessly against synthetic data, with no game install and no GPU.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | classes with `ID` 0, 1, 2 loaded | the lookup takes bytes 0, 1, 3, 83 | none; class 0; class 2; none — and the classes are unmutated |
| AC-2 | unit | a cell, canvas `W x H`, centre `(cx, cy)`, drawn frame `fw x fh` equal to / smaller than / larger than the canvas, a `lift`, an `originY` | the anchor function is called | it returns exactly the *Sprite anchor* top-left, each halving truncating toward zero, and the equal-frame case at `lift = originY = 0` reduces to `(col*32+16-cx, row*32+16-cy)` |
| AC-3 | unit | a synthetic map: two resolving cells on adjacent rows, a zero cell, a byte naming no class, a class with no drawable frame | the list is built flat, then displaced | exactly two entries, row-major, each with its frame size and *Sprite anchor* top-left; the displaced pair differs from the flat pair by exactly `-(lift + originY)` in Y and zero in X |
| AC-4 | unit | a synthetic frame mixing transparent and palette-index pixels, and a destination | blitted wholly inside, then overrunning each of the four edges | non-transparent pixels take their palette colour at the right offsets, transparent pixels leave the destination byte-identical, and each overrun clips without panic or out-of-bounds write |
| AC-5 | unit | a synthetic flat map whose two objects' sprites overlap; and `-statics` at `-scale 2` | composed with `-statics -objects -units -staticmarkers`; then validated | both objects paint over the terrain at their anchors, the later row owning the overlap, and all three marker layers paint over them; the `-scale 2` run is refused with an error naming the conflict and no file created or modified |
| AC-6 | unit | a placement list and a camera showing part of the map, at zoom 1 and a non-unit zoom; plus a placement whose ground cell is below the view but whose sprite reaches into it | the visibility test and transform run | only view-intersecting placements survive, in order, none duplicated; each maps to `screen = (world - camera) * Zoom` at scale `Zoom`; the tall placement is kept |
| AC-7 | integration | a `Viewer` built without a bundle and one with it; `terraintool -statics`, `-statics -flat`, `-statics -unshaded`; `mapview -statics -check` and `mapview -check`; `againrom -check` | constructed, then run headlessly | every run completes with no GPU and no texture built at construction; the bundle-less viewer holds no placements; both tools report the count only with the flag; `againrom -check` loads the bundle and fails if `objects/objects.reg` cannot be read; every no-flag image and summary is byte-identical to the pre-story output |
| AC-8 | corpus | every shipped ROM1 map, classes loaded from a lawful install | each placement list is built, and each placement's ground point is recomputed from its cell through the marker geometry | every map builds without error and its drawable count is recorded; **every placement's sprite ground point equals the marker-derived point for its cell**, in both geometries, within the bound P-4 states; bytes naming no class, and classes whose art is absent, are counted and skipped rather than failing |
| AC-9 | manual | representative maps showing **both** the art and its own cross | `terraintool` output inspected; `mapview` and `againrom` opened, panned, clamped, zoomed | every object's base sits on its own cross — none stands a tile or more away from the cell marked for it — and the art is rooted on the terrain with palette and transparency correct, follows the displaced surface, sits on the flat lattice under `-flat`, pans and zooms with the map, only on-screen objects draw, markers stay on top, both renderers place identically; every disclosed limitation is named |

Error cases: `-statics` at a scale other than 1 (AC-5, P-8); an unreadable `objects/objects.reg` when
the layer is requested (AC-7). A byte naming no loaded class, a class with no loadable sprite and an
`Index` outside its sheet are skipped, not errors: the layer draws what it can and never fails a run
on data it cannot use.

## Derived properties

- **P-1** (invariant) The lookup is total and pure: byte 0 and any byte naming no loaded class yield
  none, byte `b` yields the class whose `ID` is `b - 1` when loaded, and class state never changes.
- **P-2** (invariant) The anchor is a pure integer function of its inputs. At `lift = originY = 0` it
  is the flat anchor; otherwise it differs from it by exactly `-(lift + originY)` in Y and zero in X
  — the displacement is vertical only, and `destX` takes no origin term.
- **P-3** (invariant) The placement list is deterministic, and both renderers place from it
  unchanged, so a cell's art lands at the same world point in the raster and in the window. Class
  state and altitudes are read, never written.
- **P-4** (invariant, checkable) For every drawn placement the sprite's ground point — its top-left
  plus its own `(anchorX, anchorY)` — coincides with the marker geometry's anchor point for that
  cell. The two are computed by different code from different inputs, so this can fail. What it
  discriminates is bounded and the bound is part of the contract: the anchor terms cancel
  algebraically, so it catches a wrong cell-to-world mapping or a wrong `lift`/`originY` lookup or
  sign, and it cannot catch a wrong `CenterX`/`CenterY` convention or a canvas/frame mix-up. Those
  are caught only by AC-9, where a human sees art standing away from its cross.
- **P-5** (negative-invariant) Without a bundle or the flags, both renderers are what they were: no
  object draw, no GPU texture, images and summaries byte-for-byte the pre-story ones. With the layer
  on it only adds pixels — terrain, water and shading are unchanged wherever no object paints, and
  no existing marker's geometry, colour or position moves.
- **P-6** (completeness) Every cell whose byte resolves to a class with a drawable frame contributes
  exactly one placement, in row-major order; the only omissions are the error-case skips.
- **P-7** (invariant) The blit never writes outside the destination and never reads outside the
  frame, at any position including wholly off-image, and a transparent pixel is a no-op; the
  visibility test never adds, reorders, duplicates or drops a placement that intersects the view.
- **P-8** (negative-invariant) For a refused `-statics` scale, no output file is created, truncated
  or modified, and no image is composed.

## I/O examples

```text
terraintool render -assets <dir> -map Cross.alm -out out.png -statics -staticmarkers
# terrain: 256x256 cells (65536), ..., geometry projected, y origin -N, statics N
# every object should stand on its own cross; one that does not is a placement bug

terraintool render -assets <dir> -map Cross.alm -out out.png -statics -scale 2
# terraintool: -statics requires -scale 1 (object art is not resampled); out.png untouched

mapview -assets <dir> -map Cross.alm -statics -check
# mapview: Cross 256x256 cells (65536), tile slots .../..., statics N, water speed ...
```

## Constraints and alternatives

| Choice (all selected; *disclosed* = a named fidelity gap) | Observable trade-off |
|---|---|
| The sprite anchor and the marker anchor stay **independently derived**, never one shared function | a wrong anchor shows as art standing away from its cross instead of moving both together into a consistent wrong picture that no test could represent |
| New names `-statics`/`-staticmarkers`, `-objects`/`-units` unchanged | four adjacent flag names, against a flag silently changing what it draws under the owner |
| Opt-in in the developer tools; in `cmd/againrom` the art is unconditional and the cross rides the one marker switch already there | the diagnostics stay lean, and the layer's instrument is where the layer is played |
| Native `-scale 1` only for `terraintool -statics`, while the window scales art with zoom | no resampling on the raster path and no `-scale`-style limit in the window; blocky at high zoom |
| One sprite per cell — the object's own sheet, no shadow pass, no sibling sheet — *disclosed* | objects cast no shadow, so a lit map reads flatter than the original's. A cell issues four draws; the one drawn is the one that can be placed, the other three resting on a horizontal sun shear that is located but **not decoded** and on a weakly established selector (Out of scope) |
| The frame the class's `Index` selects, always — *disclosed* | static objects only: a cell the original would draw in a dead or an animated form draws its default frame |
| Objects beneath every marker; the camera stays clamped to the terrain rect — *disclosed* | markers stay readable over content; an edge object whose sprite overhangs the map cannot be scrolled fully into view, matching the raster path's clip at the canvas edge |

Not claimed pixel-identical to the original: the draw order, the omitted shadow and the
always-`Index` frame are disclosed choices. Tests use synthetic sprites and maps only.

## Out of scope

- The shadow pass and the sibling sprite sheet an object also ships — deferred because the shadow's
  horizontal sun shear is undecoded. They become in scope the day that shear is decoded; nothing else
  about this layer has to change to admit them.
- The dead-object form a cell can select, object animation, fire variants, destructibility,
  passability and obstacle collision, and picking or selecting an object.
- Sprite resampling on the raster path; filtering beyond nearest in the window; sub-pixel or
  rounding parity with the original.
- Slope occlusion and depth buffering; any change to ALM decoding, class loading, the 0012
  projection, terrain rasterisation, water, 0014 shading, or the 0008/0009/0015 marker overlays.

## Verification mapping

AC-1…AC-7 and P-1…P-8 are CI-automatable headlessly against synthetic data. AC-8 needs a lawful
install; counts are recorded, assets never committed. AC-9 is a developer-run visual check needing a
window.

Gate coverage: FR-1→AC-1/P-1; FR-2→AC-2/P-2; FR-3→AC-3/P-3/P-6; FR-4→AC-4/P-7; FR-5→AC-5/AC-7/P-5;
FR-6→AC-5/AC-8/AC-9/P-4; FR-7→AC-5/P-8; FR-8→AC-3/AC-7/P-5; FR-9→AC-6/AC-7/AC-9; FR-10→AC-6/P-7;
FR-11→AC-7/AC-9; FR-12→AC-1…AC-7.
