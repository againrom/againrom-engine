# Verification — 0097

Branch `story/0097-commanded-group`. Two trailered commits, `T1` and `T2`, one each. Research pin
`4920d6d`, `git submodule status` showing no leading character. Every figure below was produced from
this branch, on a clean tree, on **both roots** — `gameversions/en` and `gameversions/ru`.

## Gate

    go build ./...                       clean
    go vet ./...                         clean
    gofmt -l pkg cmd internal            prints nothing
    go test -count=1 -trimpath ./...     every package ok, with no install present
    scripts/check-no-game-assets.sh      clean (tree scan)
    scripts/check-doc-budget.sh          exit 0
    scripts/check-sdd-audit.sh           no FAIL for this story once this file exists

Run from the orchestrator seat on the committed tree, not from a working directory with edits in it.

## The acceptance criteria

All in `pkg/sim/commanded_test.go` unless named otherwise. None reads an install.

| | Witness |
|---|---|
| AC-1, AC-2 | `TestACommandedUnitKeepsWalkingPastAHostileInReach` — one shared moment read from both sides: the commanded unit keeps its destination and takes no victim, and the hostile group takes *it* |
| AC-3 | `TestACommandedUnitStaysACandidateForOthers` — the same world with the destination removed, which is the control that says the fight was available |
| AC-4 | `TestACommandedFighterStillDecidesForItsGroup` — a unit holding a victim *and* a destination is still scored |
| AC-5 | `TestArrivalEndsCommandAndTheNextDecisionEngages` |
| AC-6 | `TestBeingStruckDoesNotEndCommand` |
| AC-7 | `TestACommandedGroupContributesNoSight` — two subtests: no member is given a victim, and a third group whose only line of sight ran through the commanded member gets nothing |
| AC-8 | `TestARepeatedMoveOrderStaysUnderCommand` |
| AC-9 | `TestAnAttackCommandEndsCommand` |
| AC-10 | `TestFallingEndsCommand` |
| AC-11 | `internal/archtest`: `TestDestinationWritersMatchFR2` over the live tree, `TestDestinationWriterDiffNamesWhatChanged` and `TestCheckDestinationWritersPairsByPosition` over literal source |
| AC-12 | `TestACommandedWorldRoundTripsAndHashesToTheSameDigest` — round trip byte-identical, version pinned as the literal **19**, digest pinned as the literal **`0x8ee638a276402889`** |
| AC-13 | `TestNoCommandedUnitLeavesTheWorldUnchanged` |
| AC-14 | No new refusal was written. The state is derived from two fields the byte form already constrains, and the decoder's existing refusals are untouched — `pkg/sim`'s decode suite is unchanged and green |
| AC-15 | `pkg/sim/partyslot_test.go`: `TestAPlayerOrderNeedsNoReissuePastAHostileThatCloses`, the rewrite of the previous story's FR-6a measurement |

**Witnessed by reverting, not by reading.** With `underCommand`'s clause removed from `aiGroups`,
AC-1 and both subtests of AC-7 fail; with the predicate weakened to `HasTarget` alone, AC-4 and AC-9
fail. AC-15 fails on all four of its assertions with the clause removed — the single-order drive
reaches (6,4) instead of (12,5), sits on the hostile's contact cell, and holds a victim. AC-11 was
witnessed in both directions: a fifth writer added to `pkg/sim/combat.go` produced *"pkg/sim gained a
destination writer FR-2 does not name: pkg/sim/combat.go:262 in sendHome …"*, and deleting
`groupOrder`'s write produced *"FR-2 writer \"groupOrder\" is gone from pkg/sim …"*. Every revert was
restored and `git diff pkg/sim` was empty before the commit.

AC-2, AC-3, AC-5, AC-6, AC-8, AC-10, AC-12 and AC-13 were **not** shown to fail under those two
reverts, and are not claimed to be witnesses for them — they test other writers and other rules.

## The success conditions

### SC-1 — the reported defect

`missionrun -mission 10 -waypoint p0:25:55:0 -ticks 400`, EN and RU identical:

| | Result |
|---|---|
| before | `STOPPED SHORT of (25,57), Chebyshev 2, after 400 ticks`, `outcome undecided at tick 464` |
| after | `reached (25,55), Chebyshev 0, after 195 ticks`, `outcome undecided at tick 259` |

The *before* figure was taken by reverting the one clause on this branch and restoring it, so the two
rows differ in that clause and in nothing else.

