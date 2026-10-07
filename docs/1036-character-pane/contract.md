# Story `1036` — the character pane's corner controls, and one pane on both screens

**Contract, seat, 2026-08-23.** Base `1fcf7e0f`, research pin `d7ee0c6`.
Branch `1036-doll-corners`, worktree `wt-1036`. Stories `1034` and `1035` run in parallel in their own
branches on the owner's 2026-08-23 direction, so whichever of the three lands second and
third merges `master` first.

**Owner directive, 2026-08-23: no new experiments.**
«Новые эксперименты не запускай. То что не известно - прими решение без экспериментов.»
Every unknown this story meets is decided here or by the lane and recorded as a divergence
row. Nothing holds waiting for research. Research stays the sole authority on what ROM1
does; where it is silent, the decision is AUTHORED and the row says so.

**Owner directive, 2026-08-23: build fast.**
«Делайте истории быстро без супер тщательности. Все равно потом до 3 адверсарных ревью.»
Do not build exhaustive proof into the first pass. Evidence honesty is not on that axis:
claim less, never verify less than you claim.

**Divergence ids reserved: `DIV-307` through `DIV-316`.** Allocated with `pipeline/next-div-id.sh`
against a written reservation in `PIPELINE-STATUS.md`, re-run, and the answer confirmed
moved to `DIV-317`. **When the range is spent, stop and ask the seat.** Do not take the next
free number from inside this worktree: an unmerged branch is in none of the ledgers the
allocator scans, and two other lanes are open.

**`1034` runs in parallel rather than before this story**, so do not wait for it and do not
build on its files. Hang the six corner rectangles on the input path this build has on
`master` today. `1034` inverts which mouse button acts on the map; that inversion is its
work, not this story's, and whichever of the two lands second merges `master` first and
reconciles.

## Result

The mission screen's character pane is the town screen's character pane, and it carries the
original's controls in its own four corners. Clicking the top-right corner switches the pane between
the doll and the statistics card. Clicking the bottom-right corner opens the in-mission menu. The two
top-left and bottom-left corners toggle the two map-view flags the original toggles there. In a shop
opened from a mission the party-picker corners light up as well, and the pane is the same object
throughout.

The pointable result: `cmd/screenshot` on a real campaign mission shows the same 160x242 character
pane as the town screen, with the statistics toggle in its top-right corner; clicking that corner
switches the pane; clicking the bottom-right corner raises the in-mission menu that Esc raises today.

## Why this story exists, and why it is one story

The owner's directive: the buttons and controls in the **corners** of the doll, in the town screen
and in the mission, and within the same story the doll and stats presentation in the mission must
become the same as in town, reusing that pane as far as possible.

The reuse half is not a refactor bolted onto a feature story. It is what the original does.
`TOWN-353` establishes at High that there is **one instance** of this widget, re-parented between the
mission screen and the shop screen, with three constructor overloads of which two are dead code and
one is the only live construction site. Every screen difference is a runtime branch on
`campaign+0x3dc`. This build has two separate implementations of that one object, and the corner
controls are exactly the behaviour that the split has been hiding: four of the six rectangles are
absent from both screens, and the two that exist are shop-only and sit beside an authored control the
original does not have. Building the controls on two implementations would build them twice and would
still not produce the original's behaviour, because the gates that decide which corner is live read a
single session field that only one object can hold.

`AGENTS.md` requires a contract naming more than five behaviours to split or to record why it does
not. This one names six rectangles plus the pane unification. It does not split because the six share
one hit-test routine, one gate field and one paint routine in the original, and because five of the
six are one line of work each once the pane is one object. The ceiling section states what that
costs.

## The corpus

`EXP-0210` decoded this widget end to end and published it as twelve rows. The controls' effects are
**not** all in `claims/town.md`: five of the six messages are answered by rows in three other
ledgers, and `go run ./tools/claim -k` on each message id is how they were found. Read every row
whole from the worktree's `research/`. This contract names them and does not restate them.

