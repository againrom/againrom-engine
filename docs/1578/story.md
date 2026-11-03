# Every file a player or modder keeps is written by one file-set transaction

## Intent

Before, each writer of a kept file staged, published and rolled back on its own: the save store, the
named save set, the hall of fame, the mod pack, the base dump, the headless frame, the editor's map
writers, the test-play files, the editor profile file, the editor snapshot and `starter.ini`. Now
each states its policy as data, a `hostfile.Set`, and calls `hostfile.Commit`. Nothing changes for a
player: every file keeps its bytes, its name, its place, its refusal rules and what a failure
leaves, on both games.

## Authority

- Owner direction: the story brief (`review/story1578-host-files/BRIEF.md`) and audit-20 plan
  section R34. Owner ruling: architecture work, no player change.
- The files are Againrom's own; no claim and no divergence row applies.

## As built

| kind | builder |
|---|---|
| host-file publication (every kept file) | `hostfile.Commit` in `pkg/hostfile/hostfile.go`; each caller a `Set` |
| original-install write fence | `hostfile.Fence` and its functions in `pkg/hostfile/fence.go`, moved unchanged from the save store |
| publish without replace (platform) | `pkg/hostfile/publish_windows.go`, `publish_other.go`, `publish_link.go` |
| save folder and name validation | unchanged in `pkg/game/savestore.go` and `savenamed.go` |
| map, mod pack and SAV bytes | unchanged producers; only the file step moved |

A `Set` holds its `Files` and the policy: `Dir` and `Temp` (where and under which pattern a staged
file is made), `Sync`, `Dirs` (`Parents` made and kept, or `Owned` folders under `Root` a failure
removes, `MakeRoot`), `Discard` (a staging folder removed at the end), `Claim` and `Recover` (the
save store's claimed name and temp recovery), `Check` (a test before each step), `Ops` (the
injectable file seam) and `Words` (the error texts). A `File` has `Overwrite` (`Refuse`, `Replace`,
`Backup`), `Staging` (`Sibling` temp or `Direct`), `Seen`, `Prior`, `Remove`, `Folder` and `Backup`
(a staged folder swapped into place), `Stale` (the folder's backup lies outside the stage and is
cleared first, as a page folder's `.old` is; a mission package's backup lies inside the stage and is
not), `Stream` and `Keep`. Temp recovery refusals are returned as they are, without `Words.Create`.

Policy rows:

| caller | row |
|---|---|
| single save, screenshot (`SaveStore.write`) | Sibling, Refuse, Claim, Recover, Sync, prior texts |
| named save set, owned slots, `options.txt` (`commitNamedSaveSet`) | Sibling, Backup, Seen from preparation, `.save-dialog-*`, Check |
| save delete (`deleteNamedSaveSet`) | Remove rows with backups; a rollback publishes them again |
| hall of fame (`fameStore.add`) | Sibling, Replace, `.againrom-fame-*.tmp` |
| mod pack (`PackMod`) | new: Direct Refuse 0644; replace: Sibling Replace `.<packed name>.*` |
| mod assembly (`AssembleModMission`) | Direct Replace 0644, Parents 0755; written = files done |
| base dump (`writeNewFiles`) | Direct Refuse 0644, Owned 0755 under the dump folder, MakeRoot when missing |
| headless frame (`writeHeadlessFrame`) | Direct Refuse 0644, Parents 0755 |
| editor map save (`writeEditedMap`) | new: Direct Refuse 0600 Sync; over: Sibling Replace `.map-save-` Sync, the old mode kept |
| map with pages (`writeEditedMapPages`) | stage `.map-pages-`; pages Direct Replace 0600, Parents 0700; one Set of the map row and the folder row, backup `<folder>.old` |
| labels file (`placeMapLabels`) | the map's replace row |
| map saved into a mod (`writeModMap`) | Sibling Replace `.mapedit-*.alm` Sync |
| test-play file (`writeTestModFile`) | Sibling Replace `.testplay-*`, Sync when durable |
| test-mod copy (`copyRetainedTestMod`) | Direct Refuse 0644 |
| mission package (`installMissionPackage`) | stage `.testplay-*`; folder row, backup `<stage>/previous` when owned, Discard the stage unless it keeps the backup |
| editor profile file (`editormcp.WriteProfileFile`) | Sibling Replace with the caller's pattern, Sync, texts `write <what>: ...` after the temp's creation |
| editor snapshot (`cmd/mapedit -snapshot`) | Direct Refuse 0600, Stream the PNG encoder, Keep |
| `starter.ini` (`app.save`) | Direct Replace 0644, Parents 0755: still an in-place write |

