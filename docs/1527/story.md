# The second game's signpost opens its message box

## Intent

In the second game's first mission (`10.alm`), a click on the road signpost
near the start walks the hero to it and opens the event box with the line
"To Great Mage Skrakan's Tower" and Ok, as the EN original does. The box
holds the mission until Ok, Enter or Esc. Every other second-game signpost
with an action works through the same data. The first game is unchanged.

## Authority

Owner rulings: one builder per kind; a game difference is profile data; the
signpost answers the hover and the click although its kind record's +0x8c is
unread (DIV-4178); the card and the portrait are built where the existing
builders take a structure as data, otherwise a row states what is missing
(DIV-4180). Story brief: the signpost runs the instruction the compiler made
from its own action node, and a node the compiler omitted runs nothing
(DIV-4179).

The pinned claim tool (knowledge k371) read these claims:

- R2-ENGINE-605: the hover names the object of the last-registered rectangle
  meeting a 3x3 box at the cursor; a structure gets flag 0x800 only if its
  kind record's +0x8c is set.
- R2-ENGINE-606: a click on a hovered object with flag 0x800 sends 0x24.
- R2-ENGINE-607: 0x24 sets order 0xf, a walk to within range 1, 2 or 3 by
  building byte +0x60, then the state-15 handler.
- R2-ENGINE-608: the event box is 580x240 at (30,120), rounded to 488x232
  and centred, (268,268) at 1024x768; Enter and Esc answer as Ok.
- R2-ENGINE-609: a modal box sets a UI bit, darkens the screen, sets a
  cursor, stops the edge scroll and defers events.
- R2-SESSION-230 (High): the signpost shows no hover caption; a click walks
  the hero to it and opens a box with one line and Ok; the screen darkens and
  the frame is digest-equal 10 s later; Ok closes it.
- R2-ASSET-152: the signpost is `10.alm` Pointer id 42 and selects event 19;
  `#event19` occurs once in each `mission10.txt`; the RU line is 23 bytes.
- R2-ENGINE-512, R2-ENGINE-517, R2-ENGINE-518, R2-ENGINE-520: the type-11c
  record gives a Pointer (kinds 0x46..0x4b) +0x6c, the count of action nodes
  below 0x10002 before its node; an actor step in state 15 calls the action
  executor with +0x6c above 0; a compiled slot advances only on a successful
  compile (Medium).
- R2-ENGINE-047, R2-ENGINE-050: instant 2 raises the mission text's
  `event%d` section in the event box.
- R2-ASSET-123, R2-ASSET-124: 13 Pointers in the 46 campaign maps of each
  root, 12 with a nonzero-gate 11c record; Buildings 0x46..0x4b are
  "Pointer 1".."Pointer 6" with health maximum 0.

## As built

The first game's structure use is the route: hover, the use cursor, the click,
`sim.UseStructure`, the approach in `pkg/sim/structureuse.go` and the action on
arrival. A second-game Pointer enters it as data.

- `pkg/formats/alm/rom2.go`: `(*Map).StructureActions` decodes the 12-byte
  11c records (id, gate, position) after the 11a and 11b records of
  `Extension[1]`.
- `pkg/mapload/structureuse.go`: `PointerSlots` gives each Pointer object
  whose id a nonzero-gate record names its source action slot, the count of
  earlier action nodes that `runtimeAction` admits; the last record wins.
  `Structures` stores it as the structure's `UseAmount`.
- `pkg/mapload/script.go`: `runtimeAction` (operation below 0x10002,
  unsigned) is the one predicate the compiler and `PointerSlots` use. The
  compiler sets `ScriptInstant.Slot`, the source slot of the node each
  second-game instant came from; omitted nodes count.
- `pkg/sim/structureuse.go`: a Pointer with a slot above 0 is `Usable`; its
  approach is `structureUseGoal`, whose ranges are R2-ENGINE-607's (width 1
  gives range 1). On arrival `runStructureAction` runs the instant whose
  `Slot` equals the Pointer's slot through `runInstantObserved`, once; no
  such instant runs nothing.
