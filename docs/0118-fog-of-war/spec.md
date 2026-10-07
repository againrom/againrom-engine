# Spec — a mission opens on a closed screen

## Terms

**Fog state** — one of three values a map cell holds, per participant:
**unseen** (never looked at), **explored** (looked at once, not looked at now),
**visible** (in some owned unit's sight this instant). A fourth combination is
not reachable: a cell cannot be visible without having been explored.

**Sight predicate** — the existing budget march in `pkg/sim`. It is not a
radius. A seed in 1/(1<<k) cell is spent walking outward one cell at a time,
charged a per-step cost and the destination cell's terrain height and credited
the observer's own height, and a cell is lit while the remainder is positive.
A unit on a hill therefore sees an order of magnitude further than the same unit
in a ditch.

**Reader** — a caller of that predicate. There are two and they differ in three
parameters. The **AI reader** decides which enemies a group may acquire. The
**fog reader** decides what a participant may see. They are one implementation.

**Local participant** — the roster slot the player holds. Already carried across
the render seam.

**Shroud** — how dark a cell is drawn for its fog state.

## Problem

Nothing in this build decides what a player may see. Every structure, static,
sack and unit the map holds is drawn to everyone from the first frame, at every
distance, on ground nobody has walked. A mission cannot be scouted because there
is nothing to scout: the answer is already on the screen.

The sight predicate that would decide it exists and is correct, but it has one
caller, that caller is the enemy-acquisition sweep, and the stamp it builds is
discarded within the same call.

## Scope

Build the second reader, the plane it fills, and the three things a player can
see: a mission that opens closed, a minimap that shows what has been scouted,
and a debug key that opens the whole map at once.

This is a proof of concept graded on reach. Where a decoded clause would cost a
day and buy a difference a player cannot see, it is declared in §Divergences and
not built.

## The contract

**FR-1 — The predicate is exported to a second reader.** `pkg/sim` gains one
exported entry point that answers, for a roster slot, which cells that slot's
living entities can see right now: one byte per world cell, non-zero where lit.
It is the union of one march per living entity of that slot, each at that
entity's own sight range, because a slot sees as one animal.

**FR-2 — There is one implementation, parameterised, and the parameters are
three.** No second march is written. What distinguishes the two readers is
carried as values, not as forked code:

- **the fixed-point shift `k`** — an argument to the window builder, not a
  literal inside it;
- **the in-bounds rectangle** — the AI reader admits a ring cell on the
  rectangle inset 8 cells from every edge; the fog reader admits columns
  `7 .. W−8` and rows `7 .. H−8`. Both live in the same predicate, selected by
  the reader.
- **the seed** — computed once, from a sight radius in units of 1/256 cell.
  Both readers call it. A whole-cell radius `r` passed as `r<<8` yields exactly
  the seed the AI reader used before this story, at every `k`; that identity is
  asserted, not assumed.

The AI reader's answers do not change except through the identity above.

**FR-3 — No code names a sight radius.** Every march is seeded from the
entity's own range, which arrives from the unit definition data for a monster or
a human and from the party hero's derived sheet for the party. A constant
standing in for either is a defect, not a default.

**FR-4 — A participant has a fog plane.** One byte per map cell holding a fog
state, owned by the tier that holds the world and pushed across the render seam
as a plain plane with its dimensions. The renderer does not compute it and
cannot.

**FR-5 — The plane opens closed, only ever grows, and refreshes on a period.**
Every cell starts unseen. The visible layer is recomputed every 32 world ticks
and replaces the previous visible layer wholesale; every cell it lights is also
marked explored, and explored is never cleared. So ground a unit has left stays
lit until the next refresh and then falls back to explored, never to unseen.

**FR-6 — Terrain draws its fog state.** A cell is drawn at full brightness when
visible, at half when explored, and black when unseen. The three levels are
applied as one multiply composed onto the shading each cell already receives, so
nothing about existing shading, day/night tint or geometry moves.

**FR-7 — Drawables are gated on the plane.** A unit not owned by the local
participant is drawn only while its cell is visible; a unit the local
participant owns is always drawn. A sack is drawn only while its cell is
visible. Anything anchored to an unseen cell is not drawn at all.

*Folded from hotfix `637e513` — see `docs/hotfix/ARCHIVE.md#637e513`.* The
rule is the SET: every consumer of the entity snapshot on a draw path asks
the gate, not the two call sites that first did. Panels and what a press may
target are outside it and stay so.

**FR-8 — There is a minimap.** A panel repeating the map's terrain at one pixel
per cell, or at the largest whole-pixel scale that fits its box, showing each
cell in its fog state: unseen black, explored dim, visible full. Units the
player may see are marked — the local participant's in one colour, everyone
else's in another. It is drawn and it is inert: it takes no click and issues no
order.

**FR-9 — There is a debug reveal.** One key makes the whole plane answer
visible, without changing the plane. Pressing it again restores the plane's own
answer. It is a diagnostic affordance and sits with the existing diagnostic
keys, not with the game bindings.

**FR-10 — A mission opens closed by default.** No flag is needed to get fog;
the reveal key is what a developer uses to get rid of it.

**FR-11 — Nothing enters the hashed world.** No field is added to the
simulation's world, its byte form does not change and its version is not
bumped. A world that has never been looked at hashes exactly as it did before
this story.

## Acceptance

**AC-1** With no game install present, `go test ./...` is green.

**AC-2** The exported reader, on a synthetic flat world with one entity of range
`r` at the centre, lights the cell counts the flat-ground law gives: 9, 21, 45,
69, 105 and 145 for `r` = 1..6.

**AC-3** The two rectangles disagree only near an edge. On a synthetic world,
an observer at least 27 cells from every edge produces byte-identical planes
under both readers; an observer inside that band produces different ones.

**AC-4** The seed identity holds: for every whole-cell radius 0..20 and every
shift 1..8, the 1/256-cell seed of `r<<8` equals the whole-cell seed of `r`.

**AC-5** The world byte-form version is unchanged by this story, and the
simulation's world carries no new field.

**AC-6** A plane built from a fresh mission at tick 0 has at least one visible
cell and at least one unseen cell.

**AC-7** Refreshing the plane after a unit has moved away leaves the vacated
cells explored and not unseen, and never reduces the explored set.

**AC-8** The shroud factor is 1 for visible, one half for explored and 0 for
unseen, and a cell's four corner scales are each multiplied by it.

**AC-9** A non-local unit on a cell that is not visible does not reach the
drawn set; the same unit on a visible cell does; a local unit reaches it on
either.

**AC-10** The minimap composer is pure — it takes a plane, the terrain and a
unit list and returns an image, with no window and no engine.

**AC-11** With the reveal on, every cell composes as visible and the plane
itself is unchanged; turning it off restores the previous drawing exactly.

**AC-12** The minimap consumes no click: no input path reaches it.

## Divergences

**D-1 — One fog value per cell, not four.** The original projects fog
per-vertex and rebuilds that projection every frame, so a cell whose four
corners disagree is drawn as a gradient. We store one state per cell and draw
the cell flat. The frontier is therefore a stair rather than a ramp. This is the
largest visible difference this story ships and it is deliberate: the ramp costs
a per-vertex plane, a per-frame rebuild and a second interpolation for an edge
softness, and the closed screen it would soften does not exist yet.

**D-2 — Whole-cell sight, so a hero sees what the simulation grants him.** The
original's fog keeps sub-cell precision where its simulation truncates to whole
cells, and for a hero the two disagree: a sight of 5.996 cells reveals 145 cells
to the fog and admits 105 to the AI. Our entity carries whole cells only, so our
fog and our AI agree exactly — which is the original's own behaviour for every
monster and every human, and differs only for a hero. The seam that would carry
it is FR-2's seed, which already takes 1/256 units; closing this is one entity
field wide.

**D-3 — The reveal is ours and reveals; the original's switch is the inverse.**
The original has a switch that *skips* the periodic clear, so its live layer
freezes and the map only grows. That is not what was asked for. Ours forces
every cell to answer visible and touches no state, so what has actually been
explored is still there when it is switched off.

**D-4 — The minimap is authored.** The original certainly has a second window
whose click dispatches a small cursor set; that this window is a minimap is an
inference from its art names and not established. So nothing here claims to
reproduce it. Its size, placement, colours, scale rule and key are ours, and its
click surface is not built at all.

**D-5 — Structures and statics are drawn on explored ground.** The original
hides *every* drawable whose cell is not currently visible, buildings included.
We hide units and sacks on that rule and keep buildings and scenery once
explored. A town that vanishes when the camera turns away reads as a defect to
anyone playing this build, and the fidelity fix is one predicate, later.

**D-6 — The plane does not survive a save.** It is not simulation state and
nothing persists it. Reloading a mission opens it closed again.

## Out of scope

- The minimap's click surface, and the view-centring message behind it.
- Shared vision between participants; in single player exactly one slot's units
  reveal, the local one.
- The per-object shadow-and-body cull, and the ordering rule that puts the
  shroud sweep after every sprite.
- Preserving the map's own fog bits at load; our plane is built, not read.
- Any change to what the AI can acquire.
