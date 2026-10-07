# Spec — ALM map container, terrain grids & content sections (ROM1)

**Provenance basis.** The ALM (`M7R␀`) layout below is game-derived in the parallel research repo (the
`research/` submodule): `research/formats/alm/format.md` across EXP-0007 (original container framing),
EXP-0010 (type-0 metadata + the three grid layers), EXP-0019 (the content sections type4–9), EXP-0020
(grid cell semantics from `rom.exe` + `world.res:data/map.reg`), and **EXP-0030 (the corrected record
framing, the grid base, the placement→cell conversion, and the type-4 extension discriminator)**. EXP-0030
proved from `rom.exe` and the shipped bytes that the framing EXP-0007 published was split **8 bytes too
early**; the corrected contract below supersedes it. Confidence is the research's own: the container tiles
all **38 shipped maps** of that corpus exactly (10 EN-root `.alm` + 28 embedded in `scenario.res`; the RU
root was outside it, and `ALM-CORP-060` later widened the walk to 72 files over both roots), grid cell-counts are exact
38/38, content-section counts and coordinate-in-bounds tests are exact 38/38 (11 235 placements), the
placements are legal on the corrected grids (4.9 % derived-impassable vs 20.7 % on the old base), and
falsification passed. This story **implements the clean container + grid + content reader from that layout
and verifies it against a lawful install** (AC-9). It is a pure `formats` leaf: bytes in, structs out.
Established: the container framing, the type-0 metadata (including its content-count manifest), the
type1/2/3 grid layers with their per-cell meanings, and the type4–9 content sections (the fixed-record
object/group/unit tables and the trigger/marker trees). The remaining undecoded facts — the serialized
*leaf grammar* of the type7/8/9 trigger trees, the meanings of several type-0 scalars, and the last 8 bytes
of the type-7 payload — are tracked under Research needed and read raw, exposed without invented meaning.
Greenfield code, spec-anchored / static. **Revised 2026-07-27** on a pin bump that publishes the type-6
record's own read map (`ALM-UNIT-040`), its class keys (`ALM-CLS-038`), its owner field (`ALM-OWN-039`), and
— on the registry side — the correction that a placement's key needs **no translation** into a section index
(`REG-KEY-044`, whose contrary clause is retracted). That revision exposes a placed unit's class keys and the
two conditions under which the engine overrides them, and corrects the notes that still called that record's
tail undecoded; the rest of the published read map stays unexposed. Submodule pin: `research/` at `e61153d`.

## Problem / goal

Every ROM1 mission is an `.alm` map file: the GOG install ships 10 at its root and 28 inside
`scenario.res`. The engine needs a pure reader that validates the container, walks its fixed set of typed
records, decodes the map metadata (dimensions, name, description, content counts), the three full-map
terrain grids (tiles / altitudes / object-overlay), and the content sections (placed objects, the
player/group roster, placed units, and the trigger/marker trees). Pure `formats` leaf: byte stream in,
structs out; no VFS, engine, sim, or rendering knowledge (reading a map *into* the simulation — terrain
resolution, passability, entity spawning — is the later mapload tier).

## Format definition (game-derived, little-endian)

Source: the research submodule — `research/formats/alm/format.md` (claim IDs cited per row). A ROM1 ALM
is a **20-byte file header** followed by the number of length-prefixed typed records that header
counts, each a 20-byte record header plus its pure payload:

```
[ file header 20 B ][ recordCount records: each (20-B record header + payloadSize B pure payload) ][ ... ]
0               0x14                                                                              EOF
```

**Ten records with typeIds `{0..9}` is the shipped writer's regularity, not the container's rule** —
a distinction this contract got wrong until it was corrected, and the whole reason the acceptance
section below exists. The game's loader gates the count only at **`≥ 3`** and requires just the
type-1 and type-2 records; everything else it defaults, skips or steps over (`ALM-REQ-055`,
`ALM-REQ-056`). One shipped map proves it: the RU root's `Horror.alm` carries **four** records and
the loader accepts it (`ALM-CORP-060`). What this reader accepts is stated as **our** decision in
*Acceptance* below, never as a property of the format.

Every record's `typeId` and per-map constant live in its **20-byte record header**; the payload that
follows is **pure data**. There is no identifier overlaid on any payload, no per-record `f0/f1` pair, and
no file trailer on any shipped map (`ALM-FRAME-031` — this corrects the EXP-0007 split, which was 8 bytes
too early). That the file ends exactly at `20 + Σ(20 + payloadSize)` is a property of the shipped files,
not a rule the loader checks: it stops after `recordCount` records.

**File header (20 bytes):**

