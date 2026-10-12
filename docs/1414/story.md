# Story 1414: the second game's main menu and its load and save pages

## Intent

Each button of the second game's main menu does what its claims say, picked through the menu mask,
and Load Game is locked while no save file exists. Load Game opens the second game's own load page and
loads a chosen save. The in-game Save opens the second game's own save page: an Empty Slot row, the
listed saves, overwrite of a chosen row without confirmation, and the typed label written into the
SAV. The multiplayer buttons stay without behaviour.

## Authority

Second-game claims at pin k267:

- `R2-ENGINE-436`: button i is the mask value 0x80+0x10*i; a disabled button draws nothing and its
  release posts nothing.
- `R2-ENGINE-437`: the eight buttons post New Game, 0x426, Cut Scenes, Credits, Load Game, 0x487,
  Hall of Fame and Exit; Load is disabled when no `game*.sav` exists in the directory, whatever it
  holds.
- `R2-ENGINE-438` and `R2-ENGINE-439`: the load page's frame, caption rows 151 and 24, the list of
  10 rows under header row 25, the row hint format, Ok, Delete and Cancel, and the delete
  confirmation from patch.txt rows 50 and 51.
- `R2-ENGINE-440`: the save page's caption row 29, prompt row 30, row 144, the list under header row
  31, Empty Slot row 28, the smallest unused slot number, selection by equal label and overwrite
  without confirmation.
- `R2-ENGINE-441`: the in-game menu reaches Save and Load.
- `R2-ENGINE-442`: Cut Scenes, Credits and Hall of Fame.
- `R2-ENGINE-443`: the name box holds 0x7c bytes in EN and 0x80 in RU.
- `R2-SESSION-179`: the listing and the hidden slots 9998 and 9999.
- `R2-ASSET-097` and `R2-ASSET-098`: the menu art and mask, and the dialogs.txt rows.

## As built

- **Menu actions as data.** `pkg/render/menu/actions.go` holds one button table per game
  (`FirstActions`, `SecondActions`): each button's action and headless name, whether Load needs a
  save file, and whether Cut Scenes needs a seen scene. `menu.Load` and `menu.LoadSecond` attach
  their game's table. `ui.App.stepMenu` dispatches on the bound action, so no game branch exists in
  the menu. A locked button has no hit: no overlay, no caption, no release, and the L key does
  nothing. Multi Player and Server bind no action (DIV-2369).
- **One page builder.** `ui.SavePage` is the one builder of the list page kind: panel, caption
  texts, list rectangle and rows, message line, three buttons, the delete confirmation, and for the
  save role the name box, its limit, the Empty Slot row, hidden names and the slot rule.
  `firstLoadPage` is the first game's load window as data, with its existing geometry, words and
  draw order; the first game's load window is built through it and its frames are unchanged. The
  second game's pages are `secondSavePages` in `pkg/game/savepages.go`, wired through the
  `gameFiles` table (`savePages`, `ownSaves`).
- **Lists.** Both pages list every `.sav` in the engine's save directory whose current-session
  leaf names the second game, by label, newest last write first, ties by name descending; the
  selected row's message line is `label  DD.MM.YYYY  hh:mm:ss` (DIV-3079). The save page hides
  `game9998.sav`, `game9999.sav` and the engine's quick and timed slots.
- **Save.** Ok writes the selected row's file, or for the Empty Slot `gameNNNN.sav` with the
  smallest number from 0 to 9997 that no listed save carries and no file holds (DIV-3080). The
  label is the name box text, written through the sole SAV producer (DIV-3081). A chosen row
  whose label equals the name box text is the file; only when no row is chosen, or the chosen
  row's label differs, the first listed row (newest write first) with that label is selected and
  overwritten without confirmation (DIV-3080).
- **Captions.** Every caption, the Empty Slot text and the button words come from dialogs.txt and
  patch.txt in the install's font code page; the Empty Slot row is decoded to runes like a typed
  label, so it draws in RU.
- **Other buttons.** Cut Scenes opens only with a seen cutscene; Credits and Hall of Fame keep
  their screens (DIV-3082). Keys and sounds are DIV-3083; geometry is DIV-3078.

## Proof

- `pkg/ui/savepage_test.go`: `TestSavePageChosenRowBeatsAnEqualLabelAboveIt` (two saves of one
  label: the chosen lower row is written; the earlier lookup order fails it), the Load lock, the multiplayer buttons, Cut Scenes, the slot rows,
  overwrite without confirmation, the name limit, a refused other directory and Ok without a save.
- `pkg/game/savepages_test.go`: the slot rule, the `game*.sav` match and the row hint.
- `pkg/game/secondsavepages_release_test.go` on rom2-en and rom2-ru:
  `TestReleaseSecondGameMenuLocksLoadUntilASaveFileExists` (hover frames with and without a
  file, loss control on a non-`game*.sav` name), `TestReleaseSecondGameMenuButtonsReachTheirScreens`
  and `TestReleaseSecondGameSavePageWritesSlotsAndLoadPageLoadsThem` (Empty Slot row drawn and
  decoded, two SAVEs into slots 1 and 2, a resave over a chosen row, a cold LOAD from the main
  menu's load page with equal World state and sample, 16 next ticks, and the list order following
  the last write time) and `TestReleaseSecondGameSavePageOverwritesTheChosenOfTwoEqualLabels`
  (two saves with one label: Ok on the chosen lower row rewrites its file and leaves the upper
  file's bytes and write time; a cold LOAD in a separate front end lists both and loads the
  written one with equal World state).
- The ROM2 profile witness `pkg/game/testdata/profilewitness/rom2-{en,ru}.txt` changes only the
  save-page frame hashes.

## Open debt

- The first game's save dialog stays its own builder; only its load window moved onto the page
  builder.
- The pages draw over the main-menu art also when opened from a mission or the town.
- The native `Bsg&` SAV is neither read nor written; the engine's own document stands in
  (DIV-3079).
- DIV-3077, DIV-3078, DIV-3080 to DIV-3083 are UNKNOWN on unread windows; DIV-2369 waits for
  multiplayer.
