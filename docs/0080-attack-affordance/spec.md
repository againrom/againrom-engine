# Spec — a player can see that he can fight

**Intensity: spec-anchored / static. Terrain: brownfield** for the map screen's arming path and the
front-end's input snapshot, which both ship today, and **greenfield** for the pointer, the marker
and the cursor load.

An attack order works and is invisible. The mode is raised by a letter key nothing on screen names,
the pointer never changes, nothing marks what a press would hit, and the only statement of the mode
is one row of a diagnostic box that also carries a frame rate and a world digest. The player is
being asked to know.

What is being reconstructed answers differently, and the answer is the pointer: a click becomes an
attack **by the cursor it was made under**, and that cursor is produced by a modifier key held down.
Hold the key, the pointer becomes a sword, and while it is a sword a press is an attack. This story
builds that, keeps the shipped key beside it, and draws what a press would name.

It changes nothing about what an attack IS. The order, the seam, the victim rule, the approach and
the blow are 0075's and 0064's and are untouched here.

## Functional requirements

- **FR-1 — a HELD MODIFIER raises the attack mode, and it is a latch.**
  The map screen reads one modifier as a LEVEL, every tick, not as an edge: the mode is up on every
  tick the modifier is down and down on every tick it is not. There is no toggle in it — releasing
  the key lowers the mode with no second press, and pressing it again raises the mode again.

  **The modifier surface is UNGATED.** No selection, no ownership, no unit class and no capability is
  consulted when it raises the mode. It may be raised over an empty selection and over a selection
  the local participant does not own, and it is then a mode under which a press names orders for
  whatever the selection holds — which is nothing, in the empty case.

  Which physical key it is, is a BINDING and not a contract: it lives in the one place this
  front-end's key bindings live, it is Ctrl (either Ctrl key, the two being one modifier), and
  changing it changes no statement in this document.

- **FR-2 — the mode does not survive the window losing focus.**
  A tick on which the front-end does not hold the window's focus lowers the mode — both the modifier
  latch and the key toggle of FR-6 — on every screen and whatever else that tick is doing. Regaining
  focus does not restore it: the modifier must be pressed again, or the key.

- **FR-3 — while the mode is up, the pointer IS the attack pointer.**
  The map screen draws an attack pointer at the cursor, and the system pointer is hidden underneath
  it, so exactly one pointer is on screen at a time. When the mode is down the system pointer is
  visible and nothing is drawn at the cursor.

  It is drawn OVER everything the map screen composes — terrain, art, every overlay, the route
  strokes, the unit panel and the debug readout — because a pointer that a box can cover is a
  pointer the player loses.

  **The pointer does not change with what is under it.** It is the attack pointer for as long as the
  mode is up, over a unit and over empty ground alike. *This is a disclosed divergence.* What is
  being reconstructed swaps to a SECOND cursor over ground holding no unit and issues a different
  order — a swarm — under it. This build has no swarm order, so a second pointer here would name an
  order that cannot be issued; the press over empty ground does what 0075 already says it does, a
  plain move to the cell.

- **FR-4 — the attack pointer is the GAME'S OWN ART when the front-end was given it, and an
  authored mark when it was not.**
  The tier that owns the install reads the attack cursor's art from the graphics archive, resolves
  its pixels, and hands the map screen ONE picture and nothing else — no archive handle, no frame
  set, no animation. The map screen holds a picture or holds none, and cannot tell where one came
  from.
  - A picture is drawn with its top-left corner at the cursor.
  - No picture is an authored mark drawn at the cursor instead. It is OURS, it asserts nothing about
    the original, and its function is that an install this build cannot read the art out of still
    shows the mode.

  A resolved pixel takes its colour from the file's own palette and its coverage from the pixel's
  own four-bit level, at `(level + 1)` parts in 16; an unpainted pixel is fully transparent. Art
  that fails to load is REPORTED and is not a failure of the map: the mission opens, and the
  authored mark is what the player sees.

- **FR-5 — the unit a press would name is MARKED.**
  While the mode is up, the map screen outlines the unit a consuming press made at the current
  cursor would name as the victim — the same hit test, the same lowest-id tie rule and the same
  refusal of a dead unit that the press itself uses, so what is outlined and what would be attacked
  cannot come apart. It marks at most one unit, and it marks none when the cursor names none.
  The marker is OURS. Nothing decoded describes one.

