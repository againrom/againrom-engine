# Tasks — map-editor E2: the headless edit / document model

Legend: **files** the task may change · **fences** what it must not do · **done when** the observable
it must leave behind. Every entry is an implementation task; they land in ascending order and each
depends only on entries before it. **(small)** marks one light enough to pair with a neighbour.

## T1 — the encoder half of the field codecs (small)

**files** ADD `pkg/formats/alm/encode.go`, `pkg/formats/alm/encode_test.go`; MODIFY
`pkg/formats/alm/doc.go` (one sentence in its Strings paragraph)

DD-4, `alm`'s side of FR-4's codec rule. Nothing in `pkg/mapedit` yet.

**fences** no new import in the package; `alm.go`, `document.go` and every existing test file
unedited; the encoders return an image and an error and mutate nothing reachable from a caller.

**done when** SC-4 holds inside `alm`: for every byte 1..255 standing alone in a 64-byte field, the
image the encoder builds from the decoder's own string equals the original field or the call is
rejected, the rejected set being exactly {0x98} for the description and 0x80..0xff for the name. An
over-long encoding, an unrepresentable rune and an embedded NUL are each rejected under their own
case, with the codec's error and the round-trip mismatch asserted as distinct causes rather than as
one "rejected".

## T2 — the package, its DAG row, and the measuring apparatus

**files** ADD `pkg/mapedit/doc.go`, `pkg/mapedit/fixture_test.go`; MODIFY `internal/archtest/dag.go`,
`docs/ARCHITECTURE.md`, `AGENTS.md`

DD-7's fixture, frame walk and comparator, and DD-9 — FR-2's witness machinery. `doc.go` carries the
package comment and no code; the model itself is T3.

**fences** the frame walk is written from the contract and never calls a `pkg/mapedit` function; the
comparator takes its regions as arguments and computes none of them; no byte the rich fixture
requires to be distinctive is left at `internal/synth`'s zero.

**done when** SC-2 holds and `internal/archtest` is green with the new row: both fixture orders
accepted by `alm.Open` and `alm.OpenDocument`, the walk agreeing with `alm.Document.RecordPayload` on
all ten records of each, the distinctive bytes present and pairwise distinct, and the comparator's
own three cases — an in-region change accepted, an out-of-region change and a reordering rejected —
passing. The `ARCHITECTURE.md` row and the allow-map entry say the same thing.

## T3 — the model: load, bytes, view, the splice applier, the history, grid paint

**files** ADD `pkg/mapedit/editor.go`, `pkg/mapedit/grid.go`, `pkg/mapedit/editor_test.go`,
`pkg/mapedit/grid_test.go`

DD-1, DD-2, DD-3 and DD-6 — FR-1, FR-3, FR-7 and FR-8's shape. The grid setters are the first
mutation and the only setters here; type-0 is T4 and units are T5.

**fences** one function writes to the buffer and every setter reaches it; no offset, span, count or
decoded value is stored on the model; in non-test source `alm.OpenDocument` is named in the
constructor only and `alm.Open` in the view method only; the undo and redo cases compare against
buffers the test cloned for itself, never against a recomputed expectation.

**done when** SC-1, SC-3's grid half, SC-6's out-of-range-cell case, and SC-7 over the fixed-length
mutations this task makes available all pass, every region diff taken through T2's comparator.

## T4 — the type-0 setters: the scalars and both strings (small)

**files** ADD `pkg/mapedit/meta.go`, `pkg/mapedit/meta_test.go`

DD-4's consumer side — FR-4, and FR-8 for its rejections.

**fences** the string setters call T1's encoders and copy the returned image whole; no field width,
codec or terminator rule is restated in `pkg/mapedit`; the angle setter converts the caller's
`float32` to bits and never reads the field's existing bytes back through one.

