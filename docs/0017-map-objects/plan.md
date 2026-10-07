# Plan — the map's static-object layer

## Baseline

`Render.OriginY` **is** the projection's `MinV`, and the windowed viewer's world Y is
`Projection.Vertex(c,r) - MinV`. At `-scale 1` and zoom 1 the raster's image space and the camera's
world space are therefore the same integer lattice, so one `(lift, originY)` pair places a pixel in
both renderers.

`markerRects` centres its cross at `row*cellpx + cellpx/2 + offsetY + liftY`, reads no class and no
frame, and is fed OUTPUT pixels by `cmd/terraintool` (`-render.OriginY*scale`,
`-AnchorHeight(col,row)*scale`) but a zero lift by `pkg/ui`, which afterwards adds the world-pixel
`dy = -AnchorHeight(col,row) - MinV` — so the map-extent clip runs un-lifted there. Cells reach it
from `pkg/game.MarkerCells` in the window and from `cmd/terraintool`'s own
`anchorCells`/`unitAnchorCells` in the raster.

`objects/objects.reg` and the `objects/*.256` sheets are entries of the one `graphics.res` every
front-end already opens. `pkg/data` gives an unset object scalar its decoded default — `-1` for the
five canvas and frame keys, which all 82 shipped classes set anyway. `cmd/againrom` ships `-markers`
**on**, and the crosses it draws are the type-4 and type-6 ones, on records this layer does not
touch.

## Design decisions

### DD-1 — the byte-to-class rule has one implementation, and the bundle is keyed by the byte

`pkg/data` gains `func (c *ObjectClasses) ByCode(b byte) (*ObjectClass, bool)`: `0` answers none,
anything else is `ByID(int32(b) - 1)`. It reads the resolved collection and writes nothing (FR-1).

The loader walks `1..255` through it once and the render tier's `StaticSet` is indexed by the
placement byte, so `b - 1` exists in one place and the per-cell path is an array read. Rejected:
re-deriving `b - 1` in the render tier's builder, a second copy of one decoded rule in a tier that
would have to learn the registry's identity convention to hold it; rejected: keying by class `ID`,
which moves the copy rather than removing it.

### DD-2 — the anchor is one function whose parameter list cannot omit the canvas or the frame

`terrain.StaticAnchor(col, row, canvasW, canvasH, centerX, centerY, frameW, frameH, lift, originY
int) (destX, destY, anchorX, anchorY int)` — ten positional integers, every one required. It returns
the top-left **and** the frame's own anchor pixel, so a placement's `(anchorX, anchorY)` comes from
this function rather than from a second copy of the same halving (FR-2). `lift` is the cell's
**height**, subtracted here, not the negated pre-scaled offset the marker path passes as `liftY`:
one negation, in the formula that owns it.

Rejected: the shape the baseline declared, `ObjectAnchor(x, y, cx, cy, …)` — with no canvas and no
frame size in its signature it cannot state the contract, and agrees with it only where a frame
fills its canvas. Rejected: an options struct or a class method — a keyed literal compiles with
fields omitted, so the requirement would be a comment, not a signature.

### DD-3 — one geometry-parameterised list; the viewer builds both of its geometries at construction

`terrain.StaticPlacements(g Grid, set *StaticSet, lift func(col, row int) int, originY int)
([]StaticPlacement, StaticCounts)` is the pure, GPU-free builder (FR-3). A nil `lift` is the flat
geometry, mirroring `drawMarkers`' nil-safe `liftAt`. `Grid` gains `Overlay []uint8` — the map layer
riding to the viewer as `Altitudes` does, set beside the tile grid at both literals,
`pkg/game.LoadMapViewer`'s and `cmd/terraintool`'s. A length other than `Width*Height` yields no
placements, `validAltitudes`' rule applied to the other layer.

`StaticPlacement` carries `Cell`, `TopLeft`, the frame's own `Anchor`, and `Frame` as a **pointer**
into the set — DD-8's cache key — with `Rect()` (the exact world rectangle) and `Ground()`, the
`TopLeft + Anchor` sum. `Ground()` reads what was drawn rather than recomputing it, which makes
DD-9's check a comparison of two derivations and not of one against itself. `StaticCounts` counts
placements and the two skip kinds apart and is geometry-independent, so one record serves both
lists; `Viewer.Statics() (placements int, counts StaticCounts)` reports it without a window.

