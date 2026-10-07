# 0140 — tasks

**Kinds:** every task below is `impl`.
**Legend:** `Covers:` are the upstream ids the slice answers. `Fences:` bound it. `Done when:` is
the whole exit condition.

**Order is ruling order, not dependency order.** T5, T8 and T12 exist because T4, T7 and T11 were
wrong or incomplete when the owner saw them running. Each stays its own entry: the sequence is the
evidence that every shape was checked against the game rather than against a document.

---

## T1 — `impl` — a picker row can arm the generation screen (`pkg/ui`)

**Files:** `pkg/ui/flow.go`, `pkg/ui/app.go`, and their `_test.go` siblings.

**Covers:** FR-1 (the gate alone — no row claims it yet), FR-1b; D-1.

A gate consulted before the loader, answering in this package's own vocabulary: a row index in, a
model and a begin callback out. One assignment of the armed screen, so both doors inherit one nil
refusal. Cancel returns to the screen the gate was armed from.

**Fences:** name nothing from `pkg/data` here, and decide no row's kind — that is T2's.

**Done when:** a gate claiming a row reaches generation from the picker, one claiming none reaches
the loader, and cancel lands on the arming screen in both directions.

---

## T2 — `impl` — mission rows install the gate (`pkg/game`)

**Files:** `pkg/game/frontend.go`, and its `_test.go` sibling.

**Covers:** FR-1; D-1.

The front end installs a gate claiming exactly the rows that name a mission. Setup and party come
from the same pair the process's own door already uses, so a screen reached from the list and one
reached at startup are one screen. A fresh model per call.

**Fences:** the loader's mission arm keeps working for a caller with no gate installed.

**Done when:** a mission row opens generation, a loose map row does not, and an abandoned spread
does not follow the player to the next row he picks.

---

## T3 — `impl` — the separate generation flag is retired (`cmd/againrom`)

**Files:** `cmd/againrom/main.go`, and its `_test.go` sibling.

**Covers:** FR-2, FR-2a; D-3.

The flag stays accepted and inert, bound to a discard so no statement can read it. The mission flag
opens generation unconditionally; the headless check reports it unconditionally.

**Done when:** an invocation naming the flag parses and runs; a non-positive mission number is
refused before any window; no statement reads the flag.

---

## T4 — `impl` — the Derived block states what a character will have (`pkg/ui`, `pkg/data`)

**Files:** `pkg/ui/chargen.go`, `pkg/data/unitdef.go`, and their `_test.go` siblings.

**Covers:** FR-3, FR-3a, FR-3b; D-4, D-5.

Sight, the five-column protection family and every skill slot, all read from the mint the character
will be built from. The line budget is a function the screen exports and the painter clips to.

**Fences:** re-derive nothing. Name no column that carries no published title.

**Done when:** the block states all three; exactly one skill slot is nonzero on a fresh character
and that is asserted against the record, not assumed; a block over budget is cut rather than drawn
over the message line.

---

## T5 — `impl` — the successor never regenerates, and the block shows the right array (`pkg/ui`, `pkg/game`)

**Files:** `pkg/ui/chargen.go`, `pkg/game/frontend.go`, `pkg/data/unitdef.go`, and their `_test.go`
siblings.

**Covers:** FR-1a, FR-3; D-2, D-5.

Two corrections. The advance that follows a won mission reaches the map through a path that never
asks the gate — verified rather than assumed, and the gate's field renamed for what it is. And the
block was printing the weapon-kind five, which are zeros for a human; it now prints the elemental
five, named, in their own column order.

**Done when:** a gate claiming every row still advances a won mission straight to its successor;
the block's protection row moves when Spirit is spent.

---

## T6 — `impl` — the unit panel moves and takes the owner's arrangement (`pkg/ui`, `pkg/game`, `pkg/mapload`)

**Files:** `pkg/ui/panel.go`, `pkg/game/panelchars.go`, `pkg/game/world.go`,
`pkg/mapload/sheet.go`, and their `_test.go` siblings.

**Covers:** FR-4, FR-4a; D-6, D-7.

The panel stands bottom right and states its subject in the ruled order. Two five-wide families
become ten labelled rows plus four heading fields, each heading carrying the gate of its own block.
Sight crosses on the map sheet.

**Fences:** drop no row that carries something nothing else prints.

**Done when:** the rows appear in the ruled order; a heading is present exactly when its block is;
the one dropped family is shown to duplicate what the skill column already prints.

---

## T7 — `impl` — the minimap shows the view and takes a press (`pkg/ui`)

**Files:** `pkg/ui/minimap.go`, `pkg/ui/command.go`, and their `_test.go` siblings.

**Covers:** FR-6, FR-7; D-9, D-10.

An outline of the camera's own visible extent, never under a pixel, drawn last so a unit cannot
erase its corner. A press centres the view and holding drags it. One scale rule, read by the picture
and by the hit test.

