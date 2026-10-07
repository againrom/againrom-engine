# Verification — ALM map container, terrain grids & content sections (ROM1)

This records the **EXP-0030 corrected-framing revision** (T6 reframe + T7 real-map evidence). Earlier
revisions of this file (git history) verified the pre-EXP-0030 reader; the baseline here is the working
tree at the start of this revision.

## Environment

- Toolchain: Go 1.26.1 (`windows/amd64`), the version pinned in `go.mod`.
- Dependencies: none new. `pkg/formats/alm` imports the standard library (`bytes`, `encoding/binary`,
  `fmt`, `math`) plus `golang.org/x/text/encoding/charmap` (`Windows1251` for the type-0 description) —
  `x/text v0.40.0` was already required. `cmd/almtool` imports the standard library (`fmt`, `os`) plus
  `pkg/formats/alm` only.
- The unit suite runs with **no game install present** — every fixture is a byte stream built in test code
  from the corrected format contract. A CP1251 high byte is written as hex (`0xC0`), and the decoded
  `description` is asserted as a Unicode code point built with `string(rune(0x0410))` — no literal Cyrillic
  character appears anywhere (AC-8).
- AC-9 requires a lawful GOG "Rage of Mages" (ROM1) install. One was present on the owner's machine at
  `C:\Program Files (x86)\GOG Galaxy\Games\Rage of Mages` (10 root `.alm` + `scenario.res`); AC-9 was
  **executed** — evidence only (counts, sizes, names, a grid head sample); no game bytes committed.

## Commands

```
go build ./...
go vet ./...
go test ./...
gofmt -l $(git ls-files '*.go')
go test ./internal/archtest/          # the fail-closed DAG
bash scripts/check-no-game-assets.sh            # tree scan
bash scripts/check-no-game-assets.sh --history  # full-history scan
```

Gate results (final state, clean testcache):

```
### build      -> OK
### vet        -> OK
### test ./... -> ok  againrom/pkg/formats/alm      (13 tests, all sub-cases)
                  ok  againrom/cmd/terraintool       (downstream, re-encoded fixtures, unchanged asserts)
                  ok  againrom/cmd/mapview           (downstream, re-encoded fixtures, unchanged asserts)
                  ok  againrom/internal/archtest
                  ok  againrom/pkg/render/terrain · pkg/ui · pkg/formats/res · spr256 · game · camera · notices
### gofmt -l   -> (empty; clean)
### archtest   -> ok  againrom/internal/archtest
### asset guard (tree)    -> check-no-game-assets: clean (tree scan)
### asset guard (history) -> check-no-game-assets: clean (history scan)
```

`pkg/formats/alm` and `cmd/almtool` were already in the fail-closed `internal/archtest` allow-map and in
`docs/ARCHITECTURE.md`; no DAG or architecture edit was required, and none was made. `archtest` confirms
`pkg/formats/alm` imports only stdlib + `golang.org/x/text` and `cmd/almtool` only `pkg/formats/alm` +
stdlib (FR-6 / SC-13).

## Acceptance criteria

