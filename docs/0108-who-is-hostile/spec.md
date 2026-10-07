# Spec — a blow decides who is hostile, not only the map

## Terms

**The relation** is the world's matrix of roster slots, indexed `[me][him]`, one byte a cell. It
is directional: a slot may treat another as an enemy without the courtesy being returned.

**Bit 0** of a cell means "the row's slot treats the column's slot as an enemy", and it is the
only bit any rule of this build reads. **Bit 1** means the pair is allied and stays that way.
Bits above 1 are carried and never interpreted.

**A blow connects** when the swing has passed this build's to-hit gate — before the victim's
absorption is subtracted, and whatever health the victim has left.

**Flippable** describes a cell whose low two bits are both clear.

## Why

Until now the relation was authored once, by the map, and never written again for the life of a
mission. That is half of it. In the game a blow between two slots that were not enemies makes them
enemies, both ways, permanently — and the same rule is what gives bit 1 its meaning, because a
lock that nothing can act against is indistinguishable from a cell that is merely not hostile.

The consequence a player can watch: strike a neutral villager and his whole faction turns on you.
Over the campaign's own maps, 51 of 198 ordered off-diagonal pairs are flippable and 3 are locked,
so this is content both roots ship rather than a capability held in reserve.

## Scope

**In:** one new writer of the relation, inside the simulation, fired where a blow is resolved.

**Out, and each named rather than forgotten:**

- The victim's remembered attacker and the twenty ticks the group keeps it. It needs per-entity
  state, a byte-form version, and a notion of which slots are human participants; this world has
  none of the three.
- The turn-to-face flag the same hook sets. Its only consumer is an arm this build does not run.
- The script's own one-directional diplomacy write. No map either installed root can read authors
  it, so building it would change nothing shipped; it is a customisation seam, not fidelity.
- The mission-join and session-command writers. Multiplayer machinery with no consumer here.
- Any notification, re-scan or order invalidation on a change.
- The map loader's load-time store, which is unchanged in every respect.

## The contract

### When the flip runs

**FR-1** — When a blow **connects**, the world attempts a flip between the attacker's slot and the
victim's slot. The attempt is made before absorption is subtracted and before any test on what
damage survives it, so a blow a victim's armour eats entirely still flips.

**FR-2** — A swing that does **not** connect — out of reach, no to-hit — attempts nothing. So does
a blow where **either** party names no roster slot.

**FR-3** — A blow lands on a victim below 1 health exactly as on a living one, and flips the same.

### What a flip does

**FR-4** — The attempt is made **twice, once per direction**: attacker-row/victim-column, then
victim-row/attacker-column. Each direction is decided on **its own cell alone** and neither reads
the other, so a pair may flip one way and not the other.

**FR-5** — Per direction: a **flippable** cell gains bit 0 and nothing else. Any other cell — bit
0 already set, or bit 1 set — is left byte-for-byte alone. No bit above bit 0 is ever written and
no bit is ever cleared.

**FR-6** — **No rule special-cases two entities of the same slot.** Both directions then name one
cell, the matrix's diagonal, and on any world whose relation a map authored that cell is 2, so
FR-5's gate declines it exactly as it declines a locked pair. On a world named **no relation at
all** every cell including the diagonal is zero, and such a blow does turn the slot hostile to
itself — which is what the game would do given the same matrix, and the reason a loader forces the
diagonal rather than a rule here doing it.

**FR-7** — A slot the matrix does not hold is a no-op, not a refusal — the rule the loader's own
store already follows.

### What a flip does not do

**FR-8** — A flip notifies nothing, re-issues nothing and cancels nothing on the tick it happens.
An engagement already under way keeps advancing; a group that becomes hostile acquires at its next
decision, because that decision rebuilds its candidate list from the cell as it then stands.

**FR-9** — No field is added to any record and the byte form's **layout and version are unchanged**.
The relation is already the form's last block, so a flipped relation is a different world in the
bytes and in the digest with nothing new to encode and no version to spend.

## Acceptance

**AC-1** — A census, through this tree's own reader, of every campaign map **both** installed roots
hold: ordered off-diagonal cells split into flippable, already hostile, and locked. The two roots
must return the same three numbers.

**AC-2** — Two entities on slots whose pair is flippable in both directions, one striking the
other: both cells carry bit 0 after the blow, every other byte of the matrix is unchanged, and the
struck slot's group then acquires the striker where before it saw nothing.

**AC-3** — A **locked** pair, struck in both directions in turn: neither cell changes at all, and
neither slot ever acquires the other.

**AC-4** — An asymmetric pair — one direction hostile, the other flippable — struck: the flippable
direction gains bit 0 and the hostile direction is byte-identical, whichever of the two swung.

**AC-5** — Nothing flips on a miss, on a swing out of reach, or when either party's slot is 0.
Two entities of one slot exchanging blows on a **loaded** map leave the diagonal at 2, and on a
world named no relation the same blow sets it — both arms of FR-6, measured rather than argued.

**AC-6** — A blow whose damage is entirely absorbed flips (FR-1), and so does a blow on a victim
already below 1 health (FR-3).

**AC-7** — The byte-form version and every record length are the same integers after this story as
before, a world encoded before it still decodes, and the digest of a world **does** move across a
connecting blow on a flippable pair.

**AC-8** — The tenth mission, driven by its own two waypoints, on **both** roots, before and after.
Nothing here is tuned toward it; the stage records the two traces whichever way they come out.

## Properties

**P-1** — The flip is a pure function of the world: no draw, no clock, no float, no iteration whose
order is not the entity slice's. It consumes no position in the random stream, so no landed test's
draws move because it exists.

**P-2** — It is **monotone**. No rule in this build clears bit 0 or writes any other bit, so over a
mission hostility only ever increases, and a world's relation is a record of what has happened.

**P-3** — It is **idempotent**: the same blow applied twice leaves the matrix exactly as one blow
did, and so does any later blow on a pair already hostile.

**P-4** — One place still computes the cell offset. The flip reaches the matrix through the
relation type's own accessors and indexes no storage itself, so a transposed pair remains
impossible to write by accident.

**P-5** — The relation stays the only thing a flip touches. No entity field, no group state, no
order and no target is written by it, so the change is visible **only** through what a later
decision reads.
