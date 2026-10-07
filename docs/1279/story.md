# Mod mission abandon and restart

## Intent and authority

A mod adds in-game menu actions that leave the running mission: abandon it and return to the town it was entered from, or restart it. Authority is owner direction on tester item 29 (the Allods 2 action to refuse a mission and return to the town, and the missing restart): this is mod work, off in the unmodded game. The seam is the screen registry and the screens data file of `docs/1277/story.md`. ROM1 differences are rows `DIV-1906` to `DIV-1909` in `docs/divergences/mods.md`. `DIV-1910` and `DIV-1911` stay unused: the SAV case is added to `DIV-1907` to `DIV-1909`. Without a mod every menu frame, hash, SAV byte, scenario and release witness is unchanged.

## As built

Declaration. `data/screens.toml` takes `[[action]]` tables beside `[[screen]]` (`pkg/mod/screens.go`):

| key | meaning |
|---|---|
| `key` | identifier, `a-z0-9_`, unique in the file |
| `action` | `abandon` or `restart` |
| `menu` | text key: the entry's label, 1 to 40 characters |
| `confirm` | optional text key: the confirming row's label, the entry's label by default |

An action has no `place`: it is a row of the in-game menu over a mission. It takes a game menu slot of the 2 shared with screens, and one mod adds each action (a second mod adding it is refused naming the first). Refusals name mod, file and line. The conversion to the presentation tier is `game.ModScreens`, which maps the two kinds to `ui.ModActionAbandon` and `ui.ModActionRestart` of the existing `ModScreen` list; the front end gained no field.

Menu (`pkg/ui`). The root appends one row per action after the shipped rows, only when the game menu context says `LeaveToTown`; the town menu never has one. The row opens a confirming page of two rows, the confirming label and the install's Return word, with Return selected first; Return and Escape go back with nothing changed. The confirming row calls `flow.runModAction`, which asks the map screen's advance seam for the new `NoticeAbandon` or `NoticeRestart` action and takes the answer through `goNoticeDest`, the destination switch factored out of `takeNoticeAction`. No completion movie is requested. A label wider than its row (246 pixels in the install font) is clipped with `...`.

Rule (`pkg/game/missionleave.go`). `canLeaveMission` holds when the town is open, the mission number is positive and the world has no outcome yet. The menu context sets `LeaveToTown` from it, and `continuity` answers the two actions before the completion logic: `leaveMission` refuses (`NoticeStay`) when the rule fails.

- Abandon answers `NoticeToTown` after `beginWorldMapReturn`, the homeward travel a won mission uses. No completion, payment or economic arrival runs.
- Restart answers `NoticeToMission` with `MissionOpener(n)`, built from the party and purse the town holds.

Entry state versus current party. For a mission the front end opened from the town, the town returned to is the town as it entered the mission. A mission works on copies of the party, items, purse and hired pools; only `FinishMission` writes them back. So the party, purse, offers and finished count are those held at entry, with no snapshot to drift. Two entry effects remain, both written at entry by the ordinary door: the mission's documents are granted, and the mission is the selected one. Loot, experience and gold gained in the mission are lost.

Mission restored from a SAV. A mission LOAD installs the mission's party in the driver and no town party (`Carried` is empty), and the SAV holds no entry town. Order of choices taken: the SAV carries the party only as it stood at SAVE, so there is no separate entry party to restore; the live party is therefore carried home. `carryMissionHome`, the party-and-purse half of a won mission's return, runs without any win, payment or fame: survivors with the loot and experience they hold, city objects, hired pools and purse become the town's, and the town SAVE after arrival writes. Hired men go back to the tavern pools as at a win (the original's end-of-mission cull) and are not in the town party. Restart from a restored mission carries the party home first and opens the new mission with it. A mission the restarted driver opens is a fresh entry: leaving it returns the carried party as entered. A driver records whether it was restored (`resumed`).

Quick-spell bindings. The viewer holds a pointer to the front end's bindings, so a binding changed in the mission would reach the town. The driver records the bindings it became live with, and leaving puts them back.

Successor mission. A mission the front end opens directly after a win, with the town open, shows the row. Its entry state is the post-win town and party, so abandoning it returns there; it is not restored, so the first rule applies. No release witness builds this case: the shipped chapter-30 wins end in the town, so the route needs a campaign state not reachable from the install's town chapter.

SAV. Nothing is added to the save. The town SAV written after abandoning carries the `AgainromMods` leaf under the mod set as every modded SAV does, and is byte-equal to the town SAV written before the mission was entered (9452 bytes on the EN install, the same on RU). A cold LOAD under the mod restores the same party, gold and offers; LOAD without the mod is refused by `DIV-1802`. A mission SAV taken before abandoning is unaffected. No field is engine debt.

