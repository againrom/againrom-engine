# Spec — a mission script commands its groups, and a group's order stops being its owner's

**Intensity:** spec-anchored / dynamic. **Terrain:** brownfield.

## Why

A group's behaviour in this build is a property of *whose* it is: one on the local participant's
roster slot stands its ground, every other owned group guards a circle. That is right for the moment
a map loads and it is all this tree has, so the loaded behaviour lasts the whole mission.

On shipped content that is not what happens. A campaign map's own script hands its groups orders
while the mission runs — 135 nodes over 20 of the 38 maps, one action opcode dispatching again on
its own first parameter to select among ten behaviours — and a group's order is whatever the last
such node left. This build runs none of them.

So the order becomes state the world **holds** rather than something a decision derives, the script
gets the command that writes it, and the arms it selects get built: five of the ten sub-commands,
covering 100 of those 135 nodes. Because an order then outlives a tick, the byte form carries it and
the digest covers it — and because an order no longer implies an owner, two rules written against
the old identity have to be restated against the thing they were always about (FR-22, FR-23).

## Requirements

### The order is state

- **FR-1** Each group record carries a **group order**, one byte, and a **commanded cell**, a
  coordinate pair. The record set is unchanged — no order, death or arrival adds or removes one.
- **FR-2** At construction a group's order is its owner's: one on the roster slot the local
  participant sits at takes **Stand Ground**, every other owned group **Guard**. Its commanded cell
  is the origin, unread until a command writes one.
- **FR-3** A decision reads the **stored** order. Nothing on any tick path derives an order from an
  owner, and nothing but a group command writes one.
- **FR-21** A decision for a `(owner, group)` pair **no record names** — reachable because the
  hand-over arms rewrite an owner inside a tick — uses order **0**, what a group carries before
  anything installs one. Order 0 has **no arm here**, so the group takes no decision and every
  member is left in every field. Not an error, and not a re-derivation from the owner.
- **FR-22** A destination a **group order** wrote does not put its holder out of its group. The
  build excludes a unit executing a **player's** order; a member walking under orders 2, 4 or 5
  walks *because* of that group's order and must keep being decided for, or the arm that issued the
  walk never runs again.
- **FR-23** The release of a member that scores nothing is a rule about the member's **owner** — a
  human participant's unit is not broken off — and is stated as one, not as a rule about the order
  the group is under. After this story an order no longer implies an owner: a script may stand an
  enemy group's ground, or set the participant's group to guard.

### The command

- **FR-4** The build runs mission-script action opcode **6**, a **second dispatch** on the node's
  first plain parameter. Five sub-commands act — `1`, `2`, `3`, `4`, `5` — and every other value
  changes nothing, whether the law implements it or not.
- **FR-5** A node naming **no group**, and one naming a group no entity carries, change nothing and
  are not errors — exactly as counting such a group gives zero.
- **FR-6** Sub-command **`1`** sets the order to Guard **and re-freezes the notice base** from the
  geometry its living members hold at that moment, by the rule construction uses. It is the only
  such writer after construction.
- **FR-7** Sub-command **`3`** sets the order to Stand Ground and changes nothing else.
- **FR-8** Sub-command **`2`** sets the order to Swarm and the commanded cell to the node's second
  and third plain parameters as `(X, Y)`. It gives no member a destination.
- **FR-9** Sub-command **`4`** sets the order to Move **and issues the group a destination** — the
  same distribution a player's own group move performs, over the node's `(X, Y)`: the formation gate
  on the members' spread about their centroid, each member offset by its displacement from it when
  in formation and sent to the bare cell when not, the rate term from the slowest member on the
  formation arm alone, destinations clamped, fights ended.
- **FR-10** Sub-command **`5`** is FR-9 with the order **Swarm 2**, and differs in nothing else.
- **FR-11** The report of arms this build does not run names the **sub-command**, not only the
  opcode.

### The arms

- **FR-12** **Guard** and **Stand Ground** are unchanged: Guard clips the candidates to the circle
  about the centroid and scores by the ordinary cost; Stand Ground clips nothing and refuses any
  candidate the member could not already strike.
- **FR-13** **Swarm** clips nothing and scores by the ordinary cost. A member that scores a
  candidate engages it. One that scores none and is not idle on the commanded cell is sent there —
  **unoffset**, one cell for every member, no formation, no spread — and its fight ends. One that
  scores none and *is* idle there is left in every field.
