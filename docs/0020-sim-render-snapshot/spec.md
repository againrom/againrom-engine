# Spec — a running world under the map screen

## Problem and current behaviour

`pkg/sim` advances a world headlessly and `pkg/mapload` builds one from a decoded map, and nothing in
the program calls either: no `cmd/` binary reaches the simulation core, and the loader's only caller
is its own test. The game front-end opens a map and draws its terrain, the map's static-object art
and three diagnostic marker overlays; every marker it draws is a position read straight out of the
map's placement records. Nothing under the picture runs, and nothing says that putting a renderer
over a world leaves the world's behaviour unchanged.

The boundaries that constrain how it may be wired are already executable rather than conventional.
The package that opens a window may import the render tier and no other; `pkg/sim` may import no
`againrom` package at all; `*sim.World`'s exported method set is pinned, so a call added to it fails
a test until whoever added it has said what kind of call it is. A world already exposes its tick, its
bounds, and a read path that hands out copies.

## Functional requirements

An **entity** is one thing a world simulates, standing on one whole map cell. A **step** advances the
world one tick against a list of commands and is the only operation that changes it. A **schedule**
names the commands to apply at each tick.

- **FR-1 — one world per opened map.** Opening a map in the game MUST build a world from **the same
  decoded map the viewer draws** — one decode, so the world and the terrain on screen cannot
  describe different maps and no consistency check between them is needed. Leaving the map screen
  MUST discard that world with the viewer, and opening a map again MUST build a fresh one at tick 0.

- **FR-2 — one step per front-end tick, and nothing else writes.** While the map screen shows, one
  front-end tick MUST advance the open map's world exactly once, applying the schedule's entry for
  the world's tick as it stood before that step, and no commands at all past the schedule's end.
  Input, camera, zoom and the water phase MUST NOT change a world. *Disclosed exception:* the tick on
  which Esc leaves the map screen returns before the advance, so the world is not advanced on it —
  it is discarded on that same tick.

- **FR-3 — nowhere else.** No world MUST be advanced while the menu or the picker shows, and the
  standalone map viewer MUST NOT own or advance one at all.

- **FR-4 — the schedule is scripted and deterministic.** A schedule MUST be a function of the decoded
  map alone — never of a clock, a generator or user input — so two openings of one map advance
  identically. It is a placeholder that makes the skeleton visibly walk, not gameplay; it is finite,
  and once exhausted the world takes no further command.

  *Folded from hotfix `c5274d7` — see `docs/hotfix/ARCHIVE.md#c5274d7`.* The scripted schedule is
  test support only: no front-end path wires it, and both front-end paths are pinned against its
  absence. The wiring plan `DD-2`/`DD-3` describe is void.

- **FR-5 — draw the entities, never the records.** While a world is open the map screen MUST draw one
  marker per entity, at that entity's own cell, from the entity state the most recent step produced
  and never from the map's unit records. An entity whose cell lies outside the map's extent MUST
  simply not be drawn: a world permits an entity to walk off the grid, and that is not an error.

- **FR-6 — placed like every other marker.** An entity marker MUST be a filled square 10 pixels
  across at the native cell size, centred on its cell's centre. In displaced mode it MUST carry that
  cell's own terrain height lift and in flat mode none, by the same lift and origin the diagnostic
  markers use; it MUST then go through the same camera transform, and a marker whose placed rectangle
  does not meet the view MUST be dropped. Its colour MUST differ from all three diagnostic marker
  colours.

- **FR-7 — content under the instruments.** The map screen MUST draw, in order: terrain, the map's
  static-object art, the entity markers, then the placed-object, placed-unit and static-object
  diagnostic overlays in their existing order. Entities are the map's content, so no diagnostic may
  be drawn beneath one — a coincident diagnostic cross MUST stay readable over an entity marker.

- **FR-8 — an entity's cell and its record's marker cell agree.** At tick 0 the cell of the entity
  built from a placed unit MUST equal the cell that unit's diagnostic marker is drawn on. The two
  MUST be derived from the record independently and share only the transform from a cell to the
  screen, so a disagreement between them can only be a disagreement about the cell, and MUST show as
  an entity standing away from its own cross.

- **FR-9 — the wall stands where it is.** `pkg/sim` MUST gain no exported call and no import for this
  story; the package that opens a window MUST NOT name a simulation type; no tier boundary the import
  check enforces may be widened. The renderer MUST reach entity state only through the existing read
  path that hands out copies, and MUST NOT be able to change a world through it.

- **FR-10 — the same world, rendered or not.** Driving the front-end's map-screen tick k times over a
  map and its schedule MUST leave the world's digest equal to the digest a headless run of that same
  schedule, from a fresh world of that same map, reaches in k ticks.

- **FR-11 — headless.** Everything above but the paint itself — the tick loop, the entity-to-cell
  conversion, the placement, the lift and the cull — MUST be verifiable with no game install, no GPU
  and no window, over synthetic data.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a synthetic decoded map carrying N placed units, opened by the front-end | the map screen is reached | a world exists at tick 0, holding N entities, its bounds the map's width and height |
