# Tasks — 0057-tier-palette

Legend: **files** what the entry may change — a permission, not a prediction · **fences** what it
must not do · **done when** the observable it leaves behind. They land in ascending order, each
depending only on those before it.

## T1 — the colour-table decoder

**files** ADD `pkg/formats/pal/{pal.go,doc.go,pal_test.go}`; MODIFY `internal/archtest/dag.go`,
`docs/ARCHITECTURE.md`.

FR-1, FR-1a — DD-1, DD-2.

**fences** the package imports the standard library alone and is registered as a leaf with an empty
allow set and denied the text module by name; nothing else in the tree imports it yet. No BMP
header field is read, parsed or validated — not width, height, bit depth, `clrUsed` or the pixel
offset — and no other offset than `0x36` is consulted. The architecture document gains two rows and
loses none.

**done when** AC-1 and AC-2 hold over streams built in test code, including generated ones: the
empty stream, every length below `0x436`, a correct-length stream under every two-byte prefix, and
a table whose entries include index 0 and a byte at `0xff` in each channel.

## T2 — the name and the bounded count

**files** MODIFY `pkg/data/sprite.go`, `pkg/data/classes.go`, and the tests beside them.

FR-2, FR-3 — DD-3, DD-4.

**fences** the name is a pure string function of the same private base the sprite address is built
from — no second separator rule, no `path`/`filepath` call, no lowercasing, no existence check and
no container opened. The limit is one exported constant with the clamp as its only reader; no other
line in the tree spells 4 for this purpose.

**done when** AC-3 and AC-4 hold, including a class whose base has no separator at all and keys
below, at and above the limit.

## T3 — a class carries one frame slice per tier

**files** MODIFY `pkg/render/terrain/units.go`; ADD or MODIFY the test beside it.

FR-8 — DD-5.

**fences** one field and one method; no existing field, function or test in the package changes,
and the package gains no import. The method reads only its receiver and its argument, builds
nothing, and is defined on a nil receiver. Nothing here loads, decodes or recolours anything — this
tier is handed filled slices.

**done when** AC-6 and AC-11 hold, plus P-2, over a hand-built class: every tier below 1, above the
slice count, and naming an empty slice answers the base frames by identity, and an in-range
non-empty tier answers its own.

## T4 — the loader fills them

**files** MODIFY `pkg/game/units.go`, `pkg/game/units_test.go`.

FR-5, FR-6 — DD-6, DD-7, DD-8.

**fences** `LoadUnits` keeps its signature and its one error; every palette outcome is a skip. The
recolour shares the base frame's pixel slice by struct copy and never allocates or converts a
pixel; the equality arm returns the base slice itself, not a copy of it. The existing per-path
sheet memo is not re-keyed — the second memo sits beside it. No sheet is decoded twice.

**done when** AC-5 holds against a synthetic container built in test code carrying a sheet and four
palette addresses in the four states, and the fallback count reads 2. Mutants: the equality arm
copying instead of returning the base; the pixel slice deep-copied; the memo keyed on the sheet
address alone.

## T5 — a placement's tier reaches the draw

**files** ADD `pkg/game/tiers.go`, `pkg/game/tiers_test.go`; MODIFY `pkg/game/world.go`,
`pkg/game/frontend.go`, and the tests beside them.

FR-7, FR-9, FR-11 — DD-9, DD-10, DD-11.

**fences** `pkg/sim` and `pkg/mapload` are not edited; the resolution goes through the exported
entry points those packages already offer. Nothing is added to the world, to an entity, to the draw
seam's own type or to the four-argument constructor's signature — that one is defined in terms of
the five-argument form. The lookup is read by key and never ranged. Both draw arms — the live class
and the substituted one — take the same selector with the same number.

**done when** AC-7 and AC-8 hold over a synthetic map and definition table written in test code,
and P-4 holds over two pushes with no advance between. Mutants: the corpse arm left on the base
slice; the lookup keyed by loop index rather than by the minted id; a placement on a non-stat arm
given tier 1.

## T6 — the instrument and the still

**files** MODIFY `cmd/terraintool/main.go`, and ADD or MODIFY the tests beside it.

FR-10, FR-12 — DD-12.

**fences** the subcommand is dispatched beside the existing ones and no existing verb's flags,
output or usage text changes; the root comes from `-assets` or the environment and no install path
is written down. The still is composed through the render tier's own lit blit — no palette is
walked here. Nothing under `pkg/` changes in this entry, and the game front-end gains no flag.

**done when** the census runs on both lawful roots with the figures AC-9 states, the still of AC-10
is written outside both repositories, and a run naming no class writes no file. A test drives the
dispatch and the flag handling with no install present.

## Traceability

| Task | FR | DD | AC / SC |
|---|---|---|---|
| T1 | FR-1, FR-1a | DD-1, DD-2 | AC-1, AC-2, SC-1 |
| T2 | FR-2, FR-3 | DD-3, DD-4 | AC-3, AC-4, SC-2 |
| T3 | FR-8 | DD-5 | AC-6, AC-11, P-2, SC-3 |
| T4 | FR-5, FR-6 | DD-6, DD-7, DD-8 | AC-5, P-1, P-3, SC-4 |
| T5 | FR-7, FR-9, FR-11 | DD-9, DD-10, DD-11 | AC-7, AC-8, P-4, P-5, SC-5, SC-6 |
| T6 | FR-4, FR-10, FR-12 | DD-12 | AC-9, AC-10, SC-7, SC-8 |
