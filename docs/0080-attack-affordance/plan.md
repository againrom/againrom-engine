# Plan — the attack affordance

## Approach

Three slices, in dependency order: the MODE gains a second writer and two ways of being lowered; the
FRAME gains two things it draws from that mode; the INSTALL gains one picture it hands over. Each is
a commit and each is testable with no window.

Nothing in `decide` changes. The consuming press already reads one flag and branches on it, and
FR-6's whole content is that it keeps reading one flag.

## Design decisions

- **DD-1 — the mode is ONE predicate over TWO fields, and the fields are not merged.**
  `armed` stays exactly what 0075 made it: the key's toggle, raised through the ownership gate,
  lowered by the consuming press. A second field beside it holds the modifier latch. A method over
  the two is what every reader asks (FR-6, P-2).

  Merging them into one field was rejected and the reason is the spend: a press clears the key's
  mode, and clearing a field the modifier also writes would lower a mode the player is still holding
  the key for — the modifier would then be a toggle that a press cancels, which is precisely the
  shape FR-1 says it is not. Kept apart, `command`'s existing one-line clear is correct unedited and
  the latch is a level nothing but the level writes.

- **DD-2 — the modifier is a LEVEL in the FRONT-END's input snapshot, not in the viewer's.**
  The viewer's `Input` carries the camera's levels and is read by the standalone developer viewer
  too. Putting the modifier there would give that viewer a mode, and AC-12 would become a rule
  somebody keeps. In the front-end's snapshot, read on the map arm alone, "the developer viewer has
  no attack pointer" is a property of where the read stands — the same argument the cadence keys,
  the blow keys and the arming key already stand on (FR-1, AC-12).

  It is a level and not a press edge for FR-1's own reason: an edge would need a toggle behind it,
  and a toggle is the thing being replaced. Its zero value is "not held", so every existing literal
  in the suite means what it always meant.

- **DD-3 — focus is read as its NEGATION, at the top of the front-end's step, before the screen
  switch.** Stored as "unfocused" so that FALSE is the zero value: a snapshot literal that does not
  name it is a focused tick, which is what every literal written before this story means. Stored the
  other way up, every one of them would clear the mode and the shipped suite would fail for a reason
  that has nothing to do with focus. This is the readout flag's own inversion and its own reason.

  It is read before the screen switch, so FR-2's "on every screen and whatever else that tick is
  doing" is where the statement stands rather than a case list. It is above the Escape branch for
  the same reason: Escape returns, and a clear below it would be skipped on the tick it fires.

- **DD-4 — a popup lowers the mode inside the VIEWER's step, in the return that already drops the
  gesture latches.** That return is documented as the place where a popup drops what a gesture was
  holding, and the mode is exactly that kind of state. Putting it there also settles the ordering
  for free: the front-end's map arm runs the viewer's step FIRST and its own popup gate SECOND, so
  on a popup frame the mode is lowered before the gate returns and the modifier is never read at
  all — the raise cannot outrun the clear (FR-7, AC-9).

  Rejected: a clause in the front-end's popup gate. That gate is one early return that deliberately
  names no input, and naming one there is the defect 0077 built it to prevent.

- **DD-5 — what is drawn is decided by a PURE method and drawn by a statement that decides
  nothing**, which is the shape the unit panel, the readout and the notice already have. The method
  answers with the picture, where it goes, and whether there is one at all; the draw statement makes
  the engine calls. That is what puts AC-5, AC-6 and AC-9 inside a test with no window.

- **DD-6 — ONE predicate hides the system pointer and draws ours.** The engine's cursor mode is set
  from the very bool that method returns, on the same statement. R-1 is the risk this closes: a
  frame that hides the system pointer and draws nothing is a map with no pointer at all, and it is
  unreachable when one answer drives both.

  The engine call is made only when the answer CHANGES, against a field holding what was last asked
  for. Not for cost — it is a cheap call — but so that a front-end which never draws a map never
  asks the engine for anything, and the developer viewer's frame stays byte-identical.

