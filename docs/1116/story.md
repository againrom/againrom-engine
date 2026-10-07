# School class column

Selecting a character of another class rotates the school column through the
installed frames instead of cutting between two static faces. The character
pane changes immediately. Training controls follow the displayed endpoint.

## Contract and authority

- Cache all sixteen opaque 148x208 column frames in installed order and paint
  at (168,176). Frame 0 is fighter; frame 15 is mage. `TOWN-146/149/150`.
- Step at most once per school paint, only after strictly more than 83 ms.
  Stop at the selected endpoint. `TOWN-147`. Diamond and tavern clocks remain
  independent; inspect-only reads never advance a frame.
- Clear selected skill and price on every admitted party-picker step.
  `TOWN-138`. Skill icons and mask hits exist only at endpoints, with the
  selected character's eligibility still required. `TOWN-017/148/155`.
- Play the directly named `SFX/Town/School/Rotate.wav` once when rotation
  starts. `TOWN-147` names the file, not a numbered registry selector.
- Start immediately on a class change; a new target reverses from the current
  frame. Same-target selection does not restart the clock or sound. Canceling
  before movement at the already-displayed target is silent. These lifecycle
  choices are owner policy, `DIV-814`, not established ROM1 behaviour.

## Surfaces and exclusions

The game owns presentation state and an injectable clock. The shared room
compositor advances it for live, headless and school-dialogue paints. The UI
only consumes the current frame. School entry initializes the selected class;
exit leaves hidden state inert and reentry initializes afresh. New-game reset
clears it. No simulation, native save or research pin changes.

Missing art retains the existing static/text fallback. Missing sound remains
silent without blocking rotation. General city, tavern and shop animations
are separate owner stories. Original paint frequency, the global clock's full
lifecycle, class-change marker consumption and mid-flight retarget rules
remain unverified (`DIV-142/814`).

## Proof

Focused tests cover both fifteen-step directions, exact 83/84 ms boundaries,
long paint gaps, backward time, rapid retargets, same-class selection, pending
training, endpoint-only hit masks, dialogue/background painting, new-game and
visit boundaries, missing frames/audio and all sixteen loader addresses.
Final install and gate results are recorded in `verification.md`.
