# Spec — the group command's Patrol arm, and the per-actor state it hands members to

**Intensity:** spec-anchored / static. **Terrain:** brownfield — opcode 6 and five of its
sub-commands ship (0096); this adds a sixth. **Threshold: High** — it reaches hashed simulation
state and bumps the byte form.

## Terms

- **The group command** is mission-script action opcode 6, already a second dispatch on the node's
  first plain parameter. **Sub-command 14 is Patrol.**
- **A member** is a living entity carrying the commanded group's (owner, group) pair. This is the
  same set the four existing writing sub-commands act on.
- **The group order** is the byte a group record already carries, and the thing the engagement
  decision forks on. **Order 0** is what a group carries before anything installs one; no group a
  world can hold carries it today.
- **The actor state** is a new per-entity byte: the state machine an actor runs when its group is
  not deciding for it. Its value at construction is **guard**, and this build has **one arm**,
  **patrol**.
- **The ring** is the two cells a patrolling actor walks between: its **head**, the cell the actor
  stood on when it was commanded, and its **tail**, the cell the node named. **The leg** says which
  of the two is the current waypoint.
- **The actor pass** is the new tick pass that runs the arm.
- **Both roots** are the two lawful installs. A claim made on one is not made.

## Why

Mission 10 is the first mission of the campaign and the one the milestone is measured on. Its
`"Just Start"` trigger fires on the first evaluation pass and issues three instants, two of which
are Patrol. This build runs neither, so two placed villagers stand where the map put them for the
whole mission instead of walking the routes they were authored to walk. **Fourteen such nodes ship
on eight campaign maps**, so the arm is not a mission-10 special case.

It matters beyond the two units, because **Patrol is the first order that takes a group's members
away from the group**. Every AI decision this build makes is made at the group layer; the shipped
command clears the group's order and hands each member to its own state machine. Four more orders
of the same family follow it. This story therefore introduces the per-actor layer as **one arm
behind a dispatch**, so that adding the next is adding a case rather than rebuilding the tick.

**This contract makes no claim about mission 10's outcome.** The two commanded groups are single
villagers in the map's north-east; the outcome is measured before and after and reported in
`verification.md`, and nothing here is shaped to change it.

## Scope

**In scope.** Sub-command 14. The group-order clear and order 0 becoming storable state. The actor
state byte, its dispatch and its one arm. The ring, the leg and the walk. The byte form and the
digest.

**Out of scope.** Every other actor state, the per-actor guard included — so the leash, the guard
post and its re-anchor latch (D-3). The other four unimplemented sub-commands. The mission
description file's patrol loader, which ships no input. The player's own patrol command. Whether
the actor pass should run on a phase of its own. `missionrun`'s occupancy-blind aim.

## The contract

### The command

**FR-1** This build **runs** sub-command 14: the compile-time report of arms it does not run no
longer names it, and the runtime dispatch reaches it. The report and the dispatch stay one table
(0096 DD-8).

**FR-2** A Patrol node naming **no group**, or a group **no record names**, changes nothing —
the rule the five existing sub-commands already take.

**FR-3** The command sets the commanded group's order to **0** and writes nothing else on the group
record. In particular it does not write the commanded cell: the cell is a per-member fact here, and
whatever an earlier command left is unread while the order is 0.

**FR-4** **Order 0 is legal stored state.** *This supersedes 0096 FR-19, which refuses it.* A decode
carries it, and it means what it has always meant: **the engagement decision does not decide for
that group at all** — no member is scored, released, given a candidate or given a destination by it.
That behaviour is unchanged and is not re-implemented here; only its reachability is new.

**FR-5** Each **member** is **stopped**: it holds no destination, no stall count, no stored route,
no victim, no attack phase, no attack countdown and no group rate term.

**FR-6** Each member's actor state becomes **patrol** and its ring is built: **head** = the cell
that member stands on, **tail** = the node's (x, y), clamped into the map. **The leg is the tail.**
A ring is built per member, so members standing apart get different rings.

### The actor pass

**FR-7** The actor pass runs **once per full tick**, on the phase the mission script and the
engagement decision already run on, and **after** the engagement decision. It visits the world's
living entities in ascending id order and runs the arm named by each one's state. **An entity whose
state this build has no arm for reaches no arm and is left in every field** — the same "not
implemented, not merely inert" treatment the script's own dispatch gives an unimplemented opcode.

**FR-8** The **patrol arm**, for one actor:

1. its destination, stall count and stored route are cleared;
2. if it **stands on the current waypoint**, the leg advances to the **other** node of the ring —
   the ring is two cells, so this is the whole of the ring walk;
3. its destination becomes the current waypoint.

