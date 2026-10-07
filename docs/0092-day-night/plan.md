# Plan — 0092 day and night

## Approach

Three seams, in the order the value crosses them. `pkg/render/terrain` gains the sun model: a pure
function from an in-game minute and a switch to a `Light`, and the relight predicate beside it.
`pkg/ui`'s viewer gains the cache the model writes into, the switch, the offset and the one place a
relight happens. `pkg/game` pushes the world's tick across the existing per-tick statement, and
`pkg/ui`'s front-end binds the two keys.

Nothing is added to `pkg/sim`, nothing is stored, and no byte form is opened.

## Design decisions

**DD-1 — the sun model lives in `pkg/render/terrain`, beside the `Light` it produces.** Two reasons,
and either alone decides it. The angle is a double, and `pkg/sim` is held to no floating-point
identifier and no float literal by a source scan over its own syntax; putting the model there is not
a style breach but a gate failure. And the sun is not simulation state (FR-12): it is a parameter of
a drawing, and the package that owns the drawing's other light parameters is the one that owns this.
`pkg/render/terrain` is a leaf, imports only the standard library, and is already imported by every
tier that would ask. The **relight predicate** goes there too, and neither reason above places it —
it holds no float and is not a light parameter. It goes there because it is the sun model's own
cadence and splitting the two would let a caller ask for a sun at a moment the cadence says it may
not have one.

**DD-2 — the model is a pure function of `(minute, switch)`, and holds nothing.** No cache, no last
value, no memo. Everything stateful lives on the view (DD-3), so "what is drawn is the last
relight's sun" is a property of where the state is rather than of a discipline about when to call.
It also makes the whole of FR-1…FR-5 testable without a window.

**DD-3 — the cache is one `Light` field on the viewer, and the relief grid is rebuilt beside it.**
The original reaches every consumer through a single copy of the angle on the view object, written
by the global's only reader; the field is that copy. It is **seeded with the switch-off sun** — the
value this build already draws — so a view that is never told the time keeps drawing what it drew
(FR-10, AC-9). That the switch reads *on* while the seed is the switch-off value is not a
contradiction: the switch selects which arm the **next** relight takes, and a view that never
relights has never asked what time it is. A view whose switch IS moved before it ever holds a clock
relights from clock zero, and that is stated in FR-10 rather than prevented by a "has a clock yet"
flag — a flag would be a second piece of state whose only job is to make one unreachable call
harmless.

**DD-3a — the sprite layer is repointed at the same cache.** Two sites in the viewer take the ramp
row and the sky tint from the package's fixed daytime light rather than from the viewer, and a
comment beside them says the viewer "holds no Light". After DD-3 it does, so leaving them makes FR-6
false and makes AC-13 pass for the wrong reason — because sprites are disconnected from the sun, not
because FR-5 holds their inputs still. They are repointed and the comment corrected. Today this
changes no pixel, since FR-5 fixes both inputs; the point is that the seam is wired when D-1 closes.

**DD-4 — one entry point decides whether to relight, and it is on the viewer.** The front-end hands
over a clock and nothing else; the viewer applies the offset, tests the cadence and rebuilds. The
alternative — the front-end deciding and calling a `Relight()` — puts the predicate at the caller,
where a second caller can disagree with the first about when the sun moves. FR-6 is then a property
of there being one writer.

**DD-5 — the relight REUSES the constructor's validity verdict and must not re-derive it.** The
constructor builds the level grid inside the altitude-validity branch; a relight rebuilds it only
where a grid already stands, because a non-nil grid **is that branch's stored answer**. This is not
"test whether a grid exists", and the difference is not pedantic: the two guards disagree on an
**overlong** altitude slice, which the constructor's guard rejects and the grid builder's own
totality accepts. An unguarded rebuild would hand such a view a grid on its first relight and flip it
from unlit to lit — lighting a map this build deliberately draws unlit. The tree's own comment beside
the constructor already warns about exactly this pair of guards, and the four fixtures that
distinguish them exist; the relight is tested on all four (AC-14).

**DD-6 — the offset is added to the incoming clock and moves in whole in-game hours.** An hour is 60
minutes, 60 is three whole relight periods, and the offset is therefore invisible to the cadence
predicate: FR-11's "the same relight instants" is arithmetic rather than a claim to check. Storing
an offset rather than a second clock is what keeps one number crossing the seam (DD-2) and keeps the
world's tick the only source of time.

**DD-7 — FR-13's two keys come from two different registers, and the convention is amended rather
than bent.** The switch takes `N`, the decoded binding, which forces a relight the way the
original's does.
The offset takes `F3`, from this front-end's own diagnostic register, `F1` and `F2` being the readout
and the lattice.

That register's authored rule is *a letter changes the world, a function key changes only what is
drawn* — and `N` is now a live counterexample, since the switch changes only what is drawn. The rule
is **narrowed in the same commit** rather than quietly broken: a decoded binding takes the key the
original gave it, and the letter/function-key split governs only the bindings that are ours. A
convention with an unremarked counterexample stops being read.

**DD-8 — no byte-form version, and the witnesses already exist.** The lighting clock is **read from**
the world's tick, which the byte form already carries and the digest already covers; the switch and
the offset are view state that no simulation record holds. So there is no field to add and no version
to take, and this story allocates none.

The claim is witnessed by two pins the tree already carries, both of which must run unchanged: the
byte form's **offset partition**, which is a partition of the whole form and so fails on a field
added to the encoding rather than widening silently, and the **literal field-set table** over the
simulation's records, compared for exact equality and so failing on a field added to a struct that
the encoder never writes. A round-trip and a digest are recorded beside them but are **not** the
witness — that file's own comment says why: a field the encoder does not write changes no byte it
does write, so both are blind to exactly the shape being denied.

