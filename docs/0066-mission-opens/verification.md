# 0066 — verification

Seven tasks, seven commits, one per task. T1–T3 landed on `impl/0066-mission-opens`, merged at
`f65d766`; T4–T6 on `impl/0066-visible`; T7 on `impl/0066-script-join`.

This document states what was measured. **Most of what T4–T6 deliver is visible and no test can
reach it** — a box drawn over a map screen, its colours, its wrap on a real atlas, a mission opening
in front of a player. Those are written up as **built, not verified on screen**, and the owner's own
look is what closes them. Nothing below describes a screenshot that was not taken or a run that was
not made.

## The gate

Run from the `wt-0066-visible` worktree at `08e2bf5`.

```
go build ./...                       clean
go vet ./...                         clean
gofmt -l $(git ls-files '*.go')      prints nothing
go test -count=1 -trimpath ./...     all packages ok
sh scripts/check-no-game-assets.sh   check-no-game-assets: clean (tree scan)
sh scripts/check-doc-budget.sh       every artifact under its ceiling; spec >= plan >= tasks
sh scripts/check-sdd-audit.sh        FAIL set: this document's own absence, and nothing else
```

`check-sdd-audit` was run before this file existed and its only FAIL was
`0066-mission-opens: every task in tasks.md has landed and there is no verification.md`. Its
note/warning **counts are not comparable from a worktree** — the builds check is guarded by
`[ -d builds ]` and a worktree checks out no `builds/` — so only the FAIL set is reported.

Deletion set over the whole story:

```
git diff --diff-filter=D --name-only f65d766 HEAD    (empty)
```

`pkg/sim` is untouched across the story:

```
git diff --name-only a4f77c8 HEAD -- pkg/sim         (empty)
```

The research pin was **not** moved: `git submodule status` reads `f56bb38…` with no leading
character, the value it had at the story's start.

**Re-run for T7**, from the `wt-0066-join` worktree at `8d9427d`:

```
go build ./...                       clean
go vet ./...                         clean
gofmt -l $(git ls-files '*.go')      prints nothing
go test -count=1 -trimpath ./...     exit 0, 29 packages ok, no FAIL
sh scripts/check-no-game-assets.sh   check-no-game-assets: clean (tree scan)
sh scripts/check-doc-budget.sh       exit 0; 0066 runs under a DECLARED OVERRUN (spec/plan 14336)
sh scripts/check-sdd-audit.sh        exit 0, FAIL set EMPTY, 276 trailered commits checked

git diff --diff-filter=D --name-only 0fe4001 HEAD    (empty)
git diff --name-only 0fe4001 HEAD -- pkg/sim         (empty)
git submodule status                 f56bb38…, no leading character
```

`check-sdd-audit`'s note/warning **count is not comparable from a worktree** and only the FAIL set is
reported; 0066 appears in no note at all. The **declared overrun** is T7's own and is announced by
the gate on every run: `spec.md` and `plan.md` were at 99% and 99.9% of the tight ceiling, so adding
a requirement meant deleting one. A compaction pass ran first and returned 286 B in `spec.md` and 73
in `plan.md` against 883 B and 825 B of new material; the row in `select_ceilings` carries the
measurement and the reason.

## THE GAP — CLOSED BY T7, and named here because it decided what this story delivers

**It was:** a mission's world carried no compiled script. `mapload.StartMission` builds through
`sim.NewWorld`, which attaches none, and nothing in the tree attached one afterwards. Measured on a
lawful root, not inferred: over a started mission `World.Script()` was nil, so no trigger was ever
evaluated, no latch ever set, and **no announcement could fire and no mission could end**.

