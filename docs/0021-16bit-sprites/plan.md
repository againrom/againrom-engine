# Plan — the two 16-bit sprite decoders

## Baseline

`pkg/formats/spr256` is the container's sibling: one `Decode(data []byte) (*Sprite, error)`,
layout constants at the top of the file, errors built with `fmt.Errorf("spr256: …")`, refusal by
return — no panic path. Its trailer's bit 31 is a has-palette flag the decoder obeys, and its
`decodeFrame` requires the block to tile the grid exactly — the two rules this contract replaces
with a declaration and with tolerance, so neither is ported. `cmd/sprtool` holds the one
subcommand `png <archive> <path> <dir>`, reads the entry through `res.Open`/`ReadFile`, resolves
pixels in `colorFor` and writes `frame_%03d.png` through `writePNG`; `sprtool-out/` is
gitignored. `internal/archtest/dag.go` is fail-closed — an unregistered package is a violation —
and `noExternalFormats` denies `pkg/formats/reg` the formats tier's `golang.org/x/text` grant, its
comment leaving `spr256` undenied only because widening the map was another story's change.
`docs/ARCHITECTURE.md` carries a tier row and a DAG row per package and names the allow map
authoritative. `internal/synth` imports stdlib alone and holds no expected values. The suite runs
green with no game install, and fuzz targets run their seed corpus under plain `go test`.

## Design decisions

### DD-1 — one package, two decoders, two result types

`pkg/formats/spr16` enters beside `spr256`: bytes in, structs out. The `.16a` entry is
`DecodeA(data []byte, palette bool) (*SpriteA, error)` — presence is the caller's declaration
(FR-3), so it is a parameter, never a sniff — with `SpriteA{Palette []Color; Frames []FrameA}`
holding 256 entries when declared and nil when not, which is how AC-7's "reported absent" reads.
The `.16` entry is `DecodeG(data []byte) ([]FrameG, error)` — G for the spec's own "glyph value" —
with no wrapper type, because a `SpriteG` would hold one field and imply the palette slot the
format must never grow. Frames are `FrameA`/`FrameG{Width, Height int; Pixels []PixelA/G}`,
row-major with exactly Width×Height elements (P-1); pixels are
`PixelA{Index, Level uint8; Painted bool}` and `PixelG{Value uint8; Painted bool}` — the raw
fields and nothing else, so FR-5's bar (no resolved colours, no expanded values, no merged fields)
holds in the type shape rather than by discipline. The zero value is the transparent pixel and an
unpainted cell keeps zero fields, so a painted level or value 0 differs from transparent exactly
in `Painted` (AC-10, P-5). `Color{R, G, B uint8}` mirrors the sibling: on-disk BGRx reordered, the
4th byte dropped. On success the frame list — and a zero-area frame's `Pixels` — is empty and
non-nil (AC-14); on any error both functions return a nil result (P-2).

Rejected: one `Decode(data, format, palette)`, which makes a `.16` with a palette expressible and
merges result types the grammars keep distinct; and the sibling's `Opaque` for the flag — this
contract's word is painted, and a level is not opacity.

### DD-2 — one container walk and one cursor, shared by both grammars

`container.go` owns what the grammars share. `records(data []byte, start int) ([]record, error)`
refuses a stream shorter than `start + 4` before its trailer read, takes
`count = trailer & 0x7FFFFFFF` from the last 4 bytes — bit 31 is never read into anything, so P-3
holds by construction — and walks exactly `count` records from `start`: caps per DD-3, header and
block fitting before the trailer's first byte, each `record{width, height uint32; block []byte}`
a sub-slice of the input. Bytes between the last counted record and
the trailer are never touched — the leftover tolerance is the walk stopping at `count`, not a scan
(AC-4). `DecodeG` passes `start` 0; `DecodeA` passes 1024 when declared, else 0, and slices the
palette bytes only after the walk returns — AC-8's 4-byte declared stream refuses on length,
panic-free.

