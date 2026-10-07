# Plan — where the numeral lives

## Approach

The numeral is presentation. Everything it needs already crosses the drawing seam — a unit's health,
its owner, its cell and its sub-tick displacement — so it is built entirely inside `pkg/ui`, out of
values that package already receives, and `pkg/sim` is not opened.

One new file, `pkg/ui/numeral.go`, holds the record type, the ingest, the drift, the expiry and the
pure present. `viewer.go` gains the fields and two call sites; `app.go` gains the binding and the
toggle call; `overlay.go` gains one line in the entity setter. `cmd/missionrun/main.go` is fixed
first and separately.

## Decisions

- **DD-1 — the ingest runs in the viewer's own per-frame step, over the entities the tick has just
  pushed** (FR-1, FR-2). The entity setter is the one push of per-entity health into this package,
  but it takes no clock and a record needs an instant; the step takes one and already establishes
  the ambient baseline from it. So the ingest reads what the setter left and stamps it with the
  frame's own time, in one place — a second ingest site would be a second answer to what a figure
  shows (P-3). A front end that pushes twice between two steps is seen once, which is the same
  aggregation FR-3's merge produces.
  Rejected: a new seam carrying a per-blow damage event. It would put a damage rule on the far side
  and duplicate what the health push already carries, and it is not what the original does — the
  original's own notification carries a health level and the client subtracts.

- **DD-1a — the step runs drift, then ingest, then expiry, and it stands ABOVE the popup return**
  (FR-6, FR-7). Drift first is what makes a record born this frame stand at exactly its birth offset
  on the frame it is born; expiry last is what stops a record made this frame being reaped by the
  same call. Above the popup return, the two clocks separate for free and without a case: the
  animation counter does not advance under a popup, so nothing drifts, while the wall clock does,
  so everything still expires — which is the decoded behaviour of the two clocks and not a rule
  written for the popup.

- **DD-2 — the remembered health is a `map[uint32]int` on the viewer, written unconditionally
  before either gate is consulted** (FR-1, FR-2). Written after a gate it would drift: a blow taken
  with the display off would leave the remembered value stale and the next blow would show the sum
  of both, which is a merge across a window that has expired. Entities absent from a push are
  dropped from the map on that push, so a viewer reused across maps cannot attribute one map's
  health to another map's entity of the same id.

- **DD-2a — FR-5's absence is ASSERTED, not left to inspection.** "No severity colour is computed"
  cannot be shown by pointing at code that is not there; the witness is a test over three victims at
  the three decoded bands taking equal damage, asserting that every drawn property of the three
  figures is identical. That is what would fail if a later story reintroduced the band, which
  reading the file would not.

- **DD-3 — the record carries its accumulated offset, not its birth position** (FR-3, FR-7, FR-8,
  FR-11), and the undecoded scalar FR-8 names is taken as **one**, written as a named constant pair
  rather than folded into the arithmetic — so the two numbers a decode of that virtual would move
  are the two numbers a reader can find.
  The drawn place is recomputed each frame from the unit's current placed position plus the record's
  offset, so a figure follows a walking unit; and the offset is what the drift mutates, so a merge
  that leaves the offset alone leaves the figure exactly where it had risen to.

- **DD-4 — the drift is driven from the DELTA of the viewer's own animation counter**, sampled in
  `step` (FR-7). That counter is already the game's paced logic tick — the same clock the water,
  the animated objects and the animated structures are selected from, and the one whose period the
  cadence seam sets — so the numeral needs no clock of its own and cannot come to disagree with the
  rest of the ambient picture. Driving it from the frame instead would make the drift a function of
  the frame rate; driving it from the world's tick would need a value this package may not name.
  A delta rather than a per-frame decrement is what makes several ticks in one frame drift several
  steps, which is what the original does.

- **DD-5 — the life is tested in `step`, against the timestamp `step` is already given**
  (FR-6). This package's existing shape is a pure decision method plus a `Draw` that only uploads
  and blits, and `Draw` takes no clock; `step` takes `now` and already establishes the animation
  baseline from it. So expiry runs once per frame at the frame's own instant. The original expires
  inside its paint; both run once per frame, so no frame differs. Rejected: reading the clock inside
  the present method, which would make the expiry unassertable in a test.

