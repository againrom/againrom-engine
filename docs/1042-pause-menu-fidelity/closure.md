# Story `1042` — closure

## Result

The production in-game menu now exposes the researched campaign, standalone-map and town root
populations, with install labels, label-derived accelerators and decoded gates. Every enabled
owner-directed destination is reachable. The two exit entries open their distinct researched
confirmation populations before a destination is selected.

This story changes the player-visible game rather than the script-gap census. The EN and RU
real-install drive opens campaign mission 10, visits Game Options, Sound Options and Quest
Objectives, returns without moving the world hash, checks all five mission confirmation rows, loads
a town save, checks all three town confirmation rows, confirms the town exit, then opens a shipped
standalone map and visits Diplomacy. A second real-install witness drives every root, nested,
refusal, cancel and terminal action. Both roots resolve all 16 fields from their exact source-table
lookups; none of the 16 is byte-identical between roots.

The milestone baseline remains unchanged. Mission 10 and mission 20 each print zero `UNSUPPORTED`
nodes before and after this slice. `pipeline/check-milestone.sh` reproduces every EN and RU census
row in `pipeline/milestone-baseline.txt`, including the unchanged 240-tick drive result.

## Behaviour evidence

| Requirement | Production evidence | Witness |
|---|---|---|
| FR-1 | `missionGameMenuRows`, `townGameMenuRows` and `flow.menuRows` compose the three roots and their gates | `TestPauseRootPopulationAndDecodedGates`; EN/RU release drive |
| FR-2 | `InstallWords.Words`, `gameMenuAccelerator` and `chooseGameMenuAccelerator` resolve all thirteen menu labels and their effective keys | `TestConfirmationWordsUseDialogsLocalIndices2ATo2D`; exact per-field lookups in `TestInstallWordsOverALawfulInstall`; existing CP866 accelerator tests |
| FR-3 | `FrontEnd.gameMenuTips`, `setGameMenuTips`, `gameMenuSound`, `setGameMenuSound` connect the two settings pages to the existing option store and live player | destination test; options-store and audio-device tests; EN/RU release drive |
| FR-4 | `gameMenuContext` projects the ALM description and copied live directional relations | context and destination tests; scrolled Diplomacy pointer mutation witness; EN/RU release drive |
| FR-5 | nested pages remain on `ScreenGameMenu`; `closeLoad`, `menuUp`, `Picker.Visible` and `pageReturnRow` hold and restore the source | `TestCampaignLoadReturnRebuildsAndHoldsTheVisibleMenu`; town/main-menu/successful-load controls; scrolled-pointer witness |
| FR-6 | `flow.menuRows` composes five mission and three town confirmation rows; the action switch reaches picker, main menu or application exit | confirmation tests; retained-seam assertions; `TestReleasePauseMenuCoversFullActionPopulation`; both headless scenarios |
| FR-7 | menu context is copied read-only; only Tips Mode enters the existing option file and sound remains on `FrontEnd` | world-hash and settings tests; unchanged save and binary suites |

| Decision | Closure |
|---|---|
| DD-1 | One `gameMenuAction` value per row and one dispatcher are exercised through production Enter, pointer and accelerator paths. |
| DD-2 | `GameMenuContextSource` copies the objective and relation projection at menu open; no world pointer crosses into the client. |
| DD-3 | Every nested page remains on `ScreenGameMenu`; confirmations alone select the decoded town-sized panel. |
| DD-4 | The concrete audio player privately implements `SetSettings`; its mutex and device tests cover concurrent playback/settings reads. |
| DD-5 | Install byte strings reach the install font; literal objective and relation rows preserve prose and carry no accelerator. |
| DD-6 | No sim file, digest input, persistence form or binary version changed. Full simulation, save and binary suites remain green. |

## Twelve aspects

| Aspect | Result | Evidence |
|---|---|---|
| Data | PASS | `dialogs.txt` indices `0x22..0x2d`, `0x4c` and `0x4d` resolve on both lawful roots; ALM description and roster relations feed read-only pages. |
| Runtime state | PASS | Root/page/surface, gates, held-background flag, copied context, sound session values and exit request have explicit owners and reset paths. |
| Simulation | PASS | Navigation and information pages leave the world hash unchanged; no sim writer or digest input changed. |
| Input | PASS | Enter, pointer, ASCII/CP866 accelerators, disabled rows, duplicate-first selection and scrolled visible slots are covered. |
| AI | N/A | The slice adds no AI decision or state and does not touch concurrent story `1037`'s withdrawal work. |
| UI/HUD | PASS | Three roots, four nested destinations and two distinct confirmation populations use production composition and decoded rectangles. |
| Triggers | N/A | No trigger is read or written; the held menu prevents world cadence from reaching trigger evaluation. |
| Inventory/equipment | N/A | No row reads or changes inventory or equipment; save/world regression suites remain unchanged. |
| Persistence | PASS | Tips Mode reuses the existing option store. Sound is process-session state. Save payload and version are unchanged. |
| Campaign/session | PASS | Campaign and standalone roots are distinct; Change Map/Victory, main-menu screen selection with retained seams, application exit, cancel and Return have reachable destinations. |
| Shipped content | PASS | EN and RU installations each drive campaign mission 10, a town save and a shipped standalone map with their own label bytes. |
| Existing mechanics | PASS | Save, Load, audio playback, option persistence, Picker scrolling, overlay hold and screen/session boundaries remain connected through their production seams. |

