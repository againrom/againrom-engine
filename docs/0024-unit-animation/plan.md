# Plan — unit animation (idle and move, with facing)

## Baseline

`pkg/data` resolves a registry in three passes; a scalar set nowhere on a chain takes its
registry's `scalarDefault` table row, via `defaultsFor` (`keys.go`). `objectDefaults` holds twelve
rows, `File` the one `noInherit`; `unitDefaults` and `structureDefaults` are empty, pinned so by
`defaults_test.go`'s keep-the-zero and bijection tests. `UnitClass` already carries every
animation key verbatim: three `AnimTime`/`AnimFrame` pairs validated to equal resolved length, the
six phase scalars, `Flip`.

`terrain.UnitClass` is `{Width, Height, CenterX, CenterY, Frame *StaticFrame}`; `UnitSet` a map by
class ID; `UnitPlace` is false on a nil or frameless class, else `StaticAnchor` at the frame's own
size, the one negation of `lift`/`originY`. `game.LoadUnits`: the registry is the only error, the
read error unwrapped; `sheetCache` decodes each distinct path once at the `spr256.Sprite` level,
but `frame(path, 0)` converts a fresh `*StaticFrame` per class. `mapWorld{world, sched, units,
view}`: `tick()` indexes the schedule by `w.Tick()`, steps, `push()`es `entityDraws` — a plain
cell beside `Art` where the entry holds a frame; no clock, no per-entity memory anywhere.
`Entities()` hands copies; `Hash`/`MarshalBinary` share one traversal. `UnitCensus` and
`terraintool units` count Sprites/NoClass/NoFrame.

`ui.MapEntity` is `{Cell, Art}`; `entityLayer` walks the adopted slice once — bounds test first,
`UnitPlace` ok to sprites else squares, the viewer's own displaced lift terms — and
`overlayPasses` prepends sprites then squares before the three diagnostics. `Draw` paints sprites
through `staticImage`, one lazy texture per `*StaticFrame` pointer, zero-area skipped;
`staticScreenRects` culls on the exact placed rect and copies the frame pointer across.
`internal/archtest`: the fail-closed DAG (`terrain` stdlib-only; `ui` unable to import sim,
formats or data) and the `pkg/sim` source scan. `game/world_test.go` already digests a driven map
screen against a headless run at every k. `internal/synth` builds `UnitsReg`, multi-frame
`Sheet256`, archives.

## Design decisions

### DD-1 — FR-1 is one table, not a loader change

`unitDefaults` fills with the whole scalar inventory at once: seventeen -1 rows (`ID`, `File`,
`AttackPhases`, `DyingPhases`, `MovePhases`, `MoveBeginPhases`, `BonePhases`, `Index`, the canvas
four, the selection four, `Parent`), the spec's eight decoded-0 rows, `TileSize` 1 — and
`InMapEditor` NO row, the objects table's `IconID` shape for a key the engine record lacks. No row
is `noInherit`: `File` keeps inheriting, and the sprite base reads the resolved node — nil when
set nowhere — so a class resolving no `File` resolves no path. `loadSet` is not edited —
`defaultsFor` already turns a table into fill values — so objects and structures cannot move:
their tables are untouched data. Two `defaults_test.go` tests change BY DESIGN: keep-the-zero
loses its units half (structures keep theirs); the bijection test pins the unit rows like the
object ones. Spill tripwires: `TestAKeyNoSectionSetsTakesItsDefault` and
`TestFileDoesNotInheritInObjectsButDoesInUnits` break if objects.reg semantics move either way;
every other suite must pass unedited — `game/units_test.go`'s classes set every canvas key, so no
bundle expectation shifts. Rejected: a units branch in the loader — defaults are data here;
rejected: `File` as `noInherit` like objects' — the spec states the opposite.

### DD-2 — the descriptor is data-tier arithmetic, mirrored as plain values

