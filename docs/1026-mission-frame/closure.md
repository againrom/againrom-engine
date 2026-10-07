# Story `1026` — closure

As-built evidence for `spec.md`. This is a record, not a journal.

**This story closes, combined with story `1023`'s own landing in the same push.** `1026` alone left
one gap: the authored mission HUD did not fit a 768-pixel-tall frame, and two headless scenarios
failed on both roots. `1023` builds the right column at its decoded geometry instead of the authored
one, which fits, and both scenarios pass. `DIV-213` closes at that landing (moved to
`DIVERGENCES-CLOSED.md`); this document's aspect matrix and integration witness are updated below to
reflect the closed state rather than left describing the gap this story shipped with in isolation.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | N/A | No format, archive entry or on-disk structure is read or written differently. |
| Runtime state | PASS | `Viewer` gains `frameW`, `frameH`, `place` and `canvas`. None is persisted and none is read by `pkg/sim` or `pkg/game`. |
| Simulation | N/A | The viewport is a camera property. `pipeline/check-milestone.sh` is byte-identical before and after, both roots. |
| Player input | PASS | Three doors, witnessed at five window sizes by `TestMissionHitTestsHoldAtEveryScale`, with a control that fails if the mapping is the identity. Population below. |
| AI | N/A | Nothing here is read by an AI path. |
| UI / HUD | PASS | `DIV-213`, CLOSED at `1023`'s combined landing. The right column now builds at the decoded 160x158/80/242 geometry (`1023`), which fits the 768-pixel frame this story establishes; `dollBoxRect` and `wornBoxRect` draw the doll and refuse the worn box by `1023`'s own authored choice (`DIV-200`), not by a frame that is too short. |
| Triggers / scripts | N/A | The census over 28 campaign maps per root is unchanged: 59 unsupported script nodes on each. |
| Inventory / equipment | PASS | The doll box now draws; the worn box is `1023`'s own authored choice not to draw it (`DIV-200`), not a frame-fit refusal. The pack bar, the spellbook bar and the control panel fit and take their presses. |
| Persistence / save-load | N/A | Nothing here is serialized. `check-release-tests.sh` passes 38 of 38 on both roots, which includes the save-load release tests. |
| Campaign / session | PASS | The mission-10 drive runs end to end on both roots and reaches the same outcome at the same tick as the baseline. |
| Shipped content | PASS | 28 campaign maps per root load, compile a script and report the same census as the baseline. |
| Interactions with existing mechanics | PASS | `check-scenarios.sh` is 13 of 13 on both roots at the combined landing, including `scenarios/1005-doll-and-shop.json` and `scenarios/1005-doll-carry-over-worn.json`, the two `1026` alone left failing. |

## The hit-test population, and how it was established

No mission hit test was changed. The conversion is at the boundary, so the population that had to be
complete is the set of places a **window** coordinate can enter the mission screen.

Established by census over `pkg/ui`'s non-test sources:

1. **`ebiten.CursorPosition()` has exactly two call sites**: `readAppInput` and `readInput`. They are
   the only source of a window coordinate from the engine.
2. **`appInput.CursorX/CursorY` readers.** On the map arm they are `App.step`'s
   `noticeButtonAt(in.CursorX, in.CursorY)` and `Viewer.command`. Every other reader is a non-map
   arm — `stepMenu`, `stepPicker`, `stepTownAt`, `stepList`, `stepGameMenu`, `stepChargenDetailed`,
   `stepPreCreate` — and each of those already mapped through `a.place` before this story.
3. **`Input.CursorX/CursorY` readers** are `Viewer.step`, `panIntent` and `dragIntent`. `step` maps at
   entry and passes the mapped `Input` to the other two.
4. **Downstream state.** `v.cursorX`, `v.cursorY`, `v.pressX`, `v.pressY`, `v.dragX` and `v.dragY` are
   assigned only from the fields the three doors have already mapped, so every surface reading them
   reads frame pixels.
5. **Synthetic entry points**: `HeadlessPointer`, `headlessSelectEntityOnce` and `HeadlessDropCell`.
   All three now build their `appInput` from a window position obtained through the placement.

**What the instrument cannot see.** The census is a source scan for two identifiers and two struct
fields. A hit test that obtained a window coordinate by some other route — reading the Ebitengine
window geometry directly, or receiving a coordinate through a channel — would not appear in it. No
such route exists in `pkg/ui` today, and the scan is the whole of the evidence for that.

**A second surface the population does not cover.** `v.cursorX`/`v.cursorY` drive the hover
highlight, which resolves a cell under the cursor. A cursor in the letterbox resolves to a frame
position outside the frame, and the highlight is not bounded by the map view the way
`groundCellAt` now is. The effect is a highlight on a cell that is not drawn; no press follows from
it, because the press path is bounded.

## Mutations run

Each was applied to the production line a maintainer would change, run, and reverted with the
reverse edit pair. The reverts were verified by re-running the affected tests to green and by
`git diff --stat` on the file.

| # | Mutation | Result |
|---|---|---|
| M1 | `WindowToFrameExtended` reads the package constants `W` and `H` instead of `p.frameW`/`p.frameH` | Killed. 19 subtests of the widened `frame_test.go` fail; every pre-existing 640x480-shaped subtest still passes, which is the blind spot the widening exists to remove. |
| M2 | `Viewer.Draw` drops `op.GeoM.Translate(ox, oy)` | Killed. Exactly the four letterboxed windows of `TestMissionDrawComposesOnTheFrameAndPlacesIt` fail; the four unletterboxed ones pass. |
| M3 | `Viewer.Draw` composes on a canvas the size of the window instead of the frame | Killed. Seven of the eight windows fail; the 1024x768 window passes, because there the two sizes agree. |
| M4 | `Viewer.windowToFrame` returns its arguments unchanged | Killed. 17 subtests fail across `hitscale_test.go`; the 1:1 case passes, which is the scale every other test in the package runs at. |
| M5 | `groundCellAt` drops the `mapSurfaceCaptures` refusal | Killed. All five strip subtests and all four letterbox subtests fail. |

