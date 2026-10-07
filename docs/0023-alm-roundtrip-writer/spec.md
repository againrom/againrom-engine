# Spec — lossless ALM round-trip writer

## Problem and current behaviour

`pkg/formats/alm` decodes a ROM1 `.alm` map into an interpreted `Map`, keeping the fields the
engine needs. That typed view cannot rebuild the original bytes — dropped string tails, undecoded
record bytes, float values rather than bit patterns, and a plane it manufactures when the file
omits one — so re-serializing a `Map` would corrupt real maps.

The map-editor track needs a save path whose floor is: every byte the editor did not deliberately
change is preserved. This story adds that floor — a raw-backed document model and a writer whose
**unedited** output is byte-identical to its input — and is purely additive (FR-6).

## The container, and what acceptance leaves free

All multi-byte values little-endian. An accepted stream is a 20-byte file header followed by the
number of records that header counts — each a 20-byte record header plus `payloadSize` payload
bytes — laid end to end from `+0x14`. Bytes past the last counted record are neither read nor
refused.

File header: `+0x00` u32 magic `0x0052374D` ("M7R\0") · `+0x04` u32 hdrLen `= 20` · `+0x08` u32
**dataSize — uninterpreted, any value accepted** · `+0x0C` u32 recordCount **`≥ 3`** · `+0x10` u32
**formatVersion `≤ 1001`, any such value but `1000`** (the record-header-skipping dialect, refused
as unimplemented).

Record header: `+0x00` u32 tag `= 7` · `+0x04` u32 hdrLen `= 20` · `+0x08` u32 payloadSize ·
`+0x0C` u32 typeId — **any value, in any order**, a repeat and an id at or above 10 included ·
`+0x10` — **an uninterpreted 4-byte per-map constant, any bits accepted** (the reader exposes it
as a float32).

The records that must be there are `{type0, type1, type2}`; every other typeId is optional, a repeated
id resolves last-wins, and an id at or above 10 belongs to no section. Payload sizes are
constrained per typeId by the interpreted reader — a mismatch is a rejection (AC-4), never a free
byte — and each constraint binds only where its own record is present.

Everything acceptance pins is constant across accepted inputs; everything else varies freely and
MUST survive a round trip bit-for-bit.

## The contract

A **`Document`** is a raw-backed model of one accepted stream: it retains the bytes, interprets
none of them, and writes itself back. The header fields the interpreted reader validates or stores
without using — dataSize, tag, the record hdrLen, the per-map constant — ride the copy verbatim
like every other byte, whatever they turn out to mean.

API shape (observable contract; signatures, not internals):

```text
OpenDocument(data []byte) (*Document, error) // copy-in; accepts iff Open accepts
(*Document).Write() []byte                   // fresh buffer; unedited output == input
(*Document).Map() (*Map, error)              // interpreted view: Open of the document's bytes
(*Document).Version() uint32                 // the file header's formatVersion
(*Document).RecordCount() int                // records held; the stream's own recordCount
(*Document).RecordTypeIDs() []uint32         // typeIds in file order; a fresh copy
(*Document).RecordPayload(i int) []byte      // byte-exact payload of record i, file order; a fresh copy
```

**The decoded view and the byte view do not correspond record for record.** The interpreted reader
supplies what the engine supplies: a stream carrying no type-3 record still decodes to a full
`W·H` overlay plane, **manufactured**, with no bytes behind it anywhere in the file. A document
holds only records the stream carried, so `RecordCount` and `RecordTypeIDs` count the read ones
and never the manufactured one, and `Write` emits no record that was not read. Which side a
section came from is answerable on the interpreted side alone, through `Map.Present(typeId)`, and
**nothing may write back a record whose `Present` is false** — those bytes were never in the
input. An edit layer building on this seam inherits that obligation.

## Functional requirements

- **FR-1 — the document opens exactly what the reader opens.** `OpenDocument` MUST accept
  precisely the byte streams `Open` accepts and reject the rest with the same atomic shape: a
  wrapped error and a nil document — never a partial result, never a panic. "Same rejection"
  binds the accept/reject **decision**, not the error's text or type. dataSize and the
  per-map-constant words are never inspected by the decision.

