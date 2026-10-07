# Spec — the mission opens where the party lands

Intensity: **spec-anchored / static**. Terrain: **brownfield** in the map view and its camera (both
exist and their present behaviour must not move where this contract does not move it); **greenfield**
for the authored extent and for the mission door's use of it.

## Why

A campaign mission opens at the map's top-left corner at native scale, on every map, whatever the map
measures and wherever the party was put. On an 80x80 map with the party dropped near the middle the
player is shown a corner of empty ground and has to hunt for his own units before he can give an
order. The view has no relation to the mission it is showing.

Two things are wanted of the opening view, and they are not the same kind of requirement. **Where**
it sits is settled by what the mission's own start already decided and reported. **How much** it
shows is settled by nobody: nothing decoded states what the original set its view to at map load, or
how many tiles its viewport spanned. The first is built from evidence; the second is authored, and
this contract says so rather than letting a chosen number pass for a recovered one.

## Requirements

**FR-1 — a mission opens on the party.** When a campaign mission is opened, the map view is
positioned so that the cell the party landed on is at the centre of the view.

**FR-2 — the anchor is read, never re-derived.** The centred cell is the one the mission's start
reports for the **first party member**. Where the start placed no members, it is the cell the start
reports having decided on. Neither is recomputed from the map: the map's start table is not consulted
again, and no choice the start made is repeated.

**FR-3 — the visible extent is authored, and stated once.** The view spans a fixed number of map
**columns** when a mission opens. That number is stated in exactly one place, is identified in the
source as this project's own choice rather than a reproduction of anything, and is the only edit
required to change what a mission opens showing.

**FR-4 — the extent is a proportion, not a pixel count.** The same authored column count is shown
whatever size the view is. The number of **rows** shown is whatever the view's own proportions give
at that scale, and is not fixed by this contract.

**FR-5 — it is applied against the view size actually in force.** The start view takes effect using
the view size the window has, not the size assumed when the mission was built. It is applied **once**
per opened mission.

**FR-6 — after it is applied, the view is the player's.** Nothing moves or rescales the view again
on its own: no later frame, no world event, no change of selection, and no change of view size
re-establishes the start view.

**FR-7 — a view that was never given a start cell is unchanged.** A map opened without one — the
standalone developer viewer, and the map picker's own path — opens exactly as it did before this
contract, at the same position and the same scale.

**FR-8 — the start view obeys the view's existing bounds.** A start cell near a map edge does not put
the view outside the world, and an axis whose world is smaller than the view is centred on that axis
as it already would be. The scale stays inside the range the view already permits; where the authored
extent would demand a scale outside it, the nearest permitted scale is used and the view is still
centred on the start cell.

**FR-9 — nothing simulated moves.** No field is added to, removed from or retyped in any simulation
state; the canonical byte form and its version are unchanged; the digest over a given world is
unchanged; and a headless drive of a mission reaches the same outcome on the same tick as before.

## Acceptance criteria

| | Given | When | Then |
|---|---|---|---|
| **AC-1** | a map larger than the view, and a start cell away from its corner | the mission's view is opened and a view size adopted | the start cell's centre is at the centre of the view, on both axes |
| **AC-2** | the same map with two different start cells | each is opened | the two views differ, and each is centred on its own cell |
| **AC-3** | a start reporting a party of several members | it is opened | the view is centred on the **first** member's cell, and on no other member's |
| **AC-4** | a start reporting no members at all | it is opened | the view is centred on the cell the start decided on, and the opening does not fail |
| **AC-5** | a map with a height-displaced surface | it is opened | the cell is centred where that cell is **drawn**, carrying the same vertical displacement every other mark on that cell carries — not where a flat lattice would put it |
| **AC-6** | any opened mission and any view size | the view is measured | the number of whole map columns it spans is the authored count, and that count is obtainable from the one place that states it |
| **AC-7** | two view sizes of different width | a mission is opened at each | both span the authored column count; the number of rows differs |
| **AC-8** | a mission opened before the window's size is known, then a size adopted | the first frame is composed | the view is centred and scaled for the size actually adopted, not for the size assumed at build time |
| **AC-9** | an opened, applied start view | the view is driven for further frames, and the view size is then changed | the position and scale change only as panning, zooming and the existing bounds change them; nothing re-centres |
| **AC-10** | a start cell in a map's extreme corner | it is opened | the view lies wholly inside the world, and its edge is against the world's edge rather than beyond it |
| **AC-11** | a map narrower than the view at the authored scale | it is opened | that axis is centred on the world, exactly as the existing bound does it |
| **AC-12** | a viewer built and never given a start cell | a view size is adopted and frames are composed | its position and scale are the ones it had before this contract |
| **AC-13** | a mission driven headlessly to its outcome, before and after this contract | the drive is run | the outcome and the tick it is reached on are identical |
| **AC-14** | a world before and after this contract | it is encoded and hashed | the byte form's version, the encoded bytes and the digest are identical |

## Properties

**P-1 — invariant: a start view is a bounded view.** Applying a start view leaves the view inside the
bounds every other view movement is held to. There is no cell, no map size and no view size for which
it escapes them.

**P-2 — negative invariant: no view moves itself.** A map view that was not given a start cell never
changes its own position or scale. Every change to either comes from the player or from a change of
view size.

**P-3 — invariant: one authored place.** The visible extent at mission start is a function of one
stated value. Changing that value changes what every opened mission shows; nothing else in the tree
states or duplicates it.

## Divergence, disclosed

**The visible extent is this project's, not the original's — verdict AUTHORED.** Nothing decoded
publishes the original's view position at map load or its viewport's extent in tiles. The original
also shipped a single fixed screen while this view scales continuously, so there is no pixel answer
to recover: only a ratio to choose. The authored number is the count of native map cells that fit
across the original's whole 640x480 frame — an **upper bound on** what its map area could have shown,
since that screen also carried a panel whose geometry is not decoded, and therefore not a measurement
of what it did show. It is named as ours in the source, and it is the seam a later measurement
replaces.

**The start position is the loader's report and not the original's arithmetic.** The original picks
its drop cell at random from the map's table, seeded from a clock; this tree's start already diverges
there in a way an earlier story recorded, and this contract inherits that divergence rather than
adding to it. It reads what the start decided; it does not decide again.

## Out of scope

- **The minimap** — it shows the whole map by definition, so the view's position is not its question.
- **The right-hand panel's ornament** — the original's battle-screen furniture is undecoded, and
  building it would be inventing the very geometry this contract declines to invent.
- **Edge-scrolling** — an input behaviour, already present, and unrelated to where a view begins.
- **A follow-the-selection camera** — it would contradict FR-6, which is the requirement that keeps
  the view the player's after it opens.
- **Any change to how scaling itself works** — the permitted range and the step are not touched; this
  contract only chooses a value inside the existing range.
- **The map picker's and the standalone viewer's opening view** — FR-7 requires them unchanged.

## Traceability

| Requirement | Criteria |
|---|---|
| FR-1 | AC-1, AC-2, AC-5 |
| FR-2 | AC-2, AC-3, AC-4 |
| FR-3 | AC-6 |
| FR-4 | AC-6, AC-7 |
| FR-5 | AC-8 |
| FR-6 | AC-9 |
| FR-7 | AC-12 |
| FR-8 | AC-10, AC-11 |
| FR-9 | AC-13, AC-14 |
| P-1 | AC-10, AC-11 |
| P-2 | AC-9, AC-12 |
| P-3 | AC-6, AC-7 |