- **DD-6 — the record composes its own picture once, at birth and on each merge, and the drift never
  invalidates it** (FR-3, FR-10). The picture is a function of the number and the colour alone;
  position is not in it. So a figure drifting across the screen recomposes nothing, and the only
  recomposition is a merge, which changes the number.

- **DD-7 — the two issues of the text are one picture, not two draws** (FR-10). The shadow is
  composed into the same image, offset by one pixel down and right, and the face over it. That keeps
  the blit count at one per record and keeps the whole of what is drawn assertable from the pure
  method. Our own offset; the shape is the decode's.

- **DD-8 — the position goes through `placeArm` with a cell-local arm**, exactly as the health bar
  does (FR-11). That is the one transform that carries the displaced mode's per-cell relief lift,
  the camera and the view cull, and reusing it is what stops a figure drifting off the unit it
  belongs to on a slope. The record's offset is added to the cell footprint's anchor before the
  transform, so it scales with the zoom and stays attached at any zoom — a consequence of this tree
  having a camera, which the original has not.

- **DD-9 — the colour is a small authored table indexed by owner, with slot zero its own entry**
  (FR-4). The rule that is decoded is that the colour is a function of the struck unit's OWNER, so
  the table is indexed by owner and not by the mine/not-mine flag: collapsing it to two would make
  a four-participant map draw two colours where the original draws four. Out-of-range slots fold by
  remainder, so no owner value can index out of the table and P-1 needs no guard of its own.

- **DD-10 — `L` is split at the binding, not at the consumer** (FR-9). `readAppInput` reads the Ctrl
  modifier already; both `Chip` and the new `Numerals` field are set from the same key press and the
  same modifier read, negated for one and asserted for the other. Split at the consumer instead, the
  snapshot would carry a `Chip` that is true on a frame that did not chip, and every existing test
  that builds the snapshot by hand would be asserting a lie. This narrows an authored binding and is
  reconciled into the spec rather than left as drift.

- **DD-11 — the toggle call sits on the map arm beside the lattice and the readout**, under the
  popup gate (FR-9). It changes what is drawn and never the world, so its position among the
  viewer's statements is not observable; being on that arm is the whole of "it does nothing on any
  other screen".

- **DD-12 — the mission tool's health reader returns a pair and its unit resolver validates against
  the world** (FR-12). Two changes, because they answer two different failures: the pair is what
  makes absent distinguishable from downed at every reader, and the resolver check is what makes a
  bad reference fail before an order is issued rather than after one is reported. The resolver is
  shared by the waypoint and the attack paths, so both gain the check from one place — a waypoint
  naming an absent unit reports today that it stopped short of `(-1,-1)`, which is the same defect
  wearing different clothes.

## Risks

- **R-1 — the ingest fires on any health drop, not only on a blow.** The front end cannot tell a
  blow from a chip key or from any other cause of a health decrease, because the seam carries state
  and not events. This is accepted and is the original's own shape: its notification is named "take
  damage" and is sent by five producers, of which the melee strike is one.
- **R-2 — a record over a unit the frame does not hold cannot be placed.** Handled by drawing
  nothing and letting it expire (FR-11), rather than by destroying it, so the rule has no second
  case to get wrong.
- **R-3 — `0084` is in flight over `pkg/ui/readout.go` and possibly `viewer.go`.** Nothing here
  touches the readout; the viewer changes are field additions and two call sites. The gate is re-run
  after the rebase, because an auto-merge is not evidence the two composed.

## Success criteria

- **SC-1** Every AC has a test that fails if its clause is removed (FR-1..FR-13).
- **SC-2** `pkg/sim` has no diff, and the byte form's version literal is unchanged (FR-13, P-4).
- **SC-3** The mutation set below kills on the named cases (FR-2, FR-3, FR-6, FR-7).
- **SC-4** The full local gate is clean before the commit and again after it (P-5).

## Mutations to run

Each is a one-line change to a load-bearing predicate, run to confirm a test dies:

1. the health comparison from strict to non-strict (FR-2);
2. the merge from add-into to append-second (FR-3);
3. the life comparison from `<=` to `<` at exactly 1000 ms (FR-6);
4. the horizontal drift sign made unconditional (FR-7);
5. the drift driven by the frame rather than by the counter delta (FR-7);
6. the display gate moved from creation to drawing (FR-2).
