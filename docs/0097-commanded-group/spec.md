# Spec — a unit executing a player's order is not decided over

**Intensity:** spec-anchored / static. **Terrain:** brownfield — the engagement decision ships, the
move loop ships, and a defect reported from play is the reason this story exists.

## Terms

- **A player's order** is a command carried into a tick from outside the world: the move order for
  one unit and the move order for a group of them. Both are already in this build and neither
  changes here.
- **A destination** is the cell a unit has been told to walk to, together with the flag that says it
  holds one. It is a field this build already has, and its lifetime is already defined: an order
  writes it, arrival ends it, a route that cannot be built ends it, falling ends it, and a later
  order replaces it.
- **A victim** is the unit an attacker has been given, distinct from a destination. An entity may
  hold one, the other, both, or neither.
- **Under command** is the state this contract introduces, defined in FR-1. It is a *derived*
  property of an entity, not a new field.
- **The engagement decision** is the pass that runs once per full tick, partitions the world's
  living owned entities into the groups the map placed them in, and for each group builds the
  candidate list from its members' sight, scores every candidate for every member, and gives each
  member the cheapest one it will take.
- **A decider** is an entity the engagement decision runs *for*: one whose sight enters a candidate
  list and which may be given a victim. **A candidate** is an entity that may be given *to* a
  decider. The two roles are independent.
- **Stance** is which of the two group behaviours a group is under. This build derives it from the
  group's owner slot, and this story does not change that derivation.
- **Reach** is how far a blow carries. It is 1 in this build.
- **Both roots** are the two lawful installs, English and Russian. A claim made on one is not made.

## Why

The engagement decision ends by giving a member a victim, and giving a member a victim ends whatever
walk it was on. That is correct for a unit the decision is entitled to decide over. It is applied
today to every living owned entity, including one that is in the middle of executing an order the
player gave it — so the decision destroys the order, and nothing re-issues it.

Measured on mission 10, both roots: a party member ordered to a cell ten steps away walks eight of
them, meets a hostile standing one cell off its path, is given that hostile as a victim, loses its
destination, kills what it was given, and then stands where it stopped for the rest of the mission.
From the player's seat the group takes a step or two and stops with the whole path reachable.

The distinction the build is missing is not between owners. It is between a group a **map** authored
and a group a **command** created. A command allocates its own group and puts it under a group order
whose whole behaviour is *walk, and re-issue the walk* — no candidate list, no scoring, no engage.
A unit under that order is not choosing not to fight; it is never asked.

## Scope

**In scope.** Which entities the engagement decision is a decider for; when a unit stops being one;
and what happens on the tick it stops.

**Out of scope.** The per-actor state machine a group order of 0 needs. Patrol and Follow. Roam. The
notice radius and the group record that would freeze it. Any retaliation policy. What a player's
order does to a unit's facing, rate or formation. The mission script's own group command.

## The contract

### The state

**FR-1** A unit is **under command** exactly while it holds a destination and holds no victim. No
other condition enters, and the state is read from those two fields rather than stored beside them:
no entity field is added, removed or widened by this story.

**FR-2** Exactly two writers **originate** that state, and both are player order arms this build
already has — the move order for one unit and the move order for a group. Each of them already ends
whatever fight the unit was in, which is what makes FR-1 exact. A third writer **restores** it: the
decoder, which reproduces whatever the byte form carried, so a world encoded mid-order comes back
under command with no field of its own to carry it. A fourth gives a destination only to a unit that
already holds a victim — the approach that re-aims an attacker at where its victim now stands — and
so can never originate it. **Any future writer that gives a unit a destination without a victim
inherits this state**, and a walk that the world itself orders —
a unit sent home, a unit sent anywhere by the mission script — must therefore be built against a
distinction this contract does not yet draw. That is a named seam, not an omission: FR-1 is
falsifiable by AC-11 and a new writer breaks it loudly.

Story1164 keeps the decoder's restorer role in `World.unmarshalBinary`.
`UnmarshalBinary` delegates to it, and legacy area migration uses the same
decoder before changing cell ownership. Both read HasTarget from the same
encoded 0/1 byte. The area compatibility argument neither originates an order
nor changes destination/victim restoration. The architecture pin names the
helper that now contains the write; its exact-writer check remains active.

