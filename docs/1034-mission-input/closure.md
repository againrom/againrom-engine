# Story `1034` — closure

As-built evidence for the mission input contract. The behaviour itself is in `spec.md`; the claim
provenance and the scope decision are in `contract.md`.

Branch `1034-mission-input`, base `1fcf7e0f`, research pin `d7ee0c6`. `origin/master` was merged
twice: at `da687c82` after round 1, and at `d9d8d9f5` at the top of round 3, taking master's
`8823d86e` (story `1035`). Both merges conflicted only on appends to the same ledger tables, and both
sides were kept every time. Every gate below was re-run on the second merged branch.

Round 4 answers one P finding the owner reported against his own build of round 3: with a player
character selected, the map's `pickup` cursor appeared over a sack and the click did nothing at all,
neither walking nor taking. It merged no further master.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | N/A | No file format is read or written differently. The one file consulted, `keyboard.tsv`, is research evidence read at authoring time and not at runtime. |
| Runtime state | PASS | Four new fields on `Viewer`: `groups [10]selection` (ids, not entities), `minimapActX`/`minimapActY`, and the existing three modifier latches now read. Round 3 adds `Viewer.healthBarsHidden` (stored inverted, so "shown" is the zero value) and `flow.selfNotice`. All are viewer-local or flow-local, none is serialized, none reaches `pkg/sim`. |
| Simulation | PASS | `git diff --name-only 1fcf7e0f..HEAD -- pkg/sim/` is empty. Every order this story dispatches leaves through a seam that existed before it: `MapOrder`, `MapAttack`, `MapMarch`, `MapStance`, `MapGrab`. No hashed state, no byte form, no version bump. Round 4's pick-up order reaches `sim.World.TakeSack`, which writes the world, but through the same pre-existing primitive the pick-up key already used and with no new `pkg/sim` mechanism: the walk is `enqueue`'s ordinary `KindGroupMoveTo` and the arrival latch is driver state on `mapWorld` (`DIV-337`). |
| Player input | PASS for what is in scope, with five key groups moved out | The story's whole subject, `spec.md` B1–B7. Round 3 wired three of the eight decoded `AI-KEY-125` key groups that round 2's adversarial pass named: `Ctrl`+`N` day/night (moved off bare `N`), `Ctrl`+`H` show health, and `Pause`'s modal text. Each has its own witness in `pkg/ui/settingskeys_test.go`, listed under "Round 3 witnesses" below. Five groups were **moved out of this story's scope**, not left as a gap inside it, each with a row and a named follow-up: `F5`..`F8` quick spells (`DIV-328`), `Ctrl`+numpad `+`/`-`'s unpaced loop (`DIV-329`), `Ctrl`+`W`'s formation cycle (`DIV-330`), `Ctrl`+`F`/`L`/`U` (`DIV-331`), and the settings captions (`DIV-332`). See "Scope cut". |
| AI | N/A | No AI or script behaviour is touched. |
| UI/HUD | PASS | The hover cursor set (B3), the command panel's Swarm cell (B5), the minimap's dispatch (B6), and the `Space` panel toggle (B7). Round 3 adds the show-health gate on `healthBarScreenRects` and the `Pause` key's dialogue notice, both in `pkg/ui`. |
| Triggers/scripts | N/A | Untouched. The script-gap census confirms it: mission 10 and mission 20 report 0 `UNSUPPORTED` nodes on the en root, unchanged. |
| Inventory/equipment | PASS | The `pickup` cursor's gate (`AI-CURSOR-242`) and its click arm; the held-item cursor replacement; the `I`/`` ` ``, `X` and `J` keys. Round 4 made the click perform `ITEM-PICK-016`'s `0x21`: the ordered unit walks to the sack's own cell and takes it on arrival. `DIV-292`, which recorded the click acting for the inventory window's subject instead, is CLOSED. What is still not reproduced is on `DIV-338`. |
| Persistence/save-load | PASS | Nothing this story adds is serialized, and the one new field is ruled rather than left unruled. Group assignments are session state and are not written to a save; the original's own persistence of them is not established. Round 4's `mapWorld.pickup` is ruled `cosmetic` in `pkg/game/save_test.go` and in `docs/0143-save-and-load/spec.md` FR-5, which `TestEveryMapWorldFieldIsRuled` enforces field by field. The original's copy is on the actor (`ITEM-PICK-016`, `AI-STATE-011`) and this build's is on the driver; `DIV-337` carries the difference. |
| Campaign/session | N/A | No campaign or session transition is touched. |
| Shipped content | PASS | The key sweep is against `keyboard.tsv`'s 57 rows, not against a sample. The integration witness runs on mission 10 from a lawful install, on both preserved roots. `check-scenarios.sh` is 14 of 14 on both roots. |
| Interactions with existing mechanics | PASS | See the sweep section below: the doc population, the precedence unification, and the four pre-existing divergence rows this story moved. |