| AC | Method | Evidence / outcome |
|---|---|---|
| AC-1 | Unit `TestDecodeMinimalMap` | A 2×2 map (20-B file header, ten records in order `0,1,2,3,5,4,9,8,6,7`, **no trailer**) decodes: `HdrLen=20`, `RecordCount=10`, `FormatVersion=990`; `W=2/H=2`; name/description/counts; `Angle=π/4`; `SelectorA` = the record-header perMapConst; grids are the **exact written cells** (`Tiles={0x2005,0x0006,0x0007,0x2008}` — no overlay), `TileIndex(0x2005)=5`/`Impassable=true`; ten records indexed by header typeId; type1/2 payloadSizes 8/4. **PASS** |
| AC-2 | Unit `TestRejectBadHeader` | A wrong magic, a `hdrLen≠20`, a `recordCount<3`, a `formatVersion==1000` and a `formatVersion>1001` each return an error and a nil `*Map`. The 1000 error names the record-header-skipping dialect as unimplemented rather than calling the stream invalid. **PASS** |
| AC-3 | Unit `TestRejectRecordOutsideTheStream` | A last-record `payloadSize` inflated past EOF, and separately a `recordCount` of 11 over ten present records so the eleventh header is not there, each return an error and a nil `*Map`. Trailing bytes are no longer in this criterion — they moved to AC-16 as an **accept** case. **PASS** |
| AC-4 | Unit `TestRejectBadRecordHeader` | A record with `tag≠7`, or record `hdrLen≠20`, returns an error and a nil `*Map`. **PASS** |
| AC-5 | Unit `TestRejectRecordRoster`, `TestRejectMissingRequiredRecords` | A type-0 `payloadSize≠632` returns an error and a nil `*Map`; so do streams `{0,2,3}` (no type-1), `{0,1,3}` (no type-2) and `{1,2,3}` (no type-0). A duplicated typeId and a `typeId=10` left this criterion for AC-16, where they are accepted with defined behaviour. **PASS** |
| AC-6 | Unit `TestRejectGridSizes` | A type1 grid `≠2·W·H`, a type2 grid `≠W·H`, **and a hostile `W=H=0xFFFFFFFF` whose `2·W·H` wraps a uint64** each return an error and a nil `*Map` with no panic and no oversized allocation (the P-3 overflow case the adversarial read found). **PASS** |
| AC-7 | Unit `TestRawBodiesRoundTrip` | type7/8/9 bodies of arbitrary bytes (incl. high bytes `0xC0..0xFF`): `EntryCount`/`Count` decode and `Triggers.Body`/`Markers.Body`/`TileMarkers.Body` round-trip byte-identically (P-4). **PASS** |
| AC-8 | Unit `TestDecodeNameAndCP1251Description` | A NUL-padded ASCII `name="Level"` (+0x30) and a description (+0x78) of `{0x41, 0xC0}` decode to `"Level"` and `"A"+U+0410` — expressed as `string(rune(0x0410))`, never literal Cyrillic. **PASS** |
| AC-9 | Manual / developer-run (`almtool`) | **PASS** — see "AC-9 real-map evidence" below: a root map and a campaign map decode with exact tiling, correct counts/names/sizes, an exact type4 `kind==0x21` walk, and the record-header identity bytes gone from the tile grid. Evidence only; no game bytes. |
| AC-10 | Unit `TestType5Roster` | Two 76-B type5 records decode `{"Self",0}`/`{"Enemy",5000}` (name @+0x0c, scalar @+0x08); a payload `≠76·#5` returns an error and a nil `*Map`. **PASS** |
| AC-11 | Unit `TestType6Units` | Two 70-B type6 records decode `X`/`Y` (+0/+4) as `/256` fixed-point (`tile=X>>8`); a payload `≠70·#6` returns an error and a nil `*Map`. **PASS** |
| AC-12 | Unit `TestType4ExtensionWalk` | A type4 payload where record 0 has `kind==0x21` (8-B extension follows) and record 1 has `kind=0x07` (none): decodes two objects, `Objects[0].Ext` = the 8 bytes, `Objects[1].Ext=nil`, X/Y/Kind/ctor fields read, payload consumed exactly (P-5). A non-exact payload, and a hostile `#4=0x7FFFFFFF` with one record present, each return an error and a nil `*Map` with no panic (P-3). **PASS** |
| AC-13 | Unit `TestTriggerSectionsCountsAndRawBodies` | A type7 (`entryCount@+0`), a type9 (`count@+0`), and an empty type8 (`payloadSize=0`) decode the counts, retain the bodies raw, and accept the empty type8; a type7 of only 2 bytes (too small for its count word) returns an error. **PASS** |
| AC-15 | Unit `TestEmptyRecordsTypedFromHeader` | A map whose type8 `payloadSize` is `0` decodes: the record is typed from its own header as `TypeID=8` (`PayloadSize=0`, empty `Markers.Body`), every id present in physical order — no elimination. Separately an empty (`payloadSize=0`) type4 with `#4=0` decodes to zero Objects. **PASS** |
| AC-16 | Unit `TestAcceptsWhatTheLoaderAccepts` (7 subtests), `TestDocumentPreservesBytesNoRecordCovers` | Every stream the loader takes and this reader used to refuse is accepted: three records `{0,1,2}`; four records `{0,1,2,3}`; a type-0 declaring `#5=7`/`#4=415`/`#6=1815` with none of those records present, which decodes the counts as written and yields **zero** objects, groups and units; a `typeId=10` record, which stays in `Records` (11 entries) and reports `Present(10)=false`; a second type-3 record, where `Overlay` resolves to the **last** one's cells; bytes after the last record, which do not disturb the decode; and `formatVersion=1001`. The document half round-trips a stream carrying both an undecodable record and a trailer byte-for-byte. **PASS** |
| AC-17 | Unit `TestAcceptsWhatTheLoaderAccepts`, `TestDocumentOverAShorterRoster` | On the three-record map `len(Overlay)=4` (`W·H`) with every cell zero and `Present(3)=false`; on the four-record map `Overlay` equals the record's own cells `{00,07,00,09}` and `Present(3)=true`. `Document.Write` reproduces both inputs byte-for-byte, so the manufactured plane never reaches the bytes. **PASS** |