`data` gains `UnitAnim` and `(c *UnitClass).Anim()`, a pure integer function of one resolved
class. `(S, D)` is (16, 8) at resolved `Flip` 0, (9, 5) otherwise; the bases follow the
sheet-contract table — `MoveBase = S`, `AttackBase`, `DyingBase`, `TailBase = S + D*(MB+MV+AT+DY)`
serving bone and idle — the strides are the move slot `MB+MV` with wind-up offset `MB` and the
idle slot `ID`, and `Total` the spec's predicted total. The tracks expand by the spec's run-length
rule, order and duplicates preserved — `[0 1 2 1]` survives, a `Time <= 0` entry still consumes
its round. EVERY phase scalar is `max(x, 0)` before it enters a base, a stride or the total:
FR-1's absent-key `-1` is a sentinel, not a count, and consuming it as one subtracts a whole
direction block from every base after it — measured on the corpus, twelve classes short by
exactly `D`. Gates: `MoveOK` = `MV > 0` and a non-empty move track; `IdleOK` likewise over `ID`,
both read from the resolved scalar and unaffected by the clamp, which maps non-positive to 0.
`terrain` carries a field-for-field mirror type the loader fills (DD-4, the `StaticPixel`
precedent — that tier cannot import `data`). AC-2's expectations are hand-computed literals — the
spec's example class pins bases 9/24/34/44 and total 54 — never calls back into `Anim()`.
Rejected: deriving in `terrain` from copied raw scalars — registry semantics re-implemented in a
second tier; rejected: leaving the bases to the selector — the sheet contract in two places;
rejected: clamping in the selector instead — the mirror type would then carry two conventions,
and a class rejected at load would make a mismatch an error, which FR-5 reserves for the
registry.

### DD-3 — the selector is total, and owns the guard

`terrain.SelectUnitFrame(a UnitAnim, frameCount int, moving bool, oct, tick int) (frame int,
mirror bool)`. Moving under `MoveOK`: `MoveBase + slot*MoveSlot + MoveWind + MoveTrack[step]`,
mirrored per the direction rule; a mover failing the gate takes its idle selection; idle under
`IdleOK`: `TailBase + slot*IdleSlot + IdleTrack[step]`; otherwise the standing selection. `step`
is the tick reduced euclideanly mod the period, so ANY int is safe; the guard is the selector's
last act — an index outside `[0, frameCount)` answers `(0, false)`, sheet frame 0 unmirrored —
which also makes it total at `frameCount <= 0`, where the SEAM answers the square. The boundary
literals of the spec's two mirror rules are pinned HERE so no test inherits an off-by-one.
Standing at S 16: never mirrored, oct 5 → frame 10 plain. Standing at S 9: oct 4 → (8, plain), oct
5 → (6, m), oct 6 → (4, m), oct 7 → (2, m). Slots at D 8: oct 5 → 5, plain. Slots at D 5: oct 4 →
4 plain, oct 5 → 3 m, oct 6 → 2 m, oct 7 → 1 m. AC-3's index expectations are hand-computed
literals over these tables, the spec's worked example (16, mirror) among them; capturing one from
descriptor or selector is the tautology this DD bars. Rejected: guarding at the seam — FR-3 names
the guard as the selector's, and every caller would re-own it; rejected: a shared mirror lookup
table — one more copy of the arithmetic, free to drift.

### DD-4 — the loader carries whole sheets, shared per path

`terrain.UnitClass` becomes `{Width, Height, CenterX, CenterY int; Frames []*StaticFrame; Anim
UnitAnim}` — `Frame` is deleted, not kept beside. `sheetCache` gains a memo of the CONVERTED slice
per path: each distinct sheet is decoded and converted once, every class naming it sharing the one
`[]*StaticFrame` — pointer identity lets the frame-keyed texture cache upload a shared sheet once,
and "decoded at most once" now covers the conversion. The 0022 exclusions leave `Frames` nil,
distinct from an id naming no class; only the registry errors. `Anim` is DD-2's descriptor copied
value for value; no GPU, so `-check` reaches the full load headlessly and AC-9's install layouts
run unedited. The census's NoFrame becomes `len(Frames) == 0`. Rejected: per-class conversion,
0022's shape — a shared sheet re-converted per class splits texture identity and repeats work the
memo makes free; rejected: a precomputed frame-0 field beside `Frames` — a second copy with no
consumer.

