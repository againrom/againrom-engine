# Contract — 1019-session-end

## Result

After this story, leaving a running game for the main menu (the brooch) and then starting any
mission from the picker begins a genuinely fresh game session. The new game's town opens with no
chapter progress, no gold beyond the campaign's own starting purse, no finished or offered missions,
no carried party and no merchant stock left over from the game that was left. A mission lost before
the new game's own town has ever been reached returns to the main menu, not to the previous game's
town.

The owner reported this on 2026-08-19: "помимо этого есть баг, что состояние игры не стирается даже
когда выходишь в главное меню (brooch). Можно даже выбрать любую миссию оттуда и умереть и
почему-то загружается состояние которые было вообще до выхода в гланое меню (например загруженный
город)."

**Observable result.** Run `builds/current/againrom.exe -assets <install>`, start a campaign, play
until the town opens (win a main mission), leave to the main menu, pick any mission, and lose it.
The game returns to the main menu, not to a town carrying the previous game's chapter and gold.

## Root cause

`pkg/game/frontend.go`'s `FrontEnd` struct holds five kinds of per-session state: `Carried` (the
party the last won mission left behind), `Offered` (the mission the map list last offered), `Town`
(the between-missions state — gold, chapter, finished/available/taken offers, mercenary pool), `Shop`
(the town merchant's current stock) and `live`/`liveMission`/`liveParty` (the currently-open
mission's world, mission number and party). All five are built once — `Town` and the others at
`NewFrontEnd` (process start), `live` etc. only by opening a map — and none of them is ever cleared
by any path that returns the player to the main menu.

Three code paths reach `ScreenMenu` from a running game (`pkg/ui/flow.go`): a lost mission's own
notice (`advanceNotice`'s `NoticeToMenu` arm), Escape from the map list (`escape()`,
`case ScreenPicker`), and the in-game menu's ABORT GAME row at the town square
(`chooseGameMenu`'s `gameMenuAbortGame` case, `pkg/ui/save.go`). None of the three calls back into
`pkg/game`: `pkg/ui` has no notion of a game session, and the import DAG (`pkg/game` imports
`pkg/ui`, never the reverse) means it cannot.

The one place a new game is confirmed to start is `pkg/game/frontend.go`'s `newGameChargen`, whose
`Begin` closure runs when the player confirms character generation for a mission row picked from the
map list. Every mission row in the shipped map list opens generation (`WithMissions` prepends one row
per mission-numbered map file, and `newGameChargen` returns a non-nil `ChargenEntry` for every row
whose `Mission > 0`), so this closure is reached exactly once per "start a mission from the picker,"
including every time the player returns to the main menu and picks again. Before this story it called
only `f.townUI.resetForNewGame()` — the presentation-layer reset built for story `0142`/`DIV-128`/
`DIV-137`/`DIV-138` — and never touched the five `FrontEnd`-level fields above.

The consequence matches the report exactly. `f.continuity` (the win/lose destination wrapper) routes
a lost mission to `ui.NoticeToTown` whenever `f.Town.Open()` is true, and to `ui.NoticeToMenu`
otherwise (`TestALostMissionReturnsToTheTownOnceItIsOpen` / `TestALostMissionStillReachesTheMenuBeforeTheTownIsOpen`,
`pkg/game/continuity_test.go`, pre-existing). `Town.open` latches true the first time a campaign
reaches a town and is never cleared by anything already in this tree. A second game in the same
process therefore inherits the first game's own open latch — and, through it, its chapter, gold and
finished/available/taken state — the moment its first mission is lost, before that game's own town
has ever been reached.

## Fix

`pkg/game/frontend.go` adds `resetSessionForNewGame`, called from `newGameChargen`'s `Begin` closure
beside `f.townUI.resetForNewGame()`. It resets the five fields above: `Carried` and `Offered` to
their zero values, `Town` to a fresh `NewTown(f.Campaign)` (the same construction `NewFrontEnd` uses
at process start), `live`/`liveMission`/`liveParty` to nil/zero/nil, and `Shop` to nil (rebuilt whole
by `arriveInTown` on the new game's own first town arrival; reset anyway so no caller can ever
observe the previous game's stock in the window before that arrival).

Every other `FrontEnd` field is install-scoped — read once by `NewFrontEnd` out of the archive and
correctly shared across every game the process opens — and is left unchanged. This is verified by a
new reflect-based completeness test enumerating the whole struct
(`TestResetSessionForNewGameDropsExactlyTheSessionPopulation`,
`pkg/game/frontend_session_test.go`), on the model `pkg/game/townscreen_test.go` already established
for `townScreen`'s own per-game reset: a field added to `FrontEnd` later fails that test until it is
classified as reset or kept-with-a-reason.

## What was NOT changed

`pkg/ui` is untouched. No new callback crosses the `pkg/game` → `pkg/ui` boundary; the reset happens
where the new session is confirmed to begin (chargen's `Begin`), not where the old one was left,
because the three "leave to menu" paths have nowhere in `pkg/ui` to call back into and nothing there
needs to observe the reset — the town screen and the picker draw nothing session-specific while the
player is looking at the main menu.

Loading a save (`originalsave.go`, `resume.go`) is a second door onto the same population, not a
separate concern. `resume.go`'s `installCandidate` already replaced `Town`, `Carried`, `Offered` and
(through `liveDriver`, for a mission-branch load) `live`/`liveMission`/`liveParty`, unconditionally,
before this story began. `originalsave.go`'s `RestoreOriginal` did not: its between-mission arm set
`Carried` and rebuilt `Town` but left `Offered` and the live triple untouched, and its mid-mission arm
touched no `FrontEnd` field at all — only `townUI`'s own presentation. Round 2 of this story's own
adversarial review found the mid-mission gap as a player-visible defect reached through LOAD GAME, and
both `RestoreOriginal` arms now call the same `resetSessionForNewGame` this story added for chargen.
See `closure.md` for the full population and the enumeration method.

## Research

Not applicable. This is not a question about ROM1 behaviour: it is a completeness gap in this
implementation's own long-lived process state, established by direct code reading
(`pkg/game/frontend.go`'s own field comments, each of which documents its field's lifetime as "the
life of the process" or "for a mission" and none of which anticipates a second game in the same
process) and confirmed by a test that reproduces the reported symptom before the fix and passes after
it. No research claim is cited and none is needed.

## Twelve aspects

| aspect | applies | why |
|---|---|---|
| data | no | no new data format read |
| runtime state | yes | the five `FrontEnd` session fields |
| simulation | no | `pkg/sim` untouched; `live` is nilled, not the `sim.World` it pointed to |
| player input | no | no input path added or changed |
| AI | no | — |
| UI/HUD | no | no `pkg/ui` change |
| triggers/scripts | no | no script/trigger rule changed |
| inventory/equipment | no | `Carried` is cleared wholesale, not edited; no equipment rule touched |
| persistence/save-load | yes | `RestoreOriginal`'s two arms route through the same reset as chargen; see `closure.md` |
| campaign/session | yes | this is the story's whole subject |
| shipped content | no | applies uniformly, independent of asset root or content |
| interactions with existing mechanics | yes | `f.continuity`'s win/lose destination selection, `arriveInTown`'s shop generation, `townScreen.resetForNewGame` |

## Domains

**Campaign & Scripts** (6) — the campaign/session state (`Town`, `Carried`, `Offered`) and the
mission-flow wrapper (`f.continuity`) that reads it. **Town & Economy** (7) — the merchant stock
(`Shop`). Both domains live inside `pkg/game`; no hashed simulation state is reached and no
`pkg/sim` file changes.

## Out of scope

- **`pkg/ui`'s own screen transitions.** Read for this story (`flow.go`, `save.go`) but not changed;
  the fix is entirely on the session-start side in `pkg/game`.
- **A `pkg/ui` → `pkg/game` "left to menu" callback.** Would let the reset happen eagerly at the
  moment of leaving rather than lazily at the moment of starting again. Not needed for the observable
  contract ("anything started afterwards starts clean") and would cross the import DAG's direction;
  out of scope for a defect fix of this size.
- **Any change to what a save records or restores.** Untouched; see "What was NOT changed" above.

## Expected divergence rows

None. `DIV-172`, `DIV-173` and `DIV-174` were allocated to this story and are returned unused: this
is an implementation completeness defect, not a mismatch between this build and researched ROM1
behaviour, and no divergence row applies. `closure.md` records the same finding after implementation.

## Review ceiling

**Three passes.** One behaviour, two domains inside `pkg/game`, no hashed simulation state.
