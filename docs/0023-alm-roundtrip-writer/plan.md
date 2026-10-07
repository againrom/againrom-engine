# Plan — lossless ALM round-trip writer

## Baseline

`pkg/formats/alm` is a pure leaf (stdlib + `golang.org/x/text`, held there by `internal/archtest`'s
fail-closed import check). `Open` and `OpenInfo` share one frame walk, `walkRecords`: the
file-header checks (magic, hdrLen 20, `recordCount >= 3`, `formatVersion <= 1001` with 1000
refused as unimplemented), that many 20-byte record headers checked for tag 7 and hdrLen 20,
payload slices indexed by typeId — `{type0, type1, type2}` required, a repeat last-wins, an id at
or above 10 stepped over — and bytes past the last counted record left unread. `Open` then runs
the per-type
decoders (the 632-byte type-0, three W×H grids, the type-4 extension walk, 76/70-byte type-5/6
strides, type-7/9 count words); every rejection is `fmt.Errorf("alm: ...")` plus a nil result,
never a panic, and `cloneBytes` keeps the decoded `Map` aliasing nothing. dataSize and the
per-record +0x10 word are stored (`Map.DataSize`, `Record.PerMapConst`, `SelectorA`) but gate
nothing.

Tests are external (`package alm_test`), every fixture synthetic: `le32`-family writers,
`fileHeader`/`recordHdr`/`rec`/`recBadSize`/`recCustom`, `buildMeta`, `baseSections`,
`wrapAll`/`buildMap` over the corpus `physicalOrder`, plus `openOK`/`openReject`;
`info_test.go` also draws on `internal/synth.ALM`. The sibling leaves `res`, `reg` and `spr16`
carry `f.Add`-seeded fuzz targets, nothing under `testdata/`; `alm` has none yet. `cmd/almtool` is a verb switch over `cmdX(path) error`; `loadMap` =
`os.ReadFile` + `alm.Open`; an error prints `almtool: %v` and exits 1, a usage fault exits 2;
the command has no test file.

## Design decisions

### DD-1 — the document is one private copy; the writer is a clone

`Document` holds a single flat copy of the accepted stream plus a span index, one entry per
record the file header counts (each record's header offset and payload size, in file order).
Bytes no span covers — a trailer — ride the copy like any other. `Write` returns `cloneBytes` of
the
copy. Identity is structural: the write path never takes the stream apart, so it cannot
reassemble it wrongly, and no byte ever passes through a `float32`, so the signalling-NaN
patterns FR-2 names cannot be quieted by a conversion — dataSize, the record order, post-NUL
bytes and every undecoded run ride the copy untouched. Rejected: per-record segments
re-concatenated by `Write` — identity becomes contingent on segmentation correctness at every
write, for no gain the span index does not already give; rejected: retaining the caller's
slice — FR-3 makes the copy mandatory, and open is the one place to take it.

### DD-2 — one acceptance decision, taken on the retained copy

`OpenDocument` clones the input first, then runs `Open` on the clone: an error returns wrapped
with a nil document; success builds the span index by a post-acceptance arithmetic re-walk
(cursor += 20 + payloadSize, once per counted record). `Open` has proven every bound, but the
re-walk is bounds-checked anyway: the count is a FILE field, so an arithmetic slip here would be a
slice-bounds panic on valid input rather than an impossible state. The accept set is `Open`'s by
construction on all inputs, not merely on tested ones —
FR-1's "precisely" leaves no room for a second decision that is only demonstrably equal where a
test looked — and FR-4's parenthetical ("that call already succeeded at open time") anticipates
exactly this call. Deciding on the clone makes the accepted bytes the retained bytes. FR-5
rides `Open`'s proven totality: arbitrary input reaches nothing else, and the index walk runs
on accepted streams alone. Rejected: an independent validation walk — any drift between two
walks IS the FR-1 defect; rejected: extending `walkRecords` to export offsets — an edit to the
shipped decode path for what ten lines of arithmetic derive after the fact.

### DD-3 — the equivalence witness takes its verdicts from the spec, not from Open

Under DD-2, a test that asks `Open` what to expect and `OpenDocument` what happened compares
one expression of the decision with itself. The witness therefore derives its verdicts
independently: AC-4's fixtures are built one per clause of the spec's rejection list, and each
is asserted both ways — `Open` rejects it AND `OpenDocument` returns a wrapped error and a nil
document — pinning both entry points to the spec's enumerated decision, so a drift in either
shows as a red test rather than as two wrong answers agreeing. The accept side is pinned the
same way: AC-1/AC-2/AC-3/AC-10's streams come from the 0003 fixture builders (shared via
`package alm_test`), extended with per-record header bits and an explicit record order where
AC-2 needs them — and each is asserted accepted by `Open` itself, not only round-tripped, so
the accept half of the decision is pinned per fixture too. The fuzz target's agreement clause (DD-5) covers the unenumerated remainder,
where its job is the wrapper's atomicity, not the decision. Rejected: differential-only
testing, expected values captured from `Open` — the tautology above.

### DD-4 — accessors derive on call and clone on return; nothing is cached