The viewer builds the flat list and — when a projection exists — the displaced list, once each, at
construction, selecting between them by `Mode()` per frame (FR-8). Rejected: rebuilding on
`SetFlat`, a second build path that can drift from construction's; rejected: keeping the flat list
alone and shifting it per frame in the window, which stops the window placing from the built list
unchanged — the whole content of the raster/window agreement.

### DD-4 — the marker keeps its own derivation; the extracted centre is *the marker's*

`-staticmarkers` is a third glyph through the shipped marker geometry: `StaticMarkerColor`, the
opaque `{0xff, 0x20, 0x40}`, with `staticArmRadius = 3`, `staticArmThickness = 1`,
`StaticMarkerRects` and `DrawStaticMarkersAt`/`AtHeights`; in the window, `overlayScreenRects` with
the new glyph builder, whose transform, displaced `dy` and view cull are the shipped ones. Which
cells are marked comes from the built list; **where** a mark goes comes from the cell (FR-6). Two
entry points reach this one glyph: `-staticmarkers` in the tools, and the game's own switch (DD-7).

The cross centre `markerRects` already computes becomes `MarkerAnchor(col, row, cellpx, offsetY,
liftY int) (x, y int)`, called by `markerRects` and exported so the second derivation has a name.
Same arithmetic, no geometry moves, and the name is the marker side's alone: the sprite side never
calls it and `StaticAnchor` is never called on the marker's behalf.

Rejected: one shared "where does this stand" helper, which `pkg/game.MarkerCells`' doc comment
forbids at this exact seam — a wrong anchor would move cross and art together, leaving the screen
self-consistent and wrong and the disagreement unrepresentable. Rejected: a glyph reaching further
than the unit cross: drawn last, a superset hides a coincident unit marker entirely, where radius 3
by thickness 1 is a subset of both shipped glyphs at every scale, **strict** only at or above native:
below it `scaleDim` collapses radii 3 and 4 onto one pixel and the crosses coincide, which is the
hiding this rejects. So a glyph MUST NOT be built below native, and neither renderer builds one.

### DD-5 — one blit applies the palette, and the GPU texture is built from it

`BlitStatic(dst *image.RGBA, f *StaticFrame, destX, destY int)` intersects the frame's translated
rectangle with `dst.Bounds()` once, then walks only the clipped rows, storing an opaque pixel fully
opaque through the frame's own palette and skipping a transparent one (FR-4).
`(*StaticFrame).RGBA()` is that same blit onto a fresh, wholly transparent image, so the window's
texture and the raster's pixels come out of one implementation of the rule.

Rejected: a second palette walk in `pkg/ui` for the texture — two sites for one rule, and the
renderers could disagree per pixel while both looked plausible. Rejected: leaning on
`image.RGBA.SetRGBA`'s bounds guard instead of clipping — a test per pixel, and none at all on the
source side.

### DD-6 — the bundle is typed in the render tier and loaded in `pkg/game`; no package is added

`StaticPixel`, `StaticFrame`, `StaticClass` and `StaticSet` are standard-library-only types in
`pkg/render/terrain`, which is what lets `pkg/ui` accept a bundle over the import it already has.
`game.LoadStatics(a *res.Archive) (*terrain.StaticSet, error)` reads `objects/objects.reg` through
`pkg/formats/reg` and `pkg/data`, decodes each class's sheet with `pkg/formats/spr256` and converts
the frame `Index` selects (FR-8). Only an unreadable or unparseable registry is an error; every
sheet or index exclusion the contract names leaves the class artless and counted.

`internal/archtest`'s allow map is therefore **unedited**, and being fail-closed it is what says so:
`pkg/game` already reaches `pkg/data` and `pkg/formats/{reg,spr256,res}`, and `cmd/mapview` already
reaches `pkg/formats/res`. Rejected: a `pkg/render/statics` sibling — a new allow-map row plus an
intra-render edge to `pkg/render/terrain` for the marker geometry this layer is compared against, to
separate a layer whose geometry *is* the terrain geometry. Rejected: loading inside
`pkg/render/terrain` behind `EntrySource` — that tier is standard-library-only and the sheet decode
is not.

The layer reads `ID`, `File`, `Index`, `Width`, `Height`, `CenterX` and `CenterY` and nothing else,
so an absent-scalar default reaches it only as a value it passes through; `DeadObject`/`FireObject`
are consulted nowhere.

### DD-7 — the bundle is a load-path parameter, never a post-hoc setter

