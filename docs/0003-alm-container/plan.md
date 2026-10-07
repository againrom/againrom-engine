# Plan — ALM map container, terrain grids & content sections (ROM1)

**Intensity:** spec-anchored / static (PROFILE default for `pkg/formats/*`). **Terrain:** **brownfield**
for `pkg/formats/alm` and `cmd/almtool` — this revision corrects the shipped reader's decode behavior
(EXP-0030 proved the record framing was 8 bytes too early). The `terrain.Grid`-based render/UI packages
(`pkg/render/terrain`, `pkg/ui`, and the render half of `cmd/terraintool`/`cmd/mapview`) consume a
decoupled abstraction and are **not** changed. *Brownfield note:* the normal characterization discipline
(pin current behavior, then change) is **inverted** here — the shipped reader's current behavior *is* the
proven bug (the RED-class trigger), so pinning it would freeze the defect. Instead the synthetic suite is
**regenerated from the corrected `spec.md`** (separate context). *Honest limit of that:* the automated
suite is then spec-derived on both sides (code and tests read the same corrected spec), so a shared misread
would pass green — exactly how the retired suite passed on the wrong framing. The **ground-truth anchor is
therefore the AC-9 developer run against the lawful install**, which this revision **executes** (not merely
documents), recording the corrected per-record sizes and the decisive "identity bytes gone / grid read at
the corrected base" check as real-map evidence in `verification.md`. That run is manual (outside
`go test ./...`, golden rule 2), so it is a one-time observation, not a re-runnable regression — treated as
the confirming evidence it is, not overstated as a gate.

## Approach

Rewrite the pure `.alm` reader in `pkg/formats/alm` to the **corrected framing** of `spec.md`
(FR-1…FR-7, AC-1…AC-13/AC-15, P-1…P-5, R-1/R-2/R-4, Out-of-scope). `Open(data []byte) (*Map, error)`
validates the **20-byte file header** (magic, `hdrLen == 20`, `recordCount >= 3`, `formatVersion <= 1001`
and not `1000`), walks **that many** records from `0x14` by the `payloadSize` chain, and requires every
counted record to lie inside the stream (bytes past the last one are not read and not refused). Each record is typed from its **20-byte header** (`typeId` at record `+0x0c`,
present for every record including a zero-length payload — so there is **no** elimination). The reader then
decodes the type-0 632-byte metadata (`W@+0`, `H@+4`, `name@+0x30` ASCII→UTF-8, `description@+0x78`
CP1251→UTF-8, counts `#type5/#type4/#type6` at `+0x1c/+0x20/+0x24`, `angle@+8` and the other scalars raw),
the three `W×H` **pure** grids (type1 u16 / type2 u8 / type3 u8 at payload+0, no identity overlay), and the
content records (type4 objects with the **`kind==0x21`** extension walk, type5 roster `name@+0x0c`, type6
units `X/Y@+0/+4`, type7/9 leading count at `+0`, type8 raw). Malformed input is rejected atomically. The
package imports **stdlib + `golang.org/x/text`** only; no floats drive control flow; no global state. The
companion `cmd/almtool` is updated to the corrected framing (20-byte header, no trailer, per-record typeId
from the header). The `cmd/terraintool`/`cmd/mapview` **test** `.alm` builders are re-encoded mechanically
to the corrected framing.

**2026-07-27 revision (class keys).** A second, narrow pass over the same contract: expose a placed unit's
class keys and the two conditions under which the engine overrides them (DD12), and correct the notes that
still call that record's tail undecoded. Framing, grids and the other content sections do not move;
`cmd/almtool` is left alone deliberately, because the corpus witness for the new fields is 0016's
`classdump -sweep` — the story that declared the precondition — not a second developer run here.

`pkg/formats/alm` (stdlib + x/text) and `cmd/almtool` (`{pkg/formats/alm}`) are **already** in the
fail-closed `internal/archtest` allow-map and in `docs/ARCHITECTURE.md`, so no DAG or architecture edit is
required (and if one seemed required, that would be a red flag to stop).

## Facts verified during planning

Baseline = the working tree at the start of this revision (S-5), frozen here:

