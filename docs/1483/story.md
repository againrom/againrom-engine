# Map editor: unit block, messages and movable popups

## Intent and authority

The owner reported five map editor defects: a "Show message" action showed only part of its
message; Cancel in "New message" still made an empty page; the unit popup's override block showed
the record's raw sentinels with an "Inherited; in game" line under each value; the Add popup, the
map name input and other windows could not be moved; a unit set to Health 0 was drawn standing.

The editor's layout and controls are owner direction and need no divergence row. The record format
and what the game does with each override come from `UNIT-PLACE-034`, `UNIT-PLACESKILL-086`,
`UNIT-PLACERESIST-087`, `UNIT-PLACEGEAR-098` and rows DIV-3577..3581. What the game does with a
placement authored at Health 0 comes from `UNIT-143` and `UNIT-144`. No divergence row is opened or
changed.

## As built

### Show message

A message row's label is the page's first line, with "..." when more lines follow. The action's
popup lists the whole text, wrapped, under the Message field (`messageText`,
`pkg/ui/mapeditormessage.go`; `InspectionScriptTarget.Text`, `pkg/ui/mapeditorscriptmodel.go`).
The message popup's text box grows to the whole message as far as the window allows and scrolls by
the wheel past that.

### New message and Cancel

"New message", from a node's Message field, the field prompt or the Add window's Message, opens
the message popup on a page not yet made (`NewMessagePage`). The host is not called until Apply.
Apply calls `CreateMessage` (`pkg/game/mapmessagepages.go`): the new page, its text and speaker and
the node value as one edit and one Undo step. A refused text or speaker changes nothing. Cancel and
Back leave the map bytes, the pages and the Undo history as they were.

### Unit override block

The override fields are grouped under headings: Health and mana; Attributes; Defence; Weapon
skills; Protections; Equipment. An empty field inherits, and Apply writes the field's sentinel.
The inherited value is the field's grey placeholder: the placement's value in game at Normal
difficulty, with the other overrides applied. An overridden value is drawn in gold, with an
Inherit button that empties the field. The "Inherited; in game" lines are gone. The equipment flag
is a two-choice field.

| Field | Sentinel stored | Explicit 0 |
|---|---|---|
| Health, Mana (s16 words, `UNIT-PLACE-034`) | -1 | stored |
| Body, Mind, Spirit, Reaction, Defence, Absorption (u8, `UNIT-PLACE-034`) | 0 | cannot be stored |
| Blade, Axe, Bludgeon, Pike, Shooting skill (u8, `UNIT-PLACESKILL-086`) | 0 | cannot be stored |
| Fire, Water, Air, Earth, Astral protection (u8, `UNIT-PLACERESIST-087`) | 0 | cannot be stored |
| Skip row gear (bit 7, `UNIT-PLACEGEAR-098`) | clear | not an override |

Each byte group's heading says that 0 cannot be set because the map stores 0 as inherited; a typed
0 there is refused with that reason and writes nothing. Every write goes through the SetField path
the old popup used, so the ALM bytes for the same intended values are the same.

### Movable windows

Every editor popup and input window moves by its title bar through one drag path: `titlePress`
and `dragPopup` (`pkg/ui/mapeditorwindow.go`) over the popup builder's own layout and bounds
clamp. The list and object popups keep their place; a prompt keeps an offset from its own place,
reset when the next prompt opens. Movable now: the record list, the scripts list, the map panel,
the unit, structure, sack and other record popups, the trigger and node popups, the message popup,
the group popup, the variable popup, the Add window, the path prompts (Open, Save As), the field
and workspace prompts (map name, metadata, pages), the new map size prompt, the leave question,
the test-play and mod-save notice, the new-map mode question, the area trigger choice and the
shared-area question.

### Health 0 drawn fallen

`UNIT-144` (High): the original stores an authored Health of 0 unclamped; a walking placement
enters the death arm and stays in stage 1, holding its cells. Whether the original draws that
actor at mission start is Unknown. The engine (`sim.PrepareAuthoredBody`) spawns such a placement
fallen. The editor builds each placement's actor with the game's own spawn (`unitSheets`); a
placement that is not alive draws through the game's corpse selection (`corpseFrame`, extracted
from `entityDraws` in `pkg/game/world.go`) at the frame its fall holds, in its owner's colours.
The game's behaviour is unchanged.

## Proof

- `TestMessageChoicesUseWorkspaceFirstLine`, `TestReleaseMessagePickerParts`: labels and the
  whole text per page; the label's "..." is the loss control.
- `TestMessagePopupNewApplyCancelAndBack`: Cancel leaves bytes, pages and Undo unchanged; Apply
  makes the page and one Undo removes it. `TestMessageWizardWindowOpensPage`,
  `TestMessagePopupRemovedPageReturnsToNode`.
- `TestEditorUnitBlockWritesInheritAndExplicitZero`: empty, value, Health 0, Mana 0, Inherit and a
  refused Body 0, each against SetField's bytes; loss control: Health 0 and inherit differ.
  `TestReleaseMapEditorOverridePopupWritesTheSetFieldBytes` on EN and RU: each field kind through
  the popup equals SetField's bytes and clearing restores the installed bytes.
- `TestEditorPopupsMoveByTheirTitle` (13 windows) and `TestEditorPanelNoticeAndGroupMoveByTitle`
  (3): a drag moves by the drag and the window edge stops it. Loss control:
  `TestEditorPopupDragNeedsTheTitle`.
- `TestReleaseEditorHealthZeroUnitDrawsFallen` on EN and RU: the editor's Health 0 frame equals
  the game's held fall frame from the same bytes, art, mirror, pixels and palette; loss control:
  the standing pose differs.
- Owner frames from `TestReleaseEditorUnitBlockFrames`, outside the repository.

## Open debt

- Whether the original draws an authored Health 0 actor at mission start (`UNIT-144` Unknown).
- `UNIT-144` says the original tears down a flier authored at 0 or below. The engine spawns one
  as a fallen body (observed: the bat of `loose/Beast.ALM` record 0 at Health 0, life state 1),
  and the editor draws what the engine spawns. Whether a row is owed was not settled here; the
  game is unchanged.
- The editor's popup title for a unit ("Unit #N") is not translated.
