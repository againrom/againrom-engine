# Provenance — the arms that block a cell, and the plane this tree declines to rebuild

Pinned at research `8c92427` (`8c92427e06f2d5432f5a50aec9f4bd7e099939c9`), frozen for the story.
`claims/retracted.md` was read first: of the rows below, `TERR-PASS-051` appears there twice and
`TERR-COST-052` once. The clauses they lost are the *movement-domain* naming of the mask selector,
the consequence that a `.alm` session can only hold the ground mask, and the cost getter's reader
list. **None of the three is cited here**, and the second is overturned in the direction this
story's non-goals already assume.

**Threshold: High.** The derived grid is canonical hashed state and decides which trajectories
exist, so every row is cited at the confidence of the **clause relied on** rather than its row's
headline, and no clause carrying Medium or Unknown sets a bit of the plane. One Medium clause is
load-bearing at one remove, disclosed with what raises it inside this tree.

## Backing

| `spec.md` anchor | Claim | Confidence as published |
|---|---|---|
| FR-2(a) — tile-word bit 13 blocks ground | `TERR-PASS-049` | High — the test and its store are named instructions with their immediates |
| FR-2(b) — the range `[512,768)` blocks ground and is tested **on the word itself**, not through any classification | `TERR-PASS-049` | High — a mask and a compare standing ahead of every class compare |
| FR-2(c) — the split of `w & 0x3ff` into sub-cell, blend column and strip group; the reject at a sub-cell of 14 or more; the blend level in 1..5 with the primary taken at 3 or more; Mountain as class **8** of a 1-based ten-class enum in fixed record order; and Mountain reachable **only** as group 7's primary, which lets the compare specialise to one group | `TERR-PASS-050`, `ALM-GRID-012`, `ALM-TERR-015` | High — masks, shifts, table displacements and the five blend arms are immediates in one routine with one caller; the group pairs are the row's own and the 4x14 level table is published beside them in the terrain format spec at this pin |
| FR-3 — a nonzero type-3 cell blocks ground, whatever the code, with nothing resolved | `TERR-PASS-049` | High for the arm / Medium for **which record** that plane is — see *Open* |
| FR-4 — the border, its depth of 8, and that it covers a whole map narrower than 17 | `TERR-PASS-049` | High — the depth is a literal and the shipped count matches the closed form to the cell |
| FR-5 — bit 0 blocks ground; bit 1 blocks air and **the border is its only writer**; bit 2 marks an object and never moves without bit 0 | `TERR-PASS-073` | High — complete writer lists, both of the instrument's blind spots named and closed by hand |
| FR-5 — a cell blocks a mover iff `block & mask`, ground `0x41` and air `0x82`, so two bits serve both exactly | `TERR-PASS-051` | High for the predicate, the mask setter and the constructor's default |
| FR-6 — the block plane is what a consumer that hashes state must carry; the cost and height planes are not | `TERR-PASS-053` | High for the archive-mode test and the sweep |

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| The plane is projected onto **two bits**; bit 2 is dropped (FR-5) | Bit 2 is real and High. Nothing here can consume it: the mover that reads it — mask `0x44`, unstopped by terrain and stopped by objects — needs a per-instance mask byte, per-unit hashed state (`TERR-MOVE-057`) no story has imported. Carrying it would widen 0029 FR-1's refusal set for a reader that cannot yet exist |
| The arms are a **union**, not the ingest's sequence of assignments (FR-5) | The sequence is real (`TERR-PASS-049`, High) and, bit 2 dropped, unobservable: in the two bits kept, the border's value covers the scenery value and that covers the terrain value. It would encode an order no criterion here can discriminate — nor can the corpus, 1 OR 5 being 5 |
| Strip groups **13-15 block nothing** (FR-2) | The original indexes an unwritten table there and reads uninitialised heap (`TERR-PASS-050`, High). No behaviour to reproduce, and 0 of 880 704 shipped cells reach it |
| A cell the map carries no tile word or overlay byte for reads as **zero** (FR-1) | Ours. A decoded map always carries `W*H` of each; this is the loader declining to fail on a hand-built one, and what keeps the world constructor's error unreachable |
| Row-major `W*H` indexing, not the fixed 256-stride plane (FR-1) | Ours. The two agree at or below 256 columns, and ours does not alias above it |
| The grid is **fixed at construction** (FR-6) | 0029 FR-1, and why structures are a non-goal rather than merely undecoded: the attach model mutates the plane while the world runs |
| The instrument and its census (FR-8) | Ours: no claim describes a tool; the counts it is read against are published |

