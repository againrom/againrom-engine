# Verification — lighting the sprite layer

Windows 11, Go 1.26.1, no game install present. Every fixture is built in test
code; nothing below reads an asset. `-trimpath` throughout (Defender quarantines
one test binary without it).

## The gate

Run unpiped as one `&&` chain before every push, the last time at `43d32c3`:

```
go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))"

25 packages ok, 0 FAIL, 3 with no test files
check-no-game-assets: clean (tree scan)
check-sdd-audit: 139 trailered commit(s) in ac6bd87..HEAD checked; ok
gofmt: nothing
docs/0044-sprite-lighting/analysis.md      6559 /  7168  ok (91%)
docs/0044-sprite-lighting/provenance.md    5822 /  7168  ok (81%)
docs/0044-sprite-lighting/spec.md         12047 / 13312  ok (90%)
docs/0044-sprite-lighting/plan.md         11597 / 13312  ok (87%)
docs/0044-sprite-lighting/tasks.md T1..T4  1023/1008/864/952 of 1400 each  ok
0044: plan <= 1.2 x spec ok · tasks <= 1.2 x plan ok
```

311 test functions in the three packages this story touches, 17 of them new.

## Criteria

| Criterion | What ran | Result |
|---|---|---|
| **AC-1** | `TestSpriteRampIsFR1AtEveryRow` — 256-entry palette spanning the channel range, zero tint, all 16 rows, against an expression transcribed by hand | 12288 channel comparisons, 0 wrong; 8448 truncating divides and 2061 results clamped at 255, counted rather than assumed |
| **AC-2** | `TestLitBlitAtRowEightIsTheUnshadedBlit` — 0017's fixture frame over a background varying on both axes, whole 16x12 canvas compared | byte-identical; row 15 differs from raw, so the equality is not vacuous |
| **AC-3** | `TestSpriteRampEqualsTheTerrainLadderSampledEveryFourthRow` — exhaustive, terrain side read from the shipped `ShadeChannel` at level `4L+32` | all 4096 pairs agree |
| **AC-4** | `TestSpriteRowIsTheAmbientByteShiftedTwice` — the four suns, with θ, range and tint varied across them | 3, 0, 0, 15; the row is monotone over all 256 ambient bytes and never leaves `[0,15]` |
| **AC-5** | `TestLitBlitChangesColourAndNeverCoverage` — rows 0, 8, 15, every pixel of the destination | untouched set equals the unshaded blit's at every row; every painted pixel `A=0xff`; the alpha-0 entry paints its shaded colour |
| **AC-6** | `TestRGBALitIsTheLitBlitOntoATransparentCanvas` — one frame, three rows, a **non-zero tint** | identical `Pix`; every hole reaches the image as `{0,0,0,0}` |
| **AC-7** | `TestBothWindowPassesTakeOneRowAndOneRule` — a 10x6 object frame and a 7x11 unit frame over one palette, both drawn in one `Draw` | equal indices shade equal; every cache entry at row 3 |
| **AC-8** | `TestRenderStaticsTakeTheResolvedSunsRow` — four renders of the synthetic map through `run()` | see the finding below |
| **AC-9** | `TestWindowUnshadedToggleReturnsTheLitPixelsAndTheLitTexture` | lit, raw, lit again; third equal to first byte for byte, and the third draw hands back the **first draw's own texture object** |
| **AC-10** | `TestLitBlitRefusesExactlyWhatTheUnshadedOneRefuses` — six refusal cases x 3 rows, whole destination before/after | nothing panicked, nothing drew; rows −1 and 99 drew as 0 and 15 |

**P-1** is checked inside AC-5 and by `TestLitBlitLeavesTheFrameAlone` (16 rows,
frame unchanged). **P-2** is AC-10's whole-destination comparison. **P-3** is
structural: `spriteRow` returns a row or `spriteUnlit` and `spritePixels` has two
arms, witnessed by AC-2 and AC-8 together. **P-4** is AC-7's cache scan — every
entry at one row, both passes. All four are **sampled, not proved**, as the spec
says; AC-3 alone is exhaustive.

**SC-1** AC-1 and AC-3 hold in full, AC-3's terrain side read from the existing
transform. **SC-2** holds byte for byte over the whole canvas. **SC-3** holds on
all four suns including both clamped ends. **SC-4** holds, each refusal on a case
of its own. **SC-5** holds, the two frames differing in size. **SC-6** holds with
each render pinned at its own row — with the exception below. **SC-7** holds, the
third state compared with the first. **SC-8** is the four mutants below.

## Mutants

Each applied to production code, run over the whole tree, reverted, and the tree
confirmed byte-identical afterwards by a sha256 manifest of every tracked and
untracked file outside the submodule.

