# Plan — 0058 hit test lift

## The measurement the design rests on

Both candidate causes are present. Only one is a disagreement between the drawing and the pick, and
that one is the whole of the reported symptom.

**The relief lift.** A glyph on cell `(c,r)` is placed at world Y `r*32 + dy`, with
`dy = −AnchorHeight(c,r) − MinV`. The pick looked for that cell at `r*32`. The two are apart by
exactly `AnchorHeight(c,r) + MinV`, which is zero on flat ground **and on uniformly raised ground** —
the projection's origin cancels a typical cell's own height — so the error is invisible except where
the relief varies. Measured on a synthetic plain rising to altitude 64: the gap is 64 world pixels,
**exactly 2.00 cells**, and it collapses to 0 when the same grid is raised uniformly. That is the
owner's "roughly two cells", and it is also the whole of the rim-versus-marquee gap in the
screenshot — the rim is placed from the cell, so it lands one lift above wherever the flat pick sent
the box.

**The sprite anchor.** A unit's body is drawn from `destY = r*32 + 16 − anchorY − lift − originY`,
so its top edge stands `anchorY − 16` world pixels above its cell's footprint. For the shipped canvas
`TERR-SPR-040` records (128×128, centre (64,78)) that is `frameH/2 − 2`: **30 px for a 64-tall frame,
62 px for a 128-tall one**, with the feet landing on or just below the footprint's lower edge. The
frame heights of shipped units are not established here, so the term is stated as a range and not as
a number.

**They are not the same kind of thing, and the story fixes one.** The lift is a disagreement — the
drawing and the pick answered differently about one cell. The anchor is not: both the rim and the
footprint are placed from the cell, and the body simply stands above them, as a body does. So after
this change a box drawn around a unit **as it stands on the ground** catches it, because the
footprint is where its feet are; a box drawn around a head alone does not, and would not have before
either. **AC-1 is stated over the rim's rectangle for exactly that reason**, and whether a unit
should instead be picked by its drawn sprite is FR-1's named seam.

## The shape

**DD-1 — The cell's footprint becomes a named value in the render tier.** It already existed there
under a name belonging to one of its consumers — the blocked tint's rectangle *is* the cell
footprint, and that file's own comment said so. Renamed, it gains three more readers: the rim borders
it, the lattice outlines it, the pick tests against it.

**DD-2 — The rim and the lattice are one border builder at two thicknesses.** That is what makes
"the rim lands exactly on one lattice cell" structural rather than a coincidence of two glyphs.

**DD-3 — The UI tier gets one function from a cell to a rectangle in the view**, composing the
footprint, a displacement and the transform every glyph already takes. It is called by the pick and
by the lattice. **The rim reaches the same two pieces through the glyph path it already rides** —
four strips cannot be one rect — so what is shared is the footprint and the transform, not a single
call. FR-2 is worded to that, and the claim is not widened past it.

**Rejected: inverting the projection.** Turning a screen point into a cell through the relief needs
a search over candidate rows, because a cell's lift depends on the cell, plus rules for a point in
no cell and a point in several. None of it is needed: `decide` already walks the entity list, so
testing each entity's own rectangle costs the same walk. The inverse is unavoidable only for FR-10's
ground pick, which is out of scope for exactly that reason.

**Rejected: hit-testing the sprite's rectangle.** It would make the pick depend on art, and an
entity that resolves no frame draws a square and has no sprite rectangle — so it would need a second
rule for the artless case and a depth rule where two bodies overlap.

## The pick

**DD-4 — `decide` takes the placement as a parameter and keeps its camera for the ground pick.** It
is more testable than before: a synthetic placement can be handed to it where the geometry previously
could only be reached by building a camera. Its purity is now the ordinary kind — a function of its
arguments, one of which closes over viewer state — and its comment says the narrower thing.

**DD-5 — The gesture is compared in screen space, closed against half-open** (FR-3, FR-4). `meets` is
`x0 < X+W && x1 >= X` per axis, **guarded by a positive-extent test**: the half-open reading is only
correct for a rectangle with area, and `placeArm` scales its extent by the camera's raw `Zoom` field
while taking its position through the clamped one, so a zero or non-finite extent beside a finite
position is reachable. That was harmless while such a rectangle was only drawn.

A zero-extent gesture is then the single point, so FR-5's tap-equals-one-point-box holds where it is
true — one hit test, one candidate set — and the tap's reduction to the lowest id is what makes the
two differ where several targets hold the point. Normalisation happens where the rectangle is formed,
so orientation-independence is one expression's property.

Screen space rather than world space is deliberate: it is the space the shared transform delivers,
so the comparison is against the rectangle actually drawn. Comparing in world space would mean a
second path skipping the camera and the cull — the parallel derivation this story removes.

