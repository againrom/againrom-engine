# Analysis — `.256` sprite format decoder (ROM1)

## Source & confidence

All facts below are game-derived by the research team, not by this repo's own reverse-engineering:

- **Submodule pin:** `research/` at `72f9d47` (`againrom-research`, module `rom1research`).
- **Experiments:** `research/experiments/EXP-0002` (structure), `EXP-0003` (RLE decode), `EXP-0004`
  (palette), `EXP-0005` (overlay), `EXP-0013` (trailer / frame count / residual reads), `EXP-0016`
  (the `CSprite256` loader read directly from `rom.exe` — the RLE opcode switch, the Bucket-B inner
  block, and the has-palette flag), promoted into `research/formats/spr256/format.md`.
- **Corpus strength:** the RLE grammar decodes **23 791 / 23 791 frames exactly and losslessly**
  (per-row tokens sum to `width`, exact `height` rows, stream fully consumed — SPR256-RLE-008); standard
  structure holds for **1370 / 1384 non-empty** files (EXP-0002); `frameCount = trailer & 0x7FFFFFFF`
  matches the record walk 1370/1370 (SPR256-TRLR-016/021); palette byte order is pinned against the
  game's own VGA palette (SPR256-PAL-011); and the opcode set + trailer flag are confirmed against the
  **shipped loader itself** (EXP-0016). Everything the decoder needs — structure, RLE (incl. the `0xC0`
  alias), palette, trailer/count/has-palette flag, the overlay class, and the Bucket-B inner block
  (runtime-inert) — is derived from these experiments.

## Claim inventory (what the spec is built on)

