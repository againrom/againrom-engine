# Story 1624: a mod holds its own speaker records

## Intent

A map maker gives a clicked unit that no speaker record shows its own face and name, without
touching the install's `npc.reg`. The record lives in the mod that holds the map.

## Authority

Owner direction on the seat's proposal: a mod holds its own speaker records (a picture, a name and,
by choice, one unit of one map); the game reads them as it reads the game's records; the editor puts
the record into the mod that holds the map. A map outside a mod keeps the refusal text.

## As-built behaviour

- Data kind `speakers`, one row in `dataKinds`, file `data/speakers.toml`, key `id` in 20000..99999.
  It is not script-loaded: `modrt.readBesideManifest` reads every non-script kind beside the
  manifest, so a mod that carries the file works from the game with no script line and no editor.
- `data.NPCCatalog.WithLayer` lays the records over the one catalog: `Entry`, `Records`,
  `Portraits` answer them. Kind `NPCModPicture` draws `Picture` as it stands; `speakerCast.matches`
  refuses it, so no live actor takes it and every existing resolver arm is unchanged.
- `FrontEnd.setModSpeakers` applies the layer after the characters and rebuilds `NPCFaces`.
- `MapInspector.useModSpeakers` lays the records over a copy of the editor's table at `SetMod`.
- The speaker pick offers a record on a miss with causes NoFit, NotFirst, NoPicture, ByHero or Down
  when the map lives in a writable mod. A prompt takes the name; Enter opens the host's file window
  for the picture. `pathChoice` (`pkg/ui/mapeditorpath.go`) is the one file choice: `request` holds
  what the window lists (start directory, filters, title) and `chosen`, `cancelled`, `failed` what
  the editor does with the answer. `mapChoice` builds the Open and Save As choices from the
  unchanged `pathRequest`. The speaker picture is a second value: PNG files, starting in the mod
  folder, `editor.speaker.add.picture` as its title. The file window is the only route to a
  picture: with no window to show the choice is a no-op with a message on the line, and no path is
  ever typed.
- The writer is `writeModFile`, split out of `writeModMap` in `modmapsave.go`. Everything that can
  refuse runs before the first write: the picture is held to `mod.MaxFileBytes`, the mod reader's
  limit, the file's whole new bytes pass `mod.ReadSpeakers` with the picture in memory, and the
  merged layer is built. Then the picture is written, then the file. A write that fails puts back
  what the add changed: a new picture and a folder the add made are removed, a picture that stood
  at that path is written back, and the digest the next save compares stays right.
- `AppendSpeaker` writes strings through `writeString`, the inverse of the mod's `readString`: one
  escape table serves both, so a name holding a no-break space round-trips.
- While the mod's records did not load (`contextIssue` set) `CanAddSpeaker` is false, `AddSpeaker`
  refuses, and the click shows `editor.mod.readonly` and the issue, the line `ModSummary` shows.
- A new record's number comes from `freeSpeakerNumber`, the one allocator. The band is 20000 to
  99999; 99999 is the largest number the second game's tag reader takes whole (five bytes after
  `npc=`; the first game's reader, the catalog's `int32` and the message window's `int` allow more).
  A stable FNV-1a hash of the mod id gives the first number tried; the walk goes up, round to 20000
  at the top, past every number held by this mod's file, the layered catalog and any other mod's
  `data/speakers.toml` in the same mods folder, and past a number whose `speakers/<id>.png` a record
  of this mod names. A mod that names a number on purpose keeps the merge rule.
- The editor's clicked-unit flow always binds the record; `map` and `unit` are optional in the
  file.
- `unit` is the unit's script number (`alm.Unit.UnitID`). `UnitSpeaker` answers the bound record
  for the unit holding that number wherever it stands in the map's unit list. A unit whose number is 0
  or held by another unit of the map has no number that names it alone: the click refuses it with a
  stated message before the name is asked, and `AddSpeaker` refuses it.
- The speaker field names the new record straight after the write, as it does for an installed
  record: `refreshSpeakerChoices` reads the message cards' lists again.
- An empty name leaves the prompt open. A picture the host refuses, the 72x96 check among its
  reasons, brings the name prompt back with the name kept and the refusal on the line. Esc at the
  prompt and Cancel in the file window return to the armed speaker pick.
- A test play and a package build carry the record. The project route and the package build copy
  the mod folder whole. The single-mission test play carries the file and its pictures on the
  mission through `carryModSpeakers`. The record is not named in the project's pending files:
  naming it there made the next save into an owned folder refuse the files as unrecorded.
- A saved speaker is the number the map's message tag names; the story adds no SAV field.
- A map outside a mod keeps today's refusal text.

## Proof

Synthetic data, each test failing under a planted fault in the production code it covers.

`pkg/game/modspeakers_test.go`: write and resolver answer, bound unit, face equality between info
window and message window, cold read through `SetMod`, two mods naming one record with and without
`load-after`, a map outside a mod; the click, prompt, file choice and write over the real host; the
72x96 refusal; the binding following the script number when the unit list moves; number 0 and a
shared number refused; a game started with such a mod holding the face in `FrontEnd.NPCFaces`
(`setModSpeakers`); `carryModSpeakers`; a test play on both routes; a save into the owned folder
and a package built into another folder.

`pkg/game/modspeakers_add_test.go`: a picture over the mod reader's limit and a failed file write
leave no line, no picture and an unchanged digest; a picture already at the path comes back; a
name with a no-break space round-trips; a mod with a bad picture on one record and a mod with a
failing script are refused at the click and at the write; the number reads the mod's own file;
two mods in one folder, two in separate folders, a re-add after a record was removed, the walk
round the top of the band, the same id at the same start; the band's top read whole by both games'
tag readers.

`pkg/mod/speakers_test.go`: the name writer against the reader, the band's edges, a pending picture.

`pkg/modrt`: `TestFixtureModsGiveTheRecordedMergedData` holds the old lines byte equal and the
eight lines of the new fixture. `Mission.Carried` is left out of the JSON when empty, so a mod
with no carried files hashes as before.

`pkg/ui/mapeditorspeakeradd_test.go`, in EN and RU where words show: the chain's request and
write, the name shown at once, an empty name keeping the prompt, cancel at the prompt and in the
file window, a refused picture bringing the name back, no window to show, a refused binding at the
click.

The map choice stays equal by the existing tests: `TestEditorOpenDialogKeyAndMenu`, the other path
dialog tests in `pkg/ui/mapeditorpath_test.go`, `TestEditorFailedPathDialogDropsCancelCheckpoint`,
`pkg/game/mapinspectiondialogs_test.go`, and the modes walk with `MAPEDITOR_MODES_FULL=1`.

## Open debt

- `contextIssue` is set once at `SetMod`. A mission map that was absent at the read and saved later
  keeps the mod refused for speaker records until the mod is opened again.
- A neighbour mod's file that cannot be read or parsed contributes no numbers to the walk.
- Picture previews: the file window is the operating system's list; the editor has no list of
  files with previews. On a platform with no window (the `pathdialog` stub) the choice is a no-op.
- The record binds only the editor's answer; in the game a message tag naming the number draws the
  picture on any map.
- A mod file written before the binding changed holds a placement index in `unit`; a value that is
  no script number of the map binds no unit, and a value that is another unit's number binds that
  unit. Such a file could only have been written by the earlier candidate, which never shipped.
