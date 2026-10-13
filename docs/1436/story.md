# The map editor plays the map it edits

## Intent

In the first game's map editor the author presses F5 and plays the map being
edited, unsaved edits included, in the game. The editor stays open; closing
the game returns the author to the edit. Nothing the author saved and no
install file changes because of a test play. The story base is public main
`62ca3ea4`, pin k301. The correction base is candidate `bc7c673b`; merged main
is `f33b52a2813bc5c248dd229d9214278c22decbd6`, pin k301.

## Authority

Owner direction; no ROM1 research. The original editor has no test play. The
key, the folder names and the message lines are ours. The test play changes
nothing the game does for a player, so it takes no divergence row.

## As built

### One launch builder

`pkg/launch` is the one way an Againrom program starts another. A launch is
data: `Program{Path, Args, Dir}`. `For(path, args)` runs the program in its own
folder, `Beside(self, name, goos)` names a program beside the running one,
`Start` launches without waiting and `Run` waits with a limit and returns the
combined output. Both hide a console window (`HideConsole`, moved out of
`cmd/starter`). The starter's `run` and `start` and its game and editor paths
are built through it unchanged; the editor's test play starts the game
through `For(...).Start()` and names it with `Beside(self, "againrom", goos)`.

### One route for a mod map

The test play adds no route. It writes a guest mission and starts the game
with the existing flags; the game reads it through `data/missions.toml`,
`FrontEnd.SetModMissions` and `enterMission`, as any mod's guest mission.

### The test mod

