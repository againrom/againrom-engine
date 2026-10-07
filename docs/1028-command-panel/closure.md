# Story `1028` — the mission command panel — closure

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | Four new archive entries read at runtime (`interface/headsr.bmp`, `commandbarr.bmp`, `commandempr.bmp`, `commanddnr.bmp`, `pkg/game/commandpanelart.go`); `text/main.txt[0..7]` added to `InstallWords.Words()` as `Words.Command`. No new persisted or committed data. |
| Runtime state | PASS | New `Viewer` fields (`commandPanelArt`, `commandPanelImg`, `commandHoverImg`, `cmdOverlayHidden`, `cmdDragCell`, `cmdPendingGuard`, `cmdPendingStandGround`) and one new authored arm state, `commandMove` (`pkg/ui/command.go`). All session/presentation state, none persisted. |
| Simulation | N/A | No `pkg/sim` change. Every order the panel issues (attack, move, Guard, Stand Ground) routes through pre-existing seams (`v.armAttack`, `v.armCommand`, `a.flow.stance`, `MapOrder`) unchanged; `commandMove` normalises to `commandNone` at the single order-build boundary in `command.go`'s `decide`, so no new value crosses into simulation input. |
| Player input | PASS | Eight key rebindings and the panel's own mouse gestures (left down/drag/margin, right-up cancel), spec B2-B4. Grab and March moved a second time at the return visit (`F`, `H`; `DIV-236`, `DIV-237`), off two letters `text/main.txt` names, on the owner's 2026-08-22 ruling. |
| AI | N/A | `pkg/ai` untouched. The panel issues player-authored orders through the same seams the existing keyboard/mouse bindings already used. |
| UI/HUD | PASS | `composeCommandPanel`, `commandPanelPresent`, `commandHoverPresent` (`pkg/ui/commandpanel.go`), wired into `Viewer.Draw` (`pkg/ui/viewer.go`), replacing `hudtoggles.go`'s four-switch bar in the same slot. The readout's armed-order label (`PanelFieldOrder`, `pkg/ui/readout.go`) omitted `commandMove` until the return visit; fixed and covered by an extended `TestTheReadoutStatesWhichOrderIsArmed`. |
| Triggers/scripts | N/A | Untouched. Milestone census (below) confirms no script-node behaviour moved. |
| Inventory/equipment | N/A | Untouched in function; the four display switches (`I`/`S`/`D`/`E`) keep their existing behaviour and lose only their drawn box. |
| Persistence/save-load | N/A | No new persisted field. `commandMove` and the panel's own overlay/drag state live on `Viewer`, which is never serialized. |
| Campaign/session | N/A | Untouched. |
| Shipped content | PASS | `TestReleaseCommandPanelArtAndLabelsComposeFromTheRealInstall` (`pkg/game/commandpanel_release_test.go`) resolves and checks the four real bitmaps and the eight real `main.txt` labels against this build's own key bindings, on both roots. `TestReleaseCommandPanelComposesFromRealArtInFourStates` (`pkg/ui/commandpanel_release_test.go`, added at the return visit) drives `composeCommandPanel` itself against the same real bitmaps and compares the composed frame, byte for byte, in four panel states. |
| Interactions with existing mechanics | PASS | The S/D collision with the pre-existing display switches is resolved and recorded (`DIV-232`); the pre-existing WASD camera pan is removed and screen-edge panning (already unconditional) becomes the sole non-arrow camera input (`DIV-233`); seven pre-existing tests were extended for the new geometry, vocabulary and key set (`vocabulary_test.go`, `words_test.go`, `autocast_test.go`, `fogwalk_test.go`, `missioncolumn_test.go`, `installtext_test.go`, `frontend_session_test.go`), all green. |

No in-scope GAP.

## What was verified, and how

- `go build ./...`, `go vet ./...`, `gofmt -l` (tracked+untracked `*.go`), `go test -trimpath
  -count=1 ./...`: all clean, no skips beyond the pre-existing install-gated set.
