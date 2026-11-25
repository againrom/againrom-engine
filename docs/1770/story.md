# The Multiplayer button opens the character list in both games

## Result

In both games the main menu's Multiplayer button now opens a character list page instead of doing nothing. The page
is the game's Load window titled Characters, with Accept and Cancel, and Delete in the first game. The list is empty
for now, so Accept does nothing and Delete stays disabled. Cancel or Escape returns to the menu as it was; the other
menu buttons and the Load page are unchanged.

## Authority

- Owner: multiplayer milestone 1 is a working Multiplayer button, player list, character generator, map choice and
  map entry in both games, the first game following the second game's screens; this story is the button and the list.
- MENU-168 (High; Medium that the page shown is the character page): first-game button 2 shows the character page.
- MENU-ROUTE-163 (High): the first game's page has a list with one row New Character, Accept, Delete, Rename, Cancel.
- MENU-171 (Medium for the result): the character page's Cancel returns to the main menu.
- MENU-169 (Unknown): what Accept runs on the New Character row is unread; here Accept does nothing.
- R2-ENGINE-1844 (Medium that button 1 by mask is Multiplayer; panel controls and texts Unknown): the second game's
  button shows a three-panel view whose Cancel returns to the menu.
- R2-ENGINE-1840 (High): the second game's list ends with one text entry whose OK runs creation; not built here.
- DIV-2369: what the two pages still lack (files, New Character, creation, Rename, portrait, panels, map, session).

## Rules

1. In both games the Multiplayer button opens the character list page.
2. The page is the game's Load window with the title Characters, Accept and Cancel, and an empty list.
3. The first game's page shows Delete, disabled on the empty list; the second game's page shows no Delete.
4. Accept on the empty list stays on the page and loads nothing.
5. Cancel and Escape return to the main menu; Load then opens the Load page as before.
6. The Server button and every other main-menu button do what they did before.

## Code

| kind | its one builder | change |
|---|---|---|
| main-menu action table | `pkg/render/menu/actions.go` FirstActions, SecondActions | one new action, set on button 2 in both tables |
| main-menu dispatch | `pkg/ui/app.go` stepMenu | the new action opens the character role |
| list page | `pkg/ui/savepage.go` SavePage, loadPage | character role of the load page from game data |
| list page open and close | `pkg/ui/save.go` openListPage, closeLoad; `pkg/ui/load_window.go` loadWindow | one opener for both roles; the open role is page state |
| page buttons draw | `pkg/ui/load_draw.go` composePage | a button with no rectangle is not drawn |
| game profile row | `pkg/game/gamefiles.go` gameFiles | Delete on the character page, first game alone |
| page wiring and words | `pkg/game/load_window.go` wireLoadWindow, `pkg/words` | title and Accept from the engine's words |

## Proof

| test | rules |
|---|---|
| `TestMultiplayerOpensTheCharacterListInBothGames` | 1, 2, 4, 5 |
| `TestCharacterPageIsTheGamesData` | 3, 4, 6 |
| `TestCharacterPageIsTheGamesRow` | 2, 3 |
| `TestSecondMenuServerButtonDoesNothingAndCutscenesOpen`, `TestSecondMenuLocksLoadWithoutSave` | 6 |
| `TestReleaseMultiplayerOpensTheCharacterList`, `TestReleaseSecondGameMenuButtonsReachTheirScreens` | 1, 5, 6 |

Release tests at the release: `TestReleaseMultiplayerOpensTheCharacterList`,
`TestReleaseSecondGameMenuButtonsReachTheirScreens`.

Base failures and the gate verdicts: review/story1770-mp-button/RESULT.md.

## Open debt

- Character files, the New Character entry and creation from the list wait for the next multiplayer stories (DIV-2369).
- Rename, the portrait and the original pages' layout and texts wait for an owner capture of each original's page.
- What Escape does on the original character page is Unknown (MENU-171).
