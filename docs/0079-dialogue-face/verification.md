# Verification — the dialogue window shows who is speaking

## The gate

```
go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -count=1 -trimpath ./...
EXIT=0        32 packages ok, 0 failures, 4 with no test files

bash scripts/check-no-game-assets.sh              EXIT=0  clean (tree scan)
bash scripts/check-no-game-assets.sh --history    EXIT=0  clean (history scan)
bash scripts/check-doc-budget.sh                  EXIT=0
bash scripts/check-sdd-audit.sh                   EXIT=0  (T6 is a manual runbook and commits nothing)
```

Each exit code is the command's own, taken by branching on it rather than by
reading a pipeline's tail.

The deletion sweep over the whole story is empty:

```
$ git diff --diff-filter=D --name-only ba2d560 HEAD
$ git diff --name-only ba2d560 HEAD | grep -c '^pkg/sim/'
0
```

## What was run, and what it printed

### The corpus, on both preserved roots (AC-11)

`cmd/dlgtool census`, this tree's own parser over every event file of a root:

```
                                    EN     RU
event files                        225    228
file test says a pane              225    228
some part names a speaker          225    228
the two tests disagree               0      0
parts                              513    518
parts naming a speaker             513    518
```

Three published figures are reproduced by an independent implementation: the
225/228 file counts and their agreement on both tests, and the 513/518 part
counts. **Every event file on both roots gets the pane** — so the layout this
tree shipped before this story is the one the original never draws, which is the
finding the story exists for.

The last row is stronger than expected and was checked against a second
instrument before being believed. A `grep` over the extracted corpus counts 535
EN / 543 RU tags carrying a part number and **0 of them lacking `npc=`**, on
either root. So no shipped part names nobody, and FR-4's "leave the standing face
alone" is unreachable from shipped data — reproduced from the instructions
alone, exactly as the research row says the separation is. It is witnessed by
fixtures instead (AC-5).

### The composed window, from each root's own font and its own text (AC-12)

A one-off developer render outside the repository, composing the window through
the same `RenderNotice` the map screen calls, on `main/text/battle/m10/event01.txt`:

```
EN  font=font1  pane=true  part1 names=true speaker=21  text=(128,36)-(428,172)  portrait=(30,54)-(118,168)
RU  font=font1  pane=true  part1 names=true speaker=21  text=(128,36)-(428,172)  portrait=(30,54)-(118,168)
```

Both roots resolve identically: the pane at the claimed rectangle, the words at
the beside-pane rectangle, the same speaker number off the same tag. The two
pictures are in `builds/0079-dialogue-face/` as `en-m10e01.png` and
`ru-m10e01.png`; the Cyrillic root renders its own bytes through its own atlas
with the pane in place.

Startup on both roots, headless:

```
$ againrom -assets <en|ru> -check -mission 10
againrom: mission 10 at scenario/10.alm, 80x80, 36 entities, party at (17, 66), 11 raise(s)   [identical on both]
```

### Mutation kills

Three mutations, each of the defect the corpus cannot see, each killed:

| Mutation | Killed by |
|---|---|
| the file test collapsed onto `npc=` — the two tests made one | `TestEventHasSpeakerIsTheThreeLetterTest`, `TestTheTwoSpeakerTestsAreDifferentTests`, `TestMissionSettlesTheShapeOnceFromTheFile` |
| every part treated as naming a speaker | `TestMissionPagesTheFaceThroughTheSeam` |
| the viewer writing the face unconditionally | `TestPagingKeepsAndReplacesTheFace`, `TestMissionPagesTheFaceThroughTheSeam` |

The first is the one that matters: it passes every corpus measurement and every
playthrough, and only the authored fixture separates the two needles.

## The witness table