- The event: instant 2 reaches `Report.ScriptMessages`, and the existing
  second-game notice route (`observeSecondGameMessages`, `openDialogue`,
  `ReadEventTextFor`) opens the mission's event window with `#event19` of
  `mission10.txt`.
- `pkg/game/structureuse.go`: `restorePointerSlots` gives a loaded Pointer
  the slot its map placement derives (matched by ID, kind and cell) when the
  loaded form carried none.

### Kinds and builders

| Kind | Builder used |
|---|---|
| Hover pick and click on a map object | `Viewer.inspectionAt` and the usable-structure click (`usableStructure`, the structure-use sink) |
| Order that walks to a target and acts | `sim.UseStructure` (`KindUseStructure`), `structureUseGoal`, `applyStructureUse` |
| Script action run | `runInstantObserved` |
| Which node becomes which instruction | the compiler, `compileScriptFrom` with `runtimeAction` |
| 11c record decode | `pkg/formats/alm/rom2.go` beside the 11a and 11b decoders |
| Event text | `ReadEventTextFor` / `readSecondGameEventText` |
| Event box | the mission dialogue notice, `Viewer.SetDialogue` with `AuthoredDialogueLayout` and the second game's `NoticeOnScreen` |
| Hold and darkening under the box | `flow.syncCadence` (an open popup holds the mission) and the dialogue backdrop plan |
| Card and portrait | `Viewer.inspectionPanel`, `structureCardName`, the structure class portrait |

No new command kind, box builder, pick routine, script runner or text loader.

### Hashed state, SAV and old saves

- The Pointer's slot is the structure's `UseAmount`. The World byte form
  carries it in the existing structure-use footer (kind and amount per
  structure), and the pending use in the existing pending-use footer.
- `ScriptInstant.Slot` enters the World byte form as an outermost section,
  version 129, written only when an instant has a slot other than 0. The SAV
  current-actions program carries it as a JSON field omitted at 0; LOAD
  refuses a negative slot, as the byte form does. The first game's instants
  keep 0, so its byte form and SAV bytes are unchanged.
- SAV does not carry the Pointer's slot. A SAV LOAD derives it in
  `restorePointerSlots` from the map placement (ID, kind and cell) as soon as
  `applyOriginalStructuresCurrent` has built the loaded structures, before
  `restoreOriginalActions` restores and checks a pending use. So a SAVE taken
  while the hero walks to a signpost (F2, F4 or the timed autosave) loads, and
  the continued run equals the uninterrupted one: the hero arrives and the box
  opens on the same step. Mission open runs `restorePointerSlots` again for
  the World byte form; it fills only a slot still empty.
- A World byte form or SAV written before this story carries no source slot.
  Its Pointer takes its slot from the map through `restorePointerSlots` and
  stays usable: the hero walks to it and nothing opens. No signpost was usable
  before this story.
- The open box is viewer state. It enters neither the World hash nor SAV.

### The first game

Its claims state the same route for its own structures: UNIT-STRUCTHIT-138
(a usable structure is a hover hit through its class field +0x8c, and the
hover ID becomes the target of order 0x24), AI-STRUCTUSE-306 (0x24 sets state
15 and an approach within range 1, 2 or 3 by width, then one local use
action) and UNIT-STRUCTUSE-091 (the fountain arm spends one charge). No
first-game claim names a Pointer. Only the 11c decoder writes a Pointer slot,
and first-game maps carry no 11c records, so no first-game structure changes.

## Proof

### Focused tests

