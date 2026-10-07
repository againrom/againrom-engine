# 0165-script-cast — the mission script casts a spell and re-times an area effect

This is the contract. It is self-contained: every fact it needs is stated here, and the evidence
behind each fact is in `provenance.md`.

## Problem

Three instant opcodes of the mission script are unimplemented. Together they are the largest decoded
block left in the campaign's script gap after 0164, at 78 nodes per preserved root:

| opcode | what it does | nodes per root | maps |
|---|---|---|---|
| 21 | cast a spell from a temporary caster at a **cell** | 35 | 80, 90, 91, 101, 131, 140 |
| 24 | cast a spell from a temporary caster at a **unit** | 20 | 90, 151 |
| 29 | set the remaining lifetime of an area effect standing on a cell | 23 | 91, 101, 131 |

The three are one contract. Instants 21 and 24 share a constructor. Every shipped instant-29 node
names a cell that a shipped instant-21 node cast at, on the same map, and the spell id it matches on
is the spell that cast was for. Instant 29 has nothing to write to unless instant 21 has run.

## What is being built

A script cast is a **temporary caster**: a record the instant creates, which is not an entity, holds
no runtime id, and resolves on a later tick. What its cast lands is decided by the spell row's own
`Distribution system` column, not by which instant created it:

- a row whose column is not 1 lands an **area effect** on a cell, with a lifetime in ticks;
- a row whose column is 1 is a **point effect** and needs a target unit.

Instant 29 then writes a new remaining lifetime onto a matching area effect.

## Functional requirements

**FR-1 — instant 21 creates a script cast at a cell.** The node's six plain parameters are
`(fromX, fromY, toX, toY, spell, power)`. The record carries the source cell as two bytes, the
destination cell as two bytes, the spell id as one byte and the power as a 16-bit word. A power of 0
is stored as **99**; any other authored value is stored as it stands. All 70 shipped instant-21 nodes
across both roots author power 0, so the shipped campaign runs entirely on the 99 substitution.

**FR-2 — instant 24 creates a script cast at a unit.** The node's four plain parameters are
`(fromX, fromY, spell, power)` and its unit reference is the target; the reference is resolved by the
binder before the record reaches this package. The same source cell, spell byte, power word and
power-0 substitution as FR-1. A node whose unit reference did not resolve creates no cast, which is
the treatment every other arm of this build gives an unresolved reference.

**FR-3 — a script cast resolves on a later tick, not in the instant.** The instant appends the
record to the world's own pending list. The list is walked once per tick, at the head of the tick,
before any unbidden cast. A cast authored by the script pass of tick T therefore lands no earlier
than tick T+1. Each record is resolved once and then removed, whether or not anything landed.

**FR-4 — the landing forks on the spell row's `Distribution system` column.** A cast whose spell id
names no row of the world's spell table lands nothing. Otherwise:

- **column ≠ 1 (area).** An area effect is placed on the destination cell — the record's own
  destination for FR-1, the target unit's current cell for FR-2 — with a remaining lifetime of
  `(AreaEffectDuration << 4) + (power << 4)/10` ticks, truncated toward zero, where
  `AreaEffectDuration` is the row's own column and the second term is added only when the power is
  non-zero. A cast from FR-2 whose target the world no longer holds, or which is not alive, lands
  nothing.
- **column = 1 (point).** A cast created by FR-1 lands nothing: a point effect requires a target unit
  and a cell is not one. A cast created by FR-2 applies the row to the target through this build's
  existing unit-spell arms at the cast's own power: a damaging row rolls its damage, a restorative
  row heals, and both mark the target's spell-effect mark. A row that is neither lands nothing —
  see SC-1.

**FR-5 — an area effect is held on a cell and counts down.** An area effect carries a 16-bit cell
key, a spell id and a remaining lifetime in ticks. The key is `(uint16(y) << 8) + uint16(x)`,
computed in 16-bit arithmetic, so an x above 255 carries into the y byte. At most **six** effects
stand on one cell; a seventh is dropped. Every effect's lifetime falls by one at the head of each
tick, before anything can set one, and an effect reaching zero is removed. Nothing else in this
build reads an area effect — see SC-2.

**FR-6 — instant 29 writes a remaining lifetime.** The node's four plain parameters are
`(x, y, spell, duration)`. The arm computes the cell key exactly as FR-5 states, and for **every**
effect on that cell whose spell id equals the **full 32-bit** third parameter it writes the fourth
parameter, truncated to 16 bits, as the new remaining lifetime. It does not stop at the first match.
A key no effect stands on, and a cell whose effects all fail the comparison, are no-ops. Because the
comparison is against the whole parameter and an effect's id is a byte, an authored third parameter
of 256 or more matches nothing.

