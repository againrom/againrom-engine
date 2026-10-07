# Verification — 0096, the mission script's group command

## A DECLARED OVERRUN, and what it buys

`bash scripts/check-doc-budget.sh` is **RED on this story and deliberately so**: `spec.md` is
15131/13312 and `plan.md` 15130/13312, both over by 13%. `check-doc-budget.sh` was **not edited**.
The `spec ≥ plan ≥ tasks` chain is intact — the overrun is against the absolute ceilings alone.

Compression was spent first, twice: the *why* prose was moved out of the divergences into
`provenance.md` per S-7, and prose restating a claim's content became a citation. What is left will
not fit, and these are the clauses that will not:

- **FR-22 / AC-20.** `0097` landed `underCommand` and its own doc names the writer that would break
  its exactness — "any script-ordered move". This story **is** that writer. Without the clause,
  sub-commands 4 and 5 give every member a destination, `aiGroups` skips the whole group, and Move's
  arm never runs again. Cutting it ships a dead arm.
- **FR-23 / AC-21 / DD-17.** `0098` wrote its release exemption as a stance, and its own FR-6 says
  the rule is really about the member's **owner**, "expressed through the one stance this build has
  for it". This story destroys that identity. Cutting it silently inverts `0098`'s release
  population the first time a script stands an enemy group's ground.
- **D-7's rewrite.** The previous D-7 asserted no count is narrowed to a byte. `0098` landed exactly
  that narrowing, with a test pinning it. A divergence that is false is worse than none.

Four stories landed under this one in flight — `0095`, `0097`, `0098` and EXP-0126's four rows. That
is `0024`'s case, and the ruling this lane worked to is that a red gate with a stated reason is
worth more than a green one bought by cutting a correct clause.

## What landed

Two tasks, one commit each, trailer bijection verified in the orchestrator's own seat: exactly one
`SDD-Task: 0096-group-commands/T1` and one `.../T2` between `3566b69` and HEAD.

`git diff --diff-filter=D --name-only 3566b69 HEAD` is **empty** — no file was deleted.

## The gate

Re-run on the committed tree, clean, in this seat rather than taken on an executor's report:

```
go build ./...        clean
go vet ./...          clean
gofmt -l              prints nothing
go test -trimpath -count=1 ./...   every package ok
scripts/check-no-game-assets.sh    exit 0
scripts/check-sdd-audit.sh         exit 0 apart from this file's own absence, now filled
scripts/check-doc-budget.sh        RED, declared above
```

## The measurements this story owed

**Both roots, EN and RU, every figure below reproduced on each.**

### The census, from the build's own gap report

Not a separate count: the numbers are `Script.Unsupported()`'s, over every `.alm` in each install.

```
Par0      nodes     RUNS  skipped
1             1        1        0
2             6        6        0
3            24       24        0
4            29       29        0
5            40       40        0
10            5        0        5
11            6        0        6
14           14        0       14
15           10        0       10
TOTAL       135      100       35
```

**100 of 135 run, 35 skipped**, and every skipped node is in the order-0 family the story fences
out. The build takes the tree from 25 nodes to 100.

### Mission 10: unchanged, as predicted

`missionrun -mission 10 -waypoint u21:56:21:3 -waypoint p0:66:16:3` reports **`lost at tick 272`**
on both roots — the same outcome at the same tick as before the story. `TestTheTenthMissionIsDrivenToAWin`
was not edited and no threshold was tuned (AC-16).

This was predicted at Stage 1 and the prediction is what makes it evidence rather than an excuse.
Measured off the map's own records on both roots: mission 10 authors six group-command nodes — ids
13, 14, 16, 20, 22, 23 — and **no trigger's action list names 13, 14, 16 or 20 at all**. Only 22 and
23 are ever fired, by trigger 0 on tick 6, and both are `Par0 = 14`, Patrol, with groups 18 and 17.
The one node this story would run, id 14 (`Par0 = 4`, Move group 2 to `(66,16)`), is authored and
unreferenced.