- `scripts/check-no-game-assets.sh` and `scripts/check-claim-citations.sh` (implementation repo,
  glob): both `ok`. Citations moved from 1207 to 1208 of 1412 claims at the return visit (`DIV-236`
  and `DIV-237` each cite a claim already read by the original story or newly read for the ruling;
  `AI-PANEL-123` is the one new id cited).
- `scripts/check-claim-ids.sh` (`ok`, 1412 ids, unchanged) and `scripts/check-retraction-status.sh`
  (`ok`, 230 overturned, unchanged) (research repo, glob): the pin is frozen mid-story at `fb372e2`
  and neither number moved.
- `pipeline/check-release-tests.sh`, both roots, `AGAINROM_IMPL` pointed at this worktree: `ok (40 of
  40 install-gated tests ran and passed, 0 skipped)` on both EN and RU. 40 is 39 (the count at the
  first landing attempt) plus the one release test added at the return visit
  (`TestReleaseCommandPanelComposesFromRealArtInFourStates`); the population is expected to move by
  exactly one and does.
- `pipeline/check-scenarios.sh`, both roots: `ok (14 of 14)` on both EN and RU, unchanged.
- `pipeline/check-preserved-installs.sh`: `ok`, both roots as recorded.
- `pipeline/check-milestone.sh`, `AGAINROM_MILESTONE_DRIVE` pointed at a `missionrun` built from this
  worktree: exit 0, `ok the script gap and the drive are where they were recorded, both roots` — the
  census is unchanged against `pipeline/milestone-baseline.txt`. Mission 10 and mission 20
  specifically: `AGAINROM_ASSETS=gameversions/en missionrun -mission {10,20} -trace -ticks 1 | grep -c
  UNSUPPORTED` reads 0 and 0 on both, matching the baseline (neither mission carries a "cannot run"
  line there either). This story, and this return visit, do not move the script-node census; neither
  was expected to, since neither touches a script or AI dispatch table.
- `git diff --diff-filter=D --name-only 0a77dfce..HEAD`: empty (`0a77dfce` is master's tip merged into
  this branch at the return visit).
- `git submodule status`: unchanged from the original landing, pin frozen at `fb372e2`, no leading
  `+`/`-`.
- `git log --format='%h %(trailers:key=Co-Authored-By)' 0a77dfce..HEAD`: no trailer on any commit.

## Integration witness

The strongest available witness with real assets is now two tests together, added at different
points: `TestReleaseCommandPanelArtAndLabelsComposeFromTheRealInstall` (`pkg/game`, original landing)
resolves the panel's four bitmaps and its eight labels through the production `FrontEnd`/`InstallWords`
path from a real lawful install, on both roots, and cross-checks each label's own accelerator letter
against the key this build actually binds for that cell. `TestReleaseCommandPanelComposesFromRealArtInFourStates`
(`pkg/ui`, return visit, item 2 of the pass-1 review) drives `composeCommandPanel` itself, the function
that actually paints the panel, through its four reachable states, and compares the composed frame
byte for byte against an independently built expected frame per state. The first witnesses that the
right assets and labels are resolved; the second, added because no committed test reached the composer
before the return visit, witnesses that the composer paints the right cells for the right state. A
defect in the archive path, the `main.txt` slot, the key binding, or the composer's own draw order
fails one or the other. Both are real-asset x real-code checks, not synthetic fixtures.

