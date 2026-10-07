# Tasks — ALM map container, terrain grids & content sections (ROM1)

**Reading key.** `FR-x` / `AC-x` / `P-x` → `spec.md`; `SC-x` → `plan.md` §Success criteria; `DD-x` →
`plan.md` §Design decisions; `R-x` → `plan.md` §Risks. Task kinds: **implementation** (one coherent
product change → exactly one implementation commit, trailer `SDD-Task: 0003-alm-container/T<n>`),
**developer-run verification** (agent authors, a human runs against a lawful install; no implementation
commit).

**This is a RED-class revision.** Task IDs `T1`–`T5` (in git history) built the pre-EXP-0030 reader — the
12-byte-header / section-overlay / 8-byte-trailer / elimination framing EXP-0030 retracted. Those IDs are
**not reused**. The corrective revision is `T6` (reframe) + `T7` (real-map evidence). The
`pkg/formats/alm` and `cmd/almtool` DAG allow-map rows and `docs/ARCHITECTURE.md` entries already exist, so
**no** `dag.go`/`ARCHITECTURE.md` edit is part of any task. `T8` is the later, narrow 2026-07-27 revision
(type-6 class keys); it does not depend on `T7`.

## T6 — reframe the reader to the corrected EXP-0030 framing  *(implementation)*

Correct the container framing everywhere it is decoded, in **one coherent commit**. This is deliberately
not split: removing the `Trailer`/`f0/f1` surface makes `cmd/almtool` fail to compile, changing the framing
makes every synthetic `.alm` fixture (in-package and the two downstream tool tests) fail to parse, and the
reader is only correct together with its regenerated tests — so any intermediate commit would leave
`go build ./...` or `go test ./...` red (phase-4: never split work that is only correct together). The
`alm_test.go` regeneration is authored in a **separate context** from the reader diff (from the corrected
`spec.md`), and the two downstream test files are **mechanical re-encodes** (DD11), not behavioral changes.

- Files:
  - `pkg/formats/alm/alm.go` (MODIFY) — 20-byte file-header validation (magic, `hdrLen==20`,
    `recordCount==10`, reject `formatVersion==1000`) and the ten-record tiling walk to EOF with **no
    trailer** (DD2); typing from the **record header** `typeId`/`perMapConst` (DD3); the 632-byte type-0
    metadata at the `−8` offsets (`W@+0`, `H@+4`, `angle@+8`, counts `+0x1c/+0x20/+0x24`, `name@+0x30`,
    `description@+0x78` CP1251, slots `@+0xb8`) (DD4); the three **pure** `W×H` grids at payload+0 with the
    **overflow-safe** size checks (DD5/DD9) and `TileIndex`/`Impassable` accessors; the type4 `kind==0x21`
    extension walk (DD6); type5 `name@+0x0c`/`scalar@+0x08`, type6 `X/Y@+0/+4`, type7/9 `count@+0`, type8
    whole-payload raw (DD7); the revised FR-5 atomic-error set (DD8). **Remove** the elimination path
    (`emptyIndex`, `markersTypeID`, the `meta+0x34` gate — DD10), the `Trailer` field, and the section
    `f0/f1` fields; **rename** the header type `Section` → `Record` (and `Map.Sections` → `Map.Records`),
    reshape `Object` to `X/Y@+0/+4`, `Kind@+8` + raw ctor fields, and source `Map` header fields
    (`FormatVersion`, `RecordCount`, `SelectorA`/`perMapConst`) from the corrected framing.
  - `pkg/formats/alm/doc.go` (MODIFY) — replace the 12-byte-header / trailer / overlay description with the
    corrected 20-byte-header + per-record-header + pure-payload framing; keep the stdlib +
    `golang.org/x/text` (Windows-1251) leaf rule and the `spec.md` pointer.
  - `pkg/formats/alm/alm_test.go` (MODIFY, **separate-context authored**) — regenerate the synthetic suite
    from the corrected `spec.md`: 20-byte file header, per-record headers carrying `typeId`/`perMapConst`,
    pure grid payloads, object records at the `−8` offsets with the `kind==0x21` discriminator, no trailer.
    Retire the elimination test; add `TestEmptyRecordsTypedFromHeader` (SC-16) and the hostile-dimension
    grid case (SC-6). CP1251 high bytes as hex; decoded strings asserted as Unicode code points (AC-8); no
    Cyrillic character authored anywhere; no test reads a game install.
  - `cmd/almtool/main.go` (MODIFY) — `info` prints `hdrLen`/`recordCount`/`formatVersion`, per-record
    `typeId` + `payloadSize`, and the `20 + Σ(20+payloadSize)` tiling check (no trailer line); `meta`/
    `grid`/`content` keep their shape over the corrected reader.
  - `cmd/terraintool/main_test.go` (MODIFY, **mechanical re-encode**) — re-express `buildALM` under the
    corrected framing (20-byte file header; record header carries `typeId`/`perMapConst`; pure grid
    payloads, overlay copy deleted; `objectRecord` writes `X@+0`/`Y@+4`, `kind ≠ 0x21`; no trailer). The
    4×3 fixture map and **every asserted value stay identical** (SC-17); a moved expectation is escalated
    (R-5), not edited.
  - `cmd/mapview/main_test.go` (MODIFY, **mechanical re-encode**) — same re-encode of its `buildALM`;
    fixture map and assertions stay identical (SC-17).
