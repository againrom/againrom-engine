# Provenance — 0111

Every normative section of `spec.md` appears in exactly one register below. Claim IDs are the
`research/` submodule at this story's pin; confidence is the grade the claim carries, and where a
claim is graded twice the clause this story leans on is named. The spec reads correctly with this
file deleted.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-3 — the drawn set is the map's authored ground records and nothing else | `ITEM-SPAWN-026` | High — one call enumeration, complete on the repaired table with 0 orphan hits, every owner read; it is the row that established the `.alm` as a sack origin at all |
| FR-3 — a campaign map gets no sacks from anywhere else at load | `ITEM-SPAWN-027` | High — the gate is one `CMP`/`JNZ` at the call site and its operand has a two-instruction derivation in the map loader. It corrects `ITEM-SPAWN-013`, which is **partially retracted twice over** and whose headline must not be read |
| FR-3 — which type-8 records are ground sacks and which are not | `ALM-SACK-065`, `ITEM-OWNED-028` | High for both readings; `ITEM-OWNED-028` is **Medium** for what the actor ids identify, which this story does not use |
| FR-3, FR-4 — the drawn list is stable across a mission because nothing ages or expires a sack | `ITEM-SACK-011` | High for the two tick stubs and the absence from the actor list; **Medium** for "no *other* container ages it", whose blind spot is a wholesale sweep of the manager's list |
| FR-4 — a sack is a value at a cell, one per cell, held in a registry of its own | `ITEM-SACK-010` | High — the class record, the vtable extent, the three constructors and the merge-or-create routine are read at instruction level |
| FR-5 — the anchor a cell-anchored sprite is placed by, and that the frame's own size enters it | `TERR-SPR-040` | High for the formula, six named instructions per axis, the rivals excluded by which fields appear rather than by a fit; **amended** so that the drawn frame's size, not frame 0's, is what enters |
| Constraints — one sack per cell, and that no map can author two distinguishable ones there | `ITEM-SACK-010`, `ALM-LIM-067` | High / **Medium** — every width and guard in `ALM-LIM-067` is a named instruction but no modified map was written or loaded |
| Out of scope — a pick-up is all-or-nothing and takes the whole container | `ITEM-PICK-009` | High for the routine and the tick arm; the state transition it once graded Unknown is closed by `ITEM-PICK-016`, and the discriminator that row named was **retracted** |
| Out of scope — the pick-up is one path with two entry points and the walk is the first's point | `ITEM-PICK-016` | High — each store, each hop and both range tests is a named instruction, and the alternative is enumerated away |
| Out of scope — what a pick-up would have to pour into, and that it is unbounded | `ITEM-CONT-004` | High for the class shape and both fields' arithmetic; **Medium** for "nothing refuses" |
| I/O example — that the four mission-10 cells are ground records and the fifth is not | `ALM-SACK-065` for the grammar, `ALM-SACK-066` for the corpus | High for `ALM-SACK-065`; `ALM-SACK-066` is High for its three discriminating properties and **Medium** for the census figures. The four cells themselves are our own decode of the shipped map through our own tool, recorded as evidence rather than cited from a claim |

## Ours by choice

Engineering this build fixes because nothing decoded fixes it. Each is changeable later without
contradicting any source.

| Choice | Spec anchor | Why it is ours |
|---|---|---|
| That the `backpack` sheet is a sack's art | FR-1, I/O | No claim reads the drawn object's art address. `UNIT-CLASS-023` puts `CBackPack` in the interface family, a different class from the simulation sack, and nothing in the pin follows it to a sheet. The name, the six-frame ladder and the frames' shared centre are why we believe it; a rendered sack that is not a sack is the criterion that would refute it |
| The frame a sack draws — the sheet's first | FR-2, Constraints | Nothing names the selector. The alternatives and the cost of being wrong are the spec's own constraint table; the corpus measurement behind the choice is in `analysis.md` |
| The sack's canvas and centre — the drawn frame's own size, and that frame's centre pixel | FR-5 | A class record supplies `Width`/`Height`/`CenterX`/`CenterY` for an object, and the sack sheet has no class record in any registry this build reads. Note honestly: `TERR-SPR-040` *excludes* the canvas centre as the anchor **for an object with a class record**, on the ground that `CenterX` is not `Width/2`. Our rule is not that exclusion overturned — it is the answer for a sheet that has no `CenterX` at all, and the six frames being centred on one point of their canvas is what makes it the natural one |
| Where a sack sits in the band at an equal row — after both art planes, before an entity | FR-7 | The band's merge is already ours by choice; nothing at this pin places any of these planes in the original's own cell walk. The rule chosen is the one already in force for the two planes it joins, extended by one list |
| No overlay layer and no shadow | FR-9 | A cut, not a claim. The overlay sibling ships and is deliberately unread — nothing in this build composites one for any sprite, so a sack drawing one would be the only sprite that did |
| That the sheet is decoded once at start-up rather than per map | FR-1 | One sheet serves every map, so nothing about it is map-dependent. The observable consequence is that a missing sheet cannot fail a mission open |

## Open

Undecoded, and deliberately assigned no meaning here.

- **What selects among the sheet's frames.** The sack's value slot is recomputed as gold plus the
  sum of its items' own value slots (`ITEM-SACK-010`), which is what makes a value ladder the
  standing hypothesis — but that row is about the simulation object, and the drawn object is a
  different class (`UNIT-CLASS-023`) whose frame selection no row reads. This story draws one frame
  and reads nothing out of the sack's payload at all.
- **What an item is worth.** `ITEM-DEF-002` leaves a definition row's columns uninterpreted for
  three of the four item classes, so the value ladder could not be evaluated even if it were named.
- **What the overlay layer contributes to a sack.** It pairs one frame per frame and is far sparser
  than the base, which is the published shape of an overlay in general; what it draws on this
  particular sheet is unread.

## Removed

Statements carried in an earlier draft and dropped, so that nothing load-bearing vanishes silently.

- **A content-count frame ladder** was drafted as the contract and removed to option B of the spec's
  constraint table. It survives as a rejected alternative rather than as a deferred plan: it is not
  merely unimplemented, it is argued against by the corpus.
- **A payload field on the drawn-sack seam** — the purse and the item count crossing to the window
  tier — was drafted and removed. With one frame for every sack, nothing downstream could read it,
  and a field nothing reads is a claim that something will.
