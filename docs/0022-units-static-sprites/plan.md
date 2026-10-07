# Plan — units as static sprites

## Baseline

`pkg/sim`'s byte form is version 1 — a 29-byte header, 21-byte records — encoded by the one
traversal behind both `MarshalBinary` and `Hash`; `binary_test.go` partitions it with a
hand-written offset table and pins a fixed world's bytes, `hash_test.go` their digest by an
in-test FNV-1a. The exported method set is pinned; `internal/archtest` holds the package to
stdlib-only imports and scans its sources. `alm.Unit` already carries `ClassID int16`, the raw
stored key; `mapload.FromALM` builds one entity `{ID: i, X, Y}` per unit.

`data.LoadUnitClasses` resolves `units.reg` — inheritance, defaults, duplicate-`ID` refusal —
and each class's sprite base: `SpritePath()` is `units/` + the `[Files]` entry + `.256`, empty
with no `File` resolved. `game.LoadStatics` feeds `sheetCache`, whose `frame(path, index)`
answers nil exactly for an absent, undecodable or palette-less sheet or an index outside it — a
zero-area frame IS returned; only the registry read/parse errors, the read error an unwrapped
`*fs.PathError`. `terrain` owns `StaticFrame`, `StaticPlacement` — `Rect()` the exact world
rectangle, `Ground()` = `TopLeft + Anchor` — and `StaticAnchor`, performing the one negation
of `lift` and `originY`; the marker family stands apart, `MarkerAnchor` from the cell alone,
`EntityMarkerRects` the 10-px square on its anchor.

`pkg/ui`: `overlayPass` is `{Color, Rects}`; `overlayPasses` prepends the entity square pass
before objects, units, statics; `Draw` paints terrain, `drawStatics`, then the slice.
`overlayScreenRects` shifts each displaced arm by `-proj.AnchorHeight(cell) - proj.MinV`;
`staticScreenRects` culls on the transformed exact rect; `staticImage` builds one texture per
`*StaticFrame` on first draw; `SetEntityCells` adopts the pushed cells, no flag, no `syncWorld`.
`pkg/game`: `NewFrontEnd` loads archives, menu, the object bundle, tileset, maps — an unreadable
object registry fails startup; `loadMap` runs `LoadMapViewer` then
`openMapWorld(mv.Map, mv.Viewer)`; `push` hands the viewer `entityCells(world)`, no arithmetic.
`cmd/terraintool` wires `render` over {terrain, res, alm, game}; only
`cmd/againrom/main_test.go`'s install fixture goes through `NewFrontEnd`. `internal/synth`
builds archives, `Reg`/`ObjectsReg` (its own sorted root) and `Sheet256`.

## Design decisions

### DD-1 — version 2 is the same form with one wider record

`Entity` gains `Class int32`; `encode` writes version byte 2 and 25-byte records — class at
record offset +20, the presence byte moving to +24 — and `UnmarshalBinary` refuses every version
byte but 2, the shipped 1 included, still ahead of the length division. `headerLen` stays 29;
nothing else moves; `NewWorld` and `Step` are untouched, so the one shared traversal puts the class in the
digest. The offset table gains the class rows; the pin trio is re-pinned: hand-transcribed bytes
over pin entities now carrying distinct classes, one negative, the digest re-derived through the
in-test FNV. Rejected: reading version 1 alongside 2 — 0019 defined one version and no migration
path precisely so this change fails loudly.

### DD-2 — the loader's whole change is one field

`FromALM`'s entity literal gains `Class: int32(u.ClassID)` — the int16-to-int32 conversion IS
the sign extension of the format tier's typed field. No helper, no registry, no other field;
`Schedule` and `unitCell` untouched. Rejected: a record-to-entity helper beside `unitCell` — a
second conversion site wrapping a one-field read.

### DD-3 — the bundle reuses the object layer's frame and placement

`terrain` gains `UnitClass{Width, Height, CenterX, CenterY int; Frame *StaticFrame}` and
`UnitSet{Classes map[int32]*UnitClass}`: the set is a map — the domain a sparse signed 32-bit
key, a miss a lookup miss at any value, negative included — and a nil `Frame` on an entry is
the excluded class, distinct from no entry. The frame IS `StaticFrame` — one pixel/palette
shape, one blit, one texture cache — and placement is
`UnitPlace(col, row int, c *UnitClass, lift, originY int) (StaticPlacement, bool)`:
`StaticAnchor` over the class canvas and the frame's own size, false for a nil or frameless
class, the returned `StaticPlacement` bringing `Rect()` and `Ground()` with it.
Rejected: a parallel unit frame and placement pair — the same blit, cull and ground rewritten,
plus a second texture cache; rejected: an array set like `StaticSet`'s — there is no placement
byte to index by.