**The instrument had to be corrected first, and this is the honest part of SC-1.** The brief's and
the previous story's figure used radius **1**, and with a radius the tool aims at the open cell
within it nearest the mover — `blocked` reads terrain only. On this map that cell is (24,56), which
a unit of slot 3 is standing on. So `-waypoint p0:25:55:1` prints `STOPPED SHORT of (24,57)` both
before and after this change: it measures the tool's aim colliding with a body, not the simulation.
The radius-0 drive is the one that measures the story. Not fixed here — `aim` reading occupancy is a
tool change, and this story touches no tool.

### SC-2 — a unit not under command still fights

Both halves come out of the one radius-0 drive above, both roots identical, printed from a probe
that was not committed:

    tick 155  hero (25,57)  commanded=true   victim=false  | hostile id 0 slot 3 at (24,56) Chebyshev 1
    tick 175  hero (26,56)  commanded=true   victim=false  | hostile id 0 slot 3 at (24,56) Chebyshev 2
    tick 195  hero (25,55)  commanded=false  victim=false  | arrival
    tick 199  hero (25,55)  commanded=false  victim=TRUE   | hostile id 0 slot 3 at (24,56) Chebyshev 1
    tick 279  hero (25,55)  commanded=false  victim=TRUE   | hostile id 1 slot 3 at (26,56) Chebyshev 1

For twenty ticks the hero stands **one cell** from a live hostile, under command, and is given
nothing — FR-6, and it is the law's own answer rather than a simplification: being struck issues no
order there. Four ticks after the order ends it engages that same hostile, and later a second one.
So the population that keeps fighting is *every unit not currently executing a move order*, which is
every unit in the build except during the walk it was told to make.

### SC-3 — the authored placements at the player's slot

Over every mission both roots ship, counted through this tree's own loader with no party placed:

    map 41   slot-1 placements=3  holding-a-destination=0
    map 71   slot-1 placements=4  holding-a-destination=0
    map 150  slot-1 placements=1  holding-a-destination=0
    map 151  slot-1 placements=1  holding-a-destination=0
    total 9

Identical on EN and RU, and identical before and after the change. **Nine, on those four maps**,
reproducing the published census through an independent path. None holds a destination at load, so
none is under command, so each takes the stance its owner slot selects — Stand Ground — by the rule
this story did not touch. That the count is a fact about the maps rather than about the change is
what makes the "still Stand Ground" claim checkable.

### SC-4 — mission 10's outcome

`missionrun -mission 10 -waypoint u21:56:21:3 -waypoint p0:66:16:3`, EN and RU identical:

| | Waypoint 1 | Outcome |
|---|---|---|
| before | `STOPPED BY THE WORLD DECIDING, short of (43,46), Chebyshev 25, after 272 ticks` | `lost at tick 272` |
| after | `STOPPED BY THE WORLD DECIDING, short of (39,41), Chebyshev 20, after 272 ticks` | `lost at tick 272` |

**The outcome did not move.** `-trace` names the same arm at the same tick in both runs:
`tick 262 check 12 vip(u21) COUNTED A LOSS: its unit is not alive`, reported at 271, lost at 272.
The escorted unit dies either way; what changed is that she gets five cells further first, and her
health at the moment the check reads her (−1 before, −2 after). No test expectation, threshold or
predicate was edited — `TestTheTenthMissionIsDrivenToAWin` is untouched, and the story's diff shows
it.

## What is disclosed rather than fixed

- **A commanded unit does not stop to fight.** FR-6, and SC-2's first three rows are what it looks
  like. It is the ledger's answer, not a shortcut, but it is a behaviour change a player will notice.
- **An arrived unit returns to its group's stance**, where the original hands it to a per-unit
  acquisition. Both admit only a candidate within reach; they can differ on *which* when two stand
  within reach at once. FR-8.
- **A commanded member's sight leaves its authored group.** FR-5, AC-7. Unobservable on the campaign
  party today, which places one member.
- **The archtest check is lexical.** It sees assignments of the literal `true` to a selector named
  for the destination flag, pairing left and right by position. It cannot see a destination restored
  through a whole-struct copy or returned from a call, and it silently skips a file that will not
  parse — a path the live tree cannot reach, since `go build` compiles the same files. Stated in the
  check's own doc and not tested.

## Build

`builds/0097-commanded-group/` holds `againrom.exe` and `missionrun.exe`, both `go build -trimpath`
from this branch, and a `README.md` whose every command was run from that directory before it was
written down. `builds/` is gitignored; nothing there is committed.
