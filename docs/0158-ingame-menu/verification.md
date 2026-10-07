# verification — 0158 in-game menu

No `tasks.md`: one lane implemented the whole slice. Every FR and DD is accounted for below.

## The result someone can point at

The script-gap census is unchanged, and this story was not expected to move it. Measured from this
worktree with `cmd/missionrun` against the preserved EN root
(`AGAINROM_ASSETS=<againrom>/gameversions/en`, `-trace -ticks 1`, counting `UNSUPPORTED` lines):

| | mission 10 | mission 20 |
|---|---|---|
| master before this story | 17 | 11 |
| this branch | 17 | 11 |

Both agree with `pipeline/milestone-baseline.txt`, whose mission-10 rows sum to 17 (1 + 2 + 13 + 1)
and whose mission-20 row is 11. The story adds no script node handler, so unchanged is the expected
result and is recorded as a claim rather than assumed.

The result is in the build instead: `builds/0158-ingame-menu/`. Escape in a mission or at the town
square now raises a panel over the surface, which stays drawn and darkened, and the world stops while
it stands. Before this story the same key replaced the screen with a four-row list.

## Integration drive

`scenarios/0152-save666.json` was run through the production controller with no window, from this
branch after master was merged:

```
AGAINROM_ASSETS=<againrom>/gameversions/en againrom -headless scenarios/0152-save666.json
```

Exit 0, 36 steps. The step that saves reports `screen=gamemenu`, so the menu opened over a real
mission loaded from the lawful EN root, a save was taken through it, and the run returned to
`screen=map` and finished its assertions. That is the whole of the save/load path through the new
surface, over real assets, in one run.

**The panel was not seen on a screen.** No window was opened for this story. The drive above and the
tests below exercise the same dispatch a press reaches, and the draw path was run headless, but what
a player sees is not claimed here.

## Gate

Run from the worktree on a clean tree at `98330f2`, after merging master (`f53bcde`) and updating
the research submodule to `e885699` — `git submodule status` shows no leading character.

| Check | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `gofmt -l $(git ls-files '*.go')` | prints nothing |
| `go test -trimpath -count=1 ./...` | green, whole module |
| `scripts/check-no-game-assets.sh` | clean (tree scan) |
| `scripts/check-doc-budget.sh` | ok — spec 9627, plan 5481, provenance 3282 |
| `scripts/check-hotfix-ledger.sh` | ok |
| `scripts/check-sdd-audit.sh --story 0158-ingame-menu` | ok, with the expected note that there is no `tasks.md` |

Commits carry no `SDD-Task:` trailer, because there is no `tasks.md` for one to name, and no
`Co-Authored-By` or `Claude-Session` trailer. Checked with
`git log --format='%h %(trailers:key=Co-Authored-By) %(trailers:key=Claude-Session)' f53bcde..HEAD`:
empty for every commit.

## FR

| Id | Where it landed | What witnesses it |
|---|---|---|
| FR-1 | `flow.openGameMenu`, `App.Draw`'s `mapShowing` arm, `App.drawGameMenuOverMap` | `TestEscapeRaisesTheMenu`, `TestTheDrawPathComposesBothSurfaces` |
| FR-2 | `Viewer.menuUp`, `popupOpen`, `App.holdMapUnderMenu` | `TestTheMenuStopsTheWorldAndDarkensTheMapBehindIt`, `TestTheWorldIsToldStoppedAndNoSpanIsBanked` |
| FR-3 | the viewer's `noticeBackdropOf` under the raised popup answer; the town arm's own fill | `TestTheMenuStopsTheWorldAndDarkensTheMapBehindIt` |
| FR-4 | `flow.closeGameMenu`, `flow.escape`'s `ScreenGameMenu` arm | `TestTheMiniMenuReturnsToTheScreenItWasOpenedFrom`, `TestTheWorldIsToldStoppedAndNoSpanIsBanked` |
| FR-5 | `missionGameMenuRows`, `townGameMenuRows` | `TestTheTwoSurfacesAreTheDecodedListsInTheDecodedOrder` |
| FR-6 | `gameMenuPanelRect`, `gameMenuRowRect` | `TestTheDecodedRectangles` |
| FR-7 | `gameMenuAccelerator`, `flow.chooseGameMenuAccelerator`, `App.stepGameMenu`'s `Typed` arm | `TestTheAcceleratorComesFromTheLabel`, `TestTheAcceleratorChoosesItsRow`, `TestEverySurfacesAcceleratorsAreDistinct` |
| FR-8 | `gameMenuPickerRows` + `Picker.Choose` | `TestTheThreeStubRowsAreListedAndGreyed`, `TestTheAcceleratorChoosesItsRow` |
| FR-9 | `App.stepGameMenu`, `gameMenuRowAt` | `TestAPointerReleaseHitsTheRowItIsDrawnOn` |
| FR-10 | `flow.chooseGameMenu` | `TestSaveTellsTheFarSideWhichScreenItWasTakenOn`, `TestTheLoadRowOpensTheLoadWindowAndEscapeComesBack` |
| FR-11 | `flow.openGameMenu`'s two predicates | `TestWithNoStoreSaveAndLoadAreGreyedAndNeverCallNil` |

## AC and P

