# spec — 0158 in-game menu

The in-game menu is a popup. Escape raises it over a mission or over the town square, the world
stops while it is up, the surface behind it is darkened and stays drawn, and Escape or the return
row closes it and play continues.

Before this story the same key raised a four-row screen that replaced the surface behind it. The
surface, the row list, the geometry, the accelerators and the freeze are what this story adds.

## Terms

- **surface** — which of the two panels is up. There are exactly two: the **mission** surface and the
  **town** surface. Which one opens is decided by the screen Escape was pressed on.
- **row** — one entry of a surface: a label, an accelerator, an action, and whether it is enabled.
- **panel rect** and **row rect** — rectangles in the fixed 640x480 frame the whole front-end
  composes in.
- **the surface behind** — the map screen or the town screen the menu was opened over.

## Functional requirements

**FR-1** Escape on the map screen raises the mission surface. Escape at the town square raises the
town surface. In both cases the surface behind stays drawn and is not replaced.

**FR-2** While the menu is up no input reaches the surface behind it, and the world is stopped. The
three cadence keys, the camera, the selection, orders and the debug keys all do nothing.

**FR-3** While the menu is up the surface behind it is darkened, at the same value this project
already applies behind a dialogue notice.

**FR-4** Escape closes the menu, and so does the return row. The surface behind resumes where it
was. No world time that passed while the menu was up is applied after it closes, and no ambient
animation jumps.

**FR-5** The mission surface lists seven rows in this order: save, load, game options, sound
options, quest objectives, end quest, return to game. The town surface lists five in this order:
save, load, sound options, abort game, return to game.

**FR-6** The mission panel rect is `(100,60)-(440,400)`. The town panel rect is
`(100,100)-(440,340)`. Row `n`, counting from zero, is the panel-local rect
`(40, 40+30n)-(width-48, 40+30(n+1))`, so every row is 252 wide and 30 high and the first row's top
is 40 below the panel's.

**FR-7** A row's accelerator is the first character preceded by a single `~` in its label, lowercased.
`~~` is an escape: it contributes one literal `~` to the drawn text and no accelerator. A label with
no `~` uses the row's own fallback letter instead. The drawn text never contains a `~` that came
from the accelerator mark. Pressing a row's accelerator chooses that row, whatever case it is typed
in.

**FR-8** A disabled row is listed, is visibly distinct from an enabled one, and cannot be chosen —
not by Enter, not by its accelerator, and not by a pointer release on it.

**FR-9** Up and Down move the focused row. A pointer release inside a row's rect focuses that row and
chooses it. A pointer release anywhere else, inside the panel or outside it, does nothing.

**FR-10** Save writes through this build's save seam and reports the name or the refusal on the
menu's own message line. Load opens this build's load window. End quest leaves the mission and
releases its world. Abort game leaves the town for the main menu. Return closes the menu. Game
options, sound options and quest objectives are listed disabled.

**FR-11** Save is enabled when this build has a save seam. Load is enabled when the save store lists
at least one save. Both are disabled otherwise, and a disabled row is refused rather than reporting
a failure after the fact.

## Acceptance criteria

**AC-1** Escape on the map screen raises the menu, leaves the viewer adopted, and does not exit.

**AC-2** Escape at the town square raises the menu with the town surface up.

**AC-3** The front-end's popup answer is true while the menu is up over a map and false once it is
closed, so the dim, the camera, the ambient clock and the input gate all read one answer.

**AC-4** The far side is told the world is stopped on the first frame after the menu opens, and told
it is running again on the first frame after it closes.

**AC-5** The map advance and the viewer step run on every frame the menu is up, so no pacing span is
banked while it stands.

**AC-6** The mission surface is exactly seven rows in FR-5's order.

**AC-7** The town surface is exactly five rows in FR-5's order.

**AC-8** The two panel rects are FR-6's, and row rects are 252x30 with row 0's top 40 below the
panel's.

**AC-9** The mission accelerators are `s l o n q e r` and the town's are `s l n e r`. The quest
objectives row resolves `q` from its label and not `m` from its fallback. The abort game row carries
no `~` and resolves `e` from its fallback.

**AC-10** A label containing `~~` draws one `~` and takes its accelerator from its fallback.

