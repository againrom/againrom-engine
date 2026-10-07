# Verification — 0095, the frozen guard radius

Gate, on a clean tree at the story head, from the lane worktree:

```
go build ./...                       clean
go vet ./...                         clean
gofmt -l $(git ls-files '*.go')      prints nothing
go test -trimpath -count=1 ./...     every package ok, no install present
scripts/check-no-game-assets.sh      clean (tree scan)
scripts/check-doc-budget.sh          exit 0
scripts/check-sdd-audit.sh           FAIL set empty
```

`check-sdd-audit`'s note and warning **count** is meaningless from a lane — the guard is on a
top-level `builds/`, which this story's own build stage creates, so the count jumps by every other
story's gap at that moment and says nothing about this one. Only the FAIL set is comparable and it
is empty.

Two commits carry a `SDD-Task:` trailer, one per entry in `tasks.md`, and nothing else does. The
three untrailered commits are the workflow artifacts, the spec revision and a comment correction.

## The criteria

**AC-1** `TestABuiltWorldHoldsOneGroupRecordPerOwnedPairAscending` — two groups give two records in
ascending `(owner, group)`; an entity in slot 0 contributes none; a group all of whose members are
dead still holds one; the same entities handed in a different slice order give the same records.

**AC-2** `TestTheBaseFollowsTheGreatestDistancePlusRangeNotEitherAlone`, over three members chosen
so that the one furthest from the centroid, the one with the widest range and the one with the
largest sum are three different members. The centroid and the distances are computed by hand in the
test, not from the function under test.

**AC-3** three tests, one per edge: `TestABaseUnderTheFloorIsRaisedToIt`,
`TestAGroupWithNoLivingMemberTakesTheFloorBare`, and `TestAGeometryPastAByteIsNarrowedNotClamped`
(a geometry of 505 reaching the record as 249).

**AC-4 — the story.** `TestAGuardingGroupClipsAtItsFrozenBaseNotLiveGeometry` builds a group of
three, walks two members sixty cells apart with ordinary move commands and fells the third, then
offers a candidate that the **live** geometry at that tick would admit — base 85, radius 89 — and
the **built** one would not — base 25, radius 29. The candidate is never acquired. It is a test
about time: the expectation is the radius the world was built with, and the live value it is set
against is the one the previous build would have used.

**AC-5** is witnessed twice, and the second is the stronger. In the package, the split of the
computation into a base function and a margin reader had to leave the arithmetic bit-for-bit
unchanged, and the byte-form and digest pins are what hold that. Outside it, the whole shipped
corpus was stepped before and after and does not move: see SC-4 below, where 28 campaign missions
run for 2000 ticks produce identical displacement, identical deaths and an identical acquisition
log on both roots.

**AC-6** `TestTwoWorldsDifferingOnlyInOneBaseHashDifferently`.

**AC-7** `TestADecodedBaseCrossesTheFormWholeAtBothEdges` (bases 0 and 255) and
`TestAWorldHoldingNoOwnedEntityHoldsNoGroupAndRoundTrips`.

**AC-8** `TestUnmarshalRefusesEveryVersionButNineteen`. No test in the tree states the version as a
literal: the two that compare against it compare against the constant, and the only places the byte
`0x13` appears are the two hand transcriptions, where a literal is the whole point. Checked by
grep over every `_test.go`.

**AC-9** `TestMarshalledBytesArePinned` and `TestHashIsPinned` over a transcription extended by
hand, and `TestThePinIsThePreStoryPinPlusTheGroupSection`, which strips the section back off, sets
the previous version byte and reaches the previous version's pinned form.

**That the pins witness the section, verified from this seat by reverting rather than by reading.**
The encoder's base byte was perturbed by one and `go test ./pkg/sim/... ./pkg/mapload/...` was
re-run: twelve tests failed, including both pins and both digests. Restored, green.

**AC-10** `TestUnmarshalRefusesAMalformedGroupSection` for the two structural refusals, and
`TestAMismatchedGroupSectionKeySetIsNotRefused` for the shape FR-8 deliberately admits.

**AC-11** The tenth mission is unmoved. `TestTheTenthMissionIsDrivenToAWin` reports **`lost at tick
272`** on **both** roots — the same tick as before this story. `missionrun -trace` names the arm:

```
tick 262  check 12 vip(u21) COUNTED A LOSS: its unit is not alive (-1 hp)
tick 271  REPORT won=0 lost=1 -> lost
outcome lost at tick 272
```

The test is a standing red one and it was neither edited nor tuned; the diff carries no change to
`cmd/missionrun`, to `pkg/game` or to any predicate.

**AC-12** `TestAPairNoRecordNamesClipsAtTheMarginAlone` drives the script's own hand-over arm to
move an entity to a roster slot the freeze never saw, then shows the record set unchanged, the new
pair's base zero, a candidate at distance 5 dropped and one at distance 4 taken.

