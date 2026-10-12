# The map editor ships and opens from the starter

## Intent

A player who unpacks a release archive (Windows AMD64, macOS Intel, macOS
Apple Silicon) finds the map editor beside the game and the starter, and opens
it from the starter on the selected first-game install. Base: public main
`165270ed`; knowledge pin k275.

## Authority

Owner direction: modding lacks a map editor most; `mapedit` goes into the
release archive and the starter may open it; the editor is our own. No ROM1
behaviour is claimed, so no claim is cited and no divergence row is added.

## As built

### Version identity

`mapedit` carries the game's version, not a version of its own. The
`internal/versionres` table gained `VersionDir`: the `mapedit.exe` row builds
`cmd/mapedit/rsrc_windows_amd64.syso` from `cmd/againrom/VERSION`. The editor
embeds that object on every platform and reads its fixed file version
(`versionres.FileVersion`); `mapedit -version` prints
`buildinfo.Line("mapedit", <version>)`, the line `againrom` and `starter` print.
The release procedure is unchanged: the release commit already runs
`go run ./internal/versionres/cmd/versionres`, and
`TestCommittedObjectsMatchTheirVersionFiles`, which both release workflows
run, fails when the object is stale.

### Starter

A **Map editor** button sits after **Save settings** in the action row, built
by the starter's `button` builder. Its caption comes from a per-language table
read with the selected base's label language (`Map editor`, `Редактор карт`);
the starter had no such table for its own captions, and every other caption
stays English. The button resolves `mapedit` (`mapedit.exe` on Windows) in the
starter's own folder and refuses, with the starter's error line and without
running anything, when no base is selected, the base is not an install, or the
editor is missing. Otherwise it first runs `mapedit -assets <root> -list`
through `deps.run`, the call Check uses: the list loads the base as the window
does and exits (0.4 s on the EN root). When the list fails, the starter shows
its status line and the editor's own output in the Check output lines and
starts nothing. When it passes, the starter starts `mapedit -assets <root>`
through `deps.start`, the call Play uses. The check is what refuses a
second-game base: `mapedit -list` on both lawful ROM2 roots exits 1 on
`world/data/data.bin` (probe: `review/story1420-mapedit-ship/list-rom2-*.txt`).
The starter names no game for this. **Close after Play** does not apply to the
editor.

### Windows console

`deps.start` applies `CREATE_NO_WINDOW` (`hideConsole`), as for Play, so the
editor shows no console window. The map list check runs with the same flag
and its output is captured, so an error the editor reports while loading the
base reaches the starter's window. The editor stays a console-subsystem
program: started from a terminal, `mapedit -assets <root>` prints every error.

### Packaging

`scripts/build-tools.sh` in both release workflows builds `./cmd/mapedit`.
`scripts/package-windows.ps1` and `scripts/package-macos.py` check its
`-version` line (game version, 12-character revision), its Go build settings
and, on Windows, its version resource, and put it in the archive as an
executable. `scripts/test-windows-package.ps1` and
`scripts/test-macos-package.py` expect 12 entries, check the extracted
editor's version line (and Windows version resource) and refuse a package
without it. `scripts/release-notes.ps1` names the editor in the install text.

## Proof

- `cmd/starter/mapeditor_test.go`: the exact check line and start line beside
  the starter, the start only after the check; a failing check shows the
  editor's output and starts nothing; refusals for no base, a non-install and
  a missing binary run and start nothing; the caption follows the base
  language.
- `cmd/mapedit/version_test.go`: the version equals `cmd/againrom/VERSION`
  and `-version` prints `mapedit <version> <revision>`.
- `internal/versionres`: `FileVersion` reads back what `Syso` wrote; the
  editor row reads `cmd/againrom`.
- The Windows package, built from `scripts/build-tools.sh` binaries, passes
  `scripts/test-windows-package.ps1` with `mapedit.exe` in it; the same test
  without `mapedit.exe` fails.
- The macOS witness runs only on the macOS workflow.

## Open debt

- An editor that fails after its window starts from the starter (after the
  map list check passed) has its error output in no window.
- The editor reads first-game installs only.
