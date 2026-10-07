# 0140 — the owner's own window

**Intensity:** spec-first / static. **Terrain:** brownfield — every screen named below already
existed and every requirement here changes one.

## Why

The game had a window, a panel, a minimap and an inventory, and each had been placed by whoever
built it. The owner played it beside his own lawful install and ruled, over three days, on where
each belongs, what it states, which press it takes, and what the game opens at. This story is those
rulings and nothing else.

## Scope

**In:** where the generation screen sits in the flow; what its Derived block states; the unit
panel's corner and row order; the minimap's size, its view mark and its click; the spellbook and
the carried pack as bars along the bottom; the doll and the worn set as two boxes in the right-hand
column; a control panel of display switches; and the size the game's window opens at.

**Out of scope:** every simulation rule — nothing here changes a number, an order, a roll or a
tick; the archive and every address in it; item, spell and unit *content*; sound; the menu and the
map list themselves, apart from the one gate a mission row consults.

## Functional requirements

**FR-1 — generation is what starting a campaign does.** Choosing a **mission** row in the map list
opens the generation screen before the map. Starting the process at a mission (the asset-root
front end's mission flag) does the same. A **loose map** row does not.

**FR-1a — a campaign successor never regenerates.** Reaching the next mission by *winning* the one
before it goes straight to the map with the party that finished it. This holds however permissive
the gate of FR-1 is: the two doors are different doors.

**FR-1b — cancelling returns where it was armed from.** Cancel on the generation screen returns to
the screen that armed it — the menu when the process began there, the map list when a row did — and
does nothing else.

**FR-2 — the separate generation flag is retired.** It is still **accepted** and does nothing: an
invocation that names it must still parse and run. No statement may read it.

**FR-2a — what a bad mission number costs is bounded, not removed.** A number that is not a
positive mission number is refused **before any window opens**. A positive number whose map the
install does not carry is discovered when that map is loaded, which is after the generation screen —
the same price a map-list row has always paid for a map that will not decode. Deciding it earlier
would mean reading the archive before the window, which this door does not do.

**FR-3 — the Derived block states what a character will have.** Sight; the five elemental
protections, each named; and every skill slot, labelled, with its trained value — an untrained
slot reading 0. Each is read from the same mint the character will be built from, never
re-derived here.

**FR-3a — the five weapon-kind numbers carry no names.** They are one row, in the array's own
column order, unlabelled. Five of the ten protection and resistance columns have no published
title, and a wrong name on a screen reads as a decoded fact.

**FR-3b — the block is clipped to a measured budget.** The number of lines it may draw is computed,
not assumed, and a block that would exceed it is cut rather than painted over the message line.

**FR-4 — the unit panel stands in the bottom-right corner** and states its subject in this order:
the four statistics against the stacked health and mana pools; damage against absorption; attack
against defence; the skill column beside the elemental one, each under its own heading; then the
totals.

**FR-4a — a heading is a field of its own** and carries the gate of the block it heads. A row none
of whose cells resolve is dropped, so a heading spelled onto the row below it would move whenever
that row's gate closed.

**FR-4b — the panel takes its own presses.** A press landing on it reaches neither the selection
nor any order.

**FR-5 — the right-hand column is one fixed width.** The minimap and the unit panel are both that
width, and it does not change with what is selected, drawn or hidden.

**FR-5a — the minimap is square.** Always, at every window size, whatever the map's own shape. A
map that does not fill the square is centred in it rather than stretched.

**FR-6 — the minimap marks where the camera is looking**, as an outline of the camera's own visible
extent, never smaller than one pixel.

**FR-7 — a press on the minimap centres the view** on the cell under the cursor, and holding the
button drags it. A pixel inside the box but outside the terrain is still the minimap's press and
names no cell. This reverses `0118 AC-12`.

**FR-8 — the spellbook and the carried pack are two bars along the bottom.** The book is two rows
of cells, filled column by column; the pack is one row of larger cells below it. Both run from the
window's left margin to the right-hand column, whatever is in that column.

**FR-8a — the pack scrolls, and what it scrolls over is unbounded.** Two buttons at the bar's right
end move it by exactly one cell a press. A carrier holding more elements than the bar has cells is
reachable by scrolling to them; there is no cell count above which an element cannot be shown.

**FR-8b — a pack cell names an element of the container.** The index a press resolves to is the
index in the carrier's own container, not in the visible strip, so a double-click that equips
means the same thing at every scroll position.

**FR-9 — the doll and the worn set are two boxes in the right-hand column**, the worn set above the
doll and the doll above the unit panel.

**FR-9a — the doll follows the selection.** It shows the selected unit, whether or not that unit is
the party's own character. Where this build can compose a figure for that unit it draws it; where
it cannot it draws that unit's own drawn world frame instead, which is a **disclosed substitution**
and not this requirement met.

**FR-10 — a control panel of four switches stands under the minimap.** One switch per box — the
pack, the spellbook, the doll, the worn set — each drawn with its own state visible as pressed or
not pressed. All four start on.

**FR-10a — a switch is a setting.** It moves only when its own button or its own key is pressed,
and it moves whatever is selected. Nothing else changes it: not a selection change, not cancel, not
a mission ending.

**FR-10b — a box is drawn when its switch is on and it has something to show.** With nothing
selected, nothing is shown. The pack and the worn set additionally require that the selection is
exactly the one character this build composed them for.

**FR-10c — the control panel is always reachable.** Its place is a function of the window's size
alone, and it is not gated on a selection, on a font, or on any box it switches.

**FR-11 — every box drawn over the map takes its own presses**, and a box that is switched off
takes none: the map beneath it receives the press exactly as if it had never been drawn.

**FR-12 — the game opens at the size of the monitor**, undecorated. It is a plain window: no video
mode is changed and no display is taken exclusively.

**FR-12a — a monitor that reports nothing usable falls back** to a window of the authored size,
**decorated** — that branch is taken precisely when the screen's edges are unknown, so the window
must be one a player can move.

## Properties

**P-1 — no test opens a game install.** Every criterion below is met by a synthetic fixture, or is
evidence recorded in `verification.md` from a developer run.

**P-2 — every composition is total.** No picture, an unreadable one, an empty subject, a unit with
no art: each draws as an absence. Nothing panics and no box fails to compose.

**P-3 — the drawing tier holds no archive, no definition table and no equipment slot.** It receives
composed pictures and plain scalars.

**P-4 — no box's placement is a function of the selection**, except the two this contract names:
the doll and the worn set, which are pinned above the unit panel and move with its fitted height.

**P-5 — nothing here reaches the simulation.** No switch, no press on a box, no window size changes
a selection, an order or a tick.

## Acceptance criteria

**AC-1** Choosing a mission row in the map list reaches the generation screen; choosing a loose map
row reaches the map.

**AC-2** Winning a mission whose successor exists reaches the successor's map directly, with a gate
installed that claims **every** row.

**AC-3** Cancel on the generation screen returns to the map list when a row armed it and to the menu
when the process did.

**AC-4** An invocation naming the retired flag parses and runs and reports the same thing without
it; a non-positive mission number is refused with nothing printed to standard output and no window
opened.

**AC-5** The Derived block states Sight, five named protections and six labelled skill slots, of
which exactly one is nonzero on a freshly generated character.

**AC-6** Spending Spirit moves the protection row.

**AC-7** A block with more rows than the measured budget is cut to the budget.

**AC-8** The panel's rows appear in the order FR-4 gives, with each heading present exactly when its
own block is.

**AC-9** A primary and a secondary press inside the panel's rectangle change no selection and issue
no order.

**AC-10** The minimap's box is exactly the fixed column width and is square, both with a unit
selected and with none.

**AC-11** A press inside the minimap's terrain centres the camera on the pressed cell; a press
inside the box but outside the terrain centres nothing and reaches no map.

**AC-12** With a pack longer than the bar, one press of the forward button moves the strip by
exactly one cell, and the cell under a fixed pixel then names the next element of the container.

**AC-13** The doll is drawn for a selected unit that is not the party's character, and for none when
nothing is selected.

**AC-14** With every switch on and a subject selected, all four boxes are drawn; switching any one
off removes that box and leaves the other three; switching it on restores it.

**AC-15** A press on a switch's button flips that switch and reaches no map; a press on the control
panel's frame flips nothing and still reaches no map.

**AC-16** Cancel changes no switch.

**AC-17** The window opens at the reported monitor size and undecorated; a monitor reporting a
non-positive or absent size opens the authored size, decorated.

**AC-18** The shipped binary opens against a lawful install at the monitor's own size and stays up.