No in-scope `GAP`. Five decoded key groups were moved **out** of scope at round 3 rather than left
as a gap inside it. Each has a typed row in `docs/DIVERGENCES.md` and a named follow-up in "Scope
cut" below, which also records that this story was cut too large.

## Scope cut

**This story was cut too large.** `contract.md` names seven behaviour groups across four domains, two
of which reach hashed simulation state, and it puts the whole 57-row `keyboard.tsv` key surface
inside one of them (G7). `AGENTS.md`'s rule is that a contract naming more than five behaviours, or
combining hashed simulation with more than three domains, splits unless it records why it does not.
This contract did not split and did not record a reason. The cost is on the record: three adversarial
passes, the second of which found eight decoded key groups bound to nothing, and a round 3 that could
wire three of them and moved five out. Round 4 is the fourth lane round and was opened by the owner
against his own build rather than by a review pass: the map's `pickup` cursor promised an action the
click never performed. That defect is inside the part of the contract that stayed in scope, so it is a
fix rather than a further cut, and it is the same diagnosis. A contract this size leaves parts of
itself unwitnessed for three rounds.

Round 3 wired `Ctrl`+`N`, `Ctrl`+`H` and `Pause`. The five groups below are moved out of this story's
scope. Each is decoded, each needs a production seam this build does not have, and building one is a
mechanism rather than a key binding. Each is a typed row in `docs/DIVERGENCES.md` and each is named
here so the remainder is somebody's work rather than a hole.

**`DIV-328` — `F5`..`F8`, quick spell slots 0..3.** The seam needed is a per-character quick-spell
slot store and the route that fills it. This build has `Viewer.selectedSpell` and the spellbook
screen; it has no slot array, no slot control and no assignment gesture. `DIV-234` records the
command panel's Cast cell waiting on the same missing mechanism, so one follow-up closes both. Once
the store exists the key work is four arms in `readAppInput` selecting slot 0..3 and arming mode 5.

**`DIV-329` — `Ctrl`+numpad `+` and `-`, the unpaced owner idle loop.** The seam needed is a second,
unpaced arm on this build's tick driver. `flow.stepCadence` (`pkg/ui/flow.go`) walks a ladder whose
every rung is a period with a deadline and a catch-up bound; `SESS-CLOCK-005` gives the original's
unpaced loop no deadline test and no `AND 0xf` epoch wrap, and there is no phase or epoch here for
`Ctrl`+numpad `-` to reset. A follow-up builds the unpaced arm and the phase/epoch state that
`campaign+0x40c` selects between, then binds the two keys. The bare numpad pair already carries
`!ctrlHeld()` as of this round, so the modified pair reaches nothing and is free to take.

**`DIV-330` — `Ctrl`+`W`, the formation cycle. This one reaches hashed simulation state.** It is its
own story rather than a key binding. The seam needed is a player-command road into `pkg/sim`'s
formation byte: `setFormationMode` (`pkg/sim/formation.go`) is trigger instant 7's raw store, is
unexported, and the byte is serialized (`pkg/sim/tailbinary.go`). `AI-FORM-037` gives the player road
a remap the trigger road does not apply, `0->0, 1->2, 2->1`, so a front-end write of the raw store
would exchange the auto and on arms. The follow-up adds a command kind carrying the authored value,
applies the remap on the sim side, and witnesses the resulting state rather than the key.

**`DIV-331` — `Ctrl`+`F` retreat mode, `Ctrl`+`L` smoothing, `Ctrl`+`U` autohealing.** The seam needed
is the three settings themselves. None of the three has any state in this build. The persistence half
has a home already: `OptionsStore` (`pkg/game/tipstore.go`) is shaped for `TOWN-186`'s thirteen
persisted engine options and reads and writes only `TipsMode`. The behaviour half does not exist, and
that is what a follow-up builds first: something a retreat mode gates, an autoheal cadence, and a
sprite-scaling filter this build currently declines to apply by fixed choice. Binding the key is the
last step, not the first.