*(AC-14 — the pre-EXP-0030 "empty type8 typed by elimination" — is retired: the corrected framing carries
every record's typeId in its header, so an empty record needs no elimination. Its ID is not reused; AC-15
replaces its coverage with the header-typed behavior.)*

## Properties

- **P-1 (every counted record inside the stream).** On success each of the `recordCount` records has its
  header and payload in bounds and no two overlap; the walk advances by `20 + payloadSize` with all offset
  math in `int64`. Bytes past the last record belong to no section and are not interpreted. Exercised by
  `TestDecodeMinimalMap` (positive), `TestRejectRecordOutsideTheStream` (overrun + a counted record that is
  not there) and `TestAcceptsWhatTheLoaderAccepts` (the trailer, now an accept case).
- **P-2 (each grid is W×H cells).** `len(Tiles)==len(Altitudes)==len(Overlay)==W·H`; the minimal map
  asserts `len==4` and the exact cell values, and a **present** grid whose `payloadSize` disagrees is
  rejected (`TestRejectGridSizes`). The overlay plane holds whether it was read or manufactured:
  `TestAcceptsWhatTheLoaderAccepts` asserts `len(Overlay)==W·H` on a map with no type-3 record, which is
  the case that would otherwise leave a short plane and silently disable every consumer sizing by it.
- **P-3 (no panic / no out-of-bounds on any input).** Every enumerated malformed class returns
  `(nil, error)`; all bounds are checked before each read, all sizing is `int64`/`uint64`, the type1 size
  is checked as `payloadSize/2 == W·H` (never `2·W·H`, which could wrap), and the type4 walk grows by
  `append`. Exercised across the `TestReject*` tests, the grid-overflow case, and the type4 hostile-count
  case; a panic would fail these.
- **P-4 (raw bodies / ext round-trip).** The type7/8/9 bodies and the type4 `kind==0x21` extension are
  returned as fresh copies of the exact input bytes; `TestRawBodiesRoundTrip` and `TestType4ExtensionWalk`
  assert byte-for-byte equality.
- **P-5 (counts agree with strides).** `len(Groups)==meta+0x1c` and `76·that==size5`;
  `len(Units)==meta+0x24` and `70·that==size6`; `len(Objects base records)==meta+0x20` with the type4 walk
  consuming the payload exactly. Exercised by `TestType5Roster`, `TestType6Units`, `TestType4ExtensionWalk`.

## Independent-context review gates

The SDD profile's three gates were run by **independent subagents** (fresh context each), and their
material findings were reconciled before this revision landed:

- **Peer-prediction (Phase 1):** a zero-context reader given `spec.md` alone reproduced the container
  layout, the header-typed walk, the pure-grid base, the `kind==0x21` discriminator, the `>>8` conversion,
  and the full rejection list with high fidelity (well above the ~80% bar). It caught a garbled
  `kind==0x08==0x21` phrasing, a missing `formatVersion` clause in FR-5, and an implicit type-0-first
  sequencing assumption — all corrected in the spec before the gate closed.
