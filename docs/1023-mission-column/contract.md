# Story 1023 — the mission screen's right column at its decoded geometry

## Result

The mission screen's right-hand column is 160 pixels wide, not 300, and its vertical division is the
one the executable's own construction site gives. The character panel in its lower slot is the same
160x242 panel the shop, school, tavern and character generator draw, which is what the original does
with that object. The map viewport is the screen minus that column.

This story exists because of an owner ruling: *«давай все приводить к оригиналу»* (2026-08-21), given
when the authored width was put beside the decoded one. It overrides his own 2026-08-11 ruling that
set the constant to 300, and that earlier ruling's reasoning stays readable in the constant's own
comment — it is a record of what was believed then, not a rule to satisfy now.

## What is decoded

**The column is 160 wide.** `SESS-VIEW-028` gives the map view's construction rect as
`(0, 0, screenW − 0xa0, screenH)` — the screen minus a 160-pixel right strip — from
`L09484`…`L09485`, at the only construction site in the image. Grade **High** for the viewport,
its cell spans and the two-writer enumeration behind them; every cited address is re-read from both
roots by `tools/mapview -mode anchor`.

**The column is divided 158 / 80 / 242.** `SHOP-FIGURE-041`: the main-frame builder stacks four
widgets in that column, ids 5, 6, 7 and 8, at `(0,0,160,158)`, `(0,158,160,238)`, `(0,238,160,480)`
and `(0,480,160,screenH)`. At the shipped default of 640x480 the fourth has no height. Grade **High**
for the six immediates at a construction site a whole-image `callto:` shows is the only one, with the
arithmetic re-derived from raw PE bytes by `tools/shopfigure -mode rects`, which shares no code with
the disassembler.

**Id 7 is the character panel, and it is one object.** `SHOP-FIGURE-041`: `campaign+0xe0` is built
once as `ctor(id 7, 0, 238, 160, 480)`; a visiting screen takes it, offsets its rect by `640 − width`
through `OffsetRect`, re-sets it and adds it as its own child, and the deactivation is the exact
inverse. Grade **High**. This is the same object `1021` and `1022` place at `(480,238)-(640,480)` on
four other screens.

## What is not decoded

**Which widget is which.** `SESS-VIEW-028` is explicitly **Medium** for which panel is child 2 and
which is child 3: the ids are read and the child class is not. `SHOP-FIGURE-041` states that only id
7 is named by any instruction in the shop routines. So the three slots are decoded and the tenants of
two of them are not.

**Whether the four literal rects are absolute or relative to a parent.** `SHOP-FIGURE-041`'s rects
have `x` in `[0,160)` and the claim describes them as the right strip, while the shop's borrow
applies an explicit `+480` offset to reach the right edge. `TOWN-232` establishes for a different
object that a child's literal LTRB is not necessarily absolute and the ancestor chain must be walked.
**This does not block the story**: `SESS-VIEW-028` establishes independently, at High, that the
strip is 160 wide and on the right, and the 158/80/242 division is a vertical split that no
horizontal offset changes. A research lane opened 2026-08-21 asks the general form of this question
for a different object on a different screen, and `pipeline/RESEARCH-LOG.md` carries its brief; if a
claim publishes a rule, apply it and close the row. No experiment id is cited here on purpose: an
experiment is immutable history that cannot announce a later amendment, and this one is not in the
pin at all.

**The figure widget's own interactive controls.** No claim gives them: not their population, not
their rects, not the routine each one invokes. `SESS-VIEW-028`'s Medium covers which child class is
which widget, and `SHOP-FIGURE-041` names only id 7. The owner stated on 2026-08-21 that the card
carries controls at its four corners and gave a corner-to-function mapping; under B3 that is
testimony and is queued as a research question, not built. `DIV-217` carries the row. This story
places no control on the figure.

**Everything about the HUD's own contents.** `pkg/ui/hud.go`'s own header says it: nothing published
states a bar's pixel size, cell count, colours, scroll step, or which letters label a toggle, and
every number in that file is this project's own. The ruling moves the column's geometry to the
decoded value. It does not decode the boxes inside it, and this story must not claim it did.

## What this build does today

