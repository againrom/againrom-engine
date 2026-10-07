# Provenance — the two 16-bit sprite formats

Pinned at research `e53c779`, **bumped mid-story to `778c2a6`** — the one exception to the freeze,
taken because EXP-0047 read this format's own loaders while no code existed yet; every citation
re-checked at the bump. `claims/retracted.md` read at both pins: nine of its rows are this
format's own overturns.

## Backing

| `spec.md` anchor | Claim | Confidence |
|---|---|---|
| Shared container and frame-record layout | `SPR16A-STRUCT-001` | High — 542/542 `.16a` walk clean from offset 1024, residual exactly 4; the two formats share the shape because the `.16a` loader *is* the `.256` constructor |
| Trailer — `frameCount = trailer & 0x7FFFFFFF` (FR-4) | `SPR16A-TRLR-012` (amended), `SPR16A-RDR-017` | High for `.16a` — its loader executes exactly this mask (`AND 0x7fffffff`). The `.16` loader takes the u32 **raw**, so applying the mask there is ours — see below |
| `.16a` grammar — `op = cw >> 14`, `n = cw & 0x3FFF`; exactness | `SPR16A-RLE-002` (amended), `SPR16A-RLE-003` | High — instruction-anchored in `rom.exe`, closure exhaustive over 2727/2727 frames |
| Op `0b11` aliases (AC-13) — blank rows in `.16a`, skip in `.16` | `SPR16A-RLE-002` (amended), `SPR16A-FONT-019` | High — both blitter arms read; opposite directions in one binary, so neither transfers to the other. Still a measured absence in every shipped stream |
| `.16a` pixel word — index bits 1–8, level bits 9–12 (FR-5) | `SPR16A-PIX-011` | High — instruction-level: the word is an even byte offset into a runtime `[16][256]` LUT, forcing the 4+8 split |
| Palette — 256 × `[B,G,R,x]`, indexed by the pixels | `SPR16A-PAL-008` | High — role and channel order instruction-read via the LUT builder |
| `.16` glyph grammar — low nibble first, odd-run pad (FR-2, AC-9) | `SPR16A-FONT-013` (amended) | High — the byte blitter's control switch and nibble advance read instruction by instruction; the 4-bit value indexes a **caller-supplied 16-entry ramp**, which is why preserving it raw is the only faithful output |
| Leftover-before-trailer tolerance (FR-4, AC-4) | `SPR16A-FONT-014` (amended) | High — the walk's only exit test is the count, so `font1.16`/`font2.16`'s appended sections are bytes the engine can never reach; taking the last 4 bytes as the trailer and ignoring the middle is what decodes them |
| No engine-side header validation (Constraints A's premise) | `SPR16A-RDR-017` | High — neither loader checks `width`, `height` or `dataSize` at all: no threshold, no bound, no rejection path. Any cap is therefore ours alone |
| Cursor orientation — row 0 top | `SPR16A-PIX-011`, `SPR16A-FONT-015` | Medium — carried by visually confirmed renders (upright anchor sprites, legible glyphs), not a named instruction |
| Cap clearance (Constraints A) | `SPR16A-BOUND-016` | High as a census (width 552, height 128, frames 224, `dataSize` 20 892, payload 206 328) |

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| **Sanity caps reject with an error** — 2048/2048 w/h, 4096 frames, 2^24 `dataSize`, 2^26 stream (FR-6) | The engine validates nothing (`SPR16A-RDR-017`) and the fields are u32, so any cap is a decision. The numbers are the research format spec's suggestions, each well above the census figure it clears |
| **Palette presence consumer-declared (FR-3); bit 31 masked for both formats (FR-4, P-3)** | Decoded: the `.16a` loader gates its 1024-byte palette read on bit 31; the `.16` loader neither masks the trailer nor reads a palette — bit 31 clear on all three shipped `.16` is *required* by its raw count read. The declaration keeps palette policy at the API instead of trusting a bit only one of the two loaders consults; a consumer may derive its declaration from bit 31. Disclosed divergence: a `.16` trailer with bit 31 set decodes here (masked) where the engine would ask for ~2^31 records |
| **Literal-word bits 0 and 13–15 masked, not validated** | Zero across all 2 535 421 shipped literal words; engine behaviour with one set is unobserved, and extraction ignores them |
| **The off-corpus tolerances** — whole-block consumption with trailing no-effect ops, unfinished grids transparent, blank rows as plain cursor arithmetic anywhere, zero `width`/`height` as an empty grid | The corpus is exact — every shipped frame closes its grid and its block together, blank-row ops occur only at column 0, minima are 12×12 — so these bind only off the corpus, admitting and rejecting nothing shipped |
| **`{index, level}` and the `.16` value preserved raw** (FR-5) | The baseline's `alpha8 = nibble × 17` has no claim behind it at any grade: the `.16a` field selects a LUT row with a framebuffer term added, and the `.16` value indexes whichever 16-entry ramp the caller passes (13 ship, `base*k/15`) — no single alpha or gray represents either |
| **Painted level/value 0 admitted, distinct from transparent** (AC-10, P-5) | Never 0 in any shipped `.16a` literal (1..15), but the field admits it; transparency here is structural (skip/blank ops), never a pixel value |

## Open / undecoded

- **The palette entry's 4th byte** — no known role.
- **The runtime blend, the per-display-mode framebuffer packing, and which of the 13 text ramps a
  given string gets** — engine runtime, outside a stream decoder; named so a renderer story knows
  where the boundary ran.

## Removed from the baseline and why

- **`N = op & 0x00FF`, "bits 8–13 always 0".** Refuted: the instruction-anchored read is
  `count = cw & 0x3FFF` (`SPR16A-RLE-002`), and shipped skip runs reach 551 on 552-wide frames
  (`SPR16A-BOUND-016`).
- **The null-frame machinery** — `w>512 || h>512 || dataSize>1e6` consumes the header, skips the
  RLE block, keeps walking. No such semantics exist anywhere: the engine has no header validation
  at all (`SPR16A-RDR-017`), and on this corpus the rule skips a real 552-wide frame's pixel block
  and desyncs silently. Replaced by reject-with-error caps.
- **The baseline's corpus figures** — 2 515 187 literals, maxima 148×128, "`.16a` leftover up to
  8 350 B". Artifacts of the two rules above: the true walk finds 2 535 421 literals, width max
  552, and a residual of exactly 4 on every `.16a` — the "leftover" was discarded frame data.
- **"Alpha" and `alpha8 = alpha4 × 17`** — with the multiple-of-17 property and the
  alpha-0/transparent conflation. The field is a level (`SPR16A-PIX-011`); replication has no
  claim; expansion moved to the dump tool as disclosed presentation.
- **"The 3 `.16` files are the fonts" and "glyph metadata lives in a sibling `.dat`"** — five
  atlases ship, two of them ordinary palette-bearing `.16a` (`SPR16A-FONT-015`), and the `.dat`
  is now decoded as the per-glyph **advance table** (`SPR16A-FONT-018`) — still out of this
  story's scope, which decodes the sprite container only.
- **The provenance-basis preamble and the Research-needed sections (its R-1/R-2)** — refused by
  the doc-budget content bans; their live content is in the registers above.

## Appended 2026-08-01 — pin `130bb79`: one cited row is REFUTED on a clause this reader does not use, and two are on the new High-bar list

**`SPR16A-STRUCT-001` — REFUTED**, and the struck clause is "the frame count is **not** stored":
it is, in the same four bytes `.256` stores it in, 545/545. The row is cited above for the shared
container and frame-record *layout*, which stands, and this reader takes the count from the trailer
(`trailer & 0x7FFFFFFF`, `SPR16A-TRLR-012` / `SPR16A-RDR-017`) rather than walking to find it — so
it is already on the corrected side. The cost the overturn names is a consumer that walks the
records for a number it could read, and "had no way to reject a truncated sheet"; this reader
rejects one, because a counted record that does not fit before the trailer is an error.

**Two rows here are named by the pin's new High-bar sweep.** `retracted.md`'s *High the row's own
warrant does not carry* lists `SPR16A-STRUCT-001` ("the walk closes 542/542 … the `rom.exe`
citation covers only the leading block") and `SPR16A-BOUND-016` ("a direct census of the shipped
corpus with no model interposed"), among twenty-five rows reported and **deliberately not
re-graded**. The instruction to a consumer is to treat them as Medium. Both are cited above at
High, and the second's cell already says "High **as a census**" in the story's own words, which is
the shape the sweep flagged. Nothing here reaches hashed simulation state, so this is a note and
not a disclosure: the cap clearance `SPR16A-BOUND-016` backs is *ours* either way — the *Ours by
choice* rows already say no engine-side validation exists and therefore that every cap is ours —
and the layout `SPR16A-STRUCT-001` backs is independently closed by this tree's own 2727/2727
exact-decode run in `verification.md`.
