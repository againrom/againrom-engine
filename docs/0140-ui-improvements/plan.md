# 0140 — plan

## Strategy

Seventeen requirements over five screens, delivered as **twelve slices in ruling order** rather than
in dependency order, because the rulings arrived over three days and each was played before the next
was given. Three of the twelve exist only because an earlier one was wrong or incomplete when the
owner saw it running; that is recorded rather than folded away, since the sequence is the evidence
that each shape was checked against the running game.

The work concentrates in `pkg/ui`. `pkg/game` moves only where a picture has to be composed from
the archive, `pkg/data` and `pkg/mapload` only where a number has to reach a screen that could not
see it. Nothing below `pkg/sim` is opened at all.

## Decisions

**D-1 — the generation gate is asked in `pkg/ui`'s own vocabulary, in one place.** A row index in, a
model and a begin callback out. The wiring tier decides *which* rows want generation, so the drawing
tier never names a mission, a map or anything from `pkg/data`. One assignment of the armed screen,
so both doors — the process's own and a picker row's — inherit one nil refusal.

**D-2 — the successor is a different door, and is named as one.** FR-1a is not a condition inside
the gate; it is the fact that a won mission's advance reaches the map through a path that never asks
the gate. The gate's field is named for what it is (a *new game* gate), so the next reader cannot
mistake it for "every mission entry".

**D-3 — the retired flag is bound to a throwaway.** FR-2 keeps the flag accepted and inert. Deleting
it would turn a working invocation into a parse error; leaving it bound to a live variable would
leave a statement able to read it. Bound to a discard, neither is possible.

**D-4 — the Derived block reads the mint and measures its own budget.** Every number FR-3 adds is
already on the derived record for the character the screen will build; none is recomputed. The line
budget of FR-3b is a function the screen exports and the painter clips to, so "what fits" is
measured once instead of asserted in a comment.

**D-5 — the protection five are labelled and the weapon-kind five are not.** FR-3 and FR-3a differ
because the evidence differs: the protection column order is published and the five names are ours
to choose; five of the ten columns carry no published title at all, so that family is drawn as one
unnamed row.

**D-6 — the panel becomes ten labelled rows and four headings.** FR-4's arrangement needs one field
per drawn number, not one per five-wide family. A heading is a **field**, not a label attached to
the row under it (FR-4a): rows are dropped when nothing in them resolves, and a heading spelled onto
a droppable row would move when that row's gate closed.

**D-7 — Sight crosses on the map sheet.** The panel cannot derive it: for a person it is the derived
graph's own value and for a creature it is the units collection's scan-range column, and only the
loader sees both. That the scan-range column *is* a sheet's sight is ours and says so at the field.

**D-8 — one constant is the right-hand column, and every box in it reads that constant.** FR-5's
whole content. The minimap is a square of it, the panel pins its width to it, and the doll, the worn
set and the control panel are all that wide. The price is a clip: a panel row wider than the column
is cut by the font's own bounds rather than widening the box. Measured, not guessed — see R-1.

**D-9 — one scale rule, read by the picture and by the hit test.** Two copies that drifted by one
would centre the camera on a cell next to the one pressed and nothing would look wrong.

**D-10 — the minimap is bounded by the view and by half of it.** FR-5a is not "as large as the
constant" but "no larger than the constant"; a box that overflows a small window is a press the map
does not get across its whole width.

**D-11 — one file owns the arithmetic of every box drawn over the map.** They stack, and each needs
to know where the others stand. Two files computing "the bottom band" separately is the pair that
drifts, and a drifted click surface is a press that lands on nothing.

**D-12 — the pack's carrier becomes a slice, and the doll's cache key becomes a counter.** FR-8a
removes the reason for a fixed cell count; the container it draws was always unbounded. Losing a
fixed-length array costs the subject its comparability, so the box cached by comparing it is keyed
by a revision the setter bumps instead.

**D-13 — the switches are stored inverted, so shown is the zero value.** FR-10's "all four start on"
then needs no constructor statement anybody could forget, and every hand-built fixture in the tree
starts where the running game does.

**D-14 — the doll's rebuild key is its source and not the subject.** FR-9a makes this box follow the
selection, and a selected unit's drawn frame changes as it walks; a key bumped by the subject setter
would hold the doll of a unit selected two units ago.

**D-15 — the column reserves the minimap's square, and refusal runs downward.** FR-10c needs a place
that no selection and no fitted height can move, so the control panel hangs under a *reserved*
square rather than under the drawn one. What gives in a window too short for the whole column is the
doll and the worn set, which refuse rather than climb over the control panel.

