# Story `1030` — spec (as-built)

This document is self-contained: no claim citation stands in place of stating what this build does.
Claim provenance is `provenance.md`.

## Package placement

The 28-slot table's own resolution (art bytes to premultiplied pictures) lives in `pkg/game`
(`pkg/game/cursorregistry.go`, `LoadCursorRegistry`), on `LoadAttackPointer`'s own precedent
(`pkg/game/cursor.go`): `pkg/game` is the tier `internal/archtest` lets reach both the formats tier
(`pkg/formats/spr16`, `pkg/formats/spr256`) and the drawing tier (`pkg/ui`) at once, and what crosses
the boundary is resolved pictures, never a container or a decoder.

The registry's own type (`CursorSlot`, `CursorRegistry`) and the manager (`CursorManager`) live in
`pkg/ui` (`pkg/ui/cursorstate.go`), which imports only `image`. The upload cache both draw sites
share is `pkg/ui/cursortexture.go`. The screen-to-cursor table, the one screen assignment site and
the pointer-mode decision are `pkg/ui/flow.go`; the engine calls are `pkg/ui/app.go` and
`pkg/ui/viewer.go`.

`internal/archtest`'s `TestLiveTreeClean` passes with this shape: `pkg/game` already carries
`pkg/render, pkg/render/, pkg/ui` in its allow-map from story `0080`, and `pkg/ui`'s own allow-map is
unchanged.

## B1 — the 28-slot registry, and where its data reaches the screen

`LoadCursorRegistry(src terrain.EntrySource) (*ui.CursorRegistry, error)` reads the 28 rows below,
in slot order, resolves each to a decoded picture, and returns one registry or one error. A failure
on any one slot fails the whole load, naming the slot and the address it failed at, on
`LoadAttackPointer`'s own rule. `NewFrontEnd` carries that error rather than returning it
(`FrontEnd.CursorRegistry`, `FrontEnd.CursorRegistryErr`), so an install whose cursor art will not
decode starts exactly as it did before this story, with the operating system's arrow on every
screen. `FrontEnd.CheckLine` reports the error when there is one, on `FontErr`'s and
`AttackPointerErr`'s own rule: a headless run is the one place a non-fatal load failure can be said.

Authority for slot order and names: `SPR16A-CURSOR-067`. Authority for each registration's art path,
hotspot and period: `SPR16A-CURSOR-046` (23 `.16a` rows) and `SPR256-CURSOR-046` (5 `.256` rows,
slots 18–22). Frame count is not an executable constant and is not transcribed here: it is read at
decode time from the sheet the row's path names (G2 — see below).

| Slot | Name | Path (under `graphics/`) | Format | Hotspot | Period (ms) |
|---|---|---|---|---|---|
| 0 | default | `cursors/default/sprites.16a` | .16a | (5,5) | 2000000000 |
| 1 | move | `cursors/move/sprites.16a` | .16a | (15,15) | 100 |
| 2 | swarm | `cursors/swarm/sprites.16a` | .16a | (21,21) | 100 |
| 3 | attack | `cursors/attack/sprites.16a` | .16a | (3,3) | 100 |
| 4 | defend | `cursors/defend/sprites.16a` | .16a | (15,13) | 100 |
| 5 | select | `cursors/select/sprites.16a` | .16a | (3,4) | 100 |
| 6 | patrol | `cursors/patrol/sprites.16a` | .16a | (8,25) | 100 |
| 7 | cast | `cursors/cast/sprites.16a` | .16a | (15,15) | 100 |
| 8 | pickup | `cursors/pickup/sprites.16a` | .16a | (12,13) | 66 |
| 9 | arrow0 | `cursors/arrow0/sprites.16a` | .16a | (15,5) | 2000000000 |
| 10 | arrow4 | `cursors/arrow4/sprites.16a` | .16a | (16,25) | 2000000000 |
| 11 | arrow6 | `cursors/arrow6/sprites.16a` | .16a | (6,16) | 2000000000 |
| 12 | arrow2 | `cursors/arrow2/sprites.16a` | .16a | (25,15) | 2000000000 |
| 13 | arrow7 | `cursors/arrow7/sprites.16a` | .16a | (8,9) | 2000000000 |
| 14 | arrow5 | `cursors/arrow5/sprites.16a` | .16a | (8,23) | 2000000000 |
| 15 | arrow1 | `cursors/arrow1/sprites.16a` | .16a | (22,8) | 2000000000 |
| 16 | arrow3 | `cursors/arrow3/sprites.16a` | .16a | (23,22) | 2000000000 |
| 17 | sdefault | `cursors/sdefault/sprites.16a` | .16a | (2,2) | 2000000000 |
| 18 | smove | `cursors/smove.256` | .256 | (0,0) | 2000000000 |
| 19 | sattack | `cursors/sattack.256` | .256 | (0,0) | 2000000000 |
| 20 | sdefend | `cursors/sdefend.256` | .256 | (0,0) | 2000000000 |
| 21 | spatrol | `cursors/spatrol.256` | .256 | (0,0) | 2000000000 |
| 22 | scast | `cursors/scast.256` | .256 | (0,0) | 2000000000 |
| 23 | cantput | `cursors/cantput/sprites.16a` | .16a | (38,36) | 2000000000 |
| 24 | town | `cursors/town/sprites.16a` | .16a | (16,16) | 2000000000 |
| 25 | dice | `cursors/dice/sprites.16a` | .16a | (16,16) | 100 |
| 26 | wait | `cursors/wait/sprites.16a` | .16a | (16,16) | 100 |
| 27 | backpack | `cursors/backpack/sprites.16a` | .16a | (16,16) | 100 |

