# MOD starter

## Intent and authority

The MOD milestone starts with a separate program, `starter.exe`, that launches `againrom.exe` with a chosen base game, mods and parameters and keeps its settings in an ini file. Authority is owner direction only; no ROM1 behaviour is involved and no divergence row is needed. Owner rulings: a mod is files only; nothing mod-specific is compiled into `againrom.exe`, which is the engine plus a mod interpreter (the interpreter is a later story). The starter replaces the in-engine start screen of `review/mod-milestone/PROPOSAL.md`.

Mods have no effect on the game yet. `againrom -mods` resolves and lists the named mods and nothing more: no interpreter, no overlay, no data change. Without `-mods` every behaviour, hash and release witness is unchanged.

## As built

Starter window (`cmd/starter`, 760x620, Ebitengine with the bundled bitmap font; the window is drawn into an RGBA image by `render`, so `starter -screenshot <png>` draws it once with no window). It shows and edits:

- Base game: the bases of the ini, one selected. Each row shows the key, the folder, and either `OK <language>` or the missing archives. Validity is `game.InspectInstall`: the five required archives present (case-insensitive, the same set `againrom -check` opens), then the language entry of `main.res` read alone (`english` or `russian`). Demo identity is not detected; the demo folder reads as a valid `english` install. A folder is added by typing or pasting its path (Ctrl+V reads the clipboard through PowerShell) and pressing Enter or Add; Remove drops the selected base.
- Mods: the mods folder (default `mods` beside `starter.exe`), each subfolder holding a `mod.toml`. Tick to enable; enabled mods are listed first with their load order number. A manifest that fails to parse, or whose id differs from its folder name, is listed in red with the error and cannot be ticked. An enabled id with no folder stays listed in red until unticked. Play and Check refuse while an enabled mod is unusable.
- Parameters: sound (default, on, off), volume 0-100, movies, mission movies (normal, 4x, 8x), markers, saves folder. Advanced: mission number, map picker, skill, free extra arguments. Starter: close after Play, game program path.
- Buttons: Play starts the game and either stays open or closes; Check runs the game with `-check` and shows the first six output lines; Save settings writes the ini. A line above the status shows the exact command Play would run.
- Header: `Starter <version> (<revision>)` and `launches againrom <version> <revision>`, the latter read from `againrom -version` (`unknown` when the program is missing or fails).

Ini (`pkg/ini`, stdlib only). Default path `starter.ini` in the folder of `starter.exe`; `starter -ini <path>` overrides. A missing file means defaults. A line that is not blank, a comment (`;` or `#` first), a `[section]` or a `key = value` is reported in the window and kept verbatim; it is never fatal. Save rewrites only the lines of keys the starter owns; unknown keys, comments and malformed lines survive, a removed base is deleted, and a file that was byte-equal to its own save stays byte-equal. If the folder cannot be written the window says so at start and again on Save, and keeps running.

```
; Againrom starter settings
[starter]
againrom =
close-on-play = false
last-base = en

[bases]
en = C:\Games\againrom\en
ru = C:\Games\againrom\ru

[mods]
dir =
enabled = dark-rules,extra-spells

[options]
sound = default
volume =
movies = true
video = normal
markers = false
saves =
mission =
picker = false
skill =
extra =
```

Empty `againrom` means `againrom.exe` beside `starter.exe`; empty `dir` means `mods` beside it; `sound` is `default`, `on` or `off`; empty `volume` keeps the game's saved preference; `video` is `normal`, `4x` or `8x`; `enabled` lists mod ids in load order. Trailing comments are not supported (a value may hold `;` and `#`).

Game side (`cmd/againrom`, `pkg/mod`). `-mods <id,id,...>` and `-mods-dir <dir>` (default `mods` beside the game executable). `mod.Resolve` takes each id as the folder name under the mods directory and refuses, naming the mod and exiting 2, an id that is not `[a-z0-9][a-z0-9._-]{0,63}`, a repeated id, a missing folder, an unreadable or malformed manifest, or a manifest id that differs from the folder name. `-check` prints, after the usual lines, `againrom: N mod(s) resolved (no effect in this build)` and one `againrom: mod <n>: <id> <version> "<title>" applies-to=<list> dir=<dir>` line per mod. A windowed launch writes one stderr line saying the mods have no effect. `mod.toml` subset, read by `mod.ParseManifest`: top-level `key = value` lines before the first table header; values are basic or literal strings or one-line string arrays, each with an optional `#` comment; other keys are skipped (a multi-line array to its closing bracket); `id`, `title`, `version` are required strings, `applies-to` a string or string array defaulting to `["common"]`; a repeated key is an error; a table header ends the read. Nothing else of a mod is opened.

