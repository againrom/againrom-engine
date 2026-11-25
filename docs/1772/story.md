# Multiplayer map choice lists the multiplayer maps in both games

## Result

Both games now have a multiplayer map choice page that lists the multiplayer maps in the game root map folder. In the
first game it keeps maps whose type-0 word at offset 0x70 is at least 2 and hides the others; in the second game it
lists the loose root maps and shows each map's level on its row. Unreadable map files are left out and counted
without crashing, and nothing is written to disk. Selecting a map and confirming with OK returns the chosen map to
the caller, while Cancel or Escape returns to the caller with nothing chosen.

## Authority

- Owner: multiplayer milestone 1 requires a working map choice in both games, with the first game following the second
  game's screens, and no connection or protocol step between the character list and map choice.
- SESS-100 (High for the filter; Medium for the scanned directory): the first game's map dialog hides a map whose
  type-0 `+0x70` is below 2.
- R2-ASSET-403 (High): the second game's loose root map census is EN 13 and RU 37.
- R2-ENGINE-1842 (Medium): the second game's map header dword `0x88` holds the map level.
- R2-SESSION-506 (High): `srv.cfg` `[Maps]` candidate rows; not read in milestone 1.
- R2-ENGINE-1846 (High for screen bodies; Unknown for panel elements): the second game's map choice screen.
- MENU-169, MENU-170, R2-ENGINE-1845: original Select Connection and protocol pages skipped by owner ruling.
- DIV-5196: no connection or protocol step between the character list and map choice in either game.
- DIV-5197: engine panel, `pkg/words` captions, and loose root maps directly rather than `srv.cfg` `[Maps]`.

## Rules

1. The first game lists decoded loose root maps with type-0 `+0x70` at least 2 and drops lower values.
2. The second game lists decoded loose root maps and shows the level from `0x88` on each row.
3. Unreadable map files are omitted and counted without crashing; archive and mission rows are excluded.
4. Choosing a map and confirming with OK or Enter returns the chosen map and restores the caller screen.
5. Pressing Cancel or Escape returns to the caller screen with nothing chosen, and OK is disabled when empty.
6. Title, button and level captions come from the engine's word tables in the install language, and nothing is written.

## Code

| kind | its one builder | change |
|---|---|---|
| map header decoder | `pkg/formats/alm/alm.go` `OpenInfo`, `pkg/formats/alm/rom2.go` `decodeMetaROM2` | name `SecondMetaLevel` (`0x88`) read into `Info.Word74` |
| map list and row builder | `pkg/game/maplist.go` `BuildMapList`, `MapPickerRows`, `MultiplayerMaps` | add data-driven `MultiplayerMapRule` filter and shared row builder |
| game profile data | `pkg/game/gamefiles.go` `firstGameFiles`, `secondGameFiles` | supply `MultiplayerMapRule` per game (`MinWord70: 2` vs `ShowLevel: true`) |
| engine words | `pkg/words/text/en/strings.toml`, `pkg/words/text/ru/strings.toml` | add `mapchoice.title`, `ok`, `cancel`, `level`, `empty` |
| list page | `pkg/ui/savepage.go` `SavePage`, run by `stepPage` and `composePage` | add the map role (`mapRole`, `OpenMapChoice` in `pkg/ui/mapchoice.go`) as data, beside the character role |
| front-end entry point | `pkg/game/mapchoice.go` `OpenMultiplayerMapChoice`, `pkg/game/frontend.go` `App` | wire words and filtered items to the page's map role |

## Proof

| test | rules |
|---|---|
| `TestSecondGameOpenInfoReadsLevelWord88` | 2 |
| `TestMultiplayerMapChoiceFiltersFirstGameAndCountsBadFiles` | 1, 3, 4, 5, 6 |
| `TestMultiplayerMapChoiceListsSecondGameWithLevelAndCountsBadFiles` | 2, 3, 4, 5, 6 |
| `TestMultiplayerMapChoiceWordsFollowInstallLanguage` | 6 |
| `TestMapChoiceSelectOkAndCancel` | 4, 5 |
| `TestMapChoiceDrawsEachCaptionAtItsOwnButton` | 4, 5 |
| `TestReleaseFirstGameMultiplayerMapChoiceCountsInstalledMaps` | 1, 3, 5 |
| `TestReleaseSecondGameMultiplayerMapChoiceCountsInstalledMaps` | 2, 3, 5 |

Release tests at the release: `TestReleaseFirstGameMultiplayerMapChoiceCountsInstalledMaps`,
`TestReleaseSecondGameMultiplayerMapChoiceCountsInstalledMaps`.

## Open debt

- The page is not yet reached from the menu: the step that links the character list's OK to `OpenMultiplayerMapChoice`
  is still to do (the character list has landed).
- The level prints as the unsigned dword `0x88`; R2-ENGINE-1842 does not say the field is signed.
- Entering the chosen multiplayer map waits for the multiplayer map entry story.
- Original Select Connection and protocol pages are skipped for milestone 1 (DIV-5196).
- Original map choice panel elements are Unknown (R2-ENGINE-1846) and `srv.cfg` `[Maps]` is not read (DIV-5197).
