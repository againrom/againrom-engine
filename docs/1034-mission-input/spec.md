# Story `1034` — the mission input contract: as-built specification

This document states what the build does after story `1034`, not what the contract asked for. Where
the two differ, the difference is a row in `docs/DIVERGENCES.md` and is named here.

Scope: the mission map screen's whole input surface — the two mouse buttons, the hover cursor, the
click-to-order dispatch, selection, the command panel, the minimap and the mission key set. Town,
shop and character-generation input are out of scope and unchanged.

Claim provenance is in `contract.md`. Claim ids appear below only where a behaviour is a
transcription of one specific row.

## B1 — the two buttons

The mission map's primary and secondary buttons exchanged roles. Before this story the secondary
button issued orders and the primary selected; both are now the original's (`AI-INPUT-121`,
`AI-INPUT-127`).

**Primary button.** The press marks the gesture held and zeroes the travel accumulator. Every frame
the button stays down accumulates screen travel. The release decides:

- travel **at or under** the marquee threshold: the gesture is a **click**, dispatched at the release
  position;
- travel **strictly beyond** it: the gesture is a **marquee**, dispatched as a rectangle from the
  press position to the release position.

The marquee threshold is `frameWidth*10/640`, read from the mission frame's own width rather than
from the window (`Viewer.marqueeSlop`, `pkg/ui/command.go`). `AI-INPUT-121` gives the formula's three
values, 10, 12 and 16 pixels, for the ROM1 desktop's three shipped screen widths; every viewer this
build constructs for the mission map is a fixed `MissionFrameW` (1024) regardless of the window, so
the value this build ever reads is **16, always**. The comparison is strict on the marquee side, so a
release exactly at the threshold is still a click. A floor of 1 keeps a degenerate frame from making
every release a marquee.

**The outline and the cursor read the same threshold as the release** (F1, round 2 adversarial
review). Through this story's first pass, the outline (`marqueeScreenRects`, `pkg/ui/overlay.go`)
was built at `TapSlop` (4) while the release judged the same gesture at `marqueeSlop()` (16), so a
drag of 4 through 16 pixels of travel drew a visible selection rectangle and the marquee cursor and
then dispatched as a click on release, issuing a move order per selected unit at the press point. The
outline and the release now read `marqueeSlop()`: at or under the threshold builds no outline and
dispatches a click; strictly beyond it builds the outline and dispatches a marquee.

**That is this build being internally consistent, not fidelity to a single original test.** Round 2
of this story wrote that the gesture reads one comparator throughout, which overclaims. The original
applies two structurally different tests at two sites. `AI-INPUT-121` gives the RELEASE test as the
`screenW*10/640` pixel distance above, which this build reproduces. `AI-CURSOR-230` gives the
CURSOR's own marquee replacement a different test: the marquee flag non-zero and `IsRectEmpty` false
on the normalised rectangle, which is a near-zero-area test and not a pixel distance. No claim gives
the travel at which the original begins DRAWING the outline, so the outline reading the release's
threshold is this build's choice and is recorded as `DIV-335` (UNKNOWN).

**The cursor reads the claim's own test** (round 3). `marqueeCursorLive` (`pkg/ui/overlay.go`)
answers true when the gesture is a live boxing drag that the inventory window did not take and both
axes have moved, which is `IsRectEmpty` false on a normalised rectangle. A drag of 1 through 16
pixels therefore replaces the cursor with `default` while drawing no rectangle; through round 2 it
did neither, showing the ordinary hover cursor where the original shows `default`. A drag straight
down or straight across replaces nothing, because a rectangle with a zero side is empty. `DIV-290`
carries the correction.

`TapSlop` remains 4 and still governs two things on this same screen that are not this gesture: the
mission map's own inventory-drag pickup (`pkg/ui/command.go:1163`, the doll and pack) and the shop's
whole drag, where no claim gives a distance.

**Secondary button.** The press captures; every frame it stays down pans the camera by the pointer
delta and marks the gesture panned. The release with the panned mark set does nothing further. The
release **without** it cancels:

- an armed mode, an aimed command or a selected spell is lowered, and the selection is kept;
- with none of those set, the selection is cleared.

The secondary button issues no order on any path.

## B2 — a click becomes an order by the cursor it was made under

`decide` (`pkg/ui/command.go`) resolves the frame's cursor name to an order kind
(`cursorOrderKind`, `pkg/ui/missioninput.go`) and dispatches one arm per kind. This replaces the
`attack` bool and `command` byte the order struct carried before.