| AC-2 | unit | the front-end on the map screen with its world at tick t | one front-end tick, neutral input | the world stands at t+1 and exactly the schedule's entry for t was applied; the camera and the water phase still respond |
| AC-3 | unit | the front-end on the menu, on the picker, and on the map screen on the tick Esc is pressed | one front-end tick in each | no world advances in any of the three, and after the Esc tick no world exists |
| AC-4 | unit | a map opened, ticked, left by Esc and opened again | the second world read | it stands at tick 0 with the entities the first world started from |
| AC-5 | unit | a synthetic map and its schedule | the open map's world ticked k times, nothing drawn | its digest equals the final digest of a headless run of that schedule from a fresh world of that map over k ticks |
| AC-6 | unit | entity cells over a grid with a valid altitude layer and over one without | the entity layer placed | each marker is the 10-pixel square centred on its cell centre, carrying that cell's lift in displaced mode and none in flat; a marker whose rectangle does not meet the view is dropped; a cell outside the map yields none |
| AC-7 | unit | a viewer holding entity cells with all three diagnostic overlays on | the frame's draw passes read | the entity pass follows the static-object art and precedes the object, unit and static passes, in a colour none of the three uses |
| AC-8 | unit | a synthetic map whose units sit at known fixed-point positions, one of them with a low byte other than `0x80` | a world built, and the unit overlay's cells derived from the same map | entity i's cell equals unit i's marker cell, for every unit |
| AC-9 | unit | a picker row whose bytes fail to decode | chosen | the front-end stays in the picker with the reason shown and that row marked unusable; no world is built and none is advanced |
| AC-10 | unit | a viewer that was never given entity cells, and the standalone viewer | drawn and queried | no entity pass is produced and the prior flat, displaced, overlay and water behaviour is unchanged; the standalone viewer owns no world |
| AC-11 | manual | a lawful install | a campaign map opened in the built binary | every entity marker stands under the unit cross it was built from at tick 0, and the entities walk as the schedule runs |

Error cases: a chosen map that fails to decode (AC-9, P-4).

## Derived properties

- **P-1** (invariant) For any map and schedule, the world the front-end reaches after k ticks is the
  world a headless run reaches after k: a renderer's presence changes no simulated state, and no
  draw touches a world.
- **P-2** (negative-invariant) For any front-end tick that is not a map-screen tick — the menu, the
  picker, and the tick that leaves the map — no world advances.
- **P-3** (invariant) For any placed unit, the entity built from it and that unit's own diagnostic
  marker land on the same screen point at tick 0, in either terrain mode and at any camera position.
- **P-4** (negative-invariant) For any map that fails to load, no world is built, none is advanced,
  and the map screen is not reached.
- **P-5** (completeness) Every entity a world holds has exactly one marker before the cull, and no
  marker is drawn for an entity the world does not hold.
- **P-6** (negative-invariant) For any caller that gives a viewer no entity cells, no entity pass is
  produced and nothing that viewer drew before this story changes.

## I/O examples

```text
opening a map   world from the decoded map the viewer draws: one entity per placed unit, tick 0
each tick       step the world once, with the schedule's commands for the world's current tick
each frame      one 10-px square per entity cell, after the map's art, under all three crosses

k front-end ticks over (map, schedule)  ==  a headless run of (map, schedule) for k ticks
```

## Constraints and alternatives

| Choice (all selected; *disclosed* = a named fidelity gap) | Observable trade-off |
|---|---|
| The world lives in the tier that may see both the simulation and the window; the tier that opens a window names no simulation type | one more hop for the entity-to-cell conversion, against an import check that stays exactly as it is and a draw path that cannot reach a world at all |
| The renderer reads entities through the read path that already exists; no call is added to a world | one slice per advanced tick, against a second entity read path in the one package whose exported surface is small enough to pin, and whose no-mutation sweep is fail-closed on any method taking an argument |
| One step per front-end tick — *disclosed* | ground is crossed at the front-end's tick rate, which is not a rate the original is claimed to have; a logical tick rate of its own is a later slice |
| Entity markers are the map's content, drawn under all three diagnostic crosses | a coincident cross reads on top of its entity, which is the instrument; an entity alone reads as a block of its own colour. Drawn last instead, a filled square would erase a coincident cross rather than degrade it |
| The world is built from the viewer's own decoded map | the two cannot describe different maps, so the bounds agree by construction and there is no check for that to keep true |
| A placeholder square, not a unit sprite — *disclosed* | the skeleton is visible without asserting anything about how the original draws, anchors or faces a unit |

## Out of scope

- Unit sprites and class-driven art, anchoring, facing and animation.
- Click-to-select, player-issued orders, and any order source that is not the scripted schedule.
- Pathfinding, collision, movement speed, sub-cell motion and clamping to the bounds — unchanged
  from 0019, and still disclosed rather than implied.
- A logical tick rate distinct from the front-end's, and interpolation between ticks.
- Saving, loading or replaying a world from the front-end, and more than one world at a time.
- Any change to the standalone map viewer, and any entity layer in the offline raster tool.
- Sprite-anchoring fidelity and the original's own marker rounding: no research covers either, and
  nothing here asserts anything about them in either direction.

## Verification mapping

AC-1…AC-10 and P-1…P-6 are CI-automatable headlessly over synthetic maps, schedules and grids — no
game install, GPU or window. AC-11 is a live run of the built binary against a lawful install, and
the only criterion here that needs one.

Gate coverage: FR-1→AC-1/AC-4/AC-9; FR-2→AC-2/AC-5/P-1; FR-3→AC-3/AC-10/P-2; FR-4→AC-2/AC-5;
FR-5→AC-6/AC-11/P-5; FR-6→AC-6/P-3; FR-7→AC-7/AC-11; FR-8→AC-8/AC-11/P-3; FR-9→AC-10/P-1/P-6;
FR-10→AC-5/P-1; FR-11→AC-1…AC-10.
