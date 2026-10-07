# Spec — a group decides whom to fight, and nobody has to be told

**Intensity:** spec-anchored / static. **Terrain:** brownfield in `pkg/sim` — a world that has
only ever taken orders now issues them to itself, and every world's byte form changes — and
greenfield for the decision itself and for the roster row it reads.

## Why

Nothing here acquires. A blow needs an attack order, an attack order needs a command, and a
command needs a click, so a mission opens with its monsters standing still and stays that way.
This is the largest single gap between the tree and a game.

The decision is made **per group and once per full tick**, and the group — not the individual — is
what fights: over the shipped corpus 95.6 % of hostile placements sit under a group order and
0.7 % under the per-actor state machine. What this contract fixes is the group's decision half:
who is a candidate, which candidate a member takes, and what taking one does. Walking somewhere
because there is no candidate is a separate behaviour and is out of scope below.

## Requirements

### The relation

**FR-1** A world carries a **relation**: a square byte matrix over roster slots, of a fixed 50
slots per side, indexed `[me][him]` by the entity's own owner slot. It is **canonical state** — it
is carried by the byte form and enters the digest — and it does not change while a world is
advanced.

**FR-2** The relation is **directional**. `[a][b]` and `[b][a]` are independent, and no operation
in this build derives one from the other.

**FR-3** Hostility is **bit 0** of the byte, and it is the only bit any rule in this build reads.
Every other bit is carried and never interpreted.

**FR-4** A world that names no relation holds an all-zero one, and under an all-zero relation no
entity is hostile to any entity. A world built before this story existed therefore behaves exactly
as it did.

**FR-5** An entity whose owner slot is outside the matrix — slot 0, which names no slot, or a slot
at or above 50 — is hostile to nothing and nothing is hostile to it.

**FR-6** A map's type-5 roster authors the relation: for the roster entry at 1-based slot `i`, the
sixteen 16-bit words at record offset `0x2c` supply `[i][1] … [i][16]` as their **low bytes**, and
the diagonal `[i][i]` is then forced to 2. Column 0 is never written.

### The decision

**FR-7** On exactly one phase of the sixteen-tick cycle — the phase the mission script already
runs on, and after it — every group takes one engagement decision. No decision is taken on any
other tick.

**FR-8** A **group** is the set of alive entities sharing an owner slot and a group word. An
entity whose owner slot is 0 belongs to no group and takes no decision; it remains a candidate for
every other group.

**FR-9** A group's **order** is derived from its owner slot: slot 1 — the map's type-5 slot 0 —
stands its ground, and every other slot guards.

**FR-10** The **candidate population** is the group's shared sight: an entity is a candidate when
it stands within the sight radius, by Chebyshev distance, of **any** member of the deciding group.
An entity seen by one member is a candidate for every member.

**FR-11** Candidates are filtered by the relation row of the group's **first member** — its lowest
entity id — so one row governs the whole group.

**FR-12** Candidates at health below 1 are **parked, not dropped**: they move to a second list, and
when the first list ends empty the whole second list is moved back. A group that can see only
corpses therefore has a candidate list and attacks a corpse.

**FR-13** A **guarding** group clips its list to the notice radius, by Chebyshev distance from the
group's centroid; a candidate beyond it is dropped. A group **standing its ground** applies no
clip.

**FR-14** The notice radius is `max(g, minimalGuardRange) + noticeMargin`, where `g` is the maximum
over members of that member's Chebyshev distance from the centroid plus its sight radius, each
member's sum narrowed to a byte.

**FR-15** Each member is assigned the **cheapest** candidate: the minimum is strict, so the first
candidate of an equal-cost pair wins and list order decides. Assignment is recomputed from nothing
on every decision; no memory of a previous decision exists anywhere.

**FR-16** The cost of a candidate to a member is `(d << 8) + t`, lower being better, where `d` is
the Chebyshev distance between them in cells and `t` is the turn cost from the member's facing to
the direction of the candidate.

**FR-17** A **preference** term modifies the cost, read from a fixed 4x4 matrix indexed
`[member domain][candidate domain]` over the law's own domain numbering. A preference of 0
**vetoes the candidate outright** — it can never be taken. A preference of 1 raises the cost by
half of itself; a preference of 4 lowers it by a quarter of itself.

**FR-18** A group **standing its ground** scores by a second rule: a distance term above 1 vetoes
the candidate before anything else is computed, and the two preference modifiers are a doubling
and a halving rather than the two fractions of FR-17.

**FR-19** A member holding a candidate is **engaged**: it takes an attack order naming that
candidate, which ends whatever walk it was on. An engagement naming the victim the member already
holds leaves its attack cycle untouched.

**FR-20** A member holding **no** candidate is left exactly as it was. No decision at either group
order breaks off an attack.

## Acceptance criteria

- **AC-1** Two worlds differing only in one relation byte have different digests, and a world
  round-trips through its byte form with its relation intact.
- **AC-2** A world built naming no relation has the state, the byte form under the current version
  and the digest of one built naming an all-zero relation.
- **AC-3** A relation byte with bit 0 set one way and clear the other produces acquisition in one
  direction only.
- **AC-4** Two hostile entities placed apart, advanced with no commands at all, end with the
  guarding one holding an attack order on the other, and with health taken off it.
- **AC-5** No decision is taken on any tick whose phase is not the decision phase: an entity placed
  in range acquires on the first decision tick and on no earlier one.
- **AC-6** An entity outside every member's sight radius is not acquired; one inside it is.
- **AC-7** The notice radius is the arithmetic FR-14 names, and `clipToNotice` drops exactly the
  candidates beyond it — checked as arithmetic, because D-3 shows no world can present it one.