- Covers: FR-1, FR-2, FR-3, FR-4, FR-5, FR-6; AC-1…AC-8, AC-10…AC-13, AC-15; P-1…P-5; DD1–DD9, DD11;
  R-1…R-5; SC-1…SC-13, SC-16, SC-17.
- **Done when:** `go build ./...`, `go vet ./...`, `gofmt -l $(git ls-files '*.go')` (empty),
  `go test ./...` (green, **no game install present** — all fixtures synthetic), the `internal/archtest`
  DAG check, and `bash scripts/check-no-game-assets.sh` (and once with `--history`) are all clean; the
  regenerated `pkg/formats/alm` suite (`TestDecodeMinimalMap`, `TestRejectBadHeader`, `TestRejectNonTiling`,
  `TestRejectBadRecordHeader`, `TestRejectRecordRoster`, `TestRejectGridSizes`, `TestRawBodiesRoundTrip`,
  `TestDecodeNameAndCP1251Description`, `TestType5Roster`, `TestType6Units`, `TestType4ExtensionWalk`,
  `TestTriggerSectionsCountsAndRawBodies`, `TestEmptyRecordsTypedFromHeader`) passes; and the downstream
  `cmd/terraintool` / `cmd/mapview` suites pass with **unchanged expected values** (SC-17).

## T7 — RETIRED. AC-9 evidence against a lawful install  *(developer-run verification)*

**Retired 2026-08-01. Its ID is not reused.** This entry cannot ever land, and had been reported
outstanding on every `check-sdd-audit` run since the convention changed under it. It is a
*developer-run verification* task written before **evidence became a pipeline stage rather than a
numbered task**: the work it describes is owed and was done, but it is now discharged by a commit
that carries **no trailer**, and the audit's bijection is over trailers. So a task id that no
commit may claim sat permanently in the "still to come" list.

Nothing is lost by retiring it and nothing is excused. The evidence this entry commissions is in
`verification.md` — the same place it says to put it — where **AC-9 is witnessed** and marked PASS
against a lawful install, with its own section. Retiring the entry retires the *task numbering*,
not the obligation. The alternative was to delete the entry, which would destroy the record of
what was asked for and why.

**SC-14 is a separate matter and is NOT closed by this.** It names the same real-map run and
`check-sdd-audit` reports it, with fourteen other SCs of this story, as witnessed by nothing in
`verification.md`. That is reported and not enforced only because 0003 predates `WITNESS_FROM`;
it is a real gap and retiring T7 neither fixes nor excuses it.

