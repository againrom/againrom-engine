# Spec — a placed structure reaches the block plane

## Problem and current behaviour

`pkg/mapload` derives a map's block plane from five arms — the impassable tile flag, the water range,
the mountain blend, a non-zero scenery byte and the eight-cell ring — unioned into two bits. Bit 0
closes a cell to a ground mover, bit 1 to an air mover, and a byte setting any of bits 2-7 is refused
by the world constructor. `pkg/formats/alm` decodes a map's placed structures into records carrying a
fixed-point anchor, a class key and, on one key, an eight-byte extension. `pkg/formats/databin`
parses the definition table and hands each entry's parameters back verbatim. **Nothing joins them**,
so a placed structure reaches the plane in no way at all: a unit walks through a church, and a map
whose only crossing is a bridge has no crossing.

A structure is not a sixth arm: it is applied **after** them, per footprint cell, and may *subtract*
a block the arms put there — which is what lets a bridge cross water and a doorway open a wall.

## The footprint contract

A **placement** is one type-4 record: an anchor `(X, Y)` in 1/256 of a cell, a class key `kind`, and
an eight-byte extension present exactly when `kind` is `0x21`. Its **anchor cell** is
`(X >> 8, Y >> 8)`, and the plane is indexed `[0, Width) x [0, Height)` row-major from `(0, 0)`.

A **definition entry** is the `kind`-th entry of the table's `Buildings` collection, which is
one-based — entry 0 is allocated and never written — so a placement resolves an entry when
`1 <= kind && kind < Len()`. `kind` is taken as its **low byte**, so a key above 255 resolves the
entry that byte names.

Four of the entry's parameters are read, at positions 0, 1, 4 and 5 — the collection's own columns
`sizeX`, `sizeY`, `Passability`, `BuildingPresent`: the footprint's width and height in cells, the
**blocking set** and the **attach set**. `Passability`'s shipped title is the **inverse of its
content** — a set bit makes a cell impassable (FR-4) — so this contract says *blocking* everywhere,
and a builder who takes that title at its word inverts the whole pass. Each extent is the **low
byte** of its parameter, so neither exceeds 255. Both sets are **unsigned 32-bit**, bit `i` being
the cell at index `i`. The **bit index** of the cell at offset `(dx, dy)`, `dx` in `[0, sizeX)` and
`dy` in `[0, sizeY)`, is `(dy * sizeX + dx) mod 32`, and the anchor cell is the footprint's
**top-left**, so the rectangle runs right and down. A footprint of more than 32 cells therefore
**aliases** — cell 32 shares cell 0's bit, in both sets alike — and that is the rule, not a defect.

## Functional requirements

- **FR-1** A pure function MUST resolve a map's placements against a definition table into
  footprints, one per placement resolving an entry, each carrying its anchor cell, its two extents
  and its two sets, per *The footprint contract*. A placement whose key resolves no entry, and one
  whose entry carries fewer parameters than the contract reads, MUST contribute nothing and MUST NOT
  fail the resolution.
- **FR-2** An **extension placement** — one whose whole key, not its low byte, is `0x21` — MUST still
  resolve an entry as FR-1 requires. Two of its eight bytes carry extents: the **low byte of the
  first four** is a width and of the **second four** a height, and the remaining six decide nothing.
  Where those two **sum above zero** the placement MUST take both extents from them rather than from
  the entry, with an **all-set attach set and an all-clear blocking set**, whatever the entry says;
  where both are zero it MUST keep the entry's own extents and both its sets. A zero extent on
  either axis, from either source, MUST yield no footprint at all.
- **FR-3** A footprint MUST attach exactly the cells whose **attach-set** bit is set and which lie
  inside the map extent, walking the rectangle row by row from the anchor cell. A cell already
  attached by an earlier placement MUST be refused, and refusal MUST abandon the **remainder** of
  that footprint, leaving the cells already attached attached. Placements MUST be walked in the order
  the map records them.
