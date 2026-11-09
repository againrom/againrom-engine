# Story 1619: the map editor is a mode of the game program

## Intent

The Windows release archive of 0.150.0 was 94.2 MB; two steps have since cut it. This story is step 3 of `pipeline/BACKLOG.md`, section Release archive size: the map editor stops being a separate program. It runs as `againrom --editor`, and no `mapedit` program is built or shipped on any platform.

## Authority

Owner, on the 94 MB archive: «Делай эти шаги» (`pipeline/BACKLOG.md`, Release archive size, step 3). The brief reserves no DIV range; this story changes which file a player opens, nothing in the editor. No DIV row is added or changed. Architecture work; it counts 0 toward a release.

Seat amendment to the brief's "no `--editor` behaves exactly as before", made to keep the shipped route of a map opened from Windows (`CHANGELOG.md`, 0.140.2: `mapedit.exe "map.alm"`): a start of `againrom` with no flag at all and exactly one positional argument whose extension is `.alm`, in any letter case, is editor mode with that argument. Every other start is unchanged.

## As-built behaviour

`againrom --editor <mapedit arguments>` runs the map editor; `-editor` is the same flag. The game's parser (`parse` in `cmd/againrom/main.go`) stays the one parser: `--editor` is a flag of that flag set, and it ends the game's flags, so every argument after it is the editor's own, including flags the game also has (`-assets`, `-check`, `-version`). A game flag before `--editor` is refused with exit status 2, because it would otherwise be dropped; `--editor=false` leaves the game as if the flag were absent; a game flag's value spelled `--editor` (`-saves --editor`) stays that flag's value.

`againrom "map.alm"` also runs the map editor on that map: the same `editorapp.Run` call as `againrom --editor "map.alm"`. This is the route the Windows shell takes for an associated file, "Open with" and a drop, which cannot add a flag. `parse` decides it, from `editorMapExtension` and `namefold.HostEqual` over `filepath.Ext`, and only when the map path is the start's single argument. These starts stay the game: another extension, two positional arguments, any flag with a positional argument, and `-- map.alm`.

Package choice: the code of `cmd/mapedit` moved to `pkg/editorapp` (`run.go`, `mcp.go`, `profile.go`, `version.go` and the tests), which `cmd/againrom` imports. `pkg/editorapp.Run(version, args, stdout, stderr)` is the old `main`. `run`, `runAs`, `openWindow`, `newInspector`, `reportFailure`, `startMCP`, `loadPanelProfile` and every flag moved unchanged. `cmd/mapedit` is gone with its version resource.

What each program runs now:

| Platform | Package layout (`scripts/package-*`) | The starter's Map editor runs |
|---|---|---|
| Windows | `againrom/againrom.exe`, `againrom/starter.exe` | `againrom.exe -editor -assets <base>` beside `starter.exe` |
| macOS | a flat folder, `againrom/againrom`, `againrom/starter`; `scripts/package-macos.py` builds no `.app` bundle | `./againrom -editor -assets <base>` beside `starter` |
| Linux | tar and AppImage, `$appdir/againrom`, `$appdir/starter` | `againrom -editor -assets <base>` beside `starter` |

The starter's Map editor control works as before: it first runs the editor's map list (`-editor -assets <base> -list`) and shows the editor's error on failure, then starts the window. Its status line names the file it runs, now `againrom.exe`. It always uses the program beside the starter, not the starter's configured game path, as it did for `mapedit`.

The editor's window keeps `appicon.MapEditor`, set at run time. On Windows the file icon of the program is the game's icon; `internal/versionres` lost the `mapedit.exe` row and the `VersionDir` field only that row used.

`-version` after `--editor` prints `mapedit <game version> <revision>`, as `mapedit -version` did. The version is the one the game program already carries (`cmd/againrom/VERSION`, handed to `Run`); there is no second version source. `mapedit-error.log`, the flag set's name `mapedit`, the `mapedit:` message prefix, `mapedit-mcp.json` and the `againrom-mapedit` MCP server name keep their names.

