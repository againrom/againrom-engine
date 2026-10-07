# Verification — a dialog window stops the world

Environment: Windows 11 Pro 10.0.26200, Go toolchain pinned in `go.mod`, no CI. Research submodule at
`f56bb38`, checked out at its pin (`git submodule status` shows no leading character). No test below
reads a game install. Tests run with `-trimpath`: Windows Defender quarantines an untrimmed test
binary in this tree.

## The gate

```
go build ./...                          EXIT 0
go vet ./...                            EXIT 0
gofmt -l $(git ls-files '*.go')         EXIT 0, no output
go test -count=1 -trimpath ./...        EXIT 0, ok in all 30 packages that have tests
bash scripts/check-no-game-assets.sh    EXIT 0   check-no-game-assets: clean (tree scan)
bash scripts/check-doc-budget.sh        EXIT 0
bash scripts/check-sdd-audit.sh         EXIT 0
```

The deletion set across the whole story, which a green gate cannot answer — removing a package
compiles:

```
git diff --diff-filter=D --name-only 86fd668..HEAD
(empty)
```

## Baseline, driven before anything was changed

The story's premises were measured, not read, over the production map arm with a recording seam. The
probes were thrown away; their output is the S-5 baseline:

```
no notice:    10 advances over 10 frames
notice open:  10 advances over 10 frames, 0 cadence calls
Space with a notice open: 1 cadence call, flow.stopped=true
```

The world advanced behind a notice at full rate, nothing crossed the seam when one opened, and all
three cadence keys were live under one. Nothing dimmed: `Viewer.Draw` had no darkening step.

## The measurement that chose the mechanism

The far side's pacing, driven over one three-second span both ways:

```
ordinary frame:                            0 ticks
after a 3s gap with no calls:              4 ticks in ONE call
after a stopped span of the same length:   0 ticks in the resuming call
period=62000  maxCatchUp=4
```

The design this story would obviously have taken — the front-end declining to ask for an advance —
repays a quarter-second of world time in a single frame. Declaring the stop and going on asking costs
nothing on resume. Taken before the design was written, not after.

## The three gates, and what each actually found

**Peer-prediction (spec alone, zero-context reader).** The reader reconstructed the intended
behaviour including the mechanism, so the bar was cleared on intent. It found six real gaps, all
closed before the plan gate: FR-4's enumeration was narrower than the sentence framing it; whether an
order issued under a notice is refused or queued was unstated; AC-7 pinned FR-6's disappearance
rather than FR-8's substitutability; AC-9 traced to no FR; FR-7 named three exempt boxes where AC-6
pinned one; and the Constraints table named an internal seam a zero-context reader cannot resolve.

**Adversarial plan read (separate context, no access to the authoring conversation).** It checked all
eight of the plan's baseline facts against the source and found none wrong. It found two defects.

*HIGH — the suspension is enforced per advance, not per tick.* The far side reads its stop once per
advance and then runs whatever the elapsed span is worth, so a tick that raises a notice does not stop
the rest of that same advance. Verified in this seat rather than taken on report:

```
period= 62000 us: maxCatchUp=  4, one 500ms call ran n=  4 ticks; notice opens at tick 6
period=  4000 us: maxCatchUp= 62, one 500ms call ran n= 62 ticks; notice opens at tick 6  -> 56 residue
period=  1000 us: maxCatchUp=250, one 500ms call ran n=250 ticks; notice opens at tick 6  -> 244 residue
```

244 ms of world time after the notice was open, inside one advance, at a one-millisecond period; zero
at the shipped one, where the same drive ran four ticks and never reached the notice. Disclosed and
stated as the contract's granularity rather than removed — plan DD-8 carries the decision and the
rejected far-side fix. It is not observable as motion behind an open box: the notice is drawn from the
frame after the advance that raised it, and that frame already has every one of those ticks applied.

*MEDIUM — the dim's gate was not pinned to the font-carrying predicate.* A notice is marked open
whatever the font, so a dim keyed on the raw flag would darken a map with no box on it. The
implementation already asked the font-carrying predicate; the plan did not require it. Closed in
DD-6 and pinned by `TestAViewerThatCannotDrawANoticeDimsNothing`.

