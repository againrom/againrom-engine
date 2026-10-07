# Plan — read it from the clock, or do not draw it

## The one thing that decides the design

FR-2 is not a style rule, it is the whole shape. There are **three** places in this tree that hold
something rate-shaped, and only one of them is the clock:

1. The front-end's own **ladder position** — the number the three keys double and halve. It is what
   a press moves, and it is the obvious thing to draw. Drawing it is the defect this story exists to
   avoid: it is a value on the *input* path, and every clamp, truncation and refusal downstream of
   it is invisible to it.
2. The **water counter's** period in the viewer. Re-rated from the same statement, but a different
   instance carrying its own remainder, and it keeps running while the world is stopped. It is the
   drawing's clock, not the game's.
3. The **world's own ticker** — the accumulator whose period the paced advance divides elapsed time
   by to decide how many ticks fire. This is the clock. Nothing else decides whether the game moves.

**DD-1 — The cadence on the readout is read from (3), through the ticker's own `Period()`, and
from nothing else.** (1) is unreachable from where the readout draws, which is the mechanism rather
than a rule: the ladder is an unexported field of the flow, the readout composes inside the viewer,
and the viewer holds no pointer to a flow. (2) is reachable and is deliberately not used — its
period agrees today and is re-rated through a second call site, so an edit that moved one and not
the other would leave a readout that was right about the water and wrong about the game.

## Getting a world value to a window

`pkg/ui` may not name a simulation type; `pkg/game` may see a world, a bundle and a window at once.
So the readout cannot *reach* for the tick, the digest or the clock — those must arrive.

**DD-2 — They arrive by a push, on the pattern the phase push already established:** a value type of
builtins, handed over by the tier that owns the world, replacing whatever was held.

```go
type Readout struct {
    PeriodUS int    // the period the world's clock is holding
    Stopped  bool
    Tick     uint64
    Digest   uint64
}
```

Four scalars; it names no simulation type. **It is not a cached copy in the sense FR-2 forbids**, and
the distinction is worth stating because a reader will otherwise collapse the two: what FR-2 forbids
is a value *maintained* beside an authority by the code that moves it. This is *re-derived from the
authority on every push*, in one statement, from the very fields the advance reads; nothing else in
the tree writes it; and a push that stopped happening would freeze the readout visibly rather than
let it drift plausibly. There is no path that changes the clock and leaves this behind, because the
push is not on the changing path at all — it is on the frame path.

**DD-3 — The push happens on every paced call, on every return path.** The paced advance returns
early three times — the baseline call, a non-positive elapsed, and the stop — and the stop is
exactly the state the readout most needs to report. A push placed after the tick loop would never
fire for a stopped world and the readout would never learn about Space. A deferred call at the top
covers all four exits and costs one deferred call per frame.

**DD-4 — The map's tick-0 push does it too**, beside the tick-0 entity push and for that push's own
reason: a map that has been opened and not yet ticked already draws a readout of the state it
opened at, rather than a box of zeros until the first frame lands.

## Two readings of one period

The clock carries a **period in microseconds** and deliberately carries no second nominal identity —
it holds one quantity and no notion of where it came from, precisely so it cannot come to disagree
with itself about what it is doing. That rule stands: nothing is added to the ticker.

**DD-5 — The rate is derived at the readout by `terrain.RateOf`, the inverse of `RatePeriod`, added
beside it.** One file, two functions, adjacent — which is what stops the forward and reverse
mappings drifting. It clamps exactly as its forward twin does, so it is total.

The round trip is exact over the whole range: `RateOf(RatePeriod(r)) == ClampRate(r)` for every one
of the 1024 rates and for every out-of-range input, checked exhaustively rather than argued. That is
what FR-4 and P-5 rest on.

**Both numbers are drawn, not one.** The period is the quantity the clock holds and the rate is
derived from it, so drawing the rate alone would put a derivation on screen with its source hidden.
Drawing both also makes one fact visible that is otherwise invisible: a freshly opened map runs at
the game's own truncated 62 ms, and one press of `+` then `-` leaves it at our model's 62500 µs.
Same stated rate, different period — the readout shows it, and a rate-only readout would hide it.

## Reusing the panel, not copying it

The unit panel already composes a labelled box of rows into a plain image with no graphics context,
caches the picture behind a comparable key, uploads it once and places it by a corner and a margin.
Every one of those is wanted here.

**DD-6 — The composition splits at the row, not at the panel.** The subject-to-text resolution stays
per panel; the layout, measurement, box fit and paint become one shared path taking already-resolved
label/value pairs. This is a pure refactor of the unit panel: same rows in, same pixels out.

**DD-7 — There is one field space and two resolvers.** `PanelField` grows the readout's constants
and `PanelLayout` is used by both boxes. The type is already "geometry, colours, and which field each
row states in which order", which is exactly what both need, and the alternative — a second layout
type carrying the same thirteen appearance fields — is the duplication this avoids. A resolver
answers *false* for a field that is not its own, and a field a build does not define omitting its row
is behaviour the panel already specifies, so a mixed layout degrades to a shorter box and never to a
broken one.

**DD-8 — The unit panel's own layout, fields and corner are untouched.** AC-10 is then an identity
rather than a comparison: the same layout value composes the same rows through the same paint.

## The corner, and the two lines that need a seam

**DD-9 — The readout is top-left; the unit panel is bottom-left.** Two boxes anchored to different
corners of the same window cannot overlap unless their combined heights exceed the window, which for
a nine-line and a four-line box at this font is not reachable at any window the game opens at. It is
the free corner: the map screen draws nothing else in window coordinates.