- **FR-14** **Swarm 2** is gated on whether the candidate list is **empty** — the list built for
  this decision, **corpses included**, so a group seeing only a body is not empty, and its length
  read **whole** where the scorer's own emptiness test reads a low byte (D-7). Empty, it takes
  **Move's arm** (FR-15); otherwise it clips nothing and scores by the ordinary cost, a member that
  scores a candidate engages it, one that scores none is left in every field including its
  destination, and **this arm never walks**.
- **FR-15** **Move** clips nothing, scores by Stand Ground's rule, and scores **only** arrived
  members — one still holding a destination takes no decision and is left in every field.
- **FR-16** Under every order a member that scores nothing keeps its order, destination and facing,
  except where FR-13 replaces the destination.

### The canonical form

- **FR-17** A group's order and its commanded cell are canonical simulation state: both are carried
  by the byte form and both enter the digest. Two worlds differing only in one order, or in one
  commanded cell, are two worlds.
- **FR-18** The byte form is at version **20**. The group section grows in place: every offset
  before it is unmoved, the script section still closes the form and consumes what is left of it
  exactly, and every other version byte is refused with its own number named.
- **FR-19** A decode **refuses** an order byte outside the five this build writes, naming the value
  and the record: constructor and decoder accept the same set.
- **FR-20** A commanded cell is carried **whole**: no coordinate refused, folded or clamped.

## Acceptance criteria

- **AC-1** A world built over two groups on different roster slots holds the orders FR-2 gives them;
  stepping it any number of ticks leaves both unchanged.
- **AC-2** A group command moves exactly one order: the record it names changes, every other is
  bit-for-bit what it was.
- **AC-3** Each of the five sub-commands leaves the order it names; each of `10`, `11`, `14`, `15`,
  `17`, `18` and a value the law does not implement leaves order, cell and every entity untouched, as
  do a node with no group and one naming a group no entity carries.
- **AC-5** After sub-command `1` on a group whose members have moved since construction, the notice
  base is what its *present* geometry gives — not the one it was built with — and the next
  decision's clip uses it.
- **AC-6** After sub-command `4` on a group in formation, each member holds the ordered cell offset
  by its own displacement from the centroid and carries the slowest member's rate term; out of
  formation, the bare cell and no term. `5` gives identical destinations, a different order.
- **AC-7** After sub-command `2`, no member has a destination; on the next decision a member with no
  candidate is sent to the commanded cell **unoffset**, and two members standing apart go to the
  same cell.
- **AC-8** A group under Swarm engages a candidate the same group under Guard would have clipped
  away.
- **AC-9** A member under Move still holding a destination takes no decision even with a candidate
  inside its reach; having arrived, it engages that candidate. One past reach is refused under Move
  exactly as under Stand Ground.
- **AC-10** The gap report separates the sub-commands this build runs from those it does not: a
  script authoring `4` and `14` reports one gap, naming `14`.
- **AC-11** Two worlds differing only in one order byte, and two only in one cell, marshal to
  different bytes and hash to different digests.
- **AC-12** A world marshalled and decoded holds every order and cell it held, over each of the five
  orders and over coordinates at both extremes.
- **AC-13** The form opens at the version this build defines and any other is refused with its
  number named. **No test states the version as a literal.**
- **AC-14** A decode refuses an order outside the five, naming the value and record.
- **AC-15** The pinned world's bytes and digest are a hand transcription carrying both new fields.
- **AC-16** `TestTheTenthMissionIsDrivenToAWin` reports the same outcome and tick as before this
  story. The predicate is not edited and no threshold is tuned.
- **AC-17** A group under Swarm 2 seeing only a **corpse** runs its own body: no member gets a
  destination. A gate written over the living candidates fails this.
- **AC-18** A group under Swarm 2 whose candidates the preference table all veto leaves every
  member's order, destination and facing untouched — it does **not** walk, where the same group
  under Swarm does.
- **AC-19** After a hand-over gives one group's entities an owner no record names, that group takes
  no decision — with a candidate in sight and inside reach, no member's order, destination or facing
  changes — while every other group in that world decides as before.
- **AC-20** A group commanded to Move keeps deciding while its members walk: on the tick after the
  command every member still belongs to its group, and on arrival one engages. A unit walking under
  a **player's** order is still excluded, and the property pinning which functions may give a
  destination names the arms this story adds.
- **AC-21** A group at Stand Ground whose owner is **not** the local participant releases a member
  that scores nothing; the participant's own group **at guard** does not. Both reverse what a
  stance-keyed rule gives, and each fails if the release reads the order instead of the owner.

## Properties

