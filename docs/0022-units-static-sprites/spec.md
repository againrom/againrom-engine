# Spec — units as static sprites

## Problem and current behaviour

A world runs under the map screen — one entity per placed unit, stepped once per front-end tick
against a scripted schedule — and every entity draws as a 10-pixel magenta square on its cell,
under the diagnostic crosses. An entity carries an id, a cell and a target and nothing that says
WHAT it is, so nothing can pick its art. The unit registry already loads into typed classes and
the static-object layer already draws real `.256` sprites through a decoded anchor; nothing joins
a running entity to either.

This story makes the join: an entity gains the class key its unit record stores and draws as a
static sprite standing on the terrain, moving cell to cell with the world. An entity resolving to
no drawable art keeps its square; the squares and crosses stay the instrument the sprites are
checked against.

## The contract

**Class id.** An opaque signed 32-bit integer: the class key the map's unit record stores — a
unit-registry class `ID`, sign-extended from the record's 16-bit field — used DIRECTLY, the
registry being keyed by `ID`. The domain is sparse — 34 classes over 1..80 — so every consumer
MUST miss cleanly on an id naming no class.

**Sheet and frame.** The sheet is the `.256` entry `units/` + `Files[class.File]` + `.256` of the
graphics archive, decoded with the palette it carries (`File` may be inherited). It opens with
the STANDING BLOCK: 16 stored standing frames when the class's `Flip` is 0, 9 when it is not (the
missing seven mirrored at draw). The drawn frame is the standing frame for one fixed facing —
direction index 0, this story's constant — sheet frame 0 on either layout, read by the class's
RESOLVED `Flip` even where that mismatches the sheet. `Index` selects nothing here. A class whose
sheet is absent, undecodable or palette-less, or holds no such frame, has no drawable art — a
skip, never an error.

**Ground point and anchor.** An entity's ground point is its cell's centre
`(col*32 + 16, row*32 + 16)`; the sprite's top-left is the object layer's own anchor rule, at the
DRAWN frame's size — frames of one unit sheet need not share a size:

```
anchorX = (CenterX - Width/2)  + frameW/2      every /2 truncating toward zero
anchorY = (CenterY - Height/2) + frameH/2
destX   = col*32 + 16 - anchorX
destY   = row*32 + 16 - anchorY - lift - originY
```

`Width`/`Height` are the class's canvas and `(CenterX, CenterY)` its ground-touching pixel;
`lift` is the cell's four-corner mean height in displaced mode and 0 in flat — the marker
family's own lookup and sign — and `originY` the render's native vertical origin. Transparency is
structural, per decoded pixel; an opaque pixel draws fully opaque in the sheet's own palette
colour, nearest-sampled, scaled by the zoom.

**Layer order.** Terrain, the static-object art, the ENTITY LAYER, then the three diagnostic
overlays in their existing order. Inside the entity layer: sprites in ascending entity id — a
later id owns an overlap — then squares in ascending id.

## Functional requirements

- **FR-1 — the class id is canonical sim state.** Each entity MUST carry its class id, set only
  at construction or by unmarshalling; a step MUST NOT read, change or branch on it. It MUST
  enter the digest and the byte form, which takes FORMAT VERSION 2 — the I/O examples' layout,
  fixed-width little-endian, ids strictly ascending. Every other version byte — the shipped 1
  included — is refused with the receiver unmutated. The fixed-construction bytes and digest MUST
  be pinned anew. `pkg/sim` MUST gain no import and no exported call.

- **FR-2 — the loader carries the key and judges nothing.** The map-to-world transform MUST set
  entity i's class id to the class key unit record i stores, sign-extended, changing nothing else
  — ids, cells, bounds, seed, no shared memory. It MUST NOT consult the registry — resolution is
  the render side's question — so a world stays a function of the decoded map alone; the schedule
  beside it is untouched.