### DD-4 — `LoadUnits` mirrors the statics loader; the frame rule is index 0

`game` gains `UnitRegistry = "units/units.reg"` and
`LoadUnits(a *res.Archive) (*terrain.UnitSet, error)`: read, `reg.Parse`,
`data.LoadUnitClasses` — the only error path, the read error left an `*fs.PathError` — then one
entry per loaded class keyed by `ID`, canvas carried across,
`Frame = sheetCache.frame(c.SpritePath(), 0)`. That call is the whole frame rule: the standing
block opens the sheet and facing 0 is sheet frame 0 under either `Flip` layout, so the
resolved-`Flip` rule's value at this story's fixed facing is the constant 0 — and `frame`
already answers nil for exactly the contract's exclusions, a sheet with no frame 0 included.
Rejected: reading `Flip` and computing the block arithmetic — a branch both of whose arms
return 0; rejected: a unit-side sheet cache — the statics one already remembers failures.

### DD-5 — what crosses the seam is a cell and art, or none

`ui` gains `MapEntity{Cell image.Point; Art *terrain.UnitClass}` and `SetEntities([]MapEntity)`,
REPLACING `SetEntityCells`: one setter, adopted slice, no show flag, no `syncWorld` — the
shipped contract with art beside each cell; `EntityMarkers()` still counts the entities held.
On the game side `mapWorld` carries the bundle, `openMapWorld` takes it, and `push` maps each
entity to `{cell, art}` — art the bundle entry when that entry holds a frame, nil otherwise —
`entityDraws` replacing `entityCells`, still no arithmetic. Resolution thus lives in the one
tier that sees both a world and a bundle; the window tier still cannot spell a simulation,
format or data type. Rejected: pushing class ids and the set into the viewer — resolution
moved into the tier that must not ask it; rejected: keeping both setters — two overlapping
entity states to reconcile.

### DD-6 — the entity layer is two passes in the slice tests read