**Separate-context tests (T1).** The author held `spec.md` and the existing test scaffolding and was
instructed not to read the implementation. It reported one failure rather than weakening it, and the
failure was real — see the finding below.

## The finding the test author raised, and what it turned out to be

`TestAC2.../ESCAPE` failed: 31 frames after an Escape dismissal ran 51 world ticks where the other two
routes ran 50. The Escape frame makes **0** advance requests where RETURN and the button each make 1,
because Escape is answered where it is read and returns before the map arm — so that frame's time is
owed and paid by the next one.

Run down in this seat with each route driven alone on a fresh fixture:

```
RETURN:     50 ticks after dismissal
ESCAPE:     51 ticks after dismissal
the button: 50 ticks after dismissal
```

and again on a fixture the loader gave **no cadence seam**, where nothing can ever be suspended:

```
RETURN: 50   ESCAPE: 51   the button: 50
```

The same single tick with the suspension absent, which is what says it is 0066's rule and not this
story's. It is one frame, once per Escape dismissal, and it is now stated in FR-2 and disclosed. The
test derives its expectation from the request count it measures rather than from a constant, so a
later story that makes the Escape frame reach the advance fails it asking for the table to be
corrected.

## Green-but-hollow audit — two mutations, both killed

The two claims most able to pass hollowly were mutated rather than argued about.

*FR-7 is a claim about draw ORDER, and an ebiten image's pixels cannot be read back before the game
starts, so it is witnessed by parsing `Draw`.* The dim statement was moved to the end of `Draw`:

```
--- FAIL: TestTheDimIsComposedAfterTheMapAndBeforeEveryBox
    the dim is composed at statement 7, at or after "panelPresent" at 4 — that box would be dimmed
    the dim is composed at statement 7, at or after "readoutPresent" at 5 — that box would be dimmed
    the dim is composed at statement 7, at or after "noticePresent" at 6 — that box would be dimmed
```

The test also fails if the dim is mentioned by two top-level statements or by none, so it cannot
silently stop measuring.

*FR-2 is the headline claim, and the accommodation above could have blunted it.* The implementation
was mutated into the design the spec rejects — the advance withheld while suspended:

```
--- FAIL: TestAC1AnOpenNoticeStopsEveryAdvance
    0 advance requests were made over 31 suspended frames, want 31 — the suspension is declared,
    not withheld (Constraints, column B)
--- FAIL: TestAC2.../ESCAPE
    the 31 frames after a dismissal by ESCAPE ran 101 world ticks, want 51 — ... A repaid
    SUSPENSION would show the whole held span here (FR-2, AC-2)
```

101 against 51 is the discrimination that matters: a repaid suspension shows the whole held span, not
one frame. Both mutations were reverted and the suite re-run green.

## Acceptance criteria

| AC | evidence | outcome |
|---|---|---|
| AC-1 | `TestAC1AnOpenNoticeStopsEveryAdvance` — control 50 ticks, suspended 0, resumed 50, one declaration crossed | pass |
| AC-2 | `TestAC2DismissalResumesWithoutRepaying` — holds of 0, 31 and 310 frames all resume identically | pass |
| AC-3 | `TestAC3ThePlayersPauseSurvivesTheNotice` — both arms against each of the three routes | pass |
| AC-4 | `TestAC4AnOutcomeNoticeSuspendsLikeADialogueOne`; `TestBothNoticeKindsDimIdentically` | pass |
| AC-5 | `TestAC5TheCadenceKeysAreInertWhileANoticeIsOpen` — six presses under a notice, 0 declarations | pass |
| AC-6 | `TestAnOpenNoticeDimsTheWholeView`; `TestTheDimIsComposedAfterTheMapAndBeforeEveryBox` | pass |
| AC-7 | `TestNoDimWithoutANoticeOrWithATransparentValue`, three arms; `TestTheDimIsTheValueItWasGiven` | pass |
| AC-8 | `TestAC8AMapScreenThatCannotDrawANoticeIsNotSuspended`; `TestAViewerThatCannotDrawANoticeDimsNothing` | pass |
| AC-9 | `TestAC9AMapScreenWithNoCadenceSeamIsUnchanged`; `TestNoDimWithoutADrawableArea` | pass |