Restart. Built, since it falls out of the same mechanism. The restarted mission is the first entry's world again (tick and hash), with the entry purse; it can be left again.

Example mod (`pkg/modrt/testdata/mods/mission-abandon`, test data): one action, `abandon`, labelled "Abandon mission" with the confirming row "Abandon, return to town" (Russian «Отказаться от миссии», «Отказаться и уйти в город»). The owner's longer phrase does not fit a 246 pixel row in the install font (298 pixels in English, 365 in Russian), so the phrase is split between the entry and its confirming row. It uses one of the 2 game menu slots, so it loads beside `heavy-armor`. The seat copies it into `builds/current/mods/`.

## Proof

- Unit: `pkg/mod` (`[[action]]` parse, each refusal with file and line, one mod per action), `pkg/modrt` (the example mod in both languages, a second mod adding the action refused), `pkg/ui` (`modaction_test.go`: rows only over a mission that can be left and never in the town, the confirming page and its default row, abandon and restart through the seam, a refusal that stays in the menu, label clipping, registration refusals).
- Release, EN and RU, one root at a time (`pkg/game/modabandon_release_test.go` and `modabandonloadparty_release_test.go`, gated by `AGAINROM_ASSETS`):
  - `TestReleaseModAbandonOfALoadedMissionBringsTheSavedPartyHome`: a mission 30 entered from a 2-member town, played 40 steps and SAVEd, is loaded cold in a fresh front end (`Carried` empty). A quick-spell binding is changed, the mission abandoned and the party walked home: the town holds the 2 members with their worn items and pack items, gold equals the loaded purse, the bindings are the loaded ones, the town SAVE writes and a cold LOAD returns the same party; mission 30 entered again opens with that party. Loss control: with the carry switched off the town holds no member, the SAVE is refused with `native city actor population exceeds archive object space` and a restart opens with the default hero.
  - `TestReleaseModRestartOfALoadedMissionKeepsTheSavedParty`: the same SAV restarted opens with the saved party, and abandoning the restarted mission brings it home, SAVE and cold LOAD included.
  - `TestReleaseModAbandonOfAnOriginalMissionSAV`: corpus SAVs `2027-09-07/game0005.sav`, `game0011.sav`, `game0028.sav` (mission 30, loaded with unmarked-SAV acceptance, read in place) are abandoned and restarted: the non-hired party comes home, the town SAVE writes and a cold LOAD returns it. `game0030.sav` does not load as a mission and is logged.
  - `TestReleaseModAbandonReturnsToTheTownAsEntered`: a town SAV after mission 20 is loaded through the load window, mission 30 is entered through the Gates and its scroll, played 60 steps, and its purse raised by 777. The menu has 8 rows against 7, the entry label is matched as install font glyph pixels, the confirming page has 2 rows and Return gives the mission back unchanged. Confirming lands in the town on the homeward route with the party, gold, finished count and offers equal to entry, mission 30 not done and offered. After arrival the town SAV is byte-equal to the one written before the mission and carries the mod leaf; a cold LOAD restores the entry state and LOAD without the mod is refused. Loss control: a completion of the same mission with the same purse raises the town gold by at least 777 and marks the mission done. The mission is entered again with the entry purse.
  - `TestReleaseUnmoddedMissionMenuHasNoAbandonEntry`: without the mod the menu has 7 rows; with the mod over mission 10 opened with no town it has 7 rows and the panel is pixel-equal to the unmodded panel.
  - `TestReleaseModRestartReopensTheMissionFromItsEntry`: a test-built mod with both actions; after 80 steps and a purse change the restart gives tick and hash equal to the first entry and the entry purse; the town is unchanged and the restarted mission is left again by abandon.
  - Offscreen screenshots, never skipped, written to `AGAINROM_SHOT_DIR` or a temporary directory: `game-menu-with-abandon-<root>.png`, `abandon-confirm-<root>.png`, `game-menu-with-restart-<root>.png`, `town-after-abandon-<root>.png`.

## Open debt

- A label wider than 246 pixels is clipped; the panel is not widened for it.
- The party walks home along the world map and a town SAVE is blocked until it arrives, as after a won mission; an instant return is not built.
- The documents granted at mission entry stay after abandoning (`DIV-1907`).
- - Restart does not carry the mission's cutscene or dialogue state.
- Abandoning a restored mission keeps what it gave the party (loot, experience) and returns its hired men to the pools; the entry state of a mission opened from the town loses both.
- The abandon and restart rows show only over a mission with a standing town; a mission from the map list has neither.