- **Shipped reader is the pre-EXP-0030 framing.** `pkg/formats/alm/alm.go` (landed via the original
  T1–T5) implements: a **12-byte** file header (`magic`, `version=20`, `dataSize`); a 20-byte section
  header `[f0][f1][tag][hdrLen][payloadSize]`; typing by the leading **payload** `u32`; a `[typeId][f32]`
  **overlay** on the first 8 bytes of every grid/content payload; an **8-byte trailer**; and an
  **empty-type8-by-elimination** path (with a `meta+0x34` cross-check). It builds and tests green. Every
  one of those framing facts is what EXP-0030 retracts.
- **EXP-0030 is the pinned research** (`research/` at `da54e6d`): `ALM-FRAME-031` (20-byte file header;
  per-record `[tag=7][hdrLen=20][payloadSize][typeId][f32]` + pure payload; no `f0/f1`, no trailer),
  `ALM-GRID-032` (grid base = record payload+0; the old base is +4 cells type1 / +8 cells type2/3),
  `ALM-PLACE-033` (anchor→cell is a bare `>>8`), `ALM-OBJ-034` (type4 extension discriminator `kind==0x21`),
  and the amended `−8` offset family (`ALM-META-008/024/025/026/027/028`, `ALM-CNT-017`, `ALM-UNIT-018`,
  `ALM-OBJ-019`, `ALM-GRP-020`, `ALM-TRIG-021/022`). Verified against the submodule.
- **DAG unchanged.** `internal/archtest/dag.go` already contains `"pkg/formats/alm": {}` and
  `"cmd/almtool": {"pkg/formats/alm"}`; `externalAllowed` already grants `golang.org/x/text` to
  `pkg/formats/*`. No `dag.go`/`ARCHITECTURE.md` change is needed. Confirmed by reading both.
- **x/text present.** `golang.org/x/text v0.40.0` is already required; `charmap.Windows1251` is available.
- **API blast radius is contained.** A grep confirms only `pkg/formats/alm` and `cmd/almtool` reference the
  `Section`/`Trailer`/`Version` surface being reshaped; the downstream consumers (`cmd/terraintool`,
  `cmd/mapview`) use the grid/coordinate fields (`Tiles`/`Altitudes`/`Overlay`/`Width`/`Height`/`Objects`/
  `Units`) that persist.
- **Downstream test fixtures encode the old framing.** `cmd/terraintool/main_test.go` and
  `cmd/mapview/main_test.go` each build a synthetic `.alm` byte stream in `buildALM` with the 12-byte
  header, `[f0][f1][tag][hdrLen][payloadSize]` section headers, the `[typeId][f32]` overlay copied onto each
  grid/record head, object records with `X@+8`/`Y@+12`, and an 8-byte trailer. Under the corrected reader
  those streams no longer parse. The synthetic maps they encode have **all-zero altitude/overlay grids**
  and type1 corner cells that resolve to the same present strip under both bases, so re-encoding them to the
  corrected framing leaves the **rendered output and every asserted value unchanged** — a mechanical
  re-encode, not a behavioral change.
- **Render/UI code is decoupled.** `pkg/render/terrain` and `pkg/ui` operate on the `terrain.Grid`
  (`Width`/`Height`/`Tiles`) type built by the `cmd` bridges; they have no dependency on the alm framing or
  on an overlay corner (no top-left carve-out in the code), so they need no change.
- **Real maps present** for AC-9 at the owner's lawful install (10 root `.alm` + `scenario.res`).

For the 2026-07-27 revision:

- **The pin moved to `e61153d`**, bringing the type-6 read map (`ALM-UNIT-040`), the class keys
  (`ALM-CLS-038`), the owner (`ALM-OWN-039`) and the re-audit of `REG-KEY-044`, which **retracted** the
  clause that a placement's `ID` must be translated into a section index. Read in the submodule: both rows
  are `● active`, `ALM-CLS-038` carries its own "corrected 2026-07-27" note, and `retracted.md` records the
  withdrawn clauses at the **High** they held while believed. The consumer is 0016's FR-3, whose declared
  precondition is exactly the class `ID` at file `+0x08`.
- **The NPC override is a *file* field, not a memory-only one.**
  `ALM-CLS-038` states that divert as `rec+0x48 & 1`, and `0x48 = 72 > 70`: an offset into the loader's
  `0x50`-byte in-memory struct, not into the file record, so that row read alone leaves it unobservable.
  `ALM-UNIT-040`'s read map closes it — the file word at `+0x0c` **is** what the loader stores at in-memory
  `+0x48` (bit 0 = NPC path), and the same row maps file `+0x0a` → in-memory `+0x0c`, the subscript that
  divert uses. The two rows use "record" for different coordinate systems and agree only when read against
  each other. `+0x0c` is four bytes wide because those reads sum to exactly 70 with no gaps.