- **Adversarial plan read (Phase 2):** a separate-context reviewer (no access to the authoring
  conversation) found a real **P-3 overflow hole** — the literal `payloadSize == 2·W·H` check wraps a
  `uint64` for a hostile `W·H ≈ 2^63`, letting a small payload pass and then panicking `make`. Fixed by
  deriving the count from the payload (`payloadSize` even ∧ `payloadSize/2 == W·H`), pinned in DD5/DD9 and
  covered by the AC-6 overflow case. It also flagged that DD11's "unchanged downstream" claim was stated as
  fact rather than an SC-17-gated outcome, and that the automated suite is spec-derived on both sides — both
  reconciled (DD11 recast as gated; AC-9 executed as the ground-truth anchor).
- **Separate-context tests (Phase 4):** the suite was authored by a subagent from `spec.md` + the public
  API, without reading the reader diff. **Green-but-hollow audit:** the tests assert exact decoded values,
  not shapes — the minimal map pins the exact pure-grid cells (proving no overlay), the type4 test pins the
  `kind==0x21` extension AND `Ext==nil` on the non-extended record, the grid test drives the 64-bit
  overflow, AC-8 uses hex + code points, and AC-15 pins the physical-order-preserving header typing. Each
  would fail on a framing regression.

## AC-9 real-map evidence (developer run, lawful install)

`almtool` (built from the T6 commit into the git-ignored `builds/0003-alm-container/`) was run against the
owner's install. Counts, sizes and names only — no game bytes.

**Root map `Islands.alm` (256×256).** `almtool info` reports the ten records with `hdrLen 20 / recordCount
10 / formatVersion 990` and `tiling: 20 + sum(20+payloadSize) = 301096 = file size (OK)`. Per-record
payloadSizes are identical to the pre-EXP-0030 run (632 / 131072 / 65536 / 65536 / 5740 / 456 / 532 / 0 /
30380 / 992) — the byte positions did not move, only the framing labels. `meta`: `256×256`,
`"Deadly Islands"`, `#5/#4/#6 = 6/287/434`, `angle=0.7853981` (π/4), `selectorA=-1.5013252` (the
`0xBFC02B6D` perMapConst, now read from the record header). `content`: groups `Self/Friend/Monsters/
Beasts/EnemyHumans/Guards` with their `{0,5000}` scalars; 287 objects walked exactly (kinds `0x23/0x06/
0x2b/…`); 434 units; `type7 entryCount=1`.

**The −8 correction is visible in the raw-body sizes.** type7 `body=988 bytes` (was 980) and type9
`body=528 bytes` (was 520): the corrected reader no longer loses the leading 8 bytes to a `[typeId][f32]`
prefix, because those live in the record header. Same eight-byte shift, in the direction EXP-0030 predicts.

**The decisive check — identity bytes gone (the EXP-0030 reproduction).** Before the fix,
`almtool grid tiles` printed the record-header identity `6d 2b c0 bf` as the head of grid row 0 (the
perMapConst read as tile cells: word `0x2b6d` = strip group 45, impossible; `0xbfc0` = bits 14–15 set).
After the fix, row 0 of `Islands.alm` reads uniform real tile words `0031/031 …` — **no `2b6d`/`bfc0`**.
Verified across all ten root maps: every first tile word is a valid small index with the high bits clear
(`Forester 00f4, Horror 00d4, Islands 0031, Kids 00f1, LuMoir 0011, Waters 0114, Beast 0114, Cross 0114,
Kids2 00f1, Tomb 0033`), and all ten tile exactly `(OK)`.

**Empty type8 now needs no elimination.** 8 of the 10 root maps ship `type8 payloadSize=0`
(Cross=90, Tomb=30 are the two non-empty, matching the historical corpus). Under the corrected framing they
are typed directly from their record headers — the pre-EXP-0030 elimination path (which existed only
because the old model put typeId in the payload) is gone, and all ten maps open with no special case.