`game.Markers` gains `Statics`, so all three crosses stay one question, and `game.LoadMapViewer`
takes a fifth argument, `game.StaticLayer{Set *terrain.StaticSet; Art bool}` — the bundle and
whether sprites paint. `ui.NewViewerWithStatics(title, g, set, statics, art, markers)` is a new
constructor `NewViewer` delegates to with a nil bundle and both switches off (FR-11). Art and cross
are **two** switches: either flag must work without the other, yet both need the bundle to know
which cells resolve, so `-staticmarkers` alone loads it, builds the lists and paints no sprite.
`FrontEnd` gains `Statics`, loaded once in `NewFrontEnd` and passed as `StaticLayer{Set: f.Statics,
Art: true}` beside a `Markers` whose three fields all come off the game's one flag; `cmd/mapview`
loads the bundle on either new flag through `game.OpenGraphics(path) (*res.Archive,
*terrain.Tileset, error)` — one open for both, `OpenTileset` kept over it.

Rejected: a `Viewer.SetStatics` after construction — the shipped defect the marker parameter exists
to close, where the game drew terrain and nothing else for five stories. Rejected: a
`LoadMapViewerStatics` twin — a caller could keep the four-argument entry point and silently draw no
objects. Rejected: widening `ui.NewViewer`, whose every call site and test would grow arguments it
never uses; the precedent for that seam is a second function.

`cmd/againrom` gains no flag and its `-markers` default stays **on**; what the flag covers grows.
The earlier reason for leaving it alone was void — those crosses mark type-4 and type-6 records, so
the game would have drawn trees with no cross of their own and the instrument for this layer would
have lived only in a binary the owner does not run. The real reason: it must exist where the layer is
played, on the one question 0010 DD33 gave that binary.

### DD-8 — culling and the transform are one pure function; textures reach the GPU on first draw

`pkg/ui` gains `staticScreenRects`: each placement's `Rect()` through `cam.WorldToScreen` with its
size scaled by `cam.Zoom`, kept when it meets the view — the same field read and cull shape the
marker transform uses. The exact world rectangle is what keeps a sprite whose ground cell is below
the view but whose crown enters it, and survivors hold their order with nothing added or duplicated
(FR-10). With the art switch on, `Draw` paints them between the terrain and the first overlay pass,
nearest-sampled, `GeoM` scaling by `Zoom` then translating (FR-9); the switch gates the draw, never
the build.

A frame's `*ebiten.Image` is built on its first draw and cached by frame identity, so construction
and every `-check` run build none. Rejected: culling by ground cell, or by the camera's own tile
band — both drop exactly the tall sprite the contract keeps. Rejected: uploading at construction —
it makes a headless check need a GPU, the one thing `-check` exists to avoid.

### DD-9 — `terraintool`: the refusal first, the layer between, every ground point checked

Both flags default off. `-statics` at a scale other than 1 returns its error immediately after the
required-flag check — before the archive is opened and far before `os.Create` — so nothing is
created, truncated or modified and no image is composed (FR-7). The blit loop runs after the
compositors and before all three marker calls, in list order (FR-5); `statics N` is reported on the
flag alone and sits immediately before the `objects` token in both tools, keeping the object and
unit tokens adjacent.

Whenever a list is built, each placement's `Ground()` is compared with the marker geometry's own
anchor for its cell, and a disagreement fails the run naming the cell. The comparison is **native**,
at `cellpx = CellSize` — `MarkerAnchor(col, row, CellSize, -render.OriginY, -proj.AnchorHeight(col,
row))` — so no `-scale` enters it and `-staticmarkers` stays usable at any scale. The negation is
written at that call site while the builder took `lift`/`originY` as arguments: the two stay
separate expressions over the same two sources, so a wrong sign, lookup or cell-to-world mapping on
either side shows and only the anchor cancels — the bound the contract states. Both flags widen the
guard on the `Projection` `-objects`/`-units` already build and reuse it; the window runs no such
check, placing from the same builder. The failure is an internal-consistency one, deliberately
outside the contract's error list: a run whose two derivations disagree composes a wrong picture, and
refusing to write it is honest.

Rejected: a summary token for the mismatch count, which changes a summary shape three stories pin
for a number that is either zero or a bug; rejected: checking inside the builder, since an invariant
enforced by the code it constrains cannot fail. A test alone would not do: no synthetic fixture
reaches the class data the corpus criterion is about.

### DD-10 — the synthetic fixture path, and the five existing test files that change

