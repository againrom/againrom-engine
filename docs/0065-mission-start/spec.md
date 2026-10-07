# Spec — the campaign mission start

Intensity: **spec-anchored / static**. Terrain: **brownfield** in `pkg/mapload`, greenfield
elsewhere.

## Context

A campaign mission is named by a number. Today the tree can decode a map that is handed to it and
build a world from that map's own placements, and it can reach an archive's contents; nothing turns
a number into a mission, nothing puts the player's own units anywhere, and a placed unit that names
a scenario NPC resolves to nothing at all.

Three things this story is answerable for are not what a first reading of a map suggests. A campaign
map places **nobody** for the player: the player's units already exist and the load only positions
them, so where the party appears is not a placement record but one cell the map's script authorises.
The four paths a placement takes to a definition are **ordered and nested**, with the class key
deciding first, so a record can carry a definition id that is never read. And the map's contribution
to the start is a *list* of cells picked from at random, not a first entry.

## Functional requirements

**FR-1 — a mission number names a map.** There is one derivation from a campaign mission number to
the address its map is read at: the campaign container's own identity segment, the number in decimal,
and the map extension. A number that is not positive names no map and is refused before anything is
read.

**FR-2 — starting a mission yields a populated world.** One entry point takes the opened archives, a
mission number, a party and a difficulty, and answers the decoded map, a world built from it, and a
report of where the start put things. A failure to read or decode names the address it failed on.

**FR-3 — the placement arms are ordered, and the class key decides first.** A placement whose class
key is at or above 26 resolves against the Units collection on its two keys, whatever its flag word
and whatever its definition id hold. Below 26, and only below 26, three further arms are tried in
this order and no other: the NPC flag, then a definition id that is neither absent nor unwritten,
then the primary key as a type id. A placement that takes the NPC arm never reads its own definition
id. The band test reads the class key as the signed word the record stores; each search reads its
keys as their low bytes.

**FR-4 — the NPC arm reaches a definition.** The npc subscript is the record's secondary key, whole
and untruncated. It names a section of the campaign container's npc registry; that section's
definition id is searched down the Humans collection as a server id. A subscript naming no section,
a section carrying no definition id, and a definition id of exactly 26 each reach no entry — the last
because that value is a sentinel meaning *compose this from the section's own flags*, and what it
composes is not established.

**FR-5 — the drop cells come from the map's own script.** The map's trigger payload is read as three
counted arrays of fixed-size records, the first being the actions. Every action carrying the drop
opcode contributes one cell: the low byte of its first stored value as the column, the low byte of
its second as the row, in the array's own order. A payload the framing does not consume exactly, and
a map carrying no trigger record, contribute no cells at all — and are distinguishable, at the
report, from a map whose script authorises none.

**FR-6 — one cell is picked, and a degenerate cell falls back.** The start draws one cell uniformly
from the list FR-5 yields. When the list is empty, or when either coordinate of the drawn cell is
zero, the start instead draws each coordinate independently and uniformly from 30 to 100 inclusive.
The report says which of the two happened.

**FR-7 — the party stands at the drop cell.** The party's first member — the hero — stands on the
drop cell exactly. Each further member stands on the first cell of a fixed outward walk from the drop
cell that lies inside the map, that the map's own derived plane does not close to a ground mover, and
that no earlier member of the same start has taken. A member for which the walk finds no such cell
within its bound stands on the drop cell, and the report says how many did.

**FR-8 — the party is in the world.** Party members are entities of the built world, in party order,
**after** every entity the map's placements built, so a placement's entity id is unmoved. Each
carries the class key its caller supplied, the ground domain, and the same health and speed a
placement that resolves to no definition carries.

## Acceptance criteria

| | Given | When | Then |
|---|---|---|---|
| **AC-1** | mission number 10 | its address is derived | it is the campaign identity, `10`, and the map extension; 0 and a negative number are refused |
| **AC-2** | a lawful install's campaign and world archives | every shipped map's placements are resolved | the four arms count 6672 / 1405 / 15 / 2 over 8094 records, and `10.alm` counts 19 / 14 / 2 / 0 |
| **AC-3** | `10.alm`'s two script-named placements | they are resolved | both take the NPC arm, both reach a Humans entry, and neither is the entry its own definition id names |
| **AC-4** | every shipped map | its drop cells are read | each yields exactly one, and `10.alm`'s is column 17, row 66 |
| **AC-5** | a map whose script authorises no cell, and one whose drawn cell has a zero coordinate | a mission is started | each coordinate is between 30 and 100 inclusive, both bounds reachable, drawn independently |
| **AC-6** | a party of more than one on a map with blocked ground beside the drop cell | it is placed | the hero is on the drop cell, no two members share a cell, and no member *past the first* stands where the plane closes the ground |
| **AC-7** | a trigger payload the three-array framing does not tile exactly | the drop cells are read | none is returned, and the start reports the fallback rather than a cell |
| **AC-8** | one mission started twice, in two processes | the worlds are compared | the drop cell, every party cell and the world's digest agree |
| **AC-9** | a map with placements at or above the band whose records also carry a definition id | a world is built | those placements carry their Units entry's health, speed and domain, not the provisional pair |

## Properties

**P-1 — no clock, no environment, no process-global source.** Everything a start chooses follows from
the map, the party, the difficulty and one constant seed. The draws are consumed in a fixed order:
the pick first when there is a list, then the two fallback coordinates when one is needed.

**P-2 — a start reads nothing but the archives it is handed.** No path outside them is opened, and no
asset root is spelled in any shipped source.

**P-3 — no party member past the first stands on a closed cell or on another member's.** Total on any
map, any party size and any drop cell, the single exhaustion case of FR-7 excepted, which the report
counts. The first member is where FR-7 puts it, closed or not: a drop cell is the map author's
statement about where the mission begins, and a loader that moved it would be answering a different
question. *(P-3 as first written said "a party member", which contradicted FR-7's own sentence in the
same document; narrowed here to what FR-7 always said.)*

**P-4 — an arm is a function of one record and the band alone.** No arm consults the map, another
record, or the contents of a collection; which arm ran is answerable with no table present at all.

## Out of scope

- The script runtime — evaluating conditions, firing triggers, the win and lose counters. This story
  reads one action's stored values at load and interprets no opcode else.
- Combat, damage and death.
- A mission's dialogue and anything that shows it. `main.res::text/battle/m<n>/…` is a named seam.
- The per-player drop override, which needs a player object this tree does not have.
- Character generation, the party's own statistics, and any inventory.
- Saves, the town, and moving between missions.
- Any front-end door: nothing in `cmd/againrom` gains a way to start a mission.
- Validating a mission number against the campaign's declared mission count.
