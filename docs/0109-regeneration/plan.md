# Plan — 0109

Four slices, bottom up. The record and the form first, because every pinned digest in two
packages moves with the record's width and nothing else should move in the same commit. Then the
pass, which by P-4 moves no digest at all. Then the placement, which moves the loader's. Then the
panel.

## The shape of the change

One new file, `pkg/sim/regen.go`, holding the pass, its constants and one arithmetic helper. Six
fields on `Entity`, one fault function beside the five already there, eighteen bytes at the entity
record's tail and a version. Four numbers carried through `pkg/mapload`'s spawn block. One field
and one row on the panel.

`pkg/data` is untouched: it already streams both periods and both mana pairs.

## FR by FR

**FR-1 — the record.** All six at `Entity`'s **tail**, in one block and in the order the form
writes them: `Mana, MaxMana int32`, then `HealthRegenPeriod, ManaRegenPeriod int32` named as
`pkg/data` names them so one number has one name across the tiers, then `HealthHundredths,
ManaHundredths uint8`. Declaration order is pinned by `nostate_test.go`, so it is a decision and
not a layout preference; at the tail it disturbs no existing row of that pin. The two remainders
are bytes because a hundredth of a point is what they hold and 99 is their largest legal value —
the field's own width, not a saving.

**FR-8 — the fault.** `regenFault(e Entity) error` beside `decayFault`, refusing a remainder
above 99 and nothing else, and in **`reachFault`'s** shape rather than `decayFault`'s: **the
decoder is its only caller.** The constructor folds each out-of-range remainder to 0
unconditionally and inline, before this function could have anything to refuse — a constructor
call would be one that can never return non-nil. No refusal is added for a negative pool, a
negative period, a negative maximum or a mana above its maximum: each is a value the arithmetic
answers for (DD-2, DD-7, DD-10) and none is a shape the pass can produce.

**FR-9 — the form.** `formatVersion` 25, the entity record eighteen bytes wider, the six numbers
appended at the tail in the order FR-1 lists them, `regenFault` on decode, one version paragraph
in the voice of those above it, and the offset-table rows.

**The live version at the branch point is 24 and this story takes 25.** It was allocated 26 on a
false premise — that `0108` held 25 — and `0108` and `0110` each spend none, so 26 would have left a
hole. The lane refused to renumber itself and reported the conflict instead, which is the rule
working; the seat corrected it. Read `formatVersion` and `entityLen` off the branch point rather than
off any figure in this paragraph.

Both version-named tests — the sweep and the single-version refusal — are **renamed to carry no
number** (DD-11). `preRegenFormVersion` is a **literal**, in `prePostFormVersion`'s own shape: a
constant derived as `formatVersion - 1` would make the refusal assertion self-fulfilling and buy
nothing. The refusal's message assertion takes both numbers through `strconv`.

**FR-2 — the pass.** `regenPass()` on `World`, called from `stepWorld` immediately before
`decayPass()` and after every arm of that sub-tick that can change a health. It tests its own
phase — the ladder's shape, and no more than its shape: `decayPass` is itself called every
sub-tick and phase-gates only its walk, so there is no dispatch here to join. The two calls are
adjacent because in what is being reconstructed they are two loops of one dispatch, and the
live-list one runs first.

**FR-6 — who is advanced.** `if !e.Alive() { continue }` and nothing else. See DD-1.

**FR-3, FR-4 — the two arms.** One helper serves both, because they differ by exactly a factor
and a cadence and a rule written out twice is a rule that comes to differ between its two
copies:

    func regenerate(cur *int32, max, period int32, rest *uint8, double int64)

It returns at once on `max <= 0`, on `*cur >= max` or on `period <= 0` — **all three, and the
first is not optional** (DD-10) — then computes the gain, the accumulator, the remainder and the
capped store, in that order. The caller passes 2 for health and 1 for mana, and calls the health
arm only on a qualifying tick.

Five named constants, in the file with the pass: `regenPhase = 12`, `manaRegenCycle =
scriptCycle`, `healthRegenCycle = 4 * scriptCycle`, `regenScale = 100` and `regenPercent = 100`.
`regenPhase` is written out rather than aliased to `decayPhase`: the two share a value and not a
reason — the ladder's is a free choice between two legal slots and this one is the dispatch slot
itself — and an alias would let a later story's decision about the ladder move regeneration.
The last two are the same number and are two constants on purpose: one is the fixed point's
scale and the other is the modifier term at a modifier of zero, they cancel by arithmetic
accident rather than by identity, and folding them into one would make the expression look like
`max * 2 / period` — which truncates to zero for most units and is the single way this story can
be built so that it appears to work and heals nobody.

**FR-5 — the rate.** `regenRate = 1`, a sixth constant, written into the product rather than
omitted from it, so the cut is a value a later story changes and not a shape it has to rebuild.

**FR-7 — where the numbers come from.** `spawnBlock` gains `mana, manaMax, healthPeriod,
manaPeriod`. The creature arm reads all four off `UnitDef`. The humans arm reads the mana pair off
`HumanDef` and takes both periods from `data.UnitDefaults()`, because the Humans table carries no
such column and the base constructor's 100/50 is what a hero keeps — one source for that pair,
never a literal written here. **The unresolved arm needs all four written out too**: it holds the
whole constructor definition in a local, but the block it returns is a composite literal naming
its fields one at a time. `start.go`'s party mint writes the two periods off the same
`data.UnitDefaults()` and writes no mana pair at all.