## Files to touch

Module `againrom` (this repo):

- `pkg/formats/alm/alm.go` — **MODIFY**. Reframe the reader: 20-byte file-header validation (DD2), the
  ten-record tiling walk to EOF with **no trailer** (DD2), typing from the **record header** (DD3), the
  632-byte type-0 metadata decode at the `−8` offsets with the CP1251 description (DD4), the three pure
  `W×H` grids at payload+0 (DD5), the type4 `kind==0x21` extension walk (DD6), type5/type6 at the `−8`
  offsets and type7/8/9 count-then-raw-body (DD7), the revised FR-5 atomic-error set (DD8), overflow-safe
  sizing (DD9). Remove the elimination path (DD10), the `Trailer` field, and the section `f0/f1` fields;
  rename the header type `Section` → `Record`. Keep the `TileIndex`/`Impassable` accessors.
- `pkg/formats/alm/doc.go` — **MODIFY**. Replace the 12-byte-header / trailer / overlay description with the
  corrected 20-byte-header + per-record-header + pure-payload framing; keep the stdlib + `golang.org/x/text`
  (Windows-1251) leaf rule and the `spec.md` pointer.
- `pkg/formats/alm/alm_test.go` — **MODIFY**. Regenerate the synthetic suite from the corrected `spec.md`
  (separate context): 20-byte file header, per-record headers carrying `typeId`/`perMapConst`, pure grid
  payloads, object records at the `−8` offsets with the `kind==0x21` discriminator, no trailer. Retire the
  elimination test; add the empty-record-typed-from-header test. CP1251 high bytes as hex; decoded strings
  as Unicode code points (AC-8).
- `cmd/almtool/main.go` — **MODIFY**. `info`: print `hdrLen`/`recordCount`/`formatVersion`, per-record
  `typeId` + `payloadSize`, and the `20 + Σ(20+payloadSize)` tiling check (no trailer line). `meta`/`grid`/
  `content` unchanged in shape; `content` reads the corrected coordinate offsets via the reader.
- `cmd/terraintool/main_test.go` — **MODIFY** (mechanical). Re-encode `buildALM` to the corrected framing
  (20-byte file header; record header carries `typeId`/`perMapConst`; pure grid payloads with no overlay
  copy; `objectRecord` writes `X@+0`/`Y@+4` with a non-`0x21` `kind`; no trailer). The 4×3 fixture map and
  every asserted pixel/summary value stay identical.
- `cmd/mapview/main_test.go` — **MODIFY** (mechanical). Same re-encode of its `buildALM`; the fixture map
  and assertions stay identical.

No `internal/archtest/dag.go` or `docs/ARCHITECTURE.md` change (both already list the package and the tool).

The 2026-07-27 revision touches two of those files again and no others (`cmd/almtool` keeps its `content`
summary — see *Approach*): `pkg/formats/alm/alm.go` — **MODIFY**, widen `Unit` and `decodeUnits` (DD12) and
replace the `Unit` doc comment, which asserts an undecoded tail `ALM-UNIT-040` has published (a stale
"undecoded" note is worse than a missing field: it tells the next reader not to look);
`pkg/formats/alm/alm_test.go` — **MODIFY**, extend the type6 case to AC-11 as revised (SC-10).

## Design decisions

Every architectural choice is settled here; each records the alternative it beat.

- **DD1 — In-memory byte-slice decode; atomic `(*Map, error)`.** `Open([]byte)` parses a whole in-memory
  `.alm` stream and yields `(*Map, nil)` on success or `(nil, error)` on any rejection — never a partial
  `Map` (FR-5). *Rejected:* an `io.Reader`/streaming decode — the metadata needed to size grids and content
  walks lives in one record but the tiling invariant spans the whole file, and a leaf over an in-memory
  buffer gains nothing from streaming.