| Row | Subject |
|---|---|
| `TOWN-344` | the widget: `campaign+0xe0`, id 7, vtable `L00722`, eight interactive behaviours, no child controls |
| `TOWN-345` | every rectangle is panel-relative to an ancestor-accumulated origin; the constructor rect and the mission-screen accumulation |
| `TOWN-346` | **the six rectangles**, their bounds, their gates, their test order, and the absence of any early return between them |
| `TOWN-347` | **what each rectangle posts**, and the map-view field rect C writes directly |
| `TOWN-348` | a held item pre-empts all six rectangles, on six values of `[sess+0x3cc]+0x18`, in two arms |
| `TOWN-349` | the per-pixel slot-id map in `this+0x78`, sampled by double-click and by drag — the principal control surface, which is not a rectangle |
| `TOWN-350` | `vt+0x78` pick-up and `vt+0x7c` drop, with the equipment array at `member+0x15c` |
| `TOWN-351` | `this+0x5c` is the map view; the widget's own message arms, including what `0x40e`, `0x40f` and `0x412` do to its own fields |
| `TOWN-352` | `campaign+0x3dc` is a bitmask; mission is 1, a shop opened from a mission is 3 |
| `TOWN-353` | **one instance, re-parented**; the six rects and the pixel map are identical on both screens and only the gates differ |
| `TOWN-354` | each rectangle's own art, and the mask each blit is drawn under |
| `TOWN-356` | **which archive entry each of the eleven blit globals is**, corrected against each string's own bytes; added at the lane, 2026-08-23, having been left out of this table |
| `TOWN-355` | **three gate mismatches and two overlapping rectangle pairs**: a live rectangle that is never drawn, and one click that posts two messages |
| `SHOP-FIGURE-041` | the id-7 object's two presentation modes and the shop transfer |
| `SHOP-PICKER-043` | rects D, E and F as the three 32x32 rectangles, and the picker binding |
| `AI-CURSOR-177` | the map view's private dispatcher, 132 ids `0x401`-`0x484`, decoded as a table |
| `AI-KEY-125` | `0x412` is the focused-character-panel Tab message |
| `MENU-ESC-010` | Esc with `campaign+0x3dc == 1` posts `0x416`, which is what rect F posts |
| `TOWN-136`, `TOWN-137`, `TOWN-139`, `TOWN-140` | `0x414` and `0x415` as the party-picker previous and next |
| `SESS-VIEW-028` | the three shipped resolutions, which decide whether rect C is live |

## What this build does today, measured 2026-08-23 on `origin/master` `f806e48a`

**Re-measured against the base `1fcf7e0f` at the start of the lane, 2026-08-23.** A record of what
a build does not do is an absence claim and it decays.

**The commit count in this paragraph was wrong.** It read *four commits have landed since this was
written*; `git rev-list --count f806e48a..1fcf7e0f` is **29**. Of the files this section names, only
`pkg/ui/viewer.go` changed across those 29, and the change moved the doll's own draw site from
`viewer.go:3269` to `:3325`. Every other statement below re-measured true on `1fcf7e0f`.

- **Two implementations of one object.** The shop and town path is `pkg/ui/shopscreen.go` with
  `pkg/game/characterpane.go` behind it; the mission path is `pkg/ui/inventory.go` (`dollSubject`
  :701, `dollBox` :792, `dollInventoryActive` :923, `dollPresent` :1117, `dollFigureSlotAt` :1282,
  `dollFigureSlotAtUnsuppressed` :1311, `dollFigureSlotAtMask` :1318) drawn from
  `pkg/ui/viewer.go:3325` (`:3269` at `f806e48a`). `pkg/game/characterpane.go` already models the two presentation modes.
- **The mission pane has no hit test but the slot mask.** `dollFigureSlotAt*` is `TOWN-349`'s pixel
  map. None of the six rectangles exists on the mission screen.
