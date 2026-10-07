# Provenance — the footprint, the two masks, and a title that means its own inverse

Pinned at research `acb8fb0` (`acb8fb085757f5436ebb5f737255fe1630662829`), re-pulled for R-2, R-3
and R-4 and frozen again there. The three arrived together, in one experiment, and all three
**confirm** the stance the contract already carried — which is the outcome a reader should trust
least on its own, and the reason each row below names the instruction chain rather than the fit.

`claims/retracted.md` and the standing-corrections table were read **first**. Seven rows cited below
appear there; each is cited as amended, never as first published:

- **`TERR-STRUCT-070`** — REFUTED in a parenthesis only: the anchor is not two coordinate bytes of
  the object, that field being a pointer. **The bit-index arithmetic is unchanged**, and the same
  round *closed* the row's Medium anchor clause to High.
- **`TERR-STRUCT-074`** — its polarity clause's *reason* SUPERSEDED, the clause raised to **High**
  (`TERR-STRUCT-078`); the bit-index and shift-mask clauses cited below are untouched.
- **`TERR-STRUCT-068`** — last arrow SUPERSEDED: the attach reaches the recompute on one of two
  exits. No shipped attachment takes the other, and nothing here attaches at runtime.
- **`ALM-OBJ-019`** — the "objects/structures" label is wrong; these records resolve against the
  structure roster alone. Cited only for the record layout and the walk.
- **`ALM-CLS-037`** — durability gloss and two invented class names gone. Cited only for the two-key
  class split, which stands.
- **`TERR-STRUCT-075`** — its Medium axis clause SUPERSEDED on **both** halves: the instructions do
  say (`ALM-OBJ-061`), and the off-plane statistic it rested on could never have decided, 37 of the
  38 shipped maps being square. The clause is now High and the row is otherwise untouched.
- **`ALM-OBJ-034` / `TERR-STRUCT-077`** — the override **gate** SUPERSEDED: both rows attributed it
  to the extension kind, and the branch had never been read. It is the sum of the two caller bytes
  (`TERR-STRUCT-090`). The row has two clauses and **the one FR-2 cites is not the one that moved**:
  the *discriminator* — the whole key against `0x21`, one named `CMP` immediate — is untouched and
  High; the trailing "their low bytes override the footprint" is what went. `retracted.md` carries
  the supersession and the ledger row does not, so reading either alone gives the wrong answer.
- **`ALM-CLS-036`** — amended at this pin, naming `ALM-OBJ-062` and `TERR-STRUCT-090`. The amendment
  is about the override; the clause cited below is the one-based subscript and its guard, untouched.

**Threshold: High.** The plane is canonical hashed state, so every row is cited at the confidence of
the **clause relied on**, not its row's headline. Nothing load-bearing is now below it: the one
Medium clause and the two unsourced details this file carried are the three rows closed below.

## Backing

