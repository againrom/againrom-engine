# 0140 — verification

Taken on the tree at `05a8196` plus this stage's own two tests, on a clean working tree.
Every command below was run before it was written down.

## SC-1 — the gate

```
go build ./...                             ok
go vet ./...                               ok
gofmt -l $(git ls-files '*.go')            0 files
go test -count=1 -trimpath ./...           EXIT=0, 34 packages ok, no FAIL, no panic
sh scripts/check-no-game-assets.sh         check-no-game-assets: clean (tree scan)
sh scripts/check-doc-budget.sh             EXIT=0 (0140 at 81% spec, 73% plan, 92% tasks overhead)
sh scripts/check-sdd-audit.sh              EXIT=0, 0 FAIL
sh scripts/check-hotfix-ledger.sh          check-hotfix-ledger: ok
git submodule status research              7eed909d01bc… research (heads/master), no leading character
```

`go test` is run with `-trimpath` on this machine because Windows Defender quarantines one test
binary otherwise; the flag changes nothing this story asserts.

## SC-2 — both lawful roots, headless

```
againrom.exe -check -assets <ru>
againrom: 62 map rows, 8 of 8 buttons have a mask region; hero Body 43, Reaction 26, Mind 15,
Spirit 15, Blade 10, Iron Short Sword 10-16, to-hit 49, defence 8

againrom.exe -check -assets <en>
againrom: 66 map rows, 8 of 8 buttons have a mask region; hero Body 43, Reaction 26, Mind 15,
Spirit 15, Blade 10, Iron Short Sword 10-16, to-hit 49, defence 8
```

The two differ in **map rows and nothing else**: 66 against 62, because the two installs carry
different loose maps. Every other field matches character for character.

## AC-4 — the retired flag, on a real install

```
againrom.exe -check -mission 10 -assets <ru>
againrom: 62 map rows, 8 of 8 buttons have a mask region; hero Body 43, ...
againrom: mission 10 at scenario/10.alm, 80x80, 36 entities, party at (17, 66), 11 raise(s);
          health 145/145, mana 0/0
againrom: winning mission 10 opens mission 20 at scenario/20.alm, 144x144, 57 entities;
          hero health 145/145, mana 0/0, skill xp [0 1593 0 0 0 0]
againrom: chargen offers Sex, Class, Skill, Body, Reaction, Mind, Spirit; budget 140

againrom.exe -check -mission 10 -chargen -assets <ru>
… byte for byte the same four lines, EXIT=0 both times
```

The generation line is reported **with the flag and without it**, which is the whole of "accepted
and inert". The refusal half is `cmd.TestChargenGeneration/a_windowed_-mission_naming_no_mission_is
_refused_before_any_window`: `-mission -3` exits 2, prints nothing to standard output, and says
`names no mission` on standard error.

**FR-2a's second clause was found here and is a real limit, not a passing note.** `-mission 999`
against the `ru` root does **not** refuse: `game.MissionMap` decides only that a number is positive
and composes an address for every other, so a positive number naming a map the install does not
carry opens the generation screen and fails when that map is loaded. Measured twice — the process
was still running at a six-second timeout with a window open, and a run left to finish exited **0
having printed nothing to either stream**. That second measurement is the sharper one: the player is
told nothing at all, and learns the number was wrong only after spending a spread and confirming it.
The code says so at its own site (`cmd/againrom/main.go`, "only the half that needs the map itself
moved onto the screen"); the contract did not, and FR-2a was written at this stage to say it.

## SC-3 — where every box stands

From a throwaway probe over the placement functions, at three window sizes, against a unit panel
390 pixels tall, which is what a full party hero's sheet composes to:

```
2560x1440  minimap (2248,12)-(2548,312)  | toggles (2248,324)-(2548,366)  |
           worn (2248,586)-(2548,758)    | doll (2248,766)-(2548,1026)    |
           panel (2248,1038)-(2548,1428) | book (12,1285)-(2236,1362) x63 |
           pack (12,1370)-(2236,1428) x42

