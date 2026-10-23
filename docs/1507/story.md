# Base and mod launch arguments as rows of the launch table

## Intent

Every argument the starter passes to the game, and the game parses, comes from one row of the launch
table in `pkg/launch/settings.go`: key, flag, kind, whether it repeats, which row it needs, and the
flag's help text. The starter builds its argument vector from the rows only; the game registers each
such flag from the rows only, with the row's help text; the map editor's test play builds its game
arguments from the same rows. Behaviour is unchanged: the same vectors element by element, the same
parsed values, the same `-help` output and the same settings file bytes.

## Authority

- Owner decision 2 on audit `pipeline/reviews/arch-audit-027ef24e.md`, rank 29: the base and mod
  arguments become table rows with their usage text, and the starter builds none by hand.
- Seat rule "One builder per kind". Refactor only; no ROM1 fact moves, no claim is cited, no
  divergence row is written, no player-visible change.

## As built

### The table

- `Setting` gains `Usage` (the game flag's help text; `Help(list...)` fills a `%s` with a list the
  game supplies), `Repeat` (any number of values, each passed as its own flag) and `Needs` (the key
  of a row whose blank value leaves this row out). `Option` gains `Usage` for a `Choice` value's own
  flag (`-4x`, `-8x`).
- New kind `Verbatim`: passes `-flag value` with the value as given unless it is empty.
- New rows, not kept in `[options]`: `Assets` (`-assets`), `Base` (`-base`, help lists the profile
  ids), `Mods` (`-mods`), `ModsDir` (`-mods-dir`, needs `mods`), `ModSetting` (`-mod-setting`,
  repeats, needs `mods`), `ModsAcceptUnmarked` (`-mods-accept-unmarked`, a `Switch`, needs `mods`),
  `Check` (`-check`) and `Version` (`-version`).
- `Settings` stays the thirteen `[options]` rows, so the settings file's keys and bytes do not move.
  `Table` is every row in argument order: `Assets, Base, Mods, ModsDir, ModSetting,
  ModsAcceptUnmarked`, the `Settings` rows, `Check, Version`. `ByKey` searches `Table`.
- `Build(values func(Setting) []string)` walks `Table`: a row whose `Needs` row is blank passes
  nothing; a repeated row passes each value; any other row passes its first value. `One` adapts a
  one-value-per-row function.
- The help texts of the thirteen `[options]` rows moved from the game's binding into the rows.
- `mod.SettingFlag.String` writes the `<mod>.<key>=<value>` text that `mod.ParseSettingFlag` reads.

### The starter

`settings.launchArgs(gameLaunch)` is the one builder: `gameLaunch` carries the root, the detected
profile id (empty when not nameable), the mods directory, the mod setting texts and whether the
launch is a check; the settings give the mod list, accept-unmarked and the `[options]` rows. Play,
Check and the command line shown in the window go through `app.gameArgs`; the version probe passes
`launch.Version.Args("true")`. `withBase` and the hand-built `-assets`, `-mods`, `-mods-dir`,
`-mod-setting`, `-mods-accept-unmarked`, `-check` and `-version` are gone.

### The game

`registerLaunchFlags` defines a flag for every row of `Table` (a `Choice` row's options name their
own flags), with the row's default and `Help` text. `launchFlags` binds each flag to its option; the
`-mod-setting` binding is the function that appends `mod.ParseSettingFlag`'s result. `baseUsage`,
`modsUsage`, `modsDirUsage`, `modSettingUsage`, `modsAcceptUnmarkedUsage` and the inline
declarations of `-assets`, `-check`, `-base`, `-version`, `-mods`, `-mods-dir`, `-mod-setting` and
`-mods-accept-unmarked` are gone. `skillUsage` is `launch.Skill.Help(skillNames()...)`.

### The map editor's test play

`game.TestPlayArgs` builds `-assets`, `-base`, `-mods`, `-mods-dir` and `-mission` from the rows and
now returns an error when the `-mission` row refuses the number; `MapInspector.TestPlay` reports it
as a test play error. Test play mission numbers are guest numbers from 1 up, which the row accepts.