**A reason on record was wrong and is corrected.** `analysis.md` first said that node sat on
trigger 4, which `Script.Inert` marks. It does not sit on any trigger; the earlier statement
confused a compiled instant index with a node id. The conclusion held and the stated reason did
not, which is the shape that survives review because everyone checks the conclusion.

### A mission the story does move

`missionrun -mission 151 -trace` — `scn:151`'s first evaluation pass fires four group commands and
this build runs all four, with no `[THIS BUILD DOES NOT RUN IT]` and no group-command gap reported
for the map. `-mission 91` is a second. Both roots agree.

### The gap report names the sub-command

On mission 10 the report reads `groupcmd sub-command 11`, `… 14`, `… 15` — five nodes — and the
`Par0 = 4` node has disappeared from it because it now runs (AC-10, on shipped content).

## Witnessed by reverting, not by reading

- **The order byte in the byte form.** Reverted in this seat: commenting out `b[o+9] = g.order`
  turns `pkg/sim` red; restoring it returns green.
- **The commanded cell.** T1's executor found the suite stayed **green** when it reverted the two
  writes — the field was unwitnessed, because no construction path it built ever wrote a nonzero
  pair. It added `TestADecodedCommandedCellCrossesTheFormWholeAtBothExtremes` and confirmed that
  test alone catches the revert. Reported because a field that was briefly unwitnessed is worth
  more on the record than a clean claim.
- **FR-23** was **not implemented on T1's first pass** and its report did not say so. Caught here by
  reading `decide` rather than by running the gate — the audit could not have caught it, since it
  defers witnessing until every task lands. The executor was sent back; both release sites now read
  `g.owner != SelfSlot` and `TestAReleaseKeysOnTheOwnerNotTheOrder` witnesses both arms. No
  pre-existing test needed an edit, which is exactly what FR-23 predicted: on every world `0098`
  could build, the two rules agree.
- The three arms and the Swarm 2 gate were each reverted by T2's executor; the gate is caught by
  `TestSwarm2SeesOnlyACorpseAndRunsItsOwnBody`, the walk by `TestASwarmCommandGivesNoDestinationUntilTheNextDecision`,
  the arrival filter by `TestMoveScoresOnlyArrivedMembers`.

## The instrument's blind spot

`missionrun`'s `aim` picks the open cell nearest the mover by **terrain alone**, not occupancy, so a
drive can print an identical line before and after a change that works. Mission 10's unchanged
`lost at 272` is **not** rested on that line alone: it is corroborated by the node census above,
which says independently that the mission reaches no implemented sub-command. The stop cell did
move, from `(43,46)` to `(39,41)` — that is `0098`'s release, not this story. Fixing `aim` is a tool
change and was out of scope.

## What this story did not do

- **Patrol, Follow, Attack and Defend** — 35 nodes, D-1. Mission 10 needs Patrol, and that is the
  next story rather than a corner of this one.
- **Swarm 2's fallback far side** is unobservable in this build (D-4). No test claims to witness it;
  AC-17 witnesses the gate's condition and AC-18 the branch that does not walk.
- **No owner-review artifact**, per the standing rule that a story's deliverable is the build.
- The build is `builds/0096-group-commands/` — `missionrun.exe`, `almtool.exe` and a README whose
  every command was run from that directory, on both roots, before it was written down.

## Every criterion, and what witnesses it

All in `pkg/sim` unless named otherwise; every one runs green with no game install present.

