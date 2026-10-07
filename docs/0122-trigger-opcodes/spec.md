# Spec — 0122-trigger-opcodes

Four mission-script arms this build could not evaluate, two arms that are dead in the thing being
reconstructed and are now dead here too, and the one record change all four need.

## The contract

### The player reference on a compiled check

**FR-1.** A compiled check carries **two player references**, each a roster slot with a presence
flag of its own: the first player a node names, and the second. They are **carried, not
resolved** — a roster slot names exactly one id space, so there is nothing to look one up in and
nothing that can fail — and they do **not** join the plain parameters, because those are packed in
encounter order and admitting a reference into that packing would move the parameters of every
other node that carries one, including arms this build does not evaluate.

Each needs its own flag rather than a reserved value: a node that named no player and a node that
named slot zero are two different things, and the arms answer them differently.

**FR-2.** The binder fills both from the node's own player parameters in encounter order: the
first such parameter fills the first slot, the second fills the second, and a third is dropped.
A node naming no player fills neither.

**FR-3.** The byte form carries both, and its version moves. A form written before this story
says nothing about a player reference, and "no player" is not a gap a reader may fill — it is the
claim that no check in that program measures anything about a player, which a program containing
one contradicts exactly. A decoder that supplied absent references would hand back a world whose
checks measure less than the world the bytes were cut from, silently, and whose digest then means
something else.

### The four arms

**FR-4 — check 8, how many units a player has.** The register takes the number of the named
player's **living** entities. A check naming no player measures nothing and writes no register.
A player owning no living entity answers **zero**, which is an answer and not a silence.

**FR-5 — check 15, how far the nearest of a player's units is.** The register takes the
**minimum**, over the named player's living entities, of the distance from that entity to the
authored cell (p0, p1), and **0xff when the player has none**. The distance is the one every
other arm of this file measures — the seam scriptDistance already is — and the result passes
through the same byte mask its four sibling arms use. A check naming no player writes no
register, which is not the same as the 0xff a named player with no units answers.

**FR-6 — check 10, what one player thinks of another.** The register takes
`relation[first][second] & 3`: the low two bits of the relation cell, the first player naming the
row and the second the column. A check naming fewer than two players writes no register. A slot
the matrix does not hold reads as zero, which is the relation's own existing rule and not a new
one.

**FR-7 — instant 10, change what one player thinks of another.** The relation cell at
`[p0][p1]` becomes `(cell &^ 3) + p2`, where the three values are plain parameters. Five
properties, and none of them is incidental:

1. **One direction.** The mirror cell is not touched. A mutual change is two nodes, and a map
   that wants one authors both.
2. **Bits 2 to 7 survive.** It is a read-modify-write and not an assignment.
3. **The parameter is added, not or-ed.** A `p2` of 4 or more carries into the surviving bits,
   and the sum truncates in the byte. Nothing clamps `p2` anywhere.
4. **The lock bit is not respected — it is cleared.** Combat's own hostility flip declines a
   locked pair; this arm has no such test, so a script can un-ally a pair the map declared
   permanently allied.
5. **It is bare.** No notification, no re-scan, no order invalidated. Whatever was decided under
   the old relation stands until it is next decided.

An instant naming a slot the matrix does not hold changes nothing.

### The two dead arms

**FR-8.** Check opcodes **11 and 13** are **dead arms**, and they are reproduced as dead rather
than treated as unimplemented. Three consequences, and they are the whole of the difference:

- the arm is dispatched and **writes no register**, leaving the slot at whatever it last held;
- the arm is **not reported as unsupported**, because it is not missing;
- **a trigger reading such a slot is still evaluated** — it is *not* inert.

That last is the point. Treating a dead arm as unimplemented poisons its register and silences
every trigger that reads it, which is a different, entirely plausible mission in which a whole
authored chain never fires.

### What stays true

**FR-9.** Everything this file already promises about loudness still holds for every arm outside
FR-4..FR-8. An arm this build does not implement writes no register, is named before the first
tick, and inerts each trigger reading it. "Not implemented" and "evaluated false" remain
distinguishable from outside without running anything.

## Divergences

**D-1 — the byte-form version number was chosen without an allocation.** The version this story
takes was picked in the lane, not handed to it, because the lane had no channel back to the seat
that allocates them. Three later numbers were known to be out to other lanes and this one is
above all of them. **It is written in exactly one place and renumbering it is a one-line edit.**

