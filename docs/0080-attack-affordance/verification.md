# Verification — the attack affordance

Environment: Windows 11, Go toolchain pinned in `go.mod`, both lawful installs
present at `gameversions/en` and `gameversions/ru`. `go test` is run with
`-trimpath`, which this project's notes require locally.

## The gate

Branched on exit code, not on grep:

```
go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -count=1 -trimpath ./...
GO_GATE=0

bash scripts/check-no-game-assets.sh      ASSETS=0
bash scripts/check-doc-budget.sh          BUDGET=0
bash scripts/check-sdd-audit.sh           SDD=0
```

`check-sdd-audit` read `SDD=1` until this file existed, with the single line
`FAIL 0080-attack-affordance: every task in tasks.md has landed and there is no
verification.md` — recorded because it is the evidence that the audit is
actually looking at this story rather than passing it over.

## AC by AC

Every case below is a named test. Package `ui` cases drive `App.step`, so what
they witness is the shipped dispatch and not a struct.

| AC | Evidence |
|---|---|
| AC-1 | `TestTheModifierIsALevelAndNotAToggle` — three frames: down raises, up lowers, down raises again. A toggle would have been flipped twice by the third and would read down |
| AC-2 | `TestTheModifierSurfaceIsUngated`, both cases. Each first asserts that the KEY refuses the same selection, so the case cannot pass by the gate having quietly gone away |
| AC-3 | `TestTheModifierIsNotSpentByThePressAndTheKeyStillIs/a press under the held modifier…` — two presses under one held modifier yield four attacks, two per press, ascending id, and no move order |
| AC-4 | `TestFocusLossLowersTheModeWhicheverSurfaceRaisedIt`, both writers. The modifier case carries the modifier ACROSS the unfocused frame, which is the only shape that catches the map arm re-raising it |
| AC-5 | `TestThePointerIsDrawnExactlyWhileTheModeIsUp` — the three returns of `attackPointerPresent`, plus the never-stepped viewer |
| AC-6 | `TestTheSystemPointerIsHiddenExactlyWhenOursIsDrawn` — the wanted state equals the pointer's own bool at every step, the engine is told once per transition and not at all on an unchanged frame |
| AC-7 | `TestLoadAttackPointerResolvesPaletteAndCoverage` (four cells, expectations written out rather than recomputed from the rule) and `TestLoadAttackPointerRefusesWhatItCannotResolve` (five refusals) |
| AC-8 | `TestTheMarkedUnitIsTheUnitAPressWouldName` — five cases. The tie case drives the snapshot in DESCENDING id, so a first-match implementation marks the wrong unit |
| AC-9 | `TestAPopupTakesTheModeAndDoesNotGiveItBack` (state) and `TestAPopupDrawsNeitherInstrument` (drawn) |
| AC-10 | `TestTheModifierIsNotSpentByThePressAndTheKeyStillIs/the key is still spent…`, and 0075's own suite unedited: `TestArmingIsGatedOnOwnershipAndNotOnClass` and `TestAnArmedPressMakesOneOfTwoOrders` are green with no change to either |
| AC-11 | `TestTheAttackPointerIsLoadedAndIsNeverFatal` — both shapes assemble, the reason names the address, and the summary reports it only on failure |
| AC-12 | `TestThePointerIsDrawnExactlyWhileTheModeIsUp/the standalone developer viewer draws neither` |
| P-1 | `internal/archtest` unchanged and green: no package was added, and what crosses into `pkg/ui` for the pointer is `*image.RGBA` |
| P-2 | The mutation below: forcing `attackMode` to the key's field alone fails 18 cases across five files |
| P-3 | `pkg/sim` is untouched by this branch: `git diff --stat 63769f0..HEAD -- pkg/sim` is empty. The tree's `formatVersion` reads 13 and every byte of that move is 0076's, arriving under this branch's feet at the rebase; it was 12 when this story began and this story did not touch it |
| P-4 | The gate above |

## Mutation — five, each killed by named cases

Each mutation was applied, the suite run, and the source restored.

