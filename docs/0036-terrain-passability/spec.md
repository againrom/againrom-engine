# Spec — terrain passability: the block plane a map already describes

## Problem and current behaviour

`mapload.FromALM` hands the world constructor **no grid**, so every cell of a loaded map is passable:
units cross water, stand on mountain, walk through scenery, and the outer ring is as open as the
middle. Nothing here derives passability from a map — there is no classification to correct, only
one to write.

A world carries one grid byte per in-bounds cell, row-major over its `W x H` bounds: bit 0 blocks a
ground mover, bit 1 blocks an air mover, and bits 2-7 are reserved — a set one refused rather than
masked. The grid is canonical state, carried by the byte form at version 5 (its cell count in the
header, its cells straight after) and entering the digest; it is fixed when a world is built, and a
world built with none holds an all-zero grid of that size and those bytes. Bit 0 is read by routing
and by the route decoder. **Bit 1 is carried, hashed and read by nothing.**

A decoded map carries what a derivation needs: `Tiles`, one 16-bit word per cell; `Overlay`, one byte
per cell — the static-object placement layer, a full plane read or manufactured; and the extent.
Nothing else is consulted below.

## Functional requirements

- **FR-1 — one derivation, total and pure.** A world built from a decoded map MUST take a
  passability grid derived from it by the rules below: exactly `W*H` bytes, row-major from `(0,0)`,
  one per in-bounds cell — and no cells at all where either extent is not positive. The derivation
  MUST be a function of the extent, the tile plane and the overlay plane ALONE, so two maps agreeing
  on those three MUST yield equal grids however else they differ; it MUST be integer-only and read
  no clock, file or generator. And it MUST NOT fail: a plane shorter than the extent — an absent one
  included — MUST read as zero past its end, one longer MUST be ignored past the extent, and no map
  MUST produce an error, a panic or a grid of any other length.
- **FR-2 — three arms read the tile word.** For a cell's word `w`, let `i = w & 0x3ff`,
  `s = i & 0xf`, `b = (i >> 4) & 3` and `g = (i >> 6) & 0xf`. The cell MUST block ground when any of
  **(a)** bit 13 of `w` is set; **(b)** `i` lies in `[512, 768)` — a test on the index alone, never
  on what (c) would return; **(c)** `g == 7`, `s <= 13` and `level[b][s] >= 3` for the table below.
  Bits 10-12, 14 and 15 of `w` MUST reach none of the three, so two words differing only there MUST
  classify alike; and a `g` of 13 or more MUST NOT block through (c).

```
level[b][s]     s =  0  1  2  3  4  5  6  7  8  9 10 11 12 13
        b = 0        2  3  2  4  3  4  2  2  2  2  4  4  4  4
        b = 1        3  5  3  3  1  3  2  4  2  2  4  2  4  4
        b = 2        2  3  2  4  3  4  2  4  2  2  4  2  4  4
        b = 3        5  5  5  5  5  5  2  2  2  2  4  4  4  4
```

- **FR-3 — a placed scenery cell blocks ground.** A cell whose overlay byte is nonzero MUST block
  ground, whatever that byte is: no code is resolved, no range checked, and a code naming nothing
  MUST block as one naming something does.
- **FR-4 — the border.** A cell MUST be a border cell when its column is below 8, its row is below
  8, its column is at least `W-8`, or its row is at least `H-8`. Every border cell MUST block ground
  **and** air. A map with either extent at 16 or below is therefore border throughout.
- **FR-5 — the byte, and what it does not carry.** A grid byte MUST set bit 0 exactly when FR-2,
  FR-3 or FR-4 blocks that cell's ground, and bit 1 exactly when FR-4 covers it — so **the border is
  the only source of the air bit**. Bits 2-7 MUST be clear in every derived byte, so every derived
  grid is one the world constructor accepts. The five arms are a **union**: no ordering among them
  is asserted, and no ordering is observable.
- **FR-6 — the world carries it, and only this path sets one.** A world built from a decoded map
  MUST hold exactly those bytes, carry them in the grid section of its canonical byte form and have
  them in its digest, so two maps differing in one derived cell MUST NOT hash equal. No other way of
  assembling a world MUST derive or default a grid, and no grid MUST change while its world is
  advanced. The form's version MUST NOT move: this story adds no field and no section.
