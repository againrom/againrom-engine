# Spec — the death animation: the fall, and the body it leaves

## Problem and current behaviour

The open map screen draws every entity through one selection — walking, idle, or standing —
chosen from the step it took and the direction it last moved in. **A unit at zero health or
below is drawn exactly as a live one**: it stands upright, and if its class carries an idle
cycle it plays it for as long as the map is open. The sheet block reserved for dying is
addressed by nothing.

Nothing else in the picture says the unit is not alive. The seam carries a life state, read by
the health bar — a dead unit draws none, a downed one an empty track — and by the selection
and order filters. There is no tint, no fade and no other mark.

The simulation keeps a unit that has died: it is not advanced, holds no order, contributes no
occupancy, is never removed, and its id is never freed. Health is signed and never clamped; a
kill leaves it at `-1`; damage refuses an entity already dead; there is no heal and nothing
decrements a health over time. So a health at or below zero never changes again, and a body
stays on the field for as long as the map is open — both already true, neither this story's
to arrange.

## Functional requirements

- **FR-1** The class animation descriptor MUST carry the **length of one direction's slot in
  the dying block**, derived by the same arithmetic and the same absent-phase clamp as the
  bases it already carries, and the render tier's mirror MUST carry it value for value. A
  class declaring no dying phase MUST come out at zero, not at the sentinel for the absence.
- **FR-2** The frames a death is drawn from MUST come from the class the dying key names —
  **that class's sheet, layout switch and descriptor**, never the dying unit's. The resolution
  MUST be a **single hop** by class id, performed once when the bundle is loaded, and MUST be
  total: naming a class the bundle does not hold resolves to **no corpse art**, and naming
  itself resolves to itself.
- **FR-3** There MUST be exactly one death selection, a pure function of the corpse class's
  descriptor, that class's own frame count, the facing octant and the ticks since death. It
  answers a sheet frame and whether that frame draws reflected, or that it has none.
  - One dying frame MUST be held for **exactly two ticks**, so the block plays once over
    `2 x slot` ticks, first frame to last.
  - Once it has played out, the selection MUST **hold its last dying frame** at every later
    tick count, however large.
  - The direction rule and the mirror MUST be the animated blocks' own — the rule the walking
    and idle selections already take — read off the **corpse class's** layout.
  - It MUST answer "no frame" for a class with no dying block and for any index outside the
    frame count it was handed, and MUST answer an index it has not proved for no input.
- **FR-4** The tick count MUST be the map screen's own scene clock less a **per-entity memory
  of the first tick that entity was observed not alive**, written once per entity and never
  rewritten. That memory MUST be render-side state alone — born with the map screen, dropped
  with it, never in the simulation, the canonical byte form or the digest — and MUST be looked
  up by id and never iterated.
- **FR-5** An entity the simulation reports **not alive — downed and dead alike** — MUST be
  drawn through the death selection and MUST NOT reach the walking, idle or standing one. What
  crosses the seam for it is the **corpse class's** art beside the selected frame and mirror
  bit, so placement, anchor, cull and texture downstream treat it as the ordinary sprite it is.
- **FR-6** When the death path yields no frame — no corpse art resolved, that class holding
  none, or the selection refusing its index — the entity MUST keep the drawing it had before
  this story: its own class's live selection, or the square where that class has no art. **No
  unit ever stops being drawn on account of dying**, and no input here is an error.
- **FR-7** A body MUST keep the facing it died with for as long as it is drawn. It MUST NOT
  turn and MUST NOT re-enter a walking or idle cycle.
- **FR-8** The simulation MUST NOT change: no new field, no change to the canonical byte form
  or its digest, no change to what a step does. For one command stream, `k` ticks MUST reach
  the same state and the same digest whatever was drawn between them, and whether or not
  anything died.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | classes whose phase scalars are written by hand, including ones declaring no dying phase and ones declaring none of several | the descriptor is derived | the dying slot is the declared count, zero where it is absent, and `dying base + directions x slot = the base of the block after it` holds in every case |
| **AC-2** | unit | a bundle in which one class names a second, one names itself, one names a class the bundle does not hold, and one names a class that itself names a third | the corpse links are resolved | each class's corpse is the class its own key names; the self-namer is itself; the dangling namer has none; the chain-namer stops at the class it names, never at the one beyond it |
| **AC-3** | unit | a corpse descriptor with a known dying base, direction count and slot length | the selection is asked at every tick count from zero to twice the slot | the frame advances by one every **two** ticks, from the block's first frame for that direction to its last, each being the base plus that direction's offset plus the phase |
| **AC-4** | unit | the same descriptor | the selection is asked past the end of the run, and at very large counts | it answers the **last** dying frame of that direction every time, never leaving that direction's slot |
| **AC-5** | unit | descriptors at both layouts — eight stored directions and five — one with no dying block, and a frame count smaller than the index the arithmetic produces | the selection is asked at every octant | at eight the octant is the slot and nothing mirrors; at five the upper octants fold onto the lower slots and draw reflected; the absent block and the short sheet both answer that they have no frame |
| **AC-6** | unit | a hand-built world, a class whose corpse is a second class with a sheet of its own, an entity killed at a known tick | the pushed entities are read on the tick it dies and after | on the death tick it carries the corpse class's art and that class's first dying frame for its facing; two ticks later the second; the block plays out and then holds |
| **AC-7** | unit | the same world, one entity damaged to exactly zero and one killed outright | both are read | both are drawn through the death path and both play the same fall, the downed one included |
| **AC-8** | unit | a corpse class holding no frames; a class naming a corpse the bundle lacks; a corpse class whose dying block is absent | the pushed entities are read after each dies | every one is still drawn — its own class's art where it has some, the square where it has none — and none carries a death frame |
| **AC-9** | unit | an entity walked west, then killed | frames are read for many ticks after it dies | its drawn direction is the one it walked, unchanged at every later tick, and its frame is a dying frame of that direction and never a walking or idle one |
| **AC-10** | unit | one command stream over a hand-built world, containing kills and damage | `k` ticks are run with the snapshot built at every tick, and again with none built at all | the pinned simulation fields, the byte form and the digest agree at every tick index; and a snapshot built twice with no advance between answers identically, death frames included |
| **AC-11** | manual | the game on a map against a lawful install | a unit is killed and watched | it falls where it stood, over its own class's fall, keeps the direction it faced, then lies still and stays drawn |

