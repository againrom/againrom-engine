# 0164 — Map presence: taking a unit off the map and putting it back

## Subject

The mission script can remove a unit from the map and put it back. Five instant opcodes do it:
16 (remove one unit), 17 (return one unit), 18 (remove one unit and place a second at its cell),
32 (remove every member of a group) and 33 (return every member of a group). This build compiles
all five today and runs none of them.

Across the 28 campaign maps of both preserved roots, 81 authored script nodes carry these five
opcodes: 29 of opcode 16, 27 of 17, 3 of 18, 11 of 32 and 11 of 33. 56 of the 81 are reachable
from a trigger. Every node this build cannot run is a node the original executes and this one
skips.

## Contract

### FR-1 — An entity carries an off-map bit

`Entity.OffMap` is one boolean. It is state and not position: an off-map entity keeps its
coordinates, its owner, its group, its command group, its health, its container, its equipment,
its order and its attack target exactly as it held them.

Every value is legal. The constructor takes no clause for it and the decoder refuses no value:
there is nothing to fold and nothing to reject.

### FR-2 — Instant 16 takes the named unit off the map

It resolves the node's first unit reference through the world by id. A node naming no unit, or
naming an id this world does not hold, changes nothing.

It is idempotent: a unit already off the map is left as it stands and no state is written.

It writes `OffMap = true` and nothing else. Group membership, owner, position, container,
equipment, health, order and attack target are all untouched.

### FR-3 — An off-map entity is not on the map

One rule, read at every point that asks whether an entity is present:

1. it occupies no cell — nothing contends with it and it contends with nothing;
2. it is not a candidate of any acquisition walk;
3. its group does not decide for it, and it stamps no sight;
4. the move loop does not advance it;
5. it neither strikes nor is struck: an attacker holding an off-map victim loses that victim,
   exactly as it loses a dead one;
6. it is not drawn, and so cannot be selected or clicked.

### FR-4 — A group's count stays whole

The script's group-count check (check opcode 1), the group hand-over (instant 22) and the group
order (instant 6) read an off-map member exactly as an on-map one. A group whose members are all
off the map still answers its full count.

This is the one place where the off-map bit is deliberately not read. Membership is not presence.

### FR-5 — Instant 17 returns the named unit to its retained cell

The cell is the one the unit still carries: nothing in the node authors a coordinate. The search
is bounded and can fail:

1. the retained cell, attempted twice;
2. six cells drawn at `(x - 1 + r0, y - 1 + r1)` with `r0`, `r1` each uniform in `[0, 3]`;
3. the 3x3 square `[x-1, x+1] x [y-1, y+1]`, scanned exhaustively.

The first cell that the unit fits on ends the search. Fitting is this build's own placement test:
the cell is in bounds, its terrain is open to the unit's own movement domain, and no other
counted entity of that domain's layer stands on it.

On success the unit's coordinates become the found cell and `OffMap` becomes false. On failure
nothing at all changes: the unit stays off the map, at the cell it retained, and there is no
retry.

A node naming no unit, an id this world does not hold, and a unit already on the map each change
nothing.

### FR-6 — Instant 18 removes the first unit and places the second at its cell

It takes the first unit off the map by FR-2's arm. It then places the second unit at the first
unit's retained cell through FR-5's search **without the two exact-cell attempts** — the six
random attempts and then the exhaustive 3x3.

The second unit is placed whether or not it was off the map. On success it stands at the found
cell with `OffMap` false. On failure nothing about the second unit changes, and the first unit
stays off the map: the removal is not undone.

A node naming fewer than two units runs neither half. The two references naming one entity is
refused whole, on instant 28's own ground.

### FR-7 — Instants 32 and 33 are 16 and 17 per group member

Each resolves the node's group reference and applies FR-2's arm (32) or FR-5's arm (33) to every
entity carrying that group, in ascending entity id. Neither tests the member first. Neither
changes group membership, so a group emptied of map presence still has all its members.

A node naming no group changes nothing. A group id no entity carries changes nothing, which is
not a failure.

Membership is read on the map's own group word, which is what every other script arm over a group
already reads.

### FR-8 — The off-map bit is canonical

It is carried by the byte form and it enters the digest. `formatVersion` becomes 48 and the
entity record grows by one byte, from 222 to 223. A world with a unit off the map survives its
own save and load with that unit still off the map.

### FR-9 — The five opcodes leave the unsupported set

`scriptInstantSupported` answers true for opcodes 16, 17, 18, 32 and 33, so the compile-time
report stops naming them and the campaign census of unrunnable nodes falls by 81 per root.

## Acceptance

- **AC-1** — Instant 16 on an on-map unit sets the bit and changes no other field of that entity.
- **AC-2** — Instant 16 on an already off-map unit leaves the world digest unchanged.
- **AC-3** — An off-map unit occupies no cell: another unit routes onto the cell it stands on.
- **AC-4** — An off-map hostile is not acquired: a group that would otherwise attack it acquires
  nothing.
- **AC-5** — An attacker holding a victim that is taken off the map has no attack target on the
  next tick.