**`DIV-332` — the settings captions.** The seam needed is a transient on-map caption surface, plus
the `main.txt` index pairs in `ui.Words`. This one is blocked on research before it is blocked on a
seam: `keyboard.tsv` gives every setting's line range (`[94..96]`, `[97..99]`, `[100..101]`,
`[102..103]`, `[104..105]`, `[106..107]`, `[218..220]`, and `[108..116]` for the numpad speed step)
and no claim names the surface those lines are drawn on, its rectangle, its font or its dwell. The
follow-up needs that claim first. It also applies to the three settings this build already wires,
which flip silently today, so it is not confined to the moved-out groups.

The four groups of `DIV-321` are a different class and are not follow-up work of this shape: nothing
decodes what `F12`, `Backspace`, the `Alt`+`B`..`Y` band or `Tab`'s `0x412` message do, so there is
no destination to build. That row's revisit condition is a claim, not a seam.

## Integration witness

`pkg/game/missioninput_release_test.go`, two install-gated tests on a real campaign mission:

- `TestReleaseMissionMapLeftTapSelectsThenOrdersOnRealMissionContent`
- `TestReleaseMissionMapRightUpCancelsOnRealMissionContent`

Both open mission 10 through `FrontEnd.MissionOpener` and drive `App.step` — the same entry a
windowed session's `Update` calls — through `App.HeadlessPointer`. The chain under test is the whole
one: window-to-frame mapping, the map arm's gate, the hover cursor computed from real terrain and
real entities, `decide`, the `Map*` seams, and `mapWorld.enqueue`. The result is observed at the
simulation's own command queue, which is the value the production path produces.

They discriminate against the build this story replaced. Before it, the left button selected and the
right ordered; the first test asserts that a left tap on ground with a selection standing queues one
`KindGroupMoveTo` naming the tapped cell, and the second that a right press and release queues
nothing and clears the selection.

Mutation-proved at three production sites a maintainer would change, each reverted with the same edit
pair:

| Mutation | Result |
|---|---|
| `g.tap = v.held` → `g.tap = false` (`command.go`) | FAIL — the selection step |
| `inside = inside && kind != orderKindMove` after the ground extent test (`command.go`) | FAIL — "a left tap on ground with one unit selected queued 0 commands, want 1" |
| `v.sel = nil` removed from the right-up cancel arm (`command.go`) | FAIL — "after a right press and release with nothing armed, SelectedUnit = (35,true), want no selection" |

`App.HeadlessPointer` gained the secondary button's three edges (`right-press`, `right-move`,
`right-release`) so that this witness could exist at all: a vocabulary that can only press the primary
button can drive half of this story's contract and cannot see the other half.

### Round 3 witnesses

Three key groups were wired at round 3. Each has its own witness in
`pkg/ui/settingskeys_test.go`, and each was mutation-proved at the **use** site rather than at the
predicate's own declaration. Every mutation below was reverted with the same edit pair and the file's
sha256 checked byte-identical afterwards.

| Wired | Witness | Mutation at the use site | Result |
|---|---|---|---|
| `Ctrl`+`N` day/night, moved off bare `N` | `TestTheSettingsKeysCarryTheirModifier/TimeFlow`, `TestBareNIsNoLongerBound` | `&& ctrlHeld()` dropped from `TimeFlow` in `readAppInput` | FAIL: "TimeFlow is bound to \"inpututil.IsKeyJustPressed(ebiten.KeyN)\" with no unnegated ctrlHeld()" |
| `Ctrl`+`H` show health | `TestTheSettingsKeysCarryTheirModifier/ShowHealth` | `&& ctrlHeld()` dropped from `ShowHealth` | FAIL, same shape |
| `Ctrl`+`H`'s effect | `TestShowHealthKeyHidesTheHealthBars` | the `v.healthBarsHidden` early return deleted from `healthBarScreenRects` (`overlay.go`) | FAIL: "2 health bars survived the show-health key, want 0" |
| `Pause`'s modal text | `TestPauseKeyShowsTheModalTextAndOneDismissalClosesIt` (RETURN and ESCAPE), `TestPauseKeyIsRefusedWhileANoticeStands` | the `if in.PauseText` arm deleted from the map step (`app.go`) | FAIL: "the Pause key opened no notice", both dismissal keys |