Versions. Each program has a tracked one-line file `MAJOR.MINOR.PATCH`, embedded with `go:embed` so a plain `go build` carries it: `cmd/againrom/VERSION` (0.9.0, pre-1.0 until the complete game) and `cmd/starter/VERSION` (0.1.0). The revision is the existing build stamp (`internal/buildinfo.Revision`, the Go VCS revision that `pipeline/build-candidate.sh` already checks; `launchStamp` now calls it). `-version` on both prints `<name> <version> <revision>` and exits 0. The main menu draws `Againrom <version> (<7-character revision>)` in its bottom-right corner with the 6x16 debug face the other plain-text overlays use (`App.SetMenuLabel`, set by `cmd/againrom` for a windowed launch only; the box lies on background, overlaps no button, and the change is confined to it, tests in `pkg/ui/menulabel_test.go`). Each executable carries a Windows VERSIONINFO resource (FileVersion, ProductVersion, ProductName, FileDescription, InternalName, OriginalFilename) from a committed COFF object `cmd/<program>/rsrc_windows_amd64.syso`, written by `internal/versionres` (pure Go, no dependency). A test regenerates each object from its VERSION and compares bytes, so editing a VERSION without regenerating fails.

Bump rule (proposal). The seat bumps at landing: PATCH for a hotfix, MINOR for a story, only for the programs the change touches. To bump a program edit its `VERSION` and run, from inside the module:

```
go run ./internal/versionres/cmd/versionres
```

and commit the changed `VERSION` and `.syso` files together.

Build. `pipeline/build-candidate.sh` builds `./cmd/...`, so `starter.exe` joins `builds/candidate-*` with no script change. Its closing `tool names equal to builds/current` diff will list `starter.exe` as new on the first candidate. `starter.ini` lives beside `starter.exe`, so a promotion that replaces `builds/current` must carry it (seat step). `go build` run in a linked worktree stamps the seat's revision; `build-candidate.sh` builds in `engine/` and is unaffected.

Boundaries. `pkg/ini` and `pkg/mod` are leaves in `internal/archtest`'s allow-map; `cmd/starter` names `pkg/game`, `pkg/ini`, `pkg/mod` and `internal/buildinfo`, and is the one command permitted the bitmap font packages. Nothing reaches `pkg/sim`.

## Hashed state

None. No simulation, save or mission code changed. `App.SetMenuLabel` changes only the composed main menu frame of a windowed launch.

## Proof

Unit tests: `pkg/ini` (byte-identical round trip, unknown keys, comments, malformed lines, duplicates, new files), `pkg/mod` (subset parser accept and refuse tables, scan, resolve), `cmd/starter` (settings load/store, exact argv table, refusals, base and mod interaction, play, check, version probe, unwritable folder, screenshot), `cmd/againrom` (`-mods` parse, `-check` listing, refusal by name, default directory), `internal/versionres` and `internal/buildinfo`, `pkg/ui` menu label, `internal/archtest` rows for the new packages.

Release witnesses (gated, `internal/gatedtests/testdata/population.txt`): `cmd/againrom.TestReleaseCheckListsAModOverTheInstall` (`-check -mods <temp mod>` lists the mod and passes; a missing mod fails naming it) and `pkg/game.TestReleaseInspectInstallNamesTheLanguage`. Both pass on the EN and RU installs, one root at a time (`TestReleaseInspectInstallNamesTheLanguage` reports `english` and `russian`). The EN and RU `Release` tests of `cmd/againrom` and `pkg/ui` pass unchanged: the menu label is set only by the windowed `cmd/againrom` path, so no existing menu witness draws it and none changed. A real-install menu frame with the label, composed through `HeadlessFrame` for both installs, was inspected by eye: the label sits on background right of the bottom-right ornament.

Screenshot of the starter window: `review/story1266-starter/starter.png`, drawn offscreen by `starter -screenshot` over a constructed ini and mods folder (no window opened).

## Open debt

- Base identity by profile (EN, RU, demo) is a later story; the demo folder is shown as a valid `english` install and fails `againrom -check` (`NewFrontEnd` requires an asset the demo lacks).
- Mod interpretation, overlays, conflict and load-after resolution, and the mod-set digest are later stories. `requires`, `conflicts`, `load-after` are read by no code.
- The starter shows a console window flash only if built with the default subsystem; building it with `-ldflags=-H=windowsgui` would remove it, and `build-candidate.sh` does not pass that flag.
- Paste reads the clipboard by running PowerShell (about 0.3 s); there is no native clipboard call. Not exercised interactively: the window itself was driven only through tests and the offscreen render.
- No reorder control for load order; untick and retick to change it.