- **FR-7 — one map, one world.** The same decoded map MUST yield the same grid, byte form and
  digest in every process and on every machine.
- **FR-8 — an instrument.** A developer command MUST report, for a `.alm` file named on its command
  line, how many cells of the derived plane hold each byte value, how many block a ground and how
  many an air mover, and how many each of FR-2's three arms, FR-3 and FR-4 would block **with the
  others absent**, so the five overlap and need not sum. It MUST take that path as an argument, embed
  none, and MUST NOT be part of the test suite.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | interior cells carrying, one each: a plain land word; bit 13; a word in `[512,768)`; `g=7, b=1, s=1`; a nonzero overlay; nothing at all | the grid derived | the plain and empty cells are `0x00`, the other four `0x01`: ground blocked, air clear, bits 2-7 clear in all six |
| **AC-2** | unit | strip group 7 at all four blend columns and all fourteen sub-cells | derived | exactly **35** of the 56 block, the ones at level 3 or more, and the pair `b=1,s=0` and `b=1,s=6` — levels 3 and 2 — falls either side of the compare |
| **AC-3** | unit | group 7 at sub-cells 14 and 15; strip groups 13, 14 and 15; and bits 10, 11, 12, 14 and 15 added to a **group-7 blocking** word and to an open one | derived | the first five block nothing, and no added bit changes any cell's byte |
| **AC-4** | unit | words inside `[512,768)` with sub-cell 8 and with sub-cell 9, and the one at sub-cell 4 with `(i & 0x30) == 0x10` | derived | all three block ground: the range is read whole, so no sub-case escapes it |
| **AC-5** | unit | overlay codes 1, 82, 246 and 250 on cells whose word blocks nothing, and code 0 on a cell whose word blocks | derived | the four block ground and the fifth is exactly what its word made it |
| **AC-6** | unit | a 24x24 map; then a 16x24 and a 24x16 | derived | on the first, exactly the cells within 8 of an edge carry the air bit and each the ground bit too, no interior cell carrying air; on the other two every cell carries both |
| **AC-7** | unit | a hand-built map whose interior exercises all five arms | its world marshalled, hashed, then unmarshalled into a second world | the grid section matches an expectation written from the contract — the interior cell by cell, the border by rule; the digest equals a value derived from those bytes outside this package; the second world's form is byte-for-byte the first's |
| **AC-8** | unit | a world from a decoded map; one built by hand over a grid; one over none; and a second map differing only in altitudes, units, objects, triggers and name | each read through its byte form | the first carries the derived bytes, the two hand-built ones exactly what they were given, and the fourth's grid equals the first's |
| **AC-9** | unit | a map with a water channel and one land gap, a unit ordered across; the gap closed by water; the gap closed by a body | advanced to arrival, and 16 ticks | it arrives having never stood on a cell whose ground bit is set; closed by water the order ends in the tick that finds it, a re-search from an unmoved cell reading the same terrain; closed by a body it holds cell, target and route and gives up inside the sixteenth |
| **AC-10** | unit | no map at all; maps with no tile plane; a tile plane shorter and one longer than the extent; no overlay plane; a width of 0; a negative height | each derived and each built into a world | a grid of exactly the extent's cell count where the extent has cells and none at all where it has none, a usable world every time, no error and no panic |
| **AC-11** | manual | the instrument over a lawful install's shipped maps | run | it prints the per-value census, both blocked counts and the five per-arm counts, read against the published corpus figures |
| **AC-12** | manual | the game on a lawful install, a map with a lake, a mountain range and trees | a group boxed and ordered across each | the group routes around water and mountain, will not enter a tree's cell, and cannot enter the outer eight rings |

**Error cases:** None applicable. The derivation refuses no map: a mis-sized plane, an absent one
and a non-positive extent are **resolved outcomes**, which is what lets the loader keep a signature with
no error a caller could act on.

## Derived properties

- **P-1 (invariant)** — Every derived grid has exactly the extent's cell count and no byte with a
  bit above bit 1, so no derived grid is one the world constructor refuses.
- **P-2 (invariant)** — In every map, the air bit is set on the border cells and on no others.
- **P-3 (completeness)** — Every in-bounds cell gets exactly one byte, and no map shape makes the
  derivation raise, panic or return a length other than the extent's.
