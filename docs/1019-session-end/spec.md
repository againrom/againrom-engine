# Spec — 1019-session-end

Self-contained. Canonicalized to the behaviour actually built, at the landing.

## Behaviour

Starting a mission from the main menu's map list begins a new game session. A new game session
carries none of a previous game session's progress, even when both run in the same process.

Specifically, when a mission row is chosen from the picker and character generation is confirmed,
before the new mission's map opens:

1. The between-missions state (`FrontEnd.Town`) is a fresh one, built over the same `Campaign` the
   process loaded at startup: no chapter reached, the campaign's own starting purse, no mission
   marked finished, no offer marked available or taken, and the mercenary pool at the campaign's
   own initial counts.
2. The carried party (`FrontEnd.Carried`) is empty. A subsequent won mission fills it exactly as it
   would in a process's first game.
3. The map list's own bookkeeping of what it last offered (`FrontEnd.Offered`) is zero.
4. The currently-open mission's world, mission number and party
   (`FrontEnd.live`/`liveMission`/`liveParty`) are cleared. Nothing above the map screen still
   references a mission the player left running or abandoned before returning to the main menu.
5. The town merchant's stock (`FrontEnd.Shop`) is cleared. It is rebuilt whole, from the fresh
   `Town`'s own chapter, on the new game's first town arrival (`arriveInTown`), exactly as it always
   has been; this only removes the window in which a caller could observe the previous game's stock
   before that arrival.

Every other part of the front end — the loaded archives, the tileset, the object/unit/structure
bundles, the font, the campaign definition itself, the sound configuration, the persisted option
store, and every other install-scoped field — is unaffected and continues to serve every game the
process opens, exactly as before this story.

## Observable consequence

A mission lost before the new game's own town has been reached returns the player to the main menu.
Before this story, if an earlier game in the same process had already reached its town, the new
game's first loss returned to that earlier game's town instead — visibly showing its chapter, its
gold and its finished-mission state.

## Where the reset happens

`pkg/game/frontend.go`, `FrontEnd.resetSessionForNewGame`, called from every site in this tree that
commits to installing a genuinely different game into the one long-lived `FrontEnd`:

- `newGameChargen`'s `Begin` closure, immediately after `f.townUI.resetForNewGame()` (the
  pre-existing presentation-layer reset) and before `f.MissionOpenerWith` builds the opener for the
  chosen mission. Every mission row in the shipped map list opens character generation before its
  map (`WithMissions` in `pkg/game/maplist.go` prepends one row per mission-numbered map file;
  `newGameChargen` returns a non-nil `ui.ChargenEntry` for every row whose `Mission > 0`), so `Begin`
  runs exactly once per mission chosen from the picker — the first time a process starts a game and
  every time afterwards, including after a return to the main menu.
- `pkg/game/originalsave.go`'s `RestoreOriginal`, on both its arms (LOAD GAME with an original-format
  save). The `n == 0` (between-mission) arm calls it before decoding the save's own party, so the
  fallback `RestoreParty` uses when the save's own characters cannot be decoded is a fresh default
  party, never the discarded game's own carried one; `Carried` and `Town.gold` are then overwritten
  with what the save decodes. The `n != 0` (mid-mission) arm calls it with no later overwrite: a
  mid-mission original save decodes no town/campaign/party history of its own, and the restored party
  is passed to the returned opener as an argument, never through `Carried`.
- `pkg/game/resume.go`'s `installCandidate` (LOAD GAME with a native `.ags` save) needed no new call:
  it already installed `Town`, `Carried` and `Offered` unconditionally and cleared
  `live`/`liveMission`/`liveParty` on both its branches (directly on a town-only load, through
  `liveDriver` on a mission load), before this story began.

`pkg/ui` is not changed and carries no new callback. The three paths that return the player to the
main menu (`pkg/ui/flow.go`'s `advanceNotice`'s `NoticeToMenu` arm and its `escape()`'s
`ScreenPicker` arm, and `pkg/ui/save.go`'s `chooseGameMenu`'s `gameMenuAbortGame` arm) are unchanged;
none of them needs to observe the reset, because nothing session-specific is drawn while the main
menu is showing.

## Not covered

- **Ending a session eagerly, at the moment the player leaves for the menu.** The reset happens
  lazily, at the moment the next game is confirmed to start. The observable result — "anything
  started afterwards starts clean" — is the same either way; the lazy placement avoids adding a
  `pkg/ui` → `pkg/game` callback across the import DAG's fixed direction.
- **A plain map row (`Mission == 0`).** These do not open character generation and are not a "game"
  in the sense this story addresses; they are unaffected, exactly as before.

See `closure.md` for the enumeration method that established this is the whole population of
session-install sites, and for what a review of this story's own surface has and has not walked.