**D-2 — the count of living units is derived, not read.** The arms behind FR-4 and FR-5 contain
no health test in the thing being reconstructed; they are living-only because a death removes an
actor from its group earlier in the same tick than any script pass can observe. This build has no
group object to remove anything from, so it applies the death predicate directly. The set is the
same and the route to it is not. This is the reasoning the group count already uses, now stated
where both arms can see it.

**D-3 — a group has no owner here.** The arms behind FR-4 and FR-5 walk *every group of a player
and every member of every group*. This build's entities carry an owner and a group word
independently, and nothing owns a group, so the two arms walk the entities the player owns. On
any world this tree can build the two sets are identical; on a world where a player owned a group
containing an entity owned by somebody else they would not be, and no such world is
representable.

**D-4 — the diplomacy write's carry is reproduced and unexercised.** Property 3 of FR-7 is
built exactly, including the truncation. No shipped map authors an addend above 2, so nothing
that ships reaches it.

**D-5 — thirteen further arms are refused for want of a decoded meaning**, and are enumerated
with their node counts in the last section. Each writes no register, is named before the first
tick, and inerts its readers, exactly as FR-9 requires.

## Acceptance

**AC-1.** A world whose script names check 8 over a player with three living entities and one
felled one writes 3; over a player with none, 0; over no player at all, nothing — and the trigger
reading that register is inert.

**AC-2.** A world whose script names check 15 writes the least distance from the authored cell to
any of that player's living entities, ignores the player's dead, and writes 0xff when the
player has no living entity. A check naming no player writes no register.

**AC-3.** A world whose script names check 10 writes the low two bits of the relation cell and
drops every higher bit; naming fewer than two players it writes nothing.

**AC-4.** A world whose script fires instant 10 leaves the mirror cell untouched, leaves bits 2
to 7 of the written cell untouched, adds rather than replaces, and overwrites a locked pair.

**AC-5.** A trigger whose condition names a check of opcode 11 or 13 is **not** inert, that check
writes no register, and neither opcode appears in the unsupported report.

**AC-6.** A world carrying a check with either player reference set round-trips through the byte
form unchanged, and a form at the previous version is refused with a message naming both
versions.

**AC-7.** `go build`, `go vet`, `gofmt` and `go test -count=1 ./...` are green with no game
installed, and no test opens a map file.

## Derived properties

**P-1.** Adding four check arms arms triggers rather than merely adding behaviour: every trigger
whose conditions read only implemented registers stops being inert. Nothing else in this story
changes which triggers are inert.

**P-2.** `scriptDistance` gains its fifth caller and stays the only place a distance is measured
in this file.

**P-3.** The relation gains its second runtime writer. The first declines a locked or already
hostile pair; this one declines nothing. Both go through the one place that computes the stride.

**P-4.** Nothing here reads a float, a clock, an environment or a random source, and nothing
leaves `pkg/sim` except the binder's two new assignments.

## What is not built

Named so that the next story starts from a number. Node counts are the shipped EN corpus.

| Arm | Nodes | Blocked on |
|---|---|---|
| Instants 16, 17, 18, 32, 33 — a unit off the map and back | 81 | An **authored decision** about what an off-map unit is: whether it ticks, can be struck, holds its cell, or is still counted by check 15 and the group count. The removal itself is decoded; those four are not |
| Instant 21 — the spawn | 35 | The six-argument order is not interpreted, and `pkg/sim` holds no unit template table |
| Instants 12, 13, 20, 28 — loot and equipment | 56 | A per-actor container that resolves a named item |
| Checks 12, 16, 17 — item questions | 30 | The same, plus an item reference the binder currently drops |
| Instant 29 | 23 | The meaning of the field it writes |
| Instant 24 | 20 | The callee |
| Sub-commands 11 and 15 — Defend and Follow | 16 | Two per-actor states this build's actor machine does not have, each carrying a subject and a range that would widen the entity record again. **The law is complete; the machine is not** |
| Check 21 | 16 | The meaning of the field it reads, and a structure reference the binder drops |
| Check 4 | 10 | The meaning of the field |
| Sub-command 10 | 5 | The per-member scorer it calls |
| Instant 34 | 5 | Three field meanings |
| Instants 7, 23, 25; check 9 | 10 | Callees or field meanings |

**Total left BLOCKING: 307 nodes** — 56 condition nodes and 251 action nodes, of which the
off-map family at 81 and the spawn at 35 are the largest single group.
