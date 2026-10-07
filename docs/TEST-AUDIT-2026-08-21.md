# Test audit, 2026-08-21

Owner instruction, 2026-08-21: audit the test suite, because many tests look meaningless and the
most important end-to-end tests are the question.

Three independent agents audited three disjoint scopes at commit `16177e9`, each in its own
worktree, each with no knowledge of the others' findings. Scope 1: `pkg/sim`, `pkg/mapload`,
`pkg/mapedit`, `pkg/formats/*`, `pkg/vfs`, `pkg/data`, 1,902 test functions. Scope 2: `pkg/game`,
`cmd/*`, `internal/`, 1,153 test functions. Scope 3: `pkg/ui`, `pkg/render/*`, `pkg/audio`, 1,197
test functions. Total population 4,252.

Each used two instruments: a Go AST census over the whole population, and a mutation campaign of
production lines selected before any test was read. All three finished with an empty
`git status --porcelain` and every mutation reverted.

## The headline result

89 production lines were mutated across the three scopes. 41 mutations survived the full test suite.

| Scope | Mutations | Survived | Rate |
|---|---|---|---|
| `pkg/sim` alone | 16 | 2 | 13% |
| Scope 1 total (sim, mapload, mapedit, formats, vfs, data) | 29 | 5 | 17% |
| `pkg/render/*` | 10 | 2 | 20% |
| `pkg/audio` | 6 | 3 | 50% |
| Scope 2 (game, cmd, internal) | 27 | 16 | 59% |
| `pkg/ui` | 17 | 15 | 88% |
| **All three scopes** | **89** | **41** | **46%** |

The spread is the finding. `pkg/sim` is the determinism wall and it kills 87% of introduced defects.
`pkg/ui` has a comparable number of tests per composer and lets 88% through. The difference is not
test count. It is what the tests assert once they run: `pkg/render`'s tests read pixel content at
specific coordinates against independently derived values; `pkg/ui`'s more often check a total, a
presence, or one representative coordinate that a one-pixel shift does not move.

## The owner's premise, measured

**"Many tests look meaningless."** The naive forms of meaningless are rare. Across 4,252 functions,
manual review of every candidate the three censuses flagged confirmed roughly 15 tests that restate a
constant or compare a function to itself: 3 of 1,197 in scope 3, 8 of 1,153 in scope 2, and a small
number in scope 1. That is under 0.4% of the population, and every one of the three auditors reported
that their own census tool overstated its category by a large factor before manual reading corrected
it.

**The premise is right about the outcome and wrong about the mechanism.** The tests are not
tautological. They are blind: they call production, assert something true, and fail to distinguish
the correct value from a wrong one. A census that looks for tautologies finds almost nothing here. A
mutation campaign finds that 46% of introduced defects ship.

## Two findings re-verified at this seat

Both were re-run at master `fad2830`, not at the audit's own base, and against `go test ./...` over
the whole tree rather than one package.

**`data.SkillCap`, `pkg/data/recompute.go:34`.** Changed from 100 to 101. All 39 packages pass. The
covering assertion, `pkg/data/recompute_test.go:326`, reads
`if got := h.Recompute(...).Skill[data.SkillBlade]; got != data.SkillCap`. Production clamps to
`SkillCap` and the test compares the result to `SkillCap`, so the assertion holds for every value the
constant could take. The floor assertion four lines below has the same shape against `SkillFloor`.
This is the incident `docs/HARNESS-CASEBOOK.md` records for the Heal hotfix, live in another package.

**The draw filter, `pkg/ui/app.go:2605`.** Changed `op.Filter = ebiten.FilterNearest` to
`ebiten.FilterLinear`. All 39 packages pass. Three sibling call sites were mutated the same way by
the scope-3 audit (`viewer.go:2686`, `statics.go:380`, `healstars.go:55`) and all three survived. The
last of those is the sharper case: `pkg/ui/static_draw_test.go` has a fake that records every
`op.Filter` a draw call used and asserts `FilterNearest`, and the mutation still survived, so the
fixture does not drive execution through that one of the file's three assignment sites. A check built
for exactly this defect class exists and does not reach the line.

## Three classes found independently by more than one audit

**Self-referential constant comparison.** The test compares production's output against the same
production constant production computed it from. Confirmed instances: `data.SkillCap` and
`data.SkillFloor` (`pkg/data`), `shopMarkColor` (`pkg/ui/shopscreen_test.go:476`), and the diplomacy
offset `sessDiplomacy` (`pkg/formats/sav`, where `TestSessionAccessors` writes and reads through the
same offset constant so an off-by-one moves both together). Three auditors, three packages, one
shape. The fix is one line each: put the literal on the right-hand side.