A consequence, stated because it is observable: the order is re-issued every pass, so **a patroller
never accumulates a stall count** and never gives a blocked walk up.

**FR-9** A patroller is **not decided over by the group layer**. *Revised during implementation:*
this needs a **clause of its own** in the decision, not merely FR-4. Order 0 became storable and so
also became a value the decision's own reachability test admits, and that test was the only thing
stopping it short of a group nothing has an arm for. The clause is checkable by deletion (AC-7).
*Divergence, disclosed (D-3):* the law's arm runs the per-actor **guard**
before the ring walk, so a shipped patroller still fights and only advances when guard leaves it
idle. This build has no per-actor guard, so **a patroller in this build does not acquire and does
not fight**, and the arm's step 2 is unconditional where the law's is gated.

**FR-10** A patroller that stops being alive **holds no patrol**: its ring and leg are cleared and
its state returns to guard, at the one site the felling already clears its order, its crossing, its
group term and its victim.

### The state

**FR-11** The actor state is written into **every** entity by the constructor, unconditionally, at
**guard**. It is not an input a caller supplies: the law's actor initialiser installs it and the
map's own spawner writes none.

**FR-12** The **byte form is version 21** and version 20 is refused, as every earlier version is.
The entity record's tail gains the state byte, the ring and the leg; the group record's order byte
now admits 0 beside the five. No section moves and no count is added.

**FR-13** The form **refuses**, naming the value and the record: a state outside {guard, patrol}; a
leg outside {0, 1}; a ring or a leg on an entity not in the patrol state; and a patrol state on an
entity that is not alive. The constructor **normalises** each of the last three where it could have
refused — the relation the order fields, the crossing and the decay stage already stand in.

## Design decisions

- **D-1 — the seam.** The dispatch is keyed on the state byte and has **one arm**. The next order
  of this family adds a constant and a case; it does not move the pass, the field, the form or the
  group-order clear. This is G2 as a seam rather than a feature, and **SC-1 is what makes the claim
  checkable**: deleting the arm must leave the dispatch, the pass and the state compiling.
- **D-2 — the leg is an index, not a cell.** The law stores the current waypoint's **cell** and
  searches the list for it on arrival, falling back to the head when the search misses. Over every
  ring **this build can construct** the two agree exactly: the ring has two nodes, the command makes
  the tail current, and the tail is in the list. The one input they part on needs a routine
  `AI-PATROL-019` measures as having no live caller.
- **D-3 — the guard post and its re-anchor latch are not modelled, and that is a decision.** The
  law's command writes a guard post at the actor's cell, and the arm sets a latch on every advance
  which the next entry consumes to move that post to where the actor now stands — which is why the
  guard leash never pulls a patroller home. **Both are written by the command and read only by the
  per-actor guard**, and this build has no per-actor guard. Modelling them here would add two
  fields to the record, the form and the digest with **no reader**, and the story that adds the
  guard arm has to add the latch's consume-on-entry anyway. Named as the seam, not omitted.
- **D-4 — the pass runs after the engagement decision, on the same phase.** Ours. Nothing read
  orders the actor machine against the group decision. The reasoning is the one already written for
  the script/decision order: the group layer's clear is what hands the actor over, so it happens
  before the actor acts on it, in the same tick.
- **D-5 — the ring is clamped once, where it is built.** The node's cell is the only value from
  outside, so the arm needs no clamp and can never write a destination the map does not hold.

## Acceptance

- **AC-1** On mission 10, `almtool script` reports sub-command 14 as **2 nodes not implemented**
  before and **not at all** after. The counts for the other unimplemented arms are unchanged.
- **AC-2** On both roots, mission 10's two commanded groups — one villager at (64,21) commanded to
  (53,19), one at (67,12) commanded to (54,13) — **leave their placement and come back to it**
  within a bounded run. A build that never dispatched the arm leaves both standing.
- **AC-3** A world holding a patrolling actor **round-trips**: the same bytes and the same digest
  out of decode as into encode.
- **AC-4** A version-20 form is refused, naming both versions.
- **AC-5** Each refusal of FR-13 is witnessed on a form built to carry exactly it.
- **AC-6** Mission 10's outcome is **recorded before and after** in `verification.md`, on both
  roots, together with what was measured about the cause. Neither value is an acceptance condition;
  no test, predicate or number is tuned to move it.
- **AC-7** A group under order 0 is stepped and **no member of it is scored, released or moved by
  the engagement decision**.

## Supplementary criteria

- **SC-1** Deleting the patrol arm alone leaves the pass, the dispatch and the state field
  compiling, and leaves every entity in every field on every tick. (D-1.)
- **SC-2** The report of arms this build does not run and the runtime dispatch stay one table: no
  sub-command is claimed by one and not the other.