| Test | States | Loss control |
|---|---|---|
| `TestStructureActionsReadThe11cRecordsAfter11aAnd11b` (`pkg/formats/alm`) | the 11c records from a synthetic extension | base has no decoder (compile failure) |
| `TestPointerSlotsCountTheNodesBeforeThePosition`, `TestPointerSlotNamesTheInstantCompiledFromItsNode` (`pkg/mapload`) | the slot counts runtime nodes; with an omitted node before the Pointer's, the slot names the instant compiled from its own node; a high-bit node counts by the compiler's unsigned test | the signed count with `instants[slot]` indexing: four tests fail, two here and two in `pkg/sim` |
| `TestPointerUseWalksThenRunsItsSlotOnce`, `TestPointerWithoutAPositiveSlotIsNotUsable`, `TestPointerSlotPastTheProgramRunsNothing`, `TestRestoreStructureUseMetadataFillsOnlyAnEmptyPointerSlot` (`pkg/sim`) | walk within range 1, byte-form reload mid-approach hash-equal, event 19 raised once, slot 0 not usable, a slot no instant carries runs nothing, the restore fills only an empty slot | base `Usable` refuses kind 0x4a ("use was not admitted") |
| `TestPointerRunsTheInstantCompiledFromItsSlot`, `TestScriptSlotsSurviveTheByteForm` (`pkg/sim`) | the instant is found by its source slot; the slot survives the byte form | the same four failures as the `pkg/mapload` row |
| `TestPointerWithASlotAnswersHoverClickAndCard` (`pkg/ui`) | the use cursor over the Pointer, the card at 1000/1000, one click sends the structure use; slot 0 answers neither | base: the cursor is not the use cursor |
| `TestPointerArrivalOpensItsEventWindowAndHoldsTheMission` (`pkg/game`, subtests ok, enter, escape) | arrival opens the event window with event 19's line; the hash holds over 48 steps open; a click off Ok keeps it; Ok at its centre, Enter and Esc close it; the mission runs on; no reopen | the Pointer arm of `Usable` disabled: all three fail with an empty notice |
| `TestPointerUseSavedOnTheWayColdLoadsAndRunsAsTheUninterruptedRun` (`pkg/game`) | a mission file with the Pointer in its Buildings LOADs; a SAVE (game menu) with the use pending cold-LOADs hash-equal with the use pending; both runs stay hash-equal step by step, the box opens on the same step with the hero beside the Pointer and event 19's line, and after Enter the hashes are equal | `b215bf3b` (slot derived only at mission open): the cold LOAD stays on the Load page with "sim: restored current actions: sim: invalid structure-use actor/target 0/0" |
| `TestCurrentScriptProgramRefusesANegativeActionSlot` (`pkg/game`) | a SAV program instant with slot -1 is refused; 0 and 1 compile | `b215bf3b`: slot -1 compiles |

### Witness over the installs

A headless diagnostic test, untracked and kept out of every commit
(`pkg/game/signpostwitness_diag_test.go`), ran mission 10 from its start on
rom2-en and rom2-ru. The signpost is structure 15, kind 0x4a, cell (14,40),
1x1, slot 24; the hero (29) starts at (11,41). EN and RU are equal except the
text and the card's health word.

| State | Original, EN capture at 1024x768 | Ours, EN and RU |
|---|---|---|
| m4.04 hover | no caption; card ROAD SIGN, HEALTH 1000/1000; road sign in the oval | use cursor (first hit of a 2-pixel scan at 206,384); no caption; card USABLE POINTER 5, HEALTH 1000/1000 (DIV-4180); the oval is not in the witness frames |
| m4.05 click | the hero walks to the signpost; the box with "To Great Mage Skrakan's Tower" and Ok; the whole screen darker | the box opens 46 steps after the click with the hero at (13,39), 1 cell from the signpost; EN body "To Great Mage Skrakan's Tower", RU the 23-byte line of R2-ASSET-152 |
| m4.05 box place | box fill about (275,283)-(743,471) (crop at half size, scale 1.98 assumed) | box 488x232 at (268,268); fill (276,283)-(739,476); probes (467,444) and (472,439) keep it open, (468,440), Ok's corner, closes it |
| m4.06 wait 10 s | digest-equal to m4.05 | World hash equal over 160 steps open |
| m5.10 Ok at (508,452) | the box closes; the mission runs | Ok is (468,440)-(548,466); Ok closes the box, the hash moves, no reopen, no pending use |

