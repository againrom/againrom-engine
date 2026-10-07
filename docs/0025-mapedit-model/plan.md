# Plan — map-editor E2: the headless edit / document model

## Baseline

`pkg/formats/alm` is a leaf on the standard library plus `golang.org/x/text`, which
`internal/archtest` grants to `pkg/formats/*` alone and denies to every package its allow-map does
not register. `OpenDocument` decides by running `Open` on its own private clone; `Write` clones
that copy and `Document.Map` is `Open` of it. Records may be emitted in any typeId order —
`walkRecords` indexes payloads by typeId, and the decode order is fixed — while `internal/synth.ALM`
emits type-0 first with zeroed constants, slots and undecoded unit bytes: it produces neither the
rich fixture nor a permuted order.

## Design decisions

### DD-1 — one flat buffer and an edit log; nothing derived is stored

The state is the accepted bytes, an ordered log of edits and a cursor into it. Record spans, W and
H, the unit count and every field offset are computed from the current bytes where they are used. A
length-changing mutation then has nothing to invalidate, which removes the stale-index bug class
rather than defending against it. Rejected: a span index cached at load — every splice before a
record's start silently rots it, to save ten iterations of a four-byte read; rejected: keeping the
`*alm.Document` and rebuilding it per mutation — a second acceptance decision on a mid-edit state,
letting a mutation fail for a reason no requirement names.

### DD-2 — acceptance is `alm.OpenDocument`'s alone, and the accepted bytes are its own

`New` hands the caller's slice to `alm.OpenDocument`; an error returns wrapped beside a nil model,
and on success the buffer is that document's `Write()` — the bytes acceptance was taken on — after
which the document is dropped. No second decision exists to drift from the first, and the caller's
slice, mutable behind our back, never becomes the edited buffer. Rejected: a validation walk here —
two expressions of one accept set is the defect FR-1's "exactly" forbids, and a roster guard at
load is that same defect wearing the other sign.

### DD-3 — validate, then one splice group, and the group is the history entry

An edit is a list of parts, each a pre-image offset with the bytes it removes and the bytes it
inserts, both held as copies. One function writes to the buffer: it applies the parts in descending
offset order, so each pre-image offset is still valid when its turn comes, then truncates the log at
the cursor and appends — FR-7's discard clause, in the applier rather than in a setter. Undo
applies the mirror parts ascending: reverting a part cancels the shift it contributed, so every
later part is back at its recorded pre-image offset by the time its turn comes. Both orders hold
because parts within an edit never overlap. Each setter runs its whole rejection surface before
constructing the edit, so a rejection has written and recorded nothing: FR-8 and FR-7's
record-every-accepted-mutation clause are shapes, not disciplines — the applier has no branch that
skips the log. Rejected: a whole-buffer snapshot per edit — undo becomes true by construction, so the
AC-5 witness stops telling a correct implementation from a wrong one; rejected: per-setter writes with
an ad-hoc rollback — rollback correctness re-argued at every call site.

### DD-4 — the field codecs become a pair, both halves inside `pkg/formats/alm`

A new `encode.go` there adds `EncodeName` and `EncodeDescription`, each returning the **field image**
— encoded bytes then NUL fill to that field's own length — or an error. Beside FR-4's two rejections
each adds a third: any input whose field image does not decode back,
through this package's own `decodeASCII`/`decodeCP1251`, to the string it was given. That makes P-6 a
property of our contract rather than of `x/text`'s default error policy, and alone rejects an embedded
NUL, which CP1251 encodes happily and the reader truncates at. `alm` gains both directions of a rule
it owned one direction of and no mutation surface; `pkg/mapedit` never names a codec or the number
64. Rejected: a CP1251 encoder in `pkg/mapedit` —
the `x/text` grant is the formats tier's and the check is fail-closed; rejected: exporting the raw
codec and doing the length check and fill here — one field rule split across the import wall is the
drift the pair exists to prevent.

### DD-5 — nothing unit-shaped crosses the boundary except bytes

