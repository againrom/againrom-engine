# Story 1065 — multi-cell actor occupancy

## Player result

Actors with a `TokenSize` of two or three now occupy every cell of their square
footprint. They cannot walk or Teleport through another actor by overlapping a
non-anchor cell on the same occupancy layer, and a large actor cannot move,
Teleport or return from off-map with part of its destination footprint on
blocked terrain or outside the map. A moving flyer keeps the decoded soft
crossing rule, but it may become a resting, counted actor only where its whole
footprint fits; a refused stop keeps its movement and pursuit intact.

## Authority and scope

`TERR-FOOTPRINT-147` establishes all-or-nothing registration across an actor's
`n x n` footprint. `TERR-PASS-051` establishes the block-mask test across every
covered cell. `TERR-MOVE-057` establishes that shipped `TokenSize` values two
and three reach simulation actors. `MOVE-AREA-038` establishes that movement
reads the block planes and does not re-read cell-record occupant slots.

This story closes `DIV-054`. It does not restore the one-actor cell-record cap
(`DIV-055`), add area-effect movement cost (`DIV-021`), or change stacked
effect recomputation (`DIV-053`). It adds no canonical state and changes no
save-form layout.

## As built

- The route scratch counts every in-bounds cell of each counted actor's square
  footprint on the actor's movement layer.
- Terrain-only and unit-aware route admission validate every destination cell.
  Unit-aware admission subtracts the mover's old overlapping cells, so an
  adjacent large-actor step remains possible. A same-version save accepted by
  an earlier build remains loadable when its stored route tested only the
  anchor; the route is retired and rebuilt under the complete predicate at its
  first use. A Wall of Earth invalidates a live stored route when it closes any
  covered cell rather than only an anchor.
- A committed move removes the complete old footprint and adds the complete new
  footprint before the next actor resolves. Starting a large-flyer pursuit
  removes its complete resting footprint. When a moving flyer tries to rest,
  the same complete terrain and occupancy fit runs before its order or plane is
  changed; refusal leaves it moving, while success registers every cell.
- Script return placement and both Teleport forms use the same complete terrain
  and same-layer occupancy fit. Admission and release both reject a bad
  destination before mana, recovery, route or position changes.
- Ground and ghost actors continue to share one occupancy layer. Air actors use
  the separate decoded layer, so a ground and air footprint may overlap.

## Proof

`pkg/sim/footprint1065_test.go` covers a blocked far cell, an occupied far cell,
old/new footprint overlap during an adjacent step, complete occupancy
replacement, a large flyer's complete rest footprint, script placement,
Teleport admission and release-time occupancy revalidation before mana, and a
same-version legacy route which still loads byte-for-byte and replans before it
can cross a blocked non-anchor cell, plus dynamic wall invalidation. The
ordinary-pursuit regression reproduces a moving 2x2 flyer reaching attack range
inside a resting flyer: it refuses that stop, closes one more cell, attacks from
a valid resting footprint, and round-trips with the same hash and no resting
same-layer overlap. The generated indexed-versus-linear oracle now ranges over
domains and footprint sides zero through three. Existing area-overlap tests
retain total behaviour for an already-overlapping decoded or synthetic world,
which remains `DIV-055` rather than part of this story.

Verification is the full Go, no-asset, divergence, paired EN/RU release and
full-scenario chain plus the movement milestone census. The exact pushed-SHA
review report records those gate results and the census comparison.
