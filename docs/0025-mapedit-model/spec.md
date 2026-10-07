# Spec — map-editor E2: the headless edit / document model

## Problem and current behaviour

`pkg/formats/alm` ships a raw-backed `Document` with no mutation surface: 0023 deferred the write
side, so a map can be loaded and losslessly re-saved, but not changed.

The editor track needs headless typed mutations over it — paint a grid cell, edit the type-0
metadata, place, move and delete units — each byte-exact outside the edit, with undo and redo. An
editor that regenerated untouched regions from an interpreted view would destroy exactly the
bytes 0023 exists to preserve, the 50 undecoded bytes of every unit record among them. This story
adds `pkg/mapedit`.

`alm` has since widened to the engine's own accept set: three records and up, only type-0, type-1
and type-2 required, an absent type-3 manufactured as a `W·H` zero plane, an absent type-4..9 read
as empty. The model shipped against a roster fixed at ten and refuses anything else at load. That
refusal is a stopgap, and removing it is this revision.

## Functional requirements

- **FR-1 (load).** The model MUST be built from an ALM byte stream, accepting **exactly** what
  `alm.OpenDocument` accepts and rejecting the rest atomically — an error and no model, never a
  panic. Before any mutation it MUST expose the serialized bytes — a fresh buffer, identical to
  the input — and the view, equal to `alm.Open` of the input.

- **FR-2 (preservation — the load-bearing invariant).** Every mutation partitions the loaded bytes
  into a **target region**, the bytes it deliberately edits, and the **carried** remainder.
  Carried bytes MUST appear in the output byte-identical in content and relative order, produced
  by carrying the loaded bytes and never regenerated from an interpreted value. All little-endian;
  type-0 offsets are payload-relative:

  | Mutation | Target region |
  |---|---|
  | grid-cell set | that cell's bytes at `(y·W+x)·elem` — 2 B per type-1 tile, 1 per type-2 altitude or type-3 overlay |
  | type-0 scalar edit | that field's 4 bytes: f32 angle `+0x08`, stored scalars `+0x0c`, `+0x10`, `+0x14`, `+0x70`, `+0x74`, low-bit word `+0x18` |
  | type-0 string edit | that field's whole 64 bytes: name `+0x30`, description `+0x78` |
  | unit move | that record's X and Y words (8 B) |
  | unit place, unit delete | the added or removed 70 bytes, type-6's `payloadSize` word, `#type6` at type-0 `+0x24` |

  Carried is everything else, including four easily missed: the file header's
  `dataSize` (nothing here changes W or H), the 448-byte region of seven 64-byte text slots at
  type-0 `+0xb8`, post-first-NUL bytes of an untargeted string field, and a moved record's 62
  non-coordinate bytes. A length-changing mutation relocates every later byte by the size delta,
  so the invariant binds carried **content**, not offset (P-1).

- **FR-3 (grid paint).** The model MUST support setting one tile, altitude or overlay cell by
  `(x,y)`; the view's matching cell then equals the new value. An `(x,y)` outside `[0,W)×[0,H)`
  MUST be rejected atomically (FR-8). So MUST a set on a layer the stream does not carry —
  **type-3 alone**, since acceptance requires type-1 and type-2 — even though the view holds a
  manufactured plane for it.

- **FR-4 (type-0 edits).** The model MUST support setting each type-0 scalar named above and both
  strings; the view's matching field then equals the new value. A **string setter MUST encode with
  exactly the codec the reader decodes that field with** — ASCII for the name, Windows-1251 for
  the description — so a value the reader produced from that field is written back as the same
  bytes or rejected, never as different bytes. A string edit rewrites the **whole 64-byte field**,
  encoded bytes then NUL fill, losing that field's content past its old first NUL. An encoding
  over **63 bytes** (the field is 64 and MUST stay NUL-terminated), or a rune that codec cannot
  represent, MUST be rejected atomically (FR-8).