- **DD2 — Counted walk, every record in bounds.** Validate the 20-byte file header (magic;
  `dword@0x04 == 20` = `hdrLen`; `dword@0x0c >= 3` = `recordCount`, the loader's own gate; reject
  `dword@0x10 == 1000` and `> 1001` = `formatVersion`). The `formatVersion` value is **otherwise read-and-exposed, not gated on its exact
  value below the gate** — the corpus is uniformly `990`, and the reader rejects the `== 1000`
  header-skipping dialect it does not implement plus anything above the loader's `1001`; it does not
  reject `991`/`1001`. Walk from `0x14`; for each counted record read the 20-byte header (`tag`,
  `hdrLen`, `payloadSize`, `typeId`, `perMapConst`), require `tag == 7` and `hdrLen == 20`, and advance
  by `20 + payloadSize`, requiring every step to stay `≤ len` (P-1). All offset
  math in `int64` so a hostile `payloadSize` cannot wrap. *Rejected:* trusting the header `dataSize`
  (`4·W·H + 72`) to bound the walk — the loader never reads it and its `+72` is unexplained (R-4); the
  tiling invariant is the authoritative structural check. *Rejected:* enforcing the corpus physical order
  `0,1,2,3,5,4,9,8,6,7` — FR-5 does not list order, so requiring it would reject files the spec accepts.
- **DD3 — Type each record from its header; index by `typeId`.** Read `typeId` from the record header
  (`+0x0c`) and `perMapConst` from `+0x10`; index records by `typeId` into a fixed `[10]` table — the size
  of the loader's own jump table, so a `typeId ≥ 10` indexes nothing and is stepped over, and a repeat
  overwrites (last-wins), both as the loader does them. The table carries a **present** flag per id, which
  is what lets an absent record be a decision rather than an empty payload. Because `typeId` is a header
  field it is present for a zero-length payload, so an empty record is typed directly. *Rejected:* the pre-EXP-0030 typing from the
  leading **payload** `u32` — that was the framing bug (`ALM-FRAME-031`: the `typeId`/`f32` are the last two
  words of the 20-byte header, not the payload head), and it is exactly what forced the elimination path for
  empty payloads.
- **DD4 — Type-0 metadata: decode named fields, expose the rest raw.** From the 632-byte type-0 payload
  (rejecting `payloadSize ≠ 632`) decode `W`/`H` (`+0x00`/`+0x04`), `name` (`+0x30`, 64 B, NUL-terminated,
  ASCII→UTF-8), `description` (`+0x78`, 64 B, NUL-terminated, **CP1251**→UTF-8 via `charmap.Windows1251`),
  `angle` (`+0x08`) as a raw `float32`, and the three content counts `#type5`/`#type4`/`#type6`
  (`+0x1c`/`+0x20`/`+0x24`). The remaining located `u32` scalars (`+0x0c…+0x14`, `+0x18` bitmask, `+0x28`,
  `+0x2c`, `+0x70`, `+0x74`) and the 448-byte `+0xb8` trailing slots are exposed as raw values in a `Meta`
  sub-struct — no invented meaning (R-1). The per-map `perMapConst` from the type-0 record header is exposed
  raw. A CP1251 decode never rejects the map (an unmapped byte substitutes) — string decoding is not in the
  FR-5 rejection set. *Rejected:* decoding `name` as CP1251 too — the spec assigns `name` ASCII and
  `description` CP1251, and for actual ASCII the two agree; *rejected:* asserting meanings for the raw
  scalars/floats — R-1 forbids it.
- **DD5 — Grids as `W×H` pure cells at payload+0; optional bit accessors.** Decode `type1` into `[]uint16`
  (`W·H` little-endian words, `payloadSize == 2·W·H`) and `type2`/`type3` into `[]uint8` (`W·H` bytes each,
  `payloadSize == W·H`), all from payload+0. Every cell is real terrain data — there is **no** identity
  overlay (the `typeId`/`perMapConst` are record-header fields, `ALM-GRID-032`). **Enforce the size
  invariants overflow-safely** (see DD9): compute `cells = uint64(W)*uint64(H)` (two `u32` factors — cannot
  wrap `uint64`); for type2/type3 require `uint64(payloadSize) == cells`; for type1 **do not compute
  `2·cells`** (which *can* wrap) — instead require `payloadSize` even and `uint64(payloadSize)/2 == cells`.
  Because each equality ties `cells` to the already-in-bounds `payloadSize`, the subsequent
  `make([]uint16, cells)` / `make([]uint8, cells)` is bounded by the input length. Package funcs
  `TileIndex(cell) = cell & 0x03ff` and `Impassable(cell) = cell & 0x2000 != 0` expose the type1 cell
  meaning (`ALM-GRID-012`). *Rejected:* the literal `payloadSize == 2·W·H` translation `uint64(payloadSize)
  == 2*cells` — a hostile `W·H ≈ 2^63` makes `2*cells` wrap to a small value that a small `payloadSize`
  matches, then `make` panics (a P-3 violation); the payloadSize-derived form above avoids it. *Rejected:*
  skipping or masking any leading cells — there is no overlay to skip; every cell is terrain (the
  pre-EXP-0030 "corner carries the id" note is void). *Rejected:* resolving tile index → terrain class —
  needs `map.reg`, out of a pure `formats` leaf.
