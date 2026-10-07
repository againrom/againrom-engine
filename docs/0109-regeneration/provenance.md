# Provenance — 0109, regeneration

The contract is one decoded routine plus the fields it reads. Everything normative rests on
`HERO-REGEN-021` as amended four times; the supporting rows fix the clock it runs on, the fields
it reads and the state it refuses to touch.

## What the contract rests on

| Spec anchor | Source | Confidence |
|---|---|---|
| One pass per full tick moves both pools; health and mana each keep a hundredths remainder in a byte of their own; the gain is a product divided once at the end (FR-3, FR-4, FR-5) | `HERO-REGEN-021` | High — the routine is read whole, both accumulators, both periods and the filter are named instructions |
| Health is doubled and mana is not (FR-4) | `HERO-REGEN-021` — `SHL EAX,0x1` on the health arm alone | High |
| Health advances on one full tick in four, mana on every full tick (FR-3) | `HERO-REGEN-021`, cadence fixed by `SESS-TICK-006` | High for both — the filter is the arm's own `AND`, the cadence is the dispatch slot `server+0x04 % 16 == 12` |
| Health is gated on `health < healthMax` **and** a non-zero period **and** the four-tick filter; mana **only** on `mana < manaMax` (FR-3, FR-4) | `HERO-REGEN-021` | High |
| A full tick is 16 sub-ticks, and the four-tick filter is on the absolute full-tick counter (FR-2) | `SESS-TICK-004`, `SESS-CLOCK-005` | High — the 16:1 relation is the gate's own compare; the rate ladder was re-read out of the PE |
| The pass runs on the live list and leaves non-positive health alone; health exactly 0 is a fixed point (FR-6) | `HERO-DECAY-069`, `HERO-ZERO-070` | High — both branch displacements are `rel8` onto the same epilogue; two independent instruments agree on the writer set |
| The two periods default to 100 and 50 and the Units row overrides them; the Humans table has no such column, so a hero keeps the defaults (FR-7) | `HERO-REGEN-021` as amended, `UNIT-CTOR-004` (already spent by 0049 FR-3) | High — both fills are named stores in the constructor and the streamer |
| Every class that could regenerate mana ships a usable period: 50 of 56 ship a zero one and **0 of 56** pair a zero period with a non-zero mana maximum (DD-2) | `HERO-REGEN-021` as amended | High for the census; it is a count over the shipped table |
| No stat is read by the tick — Spirit reaches mana only through `manaMax` and Mind not at all (Out of scope) | `HERO-REGEN-021` first amendment, `MAGIC-SPIRIT-011` | High — the enumeration is complete at 36 hits and every one was read |
| A hero's mana maximum is derived rather than streamed (Out of scope) | `HERO-MP-006` | High — the arithmetic and both multipliers are named instructions |
| The two modifier fields are written only by effect arms (DD-4) | `HERO-EFFECT-019` | High — the table is read from the image and `+0xe8+2i` has no other writer |

## Ours by choice

| Statement | Why it is ours |
|---|---|
| A non-positive period means no regeneration on that arm, on both arms (DD-2) | The health arm has that gate; the mana arm has none and the divisor is unguarded. Nothing decoded says what a zero mana period does, because no shipped row reaches it |
| A non-positive **maximum** means no regeneration on that arm, on both arms (DD-10) | Both maxima are unsigned in what is being reconstructed, so a negative one is a state it cannot hold and says nothing about. This record holds them signed, and the gain takes the maximum's sign, so the rule is ours and is what keeps the arithmetic monotone |
| The remainder is a `uint8` holding 0..99; the constructor folds anything above to 0 and the decoder refuses it (DD-3) | The byte's own width is decoded; what an out-of-range one means is not, and this tree's constructor/decoder split already answers that question the same way for four other fields |
| The accumulator is computed in 64 bits (DD-5) | The width of the original's intermediate is not published and does not need to be: 64 bits is wide enough that no representable maximum, period or health can overflow it, so the choice cannot change an answer |
| The pass is gated on `Alive()` (DD-1) | The decoded entry test is on health alone. For every entity with a health system `Alive()` is exactly `HP > 0`; the health-system-less 0/0 entity it also admits is a shape the original has no actor for |
| Which of the four full ticks in a period carries health is **not** ours | The filter is on the absolute counter and this package's tick is that counter times 16, so the phase is fixed rather than chosen — unlike the decay ladder's, which its own comment records as ours |
| The panel's mana row and its label (FR-10) | Every row of that panel is ours; the layout is authored and no claim describes it |

## Open, and what each costs

- **The rate-3 idle bonus (FR-5).** Cut. It needs the sub-tick a unit's current action run was due
  to end, which no field here holds. The cost is stated in the spec: everything regenerates at the
  base rate, so an out-of-action unit recovers at a third of the decoded rate. `HERO-REGEN-021`
  publishes the trigger and the threshold, so the seam is a field, not a decode.
- **The two regeneration modifiers.** Cut, and cheaper: the fields have no writer in this tree,
  so carrying them would serialise a zero forever. `HERO-EFFECT-019` names the arms that write
  them, so the effect story that adds them adds the term with them.
- **A hero's derived mana maximum.** Out of scope, on the health pair's own standing divergence
  (0078): a party member keeps `SpawnHP` and now keeps no mana pool at all. `HERO-MP-006` has the
  arithmetic when a story wants it.
- **The dying arm.** Out of scope. `HERO-DECAY-069` and `HERO-DWELL-065` publish it whole,
  including the teardown guard it feeds; 0033's dwell-driven teardown is what it would have to be
  reconciled with, and that is that story's contract to revise.
- **How many shipped rows carry a mana pool.** Not published. The verification stage measures it.

## Removed

Nothing. No statement was dropped between the first draft of the contract and this one.
