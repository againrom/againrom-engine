# A window mode and dragging the mission map with the middle button

## Intent

Owner direction, from a tester report on Debian: the game ran only over the
whole screen with no setting for a window, and the map moved only by the arrow
keys, the screen-edge bands and the minimap. The owner asked for:

1. A window mode: a starter checkbox and Alt+Enter in the game, switching
   between the window and the whole screen while the game runs. Native on
   Windows and macOS; any other desktop gets the same through the engine where
   it works and a no-op where it does not.
2. Dragging the mission map with the held middle button (the wheel). The
   original lacks it, so one divergence row records it as Againrom's own.

Base: `9609c508` (game 0.127.5, starter 0.9.0, knowledge pin k333).

## Authority

| Leaf | Authority |
|---|---|
| window mode, Alt+Enter, drag | owner direction; tester report |
| the original takes 640x480, 800x600 or 1024x768 from `-640`, `-800`, `-1024` | `DLG-PANEL-035` (High); second game `R2-ENGINE-238` (High) |
| Enter with Alt held is a system key; its handler acts on digits, S and B..Y only, so it reaches no Enter arm | `MENU-062`, `MENU-082` (High); the library default is Unknown (`MENU-089`) |
| the frame procedure forwards only key-up, mouse move, the left and right button messages and `0x400` to the root | `MENU-082` (High) |
| the default mission input surface has no middle-button action | `AI-KEY-125` (High) |
| the right button is the original's drag pan | `AI-INPUT-127` |

Search for the display mode: `knowledge/claims/*.md` at k333 for
fullscreen, full screen, windowed, DirectDraw cooperative level, display mode,
Alt+Enter, `WM_MBUTTON`, middle mouse and wheel button. No claim reads the
display mode the original requests, a windowed mode or an Alt+Enter handler;
no second-game claim names any of them. The window row is therefore a
`DEVIATION` that also names the unread display mode.

## Bindings read before choosing Alt+Enter

Population: every `in.Enter` reader in `pkg/ui` (menus, load and save
windows, the picker, generation, the gold modal, the cheat chat, notices,
documents, the ending, mod screens, sound options, cutscenes) and every Alt
reader (`altHeld`, `Input.Alt`, `readAltLetter`, the headless key table).
No screen binds Alt+Enter or tests Alt together with Enter. Every Enter
reader ignored Alt, so Enter with Alt held acted as Enter. The original's
handler routes Enter with Alt held away from every Enter arm (`MENU-082`), so
Alt+Enter no longer acts as Enter here either: `splitAltEnter` turns it into
the toggle and clears Enter on that tick. Enter alone is unchanged.