- **DD6 — type4 `kind==0x21` extension walk, consuming the payload exactly.** Walk exactly `#type4` base
  records of 20 bytes, decoding `X`/`Y` (`+0x00`/`+0x04`, `u32` fixed-point; tile `= value >> 8`, an integer
  shift), `kind` (`+0x08`, `u32`), and the raw ctor fields (`+0x0c` u16, `+0x0e` u32, `+0x12` u16). Iff
  `kind == 0x21`, consume 8 more bytes (the extension, retained raw). The walk requires `cursor ==
  len(payload)` at the end (P-5, exact consumption). *Rejected:* the pre-EXP-0030 coordinate-OOB proxy —
  `ALM-OBJ-034` resolves the discriminator to a definite `kind == 0x21` (the proxy was a heuristic that
  mis-walked Cross/Horror); *rejected:* a fixed `20·#4` size assumption — the `kind==0x21` records carry the
  8-byte extension, so a rigid stride mis-tiles them.
- **DD7 — type5/type6 fixed records; type7/8/9 count-then-raw-body.** `type5`: `#type5` records of 76 B
  (rejecting `payloadSize ≠ 76·#5`); decode the `+0x08` scalar (raw) and the `+0x0c` NUL-terminated ASCII
  `name` (scanned within the record). `type6`: `#type6` records of 70 B (rejecting `payloadSize ≠ 70·#6`);
  decode `X`/`Y` (`+0x00`/`+0x04`, `u32` fixed-point) and, since the 2026-07-27 revision, the class keys and
  the two override words (DD12); the rest of the published 70-byte read map is dropped. `type7`/`type9`: read the leading count word at `+0x00` (`entryCount`/`count`, requiring
  `payloadSize ≥ 4`) and keep the bytes after `+0x04` as an unmodified `Body`. `type8`: keep the whole
  payload as an unmodified `Body` (empty when `payloadSize == 0` — there is no identity to skip). The type4
  extension bytes and the type7/8/9 bodies are the raw-preserved regions and round-trip byte-identically
  (P-4). *Rejected:* attempting the type7/8/9 serialized leaf grammar or the per-record ids — R-2 marks them
  undecoded.
- **DD8 — The FR-5 atomic-error set.** Reject with `(nil, error)` on: a bad magic; a file-header
  `hdrLen ≠ 20` or `recordCount < 3` or `formatVersion == 1000` or `> 1001`; a counted record's header or
  payload past `len`; a `tag ≠ 7` or record `hdrLen ≠ 20`; a missing type-0, type-1 or type-2 record; a type-0 `payloadSize ≠ 632`; a type1 `payloadSize ≠ 2·W·H` or a type2/type3
  `payloadSize ≠ W·H`; a type5 `payloadSize ≠ 76·#5` or a type6 `payloadSize ≠ 70·#6`; a type4 walk that
  does not consume the payload exactly; a type7/type9 `payloadSize < 4` (the count word does not fit).
  Rejection is atomic — no partial `Map` escapes and no path panics or reads out of bounds (P-3).
  *Rejected:* lenient recovery that returns a best-effort partial map — it would drop the FR-5/P-1/P-3
  guarantees the exact-tiling format enforces.
- **DD9 — Overflow-safe sizing; no floats in control flow.** All bounds and stride arithmetic is `int64`/
  `uint64`; sizes are derived by equality against the already-in-bounds `payloadSize`, so every allocation
  is bounded by the input length. The one product that can wrap is the type1 **`2·W·H`**: it is enforced as
  `payloadSize` even ∧ `payloadSize/2 == W·H` (DD5), never as `payloadSize == 2·(W·H)`. The fixed-record
  strides `76·#5` / `70·#6` are a small constant × one `u32` (max ≈ `3.3e11`, no wrap) and are checked
  against `payloadSize` before any `make`; the type4 walk grows its slice by `append` (never a `make` sized
  from an untrusted count). Coordinate decoding uses the integer shift `>> 8`, never a floating `/256.0`;
  the only `float32` values (`angle`, `perMapConst`) are decoded via `math.Float32frombits` and returned
  inert — no branch reads a float (FR-6). *Rejected:* trusting `W·H` or `2·W·H` into `make` unchecked.
