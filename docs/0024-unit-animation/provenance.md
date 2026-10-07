# Provenance — unit animation

Pinned at research `3d95f2a`, frozen for the story. `claims/retracted.md` read first: the three
overturns on this ground — `TERR-SPR-041`'s dispatch identification, `TERR-SPR-038`'s literal-zero
sixth argument (it is the mirror flag), and `REG-KEY-044`'s `0x11c` allocation figure (`0x12c`) —
are absorbed below. Re-pointed to `03a9448`, 2026-07-30, where two more have fallen at
**High**: `TERR-SPR-048`'s `vt+0x2c` is the unit's **shadow** (the body is `vt+0x28`,
`TERR-SPR-065`, placed by `TERR-SPR-067`), and `TERR-SPR-038`'s fifth argument is that shadow's
sun shear (`TERR-SPR-066`). Neither moves a contract here.

## Backing

| `spec.md` anchor | Claim | Confidence |
|---|---|---|
| The block layout: bases arithmetic on the class's own scalars, `(16,8)`/`(9,5)`, move slots `MB` then `MV`, bone and idle sharing one base | `SPR256-UNIT-024` | High — every base, stride and immediate a named instruction, the routine read end to end; the tiling discriminates every rival / **Medium** for the 33-of-34 figure |
| The selection arms: move = `MB + track[phase mod len]`; the idle track indexed with no modulus; the standing frame = the 16-way facing verbatim | `TERR-SPR-047` (amended) | High (arms, jump table, modulus) / Medium (the state *names*) / Unknown (whether states 2 and 4 ever draw). Amended: the switch is duplicated, the copy read the shadow's, identical arm for arm |
| The built tracks: run-length expansion of `AnimFrame` by `AnimTime`, ending when either side empties; move's modulus is the built length | `REG-UNITS-049` | High — the loop reproduced instruction by instruction |
| `units.reg` absent-everywhere defaults: -1; the eight zero-keys; `TileSize` 1; `File` inherits (FR-1) | `REG-UNITS-049` | High — each key→offset→default a named read; `InMapEditor` excluded on the objects precedent (`REG-OBJ-046`): no engine field, no default decided |
| `Flip` halves the sheet; the reflection idioms `16-g` / `8-d`; the mirror is a blit argument, not a placement one — a mirrored frame fills the same rectangle | `REG-UNITS-051`, `TERR-SPR-065` | High — all five `Flip` reads enumerated, both idioms named; the destination arithmetic is built before and without the flag |
| Bone/idle base sharing is safe (the five idle classes are the five without bone frames); the corpse arms substitute `classes[Dying]`'s sheet — grounds to keep the death chain OUT | `REG-UNITS-050` | High — exhaustive over the registry |
| The `Phases`-vs-track mismatches the gates and guard must survive (Goblin ×2, Ghost) | `REG-UNITS-018` (amended) | High as amended — the length-agreement clause withdrawn; the loader iterates the array's own count |
| Anchor at the drawn frame's size, placement, cull, texture identity — inherited via 0022 | `TERR-SPR-040`/`043`/`067`, `SPR256-FRAME-023` | High — not re-litigated |

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| The looping step, idle included | The engine's idle arm takes no modulus and what bounds its phase counter is unlocated (`TERR-SPR-047`). The loop is the only total reading; it coincides wherever the engine stays in range |
| The scene tick + entity id as the clock | The engine's own advance is unlocated (below); the clock is 0022's wall applied to time, the id offset cosmetic and deterministic |
| Octant 0 = screen south; 8 octants embedded as EVEN 16-way indices | The compass anchor is undecoded (below); the reflection idioms fix only the vertical mirror axis. The standing block is 16-way (`REG-UNITS-051`) — a sign-octant facing reaches only even frames, disclosed |
| Facing memory: the last movement octant, render-side, per open map | The engine keeps facing in unit state (`+0x6c`), written by unlocated code; ours reconstructs it from snapshots, adding no canonical state |
| Moving iff `HasTarget ∧ delta ≠ 0` | The snapshot's own vocabulary (0019/0022); the engine's own classifier is not decoded (below) |
| The gates `MV > 0 ∧` non-empty track, `ID > 0 ∧` non-empty track | The engine tests `IdlePhases != 0` and would divide by an empty move track's length; the stricter form is totality, divergent only on data no shipped class carries |
| Guard target: sheet frame 0, unmirrored | The engine has no guard at all — `Unit33` addresses index 207 of its 129-frame sheet (`SPR256-UNIT-024`, the registry's own `File`/`Flip` inconsistency). Frame 0 is 0022's landed frame and exists whenever art does |
| Whole-sheet decode at load; textures per frame pointer | 0017/0022 mechanics extended; no engine claim consumed |
| Units stay unlit, carried from 0022 | Decided against us: sprites ARE lit (`TERR-LIGHT-059`…`064`, High) and the blit's fifth argument is the sun shear (`TERR-SPR-066`) — a disclosed divergence, not an open question |

## Open / undecoded

- **The compass anchor** — which screen direction stored direction 0 depicts. Cosmetic; AC-11
  arbitrates, the fix is one constant. The likely bearer — the per-dir step tables
  `world+0x58eb0/8` (`TERR-MOVE-056`) — is published without contents; a small research read
  could settle it if the visual check disputes the choice.
- **What advances the engine's animation phase and state** (`unit+0x70`, `+0x74`, the corpse
  stage `+0x15a`): no incrementing instruction found (`REG-UNITS-050`); the clock, classifier
  and loop above are choices, not transcriptions.