- **FR-5 (units).** A type-6 record is 70 bytes: X `+0x00` and Y `+0x04` are whole u32 fixed-point
  words (integer tile = `value>>8`); the other 62 hold class keys, an owner slot, sentinel runs and
  id words this model never interprets. The model MUST support, rejecting an out-of-range index or
  a record length other than 70 atomically (FR-8):
  - **place** — a **complete 70-byte record from the caller**, appended verbatim after the last
    record with `#type6` incremented, returning the new highest file-order index; no byte of it is
    synthesized, normalized or reinterpreted;
  - **move** — by index, changing only that record's X and Y words, to raw fixed-point values
    **not** checked against W and H;
  - **delete** — by index, removing the record, decrementing `#type6`, shifting later units down;
  - **read** — the unit count, and the byte-exact record at an index as a fresh copy, so a caller
    can place by cloning.

  After each mutation the view's units reflect the change and its count equals the record count.
  On a map carrying **no type-6 record** all four MUST be rejected atomically and the count MUST
  read zero whatever `#type6` says: without the records it names that word is inert. An absent
  record is not an empty one (AC-3).

- **FR-6 (the round trip survives editing).** After any sequence of successful mutations,
  re-reading the bytes with `alm.OpenDocument` and writing back MUST reproduce them exactly, and
  the view MUST succeed and reflect every mutation.

- **FR-7 (undo / redo).** The model MUST keep an ordered edit history: every **accepted** mutation
  is recorded, one whose new value equals the old included, and no rejected one is. Undo MUST
  restore the bytes to exactly their state before the most recent not-yet-undone mutation, Redo
  MUST re-apply the most recently undone mutation byte-identically, and a new mutation after an
  undo MUST discard the redo history. Undo with nothing to undo, and Redo with nothing to redo,
  MUST be no-ops leaving the model unchanged and reporting that nothing happened.

- **FR-8 (atomicity).** Every mutation MUST be atomic: on rejected arguments it returns an error
  and leaves the bytes, the view and the history unchanged, entering no history.

## Acceptance criteria

The **rich fixture** is one map whose every carried region holds a distinctive non-zero value: a
non-default `dataSize`, the per-map constants, a signalling-NaN angle, post-NUL string bytes, the
text slots, and a unit record's undecoded `+0x14`/`+0x35`/`+0x3b`/`+0x40`/`+0x42`.

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a map loaded into the model | read before any edit | the bytes equal the input exactly; the view equals `alm.Open` of the input |
| AC-2 | unit | the rich fixture | each on a fresh load: a tile, altitude and overlay cell; the name, the description (Cyrillic, as code points) and each u32 scalar; the unit moved by index | every output differs from its input only inside that mutation's target region; every byte named above survives, text slots and NaN bits included; the view's matching cell, field or coordinate equals the new value |
| AC-3 | unit | a map with an empty type-6 payload, and one with N≥1 units | a 70-byte record is placed, then a unit deleted by index | `#type6` rises then falls by one; type-6 grows and shrinks by exactly 70 bytes with a matching `payloadSize`; the placed record reads back byte-identical at the returned index; other records are unchanged; the view reflects both, and the bytes re-open via `alm.OpenDocument` and write back identically |
| AC-4 | unit | a loaded map | a description over 63 encoded bytes; one with a rune Windows-1251 cannot encode; a name with a non-ASCII rune; a unit op at an out-of-range index; a place with a 69- and a 71-byte record; a cell set at an out-of-range `(x,y)` | each is rejected; bytes, view and history unchanged |
| AC-5 | unit | a loaded map | M, Undo, Redo; M1, M2, Undo, Undo, Redo, Redo; a new mutation after an Undo | Undo gives the pre-M bytes, Redo the post-M bytes; Undo, Undo restores the original and Redo, Redo post-M2; the mutation after an Undo makes a later Redo a no-op |
| AC-6 | unit | a freshly loaded model, empty history | Undo, then Redo | each is a no-op reporting nothing happened; the bytes are unchanged |
| AC-7 | fuzz | a seeded pseudo-random sequence of in-scope mutations interleaved with Undo and Redo | applied step by step | after every step the bytes re-open and write back identically and the view succeeds; undoing to the empty history restores the input exactly; no panic |
| AC-8 | unit | maps `alm` accepts that omit a record: no type-3; no type-6 with a non-zero `#type6`; three records; a type-6 past the tenth; a count stopping before both | loaded, then every setter driven | each loads and its view equals `alm.Open` of it; the setters for records it carries succeed and re-open; `SetOverlay` with no type-3 and all four unit calls with no type-6 are rejected, bytes, view and history unchanged; the count is zero with no type-6 |

