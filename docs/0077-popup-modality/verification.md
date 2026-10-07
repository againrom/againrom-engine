# Verification — a popup takes everything onto itself

Environment: Windows 11, Go 1.26.1 (`go.mod`'s pin), `windows/amd64`. Worktree
`againrom/wt-0077` on `impl/0077-popup-modality`, rebased onto `origin/master` at `d22630d` before
any code was written — 0075's attack order had landed by then, so the order path this story gates
is the one that carries it. Research submodule at the story's pin, `20921e2`, no leading character.

## The gate

```
go version                                        go1.26.1 windows/amd64
go build ./...                                    clean
go vet ./...                                      clean
gofmt -l $(git ls-files '*.go')                   empty
go test -count=1 -trimpath ./...                  ok, every package
bash scripts/check-no-game-assets.sh              exit 0 — clean (tree scan)
bash scripts/check-doc-budget.sh                  exit 0
bash scripts/check-sdd-audit.sh                   exit 0 — 103 notes/warnings, none enforced
```

The same set ran green on the rebased tree with no change applied, before T1, so a green gate here
is not a gate that was already red. Exit codes were read from the command itself, not from a pipe.

`check-sdd-audit`'s one note about this story is `3/4 tasks landed, still to come: T4`, which is
correct and expected: T4 is a manual-runbook task and a build task commits nothing, `builds/` being
gitignored. Its note/warning **count** is meaningless from a lane worktree — `builds/` is untracked
rather than ignored, so a worktree has none and the count is taken from the orchestrator's seat or
not at all. Only the FAIL set is compared, and it is empty.

Deletion sweep, the check a green gate cannot make for itself:

```
git diff --diff-filter=D --name-only d22630d..HEAD     (empty)
```

Task↔commit bijection, one id per implementation commit and none twice:

```
f8d879e  0077-popup-modality/T3
e5a2afe  0077-popup-modality/T2
b8bdc07  0077-popup-modality/T1
632c5d1  (no trailer — the artifact commit, outside the mapping)
```

## What was run, and what it printed

Every case below is in `pkg/ui`, needs no window and reads no game install. Each ran under
`go test -count=1 -trimpath`.

```
--- PASS: TestAPopupTakesEveryMapInput                        AC-1
--- PASS: TestAPopupTakesTheBlowsAndTheDiagnostics            AC-2
--- PASS: TestAPopupPinsTheCamera                             AC-3, P-2
--- PASS: TestAPopupAbandonsTheGestureInFlight                AC-4
--- PASS: TestAPopupHoldsTheAmbientClock                      AC-5
--- PASS: TestAPopupHoldsTheWorldDrivenPicture                AC-5a
--- PASS: TestTheDimIsComposedAfterEverythingButThePopup      AC-6
--- PASS: TestAScreenThatCannotDrawAPopupIsUntouched          AC-7, P-1
--- PASS: TestAPopupStillAnswersItsOwnThreeInputs             AC-8
--- PASS: TestWithNoPopupNothingIsGated                       AC-9
--- PASS: TestAPopupLeavesTheSuspensionSeamAlone              AC-10, P-4
```

**AC-1, AC-2 each run their gated case against a control on a fresh fixture.** With no popup open a
tap moves the selection and a secondary press reaches the order seam; with one open the selection is
the one held when it opened, all three seams are empty, and the arming key leaves the mode down. The
control is asserted with `t.Fatal`, so a fixture that never answered fails the case rather than
passing it.

**AC-3 is four inputs × two conditions.** Each of the pan key, the edge margin, a Shift-drag and the
wheel is required to move the camera with no popup and to move nothing with one. Two of the four
needed a fixture fact to be a real test at all, and both are the kind that turns a case into a
tautology: an unmodified drag on a map screen is a selection rectangle and pans zero **by design**,
so the drag case is driven with Shift; and a map opens with its camera against the left-hand clamp,
so the edge case uses the **right** edge. Driven the obvious way, both would have passed by moving
nothing in either condition.

**AC-5 holds a popup for 100 frames** — 10 s of the fixture's wall clock at a 62 ms ambient period —
and requires the counter to be byte-identical across all of them, then requires the two frames after
the dismissal to advance it by at most 4. A gate placed on the call rather than after the baseline
would have paid ~161 there. The bound is hand-computed from the fixture's schedule
(2 × 100 000 µs ÷ 62 000 µs = 3 and a remainder), not read back.

**AC-5a asserts the map picture's second time source over the drawn state**, not over a counter:
every unit's frame, its displacement between two ticks and its route are the ones drawn on the frame
the popup was raised on, across 40 held frames. It is written that way because AC-5 alone would have
certified a build in which units still slide behind the box.

**AC-6 is witnessed by parsing `Viewer.Draw`'s AST**, which is 0073's own mechanism amended rather
than replaced: the dim's statement must come after `overlayPasses`, `pathScreenSegments`,
`panelPresent` and `readoutPresent`, and before `noticePresent`. The helper still fatals if any of
those names is mentioned by two top-level statements, so a second dim is not expressible either.

**AC-7 is the error case.** A map screen with no font is handed two notices of both kinds and then
drives all four camera inputs, the ambient clock, an order and a frame composition: everything
answers, and no dim is composed. P-1 is the same case read as the negative invariant.

**AC-10 runs over both states of the player's pause.** Eight held frames ask for eight advances,
the far side is told `stopped: true`, the world runs zero ticks, and the pause setting comes back out
of the dismissal unchanged in both legs.

**The three amended witnesses.** Two shipped assertions failed when the gate was applied, which is
what a brownfield pin is for, and both were amended rather than deleted:

- `notice_test.go` asserted that a press away from the button reaches the map's gesture — 0066 FR-8.
  It now asserts the press reaches **nothing**, and keeps 0066's own half: the advance is still asked
  for and the notice is not dismissed by a press at the map.
- `halt_test.go` asserted that the camera still pans under an open notice — 0073 FR-1's "a
  suspension stops the world, not the screen". It now asserts the camera does not move, and keeps
  what that block was really pinning: a frame carrying an input still runs no world time.

## The two judgment gates, and what they found

Both are required for spec-anchored and both ran in a separate context. Neither returned nothing,
and the defects each found are the evidence they were worth running.

**Peer-prediction** (a zero-context reader given `spec.md` alone) reconstructed the design — one
answer, four readers — and found the contract underdetermined in four places that mattered. FR-6
carried two sentences that could not both be load-bearing: "the whole drawable area darkens" and "its
rectangle … exactly what it was". And AC-5 asserted a single counter where FR-5 named five moving
things, so it would have **certified a wrong build as right**. FR-6 was rewritten to say the
rectangle already covers the whole area and only the draw order moves; AC-5a was added. The reader
also could not resolve the three dismissal inputs, the blow keys or the diagnostics, none of which
the spec had named; all are now named.

**The adversarial read** (a separate context given spec + plan, told to surface underspecification
rather than fill it) found four material gaps:

1. **A fifth path writes the camera** — declaring the window's size re-clamps the view, writing both
   position and scale, and it is reached on a resize and on the notice's own dismissal arm. P-2 as
   written was false. P-2 now speaks of the **player's** ways to the camera, and the re-fit is
   disclosed: a view that stopped re-fitting would be a broken picture, not an intercepted one.
2. **The advance-then-step order becomes load-bearing** for AC-4 and the shipped comment at that call
   site said the two were independent. It is corrected in T1, and DD-3 records that the correction is
   the only mechanism standing between a future reorder and a silent FR-4 failure.
3. **AC-3's control could not pass** on a command-mode viewer without Shift. Corrected in the spec
   and driven that way.
4. **The Escape arm never reaches the advance AC-10 counts.** AC-10 and FR-8 now say "every frame
   that reaches the map screen's own arm", and name the exception.

It also corrected three plan statements that were wrong rather than incomplete: the camera has **two**
call sites and not one, the slop accumulator is written in **both** the step and the resolver, and
the ambient setter's paragraph is **narrowed** rather than superseded — the type still takes no stop
and water still runs under the player's own pause. All three are fixed in the plan and in the code
comments the story writes.

## Pre-task gate

The defect class named before running it: an **orphaned-state bug** — a latch written on one side of
the new gate and read on the other. It found one, before any code was written: `held` is the gesture
resolver's press latch, the arm's gate skips the resolver, so a press taken before a popup opened
would have outlived it and a later release with no press would have been judged a tap at a stale
press point. That is DD-5, and it is why the viewer's gate clears two latches and not one.

## Limitations, stated

- **The far side's own rule is read, not re-proved here.** AC-5a's stand-in pushes its drawn state
  from inside the tick loop because that is what `pkg/game` does; this story asserts that a popup
  therefore freezes the picture, not that the push is where it is. A change moving that push out of
  the loop would break FR-5 with every case in this file still green, which is exactly why AC-5a
  asserts over the drawn state rather than over the ambient counter.
- **Nothing here reaches `pkg/sim`.** P-3 is not witnessed by a new test: the story adds no
  simulation state to witness. `internal/archtest`'s import and source scans over `pkg/sim` ran green
  in the full test run, and the diff touches no file under `pkg/sim` or `pkg/game`.
- **A button held across a dismissal may draw a rectangle that selects nothing.** Its release finds
  no press latch, so it decides nothing; it is the shape a button already held when a map opens has
  always had. Accepted and recorded, not fixed.
- **No pixels were read back.** This package cannot compose a frame before the engine starts, so
  AC-6 is asserted over the draw method's structure. That the resulting picture is what the author
  wants is his judgement.

## The manual observation

Not made. `builds/0077-popup-modality/` holds this branch's binary and a README giving the exact
invocation against a lawful install through `-assets`/`AGAINROM_ASSETS`, and the four things to look
at. The `builds/README.md` index lives in the orchestrator's tree, not in a lane worktree, which has
no `builds/` at all.

**No claim is made here about how the result looks.** Whether the darkening reads as one intercepted
screen is the product author's judgement, and 0074 was right to refuse to make it; this story makes
it no more than that one did. What is recorded above is what was measured.

## Conclusion

Every AC and both derived properties that this tier can reach have actual evidence, and the two that
it cannot — the appearance, and the far side's push site — are stated as limits rather than covered
by an assertion that would not discriminate. The full gate is clean, the deletion set is empty, and
the bijection holds. The contract's four amendments to landed stories are each carried by an amended
witness rather than by a deleted one, so nothing that was pinned has become unpinned.

The confidence this supports is: the behaviour the contract specifies is implemented and asserted
against controls that fail when the gate does not fire. It does not extend to the ruling being
satisfied — that needs the build in front of its author.
