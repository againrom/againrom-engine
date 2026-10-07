# Provenance — 0103

Research pin: **`53f8bb7`**, read through the submodule at `research/` (`git submodule status` shows
no leading character). Every row below was read at that pin with `go run ./tools/claim <ID>`; no
ledger was opened by hand.

## Claims relied on

| Clause | Claim | Confidence |
|---|---|---|
| The type-8 section is the map's authored loot, not a marker tree: the loader's case-8 arm builds a record list at `map+0x2dc` and one ungated consumer walks it and calls the sack maker | `ALM-SACK-065`, `ITEM-SPAWN-026` | High — the jump-table arm was read out of the image, and the consumer's single call site is a `CALL` immediate inside the `.alm` loader; the call enumeration is complete with 0 orphan hits |
| FR-2 the head fields and their consuming instructions — element count, owner, the two axis words, gold | `ALM-SACK-065` | High — every field is named with both the instruction that reads it and the instruction that consumes it |
| FR-1 the payload carries no count word; the record count is the type-0 metadata word at `+0x2c` | `ALM-TRIG-050` | Medium — corpus closure over 38/38 maps, with the loader's own case-8 bound as the mechanism |
| FR-2 the head is 20 bytes only at format version 989 and above, and the fifth word is zeroed below it | `ALM-TRIG-050` | High for the gate (one cited compare); the shipped corpus never exercises the short head |
| FR-2 the 10-byte element: the code is the low `u16` of the leading word, the second word is read only on the stock arm, the link is a 1-based type-9 index | `ALM-SACK-065` | High — each of the three has its reading instruction cited |
| FR-6 the item code's four fields, the class window 1..14, and the class-14 index widening to a whole byte | `ITEM-CODE-029`, `ALM-LIM-067` | High for the masks and the allocation sizes (named immediates on one path); the meaning of the other two fields is **Unknown** and D-4 declines to give them one |
| FR-6 the class nibble is bits 8..11 rather than 12..15 | `ALM-SACK-066` | High — a wrong nibble would null about an eighth of the codes and the shipped corpus nulls **none** of 181 |
| FR-7 the cell is the axis word arithmetic-shifted right by 8 | `ALM-SACK-065`, `ALM-TRIG-050` | High for the shift (`SAR ...,0x8` at both cited sites); the `0x80`-centred anchor agrees on 180/180 records |
| FR-17 the axis order X then Y | `ALM-SACK-066` | High — 23 of 28 check-14 nodes name a cell a ground record occupies and **0** do so with the axes swapped |
| FR-9, FR-10 a sack is one object per cell with a purse and a container, and the maker merges into a sack already there rather than building a second | `ITEM-SACK-010`, `ALM-SACK-065` | High — the merge-or-create routine and its cell lookup were read at instruction level |
| the sack is not an entity: no tick, no timer, not in the actor list | `ITEM-SACK-011` | High for the two stub vtable slots and the absent registration; **Medium** that nothing else ages it |
| FR-16, FR-17, FR-18 check opcode 14 is a cell-to-sack existence test, its two coordinates are read as bytes, and the returned pointer never reaches the register | `TRIG-SACK-022` | High for the arm and the lookup; **Medium** for *sack* — the identification is the caller set, five of six of which are sack work |
| the Why: 28 check-14 nodes on 11 maps reached by 33 condition slots | `TRIG-SACK-022` | High — a walk of every shipped map's own records |
| the Why: 30 of 38 maps carry a section, 137 ground records, 43 stock records, 181 elements, and the walk closes 38/38 | `ALM-SACK-066` | **Medium** — corpus agreement. Quoted in `spec.md` as scale, never as a threshold; AC-1 re-measures it |
| FR-21 a record whose owner is non-zero stocks an actor and its coordinates and gold are never read | `ITEM-OWNED-028` | High for the arm and its two destinations; **Medium** for what the owner ids identify, which this story does not need |
| the Why: the scatter is multiplayer-only at **both** call sites, so a campaign map with no type-8 section has no loot | `ITEM-SPAWN-027`, `ITEM-SPAWN-013` (retraction) | High — the second gate is one cited compare, and the retraction was read |
| D-6 five live check-14 nodes name a cell no ground record occupies, four cited by triggers | `ALM-SACK-066` | Medium — corpus |
| D-8 the purse is a `u32` added into, and shipped values reach 500 000 | `ALM-LIM-067`, `ALM-SACK-066` | Medium |
| out of scope: a pick-up takes the whole sack, credits the purse to the player and destroys the sack | `ITEM-PICK-009`, `ITEM-PICK-016` | High for the routine; cited here only to establish that the path this story does **not** build needs an actor container and a player purse |

