# Plan — two numbers in the record, and one predicate on either side of the seam

## Baseline

`sim.Entity` is `{ID, X, Y, TargetX, TargetY, Class, HasTarget, Stall}` — every field an integer or
a bool, which is why `Entities()` hands out copies reaching nothing of the world. `World` holds
`tick, rng, bounds, mode, grid, entities, routes`; `nostate_test.go`'s reflect table pins both field
sets by name and declared type.

`NewWorld` is the only constructor: it copies, sorts by id, refuses a duplicate id and a stall at the
limit, and **zeroes** a targetless entity's target and stall. `clearOrder(i)` drops target, stall and
route together, and the byte form refuses a nonzero stall and a stored route on an entity with no
target — the pattern this story extends.

The form is `headerLen 34`, `entityLen 26` with presence at +24 and stall at +25, then one route
section per entity. `encode` is the single traversal behind `MarshalBinary` and `Hash`;
`formatVersion` is 4 and every other first byte is refused. Four pins ride on it: `binary_test.go`'s
offset table, its hand-transcribed `pinBytes` and its length assertion; `hash_test.go`'s second FNV
with `pinDigest` and the cross-check between them; the derived `rlxTick1Digest`, `hybTick1Digest`
and `rtfDigest`; and a hand-written world per retired version in
`TestUnmarshalRefusesAWellFormedOlderStream`.

`newRouteScratch(w)` builds an occupancy **count** plane from every entity's cell; `moved` keeps it
level with the walk; `enterable` subtracts the asker's own presence; `terrainOpen` reads bounds and
grid alone. `Command` is `{Entity, X, Y}` with no kind, and `Step`'s first phase writes the three
target fields per command in slice order, through `indexOfEntity`.

`mapload.FromALM` builds one entity per `alm.Unit`, id from the slice index — the only spawn path in
the tree. `pkg/game`'s `mapWorld` holds world, schedule, bundle and viewer; `enqueue` makes a
`sim.Command` and marks its entity commanded in one statement; `tick()` drains the queue into exactly
one `Step`, then pushes `entityDraws()`. The seam — `ui.MapTick`, `ui.MapOrder`, `ui.MapCadence` —
is handed back by `ui.MapLoader` and bound in `loadMap`.

`ui.MapEntity` is `{ID, Cell, Art, Frame, Mirror}`; `decide` is the pure selection function — box,
tap, right-press — and `presentSelected` the one predicate behind the marks (`selectionCells`) and
the orders. `overlayPasses` appends one `{Color, Rects}` pass per glyph, built by
`overlayScreenRects(show, cells, rectsFn)`, which applies the per-cell relief lift and the camera
transform; `terrain.SelectionMarkerRects(col, row, cols, rows, cellpx)` is such a builder. `appInput`
carries one press-edge bool per key, read on the map arm of `App.step`.

## Design decisions

### DD-1 — two int32 fields and three predicates

`Entity` gains `HP` and `MaxHP int32` and three exported methods `Alive()`, `Downed()`, `Dead()`,
each a comparison on those two. `pkg/game` reads the predicates; nothing outside `pkg/sim` recomputes
the rule (FR-1, FR-6).

Rejected: **a `State()` returning an enum** — a fourth value to keep exhaustive, and a switch at every
reader instead of the question that reader asks. Rejected: **an unexported derivation with a flag on
the far side** — the render tier would own a copy of the rule, C-1's hazard one tier out.

### DD-2 — the numbers sit at the record's tail; the version becomes 5

`entityLen` becomes 34, `HP` at record offset +26 and `MaxHP` at +30, little-endian like every other
field; `formatVersion` becomes 5. No range check: every int32 pair is constructible, so refusing one
would build a world the constructor can make and the decoder cannot read (FR-2).

Rejected: **inserting the fields before the presence byte** — two offsets moved for nothing.
Rejected: **reading a version-4 form as "health absent"** — "no health" is a claim about the
simulation, not a gap for a reader to fill, and 0045 refused the same trade.

### DD-3 — the constructor normalises the order; the decoder refuses it

`NewWorld` clears target, stall and route for an entity that is not alive, as it already zeroes a
targetless one's residue; `UnmarshalBinary` refuses a form carrying any of the three on such a unit,
as it already refuses a stall on a targetless entity. The two therefore stay in the relation they are
in today: the constructor cannot produce what the decoder refuses (FR-2, P-2).

