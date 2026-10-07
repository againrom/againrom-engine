# Spec — the game says what it is doing

Intensity: **spec-first / static**. Terrain: **brownfield** in `pkg/ui` and `pkg/game` — the panel
composition splits and the entity seam widens by one scalar; **greenfield** for the readout itself.

## Problem and current behaviour

Space, `=` / numpad `+` and `-` / numpad `−` move the world clock. A rate ladder that doubles and
halves between 1 and 1024 ticks a second, and a stop beside it, have been shipped and reachable for
several stories. **Nothing on screen says so.** A press produces no text, no marker and no number.

That is not a cosmetic gap, it is an evidence gap, and it has three separable parts:

- At 16 ticks a second against 32, the only observable difference is how fast a walking unit crosses
  a cell — so on a map where nothing is walking, the three keys are indistinguishable from three
  keys that are not bound at all.
- At either **end** of the ladder the clamp holds the rate still and a press is *designed* to change
  nothing. A working key at the clamp and a broken key produce the identical picture.
- The stop is the one state a still picture cannot report, because a stopped world and a world with
  nothing to do look the same.

The map screen already draws one text box — the unit information panel, bottom left, carrying the
selection count. It draws it only for a selected unit, it states nothing about the world, and it is
a **substitution point** for the original's own interface, so it is the wrong place to put a
development instrument.

## What the readout is

**FR-1 — The map screen draws a readout of the running game's parameters.** It is a labelled box of
one value per line, drawn over the map in window coordinates, stating what the world and the view
are presently doing. It is **ours**: a development instrument, not a reconstruction of anything the
original drew, and it asserts nothing about the original.

**FR-2 — Every value it states is read, at the moment it is stated, from the thing that owns that
value.** No value on the readout is accumulated, incremented, latched or remembered on the path
that *changes* it. Specifically: the cadence is read from the clock the world's advance is decided
by, and never from the key ladder that asks that clock to move.

This is the requirement the story exists for. A number kept beside the authority agrees with it
until the first clamp, refusal or truncation and is silently wrong afterwards — and a debug readout
that is wrong is worse than none, because it is believed over the game.

**FR-3 — It states these values, with FR-13's beside them and, by default, no others:**

| Line | What it is |
|---|---|
| Cadence | The rate the world's clock is running at, in **whole ticks a second**, and the stop beside it — the two quantities the three keys move. Stopped is stated as such and not as a rate of zero, because the rate is unchanged by the stop and is what the world resumes at. |
| Tick length | The **period** that same clock holds, in microseconds. It is the quantity the clock physically carries; the rate above is derived from it. |
| Tick | The world's own tick number. |
| Digest | The world's full-state digest, as 16 hexadecimal digits. |
| Cursor | The map cell the cursor presently resolves to, or a marker for a cursor that resolves to no cell. |
| Speed | The selected unit's own speed input, absent when nothing is selected. |
| Crossing | How many ticks the selected unit's **current** cell crossing runs for, absent when nothing is selected. |
| Entities | How many entities the frame was given to draw. |
| Frame rate | Frames a second, as the engine measures it, rounded to a whole number. |

**FR-4 — Cadence and tick length are two readings of ONE quantity and cannot disagree.** The rate is
computed from the period by the one function that is the inverse of the one that computes a period
from a rate. Over the whole rate range those two functions round-trip exactly, so a rate stated on
the readout is the rate the clock would be re-rated to by stating it back.

**FR-5 — Crossing is the length of the crossing the unit is actually on**, as the simulation
recorded it when it took that step — not a length recomputed here from the speed. A unit that has
taken no step, and one whose speed does not rate it, is on no such crossing and the line states
that rather than a number.

**FR-6 — The cursor line states the cell the *ground pick* resolves**, which is the cell a right
press would order a unit to. On sloped ground that is knowingly not the cell the drawn lattice puts
under the cursor: the lattice rides the placement surface and the ground pick resolves through the
flat one. Stating the pick's answer is deliberate — it is what the game will act on, and the
readout is what makes that divergence visible for the first time.

**FR-13 — It states the selected unit's group rate term on a line of its own,
beside its speed and never folded into it.** A unit ordered as part of a group
carries that group's slowest speed, and while it carries one *that* is the rate
it moves at — its own speed is inert. The term outlives the order it came from:
arriving does not clear it, and neither does another member dying, so a survivor
keeps walking at a pace nothing on screen accounts for.

Two lines rather than one, because folding them would hide exactly that. A
single "effective speed" would show the right number and destroy the reason it
is that number; two lines show a unit whose own speed is 40, whose group term is
12, and which is therefore crawling — which is the whole diagnosis, read off the
box. The term is stated as absent when the unit carries none.

## Where it is, and when

**FR-7 — It is on by default.** A readout that must be switched on before it can report the key
that was just pressed does not answer the request that produced this story.

