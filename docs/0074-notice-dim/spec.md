# Spec — the dim behind a notice is the one the original applies

**Intensity: spec-anchored / static. Terrain: brownfield.** The dim already ships; this changes what
it is, and with it what the product claims about it.

A **notice** is the box of text the map screen draws over the map, and while one is open the map
behind it is darkened. Today that darkening is an authored value — black at half opacity — chosen
because nothing established what the original does. What the original does is now established, and
this contract replaces the authored strength with it.

It replaces a **value**, not a mechanism. The darkening's rectangle, the frames it appears on, what
it stands over and under, the seam that supplies it and the suspension it accompanies are all
untouched, and are restated below only as the requirement that they do not move.

## Functional requirements

- **FR-1 — the dim MULTIPLIES the picture behind it and adds nothing to it.** Every channel of every
  pixel it covers is scaled by one common factor strictly between 0 and 1. No channel is raised, no
  hue is moved, a black pixel stays black, and no pixel is turned black that was not black already.
  This is a property of the value itself and not of what happens to lie behind it: it must hold for
  every possible picture, not for the pictures a map happens to produce.

- **FR-2 — that factor is 13/16, to the nearest strength the supplied value can carry.** The
  darkening is 13/16 = 0.8125 of the picture's brightness. Where the supplied value's own precision
  cannot spell that exactly, what ships is **the nearest strength it can spell** — nearest is the
  requirement, and the particular number is a consequence of it. The residual is bounded: across the
  whole range a channel can take, the shipped strength and 13/16 differ by less than **a quarter of
  one step** of that range.

- **FR-3 — nothing else about the dim moves.** The rectangle it covers, the frames it is composed on
  and omitted from, the surfaces it passes under, the fact that it is one supplied colour whose
  transparency carries its strength, and the behaviour of a map screen that cannot draw a notice, are
  each exactly what they were before this story. A frame composed with no notice open, and a frame
  composed under a fully transparent supplied value, are the frames they were.

## Acceptance criteria

| # | GIVEN | WHEN | THEN | level |
|---|---|---|---|---|
| **AC-1** | the value the map screen ships | it is read | all three of its colour channels are zero, which is what makes compositing it a scaling rather than a wash | unit |
| **AC-2** | the value the map screen ships | its strength is compared against every strength the value's type can carry | none of them is closer to 13/16 than the shipped one, and the comparison is made by searching that whole set rather than against a remembered number | unit |
| **AC-3** | the value the map screen ships | its strength is compared with 13/16 | they differ by less than a quarter of one step of a channel's range | unit |
| **AC-4** | the value the map screen ships | it is inspected for the two degenerate strengths | it is neither fully transparent nor fully opaque — it darkens, and it does not hide | unit |
| **AC-5** | a map screen with a notice open | the frame is composed | it carries one dim over the whole drawable area, composed after the last of the map picture and before the unit information panel, the debug readout and the notice — the same rectangle in the same place as before this story | unit |
| **AC-6** | a fully transparent supplied value; a map screen whose lettering failed to load; a map screen with no drawable area | frames are driven and notices are pushed | no dim is composed and nothing else about the screen has moved | unit |

AC-2 is the discriminating one and is written the way it is on purpose. Asserting the shipped number
would witness only that nobody changed it; asserting that no expressible strength is closer
witnesses that it is the **right** one, and it goes on being the right assertion if the precision of
the supplied value, or the factor itself, ever changes.

## Derived properties

- **P-1 — negative invariant.** No composition of the shipped value can brighten a channel, shift a
  hue, or drive a pixel to black. This holds for every picture behind it, not for a sampled set.

- **P-2 — invariant.** Nothing here reaches simulation state. No world field, no byte-form record, no
  serialized version and no digest gains a member or changes value, and two worlds driven the same
  number of ticks agree exactly as they did before this story.

## Constraints

The strength is carried by the transparency of one colour, and that transparency has finite
precision. 13/16 does not land on it. Three ways out:

| | A: the nearest strength the value already carries | B: a wider colour, so the strength lands exactly | C: a strength beside the colour rather than inside it |
|---|---|---|---|
| how far from 13/16 | 0.09% low | ~0.0006% low | exact |
| what the drawn surface can hold | one step in 256 | one step in 256 | one step in 256 |
| the supplied seam | unchanged | a different type at every point that supplies or stores it | two numbers that must agree about one thing |

**A.** B and C buy precision below what the surface being drawn can represent, and they buy it by
changing the seam whose whole purpose is that a later answer moves a value and not a structure. The
residual A leaves is smaller than the error the drawn surface's own precision already carries.

## Out of scope, and disclosed

**Ours is applied per frame; the original applies it once.** The original darkens its screen a single
time, destructively, and the darkening survives because nothing repaints while the box is up. This
front-end repaints every frame, so ours is recomputed from the undimmed picture each time. The
mechanisms differ and the appearance does not — a player is shown one application of the factor
either way — and the difference has no consequence of its own in either direction: ours cannot
compound, and nothing in this tree can re-apply it. What *would* reveal it is a surface that changes
behind an open box, and this tree has one: the water, which keeps animating behind a suspended world.
That is the previous story's disclosed divergence and it is not closed here.

**The original darkens the whole screen; ours stops at three surfaces.** The dim already covers the
whole drawable area, but the unit information panel, the debug readout and the notice itself are
composed after it and keep full brightness. The notice **agrees** with the original, which draws its
own window after darkening. The debug readout has no counterpart there at all. The unit information
panel is a genuine disagreement: in the original it is already on the screen when the darkening runs.
It is not closed here. The ruling this behaviour rests on is that there must be a dimming on any
dialog window — a ruling about whether, not about what it covers — and the reason the three surfaces
were exempted is a product reason that a fact about the original does not contradict: an instrument
must stay readable exactly when a player wants to read it. Changing it is the product author's call.

**The factor is reproduced; the arithmetic around it is not.** The original scales the 5- and 6-bit
channels of a 16-bit surface and truncates; this tree scales 8-bit channels and rounds once. So the
same factor gives different bytes — at 5 bits the original's own full level comes out at 0.8065 of
full against our 0.8118, and its lowest non-black level comes out black. That is the render tier's
colour depth, it predates this story, and every pixel of the map already carries it.

**None of the original's machinery is built.** Whatever table, level ladder or surface-locking the
original uses to achieve a per-channel scaling is its mechanism for the same result. This story takes
the result.

**The engine margin keeps its own strength.** The margin is drawn at half brightness and stays there.
The two are different values with different standing — one reproduces something and one does not —
and making them equal would mean asserting of one what is only known about the other.

**The suspension, the notice and the seam are untouched.** No behaviour of the pause, of the box's
geometry, palette, wrapping, button or dismissal routes, and no shape of the supplying seam, changes
here.

## Verification mapping

Every AC is CI-automatable and none needs a live run or a game install: FR-1 and FR-2 are properties
of a value, and FR-3's are properties of a frame's own decision, all reachable with no window. One
manual observation is recorded beside them — a campaign mission played to a dialog window — because a
strength is judged by eye even when it is measured by arithmetic.

## Gate check

FR-1 → AC-1, AC-4, P-1. FR-2 → AC-2, AC-3, AC-4. FR-3 → AC-5, AC-6. P-2 covers every FR at once: none
of them is permitted to reach the world. AC-4 serves both FR-1 and FR-2 and is listed under both: a
fully transparent value would satisfy FR-1 vacuously, and an opaque one would satisfy neither.