The two settings witnesses read the binding out of `app.go`'s own source with an AST walk
(`bindingSource`) rather than by simulating a key, because Ebitengine's key state is not injectable
from a test. They therefore witness the binding and its modifier polarity, not a keystroke reaching
the arm; the arm's effect is witnessed separately by the two behavioural tests in the same file.

`AI-CURSOR-230`'s cursor test was also corrected at round 3 and is witnessed the same way.
`TestMarqueeCursorFollowsTheClaimsOwnRectTest` hovers a plain unit with nothing selected, where the
decoded cascade answers `select`, and asserts the name `missionHoverCursor` produces at nine drag
shapes. Reverting the use site in `missioncursor.go` to round 2's `len(v.marqueeScreenRects()) > 0`
fails it, including at the 1-pixel and 5-pixel drags inside the release threshold's band, which is
the defect (`DIV-290`, `DIV-335`).

### Round 4 witnesses

The defect had two halves and only one of them was wrong. `AI-CURSOR-242`, read whole, gates the
`pickup` cursor on exactly one selected `CUnit`, that object's own `+0x18c` bit `0x1`, and then
either the hover mask equal to the drawable bit or the hit object being the selection itself with
that bit set. There is no distance term, so the cursor was correct over a distant sack.
`AI-CLICK-050` gives the click under it opcode `0x21`, and `ITEM-PICK-016` reads `0x21` as naming an
arbitrary cell and walking the ordered actor to it. The click was the wrong half.

Two new files carry the witness, and each observes a production result rather than a predicate.

`pkg/ui/pickupclick_test.go` drives the real hit test. `pcOnMap` parks an `App` on the map, presses
at the hero's own cell through `atPress` so the selection is made by the production selection path,
and puts one sack seven columns and four rows away.

| Test | What it observes |
|---|---|
| `TestTheMapPickupCursorAppearsOverADistantSack` | `v.mapCursorName()` — the whole production cascade, not `pickupGate` |
| `TestAClickUnderThePickupCursorReachesTheSeamWithTheUnitAndTheSacksCell` | the recorded `MapGrab` call after `atTapPoint`, asserting the payload is the selected unit's id and the sack's own cell, and that no `MapOrder` was issued beside it |
| `TestAClickUnderThePickupCursorPastTheMapExtentOrdersNothing` | `decide`'s return with a `groundAt` that resolves nothing |

`pkg/game/pickuporder_test.go` drives the far side and observes the simulation world.

| Test | What it observes |
|---|---|
| `TestThePickupOrderWalksTheOrderedUnitToTheSackAndTakesItOnArrival` | the entity's cell each tick, then `w.Sacks()` and `w.Carried` after arrival, in the order the defect was reported: the unit must move, and then the sack must be taken |
| `TestThePickupOrderRefusesACellWithNoSackOnIt` | `mw.pending` and the entity's cell, so a refused order starts no walk |
| `TestALaterOrderCancelsAStandingPickup` | `w.Sacks()` after the unit reaches the same cell under a plain move order |
| `TestAStandingPickupDisarmsWhenTheSackGoesAway` | the latch after a competing `TakeSack` by a second entity |
| `TestThePickupOrderTakesForTheORDEREDUnitAndNotTheInventorySubject` | `w.Carried` for both entities, with the inventory subject and the ordered unit deliberately different |

Eight mutations at the production **use** sites, each reverted with the same edit pair and the file's
sha256 checked byte-identical afterwards.

| Mutation | Result |
|---|---|
| `mw.settlePickup()` dropped from `tickWithCastSink` | FAIL — the arrival test |
| `mw.enqueue(...)` dropped from `orderPickup` | FAIL — "the ordered unit never reached (8,3) ... the pick-up order issued no walk" |
| the sack lookup dropped from `orderPickup` | FAIL — the refusal test |
| `mw.cancelPickup(id)` dropped from `queueGroup` | FAIL — the cancel test |
| the arrival comparison dropped from `settlePickup` | FAIL — three tests |
| `pick(ord.entity, ord.x, ord.y, true)` reverted to the key payload `pick(0, 0, 0, false)` in `app.go` | FAIL — the seam payload test |
| the cell dropped from `decide`'s pick-up arm | FAIL — the seam payload test |
| `takeSackFor(id)` reverted to the inventory subject in `settlePickup` | FAIL — "Carried(8) = [], want the sack's one code" |