The darkening: the catalogue states "whole screen darker" for m4.05; in the
crops the map strip left of the box, (0,20)-(130,390) of the original panel,
falls in mean luma from 34.35 (m4.04) to 29.22 (m4.05 and m4.06). Ours darkens
through the dialogue backdrop plan over the whole 1024x768 frame (the witness
reports the plan showing 1 over (0,0)-(1024,768)); the witness frames are the
CPU map frame without the backdrop, so our darkened pixels are unmeasured.

### Shipped maps

`TestDiagSignpostCount` and `TestDiagSignpostShift` over the 46 campaign maps
of each root, EN equal to RU, compiled with the first mission's party:

| Map | Pointer ids | Runs |
|---|---|---|
| 10 | 42 | event 19 |
| 20 | 87, 88, 89 | events 12, 13, 14 |
| 21 | 30, 31, 32 | events 9, 7, 8 |
| 30 | 74 | event 19 (two earlier nodes omitted) |
| 40 | 168, 169, 170 | event 16 (three earlier nodes omitted) |
| 50 | 46 | event 23 |
| 76 | 28 | nothing: its one record has gate 0, so it is not usable |

12 Pointers take the use. No shipped action node has an operation at or above
0x80000000.

### First game unchanged

The `pkg/sim`, `pkg/ui` and `pkg/game` packages pass, the first game's
structure-use, hover and click tests among them. The first game's instants
keep slot 0, so the byte-form section and the JSON field are absent and its
World and SAV bytes are unchanged; frames are byte-equal by the release route
witness below, which this lane did not run.

### Release tests for the release

By owner ruling, release tests run at the release, not in this lane:

- `TestReleaseRouteWitness` on rom2-en and rom2-ru: the signpost steps change
  (the box opens), and a second-game World with a compiled script carries the
  version 129 section, so its World hash changes.
- `TestReleaseRouteWitness` on en and ru: byte-equal.
- The ROM2 save groups of `pipeline/check-rom2-save-tests.sh` on rom2-en and
  rom2-ru: a mission SAVE now writes `Slot` in the current-actions program,
  and a SAV LOAD derives the Pointer's slot before the pending use.
- `TestReleaseStructureUse1150` on en and ru: the first game's SAV LOAD now
  passes its structures through `restorePointerSlots` before the pending use;
  its mid-approach cold LOAD must stay hash-equal.

## Open debt

- DIV-4178 (UNKNOWN): the Pointer's client class, its kind record's +0x8c
  and the rectangle it registers are unread; the signpost answers by owner
  direction.
- DIV-4179 (DEVIATION): one run per arrival on the clicked Pointer, and the
  Pointer's own node or nothing where the original may run a later node.
- DIV-4180 (UNKNOWN): the card reads USABLE POINTER 5, not ROAD SIGN; the
  portrait is unobserved.
- Unknown, no row: the amount of the darkening under the box in our frame.
- DIV-4181 is returned unused.

### Research questions

1. Which runtime class does the client build for a Pointer, and what does
   its kind record hold at +0x8c and +0x64?
2. Which rectangle does a Pointer register for the hover: the four callers
   of `L2.03008`, or a run hovering the top of the signpost sprite and its
   foot cell.
3. In the original, after Ok on the signpost box with the hero standing next
   to the signpost, does the box open again? Record 30 s with no input.
4. On a map where a reference before a signpost's node fails at mission
   start, which instruction does the signpost run?
5. Which text row or table gives the card's name for a Pointer, which
   picture fills its portrait oval, and what do EN and RU show?
6. What colour and amount does the darkening under a modal box apply
   (`L2.03082`), and over which part of the screen?
