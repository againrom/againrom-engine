# Spec — the deterministic walking skeleton

## Problem and current behaviour

The engine decodes and renders maps, but nothing *runs* one. `pkg/formats/alm` decodes a map's
placed units into `Map.Units`, each with a `u32` fixed-point `X`/`Y` in 1/256 cell, and the viewers
draw a marker on each. `pkg/sim` and `pkg/mapload` hold a `doc.go` apiece and no code: no runtime
world, no tick, no command path, no save, no replay.

`docs/ARCHITECTURE.md` makes the **determinism wall** load-bearing from day one —
`Step(state, commands) -> state`, integer math, a seeded RNG, no IO, clock or floats — with commands
as the only write path and fresh state tiers beside it. Only its *structural* half stands: `internal/archtest`
holds `pkg/sim` to the standard library and no other tier, tests included, and a landed case of it
records that `os`, `time`, `math/rand` and floating point lie outside what that catches.

Boundaries like these are untestable to retrofit once breadth exists. This story stands the headless
core up in `pkg/sim` and proves the wall on one entity, the transform from a decoded map living in
`pkg/mapload` — the tier the DAG created for that join. Entity behaviour is deliberately trivial: a
move order advances an entity one cell per tick.

## Functional requirements

A **world** is the canonical runtime state: a monotonic tick, one seeded integer RNG, a set of
**entities** — each with a stable **entity id**, an integer cell position and an optional move target
— and the map's cell bounds. A **command** is a serializable value and the only way to change a
world; this story defines one, **move-to** `(entity, x, y)`. A **step** advances it one tick.

- **FR-1** `pkg/sim` MUST define the world and its entities as **fresh types** — never an extended
  `alm` or `reg` record, and holding no reference to one — ordered by ascending id wherever entity
  order is observable, independent of insertion order. A world MUST expose entity state for reading
  through a path that admits no mutation.
- **FR-2** A step over a command slice MUST be the ONLY exported operation that **advances** a world:
  no exported call may set a position, the tick, the bounds or the RNG **individually**. Replacing a
  whole world from its own canonical byte form is FR-7's inverse, not a field write, and is the only
  other exported way a world's contents may change.
- **FR-3** A step MUST perform, in order: (a) apply each command in slice order — a move-to sets the
  named entity's target, is ignored when no such entity exists, and a later command for the same
  entity overwrites an earlier one; (b) advance every entity **in ascending entity id** one cell
  toward its target, each axis by `sign(target - pos)`, clearing the target on arrival; (c) increment
  the tick. All arithmetic MUST be integer. It MUST mutate the world in place;
  `state, commands -> state` is semantics, not a value-copy return: identical prior state and
  commands yield an identical next state.
- **FR-4** A step MUST NOT constrain movement to the bounds: a target may name any cell and an entity
  may walk off the grid. Bounds are recorded state — hashed and serialized — and not yet a movement
  constraint.
- **FR-5** A world MUST own exactly one integer RNG whose state is part of its canonical state,
  round-trips through the byte form, and follows from the seed alone. The seed MUST be a
  deterministic input — explicit at construction, a fixed named constant in the loader — and MUST NOT
  come from the clock or any other nondeterministic source. Nothing in this story consumes it.
- **FR-6** A world MUST expose a deterministic 64-bit digest of its full canonical state, and every
  field the byte form carries MUST enter it, so that worlds with identical byte forms hash equal.
- **FR-7** A world MUST marshal to a versioned, self-contained byte form and back, the round-trip
  reproducing an identical digest and identical subsequent stepping. The form MUST be a
  format-version byte, then a fixed-width little-endian encoding of the tick, the RNG state, the
  bounds, the entity count and each entity — id, position, target — ids strictly ascending.
  Unmarshalling MUST return an error and leave the receiver unmutated on bytes that are truncated,
  over-long, carry an unknown version, or decode to non-ascending or duplicate ids. For a fixed seed
  and construction the bytes and the digest MUST be pinned, so a change to either fails a check
  rather than silently breaking a save.
