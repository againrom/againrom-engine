# Closure — 1019-session-end

As-built. Not a journal.

## Twelve-aspect matrix

| aspect | verdict | evidence |
|---|---|---|
| data | N/A | no data format read or changed |
| runtime state | PASS | `FrontEnd.resetSessionForNewGame` (`pkg/game/frontend.go`); completeness proven by `TestResetSessionForNewGameDropsExactlyTheSessionPopulation` (`pkg/game/frontend_session_test.go`), a reflect-based walk of every `FrontEnd` field |
| simulation | N/A | `pkg/sim` unchanged; `live` is nilled, the `sim.World` it pointed to is not touched |
| player input | N/A | no input path added or changed |
| AI | N/A | — |
| UI/HUD | N/A | `pkg/ui` unchanged |
| triggers/scripts | N/A | no script/trigger rule changed |
| inventory/equipment | N/A | `Carried` is cleared wholesale, not edited |
| persistence/save-load | PASS | `pkg/game/resume.go`'s `installCandidate` already installed `Town`/`Carried`/`Offered`/`live`/`liveMission`/`liveParty` unconditionally and needed no change (verified by reading `installCandidate` and tracing `liveDriver`, not taken from the round-2 review's report). `pkg/game/originalsave.go`'s `RestoreOriginal` did not, on either arm, and both now call `resetSessionForNewGame` (round 2's fix; see Root cause and Fix below) |
| campaign/session | PASS | the story's whole subject; see the integration witness below |
| shipped content | N/A | applies uniformly, independent of asset root or content |
| interactions with existing mechanics | PASS | `f.continuity`'s win/lose destination selection (`Town.Open()`), `arriveInTown`'s shop generation, `townScreen.resetForNewGame` — all read by the fix and exercised by the tests below |

No known in-scope GAP.

## Root cause