| Id | Evidence |
|---|---|
| **AC-1** | `TestTheShapePushedIsTheShapeComposed`, `TestMissionSettlesTheShapeOnceFromTheFile` — the pane and the 128,36 text rectangle on every part of a speaker-bearing file |
| **AC-2** | `TestEventHasSpeakerIsTheThreeLetterTest`, `TestAPaneWithNoSpeakerNeverGetsAFace` — the pane is drawn, no face ever arrives |
| **AC-3** | `TestTheShapePushedIsTheShapeComposed`, `TestMissionSettlesTheShapeOnceFromTheFile` (row "neither") — no pane, words at 48,36 |
| **AC-4** | `TestTheTwoNoticeShapesDifferInExactlyTwoFields` — the two resolved values compared whole, with the pane and text rectangles normalised away; a field added later cannot escape it |
| **AC-5** | `TestPagingKeepsAndReplacesTheFace`, `TestMissionPagesTheFaceThroughTheSeam` |
| **AC-6** | the same two, third step — a differently-named speaker replaces the face |
| **AC-7** | `TestAnEmptyPaneIsStillPainted`, `TestMissionPagesTheFaceThroughTheSeam` fourth step, `TestMissionWithNoFaceSourceDrawsThePaneEmpty` |
| **AC-8** | `TestEventPartSpeakerReadsThePartsOwnTag` — `<NPC=21, Part=1, female, tips=2 >` |
| **AC-9** | `TestTheOutcomeNoticeNeverHasAPane`, `TestMissionOutcomeDropsTheSpeaker` — including with the shape flag forced on |
| **AC-10** | `TestAFontlessViewerIsUntouchedByTheSpeaker` |
| **AC-11** | the census above, both roots |
| **AC-12** | the two renders and the two startups above, plus the runnable build. **Limitation: no windowed observation was made from this seat** — see below |
| **P-1** | `git diff --name-only ba2d560 HEAD` names no file under `pkg/sim`; the digest and invariance suites are untouched and green |
| **P-2** | both asset scans clean, tree and history; every fixture is bytes built in test code and `dlgtool` prints counts only |
| **P-3** | `TestTheCacheSeesTheShapeAndTheFace` — an unchanged frame recomposes 0 times, a changed shape or face recomposes exactly 1, with the words held identical |
| **P-4** | `TestAFaceDoesNotOutliveItsWindow` over all three closing paths, `TestMissionCloseDropsTheShape` |

Plan criteria 1–8 and 11 are the unit evidence above; 9 is the census; 10 is the
render and the build.

## Limitations, stated

**The face art is not resolved and no picture is shipped.** This is the story's
disclosed open hop and not a defect found here. `REG-NPC-058` publishes the
registry's picture-bearing keys with their presence counts and measured domains
and grades **Unknown** what any of them indexes; no ledger names a resource for
this window's pane. Everything the pane needs except the picture is built and
under test, and the seam that takes one is `game.FaceSource`, nil on every
shipped path. The behaviour the player sees is FR-5's, asserted rather than
assumed.

**AC-12's judgement half was not made.** The window was composed from both roots'
real fonts and real text and the resulting pictures are recorded, which is
stronger than a screenshot of a launch — but no one looked at a running window
from this seat. The runnable build exists for that observation.

**The pane's own paint is ours and reproduces nothing.** Its fill and border are
this project's, like the window's frame and the button's word.

**The census walks parts contiguously from 1.** It stops at the first part number
a file does not hold, which is why it reports 513/518 where a raw tag count
reports 535/543: a file carrying two variant tags for one part number has that
part counted once. The published part figure is the contiguous one, and the two
instruments agree on it.

## Conclusion

The window now draws the shape the game actually uses, on both roots, chosen by
the file-level test and paged by the per-part one, with the face standing,
persisting and clearing by the rules the decode fixes. Every acceptance criterion
has evidence or a stated limitation, and the one thing absent — the picture
itself — is absent because no published claim can say where it is kept, is
disclosed in the contract, and lands later through a seam that moves nothing.