## Properties

| P | evidence | outcome |
|---|---|---|
| P-1 | `TestP1NoNoticeSuspendsAScreenThatCannotDrawOne` — eight notices of both kinds over eight runs, 400 ticks, 0 declarations; `TestAViewerThatCannotDrawANoticeDimsNothing` repeats four pushes and then watches the dim arrive with the font | pass |
| P-2 | `TestP2ThePauseSettingIsChangedByThePauseKeyAndByNothingElse` — six notices opened and closed around one key press, observed through the world rather than through a field | pass |
| P-3 | neither commit touches a file under `pkg/sim` or `pkg/game` product code; the whole suite including every pinned digest is green and no serialized version moved | pass |
| P-4 | `TestP4TheCadenceIsDeclaredOnlyWhenItChanges` — seven steps, about 250 frames, exactly four declarations | pass |

## Success criteria

| SC | outcome |
|---|---|
| SC-1 | met — AC-1 and AC-2, driven over the production map arm |
| SC-2 | met — AC-3, through each of the three dismissal routes |
| SC-3 | met — AC-4 and AC-5 |
| SC-4 | met — AC-6 and AC-7, with the order mutation as the discriminator |
| SC-5 | met — AC-8 and AC-9 |
| SC-6 | **not run here** — see *Limitations* |
| SC-7 | met — the gate block above |

## Limitations, and what is not claimed

**SC-6 is the owner's to run, and it is the criterion the ruling actually came from.** It needs a
lawful install and an eye: the dim's STRENGTH is a judgement, and nothing automated can say whether
half opacity is the dimming that was asked for. A runnable build is provided under
`builds/0073-dialogue-halts/` with the invocation and the four things to look at. Until that run
happens this story claims the dim EXISTS, covers the map, stops at the panel and is replaceable — and
does not claim it looks right.

**Nothing here is a claim about the original.** No published claim connects a dialog window to the
engine's own run bit, whose writers were never enumerated. The pause and the dim are the owner's
ruling as this product's author.

**The two sides of the seam are witnessed separately and cannot be witnessed together.** The import
graph runs `game → ui` and the front-end's step is unexported, so no single test can drive the real
front-end over the real world. What crosses the seam is measured in `pkg/ui` against a stand-in built
to the far side's own contract — baseline consumed on every call, ticks credited only when running;
what the far side does with a stop was pinned in `pkg/game` before this story. The composition is
therefore an inference from two measurements rather than one measurement, and that is the weakest
link in this story's evidence.

**Two frames of world time are outside the suspension**, both measured, both stated in the contract,
neither reclaimed: the advance that raises a notice runs out its own span (up to one catch-up span,
zero at the shipped cadence), and the frame that dismisses by Escape asks for no advance (one tick,
once). Residual product risk: at a fast cadence the first is up to a quarter-second of world time,
invisible per frame but real in the world's own timeline.

**Water still moves behind a stopped, dimmed map.** Inherited from this tree's pause, disclosed in the
contract, not closed here.

## Recorded deviations

**T1's tests were authored in a separate context; one mechanical repair was made in this seat.** The
author collided a method name with a field name on its own stand-in; the method was renamed and no
assertion, expectation or hand-computed number was altered. Separately, its accommodation of the
Escape finding — an unmeasured settling frame in AC-3 and AC-9 — was kept, and the AC-2 case that
measures the finding head-on was changed from a constant to an expectation derived from the request
count it measures. The mutation above is what says that change did not blunt it.

**T2's tests were authored in the implementing context**, which departs from the profile's
separate-context default and is recorded rather than glossed. The dim's assertions are geometry and
gating; its one hard claim, FR-7's order, is discharged by the mutation rather than by the author's
independence.