`Version` reads the u32 at +0x10 of the copy; `RecordCount` is the span index's length;
`RecordTypeIDs` reads each indexed header's typeId word from the copy into a fresh slice of that
length; `RecordPayload(i)` clones the indexed payload bytes; `Map()` is `Open` of the copy, run
per call. The document stores no decoded state, so no accessor can disagree with the
written bytes, every return is fresh, and two `Map` results share nothing a caller's mutation
could couple. `RecordPayload`'s bounds discipline is the index slice's own: an `i` outside
`0..RecordCount()-1` reaches the subscript unguarded and panics exactly as FR-4 specifies —
misuse, not data. Rejected: caching the open-time `*Map` — one shared mutable result across calls, the
aliasing coupling FR-3's copy-out discipline exists to exclude; rejected: an error-returning
`RecordPayload` — it would grade misuse as data.

### DD-5 — the fuzz target

`FuzzDocumentRoundTrip` seeds, by `f.Add` in code like the sibling leaves (nothing under
`testdata/`), an accepted minimal map, a permuted-order variant, AC-10's off-shape streams and
one fixture per AC-4 rejection class, then asserts on every input: `OpenDocument` and `Open` agree accept/reject,
rejection carrying a nil document; on accept, `Write` equals the input after a working clone —
not the engine's slice, whose in-place reuse would poison mutation and corpus recording under
`-fuzz` — was handed to `OpenDocument` and zeroed post-open (identity and copy-independence in
one assertion), and a second `Write` after mutating the first stays equal. Seed entries run under plain
`go test`, so AC-7's floor costs no fuzzing time. Under DD-2 the agreement clause guards the
wrapper's atomicity — never a document beside an error — while the identity clause is P-1 and
FR-5 live. Rejected: a fuzz oracle re-implementing acceptance — DD-2's drift-prone second
walk, smuggled into a test.

### DD-6 — additive layout

Everything lands in new files — `document.go`, `document_test.go` and `fuzz_test.go` in the
package, `roundtrip_test.go` in the command — beside two narrow touches: one round-trip
paragraph in `doc.go`'s package comment, and the verb wired into `cmd/almtool/main.go` (the
switch, `usage()` and the package comment's verb list). `alm.go` is not edited at all, so FR-6 is inspectable from the diff:
every existing identifier, result and consumer stands because nothing defining them changed;
the leaf's imports gain nothing (`document.go` needs only stdlib the package already uses) and
`internal/archtest` runs unedited. Rejected: folding the document into `alm.go` — a reviewer
would have to re-read the decode path to see it unchanged.

### DD-7 — the instrument

`almtool roundtrip <file.alm>`: read, `OpenDocument`, `Write`, compare. Identity prints the
size and MD5 digest (the repo's evidence convention) and exits 0; a mismatch prints the first
differing offset — length divergence reports the shorter length — and a rejection prints the
wrapped error, both failures exiting non-zero through the existing `cmdX` error path. The
comparison lives in a pure `firstDiff(a, b)` helper: with an honest `Write` the mismatch branch
is unreachable in-process, but it exists for AC-9's corpus honesty — "identical" must be a
measurement, not an assumption — so the arithmetic a corpus run trusts must itself be pinned. A
small `package main` test drives the verb over synthetic streams in a temp dir (accept → nil
error, reject → error) and pins `firstDiff` directly. Rejected: reporting from inside the
library — the leaf gains presentation logic it has no consumer for; rejected: shipping the verb
untested like its siblings — its siblings' output is read by a human, this one's is recorded as
evidence.

## Success criteria

- **SC-1** AC-1's round-trip and view clauses: unedited `Write` of the minimal accepted map
equals the input byte-for-byte; `Map()` equals `Open` of it in result and error (FR-1, FR-2,
FR-4).
- **SC-2** AC-2 and AC-3 hold: the free header fields, the never-shipped record order, the
sNaN per-map bits, post-NUL bytes, the undecoded type-5 head and type-6 tail, a NaN-bit angle
and the type-7/8/9 bodies (empty type-8 included) all survive (FR-2).
- **SC-3** AC-4 and AC-10 hold — the rejection set, and the shapes the reader's acceptance takes
that the shipped writer never emits — with verdicts written from the spec's own lists, both entry
points asserted per fixture (FR-1, FR-2, FR-4).
- **SC-4** AC-1's navigation clause: `Version`, `RecordTypeIDs` and each `RecordPayload` agree
with the stream (FR-4).
- **SC-5** AC-5 and AC-6 hold: zeroing the input, mutating a returned write buffer and
overwriting navigation results change nothing observable (FR-3, FR-4).
- **SC-6** AC-7 holds: the fuzz target's seed corpus runs green under plain `go test`,
asserting agreement, identity and copy-independence (FR-1, FR-5).
- **SC-7** AC-8 holds: the full suite, `archtest` and the gates stay green with no source
change outside `pkg/formats/alm` and `cmd/almtool` (FR-6).
- **SC-8** the instrument, both halves: unit — the verb answers accept and reject over
synthetic temp-dir streams and `firstDiff` is pinned on unequal, prefix and equal buffers;
corpus — AC-9's 38 maps all report identical, sizes and digests recorded in `verification.md`
(FR-7, FR-2).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-2, DD-3 | SC-1, SC-3, SC-6 |
| FR-2 | DD-1 | SC-1, SC-2, SC-3, SC-8 |
| FR-3 | DD-1, DD-4 | SC-5 |
| FR-4 | DD-4 | SC-1, SC-3, SC-4, SC-5 |
| FR-5 | DD-2, DD-5 | SC-6 |
| FR-6 | DD-6 | SC-7 |
| FR-7 | DD-7 | SC-8 |
