# 0131 — the campaign advances: plan

## Approach

The campaign reads its own routing key and keeps it apart from the ascending
order this tree authored; the notice seam grows a third destination that carries
a map to open; the mission-end decision chooses between the two; and the check
mode reports the whole thing without a window.

The seam's widening is a **type** change and the decision that uses it is a
**behaviour** change, and they are landed apart: the widening is behaviour-
preserving in every package that names the seam — every producer hands back
nothing and the one consumer can already act on something — so the tree stays
green and the game identical, and what changes the game afterwards reads as one
diff. Widening a type is not local, so that first, mechanical step reaches
`pkg/game` as well as `pkg/ui`.

## Facts verified during planning

- `FinishMission` already captures the party unconditionally and before it looks
  at the campaign at all, landing it in `FrontEnd.Carried`; `NextParty` reads
  that field and `MissionOpener` calls it, so every mission this front end opens
  by number already starts with the carried party. **FR-4 is satisfied by the
  door T3 opens, not by new carrying code.**
- `FinishMission` composes **four** sentences today. `continuity_test.go` pins
  two of them verbatim, the third by substring, and the carry on all of them;
  `flow.advanceNotice`'s two destinations are pinned by `pkg/ui/notice_test.go`
  and `pkg/ui/halt_test.go`, and `notice_test.go` also pins `ui.MapAdvance`'s
  exact type by reflection. Those are the characterization pins this brownfield
  change stands on and they are green at the baseline; no task re-creates them,
  and the one sentence with no verbatim pin gains one where it is changed.
- `ui.MapAdvance` is the **sixth** member of `ui.MapLoader`'s eight-member
  tuple, so widening `MapAdvance` itself moves no position there: every loader
  literal keeps compiling, and only the seam's own implementations and its call
  site change.
- `MissionMap` refuses `n <= 0`, composes an address for every other `n`, and
  consults no registry — so a successor with no `[Mission<n>]` section still has
  an address, and FR-7's boundary is the sign, not the section list.
- `pkg/ui/app.go`'s map-screen arm returns early after an advance on the test
  `a.flow.screen != ScreenMap`. Every dismissal that goes anywhere today goes to
  another screen, so that test means "the advance navigated" **only while no
  destination is a map screen**.
- `Campaign.NextMission`'s ascending order is this tree's own and its doc says
  so. The only reader of `AutoGetMission` anywhere is `campaign_test.go`'s
  cross-check, whose comment calls reading the key as "the mission granted when
  this one ends" a fact this tree is not entitled to — true against
  `REG-SCN-059`, and superseded.
- Every route in `app.go` that adopts a viewer calls `syncViewerLayout` on the
  same statement — `OpenMission`, both `choose()` sites, the generation screen's
  door, the **Escape** dismissal. `App.Layout` is the engine's callback, so the
  viewer a frame's `Update` adopts is not the one that frame's `Layout` sized;
  and an armed start view is applied in `Viewer.Layout` and nowhere else.

## Files to touch

| Path | Intent | Serves |
|---|---|---|
| `pkg/game/campaign.go` | MODIFY | FR-1, FR-2 |
| `pkg/game/campaign_test.go` | MODIFY | AC-1, AC-2, AC-3, P-4, and the cross-check whose stated reason `REG-SCN-063` supersedes |
| `pkg/ui/notice.go` | MODIFY | FR-3 |
| `pkg/ui/flow.go` | MODIFY | FR-3, FR-8, FR-10 |
| `pkg/ui/app.go` | MODIFY | FR-9, and the layout the new viewer needs |
| `pkg/ui/advance_test.go` | ADD | AC-9, AC-13, P-3, P-5, P-6 |
| `pkg/ui/notice_test.go`, `pkg/ui/halt_test.go` | MODIFY | the seam's pinned type and its two test producers |
| `pkg/game/world.go` | MODIFY | the driver's own producer of the seam |
| `pkg/game/world_test.go`, `pkg/game/frontend_test.go` | MODIFY | the same, from the tests that call it |
| `pkg/game/frontend.go` | MODIFY | FR-3, FR-4, FR-5, FR-6, FR-7, FR-11 |
| `pkg/game/continuity_test.go` | MODIFY | AC-4, AC-5, AC-6, AC-7, AC-8, AC-12, P-1, P-2 |
| `cmd/againrom/main.go`, `cmd/againrom/main_test.go` | MODIFY | FR-11 |