| Cursor | Order kind | ROM1 opcode | What this build does |
|---|---|---|---|
| `select` | `orderKindSelect` | — | selection only, no order |
| `move` | `orderKindMove` | `0x16` | one `MapOrder` per selected unit at the clicked cell |
| `attack` | `orderKindAttack` | `0x19` / `0x16` | `MapAttack` naming the topmost hit entity; with no entity under the click it becomes `orderKindMove` |
| `swarm` | `orderKindSwarm` | `0x1a` | `MapMarch` with `patrol` false |
| `patrol` | `orderKindPatrol` | `0x1d` | `MapMarch` with `patrol` true |
| `pickup` | `orderKindPickup` | `0x21` | one `MapGrab` carrying the selected unit's id, the sack's own cell and `aimed` true |
| `cast` | `orderKindCast` | `0x25`/`0x1e` at a unit, `0x1f`/`0x26` at a cell | `MapAttack` carrying the spell id, `cell` set for the cell form |
| `defend` | `orderKindDefend` | `0x1b` | nothing (`DIV-288`) |
| `town` | `orderKindTown` | `0x24` | nothing (`DIV-289`) |
| any other | `orderKindNone` | — | nothing |

A cursor this table does not name produces nothing, which is the original's own shape: it compares
the current cursor against each registered cursor's handle, one arm per cursor, and a cursor with no
arm falls out of the chain. The eight edge arrows, the five minimap cursors, `default`, `backpack`,
`wait`, `cantput` and `dice` are all in that position.

Diplomacy is not consulted at click time. The qualifying test at the attack arm is the hit test's own.

Every remaining arm names a cell, so one ground extent test serves all of them. Outside what
`groundCellAt` resolves, a click orders nothing.

A click with nothing selected goes to selection, whatever the cursor.

### The pick-up order

`ITEM-PICK-016` (High) reads opcode `0x21` end to end. The arm takes a cell from the command's own
`cmd+0x0a` and `cmd+0x0c`, which may be any cell on the map, looks a sack up there in the world sack
registry, and refuses with the engine string `"Sack not found at "` when none stands there.
Otherwise it writes `actor+0x50 = 2` and `ord+0x0a = (row<<8)|col`. `AI-STATE-011` (High) arm 2 is
the only consumer of that state: each tick it compares `ord+0x0a` against the actor's own cell,
walking while they differ and setting `ord+0x08 = 7` when they agree, and the take runs on the
following tick. The cell and the arrival test are carried by the ordered actor.

This build composes the same behaviour from two mechanisms. `decide` emits one `orderKindPickup`
carrying the selected unit's id and the sack's own cell, and refuses when the press resolves to no
cell. `App.step` calls `MapGrab(entity, col, row, true)`. `mapWorld.grab`'s aimed arm calls
`orderPickup`, which refuses a cell with no sack on it, enqueues an ordinary move for that unit to
that cell, and arms one `pickupIntent` on the driver. `settlePickup` runs once a tick in the
post-step window that `rearm` and the odometers already occupy: it disarms when the entity or the
sack is gone, does nothing while the entity is not standing on the cell, and calls `takeSackFor` for
the **ordered** unit when it is. A later order for that unit cancels the intent, performed by
`cancelPickup` from `queueGroup`, `strike` and `castAt`. One intent is held at a time and it is not
serialized (`docs/0143-save-and-load/spec.md` FR-5 rules `pickup` cosmetic).

`DIV-337` carries the composition. `DIV-338` carries the three stores and the refusal string this
build does not reproduce: the destination slot `cmd+0x0e`, the completion state `actor+0x50 = 0xc`,
and the message, which this build refuses silently.

The pick-up **key** is a different action and is unchanged: `MapGrab(0, 0, 0, false)` leaves the
entity and the cell unread and takes the sack under the inventory window's subject. `DIV-292` was
the row for the click acting for that subject too, and it is closed by this round.

## B3 — the hover cursor

`hoverCursorName` (`pkg/ui/missioninput.go`) is a transcription of the cascade at `L01515`, the
one an ordinary hover runs (`AI-CURSOR-226`). It is not `AI-CURSOR-052`'s cascade at `L00633`,
which is reachable only through gates an ordinary hover fails.

Inputs: the present selection, the selection summary word, the hover mask, the hit entity, and the
Ctrl and Alt latches. Shift is not read; it is established to change nothing on an ordinary hover.

1. **Nothing selected** — `select` when the mask carries any actor bit, `default` otherwise.
2. **The selection is foreign-owned or is a structure** (`selSummaryForeign|selSummaryStructure`) —
   the same pair.