**AC-11** A disabled row does nothing on Enter, on its accelerator, or on a pointer release on it.

**AC-12** Escape on the menu returns to the surface behind it, and so does the return row.

**AC-13** End quest leaves the mission for the map list and releases the world. Abort game leaves the
town for the main menu.

**AC-14** A pointer release outside the panel changes no row, chooses nothing and reaches nothing
behind the menu.

**AC-15** No drawn row text contains a `~` from the accelerator mark.

## Properties

**P-1** No menu state reaches the simulation. This story names no simulation type, adds no
serialized field and does not move the byte-form version.

**P-2** The menu composes with no game install present: every label, colour and rectangle it needs is
authored in this repository or is a decoded number.

**P-3** The panel draws only inside its own panel rect.

## Authored, and what each one costs the player

Where the decode has no answer, this section states the answer taken and what a player sees. The
ids are AU-* and not SC-*: SC belongs to `plan.md`.

**AU-1 The panel frame is this project's own.** `MENU-ART-014` decodes the frame as a nine-patch out
of `graphics\interface\lm.256`, and grades **Unknown** how the tiling covers the 44-px band the
truncating tile counts leave inside the right and bottom edges. This build draws a filled rectangle
with a one-pixel border, in the same palette as this project's dialogue box. **What the player sees:**
the in-game menu is a plain dark box in the game's own palette, not the original's carved frame. The
panel rect, the row rects and the row pitch are the decoded ones, so the layout is unchanged when
the frame art is added.

**AU-2 The row labels are authored English.** The original reads its labels from
`main\text\dialogs.txt` by index. `MENU-ITEM-011` cites the indices; the descriptor base that turns
an index into a line of that file is not decoded, so the mapping cannot be derived and is not
guessed. **What the player sees:** the menu reads in English on a Russian install, and the
accelerators are the English ones. `MENU-KEY-013` establishes that the Russian release marks
different letters, so this is a real difference and not a translation gap. The accelerator *rule* is
the decoded one — the letter after the `~`, not a hard-coded key — so supplying Russian labels moves
the accelerators with them and changes no code.

**AU-3 The diplomacy row is not built.** `MENU-ITEM-011` decodes the mission surface as eight rows of
which seven show, with load and diplomacy exclusive on the word `campaign+0x6bc`. That word's meaning
is not decoded. This build always shows load. **What the player sees:** the mission menu never offers
diplomacy. Nothing in this build could act on it.

**AU-4 Three rows are listed and disabled.** Game options, sound options and quest objectives have no
screen behind them in this build. **What the player sees:** three rows in the middle of the mission
menu, and one in the town menu, that are visibly greyed and do nothing when picked. They are listed
rather than omitted because the decoded surface is seven rows and five rows, and a shorter list would
be a different surface. `MENU-ITEM-012` decodes every town row as built **enabled**, so the town's
sound options row is a disclosed divergence and not a decoded state.

**AU-5 End quest and abort game act at once.** `MENU-ITEM-012` decodes `0x41c` as raising a
confirmation panel — five rows in a mission, three in the town — and grades the exit rows' target
**Unknown**. Neither confirmation panel is built. **What the player sees:** picking end quest drops
him straight out of the mission to the map list, and abort game straight out of the town to the main
menu, with nothing asking whether he meant it. A save taken from the same menu is the only way back.

**AU-6 A click outside the panel does nothing.** `MENU-INPUT-016` (corrected) establishes that the
panel is the root's capture object while it is up and that a click outside it reaches no other child
of the root; what the panel's own handlers do beyond the slots read is Unknown. This build matches
the reach and adds no effect of its own. **What the player sees:** clicking the map or the town
behind the menu neither moves the camera, nor selects, nor orders, nor enters a room. He must close
the menu first.

**AU-7 Save and load are gated on this build's own store.** The decoded predicates are the undecoded
mode word and a `FindFirstFileA` over `game*.sav`. This build asks its own save seam instead: save is
enabled when a store exists, load when that store lists at least one save. **What the player sees:**
in a build with no store both rows are greyed, and in a build with a store load stays greyed until
the first save is written.

## Out of scope

The confirmation panels, the options screens, the quest objectives screen, diplomacy, the nine-patch
frame art, the label file and the Russian accelerators. Each is named above with what it costs.