**Fences:** the two tests enforcing the box's inertness are replaced by their negations, not
deleted.

**Done when:** every terrain pixel names a cell inside the grid in both the scaled and the sampled
branch; a pixel inside the box but outside the terrain names none and still reaches no map; a cursor
leaving the picture mid-drag centres nothing.

---

## T8 — `impl` — one constant is the right-hand column (`pkg/ui`)

**Files:** `pkg/ui/panel.go`, `pkg/ui/minimap.go`, and their `_test.go` siblings.

**Covers:** FR-5, FR-5a; D-8, D-10.

The panel pins its width to the constant and the minimap is a square of it, so neither is a function
of the selection. The terrain is centred in the square both ways. The box is additionally bounded by
the view and by half of it.

**Done when:** the box is that constant and square with a unit selected and with none; the whole
suite passes at the harness's own window size.

---

## T9 — `impl` — the spellbook and the pack become bars (`pkg/ui`, `pkg/game`)

**Files:** `pkg/ui/hud.go` (new), `pkg/ui/inventory.go`, `pkg/ui/spellbook.go`,
`pkg/ui/command.go`, `pkg/ui/viewer.go`, `pkg/ui/panel.go`, `pkg/game/inventory.go`, and their
`_test.go` siblings.

**Covers:** FR-4b, FR-8, FR-8a, FR-8b, FR-11; D-11, D-12.

One file owns the arithmetic of every box drawn over the map. The book is two rows filled column by
column; the pack is one row of larger cells that scrolls by one a press. The carrier's cap goes: it
was a limit of the window and the window now scrolls. The panel starts taking its own presses.

**Fences:** compose no icon in the drawing tier; add no second placement expression beside the
shared one.

**Done when:** a press resolves to the container's index at every scroll position; the cap is gone
rather than raised; a press anywhere on the panel reaches no order.

---

## T10 — `impl` — the game opens at the size of the screen (`pkg/ui`, `cmd/againrom`)

**Files:** `pkg/ui/app.go`, `cmd/againrom/main.go`, and their `_test.go` siblings.

**Covers:** FR-12, FR-12a; D-16.

The startup size is a pure function of the reported monitor size, so both answers are testable
without opening a window. Undecorated, because the engine sizes the inside of a window.

**Done when:** a reported size is taken whole and undecorated; a non-positive or absent one falls
back to the authored size, decorated; neither answer is ever non-positive.

---

## T11 — `impl` — four display switches, and the column carries the doll and the worn set (`pkg/ui`)

**Files:** `pkg/ui/hudtoggles.go` (new), `pkg/ui/hud.go`, `pkg/ui/inventory.go`, `pkg/ui/panel.go`,
`pkg/ui/minimap.go`, `pkg/ui/app.go`, `pkg/ui/command.go`, `pkg/ui/viewer.go`, and their `_test.go`
siblings.

**Covers:** FR-9, FR-9a, FR-10, FR-10a, FR-10b, FR-10c, FR-11; D-13, D-14, D-15, D-17.

The switches are stored inverted so shown is the zero value. The doll and the worn set move into the
column, worn above doll above the panel, and the column narrows to make the room. The doll follows
the selection and takes its rebuild key from its source. What refuses in a short window is the two
boxes, never the control panel.

**Fences:** invent no archive address for a unit this build cannot compose a figure for. Do not take
a letter off the camera.

**Done when:** every switch defaults on and moves whatever is selected; a box is drawn only with its
switch on and something to show; the doll is drawn for a selected non-party unit; a press on a
button flips one switch and reaches no map; the minimap reads no fitted height at all.

---

## T12 — `impl` — the spellbook's own switch reaches its own bar (`pkg/ui`)

**Files:** `pkg/ui/spellbook.go`, `pkg/ui/inventory_test.go`.

**Covers:** FR-10b, FR-11.

One of the four boxes was not asking its switch, so its button flipped a flag nothing read. The
statement goes where the other three already have it.

**Done when:** all four switches are exercised against one fixture on which all four boxes are
drawn, a whole row asserted per switch, and reverting the statement reddens it.

---

## Traceability

| Task | Commit | Covers |
|---|---|---|
| T1 | the picker gate | FR-1, FR-1b |
| T2 | the rows that claim it | FR-1 |
| T3 | the retired flag | FR-2, FR-2a |
| T4 | the Derived block | FR-3, FR-3a, FR-3b |
| T5 | the successor and the right array | FR-1a, FR-3 |
| T6 | the unit panel | FR-4, FR-4a |
| T7 | the minimap's view and press | FR-6, FR-7 |
| T8 | the fixed column | FR-5, FR-5a |
| T9 | the two bars | FR-4b, FR-8, FR-8a, FR-8b, FR-11 |
| T10 | the window | FR-12, FR-12a |
| T11 | the switches and the column | FR-9, FR-9a, FR-10, FR-10a, FR-10b, FR-10c |
| T12 | the switch that reached nothing | FR-10b |