| | Witness |
|---|---|
| **AC-1** | `guardradius_test.go` `TestABuiltWorldHoldsOneGroupRecordPerOwnedPairAscending` — its expected records now carry `order: orderStandGround` for slot 1 and `order: orderGuard` for slot 2, and it rebuilds from reversed input; `TestStepLeavesTheGroupRecordUntouched` is the second half, comparing whole records over 20 ticks |
| **AC-2** | `groupcmd_test.go` `TestAGroupCommandMovesExactlyOneOrder` |
| **AC-3** | `TestTheFiveSubCommandsWriteTheOrderTheyName`, `TestAnUnimplementedSubCommandChangesNothing`, `TestANodeNamingNoGroupOrAnAbsentGroupChangesNothing` |
| **AC-5** | `TestSubCommandOneReFreezesTheNoticeBase` |
| **AC-6** | `TestSubCommandsFourAndFiveDistributeLikeThePlayersOwnGroupMove` |
| **AC-7** | `TestASwarmCommandGivesNoDestinationUntilTheNextDecision` |
| **AC-8** | `TestSwarmEngagesWhatGuardWouldHaveClipped` |
| **AC-9** | `TestMoveScoresOnlyArrivedMembers` |
| **AC-10** | `TestTheGapReportNamesTheSubCommand`, and on shipped content in the trace above |
| **AC-11** | `hash_test.go` `TestEveryFieldChangesTheDigest` |
| **AC-12** | `guardradius_test.go` `TestADecodedCommandedCellCrossesTheFormWholeAtBothExtremes`; `binary_test.go` `TestMarshalRoundTripsAndReMarshalsIdentically` |
| **AC-13** | `release_test.go` `TestTheByteFormsVersionMatchesTheConstantAndAFixedWorldsDigestIsPinned` and `binary_test.go` `TestUnmarshalRefusesEveryVersionButTwenty` — **both read the constant**; the one test in the tree that stated a version as a literal was this story's to fix, and it was |
| **AC-14** | `guardradius_test.go` `TestUnmarshalRefusesAnOrderOutsideTheFive` |
| **AC-15** | `binary_test.go` `TestMarshalledBytesArePinned`, `TestThePinnedBytesDecodeBackToThePinnedWorld`, `TestThePinIsThePreStoryPinPlusTheGroupSection` |
| **AC-16** | `cmd/missionrun` `TestTheTenthMissionIsDrivenToAWin`, unedited; and the drive above, both roots |
| **AC-17** | `TestSwarm2SeesOnlyACorpseAndRunsItsOwnBody` |
| **AC-18** | `TestSwarm2LeavesAnAllVetoedGroupUntouchedWhereSwarmWalks` |
| **AC-19** | `engage_test.go` `TestAPairNoRecordNamesDecidesNothing` |
| **AC-20** | `TestAMoveCommandedGroupKeepsDecidingWhileItWalks`; `internal/archtest` `TestDestinationWritersMatchFR2` |
| **AC-21** | `release_test.go` `TestAReleaseKeysOnTheOwnerNotTheOrder`, both arms |
| **P-1** | AC-1's second half — no tick moves an order — plus `TestNoCommandedUnitLeavesTheWorldUnchanged` |
| **P-2** | AC-19 for the missing record; AC-3 for every order a record can hold |
| **P-3** | `binary_test.go` `TestMarshalRoundTripsAndReMarshalsIdentically` |
| **P-4** | `TestAnUnimplementedSubCommandChangesNothing` compares the whole world, not the group |
| **P-5** | AC-14 with AC-12: the decoder refuses exactly the orders the constructor cannot produce, and accepts every cell it can |
| **SC-1** | Every AC above has a test, and each of the four reverts below took its own named test red |
| **SC-2** | The gate, above: green with no install present |
| **SC-3** | The census, above, from `Script.Unsupported()` and not a separate count |
| **SC-4** | Mission 10, above, both roots, before and after: `lost at 272` unchanged |
| **SC-5** | The revert checks, above: the order byte reverted in this seat, the commanded cell by T1's executor — which found it **unwitnessed** and added the test that catches it |
| **SC-6** | `scn:151` and `scn:91`, above, both roots |
| **SC-7** | Stated rather than claimed: AC-17 and AC-18 witness the gate's condition and its no-walk branch. **Neither witnesses the fallback**, which D-4 says is unobservable here, and no test pretends to |

## An open this story names rather than handles

**A group command whose target is the party.** Three of mission 10's six op-6 nodes carry a
`Target_Unit` of `10001`, the hero ordinal, which the binder reports unresolved because no party
stands on the map. Measured over all 38 maps on both roots: of the 135 nodes, **21 carry a unit or
player reference and every one of them is `Par0 = 10`, `11` or `15`** — the order-0 family this
story does not implement. **Not one of the 100 nodes this story runs carries a unit or player
reference at all.** So the question does not arise for any implemented sub-command, and it is
recorded as an open for whichever story implements that family rather than silently handled here.
