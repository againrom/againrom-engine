# Tasks — lossless ALM round-trip writer

Legend: **files** the task may change · **done when** the observable it must leave behind. Every
entry is an implementation task; they land in ascending order, and every dependency of a task is
a task before it.

## T1 — the document: acceptance, retention, the writer, the interpreted view

**files** new `pkg/formats/alm/document.go`, new `pkg/formats/alm/document_test.go`;
`pkg/formats/alm/doc.go` (the round-trip paragraph)

DD-1, DD-2, DD-3, DD-6 — `OpenDocument`, `Write`, `Map` and the span index; navigation waits
for T2.

**done when** SC-1, SC-2 and SC-3 pass (FR-1, FR-2, FR-6's package half). `alm.go` and every
existing test file are not edited; no new import enters the package; no byte of the copy is
routed through a `float32`; every rejection verdict in the tests is written from the spec's
list, never captured from `Open`.

## T2 — navigation and ownership

**files** `pkg/formats/alm/document.go`, `pkg/formats/alm/document_test.go`

DD-4.

**done when** SC-4 and SC-5 pass (FR-3, FR-4). The accessors and `Map` derive from the copy on
every call — the document stores no decoded state; an out-of-range `RecordPayload` index panics
via the slice subscript, unguarded, asserted by a recovered-panic probe; the ownership cases
assert against a clone taken before the mutation.

## T3 — the fuzz target

**files** new `pkg/formats/alm/fuzz_test.go`

DD-5.

**done when** SC-6 passes (FR-1's fuzz clause, FR-5). The seeds cover an accepted map, a
permuted-order variant and each AC-4 rejection class, added by `f.Add` in code with nothing
under `testdata/`; the target asserts decision agreement with `Open`, round-trip identity and
copy-independence, zeroing only its own working clone, never the engine's slice; plain
`go test` runs it green with no `-fuzz` flag.

## T4 — `almtool roundtrip`

**files** `cmd/almtool/main.go` (the verb: the switch, `usage()`, the package comment's verb
list), new `cmd/almtool/roundtrip_test.go`

DD-7.

**done when** SC-8's unit half passes (FR-7). The verb reports size and MD5 digest on identity,
the first differing offset on mismatch (shorter length on divergence), the rejection otherwise,
failures exiting non-zero through the existing error path; existing verbs and their output are
unchanged; the test's streams are synthetic bytes in a temp dir, no install path anywhere.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1, FR-2, FR-6 | DD-1, DD-2, DD-3, DD-6, SC-1, SC-2, SC-3 |
| T2 | FR-3, FR-4 | DD-4, SC-4, SC-5 |
| T3 | FR-1, FR-5 | DD-5, SC-6 |
| T4 | FR-7 | DD-7, SC-8 |