`internal/archtest/hostfile.go` reads production code in `pkg`, `cmd/againrom`, `cmd/mapedit` and
`cmd/starter`. Outside the transaction file and the three primitive files it refuses `os.Rename`,
`os.CreateTemp`, `os.MkdirTemp`, `O_EXCL`, any `MoveFileEx` name, `os.Link`, `os.WriteFile`,
`os.Create`, an `os.OpenFile` whose flags create or are not plain constants, the system-call
forms (`Rename`, `Link`, `MoveFile`, a creating `Open` or `CreateFile`), and a dot import of `os`
or `syscall`, as a call or a function value. Its named entries are the two write probes
(`cmd/starter/app.go`, `pkg/game/runtimeprofile.go`) and developer output nobody keeps: the cutscene
and route witnesses, the frame log, the audio probe, the editor's failure report and the starter's
`-shot` picture. An entry that matches nothing fails.

## Proof

- Record test, committed alone before the change (`720093087`): `TestHostFileRecord` (`pkg/game`),
  `TestHostFileRecordProfileFile` (`pkg/editormcp`), `TestHostFileRecordStarterIni`
  (`cmd/starter`). Each records files and bytes after a write, the answer over an existing
  destination, each fence refusal, and the temp and backup names a refused or failed write leaves,
  with failures injected at each step of the save seams. Equal after every commit of the change.
  Its existing golden lines are unchanged; two rows were added after review found two failure
  paths that differed from the base: `single save recovery` (a leftover temporary save the store
  cannot remove: the base's text, without the `create temporary save: ` prefix) and `install
  previous` (a mission package whose ID is `previous` fails on the rename and the old package
  stays). The save folder's listing cannot be refused through the store's `Ops`, so that recovery
  text is not recorded. A later commit writes the temporary save's process ID as `#`
  whatever its length; before, a process ID under five digits made the record differ.
- Planted change: the base dump's row from Refuse to Replace fails `TestHostFileRecord` at
  `dump over a file`, recorded as an `ErrBaseDumpExists` refusal and now `ok`.
- Guard: `TestCheckHostFilesNamesEachShape` plants one instance per shape and the clean forms;
  `TestHostFilesAreOneTransaction` reads the live tree. `TestLiveTreeClean` (the `dag`) holds the
  `pkg/hostfile` row and its importers.
- Focused tests: `./pkg/game -run 'Save|Named|Publish|ModPack|ModAssemble|ModMap|MapSave|MapText|TestPlay|BaseDump|HeadlessFrame|RuntimeProfile|Fence|Original|HostFile|Fame|Delete|Screenshot|Options' -skip '^TestRelease'`,
  `./pkg/editormcp -run 'Profile|HostFile'`, `./pkg/hostfile`, `./cmd/starter`, `./cmd/mapedit`.
- No install was read. The record cannot write the headless frame, the editor snapshot or a
  successful mod assembly without one; only their refusals are recorded. The release tests that
  save, pack or assemble a mod, dump a base, save a map from the editor or into a mod, start a test
  play or write a headless frame carry those writes on EN and RU.

## Open debt

- The fence is one package with the transaction, but callers still call it before `Commit`; a
  `Set` does not carry its fence as a field.
- The guard reads names, not types: a caller that drives `hostfile.OS` or another `Ops` value
  directly outside a `Set` is not refused.
- The two write probes keep their own `os.CreateTemp`.
- No release test writes `starter.ini`; its in-place write is covered only by the record test.
