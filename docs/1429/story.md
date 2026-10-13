# The map editor gives a map its win and its loss

## Intent

In the first game's map editor the author gives a map that has none a win arm
and a lose arm. The mission is won when a chosen unit dies, a chosen group is
wiped out or a chosen structure is destroyed, and lost when a chosen unit of
the party's side dies. A new map from story 1427, with a drop cell, a party and
one enemy, saved into a mod, is won by killing the enemy and lost when the
chosen unit dies. The story started at public main `20c015a5` (release 0.121.0,
pin k287).

## Authority

Owner direction (map editor, priority; the open debt of story 1427). Claims read
from the pin: `MISSION-WIN-003` (instant 4 is the only win), `TRIG-ACT-004`
(instant 5 loses), `TRIG-COND-003` (the check vocabulary), `TRIG-CMP-006` (the
comparison codes; the three pairs are ANDed), `TRIG-FIRE-007` (the once flag),
`TRIG-BIND-010` and `TRIG-BINDORDER-101` (ids, order, the 5000 threshold),
`TRIG-COUNT-015` and `TRIG-EMPTY-018` (check 1 authored as "is this group wiped
out"), `TRIG-CHECK-053` (check 21 reads a structure's +0x42 word),
`MISSION-VIP-004`, `MISSION-SLOT-008` (100 registers, 1000 triggers),
`ALM-TRIG-044` to `ALM-TRIG-047`, `ALM-NODEREAD-232`, `ALM-TRIGREAD-233`. Rows:
`DIV-3227` (the records, ids and limits), `DIV-3228` (unit and group arms),
`DIV-3229` (the structure arm), `DIV-3230` (the lose arm's party side).
`DIV-3231` to `DIV-3234` are returned unused.

Choices the claims leave open:

- The check and comparison of each arm. Over the 38 EN maps (instrument: a
  throwaway test over `alm.Script` of every installed map), the triggers that
  run instant 4 or 5 on a unit read check 5 `== 0` against a constant node of
  value 0, and on a group check 1 `== 0` against the same; no shipped win
  trigger names a structure. The unit and group arms write those shapes. The
  structure arm is check 21 `<= 0` (`DIV-3229`).
- The constant node's type code. 21 shipped constants of value 0 carry the int
  type and 11 the constant type; the arm writes the int type.
- The trigger's unread bytes: the 64 opaque bytes are zero; once is 1 (28 of 28
  campaign win triggers); the second and third pair, the other three action
  slots and their comparison codes are zero.
- The party's side is roster slot 1 (`DIV-3230`).

## As built

### One table, one batch

`pkg/mapedit` adds `KindTrigger` as the third row of the section table: the
type-7 payload's third array, 184-byte records, no settable field. `PlaceAll`
appends records of several arrays of one counted payload as one edit. Its
parts are the records, each array's count word and the payload size word; where
one array's new records end at the next array's count word, the two are one
splice (records, then the word). One Undo restores the exact earlier bytes.
`Place` stays the one-record case. `AddArm` (`arm.go`) builds an arm from a row
of the arm table and calls `PlaceAll`; no second script writer exists.

| arm | records appended, in file order |
|---|---|
| win, unit | instant node opcode 4; check node opcode 5, slot 0 Target_Unit; constant node `0x10002` slot 0 int 0; trigger |
| win, group | the same, check opcode 1 with Target_Group |
| win, structure | the same, check opcode 21 with Target_Structure, trigger comparison code 5 |
| lose, unit | instant node opcode 5; check opcode 5 with Target_Unit; constant; trigger |

Each node is 796 bytes with the label, opcode, id and the one slot set and every
other byte zero; the trigger is 184 bytes: name at +0x00, the check id and the
constant id at +0x80 and +0x84, the instant id at +0x98, the comparison code at
+0xa8 (0 for `==`, 5 for `<=`) and 1 at +0xb4. The three counts move by 1, 2
and 1 and the type-7 payload size grows by `796 + 2*796 + 184 = 2572` bytes. Ids
are one above the highest of the array. An arm is refused, with nothing
written, when it would pass 100 conditions or 1000 triggers, when an id would
pass 5000, or (host) when an authored variable names either register.

### The host and the window

`pkg/game/mapinspectionscript.go`: `scriptGoalRows` appends, before the protect
row, the three win rows when the panel lists no win row, and the lose row when
it lists no lose row. `SetGoalTarget` on such a row picks the value as it does
for a retarget (a unit's id; a unit's group id; a structure's +0x12 word),
checks the lose arm's unit belongs to roster slot 1, checks a structure
arm's structure starts with health, checks the register collision for both new
checks and calls `AddArm`.

Panel rows (the script panel opened by G; no new mode or key):

| rule | text | pick |
|---|---|---|
| Win | add: the death of a unit wins | Unit |
| Win | add: a group wiped out wins | Unit (its group) |
| Win | add: a structure destroyed wins | Structure |
| Lose | add: the death of a party unit loses | Unit |

Enter arms the pick and a click names the record, as for a retarget. After an
add, the new arm is listed as an ordinary row with its target and its add row is
gone. The help lines are unchanged (`ui.EditorHelpLines`; the pick line already
says unit or structure).

### The report

No new checker. A map with no win arm keeps the findings the rule table already
names; the new arms use ids that resolve, so `script-reference` stays silent.

## Proof

- `pkg/mapedit` (`arm_test.go`): each arm's bytes equal an independently
  written map (`synth.ALM` over hand-laid records); `alm.Open` accepts the
  result and reads two conditions, one action and one trigger more; one Undo
  and Redo give the exact bytes; an arm on a new map's empty script and two arms
  in a row; refusals (unit 0, over 65535, a full register file, a payload that
  is not one array) write nothing.
- `pkg/game` (`mapeditarms_test.go`): the rows on an armless map and none on a
  map with both arms; each arm's records and Undo to the exact bytes; refusals
  (the wrong record kind, a unit of the enemy side for a lose arm, a variable on
  the register); the window keys end to end.
- Release, EN and RU (`mapeditarms_release_test.go`):
  `TestReleaseMapEditorNewMapIsWonAndLostByItsArms` builds `new/32x32` with a
  drop cell, an enemy (owner 2) and a unit of owner slot 1, adds the win arm on
  the enemy and the lose arm on the second unit through the script host, and
  saves into a temporary mod through the window's Save. The mod's mission 20
  starts headless: 200 frames end undecided, killing the enemy wins, in a second
  run killing the other unit loses, and the same map saved without the arms,
  with both deaths, stays undecided (the loss control).
  `TestReleaseMapEditorGroupAndStructureArmsOnAnInstalledMapCopy` takes a loose
  installed map (no win arm), adds a group arm and, separately, a structure arm,
  and plays each: another group wiped out or another structure felled does not
  win, the named one does. Structure health is set with
  `ImportOriginalStructureHealth`, so the test does not exercise combat damage.
  `TestReleaseEveryInstalledMapListsItsScriptArmsAndWritesItsBytes` (all
  installed maps) now also asserts no map with a win row offers to add one; an
  unedited save still equals the installed bytes.

## Open debt

- A map with a win arm gets no add-win rows (nor one with a lose arm a lose
  row); a second win cannot be added from the panel.
- An arm is one pair. Several conditions per trigger (`AND` of pairs), a
  comparison other than the arm's own, or a threshold other than 0 are not
  offered; the author edits the node fields (E).
- The win arm on a structure rests on this build's reading of the +0x42 word as
  health (`DIV-3229`); the original may differ.
- The lose arm takes only roster slot 1; a lose by a group or a unit of another
  slot is authored by hand.
- Whether the original's builder accepts the records as written is not
  witnessed; the proof runs this build.