| `spec.md` anchor | Claim | Confidence as published |
|---|---|---|
| *Footprint contract* — a placement's anchor cell is its stored coordinates shifted right by 8, the **first** of the two being the column | `TERR-STRUCT-075`, `ALM-OBJ-061` | High for the two shifts and the position object they feed / **High** for which coordinate is which — nine named instructions from the loader's two `LEA`s to the packed cell key, with no permutation anywhere on the chain, and the shipped masks reading as deck patterns only under it |
| *Contract*, FR-1 — the key is a structure-roster identity, taken as its low byte, indexing the `Buildings` collection one-based under the guard `key != 0 && key <= count − 1` | `ALM-CLS-036`, `DAT-BLD-005` | High for the guard and the 1-based subscript / High for the index law, carried by the skip-0 grammar and an ordered name agreement across two authored rosters |
| *Contract* — the collection is one-based because entry 0 is allocated and never written, so its count word is one larger than its entries | `DAT-GRAM-003` | High — transcribed from the loader plus the eight serialize bodies; the walk tiles the shipped file with no residue |
| *Contract*, FR-1 — the four parameters at positions 0, 1, 4 and 5, the column titles shipped for them, and each extent narrowed to its **low byte** | `TERR-STRUCT-070`, `TERR-STRUCT-077` | High — four named parameter copies with their addresses, the titles the shipped file's own, and both extents reaching the object through a named **byte** store |
| *Contract*, FR-3 — the bit index runs continuously across the rectangle, the anchor is the **top-left**, and the shift count is masked to five bits, so a footprint past 32 cells aliases | `TERR-STRUCT-070`, `TERR-STRUCT-074` | High — the arithmetic is named instructions; top-left rests on an absent subtraction over a chain read end to end; the mask is why the wider intermediate is harmless |
| FR-2 — the extension discriminator is the whole key, not its low byte | `ALM-OBJ-034` | High — one named compare immediate, and the rule turns a partial record walk into a complete one |
| FR-2 — the arm takes the caller's extents and writes both masks, so the whole rectangle attaches with an all-clear blocking set; its own over-32 branch writes the same value on both sides and decides nothing | `TERR-STRUCT-077` | High for the six stores and the dead branch's two identical immediates |
| FR-2 — the extension's **first** four bytes carry the width and its **second** four the height, each narrowed to a low byte, the other four reaching no instruction | `ALM-OBJ-062` | High — six named stores and the two push orders joining them; a transposed reading needs a single instruction to say otherwise and none does. Its own Medium is on the **corpus**, which separates the two only three times and cannot see the field width at all; the reading rests on the image, where it is High |
| FR-2 — the override arm is selected by the **sum** of the two extension bytes, not by the kind, so two zero bytes take the table arm | `TERR-STRUCT-090` | High — the seven instructions of the test, the kind byte not among its operands, and the four literal-zero pushes that close every other route to the arm |
| FR-3 — the **attach set** is tested in the rectangle walk, the **blocking set** in the per-cell recompute; the call order fixes which is which | `TERR-STRUCT-070`, `TERR-STRUCT-071` | High — each test is a named instruction, and the swapped-column rival is excluded by call order rather than by counting |
| FR-3 — a cell already carrying a structure is refused, and the remainder of that footprint abandoned | `TERR-STRUCT-068` | High — the refusal and the abort are named instructions with their branch targets |
| FR-3 — placements are attached in the order the map records them | `ALM-OBJ-034`, `ALM-OBJ-019` | High — the extension discriminator makes the record walk forward and sequential, so file order is the only order there is |
| FR-4 — one arm sets a cell's ground and object bits together, the other clears exactly those two, and it is applied against the **terrain baseline** restored first, so a structure can subtract water, scenery or mountain | `TERR-STRUCT-071` | High — both stores, their operands, and the mask's single route into the byte |
| FR-4, *Contract* — a **set** blocking-set bit closes the cell, a **clear** bit opens it; the shipped title is its inverse | `TERR-STRUCT-078` | **High** — both branch targets fixed by their own `rel8` bytes and re-read from the PE; the rival is excluded by the taken block's first bytes, not by a census |
| FR-4, P-4 — neither arm touches the air bit, and the map margin is that bit's only writer | `TERR-PASS-073`, `TERR-PASS-051` | High — complete writer lists with the instrument's blind spots closed by hand; the mask predicate and its three values are named instructions |
| FR-5 — the pass runs strictly **after** the ingest, from the placed object's own construction, and the ingest's writes are assignments, so an attach running first would be erased | `TERR-STRUCT-072` | High — the load ordering, the global publication between the two, and both call sites are named instructions |
| FR-1, FR-3 — **every** placement resolves and attaches by one rule; the shop class reaches the resolver through its base-class constructor and, pushing two literal zeros, always takes the table arm | `ALM-CLS-063`, `TERR-STRUCT-072`, `ALM-CLS-037` | High — one direct `CALL` immediate inside the constructor's own listing. Two call sites, **three** callers: a call-site count could not have shown it, which is what the published figures assumed rather than tested |

The five arms this pass is applied after are `0036`'s contract and are not re-derived here.

**`TERR-STRUCT-074`'s Medium on the direction is withdrawn, and its reason with it**: over the 66
shipped `Buildings` rows the code's reading gives **35** classes blocking every attached cell and
**2** none, the rival **2 and 35**. The corpus does separate them; bridge joins were the wrong
statistic. So AC-9 is a live check on the direction; AC-2, AC-3 and AC-6, taking it from FR-4, are
not. `analysis.md` still calls it published Medium with the corpus barely separating it; it stands
as written, with this governing.

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| The plane stays **two bits**; the object bit is not carried (*Constraints*) | The bit is real and High, set and cleared only ever paired with the ground bit. The mover reading it apart from the ground bit needs per-instance state no story has imported — `0036`'s reason, unchanged |
| The pass runs **once, at derivation**; no per-cell record, no terrain baseline, no runtime attach, detach or demolition, no record-present marker bit (*Constraints*, *Out of scope*) | All four are decoded and High. None is reachable from a plane derived once and fixed at construction, and the marker bit reaches the original's save format, which nothing here writes |
| A footprint cell outside the map extent is **dropped** (FR-3, AC-7) | The original writes it into a fixed-stride plane whose columns past the map are unused, so below that stride the write is unobservable — and no shipped placement puts a footprint cell off-plane. Dropping matches where the corpus can see, and is defined where it cannot |
| An entry with too few parameters yields **no footprint** (FR-1) | The original reads the four positions unguarded, and a parsed `Buildings` row is the collection's own width, so the case cannot arise from a file it loads. This is the derivation staying total, not a claim about the game |
| A **zero extent** is a skip rather than a resolved footprint of no cells (FR-2) | **A zero extent CAN arise, and the earlier reading here — "narrows both extents to unsigned bytes, so it cannot" — was wrong: the narrowing rules out a negative, not a zero.** `TERR-STRUCT-090` names the case outright, a `0x21` record whose two extension bytes are both zero, and `ALM-OBJ-062` finds no shipped record that is one. Where it arises the original's own walk is bounded by the extent and attaches **nothing**, so the plane is identical either way; what differs is only which counter of *our* census the placement lands in. Ours, and disclosed rather than discovered (G2) |
| The movement cost the opening arm also assigns is not carried (*Constraints*) | The assignment is High and names its source. Routing takes a flat step cost by an earlier decision, so there is no plane for it to land in |
| The table is an **argument**; no world is built from an installed file at this tier (FR-6) | Ours, and the rule the unit-definition argument already follows: one map with two tables is two worlds, and a tier reaching for an installed file would make that the map's fault |
| The census, its counters and the two directions counted apart (FR-7) | Ours: no claim describes a tool. What it is read against is published |
| The **tint** — its colour, its opacity, its flag and its being a developer tool rather than a game screen (FR-9) | Ours entirely. The original draws no such thing; the plane it tints is the decoded one, and every claim behind that plane is above |

