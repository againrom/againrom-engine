# Analysis — ALM map container, terrain grids & content sections (ROM1)

## Read this first — what in this file is history (index added 2026-08-01, pin `130bb79`)

**Everything below is preserved exactly as written and a good deal of it is dead.** This file is
the story's own record of what was believed in July 2026, and it is worth keeping: it is why the
reader, the spec and the tests read the way they do. But it is full of statements that are easy to
quote as current — a 12-byte header, a mandatory trailer, exactly ten sections, type ids inside the
payload — and none of them is true of the format. This table says which, so that can be seen
without reading the story chronologically. It is the same discipline `research/claims/retracted.md`
uses on the research side, and it governs: **where this index and the text below disagree, the
index is right.**

Ten of this file's claim ids now carry an overturn row in `claims/retracted.md`. The `Kind` column
there separates a claim that was **wrong** (REFUTED) from one that was **replaced by something
more precise** (SUPERSEDED); a dash means the audit has not classified it yet, which is not the
same as *neither applies*.

| What this file states | Status now | Read instead |
|---|---|---|
| **A 12-byte file header** (`ALM-HDR-001`, *Source & confidence*, `ALM-SEC-002`'s tiling identity) | Dead. The framing was split **8 bytes early** | The file header is **20** bytes and each record header is `[tag=7][hdrLen=20][payloadSize][typeId][f32]` — `ALM-FRAME-031` |
| **A mandatory 8-byte trailer**, `12 + Σ(20+payloadSize) + 8 == fileSize` (`ALM-TRL-005`, `ALM-HDR-006`) | Dead, Kind `—`. **There is no trailer**; the "trailer" was the 8-byte shortfall of a header split too early | `ALM-FRAME-031`. Bytes past the last record are simply never reached — the loop bound is `recordCount` |
| **Section type ids live in the payload's first `u32`** (`ALM-SEC-003`, `ALM-SEC-002`) | Dead. The `typeId` is a **record-header** field | `ALM-FRAME-031`; the payload is pure payload from byte 0 |
| **Exactly 10 sections, `{0..9}` once each, order `0,1,2,3,5,4,9,8,6,7`** (`ALM-SEC-003`) | Dead as a contract, Kind `—`. A writer regularity read as a rule | `ALM-REQ-055` / `ALM-REQ-056`: the only count gate is `recordCount >= 3`; the open path requires `{type1, type2}` and nothing else; a repeated id overwrites and an id ≥ 10 is seeked past. Order is unenforced but carries three real precedence relations (`ALM-ORD-057`). `ru/Horror.alm` ships **four** records and the loader accepts it (`ALM-CORP-060`) |
| **Grid payloads begin at payload+0, with `[u32 typeId][f32]` overlaid on the first 8 bytes** (`ALM-GRID-011`) | Dead, Kind `—`. Nothing is overlaid | `ALM-GRID-032`: the payload starts 8 bytes later. A decoder on the old base is **+4 cells off in X on type-1 and +8 on type-2/type-3** |
| **type-3 = an occupancy/"blocked" layer** (`ALM-GRID-014`'s role) | Superseded, Kind `—` | It is the **static-object placement layer**: code `c ≠ 0` is `objects.reg` section index `c − 1`, read at three renderer sites; occupancy is *derived* from it (`ALM-CLS-035`). The census re-runs exactly on the corrected base, so its stale-base flag was a false alarm |
| **type-1 bit 13 = the impassable flag** (`ALM-GRID-012`) | **SUPERSEDED**. It blocks, but it is the *smallest* of three arms | 7 464 cells against 136 622 for the strip-pair class and 87 584 for the raw water test, denominator 880 704 (`TERR-PASS-049`, `TERR-PASS-050`) |
| **`data/map.reg` terrain `Cost`/`Pass` figures** (`ALM-TERR-015`) | **SUPERSEDED** — withdrawn on a moved `.reg` framing, then **restored byte for byte**. Nothing was ever refuted | `ALM-TERR-043`, re-walked on the `0x18` framing, trailing `7` included. The `Pass` half is **inert**: 0 of the 10 `Pass*` key strings exist in either shipped binary |
| **One derived passability plane holding an enum** (`ALM-TERR-016`) | **REFUTED** on both counts | **Two** planes plus a copy, and the byte is a **bitmask**: a cell blocks a mover iff `block[cell] & mover.mask`. Under the enum reading no mover would ever be blocked anywhere (`TERR-PASS-049`…`TERR-PASS-051`) |
| **type-4 `kind` splits objects from structures across two registries** (`ALM-OBJ-019`'s label) | Superseded, Kind `—` | Every type-4 `kind` is a **`structures.reg` `ID`** (domain `1..66`, 65 distinct, never 0 over 3141 records); the split is between two **C++ classes**, not two registries. `objects.reg` is reached from the type-3 grid instead (`ALM-CLS-035`) |
| **type-8 = box/circle `X/Y` marker regions, "the geometry behind the *is unit in a box / circle* checks"; type-8 and type-9 read by a generic named-node parser** (`ALM-TRIG-022`, line 69) | **Dead in every part**, Kind `—`. See the paragraph below | `ALM-TRIG-049` / `ALM-TRIG-050` |

**type-8, since it is the one this file is most likely to be quoted for.** Three separate things
in that row are gone, and the replacement stops well short of a meaning:

- *Not a named-node tree.* Neither type-8 nor type-9 is parsed by the type-7 machinery, and neither
  record shape has a name field at all (`ALM-TRIG-049`, `ALM-TRIG-050`).
- *Not the trigger geometry.* The runtime checks that test a box and a radius take their corners
  and their radius from **the check node's own parameter slots**, and never touch a type-8 record.
  The **only** routine in the image that consumes type-8 or type-9 is `R0461`, whose own
  strings name it a **spellbook / building-caster** builder. The type-6 record's `+0x18` is a
  1-based link into this section, used by 43 records, and the loader copies that unit's id into the
  linked entry (`ALM-UNIT-048`).
- *No count word.* type-8's record count is `meta+0x2c`, the loader's own case-8 bound; the payload
  is that many records of `20 + 10n` with `n` the record's leading `u32`. The 20-byte head is a
  **format-version-990-only** shape — the loader reads the fifth head dword only when
  `formatVersion >= 0x3dd` and zeroes it otherwise — so a consumer meeting an older map reads 16
  (`ALM-TRIG-050`). type-9 is `[u32 count]` + `count × (26 + 6n)`, its count at payload **+0**, not
  `+8` (`ALM-TRIG-049`).

**What type-8 *means* is Unknown, and is written down as Unknown rather than re-guessed.** Both
rows are **Medium** and corpus-only — each is the unique model in its family that closes 38/38 —
and both state their own Unknown: every head field except the coordinates, and what a 6-byte or
10-byte element is. Naming this section after `R0461`'s strings would be a second guess
replacing the first. The reader is right to keep the payload raw, which is what FR-4 and AC-13
already require; what must not be repeated is a *role* sentence with no consuming instruction
behind it — that is precisely how the box/circle gloss survived from EXP-0018 to EXP-0080.

**Not fixed here, and owed:** `spec.md` carries the same reading in its own summary table
(`:110` "condition/**marker regions** — serialized tree") and at `:241` ("box/circle `X/Y`-bearing
serialized marker tree", citing `ALM-TRIG-022` by id), plus "the trigger/marker trees" at `:18` and
`:34`. That is contract text and a contract revision is not this sweep's to start. The behaviour is
unaffected — the reader keeps type-8 raw either way — but a summary table is what a consumer reads
first, which is the same defect research repaired in its own `formats/alm/format.md` on 2026-08-01.

## Source & confidence

All facts below are game-derived by the research team, not by this repo's own reverse-engineering:

- **Submodule pin:** `research/` at `aef1257` (`againrom-research`, module `rom1research`).
- **Experiments:** `research/experiments/EXP-0007-alm-container` (the 12-B header, the 20-B section chain,
  the trailer, the exact tiling), `EXP-0010-alm-grids` (the type-0 632-B metadata record and the
  type1/2/3 grid layers), `EXP-0019-alm-content` (the content sections type4–9: the object/group/unit
  fixed-record tables, the metadata count manifest, and the trigger/marker trees), and
  `EXP-0020-alm-grid-semantics` (the per-cell meaning of each grid, from `rom.exe` +
  `world.res:data/map.reg`), promoted into `research/formats/alm/format.md`.
- **Corpus & strength:** **38 maps** — the 10 standalone root `.alm` **+** the 28 embedded in
  `scenario.res` (`N.alm`, an `&YA1` archive — ALM-LOC-007). The container tiles the file exactly
  (`12 + Σ(20 + payloadSize) + 8 == fileSize`) with **0 gaps/overlaps/OOB** across all 38; grid
  cell-counts are exact **38/38**; `dataSize == 4·W·H + 72` holds 38/38; the fixed-record content strides
  are confirmed **twice** (tiling GCD **and** the type-0 count manifest) with type6 coordinates
  **8094/8094 in-bounds**; falsification passed. Established: the container framing, the type-0 metadata
  (incl. its content-count manifest), the type1/2/3 grid layers **with their per-cell semantics**, and the
  content sections type4–9 (fixed-record tables type4/5/6 + the top-level of the type7/8/9 trigger trees).
  Not decoded (upstream open): the serialized *leaf grammar* of type7/8/9, the type4 extension
  discriminator + per-record ids of type4/5/6, most type-0 scalar *meanings*, the section-header `f0/f1`,
  and the 8-byte trailer — all exposed raw by the reader.

## Claim inventory (what the spec is built on)

> **Re-grounded 2026-07-30, at pin `03a9448`.** This table is the story's own history and is left as
> written; two of its rows have since been overtaken and a reader must not build on them.
>
> - **`ALM-SEC-003` has lost its count, set and order clauses as a contract**, at the High they held
>   here. The loader's only count gate is `recordCount >= 3`; the open path requires `{type1, type2}`
>   and nothing else; the physical permutation is unenforced, though three precedence relations in it
>   are real (`ALM-REQ-055`, `ALM-ORD-057`). The refutation is from code and from shipped bytes, not
>   from a wider corpus reading: `ru/Horror.alm` carries **four** records and the loader accepts it
>   (`ALM-CORP-060`). Its surviving clause — `typeId` is the dispatch word — is unaffected, and was
>   already re-seated in the record header by `ALM-FRAME-031`.
> - **The corpus of 38 was the EN root plus `scenario.res`.** The union over both preserved roots is
>   44 names / 72 walked files, of which 71 carry ten records and one carries four (`ALM-CORP-060`).
>   Every 38/38 figure below is sound for the files it names and is not evidence about the RU root.
>
> What the reader does with that is a decision, not a consequence, and it is argued where decisions
> belong — `spec.md`'s *Acceptance* section. In short: it follows the loader wherever following it
> invents nothing, and stays deliberately stricter in three places.

| Claim | Statement | Confidence | Used by spec |
|---|---|---|---|
| ALM-HDR-001 | 12-B file header: magic `M7R␀` + `u32 version = 20` + `u32 dataSize = 4·W·H + 72`; `W,H` live in the type-0 payload | High | header (FR-1), W/H (FR-2) |
| ALM-SEC-002 | Length-prefixed section chain; 20-B header `[f0][f1][tag=7][hdrLen=20][payloadSize]`; tiles exactly `12 + Σ(20+payloadSize) + 8 == fileSize` | High | section walk, tiling (FR-1, P-1) |
| ALM-SEC-003 | Exactly 10 sections; typed by the leading payload `u32`; typeIds `{0..9}` once each, order `0,1,2,3,5,4,9,8,6,7`; every payload begins `[u32 typeId][f32 perMapConst]`; no ASCII tags | High | roster, typeId index (FR-1) |
| ALM-SEC-004 | Size profile: type0 = 632 B; type1 = `2WH`; type2/3 = `WH`; type4 = `20·#4`(+8·ext); type5 = `76·#5`; type6 = `70·#6`; type7/8/9 variable | High | size validation (FR-5) |
| ALM-TRL-005 | 8-B trailer (two u32): zero on the 10 standalone maps, small non-zero ints on the 28 campaign maps; meaning unknown | High (existence) / Unknown (meaning) | trailer preserved (FR-4, P-4); R-3 |
| ALM-HDR-006 | Section-header `f0`,`f1` located but uninterpreted (type0 `10,990`; grid layers `0,0`; else variable) | Unknown (meaning) | opaque header fields; R-3 |
| ALM-LOC-007 | `scenario.res` is an `&YA1` archive of 28 embedded campaign maps (`N.alm`) + 3 nested `&YA1` nodes; all 28 conform — 38 total | High | corpus scope; AC-9 |
| ALM-META-008 | type-0 632-B field layout, constant across 38 maps: `typeId@+0`, `f32 selectorA@+4`, `W@+8`, `H@+12`, `f32 angle@+10`, u32 scalars @`+14,+18,+1c,+20(bitmask),+24,+28,+2c,+30,+34,+78,+7c`, `char[64] name@+38`, `char[64] description@+80`, 440-B `"<None>"` slots @`+c0` | High (layout) / Unknown (most scalar meanings) | type-0 layout (FR-2); R-1 |
| ALM-META-009 | The two type-0 floats: `+0x10` = **angle** (radians ≈`[-π/4..+π/4]`, plausibly light/sun); `+0x04` = **3-value selector** `{-1.50133,-1.91545,-1.92320}` echoed into every section's `perMapConst` | High (fields/roles) / Low (physical meaning) | floats read raw; R-1 |
| ALM-META-010 | `+0x38` = ASCII **name** (64-B NUL-terminated); `+0x80` = **description** (64-B NUL-terminated; ASCII or Windows-1251) | High | name/description decode (FR-2, AC-8) |
| ALM-CNT-017 | The type-0 metadata is a **manifest of the fixed-record content counts**: `+0x24 = #type5`, `+0x28 = #type4`, `+0x2c = #type6` (38/38); the variable sections carry their own count word. Independent second proof of the 76/20/70 strides | High | counts decoded (FR-2, P-5); strides (FR-4/FR-5) |
| ALM-GRID-011 | The three grids store `W·H` cells at **payload+0** (`2WH` type1 u16 / `WH` type2/3 u8); the `[u32 typeId][f32]` id is **overlaid on the first 8 bytes** (makes `dataSize=4WH+72` exact); cell count exact 38/38 | High | grid decode (FR-3, P-2) |
| ALM-GRID-012 | **type1 = Tiles**: u16 word, **bits 0–9 = tile index**, **bit 13 (`0x2000`) = impassable**; bits 10–12/14–15 unused (0/880 552). Terrain *class* derived from the index by `rom.exe` (strip group bits 6–9 → terrain-pair table; blend variant bits 0–5). *(Supersedes the EXP-0010 low/high byte split.)* | High | type1 cell meaning (FR-3); terrain resolution out of scope |
| ALM-GRID-013 | **type2 = Altitudes (height)**: u8 (0..~246); named by the `rom.exe` loader, copied to the sim height buffer; Mountain-high/Water-low across 38 maps — a height field, not illumination | High | type2 cell meaning (FR-3) |
| ALM-GRID-014 | **type3** = sparse **object/feature occupancy overlay**: u8 (0=empty; 86–98% zero; 98 codes); the ingest marks every **nonzero** cell object-occupied. Only nonzero-ness drives movement; exact code→object identity not pinned | High (occupancy) / Low (per-code identity) | type3 cell meaning (FR-3); code identity out of scope |
| ALM-TERR-015 | Terrain vocabulary = `world.res:data/map.reg` `Terrain`: 10 classes (Land,Grass,Flowers,Sand,Cracked,Stones,Savanna,Mountain,Water,Road) + per-terrain `Cost`/`Pass`; `rom.exe` reads them into a per-terrain table | High | terrain resolution (out of scope, mapload) |
| ALM-TERR-016 | Runtime passability grid (`sim+0x10000`) derived from type1+type3: `1`=terrain-impassable, `5`=type3-occupied, `0x1f`=map border; type2 copied to the height buffer | High (mechanism) / Medium (edge cases) | passability grid (out of scope, mapload) |
| ALM-UNIT-018 | **type6 = placed units**: 70-B records, count `= meta+0x2c`; `X@rec+8`/`Y@rec+12` (u32 `/256`, `0x80`-centred); `0xFF` inventory sentinels; `70·count == size` and all 8094/8094 in-bounds (38 maps) | High (layout/tiling/xy) / Medium (label + non-coord fields) | type6 decode (FR-4, AC-11) |
| ALM-OBJ-019 | **type4 = placed objects/structures**: 20-B base (`+0≈100`,`+2 kind`,`+6 id`,`+8 X`,`+c Y`,`+10 value`), count `= meta+0x28`; some records append 8 B (2nd `[coord][value]`); adaptive base-20+extension walk (consume 8 B when the next record's X/Y would be OOB) consumes each payload exactly (36/37) | High (base/count/tiling) / Medium (extension + label) | type4 decode (FR-4, AC-12) |
| ALM-GRP-020 | **type5 = player/group roster**: 76-B records, count `= meta+0x24`; NUL-terminated ASCII **name** @ `rec+0x14` (Self/Monsters/Villagers/Neutral/Enemy/…); scalar `{0,5000}` @ `rec+0x10`; 38/38 exact | High (layout/tiling/name) / Medium (diplomacy semantics) | type5 decode (FR-4, AC-10) |
| ALM-TRIG-021 | **type7 = trigger effect/instant list + Drop**: `[u32 typeId=7][f32 perMapConst][u32 entryCount@+8][named-node blocks]`; each block a named editor node matching `Description Instants.ini`; default first entry `"Drop Location"`; 808/992 B skirmish → ~100 KB campaign | Medium (structure + vocabulary + rom.exe) / Low (leaf grammar) | type7 entryCount (FR-4, AC-13); R-2 |
| ALM-TRIG-022 | **type8/type9 = serialized marker trees**: type8 = box/circle `X/Y` marker regions (empty on skirmish); type9 = `[u32 count@+8][records]` tile-coordinate markers (`0x2376` word). Both read by the generic named-node parser | Low/Medium (roles + leading structure) / Unknown (leaf grammar) | type8 raw / type9 count (FR-4, AC-13); R-2 |
| ALM-CODE-023 | `rom.exe` map-load path: `%d.alm` built+loaded `R0099→R0473` (`CMap`); `R0065` reads groups (type5) + drop location (type7); `R0474` reads Outpost effects; via a generic named-node accessor family — confirming type7–9 are a serialized named-node tree | High (located) / Medium (field mapping) | section roles; R-2 direction |

## Derivation notes

A few non-obvious facts the reader is built on, straight from the research:

- **The file header is 12 bytes, not the section stride.** magic / version(`20`) / `dataSize`
  (ALM-HDR-001). `W`,`H` are read from the type-0 payload (`+0x08`/`+0x0c`), not the header. The section
  headers are a separate 20-B stride pinned by the stored `hdrLen = 20`; `tag = 7` and `hdrLen = 20` are
  the constant section-header signature.
- **A section's type is in its payload, not its header.** `typeId = u32(payload, 0)` (ALM-SEC-003). For
  the grid **and** content sections this `[typeId][f32]` is *overlaid* on the first 8 payload bytes
  (ALM-GRID-011, ALM-OBJ-019) — the grid array / record 0 starts at payload+0 and its first 8 bytes carry
  the id; record 0's fields at `+8`+ are still readable. A documented artifact, not a prefix to skip.
- **The type-0 metadata is a count manifest.** `+0x24/+0x28/+0x2c` are `#type5/#type4/#type6`
  (ALM-CNT-017) — the same values as `size/stride`, a second independent proof of the 76/20/70 strides.
  This is why the reader can drive and cross-check the fixed-record walks from the metadata.
- **type1's meaning is bit-fielded, not byte-split.** EXP-0020 supersedes the earlier byte-split reading:
  bits 0–9 are the tile index and bit 13 is an impassable flag (ALM-GRID-012). The terrain *class* is not
  stored — it is derived from the index by the loader via `rom.exe`'s strip→terrain-pair + blend tables
  and the `map.reg` `Terrain` vocabulary, so that resolution belongs to the mapload tier, not this leaf.
- **type4 objects are variable-length by a coordinate-continuation walk.** Base record is 20 B; a minority
  append an 8-B second `[coord][value]`. The research-validated walk consumes the extension **when the
  next record's decoded X/Y would fall out of bounds** (ALM-OBJ-019); a trailing 8 B after the last base
  record is its extension. The reader uses this rule and requires the payload to consume exactly; the
  extension's *meaning* and the exact extend-discriminator are open (R-2).
- **There is a mandatory 8-byte trailer.** The last section ends at EOF−8; two trailing u32 run to EOF
  (ALM-TRL-005). The exact-tiling invariant is `12 + Σ(20 + payloadSize) + 8 == fileSize`.
- **Strings.** `name` (type0 + type5) is ASCII; the type-0 `description` is code-page text (ASCII /
  Windows-1251) decoded as CP1251, which reproduces ASCII unchanged (ALM-META-010). Both convert to UTF-8.

## Scope notes carried into the spec

The research decodes the container, the type-0 layout + counts, the three grid layers with their cell
semantics, and the content record layouts; it deliberately leaves the following to later work, and the
reader either exposes them raw or defers them to a downstream tier rather than guessing:

- **type-0 scalar & float meanings** (R-1) — offsets/types/ranges proven (ALM-META-008/009), the three
  counts decoded (ALM-CNT-017), but the remaining scalars, the two floats' physical role, and the 440-B
  `"<None>"` slots are the subject of EXP-0018; read raw.
- **type7/8/9 serialized leaf grammar + content field ids** (R-2) — the trigger/marker trees are decoded
  to their leading count; the parameter/enum leaf grammar, the type4 extension discriminator, and the
  per-record owner/template/item ids (and the type5 `{0,5000}` scalar) are not decoded — the bodies are
  preserved raw.
- **`f0/f1` and the 8-byte trailer** (R-3) — located, meaning unknown (ALM-HDR-006, ALM-TRL-005); subject
  of EXP-0018.
- **Terrain resolution, passability, and object identity** — *not* a research gap but a **tier boundary**:
  resolving a tile index to a terrain class + building the passability grid (ALM-GRID-012, ALM-TERR-015/
  016) and mapping a type3 code / a type4/6 owner id / a type5 diplomacy value to game meaning need
  `map.reg` and the registries, so they belong to the mapload/data tier, not this pure `formats` leaf.

When the research team closes any open item, the flow is: repull the submodule
(`git submodule update --remote research`), derive the newly-decoded field here, then update `spec.md` to
state it as established and resolve the matching *Research needed* item.

## Appended 2026-08-02 — pin `01c64e2`: `ALM-OBJ-019` gains a contested clause, and R-2's `+0x12` item is two items

`ALM-OBJ-019` now reads `● active (amended, contested)`. The new **live** contradiction **C-7**
(`research/claims/registry.md`) is about where the type-4 `+0x12` word *goes*, not about what this reader
does with it: the row says it is **sign-extended into `obj+0x10`** of the live object, and
`TERR-STRUCT-075` reads `obj+0x10` as a **pointer** to a heap-allocated 12-byte position object — stored
at `L02083` in the base actor constructor and dereferenced by both footprint routines. EXP-0081 read
that store and both dereferences but never read `+0x12`'s own store, so it **picked neither side**; the
destination is now unsourced on both sides at Medium.

**Nothing here moves.** `spec.md` types `+0x12` as a raw `u16` ctor field and this reader keeps it raw;
the contested fact concerns a runtime structure `pkg/formats/alm` never builds. What does move is the
shape of **R-2**'s remaining open list, which names "the type-4 `+0x12` word" as one item. It is two: the
field's **role** is answered — the id a type-7 `Target_Structure` parameter names, 16/16 references
landing in this field's value set (`ALM-TRIG-046`) — and its **destination** in the live object is
contested. The index at the head of this file governs; this note extends it.

## Appended 2026-08-02 — pin `acb8fb0`: `ALM-OBJ-034` is superseded on a clause this reader never used, and two of the eight extension bytes are decoded

`ALM-OBJ-034` gains a `claims/retracted.md` row, classed **SUPERSEDED**, and it was **High**. What
falls is the row's last sentence — that the extension's "low bytes override the object's footprint"
**for that kind**. The gate is not the kind: `R0486` computes
`((arg1 & 0xff) + (arg2 & 0xff)) > 0` at `L07853`…`L07854`, and the kind byte is not an operand
of it (`TERR-STRUCT-090`). An extension record whose two extent bytes were both zero would take the
table arm instead; all 8 shipped records carry non-zero bytes in both, so nothing measured under the
old framing changes.

**The clause this story cites is the discriminator, and it is untouched.** `spec.md`, `plan.md` and
`verification.md` cite `ALM-OBJ-034` for `kind == 0x21` selecting an 8-byte extension — one `CMP`
immediate at `L02098` — which is what tiles the type-4 walk 38/38. The superseded clause is about
what the engine does with those bytes afterwards, in a live object this tier never builds; the
reader keeps `Ext` raw. The same holds for the four Go sites that name the id
(`pkg/formats/alm/alm.go` twice, both `main_test.go` fixtures): each cites the discriminator alone.

**`ALM-OBJ-062` decodes two of those eight bytes, so R-2's extension item narrows.** Only file
`+0x14` and `+0x18` reach an instruction — as `u16`s into the in-memory record's `+0x0a`/`+0x0c`,
then a low byte each into `obj+0x60`/`obj+0x61`, the footprint extents. **File `+0x16`, `+0x17`,
`+0x1a` and `+0x1b` reach no instruction on that path**, and each extent is effectively one byte
wide twice over. Nothing in `spec.md` moves — `Ext` is 8 raw bytes and stays so — but R-2's "the
type4 extension discriminator … not decoded" is now half answered, and the unanswered half is four
bytes that are **dead rather than unread**.

**And the axis `spec.md` already states is now carried by the image rather than by a census.**
`ALM-OBJ-061` withdraws `TERR-STRUCT-075`'s "the instructions do not say, the corpus does" and pins
type-4 `+0x00` as the **low** axis — the column — over a nine-instruction chain with no permutation
in it, at High. `spec.md`'s `X @ rec+0x00`, `Y @ rec+0x04` is what that chain gives. The statistic
it replaces could never have decided it: 37 of the 38 shipped maps are square.

**C-7 is unchanged and still live.** The note above it governs, and this one does not touch it.