Reading a record yields a fresh copy of its bytes; placing takes a whole record and rejects any
other length; only the move writes into a record, and only its two coordinate words. A unit is
addressed by file-order index, over an id word nothing guarantees unique. The package
holds no unit type, no default record and no typed-to-bytes conversion, so no call fills the 50 bytes
the decoded view does not expose; the cheap way to a valid record is to clone one. A place is three
parts (the record appended at the end of the type-6 payload, that record's `payloadSize` word,
`#type6`) and a delete their mirror; the count comes from `#type6`, the word `alm`'s own unit walk is
driven by. Rejected: a typed placement value — its 50 invented zero bytes are why this story exists;
rejected: counting from `payloadSize/70` — it agrees today and would hide a divergence the format
cannot itself express.

### DD-6 — the view is a function of the bytes, taken on a fresh copy

`Bytes` clones the buffer and the view is `alm.Open` of that clone. No decoded state is stored, so no
accessor can report what the bytes do not say, no `*alm.Map` a caller holds can alias the buffer or
be disturbed by a later edit, and every "the view is unchanged" clause collapses to byte equality.
Rejected: caching the last view — a shared mutable result and a staleness class the model has no
reason to own; rejected: opening the buffer in place — correct only because `alm.Open` happens to
copy all it returns, another package's internal detail.

### DD-7 — the witness computes the target region itself, and the fence is mechanical

Four test-side pieces, each pinned before it measures. A frame walk written in the test package from
the contract's layout, so a test never asks the model where a region is — both sides computed the
same way is the failure this avoids. A comparator that deletes the declared regions from the before
and after images and requires the remainders equal: FR-2's carried clause stated exactly. A rich
fixture whose settable fields, carried regions and unit records are pairwise distinct and distinct
from every value a test sets, or a setter writing at the wrong offset passes by coincidence and the
comparator passes vacuously — emitted in **both record orders**, type-0 first and type-6 first,
because with type-0 first the count words a place rewrites precede the bytes it inserts and a
descending undo passes anyway. And a scan of the non-test files pinning, per enclosing function, the
`alm.` identifiers it may name, so a decoded value can be obtained only where one of the two entry
points is named, neither of them a write path. Its limit is the determinism
wall's: lexical, proving no decoded value is *named* on a write path, not that none can arrive.

### DD-8 — the fuzz driver is an operation script; coverage is the corpus's, not the input's

The corpus byte string is read as opcodes and operands applied to the rich fixture, interleaving
undo and redo. After every step the bytes re-open through `alm.OpenDocument`, write back identically
and yield a view; undoing to the empty history reproduces the input exactly. Write-back is identity
for any accepted stream, so the step check really proves acceptance — P-2's whole content. The
per-kind accept counts are asserted by a plain test over the same seed slice, never inside the target:
an input exercising two opcodes is not a defect, while a driver that has drifted into rejecting
everything must still report red. Rejected: fuzzing raw ALM streams — 0023's document target owns
that decision, and it would exercise `alm`, not this model.

### DD-10 — absence is a value in the frame, and an offset comes only from a located record

The walk returns one `span` per typeId **plus a present flag**, and `abs` moves onto `span`: an
offset is made from a record the walk found, never from an id. Reads split by what
acceptance guarantees: `required(tid)` for type-0/1/2, `optional(what, tid) (span, error)` for the
rest, its error a rejection taken before any edit exists. One fallible lookup instead would hand the
guaranteed three a branch no accepted input can reach, and an unreachable branch with a test is the
same defect as a missing one. A zero unit count where the frame has no type-6 record for `#type6` to
be about, and a place there rejected, follow rather than being added. The walk's bound becomes
the file header's own record count, a stream being free to carry more than ten records and to end in
bytes no record covers. An absent id is left one byte past EOF, not at the zero value that is the
file header's first byte — a backstop the campaign reports as a survivor, the lookup being the
guarantee.

### DD-9 — the package registers as a leaf above `alm`, and nothing else moves