## Retractions read before use

**`ITEM-SPAWN-013` is partially retracted, twice over, and `research/claims/retracted.md` was read
through the reader.** What falls is its headline — "five things make a sack, and none of them is the
`.alm`" — its exclusivity clause, and, by omission, that only the second call site of the random
scatter is multiplayer-gated. A consumer built on the old reading would scatter 10 to 25 random
sacks on every campaign map that the original places none of, and would place none of the ones the
map authors. **This story is built on the overturn** (`ITEM-SPAWN-026`, `ALM-SACK-065`) and on
nothing the retraction took back.

**`ALM-TRIG-022` is retracted** in the two clauses that matter here: that type-8 holds marker
regions, and that its only consumer is the trigger machinery. This tree's format leaf still carries
the retracted reading in the name of its type-8 field and in that field's doc comment; correcting
both is part of FR-1's task and is not a cosmetic change — the comment is what a reader of the leaf
believes.

**`ITEM-PICK-009` is partially retracted** in its named discriminator for the pick-up state
transition. Nothing here rests on that clause; it is named because the pick-up is the story this one
defers to, and whoever writes it should start from `ITEM-PICK-016`.

`ITEM-SACK-010` and `ITEM-SACK-011` are both marked **active (contested)**. Both are used only for
what they establish at instruction level — the merge-or-create rule and the absence of a tick — and
neither contested edge is load-bearing for any clause above.

## What is a request to research, not a fact used

- **A worked example.** The brief offered one map's record 0 at a named cell holding a single weapon,
  matched against that map's own `"got bow"` check. **No claim publishes it**; a keyword sweep of
  every ledger at this pin returns nothing. It exists only inside an experiment, so it is a request
  to research to publish, not something to cite. It is used nowhere in this story, and the same
  discrimination is carried by `ALM-SACK-066`'s 23-of-28 property, which **is** published.
- **A sack's sprite.** D-5. Nothing published says what the original draws for one. Until it is, a
  placed sack cannot be drawn without inventing a picture.
- **What the type-9 list is for.** D-4. Its grammar is published (`ALM-TRIG-049`) and its meaning is
  not, so the element's 1-based link into it is carried and unread.

## Ours by choice

- **Version 22 of the byte form, and the section's placement in it.** Nothing in the original
  constrains either; the form is ours.
- **Ascending (Y, X) order** (FR-11). The original packs its cell key as `(Y<<8)|X`, so Y-major is
  its own order, but this build sorts for determinism rather than to match a layout.
- **Refusing an out-of-bounds sack in the simulation and dropping it in the loader** (FR-11, FR-22,
  D-7). The original would wrap. No shipped map exercises the difference.
- **A search rather than a per-cell flag plane** (D-2).

## Owed, and to whom

- **The pick-up**, with the actor container and the player purse it needs. Owed by its own story.
  This one leaves the contents in place for it.
- **Stocking an actor from a stock record** (D-3): decoded here, applied by the story that gives an
  actor a container.
- **Drawing a sack** (D-5): owed by research first.
- **Resolving an item code to a definition row.** `pkg/formats/databin` decodes a magic-item
  collection at the raw row level and nothing builds a typed definition from it. Owed by the story
  that needs an item's numbers.
