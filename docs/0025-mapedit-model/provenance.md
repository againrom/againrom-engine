# Provenance — map-editor E2: the edit / document model

Pinned at research `3d95f2a`, frozen for the story. `claims/retracted.md` read first: beside the
pre-EXP-0030 framing rows it holds two label corrections that bear here — `ALM-UNIT-040`'s two id
words, relabelled by `ALM-UNIT-048`, and `ALM-OBJ-019`'s "objects or structures". Re-read at
`03a9448`, 2026-07-30: its "×1000 durability" gloss on type-4 `+0x0c` is withdrawn too, at
**Medium** (`ALM-CLS-037`; the field is a shop's stock cap, `SHOP-CAP-004`). No type-4 field is
read here.

## Backing

| `spec.md` anchor | Claim | Confidence as published |
|---|---|---|
| The 20-byte file and record headers, `payloadSize` at record `+0x08`, pure payload, no trailer | `ALM-FRAME-031`, `ALM-META-024` | High — six proofs, one a refutation from bytes alone |
| `dataSize` is carried, never recomputed | `ALM-HDR-001` | **Medium** — `4·W·H + 72` on 38/38, the `+72` unexplained, no consumer read |
| The 632-byte type-0 payload and every offset a setter writes | `ALM-META-008` (amended) | High — case 0's read lengths sum to exactly 632; amended, `+0x78`..`+0x277` is **one** 0x200 read, so not every boundary below is one |
| `name` `+0x30` and `description` `+0x78` are each **64 bytes**, NUL-terminated | `ALM-META-010` | High for the boundaries — re-verified landing on the string starts 38/38 / **Medium** for the labels |
| The 448 bytes at `+0xb8` are 7 x 64-byte text slots, loaded, which campaign maps fill with in-game trigger and quest strings | `ALM-META-028` (amended) | High that it is loaded / Unknown per-slot role. Amended: neither consumer subdivides the region, so 7×64 is a writer's model with no reader behind it |
| `+0x24` is `#type6`; place and delete maintain it | `ALM-CNT-017`, `ALM-META-025` | High — case 6's own loop bound; `70·count == payloadSize` 38/38 |
| The `+0x08` f32 is read into the map object | `ALM-META-027` | High that it is read and stored / **Medium** that it is a light or sun angle |
| `+0x0c/+0x10/+0x14` and `+0x70/+0x74` are stored config scalars; `+0x18` is read into a local and **discarded** | `ALM-META-026` | **Medium** (stored vs discarded) / **Unknown** (meaning). `+0x70` lies in `{1,4,8,12,16}` — 1 on all 28 campaign maps |
| Cell `(x,y)` at payload offset `(y·W+x)·elem` from the payload's first byte; type-1 cells are 16-bit tile words | `ALM-GRID-032`, `ALM-GRID-012` | High — the sim ingest's own indexing read, plus four corpus statistics agreeing on the shift |
| The type-6 record is 70 bytes; X `+0x00` and Y `+0x04` are whole `u32` fixed-point words (`/256`) | `ALM-UNIT-018` | High for layout, tiling and X/Y — case 6 reads exactly 70 B and stores the first two dwords / Medium for the label |
| The 62 bytes a move carries are live state: class keys `+0x08`/`+0x0a`/`+0x10`, owner `+0x14`, the range-checked `+0x18`, the `0xFF` runs at `+0x35`/`+0x3b`, the ids `+0x40`/`+0x42` | `ALM-UNIT-040`, `ALM-CLS-038`, `ALM-OWN-039` | High — the read map sums to exactly 70 at version 990, each field tied to a named instruction; the owner is a 1-based roster slot, 8094/8094 |
| That those two id words are the unit id and the group id in that order | `ALM-UNIT-048`, `ALM-TRIG-046` | **Medium** — corpus distinctness and a 169/169 trigger resolution, but on a trigger grammar itself Medium |

Every Medium row argues for **carrying** a value rather than computing one, so a later correction
to one cannot invalidate a mutation.

## Ours by choice

| What the spec fixes | Why nothing else was available |
|---|---|
| Every General setter names its field with the identifier the shipped reader already uses — `Angle`, `Word0C/10/14`, `Bitmask`, `Word70/74` | The alternatives are reference-derived. `ALM-META-026` has the loader **discard** `+0x18`, so a name asserting a load-time effect there is refuted, not merely unproven; `+0x0c/+0x10/+0x14` are Unknown; `+0x70`'s player-capacity reading is corroborated only at Medium. Naming one field while withholding names from its neighbours would rank confidence in a way the ledger does not support |
| Place takes a whole 70-byte record; the model interprets only X/Y | `ALM-UNIT-040` publishes all 70 bytes, the shipped `Unit` exposes 20. Zeroing the other 50 breaks every field the read map names — a record that decodes and that the game cannot use |
| A string edit rewrites the whole 64-byte field and rejects an encoding longer than 63 bytes | NUL termination is part of `ALM-META-010`'s boundary claim, and 64 encoded bytes would remove it. The whole-field fill is an editor choice over text-plus-one-NUL, whose rival keeps a residue of the replaced text in the field |
| Same-codec symmetry, with rejection as the failure mode | The codec split is 0003's, per `ALM-META-010`'s "ASCII or Windows-1251". Measured, not assumed: one byte of 256 (`0x98`) decodes to U+FFFD and cannot be re-encoded, so "byte-identical" alone is false |
| No bounds check of a coordinate against W and H | `ALM-UNIT-018`'s 8094/8094 in-grid anchors are what shipped maps do, not what the reader requires. Enforcing it makes the model a validator on one field and not on the others research also constrains |

## Open / undecoded

- **What the five stored type-0 scalars mean** (`ALM-META-026`, Unknown). `+0x28` is
  read-and-discarded, also Unknown (`ALM-META-025`), and is not settable here.
- **The per-slot role of the seven text slots** (`ALM-META-028`, Unknown).
- **Whether a placed unit needs a fresh `+0x40` id, a valid `+0x14` owner or an in-grid
  coordinate.** All three constrain the game, none constrains the file the reader accepts; left
  unenforced.

## Removed from the baseline and why

- **"Author CP1251 `@0x78..0x278` (512 B)" and the FR-4 built on it.** Two regions, not one: a
  64-byte description (`ALM-META-010`) and 448 bytes of text slots (`ALM-META-028`).
- **The X/Y low-16 / high-16 split and its preservation clause.** `ALM-UNIT-018` reads both as
  whole `u32` fixed-point words.
- **"A map with no Units section" and "the absence of the addressed grid section".** Removed as
  unreachable — all ten records existed in everything `alm.Open` then accepted — and
  **reinstated 2026-07-30**. That was our reader's acceptance read as a format fact: the loader
  gates the count at `>= 3`, requires `{type1, type2}` alone and manufactures an absent type-3
  (`ALM-REQ-055`/`056`, `ALM-CORP-060`, all High, retracting `ALM-SEC-003`). The **owner** ruled
  we must read at least what the engine accepts, superseding the orchestrator's earlier "correct
  the claim, keep the strictness"; `Open` widened, and type-3 and type-6 absence are FR-3/FR-5
  rejections again.
- **"Name CP1251"** — ASCII per `ALM-META-010`; the seven reference-derived General labels and
  **`PlaceUnit(u alm.Unit)`** — see *Ours by choice*; **`Heights int8`/`Objects`** — the shipped
  decode is `Altitudes []uint8`/`Overlay []uint8`.
- **Its R-1 and R-2 research requests.** Answered at this pin:
  `ALM-META-025`/`026`/`027` for R-1; `ALM-UNIT-040`/`048`, `ALM-CLS-038`, `ALM-OWN-039` for R-2.

## Appended 2026-08-01 — pin `130bb79`: one cited row is REFUTED, two more are on the new High-bar list

**`ALM-CLS-038` — REFUTED.** The struck clause is "the live `CUnit` holds a `units.reg` **section
index** at `+0x20`": it holds the **`ID`**, and nothing is translated. The R-2 row above cites the
claim for the 62 bytes a move carries — the class keys, owner, range-checked `+0x18` and the two id
words — none of which is the struck clause, and this model stores the keys raw and resolves
nothing. Unchanged.

**`ALM-META-025` and `ALM-META-028` are named by the pin's High-bar sweep**, among twenty-five rows
whose Confidence cell states corpus agreement as its whole warrant — reported and **deliberately
not re-graded**, with the instruction to treat them as Medium. `ALM-META-025`'s cell reads
"Corpus-universal size-identities … 38/38" with only its `+0x20` clause naming a loop bound;
`ALM-META-028`'s concedes in its own words "a writer's model with no consumer behind it". Both are
cited above at High for R-1's metadata surface. **This is a note, not a disclosure**: nothing in
this story reaches hashed simulation state — `pkg/mapedit` is an editing model and its round trip
is checked byte for byte against the file it read, which is a stronger witness for a field's
*position* than either row's census. What the lowered grade does reach is any future consumer that
takes a *meaning* from those rows; this story takes positions and counts.

`ALM-GRID-012` also gained a **SUPERSEDED** row — bit 13 is the smallest of three tile-word arms
that block, 7 464 cells against 136 622 and 87 584, denominator 880 704. It is cited above only for
the cell-at-offset indexing and the 16-bit tile word, not for the bit's meaning; this model stores
the word and interprets no bit.

## Appended 2026-08-02 — pin `01c64e2`: `ALM-OBJ-019` is now contested, on a field this model never reads

`ALM-OBJ-019` reads `● active (amended, contested)`. The new **live** contradiction is **C-7**
(`claims/registry.md`): the row states the `.alm` type-4 `+0x12` word is **sign-extended into
`obj+0x10`**, and `TERR-STRUCT-075` reads `obj+0x10` as a **pointer** to a heap-allocated 12-byte
position object — stored at `L02083` in the base actor constructor and dereferenced by both
footprint routines. They cannot both hold. **Research picked neither side:** EXP-0081 read the store
and both dereferences but not where `.alm +0x12` goes, so it can say the destination this row names
is not that field and cannot say what is. The clause is lowered to **Medium** and both rows are
marked contested. The field's *role* — the id a type-7 `Target_Structure` parameter names,
`ALM-TRIG-046` — is untouched.

The preamble above already records that **no type-4 field is read here**, and that is what protects
this model: `pkg/mapedit` carries the record's bytes and resolves nothing, its round trip checked
byte for byte against the file it read. The contested fact is about the live object the engine
builds from those bytes, which this story never builds. Nothing moves.

## Appended 2026-08-02 — pin `acb8fb0`: `TERR-STRUCT-075`'s Medium clause is superseded upward, and the `0x21` extension is decoded two bytes further

The note above cites `TERR-STRUCT-075` for reading `obj+0x10` as a **pointer**. That half was always
High and is untouched. What moved is the row's **Medium** half — which of `L02090`/`L02091` is
the low axis — now **High as amended by EXP-0091**, with a `claims/retracted.md` row classed
**SUPERSEDED**. Both old reasons are withdrawn and the answer they reached is confirmed:
`ALM-OBJ-061` reads the chain end to end and finds no permutation in it, and the "0 of 17 057
footprint cells off-plane" statistic never discriminated, 37 of the 38 shipped maps being square.
**C-7 is unchanged and still live.**

**`ALM-OBJ-062` decodes two bytes of the type-4 `0x21`-kind extension** — file `+0x14` and `+0x18`
become the footprint extents, while file `+0x16`, `+0x17`, `+0x1a` and `+0x1b` reach no instruction
on that path. That extension is named in the *out of scope* list above, and this model carries the
record's bytes and resolves nothing, its round trip checked byte for byte. A decoded field inside a
region this story deliberately preserves raw changes neither the round trip nor the scope line.
Nothing moves.