- **P-4 (negative-invariant)** — Nothing but the extent, the tile plane and the overlay plane reaches
  a grid byte: altitudes, units, objects, groups, triggers, names and header fields may differ freely
  between two maps with equal grids.
- **P-5 (invariant)** — A world built from a decoded map holds the same grid, byte form and digest
  in every run, and advances to the same cells under the same commands.

## I/O examples

```
w = 0x01D0  i 0x1d0, g 7, b 1, s 0, level 3   blocks ground
w = 0x01D6  i 0x1d6, g 7, b 1, s 6, level 2   no tile arm
w = 0x0208  i 0x208, inside [512,768)         blocks ground
w = 0x0350  g 13, a group never written       no tile arm

grid byte   0x00 open  ·  0x01 ground blocked  ·  0x03 border, ground and air
```

## Constraints

| # | Constraint | Alternatives and trade-off |
|---|---|---|
| **C-1** | The `.alm` **scenery layer is baked** into the grid at load; **placed structures are not modelled at all.** | **(A) attach structures too** — the decoded model recomputes a cell's byte from a per-cell record a structure attaches to, needing a footprint rectangle and two masks the game keeps in `Data.bin` and the object record's class resolution, neither of which this tree reads. **(B) bake a footprint from an invented size** — a classification with no source, which this loader has always refused. **(C, chosen)** bake what the ingest bakes, forgoing what attachment buys: no demolition, and **a bridge cannot open the water it crosses**. |
| **C-2** | The block byte is projected onto **two bits**. | **(A) widen the grid byte to carry the object bit** — it revises a landed story's refusal set for a mover this tree cannot express, per-unit masks being hashed state nothing has imported. **(B, chosen)** two bits — the terrain bits of the ground and air masks: lossless for every mover here, lossy for one that does not exist. |
| **C-3** | The **border is derived**, not left to the bounds clamp. | **(A) drop it** — the clamp already keeps a unit on the map, but the outer eight rings would be walkable where the original forbids them and the air bit, whose only writer this is, would stay dead. **(B, chosen)** derive it, and pay that a map of 16 or fewer cells on an axis is wholly impassable. |

**Disclosed limitations**, owned: bridge decks and doorways stay blocked — **1113
shipped cells**, 908 of them decks — so a map crossed only by a bridge has no crossing here; and a
unit placed well inside the border never moves, every neighbour being blocked too.

## Out of scope

- **Structures and every runtime change to the plane** — attach, detach, demolition, bridges,
  doorways, the per-cell record and its recomputed byte.
- **Weighted movement cost** — the cost plane and its blend; 0029 fixes a flat step cost and this
  story does not widen it — and with it the **height plane, move duration and speed**.
- **Per-mover masks and footprints** — the object bit, the mover it exists for, multi-cell
  footprints, flyers, per-unit movement types.
- **The dynamic plane** — occupancy bits, the record flag, bits 3-5, and byte parity with the
  original's serialization.
- **Resolving a type-3 code to a class**, and anything a code names.
- **Any change under `pkg/sim`** — the grid contract, the byte form and its version are 0029's and
  0033's, read here without one being moved; **routing** with it, unreachable goals and substitute
  destinations included; and the **placeholder order script**, which aims a leg at the extent and is
  not taught passability here.

## Verification mapping

AC-1 to AC-10 are unit tests over maps and worlds built in test code — no window, no clock, no
install — all ten CI-automatable; AC-11 and AC-12 need one. P-1, P-2 and P-4 are
witnessed by AC-1, AC-6, AC-8 and AC-10, P-3's totality half by AC-10's shapes alone, and P-5 by
AC-7 and AC-9 — each **sampled, not proved**.

## Gate check

FR-1 → AC-1, AC-10, P-1, P-3 · FR-2 → AC-1, AC-2, AC-3, AC-4 · FR-3 → AC-5 · FR-4 → AC-6, P-2 ·
FR-5 → AC-1, AC-6, P-1 · FR-6 → AC-7, AC-8, AC-9, P-4 · FR-7 → AC-7, P-5 · FR-8 → AC-11 ·
C-1 → AC-12 · C-3 → AC-6.