- **AC-8** A group standing its ground acquires a candidate at Chebyshev distance 1 and does not
  acquire one at distance 2, at any sight radius.
- **AC-9** A ground or ghost member never acquires an air candidate; an air member acquires either.
- **AC-10** Given two candidates, the member takes the nearer; given two at equal distance, it
  takes the one earlier in the list.
- **AC-11** A group whose only candidates are bodies selects one; a group with one living
  candidate and a nearer body selects the living one.
- **AC-12** A member re-engaged on the victim it already holds does not restart its attack cycle,
  and lands its blows on the cadence it would have landed them on with no decision running.
- **AC-13** Making a relation friendly does not end an attack already issued; the attacker keeps
  striking and is simply not re-selected.
- **AC-14** An entity of owner slot 0 acquires nothing and is acquired by nobody.
- **AC-15** A map's roster row reaches the world's relation with its low bytes and its forced
  diagonal, and a map with no type-5 roster produces an all-zero relation.

## Properties

- **P-1** *Determinism.* The decision reads no clock, iterates no map, and draws from no generator.
  Two worlds with equal byte forms advanced against equal commands stay equal, and a recorded drive
  replays byte-identical.
- **P-2** *Injectivity.* Every relation byte survives the byte form unchanged; no byte is masked,
  folded or normalised on the way in or out, so two worlds the form distinguishes stay two worlds.
- **P-3** *Idempotence of a decision.* Taking two decisions on a world nothing else has changed
  leaves the same assignment, because nothing in the decision is sticky.
- **P-4** *Completeness of the order.* A member that ends a decision holding an attack order holds
  no destination of its own, and one holding neither is unchanged in every field.
- **P-5** *The clip is inert while the radius is fresh.* Over a swept space of group geometries and
  candidate placements, no candidate FR-10 admits is ever dropped by FR-13. This is the measured
  form of D-3 and it is the test that fails the day the freeze lands, which is when it should.

## Divergence from the published law

- **D-1** *No line of sight.* The candidate population is the Chebyshev disk of the sight radius,
  with no occlusion. The law's predicate is an accumulator march whose two input grids have no
  traced writer, so its region cannot be reproduced — only bounded, since the law's region is a
  subset of this one. **A group here sees through walls.**
- **D-2** *No turn cost.* `t` in FR-16 is 0, because the routine that computes it is named at its
  call sites and never decoded. It occupies the low byte under a distance term shifted eight bits,
  so it changes no outcome except the order of equidistant candidates, which becomes list order.
- **D-3** *The notice radius is not frozen, and under a fresh one FR-13's clip cannot fire at all.*
  The law computes FR-14's geometry every decision and reads a copy frozen when guard was last
  issued; this build reads the fresh one, because the freeze needs a group record this tree does
  not carry. The consequence is stronger than "a group that spreads out widens its circle": a
  freshly computed radius is at least `max over members (distance + sight)`, and Chebyshev distance
  obeys the triangle inequality, so every candidate FR-10 admits is already inside it. **The clip
  is therefore inert in this build**, and the frozen radius is the whole of what makes it a rule.
  It is implemented and pinned as arithmetic (P-5) so that the story adding the freeze changes
  where the number comes from and nothing else.
- **D-8** *An engage on a body ends within the tick that issued it.* FR-12 selects a corpse, but
  this tree's attack cycle already ends an order whose victim is **dead** (0064), so the law's
  "a group with nothing but corpses in sight keeps attacking one" is reproduced for a **downed**
  body and not for a dead one. The selection is the part this story owns and it is witnessed
  directly; lifting the rest is a change to the cycle, not to the decision.
- **D-4** *No radius jitter.* The law adds one of {-1, 0, +1} once, on the tick a has-members latch
  flips. The latch is group state this tree does not carry, so the radius is the roll's midpoint.
- **D-5** *No attacker memory.* The law forces an AI-owned member's remembered attacker cell into
  the sight stamp for 20 decisions. Nothing here strikes from outside its own sight radius, so no
  world this build can construct can tell the difference.
- **D-6** *No invisibility and no spell term.* The law drops an invisible candidate unless a member
  is inside its see-invisible radius, and adds a flat 127 to a candidate carrying one spell id.
  This tree has neither invisibility nor spells.
- **D-7** *The reach-dependent arms are unreachable.* The law folds a candidate's domain to 0 when
  its reach exceeds 1, and scores a member of reach above 1 off row 0 with a rebased distance.
  Reach is a constant 1 here, so both arms are written and neither can fire.

## Out of scope

The **walk half** of the guard arm — a targetless member walking to its post and taking the idle
turn — and with it the post itself; the withdraw tail and its two `Data.bin` columns; the six
other group orders, patrol, roam and the player's nineteen order opcodes; the script's group
command and its diplomacy action; the join, leave and combat writers of the relation; the
per-actor state machine; and the heal and cast arms.

## Traceability

| FR | AC | Property |
|---|---|---|
| FR-1, FR-2, FR-3 | AC-1, AC-3 | P-2 |
| FR-4 | AC-2 | P-2 |
| FR-5 | AC-14 | — |
| FR-6 | AC-15 | P-2 |
| FR-7 | AC-5 | P-1 |
| FR-8, FR-9 | AC-8, AC-14 | — |
| FR-10, FR-11 | AC-6, AC-3 | — |
| FR-12 | AC-11 | — |
| FR-13, FR-14 | AC-7 | P-5 |
| FR-15, FR-16 | AC-10 | P-3 |
| FR-17 | AC-9 | — |
| FR-18 | AC-8 | — |
| FR-19 | AC-4, AC-12 | P-4 |
| FR-20 | AC-13 | P-4 |
