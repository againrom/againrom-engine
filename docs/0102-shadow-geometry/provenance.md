# 0102 — provenance

Research pin: `53f8bb7`, frozen at the story's first commit. Every fact below is read from the
submodule at that pin with `go run ./tools/claim <ID>`.

## Claims this story is built on

| Claim | Confidence | What it gives 0102 |
|---|---|---|
| `TERR-SHDW-129` | High | The blitter's own law: `X(r) = dstX + trunc(h*s/65536) - floor(r*s/65536)`, the pivot at the bottom edge of the blit rectangle, `h` and `w` taken from the **drawn** frame by the thunk, mirroring not flipping the lean, a positive shear putting the top to the right. Re-executed over 448 `(shear, height)` pairs with 0 rows disagreeing. |
| `TERR-SHDW-130` | High | The three casters' destinations term by term, and the identity of the unit's and the object's subtrahend: `T = ftol(tan(theta) * (2*floor(frameH/2) - anchorY))`, a **pixel** count. Also the structure's `dstX` with no `+16` and no anchor. |
| `TERR-SHDW-131` | High / **Unknown** on one point | That the corpus's `+` and `-` are one rule in two coordinate directions and neither may be corrected into the other; that substituting the unit rule into the blitter law gives `X(r) = base + tan(theta)*(anchorY - r)`, residue at most 1.99 px and not growing with `frameH`. Unknown: where `ShadowY` sits relative to the structure image's bottom, pending the strip frame height. |
| `TERR-SHDW-136` | High / Medium on (d)'s intent | (a) the structure carries both shear terms on the same call — a consumer implementing only the per-strip term "draws a stack of rectangles that step but do not lean". (b) `ShadowY` is read once and never tested; all fifteen compares in the routine enumerated. (c) the shipped range 28..55, 10000 on 4, 20000 on 11. (d) the dead band, its four constants, and the consequence that the smallest magnitude reaching the tangent is `0.0333`. |
| `TERR-STRUCT-103` | High | The structure's per-strip term and its `ShadowY` meaning; the two `VariableSize` bridge subclasses overriding the shadow with a bare `RET 0xc`. Amended by `TERR-SHDW-131` on the sign's reading, not on the expression. |
| `TERR-SPR-066` | High | That the blit's fifth argument is the 16.16 slope and not a brightness. Its `shear/2000` belongs to the unit's `vt+0x1c` **alternate** arm — a translated shadow — not to the `vt+0x2c` path this tree draws. |
| `TERR-SPR-067` | **PARTIAL retraction** | The unit body/shadow two-term difference stands; the *name* of the value loaded at `L07951` was wrong. It is `T`, not the 16.16 shear. This is the retraction that makes `0101`'s FR-4 wrong. |
| `TERR-SPR-043` | High / Medium on the sheet pairing | The object shadow pass pushes frame `0x0` to the two frame-size getters while the body pushes the drawn frame, and the eight shipped classes where that displaces one from the other. |

## What is ours by measurement, not by claim

- **Every structure strip frame is 32x32.** Read from the `graphics.res` of both lawful installs
  through this tree's own structure loader: 66 classes, 1300 non-nil frames, one height and one
  width in the histogram, no class mixing heights, identical on `en` and `ru`. This closes
  `TERR-SHDW-131`'s Unknown **for our consumers only** — it is a fact about the shipped sheets,
  not about the routine, and it is not offered to research as one.
- **The dead band's arithmetic.** `TERR-SHDW-136`(d) names four constants and one consequence but
  not the routine's shape. Two readings fit the words: the clamp applies to the angle *before* the
  `2/3` (giving `0.05*2/3 = 0.0333`), or the `+-0.05` is itself the returned value. **The claim's
  own derived figure does not separate them.** Swept over our `SunAngle`'s 1440 minutes, the
  smallest `ShadowY = 20000` displacement is 660.5 px under the first reading and 662.9 under the
  second — both the claim's "660" — because the angle steps by 0.0022 and therefore takes values
  just outside the band twice a day, where the second reading also multiplies by `2/3`. The 991 px
  the second reading gives applies only inside the band and is never the minimum. We implement the
  first on two grounds the claim does support: its wording is that `0.0333` **is** the smallest
  magnitude reaching `FPTAN`, which the first attains and the second only approaches; and the
  second is discontinuous, its output *falling* from `0.05` to `0.0333` as `|theta|` rises through
  the band's edge, which is not the shape of a clamp built from `0.0`, `-0.05`, `+0.05` and `2/3`.
  It is a reading, not a reading-off, and `spec.md` D-4 discloses it. The sign taken at exactly
  `theta == 0` is ours (positive) and moves one in-game minute.
- **Where the composed structure law pivots.** Substituting `appendStructureStrips`' own
  `dstY = row*32 - lift - originY - (rowTop-k)*32` into the two terms cancels `k`, leaving one
  world row `(anchorRow + TileHeight)*32 + 32 - ShadowY - lift - originY` for the whole footprint.
  That derivation is ours; only the two terms it composes are published.

## Open — a question for the pipeline, not filled in here

**Which frame the object shadow's anchor is built at.** `TERR-SPR-043` names the pushed constant
`0x0` at `L07845`/`L07846` (and `L07847`/`L07848`) as the frame argument to the two size
getters on the shadow passes, and derives a data-side consequence from it — eight shipped classes
whose drawn frame differs in size from frame 0. `TERR-SHDW-130` ends with "`frameW`/`frameH` come
through `vt+0x20`/`vt+0x24` at the drawn frame, not frame 0, on all three paths, which is
`TERR-SPR-043` holding for the shadow as well as the body" — but `TERR-SPR-043` says the opposite
of the shadow, so that sentence cites as agreement a row it contradicts. `TERR-SHDW-130`'s own
object citations (`L10785` for `T`, `L10275..L10276` for the destination) do not include
either getter address, so no instruction in it bears on the question.

We cannot settle it from what is published and we do not have the image. We keep `0101`'s frame-0
anchoring (`spec.md` FR-10, D-3), because the row whose subject *is* the frame index names the
addresses and the row that generalises does not. It affects 8 of 66 classes by at most 4 columns
and 8 rows.

## What 0101 recorded that this story removes

- `0101` D-1, "the sense of the lean is ours". `TERR-SHDW-129` settles the sense (positive shear
  puts the top to the right) and `TERR-SHDW-131` settles that the two published signs are both
  right. Nothing is authored here any more.
- `0101` D-2, "the shear angle's own routine was not read". `TERR-SHDW-136`(d) reads it: the `2/3`
  is a constant in the image and there is a dead band beside it.
- `0101` FR-4's `shear/2000`. Superseded by `TERR-SPR-067`'s partial retraction.
- `0101` FR-7's justification, "a strip's own `dstY` already carries the vertical placement an
  image row's `y` would otherwise supply". `dstY` places the strip; it says nothing about the 32
  rows inside it.