Rejected: **refusing in the constructor too** — a caller with a dead unit and a stale target has no
way to act on the error, and the residue rule already normalises rather than refuses. Rejected:
**accepting the residue** — two byte forms would then decode to one behaviour, and injectivity is
what the whole form is built on.

### DD-4 — a `Kind` field on `Command`, with the move-to as its zero value

`Command` gains `Kind uint8` with `KindMoveTo = 0`, `KindKill`, `KindDamage`; the damage amount rides
`X`, `Y` unused. Every command literal already written stays a move-to untouched, which keeps the
schedules, the queue and every existing test unedited (FR-3).

Rejected: **three command types behind an interface** — an allocation and a dynamic dispatch on the
determinism wall, for three arms. Rejected: **a second exported entry point beside `Step`** — a
second way to change a world, and the replay log would no longer be the whole of what happened.

### DD-5 — one place applies a command, one place clears an order

Phase 1 of `Step` becomes a switch over the kind, each arm reached only after `indexOfEntity`; the
kill and damage arms end in `clearOrder(i)` where the health has reached zero or below, the same call
the arrival and give-up paths make. An undefined kind falls through the switch and is ignored, as an
absent entity already is (FR-3).

Rejected: **clearing target, stall and route field by field in the new arms** — three sites to keep
in agreement with the form's refusals, where one call already is.

### DD-6 — the occupancy seed skips the dead, and nothing else moves

`routeScratch.occupy` counts only units that are not dead; `moved`, `enterable` and `terrainOpen` are
untouched, so a corpse is invisible to both relations by never having been counted. The move loop
gains one test: a unit that is not alive is skipped before its target is read (FR-4).

Rejected: **a second plane marking corpses** — a parallel structure to keep level with the walk, for
a fact the count expresses by omission. Rejected: **filtering inside `enterable`** — asked once per
neighbour per frontier cell, it would run millions of times a tick to answer what the seed answers
once.

### DD-7 — the seam carries the state and the two numbers

`ui.MapEntity` gains `Life uint8` (with `LifeAlive`, `LifeDowned`, `LifeDead` declared in `pkg/ui`)
and `HP, MaxHP int`. `entityDraws` fills `Life` from the sim's own predicates, so the classification
crosses the seam already made and `pkg/ui` still names no simulation type (FR-6).

Rejected: **the two numbers alone, the state derived in `pkg/ui`** — the rule would exist twice, in
the package that owns it and in the one that draws. Rejected: **the state alone** — the bar needs the
ratio, and a precomputed width puts glyph geometry in the tier with no cell size.

### DD-8 — one predicate for marks and orders; the hit tests skip the dead

`presentSelected` keeps an id only where the snapshot holds it **and** its entry is not dead, so the
marks, the right-press orders and the two new keys read one filter. `decide`'s tap and box branches
skip a dead entry at the hit test, where a corpse stops being selectable in the first place. Nothing
writes to the stored set (FR-7, C-4).

Rejected: **a fifth branch that prunes the set** — 0030 fixed the set as replaced by a tap or a
release and by nothing else, and a prune would make a tick a writer of front-end state.

### DD-9 — one new seam function; the chip's amount is the world side's

`ui.MapAffect func(entity uint32, kill bool)` joins the tick, order and cadence seams, bound to the
same `mapWorld` by the same line in `loadMap`. `mapWorld.affect` looks the entity up in its own world
and appends a kill, or a damage of `max(1, MaxHP/10)`, to the queue the orders use — so one advance
applies orders and blows together, in issue order (FR-8).

Rejected: **the amount crossing the seam** — the front-end would hold a damage rule and the tier
that owns the world would take a number it cannot check. Rejected: **two seam functions** — a further
loader return value to distinguish two scalars. Rejected: **widening `MapOrder`** — every call site
edited so a move can say it is not a move.

### DD-10 — the bar's geometry is the render tier's; it draws as two passes

`terrain.HealthBarRects(col, row, cols, rows, cellpx, hp, maxHP) (ground, fill image.Rectangle, ok)`
returns one cell's two rectangles, `ok` false where no bar is drawn; the fill's width is
`ground.Dx() * hp / maxHP` in integers, clamped into the ground. The viewer builds two pass slices
from it through the relief lift and camera transform the marks take, appended after the sprites and
before the selection rim (FR-9).

