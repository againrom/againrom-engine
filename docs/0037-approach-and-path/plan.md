# Plan — 0037 approach and path

## Baseline

`canonicalRoute` refuses a target that is not open under its relation **before running any wave**,
and returns having labelled nothing. `step.go`'s far call site reads that refusal twice: once for its
cost, and once to discharge the decoded budget override's goal-free half, which it says the refusal
"already establishes". A far search that finds nothing clears the order in that tick; a near failure
raises the stall count. `searchRoute` carries three parameters — relation, window half-width, budget
rule — and the two call sites in `Step` are the only choosers.

The route a mover walks is per-entity state: the far search writes one, `consume` drops its walked
head, `subGoal`'s three staleness tests replace it, and the byte form carries it. It is unexported.

`pkg/ui` holds the selection, the picker and every overlay pass, and by construction cannot name a
simulation type; `pkg/game` owns the world and builds the per-entity value the window tier receives.
No path overlay exists in either.

## Design decisions

- **DD-1 — the settle is a fourth search parameter, bool-shaped.** `settleRule`, beside the relation,
  the window and the budget rule, chosen by the two call sites in `Step` and by nothing else (FR-1,
  FR-4). It is not derived from the relation: a relation says which cells exist for a search and this
  says what it may come back with, and the file's own header already refuses folding two such choices
  into one flag because neither can then be moved without moving the other. It is not a func value
  either, for that header's other reason — a second behaviour-steering identity beside the mode byte,
  one inside the digest and one outside.
- **DD-2 — the picker takes the scratch and no relation.** It reads the label plane and nothing else
  (FR-1). Not an economy: the plane already carries the relation, the window and the budget of the
  wave that just failed, so a second passability test here would be a different predicate answering a
  question already answered — and would make a cell the budget never reached a candidate.
- **DD-3 — the ring walk is reproduced, not replaced.** Sides in the order `y+r`, `y-r`, `x+r`, `x-r`
  for `i = -r..r`, accept on **strictly** less (FR-1). A coordinate tie-break would be tidier and
  would answer differently, and the answer is observable — it is a cell a unit walks to and a digest
  covers. The growth test is `r + 1 < limit` rather than a loop to `limit`, so the bound means what
  it is meant to mean.
- **DD-4 — the budget gate goes inside the budget function, as an argument.** `generationBudget`
  gains the goal's openness and answers all four arms (FR-2). The alternative was a third
  `budgetRule` value, which would have made that a three-valued type and put the gate at the call
  sites — where `Step` would have to decide "is this goal open" before a search it is about to ask
  that of. One function turns a rule into a number; the gate is part of the rule.
- **DD-5 — the far slack comes back as a named constant.** `staticSlack = 5` beside `nearSlack = 3`
  (FR-2). It was removed when the far arm stopped reading it, on the ground that a constant nothing
  reads records a scalar worse than a sentence does. The far arm reads it again, so it is a constant
  again, and the two arms' slacks are separately visible where a shared one would silently give the
  far search the near search's.
- **DD-6 — the order takes the settled cell, and the write is in `Step`.** Not in the search (FR-3).
  A search returns a route and changes nothing about the entity that asked; the tick is what turns a
  route into an order's state, and it is the one place that already clears an order on arrival and on
  failure. This is the story's divergence and `provenance.md` carries the reason; putting it here
  keeps the divergence to one statement a reader of the tick meets in order.
- **DD-7 — the query returns `[][2]int32`, copied.** Not the internal cell type, not a slice into the
  stored route (FR-6). The stored route is hashed state, so an alias would be a write into the digest
  through a reader; and a plain pair of integers names no type of the package, so what a caller holds
  reaches into no world. The pinning sweep over `*World`'s exported methods gains a list of readers
  that take an argument, because "takes an argument" and "writes" were one list only by accident.
- **DD-8 — the route crosses the seam as a field, for every entity.** Not a second setter and not a
  map keyed by id (FR-7). The tier that knows the selection is the window tier, on the far side of a
  boundary that exists so the tier holding the world cannot be asked about drawing; reaching back
  through it to skip the copy for unselected units would put the overlay's scoping rule on both sides
  of the seam. The cost is a per-**tick** copy for entities under orders — sixteen a second against
  the display's rate — and nothing at all for every entity not under orders.
