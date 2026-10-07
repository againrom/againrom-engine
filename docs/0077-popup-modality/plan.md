# Plan — a popup takes everything onto itself

## Approach

One answer, four readers: the dim's composition, the viewer's step (camera and ambient clock), and
one early return on the map arm covering every remaining input — three gates on statements that
already exist, plus a statement move in the draw method. No type gains a field and no seam moves.

## Facts verified during planning

- The map screen's own arm makes exactly one world advance, unconditionally, including on suspended
  frames: the far side consumes its pacing baseline on every call, so the suspended span is spent as
  it passes. The cadence call that declares the stop sits immediately before it. A frame carrying
  Escape is answered before any per-screen dispatch and returns there, so it never reaches that arm
  at all — it asks for no advance, sets no cadence and steps no viewer.
- The far side declines its tick loop while stopped, and everything advanced inside it is already
  still under the shipped suspension: the scene counter a unit's walk, idle and death frame is
  selected at, the odometer those frames are also selected from, the sub-tick displacement pushed
  after the loop, and the routes in the per-tick snapshot. **Unit animation needs nothing.**
- The viewer keeps a second clock the suspension does not reach: a wall-clock accumulator advanced
  from the elapsed time between two frames, from which three surfaces are selected — the water phase,
  the animated static objects and the animated structures. Its baseline is taken **before** its
  enable test, so a disabled stretch is skipped rather than banked. Its catch-up is unbounded, where
  the world's is bounded.
- Four **player** inputs reach the camera — keyboard pan, edge scroll, drag pan, wheel zoom — all
  through the viewer's one step method, which ends in **two** camera calls, a pan taking both pan
  terms summed and a zoom. There is no middle-button and no right-drag pan. Both the map screen and
  the standalone viewer call that method. On the map screen an **unmodified** drag is a selection
  rectangle and pans zero by design — Shift is what makes a drag the pan — and edge scroll is
  suppressed entirely while the primary button is down.
- A **fifth** path writes the camera and is no player input: declaring the window's size re-clamps
  the view, writing position and scale. It is reached on a resize and on the notice's own dismissal
  arm; with the size unchanged it moves nothing.
- The step method is also where the drag gesture is latched — the press point, whether this press is
  a rectangle or a pan, and the slop accumulator, which the gesture resolver **also** writes on a
  press edge. The rectangle's **outline** is drawn from the latched state and does not read the
  resolver's own press latch.
- Selection, the rectangle, the arming key and every order kind pass through the viewer; the
  resolver's pure decision function has one caller, and the map arm calls the resolver once, after
  the step, because the release is judged against a slop the step has just raised.
- A popup is raised **inside** the world advance, from the settle pass inside the far side's tick
  loop — so it is open before the step and the resolver run on that same frame, and no popup can
  arise while one is open, because that loop is what the stop gates.
- The dim is one filled rectangle over the whole drawable area, composed after the last of the map
  picture and before the unit panel, the readout and the notice. Its position among the draw method's
  statements is the whole of the shipped rule about what dims, and a test parses that method to
  assert it; the same test fails if a name it tracks is mentioned by two top-level statements.
- The notice's open test requires a font, which is what makes a screen that cannot draw one behave
  as though nothing were pushed.
- Six comments state behaviour this story supersedes as a settled decision — one each on the ambient
  setter, the cadence stepper, the arm's input rule, the step call site, the lattice toggle and the
  suspension predicate.

## Files to touch