1920x1080  minimap (1608,12)-(1908,312)  | toggles (1608,324)-(1908,366)  |
           worn REFUSED                  | doll (1608,406)-(1908,666)     |
           panel (1608,678)-(1908,1068)  | book (12,925)-(1596,1002) x45  |
           pack (12,1010)-(1596,1068) x29

1280x960   minimap (968,12)-(1268,312)   | toggles (968,324)-(1268,366)   |
           worn REFUSED                  | doll REFUSED                   |
           panel (968,558)-(1268,948)    | book (12,805)-(956,882) x26    |
           pack (12,890)-(956,948) x17
```

Three things this table is evidence for. The **minimap and the control panel are the same rectangle
relative to the window's top-right corner at every size** — FR-10c and D-15's reservation, visible
as an invariant rather than asserted. What **refuses** as the window shortens is the doll and the
worn set, never the control panel, and the worn set goes first because it is the higher of the two.
And the two bars widen as the column narrowed: 42 pack cells and 63 book columns at 2560x1440.

## SC-4 — the size that broke once

The whole suite passes at the harness's own 640x480 window, which is the size at which an oversized
minimap once produced twenty-eight failures across nine files. The fixture that caught it is now
explicit: `newPopupFix` refuses to run if either of its two units sits under **any** of the five
boxes that swallow presses, and that check fired during this story — unit A had drifted under the
control panel and was moved.

## The witness table

Every criterion and the test that carries it. Two rows were written at this evidence stage,
because writing the contract showed nothing was watching them.

| Criterion | Test |
|---|---|
| AC-1 | `ui.TestChargenGate/a claimed row arms generation and never reaches the loader`, `/an unclaimed row loads and shows its map exactly as it always has`; `game.TestNewGameChargenClaimsExactlyTheMissionRows` |
| AC-2 | `game.TestCampaignAdvanceNeverArmsGeneration` — installs a gate claiming **every** row, then wins a mission |
| AC-3 | `ui.TestChargenGate/Escape from a gated row returns to the picker, with the row still selected`; `ui.TestChargenAppDispatch` (Escape at startup leaves `ScreenMenu`) |
| AC-4 | `cmd.TestChargenGeneration` (5 subtests), plus the developer run above |
| AC-5 | `game.TestChargenDerivedShowsSightAndTheMagicResistances`; `game.TestChargenDerivedShowsEverySkillSlot` (3 subtests) — the one nonzero slot asserted against the derived record, not assumed |
| AC-6 | `game.TestChargenDerivedShowsSightAndTheMagicResistances` — the same row read at Spirit 16 and at Spirit 40 must differ |
| AC-7 | `ui.TestChargenBlockIsClippedToWhatFits`; `ui.TestChargenDerivedFitsTheScreensLineBudget`; `game.TestChargenDerivedFitIsMeasuredAgainstTheLayout` |
| AC-8 | `ui.TestAPanelStatesACharacterInTheOwnersOrder`; `ui.TestPanelManaRowFollowsTheHealthRowsForm/a subject with no pool omits the pair AND its heading` |
| AC-9 | `ui.TestTheUnitPanelSwallowsItsOwnPresses` — **written at this stage** |
| AC-10 | `ui.TestMinimapBoxIsSquareAndDoesNotMoveWithTheSelection` |
| AC-11 | `ui.TestMinimapConsumesTheClickAndCentresTheView`; `ui.TestMinimapCellAtIsTheScaleRuleRunBackwards` — every terrain pixel swept in both the scaled and the sampled branch |
| AC-12 | `ui.TestThePackBarsHitTestNamesTheElementUnderTheCursor`; `ui.TestThePackBarScrolls`; `ui.TestAPressOnAScrollButtonScrollsAndCountsNoClick` |
| AC-13 | `ui.TestHudSwitches/the doll follows any selected unit`; `ui.TestDollPresentRebuildsOnTheSourceAndNotTheSubject` |
| AC-14 | `ui.TestEverySwitchReachesItsOwnBoxAndNoOther` |
| AC-15 | `ui.TestHudTogglePanelTakesItsOwnPresses` |
| AC-16 | `ui.TestHudSwitches/cancel does not touch a switch` |
| AC-17 | `ui.TestStartupWindowCoversTheScreen` (6 cases, including a nonsense report and no monitor at all) |
| AC-18 | the developer run below |
| FR-6 | `ui.TestComposeMinimapOutlinesTheCameraView` — **written at this stage** |
| P-2 | `ui.TestRenderDoll`, `ui.TestRenderWorn`, `ui.TestATransparentPictureLeavesTheGroundBeneathItOpaque`, `ui.TestPresentAnswersFalseWithNothingToDraw` |
| P-3 | `archtest`'s allow-map and `TestUITierAllowanceIsPinned`, run on every `go test ./...` |
| P-4 | `ui.TestMinimapBoxIsSquareAndDoesNotMoveWithTheSelection`, and SC-3's invariant above |
| P-5 | `ui.TestInventoryTogglingIsInert` (selection, orders, blows, attacks and tick count all unmoved over five switching frames); `ui.TestToggleHudPanelWritesOnlyItsOwnFlag` |

### Proven by reversion

The first row was proven when the defect it covers was fixed; the other two at this stage.

| Reverted | What reddens |
|---|---|
| the `hudShown(hudPanelBook)` half of `spellbookBar`'s gate | `TestEverySwitchReachesItsOwnBoxAndNoOther`: `with "S" switched off, "S" is drawn=true, want false` |
| `panelCaptures`'s branch in `command` | `TestTheUnitPanelSwallowsItsOwnPresses`: a right press inside `(968,899)-(1268,948)` issued 1 order; the tap emptied the selection; and the control press outside the box issued 0 |
| the `drawMinimapView` call in `composeMinimap` | `TestComposeMinimapOutlinesTheCameraView`: six named edge pixels read the terrain colour instead of the view colour, and the one-cell sampled view drew nothing |

## AC-18 — the shipped binary

Built at `builds/0140-ui-improvements/againrom.exe` from this tree with `-trimpath`, run against the
`ru` root for six seconds and still up when the timer fired. The window is `2560x1440` at position
`0,0` with `decorated=false`, and `Layout` is handed those numbers — measured earlier in this story
with a windowed probe that replicated `App.Run`, on a monitor Ebitengine reports as
`ViewSonic VA3209-QHD 2560x1440`, scale 1.

## P-1 — no test opens a game install

Twenty-three test files changed by this story. Two of them read a file at all, and neither reads an
install:

- `pkg/ui/minimap_test.go` reads `command.go` — this package's own source, relative to the package
  directory, to assert that the file dispatching a press names the minimap's capture.
- `cmd/againrom/main_test.go` builds a **synthetic** install in a temp directory (`writeInstall`),
  including deliberately undecodable maps, and points the binary at that.

## What is not verified

**A monster's doll.** `FR-9a`'s substitution is drawn and asserted; the ruling behind it is not met
and cannot be until research names where a non-human unit's figure art lives. What is verified is
that *something* is drawn for a selected non-party unit and that it is that unit's own world frame.

**FR-2a's second clause is a limit, not a criterion.** A positive mission number naming an absent
map is not refused before the window. Recorded above with the run that shows it.

**FR-6's "never smaller than one pixel" holds by construction, not by the clamp that claims it.**
Writing the criterion showed the clamp inside `drawMinimapView` cannot fire for either output
mapping this file defines: the scaled branch multiplies both bounds by the same scale, and the
sampled branch rounds the upper bound up where it rounds the lower down. The comment asserting a
reachable case ("a zoomed-in camera over a sampled 400x400 map") was corrected at this stage; the
clamp is kept as defensive and is now labelled that way.

**Nothing here is measured against the original.** Every placement, size, colour and letter in this
story is authored; `provenance.md` says which claims bound it and which question is open. No
criterion above should be read as evidence about what the original game draws.