F5 test play starts `launch.Beside(self, "againrom", GOOS)` through `launch.For(...).Start()`, the launch it used before. `self` is now the game program itself, so test play is a new process of the same file and shares no state with the running editor.

The game's start is unchanged: `pkg/editorapp` has no `init` function and no embedded file, and its package variables are a function value, a struct of two function values and a program name. Starting `againrom` without `--editor` adds one flag registration and, outside the lone map path above, one test of the argument list.

## Proof

Sizes, measured with `scripts/build-tools.sh`'s flags (`-trimpath -buildvcs=false -ldflags "-s -w -X ...stamp=..."`, `CGO_ENABLED=0`) on Windows, `bc9fa8e94` against this candidate, zipped with Python `zipfile` deflate level 9, program files only:

| Set | Files | Bytes | Zip |
|---|---|---|---|
| Before | `againrom.exe` 44,717,056; `mapedit.exe` 44,725,248; `starter.exe` 14,062,592 | 103,504,896 | 43,442,264 |
| After | `againrom.exe` 44,789,248; `starter.exe` 14,063,616 | 58,852,864 | 25,162,943 |

The full Windows package (`scripts/test-windows-package.ps1`) is 25,337,135 bytes. The mode's import adds 72,192 bytes to `againrom.exe` (0.16%); `go list -deps ./cmd/againrom` adds one package, `pkg/editorapp`, and removes none.

Focused tests (`-run`): `TestEditorMode*` in `cmd/againrom` (arguments after the flag, refusal of a game flag before it, the flag read by the game's parser, the lone map path, the version line, the failure report); the moved `pkg/editorapp` tests; `TestMapEditor*`, `TestEditorArguments*`, `TestEditorExecutable*` in `cmd/starter`; `TestBeside*`, `TestTheStarterLaunches*` in `pkg/launch`; `internal/versionres`; `TestTestPlayStartsTheProgramBesideTheEditor`, `TestRunReportsAFailureAndExitsOne` and `TestVersionFlagPrintsOneLineAndExits` in `pkg/editorapp`. `scripts/test-windows-package.ps1` passed on the candidate's binaries: 12 entries, two programs. `scripts/test-linux-package.py --unit` passed. Release population: `TestReleaseCheckPrintsTheFindingsTheWindowLists` and `TestReleaseNewWritesTheBlankMapAndPrintsItsReport` moved from `cmd/mapedit` to `pkg/editorapp`; the count and subjects are unchanged.

The gates that run on the final candidate (`scripts/check-fast.sh`, `internal/archtest`, `internal/gatedtests`, the two release tests on `en`, `ru`, `rom2-en`, `rom2-ru`) are reported in the lane's return.

## Open debt

- An existing `.alm` file association points at the removed `mapedit.exe`; the player associates `.alm` with `againrom.exe` again. The Windows shell route itself was not driven; `parse` is covered by `TestEditorModeOpensALoneMapPath`.
- A `mapedit.exe` left by unzipping a new release over an old folder stays there, stale, and no release replaces it.
- `scripts/test-macos-package.py` and the Linux package run itself need their own hosts; they run at the release. No window was opened.
- `scripts/test-windows-package.ps1` needs PowerShell 7 (`pwsh`), which this host lacks; on Windows PowerShell 5.1 it was run with `System.IO.Compression` preloaded, the only difference.
- Nine rows in `docs/divergences/` (`DIV-3211` and others) still name the `mapedit` program in their text; the brief reserves no DIV range and no row is changed. The seat chooses whether to reword them.
- Older story, hotfix and CHANGELOG entries and the notes in `internal/storyguard/baseline.go` keep `cmd/mapedit` and `mapedit.exe` as history. Fixture paths named `mapedit.exe` in `pkg/game` and `pkg/editorapp` tests stay; they are arbitrary paths.
- The packaging checks no longer run `-version` of a separate editor program; the editor's version line is covered by `TestEditorModeVersionLineIsTheGamesVersion`, which the three release workflows now run.