**What these witnesses cannot see.** Neither composes a frame, so neither can see the `pickup` cursor
bitmap drawn at its hotspot; what is asserted is the name the production cascade selects.
`pkg/ui`'s witness stops at the seam and `pkg/game`'s starts after it, so the loader's own tuple
assembly is not asserted by either. That link is type-checked rather than witnessed: `MapGrab` is a
distinct function type among the eleven values of `MapOpener`, so a positional mix-up does not
compile, and both loaders pass `mw.grab` (`frontend.go:1300` and `1612`, forwarded through
`preparedMap.grab` on the resume path). A `nil` in that position would be silent, and nothing asserts
it is non-nil in production. Neither witness runs on real shipped mission content: `sim.World` has no
exported sack-drop, so a sack on a real campaign map cannot be placed from a test, and both fixtures
build their sack through `sim.NewLootWorld`. The last mutation in the table is what a fixture whose
subject and ordered unit are the same entity cannot see; it passed against a single-entity fixture,
and the fifth `pkg/game` test was written to kill it.

## Gate results

Each read from the gate's own exit code, not from a pipe.

| Gate | Result |
|---|---|
| `go build ./...`, `go vet ./...`, `gofmt -l`, `go test -count=1 -trimpath ./...` | clean, both repositories |
| `scripts/check-claim-citations.sh` | 0 — ok (1276 distinct citations resolve against 1479 claims and 220 experiments under 787 prefixes). Unmoved at round 4: `ITEM-PICK-016` and `AI-STATE-011` are newly cited by this story, and both were already cited elsewhere in the repository, so the DISTINCT count does not move. |
| `scripts/check-no-game-assets.sh` | 0 — clean (tree scan) |
| `research/scripts/check-claim-ids.sh` | 0 — ok (1479 ids, all distinct, 31 ledgers, 29 read back through `tools/claim`) |
| `research/scripts/check-regen-out.sh` | 0 — selected 11 `regen.sh`, ok (every one honours `OUT` and writes nowhere else) |
| `research/scripts/check-retraction-status.sh` | 0 — ok (237 overturned ids, every one marked) |
| `pipeline/check-div-claims.sh` (`AGAINROM_IMPL` at this worktree) | 0 — 220 live rows of 220, citing 284 distinct claim ids, at research pin `d7ee0c6`. It was 219 rows and 282 ids at round 3: `DIV-337` and `DIV-338` are new, `DIV-292` closed and left the file, and the two ids are `ITEM-PICK-016` and `AI-STATE-011`, newly cited by a divergence row. 54 rows cite a claim carrying a retraction row against 53 at round 3: `DIV-337` and `DIV-338` both enter the list and `DIV-292` left it with the file. Both new rows cite `AI-STATE-011`, whose retraction is about slot 22 alone and whose arm 2 -- the clause these rows lean on -- stands, and `DIV-337` cites `AI-CLICK-050`, whose retraction is the drag-discard clause and not the cursor-to-opcode mapping. `ITEM-PICK-016` carries no retraction row. |
| `pipeline/check-preserved-installs.sh` | 0 — ok, 162 files, both roots as recorded |
| `pipeline/check-scenarios.sh` | 0 — 15 of 15, en and ru. It was 14 before the round-3 merge of master; the fifteenth is story `1035`'s. |
| `pipeline/check-release-tests.sh` | 0 — 46 of 46 install-gated tests ran and passed, 0 skipped, en and ru. The population was 41 before this story and 43 after round 1; the two added here are this closure's witness, and the further three came in with story `1035` at the round-3 merge. |
| `pipeline/check-milestone.sh` (`AGAINROM_MILESTONE_DRIVE` at a `missionrun` built from this worktree) | 0 — the script gap and the drive are where they were recorded, both roots. Re-run at round 4 against a binary built from this branch. |

Script-gap census, en root, `cmd/missionrun -trace -ticks 1` built from this worktree: **mission 10
= 0, mission 20 = 0 `UNSUPPORTED`**. Both unchanged, and both re-measured at rounds 3 and 4. This story
moves no script node, and the claim that it does not is a measurement rather than an expectation.
The story's pointable result is therefore in `builds/current/` rather than in the census: what the
mission map does with a click, a drag and a key.

`git diff --diff-filter=D --name-only 1fcf7e0f..HEAD` is empty: nothing was deleted, across all
four rounds and both merges of master.

## Sweep: interactions with existing mechanics

