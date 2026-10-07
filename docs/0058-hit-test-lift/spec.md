# Spec — what you box is what you get

Intensity: **spec-first / static**. Terrain: **brownfield** in `pkg/ui` and `pkg/render/terrain` —
shipped selection behaviour changes; **greenfield** for the cell lattice, which is new.

## Problem and current behaviour

On mountainous terrain, dragging a box around a unit selects a **different** unit, roughly two cells
higher up the screen, and the white rim that comes back sits well above the green rectangle that was
drawn.

The two halves of the seam answer different questions about the same pixels. A unit's mark, its
sprite and its health bar are all placed by lifting the cell they stand on by that cell's own
height, so a raised cell is drawn **up** the screen. The pick does none of that: it converts the
gesture's screen pixels to a cell through the flat lattice and compares each unit's stored cell
against the result. On flat ground the two agree. On relief they are apart by that cell's own lift,
and the gap grows with the terrain — which is why nobody saw it until a map with real mountains was
played.

The asymmetry is already asserted in the tree: the marks are documented as going "through the very
lift, camera and cull every glyph takes … so it cannot drift from either", and the pick sits beside
them doing neither. Both cannot be right. **The drawing is right.**

The same defect is in the tap. Its own comment claims the box and a one-cell tap "cannot answer
differently about the very same pixels" — an invariant that is *asserted and not tested*, and that
would be broken by fixing one and not the other.

## What a pick answers

**FR-1 — A unit's hit target is the rectangle its own cell occupies in the view.** It is the
rectangle the selection rim is drawn around: the cell's footprint, carrying that cell's height lift
in displaced mode and none in flat, moved by whatever displacement the entity itself is drawn with
mid-step, and **absent for a cell the view does not reach**. Nothing about the unit's art enters it.

**FR-2 — There is one derivation of a cell's footprint and one of the transform that places it.**
Every consumer — the hit target, the selection rim, the lattice, the blocked tint — composes those
two rather than restating either. This is narrower than "one rectangle": the rim is four strips and
the target is one rect, so they cannot be one call. What is forbidden is a second expression of the
footprint or of the lift, which is exactly how the pick and the drawing came apart.

**FR-3 — A box selects every unit whose hit target the drag rectangle meets.** The drag rectangle is
the closed screen rectangle spanned by the press point and the release point, in either orientation
and at any size, including none. A hit target with no area is met by nothing.

**FR-4 — A tap selects the unit whose hit target holds the point, and clears when none does.** Where
two hit targets hold one point — which relief now makes possible between different cells, as a
shared cell already did — the **lower entity id** wins, decided over the whole set and never by
position in it.

**FR-5 — A tap and a zero-extent box ask ONE hit test of the same pixel.** The tap's answer is the
box's answer reduced: the two agree about whether anything was hit at all, and the tap is the lowest
id of exactly the set the box returns. They do **not** return the same selection where two targets
hold one point — the box takes both, the tap takes one — and the older claim that they did was false
as well as untested.

**FR-6 — A corpse is not a candidate for either**, tested at the hit and not after it, so a tap on a
corpse clears exactly as a tap on empty ground does. A downed unit is taken by both. Unchanged.

## The cell lattice

**FR-7 — The map screen can draw a faint lattice, one outline per cell the terrain pass drew and
the view reaches.** Each outline stands where that cell's own placement stands: it carries the same
lift the rim and the hit target carry, so on relief the lattice deforms with the ground.

It is drawn **first of the frame's overlay passes** — over the terrain and the cell-anchored art,
which are painted before the first pass exists and so are unreachable from underneath, and under
every glyph, sprite, bar and rim. Faintness, not position, is what keeps it readable as a
background: it is translucent, and the selection rim is opaque, twice as thick, and on the same
rectangle.

It is **not drawn where a cell is too small to carry a readable outline.** Below that size it is a
wash over the whole map rather than a lattice, and it is also where its cost lives, the band being
the view measured in cells.

**FR-8 — The lattice is off by default and is toggled by one key.** The key is a press edge on the
map screen alone; a held key toggles once. A viewer never told to draw it produces exactly the frame
it produced before this story.

**FR-9 — The lattice asserts nothing about the original game.** It is an instrument.