The work, kept as written for the record:

Run `almtool info` / `meta` / `content` / `grid tiles` against a real GOG root `.alm` and one `N.alm`
extracted from `scenario.res`, on the owner's lawful ROM1 install, and record **evidence only** — that the
ten records tile exactly under `20 + Σ(20+payloadSize)`, `W`/`H`/name/description/counts read, the
per-record sizes match `632`/`2WH`/`WH`/`76·#5`/`70·#6`, the type5 names + type7 `entryCount` read, and the
decisive **framing-correction check**: `almtool grid tiles` no longer prints the record-header identity
bytes (`6d 2b c0 bf`) as the head of grid row 0, and the tile cells/positions have shifted to the corrected
base. Never any game bytes.

- Produces: evidence in `verification.md` (Phase 5).
- Covers: AC-9; SC-14.
- **Done when:** `verification.md` records the real-map evidence including the identity-bytes-gone check; no
  game asset is committed and `bash scripts/check-no-game-assets.sh` stays clean.

## T8 — expose the type-6 class keys and the two conditions that override them  *(implementation)*

Widen the placed-unit record to what a class-resolving consumer needs, and correct the reader's own note
that calls that record's tail undecoded.

- Files:
  - `pkg/formats/alm/alm.go` (MODIFY) — add to `Unit`, decoded in `decodeUnits` from the same 70-byte
    record slice: `ClassID int16` (+0x08, **sign-extended** — read the `u16` and convert, do **not** declare
    the field `uint16`), `ClassSubID uint16` (+0x0a), `Flags uint32` (+0x0c), `DefID uint32` (+0x10). No
    resolution, no lookup, no new import; the existing `70·#6 == payloadSize` check and error text stand.
    **Replace the `Unit` doc comment** — it says the tail is undecoded and that is false. State what each
    exposed field is, that `ClassID` is a `units.reg` `ID` used as a subscript with no translation, that
    `DefID` nonzero and `≠ 0xcdcdcdcd` or `Flags` bit 0 each mean the engine resolves the record through a
    different table, that no precedence between those two is asserted, and that the remaining bytes are
    decoded upstream but deliberately not exposed (DD12).
  - `pkg/formats/alm/alm_test.go` (MODIFY) — extend `TestType6Units` to AC-11 as revised: records carrying
    the four new fields, **one whose primary key is written with the high bit set** and must read negative
    rather than ≈65 000, one with `Flags` bit 0 set, one with a `DefID` nonzero and `≠ 0xcdcdcdcd`. The
    existing `payloadSize ≠ 70·#6` rejection case stays. Synthetic bytes only.
- Covers: FR-4 (type6), AC-11; DD12; SC-10; R-6.
- **Done when:** the standing gates are clean with no game install present, `TestType6Units` asserts all
  four fields including the signed read of the high-bit record, and no package outside `pkg/formats/alm`
  is touched.

## T9 — accept what the loader accepts, and keep the edit model out of it  *(implementation)*

- **Files:** `pkg/formats/alm/{alm.go,document.go,doc.go}` (MODIFY) and their tests;
  `pkg/mapedit/editor.go` (MODIFY) plus `editor_test.go`/`arch_test.go`.
- Widen the header gate to `recordCount >= 3` and add the `formatVersion <= 1001` gate the reader
  lacked, keeping `1000` refused with an error that names it an unimplemented dialect. Walk the counted
  records, dropping the tile-to-EOF requirement; step over a `typeId >= 10` and let a repeat overwrite.
  Require `{type0, type1, type2}`; manufacture an absent type-3 as a `W·H` zero plane and decode every
  other absent section empty, with the type-0 counts inert. Expose per-`typeId` presence so a
  manufactured plane is distinguishable from a read one. Every payload-size constraint stays, binding a
  record that is present.
