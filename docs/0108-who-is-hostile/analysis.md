# Analysis — 0108, who is hostile

**Intensity:** spec-anchored / static. **Terrain:** brownfield in `pkg/sim` — the relation and
the blow both exist and both have behaviour to preserve.

## The question

Five stories have narrowed the milestone by exclusion. `0090` excluded the sight **predicate** by
experiment; `0107` excluded the acquisition path, the clip and the notice circle by measurement,
and left exactly two inputs on the decision that kills the driven unit: the sight **range**, and
**the relation the map authors**. This story was opened to find which of the two is wrong.

**Neither is.** Both were measured before anything was proposed, and the measurements are below.

## The relation, measured

Mission 10's own type-5 roster, read through this tree's loader against **both** installed roots
and identical on them:

| slot | name | placed | relation words k=0…4 |
|---|---|---|---|
| 1 | `Self` | 0 | 2 0 1 1 1 |
| 2 | `Villagers` | 14 | 0 2 1 1 1 |
| 3 | `Rogues` | 2 | 1 1 2 1 1 |
| 4 | `Beasts` | 16 | 1 1 1 2 1 |
| 5 | `Nocturnal` | 3 | 1 1 1 1 2 |

The world's own matrix, read back through `World.Relations()`, is that table verbatim: word k at
column k+1, narrowed to its low byte, diagonal forced to 2. **`Beasts → Villagers` is 1 and
`Villagers → Beasts` is 1** — the map asks for it, in both directions, and every pair on this map
agrees with its mirror.

So the interceptor hunting the driven unit is not a defect in what our tree believes. It is what
the file says. The only pair on mission 10 that is *not* hostile is `Self ↔ Villagers` — the party.

## The sight range, measured

The interceptor is `u32`, class 73, owner 4, group 7, standing at (48,44) with 10 health and a
scan range of **6**. That 6 is not a constant of ours: `0091` replaced the package constant with
each placement's own streamed column, and `AI-SIGHT-092` names that column's store —
`R0281` through the `Units` streamer's `scanRange` slot — as one of the six writers of the
byte the AI reads. The driven unit's 5 is the `Humans` streamer's slot 8 by the same row. Both
numbers are the game's.

Six against a separation of five is not a near miss to argue about, and lowering it is not
available: the number is a decoded column, not a choice.

## What the reading found instead

The relation has a half this tree does not model at all. `AI-DIPLO-082` enumerates **six** writers
of the matrix after construction. We have **one** — the map loader — and it runs once, at load.
Of the five that run during a mission:

- **The event flip.** A landed blow makes the two *players* mutually hostile: `AI-RETAL-056`'s
  hook, whose whole body is a turn flag, a remembered cell, and a call to `R0126`; that
  routine writes both directions, each gated on its own cell's low two bits (`AI-DIPLO-084`,
  `HERO-AGGRO-028`). It is High on the instructions and on the gate.
- **The script's own write**, action opcode 10 (`TRIG-DIPLO-019`) — one direction, an `ADD`, and
  it overrides the very lock the flip respects.
- **Two mission joins and session command `0x45`** — the multiplayer machinery.

**This tree already knows the flip exists and says so in a comment.** `relations.go` explains bit
1 as "locks the pair against being turned hostile by combat" and then says the writers "this story
does not implement"; `World.Relations()` says "the type keeps having no writer to find". Both were
true when written. The first is what made this story findable.

## What a shipped map reaches

Measured through this tree's own reader, replaying the loader's store, over every campaign map both
roots hold — **9 maps, 46 roster records, 198 ordered off-diagonal pairs, identical on en and ru**:

| low two bits | pairs |
|---|---|
| both clear — a blow flips it | **51** |
| bit 0 set — already hostile | 144 |
| bit 1 set, bit 0 clear — **locked**, a blow flips nothing | **3** |

No cell on any of them carries a bit above 1. So the flip is reachable on a quarter of the
campaign's pairs, and the lock that stops it is shipped content rather than a hypothetical.

The **script** write is not reachable. A census of every action opcode authored by every map either
root can read — the 9 campaign maps and the 16 loose `.alm` files beside the install — returns
`2:32 4:3 5:1 6:6 8:1 12:2 13:2 19:1 20:1 22:4 23:2 28:1` and the build-time constant form.
**Opcode 10 appears zero times.** That is why it is cut here and not merely deferred: implementing
it would change no shipped map's behaviour, and it belongs to G2 as a customisation seam.

## The tenth mission, predicted

The pair that kills the driven unit is `(4,2)`/`(2,4)`, already 1 in both directions, so the flip's
gate leaves both cells exactly as they are. The only flippable pair on that map is
`Self ↔ Villagers`, and the drive issues no blow across it — it drives one villager and orders the
party nowhere. **The prediction is that the tenth mission does not move**, and it is recorded as a
prediction rather than as a target: the baseline, both roots, before any edit, is

```
waypoint 1  u21 -> (56,21) r3 : was STOPPED BY THE WORLD DECIDING, short of (39,41), Chebyshev 20, after 272 ticks
outcome lost at tick 272
```

If it does move, that is a finding about something else and belongs in the evidence, not in a
contract adjusted until it passes.

## What we looked at and did not use

**The remembered attacker.** The same hook sets the victim's `order+0x58` to the attacker's cell
and clears `order+0x5a`, and `AI-GROUPSEE-068`'s phase (b) reads them: the cell is stamped into the
group's shared sight map and the memory is dropped after twenty ticks. `0107` recorded that there
was "no published writer for the remembered attacker" — **that is now wrong; `AI-RETAL-056` is the
writer.** It is cut here anyway, and not for want of evidence: it needs two per-entity fields, a
byte-form version, and the engine's notion of *a player who is a human participant*, which this
world has none of. It is the natural next story and it is where version 25 should go.

**The turn-now flag** of the same hook is cut for `0106`'s reason: its only consumer is the idle
turn, which this build does not run because the arm it lives in is `rand()`-gated.

**`AI-FILTER-001`'s invisibility term** — a candidate the diplomacy test would keep is dropped
anyway when `cand+0x144 & 0x8000` is set, unless a group member is close enough. No row read here
names a writer of that bit, so it is a question for research and nothing is worked around.

**`AI-SIGHT-092`'s second writer**, `L00261`, forces a sight of 5 behind `CMP word ptr [EDI +
0xe],0x18`. What `actor+0xe` holds is published in no row read here. Named, not chased.