## Design decisions

**DD-1 — the successor is a lookup on `Campaign`, beside `Main` and `Side`, and
never inside `NextMission`.** `Campaign` gains `Auto map[int]int`, holding an
entry only for a section that declares a successor, read by `AutoAdvance(n)
(int, bool)`. *Rejected:* folding it into `Offer`/`NextMission` so a caller asks
one question — FR-2 forbids exactly that, since one answer is the file's and the
other ours, they agree on a stock campaign, and one question is how they would
come to be confused. *Rejected:* storing the raw value including `-1`, which
puts the meaning of `-1` in as many places as there are callers.

**DD-2 — presence is read, not inferred from zero.** `sectionInt` answers 0 for
an absent key and for a key holding 0 alike, which is fatal here: 0 is a value
the fork treats as a successor and the sign test then refuses. A presence-aware
`sectionIntOK` is added and `sectionInt` defined in terms of it. *Rejected:*
`sectionInt` answering `-1` on absence — it also reads `TotalMissions`, where
`-1` would be a different lie.

**DD-3 — `FinishMission` returns the successor beside the sentence, zero means
none, and `Offered` keeps meaning what the MAP LIST was told.** Its shape
becomes `(int, string)`. Zero is unambiguous **because** FR-7 makes a
non-positive successor unopenable, and the refusal lives in this one function —
the only place that can compose FR-7's sentence — so no number confusable with
"none" ever leaves it. *Rejected:* a third `bool` result, a second way to say
what the sign already says.

`Offered`'s writers stay the paths that reach the map list; the advance reaches
none, so it leaves the field at nothing, its own value before the first win, and
its doc says that instead of implying a third meaning. *Rejected:* writing the
successor there — the field is fed today from `NextMission`, this tree's
ascending answer, so one field would hold the file's fact on some wins and ours
on others: the conflation FR-2 forbids, invisible on a stock campaign where the
two agree. *Rejected:* deleting a field nothing reads, a change to a shipped
type this story does not own.

**DD-4 — the seam carries an opener, not a mission number.** `ui.MapAdvance`
becomes `func() (NoticeDest, string, MapOpener)` and `NoticeDest` gains
`NoticeToMission`. `pkg/ui` cannot name a mission: its `MapLoader` is keyed by
**row index**, so a number crossing here would need a second loader keyed by
mission — a second door with its own rules about what it opens with. An opener
is the shape this package already accepts from `OpenMission` and the generation
screen. *Rejected:* a **ninth** member on `MapLoader`'s tuple; that tuple's own
documentation rejects the move once already, for the spell id, in favour of
widening a seam that already fires on the event in question — and this event is
precisely a notice advance. *Rejected:* fetching the opener by a second call,
which would let the destination and the thing to open disagree.

**DD-5 — the tier that knows the campaign refuses; the tier that draws never
inspects a number.** `NoticeToMission` is returned only with a non-nil opener
onto a positive mission, so `pkg/ui` has no sign test, no zero check and no idea
what a mission number is. Two things follow on that arm. **An opener returning
an error** puts the front end on the map list holding `err.Error()` and **not**
the seam's sentence: that sentence says the successor opens, and printing it
over a failure says the opposite of what happened — `choose()` already answers a
failed load this way and FR-8 asks for the same report. **A nil opener** is
handled with the seam's sentence, because a destination whose payload is missing
must not leave the player on a torn-down screen. Both arms set the screen
explicitly: `leaveMap` clears the viewer and the seams and does **not** touch
`screen`, so an arm that only tore down would leave a map screen with no viewer
— P-2's violation exactly.

**DD-6 — the advance enters through the one `enter` the picker and the mission
door already use.** `leaveMap()` first, so the ended mission releases its
viewer's command mode and its seams on the statement it is left on; then
`enter`, which re-establishes the cadence ladder, the far-side record and the
command mode. *Rejected:* a third entry path — what the generation screen's door
already rejected, for the same reason: skipping the cadence rung opens a mission
whose first speed key slams the clock.