No headless scripted drive presses the command panel itself inside a live campaign mission and reads
back a changed entity state (for example, a scripted `G` press followed by reading the marked
entity's stance flag through `sim`). The panel's mouse and key dispatch (B2-B4) reaches only
pre-existing, already call-tested seams (`v.armAttack`, `v.armCommand`, `a.flow.stance`, `MapOrder`),
each of which already has its own unit coverage from the stories that built it (0146, 0154, and
earlier). Building a fresh end-to-end scripted drive through the panel specifically was judged lower
value than the two witnesses above, given the panel adds no new simulation behaviour of its own; it
is named here as the residual gap rather than silently assumed covered.

The 14-scenario headless suite (`pipeline/check-scenarios.sh`) exercises full campaign missions,
including mission 10 (escort) and mission 20, end to end on both roots, and all 14 pass unchanged —
this confirms the story introduces no regression reachable by any scenario already on file, though
none of the 14 was written to press this panel specifically.

## Research reconciliation

All five governing claims (`MENU-COMBAT-017`, `TOWN-092`, `MENU-COMBAT-018`, `MENU-COMBAT-019`,
`AI-PANEL-053`) were read whole via `go run ./tools/claim <ID>` against pin `fb372e2`, matching the
contract's own citation table. `AI-PANEL-053`'s two partial retractions were read and applied: the
capability clause is not tested (`commandPanelActive` gates on ownership alone, matching
`canArmAttack`'s own pre-existing reading), and opcodes `0x18`/`0x14` are Stand Ground/Retreat, not
"aggressive"/Patrol — the panel's own cell-to-action mapping in `commandpanel.go` uses the corrected
labels. No claim cited here has been amended or retracted further since the contract was written; the
pin was not bumped mid-story.

At the return visit, `go run ./tools/claim <ID>` was run again against the same frozen pin for
`AI-PANEL-123` (Retreat's immediate opcode, cited by `DIV-237`) and for the two `text/main.txt` line
ranges the owner's ruling turns on (lines 0..14, both roots, via `restool cat <root>/main.res
text/main.txt`). `go run ./tools/claim -k` was run on "march", "standing order", "main.txt",
"accelerator", "spellbook", "Open Inventory", "worn", "equip" before any sentence stating research
does not answer a question was written into a divergence row or a code comment; none of these
searches returned a row establishing `commandMarch` as a decoded order, or establishing that the
executable reads `main.txt` lines 8..13 as live accelerators.

## Divergence ledger

`DIV-230` through `DIV-234` spent, none returned. `DIV-201` narrowed in place. Full rows:
`docs/DIVERGENCES.md`. Two pre-existing code comments (`app.go`) forward-referenced `DIV-232` for two
different facts before the ledger rows existed to settle which was which (the disabled cells and
Patrol's own key); reconciled at the landing to `DIV-230` and `DIV-231` respectively, matching the
final ledger text (commit `0b42f6cd`).

`DIV-236` and `DIV-237` are added at the return visit, both instances of the owner's 2026-08-22
ruling that a key of this project's own must not occupy or scramble a letter `text/main.txt` names.
`DIV-236`: Grab moves off `B` (the shipped Spellbook letter) to `F`. `DIV-237`: March moves off `R`
(the shipped Retreat letter, `AI-PANEL-123`) to `H`, leaving `R` free for a future Retreat order
(`DIV-230`). Neither reopens `DIV-232` (`S`→Book, `D`→Doll), ruled separately the same day.

## Open items

- No headless scripted witness drives the panel's own key/mouse dispatch inside a live mission and
  reads back simulation state (see Integration witness above). The seams it reaches already carry
  their own coverage; a dedicated end-to-end scripted press was judged out of this story's value for
  its cost.
- Status OPEN: `DIV-230` (three disabled orders; its own revisit condition now also names that `R`
  is free for Retreat's keyboard route, since `DIV-237` moved March off it), `DIV-232` (S/D
  collision, latent until an order exists to collide for), `DIV-233` (camera pan input, Unknown),
  `DIV-234` (Cast cell inert).
- Status ACCEPTED (owner-directed, not reopened by this or a later story absent a new ruling):
  `DIV-231` (Patrol's key), `DIV-236` (Grab's letter; revisits if a later decode gives the spellbook
  toggle its own key from `main.txt`'s Q/B pair), `DIV-237` (March's letter; revisits when Retreat is
  built and can claim `R` directly).
  Each row's own revisit condition is in `docs/DIVERGENCES.md`.
- `PIPELINE-STATUS.md`'s own `DIV-230` reservation line (line 607 at this writing) is not updated by
  this lane; that document is seat-owned and is reconciled at merge.

## Style

`PROSE.md` governs this document.
