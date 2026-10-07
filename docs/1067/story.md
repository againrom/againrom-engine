# 1067 — Mission character-pane cancel

## Player result

A right-button release anywhere on the mission character pane now performs the
same cancel as an ordinary right click on the map. An armed attack, aimed order
or selected spell is cleared while the selected units remain selected. With no
mode armed, the selection is cleared.

The pane owns a secondary gesture begun on it until release. Holding or moving
that gesture across the pane cannot pan the camera or leak an input into the map.
The six left-button rectangles, equipment-slot mask, command panel, minimap and
inventory drag routes are unchanged.

## Authority and implementation

`TOWN-344` (High) gives the character widget a coordinate-free
`WM_RBUTTONUP` handler that posts map message `0x405` during a mission; its
right-down and right-double-click handlers do nothing. `AI-CURSOR-177` (High)
identifies `0x405` as the map command cancel. `AI-INPUT-127` (High) gives the
cancel decision: an armed mode absorbs it and preserves selection, otherwise it
deselects.

`Viewer.cancelMapCommand` is the single consumer of that decision. Both the
ordinary map right-up and the character pane call it. The pane branch precedes
the other mission HUD surfaces and removes only its owned secondary facts from
the local command snapshot; primary presses and releases continue through the
existing corner, equipment and inventory routes. `rightDragIntent` still
advances its anchor over the pane, but emits no camera delta and raises no
map-drag mark there. No simulation, save, hash or byte-form state changes.
`DIV-217` is closed; the separate unknown session masks on town-side screens
remain `DIV-311`.

## Proof

- `pkg/ui/missionpane_cancel_test.go` exhausts every pixel of the 160x242
  mission pane and requires the same no-order cancel, including with a prior map
  drag mark.
- The same file discriminates attack, aimed-order, selected-spell and unarmed
  selection outcomes.
- Two complete press/move/release paths prove that a pane-owned gesture cannot
  start panning after leaving and that an outside gesture banks no hidden delta
  while crossing the pane.
- Both-buttons regressions require Backpack to toggle and an active pack drag
  to finish its release cleanup while the pane still owns a held right button.
- Existing character-pane geometry and command tests continue to cover the six
  left rectangles, equipment routes, command-panel right-up and ordinary map
  cancel.

## Open debt

`DIV-311` remains open for the authored session mask used by town-side screens.
This story changes only the researched mission value `CharacterPaneMission`.
