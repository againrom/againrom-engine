# Tasks — resolve, walk, wire, and let the owner see it

Legend: **files** what the entry may change — a permission, not a prediction · **fences** what it must
not do · **done when** the observable it leaves behind. They land in ascending order, each depending
only on those before it. Each entry names the mutants SC-2 charges to the lines it writes, applies
them where it wrote them and reverts them there.

## T1 — the resolution

**files** ADD `pkg/mapload/structures.go`, `pkg/mapload/structures_test.go`; MODIFY
`pkg/mapload/spawn.go`, `pkg/mapload/doc.go`

FR-1, FR-2, FR-8 — DD-1, DD-3, DD-4, DD-7, DD-11.

**fences** nothing in `passability.go` or `fromalm.go` changes and no test of either is edited; the
package gains no import. `Table`'s two existing fields and every function over them are untouched.
The cell-level counters exist and stay zero here — no walk, no plane, no occupancy. Neither the map
nor the collection is written through: the resolution reads `EntryParams` and copies out.

**done when** AC-1 and AC-4's resolution half hold, each skip counted under its own name and none of
them an error, and P-1 and P-6 hold over a generated key set. Mutants: the entry guard's upper bound
taken as `<=`; the extension test taken on the low byte instead of the whole key; the sum gate
replaced by the kind; the extents taken as the full parameter instead of its low byte; the
zero-extent test moved ahead of the extension override.

## T2 — the walk, the pass and the census

**files** MODIFY `pkg/mapload/structures.go`, `pkg/mapload/structures_test.go`, `pkg/mapload/doc.go`

FR-3, FR-4, FR-5, FR-7, FR-8 — DD-2, DD-5, DD-6.

**fences** `Passability`'s signature, body and behaviour do not change, and `Census` is not touched —
the arms are not re-derived, re-ordered or re-counted here. The occupancy is this derivation's own
and is not exported. `blockGround` is reused, never respelt, and no other bit is written by any
operator on this path. Nothing in `pkg/game`, `pkg/ui` or `cmd/` is edited.

**done when** AC-2, AC-3, AC-5, AC-6, AC-7, AC-10 and AC-4's walk half hold, and P-2, P-3, P-4, P-7
and P-8 hold — the last over anchors placed inside, overhanging and wholly outside. Mutants: the bit
index taken without the modulus; the abandon latch dropped so a refusal only skips its own cell; a
dropped cell latching the abandon; the closing arm and the opening arm exchanged; the opening arm
written as `plane[i] = 0`; the row and column loops exchanged.

## T3 — the world builder and the tool

**files** MODIFY `pkg/mapload/fromalm.go`, `pkg/mapload/fromalm_test.go`,
`cmd/classdump/databin.go`, `cmd/classdump/databin_test.go`

FR-6, FR-7 — DD-8.

**fences** `FromALM`'s and `FromALMWith`'s signatures do not change and no field is added to a world
or an entity; the census is not returned through either. `pkg/game` is not edited, so the viewer's
own plane keeps coming from the table-less call. classdump's existing per-placement lines, arm
summary and table report keep their wording character for character; the census is appended after
them.

**done when** AC-8 holds — three table shapes yielding planes and canonical world byte forms
identical to the pre-story ones, pinned against a digest taken before the change — and the census
prints both levels from a table built in test code with no install present. Mutants: `FromALM`
passing the caller's table instead of nil; the census printed from a second walk of its own; the
`Buildings` collection omitted from the tool's table.

## T4 — the tint

**files** ADD `pkg/render/terrain/blocked.go`, `pkg/render/terrain/blocked_test.go`,
`cmd/mapview/blocked.go`, `cmd/mapview/blocked_test.go`; MODIFY `pkg/render/terrain/overlay.go`,
`pkg/ui/viewer.go`, `pkg/ui/overlay.go`, `pkg/ui/overlay_test.go`, `cmd/mapview/main.go`,
`internal/archtest/dag.go`

FR-9 — DD-9, DD-10.

**fences** `cmd/againrom` and `pkg/game` are not edited at all. `pkg/ui` gains two fields, one setter,
one reporter, one rects builder and one line in `overlayPasses`; no existing pass, colour, glyph or
order moves, and the viewer still names no map, plane or table type. `pkg/render/terrain`'s existing
glyphs, colours and helpers are unchanged — `overlay.go` gains nothing but the colour beside its
siblings. `dag.go` changes one row. No asset path is spelled in any source file.

**done when** AC-11 holds over a hand-built plane, an unflagged run's summary and pass slice are
character-for-character and element-for-element the pre-story ones, and a missing table is an error
before any viewer is built. Mutants: the tint's cells taken from bit 1; the pass appended last
instead of prepended; the colour made opaque; the flag defaulted on; the table resolved from a
compiled-in path.

## Traceability

| Entry | FR | DD |
|---|---|---|
| T1 | FR-1, FR-2 | DD-1, DD-3, DD-4, DD-7, DD-11 |
| T2 | FR-3, FR-4, FR-5, FR-7 | DD-2, DD-5, DD-6 |
| T3 | FR-6, FR-7 | DD-8 |
| T4 | FR-9 | DD-9, DD-10 |
| T1, T2 | FR-8 | DD-11 |