*Folded from hotfix `2ad4c29` — see `docs/hotfix/ARCHIVE.md#2ad4c29`.* On by default is the
VIEWER's default, not the game's: the game's two front-end call sites open the readout hidden and
`F1` (FR-8) shows it. The switch is thrown at the call sites, so what is asserted is where.

**FR-8 — `F1` hides it and shows it again.** It is a press edge, read on the map screen alone, and
it takes an **F-key** because it is a diagnostic: the register that distinguishes a key that changes
the world from one that changes only what is drawn over it. `F1` was left free for it when that
register was authored; `F2` is the cell lattice.

**FR-9 — Hidden, it costs nothing.** No composition, no measurement, no digest and no engine call
is made for a readout that is not being drawn.

**FR-10 — It does not overlap the unit information panel or the selection count.** Those are drawn
in one corner; this is drawn in another.

**FR-11 — A viewer that was handed no font draws no readout, and fails at nothing.** That is the
gate the unit panel already stands behind, and it is what leaves the standalone developer viewer
— which owns no world and is handed no font — drawing exactly the frame it drew before.

**FR-12 — The readout writes nothing.** It advances no world, issues no order, moves no camera,
moves no selection and changes no rate. Its whole effect is pixels.

## What does not change

- The three cadence keys, the ladder, the clamp and the stop. This story observes them; it does not
  move them.
- The unit information panel, its layout, its fields, its corner, and the selection count on it. A
  frame that drew one before draws the identical one now.
- `pkg/sim`. Nothing is added to an entity, to the byte form or to the digest, and no simulation
  rule is read out of that package to be recomputed here.
- The cell lattice and its key.
- The standalone developer viewer's picture.

## Acceptance criteria

**AC-1** — With a world open, the readout is visible without any key being pressed.

**AC-2** — Pressing `+` at the map-load cadence doubles the stated rate and halves the stated
period; `-` halves and doubles them back.

**AC-3** — Pressing `+` past the top of the ladder leaves the stated rate at the ceiling and the
stated period at the ceiling's period, however many further presses are made; `-` past the bottom
likewise. The stated value is the clamped one at every step.

**AC-4** — Driving the clock to a rate the ladder cannot ask for — by re-rating it directly, above
the ceiling and below the floor — makes the readout state the **clamped** rate the clock adopted
and never the rate that was requested.

**AC-5** — Pressing Space states the stop; pressing it again clears it. The stated rate is
unchanged across both.

**AC-6** — While stopped, the stated tick and digest hold still; running, the tick advances by one
per tick and the digest tracks the world's own.

**AC-7** — The stated digest equals the world's own digest for the state the frame was drawn from.

**AC-8** — With a unit selected, the speed and crossing lines state that unit's own; with nothing
selected, neither line is drawn and the box is shorter by exactly those lines.

**AC-9** — `F1` hides the readout and `F1` again shows it; the frame drawn while it is hidden is
byte-identical to the frame the same state drew before this story.

**AC-10** — The unit information panel's pixels are identical with the readout shown and hidden,
and the two boxes share no pixel.

**AC-11** — A viewer holding no font draws no readout and does not fail.

**AC-12** — The cursor line states the same cell the ground pick resolves for the same pixel, and
states its absence marker for a pixel that resolves to no cell.

**AC-13** — A unit carrying a group rate term states it and its own speed as two different numbers;
a unit carrying none states its speed and the absence marker. Both lines appear and disappear with
the selection, together with the crossing.

## Properties

**P-1** — *Invariant.* The stated cadence is a function of the clock's held period alone. Two paths
that leave the clock at the same period state the same cadence, whatever sequence of presses,
re-rates or clamps got them there.

**P-2** — *Idempotence.* Drawing the readout twice from one unchanged state produces identical
pixels, and composes at most one picture.

**P-3** — *Negative invariant.* No state of the world, the selection, the camera or the clock is
different after a frame that drew the readout from what it is after the same frame with the readout
hidden.

**P-4** — *Completeness.* Every field the layout names either states a value or is omitted; no
field draws an empty or partial line, and no value is truncated to fit.

**P-5** — *Invariant.* Stating a rate back to the clock re-rates it to the period it was read from.

## Out of scope

- Any change to what the three cadence keys do.
- Reading the original's own debug or interface layer. Nothing here is a reconstruction.
- Making the ground pick agree with the drawn lattice. The readout **exposes** that divergence; the
  divergence itself is another story's.
- A configurable field set, a second readout, or persisting the hidden/shown state across runs.
- Any figure about the original's speeds beyond the ones already shipped.

## Error cases

- **No font** — no readout, no failure (FR-11, AC-11).
- **No world under the map screen** — the readout states the values it was last given, which for a
  screen that was given none is the zero state; it does not fail and does not invent one.
- **Cursor outside the map** — the absence marker, never a clamped or negative cell (AC-12).
- **Nothing selected** — the two unit lines are omitted, never drawn empty (AC-8).