**Campaign map `10.alm` (80×80, extracted from `scenario.res` to a scratch dir outside the repo).**
`info`: `tiling: 20 + sum(20+payloadSize) = 67554 = file size (OK)`, with a non-empty `type8`
(payloadSize 170) and a large scripted `type7` (37428 B). `content`: groups `Self/Villagers/Rogues/Beasts/
Nocturnal`; 18 objects; row 0 tile head `00d1/0d1 …` (identity bytes gone here too). The extracted `.alm`
lives outside the repository and was deleted after the run; the asset-guard tree scan is clean.

## Downstream impact (cross-package consequences of the framing correction)

- **Mechanical fixture re-encodes (absorbed, labelled).** `cmd/terraintool/main_test.go` and
  `cmd/mapview/main_test.go` build synthetic `.alm` byte streams in `buildALM`; both were re-expressed under
  the corrected framing (20-B file header; record headers carrying `typeId`/`perMapConst`; pure payloads;
  `objectRecord` at `X@+0`/`Y@+4` with `kind≠0x21`; no trailer). The synthetic maps and **every asserted
  value are unchanged** — SC-17 confirmed: both suites pass with no expectation edited. This is a byte-move
  of the same maps, not a behavioral change.
- **Same-bytes relabel (absorbed, labelled).** `cmd/terraintool` `-maplight` seeds the sun from two stored
  type-0 scalars; EXP-0030 shifts their labels −8, so `m.Meta.Word18/Word1C → Word10/Word14`. These read the
  **same real-map bytes** (file offsets +0x38/+0x3c), and `m.Angle` (file +0x30) is likewise byte-preserved,
  so `-maplight` is bit-identical on real maps. A no-op relabel forced by the `Meta` field rename.
- **Escalated (NOT absorbed) — downstream real-map evidence re-verification.** The framing correction shifts
  what the alm reader hands to `pkg/render/terrain` on **real** maps: the tile grid moves by the corrected
  base (type1 +4 cells / type2,3 +8 cells) and the header identity cells vanish. The render/UI **code is
  unaffected** — it operates on the decoupled `terrain.Grid` abstraction and has no overlay-corner carve-out
  in code — but any downstream story whose `verification.md` recorded a **real-map render/screenshot** did so
  against the 4-cell-shifted, identity-corrupted grid. The orchestrator should schedule real-map
  re-verification (not a code change) for **0004 (terrain-viewer)**, **0006 (water-animation)**, **0007
  (terrain-lighting)**, and **0008 (structures-overlay)** — see the report for the per-story verdict.

## T9 — the acceptance widening (2026-07-30, pin `03a9448`)

Run with no game install present; every fixture is synthetic, the four-record case included — its
**shape** is taken from `ALM-CORP-060`'s figures, never its bytes.

- **The accept set moved, and both directions are pinned.** `TestAcceptsWhatTheLoaderAccepts` (7
  subtests) and `TestRejectMissingRequiredRecords` (3) are new; `TestRejectBadHeader` swapped
  `recordCount != 10` for `< 3` and gained a `> 1001` case; `TestRejectNonTiling` became
  `TestRejectRecordOutsideTheStream`, losing its trailing-bytes case to the accept side and gaining a
  counted-record-not-present one. The document's rejection table and `OpenInfo`'s moved with them.
- **A panic was fixed, not avoided.** `Document`'s span walk was a fixed ten-iteration loop justified by
  the comment "Open has proven the frame". Widening `Open` alone would have made the first accepted
  four-record stream read a `payloadSize` word past the end of the buffer — a slice-bounds panic on the
  very file the widening exists to accept, breaking FR-5 and P-2. `spans` is now a slice and the walk is
  bounds-checked; `TestDocumentOverAShorterRoster` is the regression test and asserts the three- and
  four-record streams write back byte-for-byte.