- **FR-3 — a headless unit-art bundle.** `pkg/game` MUST load the unit classes and their drawn
  frames from the graphics archive into an in-memory bundle, GPU-free: the registry parsed and
  resolved (an unreadable or unparseable registry is the ONLY error), each sheet decoded at most
  once, the frame per the contract. Entries are keyed by class `ID` and carry the class canvas
  and the frame; an excluded class keeps a frameless entry, distinguishable from an id naming no
  class. The bundle's types MUST live in the render tier.

- **FR-4 — sprite or square, never nothing.** For every entity of the state the latest step
  produced — never from the map's unit records — the map screen MUST draw exactly ONE of: its
  sprite, when its class id resolves to a drawable frame, placed per the contract at the entity's
  own cell; or the existing square, unchanged in glyph, size and colour. An off-map entity draws
  neither, as before. A sprite is culled on its EXACT placed world rectangle, a square by the
  marker family's own cull; both take the camera the terrain uses. Sprite textures reach the GPU
  lazily on first draw, never at construction.

- **FR-5 — the cross is the instrument.** At tick 0 a resolved entity's sprite ground point —
  top-left plus its own anchor — MUST land on the world point its unit's diagnostic cross is
  centred on; the two MUST be derived independently — the sprite from the entity's cell through
  the bundle's class geometry and the anchor rule, the cross from the map record through the
  marker geometry, no class field, no frame size — sharing only the camera transform, so a wrong
  placement shows as art standing away from its own cross.

- **FR-6 — the walls stand; the world is the same rendered or not.** `pkg/sim`'s sources stay
  standard-library-only with no floating point, its source scan green and its exported method set
  unchanged. The window tier MUST NOT name a simulation, format or data type — it receives, per
  entity, a plain cell and a render-tier value or none. The standalone map viewer MUST own no
  world and change in nothing; no boundary the import check enforces is widened. Driving the map
  screen k ticks MUST leave the world's digest equal to a headless run's after those k ticks,
  class ids included; no draw, resolution or bundle load may reach or change a world.

- **FR-7 — wired at the front-end, total by default.** The game MUST load the bundle once at
  startup beside the object bundle, headlessly under `-check`; startup MUST fail when the unit
  registry cannot be read or parsed. Every opened map draws unit sprites unconditionally —
  content, not a diagnostic, no flag. A front-end without a bundle draws every entity as the
  square: byte-for-byte the screen this story inherits.

- **FR-8 — headless.** Everything but the paint itself MUST be verifiable over synthetic
  registries, sheets, maps and worlds — no game install, no GPU, no window.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a world with distinct class ids, one negative; the fixed pin construction | marshalled, unmarshalled fresh, re-marshalled, hashed | equal digest, class ids exact, re-marshal byte-identical; one changed class id changes the digest; the pin's bytes and digest equal the version-2 pinned values |
| AC-2 | unit | byte forms at version 1 and 3, truncated, over-long | unmarshalled onto a populated receiver | each refused with an error, the receiver exactly as it was |
| AC-3 | unit | a synthetic map with assorted class keys — one negative, one naming no class | the loader runs | keys in slice order, sign-extended; cells, ids, bounds, seed as before; stepping leaves the map unchanged |
| AC-4 | unit | a synthetic registry: drawable classes at both `Flip` values; absent, undecodable, palette-less and empty sheets; an unreadable registry | the bundle loads | drawable classes keyed by `ID` with canvas and the contract's frame; excluded classes keep frameless entries, distinct from a missing id; only the unreadable registry errors; no GPU |
| AC-5 | unit | a class whose drawn frame differs from its canvas, a lift, an originY | the sprite is placed | top-left per the anchor rule at the drawn frame's size; displaced minus flat is exactly `-(lift + originY)` in Y, zero in X |
| AC-6 | unit | a snapshot of resolved, unresolved and off-map entities, part of the map in view | the entity layer is built | one sprite per resolved, one square per unresolved, nothing off-map; sprites then squares, each ascending id; the cull keeps a crown-only sprite; nothing added, dropped or reordered |
| AC-7 | unit | a viewer with overlays and an entity layer; one handed neither; the standalone viewer | draw passes read | the entity layer sits after the static art, before the three overlays; squares keep glyph and colour; the bare and standalone viewers are unchanged, owning no world |
| AC-8 | unit | a synthetic map whose units resolve, both geometries, assorted cameras | tick-0 sprite ground points vs unit-cross anchor points | equal for every resolved entity |
| AC-9 | unit | a map and its schedule | the map screen driven k ticks, and a headless run | equal digests; the sim source scan passes |
| AC-10 | integration | install layouts with and without a readable unit registry | `-check` runs headlessly | the first loads the bundle, building no texture; the second fails startup naming the registry |
| AC-11 | corpus | every shipped ROM1 map, a lawful install's bundle | a world built per map, every entity resolved | per-map counts recorded — sprite, id naming no class, class without art — no error |
| AC-12 | manual | a lawful install, representative maps | opened, panned, zoomed, run | each resolving unit is its own sprite under its own cross at tick 0 and walks with the schedule; unresolved keep squares; only on-screen sprites draw; limitations named |

