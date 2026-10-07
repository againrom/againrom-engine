# Plan — a dialog window stops the world

## Approach

The stop already exists, and so does the seam it crosses: the pause key sets a switch, the front-end
resolves the three cadence keys into a rate and a stop, and one call carries both to the world,
which declines to run its tick loop while the flag is set. What is missing is a **second cause** for
that stop and a place to resolve the two causes into one answer. So the suspension is not new
machinery beside the pause — it is the pause's own switch gaining a disjunct, and the resolver
gaining a memory of what it last told the far side. The dim is genuinely new: one filled rectangle,
decided by a function of viewer state and drawn by a single call, on the shape the panel and the
readout already use.

## Facts verified during planning

- The map arm's advance call is unconditional and stands after the notice's own input block, so a
  notice open on a frame does not stop that frame from advancing the world.
- The front-end decides whether to write to the cadence seam by comparing the freshly resolved
  ladder position and stop against **its own two fields** — the player's request. There is no record
  anywhere of what the far side was actually told.
- The far side's pacing consumes its baseline on **every** call, including the stopped one, and
  returns zero from the stopped arm before the elapsed span reaches its accumulator. A call that is
  not made consumes nothing, so the span is still owed at the next call and is paid up to a
  catch-up bound of 250 ms of world time — four ticks at the map-load period.
- Its readout push is deferred and fires on the stopped return, so a world stopped this way still
  reports itself live.
- The stop reaches the world's clock and its own flag and nothing else: no world
  field, no queue, no tick, nothing the digest covers.
- Production always hands the front-end a cadence seam beside the tick; the two are assigned in one
  statement and dropped in one statement.
- `Viewer.Draw` composes terrain, the art planes, the overlay passes, the route strokes, the unit
  panel, the readout and the notice, in that order. Every box after the passes is decided by a
  `*Present`-shaped function needing no window and drawn by one call that decides nothing.
- The viewer holds the two notice layouts as authored values from construction, with a setter as
  their substitution point.

## Design decisions

**DD-1 — the suspension is the stop, declared through the existing seam, and the advance call stays
unconditional.** The front-end goes on asking for an advance on every map-screen frame and tells the
world it is stopped; the world then takes its baseline and runs nothing. Nothing new crosses, no
seam changes shape, and nothing under `pkg/sim` or `pkg/game` is touched (FR-1, FR-2, P-3).

Rejected: **the front-end declines to call the advance.** The obvious design, and wrong here for a
measured reason (`analysis.md`): the far side owes whatever it was not called for, so a dismissal
pays back up to a quarter-second of world time in one frame and every walking unit jumps. It also
freezes the world's own diagnostics for exactly the period a player is most likely reading them. The
catch-up bound exists to absorb a stall; this would have manufactured one on purpose.

**DD-2 — one resolver, and it compares against what the far side was told.** The effective stop is
`the player's switch OR a notice is open`, resolved in one place, and the write happens only when
the resolved pair differs from the pair last handed over. That pair is remembered beside the ladder
and initialised where the ladder is — at the map-load period, running, which is what both consumers
are born holding, so an opened map still needs no call to agree (FR-1, P-4).

Rejected: **comparing against the player's own fields, as today.** A notice opening changes what the
far side ought to hold without changing either field, so the comparison would see nothing to write
and the suspension would never cross. This is the assumption `analysis.md` hunts, and the one line
of this story that could have shipped silently broken.

Rejected: **a second write site.** Two writers of one seam is how two answers to "what rate is in
force" come to exist.

**DD-3 — the player's switch is never written by a notice, and the ruling lives in one predicate.**
The disjunct is computed at the resolver from a predicate asking only whether a notice is open; the
player's field is not on that path at all, so "a notice restores whatever the player had" is a
property of there being no statement that could do otherwise, rather than a restore step that could
be got wrong. That predicate is also the whole of the authored ruling: an answer that the original
did not stop its clock changes its body and reaches nothing else (FR-3, P-2).

Rejected: **saving the player's switch on open and restoring it on close.** The design this story is
most likely to be built as, and it has three failure modes this one cannot have — a save that runs
twice, a restore without a save, and a mission that ends while a notice is open and never restores.

**DD-4 — the cadence keys are gated at the resolver, not at the input read.** The keys are read on
the map arm as today and the resolver ignores them while a notice is open, so the gate stands beside
the state it protects; and the notice's own three inputs keep the one interception point they have.
Nothing else on the arm is gated — camera, selection, orders, blows and the two diagnostics are as
0066 left them (FR-5).

Rejected: **letting the keys act, since the world is stopped anyway.** The pause key decides it:
pressed under a notice it would toggle a switch with no visible effect, and the toggle would take
effect on dismissal, unannounced. A key that does nothing beats a key that arms something.

**DD-5 — the dim is a value on the viewer with its own setter, not a member of the notice layout.**
It belongs to *a notice is open* and not to a kind, and the layouts are per kind: putting it there
would be two values that must agree. It ships as an authored value from construction, on the rule
the two layouts and the panel's already follow (FR-6, FR-8).

Rejected: **a boolean plus a constant.** A boolean cannot answer the question research is expected
to come back with, which is *how dark*; a colour answers both, a fully transparent one being off.