| Mutation | Killed by |
|---|---|
| `attackMode` returns `v.armed` alone — the modifier surface removed | **18** cases across `TestTheModifier*`, `TestFocusLoss*`, `TestAPopup*`, `TestThePointer*`, `TestTheSystemPointer*`, `TestTheMarkedUnit*` |
| `attackShown` drops its `popupOpen` test — the DRAWN half of the popup rule | `TestAPopupDrawsNeitherInstrument` |
| the unfocused branch stops dropping the modifier out of the snapshot | `TestFocusLossLowersTheModeWhicheverSurfaceRaisedIt/the modifier's mode…` |
| coverage taken as `level*17` (the sprite tool's declared VIEW) instead of `(level+1)/16` (the decoded model) | `TestLoadAttackPointerResolvesPaletteAndCoverage`, two of its four cells |
| the viewer's step stops lowering the mode under a popup — the STATE half | `TestAPopupTakesTheModeAndDoesNotGiveItBack/a mode already up…` |

The second and fifth are the pair the plan claims protect different halves of
FR-7. Each mutation kills exactly one of them, which is what makes that claim a
measurement rather than an assertion.

## Against the lawful installs

The art resolves on BOTH roots and to the same bytes. Read through
`LoadAttackPointer` itself, over each root's own `GRAPHICS.RES`:

```
<seat>/gameversions/en: 32x32, 173 painted cells, 60 fully opaque
<seat>/gameversions/ru: 32x32, 173 painted cells, 60 fully opaque
md5 of the resolved picture, both roots: ad07e60229035e7466d1ed09aa25d83d
```

Rendered and looked at: the picture is a sword, its point about three pixels in
from the top-left corner — which is what the top-left hotspot rests on.

The headless check names no failure on either root, which is the summary's
silence about the pointer:

```
en: againrom: 38 map rows, 8 of 8 buttons have a mask region; hero Iron Short Sword 7-10, to-hit 39, defence 8
ru: againrom: 34 map rows, 8 of 8 buttons have a mask region; hero Iron Short Sword 7-10, to-hit 39, defence 8
en/ru -mission 10: mission 10 at scenario/10.alm, 80x80, 36 entities, party at (17, 66), 11 raise(s)
```

The shipped binary was launched on each root at the owner's own invocation,
stayed up for eight seconds, printed nothing on either stream, and was
terminated externally. That is evidence the window opens and the mission loads;
it is **not** evidence about what was drawn in it.

## Deletion set

```
git diff --diff-filter=D --name-only 63769f0..HEAD
(empty)
```

The whole gate above was run twice: once on this story's original base and again
after the rebase onto `63769f0`, which brought 0076's movement cost and its form
version with it. Both runs read the same — every exit code 0, the same 30 green
packages, the same deletion set.

## Limitations — what is NOT claimed

- **Nothing here says it looks right or reads well.** No frame was captured: the
  front-end needs a window and this seat has no way to photograph one. What is
  claimed is what the code does — which bytes resolve, which predicate answers
  when, what the seams receive. Whether the sword is legible at the window size,
  whether the outline is the right red, and whether a player finds Ctrl are the
  product author's judgement and are deliberately left to him.
- **The synthetic `.16a` fixture and the decoder are mutual inverses.** The
  container fixture in `pkg/game/cursor_test.go` is built from the format
  contract, so the unit cases would pass for a format that ran the other way.
  What closes that is the resolved picture above, taken off the shipped archives
  through the same loader and looked at.
- **The pointer's own draw is not asserted, only its decision.** `Viewer.Draw`
  needs an engine. Every branch it takes is decided above it by two pure methods
  that are asserted; the statements themselves — the upload, the blit, the two
  strokes, the cursor-mode call — are not exercised by a test.
- **The divergence over empty ground is shipped and disclosed, not resolved.**
  The original shows a second cursor there and issues a swarm order under it.
  This build shows the sword and issues the move 0075 already issued.
- **The hotspot and the frame choice are ours** and no measurement here bears on
  either.
