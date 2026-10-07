# Spec — a dialog window stops the world

**Intensity: spec-anchored / static. Terrain: brownfield** for the map screen's advance and for the
cadence the three speed keys select — both already ship the behaviour this changes — and
**greenfield** for the dim.

A **notice** is the box of text the map screen draws over the map: one part of a mission's event
text, or the sentence reporting how the mission ended. Today the world runs behind it at full rate,
nothing behind it is darkened, and every map-screen key including the three cadence keys still acts
while one is up.

This is the product's author ruling on what we ship: **there must be a pause and a dimming on any
dialog window.** It supersedes the clause of the shipped contract that says a notice "does not stop
the map being advanced". What the original engine does with its own clock and its own backdrop while
a window is up is not asserted here, in either direction.

## Functional requirements

- **FR-1 — an open notice suspends the world.** From the first frame a notice is drawn on until the
  frame it is dismissed on, the world behind it does not advance: no tick fires, no queued order is
  applied, and nothing the world animates by its tick moves. **The granularity is the frame**, and
  that is the requirement rather than a concession to it — a player observes the world one drawn
  frame at a time, so "the world is not moving while the box is up" is a statement about the frames
  the box appears on. The advance that raised the notice is not required to abandon the world time
  it was already asked for; every advance after it runs nothing.
  The map is still drawn, and every map-screen input except the three
  named in FR-5 answers exactly as it does today — the camera pans, the selection moves, the
  diagnostics toggle, and an order issued under an open notice is queued and applied by the first
  advance after the notice is dismissed rather than being refused.

- **FR-2 — dismissal resumes without repaying.** The time that passes while a notice is open is
  spent, not owed. The first advances after a notice is dismissed run no more world time than the
  same number of frames would have run had no notice ever opened — a notice held for an hour and a
  notice held for a frame resume identically. **The held span is the whole of what this covers.**
  The dismissing frame itself is not part of it: one of the three dismissal inputs is answered
  before the map screen's own frame is reached and so asks for no advance at all, and that one
  frame's time is paid by the frame after it. It is one frame, once per dismissal by that route,
  and it is the same one frame on a map screen that never suspended anything.

- **FR-3 — the player's own pause survives untouched.** A notice suspends the world *in addition to*
  whatever the player selected; it never changes what the player selected. A world the player had
  paused before a notice opened is still paused after it is dismissed, and a world the player had
  running is running again. This holds however many notices open and close in between.

- **FR-4 — it is a property of a notice being open.** Every requirement in this contract without
  exception holds for every kind of notice — the dialogue and the outcome alike — and for any kind
  added later. Not one of them is a function of which kind is showing.

- **FR-5 — the cadence keys are inert while a notice is open.** The pause key and the two speed keys
  change nothing at all while a notice is showing: not the world, not the rate, and not the setting
  they would otherwise change for later. When the notice is dismissed the cadence is the one that
  was in force when it opened. Every other map-screen input is unaffected.

- **FR-6 — the map behind a notice is dimmed.** While a notice is open the whole of the map picture
  is darkened uniformly, and the dim appears and disappears with the notice.

- **FR-7 — the dim is under everything the front-end draws over the map.** The notice itself, the
  unit information panel and the debug readout are drawn at their ordinary brightness over the
  dimmed map, and those three are the **whole** of what is exempt — there is nothing else the front
  end draws over the map. Everything else dims together: terrain, the art planes, every overlay and
  every route line, so the dim reads as one darkened picture rather than as a filter over selected
  layers.

- **FR-8 — the dim is a supplied value.** It is **one colour, its transparency carrying its
  strength** — not a strength beside a switch — and the map screen is given it, replaceable without
  changing anything else. A map screen supplied a fully transparent one draws no dim while
  suspending, resuming and preserving the player's pause exactly as FR-1 to FR-5 require.

## Acceptance criteria

| # | GIVEN | WHEN | THEN | level |
|---|---|---|---|---|
| **AC-1** | a map screen advancing normally | a notice opens and frames keep being driven | not one advance from the frame after the notice opened onward reaches the world, until it is dismissed | unit |
| **AC-2** | a notice held open across many frames' worth of elapsed time | it is dismissed and frames continue | the advances that follow run the same world time as the same frames run with no notice, plus at most the one frame FR-2 names for the route used — the HELD SPAN is never repaid, at any length | unit |
| **AC-3** | the player has paused the world, then a notice opens | it is dismissed | the world is still paused; and the same drive with the player *not* paused leaves the world running | unit |
| **AC-4** | an outcome notice rather than a dialogue one | frames are driven while it is open | the world is suspended and the map is dimmed exactly as under a dialogue notice | unit |
| **AC-5** | a notice is open | the pause key and both speed keys are pressed | nothing reaches the world's cadence, and after dismissal the rate and the pause setting are the ones that were in force when the notice opened | unit |
| **AC-6** | a notice is open | the frame is composed | the frame carries one dim covering the whole drawable area, composed after the last of the map picture and before the unit panel, the debug readout and the notice | unit |
| **AC-7** | no notice is open | the frame is composed | no dim is composed at all, and the frame is the one composed before this story; and with a **fully transparent** dim supplied, a frame composed with a notice open is that same frame while the world is still suspended | unit |
| **AC-8** | a map screen whose lettering failed to load, so no notice can be drawn | a notice is pushed to it and frames are driven | the world is **not** suspended, nothing is dimmed, and the screen behaves exactly as it does today | unit |
| **AC-9** | a map screen the loader put no cadence under | a notice opens and is dismissed | nothing crosses, nothing panics, and the screen is otherwise unchanged | unit |