**DD-6 — the dim is decided by a function needing no window and drawn by one call in place, and it
asks the notice's OWN open test.** Whether there is one, what colour, and over what rectangle is a
pure read of viewer state, so FR-6 and FR-7 are assertable without a graphics context. Its place is
after the route strokes and before the unit panel, which is the single statement that makes FR-7
true: everything composed before it dims, everything after does not. **The gate is the font-carrying
predicate, not the raw open flag** — a notice is marked open whatever the font, so a dim keyed on the
flag alone would darken a map that has no box on it and no way to un-darken it. One predicate answers
"is a notice showing" for the dim, the input path and the suspension alike (FR-6, FR-7, P-1).

Rejected: **darkening the map's own draw calls.** It would reach every call on the map path, thread
through both terrain geometries and the sprite transform, and still miss the overlay rectangles.

**DD-8 — the suspension is enforced per advance, and the contract is stated at that granularity.**
The far side reads its stop once per advance and then runs however many ticks the elapsed span is
worth, so a tick that raises a notice partway through an advance does not stop the rest of that same
advance. Measured: at a one-millisecond period a single advance ran 250 ticks and raised the notice
on the sixth, so 244 ticks — 244 ms of world time — ran after it was open; at the shipped map-load
period the same drive ran four and raised none. The residue is bounded by the one catch-up span an
advance may ever run and is zero wherever an advance ran at most one tick.

It is left in place and disclosed rather than removed, because **no frame shows it**: the notice is
drawn from the frame after the advance that raised it, and that frame already has every one of those
ticks applied. What a player can observe — the world not moving while the box is up — is true at the
only granularity a player has.

Rejected: **cutting the advance short on the far side**, by breaking its tick loop when the notice it
just raised is open. It would make FR-1 true tick-by-tick and it costs the thing DD-3 buys: a second
place that decides "do not advance", on the far side of the seam, so the one predicate that carries
this product's authored ruling would become two sites a later answer has to move. It also edits
product code both tasks fence out and moves what an advance returns, for a difference no frame shows.

**DD-7 — the two comments this change falsifies are corrected in the commits that falsify them.**
The map arm states that the world keeps running behind the box; a driver-level test states that the
world keeps running while a notice is up. The first becomes false. The second's *assertion* stays
true and keeps its value — it pins that the driver's own tick is not gated — but its wording no
longer says so.

## Files to touch

| path | intent |
|---|---|
| `pkg/ui/flow.go` | MODIFY — the suspension predicate, the resolver, the far-side memory, the keys' gate |
| `pkg/ui/app.go` | MODIFY — the map arm's now-false comment |
| `pkg/ui/halt_test.go` | ADD — suspension, resume, the player's pause, the kinds, the keys, the degenerate screens |
| `pkg/ui/notice.go` | MODIFY — the backdrop value, its setter and the frame decision |
| `pkg/ui/viewer.go` | MODIFY — the backdrop field, its construction, one draw call in place |
| `pkg/ui/backdrop_test.go` | ADD — the dim's presence, absence, extent and place in the order |
| `pkg/game/world_test.go` | MODIFY — one test comment this change makes false |

## Risks

- **R-1 — a notice that opens and never closes leaves the world stopped forever.** The effective
  stop is derived from current state every frame and latched nowhere, so it cannot outlive the
  notice: the frame after a dismissal resolves to the player's own switch whatever happened between,
  including a mission that ended and left the map screen. (SC-2)

- **R-2 — the far-side memory becomes a second source of truth and drifts.** It is written on the
  statement that makes the call and initialised on the statement the ladder is, so no path updates
  one without the other; a map screen with no far side still updates it, which keeps the comparison
  meaningful. (SC-1, SC-5)

- **R-3 — the dim is too dark to play behind, or too faint to be the ruling.** A judgement, not a
  measurement, which is why it is a supplied value: the automated evidence pins that a dim exists,
  covers the map and stops at the panel; the strength is judged by eye against the mission the
  ruling came from. (SC-4, SC-6)

- **R-4 — a suspended world cannot clear the condition that raised its own notice.** Nothing closes
  a notice on a timer, so a mission step gated on a world that is stopped would wait forever. The
  existing rule that an announcement arriving while one is open is discarded is what keeps this
  bounded: at most one notice is outstanding, and the input dismissing it is the input resuming the
  world. (SC-3)

## Success criteria

- **SC-1** — driven over the production map arm, an open notice suspends the advance from the frame
  after it opens, and the frames after a dismissal run exactly what the same frames run with no
  notice ever opened. (FR-1, FR-2, AC-1, AC-2, P-4)
- **SC-2** — the same drive with the player paused before the notice, and not paused, ends with the
  world in the state the player chose, for a dismissal through each of the three routes. (FR-3,
  AC-3, P-2, R-1)
- **SC-3** — the outcome notice suspends and dims identically to the dialogue one, and the three
  cadence keys pressed under either change nothing that outlives it. (FR-4, FR-5, AC-4, AC-5, R-4)
- **SC-4** — the composed frame carries one dim covering the whole drawable area while a notice is
  open, none when it is not, and the composition order places it after the map and before the panel.
  (FR-6, FR-7, AC-6, AC-7, R-3)
- **SC-5** — a map screen with no lettering and a map screen with no cadence behind it are unchanged
  in every respect by a notice pushed to them. (AC-8, AC-9, P-1, R-2)
- **SC-6** — the tenth campaign mission, played from a lawful install to its first dialog window:
  the world holds still behind it, the map is visibly darkened, and both end on dismissal. (FR-6)
- **SC-7** — `go build`, `go vet`, `gofmt`, `go test -count=1 ./...`, the asset scan, the doc budget
  and the SDD audit are clean, with no pinned digest moved and no serialized version touched.
  (P-3, FR-8)
