# Story `1023` — closure

This story closes cleanly. Its own combined landing with story `1026` closes `DIV-213`, the row that
blocked `1026`'s own closure: the two headless scenarios that failed under `AuthoredPanelLayout`'s own
truncation both pass under this story's `CompactPanelLayout` panel.

Round 1 of adversarial review found a player-visible defect (P finding): the whole column, on both
axes, stood on authored geometry rather than the decoded one — 12 pixels inside the frame's own right
edge horizontally, and 160/42 rather than the decoded 158/80 vertically, for the minimap and control
panel. Round 2 fixed both axes (`rightColumnBox`, `hudMinimapReserve`, `hudToggleBarH`,
`authoredMinimapCorner`, all `pkg/ui`), widened `missioncolumn_test.go` to derive its own left edge
independently from `MissionViewportSize().X` rather than a literal baked in at round 1, added
`hud_test.go`'s invariant test, and updated `minimap_test.go` where its own assertion had baked in the
pre-fix width. This document is rewritten to describe the round-2, as-shipped state throughout; it is
not a per-round journal.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | N/A | No new data format, no new shipped file read. `sidebarWidth`, `hudPanelTopY`, `compactPanelH` are Go constants; none is derived from a parsed asset. |
| Runtime state | PASS | `Viewer.panelLayout`, `Viewer.panelCardFont` are new fields with a single writer each (`NewViewer`'s default, `SetCardFont`) and no other mutator. |
| Simulation | N/A | No `pkg/sim` file touched. `go build ./pkg/sim/...` and `internal/archtest` (the determinism wall) are unchanged and pass. |
| Player input | N/A | No new input path. Selection, hover and drag still resolve through the same hit tests this story's own rect changes feed; `TestMissionColumnDrawsThePanelDollMinimapAndControlPanelAtTheirOwnSlots` and the extended `panel_draw_test.go`/`invclick_test.go` cover the moved rects. |
| AI | N/A | No AI file touched. |
| UI/HUD | PASS | FR-3 through FR-6 of `spec.md`; witnessed at the use site by `missioncolumn_test.go`, mutation-proved (see table below). |
| Triggers/scripts | N/A | No `pkg/game` script-interpretation file touched. `pkg/game/frontend.go`'s own two edits are both `SetCardFont` calls; no script node, check or instant is added, removed or reinterpreted. |
| Inventory/equipment | PASS | `invSlotColumns` 4 to 3 and the resulting `wornBoxSize()` change are exercised by `inventory_test.go`'s existing grid tests, extended for the new column count; `wornSlotRects` still returns a non-overlapping grid at every population size the suite checks. |
| Persistence/save-load | N/A | No `pkg/formats/sav` file touched, no save-shape change. `TestReleaseGeneratedCharacterLaunchSaveLoadAndCampaignContinuity` (which round-trips a save through this exact character view) passes on both roots. |
| Campaign/session | PASS | `check-scenarios.sh`'s 13 scenarios include two full mission-context scenarios (`0155-mission10-escort.json`, `0155-mission20-sweep.json`) and both pass with the rebuilt column geometry. |
| Shipped content | PASS | Measured against both preserved roots (EN, RU) throughout; `ROM.EXE` is the same bytes on both (`EXP-0123`), so one root suffices for logic and both are run because this story changes drawn geometry. |
| Interactions with existing mechanics | PASS | `AuthoredPanelLayout`'s only remaining production-adjacent callers are `pkg/game`'s character-generation flow (unaffected — it never called the mission panel) and `cmd/paneldump`, a developer tool; grep confirms no other production caller (see Research reconciliation). Story `1022`'s town toggle (`townCharacterMode`) is untouched, per the contract's own exclusion. |

No in-scope GAP. The story's own two UNKNOWNs (`DIV-200`, `DIV-201`) are named divergences, not gaps:
the contract's own B4/B5 call for exactly this — record the authored choice as a typed row, not leave
it undocumented.

## Mutations run

AGENTS.md coverage rule 6: a new `GeometryTests` entry earns its place by failing when the production
draw call is mutated, not when the rectangle-returning helper's own return statement is. Round 1's own
account of this section understated the coverage: it recorded the minimap and control-panel assertions
as not separately mutation-proved. Round 2 corrects that understatement by running the mutations and
recording their results, rather than by argument from a shared call path. All four mutations below were
applied directly to `pkg/ui/hud.go` or `pkg/ui/minimap.go` with the same edit-pair technique (mutate,
run, revert to the pre-mutation text, confirmed with `git diff --stat` empty before the next one),
against `go test ./pkg/ui/ -run 'TestMissionColumn...|TestHudColumnSlots...|TestMinimapBoxIsSquare...'`.

| # | Mutation | Result |
|---|---|---|
| 1 | `rightColumnBox` (`hud.go`): reintroduced the round-1 P finding, subtracting `hudMargin` from both edges — `image.Rect(area.X-sidebarWidth-hudMargin, y, area.X-hudMargin, y+h)`. | `TestMissionColumn...` failed on the panel, the doll and the control panel (all three route through `rightColumnBox`): `drawn at (852,238), want (864,238)`; `(852,488), want (864,488)`; `(852,158), want (864,158)`. The minimap does not route through this function and was unaffected, correctly. |
| 2 | `hudMinimapReserve`/`hudToggleBarH` (`hud.go`): reverted to the round-1 authored values, 160 and 42. | `TestHudColumnSlotsSumToThePanelsOwnTop` failed: `160+42=202, want 238`. `TestMissionColumn...` failed on the minimap (`drawn at (864,0), want (866,0)`; `size (160,160), want (158,158)`) and the control panel (`drawn at (864,160), want (864,158)`; `size (160,42), want (160,80)`). |
| 3 | `authoredMinimapCorner` (`minimap.go`): reverted its `Margin` from `(0,0)` to the round-1 `(12,12)`. | `TestMissionColumn...` failed on the minimap alone: `drawn at (854,12), want (866,0)`. Confirms the minimap's own placement route (`panelOrigin(authoredMinimapCorner,...)`), independent of `rightColumnBox`, is covered. |
| 4 | `minimapGeometry`'s own `box := hudMinimapReserve` (`minimap.go`): changed to `box := sidebarWidth`, the pre-`DIV-201` coupling the owner reported ("I can see it grow after I pick a hero"). | Failed in two places: `TestMinimapBoxIsSquareAndDoesNotMoveWithTheSelection` (`the box is 160 wide, want hudMinimapReserve = 158`) and `TestMissionColumn...` (`size (160,160), want (158,158)`). Confirms the coupling regression is caught by an existing test as well as the new one. |

Every mutation was reverted byte-identical (`git diff --stat pkg/ui/hud.go pkg/ui/minimap.go` empty)
before the corresponding real fix was reapplied. Mutations 1 through 4 together cover every production
call site the placement population enumeration below names: `rightColumnBox` (panel, doll, control
panel), the two decoded-size constants, and the minimap's own independent `panelOrigin` route and its
own size-coupling line.

## Placement population

Every mechanism in `pkg/ui` that places a box against the frame's own right edge or top-right corner,
checked at this landing:

| Mechanism | Disposition |
|---|---|
| `rightColumnBox` (`hud.go`) | Fixed (mutation 1). Feeds the panel, the doll and the control panel's bar. |
| `authoredMinimapCorner` via `panelOrigin` (`minimap.go`) | Fixed (mutation 3): margin `(12,12)` to `(0,0)`. Independent of `rightColumnBox`. |
| `minimapGeometry`'s own `box` variable (`minimap.go`) | Fixed (mutation 4): decoupled from `sidebarWidth`, now reads `hudMinimapReserve`. |
| `hudMinimapReserve`, `hudToggleBarH` (`hud.go`) | Fixed (mutation 2): 160/42 to 158/80. |
| `hudBandX` (`hud.go`) | Fixed: the left-stack boundary's own formula assumed `rightColumnBox`'s pre-fix, double-margined structure (`area.X - sidebarWidth - 2*hudMargin`); corrected to one `hudMargin` to keep the same 12-pixel visual gap between the left stack and the now-flush column. No test named this call site before this landing; `TestMissionColumn...` does not cover the left stack, so this fix is covered by inspection of the arithmetic against the corrected `rightColumnBox`, not by a mutation. |
| `hudToggleCellRects` (`hudtoggles.go`) | Fixed: the button row's own vertical placement changed from `bar.Min.Y + hudBarPad` (top-aligned, correct only when the bar was 42 tall, `2*hudBarPad+hudToggleCell`) to `bar.Min.Y + (bar.Dy()-hudToggleCell)/2` (centred), since the bar is now 80 tall. Both the draw path (`composeHudTogglePanel`) and the hit test (`Viewer.hudToggleCaptures`) call this same function, so the two cannot disagree by construction; `TestHudTogglePanelTakesItsOwnPresses` (`inventory_test.go`) derives its own press points from this function and continues to pass, confirming every button stays individually clickable, but does not pin the row's absolute position within the bar — no claim decodes the row's own placement inside an 80-tall bar, so centring is this project's own choice among options the decode does not rule out. |
| `dollBoxRect`, `wornBoxRect` (`inventory.go`) | Unaffected. Both delegate to `rightColumnBox` and picked up the fix automatically; their own sizes (`dollBoxSize`, `wornBoxSize`) are unrelated authored constants, unchanged. |
| `AuthoredReadoutLayout` (`readout.go`) | Out of scope. `Corner: PanelTopLeft`, not part of the right column. |
| Spellbook bar (`spellbook.go`) | Out of scope. Part of the left stack (`hudBandX`/`hudStackTops`), not the right column; grepped for `PanelLayout`/`Corner:`/`Margin:`/`hudMargin`/`sidebarWidth`/`panelOrigin`, no match. |

## Research reconciliation

Two claims are cited: `SESS-VIEW-028` (viewport rect and the 160-pixel strip, High) and
`SHOP-FIGURE-041` (the 158/80/242 division and id 7's identity and transfer mechanism, High). Both
were re-read whole at this landing (`go run ./tools/claim SESS-VIEW-028`,
`go run ./tools/claim SHOP-FIGURE-041`), not from the contract's own digest of them.

`SESS-VIEW-028`'s Medium clause is specifically about `R0387`'s own "child id 2 or 3" —
the map view's Inventory/SpellBook toggle recompute, a different construction site from
`SHOP-FIGURE-041`'s ids 5/6/7/8. The contract's own B4 cites `SESS-VIEW-028`'s Medium grade for ids 5
and 6's widget identity; re-reading both claims whole found that imprecise, because no claim carries
any confidence grade — Medium or otherwise — for ids 5 and 6 at all. `DIV-201` is recorded `UNKNOWN`,
not as a Medium reading, and `spec.md`'s own "Differences from the contract" section states this
correction.

Round 2 re-read both claims whole a second time, against the owner's 2026-08-21 ruling («давай все
приводить к оригиналу» — bring everything to the original) that a decoded value collects an authored
one where no directive drove the difference. `SHOP-FIGURE-041`'s own ids 5 and 6 rects, `(0,0,160,158)`
and `(0,158,160,238)`, name heights (158, 80) even though they name no tenant; nothing in `DIV-201`'s
own Owner-directive cell (minimap above control panel, 2026-08-11) touches either height. Both are now
the implemented literal (`hudMinimapReserve`, `hudToggleBarH`). `DIV-201` is narrowed rather than
closed: the tenant-assignment question is still open, and the row now records only that.

`AuthoredPanelLayout`'s production callers, checked at this landing:

```
grep -rn "AuthoredPanelLayout(" pkg cmd
```

returns its own declaration (`pkg/ui/panel.go`), `cmd/paneldump/main.go` (a developer tool under
`cmd/`, not a production caller), and test files. No production `pkg/game` or `pkg/ui` file calls it
after this story. Before this story, `pkg/ui/hud.go`'s `NewViewer` was its one production caller.

## Divergence rows written

| ID | Table | Subject |
|---|---|---|
| `DIV-200` | Authored where research is silent | The mission column's doll and worn boxes: doll placed under the panel by authored choice, worn box dropped from the mission column entirely. |
| `DIV-201` | Authored where research is silent, narrowed at round 2 | The mission column's minimap and control-panel slots: top-to-bottom order matches the owner's 2026-08-11 ruling; both slots' own reserved heights are now the decoded 158/80 (round 2). What stays open is the tenant assignment alone — which of ids 5 and 6 is the minimap and which is the control panel — because no claim names either slot's own tenant. |
| `DIV-213` | Closed (moved to `DIVERGENCES-CLOSED.md`) | `1026`'s own blocking row: two headless scenarios failed under `AuthoredPanelLayout`'s truncation. Closed by this story's switch to `CompactPanelLayout`. |
| `DIV-217` | Amended in place | The figure widget's own corner controls, still open, still out of scope for this story. Its Implemented-behaviour cell is corrected from "`AuthoredPanelLayout`" to "`CompactPanelLayout` (story `1023`)", since this story changed which composer the mission's live panel calls. |
| `DIV-218` | Closed (moved to `DIVERGENCES-CLOSED.md`) | A scenario asserting `AuthoredPanelLayout`'s own WEAPON/WORN truncation on the mission screen. Closed by removing the scenario: the mission's live panel is `CompactPanelLayout`, which carries no WEAPON or WORN row at all (story `1022` B3). `AuthoredPanelLayout` itself is unchanged and still truncates the same way if driven directly; it has no production caller left. |

`DIV-199`, `DIV-202` through `DIV-206` were reserved to this story and are returned unspent: this
story's own five behaviours (B1 through B5) needed exactly two new rows (`DIV-200`, `DIV-201`), one
closure (`DIV-213`, already a row from `1026`) and one amendment (`DIV-217`, already a row from an
earlier story). No fresh UNKNOWN beyond the two above was found.

## Integration witness

Re-run in full at round 2's own landing, on the merge of master (`d29116f1`, story `1025`) into this
branch. Repository chains (both repos), by the glob of `scripts/check-*.sh`, at the tree with round 2's
changes staged:

- `go build ./...`, `go vet ./...`, `gofmt -l` (implementation): clean, exit 0.
- `go test -trimpath -count=1 ./...` (implementation): 41 packages `ok`, 0 failures.
- `scripts/check-claim-citations.sh`: `ok (1191 distinct citations resolve against 1400 claims and 209 experiments under 784 prefixes)`.
- `scripts/check-no-game-assets.sh`: `clean (tree scan)`.
- `research/scripts/check-claim-ids.sh`: `ok (1400 ids, all distinct, 31 ledgers, 29 read back through tools/claim)`.
- `research/scripts/check-retraction-status.sh`: `ok (229 overturned ids, every one marked)`.
- `git diff --diff-filter=D --name-only origin/master...HEAD`: empty. No file is deleted by this
  branch relative to `origin/master`.

Seat gates, run against this worktree (`AGAINROM_IMPL=<seat>/wt-1023`) rather than
`builds/current/`, which another lane holds, from `<seat>`:

- `pipeline/check-pin-forward.sh`: `ok` — this branch and its own `origin` copy both pin `4d2df83`, the
  same as master.
- `pipeline/check-preserved-installs.sh`: `ok — 162 file(s), both roots as recorded`.
- `pipeline/check-scenarios.sh` — EN root: `ok (14 of 14)`. RU root: `ok (14 of 14)`. The scenario count
  is 14, not 13: story `1025` (merged into this branch) added
  `scenarios/1025-mission10-weight.json`. Both roots include `scenarios/1005-doll-and-shop.json` and
  `scenarios/1005-doll-carry-over-worn.json`, the two `1026` alone left failing under `DIV-213`.
- `pipeline/check-release-tests.sh` — EN root: `ok (38 of 38 install-gated tests ran and passed, 0
  skipped)`. RU root: the same, `ok (38 of 38, 0 skipped)`.

Mission script-node census, `pipeline/check-milestone.sh` itself, run with `AGAINROM_MILESTONE_DRIVE`
pointed at a `missionrun.exe` built from this worktree (`go build -trimpath -o
<scratch>/missionrun.exe ./cmd/missionrun`), never touching the shared `builds/current/`:

```
check-milestone: AGAINROM_MILESTONE_DRIVE is set -- measuring <scratch>/missionrun.exe
     ok   the script gap and the drive are where they were recorded, both roots:
```

Exit 0. The 28-map, both-root census is byte-identical to `pipeline/milestone-baseline.txt`, which this
story does not move: it touches no `pkg/sim` or script-interpretation file. Mission 10 and mission 20
specifically, both roots, direct measurement with the same scratch binary:

```
AGAINROM_ASSETS=<en root> missionrun.exe -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED   -> 0
AGAINROM_ASSETS=<en root> missionrun.exe -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED   -> 0
AGAINROM_ASSETS=<ru root> missionrun.exe -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED   -> 0
AGAINROM_ASSETS=<ru root> missionrun.exe -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED   -> 0
```

Unchanged from the baseline (0 on both missions, both roots), as expected. This supersedes round 1's
own account, which stood in with a two-mission direct measurement and reasoning about the unchanged
`pkg/sim` surface rather than running the full 28-map instrument, because `AGAINROM_MILESTONE_DRIVE`
did not exist yet at round 1.

## Open items

- `DIV-217` (the figure widget's own corner controls) stays open; a research lane opened 2026-08-21
  is asking it, and this story adds no control to the figure or the card, as its own contract
  requires.
- `DIV-192`, `DIV-193` and the remainder of the 2026-08-21 divergence census (99 open rows, 46
  actionable, 26 visible on screen) are unaffected by this story and remain a separate programme,
  named in the contract's own "Out of scope."
