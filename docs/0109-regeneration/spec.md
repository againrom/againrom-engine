# Spec — health and mana come back over time

Intensity: **spec-anchored / static**. Terrain: **brownfield** in `pkg/sim`, `pkg/mapload`,
`pkg/game` and `pkg/ui` — every changed unit is pinned before it changes; **greenfield** for the
regeneration pass itself, which is new code in a new file.

## Terms

- **Sub-tick** — one call of `Step`. It is the unit `World.tick` counts.
- **Full tick** — sixteen sub-ticks. The **full-tick counter** is `tick / 16`, and it is what the
  four-tick filter below is taken over.
- **Period** — the per-unit divisor of a regeneration rate. A larger period is slower.
- **Hundredths remainder** — the fractional part of a pool, in hundredths of a point, carried
  from one qualifying tick to the next in a byte of its own. It is the whole reason a unit whose
  gain is under one point per tick heals at all.
- **Gain** — the hundredths a qualifying tick adds to a pool, before the remainder is applied.
- **Pool** — health or mana; each has a current value, a maximum, a period and a remainder.

## Why

A unit that is struck never recovers, and mana does not exist in the simulation at all. The
player watches his own party lose health permanently across a mission. Regeneration is the rule
that makes a wound temporary, and it is the last thing missing before health means anything over
a mission rather than within a fight.

It is also load-bearing on the one mission this project drives end to end: the health a unit
holds when a blow lands decides whether it lands fatally, so the mission's outcome is downstream
of this rule.

## Scope

**In.** The two pools and their four supporting numbers on the simulation entity; the pass that
advances them; the byte form that carries them; the numbers a placement and a party member are
born with; and a mana readout for the unit panel.

**Out.**

- **The dying arm.** In what is being reconstructed the same routine also removes one health
  point every fourth full tick from a body that is still on the live list, and that is what feeds
  a teardown guard this tree does not have — 0033 drives the teardown from the dwell instead.
  Building half of the pair here would put two clocks on one field with no rule to make them
  agree. It stays where 0033 left it.
- **Effects.** Nothing in this tree writes the two regeneration modifiers, so the term they enter
  through is cut (FR-5).
- **A hero's derived mana maximum.** A party member's health is `SpawnHP` rather than a
  derivation — a standing divergence of 0078 — and his mana pool is now absent on the same
  ground. This story derives nothing from Spirit.
- **Spending mana.** No spell, no cost, no gate. Mana is regenerated and displayed and nothing
  reduces it.
- **Tuning toward a mission outcome.** AC-9 predicts what the drive prints and no number in this
  story is free to be chosen so that it does.

## The contract

### The record

**FR-1.** A simulation entity carries a **mana pool** (`Mana`, `MaxMana`), a **health period**
and a **mana period** — four signed integers at the width the health pair already has — and a
**hundredths remainder** for each pool, one byte each. The mana pair is the shape the health pair
already is, and carries the health pair's rules and no others: a maximum of zero is a unit with no
mana system, and nothing constrains a current value against it. The two periods and the two
remainders have no meaning except to the pass in FR-2.

**FR-8.** A remainder is a value in the range 0 to 99. Anything above has no meaning as a
fraction of a point: the **world constructor folds it to zero**, because a caller handed a stale
accumulator has nothing it could do with an error, and the **decoder refuses it**, because a
decoded record is a claim about a saved unit and no value above 99 is a claim this package can
have written.

**FR-9.** The canonical byte form carries all six numbers, at the entity record's tail so that no
offset in front of them moves, and the **format version moves to 25**. **Every other version byte
is refused**, as every other version byte already is; the immediately previous one is the sharp
case, because a well-formed stream at it exists and nothing but the version check separates a
correct decode from reading each record's tail out of its neighbour. A form written before this
story says nothing about a pool or a remainder, and "no mana" is not a gap a reader may fill — a
decoder that supplied one would hand back a world whose units heal on a different schedule than
the world the bytes were cut from.

### The pass

**FR-2.** Regeneration runs **once per full tick**, on one fixed slot of the sixteen-sub-tick
cycle: the slot the decay ladder's own walk is filtered on, because in what is being
reconstructed the two are loops of one dispatch. Within the sub-tick that carries it, it runs
**after every arm that can change a health** and **before the decay ladder** — three further
sub-ticks of the same full tick land after it, which is the original's order too.

**FR-6.** It advances only an entity that is **alive**. A non-positive health is left exactly
where it stands: a body is not healed, and a unit at exactly zero health stays at exactly zero
health for as long as the world holds it.

**FR-3 — the mana arm.** It runs on **every** full tick. Its one decoded gate is a current mana
below the maximum — it has **no period gate of its own**, and the two paragraphs below FR-4 add a
period gate and a maximum gate. Its gain is

    gain      = MaxMana * 100 * rate / ManaPeriod           hundredths
    acc       = Mana * 100 + remainder + gain
    remainder = acc mod 100
    Mana      = min(acc / 100, MaxMana)

**FR-4 — the health arm.** It runs on a full tick whose **counter is a multiple of four** — one
full tick in four. It is gated on a current health below the maximum **and** on a usable health
period **and** on a positive maximum, and its maximum is **doubled** where mana's is not:

    gain      = MaxHP * 2 * 100 * rate / HealthPeriod       hundredths
    acc       = HP * 100 + remainder + gain
    remainder = acc mod 100
    HP        = min(acc / 100, MaxHP)