| Offset | Type | Content | Basis (research claim) |
|---:|---|---|---|
| 0x00 | char[4] | magic `4D 37 52 00` (`"M7R␀"`, LE u32 `0x0052374D`); the loader refuses a mismatch. It is **not** the loader's only gate — there are four: the magic, `recordCount ≥ 3`, `formatVersion ≤ 1001`, then type-1 and type-2 present | ALM-HDR-001, ALM-META-024, ALM-RDR-059 |
| 0x04 | u32 | `hdrLen` — the header's **own length**, constant `20`; read and used (the loader's header helper reads `dword[cursor+4]` bytes) | ALM-HDR-001, ALM-FRAME-031 |
| 0x08 | u32 | `dataSize` — a precomputed size `= 4·W·H + 72`; the loader never reads it, read & ignored here (its `+72` is unexplained under the corrected framing — R-4) | ALM-HDR-001 |
| 0x0C | u32 | `recordCount` — the loader's loop bound, gated **only** `≥ 3`. `10` on 71 of the 72 walked shipped files, `4` on `ru/Horror.alm` | ALM-META-024, ALM-REQ-055, ALM-CORP-060 |
| 0x10 | u32 | `formatVersion` — `990` on every shipped map; gated `≤ 1001`. `== 1000` makes the loader skip record headers entirely — a second framing, unexercised by any shipped map | ALM-META-024, ALM-FRAME-031 |

`W`/`H` are **not** in the file header — they live in the type-0 record payload (below).

**Record header (20 bytes)** — repeated `recordCount` times, starting at 0x14:

| Offset | Type | Content | Basis |
|---:|---|---|---|
| +0x00 | u32 | `tag` — constant `7` on all 380 shipped record headers; stored and never re-read | ALM-SEC-002, ALM-FRAME-031 |
| +0x04 | u32 | `hdrLen` — constant `20` (= this header's own length; pins the 20-byte record-header stride) | ALM-SEC-002 |
| +0x08 | u32 | `payloadSize` — payload byte length; the pure payload follows immediately | ALM-SEC-002 |
| +0x0C | u32 | `typeId` — **the word the loader's `switch` dispatches on**, through a 10-entry jump table. `0..9` select a case; an id **at or above 10 selects none** and the loader steps over the record with no error recorded | ALM-FRAME-031, ALM-REQ-055 |
| +0x10 | f32 | `perMapConst` (`selectorA`) — a per-map `f32` (e.g. `0xBFC02B6D`), byte-identical in every record header of a map | ALM-SEC-003, ALM-META-027 |

Because `typeId` is a **header** field, it is present regardless of `payloadSize`: a record whose payload
is **0 bytes** (an empty `type8` on 8 of the 10 root maps; an empty `type4` on a map with no objects — 12
such records across the corpus) is still fully typed from its header. There is no ASCII chunk tag and no
typing by elimination.

**Record roster.** The shipped writer emits one record per `typeId` ∈ {0..9} in the physical order
**`0, 1, 2, 3, 5, 4, 9, 8, 6, 7`**, on 71 of the 72 walked files. Neither the count, the set nor the
permutation is enforced by the loader (`ALM-REQ-055`, `ALM-ORD-057`). What the **order** does carry
is three real precedence relations — type-0 before type-1, type-2 before type-3, type-0 before
types 4..8 — because each of those takes a length or a loop bound from a field the earlier record
writes. This reader satisfies all three by construction: it addresses records **by `typeId`, never
by position**, and decodes type-0 before any size check that needs `W`/`H` or the counts, so no
length is ever taken from an unwritten slot. Payload sizes and roles by type (`ALM-SEC-004`; content
counts `#N` from the type-0 metadata, `ALM-CNT-017`):

| typeId | payloadSize | Role | Decoded here |
|---|---|---|---|
| 0 | 632 (constant) | metadata | W, H, name, description, content counts |
| 1 | `2·W·H` | grid: **tiles** (u16/cell) | cells + tile-index / impassable bits |
| 2 | `W·H` | grid: **altitudes** (u8/cell) | height cells |
| 3 | `W·H` | grid: **object overlay** (u8/cell) | occupancy cells |
| 4 | `20·#4` (+ 8·ext) | placed **objects/structures** | 20-B records (X/Y/kind) + `kind==0x21` extension walk |
| 5 | `76·#5` | player/group **roster** | 76-B records (name @ +0x0c) |
| 6 | `70·#6` | placed **units** | 70-B records (X/Y @ +0/+4) |
| 7 | variable | **trigger** effect/instant list (+ Drop) | `entryCount @ +0`; node blocks raw |
| 8 | variable, **or 0** | condition/**marker regions** | serialized tree, raw |
| 9 | variable | **tile-marker** list | `count @ +0`; records raw |

`#4 = meta+0x20`, `#5 = meta+0x1c`, `#6 = meta+0x24` (`ALM-CNT-017`); the variable trigger records
(type7/8/9) carry their own count word in their payload. (The type-0 `+0x28` word EXP-0018 read as a
type7 count is read-and-discarded — the case-7 loop reads its own count from the stream, `ALM-META-025`.)

**Record 0 — type-0 metadata (632 bytes, constant on all 38 maps — `ALM-META-008`).** A fixed record;
offsets are payload-relative. The record's `typeId = 0` and its per-map `perMapConst` are **header** fields
(above), so the payload opens directly with `W`.

| Offset | Type | Content | Basis |
|---:|---|---|---|
| +0x00 | u32 | `W` — map width (cells) | ALM-HDR-001, ALM-META-024 |
| +0x04 | u32 | `H` — map height (cells; `W ≠ H` occurs, e.g. 112×144) | ALM-HDR-001, ALM-META-024 |
| +0x08 | f32 | `angle` — radians ≈`[-π/4..+π/4]` (`π/4` = `0x3F490FDA` common); read as a stored light/sun angle, physical meaning per R-1 | ALM-META-027 |
| +0x0C … +0x14 | u32×3 | located stored scalars, ranges known / meanings unknown; read raw (R-1) | ALM-META-026 |
| +0x18 | u32 | low-bit bitmask (`≤ 0x1fff`); read into a local and **discarded** by the loader; read raw (R-1) | ALM-META-026 |
| +0x1C | u32 | **`#type5`** — player/group record count (`= size5 / 76`) | ALM-CNT-017 |
| +0x20 | u32 | **`#type4`** — object base-record count (`= size4 / 20`, base) | ALM-CNT-017 |
| +0x24 | u32 | **`#type6`** — unit record count (`= size6 / 70`) | ALM-CNT-017 |
| +0x28 | u32 | read-and-discarded (EXP-0018 read it as #type7; case 7 ignores it — meaning Unknown, R-1) | ALM-META-025 |
| +0x2C | u32 | `#type8` records; `0 ⟺ type8 empty`; not required by this reader (type8 is preserved raw) | ALM-META-025 |
| +0x30 | char[64] | `name` — ASCII, NUL-terminated (≤ 21 B used; empty on most campaign maps) | ALM-META-010 |
| +0x70, +0x74 | u32×2 | located stored scalars, meanings unknown; read raw (R-1) | ALM-META-026 |
| +0x78 | char[64] | `description` — code-page text (ASCII / Windows-1251), NUL-terminated (≤ 36 B used) | ALM-META-010 |
| +0xB8 | 448 B | trailing text slots — 7 × 64-B default-empty `"<None>"` slots; meaning unknown, read raw (R-1) | ALM-META-028 |

The reader decodes `W`, `H`, `name` (ASCII → UTF-8), `description` (CP1251 → UTF-8), and the three content
counts (`#type5/#type4/#type6` at `+0x1c/+0x20/+0x24`), which drive and cross-check the content-record
strides. The `angle` float and the remaining u32 scalars are exposed as **raw values only** — no named
meaning is asserted (R-1). The per-map `perMapConst` (from the record header) is exposed raw.

**Grid layers (type1 / type2 / type3 — `ALM-GRID-011`, base corrected by `ALM-GRID-032`).** Each grid
stores **`W·H` cells at payload+0**, row-major (`W` cells per row), with `payloadSize` exactly `2·W·H`
(type1, u16) or `W·H` (type2/type3, u8). The payload is **pure grid**: nothing is overlaid on it and no
cell is lost — the `typeId`/`perMapConst` live in the record header. Cell layout:

```
cell (col, row)  ->  element index  row*W + col      (0 <= col < W, 0 <= row < H)
type1: u16 at payload + 2*(row*W + col)
type2: u8  at payload +    row*W + col
type3: u8  at payload +    row*W + col
```

This is exactly how `rom.exe` addresses the grids (`malloc(W·H·2)` + `Read(ptr, W·H·2)` with an unadjusted
destination; the sim ingest reads `tiles[row·W+col]` etc.). A reader that placed the grid 8 bytes earlier
(the pre-EXP-0030 base) renders **4 cells left** on type1 and **8 cells left** on type2/type3 — and reads
the record header's `typeId`/`perMapConst` (`6d 2b c0 bf` … ) as the first tile cells.

| Layer | Cell | Reading (research) | Basis |
|---|---|---|---|
| type1 | u16 LE | **tile word** — **bits 0–9 = tile index**, **bit 13 (`0x2000`) = impassable flag**; bits 10–12 / 14–15 unused (0 / 880 704 cells with the corrected base). Terrain *class* is derived from the tile index by the loader (strip→terrain-pair + blend tables) — a downstream concern, out of this leaf | ALM-GRID-012 |
| type2 | u8 | **altitudes** — per-cell height (0..~246) | ALM-GRID-013 |
| type3 | u8 | **object/feature occupancy overlay** — sparse (0 = empty; nonzero = an object occupies the cell) | ALM-GRID-014 |

The reader returns each grid as `W × H` raw cell values (u16 for type1, u8 for type2/3) and documents the
above cell meanings; it MAY offer typed accessors (`tileIndex = cell & 0x3ff`, `impassable = cell &
0x2000`). Resolving a tile index to a terrain class (via the `rom.exe` strip→pair + blend tables and the
`world.res:data/map.reg` `Terrain` vocabulary), building the runtime passability grid, and mapping a
type3 code to a specific object are the **mapload tier's** job — out of scope here (ALM-GRID-012/013/014,
ALM-TERR-015/016).

**Content sections (type4–9).** Like the grids, each content payload is **pure**: the first record starts
at payload+0. `X`/`Y` fields are `u32` fixed-point `/256` (integer tile = `value >> 8`; low byte usually
`0x80` = tile centre).

**Placement anchor → terrain cell is a bare `>>8`** (`ALM-PLACE-033`): no origin is subtracted, no border
inset added, no rounding applied. Combined with the grid base above, a placement at `(X,Y)` stands on tile
`(X>>8, Y>>8)` of every grid layer.

*type4 — placed objects/structures (`ALM-OBJ-019`, `ALM-OBJ-034`), `#4 = meta+0x20`, base record 20 B:*

| Offset | Type | Content |
|---:|---|---|
| +0x00 | u32 | `X` (`/256`) — decoded |
| +0x04 | u32 | `Y` (`/256`) — decoded |
| +0x08 | u32 | `kind` — `0x21` ⇒ an 8-byte extension follows this record |
| +0x0C | u16 | ctor field — raw |
| +0x0E | u32 | ctor field — raw |
| +0x12 | u16 | ctor field — raw |
| (+0x14 | 8 B | *extension*: present iff `kind == 0x21` — a footprint override `w×h` in tiles + padding, retained raw) |

The loader reads the base record as `4,4,4,2,4,2` = 20 bytes, then, iff `kind == 0x21`, reads 8
more bytes. With this rule the walk consumes every payload exactly, **38/38**. The reader walks exactly
`#4` base records, appends the 8-byte extension when `kind == 0x21`, and MUST consume the payload exactly.
The extension is the object's footprint override in tiles (`ALM-OBJ-019` as amended); this reader keeps it
raw — no consumer needs it typed.

*type5 — player/group roster (`ALM-GRP-020`), `#5 = meta+0x1c`, fixed record 76 B:*

| Offset | Type | Content |
|---:|---|---|
| +0x08 | u32 | scalar (`0` or `5000`) — raw |
| +0x0C | char[] | **name** — NUL-terminated ASCII within the 76-B record (`Self`, `Monsters`, `Villagers`, `Neutral`, `Enemy`, …) → UTF-8 |

*type6 — placed units (`ALM-UNIT-018`; the whole-record read map is `ALM-UNIT-040`), `#6 = meta+0x24`, fixed
record 70 B. That read map is the `formatVersion == 990` shape — three of the loader's reads are
version-gated, so a map at another version carries a different record length and is already refused by the
`70·#6 == payloadSize` check (FR-5):*

| Offset | Type | Content |
|---:|---|---|
| +0x00 | u32 | `X` (`/256`) — decoded |
| +0x04 | u32 | `Y` (`/256`) — decoded |
| +0x08 | **i16** | `classKey` — the primary class key, read **sign-extended** (the loader `MOVSX`es it) — decoded (`ALM-CLS-038`) |
| +0x0a | u16 | `classKey2` — the secondary class key; also the `scenario/npc.reg` subscript on the NPC path — decoded (`ALM-CLS-038`) |
| +0x0c | u32 | `flags` — the word the loader keeps at its in-memory unit `+0x48`; **bit 0 selects the NPC path** (bits 2 and 7 are passed on; the other bits have no established meaning) — decoded raw (`ALM-UNIT-040`) |
| +0x10 | u32 | `defID` — a definition id that **overrides** the class key when nonzero and `≠ 0xcdcdcdcd` — decoded raw (`ALM-CLS-038`) |
| +0x14 | u32 | `owner` — a 1-based slot in the type-5 roster (`ALM-OWN-039`); decoded upstream, **not exposed** here |
| +0x18 … +0x45 | 46 B | the rest of the read map — the `meta+0x2c`-bounded index at `+0x18`, located scalars, the two `0xFF`-sentinel runs at `+0x35`/`+0x3b`, the unique id at `+0x42` (`ALM-UNIT-040`); decoded upstream, **not exposed** |

**The class key, and the two paths that override it (`ALM-CLS-038`).** `classKey` is a `units.reg` `ID`: over
8094 records on 38 maps its domain is `1..80`, and it lies inside that registry's 34-element `ID` set on
**8094/8094** but inside its 34 section indices on only 1447/8094. `units.reg`'s class array is itself keyed
by `ID`, so a consumer subscripts it with the key **directly — nothing is translated** (`REG-KEY-044`, which
retracted the opposite reading on 2026-07-27). The key is **not unconditionally the class**: the loader takes
a definition id instead when `defID` is nonzero and `≠ 0xcdcdcdcd`, and takes the `scenario/npc.reg` table
subscripted by `classKey2` instead when `flags` bit 0 is set. Both conditions are fields of the 70-byte
**file** record — `flags` is the file word the loader stores at its in-memory `+0x48` — so a consumer can
tell an *overridden* record from an *unresolvable* one without modelling the engine's memory. `ALM-CLS-038`
names the two paths and does not order them against each other; this reader exposes all four fields raw,
asserts no precedence, and resolves nothing — registry lookup is the data tier's (Out of scope).

*type7 — trigger effect/instant list + Drop (`ALM-TRIG-021`):* `[u32 entryCount @ +0][named-node blocks]`.
Each block is a serialized editor node (a NUL-terminated name, an id, parameter values, inline
parameter-name strings, `"<None>"` target sentinels) matching `Description Instants.ini`; skirmish maps'
first entry is usually (not always — false on 5/38) `"Drop Location"`. The reader decodes `entryCount` and
preserves the node blocks **raw** — the serialized leaf grammar is undecoded (R-2). The last 8 bytes of
this payload are what EXP-0007 mislabelled a file trailer; their meaning is open (R-4).

*type8 — condition/marker regions (`ALM-TRIG-022`):* box/circle `X/Y`-bearing serialized marker tree,
preserved raw (R-2). It is **absent on most skirmish maps**, and absent means a `payloadSize` of exactly
**0** — the record header is present (with `typeId = 8`) and the payload is empty. The type-0 metadata
declares the record count at `+0x2c` (`0 ⟺ empty`), but this reader does not rely on it (type8 is preserved
raw and typed from its header).

*type9 — tile-marker list (`ALM-TRIG-022`):* `[u32 count @ +0][records]` of tile-coordinate markers. The
reader decodes `count` and preserves the records raw (R-2).

**Trailer — there is none.** EXP-0007 saw an 8-byte trailer because it split every record 8 bytes early;
those two `u32` are the **last 8 bytes of the type-7 payload** (`ALM-TRL-005`/`ALM-TRL-030`, both retracted
by `ALM-FRAME-031`). On every shipped file the chain ends exactly at EOF — a regularity, not a gate:
the loader stops after `recordCount` records and never looks at what follows.

## Acceptance — ours by choice, and where it departs from the engine

This reader takes what the loader takes, with **three deliberate exceptions**. The list is here, in
the contract, because our acceptance is a decision we own and not a fact about the format; an
earlier version of this document asserted the strict version as decoded truth, and that was wrong.

**Taken, because the loader takes it** (`ALM-REQ-055`, `ALM-REQ-056`):

| Stream | What this reader does |
|---|---|
| `recordCount` of 3 or more, in any amount | walks that many records; the count is a loop bound, not a roster size |
| only `{type0, type1, type2}` present | accepts |
| **type-3 absent** | **manufactures** a `W·H` all-zero overlay plane |
| type-4/5/6/7/8/9 absent | empty section — the loader skips these, their loop bounds being type-0 fields that read 0 once the records are gone |
| type-0 counts naming records the file does not carry | **inert**: a count is only its own case's loop bound and that case never runs (`ALM-CORP-060`'s four-record map advertises 415 type-4 records it does not contain) |
| a `typeId` at or above 10 | stepped over, no error; its bytes belong to no section |
| a repeated `typeId` | **last-wins**, which is the loader's own behaviour — its case simply runs again over the same pointer field |
| bytes after the last counted record | not read, not an error |
| any physical permutation | addressed by `typeId`, so the three real precedence relations hold regardless |

**Refused, deliberately, and each for a reason that is not "the shipped files never do it":**

1. **A missing type-0 is refused** whenever type-1 or any of 4..9 is present — which, since type-1
   is itself required, means every accepted stream carries one. Here the engine is **undefined
   rather than defaulting**: case 1 takes its grid length, and cases 4..8 their loop bounds, from
   stack slots that **only case 0 writes** (`ALM-ORD-057`). "Match the engine" names no behaviour
   to match, so this reader requires the record instead of reproducing an undefined read.
2. **Every payload-size constraint stays a rejection** — type-0 ≠ 632, type-1 ≠ `2·W·H`, type-2/3 ≠
   `W·H`, type-5/6 ≠ `76·#5`/`70·#6`, a type-4 walk that does not consume the payload exactly, a
   type-7/9 too short for its count word. The loader does not *measure* these: case 1 reads
   `W·H·2` taken from the **type-0** record and case 3 reads the **type-2** record's size
   (`ALM-SEC-004`), so a mismatch there is a desynchronised read rather than a defined outcome.
   Matching it would be matching a bug.
3. **`formatVersion == 1000` is refused as an unimplemented dialect**, distinct from an invalid one
   and said so in the error. The loader supports it by skipping every record header — a second
   framing. No shipped map exercises it, so implementing it would ship a path verifiable only
   against fixtures we invented. `formatVersion > 1001` is refused as the loader refuses it.

**Manufactured is not read, and the difference is exposed.** The overlay plane is `W·H` cells
whether it came from a type-3 record or from this reader; nothing else in the decoded map differs
between absent and empty. So the map reports, per `typeId`, whether the file actually carried that
record. A consumer that writes a map back **MUST NOT** emit a record that was not there — that
would invent bytes the input never had. Manufacturing the plane rather than leaving it short is
itself a decision with a reason on both sides: the default is *semantically exact*, since the
engine's own per-cell test is "nonzero means an object blocks this cell" so an all-zero plane is a
valid empty one (`ALM-REQ-056`); and a short or absent plane is the more dangerous failure
downstream, silently disabling every consumer that sizes its work by the plane's length instead of
failing where it can be seen.

## Functional requirements

- **FR-1 (open & walk)** — `Open(data)` validates the 20-byte file header (magic `M7R␀`, `hdrLen == 20`,
  `recordCount >= 3`, `formatVersion <= 1001` and not `1000`), then walks that many records by the
  `payloadSize` chain from 0x14 — reading each record header's `tag/hdrLen/payloadSize/typeId/perMapConst`
  and its payload. Every counted record's header and payload MUST lie inside the stream; bytes after the
  last one are neither read nor refused. It indexes records by `typeId` (the record header's `+0x0c` word),
  which is present for every record regardless of `payloadSize` — an empty (`payloadSize == 0`) record is
  typed directly from its header, with no elimination; a `typeId` at or above 10 indexes nothing and is
  stepped over; a repeated `typeId` overwrites. The accepted stream MUST carry type-0, type-1 and type-2;
  every other section is optional, absent type-3 being **manufactured** as a `W·H` zero plane and the rest
  decoding empty (see *Acceptance*). Records are addressed by `typeId`, **not** by physical position: the
  roster order is not enforced, and decoding does not depend on it (type-0 is located by its id and decoded
  before the grid/content size checks that need `W`/`H` and the counts).
- **FR-2 (decode metadata)** — decode the type-0 payload: `W` (`+0x00`), `H` (`+0x04`), `name` (`+0x30`,
  ASCII → UTF-8), `description` (`+0x78`, CP1251 → UTF-8), and the three content counts `#type5/#type4/#type6`
  (`+0x1c/+0x20/+0x24`, `ALM-CNT-017`). The `angle` float (`+0x08`), the record header's `perMapConst`, and
  the remaining located u32 scalars are exposed as raw values only — no named semantic meaning (R-1).
- **FR-3 (decode grids)** — decode type1/type2/type3 into `W × H` grids of raw cells (u16 / u8 / u8) at
  payload+0. The payload is **pure grid** — every cell is real terrain data (no identifier overlay; the
  `typeId`/`perMapConst` are record-header fields). The documented cell meanings (type1 tile-index bits 0–9
  + impassable bit 13; type2 height; type3 nonzero = object-occupied) are recorded; terrain-class
  resolution and the passability grid are **not** built here (mapload tier).
- **FR-4 (decode content sections)** — decode the content sections to the level the research establishes:
  - **type5** (roster) — `#5 = meta+0x1c` records of 76 B; decode each record's `name` (ASCII → UTF-8, at
    +0x0c) and the `+0x08` scalar; require `76·#5 == payloadSize`.
  - **type6** (units) — `#6 = meta+0x24` records of 70 B; decode each record's `X`/`Y` (u32 `/256`, at
    +0/+4), its class keys `classKey` (**i16**, +0x08, sign-extended) and `classKey2` (u16, +0x0a), and the
    two override conditions `flags` (u32, +0x0c) and `defID` (u32, +0x10) — all raw and unresolved; require
    `70·#6 == payloadSize`. The rest of the 70-byte read map is not exposed.
  - **type4** (objects) — `#4 = meta+0x20` base records of 20 B; decode `X`/`Y`/`kind` (+0/+4/+8) and the
    raw ctor fields; append an 8-byte extension for a record iff its `kind == 0x21`; consume the payload
    exactly; the extension pair is retained raw.
  - **type7 / type9** — decode the leading `entryCount` / `count` (`@ +0`) and retain the node/record
    blocks as **unmodified body bytes**. **type8** — retained raw (may be empty). The serialized leaf
    grammar of type7/8/9 is not decoded (R-2).
- **FR-5 (atomic rejection, never panic)** — reject with an error and no map: a bad magic; a file-header
  `hdrLen ≠ 20`; a `recordCount < 3`; a `formatVersion == 1000` (the record-header-skipping dialect this
  reader does not implement, and the error MUST say so rather than call the stream invalid) or
  `> 1001`; a counted record's header or payload past EOF; a `tag ≠ 7` or record `hdrLen ≠ 20`; a missing
  type-1, type-2 or type-0 record; a type-0 `payloadSize ≠ 632`; a type1 grid `payloadSize ≠ 2·W·H` or a
  present type2/type3 grid `payloadSize ≠ W·H`; a present type5 `payloadSize ≠ 76·#5` or type6
  `payloadSize ≠ 70·#6`; a present type4 base+extension walk that does not consume the payload exactly; a
  present type7/type9 payload too short to hold its 4-byte count word. A size constraint binds a record
  that is **present**; an absent record is not a size mismatch. Rejection is atomic — an error and no
  partial map, never a crash or out-of-bounds read.
- **FR-6 (purity)** — package `formats/alm` imports only stdlib + `golang.org/x/text` (Windows-1251 for
  the description); bytes in, structs out; no VFS/engine/sim/render knowledge; no floats in control flow,
  no global state.
- **FR-7 (dump tool)** — `cmd/almtool` exposes at least `info` (magic, `hdrLen`/`recordCount`/`formatVersion`,
  per-record `typeId` + `payloadSize`, the tiling check), `meta` (W, H, name, description, content counts),
  a grid dump, and a content summary (type5 names, type4/type6 record counts + sample X/Y, type7
  `entryCount`), for developer-run verification against a lawful install — never part of the test suite.

## Acceptance criteria (synthetic maps, no GOG assets)

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a synthetic minimal map: 20-B file header + ten records in order `0,1,2,3,5,4,9,8,6,7` for a 2×2 map (type-0 632 B with W=2, H=2, name, description, `#4/#5/#6`; type1 8 B; type2/type3 4 B; type4/5/6 sized to their counts; small type7/8/9), no trailer | opened | header validates; W=2, H=2; name/description/counts decode; each grid is 2×2 of the exact cells written (no overlay); records indexed by typeId from their headers |
| AC-2 | unit | a wrong magic; separately a file-header `hdrLen ≠ 20`; separately a `recordCount < 3`; separately a `formatVersion == 1000`; separately a `formatVersion > 1001` | opened | error, no map, each; the 1000 error names an unimplemented dialect rather than an invalid stream |
| AC-3 | unit | a counted record outside the stream: a `payloadSize` overrunning EOF; separately a `recordCount` naming a record whose header is not there | opened | error, no map, each |
| AC-4 | unit | a record with `tag ≠ 7`, or record `hdrLen ≠ 20` | opened | error, no map |
| AC-5 | unit | a type-0 `payloadSize ≠ 632`; separately a stream with no type-1 record, one with no type-2 record, and one with no type-0 record | opened | error, no map, each |
| AC-6 | unit | a type1 grid sized ≠ `2·W·H`; separately a type2 grid sized ≠ `W·H`; separately a hostile type-0 `W`/`H` whose `2·W·H` would overflow a 64-bit product | opened | error, no map, each — the overflow case rejects atomically with no panic and no oversized allocation (P-3) |
| AC-7 | unit | type7/8/9 node bodies with arbitrary bytes | opened | `entryCount`/`count` decode; the node/record bodies are retained raw by `typeId`, bytes unmodified |
| AC-8 | unit | a type-0 with a NUL-padded `name` (+0x30) and a `description` (+0x78) carrying a CP1251 high byte | opened | name/description decode (CP1251) to the expected trimmed strings — expected value expressed as Unicode code points, not literal Cyrillic (synthetic input, not GOG data) |
| AC-9 | manual | a real GOG map (a root `.alm`, and one `N.alm` from `scenario.res`; lawful install, developer-run) | `almtool info` / `meta` / content summary | the records tile the file; W/H/name/description/counts read; per-record sizes match `2WH`/`WH`/632/`76·#5`/`70·#6`; type5 names + type7 `entryCount` read; the type1 grid's first cells are real tile words, not the header identity; recorded as evidence in `verification.md` — no game bytes committed |
| AC-10 | unit | a type5 payload of `#5` 76-B records with NUL-terminated ASCII names at +0x0c and a scalar at +0x08; separately a type5 `payloadSize ≠ 76·#5` | opened | names decode to the expected strings and the `+0x08` scalar reads; the mismatched size → error, no map |
| AC-11 | unit | a type6 payload of `#6` 70-B records carrying `X`/`Y` at +0/+4, class keys at +0x08/+0x0a — one record's primary key written with the high bit set — and the override words at +0x0c/+0x10, one record with `flags` bit 0 set and one with a `defID` nonzero and `≠ 0xcdcdcdcd`; separately a `payloadSize ≠ 70·#6` | opened | `X`/`Y` decode as `/256` fixed-point (`tile = X>>8`); `classKey` decodes **signed** (the high-bit record reads negative, not ≈65 000) and `classKey2`/`flags`/`defID` decode as written; the mismatched size → error, no map |
| AC-12 | unit | a type4 payload: `#4` records where one record has `kind == 0x21` (an 8-B extension follows) and the others do not; separately a type4 payload that leaves ≠ 0 bytes after the walk, and a hostile `#4` far larger than the payload can supply | opened | the walk decodes `#4` records, attaches the extension pair raw only for the `kind==0x21` record, and consumes the payload exactly; the non-exact payload and the hostile count each → error, no map, no panic |
| AC-13 | unit | a type7 with `entryCount @ +0` and node bytes; a type9 with `count @ +0` and records; a type8 that is empty (`payloadSize = 0`) | opened | `entryCount` and `count` decode; bodies retained raw; the empty type8 is accepted |
| AC-15 | unit | a map whose type8 `payloadSize` is **0** (no marker tree); separately a map with an empty (`payloadSize = 0`) type4 and `#type4 = 0` | opened | each record is typed from its own header — the empty type8 decodes as `typeId = 8` with an empty body, the empty type4 decodes as zero objects; every record reads its own id in physical order; no elimination, no error; both are reported **present**, distinct from absent |
| AC-16 | unit | streams the loader accepts and this reader used to refuse: three records `{0,1,2}`; four records `{0,1,2,3}`; a type-0 whose `#type4/#type5/#type6` name records the file does not carry; a record with `typeId = 10`; a repeated `typeId`; bytes after the last record; a `formatVersion` of 1001 | opened | accepted, each. The three-record map yields a `W·H` all-zero overlay and empty content sections; the inert counts decode as written and conjure nothing; the `typeId = 10` record is part of the frame and no section; the repeated id resolves **last-wins**; the trailer does not disturb the decode |
| AC-17 | unit | the three-record map of AC-16, and the four-record one | opened | the overlay plane is `W·H` cells in both, and the map reports type-3 **absent** in the first and **present** in the second — so a caller can tell a manufactured plane from a read one |

*(AC-14 — the pre-EXP-0030 "empty type8 typed by elimination against the other nine ids" — is retired:
the corrected framing carries every record's `typeId` in its header, so an empty record needs no
elimination. Its ID is not reused.)*

## Derived properties

- **P-1** (invariant) On success every counted record lies inside the stream:
  `20 + Σ(20 + payloadSize) <= len` over the `recordCount` records, and no two records overlap. Bytes
  past the last record belong to no section and are not interpreted.
- **P-2** (invariant) Each decoded grid has exactly `W × H` cells — the overlay plane included, whether
  it was read from a type-3 record or manufactured in its absence.
- **P-3** (negative-invariant) For any input the reader never panics or reads out of bounds; any FR-5
  condition returns an error and no map.
- **P-4** (invariant) The type7/8/9 node/record bodies and the type4 extension bytes round-trip
  byte-identically.
- **P-5** (invariant) On success the decoded content counts agree with the record strides:
  `len(type5) == meta+0x1c` and `76·that == size5`; `len(type6) == meta+0x24` and `70·that == size6`;
  `len(type4 base records) == meta+0x20`.

## Research needed

> **RESEARCH NEEDED — type-0 scalar & float semantics [R-1].** The type-0 offsets/types/ranges are
> game-proven (`ALM-META-008`) and the three content counts at `+0x1c/+0x20/+0x24` are decoded
> (`ALM-CNT-017`), but the *meanings* of the remaining located u32 scalars (`+0x0c…+0x14`, `+0x18` low-bit
> bitmask, `+0x28` read-and-discarded, `+0x2c`, `+0x70/+0x74`), the `angle` float (`+0x08`, read as a
> stored light/sun angle — physical role Medium), the per-map `perMapConst`, and the 448-B trailing
> `"<None>"` text slots are not fully decoded. **Non-blocking:** the reader decodes `W`, `H`, `name`,
> `description`, the counts, and exposes the rest as raw values with no named meaning. **To resolve, tell
> the research team:** read these as named map properties in the Map Editor / `Description *.ini`.

> **RESEARCH NEEDED — trigger-tree leaf grammar [R-2].** The **serialized named-node leaf grammar** of
> type7/8/9 (parameter/enum/target encoding, nesting) is not decoded. The type4/5/6 *field ids* have left
> this item and this note no longer claims them: the type-6 record is mapped field by field
> (`ALM-UNIT-040`), its class keys and owner are named (`ALM-CLS-038`, `ALM-OWN-039`), and the type-4 fields
> — the `kind == 0x21` extension among them, a footprint override in tiles — are named on the same evidence
> (`ALM-OBJ-019` as amended). What is still open there is narrow and named: the roles of type-6 `+0x1c`,
> `+0x20..+0x2b` and `+0x40` (located as read destinations only), the type-4 `+0x12` word, and the type-5
> `{0,5000}` scalar. **Non-blocking:** the type7/8/9 bodies and the type-4 extension are preserved raw; the
> decoded type-6 fields this reader leaves unexposed are a scope choice, not a gap in the research.
> **To resolve, tell the research team:** reverse the type7/8/9 leaf grammar from the corpus + the `rom.exe`
> named-node accessors (`ALM-CODE-023`).

> **RESEARCH NEEDED — EXP-0030's open doors [R-4].** Three items EXP-0030 explicitly left open: (a) the
> **last 8 bytes of the type-7 payload** (all-zero on the 10 standalone maps, small ints on the campaign
> maps — what EXP-0007 called the "trailer"); (b) whether any **`formatVersion == 1000`** map (the variant
> that skips record headers) exists anywhere — none is shipped; (c) the `dataSize = 4·W·H + 72` identity
> holds 38/38 but the **`+72` is no longer explained** by the corrected framing (a legacy-layout
> conjecture). **Non-blocking:** the reader reads `dataSize` and ignores it, preserves the type-7 payload
> raw, and rejects the `formatVersion == 1000` shape as out of scope. **To resolve, tell the research
> team:** the EXP-0030 residuals section.

*(R-3 — the pre-EXP-0030 "section-header `f0/f1` and the 8-byte trailer, meaning unknown" — is retired:
`ALM-FRAME-031` proved neither exists. There are no `f0/f1` header words and no trailer; the bytes so
labelled were the record header's own `typeId`/`perMapConst` and the tail of the type-7 payload. Its ID is
not reused.)*

## Out of scope

- Reading a map *into* the simulation — terrain-class resolution (tile index → terrain via the `rom.exe`
  strip→pair + blend tables and the `world.res:data/map.reg` `Terrain` vocabulary), the runtime
  passability/occupancy grid, altitude → sim height, and entity spawning — the mapload/data tier on top of
  this reader (needs `map.reg`, outside a pure `formats` leaf; ALM-GRID-012/013/014, ALM-TERR-015/016).
- Resolving a type-6 class key against `units.reg` or `scenario/npc.reg`, mapping a type3 overlay code, a
  type4 `kind`, a type4/6 owner, or a type5 diplomacy value to game meaning — registry cross-refs, the data
  tier downstream.
- Executing triggers / decoding the type7/8/9 serialized leaf grammar — R-2 and its consuming tier.
- Other `.alm` dialects/versions — the `formatVersion == 1000` header-skipping variant (a second framing,
  unexercised by any shipped map) is refused as unimplemented, and so is a `formatVersion` above the
  loader's own gate of 1001.
- **Widening the reader to everything the loader tolerates.** Three refusals are kept deliberately and
  argued in *Acceptance*: a missing type-0, any payload-size mismatch, and the 1000 dialect.
- Rendering, lighting; writing/encoding `.alm`.

## Gate check

FR-1 → AC-1, AC-2, AC-3, AC-4, AC-5, AC-15, AC-16, AC-17, P-1 · FR-2 → AC-1, AC-8 · FR-3 → AC-1, AC-6,
AC-17, P-2 · FR-4 → AC-7, AC-10, AC-11, AC-12, AC-13, P-4, P-5 · FR-5 → AC-2, AC-3, AC-4, AC-5, AC-6,
AC-10, AC-11, AC-12, P-3 · FR-6 → (archtest DAG: `pkg/formats/alm` → stdlib + x/text) · FR-7 → AC-9.