`pkg/game/frontend.go`'s `FrontEnd` carries five kinds of per-session state (`Carried`, `Offered`,
`Town`, `Shop`, `live`/`liveMission`/`liveParty`), all built once at process start or on opening a
map, and none of them was cleared by any of the three code paths that return the player to the main
menu (`pkg/ui/flow.go`'s `advanceNotice` `NoticeToMenu` arm and `escape()`'s `ScreenPicker` arm;
`pkg/ui/save.go`'s `chooseGameMenu` `gameMenuAbortGame` arm). `pkg/game/frontend.go`'s
`newGameChargen` — the one place a new game is confirmed to start, reached every time a mission row
is chosen from the picker — reset only the presentation-layer `townUI` (`resetForNewGame`, built for
story `0142`) and never these five fields.

The observable consequence: `f.continuity` (`pkg/game/frontend.go`), the existing win/lose
destination wrapper, routes a lost mission to `ui.NoticeToTown` whenever `f.Town.Open()` is true.
`Town.open` latches true the first time a campaign reaches a town and nothing in this tree cleared
it. A second game in the same process therefore inherited the first game's open latch — and, through
it, its chapter, gold and finished/available/taken state — the moment its own first mission was lost,
exactly matching the owner's report.

**Round 2 (this section's own update): the same gap recurred through a second door.** The
population of sites that commit to installing a genuinely different game into `FrontEnd` is not
`newGameChargen` alone. `pkg/game/originalsave.go`'s `RestoreOriginal` — reached from LOAD GAME with
an original-format save — has two arms, split on the save's own stored mission number: `n == 0`
(taken between missions, at a town) and `n != 0` (taken mid-mission). Round 1 gave the `n == 0` arm
an explicit `f.Town = NewTown(f.Campaign)` and `f.Carried = restored`, but touched neither `Offered`
nor `live`/`liveMission`/`liveParty`; the `n != 0` arm touched no `FrontEnd` field at all, only
`townUI`'s own presentation. A player who loaded a mid-mission original save while an abandoned
mission's own world was still referenced (`live`/`liveMission`/`liveParty`) or a next mission was on
offer (`Offered`) inherited both from the game he had just left — the owner's reported symptom,
reached through LOAD GAME rather than through leaving to the menu and starting NEW GAME. Round 2's
own adversarial review found this as a player-visible defect.

**Enumeration method, applied to find the whole population (not re-derive one more site).** Every
non-test `pkg/game/*.go` file was searched for a direct write to one of `resetSessionForNewGame`'s
seven fields:

```
grep -rn "\.Town\s*=\|\.Carried\s*=\|\.Offered\s*=\|\.live\b\|\.Shop\s*=" pkg/game/*.go
```

Each hit was classified as either an ONGOING-GAMEPLAY mutation within one game (left alone — e.g.
`Town.gold` changing at the shop, `Carried` being handed forward at `FinishMission`) or a
SESSION-INSTALL site: a point that commits to a genuinely different game replacing the one already
running. Exactly three session-install sites exist in this tree: `newGameChargen`'s `Begin` closure,
and `RestoreOriginal`'s two arms. A fourth candidate, `resume.go`'s `installCandidate` (the native
`.ags` save-load entry point), was checked and found already complete rather than assumed complete
from the round-2 review's own report: it unconditionally sets `Town`, `Carried` and `Offered`; its
`townOnly` branch clears `live`/`liveMission`/`liveParty` directly; its mission branch calls
`c.activate`, which in production is always `func() { f.liveDriver(mw, n, ms.Party) }`
(`frontend.go:1470`), and `liveDriver` (`resume.go:416-419`) unconditionally overwrites
`live`/`liveMission`/`liveParty` for the newly opened mission. `installCandidate`'s mission branch
does not touch `Shop`, and this is not a gap: `arriveInTown` (`frontend.go:434-447`) is the one place
`Shop` is ever assigned, it always rebuilds it from scratch regardless of the prior value, and the
window between an `installCandidate` mission-branch load and the next `arriveInTown` is spent inside
the newly opened mission, never at the town screen, so the stale value is never observed. All three
session-install sites now call `resetSessionForNewGame`; the method is re-runnable verbatim if a
fourth site is ever added, and the grep pattern is the same one `resetSessionForNewGame`'s own doc
comment repeats.

**A single choke point was considered and rejected.** The three sites reach the install moment from
three different call shapes — a chargen closure, two branches of one save-format parser, and a
native-save installer with its own struct — with no shared caller between them above
`FrontEnd` itself; forcing one would mean routing chargen and both save formats through a new
indirection whose only job is calling `resetSessionForNewGame`, which is exactly what each of the
three call sites already does directly. `resetSessionForNewGame` is the choke point for the RESET
ITSELF (the population of fields, in one place); there is no single choke point for WHEN to call it,
because the three moments a new game is committed to are three distinct events in this tree, not one
event reached three ways.

## Fix

`FrontEnd.resetSessionForNewGame` (`pkg/game/frontend.go`) resets `Carried` and `Offered` to their
zero values, rebuilds `Town` as `NewTown(f.Campaign)` (the same construction `NewFrontEnd` uses at
process start), clears `live`, `liveMission` and `liveParty`, and clears `Shop` — seven of the
struct's 52 fields. Every other field is install-scoped (read once by `NewFrontEnd` out of the
archive and correctly shared across every game the process opens) and is left unchanged: 45 of 52.

Three call sites, all found by the enumeration above:

- `newGameChargen`'s `Begin` closure (round 1), beside the pre-existing
  `f.townUI.resetForNewGame()`.
- `RestoreOriginal`'s `n == 0` (between-mission) arm (round 2). Calls `resetSessionForNewGame` FIRST,
  before `RestoreParty` is even invoked, not merely before `Carried` is assigned: `RestoreParty`'s own
  fallback argument is `f.NextParty()`, which returns `f.Carried` verbatim when it is non-empty
  (`frontend.go:1653-1658`). Computed against the previous game's `Carried`, a save whose own
  characters cannot be decoded would fall back to the discarded game's own party instead of a fresh
  default one — a second, narrower ordering defect this section's own test found once it began
  seeding `Carried` before exercising this arm (see Tests written). `Carried` and `Town.gold` are
  then overwritten, in that order, with what this save decodes.
- `RestoreOriginal`'s `n != 0` (mid-mission) arm (round 2, the P finding). Calls
  `resetSessionForNewGame` with no subsequent overwrite: a mid-mission original save decodes no
  town/campaign/carried-party history of its own, and the restored party is passed to the returned
  opener as an argument, never through `Carried`.

`resume.go`'s `installCandidate` needed no new call (see Root cause: verified complete, not a fourth
site).

