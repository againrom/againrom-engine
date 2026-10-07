# F1 help and the C key

## Intent and authority

F1 opens the original's help panel over the map and the C key arms Cast mode, as the original does. Authority: claims `MENU-051` to `MENU-056`, `TEXT-086` to `TEXT-089` and `AI-KEY-125` on the k120 knowledge pin. Divergence rows: `DIV-294` and `DIV-234` are closed, `DIV-526` and `DIV-474` are amended, `DIV-1934` to `DIV-1936` are new. `DIV-1937` to `DIV-1939` are returned unused.

## As built

F1 (`MENU-051`). `appInput.Help` is the F1 press edge. The map arm of `App.step` reads it below the popup gate, so a notice, the in-game menu and the documents panel ignore it; the town and every other screen never read it. `flow.showHelp` opens the panel through `Viewer.OpenHelp`, owned by this tier like the Pause key's notice: OK, Return and Esc close it without asking the mission driver. A press over the open panel changes nothing. An install without `help.txt` opens nothing.

Text (`TEXT-086`). `game.LoadInstallWords` reads `main/text/help.txt` whole into `ui.Words.HelpText`. No text is in the repository; tests read the install.

Panel (`MENU-052`, `MENU-053`, `TEXT-088`). The panel is the dialogue notice with a help layout (`pkg/ui/help.go`): box 488x360 at (76,60) in the 640x480 design space, no portrait, body rectangle (40,56)-(448,272) with the height rebuilt from the font (211 px for the 15 px font, 12 visible lines at pitch 17), one OK button. The text wraps at 408 px; when the wrapped lines exceed the body height a 24 px scroll bar is added and the text rewraps at 382 px. The world stops through the popup answer, as for the Pause notice. EN: 965 bytes, 35 lines, scroll range 23. RU: 1313 bytes, 48 lines, scroll range 36. The range is lines minus 12, an assumption (`TEXT-088`).

Scrolling. Up and Down move one line, Page Up and Page Down 12. A press on the bar's arrow squares moves one line, on the track one page. A reopened panel starts at the top. Authored parts are `DIV-1934`.

C (`MENU-054`, `MENU-055`). `Viewer.castKey` is read on the map arm below the popup gate. An empty selection or a selection with the foreign or structure bit set is not consumed; this build binds C nowhere else. A nonempty selection with no spell-capable object is consumed with no message, sound or state change. Otherwise `armCast` raises Cast mode and chooses no spell: the book's selected spell is unchanged, a click with none selected orders nothing, and the mode stands with the book closed (a spell chosen in the book still needs the book shown). The Cast cell press takes the same arm. One-shot cast commands (`DIV-474`) are unchanged.

## Proof

- Unit: `pkg/ui/help_test.go` (F1 open, hold, dismissal through the seam count, refusal over other popups and with no text, scroll and clamp, scroll bar presses, short text without a bar, the C key over empty, fighter, foreign and caster selections, the C mode with the book closed) and `TestThePanelCastCellNeedsAManaPool`.
- Release, EN and RU, `pkg/game/help_release_test.go`: `TestReleaseHelpPanelOverTheMap` (control: ticks advance with no panel; F1 opens it; text length, line count and 12 visible lines match `TEXT-087` and `TEXT-088`; the composed panel stays inside its box; the world tick and hash do not move over 12 steps; F1 over help changes nothing; Page Down scrolls and clamps at the end; Esc and the OK button close it and the world runs again; F1 over the in-game menu opens nothing), `TestReleaseHelpKeyDoesNothingInTown` (cold LOAD of the town: F1 changes neither the screen nor the frame), `TestReleaseCastKeyOnTheMap` (empty selection, a fighter selected, a mage selected: Cast armed with no spell and a click orders nothing). Screenshots are written to `AGAINROM_SHOT_DIR` or a temporary directory and copied to `review/story1281-f1-help/`: `help-open`, `help-scrolled`, `help-end`, `help-town-f1`, each `-en.png` and `-ru.png`. The map screen draws onto the display with no readable frame, so the panel is composed with the viewer's own layout, font and text over a flat ground.

## Open debt

- The help text colour, the scroll range and start, the OK rectangle and the scroll routes are authored (`DIV-1934`).
- Which other states change the F1 gate is Unknown (`DIV-1935`).
- The spellbook popup that the original shows on C, the caller of the returned 0 and any preselected spell are Unknown (`DIV-1936`).
- The Pause key's notice is still the 488x232 dialogue notice (`DIV-334`); `MENU-052` says the Pause panel has the help panel's class and 488x360 size.