**FR-2 extension (1089).** `withdrawFromAny` replaces `withdrawFrom` as the
shared destination writer. It remains a genuine originator: it clears the
victim and writes an ordinary move. Automatic withdrawal calls it from the
post-dispatch threshold tail; explicit Retreat calls it from an actor arm in
an order-none group, which the group decision skips. `withdrawFrom` retains
only the positive-HP gate. FR-1's derived state and native representation do
not change. The architecture writer pin records this replacement.

**FR-2 extension (Hold Position hotfix).** `standDown` gives a member its own
cell as the destination only while the member holds a loaded attack cycle, so
the member holds a victim when the write happens, as the approach does, and
the write cannot originate FR-1's state. The attack pass that ends the cycle,
or ends it early because the victim is gone, drops the victim and then that
destination in the same pass. A member holding neither victim nor destination
is not written. The architecture writer pin records it.

**FR-2 extension (loaded-cycle order routes).** `retainCycleForState` is `standDown`'s
write for the writers of a state and no destination (Defend, Patrol, the script group
stops). It gives the member its own cell as the destination only while the member holds a
loaded attack cycle and no destination, so the member holds a victim when the write happens
and the write cannot originate FR-1's state; the attack pass that ends the cycle drops the
victim and then that destination in the same pass. The architecture writer pin records it.

**FR-2 extension (1248).** `dispatchRetreatPending` originates FR-1's state:
it clears the victim before writing the pending Retreat cell as an ordinary
destination. Retreat policy stores pending coordinates without writing
`HasTarget`; dispatch waits for an entered-zero progress invocation and the
executor activity gate. Restoring the pending continuation does not originate
a destination. The architecture writer pin names only the dispatch helper.

### What the state does

**FR-3** A unit under command is **not a decider**. The engagement decision does not score it, does
not give it a victim, and does not end its order. It is skipped where the decision decides *who*
takes part, not where it decides what any group does — no group's behaviour changes, and no arm of
the decision is edited.

**FR-4** A unit under command remains a **candidate** for every other group, on exactly the terms it
had before: it can be seen, it can be chosen, and it can be struck. FR-3 and FR-4 together are the
whole of the change.

**FR-5** A unit under command **does not contribute its sight** to the group the map placed it in,
because for as long as the command holds it is in the group the command built. A group that can see
a hostile only through a member that is under orders therefore has no candidate list. This is a
consequence of FR-3 rather than a rule of its own, and it is disclosed here because it is the one
effect of this story on a unit other than the commanded one.

**FR-6** Being struck does not end the state and does not produce a victim. A unit walking under
orders past a hostile keeps walking, and one struck from behind keeps walking. This build has no
retaliation arm and this story does not add one.

### When it ends, and what happens then

**FR-7** The state ends exactly when the destination ends, and the destination's lifetime is
unchanged by this story. So it ends on arrival; on a route that cannot be built; on the give-up the
move loop already owns; on falling; and on any later order, of either kind, that replaces the
destination or supplies a victim. **Nothing else ends it**, and in particular no elapsed time, no
distance walked and no change in what the unit can see.

**FR-8** On the tick the state ends, the unit is a decider again, under the stance its owner slot
selects and with nothing carried over from the walk. **Disclosed divergence:** the original hands an
arrived unit to a per-unit acquisition rather than back to its group's stance. Both admit only a
candidate within reach, so the two agree on *whether* a unit takes anything; they may differ on
*which*, when two or more candidates stand within reach of one unit at the same moment.

### What does not move

**FR-9** A unit that is not under command is decided over exactly as it is today — the same
partition, the same candidate list, the same scorer, the same order. A world in which no unit holds
a destination without a victim behaves identically to the world before this story, entity field for
entity field, tick for tick.

**FR-10** The canonical byte form is **unchanged**: same version byte, same field set, same
encoding. Any world's digest is the digest it had before this story. A byte form that changed would
mean this story had stored the state instead of deriving it, so this is the contract's own check on
FR-1 rather than a note about compatibility.