3. **A hostile drawable was hit** — `move` with Alt; `select` when the structure bit is set;
   otherwise `attack`.
4. **A drawable with an actor identity was hit** — `move` with Alt; `attack` with Ctrl; `pickup` when
   the pick-up gate passes; `town` when the town gate passes; otherwise `select`.
5. **Empty ground** — `swarm` with Ctrl; `move` with Alt; `pickup` when the pick-up gate passes;
   otherwise `move`.

Ctrl is tested before Alt on arm 5 and Alt before Ctrl on arm 4. That is the routine's own order, not
a preference.

Arm 5's final `move` is the visible change a player meets first: the default cursor over ground with
a selection standing is now `move`, where this build drew `default`.

**The hover mask** (`Viewer.hoverMask`) computes `AI-CURSOR-231`'s seven bits rather than one
hostility bool. The structure bits `0x20` and `0x800` are never set, because no structure is ever a
hover candidate in this build and `MapEntity` carries no class name (`DIV-284`). Arm 3's structure
branch and arm 4's `town` branch are therefore written and unreachable (`DIV-285`).

**The pick-up gate** (`pickupGate`) is `AI-CURSOR-242` whole: exactly one selected object, that
object a unit, carrying the player-character flag, with the foreign and structure summary bits clear
and the hostile mask bit clear, and either the hover mask exactly the drawable bit or the hit object
being the selection itself with that bit also set. It is not "the pointer is over a sack". The
drawable bit is read as a ground sack (`DIV-283`).

**The replacements** (`AI-CURSOR-230`) run in `missionHoverCursor` (`pkg/ui/missioncursor.go`) and
are applied as early returns, because a replacement that overwrites unconditionally and a gate that
returns early are the same function. (1) A live marquee gives `default`. (2) and (3) The pointer
inside any child widget of this build's map view gives `default`: what the original's children 2 and
3 are on the mission view is not established, and this is the approximation the story chose
(`DIV-290`). (4) The held-item case is reproduced by a different mechanism — this build draws the
dragged item's picture at the pointer and refuses to draw a manager cursor over it — so the hover
cursor does not answer while an item is held, but the slot the original names is not read. (5)
`backpack`'s own six conditions are not applied (`DIV-290`).

**An armed mode overrides the cascade**, after those replacements. `missionModeCursor` maps the
armed mode to the cursor a click is then dispatched by: 1 `attack`, 2 `move`, 4 `defend`, 5 `cast`,
6 `swarm`, 8 `patrol`. The armed mode comes first because the routine's own entry does: the cascade
is reached through a jump taken only when `view+0x99c` is zero.

**A unit in the fog changes no cursor and no click arm.** The gate is inside `hoverMask`, so it
answers for every arm at once rather than at each one.

`missionMode` derives the mode from the three pieces of arm state this build keeps, in this
precedence: `aimed == commandPatrol`, `aimed == commandSwarm`, `aimed == commandMove`,
`selectedSpell != 0`, `attackMode()`. The original keeps one number in `view+0x99c` and has no
precedence to state; this build has three writers that already lower one another, and this chain is
what makes the one remaining overlap — a spell selected while a mode is armed — a defined answer.
The Cast arm stands above the attack arm, so a player who presses the attack key and then picks a
spell casts. Before this story the attack arm was tested first and a spell picked on top of it
produced an attack order naming a spell.

## B4 — selection

`AI-SELECT-122`'s four forms, all through `decide`:

- **Plain click** replaces the selection with the topmost intersecting non-structure, regardless of
  owner.
- **Shift click** toggles an owned candidate, but only while the **old** selection summary has its
  ownership bit clear. After a plain click on a foreign actor, a Shift click on an owned one does
  nothing.
- **Plain rectangle** takes every owned non-structure overlapping by strictly more than half, and
  preserves the old selection when none qualifies.
- **Shift rectangle** toggles the same candidates under the same summary gate.

Ownership is read at the rectangle and not at the click. Ground and foreign candidates preserve the
selection: a plain click can never clear a non-empty selection, because the `select` cursor only
appears when an actor was hit.

**The selection summary** (`Viewer.selectionSummary`) is this build's `view+0x144`, rebuilt on every
read. A selection with nothing present is zero. Three of the six decoded bits have no counterpart
here (`DIV-286`); the `0x200` bit is read as the primary having a mana pool (`DIV-287`).

**The group keys** (`Viewer.groupKey`, ten slots holding ids):