Rejected: **computing the fill width in `pkg/ui`** — glyph geometry lives in the render tier, whose
constant the cell size is. Rejected: **one pass with two colours** — a pass is a colour and a rect
list, so two colours are two passes or a wider pass type every other glyph carries unused.

### DD-11 — the spawn constant lives at the one spawn site

`mapload` declares `SpawnHP = 100` and `FromALM` sets both fields from it. `NewWorld` is given no
default, so every hand-built world keeps `0/0` and stays alive and immortal (FR-5, C-3).

Rejected: **defaulting in `NewWorld`** — every synthetic world in the suite changes state and digest,
and the pins move for a reason unrelated to this contract.

## Risks

- **R-1** Every pinned form and digest in `pkg/sim` moves with the version, and a pin recaptured from
  the new encoder would agree with whatever it did — the failure the pins exist to catch. SC-4 is the
  whole mitigation.
- **R-2** A downed unit blocks its cell for the life of the world, so a corridor of bodies is a wall
  no order dissolves. Disclosed in the contract, and the price of keeping the death arm's middle
  phase without inventing its timer.
- **R-3** Every world the suite builds by hand is immortal, so the state machine runs only where a
  test asks for it: a defect in the damage arm cannot surface in an unrelated suite the way a routing
  defect can.
- **R-4** K and L are unlabelled letter keys that destroy state with no confirmation and no undo,
  beside the cadence keys.

## Success criteria

- **SC-1** AC-1 holds over its eight pairs, each expectation written by hand, the three predicates
  asserted pairwise exclusive and jointly total on every pair (FR-1).
- **SC-2** AC-2 and AC-3 hold, the ladder asserted at every one of its eleven steps rather than at
  its ends, the cleared order checked as all three fields (FR-1, FR-3).
- **SC-3** AC-4 holds with each no-op on a case of its own, every one compared by **digest** against
  a step carrying no command — the undefined kind included (FR-3, P-3).
- **SC-4** AC-6 and AC-7 hold: the offset table extended and still a partition of the whole form;
  `pinBytes` re-transcribed by hand at the new length; `pinDigest` and the three derived digests each
  **recomputed outside this tree** from their own hand-written bytes and cross-checked through the
  in-tree second FNV, none carried over by arithmetic; a whole well-formed version-4 world refused
  beside the 256-value sweep (FR-2).
- **SC-5** AC-5 holds, the corpse's cell and the downed unit's each tested in both directions, and
  the two immobile units driven for several ticks rather than one (FR-4).
- **SC-6** AC-8 holds over a corpus walked to quiescence under a stream mixing all three kinds, the
  second world compared at **every** later tick, not only the last (FR-2, FR-10, P-4).
- **SC-7** AC-9 and AC-10 hold, the map-built world read through the exported copy and the snapshot
  compared entry by entry (FR-5, FR-6).
- **SC-8** AC-11 and AC-12 hold, the selection compared **whole** before and after K and after L, the
  bar checked at both ends and in the middle, its relief offset compared as a point (FR-7, FR-8,
  FR-9).
- **SC-9** Five mutants, each applied to production code, run over the whole tree with the failing
  tests named, and reverted: the death test widened to health at or below zero, collapsing downed into
  dead; the seed skipping downed units as well as dead ones; damage clamped at zero; the version byte
  left at 4; the dead filter applied to the marks but not to the tap. Each is a design decision
  expressed as a defect — DD-1, DD-6, DD-4, DD-2, DD-8 — so a survivor is a criterion that does not
  discriminate its own decision, and is reported as one.

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-1 | SC-1, SC-2, SC-9 |
| FR-2 | DD-2, DD-3 | SC-4, SC-6, SC-9 |
| FR-3 | DD-4, DD-5 | SC-2, SC-3, SC-9 |
| FR-4 | DD-6 | SC-5, SC-9 |
| FR-5 | DD-11 | SC-7 |
| FR-6 | DD-1, DD-7 | SC-7 |
| FR-7 | DD-8 | SC-8, SC-9 |
| FR-8 | DD-9 | SC-8 |
| FR-9 | DD-10 | SC-8 |
| FR-10 | DD-7, DD-9 | SC-6 |