- **AC-6** — An off-map unit is not advanced by the move loop and holds its coordinates.
- **AC-7** — A group whose only member is off the map still answers 1 to check opcode 1.
- **AC-8** — Instant 17 on a unit whose retained cell is free returns it to exactly that cell.
- **AC-9** — Instant 17 on a unit whose retained cell and whole 3x3 neighbourhood are occupied
  changes nothing: the unit stays off the map and the digest is unchanged.
- **AC-10** — Instant 17 on a unit whose retained cell is occupied but whose neighbourhood is not
  returns it to a cell inside `[x-1, x+2] x [y-1, y+2]`.
- **AC-11** — Instant 18 takes the first unit off the map and puts the second within one cell of
  the first's retained cell.
- **AC-12** — Instant 18 naming one unit runs neither half; naming the same unit twice changes
  nothing.
- **AC-13** — Instant 32 takes every member of the named group off the map, and check 1 still
  answers the group's full count afterwards.
- **AC-14** — Instant 33 returns every member of the named group.
- **AC-15** — A world with an off-map unit round-trips through the byte form with the bit intact,
  and its digest differs from the same world with the unit on the map.
- **AC-16** — A version-46 buffer is refused.
- **AC-17** — The compiled script report names none of the five opcodes as unsupported, and a
  trigger naming one of them is not marked inert on their account.
- **AC-18** — The off-map gate is witnessed by reverting it: deleting the `OffMap` test from the
  occupancy predicate turns AC-3 red.

## Properties

- **P-1** — Which entities an arm writes does not depend on the order the world holds them in.
  Every arm resolves by id or scans the whole slice.
- **P-2** — The placement search draws only from the world's own generator. It never reads a
  clock, and the number of draws it makes is a function of the world's state alone.
- **P-3** — Every arm here is a no-op on a world it cannot resolve its references in. No arm
  writes a partial result.

## Authored, and disclosed

- **SC-1** — **Whether the original saves the off-map bit is unknown.** EXP-0169 read the arms and
  not the save writer. This build serializes it, because a save taken with a unit off the map and
  restored with it back on the map would put a unit the mission removed in front of the player.
  The claim this build makes is about **this build's own save format**, not about the original's.
- **SC-2** — **The notification packets are not reproduced.** Instant 16 emits two and instant 17
  emits three. This package has no packet layer at all; what those packets announce is the state
  change this build makes directly.
- **SC-3** — **The failure log lines are not printed.** `"Unit can't return to map - no free
  place"` and `"Unit can't enter map - no free place"` are the original's; this package writes no
  log. A failed placement is a silent no-op here.
- **SC-4** — **A failed return does not move the retained cell.** The original's placement helper
  writes the position object at each attempt, so a failed search leaves the unit's cell at the
  last attempt. This build commits no cell until an attempt succeeds. The difference is
  unobservable on the shipped corpus: every trigger naming one of these nodes carries `once = 1`
  (`TRIG-INSTCENSUS-046`), so no shipped map searches from a drifted cell.
- **SC-5** — **The random attempt window is taken from the claim's arithmetic, not from its
  summary.** `TRIG-RETURN-042` quotes the attempt as `(x - r/2 + rand(r), y - r/2 + rand(r))`,
  which at `r = 3` spans `[x-1, x+2]`, and separately summarises the six attempts as falling
  "inside the 3 x 3 square". The two disagree by one cell on each far edge. This build follows the
  quoted expression. The exhaustive stage is 3x3 in both readings and is implemented as 3x3.
- **SC-6** — **The exhaustive scan's axis order is authored.** The claim says the square is
  scanned "in order" and does not say which axis is outer. This build scans x outer, y inner.
- **SC-7** — **Instant 18's second unit is the binder's second UNIT reference, not its second
  reference of any kind.** `TRIG-MAPGROUP-043` reads the original's slot as the second reference
  of any kind. This build's binder fills its second unit slot from the second `Target_Unit`
  parameter. On the shipped corpus the two are the same slot: all 3 nodes of opcode 18 bind
  `[Unit t=4][Unit t=4]` and nothing else.
- **SC-8** — **Nothing here was watched on a screen.** No windowed game was launched for this
  story. See `verification.md`.

## Out of scope

- The other unrunnable instant opcodes (2, 7, 20, 21, 24, 25, 29, 30, 34) and the three
  unimplemented group sub-commands. They are a different contract.
- Any effect of map presence on dialogue, sound or the mission report.
- Whether an off-map unit regenerates health or mana. This build leaves the regeneration pass
  untouched, so it does — the claim says nothing about it and inventing a rule would be an
  unverified fact.

## Customisation limits this decode implies (G2)

- The placement search reaches at most `[x-1, x+2] x [y-1, y+2]` around the retained cell. This is
  an engine search bound, not a stored field: widening it changes no byte of any shipped file.
- The retained cell is read from the unit's own position, whose coordinates are bytes in the
  original (`TRIG-INSTCENSUS-046`: the addressable space is 256 x 256, and the largest shipped
  campaign map is exactly 256 x 256). This build already carries `int32` coordinates, so it meets
  the limit and does not impose it.
- Nothing about map presence is authored in a map file. There is no per-node cell, no radius and
  no retry count to lift.
