# 0166-last-arms — spec

The last decoded script operations this build cannot run: instants 7, 20, 25, 30 and 34, and group
sub-commands 10, 11 and 15. With them the campaign's mechanical script gap is zero. Only instant 2,
a broadcast that touches no simulation state, is left.

This document is self-contained. Research provenance is in `provenance.md`.

Intensity: rigor **high** (the AI order writes and the new per-player and per-cell state are hashed
simulation state), scope **medium**. Terrain: `pkg/sim`.

## FR — functional requirements

### Instant 7, the formation mode

**FR-1.** Instant 7 writes the byte `(u8)p0` into the named player's **formation mode**. The value is
written raw: there is no remapping on this path. A node naming no player, or a player outside the
world's roster, writes nothing.

**FR-2.** Every player has a formation mode and its default is **2**. A world built by any
constructor of this package has every roster slot at 2 until something writes one.

**FR-3.** The formation mode gates the group-move distribution, with three behaviours:

1. mode **0** — the group is never in formation;
2. mode **2** — the group is in formation exactly when the spread test passes, which is what this
   build did for every group before this story;
3. any other nonzero mode — the group is in formation unconditionally, with no spread test.

The mode read is the mode of the **owner of the group's first member**, since a group's members
share one owner. A member set with no owner reads slot 0's mode.

### Instant 20, drop all

**FR-4.** Instant 20 moves the **whole** of the named unit's container onto the ground at the cell
the unit itself stands on, and leaves the unit holding an empty container. No authored coordinate is
read: the node carries none.

**FR-5.** The drop **merges** into a sack already standing on that cell rather than making a second
one. Where the cell holds no sack, one is planted there.

**FR-6.** Nothing else moves: no gold, no equipment slot, no position, no owner, no group, no
health, no order and not the tick. A node naming no unit, a node naming a unit this world does not
hold, and a unit whose cell is outside the map leave the world exactly as it was found. A unit
holding nothing plants no sack.

### Instant 25, the cell-record tail

**FR-7.** Instant 25 takes `(spell, power, x, y)` and writes a six-byte **tail** onto the cell
record the pair `(x, y)` addresses. The cell is keyed as `(u8(y) << 8) OR u8(x)` — both operands
truncated to a byte before the shift, so an `x` above 255 does not carry into the `y` byte.

**FR-8.** The six bytes are `{spell, power, 0, y, 0, y}`, each narrowed to a byte. The authored `x`
is **not** copied into the tail. A cell that already carries a tail has it overwritten; a cell that
does not gains one.

**FR-9.** The tail is persistent state: it survives a mid-mission save and it is in the digest. No
rule of this build reads it, which is the shipped result — the only behavioural reader of the
original's tail tests its first byte for 26, and the two shipped nodes write 3 and 9.

### Instant 30, the attached-effect duration

**FR-10.** Instant 30 re-times the **attached effect** the named unit is carrying: where that
effect's spell id equals `(u8)p0`, its remaining duration becomes `(u16)p1`. A node naming no unit, a
unit this world does not hold, a unit carrying no attached effect, and an effect whose id does not
match all leave the world exactly as it was found.

**FR-11.** A duration of 0 is written as the arm writes it, with no refusal in front of the store.
The effect is then removed by the next decay pass, which is what an authored 0 asks for.

**FR-12.** An attached effect's remaining duration is a **word**. The largest value the campaign
authors is 60000, and it is carried whole rather than narrowed.

### Instant 34, the property setter

**FR-13.** Instant 34 is a word-wide property setter selected by `p0` on the named unit:

- selector **6** writes `(u16)p1` to current health;
- selector **15** writes `(u16)p1` to defence;
- selector **16** writes `(u16)p1` to absorption;
- every other selector stores nothing.

**FR-14.** There is no clamp, no maximum-health update and no derived-stat recomputation. A health
above the unit's maximum is written and kept. A node naming no unit, or a unit this world does not
hold, changes nothing.

**FR-15.** A health of 0 or below fells the unit through this build's own death path rather than
being stored raw, so no world this arm produces is one this package cannot marshal and read back.
The campaign authors selector 6 once, with `p1 = 1`, so no shipped node reaches this.

### Sub-command 10, the script attack

**FR-16.** Sub-command 10 first **stops** every living member of the named group: destination,
stored route, stall count, victim and attack cycle all cleared, and the group's own order set to
none.

**FR-17.** It then walks the group a second time and, for each living member, scores that member
against the named unit with this build's own ordinary target scorer. A member whose score is the
**veto sentinel** takes the acquire-in-place disposition and never engages. Every other member
takes:

- the named unit itself — acquire in place;
- every other member — **engage the named unit**, through the same order writer the player's own
  attack command uses.

**FR-18.** A node naming no group, a node naming no unit, and a node naming a unit this world does
not hold leave the world exactly as it was found — including the stop, which is not performed. A
group id no entity carries changes nothing, which is not a failure.

### Sub-commands 11 and 15, defend and follow

**FR-19.** Sub-commands 11 and 15 are one shape and two state constants: **defend** for 11 and
**follow** for 15. Each stops every living member of the named group on FR-16's own terms, then
splits on `member == the named unit`:

