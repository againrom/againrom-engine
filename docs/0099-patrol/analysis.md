# Analysis — what Patrol is, and what it is not

## What was asked

Two of mission 10's three tick-6 instants are Patrol. The lane was opened on the hypothesis that
running them would save the mission's protected unit — that the commanded groups walk her captors
away from her — with the instruction to **measure it rather than believe it**, and not to tune
anything toward a win.

## What was measured, before any code was written

All of it on `wt-0099` at base `36ca057`, assets `gameversions/ru`. `10.alm` is byte-identical
between the two roots (`md5 2d983ccbf249c5336ebb7ccc41fc405c`), so the map facts below are made on
both.

**The two Patrol nodes.** `almtool script` reports `instant arm 6 sub-command 14: 2 node(s) not
implemented`, and the compiled instants are

| slot | group | sub-command | (x, y) |
|---|---|---|---|
| 20 | 18 | 14 | (53, 19) |
| 21 | 17 | 14 | (54, 13) |

**Who is in those groups.** Group 18 is one entity, map id 57, owner 2 (`Villagers`), class 1,
standing at **(64, 21)**. Group 17 is one entity, map id 58, same owner and class, standing at
**(67, 12)**. Both are in the map's north-east corner, among the other `Villagers` placements.

**The protected unit** is map id 21 — owner 2, group 2, class 1, 45 HP — placed at **(36, 51)**,
about thirty cells south-west of both patrollers. **No script node on this map names her group.**

**What kills her.** Stepping the mission with no commands at all, she takes no damage in 400 ticks
and the mission stays undecided. Under the drive the milestone uses, she is intercepted at (43, 46)
on tick 145 by **one entity**: id 13, owner 4 (`Beasts`), group 7, class 73, 10 HP, placed at
(48, 44). It beats her from 45 HP to -2 over 105 ticks while she walks, and it is the only attacker
holding her at any tick. Her group is not commanded by the script; **group 7 is not commanded by
the script either.** She never strikes back because she is executing a player order, which the
engagement decision leaves alone by its own FR-1.

## The hypothesis is refuted

Patrol does not touch the unit that dies, the unit that kills her, or any group within twenty cells
of either. The two commanded groups are **her own faction**, single units, in the opposite corner
of the map. There is no path by which running these two nodes changes tick 145, tick 262 or tick
271.

**So this story is not the milestone's fix and does not claim to be.** What it fixes is that two
authored patrol routes are not walked, here and on seven other campaign maps. The mission-10
outcome is recorded before and after in `verification.md` as a measurement; `spec.md` AC-6 states
explicitly that it is not an acceptance condition.

The finding worth carrying forward is the one about the killer: **a single 10-HP beast decides the
first mission of the campaign**, and it does so against a unit that cannot fight back while it is
under a player order. That is a question about the release rule and the escort, not about Patrol.

## What the pin says, and what it leaves to us

The command is fully decoded and the decode is unusually complete — two routines read end to end,
every store cited, and a 38-map census of the node itself. Three things it settles that shaped the
contract:

- **The command clears the group order.** `grpAI+0x20 = 0` is not incidental; it is the write that
  decides whether any of the rest executes. This build already has that byte and already does
  nothing for a group under 0, so the arm's whole group-layer effect is one assignment — and the
  work is making 0 *storable*, which 0096 FR-19 refuses.
- **A shipped patrol is not an authored path.** It is one commanded point and wherever the creature
  happened to be standing. That is why the ring is two nodes and why the head has to be captured at
  command time.
- **The state's default is `0xb`, not zero,** and it is written by the actor initialiser rather than
  by the map's spawner. So the field is not an input, which is FR-11.

Two things it does **not** settle, both taken as ours and declared in `spec.md`:

- **When the actor machine runs relative to the group decision.** Nothing read orders them. D-4
  takes the same answer the tree already took for script-versus-decision.
- **Whether a patroller's guard call has an equivalent here.** It does not: this build has no
  per-actor guard. That makes FR-9 a disclosed divergence rather than a modelling choice — a shipped
  patroller fights and advances only when it is not fighting; this one always advances.

## The clause most likely to be missed, and why it is not modelled

`AI-PATROL-018`'s tail: the walker sets a latch on every advance, and the next entry consumes it to
move the guard post to where the actor now stands. That is the mechanism by which the guard leash
never drags a patroller back to its post — a real and non-obvious fact, and exactly the kind of
thing a build silently gets wrong.

It is **not modelled here, deliberately** (D-3). The guard post and the latch are written by the
command and the arm and read by **nothing but the per-actor guard**, which is out of scope. Carrying
them would put two fields into the entity record, the byte form and the digest with no reader, and
the story that adds the guard arm must add the consume-on-entry anyway. The decision is recorded so
that the story which does add it knows the latch is owed rather than absent.
