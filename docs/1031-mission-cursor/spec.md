# Story `1031` — the mission map's own cursor: as-built

Canonical at landing. Describes the shipped behaviour, not the original intention in
`contract.md`.

## Result

On the mission map, outside attack mode and outside a held item, this build draws its own
cursor and hides the operating system's arrow. The five behaviours (B1-B5) are one function,
`Viewer.mapCursorName` (`pkg/ui/missioncursor.go`), called from `advanceCursorManager`
(`pkg/ui/cursor.go`) on every tick the map is showing, and one predicate,
`Viewer.mapCursorPresent`, read from `Viewer.drawPointer` (`pkg/ui/viewer.go`) and from
`flow.pointerWanted` (`pkg/ui/flow.go`).

`advanceCursorManager` runs BELOW the three statements in `Viewer.step` that store this tick's
cursor position and primary-button state, and above the popup return. Every branch of
`mapCursorName` therefore names a cursor for the position `mapCursorPresent` will draw the
picture at, on one tick. Until adversarial pass 1's first finding the call stood above those
stores, and every branch chose a name for where the pointer had been on the previous tick while
the picture went where it is now: at the ordinary mouse speeds of an edge pan, a one-tick lag
puts the arrow on and off one tick late at every band crossing.

## B1 — the eight edge arrows

`missionEdgeArrow(x, y, w, h, margin int) (string, bool)` is a pure function: given a window
position and size and a margin, it returns one of `arrow0`..`arrow7` or no match. The direction
map is `AI-CURSOR-190`'s own compass, corroborated by `SPR16A-CURSOR-067`'s registered hotspots:
`arrow0` north, `arrow1` north-east, `arrow2` east, `arrow3` south-east, `arrow4` south, `arrow5`
south-west, `arrow6` west, `arrow7` north-west. A corner (both an x-band and a y-band) picks the
corner's own arrow; a single band picks the side.

`missionEdgeBands(x, y, w, h, margin int) (left, right, top, bottom bool)` is the band
arithmetic, also pure, and `Viewer.edgeScrollBands(primaryDown bool)` is the one predicate the
map's edge-scroll term and the arrow cursor both read. `panIntent` (`pkg/ui/viewer.go`) calls it
and turns each true band into a `PanSpeed` term; `mapCursorName` calls it and turns the same four
booleans into an arrow name. The arrow can therefore only show on a tick the pan runs, which is
what adversarial pass 1's second finding reported: `panIntent` suppressed the pan while the
primary button is held (FR-9, a marquee drag must not scroll) and the cursor's own copy of the
band arithmetic did not, so the arrow appeared on a tick that panned nothing. Making the two read
one predicate closes the class rather than that one instance.