- **FR-2 — unedited identity.** For any accepted input, `Write` of the unedited document MUST
  return bytes identical to that input. This binds every byte the interpreted view drops or has
  no field for, and in particular: a dataSize and a formatVersion holding arbitrary accepted
  values; each record header's twenty bytes, per-map-constant bits spelling a signalling NaN
  included; the records in their original order and original number, even when neither is the
  shipped maps' usual one; and bytes belonging to no section at all — a record whose typeId no
  decoder handles, and bytes past the last counted record. AC-2, AC-3 and AC-10 enumerate the
  payload cases.

- **FR-3 — ownership.** Copy-in, copy-out: mutating the caller's input buffer after
  `OpenDocument` returns MUST NOT change what `Write` emits, and each `Write` MUST return a
  fresh buffer whose mutation affects neither the document nor any other write.

- **FR-4 — the interpreted view and byte-level navigation.** `Map()` MUST equal `Open` of the
  document's bytes in result and error, inheriting its panic-free guarantee. `Version()`,
  `RecordCount()`, `RecordTypeIDs()` and `RecordPayload(i)` MUST agree with the written bytes,
  return fresh copies, and mutate nothing — read-only navigation for a later edit layer to build
  on. `RecordCount()` MUST equal the stream's own recordCount and is the length of
  `RecordTypeIDs()`; an index outside `0..RecordCount()-1` panics as an out-of-range slice index
  does: misuse, not data.

- **FR-5 — total on arbitrary bytes.** For ANY input, `OpenDocument` MUST either return a clean
  error or return a document that writes back identically; it MUST never panic, read out of
  bounds, or return a document that fails to round-trip.

- **FR-6 — additive, walls standing.** Every existing exported identifier of the package keeps
  its shape, results and ownership behaviour and no consumer changes; the package stays a pure
  leaf importing the standard library plus `golang.org/x/text` only; all tests stay synthetic —
  no game install, no asset fixture.

- **FR-7 — the instrument.** `cmd/almtool` MUST gain a `roundtrip <file.alm>` verb: open the
  file as a document, write, compare; report identity (size and digest) or the first differing
  offset or the rejection, exiting non-zero on either failure. Developer-run; no install path
  in source.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a synthetic minimal accepted map: ten records in the shipped order, small grids, one type-4 record carrying its optional 8-byte extension | opened as a document and written | output equals input byte-for-byte; `Map()` equals `Open` of the input; `Version`, `RecordTypeIDs` and each `RecordPayload` agree with the stream |
| AC-2 | unit | headers exercising the free fields: dataSize `0xDEADBEEF`, formatVersion 991, ten distinct per-map-constant words including a signalling-NaN bit pattern, the records in a permutation the shipped maps never use | opened and written | output equals input; every named byte survives |
| AC-3 | unit | payload bytes the typed view drops: non-zero bytes after the first NUL of the type-0 name, the description and a type-5 name; non-zero type-5 head and type-6 tail bytes; a NaN-bit angle; arbitrary type-7/9 bodies, an empty and a non-empty type-8 | opened and written | output equals input |
| AC-4 | unit | the rejection set: a short stream; bad magic; file hdrLen not 20; recordCount below 3; formatVersion 1000; formatVersion above 1001; record tag not 7; record hdrLen not 20; a payload overrunning end of input; a missing type-0, type-1 or type-2; a 631-byte type-0; grid-size, type-4-walk and type-5/6-size mismatches; a type-7 shorter than its count word | each given to `OpenDocument` | an error and a nil document every time, no panic; the decision equals that of `Open` on each |
| AC-5 | unit | a document opened from a buffer | the input buffer is zeroed afterwards; a returned write buffer is mutated | a subsequent write still equals the original input; a second write is unaffected |
| AC-6 | unit | a document whose navigation results are mutated by the caller | `RecordTypeIDs` and `RecordPayload` slices are overwritten | the document writes and navigates as before — fresh copies, accessors mutating nothing |
| AC-7 | fuzz | arbitrary byte sequences — a fuzz target whose seed corpus runs under plain `go test` | opened as a document | either a clean error or a document whose write equals the input; never a panic; the accept/reject decision agrees with `Open` on every input |
| AC-8 | integration | the existing package tests, the import-graph check, every consumer | the full suite runs | green with no source change outside `pkg/formats/alm` and `cmd/almtool`; the leaf still imports stdlib + `x/text` only |
| AC-9 | manual | a lawful EN install's 38 ROM1 maps (10 at the install root, 28 embedded in `scenario.res`, extracted outside the repo) — every one of them a ten-record file | `almtool roundtrip` on each | all 38 byte-identical; per-map size and digest recorded in `verification.md`; no asset committed |
| AC-10 | unit | shapes the shipped writer never emits: three records `{0,1,2}`; four records `{0,1,2,3}`; a typeId no decoder handles; bytes after the last counted record | opened as a document and written | each accepted; output equals input; `RecordCount` is the stream's own count and `RecordTypeIDs` the ids it carried, the unhandled one included; on the three-record stream `Map()` carries a full `W·H` overlay while `Present(3)` is false, and `Write` emits no type-3 record |

