# Story `1031` — closure

As-built, after the return of adversarial pass 3. `spec.md` is the behaviour; this is the
evidence and the open items. It is not a journal: the pass-by-pass account is in git and in
`pipeline/reviews/1031-*.md`.

## The contract's Result, item by item

| Result item | Status |
|---|---|
| the eight edge arrows over the screen edges | **delivered** (`missionEdgeBands`/`missionEdgeArrow`, exhaustive table test), and shown only on ticks the edge-scroll term runs |
| the small cursors follow the view's armed mode | **delivered** for `sattack`/`scast`/`spatrol`/`smove`/`sdefault`; `sdefend` and the three modifier-key cursors are not (`DIV-261`, `DIV-262`) |
| the held-item cursor shows while an item is on the pointer | **delivered**, unchanged from story 1005/0110's `dragItemPresent`; this story adds its interaction with the manager and `pointerWanted`, and moves its draw into `drawPointer` with the other two |
| hovering a unit selects by the hostility test | **delivered** for `attack`/`select`; the original's three further gates on `attack` are not (`DIV-270`), the `town` cursor is not (`DIV-263`) |
| on the mission map, outside attack mode, the operating system's arrow is gone | **delivered** for every case B1-B4 cover, including a hostile hover with no mode up. At pass 1's landing `mapCursorPresent`'s guard read the manager's current NAME (`CurrentName() == "attack"`) rather than which code path put it there, so a hostile hover - which legitimately names `attack` - drew nothing and left the system arrow up (adversarial pass 2, F1). The guard now reads provenance (`cursorAttackFromMode`), set only by `advanceCursorManager`'s `attackShown()` branch, and the front-end witness (`TestOrdinaryHoverHidesTheSystemPointerThroughTheFrontEnd`) has a second subtest for the hostile-hover case |
| the cursor is visible, not under the screen | **delivered** at the return: all three mission pointer pictures are composed from one statement after the right column's boxes and after the in-game menu (`TestTheMapCursorIsComposedOverThePanelsAndOverTheInGameMenu`) |

## Twelve-aspect matrix