The band is the window's own edge-scroll margin (`EdgeMargin`, 24 window pixels on every side),
the same test `panIntent` (hotfix `ecde8e52`) reads for panning, not `AI-CURSOR-190`'s own
screen-pixel bands (1px top/left, 2px right/bottom in the original's 640x480 surface). This build
has two surface models on the map (DIV-249) and no single pixel space to transcribe those two
bands into; reusing the window's own edge-scroll band means an arrow shows exactly where hovering
already pans the view. Recorded, not silently chosen.

## B2 — the minimap's own mode cursor

`Viewer.minimapModeCursor` reads the pointer against this build's own minimap widget
(`minimapCaptures`), not a transcription of `AI-CURSOR-208`'s own unestablished rectangle
(DIV-260). `minimapCaptures` tests the column's full reserved slot (`rightColumnBox`,
`hudMinimapReserve` wide, the same shape the other three right-column widgets already used),
not only the minimap's own drawn square, which is two pixels narrower than its slot so the box
can stay square (adversarial pass 3, P-1's population sweep): before this round a press in that
two-pixel strip reached the map's own pick test with no widget gate at all.
Over the widget: no selection forces `sdefault` (`AI-CURSOR-202`); otherwise the
armed state selects among `sattack` (attack mode), `scast` (a spell selected), `spatrol` (the
Patrol standing order) and `smove` (everything else, including this build's own March order,
which the original's jump table has no entry distinct from a plain move for). `sdefend` has no
armed state of this build's own to reach it from (DIV-261); the Ctrl/Alt modifier cursors
(`swarm`, forced `move`) are not read by this build's input model (DIV-262).

## B3 — the hostility test at hover

`Viewer.hoverHostilityCursor` picks the top entity under the pointer
(`topAt`/`entityPickRect`, the same pick geometry the click path uses), fog-gates it
(`fogGateEntity`, `attackTargetRect`'s own 2026-08-08 precedent: a unit the fog hides must not
leak presence through any indicator), and reads `MapEntity.Hostile`: hostile selects `attack`,
otherwise `select`. `UNIT-HOVER-020` states the hostility bit is set "for every class" the hit
test names, so a hostile structure and a hostile unit read the same way.

**Where `attack` comes from, and where it does not.** `AI-CURSOR-052` covers one block of
`R0219`, `L00633`-`L07916`, and in that block the hostility bit `0x4` appears only
inside `mask & 0x24`, whose effect is to suppress `select`. Suppressing `select` is not selecting
`attack`, and this spec asserted that it was until adversarial pass 1's third finding. The
`attack` write for an unmodified hostile hover is in a second selection tree in the same routine,
`L01515`-`L13182`, printed by `EXP-0216`'s `evidence/disasm-listings.txt`: `L00635` reads
bit `0x4` and `L01527` writes `L00628`, which `SPR16A-CURSOR-067` gives as slot 3 `attack`.
This build implements the write and does not implement its three further gates - a selection
being present (`L01524`, `view+0x140`), `view+0x144 & 0x24` (`L01525`), and hover mask bit
`0x20` (`L01526`) - nor the modifier latch at `L07921`. `DIV-270` states the question, the
reading taken and what the other reading would produce on shipped content; `DIV-262` states the
modifier latches; `DIV-263` states the mask's second bit, which has no counterpart on `MapEntity`.

**A published claim covers part of this same tree, and adversarial pass 2 corrected an unscoped
negative that said otherwise.** `AI-CURSOR-209` (Medium) reads `L01682`-`L01410` and
`L01412`-`L01411`, inside the same `L01515`-`L13182` range, for the `town` pick
below. A keyword search on the five addresses that gate the `attack` write itself
(`L01524`, `L01525`, `L13183`, `L01527`, `L01526`) returns no row at pin `753034d`, and
this spec previously generalized that scoped search into "no published claim at pin `753034d`
covers that tree" - false of the tree as a whole, true only of those five addresses.

The `town` cursor (`AI-CURSOR-209`) is not implemented: its own entry test reads in game terms
(`view+0x140==1` is exactly one object selected, `view+0x144&0x1` is "the selected object is a
CUnit", both from `AI-PANEL-061`), but its refinement's own two bits - the selected object's own
`+0x18c` bit `0x1` and the hover hit-test's own bit `0x800` - are Unknown in game terms.

**The projection.** `MapEntity.Hostile` (`pkg/ui/overlay.go`) is filled once per tick in
`pkg/game/world.go`'s `entityDraws`, from `sim.Relations.Hostile(sim.SelfSlot, e.Owner)` —
the simulation's own directional relation, read live off the session matrix. `UNIT-VPLAYER-021`
(amended; its retracted first reading is superseded by `EXP-0110`) decodes the original's own
mechanism as a cached VIEW-SIDE 32-entry row, filled at `{0, 1, 0xa}` and matching
`AI-DIPLO-005`'s session values `{0, 1, 2}` on bit 0 alone, at Medium confidence — no instruction
linking the view-side row to the session matrix was read. This build has no view-side cache and
reads the session relation directly (DIV-259): the more current answer wherever a script changes
a relation mid-mission, at the cost of the Medium link between the two records' semantics.
`pkg/ui` may not import `pkg/sim` (`internal/archtest`); `pkg/game` is the one tier that reaches
both and is where the projection is built.

**The armed-attack marker shares this section's own pick geometry and gate.**
`Viewer.attackTargetRect` reads the same `topAt`/`entityPickRect` pair as `hoverHostilityCursor`
and is gated the same way, on `v.mapSurfaceCaptures(v.cursorX, v.cursorY)` (adversarial pass 3,
P-1): before this round neither call carried that gate, so an entity whose cell straddles the
map's own right edge kept a pick rectangle reaching past it, and a pointer resting on the HUD
column named that entity for the hover cursor and the attack marker alike. The returned rectangle
is then clipped to the world viewport by `clipScreenRectToViewport` (`pkg/ui/cursor.go`), which
insets the clip bound by half of `AttackMarkerWidth` (adversarial pass 3, F2/P-2): the only
caller paints this rectangle with `vector.StrokeRect`, which centres its stroke on the rect's own
edge rather than drawing inside it, so a rectangle clipped flush to the viewport bound still
painted one pixel column into the panel. The inset makes the PAINTED span stop at the viewport
edge rather than the returned rectangle clipped to it with a stroke still centred on that edge.
`TestTheAttackTargetMarkerIsClippedToTheWorldViewport` (`pkg/ui/pointer_test.go`) asserts the
painted span, not only the returned rectangle.

## B4 — the held-item cursor

Unchanged from story 1005/0110's `dragItemPresent` (`pkg/ui/inventory.go`), which already draws
the held item's own picture on the map while a drag is active. This story's only change is
`advanceCursorManager`'s own `dragActive` branch: it leaves the manager's current cursor
untouched (`AI-CURSOR-193`'s persistence rule — a path setting no cursor leaves the one
displayed standing) rather than setting a runtime-constructed cursor of its own, and
`mapCursorPresent` excludes `dragActive` so the map's own cursor cannot draw underneath the held
item.

## B5 — draws ours, hides the system pointer, from the same answer

`Viewer.mapCursorPresent() (*image.RGBA, image.Point, bool)` is the single source both
`Viewer.drawPointer` and `flow.pointerWanted` read. It returns `false` while attack mode is shown
(`attackShown`), while a drag is active (`dragActive`), while the cursor position has never been
observed (`!hasCursor`), or while `v.cursorAttackFromMode` is set — reachable for one tick after
attack mode is lowered, since `advanceCursorManager` runs before that tick's own input is
processed and therefore names the cursor from the PREVIOUS tick's `attackHeld`
(`TestLeavingAttackModeLowersTheMapsOwnCursor`, `pkg/ui/cursorlifecycle_test.go`, already covers
this one-tick lag; `mapCursorPresent`'s own guard is what keeps `pointerWanted` correct during
it). `cursorAttackFromMode` replaces an earlier guard that refused to draw whenever the manager's
current NAME was `"attack"`, on the theory that only `attackShown` could have put it there. The
theory was false: `hoverHostilityCursor` (B3) legitimately names `"attack"` on an unmodified
hostile hover, through this same function's default branch, and the name-based guard refused that
legitimate selection too, leaving the system arrow over the enemy unit (adversarial pass 2, F1).
`cursorAttackFromMode` is set only where `advanceCursorManager` enters through `attackShown`
(`pkg/ui/cursor.go`), so `hoverHostilityCursor`'s own `"attack"` never trips it. Otherwise it
draws the manager's current picture at `(cursorX, cursorY)` minus the slot's
own hotspot, 1:1 in mission-frame pixels: `App.Draw` then scales the whole 1024x768 mission
frame to the window, exactly as it scales the 640x480 canvas every other screen uses, just at a
different frame size and scale factor (`DIV-249`, corrected at adversarial pass 2's F5 - the row
previously read the map as unscaled).

`flow.pointerWanted`'s map branch now checks three things in order — the attack pointer, the
map's own cursor, the held item — and returns true on the first that is showing. That is
`drawPointer`'s own order; the two disagreed until adversarial pass 1's return, with no
observable consequence, because a frame showing any of the three wants the system pointer hidden
whichever one it finds first. Before this
story a hover with no mode up and no drag left the system pointer showing on the map, because
nothing there drew a cursor of the build's own for that case; `TestOrdinaryHoverHidesTheSystem-
PointerThroughTheFrontEnd` (`pkg/ui/missioncursor_test.go`) is the witness, mutation-proved
against `mapCursorPresent`'s own fallback branch.

**The composition order.** All three mission pointer pictures - the attack pointer, the map
cursor and the held item - are drawn by one function, `Viewer.drawPointer`, from one statement at
the end of the frame, after the right column's boxes. `Viewer.DeferPointer(true)` moves that
statement out of the viewer's frame and into `App.Draw`, which runs it after
`drawGameMenuOverMap`, so the pointer is composed over the in-game menu as well. The owner
reported on 2026-08-22 that the cursor is drawn under the panels and under the pause menu; three
sites drew a cursor picture on the map and the three disagreed, the attack pointer and the map
cursor above the right column's boxes and the held item below them. `App.Draw` passes the
VIEWER's placement back in - `place.Scale()` and `place.Origin()` - because the map is a 1024x768
frame fitted to the window while every other screen is a 640x480 canvas fitted to it (`DIV-249`).
Composing the pointer last is authored, not decoded: `DIV-271`.

## Precedence

`mapCursorName` answers `default` while a popup stands, and otherwise tries B1, then B2, then
B3, in that order, with `default` as the fallback. The popup gate is first because `step` returns
above `panIntent`, above the drag and above the wheel while a popup is up, so an arrow, a minimap
mode cursor or a hover cursor named on such a tick describes an interaction the frame will not
perform. `AI-CURSOR-193` reads `R0219`'s entry gate as the same shape one level up, a
return when the surface state word is not exactly 1; this build answers `default` where that gate
makes no change, deliberately, so that a mission-end notice sending the flow to the map list
cannot leave the manager holding `attack` (`DIV-272`). The order follows
the decoded routines: `AI-CURSOR-190`'s and `AI-CURSOR-207`'s own blocks both test the screen
edge before falling through to anything else (`AI-CURSOR-207`: the four arrow jumps bypass the
mission-view load, the widget bounds-check, the `[EBP+0x140]==0` test and the armed-mode jump
table entirely), so a pointer at both an edge and a unit's own picked rectangle draws the edge,
never the unit. B4 (drag) and the attack pointer are checked earlier still, in
`advanceCursorManager` and `mapCursorPresent` respectively, ahead of any of B1-B3.
`TestMapCursorNamePrecedenceEdgeBeatsMinimapBeatsHover` witnesses all three orderings, each
produced by removing the winner from the row above.

## G2 — the customisation seam

The selection is a table read at call time: `mapCursorName`'s four branches (edge, minimap,
hover, default) are each one function returning a slot name, and the slot names are the strings
the registry `1030` built from `SPR16A-CURSOR-067`/`SPR256-CURSOR-046` already carries. A later
reader can add a new condition by adding a branch, or change which armed state maps to which
`.256` slot by editing `minimapModeCursor`'s switch, without touching the registry, the manager
or any other screen. No shipped file's bytes change: every slot this story selects among was
already registered by story 1030 from the shipped cursor archives, and this story adds no new
archive path, no new picture and no new byte-form field.

## What is not implemented (see `docs/DIVERGENCES.md`)

`DIV-259` (hostility reads the live session relation, not a view-side cache), `DIV-260` (minimap
widget rectangle is this build's own), `DIV-261` (`sdefend` has no armed state), `DIV-262`
(the three modifier-key cursors), `DIV-263` (`town` cursor; the hostility mask's second bit),
`DIV-270` (the hostile-hover `attack` without the original's three further gates), `DIV-271` (the
authored pointer composition order), `DIV-272` (`default` while a popup stands). `DIV-247` is
narrowed at this landing: sixteen of the 28 registered cursor slots now reach the screen, up from
three.
