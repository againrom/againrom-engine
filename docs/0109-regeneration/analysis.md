# Analysis — 0109, regeneration

## What was missing

Nothing in this tree gives health or mana back. A unit struck once carries the wound for the rest
of the mission, and `pkg/sim.Entity` has no mana at all — no pool, no maximum, no period, no
accumulator. The player's own party is the sharpest case: it is placed at 100/100 and every point
it loses is permanent.

## What is already built, and what is not

Three things had to be told apart before any of this could be scoped, because two of them look
like the thing being asked for and are not.

**The two periods are already streamed.** `pkg/data.UnitDef` carries `HealthRegenPeriod` and
`ManaRegenPeriod`, defaulted to 100 and 50 by `UnitDefaults()` and overwritten from the Units
row's slots 5 and 7. So the data half of this story landed with 0049 and this story adds nothing
below `pkg/sim`. `pkg/data.HumanDef` carries a mana pair and **no** periods, which is the shape
the Humans table itself has.

**The dead-list decay ladder is built and is a different machine.** `decayPass` in
`pkg/sim/step.go` walks bodies down one health point every second full tick after the teardown.
That is the machine 0033 built.

**The dying arm is not built.** Of the three asymmetries the regeneration row states as
instructions, none is witnessed in this tree:

| Asymmetry | Built? | Where the nearest thing is |
|---|---|---|
| health doubled, mana not | no | there is no regeneration at all |
| health one full tick in four, mana every full tick | no | as above |
| a dying actor loses one point per four full ticks | **no** | `decayPass` is the *dead*-list machine at one point per **two** full ticks, reached only after the teardown; nothing moves health while the dwell runs |

The third row is the one worth stating carefully, because it is the same routine. In what is
being reconstructed, one virtual slot carries both arms: the living arm regenerates and the dying
arm decays a body that is still on the live list, and the second is what carries a body felled at
-1...-9 down to -10 so the teardown guard can fire. This tree drives the teardown off the dwell
alone — a divergence 0033 disclosed — so building the dying arm here would put a second clock on
health with no teardown rule to make the two agree. It is left where 0033 left it.

## Where the entry test already agrees

`Alive()` is `!Dead() && !Downed()`, so it is false at exactly `HP <= 0` for any entity with a
health system. That is the routine's own entry test — non-positive health leaves the living arm —
and it means health exactly 0 is a fixed point here for the same reason it is one there: the
living arm never sees it and the dying arm this tree does not have is the only thing that could.
The agreement was not designed for; it fell out of a predicate written for the death story.

## What the arithmetic actually is, and why it cannot be simplified

The accumulator is a x100 fixed point with the remainder kept in a byte of its own. The gain per
qualifying tick is a product divided **once, at the end**, and the x100 is inside the product:

```
gain = healthMax x 2 x 100 x rate / period          (hundredths of a point)
```

At a 45-health unit with the default period of 100 this is 90 hundredths — nine tenths of a point
per qualifying tick, which only accumulates because the remainder survives the tick. Cancelling
the two hundreds to `healthMax x 2 x rate / period` looks like the same expression and is not:
it truncates to **0** for every unit whose doubled maximum is below its period, which is most of
them. Any implementation that rounds the remainder away heals such a unit never rather than
slowly.

## The two things this tree cannot supply

**The rate-3 idle bonus.** The rate is 3 instead of 1 when the actor has been idle for more than
80 sub-ticks, measured as *now minus the sub-tick the current action run was due to end*. There is
no such quantity here. `Stall` counts consecutive ticks a target could not be reached, and
`AttackCountdown` is what remains of an attack phase; neither is an action-run deadline, and
substituting one would be a different idle notion wearing this one's name.

**The two regeneration modifiers.** The `(modifier + 100)` term makes an effect a percentage of
the base rate rather than an addend, and the only writers of those two fields are the effect
arms. This tree has no effects at all — `pkg/sim` contains no such word — so the term has no
writer, only a reader.

## What that leaves open

Whether the mana arm is observable on the shipped corpus at all. The published census says 50 of
56 unit classes ship a zero mana period and **none** of those pairs it with a non-zero mana
maximum — so every row that could regenerate mana has a usable period, and the divisor hazard is
unreachable there. It does not say how many rows carry a mana pool. A count of those rows, and
whether mission 10 places any of them, is a measurement for the verification stage.
