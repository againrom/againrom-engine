# Plan — one pass over the cells, and the two bits it is allowed to set

## Baseline

`mapload.FromALM` builds one entity per `alm.Unit` — id from the slice index, cell from `unitCell`'s
fixed-point shift, the class key raw, both health fields from `SpawnHP` — and calls
`sim.NewWorld(Seed, bounds, ModeCanonical, nil, ents)`, discarding an error whose every path it
closes by construction. It reads `m.Units`, `m.Width` and `m.Height`, nothing else. `pkg/mapload`
imports `pkg/formats/alm` and `pkg/sim`; the DAG allows `pkg/data` too and it uses none.

`sim.newGrid` materialises an absent grid as `gridCells(b)` zero bytes — `gridCells` is
`int64(W)*int64(H)`, and **0 when either extent is not positive** — refuses any other length and any
bit above bit 1, and copies. `blockGround` is read by `terrainOpen` and by the route decoder;
`blockAir` has no reader outside tests. `encode` writes the grid at offset 34, and `Hash` runs over
the whole of its output, that section included.

`pkg/formats/alm` hands over `Tiles []uint16` and `Overlay []uint8` — both `W*H` after `Open`, the
overlay manufactured as zeros when its record is absent — plus `TileIndex(w) = w & 0x3ff` and
`Impassable(w)`, bit 13 as a bool. `pkg/render/terrain` splits the same word its own way,
`g = (w & 0x1fff) >> 6`, for graphics.

`pkg/game`'s fixture map is **12x9** with an all-zero tile plane and no overlay at all; tests there
advance worlds built from it and assert units move, two of them driving `mapload.Schedule`'s own
orders, and one loading a **6x5** synthetic stream. `pkg/mapload`'s own fixture is **40x24**, both
planes nil, every unit on a cell FR-4 will block, and its seed test pins `FromALM`'s digest against a
world hand-built over **no** grid — which this story's wiring breaks outright. `cmd/almtool` imports
`pkg/formats/alm` alone; `internal/archtest`'s allow-map is fail-closed on any package or edge not
listed, and `docs/ARCHITECTURE.md` **and** `AGENTS.md` each carry that table in prose — three
copies, not two.

## Design decisions

### DD-1 — one exported pure function in `pkg/mapload`; the loader gains one argument

`Passability(m *alm.Map) []byte` returns the plane and `Census(m *alm.Map) Counts` the eleven
numbers FR-8 asks for, and **both walk the cells through one unexported per-cell classifier**: the
union plane cannot yield a per-arm count, so without that shared classifier the instrument would be
counting a copy of the rule instead of the rule. Both are exported because a world has no grid
reader. `FromALM`'s only change is passing the plane where it passes `nil` today, and a nil map is
an extent of nothing (FR-1, FR-7, FR-8).

Rejected: **recomputing the arms inside `cmd/almtool`** — a second derivation that measures nothing,
agreeing with the first by having been written from it; `unitCell`'s own doc refuses that trade by
name. Rejected: **deriving inline in `FromALM`** — every arm-level criterion would then be read out
of a byte offset. Rejected: **a package of its own** — a DAG row and a tier for two functions.
Rejected: **`pkg/formats/alm`** — that leaf publishes bit fields and asserts no meaning for them,
which is 0003's rule and the reason `TileIndex` ships without a classifier beside it.

### DD-2 — the movement split is written here and shared with nothing

`i = alm.TileIndex(w)` then `s`, `b`, `g` by FR-2's shifts; bit 13 through `alm.Impassable`. The
render tier's `(w & 0x1fff) >> 6` stays where it is — the DAG forbids the import anyway, and the two
readings are different functions above bit 9, which is what makes SC-3's words a discriminator (FR-2).

Rejected: **one exported split shared by both tiers** — a third reading that is neither, making one
tier's graphics constant the other's movement contract.

### DD-3 — the mountain arm is the decoded classifier specialised to strip group 7

`g == 7 && s <= 13 && level[b][s] >= 3`, with `level` a package-level `[4][14]uint8` carrying its
claim id (`TERR-PASS-050`) in the source, as every comparable constant in this tree does. Mountain is
the primary of exactly one strip group and the secondary of none, so the class compare has one
solution and the specialisation is an identity rather than an approximation (FR-2).

