# Spec — animated water in the map viewer

**Provenance basis.** The ROM1 water-animation model, its counter, and its cadence are decoded from the
game and recorded as research claims TERR-ANIM-006…010 (with the tile-word split TERR-IDX-003 that 0004
already implements). Four agreeing `rom.exe` renderers select the animated image with identical
arithmetic, the counter has exactly two writers in the whole binary, and the phase expansion is falsified
against the 38-map corpus. This story implements that model and nothing beyond it. No reference-port
animation logic is used, and no cadence constant is invented. Greenfield.

## Problem / goal

0004/0005 draw every tile with its stored graphic, so water is frozen. The game animates it: water cells
walk a four-image cycle, offset per cell so neighbouring cells ripple diagonally rather than pulsing in
unison, advanced by a counter that ticks with the game clock. This story adds that cycle to the render
path — as a pure, unit-testable function of `(tile word, world position, counter)` — and drives it from
the viewer at the decoded cadence.

## What the research establishes

**Water is a strip group, not an index range.** A type1 tile word splits as `g = (w & 0x1fff) >> 6`
(strip group), `b = (w >> 4) & 3` (blend column), `sub = w & 0xf` (sub-cell) — TERR-IDX-003, implemented
in 0004. Water is `g ∈ 8..11`, which is the `tile3` graphic family.

**The cycle (TERR-ANIM-006).** A water cell keeps its blend column `b` and sub-cell `sub`; the renderers
override **only the strip group**, with `8 + phase`:

```
phase = (g + (worldCol + 1)·worldRow + (animCtr >> 2)) & 3
```

Since `slot = g*4 + b` and the `tile3` variant is `V = (g & 3)*4 + b`, substituting `g := 8 + phase`
gives `V = phase*4 + b` — the drawn image walks `tile3-{b, b+4, b+8, b+12}` as `phase` steps
`0 → 1 → 2 → 3 → 0`. The 16 shipped `tile3` files are exactly **4 phases × 4 blend columns**.

The `(worldCol + 1)·worldRow` term is what makes the water ripple: it offsets each cell's phase by its
position, so neighbours are out of step, and because it uses world (not screen) coordinates the pattern is
**stable under scrolling** — panning does not shift the ripple.

**The counter (TERR-ANIM-007).** `animCtr` is initialised to `0` and incremented by `1`; a whole-binary
scan finds exactly those two writes and no others, and the incrementing routine is the handler of the
logic-tick window message. So the counter advances **once per logic tick**. Because the phase uses
`animCtr >> 2`, the drawn variant changes every **4 ticks** and the full four-variant cycle completes
every **16 ticks**.