- `Document` widens in the SAME change, not after it: its span array becomes a slice and its walk
  becomes bounds-checked. A ten-iteration walk over a four-record stream reads a payloadSize word past
  the buffer, so widening `Open` alone would panic on the first file the widening exists to accept —
  against FR-5 and P-2.
- `pkg/mapedit` reaches this acceptance through `alm.OpenDocument` and cannot navigate it: its frame is
  indexed by `typeId` and an absent id resolves to the zero span, so a setter would read and write
  inside the **file header**. Restate the ten-record roster as that package's own entry gate, disclosed
  and temporary, until `docs/0025-mapedit-model` carries absence.
- Covers: FR-1, FR-5, AC-2, AC-3, AC-5, AC-16, AC-17, P-1, P-2; DD2, DD3, DD8; SC-2, SC-3.
- **Done when:** the standing gates are clean with no game install present; a three-record and a
  four-record map open, round-trip byte-for-byte and report type-3 absent/present; and the mapedit gate
  refuses both while `alm` accepts them.

## Traceability

| Requirement (spec) | Plan criterion | Task |
|---|---|---|
| FR-1 (open & walk, counted records, typeId from header) | SC-1, SC-2, SC-3, SC-4, SC-16 | T6; widened acceptance T9 |
| FR-2 (decode metadata, −8 offsets) | SC-1, SC-8 | T6 |
| FR-3 (decode pure grids) | SC-1, SC-6 | T6 |
| FR-4 (decode content sections, kind==0x21) | SC-7, SC-9, SC-10, SC-11, SC-12 | T6; type6 class keys T8 |
| FR-5 (atomic rejection, never panic) | SC-2, SC-3, SC-4, SC-5, SC-6, SC-9, SC-10, SC-11, SC-16 | T6; widened rejection set T9 |
| FR-6 (purity / DAG) | SC-13 | T6 |
| FR-7 (dump tool) | SC-14 | T6 (tool), T7 (dev-run) |
| AC-1 | SC-1 | T6 |
| AC-2 | SC-2 | T6, T9 |
| AC-3 | SC-3 | T6, T9 |
| AC-4 | SC-4 | T6 |
| AC-5 | SC-5 | T6, T9 |
| AC-6 (incl. overflow) | SC-6 | T6 |
| AC-7 | SC-7 | T6 |
| AC-8 | SC-8 | T6 |
| AC-9 | SC-14 | T7 |
| AC-10 | SC-9 | T6 |
| AC-11 (X/Y, class keys, overrides) | SC-10 | T6, T8 |
| AC-12 (kind==0x21 walk) | SC-11 | T6 |
| AC-13 | SC-12 | T6 |
| AC-15 (empty records from header) | SC-16 | T6 |
| AC-16 (the widened accept set) | SC-2, SC-3 | T9 |
| AC-17 (manufactured vs read type-3) | SC-3 | T9 |
| P-1 (exact tiling, no trailer) | SC-1, SC-3 | T6 |
| P-2 (each grid is W×H cells) | SC-1, SC-6 | T6 |
| P-3 (no panic / no OOB, incl. overflow) | SC-2, SC-3, SC-4, SC-5, SC-6, SC-9, SC-10, SC-11 | T6 |
| P-4 (raw bodies / ext round-trip) | SC-7 | T6 |
| P-5 (counts agree with strides) | SC-9, SC-10, SC-11 | T6 |
| R-1 (type4 walk mis-tile) | SC-11 | T6 |
| R-2 (hostile sizes) | SC-3, SC-6, SC-11 | T6 |
| R-3 (CP1251 decode never rejects) | SC-8 | T6 |
| R-4 (stale framing in reader/doc) | SC-1, SC-14 | T6 |
| R-5 (downstream re-encode masks a change) | SC-17 | T6 |
| DD12 (class keys + override conditions) | SC-10 | T8 |
| R-6 (the key read as the whole answer) | SC-10 | T8 |