**Error cases: None applicable.** Every input on this path is total — an absent phase count is
a block of no length, an unresolved corpse class is no corpse art, a tick count past the run
is the last frame, an index the sheet cannot hold is no frame — so there is nothing to refuse
and nothing to report.

## Derived properties

- **P-1 (invariant)** — Nothing this story adds reaches world state: for one command stream,
  every world field, the byte form and the digest at every tick index are what they are with
  nothing drawn at all.
- **P-2 (idempotence)** — The snapshot built twice with no advance between answers
  identically, the death frames and their tick counts included.
- **P-3 (completeness)** — Every entity is drawn by exactly one of three things: the death
  selection, the live selection, or the square. There is no fourth case and no entity is drawn
  by none.
- **P-4 (negative-invariant)** — No frame this story selects lies outside the **dying block of
  the sheet it came from**: every answer is at or after that block's base for its own
  direction and strictly before the next direction's, so no answer reaches the block after it.

## I/O examples

```
descriptor: dying base B, directions stored D, dying slot L, sheet frame count N
octant o, ticks since death t >= 0
  slot, mirror = the animated blocks' own rule: at D 8, (o, no); at D 5, o <= 4 -> (o, no),
                 otherwise (8 - o, yes)
  phase        = min(t / 2, L - 1)             truncating; one frame per two ticks
  frame        = B + slot*L + phase            answered only if 0 <= frame < N
  L <= 0, or frame outside [0, N)              -> no frame
  B=24 D=8 L=4 o=6 N=60:  t=0 -> 48  t=1 -> 48  t=2 -> 49  t=7 -> 51
                          t=8 -> 51  t=800 -> 51        (the run is 8 ticks long)
```

## Constraints

| # | Constraint | Alternatives and trade-off |
|---|---|---|
| **C-1** | The **decay stage is not modelled**, and only the first one is drawn. | **(A) derive the stage from health** — the engine sets the first stage unconditionally on the tick health reaches zero or below, whatever health reads, and produces every later stage from a decay walk of one health per two ticks; derived from health, an overkill blow would draw a skeleton instantly, which the engine never does. **(B) add a stage, or a countdown, to the simulation** — hashed state, another lane's work. **(C, chosen) draw the one stage this world can be in**: our simulation has no decay walk, so nothing is lost that this world could have shown. |
| **C-2** | The cadence is **decoded, not tuned**. | **(A) a named tuning constant** with a visual check as its arbiter — defensible only while the true number is unknown, and it is not. **(B, chosen) two ticks per dying frame**, read from the engine, and the run length that follows from it. |
| **C-3** | The tick count behind a fall is **render-side state**. | **(A) a per-entity counter in the world** — hashed, serialized, a determinism surface bought for a picture. **(B, chosen) a memory on the seam**, beside the two already there. |
| **C-4** | The corpse class is resolved by **one hop**. | **(A) walk the key as a chain, with a visited set and a cycle guard** — logic no evidence asks for, and a guard whose branches no data can reach. **(B, chosen) subscript once**, as the engine does, and be total about the miss. |

## Out of scope

- **The decay stages past the first** — the bone frames, the vanish, the replay of the last
  two frames when a body is struck again, and the health thresholds behind all three: each
  needs simulation state this story does not add (C-1).
- **Corpse removal** — the world still keeps a dead unit and its id, exactly as before.
- **The classes that leave no corpse at all**, keyed on a database this tree does not decode.
- **Any life-state tint, fade or desaturation.** This tree draws none and this story adds none.
- **The corpse's shadow, its overlay sheets, gibs and sound**; the attack block; and the
  health bar, the selection mark and the order rules, which already answer for a dead unit.

**Disclosed limitations**, accepted and owned: a body lies in its last dying frame for as long
as the map is open, the stages that would follow not being modelled; a unit damaged to exactly
zero plays the same fall as one killed outright; and a class with no dying block, or no corpse
class, keeps standing upright — visibly not a corpse, deliberately, rather than not drawn.

## Verification mapping

AC-1 … AC-10 are unit tests over hand-written descriptors, bundles and worlds — no window, no
clock, no game install — so all are CI-automatable; AC-11 needs a lawful install and a window.
P-1 and P-2 are witnessed by the schedules AC-10 drives, P-3 by AC-6 to AC-8 read together
over one snapshot, and P-4 by AC-3, AC-4 and AC-5.

## Gate check

FR-1 → AC-1 · FR-2 → AC-2, C-4 · FR-3 → AC-3, AC-4, AC-5, P-4, C-2 · FR-4 → AC-6, P-2, C-3 ·
FR-5 → AC-6, AC-7, P-3 · FR-6 → AC-8, P-3 · FR-7 → AC-9 · FR-8 → AC-10, P-1, C-1. AC-11 reads
FR-3, FR-5 and FR-7 together in the running game.