M5's letterbox cases pan the camera to (512,512) first. With the camera at the world origin the
letterbox resolves to a negative world coordinate, which `camera.ScreenToCell` refuses on its own, so
the case would have passed under M5 and witnessed nothing.

## Integration witness

`pipeline/check-milestone.sh`, run from `againrom/` against a `missionrun.exe` built from this
branch. The script drives `implementation/builds/current/missionrun.exe` and has no override for that
path, so the branch build was copied in, the census run, and the previous binary restored and
verified by `sha256sum` against the value taken before the swap
(`e20e2f41a3b90561e218a27023c58cc122cda63669ecd5c17ea3f206c50ee5f6`, identical before and after).

- Exit 0. The census is byte-identical to `pipeline/milestone-baseline.txt`.
- 28 campaign maps per root, both roots: **59 unsupported script nodes each**, before and after.
- Mission 10 drive, both roots: `outcome lost at tick 240`, `4 of 36 unit(s) moved, 1 fell, over 240
  tick(s)`. Unchanged.

This story was not expected to move the census and did not. The number is the baseline the seat
regenerated on 2026-08-22; the earlier figure of 224 ticks in circulation belongs to the baseline
before that regeneration.

`pipeline/check-release-tests.sh`, `AGAINROM_IMPL` pointed at this worktree:

- `en`: exit 0, selected 38 install-gated tests, 38 of 38 ran and passed, 0 skipped.
- `ru`: exit 0, selected 38 install-gated tests, 38 of 38 ran and passed, 0 skipped.

`pipeline/check-scenarios.sh`, same `AGAINROM_IMPL`, re-run at the combined landing with story `1023`'s
own fix to the right column's geometry applied: `en`: exit 0, 13 of 13 passed. `ru`: exit 0, 13 of 13
passed. Both roots now pass `scenarios/1005-doll-and-shop.json` and
`scenarios/1005-doll-carry-over-worn.json`, the two this story alone left failing with `headless doll
slot N: this frame draws no doll box (switch on: true, subject: true, view 864x768, unit panel 373
tall)`. The exact figures for this combined landing's own run are in `docs/1023-mission-column/closure.md`.

## Research reconciliation

`SESS-VIEW-028` was read whole through `go run ./tools/claim SESS-VIEW-028` against the pin, and its
three spans were re-derived rather than taken from `contract.md`: `(1024 - 160) / 32 = 27` and
`768 / 32 = 24`; `(640 - 160) / 32 = 15` and `480 / 32 = 15`; `(800 - 160) / 32 = 20` and
`600 / 32 = 18.75`, truncating toward zero to 18. All three agree with the claim's own figures, and
the claim's headline and body agree with each other.

What the build takes from the claim: the map view rect `(0, 0, screenW - 0xa0, screenH)`, the
160-pixel strip, and the 27 by 24 spans at 1024x768. What it does not take: the resolution selection
order, the 800x600 arm, and the vertical recompute for an open panel.

Divergence rows written:

| ID | Table | Subject |
|---|---|---|
| `DIV-210` | Divergences | The mission runs the 1024 arm rather than the shipped 640 default. Owner directive, ACCEPTED. |
| `DIV-211` | Authored | Windowed presentation: one uniform scale, centred, letterboxed. |
| `DIV-212` | Authored | A press in the right strip or the letterbox names no map cell. |
| `DIV-213` | Divergences | The mission HUD's right column does not fit a 768-pixel frame. **CLOSED at story `1023`'s combined landing**, moved to `DIVERGENCES-CLOSED.md`. |
| `DIV-214` | Divergences | An open panel does not shorten the map view, which `SESS-VIEW-028` decodes for ROM1. |

`DIV-215` and `DIV-216` are **returned unused** and are never reissued.

## Open items

1. **`DIV-213` blocked the landing and is now closed.** The mission HUD's right column, as this story
   alone shipped it, stacked a 300-pixel minimap reservation, a 42-pixel control panel, a 172-pixel
   worn box, a 260-pixel doll box and a fit-to-content unit panel that was 373 pixels tall for a shipped
   party member; with margins, the minimap reservation and the unit panel alone needed 771 pixels
   against a 768-pixel frame, and no arrangement of the authored sizes fit. Story `1023`, landed in the
   same combined push, replaces that authored HUD with the strip's own decoded widgets — a 160-pixel
   column at `SHOP-FIGURE-041`'s own 158/80/242 division — which fits. See
   `docs/1023-mission-column/closure.md` for the full account.
2. **The hover highlight is not bounded by the map view.** Named under the population above. It draws
   a highlight on an undrawn cell when the cursor is in the letterbox. No press follows from it.
3. **`TapSlop` reads against two different frames.** The mission accumulator sums frame pixels now
   rather than window pixels, so its tolerance no longer shrinks as the window grows. The remaining
   difference from the shop's is the constant 1.6 the two frame sizes imply. Recorded in
   `pkg/ui/command.go`'s own doc block; not a divergence row, because no claim states a real distance
   for either screen's drag tolerance.
4. **800x600 is not built**, per the contract's own exclusion.