- **FR-6 — TWO ARMING SURFACES, AND THEY ARE GATED DIFFERENTLY. This is a ruling, not an accident.**
  The key of 0075 FR-1 is KEPT, with its ownership gate and its toggle and its one-press spend
  exactly as they are. The modifier of FR-1 is added beside it, ungated. The mode is up when either
  says so, and the whole of the rest of this front-end asks only whether the mode is up.

  The difference is the decode's, not ours: the ownership gate is what the COMMAND PANEL's arming
  path reads, and the shipped key is this build's stand-in for that path; the MODIFIER path reads no
  player value anywhere, at the hover or at the click. A build that gated both would be inventing a
  refusal the original does not make, and a build that gated neither would be dropping one it does.

  The two spend differently for the same reason: the key's mode is spent by the press that consumes
  it, and the modifier's is not — while the modifier is held, every secondary press is an attack.

- **FR-7 — a popup takes ALL of it.**
  While a popup stands over the map, the modifier raises nothing, the mode is DOWN, no attack
  pointer is drawn, no marker is drawn, and the system pointer is visible. A mode that was up when
  the popup opened is lowered by it and is not restored when it is dismissed.

- **FR-8 — nothing else changes.** The unarmed secondary press, the armed press's two outcomes, the
  selecting tap, the box and its modifier, the two blow keys, the cadence keys, the notice, the unit
  panel, the debug readout's contents and every seam answer exactly as they do today. `pkg/ui` still
  names no simulation type, no simulation state is reached, and the world's byte form and its
  version are untouched.

## Acceptance criteria

- **AC-1** With the modifier down the mode is up; with it up the mode is down; and a tick with the
  modifier down after one with it up raises it again, with no press anywhere.
- **AC-2** The modifier raises the mode over an empty selection and over a selection whose primary
  present unit the local participant does not own — the two cases the key of FR-6 refuses.
- **AC-3** A secondary press made with the modifier down over a drawn unit yields one attack per
  selected present unit naming that unit, and the modifier still being down on the next tick leaves
  the mode up, so a second press yields attacks again.
- **AC-4** A tick on which the window is not focused leaves the mode down, whether it was raised by
  the modifier or by the key, and a following focused tick with the modifier up leaves it down.
- **AC-5** With the mode up and a picture supplied, the frame draws that picture at the cursor and
  nothing else at the cursor; with the mode up and no picture, it draws the authored mark; with the
  mode down it draws neither.
- **AC-6** The system pointer is hidden exactly on the frames the attack pointer is drawn on, and
  visible on every other.
- **AC-7** A `.16a` stream carrying a known palette and known literal words resolves to the palette
  colour at coverage `(level+1)/16`, an unpainted cell to fully transparent, and a stream that will
  not decode yields an error and no picture.
- **AC-8** With the mode up and the cursor over a drawn unit, that unit is marked; where two drawn
  rectangles hold the cursor the marked unit is the lower id, and it is the id a press at that point
  would name; a dead unit under the cursor is not marked; with the cursor over no unit, and with the
  mode down, nothing is marked.
- **AC-9** With a popup open: a tick with the modifier down leaves the mode down; a mode raised
  before the popup opened is down on the first tick under it and is still down after the popup is
  dismissed; and no attack pointer and no marker is drawn while it is open.
- **AC-10** The key of FR-6 arms and spends exactly as it does today, and its ownership gate answers
  as it does today; the modifier does not spend on a press and the key does.
- **AC-11** A front-end whose install has no readable cursor art still opens its mission, reports
  the reason, and shows the authored mark when the mode is up.
- **AC-12** The developer viewer — the one built with no front-end around it — draws no attack
  pointer and no marker under any input, and its frame is unchanged.

## Properties

- **P-1** `pkg/ui` imports the render tier and no other, and names no simulation type; what crosses
  into it for the pointer is one picture.
- **P-2** The mode is one predicate with two writers, so no surface can read a mode a different
  surface raised differently.
- **P-3** The world's field set, its byte form and its version are unchanged by this story, and no
  hashed simulation state is reached.
- **P-4** The full local gate is clean: build, vet, gofmt, the whole test suite, and the three repo
  scripts.

## Out of scope

The swarm cursor and its order; the second and third input surfaces; retaliation and anything an
unordered or a struck unit does; the other two modifier latches and the marquee's own use of Shift;
the command panel itself; the cursor art's animation; the debug readout's contents.