AC-8 is the error case: a notice that cannot be seen must not stop a world the player is looking at
or darken a map they cannot un-darken, for the same reason it already must not swallow the key that
leaves the screen.

## Derived properties

- **P-1 — negative invariant.** For any map screen that cannot draw a notice, no advance is
  suspended and no pixel is darkened, however many notices are pushed to it.

- **P-2 — invariant.** The player's own pause setting is changed by the pause key and by nothing
  else. No notice opening, no notice closing and no mission ending writes it.

- **P-3 — invariant.** Nothing here reaches simulation state. No world field, no byte-form record,
  no serialized version and no digest gains a member or changes value, and two worlds driven the
  same number of ticks agree exactly as they did before this story.

- **P-4 — idempotence.** A frame that changes neither the cadence the player selected nor whether a
  notice is open declares nothing: the world is told about its cadence when that cadence changes and
  on no other frame, however many frames pass.

## Constraints

The suspension must be expressible without adding state to the world, because a suspension is a
statement about what the front-end asks for and not about what the world is.

| | A: the front-end stops asking for advances | B: the front-end declares a stop | C: the world learns about notices |
|---|---|---|---|
| resume | pays back the elapsed span up to the catch-up bound — a visible jump | nothing owed; the span is consumed as it passes | nothing owed |
| the world's own diagnostics while suspended | frozen at the last frame that asked | reported as stopped, live | reported as stopped, live |
| reach | none | none — nothing new crosses between the front-end and the world | the world gains a concept of the front-end's boxes |

**B.** A is ruled out by measurement rather than by argument, and C would put a front-end concern
behind the determinism wall for no observable gain.

## Out of scope, and disclosed

**The water keeps moving.** This tree's pause stops the world and not the map's own animation, and a
notice's suspension is that same pause; so water still flows behind a dimmed, stopped map. It is a
divergence this story inherits rather than one it introduces, and it is not closed here.

**Nothing is claimed about the original.** Neither the pause nor the dim is offered as fidelity. The
shipped behaviour is this product's, by its author's ruling, and a later answer about the original
is expected to change the dim's value or remove the suspension — neither of which is a change to
anything in this contract but FR-6's strength.

**The dismissing frame's own time is not this story's to reclaim.** One dismissal route is answered
where its key is read, before any per-screen dispatch, and returns there — so that frame asks for no
advance and its time is paid by the next one. It predates this story, it is unchanged by it, and it
is measurable on a map screen the loader gave nothing to suspend. Making the three routes identical
would mean reshaping an arm this contract does not otherwise touch, for one frame of world time.

**The advance that raises a notice is not cut short.** A single advance may be asked for several
ticks' worth of elapsed world time at once, and the tick that raises a notice can be any of them; the
rest of that one advance still runs. It is bounded — never more than the one catch-up span an advance
may ever run, and zero on any advance that ran at most one tick, which is every advance at the
shipped cadence and an ordinary frame rate. It is not observable as motion behind an open box: the
first frame the box is drawn on already has all of it applied, and every frame after it is still.
Cutting the advance short would put a second thing that decides "do not advance" beside the one this
contract has, on the far side of the seam, for a difference no frame shows.

**No fade.** The dim appears on the frame the notice appears on and goes on the frame it goes. An
eased transition is a second clock on the draw path and buys nothing a still box needs.

**The notice's own appearance is untouched.** Its geometry, its palette, its wrapping, its button
and its three dismissal routes are exactly what they were; this story adds a backdrop behind it and
changes nothing in front of it.

**A second notice while one is open is still discarded.** That is the shipped rule and it is
unchanged: at most one notice is ever outstanding, so "a notice is open" never means two, and no
stacking, queueing or deferral is introduced here.

**The menu and the map list gain nothing.** Neither draws a notice and neither has a world to
suspend, so neither dims and neither pauses.

## Verification mapping

Every AC above is CI-automatable and none needs a live run: the front-end's whole map arm, the
cadence it declares and the frame it composes are all reachable without a window or a game install.
One manual observation is recorded beside them — a campaign mission played to a dialog window, which
is where the ruling came from and the only place the dim can be judged rather than measured.

## Gate check

FR-1 → AC-1, AC-4, AC-8, AC-9, P-1. FR-2 → AC-2. FR-3 → AC-3, P-2. FR-4 → AC-4. FR-5 → AC-5, P-2,
P-4. FR-6 → AC-6, AC-4, AC-7. FR-7 → AC-6. FR-8 → AC-7. P-3 covers every FR at once: none of them is
permitted to reach the world. AC-9 pins FR-1 where the thing the suspension is declared through is
absent — the screen must be unchanged rather than merely not crash.
