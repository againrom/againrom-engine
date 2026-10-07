# Story `1034` — the mission input contract, brought to the original

**Contract, seat, 2026-08-23.** Base `1fcf7e0f`, research pin `d7ee0c6`.
Branch `1034-mission-input`, worktree `wt-1034`. Stories `1035` and `1036` run in parallel in
their own branches on the owner's 2026-08-23 direction, so whichever of the three lands second
and third merges `master` first.

**Owner directive, 2026-08-23: no new experiments.** «Новые эксперименты не запускай. То что не
известно - прими решение без экспериментов.» Every unknown this story meets is decided here or by
the lane and recorded as a divergence row. Nothing holds waiting for research. Research stays the
sole authority on what ROM1 does; where it is silent, the decision is AUTHORED and the row says so.

**Owner directive, 2026-08-23: build fast.** «Делайте истории быстро без супер тщательности. Все
равно потом до 3 адверсарных ревью.» Do not build exhaustive proof into the first pass. Evidence
honesty is not on that axis: claim less, never verify less than you claim.

**Divergence ids reserved: `DIV-283` through `DIV-296`.** Allocated with `pipeline/next-div-id.sh`
against a written reservation in `PIPELINE-STATUS.md`, re-run, and the answer confirmed moved to
`DIV-297`. **When the range is spent, stop and ask the seat.** Do not take the next free number from
inside this worktree: an unmerged branch is in none of the ledgers the allocator scans, and two
other lanes are open.

## Result

Playing a mission uses the original's controls. The left button acts, the right button cancels and
deselects, the cursor under the pointer names what a click will do, and every key the original binds
does what the original binds it to.

The pointable result: `cmd/screenshot` and a real campaign mission show the walk cursor over ground
with a party selected, the take cursor over a sack, and the attack cursor over a hostile unit; a
left click on ground walks the selection there; a right click with nothing armed clears the
selection.

## Why this story exists, and why it is one story

The owner's directive, 2026-08-22: *«не нужно ограничиваться 4 кликами, делаем ВСЕ ЧТО МЫ ЗНАЕМ,
чтобы потом не дорабатывать. все в рамках 1034»* — do not stop at the four items named earlier;
build everything research has decoded about mission input, in one story, so none of it has to be
reworked later.

It follows his 2026-08-22 original-input-first ruling. Every key and click this project authored was
scaffolding. Where one occupies or scrambles a binding the original names, the original's assignment
stands and ours moves.

`AGENTS.md` requires a contract naming more than five behaviours to split or to record why it does
not. **This one does not split, by owner directive, and the directive is the record.** The
consequence is stated in the Ceiling section below rather than hidden: this is a larger story than
the process is calibrated for, and the ceiling is what stops it running away.

## The corpus

`EXP-0190` decoded the mission input contract end to end and published it as seven rows.
`EXP-0109` and `EXP-0110` published the cursor and gate family before it. Read every row whole with
`go run ./tools/claim <ID>` from the worktree's `research/`. This contract names them and does not
restate them.

