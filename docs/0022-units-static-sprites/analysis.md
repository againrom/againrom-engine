# Analysis — units as static sprites

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** — the class id enters the hashed, serialized sim state, i.e. the save-format contract every later entity story builds on; no watcher tool exists, so static + discipline |
| Terrain — `pkg/sim` | **brownfield**: the shipped byte form, digest and pin change shape — the deliberate encoding change 0019's pin exists to make loud |
| Terrain — `pkg/game`, `pkg/ui` | **brownfield**: a shipped load path, tick seam and entity pass, all with behaviour to preserve |
| Terrain — the unit-art bundle and sprite pass | **greenfield** |

## The baseline is written against a build that is not this tree

Checked by grep over the module: **`openrom`, `MapScene`, `PickerScene`, `AttachSim`,
`SpawnsFromALM`, `NewWorldFromSpawns`, `SampleHeight8p8` and `render.ObjectAnchor` occur
nowhere.** What is here instead: `pkg/game`'s `FrontEnd` opens maps through the single
`LoadMapViewer` and owns the world in its unexported map-world holder (`world.go`), pushing plain
cells to the viewer after each step; the tick crosses to the window tier as the parameterless
`ui.MapTick`; `mapload.FromALM`/`Schedule` own the id and cell conventions; the object layer's
anchor is `terrain.StaticAnchor` and its bundle loader `game.LoadStatics`. Every baseline FR was
re-derived onto these seams; the removals are itemised in provenance.

## Both of the baseline's research items are closed — against it

- **Its R-1, the frame.** The baseline draws `Frames[class.Index]` and calls that its
  weakest-evidenced hypothesis. Decoded: a unit sheet is blocks whose bases are arithmetic on the
  class's `*Phases` scalars (`SPR256-UNIT-024`, exact on 33/34 sheets), the drawn frame comes
  from a nine-state animation switch (`TERR-SPR-047`), and `Flip` halves the sheet — 16 stored
  standing frames and 8 directions against 9 and 5 (`REG-UNITS-051`). For a static story the
  honest frame is the standing block's, at one disclosed facing; direction index 0 lands on sheet
  frame 0 under either layout and leaves the mirror path unexercised. `Index` enters the decoded
  unit path nowhere.
- **Its R-2, the anchor.** Decoded outright: `TERR-SPR-040`'s formula — already shipped as
  `terrain.StaticAnchor` since 0017 — taken at the drawn frame's size (`TERR-SPR-043`), and the
  unit draw is the same anchor from the unit's own position (`TERR-SPR-048`). What did NOT come
  out in the baseline's favour: that draw ignores its `(col, row, alt)` arguments, so the engine
  applies no terrain lift to a unit at the point of drawing it.

## The vertical treatment is a decision, and it is ours

0015's revision (`TERR-SPR-041` retracted, `TERR-SPR-048` published) leaves this story a fork:
the unit cross is lifted by the cell mean (ours, 0015), object art by the same mean (decoded,
0017), and the engine's unit draw by nothing — its vertical comes from unit fields whose
maintenance is open. A sprite drawn unlifted would stand exactly one cell-mean below its own
cross on every sloped cell: the instrument would read a permanent, known disagreement, and a real
placement bug would have to be seen through it. Lifted by the same mean, sprite and cross
coincide at tick 0 through two disjoint code paths — the marker family's vertical shift against
the anchor's lift argument — the same discriminating agreement 0017 runs for object art. So: the
cell mean, disclosed as ours, claiming nothing about the original's unit heights; and 0015's
downstream warning is carried forward — an entity that later gains a pixel position must carry
its height in exactly one place, or the lift is counted twice.

## What the entity carries, and who judges it

The baseline resolves the record's key through the registry in the map loader and stores a
validated id or a sentinel. Re-derived, the raw stored key wins: validation would hand `FromALM`
the registry, making a world's digest — hashed, serialized state — depend on registry contents
and giving 0020's one-decode doctrine a second input; the raw key keeps the loader's signature
and keeps a world a function of its map. It loses nothing — the key needs no translation
(`REG-KEY-044`: the class array is `ID`-keyed), and the draw side must miss cleanly anyway, since
a class can resolve and still ship no art.

**The AMBER split.** The class id enters hashed sim state, so its path is held to High — and it
is: the 70-byte read map and the `+0x08` role (`ALM-UNIT-040`, `ALM-CLS-038`), and the
`ID`-keying (`REG-KEY-044`). The render side consumes Medium-and-up: the 33/34 corpus figure and
the animation-state names are Medium, and the lift is ours by choice. The threshold holds.

## The pin is the mechanism, not an obstacle

0019 pinned the version-1 bytes and digest precisely so an encoding change fails a test instead
of silently breaking a save. This story is that change, taken deliberately: version 2, the pin
re-pinned, version 1 refused — 0019's one-version rule and its no-migration scope stand. 0019
FR-7's field enumeration is superseded for the current form by this story's contract; its rules
(versioned, fixed-width, ascending ids, refuse-and-leave-unmutated) are unchanged.

## Noted against `pkg/data`

`REG-UNITS-049` now decodes `units.reg`'s absent-everywhere defaults (−1, eight keys 0,
`TileSize` 1); 0016's `pkg/data` resolves such keys to the type's zero — undecoded at the time
and recorded as such there. Of this story's keys only `Flip` could differ, and both readings give
0; whether any shipped class reaches an absent-everywhere geometry key is not established. Filed
as a possible 0016 revision rather than silently absorbed.

## What we looked at

`pkg/sim` (world, step, binary, hash, the pin test), `pkg/mapload`, `pkg/game` (frontend,
mapload, world, statics), `pkg/ui` (viewer, overlay, statics, flow, app), `pkg/render/terrain`
(statics, overlay), `pkg/data` (classes, sprite, load), `internal/archtest`;
`docs/0015-height-projected-placements` (the revision), 0016, 0017, 0019 and 0020; in research at
`778c2a6`: `claims/retracted.md` first, then `terrain.md` (`TERR-SPR-038…048`), `spr256.md`
(`SPR256-UNIT-024`, `SPR256-FRAME-023`), `reg.md` (`REG-UNITS-049…051`, `REG-KEY-044`/`045`,
`REG-VAL-029`, `REG-ROSTER-052`), `alm.md` (`ALM-UNIT-040`/`048`, `ALM-CLS-037`/`038`); and the
staged baseline, read as a hypothesis set.

Open rather than guessed: the definition database and the record's override paths; the engine's
unit-position bookkeeping and its `+0x68` term; the compass meaning of the standing block's frame
0; the type-6 `+0x1c..+0x2b` roles; sprite lighting. Each is disclosed in provenance with what
would settle it.