**D-16 — the startup size is a pure function of the reported monitor size.** FR-12 and FR-12a are
then both testable without the one thing a `pkg/ui` test can never open. Undecorated is load-bearing
and not cosmetic: the engine sizes the *inside* of a window, so a title bar on a screen-sized one
pushes the bottom of the game under the bottom of the display.

**D-17 — two of the four switches get no key.** The owner named the four buttons by their letters
and two of those letters already pan the camera. Binding them would give one press two meanings;
taking them off the camera would remove a binding he did not ask to lose. Disclosed, not resolved.

## Files

| File | What moves |
|---|---|
| `pkg/ui/flow.go`, `pkg/ui/app.go` | the generation gate and its one assignment; the window the game opens at; the two switch keys |
| `pkg/ui/chargen.go` | the Derived block's rows and its measured line budget |
| `pkg/ui/panel.go` | the corner, the row order, the headings, the column constant, the panel's own press |
| `pkg/ui/minimap.go` | the square, the view outline, the press and the drag, the scale rule |
| `pkg/ui/hud.go` | new — the arithmetic of every box drawn over the map |
| `pkg/ui/hudtoggles.go` | new — the four switches, their geometry, their picture and their press |
| `pkg/ui/inventory.go` | the doll and the worn set as two boxes; the pack bar and its scroll |
| `pkg/ui/spellbook.go` | the two-row bar; its own switch |
| `pkg/ui/command.go`, `pkg/ui/viewer.go` | which box takes which press; the paint order and the caches |
| `pkg/game/frontend.go`, `pkg/game/world.go` | the gate's installation; the panel's fields |
| `pkg/game/inventory.go`, `pkg/game/panelchars.go` | the pack composed without a cap; the panel's characters |
| `pkg/data/unitdef.go`, `pkg/mapload/sheet.go` | the protection names; Sight on the sheet |
| `cmd/againrom/main.go` | the retired flag; what the headless check reports |

## Risks

**R-1 — the column constant clips a real panel row.** A developer run over two missions against both
roots measures the fullest party hero at 387 pixels and a placed creature at 288. At 400 neither
clipped; at the owner's 300 the hero sheet's widest rows do. Accepted on his ruling, measured rather
than assumed, and written at the constant because it is the one thing a reader of it cannot see.

**R-2 — an opaque box too large for its window swallows the map.** This is not hypothetical: it
landed once, as twenty-eight test failures across nine files, all one cause. Mitigated by D-10 and
by the refusal rule of D-15, and watched by SC-4.

**R-3 — a monitor-sized window resamples the menu.** The 640x480 menu frame scales by an exact
rational, so a screen-sized window is only pixel-exact where the height divides evenly. Disclosed;
no mitigation, because the alternative is a window smaller than the ruling.

**R-4 — a switch that reaches no box.** Four boxes each asking their own switch is four places to
forget one, and one was forgotten. Watched by AC-14, which walks all four against one fixture.

## Success criteria

**SC-1** — the full local gate is green on the committed tree: build, vet, `gofmt`, the whole test
suite, and the four repository scripts.

**SC-2** — the shipped binary answers the headless check against **both** lawful roots, and the two
answers differ only where the installs differ.

**SC-3** — every box's rectangle is measured at three window sizes and recorded, including which
boxes refuse at which size.

**SC-4** — the whole suite passes at the harness's own window size, which is the size R-2 broke at.

## Traceability

| FR | Decisions | Criteria |
|---|---|---|
| FR-1, FR-1a, FR-1b | D-1, D-2 | AC-1, AC-2, AC-3 |
| FR-2, FR-2a | D-3 | AC-4 |
| FR-3, FR-3a, FR-3b | D-4, D-5 | AC-5, AC-6, AC-7 |
| FR-4, FR-4a | D-6, D-7 | AC-8 |
| FR-4b | D-11 | AC-9 |
| FR-5, FR-5a | D-8, D-10 | AC-10, SC-3 |
| FR-6 | D-9 | AC-11 |
| FR-7 | D-9, D-10 | AC-11 |
| FR-8 | D-11 | SC-3 |
| FR-8a, FR-8b | D-12 | AC-12 |
| FR-9 | D-11, D-15 | SC-3 |
| FR-9a | D-14 | AC-13 |
| FR-10, FR-10a | D-13 | AC-14, AC-16 |
| FR-10b | D-13 | AC-13, AC-14 |
| FR-10c | D-15 | AC-15, SC-3 |
| FR-11 | D-11 | AC-9, AC-14, AC-15 |
| FR-12, FR-12a | D-16, D-17 | AC-17, AC-18 |