**The button wording.** Fifteen production doc comments and eighteen test comments described the
order-issuing gesture as "a right press" or "a right click". That is the button this story inverted,
so every one of them became false. The population was enumerated with one `grep` over `pkg/` and
`cmd/` and closed in one pass. The sites that genuinely name the right button were left alone: the cancel
arm's own comment in `command.go`, and the ten test sites that assert the cancel or that the right
button never orders. `grep -rn "right press\|right click" --include=*.go pkg/ cmd/` now returns 15
lines — 1 in `command.go`, 10 in four test files about the cancel, and 4 in this story's own release
witness — and none of them describes the ordering gesture.

**The armed-mode precedence.** Four functions decide which mode is armed. Three disagreed once a
spell could be picked on top of an armed attack: `command.go`'s own chain, `missionMode`, and
`minimapModeCursor`'s switch. The map would dispatch a cast while the minimap drew and performed an
attack. All three now read `missionMode`.

The fourth, `commandPanelSelected` (`pkg/ui/commandpanel.go:160-175`, the command panel's own overlay
highlight), keeps its own precedence: `v.armed` (Attack) above `v.aimed == commandMove`, above
`v.aimed == commandSwarm`, above `v.selectedSpell != 0` (Cast). With an armed attack and a spell
picked on top of it, the cursor, the minimap and the click dispatch all say Cast while the panel
highlights the Attack cell. This is pre-existing (`git show 06d2dd48:pkg/ui/commandpanel.go` carries
the same precedence verbatim) and is carried outside this story, as a seat hotfix candidate rather
than a gap in it (F2, round 2 adversarial review).

**Divergence rows moved by this story.**

| Row | Move |
|---|---|
| `DIV-262` | CLOSED and moved to `DIVERGENCES-CLOSED.md`. The Ctrl and Alt cursors are implemented on `AI-CURSOR-226`'s own arms. |
| `DIV-263` | CLOSED and moved. The `town` arm exists and the mask is seven bits; what remains is split onto `DIV-284`, `DIV-285` and `DIV-289`. |
| `DIV-230` | amended. The Swarm cell left the skip mask; Defend and Retreat remain, on `DIV-288` and `DIV-291`. |
| `DIV-261` | amended. `sdefend` still has no armed state, and the reason is now `DIV-288` rather than `DIV-230`. |
| `DIV-292` | CLOSED at round 4 and moved to `DIVERGENCES-CLOSED.md`. Its own revisit condition was "a `MapGrab` that takes the acting entity, after which the order's own id is used", and that is what the seam now takes. |

**Fourteen new rows**, `DIV-283` through `DIV-296`, ten in the Divergences table and four in
"Authored where research is silent". The ledger stood at 196 live rows and 32 closed after the
round-1 merge, the 196th being master's own `DIV-319`.

**Two more at round 2**, `DIV-320` and `DIV-321`, both in the Divergences table.