- the named unit itself — acquire in place;
- every other member — the escort state for that sub-command, with the named unit as its **escort
  target** and the node's own **escort range**.

**FR-20.** The escort range is the node's third value narrowed to a byte, or **3** where that byte
is 0. Shipped ranges are 1 to 6, so no shipped node reaches the coercion.

**FR-21.** FR-18's three refusals hold here unchanged.

### The state, the byte form and the report

**FR-22.** Everything these arms write is canonical simulation state: it is in the byte form, it is
in the digest, and a world marshalled and read back is the world that was marshalled. The byte
form's version becomes **50**.

**FR-23.** The compile-time support tables gain all eight operations, so the census a build reports
before a tick has run and the arms it actually dispatches cannot come to disagree.

## AC — acceptance criteria

- **AC-1.** Instant 7 over a world whose player 2 is at the default leaves player 2 at 0 and every
  other slot at 2.
- **AC-2.** A group whose owner is at mode 0 receives a group move with **no** distribution and no
  rate term, where the same group at mode 2 and the same geometry receives both.
- **AC-3.** A group whose owner is at mode 1 receives the distribution **without** the spread test,
  over a geometry the spread test refuses.
- **AC-4.** Instant 20 over a unit holding two codes leaves the unit's container empty and a sack at
  the unit's cell holding both, in the unit's own order.
- **AC-5.** Instant 20 over a cell already holding a sack leaves **one** sack there, holding what it
  held followed by what the unit held.
- **AC-6.** Instant 25 over `(3, 20, 59, 54)` leaves one cell tail at key `(54 << 8) | 59` holding
  `{3, 20, 0, 54, 0, 54}`.
- **AC-7.** Instant 25 twice on one cell leaves **one** tail, holding the second node's bytes.
- **AC-8.** Instant 30 over a unit carrying effect 20 with `p0 = 20, p1 = 60000` leaves that effect
  at 60000 remaining; the same node over a unit carrying effect 19 leaves it untouched.
- **AC-9.** Instant 34 selector 6 writes health, selector 15 defence, selector 16 absorption, and
  selector 7 writes nothing at all.
- **AC-10.** Instant 34 selector 6 with `p1 = 0` leaves the unit not alive and its decay stage
  positive, and the world marshals and reads back.
- **AC-11.** Sub-command 10 over a group of three, one of which is the named unit, leaves the named
  unit acquiring in place and the other two holding an attack order on it.
- **AC-12.** Sub-command 10 over a member the preference matrix vetoes against the named unit leaves
  that member acquiring in place and holding no attack order, with its siblings engaged.
- **AC-13.** Sub-command 11 over a group of three leaves the named unit acquiring in place and the
  other two in the defend state, each holding the named unit as escort target and the node's range.
- **AC-14.** Sub-command 15 does the same with the follow state.
- **AC-15.** Sub-commands 11 and 15 with a range value of 0 leave every escorting member at range 3.
- **AC-16.** A world carrying every kind of state this story adds marshals and reads back equal, and
  two worlds differing only in one of those values have different digests.
- **AC-17.** The compile-time report names none of the eight as unsupported, and the campaign census
  falls by 34 nodes on each preserved root.

## P — properties

- **P-1.** No arm here reads a map, a clock or a float. Two worlds stepped from equal states through
  the same nodes are equal.
- **P-2.** Every unit reference is resolved **through the world by id**, so which entities an arm
  writes does not depend on the order the world holds its entities in.
- **P-3.** The eight arms are dispatched from the same two tables that answer the compile-time
  report, so no arm can be reachable at runtime and reported unsupported, or the reverse.
- **P-4.** Every refusal leaves the world byte-identical to the world the arm was handed.
- **P-5.** The cell tails are kept in one canonical order, so the byte form is a function of the
  logical world and not of the order the writes happened in.

## SC — scope cuts

- **SC-1.** The **per-tick behaviour** of the three new actor states is not built. A member left in
  defend, follow or acquire reaches no arm in the actor pass and is left in every field, which is
  that pass's own stated treatment of a state it has no case for. What is cut is the closing
  approach, the cover engagement fought on the protected unit's behalf, and the crowding check —
  three behaviours under a different contract from this story's setters. Nothing this story writes
  is removed by the cut: the state, the escort target and the escort range are all written, carried
  and hashed, so the arm that reads them is additive.
- **SC-2.** This build holds **one** attached effect per entity and not a list, which is 0154's own
  decision and not this story's. Instant 30's walk therefore reduces to one comparison. A unit
  carrying two effects of different spells cannot be built here, so no shipped node can distinguish
  the two readings.
- **SC-3.** Only the six-byte tail of the cell record is modelled. The original's rejection of a
  cell whose dynamic-plane bit 0 is set, and its marking of a new record for recomputation, have no
  counterpart in this tree — there is no dynamic plane and nothing to recompute — and neither is
  built.
- **SC-4.** Sub-command 10's veto reads the **ordinary** target scorer. The stand-ground variant's
  extra refusal is not consulted, because the decoded arm calls the ordinary routine.
- **SC-5.** Nothing here sends a notification. Instant 34's tail call and instant 20's sack
  announcement write no simulation state, and this build has no packet to send.
