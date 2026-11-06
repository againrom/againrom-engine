# The second game's quest objectives

## Intent

Run the second game's sub-objectives as the original's quest machine does:
one bank slot per objective written by instant 36 alone, cleared where the
original clears it, carried by SAV, announced by the original's sounds and
shown in the Quest Objectives window as one row per opened objective with the
original's state mark. Maps 21 and 31 are the witnesses. The first game's
objectives window is unchanged.

## Authority

Knowledge pin k426. The quest machine: `R2-ENGINE-045` (NewGame bank),
`R2-ENGINE-046` (instant 36 modes, events 253 and 254; its frame and banner
clauses are retracted), `R2-ENGINE-047` and `R2-ENGINE-050` (UI 0x433 events),
`R2-ENGINE-896` (slot 752 is written only by the EnterLocation clear). The
window: `R2-ENGINE-888` (rows and geometry), `R2-ENGINE-889` (the subobj.256
frames), `R2-ENGINE-890` (the text cache), `R2-ENGINE-892` (the base window).
The events: `R2-ENGINE-891`. The maps: `R2-ENGINE-894` (map 21) and
`R2-ENGINE-895` (map 31). SAV: `R2-SESSION-031` (the 4096 raw bank bytes) and
`R2-SESSION-348` (the original corpus's slots; LOAD keeps 752..767).

Where these claims say Medium or Unknown the implementation takes the claim's
own reading and the difference is a divergence row: DIV-4556 (the window around
the rows), DIV-4557 (row text styles), DIV-4558 (the text cache and the row
wrap) and DIV-4559 (event sounds and the new-mission line). DIV-2379 is amended:
its goal-banner part is gone and its FIFO delivery of ordinary messages stands.
DIV-2380 (text marks) is closed and moved to `docs/DIVERGENCES-CLOSED.md`.

## As-built behaviour

The bank is `sim.ScenarioBank`, 1024 signed slots; objective n is slot 752 + n.
`pkg/sim/scenariobank.go` is its one writer: `NewGameScenarioBank` clears every
slot and sets 768 to 10, `EnterLocation` clears 752..767, `SetScenario` is
instant 35 and `WriteObjective` is instant 36. The script runner and every store
the campaign makes into its bank (see Seat correction) call it; nothing else writes a slot.

| Instant 36 mode | Store | Event |
|---|---|---|
| 1 | 1 when the slot is 0, else unchanged | none |
| 2 | unchanged when the slot is 4, else 3 | 253 when bit 2 was clear and the slot was not 4 |
| 4 | 5 | 254, every time |
| any other | the mode as given | none |

The bank clears at a new game (`NewGameScenarioBank`), at a fresh mission entry
and at a town entry from the world map (`EnterLocation`). LOAD restores the
saved bank whole and clears nothing.

A mission SAVE writes the live World's bank into the second-game tail's 4096
bank bytes, where the original's file holds it. Before this the tail carried
the campaign's bank of the departure, so a records-only LOAD lost every
objective opened in the mission. Both LOADs, the engine document and the
records alone, restore the slots.

Events 253 and 254 write nothing and post nothing. 253 plays UI sound slot 14
(mission complete) and 254 slot 16 (mission failed), each only when that slot's
previous voice has stopped. 250 plays slot 14 the same way and posts patch.txt
row 0x61 on the map message line for 30 seconds: EN "You've got a new mission!",
RU "Вы получили дополнительную миссию!". Ordinary messages keep their FIFO.

The window keeps the owner panel with the mission briefing at its top. Below
it stands one row per cached text whose slot is nonzero, in cache order. A row
is a 24x24 subobj.256 mark at x 48..72 of the panel, from y - 3, beside its text
from x 80, wrapped at the panel width less 128; the first row is 16 below the
briefing and rows are 10 apart. Bit 2 draws frame 1, else bit 4 frame 2, else
bit 1 frame 0; a nonzero slot with none of them has no mark. The panel grows by
its rows about its centre. Texts are `#subobjective<n>` from 0 up to the first
missing section; row 0 is kept. The texts are the installed ones, CP866 decoded
for RU. Opening the window writes no slot.

Maps 21 and 31, as their pages wire them:

| Map | Trigger | Instant 36 | Shown after |
|---|---|---|---|
| 21 | T0, the first pass | (1, 1): objective 1 opens | objective 1, frame 0 |
| 21 | T1, the hero within 4 of unit 46 | (2, 1): objective 2 opens | objectives 1 and 2, frame 0 |
| 21 | T4, after T3 raised register 70 | (2, 2): objective 2 completes, 253 | objective 2, frame 1 |
| 21 | T9, the sack gone | (1, 2): objective 1 completes, 253 | both, frame 1 |
| 31 | T2, near unit 101 | (1, 1): objective 1 opens | objective 1, frame 0 |
| 31 | T3, near unit 102 | (2, 1): objective 2 opens | objectives 1 and 2, frame 0 |
| 31 | T13, after T6 near unit 100 | (2, 2): objective 2 completes, 253 | objective 2, frame 1 |
| 31 | T5, near unit 101 | (1, 2): objective 1 completes, 250 and 253; T16 wins | both, frame 1 |

Kinds and their one builder: the bank effects and slot writes are
`pkg/sim/scenariobank.go`; the bank's SAV record is `secondCampaignTail` in
`pkg/game/savform.go`; the screen is the existing `questLayout` and
`questPicture` in `pkg/ui/questobjectives.go`; a row and its mark are
`mapWorld.missionObjectives` with `objectiveMark` in
`pkg/game/questobjectives.go` and `questRowLayout` in the UI; an event's sound is
`playUISound` through `Viewer.PlayUISoundIdle`; the line is `InstallWords.Words`;
the mark art is the menu-panel art row in `pkg/game/screens/rom2.json`. The
first game reads no sub-objectives, so its window is built unchanged by the same
builder.

## Proof

Focused tests, all fixture-only:

- `TestWriteObjectiveFollowsTheStateTable`: every mode from states 0, 1, 3, 4,
  5 and 6, other modes, and indices outside the bank.
- `TestScenarioBankClearingPoints` and `TestSecondCampaignBankClearingPoints`:
  the NewGame bank, the location clear, and a town entry through the world map
  that clears 752..767 and keeps slots 768 and 900.
- `TestSecondMissionSaveCarriesTheLiveObjectiveSlots`: the fresh mission entry
  clears stale slots; the SAVE carries the live slots; both LOADs restore them;
  a later change reaches the next SAVE while the first file keeps its own value
  (the loss control); a slot patched in the file's bytes reaches the World. It
  failed before the tail fix with the departure bank's slots.
- `TestSecondGameObjectiveEventsPlayTheirEntryWhenIdle` and
  `TestInstallWordsTakeTheNewMissionRow`: one voice per slot while it plays, no
  caption, the 30-second line.
- `TestQuestObjectiveRowsStandBelowTheBriefing`, `TestQuestObjectiveRowsDrawTheirMarks`,
  `TestQuestObjectiveRowsKeepRussianText` and `TestSecondGameObjectiveRowsPerState`
  (EN and CP866 RU): rows, geometry and the mark per state.
- `TestSecondMapTwentyOneObjectivesOpenAndCompleteAsThePageSays` and
  `TestSecondMapThirtyOneObjectivesOpenAndCompleteAsThePageSays`: the maps'
  pages as scripts, the events, slots, rows and marks at each step of the table
  above. Conditions the pages leave Medium are probe slots the test sets.

Release witnesses, compiled and not run here, for EN and RU of the second game:
`TestReleaseSecondMapObjectiveWiring` (the installed maps run instant 36 where
the pages say, and map 21 shows objective 1 with frame 0 after its first passes)
and `TestReleaseSecondOriginalMissionFileKeepsTheObjectiveSlots` (original
game0009 and game0010 of mission 20 through LOAD, the window and our SAVE).

## Open debt

The window's surroundings, the text styles, the row wrap, the row 0 cache and
the sounds behind entries 0xe and 0x10 rest on Medium or Unknown readings; each
is a row above with the run that settles it. The deferred replay order of
ordinary messages stays in DIV-2379. The maps' completions on the installs are
witnessed only through the page wiring: no headless run drives the hero to the
units the triggers test.

## Seat correction

Seat correction: the first build left 17 stores into the bank in `pkg/game`
beside the writer. Every one now goes through `sim.SetScenario`, with the same
slots, values and order; a compound store (`768 += 10`) reads the slot and then
calls `SetScenario`. The callers as built are `completeBank` in
`secondcampaign.go`, `secondDepartureRow.run` and `secondDepartureReset` in
`seconddeparture.go`, `talkTo` in `secondinn.go`, `secondEngineReach`, the
reach walk and `secondEngineExit` in `secondcensus.go`, and the generator's
character-choice slots in `secondgenerator.go`. `pkg/sim/scenariobank.go` gains
nothing. `internal/archtest.CheckScenarioBankWriter` refuses any later store
into a bank slot in a non-test `pkg` file outside that writer.
