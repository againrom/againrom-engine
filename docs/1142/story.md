# Story 1142 — SAV field setters for the transplant ladder

## Why

`EXP-0262` compared an accepted and a refused city SAV: 196 fields and relations,
65 different, 35 with no producer outside a source SAV. `pipeline/SAV-OWNER-RUNS.md`
proposed bisecting the 65 by transplant: start from a document ROM1 wrote and
inject our writer's values one field group at a time, so the first rung that fails
names what breaks the load. That kit stopped at four rungs (rungs 0-1 and their
purse-only variants) because `pkg/formats/sav` exposed no setter for any other
field the table names — `reg` exported readers only, `Head.MapName` had no
setter, and the campaign record's own writer reminuted every object identity in
the document. This story adds those setters and builds the rungs they unblock.

## In-scope behaviour

Setters for four field groups, each proved to change ONLY the field it names:

1. **The embedded state store** — `reg.SetInt`/`reg.SetIntArray`
   (`pkg/formats/reg/write.go`), wrapped by `File.SetStoreInt`/`SetStoreIntArray`
   (`pkg/formats/sav/field_setters.go`). Addressed the same way `reg`'s own
   readers are: a section and a key. Covers `GameOptions/*`, `View/{X,Y}`,
   `Objects/Selection`, `SpellBook/Pressed` and any other leaf without a named
   wrapper.
2. **The campaign head** — `File.SetMapName` and `File.SetPlayerListField`.
   `SetMapName` resizes the head; every offset after it shifts, proved at the
   DECODED level (every other field's own value unchanged, only its byte
   position moves) rather than byte level, the same relationship a resized
   store array has to a later leaf. It also carries SAV-DECPAD-238's single
   trailing alignment byte across a length change of odd parity, which
   `len(Body)` staying even cannot by itself reveal (see Proof).
3. **The campaign record's scalars** — `File.SetCampaignScalar` (7 raw
   top-level dwords) and `File.SetCampaignBaseDWord` (6 dwords of the main
   record's own base).
4. **The campaign record's collections** — `File.SetCampaignArray` (6
   top-level uint16 lists), `File.SetCampaignDocuments`,
   `File.SetCampaignChildren`.

All four groups turned out BOUNDED. Groups 3 and 4 share one internal helper,
`File.editCampaign`, built on `parseCityCampaign`/`serializeCityCampaign`
(`city_campaign.go`) — a purely positional, non-reordering encode/decode pair
already in the codebase, used for round-tripping a campaign record rather than
generating one from scratch. That is what makes them bounded: this story adds no
new campaign-record codec, only mutation points into the existing one.

Half Two: rungs 2-5 of the seven-rung field-group ladder, staged in
`review/owner-run-sav-ladder-fields/`, on TWO bases (the original kit's byte-
identical original, and the owner's own `game0029.sav`, a city document ROM1
itself wrote from our framing). Rung 6 is not built — see Open debt.

## Interpretation of one ambiguous table cell

`pipeline/SAV-OWNER-RUNS.md`'s rung 5 row writes `Base.Arrays[0]` for the first
array and plain `Arrays[1..5]` for the rest. `cityCampaignBase.arrays` (the
"Base" struct) has only 2 elements (`AddHero`, `EnableMercenary`) — too few to
index at 5 — while `CityCampaignData`'s top-level `Arrays` has exactly six
(Mercenaries, PermanentMercenaries, InnNPC, InnMission, TCMission,
ShopMission), matching the row's index range. Read as
`CityCampaignData.Arrays[0..5]`, not `cityCampaignBase.arrays`; "Base." in the
row's own first cell is taken as a copy artifact of the table, not a distinct
struct reference. The row's own numbers (`[22]`, `[22],[30],[31]`) are
`EXP-0262`'s refused-file corpus, not this kit's base file — `game0025.sav`'s
own pre-rung-5 values are `Mercenaries=[14]`,
`PermanentMercenaries=[14 6 13 4 7 3 2 9 12]`, `InnNPC=[]`, `InnMission=[]`,
`TCMission=[]`, `ShopMission=[]`, already empty on three of the five touched
indices in this base. What rung 5 injects — every named index set to empty —
is unaffected by that difference; only the "before" column would differ if
restated against this base. Array index 4 (TCMission) is not in the row's list
and is left untouched by rung 5.

## Proof

### Setter boundedness — synthetic fixtures (committed)