Error cases: a refused byte form (AC-2); an unreadable or unparseable unit registry (AC-4,
AC-10) — everything else falls back to the square, never an error (AC-4, AC-6).

## Derived properties

- **P-1** (invariant) The class id is inert: stepping and every other field of the stepped state
  are independent of every class id; no exported call sets one individually.
- **P-2** (completeness) Every in-map entity contributes exactly one drawn item — sprite or
  square, never both, never none; off-map is the only other case.
- **P-3** (negative-invariant) With no bundle, no sprite and no texture exists anywhere; every
  screen is the pre-story one, byte for byte.

## I/O examples

```text
byte form v2: [2][tick u64][rng u64][boundsW i32][boundsH i32][count u32]
              then per entity: [id u32][x][y][targetX][targetY][class i32][hasTarget u8]
```

## Constraints and alternatives

| Choice (all selected; *disclosed* = a named fidelity gap) | Observable trade-off |
|---|---|
| The class id is the record's stored key, raw; no registry enters the loader | a world and its digest stay a function of the map alone — a registry edit changes what DRAWS, never what a save hashes to |
| Displaced sprites carry the markers' cell-mean lift — *disclosed* | sprite and cross coincide at tick 0 — the instrument; the original draws a unit from its own maintained position and reads no terrain height there, so nothing about its heights is claimed |
| One fixed facing, direction index 0 — *disclosed* | every unit faces one way, the mirror path unexercised; no facing source is decoded in the map record |
| Sprites in ascending id, then squares, all under the crosses; a later id owns an overlap — *disclosed* | content stays under the instruments, no sprite covers a square-only entity, and no depth rule of the original is claimed |

## Out of scope

- Animation, facing from unit state, the mirrored half-sheet, every block past standing (move,
  attack, idle, dying, the corpse chain).
- Shadows (the sibling `b` sheets), palette remap, sprite lighting, selection, picking.
- The record's NPC-diversion flag and definition-id override: such records draw by their stored
  class key.
- Sub-cell position, the original's own unit draw-position rule and its jump-height term; depth
  interleave with object art.
- Any change to the standalone viewer, the schedule, the object layer, terrain, water, lighting
  or marker geometry; version-1 byte-form migration (refused, not converted).

## Verification mapping

AC-1…AC-10 and P-1…P-3 run headlessly — AC-10 over install layouts, the rest over synthetic
data. AC-11 needs a lawful install (counts recorded, no asset committed); AC-12 is a
developer-run visual check needing a window.

Gate coverage: FR-1→AC-1/AC-2/P-1; FR-2→AC-3/AC-9; FR-3→AC-4/AC-10/AC-11;
FR-4→AC-5/AC-6/AC-7/AC-12/P-2; FR-5→AC-8/AC-12; FR-6→AC-7/AC-9/P-1/P-3; FR-7→AC-10/AC-12/P-3;
FR-8→AC-1…AC-9.
