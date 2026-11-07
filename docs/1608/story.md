# Extended map scripts in the engine

## Intent

Let a first-game mod map hold a script the original trigger record cannot:
more conditions and actions per trigger, OR of condition groups, more
variables and more triggers. This is step 2 of the extended map scripts: the
engine reads, compiles, runs, saves and loads such a map. The map editor
opens and writes one in step 3. Every shipped map, every `.alm`, and every
World, hash and byte form of an original program are unchanged.

## Authority

Owner decisions for the extended map scripts:

1. An `.almx` is the ALM layout plus one Againrom record of a typeId at or
   above 10, which the original skips (`ALM-REQ-055`).
2. The switch comes at the first construct the original record cannot hold;
   no automatic fitting back.
3. OR: a trigger fires when any of its AND groups holds.
4. Limits: 4096 variables, 10000 triggers, 64 conditions and 64 actions per
   trigger.
5. Extended variables are their own array.

Owner rulings: one builder per kind; SAV is the one save format with one
producer and a save is never refused; consuming arithmetic of randomness
matches the original, one seed. Step 3 turns the typed limit error into the
editor's confirmation. Rows DIV-4599 to DIV-4604 in
`docs/divergences/mods-maps.md`; the modder's form is `docs/MODS.md`,
"Extended map scripts (.almx)".

## As-built behaviour

### Kinds and builders