## Open, and deliberately not consumed

- **Which `.alm` record the scenery arm reads.** `TERR-PASS-049` grades the *naming* of the type-3
  and type-2 planes **Medium**: it is read off the loader's case arms and cross-checked by swapping
  them, which makes 94.8 % of every map impassable. Load-bearing here, so what raises it is recorded
  rather than assumed — this tree's `pkg/formats/alm` binds the same two records, 0012 draws type-2
  as relief and 0017 draws type-3 codes as placed objects, both on the owner's install. A swap makes
  those two visibly wrong at once.
- **Structures.** The footprint rectangle and both masks are decoded (`TERR-STRUCT-070`), as are the
  constructor attach, the recompute, the detach, and the arm that **subtracts** bits 0 and 2
  (`TERR-STRUCT-068`, `-071`, `-072`), all High. Missing is two tiers this repo has not built: a
  `Data.bin` reader and the object record's class resolution. `TERR-STRUCT-071`'s **1113 opened
  cells** are the measured price, disclosed in the contract.
- **Bit 2's consumer, bits 3-5, the dynamic plane and its occupancy bits** — High or Unknown in
  `TERR-PASS-073`; none is reachable from a grid derived once.
- **The cost plane.** `TERR-COST-052` is High that the blended byte is never the block decision.
  0029 takes a flat cost by decision, so the blend is left where it is published.
- **Whether loading a save re-runs the ingest** — `TERR-PASS-053`, Unknown. Nothing here loads one.

## Removed from the baseline, and why

- **The `Provenance basis.` preamble and the `Research needed` section** — refused by check 0.
- **A reference-derived type-class split and a reference-derived 64-value table.** Deleted at import
  rather than disclosed, as 0029 deleted the two reference step-cost constants. Nothing here
  recovers either: the split, the group pairs and the blend levels are read out of the image with no
  candidate value supplied, so this is a derivation and not a confirmation.
- **The three research items**, and the criteria withheld behind them — all three closed at High, so
  withholding a mountain criterion now would withhold it from evidence that exists.
- **The `nil`-grid case, and "`blockedAir` is never set here"** — both false against this tree;
  `analysis.md` names each and its replacement.

## Appended 2026-08-01 — pin `130bb79`: the opening sentence's count of overturn rows is now short

This file opens by saying that of the rows below, `TERR-PASS-051` appears in `claims/retracted.md`
twice and `TERR-COST-052` once. At `130bb79` two more of them do, and the sentence is left as
written because it was true at `8c92427`:

- **`ALM-GRID-012` — SUPERSEDED.** "bit 13 = impassable flag" is the *smallest* of three tile-word
  arms that block a cell — 7 464 cells against 136 622 for the strip-pair class and 87 584 for the
  raw water test — and the census denominator is 880 704, not the 880 552 first published. FR-2(c)
  cites the row for the `w & 0x3ff` split and not for bit 13, so nothing here rests on the lost
  clause; this contract already derives all three arms and takes their union.
- **`TERR-MOVE-057` — SUPERSEDED.** Its "2 = Ghost/Bee (`0x44`), 3 = Bat_Sonic/Dragon (`0x82`,
  air)" reads as two grades of one thing. They are not a ladder: `0x44` carries the object bit and
  not the terrain bit, `0x82` neither, and the two disagree on **83 203 of 880 704** shipped cells
  (`MOVE-DOM-026`). The *Ours by choice* row that drops bit 2 describes the `0x44` mover as
  "unstopped by terrain and stopped by objects" — right for this plane, and one term short of the
  whole rule, since that mover is also stopped by every **building** and by a ground occupant on
  the dynamic plane. Neither reaches a static plane, so FR-5's projection is unchanged and the
  reason for dropping bit 2 is unchanged with it.