## What does not change

**FR-10 — The destination of a move order is unchanged.** A secondary press still resolves the
ground through the flat lattice and carries the error that path has always disclosed. It answers a
different question — *which cell is this pixel on the ground* — and answering it properly needs an
inverse of the terrain's own mesh that nothing here has. **The disclosed consequence is that the two
halves stop being wrong together**: before, aiming high selected the right unit and ordered it to
the right cell; now the selection is right and the destination is still off by the relief.

**FR-11 — Everything else about a gesture is unchanged**: what makes a release a tap or a box, the
slop that decides it, the marquee outline, which frame a press is judged on, the four outcomes and
their order, and the whole of what a right press does once its cell is known.

## Acceptance criteria

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | a displaced viewer, a unit on a cell whose lift exceeds one cell | a box over the rectangle that unit's rim occupies | that unit is selected |
| AC-2 | the same | a box over the cells the unit's **stored cell** flatly resolves to, away from its rim | that unit is **not** selected |
| AC-3 | a flat viewer, units that are **not mid-step** | any box or tap | exactly the units the previous behaviour selected |
| AC-3a | a flat viewer, a unit **mid-step** | a box over where it is drawn | that unit is selected — a deliberate change on flat ground |
| AC-4 | any viewer, any point | a tap at that point, and a box pressed and released at it | the tap is the lowest id of the box's own set, and both agree on whether anything was hit |
| AC-5 | two units whose hit targets share a point | a tap on that point | the lower id, whichever order the snapshot holds them in |
| AC-6 | a box dragged right-to-left and bottom-to-top | released | the same units its corner-swapped twin selects |
| AC-7 | a unit drawn mid-step between two cells | a box over where it is **drawn** | that unit is selected |
| AC-8 | a corpse and a living unit | a box over both | the living one alone |
| AC-9 | a selected unit the view reaches whole | the frame is drawn | the rim's strips span exactly that unit's hit target |
| AC-10 | lattice on, a displaced viewer above the size floor | the frame is drawn | one outline per cell the terrain pass drew **that the view reaches**, each at that cell's own lift |
| AC-11 | lattice off | the frame is drawn | the pass slice is exactly what it was before this story |
| AC-12 | the lattice key pressed | on the map screen | the lattice toggles, once per press |
| AC-13 | a selection, a right press on the ground | the order is issued | the destination cell is what it was before this story |
| AC-14 | lattice on, a cell below the size floor | the frame is drawn | no outline at all |

## Properties

**P-1** — A hit target is a **function of the cell, the entity's displacement and the view alone**:
two entities on one cell, in one frame, neither mid-step, have the same hit target.

**P-2** — For every cell the lattice outlines, the outline and the hit target of a unit **standing
still** on that cell are the same rectangle. A unit **mid-step** is picked between two outlines and
coincides with neither — its target moves with it and the lattice cannot, having no entity.

**P-3** — On a viewer with no altitudes, a hit target is the flat cell rectangle through the camera,
so the relief half of this story is invisible on flat ground. The displacement half is not: it is
not gated on displaced mode and never was.

**P-4** — Nothing here writes to the selection outside the two paths that already replace it whole,
and nothing here reads or writes a world.

## Out of scope

- **The ground pick** (FR-10). Same defect class, different question, and it needs an inverse of the
  drawn mesh. Named as a seam, not deferred silently.
- **Picking a unit by its drawn sprite** rather than by its cell. It is the open question this story
  authors an answer to, and the nearest published evidence points the other way (`provenance.md`).
- Any change to `pkg/sim`, to what a world does with an order, or to the marquee itself.
- Persisting the lattice toggle across screens or runs.

## Error cases

An off-map entity has no hit target and is picked by nothing. A gesture whose corner leaves the
window can no longer reach cells the view does not reach — a narrowing of the previous behaviour,
and the direct consequence of FR-1. A hit target with no area, or one whose extent is not a number,
is met by nothing: a camera whose zoom was assigned directly can produce a finite position with a
zero or non-finite extent, and the hit test answers "no" for it rather than treating an empty
interval as inhabited. A lattice on a viewer holding no projection draws the flat lattice, and one
on a map of no cells draws nothing.