**The eight arrow slots (9–16) are not in arrow-number order**: arrow0, arrow4, arrow6, arrow2,
arrow7, arrow5, arrow1, arrow3. Transcribed exactly as `SPR16A-CURSOR-067` gives it, not sorted.

**Period is registered independently of frame count** (G2, `SPR16A-CURSOR-061`): `pickup` (66ms) and
`dice`/`wait` (100ms) share the fastest periods with several single-frame slots, and eighteen of the
28 rows carry a period (2×10⁹ ms) no session reaches a second advance under. Which of the 28 rows
decode to more than one frame is not stated in this table — it is read from the sheet at load, per
G2 below — and must not be inferred from the period column.

`.16a` frames are decoded through `spr16.DecodeA(b, true)` and painted per `LoadAttackPointer`'s own
per-cell rule: a painted cell's colour is the palette entry, its coverage `(level+1)/16` premultiplied
into alpha; an unpainted cell is no pixel. `.256` frames are decoded through `spr256.Decode(b)`: an
opaque cell is the palette entry at full alpha, a transparent cell is no pixel — no coverage field
exists in this format.

### The hotspot places the picture

Each registration's hotspot is the pixel within its own frame that names the cursor point. Both draw
sites subtract it: the picture's top-left goes at the cursor position **less** the hotspot, so the
hotspot pixel lands on the pixel a click names.

- Off the map, `flow.cursorPresent(cx, cy, scale)` returns the picture and that origin, and
  `App.drawCursor` draws it. The subtraction is scaled by the same factor the picture is (below).
- On the map, `Viewer.attackPointerPresent` returns the same origin for the manager's own `attack`
  frames. For the story-0080 fallback picture, which arrives through `SetAttackPointer` with no
  registration behind it, the offset is `ui.AttackPointerHotspot` — `(3,3)`, the `attack`
  registration's own hotspot argument. That constant replaces an authored choice this file's earlier
  revision defended ("that hotspot is OURS: nothing read gives one"), which was true when it was
  written and is not now: where an authored value stands in for a decoded fact and no owner
  directive drove the difference, the decoded value wins (owner, 2026-08-21). The authored cross
  drawn when no art resolved keeps the cursor position itself, which is its centre.

### The picture is scaled off the map and 1:1 on it

`App.drawCursor` scales the picture, and the hotspot offset, by `a.place.Scale()` — the same factor
the 640x480 canvas is blown up by. The pointer then occupies the same fraction of that canvas as the
original's does of its own 640x480 screen. Drawn at 1:1 window pixels, as it was before, it is about
a third of that on a 1920x1080 monitor.

The mission map's pointer stays at 1:1. That screen is not a scaled 640x480 canvas: it fills the
window at native resolution and shows more world rather than the same world larger. `DIV-249` carries
both halves, including the part that remains a difference — relative to the world drawn behind it,
this build's map pointer is smaller than the original's.

