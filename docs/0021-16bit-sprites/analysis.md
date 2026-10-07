# Analysis — the two 16-bit sprite formats

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** — the `pkg/formats/*` default: the format spec is the durable contract and doubles as the reverse-engineering documentation; no watcher tool exists, so it is static + discipline |
| Terrain | **greenfield** — a new decoder package; nothing in this tree parses either format or has behaviour to preserve |

## Three of the baseline's load-bearing rules fail against the ledger

The staged baseline is our own earlier clean-room pass over this same decoder, and its formulas are
hypotheses to re-derive, not facts to adopt. Re-deriving them against `claims/spr16a.md` at the
current pin breaks three:

- **The op-word count.** Baseline: `N = op & 0x00FF`, bits 8–13 "always 0". The instruction-anchored
  grammar reads `op = cw >> 14`, `count = cw & 0x3FFF` (`SPR16A-RLE-002`), and the census
  (`SPR16A-BOUND-016`) measures shipped skip runs of 551 px on 552-wide frames — counts that cannot
  fit 8 bits. The baseline's own corpus figures show the wound: it reports maxima of 148×128 and
  2 515 187 literal words where the ledger's walk over the same corpus finds 552×128 and 2 535 421,
  and its "`.16a` leftover up to 8 350 B" is the discarded pixel data itself —
  `SPR16A-STRUCT-001` closes all 542 `.16a` with a residual of exactly 4, so no shipped `.16a` has
  any leftover. (`.16` really is `[2-bit op][6-bit count]` bytes — `SPR16A-FONT-013`.)

- **Null frames.** Baseline: a header with `w>512 || h>512 || dataSize>1e6` is consumed but its RLE
  block skipped, and the walk continues. No claim at this pin backs any null-frame semantics, and
  on our corpus the rule is worse than unbacked: a shipped sheet is 552 wide, so the rule would
  skip a real frame's pixel block and re-parse pixel data as headers — a silent desync, not a
  tolerance. What replaces it is caps that are ours by choice, reject with an error, and clear the
  measured maxima by a wide margin.

- **"Alpha".** Baseline: bits 9–12 are a 4-bit alpha, expanded by nibble replication. The bit split
  survives and is now instruction-level (`SPR16A-PIX-011`: index bits 1–8, second field bits 9–12),
  but the field is a **level** selecting one of 16 rows of a runtime LUT with a framebuffer term
  added — not an alpha channel — and replication has no claim behind it at any grade. So the
  preservation logic the baseline's own FR-7 applied to the index extends to the level: keep
  `{index, level}` raw, and let the dump tool disclose whatever it does to make that viewable.

## What survives, and what the trailer can and cannot say

The container shape, the consumer-declared palette, the cursor and bounds discipline, and the `.16`
nibble grammar — low nibble first, the odd-run pad, a mid-run zero nibble painting a real zero —
all match the ledger (`SPR16A-STRUCT-001`, `SPR16A-FONT-013`) and stay. The trailer's low 31 bits
are the frame count as an exact identity (`SPR16A-TRLR-012`), so the spec now reads the count
rather than only walking; bit 31 correlates exactly with palette presence, but that partition
coincides with the `.16a`/`.16` extension split, so the corpus cannot make it a palette gate —
presence stays consumer-declared and the output stays independent of the bit.

Two corpus corrections fold in. Five font atlases ship, not three (`SPR16A-FONT-015`) —
`font4.16a`/`font5.16a` are ordinary palette-bearing sheets the `.16a` path covers with no special
casing. And `font1.16`/`font2.16` continue past their count trailer into appended record sections
(`SPR16A-FONT-014`), so the "ignore leftover before the trailer" tolerance is load-bearing, not
politeness: it is what decodes two of the three `.16` files at all.

## Where the decoder lands

A new `pkg/formats/spr16`, stdlib tier of the DAG beside `pkg/formats/spr256` — bytes in, structs
out, no archive knowledge. `internal/archtest` is fail-closed on any package it does not list, so
the package enters the allow map together with the code in a later stage; named here so it is not
rediscovered as a red gate. `cmd/sprtool` already dumps `.256`; whether the manual-check dump
extends it or stands alone is a plan decision — the contract stays tool-shape-free, with the
viewer's conventions (palette application, how a level or glyph value is shown) disclosed as
presentation.

## What we looked at

`docs/0002-sprites-256/spec.md` and `pkg/formats/spr256` (the sibling container),
`internal/archtest/dag.go`, `cmd/sprtool`, `docs/ARCHITECTURE.md`; in research (pin `e53c779`)
`claims/retracted.md` first — eight of its rows are this format's own overturns — then the whole
`claims/spr16a.md` ledger and `formats/spr16a/format.md`; and the staged baseline spec, read as a
hypothesis set.

Open rather than guessed: what bit 31 names; whether the count field is `[31-bit][1]` or
`[u16][u16]`; whether the loader reads the trailer at all; the `0b11`/`0xC0` quadrant's semantics;
what a `.16` glyph's 4-bit value means; whether the `.16` loader stops at the count trailer. Each
is disclosed in provenance with what would settle it.