Middle button: no engine screen read `MouseButtonMiddle` before this story
except two idle tests that wait for every button to be up (`cmd/starter`
window and the map editor's key repeat). No claim gives the original's
mission a middle-button meaning (`MENU-082`, `AI-KEY-125`).

## As built

Starter (`cmd/starter`): the settings view's Parameters group gains one
checkbox, "In a window" / "В окне", through the same toggle item the Chicken
checkbox uses. It is off by default, saved as `window` under `[options]` in
`starter.ini`, and adds `-window` to the launch only when on. The groups
below it move down one row.

Game (`cmd/againrom`, `pkg/ui/windowmode.go`): `-window` sets the App's
window form before `Run`. `windowPlan` is the one place both forms are
chosen:

- Whole screen (no flag, unchanged): the monitor's size; native fullscreen on
  macOS, a borderless window at the monitor's origin elsewhere; the decorated
  1280x960 menu window when the monitor reports no size.
- Window: decorated, resizable, centred, the largest whole multiple of the
  640x480 frame inside nine tenths of each monitor side. A whole multiple
  draws the menu art pixel for pixel; the tenth left over keeps the title bar
  and a taskbar or dock clear. 1920x1080 and 2560x1440 get 1280x960,
  3840x2160 gets 2560x1920, 1366x768 and 1440x900 get 640x480, a monitor
  smaller than one frame gets nine tenths of itself, and no monitor size gets
  1280x960.

Alt+Enter is read in `readAppInput` and acted on at the top of `App.step`,
before any screen's own dispatch, so it works on every screen: menus, picker,
generation, town and mission. It switches to the other form for the monitor
the window is on now. Linux and other desktops get the same engine calls; a
window manager that ignores one leaves the window as it was.

A window the player has maximized is restored before either form is applied.
Outside macOS, Ebitengine v2.10.5 ignores a size request on a maximized window,
so without the restore the whole screen kept the maximized work area and only
lost its frame. On macOS the size request works either way and a window in
native fullscreen does not report itself maximized, so the native fullscreen
path is unchanged; a zoomed macOS window is restored first like any other.

Mission map (`pkg/ui/viewer.go`): `middleDragIntent` anchors on the first
tick the middle button is held and returns the opposite of each later pointer
delta divided by the zoom, so the map point under the pointer stays under it.
`Camera.Pan` clamps it like every other scroll. It adds to the pan only while
neither the left nor the right button is held, so a right drag or a marquee
with the wheel also held moves the view once. A popup, a cutscene and a
cancelled pointer gesture clear its anchor; the hover readout waits while it
is held. The arrow keys, edge bands, right drag, minimap and wheel zoom are
unchanged.

Headless vocabulary: `HeadlessKey("alt-enter")` and the pointer actions
`middle-press`, `middle-move`, `middle-release`.

## Map editor, developer viewer and second game

- The second game runs the same `cmd/againrom` and `pkg/ui` App: the starter
  checkbox forwards `-window` for its bases, the window forms and Alt+Enter
  are the same, and its mission map drags with the middle button.
  `TestReleaseSecondMissionTenMiddleDragAndAltEnter` proves the drag and the
  toggle on the second game's mission 10.
- The map editor (`pkg/ui/mapeditor.go`) and the developer viewer
  (`cmd/mapview`) step the same `Viewer`, so the middle drag pans them too
  (`TestTheMiddleDragPansTheMapEditor`). Neither uses the game's window
  forms: the editor opens its own 1280x800 window and has no `-window` flag
  or Alt+Enter.

## Proof

Focused tests, each with a loss control:

| Test | What it pins |
|---|---|
| `cmd/starter` `TestWindowSettingDefaultsOffAndRoundTrips` | default off, load and store of `window`, no other setting moves |
| `cmd/starter` `TestWindowCheckboxIsOffAndPlayForwardsOnlyItsFlag` | one checkbox, off; `-window` only when on, for both games and both languages |
| `cmd/starter` `TestWindowCheckboxSavesAndReloads` | Save button writes it, a fresh starter reads it back |
| `cmd/againrom` `TestWindowFlagDefaultsOffAndAcceptsBooleanForms`, `TestWindowFlagReachesTheWindow` | flag parse and the form the launched App opens in |
| `pkg/ui` `TestWindowedSizeFitsTheMonitor`, `TestWindowPlanPerPlatform`, `TestStartupWindowRequestsNativeFullscreenOnMac` | both forms per platform; the next multiple must not fit |
| `pkg/ui` `TestAltEnterIsTheToggleAndNotEnter`, `TestAltEnterTogglesTheWindowOnEveryScreen` | the decision, and the toggle on menu, picker, mission and town with the screen and camera kept |
| `pkg/ui` `TestAltEnterRestoresAMaximizedWindowBeforeTheWholeScreen` | a maximized window is restored before the whole-screen form; a window that is not maximized gets the form alone; removing the restore fails it |
| `pkg/ui` `TestTheMiddleDragKeepsThePointUnderThePointer` | zoom 1 and zoom 2 |
| `pkg/ui` `TestTheMiddleDragIsClampedAtTheMapEdge`, `TestTheMiddlePressWithoutMovementDoesNothing` | clamp; no movement |
| `pkg/ui` `TestTheOtherScrollInputsAreUnchangedByTheMiddleButton` | pan keys, edge band, right drag, left drag of the developer viewer and wheel, with and without the middle button |
| `pkg/ui` `TestTheMiddleDragReachesTheMapThroughTheApp`, `pkg/game` `TestReleaseSecondMissionTenMiddleDragAndAltEnter` | the App route; the second game's world equals a hover-only session's |

## Gate results

Run on `f2fd99ed`, the code commit; the commit that adds this section changes
this file only.

| Gate | Result |
|---|---|
| focused tests; `go test` of `cmd/starter`, `cmd/againrom`, `pkg/ui`, `pkg/game`, `internal/storyguard`, `internal/gatedtests` | pass |
| `go test -trimpath -count=1 ./...` | pass |
| `CGO_ENABLED=0` build and vet of `./cmd/...`, windows/amd64 | pass |
| the same for darwin/amd64, darwin/arm64, linux/amd64 | fail inside Ebitengine v2.9.9, which needs cgo there; the sorted build and vet output is byte-identical at base `9609c508` |
| `check-release-tests.sh`, EN, headless only | 1311 gated tests; `TestReleaseActivePauseRenderedWorld` skipped by `AGAINROM_HEADLESS_ONLY`; two tavern tests failed only because their review directory lay outside `AGAINROM_REVIEW_ROOT`, and passed on rerun with it set |
| `check-release-tests.sh`, RU, headless only | 1311 gated tests; the same headless skip; no failure |
| `check-mission-scenarios.sh`, EN and RU | 76 of 76 |
| second-game SAVE selection, rom2-en and rom2-ru | 301 passed on each, none skipped, including `TestReleaseSecondMissionTenMiddleDragAndAltEnter` |
| `gofmt -l`, `git diff --check`, `scripts/check-no-game-assets.sh` | clean |
| `check-div-claims.sh` | exit 0; `DIV-3724` is listed for `R2-ENGINE-238`, whose retraction covers child operand roles and the unconditional default-size wording; the row cites only the three screen sizes, which stand |

No World hash, SAV digest or rendered frame moved: no golden file changed and
every release test and scenario that pins them passed. A launch without
`-window` opens as before; Enter acts as before unless Alt is held.

## Gate results on the merge with main

Run on `f7edc508`, the merge of main `0bc55fad` (Ebitengine v2.10.5, game
0.127.6, starter 0.9.1). Every modifier read on the merged tree goes through
`pkg/ui/heldkey`; the merge needed no further change.

| Gate | Result |
|---|---|
| `gofmt -l`, `git diff --check` | clean |
| `go test` of `cmd/starter`, `cmd/againrom`, `pkg/ui/...`, `internal/storyguard`, `internal/archtest`, `internal/gatedtests` | pass; storyguard settles at CommentBytes 9241961 |
| `go test -trimpath -count=1 ./...` | pass |
| `CGO_ENABLED=0` build and vet of `./cmd/...` for windows/amd64, linux/amd64, darwin/amd64, darwin/arm64 | pass on all four |
| `scripts/check-no-game-assets.sh` | clean |
| `check-release-tests.sh`, EN, headless only, review directories under `AGAINROM_REVIEW_ROOT` | 1311 gated tests, no failure; both tavern witnesses passed in place; `TestReleaseActivePauseRenderedWorld` skipped by `AGAINROM_HEADLESS_ONLY`, the script's only cause of exit 1 |
| the same on RU | the same result |

## Gate results after the maximized-window restore

Run on `ca91a907`, which restores a maximized window before a form switch.

| Gate | Result |
|---|---|
| `gofmt -l`, `git diff --check` | clean |
| `go test` of `cmd/starter`, `cmd/againrom`, `pkg/ui/...`, `internal/storyguard`, `internal/archtest` | pass; storyguard settles at CommentBytes 9242891 |
| `TestAltEnterRestoresAMaximizedWindowBeforeTheWholeScreen` with the restore removed | fails: `window calls ["apply"], want ["restore" "apply"]` |
| `go test -trimpath -count=1 ./...` | pass |
| `CGO_ENABLED=0` build of `./cmd/...` for windows/amd64, linux/amd64, darwin/amd64, darwin/arm64 | pass on all four |
| `scripts/check-no-game-assets.sh` | clean |
| `TestReleaseSecondMissionTenMiddleDragAndAltEnter`, rom2-en and rom2-ru, headless only | pass on both |

## Open debt

- The F1 help does not list Alt+Enter or the middle drag. `DIV-2951`'s owner
  rule lists the keys that story added; whether it extends to these is an
  owner decision.
- The window's size and position are not remembered across Alt+Enter or
  launches; each switch to the window recomputes it from the monitor.
- No run on a real macOS or Linux desktop; the per-platform forms are proved
  by the plan tests and the four-target builds only.
- The original's display mode and its library handling of Alt+Enter remain
  unread (`DIV-3724`).