- **States 2 and 4** of the nine-state switch: reachability Unknown (`TERR-SPR-047`).
  **Answered since; appended 2026-07-31 at the `8c92427` pin rather than substituted, because this
  is a dated record.** `ANIM-STATE-002` (High) has the nine-value draw state as a copy of a one-byte
  action code and finds **two of the nine set by nothing** — these two. So the arms are unreachable
  rather than merely unwitnessed, and this story's choice not to draw them is convergent.
- **The original's cadence** — ticks per animation step at its frame rate — is not decoded; no
  timing parity claimed.

## Removed from the baseline and why

- **The idle-from-the-end formula** (`total − IdlePhases·Dirs8 + …`) and the whole R-1
  block-order hypothesis. Superseded by the decoded layout: idle sits at the bone base
  `S + D·(MB+MV+AT+DY)` (`SPR256-UNIT-024`), equal to the baseline's expression only where
  `BN = 0` — true of every shipped idle class. **R-1 is closed.**
- **`Flip` as a symmetry *hypothesis* and its R-2.** Decoded outright (`REG-UNITS-051`): 9/5
  stored, the reflection arithmetic named, the flag an argument of the blit. **R-2 is closed**;
  only the compass anchor survives, above.
- **The static fallback `(Index, false)`.** `Index` enters the decoded unit path nowhere —
  0022's own removal.
- **The mirrored-hotspot anchor `cx = frameWidth − CenterX`.** The destination arithmetic never
  reads the mirror flag (`TERR-SPR-067`), so the mirror-anchor computation lives *nowhere*.
- **`MapScene`, `openrom`, `sprite256`, `LoadUnitSprites`, `ObjectAnchor`, `DefinitionID`.**
  None exists here; re-derived onto the 0022 seams.
- **"Gated … never `IdlePhases>1`".** Grounded now (the gates row above); the spec keeps the
  total-function form.
- **The per-`(defID, frameIndex)` texture key.** The shipped cache keys on frame pointer
  identity, which a multi-frame bundle inherits.
- **The "corpus-proven (Ghost ping-pong)" phase framing.** Not contradicted — promoted: decoded
  as the run-length expansion (`REG-UNITS-049`); the ping-pong is one instance.

## Appended 2026-08-02 — pin `01c64e2`: `ANIM-STATE-002` is narrowed in this story's favour, and the note above cites it one grade too high

`ANIM-STATE-002` moves to `● active (amended)`. The row has always carried two grades, and the
2026-07-31 note above takes its conclusion from the wrong one. **High** is that codes 2 and 4 are
written by no *immediate* anywhere in the image — a census complete under the field's own byte
width, 20 writes listed by address. **Medium** is the stronger reading *"2 and 4 are never set at
all"*, because four of those 20 writes take a register rather than an immediate. "So the arms are
unreachable rather than merely unwitnessed" is the Medium reading, and the note attributes it to
High. Corrected here rather than in place, the note being a dated record.

EXP-0084 re-read two of the four register writes and **neither is an origin**: `L02487`'s `AL` is
the return of a key/value getter called with the field's own current value as the default, under the
keys `"action"`/`"actiondir"`/`"actionphase"` — a *restore* of whatever that store was given — and
`L02488` writes `[arg+0x154]`, not the argument. The remaining two are the base constructor (with
`EBX = 0`) and the copy constructor (copying another instance's byte). Research states the clause
**narrowed, not lifted**, and holds it at Medium.

So the conclusion this story leaned on is better warranted than when it was written, at the same
grade it always actually had. Nothing in the contract moves: FR's choice not to draw states 2 and 4
was made without them, and their reachability was never a premise the spec asserts.