**done when** SC-3's type-0 half, SC-4 through the setters and SC-6's string rejections pass: each
scalar and each string lands inside its own four or sixty-four bytes with the text slots, the
signalling-NaN angle bits and the other field's post-NUL bytes carried, and a shorter replacement
leaves no residue of the old text inside its own field.

## T5 — the units: count, read, place, move, delete

**files** ADD `pkg/mapedit/unit.go`, `pkg/mapedit/unit_test.go`

DD-5 — FR-5, FR-2's length-changing half, FR-6, and FR-8 for the index and record-length rejections.

**fences** no unit type, no default record and no typed-to-bytes conversion enters the package; the
count is read from `#type6` and never derived from `payloadSize`; a move writes eight bytes and no
ninth.

**done when** SC-5 passes over both fixture orders, and SC-7's place and delete cases and SC-6's unit
rejections with it: a placed record reads back byte-identical at the returned index, every other
record is untouched, a moved record's other 62 bytes are carried, and undoing a place or a delete
reproduces the pre-image exactly.

## T6 — the fuzz driver and its coverage test (small)

**files** ADD `pkg/mapedit/fuzz_test.go`

DD-8 — FR-6 under arbitrary operation scripts.

**fences** seeds are added from a slice declared in code with nothing under `testdata/`; the target
never fails for under-coverage; the driver mutates only buffers it owns.

**done when** SC-8 passes: the seeds run green under plain `go test` with no panic, and the coverage
test over the same slice shows every mutation kind accepted at least once.

## T7 — the source fence (small)

**files** ADD `pkg/mapedit/arch_test.go`

DD-7's fourth piece — FR-2 and FR-4's "no decoded value reached on a write path".

**fences** the scan reads parsed syntax rather than file text; the per-function table is a literal in
the test, compared for exact equality rather than searched for an absence.

**done when** SC-9 passes: every `alm.` qualified identifier in the package's non-test files matches
the declared table for its enclosing function, and a deliberately wrong table makes the test fail.

## T8 — the frame carries absence, and the roster guard goes

**files** MODIFY `pkg/mapedit/editor.go`, `grid.go`, `unit.go`, `meta.go`, `doc.go`, and
`editor_test.go`, `grid_test.go`, `unit_test.go`, `fixture_test.go`, `arch_test.go`; MODIFY
`pkg/formats/alm/alm.go` (doc comments only, no behaviour)

DD-10 — FR-1's "exactly" restored, and FR-3, FR-5, FR-8 over a roster that is not ten records.

**fences** no new acceptance decision at load; the guaranteed and optional lookups stay separate,
so no setter for type-0, type-1 or type-2 gains an error path no accepted stream can take; the
thin fixtures are built from the same payloads as the rich one and are pinned against `alm.Open`
and `Map.Present` before anything measures with them; `alm` gains no behaviour and no exported
signature, and the `Groups`/`Present` comments say only what the reader does and does not default.

**done when** SC-10 passes and SC-1's acceptance half is exact again: `New` agrees with
`alm.OpenDocument` case for case with the never-looser direction asserted on its own, and the
former stricter witnesses are ordinary agreement cases. The mutation campaign reaches the walk's
loop bound, the presence flag, the count guard and the two lookups, and reports the past-EOF
sentinel's own result whichever way it comes out.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-4 | DD-4, SC-4 |
| T2 | FR-2 | DD-7, DD-9, SC-2 |
| T3 | FR-1, FR-3, FR-7, FR-8 | DD-1, DD-2, DD-3, DD-6, SC-1, SC-3, SC-6, SC-7 |
| T4 | FR-4, FR-8 | DD-4, SC-3, SC-4, SC-6 |
| T5 | FR-2, FR-5, FR-6, FR-8 | DD-5, SC-5, SC-6, SC-7 |
| T6 | FR-6 | DD-8, SC-8 |
| T7 | FR-2, FR-4 | DD-7, SC-9 |
| T8 | FR-1, FR-3, FR-5, FR-8 | DD-10, SC-10 |
