# Spec — unit animation (idle and move, with facing)

## Problem and current behaviour

An entity that resolves art draws as one standing frame — sheet frame 0, one fixed facing —
wherever it walks. `pkg/data` already decodes every animation key of `units.reg`; the
sheets hold every frame; the loader converts exactly one. Nothing selects a frame from a unit's
state, facing or time.

This story animates idle and move with 8-way facing: selection is pure render-side logic over
the read-only snapshot and an app-owned clock; a `Flip` sheet's mirrored half becomes drawable;
every failure degrades to today's frame or square. `pkg/sim` does not change in any byte.

## The sheet contract

A unit's `.256` sheet is block-addressed; the block bases are arithmetic on the class's own
resolved phase scalars — nothing is stored in the sheet. Write `MB, MV, AT, DY, BN, ID` for
`MoveBeginPhases`, `MovePhases`, `AttackPhases`, `DyingPhases`, `BonePhases`, `IdlePhases`.
`(S, D)` — standing frames stored, directions stored per animated block — is `(16, 8)` at
resolved `Flip` 0 and `(9, 5)` otherwise.

| block | base | length |
|---|---|---|
| standing | `0` | `S` |
| move | `S` | `D*(MB+MV)` — per direction slot: `MB` wind-up frames, then `MV` loop frames |
| attack | `S + D*(MB+MV)` | `D*AT` |
| dying | `S + D*(MB+MV+AT)` | `D*DY` |
| bone and idle | `S + D*(MB+MV+AT+DY)` | `D*BN` / `D*ID` — one base serves both |

Every phase scalar enters this arithmetic **clamped at zero**. An absent key resolves to FR-1's
`-1`, and an absent phase contributes NO block — to no base, and not to the total. The gates
below read the resolved scalar, so an absent phase still fails them.

The **predicted total** is `S + D*(MB+MV+AT+DY+max(BN,ID))` over those clamped scalars — a
computed property, never an assumption: a sheet may hold any frame count, and a mismatch is data.

## Directions and the mirror

Facings are the eight movement sign-octants, `S=0, SW=1, W=2, NW=3, N=4, NE=5, E=6, SE=7`,
screen +y south. Stored direction 0's compass meaning is not established; the reflection rules
fix only octants 0 and 4 to the vertical axis.

- Standing is 16-way, `g = 2*oct`: at `S = 16` frame `g`, nothing ever mirrored on this layout;
  at `S = 9` frame `g` for `g <= 8`, else `16 - g` mirrored. Animated blocks are `D`-way: at
  `D = 8` slot `oct`; at `D = 5` slot `oct` for `oct <= 4`, else `8 - oct` mirrored.
- A mirrored frame draws reflected about the vertical centreline of the SAME placed rectangle:
  placement, anchor, ground point and cull never read the mirror bit.

## Cycles and clocks

- Each `AnimTime`/`AnimFrame` pair expands run-length into a **track**: `Frame[i]` appended
  `Time[i]` times (an entry whose `Time <= 0` appends nothing), both heads dropped each round,
  ending when either side empties. The track — never a `Phases` scalar — is the cycle: its
  length is the period in ticks, its values are sub-frame indices within one direction's slot.
- The **scene tick** is app-owned and monotonic: 0 when a map opens, +1 per logic tick (FR-8),
  dropped with the map — never in a world, never hashed. An entity's **effective tick** is the
  scene tick plus its entity id: a cosmetic de-sync from deterministic snapshot data.
- The current step of a non-empty track is `track[effectiveTick mod len(track)]`, looping.

## Selection

Per entity of the latest snapshot, with `(dx, dy) = (TargetX - X, TargetY - Y)`:

- **Moving** iff `HasTarget` and `(dx, dy) != (0, 0)`; every other entity is idle, one standing
  on its own target included.
- **Facing.** A moving entity faces the sign-octant of `(dx, dy)`. An idle entity keeps the
  octant of its most recent movement (render-side memory per entity id, per open map); one that
  never moved faces octant 0.
- **Move** — gate: `MV > 0` and a non-empty move track. Index = `S + slot*(MB+MV) + MB +
  moveTrack[step]`, mirrored per the direction rule. A moving entity failing the gate draws its
  idle selection.
- **Idle cycle** — gate: `ID > 0` and a non-empty idle track. Index = `S + D*(MB+MV+AT+DY) +
  slot*ID + idleTrack[step]`, mirrored per the direction rule.
- **Idle without a cycle** — the standing selection for the facing.
- **Bounds guard.** An index outside `[0, frameCount)` draws sheet frame 0, unmirrored — the
  frame drawn today. A class with no drawable art keeps the square; an off-map entity draws
  nothing, as before.

## Functional requirements