| Path | Intent |
|---|---|
| `pkg/ui/popup.go` | **ADD** — the predicate on the viewer, and the flow's derivation of it |
| `pkg/ui/flow.go` | **MODIFY** — the suspension predicate replaced at both call sites; two docs corrected |
| `pkg/ui/app.go` | **MODIFY** — the arm's early return, the step moved ahead of the diagnostics, three comments corrected |
| `pkg/ui/viewer.go` | **MODIFY** — the camera and latch gate in `step`, the clock gate in `advanceAnimation`, the dim statement moved, one doc corrected |
| `pkg/ui/notice.go` | **MODIFY** — the dim's gate becomes the predicate; its doc follows |
| `pkg/ui/popup_test.go` | **ADD** — AC-1 to AC-5a, AC-7, AC-9, AC-10, P-1, P-2 |
| `pkg/ui/backdrop_test.go` | **MODIFY** — the draw-order assertion amended to AC-6 |
| `pkg/ui/notice_test.go` | **MODIFY** — the superseded "every other input still reaches the map" assertion amended to AC-8's half of it |

## Design decisions

- **DD-1 — the predicate is rooted on the viewer, and the flow derives it.** The viewer holds a
  popup, draws it, draws the dim and owns the camera and the ambient clock; the flow needs the same
  answer plus the screen test it already applies. Rooting it the other way was rejected: the dim and
  the camera would then read a flow the standalone viewer does not have. **Its body is the notice's
  own open test and nothing else**, so the font condition FR-7 rests on lives in one place. The
  notice's own two predicates **stay, and are not duplicates of it**: they serve its three dismissal
  inputs and its advance, a property of *that box* — Return, Escape and a click on its button are not
  what would dismiss an in-game menu. FR-1 forbids two answers to **one** question; these are two.
- **DD-2 — FR-2's gate is one early return on the arm, after the advance and after the step.**
  Everything downstream is covered without naming any of it, which is what blocks an order kind added
  later by construction. The top of the arm was rejected on measurement: it would skip the cadence
  call that declares FR-8's stop, the advance FR-8 keeps unconditional, and the step that holds the
  ambient baseline — each turning the held span into a debt paid in one frame. The cadence gate stays
  above the advance, where 0073 put it: a pause pressed under a popup must not write the setting FR-8
  has the dismissal restore.
- **DD-3 — the viewer's step moves ahead of the two diagnostic toggles**, so the gate is one
  statement rather than two around them. Not observable: neither toggle reads or writes anything the
  step touches, and the resolver stays after the step, where the slop it judges is raised.
  Duplicating the step call inside the gate was rejected as two call sites for one thing.
  **The advance stays ahead of the step, and that order becomes load-bearing**, because a popup is
  raised inside the advance. The two were genuinely independent before this story and the call site
  says so; correcting that comment under DD-9 is the only thing between a future reorder and a silent
  FR-4 failure.
- **DD-4 — the camera gate is an early return inside the step, after the clock call and the cursor
  memo.** Guarding the camera calls instead was rejected twice over: there are **two** of them, so
  one guard would leave the wheel live; and it would stop the camera while leaving the drag anchored
  and the rectangle drawing, so FR-3 would pass while FR-4 failed. The cursor memo stays live — it
  moves no camera, and the readout resolves a cell from it.
- **DD-5 — that gate clears both latches, the drag's and the resolver's press latch.** Clearing the
  drag alone was rejected: the press latch is written by the resolver the arm gate skips, so a press
  taken before the popup opened would outlive it and a later release with no press would be judged a
  tap at a stale point. Both are cleared from the step — a write to the resolver's state from the
  camera method, named here rather than left to be discovered, because the step is the one method
  both entry points call and the only one that runs under the gate. **A button held across a
  dismissal may then draw a rectangle that selects nothing**, and that is accepted: the fresh gesture
  has no press latch so its release decides nothing, it is the shape of a button already held when a
  map opens, and closing it would mean a third latch carried purely to suppress an outline.
- **DD-6 — the ambient clock's gate is a second term on the existing enable test**, after the
  baseline is taken and before the counter is advanced: the one place the shipped code already skips
  a span rather than banking it, so FR-5's "spent, not owed" is inherited. A stop on the ticker type
  was rejected — the type is the standalone viewer's too. **The map picture's other time source needs
  nothing**, everything inside the far side's tick loop being already still, so FR-5 is met by one
  new gate plus an assertion over the drawn state that the second source really is held. Asserting
  the counter alone was rejected as the criterion that certifies half a freeze.
