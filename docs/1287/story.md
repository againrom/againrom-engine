# Story 1287: mission screen layout rows

## Intent

Re-read the eight open rows of the mission screen's right column and cursor
slots (DIV-200, DIV-201, DIV-284, DIV-247, DIV-248, DIV-260, DIV-286,
DIV-1533) against the current code and every claim in the pin, change what a
claim establishes where the player sees a difference, and state the rest as
open questions.

## Authority

Pin k122. Population searched: a text search of every claim file for widget 8
(`R0349`, `R1937`, `extra1024r`), `+0xfc`/`+0x100`,
`view+0x990`, the minimap paint and handler rows, the 28 cursor slot names,
`Usable`/`Indestructible`, `view+0x144` and `AI-STRUCTUSE`. Registry census:
`cmd/classdump` over the EN and RU `graphics.res`; placement census over
missions 1..160 at Normal on the EN install. B1 holds: no question below
carries an expected answer.

## As-built behaviour

- Hover over a structure. The hover hit test skips a structure whose class is
  indestructible and not usable (`AI-CURSOR-231`, class flags `REG-STR-080`,
  class array `REG-KEY-044`). With a unit selected, the pointer on a Well 1
  (class 8, 22 placements in the campaign maps) now shows `move`, not
  `select`; destructible and usable classes keep `select`. `structureHoverHit`
  in `pkg/ui/missioninput.go`. Hover inspection cards are unchanged.
- Click on the same structure. A map click is turned into an order by the
  cursor it was made under, not by what it hit (`AI-CLICK-050`), and the click
  reads the same mask as the hover. So with a unit selected the click is a
  plain move order (`move` arm, opcode `0x16`) to the structure's cell: the
  selection stays, the unit walks to the cell beside the well and stops there.
  With Ctrl held the cursor is `swarm` and the click is the swarm order; with
  nothing selected the cursor is `default` and the click deselects nothing and
  orders nothing. Before this change the same click was a `select` click that
  cleared the selection over the structure. The original's skipped-structure
  click has no claim of its own beyond the cursor dispatch; no remaining
  mismatch is recorded.
- DIV-284 narrowed: 9 of 66 registry classes are usable (15, 16, 28, 29, 34,
  35, 39, 40, 66) on both installs; 15, 16, 28 and 29 are implemented. Classes
  35, 39 and 40 are placed in shipped missions and have no health pool here, so
  their hover is not a hit.
- DIV-247 rewritten to the current census: 19 of 28 slots reach the screen, 8
  are named by no code path, `wait` is named and never composed.
- DIV-260 narrowed: the minimap answers `default` under the owner's camera-only
  minimap (DIV-320); the rectangle decides only where `default` stands.
- DIV-200 narrowed to widget 8's readout text; geometry and strip bitmap match.
  DIV-201 narrowed to what id 5 draws (UNKNOWN); id 6 is built.
- DIV-286 gains `AI-SELECT-122` (destructible structures are selectable in the
  original) and the Unknown on the `0x8` set site.

## Rows verified, not changed

- DIV-248: `flow.surfaceTransition` still sets `wait` and the exit cursor in
  one statement, and screens are entered inside one `Update`. Unchanged.
- DIV-1533: `App.step` stores the tick's pointer, `App.cursorPlacement` places
  from it; the two named tests exist. No window measurement exists. Unchanged.

## Proof

- `TestHoverCursorSkipsIndestructibleUnusableStructure` (`pkg/ui`): four flag
  pairings on one structure through the production pointer route.
- `TestReleaseHoverCursorOverStructureClasses` (EN and RU, mission 20): pointer
  hover on installed Well 1, a destructible class and a usable class; reads the
  cursor manager after the production step. Loss control: before the filter,
  Well 1 gives `select`.
- `TestReleaseClickOverIndestructibleStructureMoves` (EN and RU, mission 20):
  pointer hover and click on Well 1 with the hero selected. The cursor is
  `move`, Ctrl gives `swarm`, no selection gives `default`; the click keeps
  selection [56], queues one move order, and 400 steps later the unit has a
  target beside the well and has closed more than 20 cells. Loss control (the
  filter forced off): the cursors read `select`, `attack`, `select`, the click
  leaves selection [] and queues no order. A headless `ctrl-` pointer prefix
  holds Ctrl for the test.
- `TestReleaseStructureClassesWithUsableFlag`: the nine usable classes.
- `TestRegisteredCursorSlotsNamedByProductionCode`: the eight unnamed slots,
  with a scan control.

## Open debt

- Widget 8's readout (name, owner, `"%d/%d"` ratio at 1024x768): which class
  `+0xfc` and `+0x100` belong to for the object it reads, the sources of the
  two text lines, the rectangle, font and colour, and whether the object is the
  hovered or the selected one.
- What the `[this+0x9bc]` array of the minimap paint holds and what its masked
  pixel loop draws.
- The use effect of order `0x24` on classes 34, 35, 39, 40 and 66; the hit
  source for usable structures without a health pool.
- The surface that reaches the `dice` slot.
- Structure selection and the `0x20` consumers (DIV-286).
- A staged screen entry that composes a `wait` frame (DIV-248).