- **`pkg/mapedit` was the second half of that failure and is gated, not widened.** It reaches this
  acceptance directly through `alm.OpenDocument`. Its `frame` is a fixed `[10]span` indexed by `typeId`;
  `locate` leaves an absent id at the **zero span**, and `abs` then resolves that id's payload-relative
  offsets against offset 0 — so a setter addressing an absent section would read and write inside the
  **file header**, silently, on a stream that was accepted. `requireFullRoster` restates the ten-record
  rule as that package's own entry gate. It is **temporary and disclosed**, owned by
  `docs/0025-mapedit-model`, and it exists so that no state of the tree between this change and that one
  lets a widened stream reach `abs`. Its acceptance-equivalence test now carries an explicit *stricter*
  column — the divergence is asserted, not deleted — and checks the direction that matters: mapedit is
  never looser than the package it delegates to. Witnessed by
  `TestNewAcceptsExactlyWhatOpenDocumentAccepts` with a three-record and a nine-record stream, each
  accepted by `alm` and refused by `mapedit`.
- **No consumer changed.** `Overlay` is a full `W·H` plane in every accepted case, which is what keeps
  `pkg/render/terrain`'s `len(Overlay) != Width*Height` guard, `pkg/game`, `pkg/mapload` and
  `cmd/classdump` behaving as before; the remaining sections were already empty-safe. The full suite is
  green with no source change outside `pkg/formats/alm` and `pkg/mapedit`.

**What this did not settle.** The map list widens with the reader, so `ru/Horror.alm` becomes visible
where the game's own browser hides it (`ALM-META-058`) — an owner-ruled, disclosed divergence scheduled
separately, not implemented here. `0023`'s round-trip contract and `0025`'s model are likewise scheduled;
until `0025` lands, the gate above is what stands between them.

## Limitations and residual risks

- **Deliberately-undecoded regions are read raw, not interpreted (R-1/R-2/R-4).** The type-0 `angle` float
  and located `u32` scalars + 448-B slots (R-1), the type7/8/9 serialized leaf grammar and the type4
  extension-pair meaning (R-2), and EXP-0030's open doors — the last 8 bytes of the type-7 payload, the
  unexplained `dataSize +72`, and the unexercised `formatVersion==1000` variant (R-4) — are exposed verbatim
  or rejected, with no invented meaning. No STOP/research-team flag was raised: every byte the spec pins is
  decoded, every gap is preserved raw.
- **The type4 extension discriminator is now the definite `kind==0x21`** (`ALM-OBJ-034`), not the
  pre-EXP-0030 coordinate-OOB proxy; the walk still requires exact payload consumption as a safety net
  (`TestType4ExtensionWalk`). Confirmed on real maps: 287 objects on `Islands.alm` and 18 on `10.alm` walk
  with zero residue.
- **P-3 is covered by targeted negatives, not a fuzz corpus** — each enumerated malformed class (including
  the 64-bit grid-size overflow) has a rejection test; a broad fuzz sweep is a possible future strengthening.
- **Terrain-class resolution, the passability grid, object/owner identity, and trigger execution are out of
  scope** (the mapload/data tier and R-2's consuming tier); this leaf returns raw cells, coordinates,
  counts, and names only.

## Conclusion

Every automated acceptance criterion passes: AC-1…AC-8, AC-10…AC-13, AC-15, AC-16 and AC-17 by synthetic
unit tests, with P-1…P-5 exercised across them. AC-9 is **executed** against a lawful install — a root map and a campaign
map decode under the corrected framing with exact tiling, matching counts/sizes/names, an exact
`kind==0x21` type4 walk, the type7/9 raw bodies 8 bytes longer (the −8 correction), and the decisive proof
that the record-header identity bytes (`2b6d`/`bfc0`) are gone from the tile grid on all ten root maps. The
adversarial read caught and this revision fixed a real P-3 overflow before the reader landed; the corrected
framing also dissolves the pre-EXP-0030 elimination path (all ten root maps now open with no special case).
All gates are green with no game install present (build, vet, test, gofmt, archtest, and the asset guard on
both tree and full history); the `pkg/formats/alm` leaf stays within stdlib + `golang.org/x/text`, and no
game data entered the repository. The downstream render/UI code is unaffected; downstream **real-map
evidence** for 0004/0006/0007/0008 needs re-verification (evidence, not code) and is listed for the
orchestrator.