**DD-10 — The entity seam widens by exactly one scalar: `Speed int` on the drawn entity.** The
crossing needs no seam at all — the span of the crossing an entity is on already crosses, beside the
transit remaining, because the displacement runs over the crossing. So the speed is the one line
that costs anything, and it costs one int in a struct that is already built per entity per tick.

Recomputing the crossing length here from the speed instead was rejected outright: the rate law lives
behind an unexported function in the determinism package, this story may not touch that package, and
a second copy of a movement law in the drawing tier is the exact failure FR-2 names, one field over.

**DD-15 — The group rate term is a second scalar beside it, and a second LINE beside it (FR-13).**
It is not a variant of the speed and must not be presented as one. The simulation composes the two
with a rule of its own — a nonzero group term replaces the entity's own speed wherever a rate is
computed — and that rule lives behind an unexported seam in the determinism package, so composing
them here would be the same forbidden second copy the crossing already refuses. Two raw numbers
carry the composition rule nowhere and let the reader apply it; one composed number would carry it
in the drawing tier and hide the disagreement that makes it worth reading.

It earns a line rather than being nice to have. The term is set by an order and cleared only by
another order or by death — arriving does not clear it and a fellow member dying does not — so a
unit can walk at a pace that no state on screen accounts for. That is a live divergence with a live
consequence, and it is invisible in every other instrument this tree has.

## On by default

**DD-11 — The visibility flag is stored inverted, so "shown" is the zero value.** A flag defaulted in
a constructor is a flag every struct literal in the suite gets wrong; a flag whose zero value is the
default is one nothing has to know about. What actually gates the drawing is the font, unchanged from
the unit panel: a viewer holding none draws no readout, which is why every existing viewer — the
standalone one included — is byte-identical without a single call being added to it.

**DD-12 — Hidden means *before* everything.** The visibility test is the first statement of the
readout path, ahead of the composition, the fit, the cursor resolve and the frame-rate read; and the
digest is computed on the push side only for a viewer that reports the readout shown. So FR-9 is the
position of two tests, not a set of guards spread through the path.

## Cost

**DD-13 — The refresh key holds the stated values and nothing else**, on the unit panel's own rule.
Two of them — the tick and the digest — change on every logic tick, so the box recomposes at
`min(frame rate, tick rate)`: 16 times a second at the map-load cadence, at most once a frame at the
top of the ladder. That is the honest ceiling and it is measured rather than asserted.

**DD-14 — The frame rate enters the key rounded to a whole number**, which is what keeps it from
defeating the cache: the engine's measurement moves continuously, so a raw float in the key would
force a recomposition every frame at every cadence, including a stopped one. Rounded, a steady frame
rate is a stable key.

## Risks

**R-1 — The digest is a full encode of the world per push.** It is the whole byte form, allocated
and hashed. At one push a frame that is the readout's dominant cost on any map with many entities.
Mitigated by DD-12 — it is computed only while the readout is shown — and measured, not assumed. If
it proves to dominate, the honest fix is a line the reader can turn off, not a digest that is stale.

**R-2 — Recomposition at the top of the ladder.** At 1024 ticks a second every frame has a new tick
number and a new digest, so the cache never hits. This is the worst case and it is bounded by the
frame rate; measured under SC-7.

**R-3 — The cursor line will look wrong on a slope.** It states the ground pick, the drawn lattice
rides the placement surface, and on relief they differ. This is FR-6 and it is disclosed rather than
fixed; the risk is that a reader takes it for a defect in the readout. Answered in the spec, in the
provenance and in the build's own README.

## Success criteria

**SC-1** (FR-1, FR-3) — The composed readout, for a known state, states every field of the table
with the expected text, with no window and no graphics context.

**SC-2** (FR-2, FR-4, AC-4) — Re-rating the clock **above the ceiling and below the floor** by
values the ladder cannot ask for makes the readout state the clamped rate, and the assertion is
written so that stating the requested rate fails it.

**SC-3** (FR-4, P-5) — `RateOf(RatePeriod(r))` equals `ClampRate(r)` over every rate in range and
over out-of-range inputs at both ends, exhaustively.

**SC-4** (FR-2, AC-2, AC-3, AC-5) — A key sequence driven through the front-end — doubling to the
ceiling, halving to the floor, stopping and resuming — leaves the readout stating what the world's
clock holds at every step.

**SC-5** (FR-3, AC-6, AC-7) — The stated tick and digest equal the world's own after a run of ticks,
and both hold still across a stopped span.

**SC-6** (FR-7, FR-8, FR-9, FR-11, AC-1, AC-9, AC-11) — Shown by default; `F1` toggles it; hidden it
composes nothing; with no font it draws nothing and does not fail.

**SC-7** (P-2, R-1, R-2) — The per-frame cost of the readout, measured: a composition, a cached
frame, and the digest, each as a benchmark, with the worst case stated in real time against a frame
budget.

**SC-8** (FR-10, FR-12, AC-8, AC-10, P-3) — The unit panel composes identical pixels with the readout
shown and hidden; the two boxes' rectangles are disjoint; a frame that drew the readout leaves world,
selection, camera and clock where a frame that hid it leaves them; and the two unit lines are absent
with nothing selected.

**SC-9** (FR-5, FR-6, AC-12, P-1, P-4) — Crossing states the recorded span and its absence; the
cursor line states the ground pick's own answer and its absence marker; every field either states a
value or omits its row.

**SC-10** (FR-13, AC-13) — A unit carrying a group rate term states it and its own speed as two
distinct numbers, and the assertion is written over a fixture where the two DIFFER, so a readout
that folded them — or that stated one for the other — fails it. A unit carrying none states the
absence marker, and both lines follow the selection.