Rejected: **the whole ten-class classifier with its cost blend** — a class is not observable in a
passability grid, so no criterion could constrain it until the cost story arrives, and that story
widens this arm rather than replacing it. Rejected: **a table of booleans** — it deletes the compare
the decoded rule makes, so SC-8's threshold mutant could not be written.

### DD-4 — the arms are OR-ed into one byte in one pass over the cells

One loop over `y`, `x`: `blockGround` OR-ed by each of FR-2's three arms, by FR-3's and by FR-4's,
`blockAir` by FR-4's alone, and nothing else ever written. So the byte is a union by construction
and no ordering exists to get wrong (FR-3, FR-5).

Rejected: **reproducing the ingest's assignment order and its separate border pass** — two passes
for a precedence that is unobservable the moment the object bit is dropped, and "the last writer
wins" would need a criterion that cannot exist.

### DD-5 — the border is a comparison against the extent, not a pair of loops

`x < 8 || y < 8 || x >= w-8 || y >= h-8`, in `int`, so a narrow map underflows nothing and the
whole-map case falls out of the arithmetic rather than being special-cased (FR-4).

Rejected: **the original's two loop shapes and its 256-stride array edges** — the second half of
that shape addresses cells this representation does not have.

### DD-6 — the length rule is `sim`'s own, restated in one place

`Passability` sizes its result from `int32(m.Width)` and `int32(m.Height)` — the values `FromALM`
hands `NewWorld` — so the two cannot disagree on a decoded extent that does not fit an `int32`. It
returns `nil` for a nil map and wherever that narrowing is not positive, `W*H` bytes otherwise, which
is exactly `gridCells`; an index past a short `Tiles` or `Overlay` reads zero, and a longer plane is
never indexed past the extent. Every `NewWorld` error path therefore stays closed and `FromALM` keeps
its errorless signature. The route decoder's ground-bit refusal is no new hazard either: a route is
only ever searched over the grid it is stored beside (FR-1, FR-6).

Rejected: **returning an error, or propagating the constructor's** — no caller can act on either, for
a state this decision makes unreachable.

### DD-7 — the fixtures grow to where a leg still lands in the interior

`pkg/game`'s 12x9 fixture becomes **72x72**, `pkg/mapload`'s 40x24 one grows with it, and every unit
anchor and every hand-written target in the tests that advance a fixture world moves into the
interior. The size is not a round number: the placeholder script aims a leg at the **extent** and
caps it at 24 cells toward the roomier side, so a target clears an 8-cell border from every interior
start only once the extent reaches **64**. Below that a capped leg lands in the border, the unit
holds, and a test asserting motion fails for a reason unrelated to the derivation (FR-4, FR-6).

Rejected: **conditioning the border on the extent** — a threshold nothing decodes, invented to keep
a test green, which is the shape this story exists not to ship. Rejected: **teaching the schedule
about passability** — another story's contract, and it reads no grid today. Rejected: **handing the
fixtures a nil grid** — `FromALM` is precisely what those tests exercise.

### DD-8 — the instrument is a subcommand of `almtool`, and the DAG row moves with it

`almtool pass <file.alm>` prints FR-8's counts. `cmd/almtool` gains `pkg/mapload` in
`internal/archtest`'s allow-map and in **both** prose copies of that table — `docs/ARCHITECTURE.md`
and `AGENTS.md`. All three move together: the check is authoritative, the other two are prose about
it, and nothing mechanical catches the drift (FR-8).

Rejected: **a second binary** — a command and a DAG row of its own for one subcommand whose input is
the `.alm` file this tool exists to read. Rejected: **a test that walks an install** — forbidden.

## Risks

- **R-1** Every world a map builds moves state and digest, so every test over one moves with it. A
  fixture that stops moving reads exactly like a routing defect, so the fixtures come first — an
  ordering hazard, and DD-7 is the whole mitigation.
- **R-2** A map of 16 or fewer cells on either axis is impassable throughout, so a map fixture must
  be at least 17 on both axes to be a world in which anything can happen, and at least 64 if the
  placeholder script drives it.
- **R-3** The scenery arm blocks every nonzero code, residue included. On the shipped corpus the
  residue lies inside the border and costs nothing; a map with residue in its interior would block
  cells for no placement, and nothing here could tell.