- **FR-8** A run MUST record **one frame per advanced tick** — the tick and the commands applied at
  it, the list empty where none were scheduled — so the frame count equals the ticks advanced. A
  replay MUST apply a log's frames in order, checking each frame's tick against the world's current
  tick, reproducing a run's final digest from that run's initial world and returning an error with
  nothing further applied on a mismatch.
- **FR-9** A runner MUST advance a world N ticks against a per-tick command schedule and return the
  frame log, touching no GPU, no file and no clock.
- **FR-10** `pkg/sim`'s non-test sources MUST NOT import `os`, `time` or `math/rand`, and MUST NOT
  declare or use a floating-point type or literal; the test suite MUST check this mechanically. The
  architecture documentation MUST then record the determinism wall as behaviourally enforced instead
  of deferred.
- **FR-11** `pkg/mapload` MUST provide a transform building a world from a decoded map: one entity
  per placed unit, at the cell its fixed-point position names shifted right by 8 (the 1/256 fraction
  dropped); ids from the map's unit-slice order, the i-th unit taking id `i` from zero; bounds from
  the map's width and height; the RNG from a fixed constant seed; sharing no memory with the map.
  `pkg/sim` MUST NOT import it or any other `againrom` package.
- **FR-12** Every requirement above MUST be verifiable headlessly, with no game install, no GPU and
  no wall-clock read.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | one entity and a move-to `d = max(abs(dx), abs(dy))` cells away — straight, diagonal, and one target outside the bounds | stepped repeatedly | one cell per tick, each axis by its sign; arrival after exactly `d` ticks, then still, target cleared; the out-of-bounds target reached like any other |
| AC-2 | unit | a move-to naming an absent entity, and two move-tos for one entity in one slice | stepped | the absent-entity command changes nothing; the later wins; no exported call but the step or an unmarshal alters a position, the tick, the bounds or the RNG |
| AC-3 | unit | the same two entities inserted in opposite orders, same targets | each world stepped | both step identically and their digests are equal at every tick |
| AC-4 | unit | an initial world and a fixed per-tick schedule | run twice, independently | both runs reach the same final digest |
| AC-5 | unit | a run to tick N | marshalled at tick `k < N`, unmarshalled into a fresh world, continued to N | its digest at `k` is the original's, re-marshalling yields byte-identical bytes, and its final digest equals the uninterrupted run's |
| AC-6 | unit | a world | one field changed at a time: tick, an entity's X, its Y, its target, the RNG state, the bounds, an added entity | each change yields a digest differing from the unchanged world's |
| AC-7 | unit | `pkg/sim`'s non-test sources | scanned mechanically | none imports `os`, `time` or `math/rand`, and none declares or uses a floating-point type or literal |
| AC-8 | unit | a synthetic decoded map: N units at known fixed-point positions, one with a low byte other than `0x80`, and a known width and height | the loader runs, twice on the same map | N entities at the shifted cells, ids `0..N-1` in slice order, identical across both loads, bounds from the map; stepping either world leaves the map unchanged |
| AC-9 | unit | byte forms truncated, over-long, carrying an unknown version byte, or decoding to non-ascending or duplicate ids | unmarshalled onto a populated receiver | each returns an error and leaves the receiver exactly as it was |
| AC-10 | unit | a run that recorded frames | replayed from that run's initial world | the frame count equals the ticks advanced and the replay reaches the run's final digest |
| AC-11 | unit | a recorded log and a world whose tick does not match the next frame's | replayed | it returns an error, with that frame and every later one unapplied |
| AC-12 | unit | a world built from a fixed seed with a fixed set of entities | marshalled and hashed | the bytes and the digest equal the values pinned for them, so a changed encoding or generator fails here instead of breaking a save silently |

Error cases: a command naming an absent entity (AC-2, P-5); a byte form the contract refuses
(AC-9, P-6); a frame-tick mismatch on replay (AC-11, P-7).