- **FR-1 — decoded absent-everywhere defaults for `units.reg`.** A scalar key no section on a
  class's chain sets resolves to **-1**, except `IdlePhases`, `Dying`, `Palette`, `Projectile`,
  `ShootDelay`, `AttackDelay`, `Z`, `Flip` (**0**), `TileSize` (**1**) and `InMapEditor` (the
  Go zero: no engine record field). `Parent`'s -1 is the no-parent value
  presence still overrides; `File` continues to inherit here, unlike `objects.reg`, and a class
  resolving no `File` still resolves no sprite path. The whole scalar table moves at once;
  string and array keys, the guards, validation, the other registries: unchanged.

- **FR-2 — the descriptor.** The data tier derives, purely from one resolved unit class:
  `(S, D)` from resolved `Flip`; the bases and strides of the sheet contract; the expanded move
  and idle tracks; the two gates; the predicted total. No IO, no sheet access, no float. A
  `Phases` scalar is never a track length; `Dying` is never consulted; `BN` enters only the
  predicted total.

- **FR-3 — the selector.** A pure render-tier function of plain data — descriptor values, the
  sheet's frame count, the moving flag, the facing octant, the effective tick — returning
  `(frame index, mirror)` per the Selection rules. Total: negative or zero scalars, empty
  tracks and any tick fall through the gates and the guard; no panic, no division by zero;
  integers only, equal inputs equal answers.

- **FR-4 — the resolution seam.** The tier that owns the world and the bundle classifies
  moving/idle, owns the scene tick and the facing memory, and hands the window tier, per
  entity, a plain cell and either a selected render-tier drawable — frame, class geometry,
  mirror bit — or none. It reads the snapshot through the copy-handing accessors and writes no
  world; clock and memory drop with the map.

- **FR-5 — the loader.** A class's bundle entry carries every frame of its sheet — each
  distinct sheet decoded at most once, no GPU work — beside the canvas and the descriptor's
  plain data. Exclusions unchanged: only an unreadable or unparseable registry errors;
  an absent, undecodable or palette-less sheet leaves a frameless entry, distinct from an id
  naming no class.

- **FR-6 — the draw, never silently vanishing.** Each resolved entity draws its selected frame
  at its cell, placed by the existing, unchanged unit placement — the anchor rule over the
  class canvas at the DRAWN frame's own size, cell-centre ground point, cell-mean lift in
  displaced mode — culled on its exact placed rectangle, textured lazily on first draw under
  frame identity, mirrored frames reflected within their own rectangle. The fallback chain is
  Selection's, total and error-free: frame 0 unmirrored; the square, unchanged in glyph, size
  and colour; off-map, nothing. Layer and intra-layer order — sprites then squares, ascending
  id, under the diagnostics — are unchanged.

- **FR-7 — walls standing.** `pkg/sim` is byte-for-byte unchanged; its source scan and the
  import-graph check stay green. Driving the map screen k ticks leaves the world's digest a
  headless run's. The window tier gains no simulation, format or data vocabulary. The
  standalone viewer changes in nothing and owns no world. `-check` performs the full unit load
  headlessly; no flag is added or changed.

- **FR-8 — the advance runs at the decoded logic rate.** The scene tick and the world step it
  rides with fire at the decoded logic rate — `TicksPerSecond` at the default speed index, 16/s,
  a 62 ms tick — never at the frame rate. The seam converts elapsed wall-clock milliseconds into
  whole ticks by the tick model the water layer already paces by, carrying the sub-tick
  remainder; its first call only takes the baseline, and **at most 4 ticks** run per front-end
  call, so a stall drops the time past that bound instead of bursting, and repays nothing later.
  Wall-clock decides only WHEN a tick fires: a tick is the same integer step whatever paced it,
  `pkg/sim` is unchanged, and k paced ticks reach k direct ticks' state.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a synthetic `units.reg`: a chain setting no scalar keys; a child omitting `File` under a parent setting it | loaded | -1 / the eight zeros / `TileSize` 1 / `InMapEditor` 0, per key; `File` inherited; the other registries load as before |