**Clamp boundaries with no test at the boundary value.** A test exercises the interior and the far
side of a clamp and never constructs the exact boundary, so `<` and `<=` are indistinguishable.
Confirmed at `pkg/ui/shopscreen.go:786`, `pkg/render/terrain/overlay.go:557` and `:560`,
`pkg/audio/mix.go:144` and `:162`, `pkg/game/shopview.go:131` (where the mutated line would read
`party[i]` at `i == len(party)`), and `pkg/game/spellbolt.go:433`.

**Witnesses that `go test ./...` cannot see.** 35 install-gated tests and 13 JSON scenarios run only
from `pipeline/check-release-tests.sh` and `pipeline/check-scenarios.sh`.
`pkg/game/headlesspointer.go` — the pointer dispatch and `assert_inventory`/`assert_shop` machinery,
nine functions — measures **0.0% coverage** under `go test -coverpkg=./pkg/game/... ./...` over the
whole module, because its only execution path is two scenario files run through `go run`. The same
holds for `pkg/game/scenario.go`'s `NewPlayWorld` and `StartScenarioMission`, and for
`cmd/savecheck`, the sole caller of `resume.go`'s 13-function `Live*` API.

## The end-to-end question

They exist, and they are deep. `scenarios/1005-doll-and-shop.json` is 90 steps: it loads an original
save, navigates notices into town, trades a shipped weapon through the production shop-room hit test
with real pointer gestures, advances into the next mission, drives the inventory doll for two party
members through equip, unequip and ground-drop, saves, reloads, and re-verifies that the NPC's
equipment survived the round trip. It asserts against production data structures through the same
dispatch the real UI uses, not a shortcut API.

The problem is not that they are missing. It is that they are structurally invisible: not in
`go test ./...`, not visible to Go coverage tooling, and gated only when a person or a seat script
remembers to run them. Golden rule 2 is why, and it is correct and stays. What follows from it is
that the gated layer needs its own accounting, which is story `1024`'s `B3`.

In `pkg/sim`, 286 of 1,098 test functions advance a `World` through more than one tick, and 75 do so
three or more times. 609 never call `Step` or `Run` at all. Many of those 609 are byte-form
round-trips and cost formulas that legitimately need no tick. Which of them test behaviour that only
manifests across ticks was not classified.

## What this measures for story `1024`

`1024`'s contract states its bar as: shifting a screen's destination rectangle by one pixel must fail
at least one test. That bar is now measured rather than assumed. 13 destination-rect and canvas-size
mutations of exactly that shape were run; 6 survived, and **all 6 are in `pkg/ui`** — the seven kills
are `pkg/ui`'s two canvas-size cases and every `pkg/render` subpackage sampled. The surviving six are
`chargen_page.go:90` and `:672`, `shopscreen.go:125`, `tippanel.go:417`, and `townshell.go:257` and
`:705`.

One of the six has a named cause worth carrying into the story.
`TestPreCreateControlBoundsAndNativePixels` enumerates `chargenChoice0`..`chargenChoice3` in
`preControlRegion`'s switch and never calls it for `chargenName`, which is the case the mutation
moved. The test is not weak; its enumeration is incomplete. That is the coverage rule this repository
already states as "enumerate producers, not convenient data rows".

## Work list

Nothing here is a story yet. Ordered by consequence.

1. **The self-referential comparisons**, three named instances and a sweep for the class. One-line
   edits. `pkg/data/recompute_test.go:326` and its floor sibling, `pkg/ui/shopscreen_test.go:476`,
   `pkg/formats/sav`'s session accessors.
2. **`pkg/audio`'s channel identity.** Swapping which byte offset receives the left and the right
   sample in `Stereo()` (`pkg/audio/mix.go:114`) survives, because all three `Stereo` tests construct
   `Placement{Left: GainUnit, Right: GainUnit}` and a symmetric placement produces a byte-identical
   buffer under a swap. The gain computation is independently witnessed; which physical output byte
   carries which channel is not. This is an audible defect with no witness.
3. **The draw-filter class**, four call sites, none witnessed, one of them beside a test built for
   exactly this.
4. **The surviving destination rectangles**, which belong to story `1024`'s `B2` and are its
   measured starting set. Six survived here, at `16177e9`. All six were re-run at master
   `4de5d8b` and five still survive: hotfix `23bb8e6` added `pkg/ui/tiprectorigin_test.go`,
   which pins `shopTipRect`'s origin and width, so that one is now killed. The current five,
   with locations re-read at `4de5d8b`, are in `docs/1024-screen-harness/contract.md`.