In **both** arms the division is taken **once, on the whole product**, and the remainder is
stored **whether or not the pool was capped**.

**A period that is zero or negative regenerates nothing on its own arm**, and the other arm is
unaffected. This is the whole answer to the divisor, and it is what gives the mana arm the gate
FR-3 says it does not decode: no shipped class pairs an absent period with a pool it could fill,
so the case is unreachable on the corpus, and an unreachable divide is still a divide.

**A maximum that is zero or negative regenerates nothing either**, on both arms. What is being
reconstructed holds both maxima unsigned and cannot represent a negative one; this record can,
and without this gate a pool below a negative maximum would be driven **downwards** without
bound — the gain is proportional to the maximum and takes its sign.

**A gain that truncates to zero is a unit that never regenerates**, and that is the arithmetic's
answer rather than a defect: a period above a hundred times the doubled maximum leaves the
remainder empty every tick, so nothing ever accumulates. No shipped period reaches it.

**FR-5.** The rate term above is **1 for every actor on every tick** — the base rate. What is
cut is a threefold bonus for a unit that has been out of action for more than eighty sub-ticks; measuring
that needs the sub-tick a unit's current action run was due to end, and no field here holds one.
**What it costs:** an idle unit recovers at a third of the decoded rate, so a unit standing out
of a fight takes three times as long to come back to full as it should. It is a rate and never a
different outcome — nothing else in the contract reads it.

### Where the numbers come from

**FR-7.** A **placed** unit is born with its class row's two periods and its class row's mana
pair, by the same route its health pair already takes; a placement that resolves to no row takes
the definition tier's own defaults whole, as it already does for every other number. A **party
member** is born with those same default periods and **no mana pool**, on his health's
standing divergence. A world assembled any other way keeps exactly what it was handed — the world
constructor supplies no period, so a hand-built world does not regenerate.

### What the player sees

**FR-10.** The unit panel shows a **mana pair** in the health pair's form, and shows it **only
for a unit that has a mana pool**: a unit with a maximum of zero has nothing to say and omits the
row rather than drawing a zero.

## Acceptance

- **AC-1.** On a qualifying tick a pool below its maximum gains exactly the hundredths FR-3 and
  FR-4 compute, and on a non-qualifying tick it gains nothing. Asserted per arm, with the health
  arm shown to move on one full tick in four and the mana arm on all four.
- **AC-2.** A unit whose gain is **under one point per tick** still reaches its maximum. A unit of
  maximum 45 at a period of 100 gains 90 hundredths per qualifying tick: started at 40 with an
  empty remainder it stands still on its first qualifying tick, takes its first whole point on the
  second, and reaches 45 on the sixth. An implementation that dropped the remainder heals it
  never, and is red here and nowhere else.
- **AC-3.** Neither pool ever exceeds its maximum, at any period, from any starting value.
- **AC-4.** A period of zero and a negative period both regenerate nothing on their own arm and
  leave the other arm working. No input panics.
- **AC-5.** An entity that is not alive gains nothing. A unit at exactly zero health with a
  positive maximum is unchanged after any number of ticks.
- **AC-6.** A world carrying every new number round-trips through the byte form unchanged; a form
  at the previous version is refused with both version numbers named; and stripping the new block
  from a current form, **with the version byte put back**, reproduces the previous story's digest
  byte for byte.
- **AC-7.** A placement carries its row's periods and mana pair onto the entity, an unresolved
  placement carries the definition tier's, and a started party member carries the tier's
  periods with an empty pool.
- **AC-8.** The panel draws the mana row for a unit with a pool and omits it for one without.
- **AC-9.** Mission 10, driven on both installed roots with the milestone's own arguments
  (`-mission 10 -census -waypoint u21:56:21:3 -waypoint p0:66:16:3`, at that drive's own default
  difficulty), ends `lost` at a tick **no earlier than 272**, and the two roots print the same line. This is a gate and not an
  observation: regeneration only adds health, so the unit whose fall ends the drive should fall
  later or not at all. A drive that fails it is a **discrepancy to classify** — a finding about
  the contract or the arithmetic — and never a number to tune toward.
- **AC-10.** A measurement, recorded and not asserted: how many of the 56 parameterised Units
  rows carry a non-zero mana maximum, and how many of mission 10's placements resolve to one.

## Properties

- **P-1 — determinism.** The pass reads no clock, no float and no random source, and consumes no
  draw from the world's generator. The package's source scan stays green.
- **P-2 — monotone.** Regeneration never lowers a pool and never raises one above its maximum.
- **P-3 — total.** Every combination of health, maximum, period and remainder the record can hold
  has an answer: no division by zero, no overflow of the accumulator, no panic.
- **P-4 — inert without a period.** A world whose entities carry no period is not changed by the
  pass: every field of every such entity is what it would be with the pass removed, so no digest
  pinned against such a world moves when the pass arrives.
- **P-5 — the remainders have one writer.** The pass is the only thing that writes either
  remainder. It adds no writer of a pool: the writers of health are what they were.