Error cases: AC-4, and the error half of AC-7.

## Derived properties

- **P-1** (invariant) **Round-trip identity** — FR-2 quantified over every accepted input.
- **P-2** (negative-invariant) **Atomic, panic-free, acceptance-equivalent** — FR-1 and FR-5 over
  arbitrary bytes: never a panic or an out-of-bounds read, never a document beside an error, and
  the accepted set is exactly `Open`'s.
- **P-3** (invariant) **Copy-independence** — FR-3 across the input buffer and every write buffer.

## Constraints and alternatives

| Choice (all selected; *disclosed* = a named fidelity gap) | Observable trade-off |
|---|---|
| Byte-lossless by preservation, not regeneration: the writer emits retained bytes and MUST NOT reconstruct any header field or body from an interpreted value — dataSize never recomputed, the record order never normalized, an absent record never supplied | unedited fidelity is total and needs no format knowledge; every recompute question transfers wholly to the edit layer |
| Acceptance is exactly the shipped reader's, whatever that is — delegated, not restated — *disclosed*: that reader takes the engine loader's own four gates (magic, `recordCount ≥ 3`, `formatVersion ≤ 1001`, then type-1 and type-2 present) and stays stricter in three places: type-0 required, every payload-size constraint a rejection, `formatVersion` 1000 refused as unimplemented | a stream the game loads but `Open` rejects still gets no document; the write path adds no dialect of its own and cannot drift from the reader's |
| Bodies opaque: no payload grammar enters the document; navigation is byte-level only | the writer can never invalidate content it does not parse; an edit layer wanting typed payload access brings its own decode |
| Copy-in, copy-out | two full-size copies per round trip; the identity is never contingent on caller buffer discipline |

## Out of scope

- The edit/mutation API (`pkg/mapedit`): placing or moving content, painting grids, recomputing
  section sizes, counts or dataSize, undo/redo — including any rule for materialising a
  manufactured section as bytes.
- Assigning meaning to dataSize or the per-map constant; validating, recomputing or normalizing
  either.
- The version-1000 record-header-skipping dialect; any ROM2 dialect.
- Rendering, a save-file UI, an editor shell.

## Verification mapping

Every criterion but AC-9 runs headlessly over synthetic byte streams built in test code, the fuzz
target's seed corpus included. AC-9 is a developer run against a lawful install, from which only
commands, sizes and digests are retained; its corpus is that install's ten-record files, so
AC-10's shapes have a synthetic witness and no corpus one.

Gate coverage: FR-1 → AC-1, AC-4, AC-7, AC-10, P-2 · FR-2 → AC-1, AC-2, AC-3, AC-9, AC-10, P-1 ·
FR-3 → AC-5, P-3 · FR-4 → AC-1, AC-6, AC-10 · FR-5 → AC-7, P-1, P-2 · FR-6 → AC-8 · FR-7 → AC-9.