`internal/synth` gains a `.256` stream builder and a minimal objects-registry helper over `Reg`, so
`LoadStatics` runs through `res.OpenBytes` over an in-memory archive and every exclusion the
contract names is constructible. Everything else here is plain Go over hand-built placements, frames
and grids: no install, no window, no graphics context (FR-12). Rejected: per-package fixture
writers, four `.256` encoders for one format.

Five existing test files change and a sixth would mean an unplanned interface moved:
`pkg/game/mapload_test.go`, `cmd/mapview/main_test.go` and `flat_test.go` for the widened
signatures; `cmd/mapview/flagset_test.go`, which byte-compares `shippedUsage` while its flag
inventory is hand-written, so the flags turn it red and omitting them from the usage line turns
nothing red; and `cmd/againrom/main_test.go`, whose graphics archive of one dummy entry must now
hold a readable `objects/objects.reg` and whose marker-flag assertions gain the third glyph.

## Success criteria

- **SC-1** `ByCode` answers AC-1's four bytes as AC-1 states, and the collection compares equal
  after the calls (AC-1, P-1).
- **SC-2** `StaticAnchor` matches hand-computed values over AC-2's three frame-to-canvas relations,
  at odd canvases and centres and both signs of `lift`/`originY`; halving the canvas alone, or
  rounding, fails specific rows (AC-2, P-2).
- **SC-3** AC-3's synthetic map yields exactly two placements, row-major, the displaced list
  differing from the flat one by exactly `-(lift + originY)` in Y and zero in X, the two skip kinds
  counted apart, and a short `Overlay` yielding none (AC-3, P-3, P-6).
- **SC-4** AC-4's blits — inside, over each of the four edges, wholly off-image — land its offsets
  and colours, leave every pixel outside the clipped rectangle byte-identical, and never panic or
  write out of bounds; `RGBA()` equals the same blit onto a transparent canvas (AC-4, P-7).
- **SC-5** In both geometries every placement's `Ground()` equals `MarkerAnchor` for its cell, and
  the bound is pinned with it: perturbing the cell-to-world mapping or the lift's sign fails it,
  while perturbing `CenterX` or swapping canvas for frame size does **not** — which is what leaves
  those to SC-11 (P-4).
- **SC-6** AC-5's overlapping pair composes in its stated order under all four flags; `-statics
  -scale 2` is refused with an existing file byte-identical afterwards and a missing one not
  created, while `-staticmarkers -scale 2` renders and reports nothing (AC-5, P-8, FR-7).
- **SC-7** At zoom 1 and at a non-unit zoom, AC-6's cull and transform hold, the tall placement is
  kept, and one that met the view only before the transform is dropped (AC-6, FR-10).
- **SC-8** Each of AC-7's runs completes with no window: no placements without a bundle, placements
  and no texture with one, and with the art switch off too; `statics N` exactly on the flag; the
  game's selection carrying all three glyphs at its shipped default; and `againrom -check` failing on
  an unreadable `objects/objects.reg` (AC-7, FR-11).
- **SC-9** Without the flags and without a bundle, the PNG bytes and both summaries are
  byte-identical to the pre-story output at several scales and in both geometries; with the layer
  on, only object pixels differ and no shipped marker moves (P-5).
- **SC-10** Over every shipped ROM1 map the list builds, the drawable and skip counts are recorded,
  the ground-point comparison passes on every placement in both geometries, and no run fails on an
  unresolved byte or an artless class (AC-8).
- **SC-11** A manual pass walks AC-9's checklist: the cross-versus-base agreement in **all three**
  front-ends — the tools under `-staticmarkers`, the game under its own default — and the disclosed
  shadow, frame-selection and edge-overhang gaps recorded as such (AC-9).
- **SC-12** The gates and the import-graph check are green with **no edit to `internal/archtest`'s
  allow map**, and the five test files DD-10 names are the only ones whose **shape** changes. A test
  file a task entry names may grow purely additively — T6's does — but no existing fixture is edited.
  A new package would have failed that check fail-closed.

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-1 | SC-1 |
| FR-2 | DD-2 | SC-2, SC-5 |
| FR-3, FR-8 | DD-3, DD-6 | SC-3, SC-8, SC-12 |
| FR-4 | DD-5 | SC-4 |
| FR-5, FR-7 | DD-9 | SC-6, SC-8, SC-9 |
| FR-6 | DD-4 | SC-5, SC-6, SC-10, SC-11 |
| FR-9, FR-10 | DD-8 | SC-7, SC-8, SC-11 |
| FR-11 | DD-7 | SC-8, SC-11 |
| FR-12 | DD-10 | SC-1, SC-2, SC-3, SC-4, SC-7 |