`internal/archtest`'s allow-map gains one row, `pkg/mapedit` importing `pkg/formats/alm`; without it
the fail-closed check rejects the package on sight. `docs/ARCHITECTURE.md` gains the matching row,
its table being the allow-map's stated-identical prose form, and `AGENTS.md`'s tier list with it.
`alm` gains no behaviour and no exported signature, so its unchanged surface is inspectable from
the diff. The model splits the buffer, log and applier into `editor.go`
and takes one file per setter family (`grid.go`, `meta.go`, `unit.go`) beside its `doc.go`; tests are
an external test package throughout, one per source file plus `fixture_test.go`, `fuzz_test.go` and
`arch_test.go`. Rejected: `internal/` for the model — the editor track's later stories are its
consumers.

## Risks

- **R-1** A mutation that regenerates a region instead of carrying it satisfies every value-level
  assertion in the suite; DD-7 is the whole answer to it.
- **R-2** The offsets now exist in two packages, `alm`'s unexported constants and this model's own,
  so a later correction can leave one behind. Mitigated by witnessing every setter through the
  shipped decoder — a wrong offset makes the view disagree.
- **R-3** A caller can place a record the game cannot use. Accepted by the contract, mitigated
  only by DD-5's shape.

## Success criteria

- **SC-1** AC-1 holds, and the returned buffer, a returned record and the caller's input slice are
  each independent of the model afterwards (FR-1, FR-5).
- **SC-2** The apparatus is pinned before it measures: both fixture orders are accepted by
  `alm.Open` and `alm.OpenDocument`, their distinctive bytes pairwise distinct, the frame walk agrees
  with `alm.Document.RecordPayload` on every record of each, and the comparator accepts an
  in-region change while rejecting an out-of-region one and a reordering (FR-2).
- **SC-3** AC-2's grid and type-0 halves hold in full, each mutation on a fresh load; and a shorter
  string leaves no residue of the replaced text inside its own 64 bytes (FR-3, FR-4, P-1).
- **SC-4** P-6 both ways and on both fields: for every byte 1..255 standing alone in a field, setting
  the field to the string the reader decoded from it restores that byte exactly or is rejected — the
  rejected set being exactly {0x98} for the description and 0x80..0xff for the name, so an encoder
  that refuses everything fails here (FR-4).
- **SC-5** AC-3 and AC-2's move half hold in full, over both fixture orders and over a map with an
  empty type-6 payload as well as one with several pairwise-distinct units (FR-5, FR-6, P-1, P-2).
- **SC-6** AC-4 holds in full, each rejection leaving both history reports unchanged beside the bytes
  and the view (FR-8, P-4).
- **SC-7** AC-5 and AC-6 hold in full, against snapshots the test took itself, over both fixture
  orders and with a place and a delete among the mutations (FR-7, P-3, P-5).
- **SC-8** AC-7 holds under plain `go test` over the seeds, with no panic; and a plain test over the
  same seeds accepts every mutation kind at least once (FR-6, P-2).
- **SC-10** AC-8 holds over every thin roster: what the roster carries still edits and re-opens, what it lacks is rejected with bytes, view and history unchanged, and
  the unit count follows the record rather than `#type6` (FR-1, FR-3, FR-5, P-7).
- **SC-9** The walls: `internal/archtest` green with the new package registered against
  `pkg/formats/alm` alone; the model's non-test source naming no `alm` identifier outside its
  per-function table; `alm`'s exported surface, behaviour and consumers unchanged (FR-2, FR-4).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-1, DD-2, DD-6, DD-10 | SC-1, SC-10 |
| FR-2 | DD-1, DD-3, DD-7, DD-9 | SC-2, SC-3, SC-5, SC-9 |
| FR-3 | DD-1, DD-3, DD-10 | SC-3, SC-10 |
| FR-4 | DD-4 | SC-3, SC-4, SC-6, SC-9 |
| FR-5 | DD-5, DD-10 | SC-1, SC-5, SC-6, SC-10 |
| FR-6 | DD-6, DD-8 | SC-5, SC-8 |
| FR-7 | DD-3 | SC-7 |
| FR-8 | DD-3, DD-6, DD-10 | SC-6, SC-10 |