`cursor{w, h, pos}` owns the grid discipline both grammars cite: `skip(n)` and `rows(n)` advance
`n` and `n×w` — a blank-row op is plain cursor arithmetic from any position — `paint()` yields the
linear index then advances, and each refuses a move or paint past `w×h` while allowing a landing
exactly on it; the caps keep every cursor product overflow-free. The cursor also carries the
contract's completion rule — both decode loops announce each op's count before acting: once `pos`
equals `w×h` (a zero-area grid is born complete) a non-zero count refuses, a zero-width blank-row
op included though it would move nothing; count-0 ops pass anywhere. A block may exhaust with the
grid unfinished — the remainder simply stays
transparent — and `width` or `height` 0 is a valid header for an empty grid. Every position rule
exists once, so AC-6 and P-4 mean one thing in both formats. Rejected: a grid generic over the
pixel type — the shared discipline is this arithmetic, not the paint loop.

### DD-3 — caps and refusals: locals until success, one error surface

The caps are the spec's Constraints A numbers verbatim, as unexported constants: the stream cap
checked before anything else is read, the count cap at the trailer, width/height/dataSize per
record header. Every refusal is `fmt.Errorf("spr16: …")` with no typed classes (FR-6 lets them be
indistinguishable) and returns before any result exists: both decoders build into locals and
return only on full success, so an error yields no frames and no palette with no rollback path
(P-2). Each `make` is sized by capped fields, so no further overflow guard is needed. A control or
operand that would cross its block's end refuses at the read, before any cursor move (AC-11); the
`.16a` odd trailing byte where a word should start is the same refusal.

### DD-4 — the `.16a` block: u16 reads, two masked fields

The block decoder reads u16 control words — `op = cw >> 14`, `n = cw & 0x3FFF` — and op `0b00`
reads `n` further words, painting `PixelA{(ss>>1)&0xFF, (ss>>9)&0x0F, true}`; bits 0 and 13–15 are
masked off unconditionally, never validated — a set bit is not an error class the contract names.
Ops `0b01`/`0b10` map to `rows`/`skip`, and `0b11` is decoded by the same switch case as `0b01` —
one arm, so AC-13's `.16a` half is equality by construction rather than two paths agreeing.

### DD-5 — the `.16` block: byte controls, the pad on the final byte alone

Op and count come from one byte — `op = b >> 6`, `n = b & 0x3F` — and the three non-literal ops
reuse DD-4's cursor calls, `0b11` sharing `0b10`'s arm here: each grammar's alias matches its own
decoded blitter, in opposite directions. A literal of `n` bytes paints low nibble then
high per byte; the one exception is the run's final byte with high nibble 0 — a pad, nothing
emitted — so the pad test is `j == n-1 && b>>4 == 0`, evaluated per run, never per block, and a
mid-run zero high nibble paints value 0 and advances (AC-9). An `n = 0` literal reads and paints
nothing — a no-effect op the completion rule permits anywhere.

### DD-6 — the dump extends `cmd/sprtool`

The manual check is two new `sprtool` subcommands, `png16a` and `png16 <archive> <path> <dir>`
beside `png`: one sprite dump tool already owns the archive plumbing, the PNG writing and the
output-hygiene doc, where a sibling command would re-own all three and add an archtest row for one
story's manual check. `png16a` declares the palette present — the tool's own fixed declaration —
and renders a painted pixel as its palette RGB with alpha `level×17`, transparent as alpha 0;
`png16` renders a painted value as opaque gray `value×17`. Both print one presentation line to
stderr on every run — naming the declaration and the ramp — and carry it in the usage text, which
is AC-12's disclosure: viewing choices of this tool, stated as such, never format facts (FR-8).
Frames land as `frame_%03d.png` as `png` does, empty frames skipped with a note; the
`cmd/sprtool` allow-map row gains `pkg/formats/spr16`.

### DD-7 — the package enters the DAG enforced, not documented

The allow map is fail-closed, so `pkg/formats/spr16: {}` enters it in the same change that creates
the package — between the two the tree is red. It also enters `noExternalFormats`: neither grammar
carries text, so the formats tier's x/text grant is denied by mechanism, the way `reg` already is
— the dag comment's own rule applied to a package that is this story's to hold. `docs/ARCHITECTURE.md`
gains the matching tier and DAG rows in the same change — the tables must stay identical to the
map — and its `cmd/sprtool` row widens when the tool does (FR-7).

### DD-8 — fixtures are the spec's own tables, built beside the tests

