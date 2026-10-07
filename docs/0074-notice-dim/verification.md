# Verification — the dim behind a notice is the one the original applies

Worktree `wt-0074`, branch `impl/0074-notice-dim`, from master `b86f234`. Research submodule at
`53d7679`, `git submodule status` showing no leading character. Go toolchain as pinned in `go.mod`.

## The change

```
$ git diff --stat b86f234 HEAD
 docs/0074-notice-dim/analysis.md   |  79 ++++++++++++++++++++++
 docs/0074-notice-dim/plan.md       | 133 +++++++++++++++++++++++++++++++++++++
 docs/0074-notice-dim/provenance.md |  49 ++++++++++++++
 docs/0074-notice-dim/spec.md       | 126 +++++++++++++++++++++++++++++++++++
 docs/0074-notice-dim/tasks.md      |  44 ++++++++++++
 pkg/ui/notice.go                   |  49 ++++++++++----
 pkg/ui/noticedim_test.go           | 111 +++++++++++++++++++++++++++++++
 7 files changed, 579 insertions(+), 12 deletions(-)

$ git diff --diff-filter=D --name-only b86f234 HEAD
                                      (empty — nothing was deleted)

$ git diff --name-only b86f234 HEAD | grep -E 'pkg/sim|form|version'
                                      (empty — P-2)

$ git log --format='%h %s / %(trailers:key=SDD-Task,valueonly)' b86f234..HEAD
a53d9c9 0074 plan: SC-5 is the look, and R-2 does not answer itself /
ce252bb 0074: the dim's strength / 0074-notice-dim/T1
15adb52 0074: spec, plan and tasks for the dim behind a notice /
```

One trailered commit for one task; the other two are the artifact stages and carry none.

## The gate

```
$ go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go')
EXIT=0                                (no output from any of the three)

$ go test -count=1 -trimpath ./...
EXIT=0                                33 packages: 31 ok, 2 with no test files, 0 FAIL

$ bash scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)                                       EXIT=0

$ bash scripts/check-doc-budget.sh
docs/0074-notice-dim/analysis.md                 4713 /   7168 bytes  ok (65%)
docs/0074-notice-dim/provenance.md               5458 /  16384 bytes  ok (33%)
docs/0074-notice-dim/spec.md                     9198 /  13312 bytes  ok (69%)
docs/0074-notice-dim/plan.md                     8591 /  13312 bytes  ok (64%)
docs/0074-notice-dim/tasks.md T1                 1000 /   1400 bytes  ok (71%)
docs/0074-notice-dim/tasks.md (legend+traceability)    809 /   1200 bytes  ok (67%)
0074-notice-dim: plan <= 1.2 x spec              8591 <=  11037 bytes  ok
0074-notice-dim: tasks <= 1.2 x plan             1809 <=  10309 bytes  ok
                                                                              EXIT=0

$ bash scripts/check-sdd-audit.sh
check-sdd-audit: 292 trailered commit(s) in ac6bd87..HEAD checked
check-sdd-audit: ok (35 note(s)/warning(s), none enforced)                    EXIT=0
```

The audit's 35 notes are the pre-existing set carried by stories 0010–0055; none is 0074's and none
is enforced. Every exit code above was branched on, not printed.

## AC-1, AC-4, P-1 — the value is black, and neither degenerate strength

```
--- PASS: TestTheDimIsBlackAndNeitherDegenerateStrength (0.00s)
```

Black is what P-1 rests on and it is asserted at the source rather than sampled over pictures: the
composite is source-over, so a non-zero colour channel would ADD light on that channel for every
destination. Mutated `G: 0x00` → `G: 0x01`:

```
--- FAIL: TestTheDimIsBlackAndNeitherDegenerateStrength (0.00s)
    noticedim_test.go:54: the dim is {R:0 G:1 B:0}, not black — compositing it would ADD light on a
    channel and shift the map's hue, which is not a per-channel gain
```

The same test kills the two degenerate alphas, 0x00 and 0xff.

## AC-2 — the shipped alpha is the nearest one that spells the gain

```
--- PASS: TestTheDimIsTheNearestGainAnAlphaCanSpell (0.00s)
    noticedim_test.go:87: gain wanted 0.8125000; nearest 8-bit alpha 0x30 spells 207/255 = 0.8117647
    (residual -0.0007353)
```

The assertion is a search over all 256 alphas, not a comparison with a literal, so it discriminates
in both directions. **Both immediate neighbours are killed**, which is the strongest form this
criterion can take:

```
mutant A=0x80 (the value 0073 shipped)
    the dim ships alpha 0x80 (gain 0.498039, off by 0.314461), but 0x30 spells gain 0.811765
    and is closer to 0.812500
mutant A=0x2f
    the dim ships alpha 0x2f (gain 0.815686, off by 0.003186), but 0x30 spells gain 0.811765
    and is closer to 0.812500
mutant A=0x31
    the dim ships alpha 0x31 (gain 0.807843, off by 0.004657), but 0x30 spells gain 0.811765
    and is closer to 0.812500
```

## AC-3 — the residual is bounded