| Row | Subject | Note |
|---|---|---|
| `AI-INPUT-121` | the left button's physical contract, both gates, the marquee threshold, the selected-item placement arm | High |
| `AI-INPUT-127` | the right button's physical contract: capture-to-pan, up-to-cancel, deselect-all | High |
| `AI-CLICK-050` | a click becomes an order **by the cursor it was made under**, one arm per cursor | **partially retracted** — the drag-discard clause is gone, the arms stand |
| `AI-SELECT-122` | the four selection forms, the old-summary gate, ownership, modifier precedence, the group keys | High; Medium for exact-name `CUnit` |
| `AI-PANEL-123` | the command panel and key contract, immediate versus arming, the inventory-grid producers | High |
| `AI-PANEL-053` | two-click targeted orders, the one-shot arm, both tables | **partially retracted twice** — read the overturns first |
| `AI-MINIMAP-124` | the minimap acts on left **down**, per-cursor actions, the both-buttons rule | High |
| `AI-KEY-125` | the complete decoded default mission key surface, committed as `keyboard.tsv` | High for the mapping; four mechanisms Unknown |
| `AI-CURSOR-126` | Patrol has a cursor and an order and no shipped default gesture that arms it | High |
| `AI-CURSOR-052` | the hostility test at hover, the mask, and the two modifier cursors | **partially retracted (EXP-0218)** -- TWO clauses refuted; read `AI-CURSOR-227` before this row |
| `AI-CURSOR-226` | **the ordinary-hover cascade, whole**, at `L01515`: this is the cursor table this story builds | High |
| `AI-CURSOR-227` | which two clauses of `AI-CURSOR-052` are refuted, and why "Alt forces `move` regardless" is false | High |
| `AI-CURSOR-230` | **the five replacements** that run after the cascade, in a fixed order | High |
| `AI-CURSOR-231` | all seven mask bits, including `0x40` (a visible cell carrying a drawable on the `+0x98` plane) | High |
| `AI-CURSOR-242` | **the `pickup` gate**, whole: one slot `[EBP-0x68]`, three ANDed tests, no dedicated sack bit | High for the construction |
| `AI-CURSOR-243` | which of `AI-CURSOR-209`'s two `town` sites is in this cascade | High |
| `AI-CURSOR-209` | the `town` condition at instruction level, **amended** to carry `PARTY-FLAG-003`'s reading of `+0x18c` bit `0x1` | Medium |
| `AI-CURSOR-202`, `AI-CURSOR-203` | `view+0x140` is the selected-object count; `view+0x144 & 0x24` is ownership mismatch or a selected `CStructure` | High |
| `AI-CURSOR-233` | the routine's entry gate and its four no-cursor exits | High |
| `PARTY-FLAG-003` | `CUnit+0x18c` bit `0x1` is the player-character flag | Medium |
| `AI-KEYMOD-059` | the three gate globals are modifier-key-held latches, set on down, cleared on up and on focus loss | High |
| `UNIT-HOVER-020` | the hit test's capability mask bits | **one clause retracted (EXP-0218)**: `0x20` and `0x800` are NOT alternatives |
| `AI-PANEL-060`, `AI-PANEL-061` | what the arming gate actually reads — ownership, not capability | these are the overturn of `AI-PANEL-053`'s headline |
| `MENU-COMBAT-019` | the eight cells, their labels and their tooltip indices | the command vocabulary |
| `MENU-COMBAT-017`, `MENU-COMBAT-018` | the command panel's own rect, its children and its four shipped bitmaps | geometry |

`research/experiments/EXP-0190-combat-controls/evidence/keyboard.tsv` is 57 rows of physical input
to action. It is committed evidence, not a claim: cite `AI-KEY-125` and use the file as the table.

**Read `AI-PANEL-053`'s two retractions before its row.** One moved the arming gate from a
capability test to an ownership test — a consumer taking the original text literally builds a
per-class panel filter the original does not have. The other moved the labels of opcodes `0x18` and
`0x14` to Stand Ground and Retreat.

## The seven behaviour groups

### G1 — the two buttons' physical contract

Left: `WM_LBUTTONDOWN` starts a marquee only while the inactive gate is clear; a repeated down while
it is set is a consumed no-op; `WM_LBUTTONUP` closes the marquee and acts. The marquee threshold is
`screenW*10/640`. A rectangle strictly beyond it goes to selection.

Right: down captures and pans the camera by cell deltas; up releases; a marked drag cancels nothing;
a click with no drag cancels an armed mode, or deselects everything when no mode is armed.

**This inverts what this build does today.** `pkg/ui/command.go` issues orders on the secondary
press and does selection with the primary button. The inversion is the story, not a side effect of
it.

### G2 — a click becomes an order by the cursor it was made under