### Sites before and after

| Site at `d7c4e9f9` | After |
|---|---|
| `cmd/starter/settings.go:268-287` `launchArgs` builds `-assets`, `-mods`, `-mods-dir`, `-mods-accept-unmarked` | `launchArgs(gameLaunch)` calls `launch.Build` only |
| `cmd/starter/app.go:343-368` `modArgs` builds `-mod-setting` pairs | `modSettings` returns `mod.SettingFlag` texts |
| `cmd/starter/app.go:1187-1200` `withBase` inserts `-base` | removed; `gameArgs` sets `gameLaunch.Base` |
| `cmd/starter/app.go:1020` `launchPlan("-check")` | `launchPlan(true)`, the `Check` row |
| `cmd/starter/app.go:1091` `[]string{"-version"}` | `launch.Version.Args` |
| `cmd/againrom/main.go:429-430,456-469` flag declarations | `registerLaunchFlags` over `Table` |
| `cmd/againrom/main.go:159-177` usage constants | `Usage` of the rows |
| `cmd/againrom/launchflags.go:20-40` help texts in the binding | `Usage` of the rows |
| `pkg/game/mapedittestplay.go:132-138` `TestPlayArgs` by hand | `launch.Build` |

### Arguments outside the table

- Game flags the starter and the test play never pass: `-seed`, `-original-random`, `-chargen`,
  `-headless`, `-nomusic`, `-cutscene-check`, `-media-check`, `-sound-options-check`,
  `-owner-fidelity-check`, `-dump-base`, `-pack-mod`, `-pack-out`, `-pack-replace`. They stay
  declared in `parse`.
- The map editor's own arguments, which the starter passes when it opens the editor
  (`cmd/starter/app.go` `mapEditorPlan`: `-assets <root>`; `-list` for the map list), are the
  editor's flags, not the game's.
- The `usage` line `run` prints after a parse error is unchanged prose that names flags.
- The game's check that `-mod-setting` and `-mods-accept-unmarked` need `-mods` keeps its own text;
  `Needs` drives only what is built.

## Proof

- Argument vectors equal to main: the dump test `review/story1507-launch-args/dump/
  starter_launchdump_test.go` drives the starter app (Play, Check, the window's command line and the
  version probe) over 67 starter states: each of eight base choices alone and with mods and options,
  no mods, one mod, several mods in two orders, a mods directory (with non-ASCII and padded text)
  with and without mods, accept-unmarked on and off with and without mods, seven mod setting values
  (blank-separated words, `=`, quotes, Cyrillic text, padding), a refused, blank and undeclared mod
  setting, a custom assets path with blanks and Cyrillic text, no base selected, and every option
  row at and past its bounds. Run on the candidate and on `git archive d7c4e9f9` under the notes
  directory, the two 268-line dumps are byte-identical.
- Game parse equal: the parse dump parses the dump's 114 play and check vectors and 23 hand vectors
  (test play shapes, malformed and repeated `-mod-setting`, unknown `-base`, bare `-assets`,
  `-version`, bound refusals), then records `run -help`, `run -h`, `run -nosuch` and the flag set's
  own defaults listing (every flag with its type, default and help). Both 490-line dumps are
  byte-identical, so `-help` and every flag's help text are unchanged and unreordered.
- Tests that cover the vector and the parse: `cmd/starter/settings_test.go`
  (`TestLaunchArgsExactArgv`, `TestLaunchArgsRefusals`), `cmd/starter/launchsettings_test.go`, `cmd/starter/base_test.go`
  (`TestPlayNamesTheDetectedBase`), `cmd/starter/modsettings_test.go`
  (`TestEditedSettingsGoToPlayAndCheck`), `cmd/starter/app_test.go`
  (`TestPlayStartsTheGameWithTheExactArgumentList`), `cmd/starter/chicken_test.go`,
  `cmd/starter/shopmultiplier_test.go`, `cmd/againrom/launchsettings_test.go`,
  `cmd/againrom/modsrun_test.go`, `cmd/againrom/base_test.go`, `pkg/game/mapedittestplay_test.go`
  (the test play vector), `cmd/againrom/editortestplay_release_test.go`.