- **The town pane has two of the six, plus one the original does not have.**
  `pkg/ui/townshell.go:25` `TownCharacterRegion = (480, 238, 640, 480)` is `TOWN-345`'s accumulated
  rect at 640x480 to the pixel. `townCharacterPrev = (481, 443, 513, 475)` is rect **D**, and
  `townCharacterNext = (599, 443, 631, 475)` is rect **E**, both exact against `TOWN-346`.
  `townCharacterMode = (516, 443, 590, 475)` is **authored**: it is not any of the six, and it sits
  between the two pickers where the original puts nothing. The original's mode control is rect **C**,
  in the pane's top-right corner at `(608, 238, 640, 274)`.
- **`ShopControlBook` is not one of the six** and is not in scope here.
- **Rects A, B and F exist nowhere.** F is the in-mission menu control; the build reaches that menu
  only from the Esc key.

## The six rectangles, and what each becomes

Bounds are panel-relative to `TOWN-346`'s `(L,T,R,B)`. On the mission screen `campaign+0x3dc == 1`,
so A, B and F are live, D and E are gated off, and C is live at 640x480 and 800x600 and gated off at
1024x768. In a shop opened from a mission the field is 3 and all six are live.

| Rect | Bounds | Posts | What it does | Build today |
|---|---|---|---|---|
| **A** | `(L, B-0x28, L+0x1c, B)`, 28x40 | `0x40e` to itself and to the map view | inverts `this+0x68`; the map-view arm is in `AI-CURSOR-177`'s table and its own behaviour is **not resolved** | absent |
| **B** | `(L, T, L+0x1c, T+0x24)`, 28x36 | `0x40f` to itself, to the map view, and to the shop when `& 2` | inverts `this+0x6c`; same open arm | absent |
| **C** | `(L+0x80, T, R, T+0x24)`, 32x36 | `0x412` to itself, then writes `[mapview+0xe0] = 1` | plays sound `0xdc`, inverts `this+0x70` — read four times by the paint routine — and sets `this+0x60 = 1`. This is the doll/statistics toggle | authored rect in the wrong place, shop only |
| **D** | `(L+1, T+0xcd, L+0x21, T+0xed)`, 32x32 | `0x414` to `sess+0xcc` | party picker previous | correct rect, shop only |
| **E** | `(L+0x77, T+0xcd, L+0x97, T+0xed)`, 32x32 | `0x415` to `sess+0xcc` | party picker next | correct rect, shop only |
| **F** | `(L+0x7e, T+0xce, L+0x9e, T+0xee)`, 32x32 | `0x416` through `PostMessageA` | the in-mission menu `MENU-ESC-010` gives Esc | absent |

`this+0x68` and `this+0x6c` are written by A's and B's own arms and **read by no routine of this
class**, the paint routine included (`TOWN-351`). Their effect, if any, is in the map view. The
contract requires A and B to post the messages the original posts and to carry the toggles; it does
**not** require the map view to act on them, because nothing decoded says what the map view does with
them. That gap is a divergence row and a research question, not a hold.

## The behaviour groups

**G1 — one pane, two parents.** The mission character pane and the town character pane become one
implementation, gated on the session field rather than on which file drew it. The 160x242 body, the
two presentation modes, the slot map and the drag surface are shared. `pkg/game/characterpane.go`
is the seam that already exists.

**G2 — the six rectangles, with the original's gates.** One hit-test routine, tested in the
original's order, panel-relative to the accumulated origin so it holds at all three resolutions.
`TOWN-346` states that no arm returns between the six tests; the implementation reproduces that,
which is what makes `TOWN-355`'s overlaps observable rather than hidden.

**G3 — the held-item pre-emption.** `TOWN-348`: on six values of `[sess+0x3cc]+0x18` the routine
returns before any point-in-rect call, and on values 1 and 2 it drops into the slot the held item
names. A corner that answers while an item is on the cursor is the defect this group prevents.

**G4 — the mode toggle moves to the original's corner.** The authored `townCharacterMode` retires and
rect C takes its work, including the sound and the map-view write. This is the 2026-08-22
original-input-first ruling applied to a click: our control occupies a place the original uses for
nothing and leaves the original's own corner empty.

**G5 — the menu corner.** Rect F posts what Esc posts, so the in-mission menu has the control the
original gives it.