- **DD-7 — the dim moves to immediately before the popup's composition, and its gate becomes the
  predicate.** What darkens stays a property of one statement's position rather than a list somebody
  maintains, so a surface added later dims or does not by where it is drawn. A second dim over the
  boxes was rejected: it would compose the strength twice wherever a box leaves pixels uncovered, and
  the panel's own background does exactly that.
- **DD-8 — the suspension predicate is renamed into the new one rather than kept beside it.** Two
  answers to one question is the shape FR-1 forbids; the rename makes "raise one answer" true of the
  whole contract rather than of three quarters of it.
- **DD-9 — the six superseded comments are corrected in the commit that supersedes each.** A comment
  stating a decision the code no longer takes survives every review, because a reader checks the
  conclusion. Two are load-bearing beyond tidiness: the step call site's independence claim, and the
  ambient setter's, which must now distinguish **two stops** — the type still takes none and water
  still runs under the player's pause; only "under a popup" is false.
- **DD-10 — no new exported API and no version.** The predicate is unexported, so the standalone
  viewer cannot reach it; nothing serialized is touched, so no form version moves.

## Risks

- **R-1 — a concurrent lane is adding a new order kind to the same input path.** Mitigated by DD-2:
  the gate names no order kind and returns before both the resolver and the arming key, so a kind
  added on either side of the merge is covered. A per-kind check would not be.
- **R-2 — gating one statement too many turns the held span into a debt.** Two clocks pay it, and not
  symmetrically: the world's catch-up is bounded and the ambient clock's is not, so it would jump the
  **whole** held span. Guarded by DD-2 and DD-6, and asserted directly rather than inferred.
- **R-3 — the shipped draw-order test pins the superseded rule.** Amended with the contract rather
  than deleted; deleting it would leave FR-6 with no mechanism at all.
- **R-4 — the standalone viewer must not change.** It has no popup, so the predicate is constantly
  false there and every gate is an early return under it. Asserted, not argued.
- **R-5 — the ruling is about how the result reads, which this story cannot verify.** What is
  measurable is measured; the judgement is the author's, against a runnable build.

## Success criteria

| # | Condition | Witness |
|---|---|---|
| **1** | Selection, the rectangle, every order kind, the arming key and both blow keys reach nothing | `TestAPopupTakesEveryMapInput`, `TestAPopupTakesTheBlowsAndTheDiagnostics` |
| **2** | None of the four player camera inputs moves or scales the view, and each does when none is open | `TestAPopupPinsTheCamera` |
| **3** | A rectangle in flight is abandoned and not resumed | `TestAPopupAbandonsTheGestureInFlight` |
| **4** | The ambient counter holds, and advances by its own elapsed time after | `TestAPopupHoldsTheAmbientClock` |
| **4a** | Every unit's frame, drawn position and route holds, asserted over the drawn state | `TestAPopupHoldsTheWorldDrivenPicture` |
| **5** | The dim is composed after the panel and the readout and before the popup | `TestTheDimIsComposedAfterEverythingButThePopup` |
| **6** | A screen that cannot draw a popup is intercepted, held, pinned and dimmed in nothing | `TestAScreenThatCannotDrawAPopupIsUntouched` |
| **7** | With no popup open, every gated path answers as it did before this story | `TestWithNoPopupNothingIsGated` |
| **8** | The advance stays once per arm-reaching frame, the stop crosses, the pause survives | `TestAPopupLeavesTheSuspensionSeamAlone`, and the shipped suspension tests |
| **9** | The three dismissal inputs still advance the popup | the shipped notice tests, amended only where they assert the superseded clause |
| **10** | The gate list is clean and a runnable build exists for the author's judgement | the standing gate list; `builds/0077-popup-modality/` |