```
--- PASS: TestTheDimsResidualIsUnderAQuarterOfOneLevel (0.00s)
    noticedim_test.go:110: worst per-channel deviation over [0,255]: 0.1875 of one level (bound 0.25)
```

0.1875 is the largest amount by which any channel can come out wrong — `255 × 13/16 = 207.1875`
against `255 × 207/255 = 207.0` — so the divergence is under one rounding step everywhere in the
range. The same three mutants fail this criterion too, at 80.1875, 0.8125 and 1.1875 of a level.

## AC-5, AC-6 — nothing else about the dim moved

The previous story's whole backdrop suite, unchanged and re-run:

```
--- PASS: TestAnOpenNoticeDimsTheWholeView                       (rectangle, and the shipped colour)
--- PASS: TestBothNoticeKindsDimIdentically                      (not a function of the kind)
--- PASS: TestNoDimWithoutANoticeOrWithATransparentValue
    --- PASS: .../no_notice_open
    --- PASS: .../a_notice_opened_and_then_closed
    --- PASS: .../a_fully_transparent_value,_with_a_notice_open
--- PASS: TestTheDimIsTheValueItWasGiven                          (the seam still substitutes)
--- PASS: TestAViewerThatCannotDrawANoticeDimsNothing             (no font, four pushes)
--- PASS: TestNoDimWithoutADrawableArea                           (no Layout yet)
--- PASS: TestTheDimIsComposedAfterTheMapAndBeforeEveryBox        (the parse of Draw)
```

The last of these is the one that matters for AC-5's "same place among the composed statements": it
parses `Viewer.Draw` and requires the dim to stand after `overlayPasses` and `pathScreenSegments` and
before `panelPresent`, `readoutPresent` and `noticePresent`. It passes untouched, and no statement of
`Draw` was edited by this story.

`TestAnOpenNoticeDimsTheWholeView` compares the frame's answer against `AuthoredNoticeBackdrop()`
rather than a literal, which is why it followed the value instead of breaking on it — that is the
seam working, not a gap in the assertion; AC-2 is what pins the value itself.

## P-2 — nothing reached simulation state

The two `grep`s in *The change* are the whole of it: no file under `pkg/sim` is in the diff, no
byte-form record and no serialized version. The shipped form version is untouched — this story took
no number from that namespace.

## Success criteria

| | |
|---|---|
| SC-1 | met — AC-1 and AC-4, with the colour mutation as the discriminator |
| SC-2 | met — AC-2 and AC-3, with both neighbouring alphas killed |
| SC-3 | met — AC-5 and AC-6, the previous story's suite re-run unmodified |
| SC-4 | met — the gate block above, plus the empty deletion set and the two `grep`s |
| SC-5 | **not run here** — see *Limitations* |

## Limitations, and what is not claimed

**SC-5 is the owner's to run, and it is the half of this story arithmetic cannot reach.** The dim is
now much weaker than it was — 81% of the map's brightness against 50% — and whether the world still
reads as stopped behind the box is a judgement needing a lawful install and an eye. No runnable build
was produced for this story. Until that look happens, what is claimed is that the strength is the
measured one and that changing it costs one value; it is **not** claimed that it looks right.

**Nothing here observes a drawn pixel.** An `*ebiten.Image`'s pixels cannot be read back before the
game starts, so every assertion above is about the value and about the arithmetic the compositing
tier is documented to apply to it. That the tier applies it was established by reading it —
`vector/util.go:105` and `:80`, `colorscale.go:110`, and the default source-over blend — not by
measuring it. **This is the weakest link in this story's evidence**: if `FillRect` ever stopped
scaling a white sub-image by the colour, or the default blend changed, nothing here would fail and
the dim would silently stop being a gain.

**The factor is reproduced and the pixels are not.** The original scales the 5- and 6-bit channels of
a 16-bit surface and truncates; this tree scales 8 bits and rounds once. At 5 bits the original's own
full level lands at 0.8065 of full against our 0.8118, and its lowest non-black level lands on black.
That gap is several times the alpha's 0.0007 and belongs to the render tier's colour depth, not to
this value — which is also why exactness in the alpha was not bought.

**Ours is per frame and the original's is once.** Measured nowhere, because there is nothing to
measure: the difference is in mechanism and the appearance is one application of the factor either
way. What could reveal it is a surface that changes behind an open box, and this tree has one — the
water, which the suspension does not stop. That is the previous story's disclosed divergence and it
is untouched here.

**The whole-screen extent is a disagreement left open, not closed.** The original darkens everything
already on its screen, including its own information panel; ours exempts the unit information panel,
the debug readout and the notice. The notice agrees with the original and the readout has no
counterpart there; the panel is a real divergence. Closing it is one statement's position inside
`Viewer.Draw` and a change to the previous story's FR-7, and it needs the product author, because the
ruling the behaviour rests on reaches whether there is a dim and not what it covers.

## Conclusion

Every automatable criterion is met with the mutants that discriminate it recorded, the gate is clean
on all five checks with the exit codes branched on, the deletion set is empty and nothing reached
simulation state. The one criterion not met is not met because nothing in this seat can run it, and
it is named rather than absorbed.