- **FR-4** For each attached cell the derivation MUST read the **blocking-set** bit at that cell's
  index and, where that bit is **set**, close the cell to a ground mover; where it is **clear**, the
  cell MUST be **opened** — overriding whatever the five arms said. The air bit MUST be left exactly
  as the arms left it, and no bit above bit 1 MUST be set.
- **FR-5** The plane MUST be the five arms unioned and then this pass applied, in that order and
  never interleaved. It MUST stay one byte per in-bounds cell, row-major, and remain a function of
  the map and the table alone.
- **FR-6** The definition table MUST be an **argument** to world building, never read from an archive
  at this tier. Built with no table, a world and its plane MUST be byte-identical to what that map
  produced before this story.
- **FR-7** A census instrument MUST report two levels of count, kept apart and each level's counters
  disjoint. **Per placement:** resolved, and one counter per reason a placement was skipped whole.
  **Per cell:** attached, closed, opened, dropped for lying outside the extent, refused as already
  attached, and **abandoned** — attach-named but never walked, a refusal having ended that footprint.
  It MUST be reachable from the tool that reads an archive's table.
- **FR-8** The resolution, the walk, the pass and the census MUST be exercisable headlessly against
  synthetic maps and tables, with no game install present.
- **FR-9** A developer instrument MUST tint, over a drawn map, exactly the cells the derived plane
  closes to a **ground** mover, taking its table from the same install it takes the map from, so a
  bridge reads as an open line across a closed river. It MUST be off by default, leaving an
  unflagged run's image and report what they were, and MUST NOT reach the game's own screen.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | `Buildings` with entry 0 unwritten and entries written through 33: entry 1 full, entry 2 carrying five parameters, entry 3 with `sizeX` above 255; keys 0, 1, 2, 3, 34, `0x121` | resolved | 1 yields a footprint, 3 one of width `sizeX & 0xFF`, `0x121` the entry its low byte names and **no** extension; 0 and 34 count unresolved, 2 counts short; the table is unmutated |
| AC-2 | unit | an entry `3 x 2`, attach set naming five of its six cells, blocking set naming two of those five | walked from an anchor cell | exactly five attach, row-major from the anchor running right and down; the two the blocking set names take one outcome, the other three the other, per FR-4; the sixth cell's byte is untouched and uncounted |
| AC-3 | unit | an entry `11 x 4` whose sets each make bit 0 and an unmasked shift by 32 disagree | walked | cell 32 attaches, and closes or opens, exactly as bit 0 says — an index taken without the modulus gives the other |
| AC-4 | unit | an extension placement whose eight bytes give extents differing from its entry's; one resolving no entry; one with a zero extent; one whose two extent bytes are both zero | resolved and walked | the record's extents win and every cell attaches, each taking the outcome an all-clear blocking set gives; the next two yield nothing; the last is its entry's own rectangle under its entry's own two sets |
| AC-5 | unit | two placements overlapping on the second attach-named cell of the later one's walk | walked in record order | the earlier keeps every cell, the later the cells before the clash and none after; the clash counts one refusal and the attach-named cells behind it count abandoned; reversing the order reverses which is which |
| AC-6 | unit | a structure on water, mountain, scenery and the border, its blocking set naming some of those cells and not others | the plane is derived | a cell the pass opens is open whichever arm had closed it; a cell it closes is closed; every air bit is the arms' own; no byte sets a bit above bit 1 |
| AC-7 | unit | a footprint running past the last column and row, and one anchored wholly outside | the plane is derived | in-extent cells behave as AC-2 states, out-of-extent cells are dropped and counted, no cell of the next row is touched, the outside placement changes no byte |
| AC-8 | unit | one map built with no table, a nil table, a table whose `Buildings` is absent; and one map with placements, censused | worlds built, planes derived | all three planes are byte-identical to that map's pre-story plane, as are the worlds' byte forms and digests; the censused run reports both count levels with no install |
| AC-9 | corpus | every shipped map and the definition table, from a lawful install | censused through the tool | every map censuses without error, and both levels of count are recorded so they can be set against independently published figures for the same pass, a disagreement reported rather than reconciled |
| AC-10 | unit | two neighbouring cells the five arms leave **open**, both attach-named by one footprint, its blocking set naming the first cell's index and not the second's | the plane is derived | the **named** cell's byte has bit 0 **set** and the **unnamed** cell's has it **clear**; a derivation answering the reverse on this one pair fails |
| AC-11 | unit | a plane closing some cells to a ground mover, one to an air mover alone, and leaving the rest open | the tint's cells are derived | exactly the ground-closed cells come back, row-major, and the air-only cell is not among them |