- **DD10 — RETIRED.** The pre-EXP-0030 decision "type a zero-length section by elimination against the other
  nine ids, gated on `meta+0x34`" is void: with the corrected framing every record's `typeId` is a header
  field (DD3), so a `payloadSize == 0` record is typed directly and needs no elimination, deferral, or
  metadata cross-check. The whole `emptyIndex`/elimination path, the `markersTypeID` special case, and the
  `meta+0x34` gate are removed. Its ID is not reused.
- **DD11 — Mechanical downstream fixture re-encode.** The `cmd/terraintool`/`cmd/mapview` test `.alm`
  builders are re-expressed under the corrected framing so `go test ./...` stays green: a 20-byte file
  header; each record header carries `[tag=7][hdrLen=20][payloadSize][typeId][perMapConst]`; grid/record
  payloads are pure (the `[typeId][f32]` overlay copy is deleted); `objectRecord` writes `X@+0`/`Y@+4` with
  a `kind (+0x08) ≠ 0x21`; no trailer. **Why this is expected to be behavior-preserving, and why it is a
  *gated* claim not a settled fact:** the shipped reader read each grid from its (old) payload byte 0, so
  the fixtures deliberately placed padding at the cells the `[typeId][f32]` overlay would land on (grid
  head) and the map's interesting cells *after* it, at the same logical index the reader consumed. Moving
  the identity 8 bytes out of the payload and into the record header leaves those interesting cells at the
  same index; the only per-cell change is the type1 grid-head cell(s) whose value the overlay had set (e.g.
  `0x0001`→`0x0000`), and both resolve inside the same present `tile1-00` strip, which `solidStrip` renders
  as one flat colour — so no sampled pixel, placeholder count, or object/unit count moves. **This is
  verified, not assumed:** SC-17 re-runs both downstream suites with their expected values **unchanged**;
  if any expectation moves, that is a genuine downstream contract change and is **escalated** as its own
  story revision (R-5), never edited to make the test pass. *Rejected:* leaving the downstream tests red —
  that breaks the repo-wide gate; *rejected:* treating "unchanged" as a foregone conclusion rather than an
  SC-17-gated outcome.
- **DD12 — Expose the type-6 class keys *and* the two conditions that override them; expose nothing else of
  the record.** `Unit` gains `ClassID int16` (+0x08, decoded **sign-extended** because the loader `MOVSX`es
  it), `ClassSubID uint16` (+0x0a), `Flags uint32` (+0x0c) and `DefID uint32` (+0x10). All four are raw:
  this leaf resolves nothing and takes no registry dependency — which is also why its confidence suffices:
  `ALM-CLS-038` grades the field *identities* **High** (the corpus excludes the section-index reading
  outright, failing on 82 % of 8094 records) and reserves its Medium/Unknown for the resolution step this
  leaf does not take. *Rejected:* **the key alone, with the
  overrides bounded in spec prose** ("authoritative unless overridden; this reader cannot say when") — the
  cheaper edit, and it hands every consumer a question it cannot answer from what it was given. The named
  consumer (0016 FR-3) *counts* references that fail to resolve and exits non-zero when a type-6 one does,
  so with the key alone a legitimately overridden record is indistinguishable from broken game data and the
  sweep reports the shipped maps as broken; eight bytes per unit make "overridden" something it counts
  rather than a caveat it takes on trust. *Rejected:* **`uint16` for the primary key** — the loader sign-extends it and a `uint16` read
  turns a negative key into ≈65 000 silently; the shipped domain is `1..80`, so no fixture built from
  real-shaped values would catch it, which is why AC-11 demands a high-bit record. *Rejected:* **exposing
  the owner (`+0x14`), the `+0x18` index or the unique id (`+0x42`)** — `ALM-UNIT-040` publishes the whole
  map and none of the rest has a named consumer; exposing a field because research has it is how a leaf's
  contract grows without anyone deciding to. *Not asserted:* which override wins when both apply —
  `ALM-CLS-038` names the two paths and does not order them, and neither side needs the order: this reader
  resolves nothing, and *either* condition already tells a consumer that `units.reg` is not the table the
  engine consults. A stated bound, not written around.

## Risks (product)