```
M1  DD-3  the ramp numerator's  * 2  dropped                         4 killed
      TestSpriteRampIsFR1AtEveryRow, ...DoesNotClampTheSumBeforeTheMultiply,
      TestSpriteRowEightReproducesTheRawPalette,
      TestSpriteRampEqualsTheTerrainLadderSampledEveryFourthRow
M2  DD-2  RGBALit calls BlitStatic (standalone image left unshaded)
      at T2, before any caller was converted                         1 killed
      re-run over the finished tree                                  4 killed
      TestRGBALitIsTheLitBlitOntoATransparentCanvas, and all three window tests
M3  DD-7  terraintool's row from DefaultDaytime, not the resolved sun 1 killed
      TestRenderStaticsTakeTheResolvedSunsRow (8 assertions)
M4  DD-5  the window's key narrowed back to the frame pointer         2 killed
      TestBothWindowPassesTakeOneRowAndOneRule,
      TestWindowUnshadedToggleReturnsTheLitPixelsAndTheLitTexture
M5  DD-6  spriteRow gated on !Lit() instead of v.unshaded             1 killed
      (not in SC-8; DD-6 is a substantive mechanism SC-8's four miss)
      TestWindowSpriteRowIsTheFixedSunsAndNothingElse
```

No survivors, declared or otherwise.

## Findings in the papers

**AC-8 contains one false clause, and it is refuted by FR-1 rather than by the
code.** AC-8 asks that "the two lit renders differ from it and from each other".
At `-ambient 0x20` the row is `32>>2 = 8`, whose gain is exactly 1.0, so that
render's sprite pixels **are** the raw palette and are byte-identical to the
unshaded render's — which is FR-1's own row-8 clause and the spec's own I/O table
(`-ambient 32 … gain 1.0000 (raw palette)`). The three renders AC-8 names are all
run and all **pinned at their own rows**, which is SC-6's requirement and the
load-bearing half; the equality at row 8 is asserted as the equality it is. A
fourth render at `-ambient 60` (row 15) was added to carry the clause's
discriminating content — a lit render differing both from raw and from the other
lit rows. The "differ from each other" half holds for all three.

**No acceptance criterion exercises a non-zero sky tint.** AC-1 and AC-3 both fix
it at zero and our sun's tint is `(0,0,0)` everywhere, so FR-1's "the sum `ch + t`
MUST NOT be clamped before the multiply" would have gone unwitnessed.
`TestSpriteRampDoesNotClampTheSumBeforeTheMultiply` witnesses it on the one shape
that discriminates — a gain below 1 with a sum above 255 (ch 200, tint 100, row
10: 225 against a pre-clamped 191) — and AC-6 carries a non-zero tint through
both raster paths.

**AC-8's fixture cannot exercise a truncating divide.** Every channel of the two
fixture sprite colours is a multiple of 16, so no row truncates there. The
truncating half of FR-1 is pinned one tier down (AC-1, 8448 cases) and is not
claimed by the tool tests.

**DD-3's last sentence is not satisfiable under T1's fences.** "The two share the
clamping integer helper and nothing else" would require editing `shade.go`, which
T1 forbids opening. `spriteshade.go` therefore shares *nothing* with the terrain
transform — strictly further from a shared derivation than DD-3 asks, so the
identity AC-3 checks stays two independent ladders.

**T4 changed two files outside its declared list.** Widening the cache key to
`(frame, row)` broke three assertions that indexed `staticImages` by a bare frame
pointer, in `entity_overlay_test.go` and `mirror_draw_test.go`. They now spell the
key through `v.spriteKey`; no test's subject moved. Keeping the key at the frame
pointer was the alternative, and DD-5 rejects it.

## What is not witnessed, and what would witness it

- **That the GPU texture holds the bytes of the image it was built from.** An
  `*ebiten.Image` cannot be read back before the game starts, so FR-5's window
  half is witnessed at the CPU image both front-ends' pixels come from
  (`RGBALit` = `BlitStaticLit` onto a transparent canvas, AC-6) plus the cache's
  own identity assertions. The upload itself is unwitnessed. What would witness
  it: a windowed harness with a real graphics context, drawing one frame and
  reading the framebuffer back.
- **That the shipped art looks right at row 3.** No test may read a game install,
  so nothing here is evidence about a real sheet. What would witness it: an
  owner-review render from a lawful install, which is the owner's seat.
- **The per-unit selector.** This story assigns it no meaning and no test asserts
  anything about it. Nothing here may be read as deciding it, and a story
  carrying lighting into hashed simulation state must re-open it.

## Conclusion

All ten acceptance criteria have executable evidence; AC-8 carries one clause
that FR-1 refutes, disclosed above and asserted as the equality it is. All four
derived properties are witnessed, three of them by sampling as the spec states.
All eight success criteria hold. Five mutants, five killed, no survivors.