- **DD-7 — the pointer is drawn LAST, after the notice.** FR-3 wants it over everything, and the
  notice is the only thing already drawn over the two boxes. A pointer under a dialogue box is a
  pointer the player cannot find; and since the mode is down whenever a box is up, the two never
  compete for the same pixels in this build.

- **DD-8 — the draw asks the popup question too, and that is not redundant with DD-4.** DD-4 lowers
  the STATE, which is what makes FR-7's "not restored when it is dismissed" true. The draw's own
  test is what makes FR-7 true on a frame composed without a step having run before it. They protect
  different halves and both are one condition.

- **DD-9 — the marker is the hit test CALLED, never a second rule.** The unit outlined is whatever
  the existing top-of-stack pick names at the current cursor, over the existing per-entity pick
  rectangle. That is what makes AC-8's "it is the id a press at that point would name" a property of
  there being one body rather than a sentence asserted beside two — the same argument that put the
  selecting tap and the attack press on one function in 0075.

  It is a STROKED rectangle, drawn like the route overlay rather than added to the pass slice: a
  pass carries filled rectangles, and a filled one would cover the unit it is pointing at.

- **DD-10 — the picture crosses as ONE `*image.RGBA`, premultiplied.** The tier that owns the
  install resolves the art whole — archive read, container decode, palette, coverage — and the map
  screen receives a picture in the very shape its three existing boxes already upload to the engine.
  So `pkg/ui` gains no format knowledge, no archive handle and no new import, and P-1 holds without
  a package being added to the graph.

  Nil is "none supplied", which is the developer viewer's state and the state of a front-end whose
  install would not yield the art. It is the font's own shape and its own reason.

- **DD-11 — coverage is `(level+1)` in 16 and it is applied at LOAD, not at draw.** The decoded
  model gives the source multiplier at the blit; expressing it as a premultiplied alpha in the
  picture is the same composite performed once per load instead of once per frame, and it is what
  lets the picture go to the engine through the existing upload path with no draw-time option.

- **DD-12 — frame 0 of the ten, top-left at the cursor.** Both are ours and both are recorded as
  ours. The frame choice is forced: nothing read gives an animation cadence, and a made-up one would
  be a moving pointer nobody could justify. The hotspot is a placement, and this one puts the art's
  own point within a few pixels of the pressed pixel, which the system cursor decides regardless.

- **DD-13 — the authored mark is two crossed strokes at the cursor.** It is deliberately not a
  drawn sword: a hand-drawn imitation of art we could not load would be the worst of both, asserting
  a shape while claiming to assert none. Two strokes say "a mode is up" and say nothing else.

- **DD-14 — a cursor that will not load is a REPORTED reason and a running mission.** Same rule the
  font already has on the same startup path: an instrument that fails to load must not cost the
  player the mission. FR-4's authored mark is what makes that survivable rather than silent.

## Risks

- **R-1 — a hidden system pointer and nothing drawn.** The map would have no pointer at all and the
  player could not aim anything, which is worse than the defect this story fixes. Closed by DD-6:
  one bool drives both.
- **R-2 — the art resolves on one root and not the other.** The two installs ship different
  archives. Closed by verifying the load on both roots, and bounded by DD-14 either way.
- **R-3 — the modifier is held while the front-end changes screen.** The mode lives on the viewer,
  which is dropped when the map screen is left, so a map opened later is never armed. That is 0075's
  own answer and this story does not change it.

## Success criteria

1. The mode answers to the modifier as a level, ungated, and to the key exactly as before (FR-1,
   FR-6 — AC-1, AC-2, AC-3, AC-10).
2. Focus loss and a popup each lower it, and neither restores it (FR-2, FR-7 — AC-4, AC-9).
3. The frame draws the picture when there is one, the authored mark when there is not, and neither
   when the mode is down; the system pointer is hidden on exactly those frames (FR-3, FR-4 — AC-5,
   AC-6, AC-12).
4. The marked unit is the unit a press would name (FR-5 — AC-8).
5. The load resolves palette and coverage as decoded, refuses a malformed stream, and a mission
   whose art will not load still opens (FR-4 — AC-7, AC-11).
6. Nothing else changes and the full local gate is clean (FR-8 — P-1, P-3, P-4).