**DD-6 — The half-open end raised by one goes with the cell range.** A closed gesture rectangle meets
what it touches with no such adjustment. The raising itself stays where it is, in a method this
story stops calling (DD-7); nothing inherits it.

**DD-7 — The camera's cell-range inverse is kept and re-documented, not deleted.** It has no
production caller after this change; two `pkg/game` tests still build a band with it. Deleting it
would rewrite tests in another package for no gain, and it carries no lift arithmetic to drift. What
must not survive is its comment claiming to be the box-selection path.

## The lattice

**DD-8 — It rides the placement surface, and that is a decision against the obvious one** (FR-7). Two
surfaces exist and the research names both: the terrain is drawn on the per-corner mesh, and every
placement stands on the mean of a cell's four corners. `TERR-SPR-039` measures them agreeing on
9.26 % of sloped cells, with `mean − corner` running −73…+55 rows. A lattice on the terrain's mesh
would stand somewhere other than the rim it exists to be read against, on about nine sloped cells in
ten — an instrument contradicting the thing it measures. Disclosed cost: on a slope the outline does
not sit exactly on the terrain's own quad. Which surface to outline is a choice about what the
instrument is for, not a reconstruction of anything the original draws (FR-9).

**DD-9 — The band is the terrain pass's own walk**, named once and iterated by both terrain paths and
the lattice. That band is a **superset** of what is visible — the displaced row range is bounded by
the grid's altitude extremes, not by the rows it names — so the lattice's own outlines are the band
**after** the view cull, which is what AC-10 is worded to and what its test stands on.

**DD-10 — One pass, first in the slice, translucent.** First it can hide nothing. Under the blocked
tint, deliberately: a tinted cell should read as tinted rather than as outlined. It cannot be under
the terrain or the cell-anchored art, which are painted before the slice exists; the colour is what
makes that safe rather than the position, as the tint's own comment already records.

**DD-11 — The toggle is a press-edge field on the input snapshot, read on the map arm alone** (FR-8), with
the state on the **viewer** and a setter, a toggler and a reader beside the other overlays' — so the
standalone viewer and the map screen cannot offer different behaviour. `F2`, which authors the
front-end's diagnostic register: an action the map screen performs takes a letter, a diagnostic it
shows takes a function key.

**DD-12 — A legibility floor, which is also the cost bound.** Below sixteen screen pixels a cell
cannot carry a readable outline, and that is the same threshold the cost argues for (R-2). One
number, chosen on the legibility side: the instrument stops when it stops informing.

## Risks

**R-1 — Boundary agreement on flat ground.** The old world-space band test expands to exactly the
new comparison, so the two agree in exact arithmetic. They are not the same *rounding*: production
zoom is `1.2^n`, and `(col*32 − camX)*z` need not round as `floor((camX + sx/z)/32)` did. **A
one-ULP boundary flip at a non-dyadic zoom is accepted and is not bounded here**; the suite runs at
zoom 1 and 2, both exact, so it verifies the algebra and not that. Two real narrowings ride with it,
both stated in the spec: the view cull is new, so a gesture past the window edge no longer reaches
cells outside the view; and the displacement is carried on flat ground, so a mid-step unit moves.
That second one was invisible to "the existing suite passes unchanged" — no pick test sets a step or
a phase — and is pinned by a new test rather than by that argument.

**R-2 — Frame cost of the lattice.** Measured, not assumed. On a 144×144 map at 1280×720: 920 cells
and 146 µs at native zoom; 3600 cells and 618 µs at the floor; and, without a floor, 20 736 cells,
82 944 strips and 3.5 ms at `ZoomMin` — the whole map, because the band in cells grows as the inverse
square of the zoom. The floor is what makes "bounded by the view" true rather than aspirational.

**R-3a — Nothing else about a gesture moves (FR-11).** The slop, the latch, the marquee, the branch
order and the right press's own body are untouched; the existing suite passing unedited is what says
so, and it is the one claim in this plan that argument alone would not have earned.

**R-3 — Two derivations deliberately kept apart.** The sprite anchor and the marker path are
independently derived on purpose, so a wrong lift shows as art standing away from its own cross.
The sharing here is between the marker/rim path and the pick; the sprite side keeps its own
expression over the same two sources.

## Success criteria

**SC-1** — FR-1..FR-6 hold on a viewer with real relief, tested without a window.
**SC-2** — FR-5's invariant is a test over a fixture that **contains** the overlap it can fail on.
**SC-3** — The rim's strips and the hit target are shown to span one rectangle (P-2, AC-9).
**SC-4** — The flat-ground suite passes unchanged, and the mid-step carve-out is pinned (AC-3, AC-3a).
**SC-5** — Lattice on/off, band, lift, floor and key are tested; off is byte-identical (AC-10..14).
**SC-6** — The order destination is shown unchanged (AC-13, FR-10).