**FR-7 — the spell table carries the two columns FR-4 and FR-5 need.** A spell row gains an **area**
flag, true when its `Distribution system` column is not 1, and its **area duration** column. Both are
read from the installed `Data.bin` `Spells` collection by parameter slot, at slots 8 and 11, and both
reach the world's own spell table.

**FR-8 — the byte form carries both new kinds of state.** The pending script casts and the standing
area effects enter the canonical byte form and therefore the digest. The form's version becomes
**49**. A form is refused on each of six states no tick can leave: a stored cast naming spell id 0; a
stored cast with a power of 0; a stored cast whose target flag byte is neither 0 nor 1; an area
effect carrying spell id 0; effects that are not in ascending cell-key order; and more than six
effects on one key.

## Acceptance criteria

**AC-1** A world stepped after an instant-21 node fires holds one pending script cast at the end of
that tick, carrying the node's own source and destination bytes, spell byte and a power of 99 for an
authored 0.

**AC-2** A world stepped after an instant-24 node fires holds one pending script cast naming the
resolved target entity; a node whose reference did not resolve leaves the pending list empty.

**AC-3** A pending cast is gone from the world one tick after it was created, and the area effect it
placed stands on the destination cell with the FR-4 lifetime.

**AC-4** An instant-21 cast of a `Distribution system` = 1 row places no area effect and changes no
entity.

**AC-5** An instant-24 cast of a damaging row lowers the target's health; of a restorative row raises
it; of a row that is neither leaves every entity byte-identical.

**AC-6** An area effect's remaining lifetime falls by exactly one per tick and the effect is gone
from the world on the tick its lifetime would reach zero. An effect instant 29 wrote to zero is gone
on the next tick and does not wrap to 65535.

**AC-7** An instant-29 node writes its duration onto **every** matching effect on the keyed cell and
onto no effect on any other cell; a third parameter of 256 or more matches nothing.

**AC-8** Instant 29's key arithmetic is 16-bit: an authored x of 300 with a y of 0 addresses the same
cell as an x of 44 with a y of 1.

**AC-9** A world holding pending casts and area effects round-trips through `MarshalBinary` and
`UnmarshalBinary` unchanged and hashes equal; a version-48 buffer is refused by version.

**AC-10** Each of the six refusals FR-8 names is refused by name, and a buffer holding seven effects
on one key is refused.

**AC-11** `Script.Unsupported` names no instant of opcode 21, 24 or 29 for any of the 28 campaign
maps of either preserved root.

## Properties

**P-1** An unobserved step and an observed step of a world holding pending casts or area effects
produce the same bytes. Neither new kind of state is reported through `CastEvent`.

**P-2** A cast that lands nothing leaves the world byte-identical to one in which the instant never
ran, except for the pending record's own creation and removal.

**P-3** Nothing in this story reads an entity's owner, and nothing branches on which roster slot a
script cast's target belongs to.

## Scope cuts

**SC-1 — the per-spell point-effect arms are not built.** `MAGIC-CEIL-013` enumerates thirteen
distinct expressions the spell power feeds, one or more per spell, and this build implements the two
that are already here: the damage roll and the heal. The six spells the shipped instant-24 nodes
name — Stone Curse, Bless and the four Protections — are all point effects, and none of them is
damaging or restorative. **Every one of the 20 shipped instant-24 nodes therefore constructs a cast,
resolves it at its target and lands no state.** The instant is implemented, its effect is not, and
the census counter falls for a node whose visible outcome is unchanged. This is stated again in
`verification.md`.

**SC-2 — an area effect does nothing to units standing in it.** It is placed, it counts down and it
is removed. It deals no damage, blocks no movement and draws no picture. The burst picture and the
per-cell pattern are `MAGIC-BURST-031`'s ground and are not this story.

**SC-3 — the cast-time countdown is not modelled.** The decoded state machine counts down the
spell's own cast time between success and applying the effect. No claim in this story's pin names the
column that time comes from, so the pending cast resolves on the first tick that reaches it, which is
the machine's own minimum of one tick. The number of ticks a real cast waits is not claimed.

**SC-4 — a cast is not retried.** The decoded machine retries a cast that returns zero on a later
actor tick. What makes it return zero is not decoded here, so a cast that finds no target lands
nothing and is removed rather than held.

**SC-5 — no area effect stacking rule is chosen.** Six slots per cell and a dropped seventh is the
decoded structure. Whether a second effect of the same spell on the same cell replaces the first is
not decided by any shipped map: every shipped trigger naming these opcodes is fire-once, and no
shipped cell receives the same spell twice.