- New tests: `TestLaunchTableFlagsCarryTheRows` (every flag of `Table` carries its row's help; the
  base, mod and check rows built by `Build` parse back; without mods the needing rows pass nothing)
  and `TestSettingFlagStringReadsBack`.
- Changed tests, no asserted value changed:
  - `cmd/starter/settings_test.go`, `launchsettings_test.go`, `shopmultiplier_test.go`: the call
    `launchArgs(root, mods)` became `launchArgs(gameLaunch{Root: root, ModsDir: mods})`.
  - `cmd/starter/literals_test.go`: `-base`, `-check`, `-version` and `-mod-setting` left
    `notCaption`, because `TestNotCaptionIsUsed` refuses an entry the window files no longer hold.
  - `cmd/againrom/launchsettings_test.go`: `launchFlags` and `registerLaunchFlags` take the `-mods`
    text; `launchset.Build(value)` became `launchset.Build(launchset.One(value))`.
  - `cmd/againrom/editortestplay_release_test.go`: `TestPlayArgs` returns an error as well.
- Loss control, each broken in `pkg/launch/settings.go` and restored from commit `ad77fcb8`:
  - `ModSetting` without `Repeat`: `TestEditedSettingsGoToPlayAndCheck` fails (only the first mod
    setting passes); every `cmd/againrom` parse panics, first in `TestParseBaseFlag`, because the
    row's binding is a function, not a `*string`.
  - `ModsDir` flag renamed `mods-folder`: `TestPlayStartsTheGameWithTheExactArgumentList`,
    `TestEditedSettingsGoToPlayAndCheck`, `TestAStoredKeyTheModNoLongerDeclaresIsNotPassed`,
    `TestLaunchArgsExactArgv` and
    `TestTestPlayWritesTheEditedMapIntoTheTestModAndStartsTheGame` fail; the game panics
    "launch flag -mods-folder sets no option" at its first parse.

## Gates

Run on `ad77fcb8`; `origin/main` was still `d7c4e9f9`, so the merge was a no-op.

- `gofmt -l`, `go vet` of the touched packages, `git diff --check`,
  `scripts/check-no-game-assets.sh`, `check-div-claims.sh`: clean. `internal/storyguard` comment
  bytes fell 3; the baseline moved with it.
- `go test -trimpath -count=1 ./...`: every package passes except two `pkg/game` tests,
  `TestMapPagesSaveAsPreservesLanguageTree` and `TestMapPagesPackageCopiesLooseTreeAndEdits`, both
  "Access is denied" on a `.map-pages-*` rename. Rerun alone: the second passed, the first failed
  the same way, then passed alone on a further run; on `d7c4e9f9` it passed.
- Release set, EN, headless only: 13 packages, 1418 gated tests, no failure; the one skip is
  `TestReleaseActivePauseRenderedWorld`.
- Release set, RU, headless only: 1418 gated tests; the same single skip; two failures, both
  "Access is denied" on a `.map-pages-*` rename: `TestReleaseMapEditorArmsOpenAndKeepBytes`
  (`scenario/81.alm`) and `TestReleaseMapEditorForestWithoutAStrokeSavesUnchanged`. Each rerun
  alone on RU passed.

## Open debt

- The game's "-mod-setting and -mods-accept-unmarked need -mods" refusal is not derived from
  `Needs`; `-mods-dir` needs `-mods` in the starter's build but not in the game's parse.
- `Verbatim` passes nothing for an empty value; before, an enabled mod list always passed
  `-mods-dir` even if the directory text were empty. The starter's mods directory is never empty
  (`mod.DefaultDir` joins `mods` to the program folder), so no reachable vector changed.
