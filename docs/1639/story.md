# Every mission key is a row of the one mission key table

## Intent

A key the player presses in a mission is rebound by changing one row. Every mission key of both games is a row
of `missionBindings` in `pkg/ui/missionbindings.go`, and one reader, `readMissionKeyRows`, turns the rows into the
fields of `appInput`. No key does anything different in either game.

## Authority

Owner direction in the story brief («Да переносим в таблицу»): the mission keys still read where the application
input is read (`readAppInput` and the helpers it called) move into the table; keys of other kinds stay. It is
architecture work with no player-visible change. The one difference from the original found on the way is left
and recorded as DIV-4896.

## As-built behaviour

The row. `missionKeyBinding` is one form for every row: the games it applies to (`only`, both when zero), the
scenario name, the key, a test of each modifier (`modAny`, `modHeld` or `modFree` for Ctrl, Shift and Alt), `hold`
for a level instead of a press edge, the action and its argument (the quick spell slot or the group digit). The
second game's Support on L is a row marked `only: secondGameKeys`; the policy flag that appended it is gone.

The profiles. `missionKeyPolicies` holds per profile the games whose rows apply and `orders`, the Ctrl test of the
order letters P, G, D, C, S, T, R and M: any in the first game, free in the second. The P row's action is the
profile itself (Patrol or Pickup). The rows are 84 for the first game and 86 for the second, over 66 keys; the
table test lists each of them.

The reader. `readMissionKeyRows(p, pressed, held, ctrl)` sets the action of every row whose key is down (a press
edge, or the level for a `hold` row) and whose three modifier tests admit Ctrl, Shift and Alt. Where two rows set
one field the rows keep the base's precedence: the lowest group digit wins, and a bare zero is the clock pause and
not group 0. `readAppInput` reads the keys of other kinds into its literal and then calls the reader once with
the engine's press edge and held level. The scenario harness presses a named row through `missionKeyNamed`, which
reads the same rows.

A rebind changes one row and the matching line of `TestMissionKeyTableListsEveryKeyOnce`. Moving Worn from X to
V:

```
-		{name: "x", key: ebiten.KeyX, action: missionKeyWorn},
+		{name: "v", key: ebiten.KeyV, action: missionKeyWorn},
```

```
-		"e = select-all", "i = inventory", "backquote = inventory", "x = worn", "b = book", "q = book",
+		"e = select-all", "i = inventory", "backquote = inventory", "v = worn", "b = book", "q = book",
```

`TestTheFourKeysAreBoundOnceEach` (`pkg/ui/vocabulary_test.go`) names X among the keys bound once; a rebind of one
of its ten keys also edits that list.

## The boundary of the table

These keys are of other kinds and are read outside the table. `internal/archtest` names each site and field with
its kind.

| Key | Field or site | Kind |
|---|---|---|
| Esc | `Escape` | menus and dialogues; the one back key of every screen |
| Enter, numpad Enter | `Enter` (`enterPressed`) | menus and dialogues; Alt+Enter is the window toggle |
| Up, Down, Left, Right (press) | `Up`, `Down`, `Left`, `Right` | lists, notices and text fields |
| Page Up, Page Down | `PageUp`, `PageDown` | lists and notices |
| Home, End (press) | `Home`, `End` | text fields and the help notice; on the map Home and End are also the ZoomReset and SpeedReset rows |
| Backspace (map message clear) | `ClearMessages` | the `backspace` row, both games, Alt free; the map arm reads `ClearMessages` and never `Backspace` |
| Backspace, Delete, Ctrl+A, typed characters | `Backspace`, `Delete`, `SelectText`, `Typed` | text fields only (`Backspace` is the text-field read; the map's message clear is its own row, above) |
| Tab | `PaneMode` | the character pane |
| bare L | `Load` | the main menu's Load; the map's L keys are rows |
| Alt+A..Z | `AltLetter` (`readAltLetter`) | the debug console; the letter chooses the console action |
| any key | `AnyKey`, `AnyHeld` | cutscene skip |
| arrows (held) | `readInput` | camera pan, shared with the map editor and the standalone viewer |
| Ctrl, Shift, Alt, Command | `readInput`, `altHeld`, `platformCommandModifier`, `shiftHeld` | modifier levels the rows test |
| editor keys | `readEditorInput` | the map editor |
| Esc of the standalone viewer | `Viewer.Update` | the developer viewer |
| starter keys | `cmd/starter/window.go` | the starter window |

## Proof

- `TestMissionKeyTableListsEveryKeyOnce`: every row of each game as one line (key, modifier tests, hold, action,
  argument), no line twice, one key per name.
- `TestMissionKeyRowsReadAsTheBaseRead`: a transcription of the base's readings (the literal fields of
  `readAppInput`, `barePauseKey`, `groupDigit` and the base table) against the reader, for every key alone and
  every pair of the contested keys (A, F, L, W, Space, =, -, the numpad signs, Home, End, Z, both digit rows,
  F4..F12), under the eight Ctrl, Shift and Alt combinations, as an edge, a level and both, in both games.
- `TestMovedMissionKeyNamesPressAsTheBaseCasesDid`: the scenario names of the moved keys give the frame the
  base's own cases gave, in both games.
- Nine source-shape binding tests now check the row of their key, its Ctrl test and its press edge instead of the
  text of `readAppInput`; their expectations are unchanged.
- `internal/archtest`: `TestMissionKeysAreReadOnlyThroughTheTable` parses the production files under `pkg` and
  `cmd` that read a key and refuses a key read outside the reader and the named sites of other kinds, a mission
  field in `readAppInput`, and a scenario case of a row's key that sets a mission field itself.
  `TestMissionKeyGuardRefusesReadsOutsideTheTable` holds its refusals on planted sources.

## Open debt

- DIV-4896: a row that does not test Alt acts with Alt held, where `MENU-082` routes most such keys to no arm.
- The scenario name `alt-f12` presses the F12 row with Alt held. In the second game that casts quick spell slot 9,
  as the physical chord does; the base harness raised FPS instead, which the map drops under Alt.
- Unknown: the second game's Alt routing; whether the camera arrows and the Alt letters should become rows.
