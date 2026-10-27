# Runtime ids after LOAD as the original gives them

## Intent and authority

Every actor, Building and Sack carries a runtime id, the Token `+0x04` word a SAV
stores. The engine now gives, releases and rebuilds those ids the way the
original does, so the next unit, Sack or Building made after a LOAD takes the
original's id, and a SAVE writes the ids the World carries.

Owner rulings: SAV is the only save format; a field is written from the current
state, then a rule derived from evidence, then the loaded document's bytes, then
a constructor value research established; the second game behaves as the first
until disproved.

Claims (knowledge pin k362):

| Rule | Claims | Grade followed |
|---|---|---|
| One bitmap, lowest clear non-zero bit, for actors, Buildings and Sacks; no Item or Effect id | `SAV-1368`, `SAV-ID-015`, `MOVE-ID-016` | High; Medium that no other routine gives an Item or Effect id |
| Release: Building or Sack on destruction, actor at decay stage 5 or party removal; Item and Effect bits never | `SAV-1370` | High / Medium |
| An Item or Effect built new holds 0 | `SAV-1369` | High |
| LOAD: clear to bit 0, mark the low 16 bits of each non-zero saved `+0x04` in archive order, a Sack's constructor first taking the lowest clear bit | `SAV-1371`, `SAV-1372`, `SAV-DOC-053`, `SAV-1337` | High for the arithmetic; Medium for the list order |
| One Outpost per opcode-0x10003 check, built after the map | `SAV-1373` | High / Medium |
| The tick-list insertion gives a spawned actor, the Ghost among them, its id | `MOVE-187`, `MOVE-ID-016` | High / Medium |
| No owner id repeats in the corpus | `SAV-1374` | High |

## As built