- **R-1 (plan) — the type4 walk mis-tiles a payload.** *Mitigation:* the discriminator is now the definite
  `kind == 0x21` (`ALM-OBJ-034`), not a coordinate proxy; the walk still requires `cursor == len(payload)`
  exactly and rejects any non-exact consumption (DD6/DD8); the extension bytes are preserved raw; AC-12
  exercises a `kind==0x21` record, a non-extended record, a non-exact payload, and a hostile count.
- **R-2 (plan) — hostile sizes: allocation / wrap.** *Mitigation:* all offset/stride math is `int64`/
  `uint64` (DD9); counts and cell totals are validated by equality against the in-bounds `payloadSize`
  before any `make`; the type4 walk grows by `append`; the negative ACs assert an error and no panic (P-3).
- **R-3 (plan) — CP1251 description decode rejecting a map.** *Mitigation:* the `charmap.Windows1251`
  decoder substitutes an unmapped byte rather than erroring, and string decoding is deliberately outside the
  FR-5 rejection set (DD4); AC-8 exercises a defined high byte and asserts the decoded code points.
- **R-4 (plan) — stale framing leaking into the reader or its doc.** The shipped `doc.go`, `almtool`, and
  the reader constants still describe the 12-byte header / trailer / overlay. *Mitigation:* `doc.go` and
  `almtool` are rewritten to the corrected framing, the old constants (`fileHeaderSize=12`, `trailerSize`,
  `idOverlaySize`, `markersTypeID`) are removed, and the peer-prediction gate re-ran on the corrected spec.
- **R-5 (plan) — the downstream mechanical re-encode masks a real contract change.** *Mitigation:* the
  re-encode is validated by re-running `cmd/terraintool`/`cmd/mapview` tests with **unchanged expected
  values**; any expectation that has to move is escalated to the orchestrator as a downstream story revision
  (0004/0006/0007/0008), not absorbed as a fixture tweak (DD11).
- **R-6 (plan) — the exposed key is read as the whole answer.** `+0x08` lands inside `units.reg`'s `ID` set
  on 8094/8094 shipped records, so a sweep can come back clean with the overrides never exercised and the
  gap stays invisible until a map uses one. *Mitigation:* the two conditions are contract, not a footnote
  (DD12), and the `Unit` doc comment says so at the point of use. *Not mitigated here:* nothing in this
  story counts how many shipped records set either condition — no developer run is added and
  `verification.md` is unchanged. That census is 0016's `classdump -sweep`, under its own AC-9.

## Success criteria

Each maps to an upstream requirement and a verification method (unit = synthetic Go test; archtest = the
existing `internal/archtest` live-tree check; dev-run = `almtool` against a lawful install). SC-17 is the
exception: it maps to the repo-wide `go test ./...` gate (AGENTS.md) and the plan-internal R-5/DD11, not to
a `spec.md` requirement — the downstream packages are outside this spec's scope, and their re-encode is a
mechanical cross-package consequence of the framing correction, not a new contract.

1. **SC-1 (FR-1, FR-2, FR-3, AC-1, P-1, P-2; DD2–DD5)** — a synthetic minimal 2×2 map (20-byte header, ten
   records, no trailer) decodes: header validates; `W=2`, `H=2`; name/description/counts decode; each grid
   is `2×2` of the exact cells written (no overlay); records indexed by typeId from their headers. *Test:*
   `TestDecodeMinimalMap`.
2. **SC-2 (FR-1, FR-5, AC-2; DD2, DD8)** — a wrong magic, a `hdrLen ≠ 20`, a `recordCount < 3`, a
   `formatVersion == 1000` and one `> 1001` each yield an error and a nil `*Map`. *Test:*
   `TestRejectBadHeader`.
3. **SC-3 (FR-1, FR-5, AC-3, P-1; DD2)** — a `payloadSize` overrunning EOF, and separately trailing bytes
   after the tenth record, each yield an error and a nil `*Map`. *Test:* `TestRejectNonTiling`.
4. **SC-4 (FR-1, FR-5, AC-4; DD2)** — a record with `tag ≠ 7`, or `hdrLen ≠ 20`, yields an error and a nil
   `*Map`. *Test:* `TestRejectBadRecordHeader`.
5. **SC-5 (FR-1, FR-5, AC-5; DD3, DD4, DD8)** — a type-0 `payloadSize ≠ 632`, a duplicated `typeId`, and a
   `typeId = 10` each yield an error and a nil `*Map`. *Test:* `TestRejectRecordRoster`.