### DD-5 — the seam owns the clock, the memory and the classification

`mapWorld` gains `scene int` and `facing map[sim.EntityID]int`, born with the map and dropped with
it — never in `pkg/sim`, never in anything hashed (P-4). `tick()` increments `scene` once before
its push; the constructor's tick-0 push runs at 0. The push classifies per the spec's Selection: a
mover's sign-octant is written to `facing[id]`, an idle entity reads its memory (octant 0 if it
never moved) — the memory is looked up by id, never iterated, so map-order nondeterminism cannot
reach the viewer. The effective tick is `scene + int(id)`. `ui.MapEntity` becomes `{Cell; Art
*terrain.UnitClass; Frame *terrain.StaticFrame; Mirror bool}`: the seam hands the SELECTED
drawable — `Frames[frame]` and the mirror bit from `SelectUnitFrame` — or nil for the square (no
class, or no drawable art); the window tier receives what to draw, never how to select. A nil
bundle hands every entity none; the world is read only through `Entities()` copies and written
never. Rejected: clock or memory in the viewer — the window tier gains selection vocabulary, the
standalone viewer dead state; rejected: `w.Tick()` as the clock — the spec makes it app-owned, and
cadence tied to canonical state is the coupling P-4 walls off; rejected: facing as sim state — a
byte-form change FR-7 forbids.

### DD-6 — the mirror is a ridden bit and a GeoM, never a second texture

`terrain.StaticPlacement` gains `Mirror bool`, set by `UnitPlace` — now `UnitPlace(col, row int, c
*UnitClass, f *StaticFrame, mirror bool, lift, originY int)`, placing by the unchanged anchor rule
at the DRAWN frame's own size, false on a nil class or frame. Placement arithmetic, `Rect()`,
`Ground()` and the cull never read the bit; it rides `staticScreenRects` onto `staticScreenRect`
as the frame pointer does, so the cull cannot desynchronise the pair. `Draw` reflects a mirrored
sprite inside its own rectangle — `Scale(-zoom, zoom)`, then translate by `s.X + s.W` — same
texture and cache key, no new upload; the object layer never sets the bit and draws
byte-identically. The fallback chain stays total: the guard gives frame 0 unmirrored, the seam
none, the bounds test off-map, `UnitPlace` false the unchanged square; the pass order — sprites,
squares, diagnostics, ascending id — is not edited. Rejected: a pre-flipped `RGBA()` cached per
(frame, mirror) — double textures and a widened identity key; rejected: a parallel mirror slice
beside the placements — culled entries desync the pairing.

### DD-7 — the witnesses are designed against vacuity

AC-7: the driven-vs-headless digest test is extended — the driven bundle now RESOLVES a Flip-1
mover and an idle-cycle class, and the test asserts the viewer holds sprites mid-run, so digest
equality (at every k, and against the untouched-map digest) is measured over a screen that ticked
the scene clock, wrote facing memory and selected mirrored frames; a leak moves one side. P-3: the
nil-bundle pass slice is compared structurally against the artless derivation at tick 0 AND after
k driven ticks — a scene tick or memory reaching that screen fails equality — with no sprite pass
and no texture map. AC-10: `game.UnitAnimAudit(set)` sweeps, per class, predicted total vs
`len(Frames)` and the full selection domain — both states, 8 octants, every step of each track —
counting in-range vs guarded; `terraintool unitanim` prints one line per class and a summary,
non-zero only on a load failure; a unit test pins the audit's rows over a hand-assembled set with
a deliberately short sheet. Rejected: a digest witness over an artless bundle — vacuously equal,
selection never runs; rejected: the instrument on `cmd/mapview` — the spec freezes the viewer.

### DD-8 — the seam paces the advance; the front-end's call stays one per Update