- `Ctrl`+digit assigns the owned members of the current selection to that slot.
- a plain digit recalls the slot, replacing the selection.
- `Shift`+digit augments the current selection with the slot's still-present members.
- `Alt`+digit recalls and centres the camera on the recalled members' world mean.
- `Ctrl` wins over `Alt`.
- A recall whose slot has no present member leaves the selection alone.

**`E`** (`Viewer.selectAllOwnedUnits`) selects every owned live unit, sorted by id. Dead entities are
excluded. A local participant of zero takes everything. `AI-SELECT-122`'s "exact-name" clause is read
as "a unit and not a structure", which every member of this build's entity snapshot already is
(`DIV-284`).

## B5 — the command panel

Eight cells in `AI-PANEL-123`'s order: Attack, Move, Guard, Defend, Cast, Swarm, Stand Ground,
Retreat.

Six are enabled. Guard and Stand Ground are immediate one-click commands through `MapStance`.
Attack, Move and Swarm arm a mode that the next click consumes. Cast is enabled and arms nothing of
its own: it reflects the spellbook's own selection, which is what sets `selectedSpell` and therefore
armed mode 5 (`DIV-234`). Defend and Retreat ship disabled through `commandPanelSkipMask`, which is
the original's own mechanism for an unusable cell (`TOWN-092`'s skip mask); this build has neither
order (`DIV-288`, `DIV-291`).

Patrol is not one of the panel's eight cells. It is armed by the `P` key alone.

**Swarm left the skip mask at this story.** `AI-PANEL-123` gives cell 5 armed mode 6 and click opcode
`0x1a`, and `commandSwarm` — reached until now only by a key — is that order. `commandPanelSelected`
highlights the Swarm cell while it is aimed, and `pressCommandPanelCell` arms it. The order itself is
unchanged: it leaves through `MapMarch` with `patrol` false, which is story `0146`'s authored advance
order rather than a decoded `0x1a` (`DIV-230`).

**The gate is ownership, not per-class capability** (`AI-PANEL-053`'s overturn, `AI-PANEL-060`,
`AI-PANEL-061`). The panel is inactive for no selection, for foreign ownership and for the structure
flag, for every cell.

**Cast carries one further clause, wired at this round** (F9, round 2 adversarial review).
`AI-PANEL-060` reads `R0093` returning the arming mask `0xef` — every mode but cast — for any
owned selection, and adding cast's own bit only when the selection carries the spell flag
(`view+0x144 & 0x200`). This build reads that flag as the primary having a mana pool (`DIV-287`), and
`commandCastActive` (`pkg/ui/commandpanel.go`) tests it: a press on the Cast cell with no mana pool
under the primary is judged the way a press on a disabled cell is, clearing the overlay rather than
arming Cast selected. Only the press reads this clause. No cited claim describes the paint routine's
own test, so `composeCommandPanel`'s Disabled overlay and the hover tooltip do not read it and still
draw and label the Cast cell as if it were enabled whether or not the primary has mana — a player can
see the cell painted active and its label on hover, and find that a press on it clears the overlay
instead of arming it.

## B6 — the minimap

The minimap acts on the primary button's **down** edge, not its release (`AI-MINIMAP-124`). What the
press does is decided by the small cursor the widget is wearing, which `minimapActionCursor` derives
from `missionMode` — the same armed mode the map's own click dispatch reads, so the icon a player
sees and the action his press performs are one answer.

| Small cursor | Armed mode | What the press does |
|---|---|---|
| `sdefault` | nothing selected | centres the camera on the named cell; issues no order |
| `smove` | none, Move, Swarm | one move order per selected unit at the named cell |
| `sattack` | Attack | attack naming the entity standing on the named cell, and a **swarm** order at that cell with none |
| `spatrol` | Patrol | one patrol order per selected unit |
| `scast` | Cast | nothing; the claim is explicit that cast discards its hit-test result |
| `sdefend` | Defend | unreachable: nothing arms mode 4 (`DIV-288`) |

A pixel inside the widget but outside the terrain names no cell and does nothing.

**The entity lookup is fog-gated.** An enemy standing in the dark is not a target the minimap can
name, on the same terms `hoverMask` and `minimapMarks` already carry: the widget does not draw him,
and without the gate a player could pick him off a blank corner of the overview and learn he is there
from which order came out.

**A left drag repeats the action per delivered move**, not per frame: the action re-runs only on a
frame whose pointer position differs from the one it last ran at, so a held button on one pixel
issues one order. The **left release** issues nothing; it only lowers the grab.