AC-1 and AC-2 are `TestEscapeRaisesTheMenu` and `TestTheMiniMenuReturnsToTheScreenItWasOpenedFrom`'s
town case, which opens the town surface and returns to it.
AC-3 is `TestTheMenuStopsTheWorldAndDarkensTheMapBehindIt`, which asserts the one popup answer and
the dim it produces, and `TestTheTownMenuRaisesNoMapPopup` for the surface with no viewer.
AC-4 and AC-5 are `TestTheWorldIsToldStoppedAndNoSpanIsBanked`.
AC-6 and AC-7 are `TestTheTwoSurfacesAreTheDecodedListsInTheDecodedOrder`.
AC-8 is `TestTheDecodedRectangles`.
AC-9 and AC-10 are `TestTheAcceleratorComesFromTheLabel` and `TestEverySurfacesAcceleratorsAreDistinct`.
AC-11 is `TestTheThreeStubRowsAreListedAndGreyed` for Enter and the pointer gate, and
`TestTheAcceleratorChoosesItsRow` for the third input.
AC-12 is `TestTheMiniMenuReturnsToTheScreenItWasOpenedFrom`, which drives both doors on both surfaces.
AC-13 is `TestSaveTellsTheFarSideWhichScreenItWasTakenOn`'s leaving arm and
`TestTheApplicationDrivesTheMiniMenuAndTheLoadWindow`, which also asserts the world's seams are
released.
AC-14 is `TestAPointerReleaseHitsTheRowItIsDrawnOn`, which includes three points inside the panel
that are on no row.
AC-15 is `TestTheDrawnLabelCarriesNoMark`.

P-1: this story names no simulation type, adds no serialized field and does not move the byte-form
version. `pkg/ui` cannot import `pkg/sim`, which `internal/archtest` enforces, and the diff touches
no file under `pkg/sim`.
P-2: every label, colour and rectangle is authored in `pkg/ui/gamemenu.go`. The whole suite runs
green with no game install present, which is golden rule 2 and is what the gate above ran under.
P-3: `TestTheDecodedRectangles` asserts every listed row's rect is inside its panel, and
`TestThePanelPaintsInsideItsOwnRect` runs the paint over an offscreen frame with a nil list, no rows
and a real surface.

## DD

DD-1: the menu is still a `Screen` value; `flow.mapShowing` is the widened screen test. Witnessed by
every test that reads `f.screen == ScreenGameMenu`, and by `pkg/game/headless.go` continuing to drive
it unchanged in the integration run above.
DD-2: `Viewer.popupOpen` gained one disjunct. Reverting it fails
`TestTheMenuStopsTheWorldAndDarkensTheMapBehindIt` at "the menu is up over the map and no popup is
reported" — checked by deleting `|| v.menuUp` and re-running.
DD-3: `App.holdMapUnderMenu`. Reverting it — deleting the call from the `ScreenGameMenu` arm — fails
`TestTheWorldIsToldStoppedAndNoSpanIsBanked` at "the advance ran 0 times on a menu frame" and "the
far side was told []".
DD-4: the map path has no dim statement of its own; the town arm fills its canvas with
`AuthoredNoticeBackdrop()`. `TestTheMenuStopsTheWorldAndDarkensTheMapBehindIt` asserts the map path's
dim is that same value.
DD-5: `paintGameMenu` takes a destination and writes frame coordinates.
`TestTheDrawPathComposesBothSurfaces` asserts the map path allocates its own 640x480 overlay frame
and that the town path composes into the canvas.
DD-6: `gameMenuAccelerator` resolves from the label on every lookup; no row stores one.
`TestTheAcceleratorComesFromTheLabel`'s quest-objectives case is the discriminating one — a stored
letter would have to store `q` beside a fallback of `m`.
DD-7: `flow.openGameMenu` answers both predicates once; `flow.menuRows()` derives the list from them.
Witnessed by `TestWithNoStoreSaveAndLoadAreGreyedAndNeverCallNil` and by `flow` holding no field that
maps a row to what it does, which `TestFlow`'s field census enforces.
DD-8: nothing in the diff opens `lm.256` or `dialogs.txt`. `grep -rn "lm\.256\|dialogs\.txt"` over
`pkg/` and `cmd/` returns nothing.
DD-9: `gameMenuRow.Action` is a constant and `chooseGameMenu` is one switch.
`TestTheTwoSurfacesAreTheDecodedListsInTheDecodedOrder` asserts the order by position.

## SC

SC-1: the gate table above.
SC-2: the census table above.
SC-3: `TestTheDecodedRectangles` writes the four rectangles out as literals — `(100,60)-(440,400)`,
`(100,100)-(440,340)`, `(140,100)-(392,130)`, `(140,140)-(392,170)` — rather than calling the layout
helpers on both sides of the comparison.

The spec's AU-1 through AU-7 are the authored divergences, not checks. Each is stated in `spec.md`
with what it costs the player. AU-4 is witnessed by `TestTheThreeStubRowsAreListedAndGreyed`, AU-7
by `TestWithNoStoreSaveAndLoadAreGreyedAndNeverCallNil`. AU-6 is witnessed by
`TestAClickOutsideTheEscMenuReachesNothingInTheTown` and
`TestAClickOutsideTheEscMenuReachesNoOrderOnTheMap`. AU-1, AU-2, AU-3 and AU-5 are statements about
what is absent from the build and are witnessed by the absence itself: no sprite bank is loaded, no
label file is read, no diplomacy row is constructed and no confirmation panel exists.

## Behaviour changed for an earlier story

0143 FR-10 said a front end with nil save seams keeps SAVE and LOAD reporting that there is no store.
0158 FR-11 greys both rows instead. The refusal message in `chooseGameMenu` is kept as a guard but is
now unreachable through the menu. `cmd/againrom` installs the seams, so the shipped build is
unaffected; what changes is the two rows' appearance in a front end that installs none.