- **R-5** The placeholder script aims at the extent, so a capped leg now names a border cell and its
  unit stalls and gives up. It is left alone deliberately — it reads no grid, and teaching it one is
  another story's contract — but a test that drives it and then asserts only that *something*
  changed can go quietly vacuous, a target field moving where no cell does.
- **R-4** The blend table and the group pairing are decoded constants with no in-tree derivation. A
  transcription error in one of 56 numbers moves routes on real maps and no criterion short of
  walking the whole table notices; SC-2 walks it, SC-9 is the backstop.

## Success criteria

- **SC-1** AC-1 and AC-5 hold, each arm on a cell of its own well inside the interior, every byte
  compared **whole** rather than by mask so a stray high bit fails, the five overlay codes each its
  own case (FR-2, FR-3, FR-5).
- **SC-2** AC-2 holds over **all 56** group-7 cells, each expected level written by hand rather than
  read from the production table, the count of 35 asserted and the straddling pair named (FR-2).
- **SC-3** AC-3 and AC-4 hold, each case the corpus cannot witness written separately: the two
  rejected sub-cells, the three unwritten strip groups, the three water sub-cases, and the **five**
  ignored bits added to an open word and to a **group-7 blocking** one. That word must be group-7 and
  nothing else — the two splits agree on every other cell, so any other choice leaves SC-8's split
  mutant alive (FR-2).
- **SC-4** AC-6 holds: the border compared cell by cell over 24x24 against a predicate written from
  FR-4's own words, the air bit counted at 512 of 576 and absent from all 64 interior cells, both
  degenerate extents checked whole (FR-4, FR-5).
- **SC-5** AC-7 holds. Nobody hand-writes 576 bytes, so the expectation is split and both halves are
  written from the contract rather than from the code: the interior cell by cell as a literal table,
  the border by a predicate written from FR-4's words. What pins the whole section is the **digest**,
  recomputed outside this tree from those same bytes and cross-checked through a second FNV in the
  test, neither carried over from a run of the derivation. The round trip is compared byte for byte
  rather than by digest, and taken after a tick in which a unit routed, so the decoder's own
  ground-bit refusal is exercised and not assumed (FR-6, FR-7).
- **SC-6** AC-8 and AC-10 hold: the three build paths read through their byte forms, the fourth map
  differing in every section but the three that feed a grid, and each degenerate shape its own case,
  shown to yield a usable world (FR-1, FR-6).
- **SC-7** AC-9 holds in **both** routing modes, the channel inside the interior so what turns the
  unit is the water and not the border, every cell checked against the grid after every tick (FR-6).
- **SC-8** Six mutants, each applied to production code, run over the whole tree with the failing
  tests named, and reverted: the blend compare widened to 2 or more; the sub-cell guard deleted; the
  movement split replaced by the render tier's; the border depth set to 7; the scenery arm gated on a
  code below 247; the air bit set by the water arm as well. Each is a design decision expressed as a
  defect — DD-3, DD-3, DD-2, DD-5, DD-4, DD-4 — so a survivor is a criterion that fails to
  discriminate its own decision and is reported as one. The second one's kill rests on DD-3's table
  being 14 wide: at 16 the tail entries read 0, the compare is false, and it survives silently
  (FR-2, FR-3, FR-4, FR-5).
- **SC-9** AC-11 is run over the install's shipped maps and its counts recorded **beside the
  published corpus figures** — three value counts, FR-5 leaving only `0x00`, `0x01` and `0x03`
  reachable, two blocked counts and five per-arm counts — with
  every difference explained rather than rounded away. This is the story's only end-to-end check and
  the only one the derivation cannot pass by agreeing with itself (FR-8).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-1, DD-6 | SC-6 |
| FR-2 | DD-2, DD-3 | SC-1, SC-2, SC-3, SC-8 |
| FR-3 | DD-4 | SC-1, SC-8 |
| FR-4 | DD-5, DD-7 | SC-4, SC-8 |
| FR-5 | DD-4 | SC-1, SC-4, SC-8 |
| FR-6 | DD-1, DD-6, DD-7 | SC-5, SC-6, SC-7 |
| FR-7 | DD-1 | SC-5 |
| FR-8 | DD-8 | SC-9 |