- **DD-9 — the seam value stops being comparable, and its comparisons become deep.** A slice field
  costs `==` on the seam type (FR-7). Three landed tests compared snapshots with it; each becomes a
  deep comparison, which is what each of them meant and is strictly stronger — two entries alike in
  every scalar but heading down different routes are no longer the same snapshot.
- **DD-10 — the overlay is a scoping rule and a transform, with no state.** The preview decision is
  pure and is the whole of it; the segment build reads that and asks nothing else (FR-8, FR-9). There
  is no cache and nothing to invalidate: the route is state the tick maintains, and an invalidation
  rule on this side would be a second opinion about when it went stale, held by the tier with no
  state to answer from.
- **DD-11 — there is no count in the decision, and the count was never the cost.** *(amended
  2026-08-01 with FR-8a; it read "the cap skips whole and counts the drawable".)* The cap is removed
  rather than raised, because a larger number is the same judgement with a different digit. A frame
  pays for the **legs** it issues, and a unit's legs are its route's length — 1 to 251 on a 256x256
  map — so a count of units misses the real quantity by up to that factor **in either direction**,
  and the cap refused a cheap picture and drew an expensive one. What bounds the cost is DD-15, which
  is not a count at all.
- **DD-12 — the path point does not reuse `placeArm`.** A separate cell-centre transform sharing the
  lift and the camera but **not the cull** (FR-9). `placeArm` answers false for an arm the view does
  not reach, and a culled endpoint is not a culled segment: a leg with both ends off screen may cross
  the middle of it, so culling on endpoints would blink a path out exactly as its unit walked past
  the edge. The drawer clips.
- **DD-13 — the strokes are drawn outside the pass slice.** An overlay pass carries rectangles;
  folding a polyline into it would widen every pass with a field one of them could hold (FR-9). The
  loop runs after them, so the line is over everything and covers nothing.
- **DD-14 — the criterion this contract falsifies is amended in place, upstream.** FR-1 makes 0045's
  AC-2 false — "each target is cleared on the first tick, the unit unmoved", over three fixtures two
  of which are now walked. A criterion is a live assertion, so it is repaired in that story's own
  contract in the same commit that changes the behaviour, rather than left for its tests to report.
  Its FR-7 is **not** touched and the distinction is the point: a far search that finds no route
  still ends the order in that tick, and what moves is when a far search finds none. The same
  supersession reaches 0036's AC-9 through `pkg/mapload`, where a crossing closed by water was
  already witnessed against the tree rather than against AC-9's sentence, and it is re-witnessed the
  same way.
- **DD-15 — the leg that cannot reach the window is not built.** A bounding-box reject against the
  view, inside the segment build, taking the place the cap held (FR-9a). It is admissible where the
  cap was not because it is **exact**: a box that misses the view holds no point inside it, so no leg
  it drops could have painted a pixel. That makes it a removal of work rather than a limit on the
  picture, and it is what pays for FR-8a — at the default zoom most of a full-map group order's legs
  are outside the window, which SC-11 measures. It sits in the build and not in the draw loop because
  the build is the pure value a test can hold. The margin is the whole `PathWidth`, twice what a
  centred stroke reaches past its line: too generous costs a leg, too tight costs a pixel.

## Risks

- **R-1 — the sweep comes back.** Reopening the pre-sweep refusal restores a whole-map wave for a
  goal that cannot be labelled, which is the shape 0029 was a performance revision about. Two things
  bound it and both are contract: FR-2's gate makes the budget a function of the distance ordered
  rather than a flat thousand, and FR-3 makes the sweep run once per order rather than once per tick.
  Removing either reintroduces the defect, so both carry criteria.
- **R-2 — the two modes now answer an order differently.** Under FR-4 a blocked goal is walked toward
  in one mode and refused in the other. The mode contract survives literally — the clearing rule is
  the same and what differs is whether the far search finds a route — but a reader comparing the two
  modes on one world will see a unit move in one and not the other, and that is disclosed rather than
  smoothed over.
