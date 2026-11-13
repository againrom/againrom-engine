# The starter shows every setting of a mod

## Intent

A player sets every setting a mod declares in the starter, not only the first five. The settings panel scrolls
through all of them with the wheel, as the base list and the mod list do, and each type has a control that fits
it. A value the game would refuse is shown refused in the panel with the game's own words.

## Authority

Owner direction for the mod workbench, point 2 («понять как настраивать мод»), story T5 of the mod structure
specification: the panel scrolls or pages through every declared setting of the selected mod, and `settingRows`
becomes the visible window. The starter is the project's own program; no original behaviour is involved and no
DIV row is needed.

## As-built behaviour

The list. `scrollList` (`cmd/starter/app.go`) is the starter's one scrolling list: a wheel area, a row height
and the rows in view, with `row(k)` for the rectangle of row k in view and `app.window(list, n)` for the clamped
first row and the count in view. Its offsets are `app.scroll`, one per list; the wheel step is
`widgetkit.ListWheelLines`, one row a notch, as before. The base list (5 rows of 18 px from y 66) and the mod list
(6 rows of 18 px from y 250) are its first two rows of `scrollLists`, with the rectangles and wheel areas they had;
the settings panel is the third (`settingRows` = 5 rows of 24 px under the panel heading). `wheel` moves the list
under the pointer. The starter cannot import `pkg/ui` (`internal/archtest` import table), so it does not use the
game's `scrollSpec` model; it shares the model's wheel step through `widgetkit`.

The panel. The selected mod's settings are drawn from the first row in view. When there are more than five, the
heading line shows `1-5 of 12, the wheel scrolls` (Russian: `1-5 из 12, листает колесо`), replacing the old
"N more setting(s) are not shown" line. Selecting another mod in the list returns the panel to its first setting.
A focused integer field is a row in view, so the wheel over the panel drops that focus rather than let the field
edit another setting.

Controls per declared type (`mod.ParseSettings` accepts exactly these three):

| type | control | bounds |
|---|---|---|
| `int` | a text field and `-` / `+` buttons | the label shows `(min..max)`; `-` and `+` step by one and stop at the bounds; a value outside them or not an integer steps from the nearest bound or the default |
| `bool` | a switch (`on` / `off`) | none |
| `choice` | a button that steps to the next choice and wraps | the declared choices |

Refusal. A stored value that `mod.Setting.Parse` refuses shows Parse's words beside the control in the refusal
colour, for example `1001 is outside 0..1000`, `"1x" is not an integer`, `"maybe" is not true or false`,
`"top" is not one of low, mid, high`. These are the words the game's loader gives after `mod "<id>" setting <key>: `
(`mod.ResolveSettings`); the command line, Play and Check refuse with that full loader text as before. An empty
value is the default and is not refused.

Text. Each row shows the declaration's `label` through `mod.Setting.Label` in the language of the selected base,
the language of every starter caption, with the reference language's label when the mod gave none for it. A
declaration has no description key (`settingFrom` refuses any key but `type`, `label`, `default`, `min`, `max`,
`choices`), so the panel shows the title alone; adding one would change the mod file format.

Values. Unchanged route: `settings.ModSettings` holds the stored values, `settings.store` writes them to
`[mod.<id>]` of `starter.ini` on Save settings, `modSettings` passes each stored non-empty value as
`-mod-setting <id>.<key>=<value>` in declaration order. Focusing an integer field or tabbing over it no longer
stores the default: before, `tab` and the per-frame empty `typed` call created a stored value, so a later Save
wrote `key = <default>` for a setting nobody typed into. A setting not touched keeps its stored line byte for
byte (`ini.File.Set` keeps a line whose value is equal) or stays absent.

Both games. The panel reads no game identity; the same synthetic mod draws the same panel items on a rom1 and a
rom2 base of each language.

## Proof

Focused tests, `cmd/starter`, a synthetic mod in `t.TempDir()` with 12 settings, four of each type
(`modsettingslist_test.go`):

- `TestEverySettingOfAModIsReachableByTheWheel`: every label of the 12 comes into view by the wheel, EN and RU;
  the window line reads `1-5 of 12` at the top and `8-12 of 12` at the end and does not pass it.
- `TestEverySettingIsEditedStoredAndPassed`: 10 settings edited (typed, `-`/`+`, switch, choice), including
  rows reached only by scrolling; `starter.ini` holds each value, the untouched stored `lives=4` keeps its bytes,
  the untouched `tier` stays absent; Play passes the 11 stored values in declaration order; a new starter reads
  them back.
- `TestOutOfBoundsValuesAreRefusedWithTheLoadersWords`: the row's words equal the tail of `mod.ResolveSettings`'s
  error; Play refuses with the full loader text; `+` from 1001 gives 1000; `-` stops at the minimum; a non-integer
  steps from the default; a stored bool and choice the loader refuses show its words.
- `TestFocusOrTabWithoutTypingStoresNothing`, `TestTheWheelOverThePanelDropsASettingFocus`.
- `TestThePanelIsTheSameOnBothGames`: rom1-en against rom2-en and rom1-ru against rom2-ru, at the top and at the
  end. With `AGAINROM_STARTER_MOD_FRAMES` set it writes the eight frames.
- `TestTheListsKeepTheirRows`: the base and mod list rows and wheel areas are the old rectangles, and the wheel
  over one list moves no other. The existing `TestWheelScrollsOnlyOverLists`, `TestStarterModListFrames` and
  `TestStarterFramesKeepTheirPixels` pass unchanged.
- `TestPanelCaptionsFit`: every panel text fits its width in EN and RU.

Release tests: none draws the starter or runs it (search of `*_test.go` for the starter package and program name:
only `cmd/starter` tests and `internal/archtest` guards). The release tests that pass mod settings to the game
are `TestReleaseLoadPageListsEveryModMismatch` and `TestReleaseSecondGameLoadPageListsModMismatches`
(`pkg/game/modmismatch_release_test.go`), `TestReleaseModSpellKeysEditTheInstalledTable` and
`TestReleaseModSpellEditsReachACastAndTheSave` (`pkg/game/modspell_release_test.go`); they build the settings
themselves and do not pass through the starter. The argument form is unchanged, so they witness the game side of
the same arguments on EN and RU at the release.

## Open debt

- A setting has no description in the declaration format; one needs a format change and the mod settings page of
  the workbench (P15) to write it.
- The panel has no scroll bar or page keys, as the two other starter lists have none; the window line and the
  wheel are the only means.
- A choice steps forward only, as before.
