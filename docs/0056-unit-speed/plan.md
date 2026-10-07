# Plan — a law with two absent inputs, and a duration where a sub-cell position would be

## Approach

The rate law is transcribed as **one pure function** of six values — the domain, the mover's
speed, the two cells' cost bytes and their height bytes — and a second turning its answer
into a transit length. Both live in a new `pkg/sim/rate.go`, both are integer-only, and both
are exercised over their whole input domain by tests that need no world.

`Step` calls the first with **zeros for the four terrain bytes**, because this tree carries
neither plane. That is not a simplification: at a zero cost mean the law itself substitutes
8 and at a zero height delta its tilt is the identity, so what comes out is what the law
computes for level cost-8 ground. Omitting the two terms instead would leave a fork nothing
exercises and no seam to plug a plane into.

The mover's position stays a **whole cell**. A transit is a **duration**: the mover takes
its next cell as it does today, then owes `ceil(256/step)` ticks during which no rule
advances it. That reproduces the law's cells-per-tick exactly while leaving occupancy,
contention, the two searches and every consumer of a position where they are.

The rest follows: three fields on the entity record, a version bump and refusals in the
byte form, one column read at the loader, and — a mover now spending many ticks between two
cells — a seam that keeps its step and a displacement over the transit, not the tick.

## Facts verified during planning

- **Our tick is the sub-tick.** `MOVE-CLOCK-032` puts one displacement per actor per
  sub-tick and the presentation tick in one loop iteration; `0041-world-clock`'s FR-2 and
  `TERR-ANIM-008` put our tick at the map-load period of 62 ms with a water cycle of sixteen
  of them, and sixteen sub-ticks are one full tick. Read, not assumed — the argument is in
  `provenance.md`.
- **`internal/archtest`'s float scan reads production sources only.** `determinism.go`'s
  loader skips `_test.go` (line 158) and `CheckSimTests` holds `pkg/sim`'s tests to the
  standard library, not to the float ban. So the diagonal's exactness can be proved
  **in-tree**, against the law's own shipped double, by a test in `pkg/sim`.
- **No `data/map.reg` reader exists anywhere in the module**, and `alm.Map` carries
  `Altitudes` but no world does: both terrain terms are absent inputs, not omitted terms.
- **`data.UnitDef` already carries `Speed`, decoded by 0049 and read by nothing**, and
  `data.UnitDefaults()` already holds the constructor's `Speed: 10`.
- **The seam already remembers a previous cell and a facing per entity**, keyed by id, never
  iterated, written in `recordCells` before the advance; and **`entityShift` already
  displaces every glyph an entity draws by one vector** over `phaseUS/phasePeriodUS`. A long
  transit needs that memory held and that denominator changed, not a second memory.
- Gate baseline on the branch before any edit: `EXIT=0`, **FAIL set empty**.

## Design decisions

**DD-1 — The rate is a pure function taking both terrain planes' bytes, and `Step` passes
zeros.** `rateOf(d Domain, speed int32, costSrc, costDst, hSrc, hDst uint8) int32`. The
byte-typed parameters are the planes' own width, so the law's byte-wide cost add and its
`int8` height difference are the types' own arithmetic rather than masking written out. It
reads no world, so it is testable over its whole domain without one, and a plane arriving
later changes its **callers** and not it. Rejected: reading the planes inside, which puts a
world in the signature and spells the absent terms as `if plane == nil`.