## B2 — one manager, one current cursor

`ui.CursorManager` (`pkg/ui/cursorstate.go`) is the one current-cursor state. One instance is owned
by `flow` (`pkg/ui/flow.go`, field `cursor`, never nil after `newFlow()`), shared by `App` and handed
to every map `Viewer` through `Viewer.SetCursorManager` at `enter()` time. A standalone `Viewer` (for
example `cmd/mapview`) is handed none and keeps `cursorMgr == nil`, which every method on it treats
as "no manager," preserving every pre-story caller's behaviour unchanged.

`SetCursor(name)`:
- copies the named registration's `FrameCount` and `PeriodMillis` into the manager and zeroes the
  frame index and the last-tick field;
- is a no-op when `name` is already the current cursor (`AI-CURSOR-193`'s idempotence guard);
- is a no-op when `name` is not in the installed registry, or no registry is installed yet — the
  same "leaves the displayed cursor unchanged" behaviour B4 relies on for a caller that sets no
  cursor at all.

Nothing outside `SetCursor` writes the current-cursor field. No draw path, no screen transition, no
input handler clears it directly.

## B3 — the animation counter, and every frame it reaches

`CursorManager.Advance(nowMillis)` moves the frame index forward once `nowMillis - lastTick` exceeds
the current slot's `PeriodMillis`, wraps the index to 0 before any reader can observe it at or past
`FrameCount`, and does nothing for a slot with `FrameCount <= 1`.

**The first `Advance` after a `SetCursor` advances.** `SetCursor` zeroes `lastTick`, and this call
subtracts from zero, which is the decoded shape: `R0321` zeroes `mgr+0x30` on every set and
`R0324` then compares `now - mgr+0x30` against the period, so the compare passes on the first
evaluation. `AI-CURSOR-172` states the consequence — a single set-cursor call can only be observed to
leave the frame index at 0 or 1. An earlier revision of this build recorded a baseline on that first
call instead, holding the frame at 0 for one extra period, and said in this document that it mirrored
the decoded routine. Every one of the nine multi-frame registrations carries a period of 66 or 100 ms,
far below any clock reading this build passes in, so the dependence on the clock's magnitude is not
reachable on shipped data.

`App.step` calls `flow.cursor.Advance(now.UnixMilli())` once per frame, ahead of every other
per-frame concern. `Viewer.step` calls `advanceCursorManager`, which sets the map's own cursor —
`attack` while `attackShown()`, `default` otherwise — and then advances the same shared manager. The
selection is one frame behind the mode, because the clock call must stand above the map arm's own
popup return and the attack mode is raised below it; neither edge is visible, since the picture drawn
on the raising frame is frame 0 of the same sheet and nothing at all is drawn on the lowering frame.
Lowering to `default` matters on a path that does not involve the mode at all: a mission that ends
raises a notice, and the flow then reaches the map list, which runs no transition — a manager left on
`attack` drew the animated sword over the list of maps.

**Every advanced frame reaches the screen.** `cursorTexture` (`pkg/ui/cursortexture.go`) is the
upload cache both draw sites use, and it rewrites the engine texture when the source picture differs
from the one it holds. The viewer's own upload was previously gated on a flag set once per viewer at
map open and cleared by the first write, so the manager advanced the index, `attackPointerPresent`
returned frames 1..9, and none of them was ever uploaded: the player saw frame 0 standing still for
the whole session.

**The authored `attackCursorFrame` constant is retired.** `pkg/game/cursor.go` now defines
`attackFrame0 = 0`, documented as the single static-frame accessor `LoadAttackPointer`'s own picture
uses; the animated path is `CursorManager`'s, not a second counter.

## B4 — the surface transition rule, and the one owner of the system pointer

`flow.surfaceTransition(exit string)` sets `wait` then `exit` in one call, mirroring the pattern
`TOWN-372` gives for eleven of seventeen original routines (ten `wait`/`default`, the main menu
`wait`/`select`) and for the one that sets `default` only.

**`flow.setScreen` is the one site in this package that assigns `f.screen`**, and it runs the
transition for the screen being arrived at, from one table (`flow.screenExitCursor`). That is the
whole of the mapping, and it is what makes the cursor a property of the screen rather than of the
path taken to it. The population it replaced was fifteen assignments over `flow.go`, `save.go` and
`town.go`, three of which restored a remembered screen without any transition at all: Escape from
character generation restores `f.chargenBack`, Escape from the load window restores `f.loadBack`, and
the in-game menu's exit restores `f.menuBack`. The main menu reached by backing out of generation
drew `default` instead of `select`. `TestScreenIsAssignedOnlyBySetScreen` scans this package's own
production source and fails on a new write to the field in any of four syntactic forms: assignment,
composite literal, increment, and taking the field's address. It is not closed against a write
through a pointer obtained where the scan cannot follow, or from outside package `ui`; the field is
unexported and the address form is what would produce such an alias here.

`newFlow`'s own `&flow{screen: ScreenMenu, ...}` is the one composite literal, allowed in the scan by
name. It runs no transition, which is correct rather than an exemption: no registry is installed at
that moment, every `SetCursor` is a no-op, and `App.SetCursorRegistry` runs the current screen's
entry when one arrives.

| This build's screen | Exit cursor | Original pattern stood in for |
|---|---|---|
| Main menu | `select` | `MENU-CURSOR-046`'s own routine, matched exactly (`wait`/`select`) |
| Character generation | `default` | `TOWN-372`'s chargen routine at `R0909` (`music\chrgen.wav`) |
| Mission map | `default` | `TOWN-372`'s map routine at `R1317` (`music\map.wav`) |
| Town | `default` | stands in for **four** routines `TOWN-372` names individually by address and music path — shop `R1315`, inn `R1318`, school `R1319`, town square `R1320` — one of whose siblings, `L08084` (`music\inn_ssi.wav`), sets no cursor at all |
| Picker, Load, in-game menu | none | authored: consistent with the five-routine "sets no cursor" family, not a claim of correspondence to any one of the five |

The step from a pushed music path to a surface identity is Medium in `TOWN-372` and
`MENU-CURSOR-046`, and this build's screen set was authored before this story, so the mapping is
authored throughout. `DIV-245` carries it, including the town's one-for-four.

`App.SetCursorRegistry` runs the current screen's own entry from the same table when a registry
arrives. Every `SetCursor` before that point is a no-op, so without it the screen showing at install
time — always the boot menu in the shipped front-end — would carry no cursor at all.

**The wait cursor is never drawn.** Both sets land in one statement, and this build's surface loads
run inside a single `Update` call, so the engine composes no frame between them. No arrangement of
the two sets inside a synchronous entry can put a `wait` frame on screen. The ten frames of
`cursors/wait/sprites.16a` are decoded, resolved and never displayed. `contract.md`'s Result names
this cursor as pointable in `builds/current/` and it is not delivered; `DIV-248` is the row.

### One owner of the system pointer

The engine's cursor mode is one piece of global window state, and this build draws a pointer of its
own from two places: `App.drawCursor` on every non-map screen, and `Viewer.Draw` for the map's attack
pointer. Each used to keep its own "last told" cache and neither saw the other's writes. On the
ordinary new-game path the engine was left Hidden on the way out of the town, `App.Draw` returns
inside its `mapShowing()` branch before `drawCursor`, and the viewer's own cache already read "not
hidden" — so no call was made and the mission map had **no pointer at all** until the player toggled
attack mode on and off once. After that toggle the mirror state was reachable: the engine Visible,
the manager's cache still Hidden, and two pointers on screen in the town.

As built:

- `flow.pointerWanted()` is the one decision, for the whole window: on the map, whether the viewer
  draws an attack pointer this frame; off it, whether the manager has a picture.
- `flow.syncPointerMode()` brings the one cache — the shared manager's — to that answer and reports
  whether the engine must be told.
- `App.applyPointerMode()` makes the call, once per frame, **above** the branch in `Draw`, so it runs
  whichever screen is showing and whether or not the canvas has been placed yet. It is the only
  `ebiten.SetCursorMode` call an App session makes.
- `Viewer.Draw` makes the call itself only when `v.cursorMgr == nil`: the standalone developer viewer
  has no App above it and no second writer, and its frame is unchanged down to the calls it makes.

The invariant is the one `pointerModeChange`'s own header states and story `0080` was built for: a
frame that hides the system pointer and draws nothing is unreachable, and so is a frame that draws
one and leaves the system pointer up.

## G2 — the table seam

`cursorRegistrations` (`pkg/game/cursorregistry.go`) is a 28-row Go table, not 28 literals scattered
through the loader. Frame count is never asserted in it — it is `len(sprite.Frames)` from the decode
— so lifting the slot count past 28, changing a period, or changing a hotspot touches only this
table and changes no byte of any shipped file. The seam the contract's G2 section asks for is this
table.

## What is witnessed, and by what

| Behaviour | Witness | Mutation it was proved against |
|---|---|---|
| every path to a screen runs that screen's transition | `TestEveryPathToAScreenRunsThatScreensTransition` (`pkg/ui/cursorlifecycle_test.go`) | `setScreen` stops running the table |
| no path assigns a screen without one, in any of four write forms | `TestScreenIsAssignedOnlyBySetScreen` | one path restored to `f.screen = f.chargenBack`; a second composite literal setting the field |
| the scan's four arms each see the form they close | `TestTheScreenWriteScanSeesEveryFormItCloses` | each arm of `screenWrites` deleted in turn, over a fixture carrying one site of each form |
| the boot screen's transition runs when the registry arrives | `TestTheBootScreensTransitionRunsWhenTheRegistryArrives` | the `SetCursorRegistry` wire cut |
| the system pointer is hidden exactly when this build draws one | `TestTheSystemPointerIsHiddenExactlyWhenThisBuildDrawsOne` | `pointerWanted`'s map arm deleted |
| leaving attack mode lowers the map's own cursor | `TestLeavingAttackModeLowersTheMapsOwnCursor` | the lowering arm removed |
| the picture is placed by its hotspot | `TestCursorPictureIsPlacedByItsHotspot`, `TestThePointerIsDrawnExactlyWhileTheModeIsUp` | `cursorPresent` and the story-0080 arm of `attackPointerPresent` return the cursor point |
| the map's animated pointer is placed by the registration's own hotspot, on the arm the game runs | `TestTheMapsAnimatedPointerIsPlacedByTheRegistrationsOwnHotspot` | the manager arm of `attackPointerPresent` dropping the hotspot, written so it compiles; the manager arm removed altogether |
| 28 registrations name 28 distinct sheets | `TestCursorRegistrationsNameOneSheetEach` (`pkg/game/`) | one row's path repointed at another row's |
| every animation frame reaches the upload | `TestEveryAnimationFrameReachesTheUpload` | `cursorTexture.stale` becomes the old fresh flag |
| the first advance after a set advances | `TestCursorManagerSetCursorResetsFrameAndClock` (`pkg/ui/cursorstate_test.go`) | the baseline suppression restored |
| all 28 slots resolve from a real install, no two to the same art | `TestReleaseCursorRegistryResolvesAllTwentyEightSlotsFromTheRealInstall` (`pkg/game/`) | one wrong hotspot row in `cursorRegistrations`; the loader silently reading `default`'s sheet for `select` |

The last is install-gated and runs under `pipeline/check-release-tests.sh` on both roots. Its
expected values are transcribed from the claim rows, not read from `cursorRegistrations`, so a wrong
row in that table fails it.

## Cut from this story

The mission map's own cursor selection (arrows, small minimap cursors, held-item cursor, hostility
test, click-order rule) is story `1031`, per the contract. This build's map screen sets `default` on
entry and while no mode is up, and `attack` while attack mode is up; no other of the 28 slots has a
screen that selects it in this tree. `DIV-247` names this a gap, not a difference — the selecting
logic is decoded (`AI-CURSOR-190`..`193`) and deferred, not absent from research.

## Open items carried into `closure.md`

- Whether the counter is ever observed to advance past frame index 1 in play is Medium in both
  `AI-CURSOR-172` and `SPR16A-CURSOR-061` (`DIV-246`).
- The `wait` cursor is set at every transition and never drawn (`DIV-248`).
- The map pointer is drawn at 1:1 while every other screen's is scaled with the canvas (`DIV-249`).