- **P-1** *Invariant.* The record set and its key set are unchanged: only the two new fields are
  writable, and only by a group command.
- **P-2** *Totality.* Every group a decision can be taken for has an order — its record's, or
  FR-21's zero — and every order a **record** can hold has an arm. Zero is the one order no record
  holds and the one with no arm; it is answered by taking no decision, not by a default.
- **P-3** *Idempotence.* Marshalling a decoded world reproduces the bytes it was decoded from.
- **P-4** *Negative invariant.* A sub-command this build does not run leaves the world bit-for-bit
  unchanged — not merely the group it names.
- **P-5** *Symmetry.* Constructor and decoder accept the same orders and the same commanded cells.

## Divergence from the published law

- **D-1** *Four sub-commands are absent rather than approximated.* `10`, `11`, `14` and `15` — 35 of
  the 135 shipped nodes — set the order to the value handing every member to a per-actor state
  machine, then write that machine's state. This tree has no such machine, and it does **not** write
  the order byte alone: a commanded group would lose the behaviour it had and gain none.
- **D-2** *Move's arm substitutes a scorer.* The law hands an arrived member to the standing
  per-actor acquisition rule — everything visible, nearest first, a turn tiebreak, the pick
  discarded unless within reach. This build uses Stand Ground's rule, which agrees on the reach cap
  and nothing else. The law's arm also turns reach into a stop distance and **re-issues the move**
  while the rate term is set; neither has anything here to write to.
- **D-3** *A player's group move does not touch a group record.* In the law a player order allocates
  a **fresh** group at order 0 whose byte the setter moves, leaving the placed group's untouched; a
  scenario command has no such allocation, and this tree can allocate no group. The placed group's
  order is left alone on the player path.
- **D-4** *Swarm 2's gate is built; its far side is silent.* Neither arm is observable here — Move's
  arm scores nothing against an empty list and its re-issue is unwritable (D-2) — so what is
  witnessed is the gate's condition and the branch that does not walk, not the fallback.
- **D-5** *The commanded cell is carried at the coordinate's own width.* The law packs it into two
  bytes; no map this tree can load is 256 cells on a side. A customisation limit.
- **D-6** *The idle-turn and heal arms are still absent*, unchanged from 0086: a member with nothing
  to fight and nowhere to go is left standing, where the law turns it on the spot or heals it.
- **D-7** *The candidate count is read at **two** widths and the difference is real.* The scorer's
  emptiness test reads its low byte — already built, a customisation limit at exactly 256 visible
  candidates — while order 5's gate reads it whole. This story adds the second width rather than
  reusing the first: at 256 the gate does **not** fire, the arm runs its own body, the scorer finds
  nothing and the group stands. The byte test there would have sent it walking.

## Out of scope

- **The per-actor state machine**, and with it patrol, follow, attack and defend (D-1) — the next
  story. **Roam**, whose order value ships in 0 nodes over 38 maps.
- **The walk home to a post**, and **the has-members latch, its counter and the radius roll** — both
  unchanged from 0095's out-of-scope list, the second reachable for the first time now that a second
  freeze moment exists.
- **The player's own order vocabulary**, and **the formation distribution**, which this story calls
  without touching. The sight march, the diplomacy matrix, the trigger's announcement.

## Traceability

| Requirement | Criteria | Properties |
|---|---|---|
| FR-1 | AC-1, AC-11 | P-1 |
| FR-2 | AC-1 | P-1, P-5 |
| FR-3 | AC-1, AC-8 | P-2 |
| FR-21 | AC-19 | P-2 |
| FR-22 | AC-20 | P-2 |
| FR-23 | AC-21 | P-4 |
| FR-4 | AC-3 | P-4 |
| FR-5 | AC-3 | P-4 |
| FR-6 | AC-3, AC-5 | — |
| FR-7 | AC-3 | — |
| FR-8 | AC-3, AC-7 | — |
| FR-9 | AC-3, AC-6 | — |
| FR-10 | AC-3, AC-6 | — |
| FR-11 | AC-10 | — |
| FR-12 | AC-8, AC-16 | P-2 |
| FR-13 | AC-7, AC-8 | P-2 |
| FR-14 | AC-8, AC-17, AC-18 | P-2 |
| FR-15 | AC-9 | P-2 |
| FR-16 | AC-9 | P-4 |
| FR-17 | AC-11, AC-12, AC-15 | P-3 |
| FR-18 | AC-13, AC-15 | P-3 |
| FR-19 | AC-14 | P-5 |
| FR-20 | AC-12 | P-5 |