## Open, and deliberately not consumed

- **Structure art.** Unconsumed here, and **no longer unasked**: this pin publishes the draw path
  whole — `TERR-STRUCT-100` through `TERR-STRUCT-107`, the anchor, the per-cell strip loop, the three
  frame arms, the shadow shear, the two passes `Flat` selects between, the two hard-coded bridge
  classes and the owner-independence of all of it. It is a sibling story's material, not this one's;
  what this file must not still say is that it does not exist. The **request is answered**.
- **The per-cell record's six further pointers**, four quadrupling a cell's cost and one also
  blocking: **Unknown** in `TERR-STRUCT-069`, its instrument and blind spot both stated. Unreachable
  from a plane derived once.
- **The attach's second exit**, which writes no plane byte: `TERR-STRUCT-076`, High for both exits,
  **Unknown** for whether play reaches the silent one. No shipped attachment takes it.
- **The durability, scan and health values** the entry and the record carry — decoded, the record's
  own scalar re-read as a shop's stock ceiling rather than a durability. No consumer here.

## The three open values, closed

Each was a **value, not a shape** — every requirement held at either answer — and each is now High.
What that changed in the contract, and what it did not:

- **R-2, which coordinate is the column** → `ALM-OBJ-061`, **High**. The first. The contract already
  read `(X >> 8, Y >> 8)` against `[0, Width)`, so **no clause moved**; what moved is that the
  assignment is now carried by an unbroken instruction chain instead of by a statistic that could
  not have discriminated. The transposed build is not merely unpreferred: it puts 178 footprint
  cells off the map and collapses bridge joins 93 to 5.
- **R-3, the extension's byte layout** → `ALM-OBJ-062`, **High**, and `TERR-STRUCT-090` beside it.
  The stance was right about which four bytes are which. The **gate** was not: FR-2 said the whole
  key selects the override, and the branch tests the sum of the two extent bytes, so a `0x21`
  placement carrying two zero bytes takes the table arm. FR-2 and AC-4 are narrowed to that. No
  shipped record exercises it — all 8 carry non-zero bytes in both — which is exactly why only the
  instruction could have said so.
- **R-4, whether the shop-class placements attach** → `ALM-CLS-063`, **High**. They do, through the
  base-class constructor, and they always take the table arm. The contract distinguishes no class,
  so again **no clause moved** — but the alternative would have needed an exclusion in FR-1, and the
  cost of getting it wrong is 450 cells and a component boundary on two maps.

**Does FR-2's last sentence hold at every input, or only where the corpus reaches?** At every input.
`TERR-STRUCT-090` makes a zero extent **reachable** — by two zero extension bytes through the table
arm, or by one through the override — and the original has behaviour for it rather than none: its
walk is bounded by the extent and attaches no cell. "No footprint" and "a footprint of no cells"
write the same plane, so there is no divergence to disclose. What the two readings do part on is
which counter of *our* census a placement lands in; see *Ours by choice*.

## Removed, and why

- **Every corpus count** — cells opened and attached, placements, the deck-cell and component figures,
  the containment relation between the two masks, and which shipped entry exceeds 32 cells. All
  published, none of it contract: the builder writes identical code with or without them.
- **The named classes** whose shipped masks are deck patterns and whose doorways open a wall. They
  motivate the story and decide nothing in it, and naming them would put the original's own content
  strings in a file that does not need them.
- **The front-end wiring**, cut to keep the story one story. Its cost is disclosed in *Out of scope*
  rather than left as an omission.
- **The `[R-x]` markers**, with the three values they stood for now closed above.