**G6 — the art, under the original's masks.** `TOWN-354` binds each rectangle to its own bitmap and
the mask it is drawn under. Drawing a live control is not optional: three of `TOWN-355`'s findings
are that a rectangle answers clicks while nothing is painted there, and reproducing that faithfully
is a decision this story makes deliberately rather than by omission.

## What is deliberately not in this story

- **The map view's own `0x40e` and `0x40f` arms.** Not decoded, and not asked: the owner's
  2026-08-23 directive decides them here. This exclusion covers the map view's own windows alone.
  What rects A and B DO is in scope and is built -- see the *Divergence rows* section below, and
  `DIV-307`. An earlier revision of this line read *A and B post; the map view ignores them*, which
  contradicted that section and would have shipped two parsed-but-unconsumed messages.
- **`ShopControlBook`, the merchant control, the shelf and table grids.** Not among the six.
- **The double-click and drag paths of `TOWN-349`/`TOWN-350`.** The slot map already works on both
  screens; this story unifies it and does not re-derive it.
- **Resolutions other than the three `SESS-VIEW-028` enumerates.**

## Ceiling

**Four adversarial passes.** `AGENTS.md`'s own ceiling is three ordinarily, four for a story
touching more than three domains; this story's own Domains section below names four, so four is
the ceiling that applies, not a decision this contract makes.

**The chain ends at the first pass with no P finding, before the ceiling.** `AGENTS.md` classes
every finding and only one class returns a story: **P**, the player sees it wrong or hashed
simulation state is wrong. **W** -- production is correct and the witness cannot see it -- is a
ledger row. **D** -- production and witness are correct and a document says something untrue -- is
fixed in place or is a ledger row. Neither returns the story. A pass that finds no P therefore
closes the chain, and its W and D findings are applied without another round.

The trade is deliberate and its price is named in the same rule: stopping at the first pass with no
P finding would have shipped one defect that `1005` found on its twelfth pass. A hotfix costs one
ledger row and no lane; a review round costs a lane and a review. While a round returns fewer than
about one player-visible defect in three, it costs more than it prevents.

Returning the story to apply a pass's findings is not a pass. The ceiling counts passes, not fixes.

The size risk is G1, not the six rectangles. Merging two implementations of one widget is the kind of
change whose defects are invisible on the screen that already worked. The mitigation is stated as an
order, not a hope: **G1 lands with the town screen's existing witnesses green and unchanged**, and
the mission screen's new witnesses are written against the same helpers. `check-release-tests.sh` and
`check-scenarios.sh` are named in the brief, on both roots — this story changes what the screen draws,
and a green repository chain says nothing about either.

If the ceiling is reached, the scoping diagnosis is that G1 and G2 were one story when they were two.

## Domains

UI/HUD, inventory/equipment, input, campaign/session. The adversarial reviewer walks those four
domains' interfaces.

## Divergence rows

`TOWN-355` supplies three that exist before a line is written, and they are the original's own
defects rather than ours:

1. Rects A and F answer clicks under `& 1` while their art is gated on `!(& 0x226)` and `!(& 0x400)`,
   so in a shop opened from a mission both are live and neither is drawn.
2. Rects D and E keep their hit test under `& 0x226` and lose their art when `(& 0x200)` and
   `sess+0x6bc == 2`.
3. Rect A overlaps rect D over 27x32 pixels and rect E overlaps rect F, with no early return between
   the tests, so one click in either overlap posts two different messages.

**The undecoded map-view arms behind A and B are decided here, not asked.** Under the 2026-08-23
directive: A and B post the messages the original posts and carry their own toggles, and this
build's map view ignores both. That is a DEVIATION row with the decision stated, not a hold and not
a research request.

A fourth row records those two arms. A fifth records rect C's gating at
1024x768: the original's own condition is `screenH > 722` with `B - T` always `0xf2`, and `TOWN-346`
states plainly that the last step assumes the containers above `campaign+0xd4` contribute no vertical
origin, which was not read.

The range is `DIV-307`..`DIV-316`, reserved in `PIPELINE-STATUS.md`. When it is spent, stop and
ask the seat.