| Aspect | Result | Notes |
|---|---|---|
| Data | N/A | No new archive path, table or byte-form field. The five `.256` and eight arrow `.16a` slots this story selects among were already loaded by story 1030's registry. |
| Runtime state | PASS | Four fields added to `Viewer`: `mapCursorTex` (a texture-upload cache, the pattern `attackPointerTex` already uses), `primaryDown` (this tick's primary-button state, so `panIntent` and `mapCursorName` read one predicate), `pointerDeferred` (whether the caller composes the pointer) and `cursorAttackFromMode` (whether the manager's current "attack" name came from `attackShown()` rather than from `hoverHostilityCursor`, adversarial pass 2's F1). `MapEntity.Hostile` is a field on `MapEntity`, not on `Viewer`; it is the hostility projection B3 reads (`pkg/game/world.go`'s `entityDraws`), listed separately below and not counted here. All four `Viewer` fields are per-tick or per-session, none is persisted. |
| Simulation | N/A | No `pkg/sim` file touched. `Relations.Hostile` is read, not added; the milestone census is unchanged (measured below). |
| Player input | N/A | No press or key added. `Input.PrimaryDown` is already read by `panIntent`; this story stores it on the viewer so one predicate answers for both the pan and the arrow. |
| AI | N/A | No `pkg/sim` AI code touched. |
| UI/HUD | PASS, was GAP at pass 1's landing and again at pass 3's | `mapCursorName`/`mapCursorPresent`/`edgeScrollBands` (`pkg/ui/missioncursor.go`), wired from `advanceCursorManager` (`pkg/ui/cursor.go`), `Viewer.drawPointer` and `Viewer.DeferPointer` (`pkg/ui/viewer.go`), `flow.enter` and `flow.pointerWanted` (`pkg/ui/flow.go`) and `App.Draw` (`pkg/ui/app.go`). `mapCursorPresent`'s guard drew nothing on a hostile hover until pass 2's F1 fix (`cursorAttackFromMode`). At pass 3, `hoverHostilityCursor` and `attackTargetRect` both named an entity under a pointer resting on the right-column panel, past the map viewport's own right edge, with no gate excluding that position (P-1); a third instance of the same shape was found in the same sweep, a two-pixel gap in `minimapCaptures`'s own coverage of its column slot. All three are fixed and witnessed: `TestHoverAndAttackCursorsRequireTheMapSurfaceAtAPannedCamera` (`pkg/ui/missioncursor_test.go`, both call sites, mutation-proved against each gate in turn) and `TestMinimapCapturesTheFullColumnSlotNotJustItsOwnBox` (`pkg/ui/minimap_test.go`, mutation-proved). PASS is re-verified against the pass-3 fixes, not carried forward from pass 2. |
| Triggers/scripts | N/A | Unrelated subsystem; census unchanged. |
| Inventory/equipment | N/A | B4 reuses the existing held-item mechanism unmodified. Its draw moved into `drawPointer` and its picture and hotspot are unchanged. |
| Persistence/save-load | N/A | No field added to any save-affecting type. `MapEntity.Hostile` is rebuilt every tick from live session state, not carried. |
| Campaign/session | N/A | No campaign or session field changed shape. |
| Shipped content | PASS | The eight arrow slots, five minimap slots and the `attack`/`select` pair are all story-1030 registrations resolved against both real installs already; this story adds no new resolution and changes no shipped byte. `check-release-tests.sh` and `check-scenarios.sh` both pass on both roots. |
| Interactions with existing mechanics | PASS | `attackShown()`/`dragActive` keep first claim on the frame exactly as before (`mapCursorPresent`'s own exclusion guards, mutation-proved below). The attack target marker is NOT drawn by `drawPointer`: it is a world annotation at a world position, drawn at its own call site in `Viewer.drawFrame` (`pkg/ui/viewer.go`). Measured draw order (adversarial pass 3, D-2, correcting "composed under the right column's boxes"): panel `:3078`, minimap `:3165`, marker `:3256-3259`, doll `:3325`, controlPanel `:3373` — the marker is composed ABOVE the panel and the minimap and BELOW the doll and the control panel, over two of the four right-column layers and under the other two, not under all four. At pass 1's landing the marker's own rectangle was unclamped and could extend under the minimap/command-panel/character-panel column for a target near the world viewport's right edge (adversarial pass 2, F2); `attackTargetRect` now clips its result to the world viewport (`clipScreenRectToViewport`), scoped to the marker's own return site and not to the shared `entityPickRect`/`cellScreenRect` hit-test chain, which stays unclamped. The clip additionally insets by half of `AttackMarkerWidth` (adversarial pass 3, P-2/F2), because the draw order alone never kept the marker off the column — the clip is what does, and the returned rectangle clipped flush to the viewport bound still let the stroke paint one pixel past it. The fog gate on `hoverHostilityCursor` is classified in `fogWalkTable` (`fogGateEntity`, on `attackTargetRect`'s own precedent) and `minimapModeCursor` is classified `fogWalkNotADraw`; `TestEveryWalkOverTheEntitySnapshotIsClassified` enforces both. |

No in-scope GAP left unaddressed. This line was false at pass 1's landing: UI/HUD carried a GAP
(F1, the hostile-hover draw guard) that this matrix's own PASS verdict did not catch, because the
front-end witness at that time exercised only the empty-cell case. It was false again at pass 3's
landing: UI/HUD carried a GAP (F1, the missing positional gate on the hover cursor and the attack
marker, P-1 above) that the pass-2 witnesses did not catch, because neither exercised a camera
offset that is not a multiple of the cell size. It holds now against the panned-camera witness.

## The defects the three returns closed, and the one still open

**P-1, the one-tick position lag.** `advanceCursorManager` was called from `Viewer.step`
immediately under `advanceAnimation`, above the three statements that store the tick's cursor
position. Every branch of `mapCursorName` named a cursor for the previous tick's position while
`mapCursorPresent` drew the picture at this tick's. The call now stands below those stores and
still above the popup return, which story 1030's B4 requires: a mission-end notice sending the
flow to the map list must not leave the manager holding `attack`. Witness
`TestTheMapCursorNameFollowsThisTicksPositionAndNotTheLast`.

**P-2, the arrow on a tick that pans nothing.** `panIntent` suppressed the edge-scroll term
while the primary button is held and the cursor's own copy of the band arithmetic did not. Closed
as a class rather than as an instance: `Viewer.edgeScrollBands` is now the one predicate both read,
so no future divergence between them is possible without editing one function. It also covers a
second class of tick the report did not name - `step` returns above `panIntent` while a popup
stands - which `mapCursorName`'s popup gate closes. Witnesses
`TestTheEdgeArrowShowsOnlyOnTicksTheEdgeScrollTermRuns` and
`TestAPopupOverTheMapLeavesTheMapsOwnSelectionAnsweringDefault`.

**P-3, `attack` on an unmodified hostile hover. The premise did not hold and no production
line changed.** The report's claim was that no path writes `attack` outside the Ctrl gate. That is
true of the block `AI-CURSOR-052` reads (`L00633`-`L07916`) and false of the routine:
`EXP-0216`'s own `evidence/disasm-listings.txt` prints a second selection tree at
`L01515`-`L13182` whose `L01527` writes `L00628` - slot 3 `attack` by
`SPR16A-CURSOR-067` - with bit `0x4` set, bit `0x20` clear, no Alt and no Ctrl test on the path.
The reading taken is that tree. What changed is the documentation that had asserted
`AI-CURSOR-052` gives `attack`, which it does not: `hoverHostilityCursor`'s doc comment,
`spec.md`'s B3 and the research reconciliation below. `DIV-270` states the question, the reading
taken, the three gates of that tree this build does not reproduce, and what the other reading
would produce on shipped content.

**P-4, the cursor under the panels and under the pause menu.** A class of three sites, all
closed at once. `Viewer.drawPointer` draws the attack pointer, the map cursor and the held item
from one statement at the end of the frame; `Viewer.DeferPointer(true)`, set in `flow.enter`
beside `SetCursorManager`, moves that statement into `App.Draw` after `drawGameMenuOverMap`.
`flow.pointerWanted`'s map branch was reordered to match `drawPointer`, and its comment claiming
the two already matched was corrected. Witness
`TestTheMapCursorIsComposedOverThePanelsAndOverTheInGameMenu`, two cases.

**P-5, the hostile-hover draw guard (adversarial pass 2's F1).** `mapCursorPresent`'s fourth
exclusion read `v.cursorMgr.CurrentName() == "attack"`, on the theory that only
`advanceCursorManager`'s `attackShown()` branch could leave the manager naming `attack`. The
theory was false: `hoverHostilityCursor` (B3) legitimately names `attack` for an unmodified
hostile hover, through `mapCursorName`'s own default branch, and the guard refused to draw that
selection - the cursor drew nothing and the system arrow showed over the hostile unit. Deleting
the guard outright breaks a different, real requirement: `TestLeavingAttackModeLowersTheMapsOwnCursor`
(story 1030) covers a one-tick transitional state where `advanceCursorManager` (called inside
`v.step`) still sees the previous tick's `attackHeld` and writes `attack` through the mode branch,
before `App.step`'s later `v.setAttackHeld()` lowers it; the guard's job during that one tick is to
keep `pointerWanted` and the drawn picture agreeing that nothing is shown. The fix is a provenance
field, `cursorAttackFromMode`, set `true` only by `advanceCursorManager`'s `attackShown()` branch
and `false` by every other branch including the hover's, read by the guard instead of the name.
Both requirements hold together: `TestOrdinaryHoverHidesTheSystemPointerThroughTheFrontEnd`'s new
"hostile unit" subtest and the unchanged `TestLeavingAttackModeLowersTheMapsOwnCursor` both pass.

**P-6, the attack target marker crossing the right column (adversarial pass 2's F2).**
`attackTargetRect`'s own screen rectangle was unclamped, built from `entityPickRect` alone. A
hostile unit near the world viewport's right edge produced a marker rectangle extending past
`v.cam.ViewW`, into the space the minimap, command panel and character panel occupy. The measured
draw order (adversarial pass 3, D-2, correcting this paragraph's own earlier "composed under those
layers by draw order") is panel `:3078`, minimap `:3165`, marker `:3256-3259`, doll `:3325`,
controlPanel `:3373`: the marker is drawn above two of the four layers and below the other two, so
an unclamped straddling rectangle painted over part of the panel and the minimap and was itself
painted over by the doll and the control panel — not uniformly under all three as this paragraph
previously stated. The decision taken is to clip the marker to the world viewport rather than
reorder it in the draw sequence, which corrects the straddle regardless of which side of each
layer the marker draws on: `clipScreenRectToViewport` intersects `attackTargetRect`'s own return
value against `(0, 0, v.cam.ViewW, v.cam.ViewH)`, scoped to that one return site.
`entityPickRect`/`cellScreenRect`, the shared hit-test chain the click path and the minimap both
read, are untouched, so clipping the marker's presentation does not narrow what a click can hit
near the edge. `TestTheAttackTargetMarkerIsClippedToTheWorldViewport` (`pkg/ui/pointer_test.go`)
places a unit whose unclamped pick rectangle crosses the viewport's right edge, asserts the marker
no longer does, and asserts the same unit is still hittable through the unclamped chain.

**P-7, the hover cursor and the attack marker read the pick geometry with no positional gate
(adversarial pass 3, F1/P-1), closed as a class of three sites.** `hoverHostilityCursor` and
`attackTargetRect` both call `topAt`/`entityPickRect` with no gate of their own, unlike
`minimapModeCursor`, which is gated on `v.minimapCaptures`. `entityPickRect` is culled only when a
rectangle has NO intersection with the viewport at all (`placeArm`/`placeLifted`,
`pkg/ui/overlay.go`), so an entity whose cell straddles the map's own right edge keeps a pick
rectangle reaching past it. At the camera origin every cell boundary this build's frame size uses
is 32-pixel-aligned with the viewport's own edge and the straddle cannot occur; a camera panned by
an amount that is not a multiple of the cell size opens it. Both calls are now gated on
`v.mapSurfaceCaptures(v.cursorX, v.cursorY)`. The sweep also enumerated `decide`'s own two `topAt`
calls (`pkg/ui/command.go:478,513`), reached only through `command`'s widget-swallow sequence, and
found the sequence itself incomplete: `minimapCaptures` tested only the minimap's own drawn square
(`g.Box`, `hudMinimapReserve` wide), not its full column slot (`MissionPanelW`), leaving a
two-pixel strip along the slot's own left edge swallowed by nothing — not the map
(`mapSurfaceCaptures` answers false there) and not any of the other three right-column widgets
(their own rects start lower). `minimapCaptures` now tests the full slot via `rightColumnBox`, the
shape the other three widgets already used. Witnesses:
`TestHoverAndAttackCursorsRequireTheMapSurfaceAtAPannedCamera` (`pkg/ui/missioncursor_test.go`,
both call sites at a camera panned to X=16, cell (27,20), cursor (879,656) — the reviewer's own
reproduction) and `TestMinimapCapturesTheFullColumnSlotNotJustItsOwnBox`
(`pkg/ui/minimap_test.go`). Each gate was mutated away and the corresponding assertion observed to
fail, then reverted.

**P-8, the marker's returned rectangle is not the marker's painted span (adversarial pass 3,
F2/P-2).** `clipScreenRectToViewport` clipped the marker's rectangle flush to the world viewport
bound, but the only caller paints it with `vector.StrokeRect`, which centres a
`AttackMarkerWidth`-wide (2px) line on the rectangle's own edge rather than drawing inside it, so
a rectangle clipped flush to `v.cam.ViewW` still painted `AttackMarkerWidth/2` pixels past it —
one pixel column into the right column, at any camera position, not only a straddling one.
`clipScreenRectToViewport` now insets the bound it clips to by half the stroke's own width, so the
PAINTED span stops at the true edge. This also corrected a falsified comment in
`pkg/ui/viewer.go` claiming the marker "never strokes into the column regardless of where in the
draw order it falls" — true of the returned rectangle, false of the paint, and only the first was
checked (D-2, above, corrects the same paragraph's draw-order claim). Witness:
`TestTheAttackTargetMarkerIsClippedToTheWorldViewport` now asserts the painted span
(`got.X+got.W+AttackMarkerWidth/2`), not only the returned rectangle, and its cursor position was
moved on-map (the previous fixture placed it past `v.cam.ViewW`, which P-1's own new gate on
`attackTargetRect` now correctly excludes, and the test was failing there for that reason before
this edit). Mutation-proved twice: against the pre-fix zero-inset clip (red), and against the
pre-fix clip with `AttackMarkerWidth` also raised from 2 to 4 (red, harder).

**What the return did NOT remove: the attack-mode one-tick lag.** Moving
`advanceCursorManager` below the cursor stores fixes the POSITION lag and not the attack-mode
lag, and the reason is structural. The attack latch is raised in `App.Update` after `v.step`
returns, so `advanceCursorManager` reads the previous tick's `attackHeld` however far down `step`
it is called. `mapCursorPresent`'s own `"attack"` guard is what keeps `pointerWanted` correct for
that one tick, and `TestLeavingAttackModeLowersTheMapsOwnCursor` (story 1030) still covers it.
Closing it needs the latch moved or the manager advanced from `App.Update`, which is a different
story's decision.

## The count sweep

The return's D finding was one wrong count in `DIV-247`. Closing the class rather than the
instance means recomputing every counted population in this story's own rows and in the landing
narrative that quotes them, from the code or from the cited claim read whole, not from the
previous revision.

Sixteen counted populations were found across `DIV-247`, `DIV-249`, `DIV-259`-`DIV-263` and the
landing narrative. Fourteen were recomputed. Six of the fourteen were wrong:

| Where | Said | Is | Source recomputed from |
|---|---|---|---|
| `DIV-247` | fourteen slots reach the screen | sixteen | `cursorregistry.go`'s 28 entries against every name `missioncursor.go`, `cursor.go` and `flow.go` can produce |
| `DIV-247` | a fifteenth, `wait` | a seventeenth | same |
| `DIV-247` | the remaining thirteen | eleven, now enumerated in full | same |
| `DIV-247` | `DIV-261`-`DIV-263` name three of them | four (`sdefend`, `swarm`, `move`, `town`); `DIV-262` covers two slots | the three rows read whole |
| `DIV-262` | two modifier-key gates | three latches, all read in the block the row cites | `AI-KEYMOD-059` and `EXP-0216`'s listing |
| landing narrative | fourteen of the 28 | sixteen | same as `DIV-247` |

The first and the last are the same number in two places, so the six are five distinct defects.
`DIV-247`'s own enumeration of sixteen names was correct as written; only the number in front of
it was wrong, which is the shape a reader cannot see without counting the list.

Two populations were named here as NOT recomputed at pass 1's landing: `DIV-249`'s "about 5% ...
against 1.7%" and `DIV-261`'s "eight-entry jump table". Adversarial pass 2 recomputed both.
`DIV-261`'s count is correct against the cited claim. `DIV-249`'s was wrong, and naming a figure
as "not recomputed" is not the same claim as naming it correct - the row compared a
monitor-independent frame fraction (5%, off the map) against a monitor-dependent window-pixel
fraction on one named 1920x1080 monitor (1.7%, on the map), which are not the same measure, and it
did so while also misdescribing the mission map as an unscaled, native-resolution surface rather
than a 1024x768 frame fitted to the window the same way every other screen's 640x480 canvas is
(adversarial pass 2, F5). The corrected figures are monitor-independent on both sides: 32/640 = 5%
off the map, 32/1024 = 3.1% on it, a fixed ratio of 0.625. One further correction was found by the
sweep and is not a count: `DIV-262` said Ctrl forces `swarm`, where `AI-CURSOR-052` reads Ctrl
held as `mask & 3` selecting `attack` and its absence selecting `swarm`. That is the claim's
second arm read as its only one.

## Integration witness

Re-run at pass 3's return, against the branch tip (`c4024ea8`, which also carries a merge of
`origin/master` at `b02632b1` — story 1033 landed while this story was in review) rather than
repeated from pass 2's own numbers.

- `pipeline/check-release-tests.sh`, `AGAINROM_IMPL=<worktree>`: **selected 41** install-gated
  tests (38 `AGAINROM_ASSETS`, 1 `AGAINROM_ORIGINAL_SAVES`, 2 `AGAINROM_SAVE_666`); 41 of 41 ran
  and passed, 0 skipped, on `gameversions/en` and again on `gameversions/ru`. Unchanged from pass
  2: this round adds no install-gated test, because P-1/P-2 are observable only through synthetic
  fixtures, not through a real install.
- `pipeline/check-scenarios.sh`, same override plus `AGAINROM_ASSETS=<root>`: **14 of 14** on
  `gameversions/en` and 14 of 14 on `gameversions/ru`. Unchanged from pass 2. These load original
  saves and drive real campaign missions headless; they are the integration witness for the
  selection half of the story and they do not reach `App.Draw`, so they say nothing about the
  composition half.
- `go build ./...`, `go vet ./...`, `gofmt -l` (empty), `go test -count=1 -trimpath ./...`: clean
  in both repositories, **41 packages ok** in `implementation` (unchanged package count; the new
  panned-camera and slot-coverage tests are new functions in existing test files), 7 test packages
  ok in `research`. Every `scripts/check-*.sh` by glob: implementation `check-claim-citations`
  (**1243** distinct citations resolving against 1453 claims, 216 experiments, 786 prefixes - one
  more than pass 2's own closure recorded, from story 1033's merge, not from this round's own
  doc-comment edits, which cite no id the tree did not already cite) and `check-no-game-assets`
  clean; research `check-claim-ids` (1453 ids, 31 ledgers, 29 read back through `tools/claim`),
  `check-regen-out` (selected 7, all honour `OUT`), `check-retraction-status` (232 overturned, all
  marked) - unchanged from pass 2: this round touches no file under `research/`, and the pin is
  frozen at `753034d`.
- `pipeline/check-div-claims.sh`, `AGAINROM_IMPL=<worktree>`: exit 0, printed the tree it read
  (`wt-1031` at `c4024ea8`, research pin `753034d`); **176 live rows of 176**, citing **253**
  distinct claim ids (down from 256 at pass 2's own closure - the merge with story 1033 changed
  which rows the ledger carries; this round adds no divergence row and cites no new claim id: P-1,
  P-2 and the minimap slot fix are defects in this build's own code, not ROM1 mismatches, and none
  of the three new tests cites a claim), 0 closed, none retracted.
- `pipeline/check-milestone.sh` with `AGAINROM_MILESTONE_DRIVE` pointed at a `missionrun` built
  from this branch: exit 0, *"the script gap and the drive are where they were recorded, both
  roots"*. Missions 10 and 20 measured directly with `missionrun -mission N -trace -ticks 1 |
  grep -c UNSUPPORTED` on `gameversions/en`: **0 and 0**, the values
  `pipeline/milestone-baseline.txt` carries and unchanged since pass 1's own measurement. This
  round is a positional gate on the cursor/marker pick and a stroke-width inset; it touches no
  script-node handling. The story's result is therefore in `builds/current/` and not in the
  census.

**What the composition witness can see, and what it cannot.**
`TestTheMapCursorIsComposedOverThePanelsAndOverTheInGameMenu` replaces three package-variable
draw seams - `blitColumnLayer`, `blitPointerLayer`, `blitMenuOverMap` - and records the ORDER of
the calls production makes through `App.Draw`. It observes a real ordering produced by production
code and it does not observe pixels: `ebiten` composites nothing without a running game loop, so
no test in this repository can read back what covers what. The claim the test supports is that
the map cursor's draw call comes after every right-column box's and after the in-game menu's, and
that is the whole claim - **but at pass 1's landing the first case's own fixture composed only one
of the four column layers** (`minimap`), so the case could not have witnessed the claim it stated
for `panel`, `doll` or `controlPanel` (adversarial pass 2, F3). The fixture now installs a font, a
selection, an equipped pack and command-panel art so all four layers compose, and the case asserts
the map cursor's index against all four rather than against whichever layers happened to render.

**Not witnessed: no live game window was driven this session.** The owner's desktop is in use and
three other lanes are running on this machine, so no synthetic keystroke or click was sent and no
screenshot was taken. The owner's own 2026-08-22 report is the observation the fix answers; this
lane did not reproduce it on screen and does not claim to have. What was reproduced is the source
condition he named: three draw sites in two orders, and two layers composed over them, all read
from the code and all now composed from one statement.

**Mutation proofs, the return's own new code.** Each mutation was applied, the named test run and
seen to fail, the mutation reverted with the same edit pair that applied it, and the test and
`gofmt` re-run clean. The case-1 row was false at pass 1's landing: the case passed unchanged
against every mutation tried, including a full revert of the composition fix, because its fixture
composed only one of the four column layers (F3, above). It is re-verified here against the fixed
fixture.

| Mutation | Test killed |
|---|---|
| `advanceCursorManager` moved back above the cursor stores in `step` | `TestTheMapCursorNameFollowsThisTicksPositionAndNotTheLast` |
| `edgeScrollBands`'s `primaryDown` guard removed | `TestTheEdgeArrowShowsOnlyOnTicksTheEdgeScrollTermRuns` |
| `mapCursorName`'s `popupOpen()` gate removed | `TestAPopupOverTheMapLeavesTheMapsOwnSelectionAnsweringDefault` |
| `drawPointer`'s call moved back above the panel block in `drawFrame`, against the fixed fixture | `TestTheMapCursorIsComposedOverThePanelsAndOverTheInGameMenu`, case 1 |
| Same, with the `!v.pointerDeferred` guard also removed so the call runs there | same test, case 1 |
| `App.Draw`'s `drawPointer` call moved above `drawGameMenuOverMap` | same test, case 2 |
| `mapCursorPresent`'s `cursorAttackFromMode` guard forced to always return "nothing to draw" | `TestOrdinaryHoverHidesTheSystemPointerThroughTheFrontEnd`, "hostile unit" subtest |
| `attackTargetRect`'s viewport clip removed (return the unclipped rectangle) | `TestTheAttackTargetMarkerIsClippedToTheWorldViewport` |
| `hoverHostilityCursor`'s `mapSurfaceCaptures` gate removed (pass 3, P-1) | `TestHoverAndAttackCursorsRequireTheMapSurfaceAtAPannedCamera` |
| `attackTargetRect`'s `mapSurfaceCaptures` gate removed (pass 3, P-1) | same test |
| `minimapCaptures` reverted to `g.Box` alone, no `rightColumnBox` (pass 3, P-1) | `TestMinimapCapturesTheFullColumnSlotNotJustItsOwnBox` |
| `clipScreenRectToViewport`'s stroke inset zeroed (pass 3, P-2) | `TestTheAttackTargetMarkerIsClippedToTheWorldViewport`, painted-span assertion |
| Same, with `AttackMarkerWidth` also raised from 2 to 4 (pass 3, P-2, the reviewer's own instruction) | same assertion, reds harder (painted right edge 202 against ViewW 200) |

The earlier round's proofs stand and are unchanged: `mapCursorPresent`'s top guard forced to
"nothing to draw" kills `TestOrdinaryHoverHidesTheSystemPointerThroughTheFrontEnd`, and
`mapCursorName`'s B1 branch disabled kills `TestMapCursorNamePrecedenceEdgeBeatsMinimapBeatsHover`.
`missionEdgeArrow` and `missionEdgeBands` are not separately mutated: the table test is exhaustive
over the 3x3 band grid on both axes plus the four out-of-window positions and both degenerate
sizes, so any one-line change to either function changes at least one row's expected output.

## Research reconciliation

`AI-CURSOR-190`'s eight-arrow direction map and screen-position test are implemented as decoded,
except the margin: the window's own edge-scroll band is substituted for the original's asymmetric
1px/2px screen-pixel bands, because this build's mission frame is a different size from the other
screens' (1024x768 against 640x480, `DIV-249`) and there is no single pixel space to transcribe
the original's own bands into; recorded in `spec.md` and not a new divergence row.
`AI-CURSOR-207`'s precedence (edge before the widget/mode tree) is implemented.
`AI-CURSOR-202`/`AI-CURSOR-203`'s `sdefault` forcing is implemented. `AI-CURSOR-208`'s widget is
not read at instruction level and this build's own widget stands in (`DIV-260`).
`AI-CURSOR-209`'s `town` condition is read at instruction level, inside the same
`L01515`-`L13182` tree the `attack` write below is read from, and not implemented, its
game-side meaning being the row's own stated Unknown (`DIV-263`). `AI-CLICK-050` was not used, per
its own partial retraction and the contract's exclusion.

**The hover cursor, restated after the two returns.** `UNIT-HOVER-020` gives the hover mask's five
bits. `AI-CURSOR-052` turns that mask into a cursor in ONE block, `L00633`-`L07916`, and in
that block bit `0x4` only suppresses `select`. The earlier text of `spec.md` and of
`hoverHostilityCursor` read that as evidence for `attack`, which it is not. `attack` for an
unmodified hostile hover comes from the second selection tree in the same routine, printed in
`EXP-0216`'s committed evidence. The five addresses that gate the `attack` write itself
(`L01524`, `L01525`, `L13183`, `L01527`, `L01526`) are cited by no published claim at pin
`753034d` (`go run ./tools/claim -k` returns no row for any of the five). `AI-CURSOR-209` (Medium)
covers a different part of the same tree, `L01682`-`L01410` and `L01412`-`L01411`,
for the `town` pick's own entry test and refinement; those instructions reconverge with the
`attack` write's own path before the hostility test and do not gate it. This closure previously
stated "no published claim covers this tree" without that distinction, which pass 2's F4 corrected
(`provenance.md` carries the full trace). Two further observations about the cited claims, both
for research to take or leave rather than for this lane to assert as corrections:

- `AI-CURSOR-052`'s "appears only inside `mask & 0x24`" is scoped to the block it read. The same
  routine reads bit `0x4` at `L00634` and at `L00635`, and the second selects `attack`.
- The `select` write at `L00637`, inside that block, is gated on `[L00670]`, which
  `AI-KEYMOD-059` gives as the Shift latch. `AI-CURSOR-052` does not name it, so the block's
  no-modifier outcome is "no cursor written", not `select`.

`UNIT-VPLAYER-021`'s retracted first reading (`claims/retracted.md:277`) was not the version used;
`provenance.md` documents reading the active amended row (`claims/unit.md:38`) and confirming its
own Medium cap on the view-side-row-to-session-matrix link, which `DIV-259` carries forward rather
than resolving by assertion.

`DIV-247` is narrowed at this landing: sixteen of the 28 registered cursor slots now reach the
screen, up from three. Its row text, its remaining-slot enumeration and the landing narrative that
quotes it are corrected in place.

## Divergence ids

Allocated to pass 1's return: `DIV-270` through `DIV-273`. Three spent, one **returned unused**.
Allocated to pass 2's return: `DIV-280` through `DIV-282`. **All three returned unused**: F1, F2
and F3 are defects in this build's own code and tests, not ROM1 mismatches, and F4/F5 are
corrections to the existing rows named above, made in place rather than as new rows.

- `DIV-270` — the hostile-hover `attack` cursor selected without the original's three further
  gates (`view+0x140`, `view+0x144 & 0x24`, hover mask bit `0x20`). FIDELITY-DEBT. Corrected at
  pass 2 (F4): a claim covers part of the same tree (`AI-CURSOR-209`), not none of it; the row's
  basis and revisit condition are rewritten, not closed.
- `DIV-271` — the pointer composed last, over every panel and over the in-game menu, on an
  authored order. UNKNOWN: nothing decoded says where the original composes its cursor, or
  whether it composes one at all rather than handing it to the system cursor.
- `DIV-272` — the mission cursor answers `default` while a popup stands, where `AI-CURSOR-193`'s
  entry gate makes no change. DEVIATION, with story 1030's B4 as the stated reason.
- `DIV-273` — **returned unused.** It was held for the third modifier latch; that belongs to
  `DIV-262`'s own subject and is corrected into that row in place rather than given a row of its
  own.

`pipeline/next-div-id.sh` was run before and after the ledger edits and answered `DIV-274` both
times, from the reservation in `PIPELINE-STATUS.md`. The script reads the seat's own checkout and
not this worktree, so it cannot see the three new rows; the reservation is what holds the number
either way, and every id spent here is inside the allocated range.

## Open items

- `DIV-270`: a claim naming which of the two selection trees an ordinary hover reaches (open in a
  research lane at this landing). `view+0x140`, `view+0x144` bit `0x1` and hover mask bit `0x20`
  are already given in game terms, by `AI-CURSOR-209`/`AI-PANEL-061` and `UNIT-HOVER-020`
  respectively - `DIV-270`'s revisit condition asked for these redundantly until pass 2's F4
  corrected it. Until the open question is answered, this build shows `attack` on a hostile hover
  in cases the original shows `select`, the largest being "nothing selected yet".
- `DIV-271`: whether the original draws its mission cursor itself or hands it to the system
  cursor, and where in its own frame it draws it if it does.
- `DIV-272`: what the original's cursor shows while a modal is up on the mission surface.
- `DIV-260`: a claim reading `AI-CURSOR-208`'s widget rectangle at instruction level.
- `DIV-261`/`DIV-262`: a `sdefend` armed state and a Ctrl/Shift/Alt modifier reading are each a
  small, separable follow-on; neither was in this story's own five behaviours.
- `DIV-263`: `R0218`'s own bits and the `town` condition's game-side meaning.
- `DIV-248` (story 1030's own open row, unaffected by this story): the `wait` cursor is still set
  and never composed, for the same synchronous-entry reason.
- The attack-mode one-tick lag, described above. Not a divergence row: it is a defect of this
  build against itself, with an existing test covering the guard that hides it.
