# Map editor trigger editor and trigger areas

## Intent and authority

Owner direction: a trigger's popup reads as the rule it is, each line edits
with the control its meaning needs, and the places triggers watch show on the
map. Standing owner rulings: the editor is Againrom's own, the original editor
is evidence only of what it writes, and one builder serves each kind. ROM1
authority is the pinned knowledge (k320): the script payload and parameter
types (ALM-TRIG-046), binding (TRIG-PARAM-030, TRIG-BIND-010), comparisons
(TRIG-CMP-006), distance and the box check (TRIG-DIST-014,
TRIG-CHECKGEOM-130), the original editor's value and delete behaviour
(TRIG-EDITVAL-119, TRIG-EDITDEL-120), and one claim per named operation, cited
beside each row of `pkg/mapload/scriptsignature.go`. The editor writes through
the existing host field and record calls; no divergence row is written. One
slot order differs from the shipped maps: a Follow group order (operation 15)
is written with the unit, distance and group in parameters 1, 2 and 9, where
all 10 shipped Follow nodes per install hold group, unit and distance in
parameters 1, 2 and 3. Both orders bind the same (`TRIG-PARAM-030`). Base engine main `804b4114`,
pin unchanged.

## As built

Trigger popup. Selecting a trigger (list row, area, area label) opens a popup
titled "Trigger N" with: Name; Fire once as a switch; "Conditions: all must
hold" with each pair as left condition, comparison in words (equals, does not
equal, is greater than, is less than, is at least, is at most; a code above 5
reads "never holds (code N)"), right condition, and under each condition its
parameters by meaning; "Actions: run in order" with each action and its
parameters. A condition or action line is a drop-down of the map's nodes of
that family plus "New condition" or "New action"; Edit opens the node, Remove
drops the pair or action. A removed pair moves the later pairs up, because a
first pair with left id 0 drops the trigger (TRIG-BIND-010); a removed action
empties its slot. Add condition pair and Add action show while a slot is free.
Apply writes every changed field through `InspectionFieldSet.SetFields` as one
Undo step; a new node chosen in a slot is added by the same call
(`InspectionFieldValue.New`). Cancel reloads. A pair with one side empty or an
action slot left unchosen is refused with its reason. New adds a trigger
through `AddScript("Trigger")` and opens it; Delete removes it through the
record call. Edit and New refuse while the trigger has unapplied changes, and
Back while the node has, with "Apply or cancel the changes first."

Node popup. Label; Operation as a drop-down of the operations the decoder
knows; Parameters, one row per parameter by its role: unit, group, player and
structure as drop-downs of the map's values with names, member counts and
cells; a cell as one "x,y" field with an "On map" button that arms a cell
pick; numbers and constants as number fields; a group command's command as a
drop-down. Choosing another operation or command retypes the slots to its
signature where a claim states one, keeping a slot's text while its type
stays. Back returns to the trigger. During the cell pick the status strip
reads "Click a cell on the map for the parameter; Esc cancels."

Names. `pkg/mapload/scriptsignature.go` holds the signature and role of each
named operation with its claim. Check 20 and actions 1, 7, 14, 15, 31 and
0x10003 have no claim naming what they do; they read "Condition N" or
"Action N" with their parameters by type. The UI reads the table through
`InspectionScriptModel.Catalog`, which `pkg/game` fills; `pkg/ui` does not
import `pkg/mapload`.

Areas. `mapload.ScriptAreas` returns, per trigger, the places its nodes name:
check 2 is an inclusive box; check 3 is a Chebyshev square of its own radius;
check 6 is a square around the unit's one placement; checks 7, 15 and 16 are a
square whose radius comes from the constant on the other side of the pair
(`<` and `>=` use the constant less one, the other operators the constant,
mirrored when the constant stands left), else their cell; every other X/Y
pair is a cell. Rectangles are clipped to the map. A place not drawn carries
its reason: no place, unit not placed, outside the map, reversed box, point
only, unresolved id, radius wider than 255. The map outlines every area in its
trigger's colour (eight colours by trigger index) and a square's centre with a
cross; the highlighted trigger is gold and doubled. One label per trigger,
"T<number> <name>", stands at its first area's corner, moves down past a label
already placed and left of an open popup, and is left out where a popup still
covers it. Selecting a trigger centres the map on its areas. A click on a label
or inside an area (the smallest area under the cell) selects its trigger.
View > Trigger areas shows or hides them.

Builders. The trigger and node popups are the object popup
(`pkg/ui/editorpopup.go` layout and paint, `pkg/ui/widgetdropdown.go`
drop-downs, the object edit state, typing and Apply). The popup builder gains
a switch row and several buttons per row; no new paint path. Area lines are
the editor's marker lines; labels are drawn on the chrome picture.

Words. 135 new `editor.*` words in English and Russian through `pkg/words`.
Titles and headings use `font1`, rows and values `font4`.

## Proof

- `pkg/mapload`: one test per area shape on a synthetic map (cell; inclusive
  box and reversed box; Chebyshev square; square around a unit with its one
  placement) plus the notes of what is not drawn, and
  `TestScriptSignaturesAreWellFormed`.
- `pkg/game` synthetic: `TestTriggerPopupReadsAsTheRule`,
  `TestTriggerPopupApplyWritesTheFieldCallBytes` (comparison, Fire once,
  remove action, add pair, remove pair, new action node, new and delete
  trigger, each with one Undo and Redo),
  `TestTriggerPopupRefusesAHalfPairWithItsReason`,
  `TestNodePopupEditsByMeaningAndWritesTheFieldCallBytes` (unit drop-down,
  cell pick, number, operation change),
  `TestTriggerAreasShowSelectAndHide`.
- Release, EN and RU: `TestReleaseMapEditorTriggerPopupWritesTheFieldCallBytes`
  on `scenario/10.alm`, `100.alm` and `101.alm`: a condition parameter, an
  action parameter, a comparison, Fire once, add and remove a pair, remove and
  add an action, create and delete a trigger. Each Apply equals the bytes of
  the host's `SetFields`, `AddScript` or `Delete` on a second editor; one Undo
  restores the bytes before, Redo the bytes after.
- Release, EN and RU: `TestReleaseMapEditorTriggerAreaCensus` over every
  installed map. EN: 32 maps with triggers, 421 triggers, 267 drawing an area,
  394 areas. RU: 31, 418, 266, 393. Every area lies inside its map; the editor
  model carries the decoder's areas.
- Release, EN and RU: `TestReleaseMapEditorTriggerFrames` drives the trigger
  list, a trigger popup, an action's operation drop-down, a cell pick and the
  areas with one trigger highlighted, requires every installed-font glyph kept
  outside an open list and no operation name lost to the font encoding.

## Open debt

- Check 20 and actions 1, 7, 14, 15, 31 and 0x10003 keep numbered names and
  type-named parameters until a claim names them.
- A parameter that names an item reads as a number field.
- Places not drawn: a unit with no single placement (hero ordinal, static
  name), a distance check without a constant on the other side, and a radius
  over 255.
- The help lines and key hints stay English in the Russian editor.
- The Follow signature row writes its slots in another order than the
  shipped nodes; a host refusal after an operation change (a reference left
  at 0) reads in English in the Russian editor.