`TERR-PASS-051`'s three consequence counts are re-scoped at this pin — `430 898 / 229 092 /
158 976` is the **ingest** plane, `441 540 / 242 179 / 158 976` the plane after the structure pass
(`MOVE-DOM-026`). This story's FR-8 census is its own instrument against its own derivation and is
not read against either set, so nothing in `verification.md` moves.

## Appended 2026-08-02 — pin `01c64e2`: two structure rows lost a clause each, and the price disclosed above is in the wrong unit

- **`TERR-STRUCT-070` — `● active, as amended`, and stronger than when cited.** What
  `retracted.md` classes **REFUTED** is a parenthesis: the footprint anchor is *not* `obj+0x10`
  byte 0 = col, byte 1 = row, because `obj+0x10` is a **pointer** to a 12-byte position object and
  the two bytes are `[*(obj+0x10)+0]` and `[+1]` (`TERR-STRUCT-075`). **The bit-index arithmetic is
  unchanged** — the field named was wrong, not the formula. Separately, the row's **Medium** anchor
  clause is **closed to High**: EXP-0081 read the store at `L02083` and the chain behind it, so
  top-left is now carried by an *absent subtraction* rather than by the 68.5 % deck-on-water fit.
- **`TERR-STRUCT-068` — `● active, as amended`**, its last arrow **SUPERSEDED**. `R1357`
  has two exits and only the record-creating one calls the recompute; the other stores the building
  and returns without writing a plane byte (`TERR-STRUCT-076`). **0 of 17 057 shipped attachments
  take it**, the ingest creating no records, so no figure here moves — but "attach one cell →
  recompute that cell" is not what the routine does.

**`TERR-STRUCT-071`'s 1113 opened cells is the right count in the wrong unit.** EXP-0081 measured
the same foreclosure as **connectivity**: over the 38 shipped maps, bridge-deck cells free go
**391 of 1 299 → 1 299 of 1 299**, and **92 of the 104 bridge placements join two ground components
the ingest leaves disjoint** — `Islands.alm`'s largest component **22 189 → 35 135**, `scn:121.alm`
**516 → 1 814**. Under the ingest plane alone **no** bridge joins anything (`TERR-STRUCT-074`, High
for the mechanism; Medium for reading the figures as a statement about *play*, being plane
reachability with no occupancy, triggers or mission gates). Nothing derived changes and C-1 still
chooses (C) — but a price quoted in cells reads as small, and the same price quoted in connectivity
is a world cut into islands. That is the correction, and it runs against this story rather than for
it.

`spec.md` C-1 and its *Disclosed limitations*, `analysis.md`'s "measured price" and
`verification.md`'s foreclosure paragraph all quote the 1113 figure. None of the three is wrong —
each already says in words that a map crossed only by a bridge has **no crossing here** — but all
three are unquantified where a number now exists. Correcting a landed contract is a story, not a
sweep; this note is the record until one is opened.
## Appended 2026-08-02 — pin `9ff259c`: the polarity clause goes to High

`TERR-STRUCT-074` is `● active (amended)`: its **Medium** polarity clause — a set `Passability` bit
blocks — is **High** on one branch displacement (`TERR-STRUCT-078`), its reason SUPERSEDED.
The corpus does separate them: 35 classes block their whole footprint against 2 under the
code's reading, 2 against 35 under the rival; bridge joins were the wrong statistic. **The clause
this story cites is the other** — the connectivity figures, still Medium about play — untouched,
as is the note above and its verdict on the cells-vs-connectivity price.

## Appended 2026-08-02 — pin `acb8fb0`: the anchor axis is settled by instructions, and the 17 057 figure is now tested rather than assumed

**`TERR-STRUCT-075`'s Medium clause is superseded upward.** The `01c64e2` append above cites the row
for the two bytes being `[*(obj+0x10)+0]` and `[+1]`, and notes its anchor clause closing to High.
The remaining **Medium** — which of `L02090`/`L02091` is the low axis — is now **High as amended
by EXP-0091**, carrying a `claims/retracted.md` row classed **SUPERSEDED**. Both of its reasons are
withdrawn and its answer is confirmed: `ALM-OBJ-061` reads the chain from the loader's two `LEA`s to
the packed cell key and finds no permutation, and the "0 of 17 057 footprint cells off-plane"
statistic never discriminated — 37 of the 38 shipped maps are square, so it reads 0 under both
assignments. Re-made as connectivity the corpus does separate them: bridge joins 93 against 5.

**The 17 057 attachment count quoted above is unchanged, and better founded.** `ALM-CLS-063` finds
the `Shop` constructor reaching the footprint resolver through its base class — two call sites,
three callers — so the census that produced 3 141 placements and 17 057 cells "was right, but
assumed this rather than tested it". The 50 shop placements attach 450 cells of that total.

**Nothing this story derives moves.** `spec.md` C-1 models no placed structure at all, so the
resolver's caller set, the extension's override gate (`TERR-STRUCT-090`) and the extents'
field-by-field decode (`ALM-OBJ-062`) all sit outside the derivation. The cells-vs-connectivity
price disclosed above still needs a story, unchanged in kind.
