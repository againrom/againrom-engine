# Analysis — what a frozen radius is worth, measured before it was built

## What was reported

Units and enemies drift across the map and eventually collide into fights; a village kills the
beasts near it with no player involved. The report is a player's, so it is a question and not a
verdict — but three published rows already say the same thing about the mechanism, which is why it
arrived as a story.

## The premise handed to this lane, and what measurement did to it

The premise was a positive feedback loop: the notice circle is recomputed every tick from the
group's own spread, so a group that acquires something walks toward it, spreads, widens its circle,
and finds something further away. `pkg/sim/engage.go`'s `noticeRadius` does recompute per tick and
its own comment says so, so the first link is real. **The loop is not.**

Measured on the built worlds of the 28 campaign missions, EN root, stepped with **no commands at
all** for 2000 ticks — the recomputed radius of each guard group, first value against widest:

| Mission | Guard groups | Grew at all | Widest growth |
|---|---|---|---|
| 90 | 53 | 2 | 2 cells |
| 70 | 48 | 2 | 4 cells |
| 130 | 37 | 1 | 1 cell |
| 40 | 14 | 0 | 0 |

And in every one of those five instances the radius was **smaller at the end than at the start** —
a group converging on its target closes up, so its spread term falls. The recomputation is a real
divergence from the law and it is worth removing, but it is not an engine and there is no runaway.

## What the drift actually is

The same sweep — 28 missions, 2000 ticks, no commands — moves **79 of 2357 entities** a total of
**501 cells** and kills **35**. Logging the first attack order each entity takes on the worst map
(mission 90, 20 deaths) says where those come from:

```
tick 7  entity 1 owner 6 group 115 at (106,14) -> victim 10 at (107,15), member-victim 1
tick 7  entity 10 owner 7 group 36 at (107,15) -> victim 1 at (106,14), member-victim 1
tick 7  entity 117 owner 5 group 33 at (125,31) -> victim 25 at (126,30), member-victim 1
```

Twenty-five of the twenty-eight acquisitions happen on the **first decision tick**, between units
standing **one cell apart** on the map as it is authored. Two hostile rosters are placed
interpenetrated and see each other immediately. No radius, frozen or not, can prevent that: the
victim is inside every circle. The one late acquisition — tick 119, at 10 cells — is a unit that
walked into a neighbouring group's circle while chasing its own victim, and that group's radius was
**15 at load and 15 at the moment it fired**, so freezing changes it by nothing there either.

A census of the load-time circles says the same from the other side: over the 28 campaign missions,
**235 of 679** guard groups already hold a hostile inside their load circle, and **141** do under
the narrower floor-only reading of D-3. Both roots give identical numbers.

So freezing is **correct and not sufficient**. What remains to explain the report is not in this
story's scope and is named here so the next one starts from a measurement: the visibility gate — a
candidate inside the circle must also be seen, and `AI-LOS-089`'s march is the only thing keeping
most of those 235 from firing — and the absent walk-home break-off, which is why a unit that does
start a fight never goes back.

## What the handed defect location missed

Two writers were named: `noticeRadius` and `decide`. Enumerated in this tree, the movement and
acquisition writer sets are:

- **Position:** exactly one, `pkg/sim/step.go:573` (`e.X, e.Y = step[0].x, step[0].y`). There is no
  other writer of a coordinate anywhere under `pkg/sim`, and no path appends an entity to a world.
- **Attack order:** exactly one, `orderAttack`, with two callers — `decide` and the attack command
  at `pkg/sim/step.go:322`.
- **What makes a member walk:** not `decide` directly. `orderAttack` *clears* the order, and the
  move loop re-aims an attacker at its victim on the same tick (`pkg/sim/combat.go:230`). That is
  the writer the premise did not name and it is the one that produces the drift: an acquisition is
  a standing instruction to close the distance, re-established every tick for as long as the victim
  exists.
- **The clip origin** — the group centroid — is recomputed per decision, and that is **correct**.
  `AI-RADFREEZE-075` says only `grpAI+0x2c` is written and never read; `grpAI+0x28`, the cell
  centroid, is read by the arm every tick (`AI-GRPGUARD-074`). The circle follows the group. Only
  its radius is frozen.

## Where the record has to live

`aiGroup` is built per decision and stored nowhere, and `pkg/sim/group.go`'s group order is a
correlation tag inside one advance that survives nothing. Neither is a place to freeze a value on.
The entity set is fixed when a world is built — nothing appends — so a partition keyed on
`(owner, group)` over every owned entity is fixed for the world's life, which is what makes a
record a constant rather than something a tick has to maintain.