`AI-CLICK-050`'s arms, all of them: `move` → `0x16`; `pickup` → `0x21`; `attack` → `0x19` at a
qualifying target and `0x16` otherwise; `swarm` → `0x1a`; `patrol` → `0x1d`; `defend` → `0x1b`, and
nothing at all with no actor under the cursor; `select` → selection and no order; `town` → `0x24`;
`cast` → `0x25`/`0x1e` at a unit and `0x1f`/`0x26` at a cell.

No diplomacy is consulted at click time. The class test at the attack arm is a runtime-class test.

An arm whose order this build has no implementation for is a named GAP with a divergence row, not a
silent omission and not an invented order.

### G3 -- the hover cursor set, from the cascade an ordinary hover actually runs

**`AI-CURSOR-052` is not the cascade an ordinary hover runs.** `AI-CURSOR-226` (High) establishes
that `R0219` has two cascades; the one `AI-CURSOR-052` reads at `L00633` is reachable
only through four gates an ordinary hover fails, and every other hover runs the cascade at
`L01515`. `1031` built B3 from the older reading and its file comments say so. **Build from
`AI-CURSOR-226`.**

The ordinary cascade, in its own order, with the mask recomputed into `[EBP-0x60]`:

1. `view+0x140 == 0` (nothing selected) -- `select` on `mask & 0x23`, `default` otherwise.
2. `view+0x144 & 0x24` set -- the same pair. In game terms (`AI-CURSOR-203`, `AI-PANEL-061`): the
   selection is foreign-owned, or it is a `CStructure`.
3. `mask & 0x4` (hostile) -- `move` with the Alt latch set; `select` when `mask & 0x20`; otherwise
   **`attack`**. **A hostile `CStructure` therefore gives `select`, not `attack`**, because every hit
   `CStructure` carries `0x20` (`AI-CURSOR-231`). This build answers `attack` there today and that is
   a player-visible defect this group fixes.
4. `mask & 0x23` (a unit was hit) -- `move` with Alt; `attack` with Ctrl; `pickup`; `town`; `select`.
5. empty ground -- `swarm` with Ctrl; `move` with Alt; `pickup`; else **`move`**.

Arm 5's final `move` is the owner's first requirement: **the default cursor over ground with a
selection is `move`**, at High, and this build draws `default` there today.

**Shift changes nothing on an ordinary hover** -- the Shift latch is read at exactly one address in
the whole routine and it is in the other cascade (`AI-CURSOR-226`).

**`pickup` is not "the pointer is over a sack".** `AI-CURSOR-242` gives the gate whole, and every
term of it is buildable: exactly one selected object (`view+0x140 == 1`) that is a `CUnit`
(`view+0x144 & 0x1`), carrying the player-character flag (`+0x18c` bit `0x1`, `PARTY-FLAG-003`,
Medium), with `view+0x144 & 0x24` clear and mask bit `0x4` clear, and **either** the hover mask
exactly `0x40` **or** the hit object is the selection itself with `0x40` also set.

**`town` is buildable now and `DIV-263` closes with this group.** `AI-CURSOR-209` is amended to
carry `PARTY-FLAG-003`'s reading of `+0x18c` bit `0x1`, and `AI-CURSOR-231` gives bit `0x800` as a
hit `CStructure` whose cell record's `+0x8c` is non-zero. Both halves of the condition now have a
game-level reading. The seat re-typed that row UNKNOWN to FIDELITY-DEBT on 2026-08-23 for exactly
this reason; the row's Status cell is the lane's to move to ACCEPTED at the landing.

**The five replacements run after the cascade** (`AI-CURSOR-230`), in this order, each overwriting
the last rather than being skipped by it: the marquee gives `default`; child 2's rect gives
`default`; child 3's rect gives `default`; a held item gives the held-item cursor; and inside that,
`backpack` under the six conditions the row names.

`DIV-262` closes with this group. `DIV-261` (`sdefend` has no armed state) closes only if G5 gives
Defend one; `DIV-230` says this build has no Defend, Swarm or Retreat order in `pkg/sim` at all, so
G5 creates them and both rows close together, or neither does and the closure says so.

### G4 — selection has four forms

