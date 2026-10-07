# Analysis — the two checks mission 10 cannot answer

Two check arms are unimplemented on the campaign's first mission, and this story was opened on the
reading that both stand between that mission and a win that fires. What we did not know was which of
the seven inert triggers each arm owns, and whether either one gates the arm that wins.

## What was measured

The campaign's 28 maps were taken out of `scenario.res` on **both** installed roots and compiled
through the binder. For each map, and for each of five hypotheses — nothing added, arm 1 alone,
arm 14 alone, both, every arm — the compiled program's register ownership was replayed to work out
which triggers stop being inert, and which of the live ones carry the win instant.

Mission 10's map is **byte-identical across the two roots**, so its whole reading is one reading.
Two of the 28 differ across roots; only one of those two, `100.alm`, differs in compiled trigger
count (17 against 16), and it carries no reachable win under any of the five hypotheses.

## Whole-campaign counterfactual

Triggers summed over the 28 maps, then the number of maps in which some **live** trigger carries the
force-complete instant. The two roots agree on every column.

| hypothesis | live | inert | maps with a live win-carrying trigger |
|---|---|---|---|
| nothing added | 165 | 233 | 11 / 28 |
| arm 14 alone | 192 | 206 | **11 / 28** |
| arm 1 alone | 224 | 174 | **20 / 28** |
| arm 1 and arm 14 | 251 | 147 | 20 / 28 |
| every check arm | 398 | 0 | 28 / 28 |

**Arm 14 moves no map into the reachable set and arm 1 moves nine.** That is the whole reason this
story implements one arm and not two.

## Mission 10, node by node

Sixteen checks, twenty-seven instants, twelve compiled triggers, one trigger dropped by the binder.
Five checks are arm 1 and one is arm 14; the eight unresolved references are all the hero ordinal.
Seven triggers are inert: six read one of the five arm-1 registers and one reads the arm-14 register.

The win chain is three triggers at map positions 2, 3 and 4:

- **position 2** — the hero within 3 of `(36,51)` **and** the count of group 1 at zero, then hand
  group 2 to player 1. **Inert**: its second pair reads an arm-1 register.
- **position 3** — the escortee (the single member of group 2, placed at `(36,51)`) within 3 of
  `(56,21)`, then increment mission variable 50. Live.
- **position 4** — the hero within 3 of `(66,16)` **and** variable 50 nonzero, then force complete.
  Live.

The arm-14 check reads cell `(38,64)`; its one reader is the trigger at map position 5, whose only
instant raises a message. It is on no chain that decides anything.

## The surprise, and what it costs

**The win trigger is not inert today** — it is live, and vacuously satisfied on its first term,
because the hero reference does not resolve and an unresolved reference writes no register, leaving
a permanent zero that compares below 3. Its second term is what holds: variable 50 stays zero.

So the win instant is reachable and the chain that arms it is not, and the blockage is not only this
story's. The escortee is placed 30 cells (Chebyshev) from the cell position 3 measures, and nothing
in this build moves it there. What the shipped map does is hand it to the player at position 2 —
through the group-to-player instant, which this build does not run. **This story removes the first of
two gates on that chain; the second is that instant, and it is not a check.** The measurement also
says the hero-distance terms are currently vacuous rather than satisfied, so the chain's arithmetic
is only visible once the hero binds.

## Where group membership comes from

The map already carries it: the placed-unit record's group word is decoded and exposed, and the
binder already pairs each record with the entity it becomes. What is missing is at both ends —
the compiled check drops the group parameter, because the binder packs only the plain-integer
parameter kinds and a group reference is not one of them; and the entity carries no group at all.
The one group-shaped field an entity has is the rate term, which is a speed and not a membership.

## What is not established

Nothing read says whether the arm counts **every** member or only the living ones. The distinction is
not cosmetic: the shipped trigger compares the count against zero, so under the whole-membership
reading mission 10 can never be won, and under the live-membership reading it can. The engine's own
arm was not read at instruction level by anything published.
