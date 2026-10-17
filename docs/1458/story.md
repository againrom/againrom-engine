# 1458 — One Add window in the map editor

## Result

The map editor adds every new record through one Add window. A unit,
structure, sack, group, player, trigger, action or condition is a data row of
`addKinds` in `pkg/ui/mapeditoradd.go`; one popup asks that row's steps top to
bottom and is built and painted by the editor's one popup builder
(`popupSpec`, `layoutPopup`, `drawPopup`). Every records list shows an Add
button; U, J, S, the A key of each list and the trigger popup's New button
open the same window. Finishing adds the record through the existing host
calls as one Undo step and opens the new record's popup. Cancel, Esc or the
close button at any step writes nothing. For the same inputs the ALM bytes
equal those of the routes the window replaced, on installed EN and RU maps.

## Intent

Owner direction (backlog): adding a record is one clear flow, not a key, a
canvas click and a typed number. Every list tab gets an Add button that opens
one top-to-bottom window built once from per-kind step data, ending in the new
record's popup. Owner rulings: the editor is Againrom's own, and the original
editor is evidence only of what it writes; one builder per kind.

## Authority

Owner direction only. No ROM1 claim is involved: the window changes how the
editor asks for values, not what it writes. The bytes each kind writes are
those of the existing host calls (`AddUnitTier`, `AddStructure`, `AddSack`,
`AddPlayer`, `AddScript`, `SetFields`), unchanged. No divergence row is added.

## As built

### Step data

| kind | list (filter) | steps |
|---|---|---|
| Unit | Units (1) | class (list, highlighted class's picture beside it), tier (list; asked only when the host offers two or more tiers), owner (players by name), cell |
| Structure | Structures (2) | kind (list with picture), owner, cell |
| Sack | Sacks (3) | cell, items (one or more) |
| Group | Groups | owner, group number, members (one or more of the owner's units) |
| Player | Players | name, colour slot, control (human or computer) |
| Trigger | Triggers (7) | name, once |
| Script | Script records | record kind (action, condition, trigger), then that kind's steps |
| Action, Condition | none | none |

A step is `addStep{Key, Word, Control, Optional, Kinds}`; the controls are
choice, choice with picture, number, name, cell, many and kind. A cell step
is a `popupPick` row: its button hands the next map click to the window
(`pickAddCell`), Esc returns to the window, and `x,y` may be typed. Up, Down
and the wheel walk a step's list; a typed value moves the highlight to its
row. Back returns to an earlier step with its value kept; a changed value
clears the steps after it. While the window is open the selected record's
popup is hidden; it returns on Cancel.

Picture column: `popupSpec.ListAside` keeps a column right of the list, and
the window draws the highlighted choice's picture there: a unit class's
standing sprite, grown by a whole factor, or a structure type's information
picture.

Two painter fixes the window's frames exposed: a popup switch button clips a
long value with an ellipsis (the Russian trigger "once" value overflowed its
button), and the group popup clips its member lines to the popup.

### Host

`ui.InspectionNewRecords`, implemented by `mapInspectionEdits` in
`pkg/game/mapinspectionadd.go`:

- `NewRecordStep` offers a step's choices and default from the host's
  existing catalogues (`RecordChoices`, `UnitTiers`); a field's default is the
  value the kind's bare add call writes.
- `CheckNewRecord` refuses a value with its reason (class, tier, owner slot,
  structure kind, item code, a group number any unit of the map already
  carries, a member
  of another player, a cell outside the map) and, once the record is complete,
  runs the whole add on a copy of the model.
- `AddRecord` runs the kind's existing host call and then `SetFields` for
  every other value on one model: `mapInspectionEdits.batch` makes the nested
  host calls apply to the same clone, and `mapedit.Editor.Squash` folds them
  into one history entry.
- `NewRecordPreview` is a unit class's standing sprite or a structure type's
  information picture.

A group has no record of its own in the ALM: its add sets each member's
`group` field, as the unit popup does. A map script names a group by its
number alone (AI-366), so the group step offers the lowest number no unit of
the map carries, and refuses a number a unit of another player carries. A
number the chosen player already carries is refused as before.

### Removed

The per-kind prompts `promptNewSack`, `promptNewUnit`, `promptNewUnitTier`,
`promptNewStructure`, `promptNewScript`; the picks `pickSack`, `pickUnit`,
`pickStructure`; `keyPlayer`, `addScriptKey`, `askUnitTier`,
`newRecordField`; the host interfaces `InspectionUnitTiers`,
`InspectionPlayers`, `InspectionScriptAuthor` and `AddSack`, `AddStructure`
on the UI interfaces; `UnitTierLabel` and the word `editor.unit.tier.new`.

### Words

New `editor.button.*` (add, back, next, finish, remove) and `editor.add.*`
(titles, step labels, control choices and refusal notes) in
`pkg/words/text/en` and `ru`. `TestEveryUsedIDIsInTheEnglishTable` also reads
a literal id held in a data-row field whose name ends in `Word`.

## Proof

- Bytes: `TestReleaseMapEditorAddWindowWritesTheRouteBytes`
  (`pkg/game/mapeditadd_release_test.go`) adds each of the eight kinds through
  the window on the first installed map with two players whose player 1 owns
  two units, and compares with the host calls the old routes made for the
  same values. One Undo restores the source bytes; Esc at each step leaves the
  bytes unchanged. Loss control: a sack window given one of its two items does
  not match the route. The window's SHA-256 values equal those the base
  `9ac65f00` UI routes wrote for the same inputs, recorded on EN and RU before
  the change, except the group case, whose number is now the lowest one no
  unit of the map carries.
- Group number: `TestReleaseMapEditorAddWindowGroupNumberIsMapWide`
  (`pkg/game/mapeditgroupid_release_test.go`) adds a group of player 3 on
  `scenario/10.alm`, where the script names player 2's group 2. The number
  offered is one no unit carries, typing 2 is refused with its reason, and
  after Finish group 2 has the same carriers and the script names it as many
  times as before. Loss control: with the owner-only default and check the
  test fails at the refusal.
- One builder: `TestMapEditorAddKindsAreRowsOfOneBuilder` checks every row's
  words in both tables, every step control, every list's Add button and that
  no retired route name stands in `pkg/ui`, with a loss control on a planted
  file. `TestMapEditorEveryAddRouteOpensTheAddWindow`,
  `TestMapEditorAddCancelAtEveryStepWritesNothing`,
  `TestMapEditorAddCellPickedOnTheMap`, `TestMapEditorAddTierBackAndRefusal`
  and `TestMapEditorAddManyGroupAndScriptKinds` cover the routes, cancel, the
  map pick, the tier step, Back, refusals and the many steps.
- Frames: `TestReleaseMapEditorAddWindowFrames` writes each list, each kind at
  each step and the popup it ends in; every glyph of the installed fonts is
  kept.

## Open debt

- The class and structure lists show the highlighted choice's picture beside
  the list, not a picture on every row; the shared list box draws text rows.
- A group has no name in the ALM; the window asks its owner, number and
  members.
- The trigger popup's "New" opens the Add window for a trigger; script nodes
  of a trigger's slots are still created from the trigger popup.