**FR-10 — the panel.** `PanelFieldMana` beside `PanelFieldHealth`, `Mana, MaxMana int` on
`PanelSubject`, one arm in `panelText` returning `"%d/%d"` and reporting `s.MaxMana > 0`, one
`{Field: PanelFieldMana, Label: "MANA"}` row in the layout directly under the `HP` row, and the
same pair on `ui.MapEntity` (`pkg/ui/overlay.go`) beside its health pair, filled at `pkg/game`'s
one site — the panel is built from a `MapEntity` and never from a `sim.Entity`. The
`MaxMana > 0` test is the rule `PanelFieldCount` already uses: a field with nothing to say omits
its row.

## Decisions and divergences

**DD-1 — the pass is gated on `Alive()`.** The decoded entry test is on health alone, and for
every entity with a health system `Alive()` is exactly that test. The one entity it also admits is
the 0/0 one this tree lets a hand-built world hold, and both arms' gates are false for it unless
it was given a mana maximum — a shape the original has no actor for, so our answer for it is ours.

**DD-2 — a non-positive period regenerates nothing, on both arms.** The health arm has that gate
decoded; the mana arm has none and divides unguarded. The census says the case is unreachable on
the shipped corpus — no class pairs an absent period with a pool it could fill — so reproducing
the trap would reproduce a behaviour nothing exercises, and a negative period is not decoded at
all. One rule for both arms, so the two cannot come to disagree about what an absent period means.

**DD-3 — the remainder is a byte and out-of-range is folded by the constructor, refused by the
decoder.** The split is `Reach`'s: a caller cannot act on an error about residue, and a decoded
record is a claim rather than a default someone reached for.

**DD-4 — the modifier term is cut and fixed at 100.** Its two fields are written only by effect
arms and this tree has none, so carrying them would serialise a zero forever and buy a reader for
a field with no writer. The term stays visible as `regenPercent`, so the effect story adds a
field and multiplies, rather than rediscovering where the term goes.

**DD-5 — the accumulator is 64 bits.** The original's intermediate width is not published and
does not need to be: at the widest values the record can hold the product is under 10^12, so no
representable input can overflow it and the choice cannot change an answer. A 32-bit accumulator
could, which is why this is a decision and not an implementation detail.

**DD-6 — the four-tick phase follows from the model rather than being chosen.** The filter is
`counter mod 4 == 0` on the absolute full-tick counter, and this package models that counter as
its own tick divided by sixteen. Given that model, full ticks 0, 4, 8 ... carry health by
arithmetic and there is no phase left to pick. What is assumed rather than derived is the model
itself — that tick 0 here is sub-tick 0 there — and that is the definition of tick 0 rather than
a second fact. It differs from the ladder's own comment, which records its phase as **ours**,
because that filter is on a counter this package does not model at all.

**DD-7 — the remainder is taken non-negative.** `acc` is non-negative for every state the pass can
reach, and a hand-built world holding a negative mana can make it negative. Go's `%` would then
answer a negative remainder, which the byte cannot hold and the decoder would refuse. The helper
adds the scale back when the remainder is negative and takes the quotient from the adjusted value,
so the two agree and the function is total over every `int32` — P-3, at the cost of three lines.

**DD-10 — a non-positive maximum regenerates nothing.** Both maxima are unsigned in what is
being reconstructed, so a negative one is a state it cannot hold and says nothing about. This
record holds them signed, on the tree's standing rule that a negative stays visibly negative. The
gain is proportional to the maximum and takes its sign, and the `cur < max` gate does not close
under a negative one — so without this, the mana of an entity below a negative maximum falls
forever and eventually wraps its own width, breaking P-2 and P-3 on a state the constructor
accepts. The health arm is shielded by `Alive()` and the mana arm is not, which is why this is one
gate in the shared helper rather than a clause on one arm.

**DD-11 — both version tests are renamed to carry no number.** The tree's own record argues the
opposite: the sweep's name is said to move with the number so the two cannot come apart. That
argument has failed three times running and been repaired after the fact each time, and the same
comment ends by naming the real fix as a check that reads the name rather than a paragraph asking
the next lane to. A version-free name loses nothing, because the assertions already read
`formatVersion`: the number was in the name as a label, and a label that must be edited at every
bump is the staleness rather than a guard against it.

**DD-8 — the party member gets periods but no pool.** His health is `SpawnHP` rather than a
derivation already; a mana pair invented here would be a second authored number, and the derived
one is published whenever a story wants it.

**DD-9 — the pool fields are carried by death unchanged.** The constructor's not-alive block
clears four fields that are residue of a state the unit has left. A remainder is not residue in
that sense — it is a fact about the unit, as its facing is — and a corpse's is read by nothing.
Adding a clause would move digests for no observable difference.

**DD-12 — difficulty scales health regeneration and not mana, and that is faithful rather than
overlooked.** `Adjust` moves `HealthMax` by 66/100 and 3/2 and touches neither mana field. The
gain is proportional to the maximum, so an Easy unit regenerates fewer points per tick in
absolute terms and a Hard one more, while its mana rate is unmoved. Nothing here reads a
difficulty; the effect falls out of scaling the maximum, which is where the original scales it
too, and it is written down so that a later reader does not take it for a defect.

## Risk

**The milestone moves.** Every placement on mission 10 gains a period, so the drive's health
values change and its outcome may. AC-9 states the prediction and the direction; the number is
recorded and the baseline is the orchestrator's to move. The one failure mode to guard is tuning
a constant until the drive prints something nicer — no constant in this story has a free value.

**The rebase.** 0108 lands version 25 and may change the entity record's width. Every absolute
number in this plan about the form is relative: read `formatVersion` and `entityLen` from the
branch point and add.

**A green-but-hollow build.** Dropping the remainder, or cancelling the two hundreds, yields a
pass that compiles, is deterministic, moves no digest it should not, and heals no unit whose
doubled maximum is under its period. AC-2 is the only assertion that separates the two.