**DD-7 — the frame guard compares the viewer, not the screen, and lays the new
one out before returning.** `app.step`'s map arm captures `a.flow.viewer` before
dismissing; if it differs afterwards the arm calls `syncViewerLayout` and
returns. The screen test standing there holds only while no destination is a map
screen, the exact assumption this story breaks — so without the change the frame
runs on against the successor, and without the sync the successor is the one
route into a map screen that never sized its camera to its window nor applied
its armed start view. *Rejected:* clearing `in.Enter` beside the already-consumed
click, which answers for two of three inputs and none of the rest of the frame.
*Rejected:* leaving the sync to the engine's next `Layout` — every other route
lays out on the statement it adopts a viewer, and a rule with one exception is a
rule nobody can apply.

**DD-8 — the headless report runs the real decision over a mission started and
not won.** `FrontEnd.AdvanceLine` starts the mission, calls `FinishMission` on
its world, and where a successor comes back starts **that** mission with the
party the call carried. All of that is the shipped path; what it does not
exercise is the **recognition** of the win — `continuity` reading the world's
outcome — which the unit tests witness over a world that really was won. The
line prints wherever the check mode is asked about a mission, so the one test
pinning that mode's output moves with it. *Rejected:* forcing an outcome onto a
world: nothing above `pkg/sim` can write one, and a faked report is evidence
about the fake.

## Risks

**R-1 — a player who does not want the successor.** The advance takes the choice
away for one mission. Escape still leads to the map list and every row there is
still choosable, so nothing traps him — but he arrives before he is asked.
Accepted: it is what the campaign declares.

**R-2 — the dismissing press acting twice.** The press that closes the banner is
in hand when the next mission's viewer is adopted, and the map screen's arm was
written when that could not happen.

**R-3 — a campaign declaring a cycle.** Nothing forbids `10 -> 20 -> 10`. Each
advance is a fresh load driven by a win, so it costs a mission per step and
accumulates nothing: a campaign that does not end, not a hang. Not defended
against — a limit would be ours and would break a lawful mod.

**R-4 — the pause between banner and next map.** The successor's map is read and
decoded on the dismissing frame, the same cost a picker row pays on its click.


## Success criteria

| # | Condition | How |
|---|---|---|
| SC-1 | Mission 10 declares successor 20 and nothing else does, on a fixture and on both lawful roots | unit; the check mode |
| SC-2 | A section holding `-1` is indistinguishable from one holding no key | unit |
| SC-3 | Dismissing a won mission that declares a successor leaves the front end on that mission's map screen, never on the map list | unit, through a stub opener |
| SC-4 | The hero in the opened successor carries the experience, pack and worn set his entity ended the won mission with | unit |
| SC-5 | Every path off a win carries the party: successor, none, non-positive, unreadable map, no campaign | unit, a table over the five endings |
| SC-6 | A non-positive successor and an unopenable map produce different messages, both on the map list | unit |
| SC-7 | No input of the dismissing frame reaches the opened mission | unit, over the front end's own step |
| SC-8 | A second dismissal opens nothing | unit |
| SC-9 | `-check -mission 10` on each lawful root reports successor 20, opens it, and reports the hero's pools and skill experience there; `-mission 20` reports no successor | developer run, both roots, verbatim |
| SC-10 | `go build`, `go vet`, `gofmt`, `go test -count=1 -trimpath ./...`, and the four repo scripts | developer run |
| SC-11 | The successor's map screen holds the real window's view and the start view the map list's own door would have given it, on each dismissing input | unit, over the front end's own step |

## Traceability

FR-1, FR-2, AC-1, AC-2, AC-3, P-4 → SC-1, SC-2, at DD-1 and DD-2.
FR-3, AC-4, AC-13, P-6 → SC-3, SC-11, at DD-4, DD-5, DD-6, DD-7.
FR-4, AC-5 → SC-4, at DD-3. FR-5, FR-6, AC-6, AC-12, P-1 → SC-5, at DD-3.
FR-7, FR-8, AC-7, AC-8, P-2 → SC-6, at DD-3, DD-5, DD-6.
FR-9, AC-9, P-3 → SC-7, at DD-7. FR-10, P-5 → SC-3, SC-8, at DD-3, DD-6.
FR-11, AC-10, AC-11 → SC-9, at DD-8. Every criterion also stands on SC-10.
