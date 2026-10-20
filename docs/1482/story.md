# Writer carrier bits for Againrom SAVs

## Intent

The continuation supplement's `WriterCarriers` field names the original carriers the writer built
from its World; each set bit makes that carrier's milestone-2 byte instrument check an
Againrom-written SAV (`docs/SAV-SUPPLEMENT.md`). The writer recorded 0, so no Againrom-written SAV
was checked by any original-carrier instrument (SAV gap audit G18, DIV-2885). This story sets
every bit whose instrument passes on the files the writer produces and states, for each other
bit, which part of the written carrier keeps it clear. The player sees no change.

## Authority

`docs/SAV-SUPPLEMENT.md`: a carrier implementation sets its bit only after its writer builds that
complete carrier from state, and its story names the bit and proves the corresponding
original-carrier byte instrument. The field is metadata: it creates nothing at LOAD and changes
no World state. Owner SAV ruling (AGENTS.md, "SAV is the only save format"). The terrain
instrument reads `SAV-BLOCK-011`'s block array inside `TERR-PASS-053`'s sweep window.

## Result

The writer records `WriterCarriers` `0x400` (bit 10, TerrainBlockPlane) on every first-game SAVE.

| Bit | Carrier | Writer | Reason the bit stays clear |
|---|---|---|---|
| 0 | Buildings | clear | A mission-start World writes its Building cell links from the constructor document and marks them absent (`AbsentStructureCells`); LOAD drops them. On the EN writer census the instrument differs on 69 of 83 world files (retained Building fields). DIV-2885 |
| 1 | CellRecords | clear | The written cell table comes from the document and is marked absent (`AbsentCellRecords`, `AbsentCellTails`); LOAD drops the records (`LayerCount present=false`). Census: 83 of 83. DIV-2885 |
| 2 | DeadRoots | clear | LOAD of an Againrom-written SAV does not rebuild the dead roots the instrument reads: a new game's kills are omitted, and on the census the root order, source tuple and current tuple differ (83 of 83). DIV-2885 |
| 3 | DyingOwnerActors | clear | A mission-start World's actors carry no source binding, so a dying actor written from a new game has no live source identity ("dying source has no live source binding", mission 10 and 20 probes). Census: passes, 20 files, 77 subjects. DIV-2885 |
| 4 | Groups | clear | The World holds no Player key for a Group owner named by slot and no source archive binding for a fresh actor (`docs/1395/story.md`); every written Group member reads as "actor archive has no live entity". Census: 83 of 83. DIV-2885 |
| 5 | MoverRoutes | clear | Same actor binding: "orphan order entity" and "expected live mover/order source unavailable"; a new game writes `MotionCarrier` false. Census: 83 of 83. DIV-2885 |
| 6 | Players | clear | The Player Diary is written from the document and marked absent (`AbsentDiaries`); F58 actors have no live binding. Census: 65 of 83. DIV-2885 |
| 7 | Sacks | clear | LOAD binds a written Sack's Token, gold and item graph otherwise than the instrument reads an original's ("retained Token/Gold/lifecycle differs"). Census: 82 of 83; new game 8 of 8. DIV-2884, DIV-2885 |
| 8 | SessionBlock | clear | A mission-start World holds no raw session head; the writer writes it from the constructor and marks it absent (`AbsentSessionHead`), LOAD drops it (`RawSessionHead` live zero). New game: 8 of 8; census passes. DIV-2885 |
| 9 | SpellEffects | clear | LOAD of an Againrom-written SAV binds no typed saved world spell effect (`typed roots: live=0 raw=21`, `live=0 raw=1`): 2 of the 4 census files with effects differ; new games write none. DIV-2885 |
| 10 | TerrainBlockPlane | set | The Blocks are the World's cell planes (`currentSpatialPlanes`, `currentBlockPlaneDelta`); LOAD rebuilds the planes from them with the one plane builder |

Counts are from `TestMilestone2<Carrier>` run with `AGAINROM_M2_ORIGINAL_CONTROL=1` on EN files
this tree wrote before the bit was set: 8 New Game save points (`savGateNewGames`), one town SAVE,
two mission-10/20 dying saves, and the writer census output of 125 originals (83 world, 42 town).