`mapWorld` gains `clock *terrain.Ticker` at `DefaultSpeedIndex` and `last time.Time`, born with
the map and dropped with it exactly as `scene` is. `ui.MapTick` KEEPS its parameterless signature
and the map-screen arm is not edited: 0020's "exactly one call per map-screen tick" stays
literally true and its counting witness runs unedited, while what happens behind the seam is the
question that tier is defined as unable to ask (0020 DD-1). The method value the front-end
receives becomes `mw.paced` — `paceTo(time.Now())` — and `paceTo(now)` is the tested unit:
elapsed milliseconds since the previous call, the FIRST call taking the baseline and advancing
nothing (the viewer's own pinned rule), handed to `Ticker.Advance`, whose whole-tick count is
clamped to `maxCatchUpTicks = 4` and then run as that many `tick()` calls. 4 is a bound, not a
rate: it keeps real-time pacing down to a 4 fps front-end at 16 tps and caps one call at 248 ms
of world time. `tick()` is NOT edited — one step, one `scene` increment, one push, no clock — so
every digest witness drives it unchanged and wall-clock reaches nothing a tick DOES; the ticks
the clamp drops are dropped, never queued, so a stall costs world time once instead of a lurch
or a spiral. Rejected: the loop in `ui`'s arm — a second cadence beside the viewer's own ticker,
and 0020's contract broken where it is not wrong; rejected: `time.Now()` inside `tick()` — every
digest test becomes clock-dependent and P-4's wall falls; rejected: sharing the viewer's water
ticker — `SetAnimated(false)` would then stop the world.

## Success criteria

- **SC-1** AC-1: the -1s, the eight zeros, `TileSize` 1, `InMapEditor` 0, `File` inherited, a
class resolving no `File` yielding no sprite path; the other registries load as before, DD-1's
tripwires unedited (FR-1).
- **SC-2** AC-2: expansion (ping-pong, zero-time entry, empty pair), bases, strides and totals
as hand-written literals at both `(S, D)`, bone and idle sharing `TailBase`, the gates failing
on empty tracks and non-positive scalars; an absent `-1` phase deriving the descriptor a
resolved 0 derives, block for block (FR-2).
- **SC-3** AC-3 and the selector half of AC-4: DD-3's tables verified literally at both
layouts, mirror exactly on octants 5–7 at Flip 1 and never at 0, steps advancing, looping,
de-syncing by the id difference; the guard answers `(0, false)` out of range, at `frameCount`
0, at a negative tick; equal inputs, equal answers (FR-3, P-1).
- **SC-4** AC-8 and AC-9: one decode per distinct path, the slice shared; frameless
exclusions distinct from a missing id; only the registry errors; the two install-layout
`-check` cases pass over the full-sheet loader (FR-5).
- **SC-5** AC-4 through the seam and AC-5: a short sheet draws frame 0 unmirrored; no art and
no class draw the square; the walker keeps its octant at rest, the never-moved faces 0, the
memories independent per id (FR-4, FR-6).
- **SC-6** AC-6: mirrored and plain placements are identical rectangles at the drawn frame's
size; displaced minus flat is exactly `-(lift+originY)` in Y and 0 in X (FR-6).
- **SC-7** AC-7 as DD-7 designs it: digests equal at every k with selection live; the sim
source scan and the import DAG pass unedited (FR-4, FR-7, P-4).
- **SC-8** P-2 and P-3: every in-map entity draws exactly one item over a mixed snapshot; the
nil-bundle pass slice equals the artless derivation at tick 0 and after k ticks, with no
sprite pass and no texture (FR-6, FR-7).
- **SC-9** AC-10's two halves: the audit unit-tested over synthetic sets; the corpus figures
recorded at verification (FR-2, FR-3).
- **SC-10** AC-12: `paceTo` over hand-written elapsed runs — the first call advancing nothing,
under, at and over one tick with the remainder carried, a long stall firing exactly
`maxCatchUpTicks` — and k paced ticks reaching the digest of k direct ones, the scene clock
rising once per tick rather than once per call (FR-8).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-1 | SC-1 |
| FR-2 | DD-2 | SC-2, SC-9 |
| FR-3 | DD-3 | SC-3, SC-9 |
| FR-4 | DD-5 | SC-5, SC-7 |
| FR-5 | DD-4 | SC-4 |
| FR-6 | DD-5, DD-6 | SC-5, SC-6, SC-8 |
| FR-7 | DD-5, DD-7 | SC-7, SC-8 |
| FR-8 | DD-8 | SC-10 |
