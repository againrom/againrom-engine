# Map editor trigger wizard and drop-zone tool

## Intent and authority

Owner feedback: a map maker should build a trigger from a ready list in plain words, not by wiring
condition and action nodes by hand. The owner's design is a "When / And if / Do" wizard built through
the popup builder, with the pair view behind a Details button; items whose parameters are chosen in
the row (a unit picked by a click, a player, a typed number, a cell or area drawn on the map); the
wizard makes the nodes, the pair and the Constant itself, and shows a comparison only where the item
is numeric. Two area conditions: "a unit is inside an area" (a box) and "a unit of a player is near a
point" (a square of radius N). A drop-zone tool on the Tools menu and on D shows where the hero and
the others land and writes the shipped drop shape; several zones are allowed and the game picks one.

The editor's layout and controls are owner direction and take no divergence row. The wizard writes
only what the trigger engine already reads, and the drop zones only what the mission start already
reads:

| Claim | What the editor relies on |
|---|---|
| `MISSION-DROP-002` | each drop action is a candidate; the start picks one at random; X or Y 0 falls back |
| `TRIG-DROP-013` | a drop action is the build-time operation with an X and a Y |
| `MISSION-SCATTER-053` | the hero takes the drop cell; the others land within radius `max(5, floor(sqrt n)+4)`, offsets `-r/2 + draw(0..r)` |
| `TRIG-NEAREST-029` | check 15 is the Chebyshev distance from a cell to a player's nearest unit, so `<=` N is a square of radius N |
| `TRIG-HEROORD-075` | hero ordinals 10001 and up name nobody on a map for more than one player |

No frame, World hash or SAV byte of either game changes: the wizard and the tool only write map
files, through the editor's existing record calls.

## As built

### The wizard

Selecting a trigger opens the wizard (`pkg/ui/mapeditorwizard.go`); Tools > New trigger adds one. The
popup shows Name, Fire once, then When (the first condition pair), And if (the others) and Do (the
actions), each row one item of the ready list with its parameters inline, and Add condition and Add
action rows while slots remain (3 pairs, 4 actions). Buttons: Apply, Cancel, New, Delete, Details.
Details opens the pair view of earlier stories, which gains a Wizard button back; Back from a node
opened there returns to the pair view. The trigger popups widen to hold their whole button row
(`fitButtons`).

| Item | Writes |
|---|---|
| Hero enters area | check 2 (unit, box) `==` Constant 1 |
| Any unit of a player near a point | check 15 (player, cell) `<=` Constant N, the radius |
| Unit near a point | check 3 (unit, cell, radius) `==` Constant 1 |
| Unit is dead | check 5 (unit) `==` Constant 0 |
| Units alive in a group | check 1 (group), comparison chosen, Constant N |
| Units a player has | check 8 (player), comparison chosen, Constant N |
| Always (at the start) | Constant 1 `==` Constant 1 |
| Give gold | action 23 (player, gold, default 500) |
| Win the mission, Lose the mission | action 4, action 5 |
| Show a message | action 2 (message page) |
| Give a unit, a group to a player | action 19, action 22 |
| Move a unit to a point | action 27 |
| Take a unit, a group off the map; bring it back | actions 16, 17, 32, 33 |
| Drop zone (the party starts here) | the drop action (cell) |

- A unit row offers the placed units and heroes 1 to 8; its "On map" button arms a click on a unit. A cell or
  area row has "Draw on map": the popup hides, a drag draws the box (or the centre and radius of the
  square), and the release returns to the wizard with the value in the row. Pending areas are
  outlined in gold.
- A map for more than one player warns on a hero ordinal; the player items choose the player in the
  row, so they work on any map.
- Apply writes the whole trigger as one edit and one Undo step (`InspectionTriggerWizard.WriteTrigger`,
  `pkg/game/mapinspectionwizard.go`). A row read unchanged keeps its nodes. A changed row rewrites its
  node in place when no other trigger slot names it, else makes a new node; an identical node is
  reused. A node the trigger named before and no trigger names after is deleted when that moves no
  variable register. Nothing another trigger names is changed or deleted.