It was not a defect introduced here: `0063` and `0065` each put the piece out of their own scope, and
`0066`'s FR-6 and FR-10 describe what happens *when* a trigger fires while no clause said the world
runs its script — and "Out of scope" did not name it either. **A presumed join that nothing contracts
is a hole, not a cut**, so on the owner's ruling the story that presumed it closes it, as T7 rather
than a new number. `spec.md` gains **FR-12** and **AC-25**/**AC-26**, `plan.md` **DD-18** and
**SC-16**.

The join is one construction — the world rebuilt through `sim.NewScriptedWorld` with the program
`game.StartMission` already compiles — in `pkg/mapload`, where the seed, the block plane and the
bounds already live. It is a **sibling**, `StartMissionScripted`, not a change to `StartMission`:
changing that in place would have broken `TestStartMissionWithNoPartyBuildsThePlainWorld`, whose
stated purpose is catching *"a start that changed a world nobody asked it to change"* — and that test
is right, because a map opened with no mission still wants a world running nothing. Both entry points
stay and the plain one is untouched.

## What each success criterion rests on

| SC | Verdict | What was measured |
|---|---|---|
| SC-1 | **built, owner's look** | `TestMissionOpenerShowsTheMapScreen` drives the door to `ScreenMap` over a synthetic install; the windowed run on both roots started and stayed up. Nobody looked at the screen |
| SC-2 | **verified** | Three failures, three distinct messages, each exit 1 — `TestMissionFlag/the three startup failures…`, and on the EN install for the two that a lawful root can reach |
| SC-3, SC-4, SC-5 | **verified in T1** | unchanged by T4–T6; the package's existing rendering tests remain unedited |
| SC-6, SC-7 | **verified in T2** | unchanged |
| SC-8 | **verified in T3** | unchanged |
| SC-9 | **verified** | `TestARaiseWhileANoticeIsOpenIsDiscarded`: the second raise is dropped, closing the first shows nothing, and four further cycles show nothing |
| SC-10 | **verified** | `TestAdvancingADialoguePagesThenCloses`: parts 1, 2, 3, then closed |
| SC-11 | **verified** | `TestNoticeLinesBreaksAtTheLastSpaceThatFits`, `…BreaksInsideAnOverWideWord`, `TestNoticeLayoutClampsToTheArea` — every expectation is arithmetic over the test font's own advances |
| SC-12 | **verified** | `TestTheOutcomeNoticeIsShownOnceAndReplacesADialogue` for both endings; `TestNoticesDoNotReachTheWorld` compares 64 ticks and 64 digests against a driver producing none |
| SC-13 | **verified** | `TestAdvancingTheOutcomeSelectsItsDestination`; the front-end half in `TestAdvanceNavigatesTheSeamsDestination` |
| SC-14 | **verified** | the two `git diff` commands above |
| SC-15 | **verified** | `TestNoticeLayoutIsPure`: eight identical calls agree, and a halved text area does not |
| SC-16 | **verified** | `TestStartMissionScriptedRunsTheScript` (latch set and `OutcomeWon` reached after one 16-tick cycle), `TestStartMissionScriptedWithNoScriptBuildsThePlainWorld` (hash and report equal to the plain start's, for a party of none, one and five), `TestStartMissionRunsTheMissionsScript` at the door, and the run on **both lawful roots** below |

## Properties

| ID | Verdict | What was measured |
|---|---|---|
| P-1 | **verified** | `git diff --name-only a4f77c8 HEAD -- pkg/sim` is empty; the byte form stays at version 10 and gains no field; `pkg/sim`'s own byte-form and digest pins pass |
| P-2 | **verified in T3, unchanged** | `TestAnnouncerRaisesAOneShotExactlyOnce`, `TestAnnouncerSeesEveryNonAdjacentFiring`, and the conceded merge in `TestAnnouncerMergesAdjacentFiringsOfARepeatingTrigger` |
| P-3 | **verified in T1, unchanged** | selector 0 is the identity and the text package's existing rendering tests are unedited and green |
| P-4 | **verified** | `TestNoticesDoNotReachTheWorld` runs two drivers over one script — one producing notices, one producing none — and compares the tick and the digest at each of 64 steps; `TestSamplingDoesNotReachTheWorld` does the same for the sampler alone |
| P-5 | **verified** | `TestNoticeLayoutIsPure`; the composition takes a layout, a font and a string and reads nothing else, and `RenderNotice` needs no viewer, window or graphics context |

## Acceptance criteria — where each is closed

**Closed by a test:** AC-2, AC-5, AC-6, AC-7, AC-8, AC-10, AC-11, AC-12, AC-13, AC-14, AC-15,
AC-18, AC-19, AC-20, AC-21, AC-23, AC-24.

**Closed by the owner's look, built here:** AC-1 (the map screen opens on the named mission — driven
to `ScreenMap` in a test, never seen), AC-3 (the menu opens with no flag — the check-mode half is
tested, the window is not), AC-9 (Escape advances and does not leave — driven through `App.step`,
never seen), AC-16 and AC-17 (the two destinations — the seam and the flow are both tested, the
screens are not), AC-22 (a fontless front end — `TestNoticeNeedsAFont` pins that nothing is
presented and that Escape is not swallowed; that no pixel is drawn follows from the same gate and
was not observed).

**AC-4** is closed by `TestTheMissionPartyIsOneMemberAtTheStartCell` and by the headless report on
both lawful roots: mission 10 builds 36 entities and puts the party at (17, 66), identically on EN
and RU.

**AC-25** is closed by `TestStartMissionRunsTheMissionsScript`, which drives `game.StartMission`
over a synthetic archive whose type-7 payload carries a message action, two constants and a
fire-once trigger comparing them: after one 16-tick cycle latch 0 is set and the announcer raises
event 7. `TestStartMissionScriptedRunsTheScript` closes the same criterion in `pkg/mapload` and goes
one further — the trigger's win instant runs, so the world reaches `OutcomeWon`. **AC-26** is closed
by `TestStartMissionScriptedWithNoScriptBuildsThePlainWorld`: for a party of none, one and five the
scripted start with a nil program hashes exactly what `StartMission` hashes and reports the same
`Start`.

## What was run against the lawful installs

Both roots, on 2026-08-02. `builds/0066-mission-opens/README.md` carries the exact invocations.

- **Headless, both roots:** `-check -mission 10`, `20`, `30`. Every number reported — extent, entity
  count, start cell, raise count — is **identical on EN and RU**. Mission 10: `80x80, 36 entities,
  party at (17, 66), 11 raise(s)`.
- **Failures, EN:** `-mission -3`, `11`, `999` each exit 1 with the message the tests pin.
- **Windowed, both roots:** `-mission 10` ran for twelve seconds under a timeout, exited only when
  killed, and printed nothing. **That is the entire measurement.** No screenshot was taken and the
  screen was not looked at.

`scenario/1.alm` ships on neither root: the campaign's own numbers begin at **10**.

### T7 — does a trigger actually evaluate on a lawful install?

Both roots, on 2026-08-02. The shipped headless mode still reports what it reported, unchanged:

```
$ againrom -assets <EN> -check -mission 10
againrom: 38 map rows, 8 of 8 buttons have a mask region
againrom: mission 10 at scenario/10.alm, 80x80, 36 entities, party at (17, 66), 11 raise(s)
$ againrom -assets <RU> -check -mission 10
againrom: 34 map rows, 8 of 8 buttons have a mask region
againrom: mission 10 at scenario/10.alm, 80x80, 36 entities, party at (17, 66), 11 raise(s)
```

`-check` does not step the world, so it cannot answer the question. A **throwaway developer tool**
was used for that and then deleted, unbuilt and uncommitted: it starts mission 10 through
`game.StartMission`, builds a `game.Announcer` over `Mission.Raises`, steps 320 ticks — twenty whole
script cycles — and samples per step. Its output was **byte-identical on EN and RU**:

```
mission 10 at scenario/10.alm: 80x80, 36 entities, party at (17,66)
RaiseErr=<nil>  raises=11
World.Script() != nil: 16 check(s), 27 instant(s), 12 trigger(s)
  tick 7: ANNOUNCEMENT event 15
  tick 7: ANNOUNCEMENT event 16
  tick 7: trigger latch 0 SET
  tick 7: trigger latch 12 SET
after 320 ticks: tick=320 outcome=0 won=0 lost=0 latches-set=2 announcements=[15 16]
registers 0..9: 0 0 0 30 0 0 0 0 0 0
```

**The machinery runs.** The first script pass lands on tick 7, two of mission 10's twelve triggers
hold on it, both latches are set and both announcements are raised. Register 3 holds 30, which is a
constant check the compile preset — the world is evaluating the program, not merely holding it.

The **counterfactual was measured on the same install rather than reasoned about**: the same tool
built through `mapload.StartMission`, which is what `pkg/game` used before T7, and printed
`World.Script() == nil -- THE WORLD RUNS NO SCRIPT`. The defect and its fix were both observed on a
lawful root.

**The mission does not end, and that is expected.** `outcome=0`, both counters 0 after twenty
passes. Mission 10's own first link needs two brigands dead and units carry no combat numbers yet —
a different story, in flight. What T7 claims is that checks evaluate, triggers fire and
announcements are raised; a **win was not observed on a real install** and is not claimed.

All eleven authored events resolve to real text on both roots — `found=true` and a non-empty part 1
for every one of `event01`, `02`, `03`, `04`, `10`, `14`, `15`, `16`, `17`, `18`, `19` — so the two
that fired have words to show. **Nothing was drawn on a screen for this task**: no windowed run was
made and no notice was seen. That the raised announcement reaches a notice is T4/T5's tested wiring,
not something this task observed.

## Divergences and cuts, each named

- **DD-2's merge**, unchanged from T3: a repeating trigger firing on two consecutive passes raises
  once. Measured in `TestAnnouncerMergesAdjacentFiringsOfARepeatingTrigger`.
- **The break class is ours.** `DLG-WRAP-009` fixes the rect, the pitch and the clamp; where a line
  may break is this project's. Tab, CR and LF join the space, because the atlas has no record for
  one — every byte below `0x20` selects the space glyph — so a byte that drew as a space and did not
  break as one would make a two-line authored paragraph a single unbreakable word.
- **The outcome notice's box, colours and words are ours**, and the spec says they are. The two
  endings are distinguishable by their words alone; the frame is one box.
- **A script that will not decode is not fatal.** `Mission.RaiseErr` carries the reason and the
  mission still starts. That is the mission-start path's own behaviour — `DropCells` already answers
  a body it cannot tile with no cells and no complaint — rather than a softening of it.
- **The first startup failure names the number, not an address.** FR-1 asks each of the three to name
  "the address it failed at"; a number that is not a mission number has no address, and the message
  names the number instead. The other two name the address.
- **CLOSED IN T7 — was: no test drives `game.StartMission` over a map that actually carries a
  raise-message action.** `internal/synth` could not build a type-7 body. It can now
  (`ALMOptions.Type7Payload`), and `TestStartMissionRunsTheMissionsScript` drives the whole wiring
  `StartMission → CompileScript → StartMissionScripted → Announcer` over a synthetic archive. The
  zero value is still the payload the builder produced before, so the won't-decode case is unchanged.
- **CUT: no pixel of the notice was compared against anything.** `TestNoticeIsComposedAtItsAuthoredSize`
  proves the picture is the authored box and that no ink leaves it; what it looks like is the owner's.

## Found while doing this — FIXED IN T7's pass, in its own untrailered commit

`pkg/render/text/text.go`'s `Convert` doc said the conversion "lands where no unmoved byte sits, so
the map is **INJECTIVE over the whole 256**". This story's own `provenance.md` records that the claim
this repeats is **wrong**: `0xB0..0xDF` and `0xF0..0xFF` are themselves unmoved, so each of their 64
bytes shares a record with the byte that moves onto it and the map is 64-to-1 over the full range. It
is injective on ASCII plus the two moved blocks, which is the domain `TEXT-FIT-004` measured the
Russian corpus to occupy exhaustively — so the consequence is sound for shipped data and only the
justification is false. `SC-4` states both halves correctly, and `convert_test.go` already pinned
both; only the comment still asserted the falsehood.

The T4–T6 lane found it and correctly declined to edit a landed task's file from a later task's
commit. It is fixed now as a **comment change and nothing else**, in a commit carrying no trailer
because it belongs to no task. `SC-4` was **not narrowed** — research has an open round on the same
question, and the comment now says what `SC-4` says and stops there.