`TestPlay` (`pkg/game/mapedittestplay.go`) writes the mod `testplay` into the
folder `testplay` of the editor's player profile, the profile
`ResolveRuntimeProfile` picks for the editor's own executable: the folder
beside the editor, or `Againrom/` beside it inside an install, or the user
configuration directory's `Againrom/`. The destination is
`<profile>/testplay/testplay`. For a profile inside an install,
`RuntimeProfile.folder` grants writes beneath the private profile only;
every other directory keeps the editor's install fence (`fenceEditorWrite`).
The mod folder holds `mod.toml`, `main.star`, `data/missions.toml`,
`mission/map.alm` (the editor's save bytes of the map as it stands) and
`mission/text/` (the pages of the installed mission the map was opened from,
read by `installedPages`, which `AssembleModMission` also uses, or the pages of
the map's entry in the `-mod` folder).

The destination records ownership in `.testplay-files.json`:
`Version` is 1 and `Files` is the sorted list of generated slash paths.
`testPlayOwnership` (`pkg/game/mapedittestplay_files.go`) refuses an existing
destination without a valid, complete record. Before reading the record, it
also refuses a destination equal to, containing or inside the opened host
map's folder or the selected `-mod` folder. These checks use physical paths
and apply even when a test play owns the destination.

A later test play replaces only recorded files and removes only recorded
pages absent from the current mission. Unrecorded files remain. A generated
file that collides with an existing unrecorded file is refused before any
publication. The generated files and updated ownership record are published
through dot-named files moved over their targets.

### The game's own mod load

Before publication or start, `validateTestMod` stages the complete resulting
mod in a temporary folder under the profile. It mirrors retained unrecorded
files, adds the new generated files and omits stale recorded pages. It runs
`mod.Resolve` and then `modrt.Load` with the editor's install base, no mod
settings and ordinary runtime options, as the game's startup and `-check`
do. This uses the game's file-size, file-count and declaration checks.
The staging folder is removed after validation. A refusal leaves the
destination unchanged and reaches the editor's message line.

The editor's map report and `CheckModMap` remain separate report gates before
staging. Their errors show the report; warnings do not stop a test play.

The declaration is a guest mission. `testPlayGuest` takes the first main
mission of the install's campaign whose win opens a town granting a
companion, that town and companion, and the lowest number after the mission
that `checkGuest` admits. `checkGuest` is the guest check `SetModMissions`
runs, factored out unchanged. On the EN and RU installs that is mission 21
after mission 20, chapter 30, companion 22.

### The start

The game is started with `-assets <root> [-base <id>] -mods testplay
-mods-dir <profile>/testplay -mission <n>` (`TestPlayArgs`). `-mission` is the
shortest existing start: it opens character generation and then mission n.
No existing route puts a party in a chosen mission without it; the picker
route also opens generation.

### Refusals

| case | effect |
|---|---|
| the window's report holds an error | the report shows "Test play refused"; nothing is written or started |
| `CheckModMap` of the generated mission holds an error | the same |
| the destination exists without a valid ownership record | message line names the folder; destination unchanged; nothing starts |
| the destination equals, contains or lies inside the opened map's folder or selected `-mod` folder | the same, including an owned destination |
| a generated file collides with an unrecorded destination file | message line names the file; destination unchanged; nothing starts |
| `mod.Resolve` or `modrt.Load` refuses the staged result | message line gives the loader error; destination unchanged; nothing starts |
| the game program is not beside the editor | message line names it; nothing written |
| the launch fails | message line names the program and the error |
| a second-game install | `campaignService.testPlayRefusal` answers the message line |

Warnings do not stop a test play. The window's key row `testplay` (F5) is one
row of `editorKeyRows`; the help line reads `N: new map / U: unit / F5: play`.

## Proof

- `pkg/launch`: `TestTheStarterLaunchesBuildTheRecordedProcess` compares the
  process of each starter launch with the recipe main built (path, arguments,
  directory, console flag); loss controls: another folder or argument differs.
- `pkg/game`: `TestTestPlayWritesTheEditedMapIntoTheTestModAndStartsTheGame`
  (edited save bytes with an unsaved altitude edit, the mod entry's page, the
  declaration accepted by `modrt.Load` and `SetModMissions`, the started
  arguments, the opened file and the `-mod` folder byte-identical, the map
  still unsaved; a second test play of `scenario/20.alm` takes the installed
  page and drops the first one),
  `TestTestPlayOfAMapWithAnErrorStartsNothing` (loss control: a warning
  plays), `TestTestPlayNamesAMissingOrFailedGameProgram`,
  `TestTestPlayWritesNothingIntoTheInstall`.
- Ownership fixtures:
  `TestTestPlayRefusesAnAuthoredDestination` preserves an authored mod at the
  destination, also when selected and opened with an unsaved edit.
  `TestTestPlayPreservesUnownedPagesAndRemovesOnlyRecordedPages` refuses an
  unowned destination and keeps an author page added to an owned destination;
  its replacement control writes the edited map and removes a stale recorded
  page.
  `TestTestPlayRefusesAnOwnedDestinationOverlappingTheOpenedMapFolder` and
  `TestTestPlayRefusesAnOwnedDestinationOverlappingTheSelectedModFolder`
  each check equality, containment and being inside after a successful first
  test play.
  `TestTestPlayRefusesAnUnrecordedPageCollisionInAnOwnedDestination` refuses
  to replace an author page with a generated page. The refusal fixtures check
  destination and source snapshots and record no additional start.
- `TestTestPlayLoadsRetainedUnrecordedPagesBeforePublishing` includes a
  retained author page in the staged load: an oversized page refuses without
  changing the destination; an in-bound page plays and remains.
  `TestTestPlayInstallProfileWritesOnlyThePrivateFolderAndReplacesIt` checks
  two replacements under the implicit install profile. Existing installed
  files stay byte-identical; new files remain under `Againrom/testplay/`.
- Loader fixture:
  `TestTestPlayRefusesAModLoadBeforeReplacingTheDestination` supplies a
  selected mod page of `mod.MaxFileBytes+1`, requires the game's size refusal
  before replacing an owned destination, and preserves the source. Its
  in-bound page control starts the same mission.
- `pkg/ui`: `TestTheTestPlayKeyPlaysTheMapAndKeepsTheWindow`,
  `TestATestPlayTheReportRefusesShowsTheReport`; the modes table
  (`testdata/mapeditor_modes.golden`) gains the `testplay` column and the new
  help line, and is otherwise unchanged (it passes on the old help line
  without the column).
- Release, EN and RU, headless:
  `cmd/againrom.TestReleaseTestPlayStartsTheEditedMapAsAGuestMission`. The
  editor opens installed `scenario/20.alm`, deletes the first creature and
  presses F5 with a recording launch; the game's `run` then takes exactly the
  recorded arguments plus `-check`. It reports mission 21 at
  `scenario/21.alm` with 56 entities; loss control: the unedited map's test
  play reports 57.
- Release, EN and RU, headless:
  `cmd/againrom.TestReleaseTestPlayRefusesAnOversizedModPageBeforeWritingOrStarting`
  assembles installed mission 20 into a temporary authored mod and selects
  and opens it in the editor. A page of exactly `mod.MaxFileBytes` passes F5
  and the recorded arguments plus `-check`, creating an owned destination.
  The same page at `mod.MaxFileBytes+1` then produces the loader's size
  refusal in `Message()`, records no launch and preserves the complete
  profile and source snapshots. The launch is a recording fake throughout.
- The original story tests fail on main `62ca3ea4`: the package, the type
  and the key do not exist there. The correction refusal tests fail on
  candidate `bc7c673b`, which writes or starts where a refusal is required.
  The oversized-page release loss control fails on EN and RU at that
  candidate; the in-bound page passes the game's `-check` on both roots.

## Open debt

- A test play starts with character generation; a start straight into the
  mission with a party does not exist.
- What a win of the test mission opens is Unknown: no test plays it to its
  end. Started by `-mission`, it holds no stashed campaign.
- The game's own warnings of the test mod print on its error stream, which a
  game started from the editor has no console for.