| AC-2 | unit | anim pairs with a `Time = 0` entry, a `[0 1 2 1]` ping-pong, an empty pair; classes at both `Flip` values, assorted scalars | descriptors derived | run-length expansion, period = track length, the ping-pong survives, the zero entry adds no tick, the empty pair fails its gate; bases, strides, predicted totals per the table at both `(S, D)` pairs, bone and idle sharing one base |
| AC-3 | unit | both layouts, 8 octants, both states, a tick run, two ids | selections read | indices follow the formulas; mirror exactly on octants 5–7 (standing: `g > 8`) at `Flip` 1, never at 0; steps advance, loop, de-sync by the id difference |
| AC-4 | unit | a sheet shorter than a selection reaches; a class with no art; an id naming no class | drawn through the seam | frame 0 unmirrored; the other two the square; nothing errors |
| AC-5 | unit | one entity walking then stopping; one that never moved | successive snapshots pushed | the walker keeps its last octant while idle; the other faces octant 0; the memories are independent |
| AC-6 | unit | one frame drawn mirrored and plain; a lift and an originY | placed | identical rectangles at the drawn frame's size; displaced minus flat is `-(lift+originY)` in Y and 0 in X |
| AC-7 | unit | a map and its schedule | the map screen driven k ticks, and a headless run | equal digests; the sim source scan passes |
| AC-8 | unit | a synthetic registry: multi-frame sheets, a shared sheet, the exclusion set | the bundle loads | every frame decoded, one decode per distinct path, no GPU; frameless entries for exclusions; only the registry errors |
| AC-9 | integration | install layouts with and without a readable unit registry | `-check` runs | the first passes, loading full sheets headlessly; the second fails naming the registry |
| AC-10 | corpus | a lawful install's unit classes | the corpus instrument runs | per-class predicted totals vs sheet frame counts, and the full selection domain — both states, 8 octants, every step — recorded: every selection in range or guarded |
| AC-11 | manual | a lawful install, representative maps | opened, run, watched | units walk with 8-way facing under their crosses; a `Flip` class mirrors correctly; an idle-cycle class animates at rest; the anchor limitation noted |
| AC-12 | unit | a paced seam over a synthetic map; elapsed runs under, at and over one tick, and a long stall | fed elapsed milliseconds | ticks fire at the decoded rate, the remainder carried, the first call advancing nothing; the stall fires the bound and no more; k paced ticks hold k direct ticks' digest |

Error cases: the registry (AC-8, AC-9); everything else falls back, never errors (AC-4).

## Derived properties

- **P-1** (invariant) Selection is pure: `(frame, mirror)` is a function of descriptor, frame
  count, state, octant and effective tick alone.
- **P-2** (completeness) Every in-map entity draws exactly one item — sprite or square; off-map
  is the only other case.
- **P-3** (negative-invariant) Without a bundle, every screen is byte-identical to today's.
- **P-4** (invariant) No render-side state — scene tick, facing memory, textures — reaches a
  world; digests are a headless run's at every k.

## I/O example

```text
class: Flip=1 (S=9, D=5), MB=1, MV=2, AT=2, DY=2, BN=2, ID=0
      sheet 54 frames = the predicted total 9 + 5*(1+2+2+2+2)
move: Time=[2,1], Frame=[0,1] -> track [0,0,1], period 3
entity id 1, octant 6 (E), scene tick 6 -> effective 7, step 7 mod 3 = 1 -> sub-frame 0
octant 6 > 4 -> slot 8-6 = 2, mirrored
index = 9 + 2*(1+2) + 1 + 0 = 16 -> (16, mirror): drawn reflected in place
```

## Constraints and alternatives

| Choice (all selected; *disclosed* = a named fidelity gap) | Observable trade-off |
|---|---|
| The looping step, idle included — *disclosed*: the original bounds its idle phase by bookkeeping not modelled here | a cycle never runs out; agreement wherever the original stays in range; no cadence parity claimed |
| The clock is the decoded logic tick plus an id offset — *disclosed*: the original's animation timer is not modelled | cadence holds at any frame rate; a stall drops time and never repays it; the de-sync is cosmetic and deterministic |
| Octant 0 = screen south — *disclosed*: stored direction 0's compass meaning is unestablished | a wrong anchor shows as units facing sideways to their walk; one constant fixes it |
| Idle keeps the last movement octant, reconstructed render-side | facing survives a stop, forgets with the map, adds no canonical state |

## Out of scope

- Attack, dying, bone, the corpse chain and its cross-sheet class substitution; `MoveBeginPhases`
  as a played wind-up state; rotation.
- A settable speed index — the default's 16/s is the only rate; sub-cell position; depth
  interleave with object art; shadows; palette remap; `Z`-based air layering; selection and
  picking.
- Any change to `pkg/sim`, `pkg/mapload`, the ALM decode, terrain, markers, the schedule or the
  standalone viewer; `objects.reg`/`structures.reg` defaults.

## Verification mapping

AC-1…AC-9, AC-12 and P-1…P-4 run headlessly over synthetic registries, sheets, maps and worlds —
no game install, no GPU (AC-9 over install layouts), no clock (AC-12 takes its elapsed times as
arguments). AC-10 is a developer-run corpus tool over a lawful install — no install path in
source, figures recorded, no asset committed. AC-11 is a developer-run visual check needing a
window.

Gate coverage: FR-1 → AC-1 · FR-2 → AC-2, AC-10 · FR-3 → AC-3, AC-4, AC-10, P-1 · FR-4 → AC-5,
AC-7, P-4 · FR-5 → AC-8, AC-9 · FR-6 → AC-4, AC-6, AC-11, P-2 · FR-7 → AC-7, AC-9, P-3, P-4 ·
FR-8 → AC-12, AC-7.