| Claim | Statement | Confidence | Used by spec |
|---|---|---|---|
| SPR256-STRUCT-001 | Layout `[1024B palette][frames][4B trailer]`; frame = `[u32 w][u32 h][u32 dataSize][data]` | High | format definition, FR-1 |
| SPR256-COUNT-002 | Frame count = records before the trailer (walk-derived); **also** stored as `trailer & 0x7FFFFFFF` (refined by TRLR-016/021; the two agree) | High | frame-list walk + trailer count (FR-1) |
| SPR256-TRLR-016 | Trailer's high bit set (paletted) makes the first `u32` an impossible width = end-of-frames **sentinel**; refined by TRLR-021 (the high half is a 1-bit flag, not a `0x8000` word) | High | trailer, termination, FR-1 |
| SPR256-TRLR-021 | Trailer = `[31-bit frameCount][bit31 = has-palette]`; loader computes `frameCount = trailer & 0x7FFFFFFF` and reads the 1024-B palette **iff** `trailer & 0x80000000` (`rom.exe`, EXP-0016) | High | trailer, palette-presence (FR-1/FR-2); unifies the no-palette variant |
| SPR256-PAL-003 | Leading 1024 B = 256-entry palette × 4 bytes | High | palette region |
| SPR256-VAR-004 | No-palette variant (6 projectile arrows): frames from offset 0 = the same format with trailer bit 31 clear (TRLR-021). External borrowed palette source TBD | High (presence) / Medium (source) | FR-2 (presence read from bit 31; borrowed source is the consumer's) |
| SPR256-EXC-005 | 8 sprites: one frame + trailing non-record section (refined by EXC-017/020) | High | Bucket-B variant |
| SPR256-EXC-017 | Bucket-B section is bracketed `0x80000001 … 0x80000001` (8/8); inner block not a frame / not same-width RLE; two size regimes | Medium | Bucket-B variant; AC-10 |
| SPR256-EXC-020 | The `CSprite256` ctor indexes exactly `frameCount` (= 1 here) frames; the Bucket-B inner is loaded but **never indexed/decoded/blitted** — runtime-inert (`rom.exe`, EXP-0016) | High | Bucket-B runtime-inert; AC-10 |
| SPR256-CORPUS-006 | Corpus maxima: 23 791 frames; **max W = 640**, H = 480, ≤ 256 frames/sprite (**falsifies a 512 width cap**) | High | no null-frame width threshold |
| SPR256-RLE-007 | RLE `[2-bit op][6-bit count]`: `0x00` literal-N, `0x40` N blank rows, `0x80` N transparent px | High | RLE grammar, FR-1 |
| SPR256-RLE-008 | Decode exact/lossless: per-row tokens = `width`, exact `height` rows, stream fully consumed | High | RLE validation, FR-4, P-1/P-4 |
| SPR256-RLE-009 / -019 | `0xC0` quadrant never used in ROM1 data (0 / 23 895 frames, 3.67 M controls) | High | data never emits it |
| SPR256-RLE-020 | Loader dispatches `c & 0xC0` three-way `if(0x00)/elif(0x40)/else`; the `else` handles **both `0x80` and `0xC0`** ⇒ `0xC0` is a decode-time **alias of `0x80`**, no 4th opcode (`rom.exe`, EXP-0016) | High | alias `0xC0`→`0x80` (FR-4, AC-11) |
| SPR256-RLE-022 | Loader confirms EXP-0003 opcode labels byte-for-byte: `0x00` opaque run, `0x40` skip rows, `0x80` transparent skip; row ends at `col==width` | High | RLE grammar corroboration |
| SPR256-RLE-010 | Grammar corpus-universal | High | RLE grammar |
| SPR256-PAL-011 | Palette entry = `[B, G, R, reserved]` (**BGR**); order pinned to `cursors/default.256` VGA palette; 4th byte 0 in 99.96% of 350 720 entries | High | palette BGR→RGB (AC-2) |
| SPR256-PAL-012 | Palette index 0 = reserved transparent key — 0 literal occurrences in 9 541 025 literals | High | transparency model, AC-2 |
| SPR256-PAL-018 | Non-zero reserved 4th byte occurs **only** in `cursors/pickup.256` + `attack.256` (138/350 720); their 1024 B is not a standard palette | High | out-of-scope framing (not a standard palette) |
| SPR256-PAL-013 | 181 sprites carry a zeroed low palette (overlay decals + a 2-file anomaly) | Medium | out-of-scope framing |
| SPR256-OVL-014 | `spritesb.256` = base-palette overlay, 1 frame per base frame (180/180); blend rule open | High | out of scope (blend = engine) |
| SPR256-OVL-015 | `cursors/attack.256` + `pickup.256` decode to non-image bands; cause open | Low | out of scope (anomaly) |

## Derivation notes

Three non-obvious derivations from this research shape the decoder:

- **No null-frame size threshold.** A naive decoder might treat an over-large header (`width > 512`) as
  an invalid "null frame" to skip. The research records **real widths up to 640** — SPR256-CORPUS-006
  explicitly falsifies a 512 cap — so such a threshold would misclassify legitimate 513–640-wide frames,
  skip their RLE blocks, and desync the stream. **Consequence:** no width/size threshold at all.
- **The trailer is `[31-bit frameCount][bit31 = has-palette]`.** The loader takes
  `frameCount = trailer & 0x7FFFFFFF` and reads the leading 1024-byte palette **iff** `trailer &
  0x80000000` (SPR256-TRLR-021). For paletted sprites bit 31 is set, which also makes the first `u32` an
  impossible width — the end-of-frames sentinel a frame-walker stops at; the no-palette arrows are the
  same format with bit 31 clear. **Consequence:** the decoder reads palette *presence* from bit 31 (not
  a content heuristic), walks frames to the trailer, and cross-checks the count against
  `trailer & 0x7FFFFFFF`.
- **`0xC0` is a decode-time alias of `0x80`.** The loader's opcode switch is a three-way
  `if(0x00)/else if(0x40)/else`, and the `else` handles `0x80` and `0xC0` alike — a transparent skip
  (SPR256-RLE-020, verified in two structurally-opposite blitters + a 14-variant idiom census). ROM1
  data never emits `0xC0` (SPR256-RLE-009/019, 0 of 23 895 frames). **Consequence:** the decoder aliases
  `0xC0`→`0x80` to match `rom.exe` — a code-defined but data-unused path.

## Scope notes carried into the spec

The research establishes facts this story deliberately leaves to the engine, not the decoder: the
`spritesb.256` **overlay blend rule** (SPR256-OVL-014, open in research) and the attack/pickup banded
anomaly (SPR256-OVL-015 / PAL-018) — the decoder decodes both by the same grammar; interpreting/
compositing the result is a consumer concern. The 181 zeroed-low-palette sprites (SPR256-PAL-013) and
the no-palette variant's borrowed palette source (SPR256-VAR-004) are external-palette questions, not
this format. The Bucket-B inner block is loaded but runtime-inert (SPR256-EXC-020); its authoring origin
is non-runtime and out of scope. None of these block the decoder — every fact it needs is derived from
the experiments above.