`AI-SELECT-122`, whole: plain click replaces with the topmost intersecting non-structure regardless
of owner, or a structure whose class field is zero. Shift-click and Shift-rectangle toggle an owned
candidate **only when the old selection summary has its bit clear**, so after a plain click on a
foreign actor a Shift-click on an owned one does nothing. Ground and foreign candidates preserve the
selection. A plain rectangle takes every owned non-structure overlapping by strictly more than half,
and preserves the old selection when none qualifies.

The group keys are part of this row and part of this group: Alt after a one-object click expands to
the stored group without centring; `E` selects every owned unit of the same name; a plain or Shift
digit selects or augments a group; Ctrl+digit assigns; Alt+digit selects and centres; Ctrl wins over
Alt.

### G5 — the command panel and its keys

`AI-PANEL-123`'s table: Attack mode 1 → `0x19` at a target and `0x16` on ground; Move mode 2 →
`0x16`; Guard immediate `0x17`; Defend mode 4 → `0x1b` at a target and nothing on ground; Cast mode
5 → `0x1e`/`0x1f` for a spell and `0x25`/`0x26` for an item; Swarm mode 6 → `0x1a`; Stand Ground
immediate `0x18`; Retreat immediate `0x14`.

Guard, Stand Ground and Retreat are one-click commands. Attack, Move, Defend, Cast and Swarm arm a
mode and the next click consumes it. **The arm is one-shot and is cleared after every consuming
click.**

The panel is inactive for no selection, for foreign ownership and for the structure flag. **The gate
is ownership, not per-class capability** — that is `AI-PANEL-053`'s overturn and building the
capability filter is the recorded way to get this wrong invisibly.

Cast alone additionally needs the spell-capable flag.

### G6 — the minimap's own input

`AI-MINIMAP-124`: the minimap acts on left **down**, not up. Default moves the camera; `move` emits
`0x16`; `attack` emits `0x19` on any non-zero cell object id and `0x1a` otherwise; `defend` emits
`0x1b` only with an id; `cast` discards its hit test and emits nothing; `patrol` emits `0x1d`. A
left-drag repeats the current small-cursor action per delivered move. Right down and right-drag move
the camera, with no capture. With both buttons held the left action runs and no camera pan occurs.
Left up only clears a special cursor; both double-clicks and right up are no-ops.

### G7 — the mission key surface

`keyboard.tsv`'s 57 rows, cited through `AI-KEY-125`.

**Every key this build occupies against the original moves.** The owner's original-input-first
ruling already moved `B` and `R`. Measured on master `f806e48a`, at least two more are occupied:
`pkg/ui/app.go` binds `S` to the spell book and `D` to the paper doll, while the original binds `S`
to Swarm and `D` to Defend, and `text/main.txt` line 8 names the book as `<Q>,<B>`. **Measure the
whole set yourself against `keyboard.tsv` before moving any of them** — that list is two entries the
seat happened to find, not the population.

A key whose destination does not exist in this build is a named GAP with a divergence row: F1 help,
and F3's Diplomacy phase, whose menu destinations are `DIV-099`'s subject. Do not invent a
destination and do not leave the key silently inert.

## What is deliberately not in this story

- **Town and shop input.** Different screens with their own decoded contracts.
- **The character card widget's own controls.** That is `DIV-217` and its own story.
- **The inventory grid's internal transfer paths and the Drop Gold modal.** `AI-PANEL-123` decodes
  them and they are inventory work, not mission input. The **map-side** emissions the same row names —
  the selected-item placement arm's `0x23` and `0x22`/`0x32` — are in G1, because they are what a map
  click does while an item is held.
- **Building menu destinations that do not exist** (`DIV-099`).
- **The Patrol default gesture.** `AI-CURSOR-126` establishes that the original has none. The cursor
  and the order exist and no shipped default input arms mode 8. Do not add one.