Every error case above is a **skip**, never a failure.

## Derived properties

- **P-1** (invariant) Resolution is pure and total: any map and table yield a footprint list, neither
  is mutated, and one key against one table always yields one entry.
- **P-2** (invariant) The plane is a function of the map and the table alone; `Width * Height` bytes
  for a positive extent, none otherwise.
- **P-3** (negative-invariant) No derived byte sets a bit above bit 1, at any table, so the world
  constructor's reserved-bit refusal cannot fire on such a plane.
- **P-4** (negative-invariant) The derived **air** bits are exactly those the five arms alone produce
  — a structure never sets and never clears one, at any table.
- **P-5** (negative-invariant) With no table, a nil-collection table, or no resolvable placement, the
  plane and the world byte form are the pre-story ones, byte for byte.
- **P-6** (completeness) Every placement is accounted for once: resolved plus the per-placement skips
  equal the map's placement count.
- **P-7** (completeness) Every cell a footprint's attach set names is accounted for once: attached
  plus dropped plus refused plus abandoned equal it, and closed plus opened equal attached.
- **P-8** (negative-invariant) No cell outside the map extent is read or written, at any anchor,
  extent or table, and the walk terminates for every entry a table can hold.

## I/O examples

```text
classdump -databin <world.res> <map.alm>
structures: 137 resolved, 0 unresolved, 0 short, 0 zero-extent
cells: 812 attached (796 closed, 16 opened), 3 dropped, 2 refused, 5 abandoned
```

## Constraints and alternatives

| Choice (all selected; *disclosed* = a named divergence) | Observable trade-off |
|---|---|
| The plane stays **two bits**; the original's third bit, marking a cell as occupied by an object or building, is not carried | A mover stopped by buildings but not terrain cannot be expressed |
| The pass runs **once, at derivation**, the plane fixed at construction — *disclosed* | Nothing can be demolished; a demolition story adds the original's per-cell record, not a second derivation |

**Named seams (G2).** The **32-cell aliasing** (contract) and **one structure per cell** (FR-3) are
declined lifts: each would change behaviour on placements the original ships, so neither may be
lifted silently. The sets' 32-bit width is a data width — an extended form must be additional.

## Out of scope

- **Structure art** — the sprite, its anchor, its animation, its shadow, its ruin. The existing
  diagnostic marker is unchanged.
- The structure as a **simulation entity**: health, destructibility, ownership, selection, the shop
  branch's behaviour, the durability and scan values the entry carries.
- Runtime attach, detach and demolition; the per-cell record, its terrain baseline and its marker
  bit; save and load of any of it.
- The cost and height planes, and any change to routing, to the five arms, to ALM decoding, to table
  parsing or to class loading.
- Wiring the shipped **game** front-end to a definition table, for structures or for anything else.

## Verification mapping

Every unit criterion is CI-automatable headlessly against synthetic maps, tables and planes. AC-9
needs a lawful install; counts recorded, no game bytes committed.

Gate coverage: FR-1 → AC-1, P-1 · FR-2 → AC-4 · FR-3 → AC-2, AC-3, AC-5, AC-7, P-8 ·
FR-4 → AC-6, AC-10, P-3, P-4 · FR-5 → AC-6, P-2 · FR-6 → AC-8, P-5 · FR-7 → AC-8, AC-9, P-6, P-7 ·
FR-8 → every unit criterion, each run headlessly · FR-9 → AC-11.