Error cases: AC-4 and AC-8, and the boundary AC-6.

## Derived properties

- **P-1** (invariant) For any fixed-length mutation the byte diff is a subset of its declared
  target region; for a length-changing one every carried byte appears identical in content and
  relative order, relocated only by the size delta.
- **P-2** (invariant) For any edited model, re-opening its bytes and writing them back reproduces
  them exactly.
- **P-3** (invariant) Undo restores the pre-mutation bytes and Redo the post-mutation bytes,
  exactly.
- **P-4** (negative-invariant) For any mutation rejected for invalid arguments, the bytes, the
  view and the history do not change.
- **P-5** (negative-invariant) Undo on an empty history, and Redo on an empty redo stack, leave
  the bytes unchanged.
- **P-6** (invariant) For any string the reader decodes from a field, setting the field to it
  writes that field's original bytes back or is rejected, never different bytes.
- **P-7** (negative-invariant) For any call addressing a record the accepted stream does not
  carry, the call is rejected and the bytes, the view and the history do not change.

## I/O examples

Observable API shape in `pkg/mapedit`.

```text
New(data []byte) (*Editor, error)
(*Editor).Bytes() []byte
(*Editor).Map() (*alm.Map, error)
(*Editor).SetTile(x, y int, v uint16) error      // and SetAltitude, SetOverlay (uint8)
(*Editor).SetAngle(v float32) error
(*Editor).SetWord0C(v uint32) error              // and SetWord10/14/70/74, SetBitmask
(*Editor).SetName(s string) error                // and SetDescription
(*Editor).UnitCount() int
(*Editor).UnitRecord(i int) ([]byte, error)
(*Editor).PlaceUnit(record []byte) (int, error)  // returns the new index
(*Editor).MoveUnit(i int, x, y uint32) error
(*Editor).DeleteUnit(i int) error
(*Editor).Undo() bool                            // and Redo, CanUndo, CanRedo
```

## Constraints

- **The roster is whatever the stream carries.** Only type-0, type-1 and type-2 are guaranteed;
  type-3 and type-6 — the only others this model addresses — may be absent, and an absent record
  is not an empty one. No mutation adds or removes a record, so the record-count word never
  changes.
- **Additive, walls standing.** Every exported identifier of `pkg/formats/alm` keeps its shape and
  results, no consumer changes, and `pkg/mapedit` is registered in the dependency DAG importing
  `pkg/formats/alm` and the standard library and **no external module** — the `golang.org/x/text`
  grant belongs to the formats tier.
- **Scope is ROM1**, the version-990 dialect `alm` accepts. The editor is a stateful
  single-goroutine session.

## Out of scope

- **type-4 record place/move/delete** and its `0x21`-kind extension, the type-5 roster, and the
  type-7/8/9 payloads, which stay opaque; the type-0 count words other than `#type6`, the
  `+0x28`/`+0x2c` words, and the text slots.
- **Creating a record the map does not carry** — a header, an empty payload and the file header's
  record count. The answer to FR-3's and FR-5's absent-record rejections, and the only mutation
  that would change the roster; deferred, not ruled out.
- **Map resize** (W, H and reallocating the grids) and recomputing the file header's `dataSize`.
- **Validating an edit against the game** — a coordinate inside the grid, a class key or owner
  slot that resolves, a unique id word.
- **Persisting the undo history**; a save-file UI, a diff view, pre-save validation; any editor
  shell, GUI or `cmd` tool; any change to `alm.Open`, `alm.Map` or the existing decode and
  round-trip behaviour; the version-1000 dialect; ROM2.

## Verification mapping

All CI-automatable over synthetic byte streams; a seeded fuzz loop covers AC-7.

## Gate check

FR-1 → AC-1, AC-8 · FR-2 → AC-2, AC-3, P-1 · FR-3 → AC-2, AC-4, AC-8 · FR-4 → AC-2, AC-4, P-6 ·
FR-5 → AC-2, AC-3, AC-4, AC-8 · FR-6 → AC-3, AC-7, P-2 · FR-7 → AC-5, AC-6, P-3, P-5 ·
FR-8 → AC-4, AC-8, P-4, P-7.