## As built

- `currentWriterCarriers` (`pkg/game/savactioncurrent.go`) is `currentCarrierTerrainBlockPlane`.
  The one producer, `FrontEnd.ExportCurrentSave`, records it in a mission SAVE when the saved
  World holds the cell planes (`projectCurrentActions`), and in a town SAVE when the install
  carries the `map.reg` cost table its Worlds build the planes from (`currentCityDocument`). Every
  save point (dialog, quicksave, timed and mission-start autosave) reaches that producer.
- A second-game World holds no planes and its install names no cost table (DIV-3095, DIV-3093);
  its mission and town SAVs record 0.
- LOAD reads the field only for admission; no World field, hash input or LOAD branch changed.

## Proof

- `TestMilestone2WriterCarrierInstruments` (session corpus audit, run by
  `scripts/check-milestone2-acceptance.sh`): writes the 8 New Game save points and one town SAVE
  through the SAVE dialog route, requires each to record `0x400`, and runs
  `TestMilestone2TerrainBlockPlane` on them in a child process with the original control unset,
  so the written bit selects the instrument; every file must pass and no file may fall back to the
  round-trip rule. Loss controls: a copy whose middle Block row has dynamic byte `0x0f`, and a copy
  with a row below the sweep window, each fail beside the clean copy.
- Writer census: each written world file is checked by the claimed carriers' per-file comparison
  (`milestone2WriterCarrierCheck`); a difference is a census mismatch under
  `writer carrier|WriterCarriers`, and `SAV-WRITERCENSUS-CARRIER` reports the files and records.
- `TestMilestone2WriterCurrentRoundTrip` requires the mission and town SAVs to record the mask and
  exactly its bits to select their instruments; `TestCurrentActionWriterRecordsHeldCarriers` pins
  the record to the planes rule on the fixture World; `TestReleaseSecondMissionSaveLossControls`
  requires a second-game SAV to record 0.
- The terrain instrument detects corruptions of the Blocks stream (order, window, dynamic gate).
  A changed Static or Dynamic value inside a valid row is read back by LOAD and passes the
  instrument, for an original and an Againrom SAV alike; the M2 writer rule's World hash across
  SAVE and cold LOAD covers the planes, which the World hash includes.

## What moved

Only the `WriterCarriers` value of a first-game SAVE: `0` becomes `1024`, and the file grows by 4
bytes. World hashes, LOAD and simulation do not move. Release pins of SAV digests moved with it:

| Pin | Where | Moved |
|---|---|---|
| `TestReleaseProfileWitnessIsUnchanged` | `pkg/game/testdata/profilewitness/rom1-en.txt`, `rom1-ru.txt` | `save=` digest on 7 lines each |
| `TestReleaseTownRoomTraceIsUnchanged` | `pkg/game/testdata/townroomtrace/*.txt` | `save=` digest on 41 lines each |
| `TestReleaseTownSquareTraceIsUnchanged` | `pkg/game/testdata/townsquaretrace/*.txt` | `save=` digest on 16 lines each |
| `TestReleaseOriginalControlResaveUnchanged` | `AGAINROM_ORIGINAL_CONTROL_RESAVE_SHA256` (seat environment) | `7d1fde99…` becomes `2bbf51ed…` |

In the rewritten trace and witness files every changed token is a `save=` digest. Built with
`currentWriterCarriers` set to 0, all four tests pass against the old pins on EN and RU. The control
resave written by this build, re-encoded after its `WriterCarriers` value is set back to 0, is
byte-identical to the resave the build with the value at 0 writes (1241570 and 1241566 bytes).
The second game's profile witnesses (`rom2-en.txt`, `rom2-ru.txt`, read by
`TestReleaseSecondGameProfileWitnessIsUnchanged`) do not move.

## Open debt

- Bits 0..9 stay clear (DIV-2885). Most share two causes: a mission-start World's actors carry no
  source archive binding, and LOAD of an Againrom-written SAV drops or rebinds the carriers the
  supplement marks absent. Closing them moves World or SAV bytes and needs its own story.