| Kind | Builder | Where |
| --- | --- | --- |
| type-7 walk | `alm.FrameScript`, unchanged | `pkg/formats/alm/script.go` |
| extended record | `alm.DecodeScriptRecord`, `alm.EncodeScriptRecord` (`AppendScriptRecord` appends the encoder's bytes) | `pkg/formats/alm/scriptrecord.go` |
| script model | `alm.ScriptTrigger` gains `Group`, `Conditions` and `Actions`; `Map.Script` fills them from the record | `pkg/formats/alm/script.go`, `alm.go` |
| 184-byte trigger writer | `mapedit.NewScriptTrigger`, `SetTriggerSlots`, unchanged | `pkg/mapedit/script.go` |
| limits | `scriptprofile.Extended` | `pkg/scriptprofile/profile.go` |
| compile | `mapload.compileScriptFrom` (`extendedTrigger`) into `sim.NewExtendedScript` | `pkg/mapload/script.go` |
| evaluation | sim `triggerHolds` | `pkg/sim/script.go` |
| byte form and hash | the script section in `scriptbinary.go`, wrapper `XSC1` | `pkg/sim/scriptbinary.go`, `formwrapper.go` |
| SAV supplement | `currentScriptProgram.Extended`, written by `projectCurrentActions` through `captureCurrentScriptProgram` | `pkg/game/currentscriptprogram.go` |
| map list | `mapExts`, `isMapExt` | `pkg/game/maplist.go` |
| mod mission map | `mod.(*Mission).ReadFiles` (`pkg/mod/missions.go:628`), unchanged; the gate is `CheckModMap` | `pkg/game/mapcheck.go` |
| editor | `refuseExtendedMap`, refusal only | `pkg/game/mapinspectionedit.go` |

No second builder was added. `pkg/formats/sav` is untouched.

### The record

The record is typeId 100, read only in the first-game dialect. Its payload
is version 1, the variable count and the trigger count, then each trigger's
64-byte name, fire-once word, condition count C and action count A, C
conditions of 16 bytes (group, left id, right id, code) and A action ids.
The decoder refuses, with a `*ScriptRecordError` naming the trigger and the
entry: a wrong version; a count past `scriptprofile.Extended`; a fire-once
word other than 0 or 1; a first group
other than 0 or a group step other than 0 or +1; an id outside 1..5000; a
code above 5; a payload that ends early or runs past the last trigger.
Counts are checked against the payload length before any allocation. A map
with the record and a type-7 trigger, or with two records, is refused. With
the record `Map.Script` takes the triggers from it, `Extended` is set and
`Variables` holds the count.

### The machine

`sim.NewExtendedScript(variables, checks, instants, triggers)` builds a
`scriptMachine{extended, variables}`. Each check writes a result slot of its
own (`scriptResults`), read only by conditions. Variables 0..99 are the
session registers, variables 100..V-1 the array `scriptVars`; check 19 and
actions 3 and 8 address them through `registerAt`/`setRegisterAt`, bounded by
V. Latches 0..999 are the session latches, the rest `latchTail`. A trigger
holds when any group holds, a group when all its conditions hold,
short-circuit. A trigger runs every action in its list in order. A condition
id naming no built check reads 0 (`checkResult(ScriptNone)`); an action id
naming no built action runs nothing. Variable 93 is register 93, which each
pass increments; a map with fewer than 94 variables cannot read it. The pass is the original's phase: once every 16 ticks,
checks first, then triggers in map order.

`Script.Recompile` rebuilds a program on its own machine and dialect, and
`restoreCurrentScriptRoles` uses it; `RestoreScriptBindings` and
`RestoreScriptProgram` rebuild through `newScriptOn` on the program's own
machine. Each keeps an extended program extended.

### Byte form and hash

An extended World writes the script section with no program in the base
section and carries the program, its variable count, the extended variables,
the results and the latch tail in the `XSC1` wrapper (version 133). The hash
covers them. An original World writes no `XSC1` and its bytes are the
base's.

### SAV

The session block is written as before: variables 0..99 and latches 0..999.
`currentScriptProgram.Extended{Variables, State}` holds the variable count
and `sim.ScriptExtendedState{Variables, Results, Latches}` for the rest, in
the `/CurrentState/AgainromActions` leaf. LOAD compiles the program through
`NewExtendedScript` and applies the state with
`World.RestoreScriptExtendedState`. A second-game program with `Extended`
is refused.

### Name gates

The map list lists `.alm` and `.almx`, compared without regard to case. A
mission map read from `scenario/NN.alm`, a mod mission's map among them,
honours the record. `CheckModMap` refuses a map carrying the record whose
name is not `.almx`. The map editor refuses a file named `.almx` and any map
carrying the record, naming step 3. `almtool script` prints the record line
and each condition's group.

## Proof

- Census of every shipped map: `DirMaps`, `ArchiveMaps`, the loose maps'
  compiled programs and missions 1..300, 320 ticks each, with hash and script
  section digests, over `gameversions/en`, `ru` and `rom2-ru`. 412 lines
  (en 104, ru 96, rom2-ru 212): 155 opened maps (38, 34, 83) and 102 missions
  run. Base and candidate outputs are byte-equal (`cmp`). 0 of the 155
  opened maps carry the record. The instrument was a temporary test, not
  committed.
- `pkg/formats/alm/scriptrecord_test.go`: layout and round trip; the record
  supplies the triggers; record beside a type-7 trigger and two records
  refused; each limit refused with a typed error; truncated and oversized
  counts refused.
- `pkg/sim/scriptextended_test.go`: OR of two groups, each branch fires and
  a third case fires neither; the extended World crosses the byte form with
  its state, an edit of a variable, latch or result changes the hash, cut
  forms are refused; an original World writes no `XSC1`.
- `pkg/mapload/scriptextended_test.go`: groups, lists and misses compile;
  the second game refuses the record.
- `pkg/game/extendedscript_test.go`, on a synthetic `.almx` with 150
  variables (variable 120 past 100), one trigger of 6 conditions in two
  groups and 6 actions, and a trigger at position 1001:
  `TestMapListHoldsExtendedMapFiles` (fails at base: lists `Plain.ALM`
  only); `TestExtendedMapListsLoadsAndRunsEachGroup` (lists, loads and runs
  headless; each group fires, neither fires in the third case);
  `TestExtendedScriptSaveRoundTripResumesTheSameWorld` (mid-mission SAV with
  variables 146 and 148 and latch 1001 changed before the save; equal hash,
  state and triggers after a cold LOAD; 40 ticks of equal hashes; trigger
  After fires only after the reload, the latched trigger does not; a SAV
  whose supplement says 0 loads a World where After never fires);
  `TestModMissionManifestNamesAnAlmxMap`;
  `TestMapEditorRefusesAnExtendedScriptMap`.
- `cmd/almtool/script_record_test.go`: the record line and groups print.

## Open debt

- The map editor opens and writes `.almx` in step 3.
- `ScriptFiring.Pairs` is a fixed three-pair array: a firing of an extended
  trigger records its actions but no condition values. `ScriptTriggerRun`
  lists every comparison that ran.
- The supplement leaf is bounded at 8 MiB. A mod map is at most 1 MiB
  (`mod.MaxFileBytes`); that the JSON of its largest program stays under the
  bound is inference, not measured.
- The second game has no extended record; the model and typeId 100 leave
  room for one.
- The EN and RU release witnesses of SAVE/LOAD and the map list ran in no
  lane.