| Kind | Builder |
|---|---|
| Runtime-id allocator | `pkg/sim/runtimeids.go`: the marks are bit 0, the low 16 bits of each id an object carries, and the World's orphan bits; `allocateRuntimeID` gives the lowest clear bit. The document allocation of the generated SAVE (`generatedDocumentBuilder.runtime`) is removed. |
| Carriers | Entity `SourceBinding.RuntimeID` (also without a source binding), the dead manager's current id, `SavedStructure.RuntimeID`, a registry Sack's Token, and a Sack without an object row (`Sack.runtimeID`). Orphan bits and row-less Sack ids persist in an outer World form wrapper (version 128), written only when either is present. |
| Constructors that ask the allocator | Mission entry of a constructed World (party, then Building rows, then the other on-map actors, then the map's Sacks in Sacks order, `SAV-ID-015`; `World.InsertSackRuntimeIDs`), placement of an off-map actor without an id, the Control Spirit ghost, the summon cheat, a poured Sack, a ground Sack's object row. |
| World LOAD | `restoreLoadedRuntimeIDs` (`pkg/game/savruntimeids.go`) after the actions restore: each bound live actor takes its record's id (`World.RestoreActorRuntimeIDs`), then the replay (`World.RestoreRuntimeIDs`) keeps as orphans the replay's bits no object carries. A ground Sack takes its record's id at ground restore (`World.RestoreGroundSackRuntimeIDs`). An original SAV and one the engine wrote load by the same route. |
| SAV document writer | Each record is written with the id its World object carries, a Sack without an object row included (`World.SackRuntimeIDs`, `pkg/game/savcurrentitems.go`); `projectCurrentRuntimeIDs` writes only the projectile `actiontarget` and the selection from the World. An older SAVE's runtime coordinate is read and not applied. |

A selection that repeats an id names as many actors; when more actors carry the
whole id, the lowest entity ids take the entries (`DIV-2504`). The second game
uses the same allocator, LOAD and SAVE (`DIV-4106`).

No shipped first-game map holds an opcode-0x10003 action or condition node
(28 of 28 maps in the EN archive read), so no Outpost is built and none is
modelled.

## Proof

Instruments: `pkg/game/runtimeidreplay_release_test.go` (release, EN and RU),
`pkg/game/runtimeidsave_test.go`, `pkg/sim/runtimeids_test.go`.

- Replay witness. The witness reads each file's Token-headed records from the
  body bytes in archive order and computes the bitmap and the next id. Over the
  owner corpus and the install's top-level SAVs, our World after LOAD holds the
  same bits and gives the same next id. EN: 83 of 83 original files, 1 of 1
  engine file, 22 of 22 install files; 3 files are the `SAV-1371` example
  `3407c9fce2ac9ce0` and replay to 23. RU: 83 of 83 original, 1 of 1 engine,
  4 of 4 install files; 2 files are that example.
- Pair search. The corpus holds one LOAD-then-SAVE pair with both sides:
  `2026-09-24/exp-engine-lineage/game1017-engine-from-0017.sav` saved by the
  original as `game0001-original-resave1017.sav`. The original created no Sack,
  unit or Building between them (277 owner records on each side); the replay
  gives 281 next. The other resaves name sources the corpus does not hold.
- Writer sensitivity. A live change of an actor's World id moves the written
  word, and ids the World repeats are written repeated. Against base `ef2b429d`
  the repeated case fails: actor 3 written 1, World 77.
- LOAD then SAVE keeps every owner id: 83 of 83 original files, 22 of 22 EN
  and 4 of 4 RU install files; a second SAVE of the written file keeps every id
  in every file on both roots (106 EN, 88 RU).
- Cold LOAD of the written file equals the continued World's allocator and its
  hash after one tick in 82 of 83 original files on each root and in every
  install file. The exception, `2027-09-07/game0125.sav`, is the Effect debt
  below.
- A World that built a Sack since its last LOAD, or never loaded, lacks the bits
  the next LOAD's Sack constructors take (`SAV-1371`). Release and ordinary
  tests that compare a cold LOAD with the continued World first give the
  continued World the file's orphan bits (`adoptLoadedAllocator`); every carried
  id is still compared. The M2 round trips do the same: the SAV round-trip
  gate, the writer admission cycles, the dead-units census and the frozen
  terminal fine witness.
- M2 (`scripts/check-milestone2-acceptance.sh`), EN and RU: every instrument
  passes except the round-trip gate's `2026-08-14/game0013.sav` and its copy
  under `EXP-0278-human-runtime-en` in the dying variant (entity Class,
  `stoodDown`, `areaCostLive`); `origin/main` `f43dc35b` fails the same two
  cases with the same fields, and its gate census lines equal this branch's.
  The writer census lines (rules, families, debts, mismatches, sensitivity
  controls) equal `origin/main`'s on EN.
- ROM2 save-test groups: 122 of 122 declared subjects pass on EN and RU.
- Profile witnesses (`pkg/game/testdata/profilewitness/`): `world=` and
  `save=` fields moved, no `frame=` field. The first mission's map Sacks carry
  ids; the second game's new-game Weapon and Item hold 0, not 2 and 3
  (`SAV-1369`).
- Loss control. With the replay call removed from `restoreLoadedRuntimeIDs`,
  the release witness fails (EN: 9 of 83 original, 0 of 1 engine and 3 of 22
  install files then match the replay) and the ordinary
  `TestCurrentRuntimeLoadReplaysTheSavedIDs` fails: its Sack fixture loads
  marks `[0 501 502 503 61185]`, next 1, where the replay gives
  `[0 1 501 502 503 61185]`, next 2.

## Open debt

- An actor Effect the LOAD does not bind is written anew at SAVE with id 0 and
  default Token fields (base did the same). The next LOAD's replay then lacks
  its mark: in `game0125.sav` the cold World leaves bit 11 clear where the
  continued World holds it. In EN, SAVE wrote other Item or Effect ids than
  LOAD read in 3 original files and 1 install file.
- A town SAVE has no World and gives a new actor the lowest id its document
  leaves free (`DIV-2504`).
- `2026-10-07/saveorcsdontgo.sav`, written by an earlier engine, gave departed
  actors ids; a current SAVE writes a departed actor's record with no id and
  none for an actor that leaves no body. Only its LOAD replay is checked.

## Research questions

- B1: LOAD a first-game SAV, create one actor before any death and SAVE; record
  the new actor's `+0x04`.
- B1: LOAD a SAV whose selection names an id two actors carry; record which
  actors are selected.
- B1: on a map with an opcode-0x10003 node, if one exists in any installed map,
  record the Outpost `+0x04` and `+0x80` after LOAD.
- B1: LOAD a SAV that holds a non-zero Item or Effect `+0x04` and SAVE at once;
  record that word.
- B1: start a mission with several ground Sacks on the map and SAVE at once;
  record each Sack's `+0x04` with its cell.
- B1: start a mission with a ground Sack, SAVE, LOAD that SAV, create one Sack
  and SAVE; record the new Sack's `+0x04`.
- B1: start a mission, SAVE, then in the same session and after a LOAD of that
  SAV create one Sack each and SAVE; record both new Sacks' `+0x04`.