- Rows fill the trigger's used pair and action slots in order, then its empty ones; an empty slot no
  row reaches keeps the words the map holds there (an empty pair's comparison included), so Apply
  with no edit leaves the trigger's bytes unchanged. A kept pair is not refused for a comparison
  word outside 0..5 that it already holds.
- A typed number must lie in -2147483648..2147483647, the 32 bits a node parameter holds; a value
  outside is refused with the reason and nothing is written.
- A shipped row the list cannot express shows as "Other: ..." and is kept byte for byte; Details
  edits it.

### The drop-zone tool

Tools > Drop zones, or D, arms the tool (`pkg/ui/mapeditordrop.go`). For every zone it draws the hero
cell and the square of the scatter radius for the units of player 1 plus the hero, the selected zone
in gold. A click on an empty cell adds a zone and selects it, a click on a zone selects it, a drag
moves it, Del deletes the selected one; Esc, D or a right click end the tool. Each zone is the shape
scenario 10 ships (`mapedit.Editor.AddDropZone`): an action "Drop Location", build-time operation,
types X and Y, and a trigger "Deploy Location" with no condition, that action only, firing once. A
delete removes the action and its trigger. A zone on a zero X or Y is refused (add and move): the
game reads a zero coordinate as "no cell" and drops the hero on a random cell of 30..100
(MISSION-DROP-002), so a zone the tool showed there would start the hero elsewhere.

## Proof

- `pkg/game`, ordinary: `TestTriggerWizardBuildsTheHeroAreaExample` (the owner's example from an empty
  map: drop zone, "Hero enters area" drawn on the map, give 500 gold, win; read back through the
  script reader and the Details view), `TestTriggerWizardEditsAndRemovesRows`,
  `TestTriggerWizardLeavesSharedNodesAlone`, `TestTriggerWizardPicksAUnitOnTheMap`,
  `TestTriggerWizardMapStartsInTheGame`, `TestDropToolAddsMovesAndDeletesZones`,
  `TestDropToolSquareHasTheScatterRadius`, `TestTwoDropZonesLandTheHeroOnEither` (24 seeds land the
  hero on both zones and nowhere else; the game reader reads 2 drop cells).
- `pkg/ui`: `TestDropScatterFollowsTheClaim`.
- Release, EN and RU (`pkg/game/mapeditorwizard_release_test.go`):
  `TestReleaseTriggerWizardOpensShippedTriggers` opens every trigger of every installed map in the
  wizard: EN 421 triggers, 118 shown fully as rows, 229 partly, 74 only as Other; RU 418, 116, 228,
  74. Every trigger offers Details.
  `TestReleaseDropZoneMatchesTheShippedShape`: a zone added on scenario 10's drop cell (17,66)
  equals its action and trigger. `TestReleaseTriggerWizardMapPlaysTheTrigger`: the example added to
  scenario 10 with the installed unit table; the hero walks three cells east; an area on the walk's
  end gives the win and 500 gold, the loss control with the area elsewhere gives neither.
  `TestReleaseTriggerWizardFrames` writes the owner frames outside the repository.
- Correction after review: `TestTriggerWizardApplyWithoutEditKeepsTheBytes`,
  `TestTriggerWizardEditKeepsSlotPositions`, `TestTriggerWizardRefusesANumberOutsideThirtyTwoBits`,
  `TestDropZoneRefusesACellTheGameWouldMove`, and the release
  `TestReleaseMapEditorWizardApplyWithoutEditKeepsEveryTrigger`, which presses Apply on every trigger
  of every shipped map (421 on EN, 418 on RU) and finds no byte changed (105 changed before).
- Existing pair-view tests, ordinary and release (`TestReleaseMapEditorTriggerPopupWritesTheFieldCallBytes`,
  `TestReleaseMapEditorTriggerFrames`), press Details first; `TestScriptParameterOpenBackStack` checks
  that Back returns to the pair view. The drop routes of the arm and window-key tests drag the map's
  one zone, which writes the bytes `SetDropCell` writes. The D help line is "Drop zones: click / drag /
  Del" so it fits the rail (`TestReleaseMapEditorHelpLinesFitTheRail`).
- `pkg/ui/testdata/mapeditor_modes.golden` is rewritten: the Tools menu on the bar and the tool's
  status line change the hashed chrome.

## Open debt

- 303 of 421 EN triggers keep at least one row as Other: checks and actions outside the list
  (variables, spells, timers, unit fields) are edited in Details only.
- "Hero enters area" names hero 1 by default; on a map for more than one player the row must name a
  placed unit, and the wizard warns until it does.