- **R-3 — the seam grows a slice.** DD-9's loss of comparability is a real narrowing of what callers
  may do with the value, and it cannot be undone later without removing the field.
- **R-4 — the overlay draws from the cell, not from the interpolated sprite.** Between ticks a walking
  unit is drawn part-way to its next cell while the line's tail stays on the cell it is leaving, so
  the two meet exactly at tick boundaries and part by at most one cell in between. Accepted: a route
  is a statement about cells.
- **R-5 — zoomed far enough out, DD-15 has nothing to reject.** At `ZoomMin` the whole of a 256x256
  map is nearly inside the window, so most legs survive the reject and the cost comes back; SC-11
  measures it at both zooms and `verification.md` records what it is. Disclosed and accepted rather
  than capped: the failure mode is a frame rate that recovers as the routes shorten, where the cap's
  was the overlay disappearing, and a number chosen to bound it is the judgement FR-8a removed.

## Success criteria

- **SC-1** The full gate passes unscoped at the head that is pushed, over every package in the
  module, with the FAIL set recorded and attributed.
- **SC-2** The tree's landed byte-form and digest pins are unmoved: no version byte, no field set and
  no encoded shape changes anywhere in this story.
- **SC-3** The settle machinery, present but unreached, moves no digest — so the behaviour change is
  attributable to the call site that turns it on and to nothing else.
- **SC-4** The determinism scan over `pkg/sim` stays green: nothing added imports a clock, a file or
  a generator, and no float type or literal appears.
- **SC-5** Mutating the ring walk's accept from strictly-less to less-or-equal is killed, and the
  restore verified — the tie-break is contract, not an artefact.
- **SC-6** Mutating the picker so the first labelled cell ends the scan is killed: the ring is
  scanned whole.
- **SC-7** Mutating the budget gate so the flat override applies whatever the goal is, is killed.
- **SC-8** Moving the near call site's budget rule to the flat one is killed, **through `Step`** —
  the criterion nothing in the tree carried before this story.
- **SC-9** *(amended 2026-08-01 with FR-8a; the first mutant was "the cap from skip-whole to trim",
  and there is no cap for it to name.)* Mutating the preview so that some count of drawable units
  answers with fewer than all of them is killed, and mutating the preview's scope from the selection
  to every targeted entity is killed.
- **SC-10** The benchmark shape this story changes most — a group ordered at a terrain-sealed cell —
  is re-run and reported beside what it measured before, rather than quietly replaced.
- **SC-11** **The cost of drawing them all is measured, not assumed** (FR-8a, FR-9a, R-5). Over the
  shape a full-map group order actually produces — the route length taken from the tick rather than
  chosen — at two group sizes and two zooms, on both arms: every leg issued, and only the legs that
  meet the view. The legs counted and the milliseconds reported separately, because the count is
  exact and the milliseconds measure the CPU half of the stroke alone.
- **SC-12** Mutating DD-15's reject into an **endpoint** test — the rule DD-12 forbids — is killed:
  it drops a leg that crosses the view, and the fixture that catches it is the one whose two
  endpoints are both outside.

## Traceability

FR-1 → DD-1, DD-2, DD-3 → AC-1, AC-2, SC-5, SC-6 · FR-2 → DD-4, DD-5 → AC-7, SC-7, R-1 ·
FR-3 → DD-6 → AC-5, SC-10, R-1 · FR-4 → DD-1 → AC-4, AC-8, SC-8, R-2 · FR-5 → AC-3 ·
FR-6 → DD-7 → AC-9 · FR-7 → DD-8, DD-9 → AC-12, R-3 · FR-8 → DD-10 → AC-10 ·
FR-8a → DD-11 → AC-11, SC-9, SC-11 · FR-9 → DD-12, DD-13 → AC-12, R-4 ·
FR-9a → DD-15 → AC-13, SC-11, SC-12, R-5 · FR-10 → AC-6, P-1, P-2, SC-2, SC-3, SC-4 ·
the upstream repair → DD-14 · the gate itself → SC-1