There is no in-scope GAP.

## Candidate verification

All commands below measured the correction candidate in the story worktree. The final handoff names
the exact pushed SHA; `check-seat-tree.sh` is pointed at that same checkout.

| Gate | Result |
|---|---|
| `go build ./...`; `go vet ./...`; `go test ./...` | PASS |
| `scripts/check-no-game-assets.sh` | clean tree scan |
| `scripts/check-claim-citations.sh` | 1,293 distinct citations resolve against 1,479 claims and 220 experiments |
| Research `go test ./...` | PASS |
| Research claim-id, retraction and regeneration gates | 1,479 distinct ids; 237 overturned ids marked; 11 regeneration scripts honor `OUT` |
| `check-div-claims.sh` | Story worktree and pin named; 237 of 237 live rows structurally selected |
| `check-pin-forward.sh` | Story and master both pin `d7ee0c6` |
| `check-preserved-installs.sh` | 162 files unchanged |
| `check-seat-tree.sh` | clean correction candidate, exact branch and SHA named |
| `check-scenarios.sh`, EN and RU | 15 of 15 on each root, at the story candidate |
| `check-release-tests.sh`, EN and RU | 52 of 52 on each root, zero skipped, at the story candidate |
| `check-milestone.sh` with the story-built drive | every EN/RU census and the 240-tick drive match the baseline |
| Direct mission 10 / mission 20 trace | zero / zero `UNSUPPORTED` lines |

The repository is gofmt-clean, the research submodule remains at the pinned commit and no commit in
the story range carries a `Co-Authored-By` trailer.

## Mutation and integration witnesses

`TestCompleteGameMenuPaintFrames` records the whole ordered production paint stream for all nine
states: panel, border, focus, every install-font label, every accelerator underline and every
disabled overlay. Independent frozen digests make any one-pixel, colour, label, population or
draw-order mutation fail. This closes `DIV-404`; the earlier rectangle witness remains as a readable
geometry diagnosis.

The production pointer witness adds `Picker.Visible`'s top to the clicked visible slot. Replacing
`top+row` with `row` made `TestScrolledDiplomacyReturnUsesTheVisiblePointerSlot` remain on page 4
instead of returning to the root; restoration was byte-identical. The pre-mutation `app.go` hash was
`1b18885d4837fe1cc496e34f6fa4c9da415779ff`.

`TestReleasePauseMenuVisitsEveryDestination` remains the real-install campaign/session and raw-label
drive. `TestReleasePauseMenuCoversFullActionPopulation` separately drives the complete action and
destination population, closing `DIV-406`. `TestInstallWordsOverALawfulInstall` compares all sixteen
resolved fields to their exact source lookups on both roots and closes `DIV-405`; it also reports zero
byte-identical pairs. These are headless production dispatch witnesses, not claims that a desktop
window was visually observed.

## Research reconciliation

`MENU-ITEM-011` matches the two mission root populations, row order, install bindings and decoded
gates. `MENU-ITEM-012` matches the town root, the five-row mission confirmation, the three-row town
confirmation, their install labels, all-enabled state and `(100,100)-(440,340)` rectangle.
`MENU-KEY-013` matches the marked-label walk and CP866 fold. It does not decode the constructor
fallback: Change Map's unmarked EN `C` is owner-authored.

Research leaves the mode word, submenu contents and confirmation targets unresolved. Owner-directed
coherent meanings are recorded in `DIV-099`: campaign maps use Load and Quest Objectives;
standalone maps use Diplomacy; the settings and information pages use existing live state; Change
Map and Victory release to the picker without manufacturing a win; Exit to Main Menu selects the
main-menu screen while retaining the current viewer/session seams; Exit to Windows requests
application exit. The prior omission debt is closed. Pass-1 W rows `DIV-404` through `DIV-406` are
closed by their discriminating witnesses.

## Review

Pass 1 reviewed exact pushed SHA `d6a7871ea4aa2a59b623fcf660bfa298b2d0bdbf` and returned one P,
three W and two D findings, with the original surface exhausted. This correction fixes the central
Load return and all five witness/truth deltas. The fresh-pass remaining surface is the correction
delta listed in `contract.md`; the durable pass record is `verification.md`.