No test of this story's own names the version number. That is not an argument against the existing
pin, which is legitimate and catches the un-bumped field; it is the narrower rule that a story must
not assert `version == N` to mean *this story added nothing*, because that assertion holds equally
when some other story added a field, and it breaks a textually clean merge when another story
legitimately takes the next number.

**DD-9 — every angle constant is its own decimal literal.** None is derived from another and none is
computed from `math.Pi`. The two literals differ from `pi/4` at the eighth decimal, `720 x` the day
step is a truncated `pi/2` rather than `pi/2`, and the night step is three times the day step — so
every one of those relations is a **check** the tests make, and expressing any of them as a
derivation would turn a check into a tautology. This is the rule the tree's existing angle constant
already states for itself.

**DD-10 — the whole grid is rebuilt on every relight, and no incremental path is built.** The
original has none: a changed angle reaches the screen only through the full per-vertex rebuild plus
the shading-table rebuild behind it. At the shipped speed a relight is one rebuild per ~20 s of real
time, so the cost is bounded by measurement rather than by design (SC-3).

**DD-11 — the clock push goes INSIDE the existing per-tick push, not beside its call sites.** That
push has **two** callers — the constructor and the tick — so a statement placed beside them is two
lines that can drift; placed inside, "the sun's clock is the world's tick" is one line. Both callers
are needed: the tick one is the cycle, and the constructor one is why a mission opens already lit for
06:00 rather than at the seed, which matters because a map opened and left stopped never ticks.

**One world tick is one sub-tick** — the same unit the mission script and the engagement decision
already take modulo 16 — so no conversion crosses the seam. And the push is per **tick**, not per
frame, for a reason rather than for symmetry: the pacer runs up to a bounded number of whole ticks
inside one frame, so a frame-level push of only the last one would step over a relight instant and
freeze the sun for a period. That bound is below the relight period at every rung of the cadence
ladder, so no single frame can ever fire two relights — measured in verification, not assumed here.

## Risks

**R-1 — the relight is a full pass over up to 65 536 vertices, inside `Update`.** If it costs more
than a frame the map hitches once every 20 s. Measured, not assumed (SC-3); if it were too slow the
answer is not an incremental path (DD-10) but a lower relight rate, which is a number.

**R-2 — the diagnostic offset reaching the world.** FR-11 bounds it to what is drawn; the risk is
that a later hand routes it into a tick. The offset is held on the viewer, which cannot spell a
world, so the type system carries this rather than a rule (SC-4 measures the digest is untouched).

**R-3 — a wrong band boundary looks right.** Four bands, two of them literal, and a boundary off by
one hour changes 60 of 1440 minutes and nothing visible. The band census (AC-2) and the endpoint
table (AC-1) are what catch it, and both come from the published landmarks rather than from this
implementation.

## Success criteria

- **SC-1** The four published landmarks reproduce from the shipped code — the 960/480 computed
  census, the 720/240/240/240 band census, the angle extremes, and 72 unforced relights per in-game
  day — together with the two arithmetic identities nothing published asserts: `720 x` the day step
  is a truncated `pi/2`, and the night step is the day step's threefold. (FR-3, FR-7)
- **SC-2** `go build`, `go vet`, `gofmt`, the whole suite with `-count=1 -trimpath`, and the three
  repo scripts, all green — with the byte form's offset partition and the simulation's literal
  field-set table both passing unedited. (FR-12)
- **SC-3** A relight over a 256x256 height grid is benchmarked, and the figure is recorded beside the
  20-in-game-minute period it happens at. (DD-10, R-1)
- **SC-4** A mutation battery over the sun model and the relight predicate, with survivors reported
  rather than a clean score, and every expectation written as a literal rather than computed from
  the constant it pins. (FR-3, FR-7)
- **SC-5** The mission drive is run on both roots and its exact line recorded, before and after.
  (P-1)
- **SC-6** The manual criterion, and the only one that reaches a screen: over **sloped ground** on a
  mission's map the relief visibly changes as the lighting clock is scrubbed, and returns to its
  opening picture after 24 hourly steps. It must be attempted on a hillside and nowhere else, because
  flat ground is invariant under the whole cycle (AC-10) and would read as the feature being broken.
  The scrub gives **no on-screen acknowledgement** — nothing displays the in-game hour — so the
  operator counts presses; that is a stated limitation, not an oversight. (FR-8, FR-11, AC-12)

## Traceability

| FR | Design decisions | Success criteria |
|---|---|---|
| FR-1 | DD-1, DD-2, DD-11 | SC-1 |
| FR-2 | DD-2, DD-3, DD-7 | SC-1 |
| FR-3 | DD-2, DD-9 | SC-1, SC-4 |
| FR-4 | DD-9 | SC-1, SC-4 |
| FR-5 | DD-2 | SC-1 |
| FR-6 | DD-3, DD-4 | SC-1 |
| FR-7 | DD-4, DD-10 | SC-1, SC-3, SC-4 |
| FR-8 | DD-5, DD-10 | SC-3, SC-6 |
| FR-9 | DD-4, DD-7 | SC-1 |
| FR-10 | DD-3, DD-5 | SC-2 |
| FR-11 | DD-6, DD-7 | SC-6 |
| FR-12 | DD-1, DD-8 | SC-2, SC-5 |
| FR-13 | DD-7 | SC-6 |