## The properties

**P-1** `TestStepLeavesTheGroupRecordUntouched` — twenty ticks leave the record set and every base
bit-for-bit as built. The field-set pin `TestTheCanonicalWorldsFieldSetsArePinned` carries the
other half: `World` and the record type are written out field by field, so a second place to keep
this state fails there rather than being noticed.

**P-2** No path can ask for a radius it does not hold: the reader answers zero for a pair no record
names (AC-12), and every pair a decision can carry is either frozen or handed over.

**P-3** `TestMarshalRoundTripsAndReMarshalsIdentically`.

**P-4** `TestADecodedBaseCrossesTheFormWholeAtBothEdges` at both ends of the byte, and no
normalisation exists to test in the other direction.

## The success criteria

**SC-1** Every acceptance criterion above names its test. That the freeze tests fail if the radius
is recomputed was checked by reverting, from this seat: `decide`'s reader was replaced with a live
recompute and `TestAGuardingGroupClipsAtItsFrozenBaseNotLiveGeometry` and
`TestAPairNoRecordNamesClipsAtTheMarginAlone` both failed. Restored, green.

**SC-2** `go test -trimpath -count=1 ./...` is green with no install present.

**SC-3 — the census, and what it says.** Over the **28 campaign missions**, identical on the EN and
the RU roots:

| | groups |
|---|---|
| guard groups on the shipped campaign maps | 679 |
| holding a hostile inside their frozen circle, geometry reading | 235 |
| holding a hostile inside their frozen circle, D-3's floor-only reading | 141 |

The distance from each group's load centroid to its nearest hostile, same corpus, both roots:

| distance | groups |
|---|---|
| no hostile on the map at all | 24 |
| within 2 cells | 7 |
| 3 to 5 cells | 0 |
| 6 to 11 cells | 98 |
| 12 to 20 cells | 235 |
| past 20 cells | 315 |

**The plan asked which it is, and it is neither reading of the radius.** 105 groups — the 7 and the
98 — have a hostile inside **the smallest circle the law can draw**, which is the registry floor of
8 plus the arm's margin of 4. No install of the base, geometry or floor-only, separates those: the
floor is the floor. On the map where the deaths concentrate, the map's own type-5 roster names slot
6 `Friends` and slot 7 `Enemies`, authors them hostile in both directions, and stands four units of
each within two cells of the other at load — with `Villagers` and `Friends` likewise, six units
apart by two. That is an authored opening battle, not a defect in a radius.

So the freeze is correct and it is not sufficient, and what remains is not a radius question. It is
the visibility gate and the absent walk-home break-off, both named in `analysis.md` with the
measurement that points at them.

**SC-4 — the drift, before and after, both roots.** Every campaign mission started and stepped
**2000 ticks with no commands at all**:

| | before | after |
|---|---|---|
| entities | 2357 | 2357 |
| moved at all | 79 | 79 |
| total displacement, cells | 501 | 501 |
| died | 35 | 35 |

Identical, and identical on both roots. The player-driven case is identical too: 0094's run D
reproduced — mission 10, the party walked to (25,55) and then to (40,50) — drags the clubman from
(32,62) to (27,57) and acquires at tick 135, before and after, on both roots. And the acquisition
log of mission 90, the worst map, is line-for-line the same.

**This story changes no measurable behaviour on the shipped corpus, and that is a result rather
than a disappointment.** What it removes is measurable in its own right: over the same 2000 ticks,
**17 of the 679** guard groups ever had a live geometry radius differing from the one they were
built with — widest excess 4 cells, widest shortfall 7. Those 17 are the whole of the divergence
the freeze deletes, and on this corpus none of them changed an acquisition. The behaviour is
therefore witnessed in a constructed world (AC-4) and its absence on shipped content is measured
rather than assumed.

**SC-5** Recorded under AC-11: `lost at tick 272` on both roots, unchanged, with the arm named.

**SC-6 — the revert check.** Recorded under SC-1 and under AC-9: three separate reverts, each run
from this seat and each restored, and each broke a named test.

## Manual

`builds/0095-guard-radius/` holds `againrom.exe` and `missionrun.exe`, both `go build -trimpath`
from the story head, and a `README.md` whose every command was run on both roots before it was
written down. `againrom -check` reports the install's 38 map rows and the generated hero; the two
`missionrun` invocations reproduce the tenth mission's loss and the arm that decides it. Neither
binary writes a file.

## What is not verified here

- The other two install moments (D-1). There is nothing in this tree to trigger them, so their
  absence is checked by reading the command set, not by a test.
- Whether the load-time install sees a computed geometry or a zero (D-3). Both readings are
  measured in SC-3 and neither is settled; it is a question for research.
- The `+/-1` roll (D-2), which this build does not have.