Every fixture is bytes assembled in the package's test files — small helpers (a u16 writer, a
header-block-trailer assembler) local to the tests; `internal/synth` is untouched, since it holds
no expected values and a stream builder there would put the layout in two places. The spec's two
I/O examples are transcribed as fixtures with their printed grids as expected values, and its
numeric anchors pin the wide-count path: the 551-skip word `27 82` on a 552-wide frame — a count
no 8-bit field carries — and the trailer pair `01 00 00 80`/`01 00 00 00` decoding identically —
deep equality of two decodes differing in bit 31 alone (AC-2). Expected grids are
literals beside their fixtures, never values recomputed through the decoder's own arithmetic. Each
decoder also carries a fuzz target seeded in-file with `f.Add` from its malformed table — no
committed corpus — asserting only P-2, no panic and a nil result on error, and running as seed
corpus in the ordinary suite (FR-7).

## Success criteria

- **SC-1** The container walk and cursor, exercised directly: `records` reads the count through
the mask on both bit-31 trailer forms, stops at `count` leaving later bytes unread, and refuses
each cap one over its limit — width, height, dataSize, count, stream length — a header or block
crossing the trailer, and a stream shorter than its fixed regions; the cursor lands exactly on
`w×h`, refuses one step past it by skip, rows and paint alike, advances rows from a mid-row
position, and refuses a non-zero announced count on a complete grid — the zero-width form included
— while passing count 0 anywhere; `pkg/formats/spr16` is registered in the allow map, denied
x/text, and both ARCHITECTURE tables carry its rows (FR-4, FR-6, FR-7).
- **SC-2** AC-1, AC-10, AC-13 and AC-14 hold for `.16a` as written — AC-1's stream carrying its
blank rows mid-frame on a width above one, so a skip-miswired arm lands every later pixel wrong,
and the `27 82` word skipping 551; the spec's printed example decodes to its printed grid. Beyond
the ACs: a block exhausting mid-grid leaves the remainder transparent, a zero-area frame decodes
empty, trailing count-0 ops change nothing; every buffer is exactly Width×Height with only
literal-painted cells non-zero (FR-1, FR-4, FR-5, P-1, P-5).
- **SC-3** AC-2, AC-4 and AC-7 hold for `.16a`: bit-31 twins decode deeply equal; appended bytes
and a further well-formed record section change nothing; one payload decoded under both
declarations yields frames from offset 1024 and 0 respectively, palette present then nil (FR-3,
FR-4, P-3).
- **SC-4** AC-3, AC-5, AC-6, AC-8 and AC-11 hold for `.16a`, each malformed stream refusing with
no sprite; a non-zero count after the grid completes refuses; the fuzz target runs its malformed
seeds panic-free (FR-6, P-2, P-4).
- **SC-5** AC-9, AC-13 and AC-14 hold for `.16` as written — the spec's `.16` example decoding
to its printed grid; a blank-row run between literals on a width above one lands the following
pixels exactly; a block exhausting mid-grid stays transparent and a zero-area frame decodes empty;
P-1 and P-5 asserted as in SC-2 (FR-2, FR-4, FR-5, P-1, P-5).
- **SC-6** AC-2, AC-4, AC-6, AC-8 and AC-11 hold for `.16` as written; a non-zero count after
completion refuses; the fuzz target runs its malformed seeds panic-free (FR-4, FR-6, P-2, P-3,
P-4).
- **SC-7** `sprtool` builds with `png16a` and `png16` present and `png` unchanged; its allow row
names `pkg/formats/spr16` and nothing else in the map widens; the whole suite is green with no
game install and no window (FR-7, FR-8).
- **SC-8** (manual) AC-12: a real cursor `.16a` and a real font `.16` dumped through the tool are
upright and recognizable, the presentation lines printed beside them (FR-8).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-1, DD-2, DD-4 | SC-2 |
| FR-2 | DD-1, DD-2, DD-5 | SC-5 |
| FR-3 | DD-1 | SC-3 |
| FR-4 | DD-2 | SC-1, SC-2, SC-3, SC-5, SC-6 |
| FR-5 | DD-1, DD-4 | SC-2, SC-5 |
| FR-6 | DD-3 | SC-1, SC-4, SC-6 |
| FR-7 | DD-7, DD-8 | SC-1, SC-7 |
| FR-8 | DD-6 | SC-7, SC-8 |
