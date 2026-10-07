# Analysis — what the player's roster slot is, and what standing outside it cost

## What was not known

Two observations were reported from the shipped build: **enemies never attack the player**, and, on
mission 10, **two clubmen kill a witch**. Neither is a verdict. What had to be measured is which of
the two is a defect of this tree and which is the shipped map behaving as its own file says.

## The chain that was read, link by link

The engagement layer indexes everything by an entity's **owner slot**, and the party had none.

| Read | What it says |
|---|---|
| `pkg/mapload/start.go`, the party composite literal | names seventeen fields and **neither `Owner` nor `Group`**, so both take the zero value |
| its `ScanRange` field comment | states it as a known property: *"An entity of roster slot 0 belongs to no group and takes no engagement decision"* |
| `pkg/sim/relations.go`, `relationIndex` | returns `false` when `from == 0` or `to == 0` |
| `pkg/sim/relations.go`, `Byte` | returns 0 for a cell the matrix does not hold; `Hostile` is bit 0 of it |
| `pkg/sim/world.go`, `hostileTo` | is exactly `relations.Hostile(me.Owner, him.Owner)` |
| `pkg/sim/engage.go`, `candidates` | drops every candidate failing `hostileTo` |
| `pkg/sim/engage.go`, `aiGroups` | skips every entity whose `Owner == 0`, so it is in no group and takes no decision |
| `pkg/mapload/fromalm.go`, `relationFrom` | the i-th roster record is slot `i+1`, element k lands at column `k+1`; **column 0 is never written** |

So `Hostile(x, 0)` and `Hostile(0, x)` are false for every `x`, on every map. The party was neither
acquirable nor acquiring — not rarely, but **never, by construction**.

And the tree already **holds the right answer in one of the two files**: `pkg/sim/engage.go` defines
`selfSlot = 1` with the whole derivation on it, and `pkg/mapload/start.go` then creates every party
member at 0. The contradiction is internal, not a research question. The ledger below confirms which
of the two is right rather than being the sole source of it.

## Was this disclosed?

`0086` specified the boundary deliberately: FR-5 (*an entity whose owner slot is outside the matrix
is hostile to nothing and nothing is hostile to it*), FR-8 (*an entity whose owner slot is 0 belongs
to no group*) and AC-14. The boundary rule is right — slot 0 names no roster entry.

What `0086` never says is that **the party stands inside that boundary**. Its `spec.md` does not
contain the word *party*; its `provenance.md` and `tasks.md` mention one in passing, neither about
ownership. So this is a defect, not a disclosed cut. The rule and the population it silently
captured were written in two different packages, and nothing joined them.

## Which slot the party should carry

Measured against the pinned ledger, not asserted: `ALM-OWN-039` (High, an owner word is a 1-based
slot in the type-5 array, the id-word reading refuted on 713 + 2238 shipped records);
`AI-DIPLO-005` (High, matrix index = type-5 slot + 1, column 0 never written); `TRIG-EMPTY-018`
(*"roster slot 1, the player's"*); `MISSION-START-001` (counts what roster slot 1 owns over the 28
campaign maps of both roots); `PARTY-PERSIST-014` (Medium, the campaign arm reuses the surviving
`Player` for roster slot 1); `ALM-GRP-020` (the first roster name is `Self`). Six rows converge on 1
and none contests it.

## Mission 10, measured

`10.alm` is **byte-identical on both roots** (md5 `2d983ccbf249c5336ebb7ccc41fc405c`, 67554 bytes,
`restool cat scenario.res 10.alm`), so one reading covers EN and RU.

Its roster is five records, and its matrix (low byte of each word, diagonal forced to 2) is:

```
slot name          units   [1]  [2]  [3]  [4]  [5]
  1  "Self"           0      2    0    1    1    1
  2  "Villagers"     14      0    2    1    1    1
  3  "Rogues"         2      1    1    2    1    1
  4  "Beasts"        16      1    1    1    2    1
  5  "Nocturnal"      3      1    1    1    1    2
```

The two clubmen are placements 0 and 1: class id **10**, `units.reg` `DescText = "Human ClubMan"`,
owner **3 `Rogues`**. The witch is placement 2: the NPC arm, `scenario/npc.reg` `npc51`,
`Flags = "Human,Female,Mage,Face"` — owner **2 `Villagers`**, script unit id 21.

`Rogues -> Villagers` carries **1**. Bit 0 set. So does the mirror. The map authors that fight, and
`AI-DIPLO-005` establishes that shipped content uses the asymmetry deliberately. **The second
observation is the map's own behaviour and nothing is owed to it.**

## Who actually kills the witch — probed on mission 10 EN before specifying

Four runs of the same world, differing only in the party's slot and in what is ordered to move.

| Run | Slot | What moves | Result |
|---|---|---|---|
| A | 0 | nothing | 400 ticks, the clubmen acquire **nothing**, the witch is untouched at 45 health |
| B | 0 | the witch, at the milestone's own waypoint (56,21) | entity **13** — owner 4 `Beasts`, class 73 `Bee`, standing at (47,45) — acquires her at tick 103; she falls at 250 and the mission is lost at 272. **The clubmen never acquire her.** |
| C | 0 | the party, walked to (25,55) beside the clubmen and then east to (40,50) | 400 ticks, **nothing acquires the party** and the clubmen still acquire nothing |
| D | 1 | the same walk as C | both clubmen acquire the **party** at tick 135; health 100 -> 98; one loses it at 189 as the party withdraws, and at 391, having been dragged from (32,62) to (31,53), acquires the **witch** |

Three things follow, and none of them was guessable from the source.

**The clubmen are not the witch's killers in this build.** A `Bee` is, and only when she is driven
past it. The mission-10 loss is a driven quest unit walking into a guarding group's circle.

**Slot 0 was the whole reason the player could be walked into a hostile camp untouched** — run C
against run D is the same drive with one field changed.

**Our notice circle moves with the group and the original's does not.** Run D's clubman reaches the
witch only because chasing the player carried it nine cells north; `0086` DD-3 discloses that the
law freezes the radius at the geometry the group had when guard was issued
(`AI-RADFREEZE-075`), and this tree recomputes it every tick because it has neither a group record
to freeze on nor a guard setter to freeze at. So a group here can be led onto a target the original
would never have noticed. That is a **standing disclosed divergence, not this story's** — but it is
the mechanism behind a player-side report, and it now has a measurement.

`TestTheTenthMissionIsDrivenToAWin` reports **`lost at tick 272`** on both roots, and the same
**after**. The slot was a necessary condition for the player to be fought at all; it does not decide
that mission. The loss belongs to check 12, the map's protect-`u21` objective, and `u21` is the
witch — so run B is that loss with its killer named.

## Not chased

The frozen notice radius, the party's `Group` word, patrol, roam and the six missing group orders.