### What this story takes back

**FR-11** A previous story disclosed, as the cost of seating the player inside the diplomacy matrix,
that *a single move order issued while a hostile stands within reach is overridden on the next
decision, and a re-issued order is needed to cover the same ground*. **That is the defect this story
exists to remove**, and it was pinned as a measurement rather than left in prose — so the
measurement is superseded, not merely outdated, and it is rewritten here rather than deleted. What
replaces it is the stronger statement it was already comparing against: a player's unit under a
**single** order now covers the same ground as the same drive by a unit no group can see, with no
re-issue at all. The rewrite is part of this story's own diff and carries the reason in its text; no
threshold, radius or predicate is moved to achieve it.

## Acceptance criteria

| | Given | When | Then |
|---|---|---|---|
| **AC-1** | a unit holding a destination and no victim, and a hostile standing within reach of it | a decision is taken | it keeps its destination, is given no victim, and continues to advance |
| **AC-2** | the same unit and hostile | the same | the **hostile** is given that unit as its victim, if its own group would have taken it |
| **AC-3** | a unit holding neither a destination nor a victim, and a hostile within reach | a decision is taken | it is given that hostile — unchanged from today |
| **AC-4** | a unit holding a victim and a destination written by the approach | a decision is taken | it is treated as a decider, not as under command |
| **AC-5** | a unit under command that reaches its destination with a hostile within reach | the tick it arrives, and the next decision | the destination ends on arrival and the unit is given the hostile |
| **AC-6** | a unit under command | it is struck | it holds its destination, is given no victim, and keeps advancing |
| **AC-7** | a group of units given one group move order, one of which can see a hostile the others cannot | a decision is taken | none of them is given a victim, and no other group receives a candidate list built from that member's sight |
| **AC-8** | a unit under command that is ordered again to a different cell | the new order is applied | it is still under command, now toward the new cell |
| **AC-9** | a unit under command that is given an attack command | the command is applied | it is no longer under command and is given that victim |
| **AC-10** | a unit under command that falls | it falls | it is under command no longer, and holds neither destination nor victim |
| **AC-11** | the simulation package | it is searched for writers that give an entity a destination | exactly the four FR-2 names are found — two player order arms, the decoder, the approach — and no other; a fifth appearing, or one of the four vanishing, fails and says which |
| **AC-12** | any world | it is encoded, decoded and hashed | the round trip is byte-identical, the version byte is the one that already shipped, and the digest equals the digest the same world had before this story |
| **AC-13** | a world in which no entity holds a destination without a victim | it is stepped for many ticks | every entity's every field matches the same world stepped on the tree before this story |
| **AC-15** | a unit at the player's slot given **one** move order past a hostile that closes on it, and the same drive by a unit no group can see | both are stepped the same number of ticks | the two end on the same cell, and neither is the cell the hostile holds it at — the re-issue the previous story needed is no longer needed |

## Error cases

**AC-14** No new failure is introduced: the state is derived from two fields that are already
constrained by the byte form, which refuses a stall count or a stored route on an entity with no
destination and refuses any order at all on an entity that is not alive. A world that could not be
built before cannot be built now, and every refusal keeps its present message.

## Success conditions

**SC-1** The reported defect, on **both roots**, before and after: a party member driven to a cell
its build stops short of today. The before figure is `STOPPED SHORT of (24,57), Chebyshev 2, after
400 ticks`. The after figure is recorded whatever it is.

**SC-2** A unit that is **not** under command and stands beside a hostile still fights it, measured
on a real mission rather than only in a fixture — the fix must not turn the party into pacifists,
and the population that keeps fighting must be named exactly.

**SC-3** The nine authored placements at the player's own roster slot — three on map 41, four on 71,
one on 150, one on 151, on both roots — still take Stand Ground. That none of them can be under
command is shown from the maps, not argued.

**SC-4** Mission 10's outcome and tick are recorded on both roots, before and after. If it moves,
the tick and the arm are reported. **No test expectation, threshold or predicate is edited to alter
it** — falsifiable from the story's own diff.
