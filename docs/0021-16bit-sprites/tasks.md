# Tasks — the two 16-bit sprite decoders

Legend: **files** the task may change · **done when** the observable it must leave behind. Every
entry is an implementation task; they land in ascending order, and every dependency of a task is a
task before it.

## T1 — `pkg/formats/spr16`: the container, the cursor, and the DAG entry

**files** new `pkg/formats/spr16/doc.go`, `pkg/formats/spr16/container.go`,
`pkg/formats/spr16/container_test.go`; `internal/archtest/dag.go` (the package's two map
entries), `docs/ARCHITECTURE.md` (the package's two rows)

DD-2, DD-3's caps and refusal style, and DD-7's registration, landing in one commit.

**done when** SC-1 passes (FR-4, FR-6, FR-7). The package exports nothing yet — neither decoder
exists here; `cmd/sprtool` and its allow row are untouched; `noExternalFormats` gains the package
in this task, not later.

## T2 — `pkg/formats/spr16`: the `.16a` decoder

**files** new `pkg/formats/spr16/spr16a.go`, `pkg/formats/spr16/spr16a_test.go`

DD-1's A side and DD-4, over T1's walk and cursor, under DD-8's fixtures.

**done when** SC-2, SC-3 and SC-4 pass (FR-1, FR-3, FR-4, FR-5, FR-6). `container.go` and
`internal/archtest` are not edited; no expansion, colour-resolution or alpha helper enters the
package; the fuzz target lives beside the tests whose malformed streams seed it.

## T3 — `pkg/formats/spr16`: the `.16` decoder

**files** new `pkg/formats/spr16/spr16g.go`, `pkg/formats/spr16/spr16g_test.go`

DD-1's G side and DD-5, sharing T1's cursor arms and DD-8's fixture helpers.

**done when** SC-5 and SC-6 pass (FR-2, FR-4, FR-5, FR-6). No palette parameter, field or path
appears on the G side; `container.go` and the A side are not edited.

## T4 — `cmd/sprtool`: the 16-bit dumps

**files** `cmd/sprtool/main.go`; `internal/archtest/dag.go` (the `cmd/sprtool` row),
`docs/ARCHITECTURE.md` (the `cmd/sprtool` row)

DD-6.

**done when** SC-7 passes (FR-7, FR-8). `png`'s path and output naming are unchanged; the
presentation ramps exist only in this command; no test opens an archive or reads a game install.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-4, FR-6, FR-7 | DD-2, DD-3, DD-7, SC-1 |
| T2 | FR-1, FR-3, FR-5 | DD-1, DD-4, DD-8, SC-2, SC-3, SC-4 |
| T3 | FR-2 | DD-1, DD-5, DD-8, SC-5, SC-6 |
| T4 | FR-7, FR-8 | DD-6, SC-7 |