**DD-2 — The diagonal without a float is `(v·707)/1000`, and it is exact over the law's
whole domain.** The law truncates `v × K` where `K` is the shipped double
`0.70699999999999996`. Integer `(v·707)/1000` truncates toward zero and agrees with it for
every integer `v` in `0…999`, because `707v/1000` is an integer only when `1000 | v`
(`gcd(707,1000) = 1`), so for every other `v` the exact product sits at least `1/1000` away
from an integer — while `K` differs from `0.707` by under `5·10⁻¹⁷` and one double
multiplication adds under `10⁻¹⁴`, twelve orders of magnitude short of that margin. `v` is
clamped to `[1,63]`, so **the cost in exactness is zero over every input the law admits**,
and the claim is not left as an argument: a test re-executes both forms — the integer one
and `math.Float64frombits(0x3FE69FBE76C8B439)` — for every `v` in `0…999` and requires
equality (SC-2). **The bound is 999 and not larger** — caught by reading this plan, not by a
failing test: at `v = 1000` the product IS an integer, the margin vanishes, and which way
the double lands is a question about one rounding rather than about the law. The clamp sits
sixteen times below that, so nothing is owed; a test asserting equality past 999 would
assert what the argument does not carry.

**DD-3 — Where the law's divisor is zero we take a step of 1.** A `v` of 1 stepped
diagonally truncates to a step of 0 and the law computes `ceil(256/0)`, which has no value.
A minimum step of 1 gives the longest transit the 1/256 grid allows, 256 ticks. Unreachable
from shipped data, whose smallest `v` is 2; a customisation limit rather than a hazard, and
disclosed in `provenance.md`.

**DD-4 — A transit is a duration; the cell is taken at its start; the gate stands ahead of
the target test.** In `Step`'s per-entity arm, after the aliveness test and **before** the
target test:

```
if e.Transit > 0 { e.Transit--; continue }
```

Ahead of the target test because a mover that arrives mid-stride clears its order and must
still finish what it owes — that is what lets the drawn body slide home rather than freeze
in the air. The rate is frozen at the step itself, so "once per cell transit" is a position
in the code rather than a rule someone keeps. A `TransitTotal` of `t` is written with
`Transit = t-1`, the step's own tick being the transit's first, which makes the cadence
exactly `t` ticks a cell and not `t+1`.

**DD-5 — A non-positive speed is a mover with NO RATE.** One predicate, `rated(e)`, in one
place. The law's answer for a speed of 0 is `v = 1` — a cell per 256 ticks — and every world
built here before this story names no speed, so taking it would crawl every existing fixture
for a value none of them meant to set. The trade `MaxHP <= 0` already makes, and it keeps
the new behaviour opt-in at the one place that opts in, the loader.

**DD-6 — Three fields, and what the byte form refuses.** `Speed int32` at record `+35`,
`Transit uint16` at `+39`, `TransitTotal uint16` at `+41`; record 43 bytes, version **7**.
The pair rather than one number because the original holds both — the ticks a transit needs
and the ticks it has run, in the mover block its save writes whole — and because the owed
count alone cannot say how far through a transit a mover is, which is what the drawn
displacement asks. Refused by the decoder **and by the constructor**, so the two cannot come
to disagree: a `TransitTotal` above the law's maximum of 256; an owed count at or past its
own transit's length; a nonzero either on a unit that is not alive. Speed is range-checked
nowhere, for the reason the health pair is not: every int32 is a state the constructor
accepts.

**DD-7 — The seam holds a mover's step for the whole transit.** `recordCells` skips an
entity that owes transit ticks, so the previous cell it records stays the cell the transit
began at, and the derived step vector, the walking classification and the facing survive the
transit unchanged — one condition in the one function that writes the memory, rather than a
second memory beside it. `entityShift`'s fraction becomes

```
left = Transit·period + (period − clamp(elapsed, 0, period))
den  = TransitSpan·period
```

which is the present formula exactly when the span is 0 or 1, so an unrated mover and every
front-end pushing no span draw what they drew. The arithmetic is 64-bit explicitly: a span
of 256 and a period of a second put the numerator past 2³¹.

**DD-8 — The loader reads `Speed` at the statement that already reads health and domain**,
and an unresolved placement takes `DefaultSpeed`, a named constant equal to
`data.UnitDefaults().Speed`. One resolution, one statement, no second path on which a
placement gets a speed and not a health. Named at the spawn site as `SpawnHP` is, and for
the same reason.

