# Spec — a popup takes everything onto itself

**Intensity: spec-anchored / static. Terrain: brownfield** throughout — all four behaviours below
already ship, and this changes what they are.

A **popup** is a box the map screen draws over the map and holds the player in until it is dismissed.
This tree has exactly one today — the **notice**, the box of text carrying a mission's event line or
the sentence reporting how the mission ended. The author has ruled that the in-game menu, when it
exists, is a second one and must behave identically, so this contract is written about popups rather
than about notices.

Today a popup stops the world and darkens the map, and that is all it stops. Behind an open box the
river still flows and the structures still animate; the map still pans, edge-scrolls, drags and
zooms; a click still selects, a rectangle still sweeps and an order still queues. The darkening
already covers the whole drawable area; the unit information panel and the debug readout stand at
full brightness over it because they are drawn **after** it, and that draw order — not the
darkening's extent — is what exempts them.

This is the product's author ruling on what we ship: **a popup intercepts everything.** It supersedes
four clauses of the shipped contract — the panel and the readout being exempt from the dim, every
map-screen input except three still reaching the map, the camera still panning, and the water still
moving behind a suspended world.

## Functional requirements

- **FR-1 — the requirements here belong to "a popup is open", not to a kind of popup.** Every
  requirement in this contract holds for every popup the map screen shows, and none is a function of
  which popup it is. **A popup added later inherits the whole of this contract by raising one
  answer**, and a second condition beside that answer — for the dim, the input gate, a clock or the
  camera — is a defect against this requirement even where every behaviour it produces is correct,
  because it is how two of them come to disagree.

- **FR-2 — a popup takes every map-screen input except the three that dismiss it.** While a popup is
  open no map-screen input reaches the map or the world: no selection, no selection rectangle, no
  order of any kind — including kinds added after this contract — neither debug blow key, neither
  diagnostic toggle. The three that still answer are the popup's own dismissal routes — Return,
  Escape, and a click on its button — and they answer exactly as they do today. An input made while a
  popup is open is **discarded and not queued**: nothing pressed at a box is replayed at the map
  behind it when it goes. **The frame a popup is raised on is already intercepted**, not the one
  after it.

- **FR-3 — the map cannot be moved while a popup is open.** Neither the view's position nor its scale
  changes: not by the pan keys, not by the cursor at a window edge, not by a drag, not by the wheel.
  The map is exactly where and how it was on the frame the popup opened, on every frame it is open,
  and on the frame it is dismissed. A pan key still held across a dismissal pans again on the frame
  after it: a pan key is a held level, not a press.

- **FR-4 — a gesture in flight when a popup opens is abandoned.** A press or a drag the player had
  begun is dropped rather than held: no selection rectangle is drawn on any frame a popup is open on.
  Nothing is unwound, because a gesture still in flight has decided no selection — a selection is
  made when a gesture ends. A button still held across the dismissal starts a **fresh** gesture where
  it then is and never resumes from where it was pressed; the click that dismisses a popup by its
  button does not also begin one.

- **FR-5 — every animation in the map picture holds still.** While a popup is open, nothing the map
  screen draws of the map changes by the passing of time: not the water, not an animated object or
  structure, not a unit's frame, not a unit's position between two ticks, not a route. **The held
  time is spent, not owed** — an animation resumes at the phase it held, and the frames after
  dismissal advance it by their own elapsed time alone.

- **FR-6 — the dim covers everything but the popup.** Every surface the front end draws over the map
  darkens with it — the unit information panel and the debug readout included. **The popup itself is
  the whole of what is exempt.** The darkening's strength, **its rectangle** — already the whole
  drawable area and not widened here — the frames it appears and disappears on, and its being one
  supplied colour whose transparency carries its strength are each what they were. What moves is
  which surfaces stand over it and which under it.

- **FR-7 — a map screen that cannot draw a popup is untouched by every requirement above.** Where the
  lettering failed to load no box can be drawn, so there is no popup: nothing is intercepted, the
  camera moves, every animation runs, and no pixel is darkened, however many notices are pushed to it.
  The same holds for the standalone developer viewer, which has no popups at all.

- **FR-8 — the suspension and the world seam do not move.** The world is still suspended and still
  resumes without repaying the held span; the player's pause setting still survives a popup; the
  three cadence keys are **already** inert while one is open; and the map screen's arm still makes
  its one advance on every frame that reaches it, suspended or not — **a frame dismissed by Escape
  does not reach that arm**, exactly as today. Every part of this ships and is stated only as the
  requirement that this story moves none of it.

## Acceptance criteria