`sidebarWidth = 300` (`pkg/ui/panel.go:654`), with five readers across `pkg/ui/hud.go`,
`pkg/ui/inventory.go`, `pkg/ui/minimap.go` and `pkg/ui/panel.go`. The column carries five boxes
(`hud.go:24-29`): the minimap at the top, the control panel under it, and from the bottom edge up the
unit panel, the doll and the worn set. The minimap is square at `sidebarWidth` across
(`minimap.go:599`). The doll and worn boxes are `sidebarWidth` across (`inventory.go:302-319`).

`panel.go:654`'s own comment records the cost of the 2026-08-11 move from 400 to 300, measured with
`cmd/paneldump` over missions 10 and 20 on both installs: the fullest party hero composes at 387
pixels and a placed creature at 288, so 400 cleared the widest panel and 300 does not. **At 160 that
cost grows and the contract accepts it**, on the owner's ruling and on his standing rule for this
class of box: truncation is acceptable, the main information must fit.

## Behaviours

**B1 — the column is 160 wide.** `sidebarWidth` becomes 160 and every reader follows. Nothing in the
column may be wider than the column.

**B2 — the map viewport is the screen minus the column.** `SESS-VIEW-028` gives
`(0,0,screenW−0xa0,screenH)`, which is `(0,0)-(480,480)` at the shipped default, and 15 columns by 15
rows of 32-pixel cells. **First establish what this build does now** — whether the viewport reserves
the column or the HUD draws over a full-screen map — state it in `spec.md`, and make it match. Do not
take this contract's wording as a premise about the current code; it is not one, and no measurement
of the current viewport is handed to you here.

**B3 — the column's vertical division is 158 / 80 / 242, and the character panel occupies the 242
slot.** That slot is `SHOP-FIGURE-041`'s id 7 at `(480,238)-(640,480)`. The panel drawn there is the
same 160x242 card `1022` builds for the generator's lower-left, composed once and drawn on five
screens. Reuse it; do not compose a second one.

**B4 — the minimap and the control panel take the 158 and 80 slots, and that assignment is
authored.** 160x158 is near-square and 160x80 is a strip, which is consistent with a map above a row
of switches, and consistent is not decoded: research is Medium on widget identity and names neither.
Record the assignment as a typed divergence row citing `SESS-VIEW-028`'s own Medium grade. The
minimap stays square and its reserved square stays the anchor the control panel is pinned to
(`hud.go:44-48`), which is an owner-facing property this story does not change.

**B5 — the doll and the worn set have no decoded slot, and the story says so.** The decoded column
has three slots at 640x480 and this build puts five boxes in it. Either fold the doll and the worn
set into the 242 slot as alternates of the character panel, or place them by authored choice. Either
way it is a typed divergence row naming what the decoded structure has no room for, not a silent
placement. **State which you chose and why in `spec.md`.**

## Domains

Client (`pkg/ui`). One domain. Five behaviours, at the split test's ceiling and not above it.

## Ceiling

**Three passes.** One domain, no reach into hashed simulation state.

**`1023` opens when `1022` lands**, because B3 reuses the 160-wide card `1022` builds and that card
does not exist before it. It is not parallel with `1022`.

## What is authored and what is not

Decoded, cited: the column's width (`SESS-VIEW-028`, High); its 158/80/242 division
(`SHOP-FIGURE-041`, High); id 7's identity as the shared character panel and the transfer mechanism
(`SHOP-FIGURE-041`, High); the viewport's rect and its 15x15 cell span (`SESS-VIEW-028`, High).

Authored, each owing a typed row in `docs/DIVERGENCES.md`: which box occupies the 158 slot and which
the 80 slot; where the doll and the worn set go; every dimension, colour, label and step inside every
box, which were authored before this story and stay authored after it; the card's own truncation at
160 where a row does not fit.

**One row is retired rather than opened.** `panel.go:654`'s comment is the record of the 2026-08-11
ruling and its measured cost. Do not delete it and do not rewrite it into agreement with this story:
a published rationale is a record of what was believed when it was written. Add the new ruling below
it with its own date, and let the reader see both.

`DIV-199`..`DIV-206` are reserved to this story in `PIPELINE-STATUS.md`. Ids not spent are returned
at the landing and recorded as returned.

## Amendment, 2026-08-21: the owner's canonical column, measured

The owner supplied a screenshot of the mission right column from his lawful install, with
*"the right edge in the game should have such controls"*. It is preserved at
`pipeline/archive/owner-ruling-2026-08-21-stats-card/canon-mission-right-column.png`. **Read that
image; do not work from this description of it.** A description is a selection, and a selection drops
the detail the reader needed.