`pkg/formats/reg/write_test.go` and `pkg/formats/sav/field_setters_test.go`: each
setter is exercised on a `synth.Reg`/`standard()` fixture, asserting (a) the
field reads back as set and (b) every byte outside the field's own span is
unchanged for a fixed-width edit, or every OTHER field's own DECODED value is
unchanged for a length-changing one. `go test ./pkg/formats/reg/... ./pkg/formats/sav/...`
is green.

### Setter boundedness — real corpus, measured

Committed as an env-gated test mirroring the existing
`save_document_corpus_test.go` pattern:
`pkg/formats/sav/field_setters_corpus_test.go` (build tag `savdocumentaudit`,
`AGAINROM_DOCUMENT_CORPUS` env var, no asset bytes committed). Instrument: open
the corpus file twice, apply one setter to one copy, `Marshal` both, count
differing bytes over the shared prefix plus the length delta.

Run against `review/owner-run-sav-ladder/EN/{game0023,game0025,game0026,game0027}.sav`
and `gameversions/saves/2027-09-07/{game0029,game0030}.sav` (the two files named in
the coordinator's addendum), six files, all PASS:

| setter | bytes differ (typical) | note |
|---|---:|---|
| `SetStoreInt` (`GameOptions/Speed`) | 1 | fixed-width, no shift |
| `SetStoreIntArray` (`Objects/Selection`→nil) | 75-83 | store-internal shift only; campaign tail untouched |
| `SetMapName` (+2-char suffix) | 1,884-3,917 | length-preserving-parity shift over the whole rest of the body; not a scatter — see below |
| `SetPlayerListField` | 1 | fixed-width |
| `SetCampaignScalar[2]` | 4 | fixed-width dword |
| `SetCampaignBaseDWord[1]` | 1 | fixed-width dword |
| `SetCampaignArray[0]`→nil | 45 | shifts arrays[1..5]+documents+scalars+markers |
| `SetCampaignDocuments`→nil | 36 | shifts scalars+markers |
| `SetCampaignChildren`→nil | 0 | trailing field; no shift |

Exact per-file numbers (instrument: `TestFieldSetters1142CorpusBoundedness`,
`-v` log, byte-diff of two whole `Marshal()` outputs):

`game0023.sav`/`game0025.sav` (3,223 B): Speed 1, Selection −4 len/75 diff,
MapName +2 len/2,077-2,078 diff, PlayerListField 1, Scalar[2] 4, BaseDWord[1] 1,
Array[0] −2 len/45 diff, Documents −24 len/36 diff, Children 0.
`game0026.sav`/`game0027.sav` (3,215 B): Selection 83 diff (this store has one
more leaf after `Objects/Selection` than 0023/0025's, `SpellBook/Shortcuts`, so
its shift footprint is 8 bytes larger), MapName 1,961-1,964 diff, others
identical to the 0023/0025 row. `game0029.sav` (3,277 B) and `game0030.sav`
(5,432 B): every row matches `game0025.sav`'s numbers exactly except MapName
(2,121 and 3,917 respectively — proportional to file size, as expected for a
whole-tail shift) and Selection (77 on `game0030.sav`, one leaf different from
the others). A raw byte-diff count for a length-changing edit is a SHIFT
footprint, not a scatter count — the decoded-level claim (every other field's own
value unchanged) is what the synthetic tests above prove, and is not restated by
this number; this table exists to show the shift stays confined to what follows
the edited field and never touches what precedes it (checked structurally by the
synthetic tests' explicit prefix-byte-identity assertions, not by this count
alone).

### SAV-DECPAD-238 interaction (the bug this story found and fixed)

`document.go`'s `exactDocument()` enforces `len(w.b) == w.p + (w.p&1)` — an odd
natural (unpadded) endpoint gets exactly one trailing alignment byte, whose
VALUE `SAV-DECPAD-238` does not constrain ("its ten observed byte values
reject a single fixed constant"; no source-pad-to-output-pad copy occurs in
ROM1's own writer either). A naive `SetMapName` splice failed 3 of 4
synthetic test cases with `"sav: document ends at N, decoded size N+1"`
because whether that pad byte belongs flips when the name's length changes
by an odd count, and `len(f.Body)` alone (always even) cannot say which
state the OLD body was in. Fix: `document` gained an `end int` field (the
unpadded length, `w.p` at read's end); `SetMapName` reads the OLD body's
`end` before splicing, and after splicing drops the pad byte when the new
parity no longer wants one, or writes a fresh literal `0` when it newly
does. The setter does NOT carry the old byte's own value across the edit in
the latter case — it cannot, since nothing in this codec or in the located
ROM1 routines copies it — so a there-and-back pair of odd-delta `SetMapName`
edits is not a byte no-op on a body whose existing alignment byte is
non-zero, true of 46 of the 136 preserved saves under `gameversions/saves/`
and `review/`. `TestSetMapNameOddDeltaPadIsWrittenNotCarried`
(`pkg/formats/sav/field_setters_decpad_test.go`) pins this. Verified against
all 4 synthetic name lengths (`""`, `"a"`, `"10.alm"`, a 24-byte name) and
against 6 real corpus files.

### The kit — rungs 2-5, measured

`review/owner-run-sav-ladder-fields/README.md` carries the full 12-row table
(2 bases × 6 rungs, 0-5) with per-rung byte diffs against the previous rung and
against BOTH bases' own rung 0. Summary, byte counts from `Marshal()` output,
diffed pairwise between staged files:

| rung | group | O-base bytes | vs O0 | P-base bytes | vs P0 |
|---:|---|---:|---:|---:|---:|
| 0 | label only | 3,223 | 0 | 3,277 | 0 |
| 1 | purse | 3,223 | 9 | 3,277 | 8 |
| 2 | view/options | 3,219 | 96 | 3,273 | 95 |
| 3 | head | 3,207 | 2,110 | 3,261 | 2,151 |
| 4 | campaign scalars | 3,207 | 2,106 | 3,261 | 2,147 |
| 5 | campaign collections | 3,163 | 2,128 | 3,217 | 2,169 |

Every staged file round-trips (`Open` then `Marshal` reproduces itself
byte-identically) and was verified by re-opening and reading back every field
the rung's setters touched — see the tool output transcribed into the final
report, not duplicated here. Both source saves (`game0025.sav`, `game0029.sav`)
were opened read-only; nothing was written into `gameversions/`; the kit is
staged in untracked `review/owner-run-sav-ladder-fields/` and committed nowhere.
`game0030.sav` (roster 2→5) is a setter-boundedness corpus subject only, not a
kit base — the current town writer refuses any roster change outright, so it is
not a like-for-like base for a field-group test.

## Open debt

**Rung 6 ("everything left") is not built.** It is object-identity reminting —
the Diary key and whatever else the full `ExportNativeCitySave` path remints
across the document — which no setter in this story reaches, matching exactly
what the first kit's own `README.md` said blocked rungs 2-6 before this story's
setters existed. Two specific leads into that remainder were explicitly ruled
out of scope before rung 6 was considered (coordinator addendum): Diary key
reminting (measured NOT validated on load by ROM1 — original 0x02c1ed70, our
round trip 0x01000010, ROM1's own resave from our document 0x030fa740, all
accepted) and actor undecoded-pair drift with play (a research question, not a
setter gap). No DIV row is filed for this: it is unimplemented capability, not
a place where this implementation's BEHAVIOUR diverges from researched ROM1
behaviour or an owner ruling against a claim — nothing here decides ROM1 truth
differently from research.

**DIV-991 through DIV-996 (reserved allocation): none used.** Every setter
group proved bounded; no new persisted field was added (all four groups write
EXISTING fields); `formatVersion` was not moved; no ROM1-behaviour-vs-owner-
intent decision was made. All six numbers are returned unused.

**Unknowns remaining**, none newly introduced by this story:
- Whether rungs 3 (head, `SAV-SHAPE-023` vs `SAV-CITY-030`'s open disagreement),
  4 or 5 pass ROM1's LOAD on either base — the kit is staged, not yet run.
- Whether a base-dependent result (e.g. O3 passing where P3 fails) occurs; the
  played-on base carries a 76-byte-heavier body before any rung's edits.
- RU parity, per the existing kit's own `README.md`: unmeasured and, per
  `pipeline/SAV-OWNER-RUNS.md`, not worth a dedicated owner sitting given
  `rom.exe`'s byte identity across roots.

## Touched surfaces

`pkg/formats/reg/reg.go` (two unexported `Node` fields: `index`, `rawD`/`rawSz`),
`pkg/formats/reg/write.go` (new), `pkg/formats/reg/write_test.go` (new),
`pkg/formats/sav/city_campaign.go` (`campaignRecordToBase` extracted to
package scope, no behaviour change), `pkg/formats/sav/document.go`
(`document.end` field), `pkg/formats/sav/field_setters.go` (new),
`pkg/formats/sav/field_setters_test.go` (new),
`pkg/formats/sav/field_setters_corpus_test.go` (new, `savdocumentaudit`-gated),
`pkg/formats/sav/sav_test.go` (one added case to the existing
`TestAnEditChangesTheFieldAndNothingElse` table). `review/owner-run-sav-ladder-fields/`
is a new untracked kit, committed nowhere.