## Tests written

- `TestResetSessionForNewGameDropsExactlyTheSessionPopulation`
  (`pkg/game/frontend_session_test.go`). Reflect-based completeness test on `FrontEnd`'s whole field
  set, modeled directly on `pkg/game/townscreen_test.go`'s
  `TestResetForNewGameDropsExactlyTheGamePopulation`. Seeds every field non-zero, calls
  `resetSessionForNewGame`, and requires every field back at its own classified value: the five
  session fields reset (`Town` compared against a freshly built `NewTown(f.Campaign)`, not merely
  nil), every other field named in a `keepUnchanged` map with its own one-line install-scope reason.
  A field added to `FrontEnd` later fails this test until it is classified.

  **Revert-to-red, performed:** removed `f.Shop = nil` from `resetSessionForNewGame`. The test failed
  with `field Shop = &game.Shop{...} after resetSessionForNewGame, want (*game.Shop)(nil)`. Restored
  the line (re-applied the edit; no file-level copy was used) and confirmed the test passes again.

- `TestLeavingToTheMenuAndPickingAMissionDoesNotResumeThePreviousGamesTown`
  (`pkg/game/sessionend_test.go`). The integration witness: reuses `continuityFront`,
  `continuityMission`, `listAdvance` and `menuAdvance` (`pkg/game/continuity_test.go`, pre-existing
  fixtures). Wins mission 20 on the fixture campaign (opens the town, sets `Carried` and `Offered`,
  matching `TestAWinAtTheTownsBoundaryOpensTheTown`'s own setup), then calls the exact production
  entry point a player's "pick NEW GAME again, choose a mission" reaches:
  `f.newGameChargen(0).Begin(ui.ChargenResult{})`. It then loses the second game's own mission 10
  through `f.continuity`, the same wrapper `TestALostMissionStillReachesTheMenuBeforeTheTownIsOpen`
  exercises, and asserts the destination is `ui.NoticeToMenu` — not `ui.NoticeToTown`, which is what
  the unfixed code produced. It also asserts `Town.Open()`, `Carried` and `Offered` are back at a
  fresh game's own values.

  **Revert-to-red, performed:** removed the `f.resetSessionForNewGame()` call from `newGameChargen`'s
  `Begin` closure. The test failed on all four assertions:

  ```
  destination 4 after the second game's first loss, want ui.NoticeToMenu — got ui.NoticeToTown,
  which means the front end is still showing the FIRST game's own town instead of a fresh one
  Town.Open() is true after starting a new game — the previous game's town survived the reset
  Carried = [{ID:temporary:1 ...}] after starting a new game, want none — the previous game's
  party survived the reset
  Offered = 30 after starting a new game, want 0 — the previous game's map-list bookkeeping
  survived the reset
  ```

  This is the owner's own reported symptom, reproduced end to end by this test before the fix.
  Restored the call (re-applied the edit) and confirmed the test passes again.

- **`pkg/game/worldmapsession_test.go`, extended (round 2).** This file (from story `1013`) already
  drove every session-install site and asserted `townUI`'s own world-map fields; it did not assert
  `FrontEnd`'s own session fields, which is exactly what made the mid-mission gap invisible to it
  while it stayed green. This is a witness ordered with the fix, not a ledger row: the file's
  existing six tests now also call `seedFrontEndSession`/`assertFrontEndSessionCleared`, the same
  seed/assert pattern the file's own `seedWorldSession`/`assertWorldSessionCleared` established one
  layer up, applied to `Carried`, `Offered`, `Town.won`, `live`/`liveMission`/`liveParty` and `Shop`.
  `TestArriveInTownAloneDoesNotClearTheMarkerCache` gained the survival-side assertion: an ordinary
  mission-to-town return within one game must leave these fields untouched, `Shop` excepted (see
  Root cause: `arriveInTown` always rebuilds it, in or out of a new game).

  **Revert-to-red, performed on `TestRestoreOriginalClearsWorldSessionMidMission`, the test the
  round-2 review named as blind.** Commented out `f.resetSessionForNewGame()` in `RestoreOriginal`'s
  `n != 0` arm (`originalsave.go`). The test failed on all five of `assertFrontEndSessionCleared`'s
  checks:

  ```
  RestoreOriginal (mid-mission): Offered = 77, want 0 — the previous game's map-list bookkeeping survived
  RestoreOriginal (mid-mission): live/liveMission/liveParty = ...want nil/0/nil — the abandoned mission's own world survived
  RestoreOriginal (mid-mission): Town still marks the previous game's own fence mission 999999 done
  RestoreOriginal (mid-mission): Carried still holds the previous game's own member
  RestoreOriginal (mid-mission): Shop is still the previous game's own merchant
  ```

  Restored the call (re-applied the same edit pair; no file-level copy was used) and confirmed the
  test passes again, alongside the rest of the file and `go test ./...`.

  **A second, narrower defect surfaced by the same seeding, in `RestoreOriginal`'s `n == 0` arm.**
  With `Carried` seeded non-empty before the call (as production sees it, whenever LOAD GAME is
  reached from a game already in progress), the original round-2 draft of the fix — reset called
  after computing `RestoreParty(sf, f.NextParty(), ...)` — left the fence-seeded member in `Carried`
  even though `resetSessionForNewGame` ran, because `f.NextParty()` had already captured the
  pre-reset `Carried` as its fallback. `TestRestoreOriginalClearsWorldSessionBetweenMissions` caught
  this directly (`Carried still holds the previous game's own member`) as soon as it began seeding
  `Carried`; reordering the reset to run before `RestoreParty` is called (Fix, above) resolved it,
  confirmed by the same test.

## Gate

Run from `<seat>\wt-1019-session-end`, on this story's branch tip before commit:

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')` — printed
  `pkg/game/frontend_session_test.go` once (a tab-alignment gap in a struct method table), fixed with
  `gofmt -w`, then empty.
- `go test -trimpath -count=1 ./...` — every package `ok`, no skip observed in `pkg/game`
  (`ok  	againrom/pkg/game	5.338s`).
- `scripts/check-no-game-assets.sh` (the glob's only two entries in this repo today) — exit 0,
  `check-no-game-assets: clean (tree scan)`.
- `scripts/check-claim-citations.sh` — exit 0 (1162 distinct citations, all resolve against the
  pinned submodule `202b8d5`). Round 1's draft of this section named the research experiment behind
  `DIV-166` by its own id, before that experiment's pin bump landed — the same forward-reference
  pattern `origin/master`'s `214671c` corrected in `DIV-166` itself, one file over, by stating the
  fact without spelling the id. This section is worded the same way now: research opened an
  experiment on the room-composition question `DIV-166` tracks, still in its own lane at the time
  this story was written; its id is not cited here, and is written in wherever it is next cited,
  at the pin bump that carries it.

Round 2, re-run in full after the merge with `origin/master` and every edit above, exit codes read
directly rather than through a pipe: `go build ./...` exit 0, `go vet ./...` exit 0, `gofmt -l`
exit 0 (empty output), `go test -trimpath -count=1 ./...` exit 0 (39 packages `ok`, no skip),
`scripts/check-no-game-assets.sh` exit 0, `scripts/check-claim-citations.sh` exit 0 (1162
citations against the tree as it stands now, including this section's own reword).

## Game census (mandatory instrument, unrelated to this story's subject)

```
go build -o /tmp/mr ./cmd/missionrun
AGAINROM_ASSETS=<en install> /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED   -> 0
AGAINROM_ASSETS=<en install> /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED   -> 0
```

Unchanged from master: this story touches no script-op support in `pkg/sim`, and the census reads
zero on both missions before and after, as it has since the widened `check-milestone.sh` last
reported here. The story's own result does not show in this census — it never reaches `pkg/sim`. It
shows in `builds/current/`, once the seat rebuilds it from master with this merged: leaving to the
main menu and starting any mission afterwards no longer resumes the previous game's own town on a
loss, which is directly the owner's reported repro. `TestLeavingToTheMenuAndPickingAMissionDoesNotResumeThePreviousGamesTown`
is that same repro run headless, and its revert-to-red result above is the evidence it actually
witnesses the defect.

Re-run at round 2, after the merge with `origin/master` and the `RestoreOriginal` fix: 0 and 0,
unchanged.

## Research

Not applicable, as stated in `contract.md`: this is an implementation completeness defect in this
tree's own long-lived process state, not a question about ROM1 behaviour. No claim is cited.

## Surface this story's own review has and has not walked

Round 1 walked and round 2 does not re-walk: all three `ScreenMenu` return paths; every mission-start
path from the picker to the leaf, including the loose-map row and the END QUEST row; both save-load
producers (`RestoreOriginal`, `installCandidate`), each now independently verified complete on both
its branches; the successor-mission continuation correctly not resetting; both round-1 tests mutated
and confirmed as real witnesses; `pkg/game` package-level vars; 15 of the 45 kept fields in
`TestResetSessionForNewGameDropsExactlyTheSessionPopulation`'s `keepUnchanged` map independently
re-derived from their own writers.

Round 2 walked three items round 1 left unreached, closing two:

- **`Options` (`FrontEnd.Options`, an `OptionsStore` backed by `options.txt`) is correctly excluded
  from the reset population.** Read (`frontend.go:400-406`): it is a persisted user preference
  (`TipsMode`), written and read across process restarts, not per-game state — a player's tip-display
  choice should survive starting a new game, the same way it survives quitting the process entirely.
  Closed: N/A, not a gap.
- **`pkg/ui` renders nothing session-specific on `ScreenMenu`/`ScreenPicker` before a mission is
  chosen.** Read `drawMenu` and `drawPicker` (`pkg/ui/app.go:2645-2705`): `drawMenu` composes from
  `a.sel.State()`, a menu-navigation selector; `drawPicker` reads `a.flow.picker`'s header and row
  text. `a.flow.picker` (`pkg/ui/picker.go`) is built once at `NewPicker` from the install's own map
  list and mutated only by `SetUnusable` (`pkg/ui/flow.go:708,714`), which marks a row when its
  `f.load(i)` call errors — a load-capability fact about the install, not about game progress, and
  correctly stable across games on the same install. Closed: N/A, not a gap.
- **~30 pure asset-bundle fields in `keepUnchanged` are accepted on their own doc comment or on
  `NewFrontEnd` being their one constructor, not individually re-derived from a writer the way the
  15 above were.** Not closed: re-deriving all of them is a size of task this round did not attempt.
  Open, for a future story or review pass that touches `FrontEnd`'s field list again.

## Divergence rows

None opened. `DIV-172`, `DIV-173` and `DIV-174` were allocated to this story and are returned unused
— found at `PIPELINE-STATUS.md`'s reservation for this story; no ledger row was written under any of
the three.

## Scope cut

None. One behaviour (a genuinely different game must start with none of the previous one's session
state, reached through three doors: NEW GAME, and LOAD GAME on both save formats), two domains
(`Campaign & Scripts`, `Town & Economy`), both inside `pkg/game`, no hashed simulation state.

## Open items

- The ~30 asset-bundle `keepUnchanged` fields named above, not individually re-derived.
- `check-claim-citations.sh` is green on this branch (above); no open item here now.