`overlayPass` gains `Sprites []staticScreenRect`, and `overlayPasses` prepends TWO entity
passes — sprites, then squares — before the three diagnostics: the order lives inside the one
slice `Draw` walks; squares over sprites is semantic — no sprite may cover a square-only
entity. The builder walks the entities once in slice order
(ascending id, the world's own): a cell outside the grid joins neither list — one bounds test,
before either geometry; `UnitPlace` ok routes to the sprite list, everything else to the square
list, so sprite-or-square is total by construction. Sprites take `lift =
proj.AnchorHeight(cell)` and `originY = proj.MinV` displaced, 0 and 0 flat, then
`staticScreenRects` for the camera transform and exact-rect cull; squares go through
`overlayScreenRects` with `EntityMarkerRects`, glyph and colour untouched. `Draw`'s loop paints
a pass's `Sprites` through `staticImage` — nearest, scaled by the zoom the rect was computed
with, textures lazy on first draw — then its `Rects`; a zero-area frame places and culls like
any other and paints nothing, no texture built, ebiten refusing an empty image. The "after the
static art" clause rides `Draw`'s shipped `drawStatics`-before-the-slice order, untouched
here — the readable order is the slice's. A viewer holding no entities emits neither pass; an
art-less push emits the square pass alone — the 0020 slice byte for byte. Rejected: drawing
sprites in `Draw`'s body between `drawStatics` and the loop — sprites-under-squares would then
be a call order no test observes; rejected: building placements in the setter — a placement
depends on `Mode()`, live under `SetFlat`.

### DD-7 — the instrument: two pinned halves, compared exactly

AC-8's comparison runs in `pkg/game`, 0020's own seat: over an `alm.Map` literal whose entities
and unit cells arrive through the two production derivations, it asserts, per resolved entity
and camera, `UnitPlace(...).Ground()` against `MarkerAnchor` plus the marker path's own
`-AnchorHeight - MinV` over an independent projection — exact point equality through one
`WorldToScreen`, since the sprite's `originY` is `MinV` and the terms cancel identically. The
`ui` half makes it a statement about the screen: the in-package pass tests pin that the sprite
pass draws exactly `UnitPlace`'s placement at the mode's own lift terms and the square pass
exactly the family transform, so the two compared expressions are the drawn ones by
composition. A wrong lift, sign, origin or cell mapping moves
one side alone; a wrong class geometry only the sprite.

### DD-8 — the front-end loads once; the census is the game's export

`FrontEnd` gains `Units *terrain.UnitSet`, loaded in `NewFrontEnd` beside the object bundle
from the archive already open and failing startup the same way — headless, so `-check` reaches
it; the shipped install fixture gains the unit registry beside its object one. `loadMap` hands
`f.Units` to `openMapWorld`; no flag exists anywhere on the path, and a nil bundle — a
hand-assembled front-end — draws every entity as the square. For the corpus census
`game` exports `UnitCensus(m *alm.Map, set *terrain.UnitSet) UnitCounts{Sprites, NoClass,
NoFrame int}`: `FromALM`'s world, each entity classified by the bundle's three answers.
`cmd/terraintool` gains the `units` subcommand — `-assets`, `-map`, `-graphics` as `render`
takes them, one line `units: E entities, S sprites, C no-class, F no-frame`, non-zero on any
load failure. Rejected: the census on `cmd/mapview` — the standalone viewer changes in
nothing; rejected: `terraintool` building the world itself — widening its allow row to restate
a join `pkg/game` owns.

### DD-9 — fixtures: a `UnitsReg` builder, literals for everything else

`internal/synth` gains `UnitsReg`, `ObjectsReg`'s sibling writing `UnitCount`/`Unit<i>` — the
name-sorted-root rule stays written once. The bundle tests build registries whose classes sit
at both `Flip` values, inherit `File` through `Parent`, and name absent, undecodable,
palette-less and empty sheets; sheets come from `Sheet256`, maps are `alm.Map` literals, worlds
are hand-built, expected values literals beside their fixtures. `ui` tests hand-assemble
`UnitSet` values — the bundle is plain data, and an in-package `ui` test cannot import `game`.
Rejected: a fixture bundle exported from `game` — a loader standing between a test and the
data it asserts.

## Success criteria

- **SC-1** AC-1 and AC-2 hold, the version-1 refusal shown on a WELL-FORMED version-1 stream;
the offset table partitions the two-entity version-2 form with the class rows at +20 and the
presence byte at +24, and the re-pinned bytes and digest agree through the in-test FNV; two
worlds apart only in class ids step in lockstep (P-1); the method-set pin, import check and
source scan pass unedited (FR-1).
- **SC-2** AC-3 holds, the negative key arriving sign-extended; `Schedule`'s tests need no edit
(FR-2).
- **SC-3** AC-4 holds over DD-9's fixture archive, `File` inherited on one drawable class; the
one error names `units/units.reg` (FR-3, FR-8).
- **SC-4** AC-5 holds against `UnitPlace` directly; a nil and a frameless class place nothing
(FR-3).
- **SC-5** AC-6 and AC-7 hold: the two entity passes precede the three diagnostics, a viewer
holding no entities and one pushed art-less both yield the 0020 slice, and no `ui` identifier
names a simulation, format or data type (FR-4, FR-6).
- **SC-6** AC-8 holds in screen pixels, both geometries, several cameras; perturbing the
class's centre moves only the sprite side (FR-5).
- **SC-7** AC-9 holds for a k mid-schedule and one past the end; `push` hands art exactly where
the bundle holds a frame, a nil bundle pushing every entity art-less; nothing on the draw or
load path holds or mutates a world (FR-6).
- **SC-8** AC-10 holds over the two install layouts, the bundle loaded once in `NewFrontEnd`
(FR-7, FR-8).
- **SC-9** `UnitCensus` — AC-11's harness — counts the three answers apart over a synthetic
map and set; `terraintool units` prints the census line, exiting non-zero on a broken install;
no allow row moves (FR-3).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-1 | SC-1 |
| FR-2 | DD-2 | SC-2 |
| FR-3 | DD-3, DD-4, DD-9 | SC-3, SC-4, SC-9 |
| FR-4 | DD-5, DD-6 | SC-5 |
| FR-5 | DD-6, DD-7 | SC-6 |
| FR-6 | DD-5, DD-6 | SC-5, SC-7 |
| FR-7 | DD-8 | SC-7, SC-8 |
| FR-8 | DD-9 | SC-3, SC-8 |