5. **The clamp boundaries**, seven named sites across four packages.
6. **`pkg/game/headlesspointer.go`'s 0.0%**, which is not a missing test but a missing accounting,
   and is story `1024`'s `B3`.

## What the method could not see

- 89 mutations against a production population in the tens of thousands of lines. The survivor rates
  are informative about the presence of gaps, not a bound on their number.
- The census tools are syntax-only, without type resolution. Method calls are matched by name across
  receivers. Two of the three auditors reported confirmed misclassifications from this: closures
  stored in table entries, `reflect`-based schema tests, and package-level constant references all
  register as "calls no production code" when they do the opposite.
- Scope 3's constant-restatement detector was scoped to geometry constants, so `shopMarkColor` — a
  colour constant with the same defect — was invisible to it and was found only by mutation. Scope
  1's largest census bucket, 1,188 rows, was not individually reviewed, and the confirmed `SkillCap`
  tautology lives inside it.
- No auditor classified the "state not installed" category mechanically. It needs to know what the
  production line reads, which a syntax scan cannot answer.
- Install-gated tests were exercised on the `en` root only in scopes 2 and 3.

## Progress against the work list, 2026-08-21 (seat)

The work list above is left as published. This section records what has been acted on since, so a
reader does not have to re-derive it from git.

**Item 2, the audio channel identity: CLOSED** (`c240771`). Verified first rather than taken on
report, by mutation: swapping which byte offset in a frame receives the left and the right sample
passed **all 41 packages**, not only `pkg/audio`, so the gap was wider than the audit scoped it.
`pkg/audio/mix_test.go` now places a sound asymmetrically, `Left` at `GainUnit` and `Right` at
`GainUnit/4`, and pins each frame's two words to exact values. The wanted values are derived from
the two scaling functions' definitions rather than recomputed with them, which would have been item
1's defect. Mutation-proved: the swap now fails on all four words and names the channel in each
message, and the production file reverts byte-identical.

**Item 1, the self-referential comparisons: TWO OF THREE CLOSED** (`4b6dab0`). Both were measured
before being changed. `SkillCap` 100 to 50 passed all 41 packages, and that bound is decoded, so the
skill ceiling could move with nothing red. `shopMarkColor` replaced outright passed all 41 packages,
in a file whose own header states that every number asserted in it is the decoded one spelt out
rather than read back. Both now assert literals and additionally check the constant against the
literal, so a failure names the constant rather than only the clamped value or the pixel. Both carry
a comment stating the literal is deliberate, because tidying it back into the constant is what would
silently restore the defect.

**Item 1's third instance, `pkg/formats/sav`: CLOSED** (`54916e7`), which closes item 1 entirely.
`TestSessionAccessors` is a write-then-read round trip: `SetTriggerLatch` then `TriggerLatch`,
`SetDiplomacy` then `Diplomacy`. Both sides resolve their position through the same offset constant,
so an off-by-one moves the write and the read together and the round trip still agrees with itself.
That is a different shape from the two above and it needed a second path rather than a one-line edit.

`TestSessionAccessorsLandAtTheOffsetsTheFormatSpecifies` writes through the setters with distinct
values and then reads `f.Body` directly at offsets spelt out as literals: trigger results at 0 as
signed dwords, latches at 400 as bytes, the 50x50 diplomacy matrix at 1856 as bytes, the two counters
at 4362 and 4370 as dwords. Distinct values mean a wrong offset landing on another written field is
still a failure. It also checks all eight session constants against their literals. Mutation-proved
with `sessDiplomacy` moved 1856 to 1857: the original round-trip test alone exits 0, the new test
alone exits 1 and names both the byte position and the constant. `world.go` reverts byte-identical.

**Items 4 and 6 belonged to story `1024`**, which landed as `645e8ee`. Item 4's surviving
destination rectangles are now `DIV-208`: `cmd/screencensus` reports seven registered geometry
tests, of which four do not fail when the rectangle is shifted at its use site, and `AGENTS.md` rule
6 was reworded to make the use-site mutation the bar. Item 6's accounting is the census's own gated
section, which measures its population by scanning for `AGAINROM_`-gated skips rather than
remembering a number; it selected 35, and `pipeline/check-release-tests.sh` independently selected
35 by running the suite with the variables unset.

**Items 3 and 5 remain**, and both are lane-sized rather than seat-sized: item 3 is four draw-filter
call sites, item 5 is seven clamp boundaries across `pkg/ui`, `pkg/render/terrain`, `pkg/audio` and
`pkg/game`. Each clamp needs a test constructing the exact boundary value, which is what makes `<`
and `<=` distinguishable.