| # | GIVEN | WHEN | THEN | level |
|---|---|---|---|---|
| **AC-1** | a popup open, units already selected | a press, a drag and a release away from its button, and a secondary press | the selection is the one held when the popup opened, and nothing reaches the world's order seam | unit |
| **AC-2** | a popup open | both debug blow keys and both diagnostic toggles — the cell lattice and the readout — are pressed | nothing reaches the world's blow seam, and neither diagnostic changes what the frame draws | unit |
| **AC-3** | a popup open, and the same drive with none open | each pan key, a cursor inside the edge margin, a Shift-drag — the map screen's own pan gesture — and a wheel notch | the camera's position and scale are unchanged on all four while the popup is open, and each of the four moves the camera when none is | unit |
| **AC-4** | a drag latched as a selection rectangle and moved past the tap slop | a popup opens, frames run, it is dismissed with the button still held | no rectangle is drawn on any frame the popup is open on, and the frame after dismissal anchors a new gesture rather than continuing the old | unit |
| **AC-5** | a popup open across many frames' worth of elapsed time | it is dismissed and frames continue | the ambient counter the water and the animated objects and structures are selected at is the one it held when the popup opened, and the frames after dismissal advance it by exactly their own elapsed time — the held span is never repaid, at any length | unit |
| **AC-5a** | a popup open over a world holding walking units under orders | frames are driven across many frames' worth of elapsed time | every unit's frame, every unit's drawn position including its displacement between two ticks, and every route are the ones drawn on the frame the popup opened — asserted over the drawn state, not over a counter | unit |
| **AC-6** | a popup open | the frame is composed | the frame carries one dim over the whole drawable area, composed after the last of the map picture, after the unit information panel and after the debug readout, and before the popup | unit |
| **AC-7** | a map screen whose lettering failed to load, and the standalone developer viewer | notices are pushed and frames are driven | nothing is intercepted, the camera answers every one of AC-3's four inputs, the ambient counter advances, and no dim is composed | unit |
| **AC-8** | a popup open | each of its three dismissal inputs is made | each advances it exactly as it did before this story | unit |
| **AC-9** | no popup open | frames are driven and composed | the advance, the camera, the ambient counter, the gesture, the orders, the blows, the diagnostics and the composed frame are each what they were before this story | unit |
| **AC-10** | a popup open, over a world the player had paused and over one they had not | frames run and it is dismissed | the world is advanced once on every frame that reaches the map screen's own arm, and told it is stopped; and after dismissal the pause setting is the one in force when the popup opened | unit |

AC-7 is the error case: a box that cannot be seen must not take the player's inputs, stop the picture
they watch, or darken a map they cannot un-darken.

## Derived properties

- **P-1 — negative invariant.** For any map screen that cannot draw a popup, no input is intercepted,
  no clock is held, no camera is pinned and no pixel is darkened.

- **P-2 — completeness.** Every way the **player** has of moving or scaling the view is covered by
  FR-3: there is no fifth player input to the camera, and none that does not pass what FR-3 gates.

- **P-3 — invariant.** Nothing here reaches simulation state. No world field, no byte-form record,
  no serialized version and no digest gains a member or changes value, and two worlds driven the same
  number of ticks agree exactly as they did before this story.

- **P-4 — idempotence.** A frame on which no popup opens or closes declares nothing new about the
  cadence: the world is told about its stop when that stop changes and on no other frame.

## Constraints

FR-1 is the one boundary this contract fixes on the solution, and it is stated there. The
alternatives to it — a condition per behaviour, or a modality type a popup installs — are internal
wiring rather than an external contract, and they are not this document's to weigh.

## Out of scope, and disclosed

**The in-game menu is not built here.** The author names it as the second popup, and this tree has
none: leaving the map goes to the main menu rather than opening a box over it. Building one is a
later story, and FR-1 is the whole of what this story owes it.

**The player's own pause still leaves the water running.** The previous contract disclosed that this
tree's pause stops the world and not the map's own animation. FR-5 closes that for a **popup** and
for nothing else: a world paused with the pause key still has water flowing over it. The divergence
is narrowed, not closed.

**The debug readout's own numbers are not frozen.** It is an instrument of the front end rather than
part of the map picture, so the frame rate it reports keeps counting behind the dim.

**No fade, and no second dim.** The dim appears on the frame the popup appears on and goes on the
frame it goes; one dim is composed over a frame, never two.

**The popup's own appearance is untouched.** Its geometry, palette, wrapping, button and three
dismissal routes are exactly what they were.

**At most one popup is ever outstanding.** A second notice arriving while one is open is discarded —
the shipped rule, unchanged — so "a popup is open" never means two, no queue exists to drain, and no
frame falls between one closing and another opening.

**The engine margin darkens with the map**, as it already does: it is a brightness the terrain cells
carry rather than a surface over the map, so FR-6 does not reach it.

**Re-fitting the view to the window is not the player moving the map.** The view is clamped whenever
the window's size is declared, and that clamp can move the view's origin when the window is resized
while a popup is open. A view that stopped re-fitting would be a broken picture, not an intercepted
one.

## Verification mapping

Every AC above is CI-automatable and none needs a live run or a game install: the map arm, the camera,
the map picture's two time sources and the composed frame are all reachable with no window. One
manual observation is recorded beside them — a campaign mission played to a dialog window — because
whether the result reads as one intercepted screen is a judgement, and this contract does not make it.

## Gate check

FR-1 → AC-3, AC-6, AC-7, P-1 (each asserted through the same single answer, and AC-7 is what fails
first if a second one appears). FR-2 → AC-1, AC-2, AC-8, AC-9. FR-3 → AC-3, AC-7, AC-9, P-2.
FR-4 → AC-4. FR-5 → **AC-5 and AC-5a together**, plus AC-7 and AC-9: the map picture has two
independent time sources, and an assertion over either alone certifies a build in which the other
still moves. FR-6 → AC-6, AC-7, AC-9. FR-7 → AC-7, P-1. FR-8 → AC-8, AC-10, P-4. P-3 covers every FR
at once: none is permitted to reach the world.