**The right button centres and does not pan by delta.** `Viewer.step`'s right-drag branch drops its
delta while the pointer is inside the widget, and `command` centres on the named pixel. There is no
capture, so the widget answers only while the pointer is inside it. A **right release** over the
widget is a no-op: it does not reach the map's own cancel and leaves the selection standing.

**The both-buttons rule** is the claim's own: with both buttons held the left action runs and no
right-camera pan occurs. The right arm in `command` runs below the left one and is skipped whenever
the left one produced orders or the grab is up.

## B7 — the mission key surface

Read from `keyboard.tsv` (57 rows) through `AI-KEY-125`. Every key this build occupied against the
original moved. The bindings after this story, in `readAppInput` (`pkg/ui/app.go`):

| Key | Action | Note |
|---|---|---|
| `Ctrl`+`Space` | pause the world clock | `Space` alone is the original's panel toggle (`DIV-293`) |
| `Space` | open both HUD panels, or close whichever is open | new |
| `=` or numpad `+` | faster | numpad `+` is refused while `Ctrl` is held (`DIV-329`) |
| `-` or numpad `-` | slower | numpad `-` is refused while `Ctrl` is held (`DIV-329`) |
| `A` | arm Attack | unchanged; `Ctrl` excluded |
| `M` | arm Move | unchanged |
| `G` | Guard | unchanged |
| `T` | Stand Ground | unchanged |
| `P` | arm Patrol | unchanged |
| `S` | arm Swarm | was the spellbook switch |
| `B` or `Q` | spellbook | was `S`; `text/main.txt` line 8 names `<Q>,<B>` |
| `E` | select every owned unit | new |
| `0`..`9` and numpad `0`..`9` | group assign, recall, augment, centre | new |
| `I` or `` ` `` | inventory | unchanged |
| `X` | worn set | was `D` (`DIV-296`) |
| `J` | paper doll | was `D` (`DIV-296`) |
| `F9` | light step | was `F3` |
| `F10` | grid | was `F1` |
| `F11` | readout | was `F2` |
| `Ctrl`+`O` | damage numerals | was `Ctrl`+`L` (`DIV-295`) |
| `Ctrl`+`N` | day/night | was bare `N` (round 3); `keyboard.tsv` row 51 gives day/night only under `Ctrl` |
| `Ctrl`+`H` | show health, hides the health bars | new (round 3); `keyboard.tsv` row 48 |
| `Pause` | raises the dialogue notice carrying `main.txt[119]` | new (round 3); `keyboard.tsv` row 20 |

`F1`, `F2` and `F3` are vacated. Their three original actions — help, Save Game, and Load Game or
Diplomacy by campaign phase — do not exist as mission-map destinations in this build, so the keys are
bound to nothing rather than to an invented destination (`DIV-294`).

`D` and `R` are free: `D` is the original's Defend and `R` its Retreat, and neither order exists here
(`DIV-288`, `DIV-291`).

Four groups of `keyboard.tsv` rows have no decoded meaning in this build and are bound to nothing:
`F12`'s global, `Backspace`'s three containers, the `Alt`+`B`..`Y` outbound record, and `Tab`'s
character-panel message `0x412` (`DIV-321`).

**Five further groups are decoded and are not wired by this story.** Each needs a production seam
this build does not have, and building one was outside this story's scope. Each carries its own
divergence row, and `closure.md` names the follow-up work group by group.

| Group | `keyboard.tsv` rows | Row |
|---|---|---|
| `F5`..`F8` quick spell slots 0..3 | 17 | `DIV-328` |
| `Ctrl`+numpad `+`/`-`, the unpaced loop | 23, 24 | `DIV-329` |
| `Ctrl`+`W`, formation cycle | 53 | `DIV-330` |
| `Ctrl`+`F`, `Ctrl`+`L`, `Ctrl`+`U`, settings with no state here | 47, 49, 52 | `DIV-331` |
| the settings captions, shown by the original on every toggle | 47..53 | `DIV-332` |

## Headless input vocabulary

`App.HeadlessPointer` accepts six actions: `press`, `move`, `release` for the primary button and
`right-press`, `right-move`, `right-release` for the secondary. `FrontEnd`'s scenario `pointer`
command accepts the same six. The secondary half is this story's addition: a vocabulary that can only
press the primary button can drive half of this contract and cannot see the other half at all.

## What this story does not change

Town and shop input. The character card widget's own controls (`DIV-217`). The inventory grid's
internal transfer paths and the Drop Gold modal. Menu destinations that do not exist (`DIV-099`). The
Patrol default gesture, which `AI-CURSOR-126` establishes the original does not have.
