# Headless runner for modders

## Intent

A modder with the shipped game and a mod folder checks a mod's town without a window:
`againrom --headless <scenario.json>` loads a save, goes to the town, prints what the town is and
writes the frame the game draws. Without the new scenario version nothing changes.

## Authority

Owner direction T4b of the mod track (`pipeline/BACKLOG.md`), after the base dump
(`docs/1487/story.md`): the headless runner becomes a modder tool.

## As built

### Documentation

`docs/MODS.md`, section "Headless runs", for a reader who has never seen the source: the command,
the scenario keys, both stages, every command and its keys, the accepted forms of each argument
(row `target` forms, `key` names, `pointer` actions, every `at` form, `until`, `character`, the
SAVE dialog keys, `state`), where saves go (`saves`, `original_saves`, `-saves`), what `capture`
prints, the frame command and a worked example. The example is `pkg/game/testdata/headless-town.json`
byte for byte; `TestHeadlessDocExampleIsTheFixtureAndNamesTheFormat` fails when the section drifts
from the fixture or leaves out a command, scenario key, `at` form or capture field. The docs are not
in the release ZIP, which packs the README, the licence, notices and the executables.

### Scenario version 10

Version 10 gates both additions, so a file at version 9 or earlier prints the bytes it printed
before (`TestReleaseHeadlessCaptureIsUnchangedWithoutTheNewVersion`).

- `capture` on a town screen adds `town`: `id`, `name` (the name the game draws, UTF-8, the
  install's town word when the town has no name) and `art` (keys of every picture the square can
  draw, conditional ones included, first the square's picture: an install key or `mod:` and the path
  in the mod). The game menu over a town prints `town` too; the second game prints none. Existing
  fields keep their names and values.
- `frame` with `path` writes the current frame as a PNG from `app.HeadlessFrame()`, the renderer the
  release tests use, and prints `frame` with the full path, width and height.
- `pointer` `at` takes `tip_close`, the Close button of the welcome tip that hides a town's square.

### Pointer actions

The scenario runner and `ui.App.HeadlessPointer` take one list, `ui.HeadlessPointerActions`
(`hover`, `press`, `move`, `release`, `right-*`, `middle-*`, `wheel-up`, `wheel-down`), each with an
optional `shift-` and then an optional `ctrl-` prefix, parsed by `ui.ParseHeadlessPointerAction`.
The runner refused every `middle-`, `wheel-`, `shift-` and `ctrl-` form before; a file that ran
before prints the same bytes.

### Write boundary

A frame path is judged by the `-dump-base` rule (`insideInstall` in `pkg/game/basedump.go`): refused
inside the install, inside a preserved install beside it, or below any folder holding the game's
archives. It is also refused over an existing file or folder, when two steps name one path, and
when it does not end in `.png`. The paths are judged before the first step runs, the file is
created exclusively, and a failed write removes it. A relative path is read against the scenario's
folder.

### Refusals list the accepted values

`ReadHeadlessScenario` walks the file with the struct tags and refuses an unknown scenario, step,
`at`, `state`, `member`, `inventory`, `shop` or `character` key with the keys accepted there. An
unknown command lists the commands of the stage at the declared version; an unknown stage lists the
stages; the pointer, key and SAVE dialog refusals list their accepted names; an `at` naming no form
or two lists the forms.

## Proof

- `TestReleaseHeadlessExampleNamesAndPicturesTheModTown`: the documented scenario runs on a
  self-made fixture mod town and `capture` names town 2, its name and the mod picture.
- `TestReleaseHeadlessFrameDrawsTheModTownName`: the written PNG equals the renderer's frame, and
  the name region differs from a frame with another name.
- `TestReleaseHeadlessFrameRefusalsWriteNothing`, `TestHeadlessFramePathBoundary`,
  `TestHeadlessRefusalsListTheAcceptedValues` and the `cmd/againrom` tests: refusals write nothing.
- `TestHeadlessDocPointerActionsAreTheAcceptedList`: the page's action list equals the accepted
  list; a page missing `wheel-up` or naming an extra action fails it.
- `TestReleaseHeadlessPointerActionsReachTheTown` (EN and RU): `shift-press`, `shift-release`,
  `wheel-up`, `middle-press`, `middle-release` and `ctrl-hover` run through a version 10 scenario
  and close the welcome tip; an action outside the list is refused with the accepted values.
- Loss controls: a capture without town fields, a different frame and a drifted document each fail
  their test.

## Open debt

The mission stage's frame is refused: the mission screen is drawn to the window and has no headless
frame.
