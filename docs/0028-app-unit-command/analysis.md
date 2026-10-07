# Analysis — selecting a unit and ordering it from the running game

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-first / static** — the profile names UI work at this tier; no watcher tool exists, so it is discipline |
| Terrain — the pick, the tap detector, the decision function, the highlight glyph | **greenfield**: none of it exists in any form |
| Terrain — the map screen's dispatch and input snapshot, the viewer's pass list and entity setter, the advance seam, the world driver's command merge | **brownfield**: every one is shipped and drawn from today, and each is touched here |

## The baseline describes a build this repo does not contain

The old clean-room spec for this story names a scene type, an input struct, a command helper and a
wall test that are not here under any spelling. Each was re-derived against the tree rather than
renamed:

| The baseline says | What is actually here |
|---|---|
| `MapScene.Update` steps the world | the front-end's map-screen arm makes one advance call, and the camera half is a separate method the standalone viewer shares |
| the `Input` struct gains a right button | there are **two** input snapshots — the front-end's own, which reads the left button as press/release *edges*, and the viewer's, which reads it as a *level* for drag-pan. The right button is in neither |
| `MoveTo(id,x,y)` sets a target | there is no such call. A command is a plain value naming a unit and a cell, and a step applies it |
| `openrom` opens the map | the game command is `againrom`, and the map arrives through a loader closure the front-end is constructed with |
| the camera exposes no screen-to-world inverse | it does, and the displaced draw path already calls it. What is missing is the *cell* step and the map-extent answer |
| `pkg/sim/determinism_test.go` is the wall check | the wall is two checks in `internal/archtest` — an import DAG and a source scan — plus a pinned field-set test inside the simulation package |
| the snapshot carries units the app can name | it carries a cell and render art per unit and **no identity**, so selecting one is not expressible until an id crosses that seam |

The last row is the one with teeth: the seam was deliberately built so the window tier cannot name a
simulation type, and a selection needs to name a unit. A plain integer id carries no type with it, so
the seam's rule survives the addition — but it is an addition, not a lookup someone forgot.

## The placeholder script does not stop, and that refutes the baseline's override clause

The baseline reads as though the script were one opening order, so that a player's order simply
overrides it. It is not. Every unit is given twenty-four targets — six laps of a four-legged square —
one every twenty-four ticks, staggered seven ticks per unit, and the slice runs about five hundred and
seventy ticks. So last-write-per-step gives the player the unit for at most twenty-four ticks, a second
and a half at the shipped rate, after which the script takes it back.

A contract that says only "the order overrides the current target" therefore specifies a feature that
visibly does not work, and its own manual criterion — the unit walks where it was told — would fail on
any map. The story has to decide what happens to the script, and the spec does (FR-6, C-4).

## The advance is paced, so "the next step" needs saying carefully

One map-screen tick is not one world tick. Elapsed wall-clock time is converted to whole logic ticks
and between zero and four of them run per call, the first call only taking a baseline. So several
orders can pile up between two world steps, and several world steps can run inside one frame — and the
pending orders drain at the **first** of them, not spread across the four. That is why the contract is
written over *an advance* rather than over a frame.

## What a doubled command cannot show, and where a witness is still available

The walking skeleton's own evidence records that a frame's commands applied twice inside one step is
undetectable: last-write makes a repeated move-to idempotent, and that stays true while move-to is the
only kind of command. So replay idempotence is argued there, not witnessed.

This story cannot close that half — it adds no second command kind. It can close the other half, and
that half is new: an order is now *enqueued* by one subsystem and *drained* by another, so "issued
once, applied once, never again" is a statement about the command stream rather than about world state,
and a stream can be counted where a doubled last-write cannot be seen. Hence the reportable count
(FR-4) and its criterion, with the remaining gap disclosed in P-6 rather than left unmentioned.

## Three adjacent features were weighed and two deferred

- **The game cursor.** Decoded and available, and it shares no contract with this story: the pick reads
  the pointer position the engine reports, which is unaffected by where a glyph is drawn relative to
  it. Deferred, and its hotspot recorded as an open choice rather than a decoded fact.
- **A stop, and a speed range.** Wanted, and both are foreclosed in ways worth writing down once
  (`provenance.md`, *Open*). The stop was drafted into this contract and then cut: measured at 14344 B
  of contract with it against 13178 B without, i.e. it is a second concern this story's ceiling cannot
  hold, and the shape it forces — a switch, not a rate — is not a detail the clock story should have to
  find again.
- **A selection highlight that is a filled cell.** Unavailable rather than declined. The frame's passes
  are ordered so that content never covers the instrument measuring it; a highlight has to be drawn
  over the unit to be seen at all, so a filled footprint would erase the very art it marks. FR-2 states
  the visibility requirement instead of the geometry.

## What we looked at

`pkg/ui/{app,flow,viewer,overlay,picker}.go`, `pkg/game/{world,frontend}.go`,
`pkg/sim/{world,step,run}.go`, `pkg/render/camera/camera.go`,
`pkg/render/terrain/{overlay,water}.go`, `pkg/mapload/schedule.go`, `cmd/againrom/main.go`,
`cmd/mapview/main.go`, `internal/archtest/{dag,determinism}.go`, `pkg/sim/nostate_test.go`,
`docs/0019-walking-skeleton/verification.md`, `docs/0026-unit-collision/`,
`docs/0027-vfs-dispatch/`, `AGENTS.md` and both check scripts.