**The cadence (TERR-ANIM-008).** One logic tick fires every `dtMs` milliseconds, where `dtMs = 1000/tps`
(integer division, as the game's `IDIV`). A clamped speed index `0..8` maps to `tps`:

| index | 0 | 1 | 2 | 3 | **4** | 5 | 6 | 7 | 8 |
|---|---|---|---|---|---|---|---|---|---|
| tps | 8 | 10 | 12 | 14 | **16** | 20 | 24 | 28 | 32 |

**Map load pushes index 4** → 16 tps → `dtMs = 62` → a new variant every ~248 ms and a full cycle every
~992 ms (the ideal rationals being 62.5 / 250 / 1000 ms; the integer division is the game's own, and this
story reproduces it rather than correcting it). Per-variant ms is `4·dtMs`, per-cycle `16·dtMs`.

The research rates the per-index cadence and the map-load default **High**, and only "which index a
saved or multiplayer session resumes at" **Medium** — the viewer has no saved session, so it takes the
map-load default and that residual uncertainty does not apply here.

**The enable flag (TERR-ANIM-009).** Animation is **on by default**; the game's `-noanimation` and
`-detail0` switches turn it off, and when off the renderers use `phase = 0` — not a frozen current phase.
Phase 0 is therefore exactly what 0004 already draws (it forces the group to 8), so disabling animation
must reproduce the 0004 image byte-for-byte.

**Corpus falsification (TERR-ANIM-010).** Expanding all four phases of every water cell across 38 maps —
87 580 water cells, 350 320 (cell, phase) pairs — yields **0** references to an absent `tile3` file and
**0** variants past `tile3`'s sub-cell count. The reachable set is exactly `tile3-00…15`, all of which
ship. A wrong `b`/`V` split would push `V > 15` into a file that does not exist; it does not.

## Behavior definition

Every water cell draws `tile3` variant `V = phase*4 + b` at sub-cell `sub`, with `phase` from the formula
above. Non-water cells are unaffected. With animation disabled the phase is `0` for every cell, which
reduces exactly to the 0004 render.

The counter is advanced from wall-clock time by an accumulator: elapsed milliseconds are added to a
running remainder and every whole `dtMs` fires one tick. Time is read in the UI tier only; the phase
arithmetic, the speed table and the accumulator are pure and engine-free.

## Functional requirements

- **FR-1 (phase)** The terrain tier MUST expose the phase as a pure total function of the strip group,
  world column, world row and counter: `phase = (g + (col+1)·row + (animCtr>>2)) & 3`, defined for every
  input (including negative or out-of-range positions) and never panicking.
- **FR-2 (animated resolve)** The terrain tier MUST expose an animated tile-word resolution that, for a
  water word (`g ∈ 8..11`), returns the slot for group `8 + phase` while preserving `b` and `sub`, and
  that returns the identical result to the existing static resolution for every non-water word.
- **FR-3 (static equivalence)** The existing static resolution MUST remain available and MUST equal the
  animated resolution at `phase = 0`, so the animation-disabled path is the 0004 render exactly.
- **FR-4 (cadence)** The terrain tier MUST expose the speed table (`index 0..8 → tps`), the map-load
  default index `4`, `dtMs = 1000/tps` by integer division, and an accumulator that converts elapsed
  milliseconds into whole ticks, carrying the sub-tick remainder so no time is lost or double-counted.
  An out-of-range speed index MUST clamp into `0..8` rather than error or panic.
- **FR-5 (viewer)** The viewer MUST advance the counter from real elapsed time each frame and draw water
  at the current phase, MUST default to animation on at the map-load speed index, and MUST expose
  switches to disable animation and to select a speed index.
- **FR-6 (purity / DAG)** All animation arithmetic lives in `pkg/render/terrain` (stdlib only, no engine,
  no clock, no floats in control flow). Only `pkg/ui` reads the clock. The fail-closed DAG check stays
  green.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | strip groups 0..127 | classified | exactly `8..11` are water; every other group is not |
| AC-2 | unit | a water word with a known `g`, `b`, `sub` at a known `(col,row)` and counter | resolved animated | the slot is `(8+phase)*4 + b` for the formula's phase, `sub` is unchanged, and the ref reports water |
| AC-3 | unit | any non-water word, at arbitrary `(col,row)` and counter | resolved animated | the result is byte-identical to the static resolution — position and counter change nothing |
| AC-4 | unit | a water word held fixed while the counter runs `0..63` | resolved animated | the slot changes exactly every 4 ticks, repeats with period 16, and visits all four phases in the order `p, p+1, p+2, p+3 (mod 4)` |
| AC-5 | unit | every `(g, b)` water combination expanded over all four phases | resolved animated | the tile3 variant `V = phase*4 + b` lands in `0..15` for all of them (the TERR-ANIM-010 reachability bound), and the slot stays inside the 128-slot tileset |
| AC-6 | unit | two horizontally and vertically adjacent water cells with equal `g` | resolved animated | their phases differ per the `(col+1)·row` term, so adjacent cells are not in lockstep |
| AC-7 | unit | speed indices `-3, 0, 4, 8, 11` | queried | tps is `8,8,16,32,32` (out-of-range clamps) and `dtMs` is `1000/tps` by integer division — `62` at the default index 4 |
| AC-8 | unit | an accumulator at index 4 fed 10 ms twenty times, then 500 ms, then a negative elapsed | advanced | ticks fire only on whole 62 ms boundaries with the remainder carried (3 ticks over the first 200 ms), a large elapsed fires the whole quotient at once, and a negative elapsed fires none and does not rewind |
| AC-9 | unit | the static resolution and the animated resolution at counter 0 with the phase forced off | compared over all 65536 words | identical for every word (FR-3) |
| AC-10 | manual | a real GOG map with water, viewed with and without the disable switch | run | water visibly cycles with a diagonal ripple that does not shift when panning; the disable switch freezes it to the 0004 image; recorded in `verification.md` |

## Derived properties

- **P-1** (invariant) The phase is always in `0..3` for every input, including negative coordinates and
  any counter value.
- **P-2** (invariant) Animated resolution never names a slot outside `0..127`, so it can never index past
  the tileset.
- **P-3** (invariant) Non-water words are pure passthrough: for all `w` with `g ∉ 8..11` and all
  `(col,row,ctr)`, animated == static.
- **P-4** (invariant) The accumulator conserves time: after any sequence of non-negative advances, the
  total ticks fired equals `⌊(Σ elapsed) / dtMs⌋` and the retained remainder is `(Σ elapsed) mod dtMs`.
- **P-5** (invariant) The counter's low two bits do not affect the image: `animCtr` and `animCtr | 3`
  differing only below bit 2 give the same phase.

## Constraints

- Builds on the 0004 tile-word mapping and the 0005 viewer; no format, archive or `pkg/formats` change.
- `pkg/render/terrain` stays stdlib-only with no clock and no floats in control flow; the wall clock is
  read only in `pkg/ui`.
- The integer division `1000/tps` is reproduced as the game performs it, not rounded.

## Out of scope

- Relief lighting and the day/night tint (story 0007) — water here is drawn at full brightness like the
  rest of 0004's terrain.
- The dirt composite correction, and retaining palette indices through the BMP decode.
- Any other animated terrain, unit or structure animation, and fog of war.
- The `-detail0` switch's other effects: only the animation-disable behaviour is in scope.
- Multiplayer/saved-session speed resumption (the research's Medium-confidence residue) — the viewer
  always starts at the map-load default index.

## Verification mapping

AC-1…AC-9 + P-1…P-5: unit tests in `pkg/render/terrain` over synthetic tile words, positions and counter
values — no game install, no engine context. AC-10: a developer run against a lawful install, recorded as
evidence in `verification.md`.