6. **SC-6 (FR-3, FR-5, AC-6, P-2, P-3; DD5, DD8, DD9)** — a type1 grid sized ≠ `2·W·H`, a type2 grid sized
   ≠ `W·H`, and a hostile type-0 `W`/`H` whose `2·W·H` would overflow a 64-bit product each yield an error
   and a nil `*Map` — the overflow case with no panic and no oversized allocation. *Test:*
   `TestRejectGridSizes`.
7. **SC-7 (FR-4, AC-7, P-4; DD6, DD7)** — type7/8/9 bodies built from arbitrary bytes: `entryCount`/`count`
   decode and the bodies round-trip byte-identically. *Test:* `TestRawBodiesRoundTrip`.
8. **SC-8 (FR-2, AC-8, P-1; DD4)** — a type-0 with a NUL-padded ASCII `name` (+0x30) and a `description`
   (+0x78) carrying a CP1251 high byte decode to the expected trimmed strings, the description asserted as
   Unicode code points (never literal Cyrillic). *Test:* `TestDecodeNameAndCP1251Description`.
9. **SC-9 (FR-4, AC-10, FR-5; DD7, DD8)** — a type5 payload of `#5` 76-B records with names at `+0x0c` and a
   scalar at `+0x08` decodes them; a `payloadSize ≠ 76·#5` yields an error and a nil `*Map`. *Test:*
   `TestType5Roster`.
10. **SC-10 (FR-4, AC-11, FR-5; DD7, DD8, DD12; R-6)** — a type6 payload of `#6` 70-B records decodes to
    AC-11's values, the primary key **signed** (a high-bit record reads negative, not ≈65 000); a
    `payloadSize ≠ 70·#6` yields an error and a nil `*Map`. *Test:* `TestType6Units`.
11. **SC-11 (FR-4, AC-12, P-5, FR-5; DD6)** — a type4 payload where one record has `kind == 0x21` (an 8-B
    extension follows) and the others do not decodes `#4` records with the extension attached raw only to
    the `kind==0x21` record and consumes the payload exactly; a payload leaving ≠ 0 bytes, and a hostile
    `#4`, each yield an error and a nil `*Map` with no panic. *Test:* `TestType4ExtensionWalk`.
12. **SC-12 (FR-4, AC-13; DD7)** — a type7 with `entryCount @ +0`, a type9 with `count @ +0`, and an empty
    type8 (`payloadSize = 0`): the counts decode, the bodies are retained raw, and the empty type8 is
    accepted. *Test:* `TestTriggerSectionsCountsAndRawBodies`.
13. **SC-13 (FR-6)** — `pkg/formats/alm` imports only the standard library and `golang.org/x/text`;
    `cmd/almtool` imports only `pkg/formats/alm` and stdlib; the fail-closed DAG check stays green. *Test:*
    the existing `internal/archtest` live-tree check.
14. **SC-14 (FR-7, AC-9)** — `almtool info` / `meta` / `grid` / `content` over a real GOG map (a root
    `.alm`, and one `N.alm` from `scenario.res`) report the ten-record tiling, `W`/`H`/name/description/
    counts, per-record sizes, type5 names and type7 `entryCount`, and that the type1 grid's first cells are
    real tile words (the header identity bytes are gone), recorded as evidence — no game bytes committed.
    *Method:* developer-run (manual), validated on a synthetic map here.
15. **SC-15 — RETIRED** (mapped to the retired AC-14 elimination case). Its ID is not reused.
16. **SC-16 (FR-1, FR-5, AC-15; DD3, DD10)** — a map whose type8 `payloadSize` is `0`, and separately a map
    with an empty (`payloadSize = 0`) type4 and `#type4 = 0`, each decode: every record is typed from its
    own header (empty type8 → `typeId = 8`, empty body; empty type4 → zero objects), every record reads
    its own id, no elimination and no error. *Test:* `TestEmptyRecordsTypedFromHeader`.
17. **SC-17 (R-5; DD11)** — the re-encoded `cmd/terraintool`/`cmd/mapview` `.alm` fixtures pass their
    existing tests with **unchanged expected values** (summary lines, sampled pixels, object/unit counts),
    confirming the framing correction is behavior-preserving for the render bridges. *Method:* the existing
    `cmd/terraintool` / `cmd/mapview` test suites, re-run after the re-encode.