- **Five decoded key groups, moved out at round 3** because each needs a production seam this build
  does not have and building one was outside this story: `F5`..`F8` quick spells (`DIV-328`),
  `Ctrl`+numpad `+`/`-`'s unpaced loop (`DIV-329`), `Ctrl`+`W`'s formation cycle (`DIV-330`),
  `Ctrl`+`F`/`L`/`U` where this build carries no such state (`DIV-331`), and the settings captions
  the original shows on every toggle (`DIV-332`). `closure.md` names the follow-up work for each.

## Ceiling

**Four adversarial passes.** `AGENTS.md`'s ladder gives an ordinary story three, four where the
story reaches hashed simulation state or touches more than three domains, and five absolute. This
story does both: orders and stance reach hashed simulation state, and the Domains section below
names four domains. Four is therefore the cap this contract sits at, not a decision it makes.

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

**Reaching the ceiling is a scoping diagnosis, not a failure.** If the third pass still finds a
player-visible defect, land the groups that are complete, open the remainder as its own defect
story, and record in `closure.md` that the story was cut too large. The owner's directive is that
the whole surface is specified in one contract so nothing is missed; it is not a requirement that
all seven groups land in one commit. That is what happened: round 3 landed three of the eight
decoded key groups `AI-KEY-125` gives and moved five out, and `closure.md` records the cut.

**Land the groups in a witnessed sequence.** Each group gets its own witness before the next begins,
so a returned pass names a group rather than the story. G1 and G2 first, since every other group
depends on which button acts.

## Domains

Client, Sim Core, AI & Orders, and Campaign & Scripts where an order reaches a trigger. Four,
which is over the three-domain line and is the second reason the ceiling above is four rather than
three.

## The decisions taken without an experiment

Under the 2026-08-23 directive each of these is settled here rather than asked. Every one is a row.

- **Is the mask-`0x40` drawable a sack?** `AI-CURSOR-231` leaves it open and `AI-CURSOR-242` does not
  close it. **Decision: gate `pickup` on this build's own sack presence at the hovered cell.**
  `AI-CURSOR-231` reads `0x40` as a visible cell carrying a drawable on the `CMapView+0x98` plane
  with no actor class bit set; `ITEM-SACK-010` puts a sack in the world's sack registry, one per
  cell, outside the actor tick list, so it is exactly a drawable with no actor identity; and
  `AI-CLICK-050` maps the `pickup` cursor to opcode `0x21`. DEVIATION. Revisit: a claim naming which
  drawable class the `+0x98` plane holds.
- **What are `R0375`'s children 2 and 3 on the mission view?** `AI-CURSOR-230` states plainly
  that this is not established. **Decision: implement replacements (2) and (3) as `default` whenever
  the pointer is inside any child widget of this build's own map view.** DEVIATION.
- **The marquee threshold `screenW*10/640`.** This build has two surface models (`DIV-249`).
  **Decision: take it from the mission frame's own width**, the same reasoning `1031` recorded for
  the edge band. DEVIATION.
- **`AI-KEY-125`'s keys whose action has no decoded meaning** -- F12's global, Backspace's three
  containers, Alt+B..Y's outbound record. **Decision: bind nothing.** There is no behaviour to build
  and an invented one is worse than an absent one. UNKNOWN, one row for the three.
- **F1 help and F3 Diplomacy have no destination in this build.** **Decision: they are out of scope
  as screens** (see the exclusions) and the keys stay unbound rather than bound to nothing. The
  missing destinations are `DIV-099`'s subject. FIDELITY-DEBT.

## Divergence rows

The range is `DIV-283`..`DIV-296`, reserved in `PIPELINE-STATUS.md`. Beyond the five decisions
above, expect rows for:

- every `AI-CLICK-050` arm whose order this build cannot issue;
- `DIV-261` and `DIV-263` if G2 and G5 do not give them a state;
- each key whose destination does not exist;
- the marquee threshold, which is a screen-pixel formula and this build has two surface models
  (`DIV-249`, and `1031`'s own edge-band row is the precedent for how to disclose it).

`DIV-262` closes with G3.
