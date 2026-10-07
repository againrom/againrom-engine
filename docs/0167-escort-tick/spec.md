# 0167 — escort tick

## Terms

**Escort** — a living entity whose actor state is defend (8) or follow (0x11).
Such an entity carries an escort target and an escort range; nothing else in the
world does. **Escorted unit** — the entity the escort target names. **Range** —
the escort's own stored stop distance. **Distance** — the Chebyshev distance
between two entities' cells. **Actor pass** — the per-entity dispatch that runs
once per script phase, after the group decision. **Close order** — a destination
written at the escorted unit's cell. **Standing acquisition** — the target
selection an unordered unit runs. **Cover block** — the square of cells the
defender scans on the escorted unit's behalf.

## Problem

An escort stands still. 0166 landed the two states, the escort target and the
range, and the script sub-commands that write them, but the actor pass has no
case for either state, so a scripted Defend or Follow reaches no arm and no
member ever moves. The per-tick behaviour behind both states is decoded.

The behaviour is one contract with three parts. An escort out of range walks to
its escorted unit. An escort within range fights: a defender on the escorted
unit's behalf, a follower on its own. An escort standing on top of the unit it
escorts steps away from it.

## Functional requirements

**FR-1 — the two escort states reach an arm.** The actor pass dispatches actor
state 8 to the defend arm and 0x11 to the follow arm. Every other state's
treatment is unchanged, including acquire (0xc), which still reaches no arm.

**FR-2 — an arm resolves its escorted unit first.** The escort target is looked
up in the world's entity slice. An escort whose target names no entity this world
holds is left in every field for the pass: no destination, no victim, no facing
and no state is written. This is the only case in which an arm ends without
deciding anything.

**FR-3 — the range each arm tests against.** The stop distance is the escort's
own stored range, or, where that range is 0, the escort's own scan range. The
distance tested against it is the Chebyshev distance between the escort's cell
and the escorted unit's cell.

**FR-4 — out of range, an escort closes on its escorted unit.** Distance greater
than the stop distance: the escort's victim, its attack cycle, its destination,
its stall count and its stored route are dropped together, and it is given the
escorted unit's present cell as its destination. The arm ends there. Neither the
cover block, nor the standing acquisition, nor the step-away runs on a pass that
closes.

**FR-5 — within range, a defender covers its escorted unit.** Distance at most
the stop distance: the defender scores every entity that stands within 5 cells
Chebyshev of the ESCORTED UNIT's cell and that the ESCORTED UNIT is hostile to.
The polarity is the escorted unit's, not the defender's: a defender engages what
threatens the unit it protects, whether or not that thing is its own enemy. An
off-map entity is in no block. Living candidates are taken alone; where the block
holds no living candidate, its dead are taken instead.

**FR-6 — which of the cover block a defender takes.** The candidate nearest the
DEFENDER, measured in cells. Candidates of the air domain are preferred: a
defender takes the nearest air candidate where the block holds one, and the
nearest candidate of any domain otherwise. Ties fall to the lower entity id. The
defender engages the pick with an attack order, which is this package's one
representation of a pursuit.

**FR-7 — an empty cover block hands the defender to the standing acquisition.**
Where the block holds no candidate at all, the defender runs FR-9's acquisition
in place of the engagement.

**FR-8 — a defender that is now fighting does nothing further this pass.** After
the engagement attempt, a defender holding a victim ends its pass. A defender
holding none falls to the step-away test.

**FR-9 — within range and not crowded, a follower re-acquires.** Distance at most
the stop distance and at least 2: the follower runs the standing acquisition —
everything it can see, filtered to the entities it is hostile to, scored by the
reach-vetoing scorer, the cheapest taken and engaged with an attack order. A
follower that scores nothing has its destination, stall count and stored route
dropped: the close order it may have been walking under is over and it does not
walk on. A follower's own pass then ends, whether or not it engaged.

**FR-10 — an escort crowding its escorted unit steps away.** Distance below 2,
for both arms: the escort's victim, attack cycle, destination, stall count and
stored route are dropped, and it is sent to the cell reached by travelling the
stop distance from the ESCORTED UNIT's cell along the line from that cell through
the escort's own. The line is computed on the dominant axis: the escort moves the
stop distance in whole cells along whichever axis it is further out on, and the
other axis takes the proportional part of that move. The destination is clamped
to the playable rectangle, which is `[8, width-9]` by `[8, height-9]`.

**FR-11 — what neither arm writes.** No arm writes an actor state, an escort
target, an escort range, a health, a facing, a group's stored order, a patrol
ring or a transit. An escort left with nothing to do keeps its cell.

**FR-12 — the arms are integer and deterministic.** No arm reads a clock or a
random source, and none computes in floating point. Two worlds equal in every
hashed field advance to worlds equal in every hashed field.

## Acceptance criteria

**AC-1** A world holding a follower 6 cells from its escorted unit, range 3, is
stepped one script phase: the follower holds the escorted unit's cell as its
destination. Stepped far enough, it stands within 3 cells of that unit and has
stopped.

**AC-2** The same world with the escorted unit walking: after the follower has
closed, the escorted unit is walked away until the gap passes 3, and the follower
is aimed at it again on the next actor pass.

**AC-3** A defender within range of its escorted unit, with a hostile of the
ESCORTED UNIT 4 cells from that unit and 9 cells from the defender, engages that
hostile. The same world with the hostile 6 cells from the escorted unit leaves
the defender with no victim from the cover block.

**AC-4** A defender whose escorted unit is hostile to a candidate the DEFENDER is
not hostile to still engages it. A defender hostile to a candidate its escorted
unit is not hostile to does not engage it from the cover block.

**AC-5** A defender whose cover block holds an air candidate at distance 8 and a
ground candidate at distance 3 takes the air candidate.

**AC-6** A defender whose cover block is empty and which can see a hostile within
its own reach engages that hostile. One that can see nothing engages nothing.

**AC-7** A defender that engages does not step away on the same pass, with the
escorted unit on the adjacent cell.

**AC-8** An escort standing on the cell adjacent to its escorted unit, range 3, is
given a destination 3 cells from that unit, on the far side, on both arms.

**AC-9** An escort of an entity the world does not hold is left in every field
over a whole tick: cell, destination, victim, stall count and route unchanged.

**AC-10** An escort whose stored range is 0 tests against its own scan range.

**AC-11** A world advanced with escorts marshals, decodes and hashes identically
before and after the arms run, and `formatVersion` is unchanged.

**AC-12** A guarding, patrolling or acquiring entity is unaffected: the same world
without escorts advances byte for byte as it did before this story.

## Non-functional / performance

**P-1** The cover block is a sweep of the entity slice per defender per actor
pass, and the standing acquisition is one sight stamp per escort per actor pass.
Both are bounded by the entity count and neither allocates per candidate.

**P-2** No arm allocates a route; all routing stays in the move loop.

## Out of scope

The heal fork, the idle turn, an arm for actor state 0xc, and any change to the
group decision, the byte form or the script setters.