**DD-9 — The measurement is a developer verb and a headless race.** `classdump -databin`
already prints a per-placement line resolved through the world; it gains the entity's speed
and the two transit lengths its rate yields. Beside it, a test orders two movers of
different speeds the same distance and prints ticks-to-arrive — the check that catches a
rate wired to the wrong term, since a 1:1 ratio passes every arithmetic test above.

## Risks

| Risk | Mitigation |
|---|---|
| Every unit becomes about sixteen times slower, which reads as a regression rather than the fix. | It is the law's own number: at the shipped default the worked example crosses a cell per full tick, ≈992 ms. Named in `spec.md` as a disclosed limitation and in the build note, so the owner meets it as an intention. |
| The two arms of FR-1 compute the same number here, so a swapped fork would be invisible in a world test. | AC-2 separates them on the pure function, at mean costs the world cannot yet supply. Today's flyer stories are also re-run whole. |
| A version bump moves every pinned digest in the tree. | Expected — the domain's arrival did the same at version 6. Every pin is re-derived from a run, not adjusted by hand, and AC-9 re-establishes that one stream gives one digest. |
| A mover could starve on a count that never reaches zero. | The count is written once, from a value bounded by 256, and only ever decremented; P-2 and the decoder's refusal make an out-of-range pair unrepresentable, and AC-5 counts ticks-to-arrive rather than asserting arrival. |
| A mover felled mid-transit, or a world decoded mid-transit, leaves residue the form refuses. | The pair is cleared where a felled unit's order is cleared, normalised by the constructor, refused by the decoder — the three sites the target and stall rules already use. |
| The seam's skip is a second rule that must agree with the sim's gate. | Both read the same field, and the skip is one condition in the one function that writes the memory. AC-8 drives a whole transit through the seam. |

## Success criteria

| # | Criterion | How |
|---|---|---|
| **SC-1** | The law reproduces its worked example and its two arms, and every clamp, wrap and substitution behaves as transcribed. | automated |
| **SC-2** | `(v·707)/1000` equals the truncation of `v` times the shipped double for every `v` in `0…999`, the bound the argument carries. | automated |
| **SC-3** | The straight transit takes exactly **27** distinct values over `v ∈ [1,63]`, and `v` 52…63 all take 5. | automated |
| **SC-4** | A rated mover crosses one cell per its own transit, searching on no tick between; two of different speeds arrive in the predicted ratio; an unrated one crosses a cell a tick. | automated |
| **SC-5** | A world naming no speed advances cell-for-cell as it did, and the pre-existing simulation suite passes unchanged. | automated |
| **SC-6** | Version 7 round-trips a mid-transit world; each refused shape fails by name; the digest moves with each of the three fields. | automated |
| **SC-7** | A units placement takes its entry's `Speed`; one resolving to nothing and every placement of a table-free world take the default; no entity is unrated. | automated |
| **SC-8** | A mover mid-transit reads as walking with one step vector and one facing throughout and its displacement runs over the transit; an unrated one's is unchanged. | automated |
| **SC-9** | `pkg/sim` carries no float, banned import or forbidden literal, and one stream gives one digest at every tick under three pacing schedules. | automated |
| **SC-10** | On a lawful install, both roots: the shipped `Speed` alphabet, the transits it yields, and the diagonal ratio inside 0.97…1.15 — which requires 23 to be absent. | developer-run, **both roots** |
| **SC-11** | In the built application two classes of different `Speed` ordered the same distance separate visibly, the faster arriving first, both walking smoothly. | manual, owner's seat |
| **SC-12** | The gate is clean, the import graph refuses the edges it refused before, and the landing deletes no file it does not name. | automated |

## Traceability

FR-1 → DD-1, DD-5 → SC-1, SC-5 · FR-2 → DD-2, DD-3 → SC-2, SC-3 · FR-3 → DD-4 → SC-4 ·
FR-4 → DD-6 → SC-6 · FR-5 → DD-8 → SC-7 · FR-6 → DD-7 → SC-8, SC-11 ·
FR-7 → DD-9 → SC-9, SC-10, SC-12.
