# The player's own mission keys, both games

## Result

In a mission of either game, the game menu has a Keys row. Its page lists every mission action with its keys.
The player clicks a row or presses Enter on it, then presses the new key with Shift, Ctrl or Alt if wanted; Esc
cancels. The action then answers exactly that chord and no longer answers its own keys.

A key another action held moves to the new action; that action keeps its other keys and shows "not assigned"
only when none is left. Defaults gives the game its own layout back. The keys are kept in the player's profile
settings, apart for the first and the second game, and never enter a saved game. The spellbook's slot marks show
the player's keys.

## Authority

- Owner: «Надо в игру добавить возможность перебиндивать»; the seven proposals answered «Как рекомендовано».
- Owner: the game menu's rows sit with equal room over the first and under the last.
- Seat: one settings page builder first, the Keys page as its data; the map's other keys are refused at capture.
- DIV-4939: the original programs have no rebinding; owner direction.

## Rules

1. When the player binds an action to a chord, exactly that chord fires it; W does not fire on Ctrl+W.
2. When the bound chord fired another action, that action loses that key only; none left reads "not assigned".
3. When the player picks Defaults, the game reads its own mission key table again, row for row.
4. When the profile is read again, each game gets its own keys back; one game's Defaults leaves the other's.
5. When a waiting row gets F1, Enter, an arrow, Tab, PageUp, PageDown, Alt+letter or a debug chord, it refuses it.
6. When the player saves after a rebind on the Keys page, the SAV is the one written without it.
7. When the town or a menu is up, the player's mission keys do not apply.
8. When a spell slot's key is rebound, its Shift form follows it; the book's mark shows the key, drawn short if wide.
9. When Game Options or Sound Options is drawn or a slider dragged, it is as it was before the page builder.
10. When the profile is read, no chord fires two actions: the later entry loses it; fixed chords stay fixed.
11. When a Keys row waits, the pressed chord goes only to it, and the row shows the whole action name.
12. When a scripted or replayed run reads a key, it reads the game's own mission key table.
13. When the mission game menu is drawn, the room under its last row equals the room over its first; town as before.

## Code

| kind | its one builder | change |
|---|---|---|
| mission key binding | `pkg/ui/missionbindings.go` `missionBindings` | `fixed` rows, a profile store name; the reader reads `missionKeyRows` |
| player key layout | `pkg/ui/playermissionkeys.go` `missionKeyRows`, `takeOver` | a layer over the table: exact chords per listed action |
| profile settings | `OptionsStore.readAll`, `writeAll` | `FirstGameKeys.<action>`, `SecondGameKeys.<action>` entries |
| settings page | `pkg/ui/settingspage.go` `settingsPicture`, `stepSettingsPage` | new; Game Options, Sound Options and Keys are its data |
| game menu row, panel | `menurows.go` `firstGameMenuRows`, `gamemenu.go` `gameMenuPanelRectFor` | the Keys row; the panel's height follows the rows |
| engine words | `pkg/words/text/{en,ru}/strings.toml` | `menu.keys`, `keys.*` |

## Proof

| test | rules |
|---|---|
| `TestReboundMissionKeyFiresOnExactlyThePressedChord`, `TestReboundMissionKeyFiresAndTheDefaultKeyNoLonger` | 1 |
| `TestBindTakesOnlyTheChordOfATwoKeyAction`, `TestStolenMissionKeyLeavesTheOldActionUnassigned` | 2 |
| `TestMissionKeyDefaultsRestoreTheTable`, `TestKeysPageRebindsAndMovesATakenKey` | 3 |
| `TestMissionKeysSurviveAProfileReloadPerGame`, `TestMissionKeyLayoutListsEachGamesActionsAndReadsBack` | 4 |
| `TestFixedMissionKeysCannotBeCaptured`, `TestReboundSpellSlotNeverReachesADebugChord` | 5 |
| `TestMissionKeyRebindLeavesTheSaveUnchanged`, `TestMissionKeyLayoutStaysOffTheTown` | 6, 7 |
| `TestReboundSpellKeyKeepsItsShiftAndMarksTheBook`, `TestLongQuickSpellMarkStaysInsideItsCell` | 8 |
| `TestGameOptionsSpeedDragLeavesTheFocusWhereItWas`, `TestSoundOptionsDragReleasedOutsideTheWindowLeavesTheFocus` | 9 |
| `TestProfileReadNeverLetsOneChordFireTwoActions` | 10 |
| `TestWaitingKeysRowEatsItsChord`, `TestWaitingKeysRowShowsItsWholeName` | 11 |
| `TestScriptedKeyReadsTheGamesTableNotThePlayersLayout` | 12 |
| `TestMissionMenuPanelGrowsWithItsRowsAndTownStaysDecoded`, `TestTownMenuPanelFramesStayAsTheyWere` | 13 |

Release tests at the release: `TestSettingsPagesFramesInEachLanguage` EN and RU of both games (rule 9 with
`AGAINROM_SETTINGS_BEFORE`); the release witnesses that picture the mission game menu are recorded again.

## Open debt

- F1's help text comes from the installed data and still names the default keys.