**Nine more at round 3.** In the Divergences table: `DIV-328` (`F5`..`F8` quick spells),
`DIV-329` (`Ctrl`+numpad `+`/`-`), `DIV-330` (`Ctrl`+`W` formation), `DIV-331` (`Ctrl`+`F`/`L`/`U`),
`DIV-332` (settings captions) and `DIV-334` (the `Pause` key's phase gate and window, ACCEPTED). In
"Authored where research is silent": `DIV-333` (what "Show Health Off" removes), `DIV-335` (the
travel at which the outline starts being drawn) and `DIV-336` (the command panel's Cast cell paints
enabled and refuses the press). Three existing rows were corrected in place: `DIV-321` said "these
four" and listed three, `DIV-237` still said this build binds `H`, and `DIV-290` said `AI-CURSOR-230`
clause (1) was applied exactly. Read the live row count from `check-div-claims.sh` rather than adding
these by hand to the number above.

**Two more at round 4**, both in the Divergences table: `DIV-337` (the pick-up order is a
driver-tier latch and a plain move order rather than one order the actor carries, DEVIATION) and
`DIV-338` (three of `0x21`'s own stores and its refusal message are not reproduced, FIDELITY-DEBT).
`DIV-339` was reserved to this round and is unspent.

## Research reconciliation

| Claim | How it was used | Discrepancy |
|---|---|---|
| `AI-INPUT-121` | the primary button's press/marquee/act contract and the `screenW*10/640` threshold | none |
| `AI-INPUT-127` | the secondary button's capture, pan and cancel | none |
| `AI-CLICK-050` | the click-to-order table | two arms have no order in this build (`DIV-288`, `DIV-289`). The `pickup` arm was the third until round 4: it reached a seam that could only take a sack under the inventory subject's feet, so a click on any other cell was silent |
| `AI-CURSOR-226` | the ordinary-hover cascade at `L01515`, all five arms | none. It replaces `AI-CURSOR-052`, whose cascade an ordinary hover does not run. |
| `AI-CURSOR-230` | the replacements after the cascade | replacement (1), the marquee, is applied at the claim's own `IsRectEmpty` test since round 3; (2) and (3) are approximated by `groundSurfaceCaptures` and (4) and (5) are not applied (`DIV-290`). The travel at which the OUTLINE starts being drawn is this build's own choice, since no claim gives it (`DIV-335`) |
| `AI-CURSOR-231` | the seven-bit hover mask | the two structure bits are never set (`DIV-284`); the drawable bit is read as a ground sack (`DIV-283`) |
| `AI-CURSOR-242` | the pick-up gate, every term | none. Read whole at round 4 to settle which half of the reported defect was wrong: the gate carries no distance term, so the cursor over a distant sack was correct |
| `ITEM-PICK-016` | opcode `0x21` end to end: the arbitrary cell, the refusal, the walk, the take on arrival | the walk, the refusal and the take are reproduced by two mechanisms rather than by per-actor order state (`DIV-337`); three stores and the refusal string are not reproduced (`DIV-338`) |
| `AI-STATE-011` | arm 2 as the only consumer of `actor+0x50 == 2`, and `0xc` as acquire with no leash | the arrival test is the driver's, not the actor's (`DIV-337`); the completion state is not set (`DIV-338`) |
| `AI-CURSOR-209`, `PARTY-FLAG-003` | the `town` gate | written and unreachable (`DIV-285`) |
| `AI-SELECT-122` | selection's four forms and the group keys | the exact-name clause is read as "a unit and not a structure" (`DIV-284`) |
| `AI-PANEL-053`, `AI-PANEL-060`, `AI-PANEL-061` | the panel's ownership gate and the selection summary | three summary bits absent (`DIV-286`); `0x200` read as a mana pool (`DIV-287`) |
| `AI-PANEL-123` | the eight cells and their modes | Defend and Retreat disabled (`DIV-288`, `DIV-291`); Swarm reaches story `0146`'s advance order rather than a decoded `0x1a` (`DIV-230`) |
| `AI-MINIMAP-124` | the whole widget contract | the entity lookup is fog-gated, which the claim does not state; see the open items |
| `AI-KEY-125`, `keyboard.tsv` | the 57-row key sweep | `F1`/`F2`/`F3` vacated (`DIV-294`). Four keys this build had occupied were moved off the original's own assignments: the clock pause to `Ctrl`+`Space` (`DIV-293`), the damage numerals to `Ctrl`+`O` (`DIV-295`), and the doll and worn set to `J` and `X` (`DIV-296`). Round 3 moved day/night from bare `N` to `Ctrl`+`N` for the same reason. `keyboard.tsv`'s own `Pause` row (row 20) is a different subject from `DIV-293`: it is the original's modal-text key, and round 3 binds it, with the phase gate and the window recorded as `DIV-334`. Four groups are bound to nothing because nothing decodes what they do (`DIV-321`); five more are decoded and moved out of scope (`DIV-328`..`DIV-332`, see "Scope cut") |
| `AI-CURSOR-191` | the five small minimap cursors | `sdefend` unreachable (`DIV-288`) |
| `AI-CURSOR-126` | the Patrol default gesture | the original has none, so none was added |

No claim was contradicted. Every divergence above is this build lacking something, not this build
disagreeing about what ROM1 does.

## Open items

**Round 2's two reserved divergence rows are written.** `DIV-320` (minimap top-entity fog gate,
DEVIATION) and `DIV-321` (`keyboard.tsv` groups bound to nothing, FIDELITY-DEBT) are in
`docs/DIVERGENCES.md`. Both were reserved at round 2 rather than at this closure's first writing;
see the gate results section for `check-div-claims.sh`'s row count with them in place. `DIV-321` was
written saying "these four" and listing three; round 3 added the fourth, `Tab`'s character-panel
message `0x412` (`keyboard.tsv` rows 43 and 45), which is unbound in `pkg/ui` entirely.

**F10, the pointer scenario.** The headless harness has two disjoint stages (`pkg/game/headless.go`
`StageFrontEnd` / `StageMission`), and the `pointer` command driving `App.HeadlessPointer` — the
only way a scenario reaches a real mission-map hit test and `v.command`'s dispatch — is
`frontend`-stage only (`headlessCommands["pointer"]`, `frontEndAt(4)`). `assert_unit`,
`assert_world` and `order`, the only step vocabulary that reads a `sim.Entity`'s position or asks
what order it holds, are `mission`-stage only (`missionAt(2)`) and drive `*sim.World` directly,
never through `pkg/ui`. A scenario runs one stage; a file that opens on `activate`/`create_character`
(as a real campaign mission needs, to reach a party and a live `*ui.App`) cannot also call
`assert_unit`. `HeadlessMemberAssertion` (`pkg/game/headless.go:198-212`), the frontend stage's own
per-member assertion, carries XP, defense, absorption, load, capacity, temporary and skills — no
position, and no field naming what a dispatched order was. `HeadlessStateAssertion` adds screen,
purse, documents and member count, and `HeadlessInventoryState`/`HeadlessShopState` add inventory
and shop contents, none of them a mission entity's cell. `capture` stores a snapshot for
`same_character_as` alone, not a general before/after comparison.

A scenario can drive the gesture (a press, a short drag, a release, at a resolved surface, through
the same `v.step` a windowed session uses) and cannot observe its own effect: nothing at the
frontend stage reads a mission entity's position or its pending order back out. Building that
observable — a position field on `HeadlessMemberAssertion` or a new per-entity query, wired through
`FrontEnd`'s live `*sim.World` the same way `assert_inventory` already reads the container — is a
harness change beyond this round's scope and is left here rather than built silently narrower than
what the brief asked for. `pkg/ui`'s own unit-level tests are the witness for F1's fix today:
`TestTheOutlineAndTheReleaseAgreeOnOneThreshold` (`pkg/ui/command_test.go`) drives `v.step`/`v.command`
directly at five travel distances bracketing the old dead band and asserts what `v.command` returns,
mutation-proved against the reverted `overlay.go` guard. `check-scenarios.sh`'s count is unchanged by
this round.

**Not built, and not in scope.** Five decoded key groups were cut from G7 at round 3 and are named
above under "Scope cut", with a row and a follow-up each. The exclusions the contract already named
stand: town and shop input, the character card widget's controls (`DIV-217`), the inventory grid's
internal transfer paths, menu destinations that do not exist (`DIV-099`), and the Patrol default
gesture the original does not have.

**Not witnessed on screen.** No part of this story was driven through a windowed session on the
owner's desktop. The evidence above is headless: the two release tests on mission 10 from both
preserved roots, the fourteen scenarios, and the synthetic suite. A player-visible claim this closure
does not make is what the new `move` cursor over empty ground looks like in a running window.

**Round 4: what was not done.** The list is complete rather than short, because each item is a
candidate hotfix.

1. **`DIV-338`'s three stores and the refusal message.** `ITEM-PICK-016` names a destination slot
   `cmd+0x0e` written into `[actor+0x7c]+0x1c`, a completion state `actor+0x50 = 0xc` one tick after
   the take, and the engine string `"Sack not found at "` for a cell with no sack. None is
   reproduced. This build refuses silently and leaves the unit idle rather than acquiring. Whether
   the original shows that string to the player is not established by any claim, so the row records
   the string and not a display.
2. **`DIV-337`'s composition.** The cell and the arrival test are driver state on `mapWorld`, not
   per-actor order state, so they are not serialized and the cancellation is performed by
   `cancelPickup` rather than inherited from a shared `actor+0x50`. Moving them onto the actor is a
   `pkg/sim` change and a byte-form change, and it was deliberately not made.
3. **One intent at a time.** `mapWorld.pickup` holds a single `pickupIntent`, so ordering a second
   unit to a second sack replaces the first order rather than running both. The original's state is
   per-actor and has no such limit. This follows from item 2 and is not separately rowed.
4. **No witness on real shipped content.** `sim.World` exports no sack-drop, so a sack cannot be
   placed on a real campaign map from a test. Both fixtures build their sack through
   `sim.NewLootWorld`, and neither install-gated release test covers the pick-up click.
5. **No composed-frame witness.** The cursor assertions read the name the production cascade
   selects. Nothing renders the `pickup` cursor bitmap at its hotspot.
6. **The seam's own nil case.** The loader's tuple assembly is type-checked, not asserted: a `nil`
   `MapGrab` in that position would make the click silent again and no test would fail.
7. **Not driven in a window.** The defect was reported by the owner against his own build. The fix
   was not confirmed on his screen, only headlessly.