## Derived properties

- **P-1** (invariant) A step is a pure function of the prior state and the command slice: the same
  inputs yield the same next digest, and no clock, file or floating-point value influences it.
- **P-2** (idempotence) Unmarshalling a marshalled world yields a world with the original's digest,
  and marshalling that again yields identical bytes.
- **P-3** (completeness) Replaying every frame of a log from the run's initial world reconstructs the
  run's final state exactly — no command dropped, added or reordered, one frame per advanced tick.
- **P-4** (invariant) The digest and the byte form depend only on the logical world — tick, RNG,
  entities by id, bounds — never on insertion or storage order.
- **P-5** (negative-invariant) No exported call mutates a world except a step; a command naming an
  absent entity changes nothing; and no simulation value holds a reference into a decoded map, so
  stepping never changes the map it loaded from.
- **P-6** (negative-invariant) For any byte form the contract refuses, the receiver is left exactly
  as it was — no field, entity or partial decode reaches it.
- **P-7** (negative-invariant) For any frame whose tick does not match the world's, that frame and
  every later one leave the world untouched.

## I/O examples

```text
w := mapload.FromALM(m)          // one entity per placed unit, at its cell
log := sim.Run(w, schedule, 100) // 100 ticks; schedule[t] = commands submitted at tick t

w2 := mapload.FromALM(m)
sim.Replay(w2, log)              // w2.Hash() == w.Hash()

b, _ := w.MarshalBinary()
var w3 sim.World
w3.UnmarshalBinary(b)            // w3.Hash() == w.Hash()

byte form: [version][tick][rng][bounds][count] then, ids ascending: [id][x][y][target] x count
```

## Constraints and alternatives

| Choice (all selected; *disclosed* = a named fidelity gap) | Observable trade-off |
|---|---|
| The loader lives in `pkg/mapload`, never in `pkg/sim` | the core cannot name a map type at all, so the tier boundary is structural rather than a convention; one more package in the load path |
| One cell per tick on integer cells, the 1/256 fraction dropped: no speed, path, collision or clamp — *disclosed* | an entity snaps to its cell on load, then crosses walls, water and the map edge; no fidelity to the original's movement is claimed |
| A seeded RNG owned but never consumed — *disclosed* | dead state in the digest and the byte form, against fixing the single-owner pattern while it is cheap |
| An explicit fixed-width byte form and our own digest, not `gob` or reflection | more code and a version byte to maintain, against a save that changes shape only when the contract says so |
| Entity ids from the map's unit-slice order | reproducible for a given map, but not an identity the map records carry, so a world cannot be matched against one |

## Out of scope

- Rendering a running world, and interpolation between ticks.
- Pathfinding, collision, movement speed, sub-cell motion, facing and animation.
- Combat, spells, items, experience and triggers; typed unit rules from `pkg/data`; any formula
  recovered from the original.
- Networking and transport: the command and frame shapes are proven here, the wire is later.
- Any unit identity a map record carries of its own.
- Migrating an older byte form (one version is defined and every other refused), save containers on
  disk, compression, and more than one world at a time.

## Verification mapping

AC-1…AC-12 and P-1…P-7 are CI-automatable headlessly against synthetic worlds and maps — no game
install, GPU or window — and AC-7 is a mechanical source scan.

Gate coverage: FR-1→AC-3/AC-8/P-4; FR-2→AC-2/P-5; FR-3→AC-1/AC-2/AC-4/P-1; FR-4→AC-1/P-1;
FR-5→AC-6/AC-12/P-4; FR-6→AC-6/AC-12/P-4; FR-7→AC-5/AC-9/P-2/P-6; FR-8→AC-10/AC-11/P-3/P-7;
FR-9→AC-4/AC-10/P-1; FR-10→AC-7/P-1; FR-11→AC-8/P-5; FR-12→AC-1…AC-12.