His images are evidence from a lawful install. They are not owner testimony, so `B3` does not apply
to them, and they are not research: research remains the sole authority on what ROM1 does, and any
disagreement between an image and a published claim is a question for research rather than a
correction made here.

### What was measured

The file is **185 x 769 pixels**. A column 160 wide is 480 tall at 640x480, 600 at 800x600 and 768
at 1024x768. A 769-row image is a full-height capture of a **1024x768** screen; it cannot be a
640x480 one. The extra 25 pixels of width are map to the left of the column.

Two instruments were run over the image, with different failure modes. The first ranked rows by the
step in mean brightness; in the text-dense lower half it returned rows at one-line spacing, because
text baselines are brightness steps and are not seams. The second smooths each row's mean colour
vector over a 9-row window and ranks the change across a gap, which averages text out and keeps a
change of background palette. Only the second is used below. It is blind to a seam thinner than its
window, which the first is not, so neither alone is evidence and the agreement below is with the
decoded rects rather than between the two instruments.

`SHOP-FIGURE-041` gives four stacked widgets at `(0,0,160,158)`, `(0,158,160,238)`,
`(0,238,160,480)` and `(0,480,160,screenH)`. At 1024x768 the internal seams fall at y = 158, 238 and
480. The measured palette changes nearest those three rows are **150, 230 and 486**, within 8 rows
of each on a 769-row image.

| decoded slot | id | height at 1024x768 | what the image shows there |
|---|---|---|---|
| `(0,0,160,158)` | 5 | 158 | the minimap, in an ornate frame, with the viewport rectangle drawn on it |
| `(0,158,160,238)` | 6 | 80 | a command icon grid, two rows of five cells |
| `(0,238,160,480)` | 7 | 242 | a full-body figure in an oval frame |
| `(0,480,160,768)` | 8 | 288 | an ornamented horizontal strip, then the statistics card |

### What this does not establish

**Widget identity is still Medium and this measurement does not raise it.** `SESS-VIEW-028` is
explicitly Medium on which panel is which child, and an image cannot answer it: agreement between a
region's appearance and a rect's position is consistent with the assignment above and with others.
What the measurement establishes is that four regions exist and where their boundaries fall.

**Whether slot 8 holds a second instance of the id-7 panel class, or a different object, is not
answerable from an image.** `SHOP-FIGURE-041` establishes that id 7 is one object the visiting
screens borrow, and this tree already draws that 160x242 slot with two presentations: a figure
background when it shows the character, and a text background when it shows statistics. The image
shows both presentations at once, in two different slots. That is a question for research.

**`B1` binds anyone briefing research on it.** Do not hand over the table above, the slot-to-content
assignment, or the measured seam rows. Ask which child class each of the four ids is constructed
with, and let the answer arrive without our expectation attached.

### The card's content is the same card

The statistics card in slot 8 carries the same rows, in the same order, as the character-generation
card in `canon-card-danath-named.png`: the four statistics with health and mana beside them, a
damage-and-absorb row, an attack-and-defense row, five skills beside five resistances, then weight,
experience, sight and speed. The owner's ruling that this window appears wherever statistics are
shown is therefore consistent with the original rather than a divergence from it.

### The conflict this exposed, and the owner's ruling on it

**At 640x480 the fourth slot has zero height.** `(0,480,160,480)` is empty, and this tree ships a
640x480 virtual frame (`pkg/ui/shopscreen.go:32`). So the decoded column at the shipped resolution
has three slots, and the canonical column the owner showed has four. The statistics card he is
pointing at occupies the slot that does not exist at 640x480.

Two of his own rulings meet here and point different ways. *"Bring everything to the original"*
(2026-08-21) makes the decoded division authoritative, and at 640x480 that division has no room for
the card. *"This window must be everywhere we show anyone's statistics, anywhere"* (2026-08-21, same
day) requires the card in the mission column.

Three ways out were put to the owner: ship 640x480 without the card in the column,
support 1024x768 for the mission screen, or author the slot at 640x480 as a typed `DEVIATION`.

**He chose 1024x768** (2026-08-21):

> «надо поддержать 1024 вне города»

and then, on his own, the mechanism and the reason:

> «в городе псевдорежим 640х480 ... а в игре самой хочется иметь 1024х768 хотя бы псевдо»

Full text: `pipeline/archive/owner-ruling-2026-08-21-resolution.md`. Three things follow.

**The mission composes at a virtual 1024x768 frame, scaled to the window.** *Pseudo* is his own word
and the mechanism already ships for the town family: `pkg/render/frame` composes at a fixed virtual
size and fits it into the window at one exact rational scale, letterboxed. He is not asking the game
to run at the desktop's own resolution.

**The town family keeps its virtual 640x480**, and he gives the reason: the original nails that
composition down, so it is reproduced rather than re-laid-out.

**His ground for 1024x768 is fidelity, not preference.** The original's own composition is designed
for it, which is what `SESS-VIEW-028` measures: the viewport is the screen minus a 160-pixel right
strip, giving 27 columns by 24 rows at 1024x768 against 15 by 15 at 640x480. At that frame the right
strip is 160 by 768, the fourth slot has real height, and the statistics card has somewhere to sit.

**800x600 is neither asked for nor excluded.** He named 1024 and named 640 as the town's. The middle
arm is decoded and stays out of scope: it is not built on his silence.

**This story does not build the frame.** That is story `1026` — the virtual frame, the viewport
span and input mapping through it — and this story builds the column's contents on it. The split
is because one contract carrying both would name more than five behaviours, which this file's own
Ceiling section forbids without recording why. **`1023` opens when `1026` lands.**

## Amendment, 2026-08-21: no stats/doll toggle in the mission column

**Owner directive.** Full text: `pipeline/archive/owner-ruling-2026-08-21-card-corner-controls.md`.
The stats/doll toggle is built for the town and is not needed in a mission. His stated reason is that
it would be needed at 640.

**That follows from the resolution ruling of the same day.** At 640x480 the right strip is 160 by 480
and the figure and the statistics card cannot both occupy it, so one must toggle the other away. At
the virtual 1024x768 frame story `1026` builds, the strip is 160 by 768 and both fit. The toggle is a
consequence of the smaller frame, not a feature of the column.

**What this story builds because of it.** The mission column presents the figure and the statistics
card together. It carries no stats/doll toggle and no mode state. Story `1022`'s town toggle
(`townCharacterMode`) is untouched: the town keeps its virtual 640x480 and therefore keeps the
constraint the toggle answers.

**What it does not change.** The rest of that message is testimony about where the card's controls sit
and what each one does. B3 makes it a question, `DIV-217` records the divergence, and a research
lane was opened 2026-08-21 asking it with the mapping withheld; `pipeline/RESEARCH-LOG.md` carries
the brief. No experiment id is cited here, for the same reason the question above cites none: an
experiment is immutable history that cannot announce a later amendment, and this one is not in
the pin at all. This story does not place a control on the figure either way.

## Out of scope

- The contents of any box in the column: bar sizes, cell counts, colours, scroll steps, toggle
  letters. All authored, all unchanged, none decoded.
- The figure widget's own controls. `DIV-217`: research is silent on them and the owner's
  2026-08-21 statement about them is testimony, and a research lane was opened for it the same day.
  This story draws the figure and the card; it adds no control to either.
- The left band — the pack bar and the spellbook bar. `SESS-VIEW-028` says nothing about them.
- The virtual frame itself, the viewport span and input mapping through it. The owner's 2026-08-21
  ruling moves the mission to a virtual 1024x768 frame and story `1026` builds it; this story builds
  the column's contents inside the strip that frame gives it.
- 800x600. `SESS-VIEW-028` gives its spans as well, at 20 columns by 18 rows. The owner named 1024
  and named 640 as the town's, and said nothing about the middle arm, so it is not built.
- The remaining divergence rows the 2026-08-21 ruling reaches. A census taken at the ruling found 99
  open rows, 46 of them actionable under it, of which 26 are visible on screen. That is a programme,
  not this story.

## Gates

Both repository chains by glob. `pipeline/check-release-tests.sh` and `pipeline/check-scenarios.sh`
on both roots, by name — this story changes what the screen draws, the repository chain cannot reach
those tests, and a skip and a pass both print `ok`. `pipeline/check-milestone.sh` before and after:
this story changes the map viewport, and a viewport change that moved the census is a fact the
closure owes. Report the number each script prints, not its verdict. Deletion set empty or explained.
