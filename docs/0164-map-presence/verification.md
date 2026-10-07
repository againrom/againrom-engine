# 0164 — verification

Branch `0164-map-presence`, base `ac0b732`, research pin `a5ea82f`. No `tasks.md`: the slice was
implemented in one lane context, so every FR, AC, P and DD is accounted for below instead.

## The result someone can point at

The campaign-wide count of script arms this build cannot run **fell from 506 to 425**, and it is
the same number on both preserved roots. The fall is 81, which is every authored node of the five
opcodes.

Both figures come from `scripts/campaign-sweep.sh`, run today against the previous build and this
one. The instrument is the same file in both runs.

```
builds/0163-campaign-drive/againrom.exe   28  2000  0 decided   506  10072
builds/0164-map-presence/againrom.exe     28  2000  0 decided   425  10037   (en)
builds/0164-map-presence/againrom.exe     28  2000  0 decided   425  10037   (ru)
```

The second column, `reached`, is how many of those unrunnable arms the unattended drive actually
arrived at — `playCensus` (`pkg/game/scenario.go`) counts an unsupported instant in a trigger that
FIRED, not one merely compiled. It fell by **35**, on nine of the ten maps whose `unsupported`
count fell:

```
m60   unsupported  36 ->  20    reached  756 -> 751
m70   unsupported  16 ->  10    reached    3 ->   1
m80   unsupported  16 ->  12    reached    3 ->   1
m90   unsupported  31 ->  23    reached  389 -> 379
m91   unsupported  16 ->  14    reached  128 -> 128
m120  unsupported  22 ->   7    reached    2 ->   1
m131  unsupported  46 ->  36    reached  883 -> 878
m140  unsupported  22 ->  20    reached  501 -> 500
m141  unsupported  10 ->   8    reached    2 ->   1
m150  unsupported  36 ->  20    reached  133 -> 125
```

Each `unsupported` fall is exactly that map's own count of the five opcodes: 8+5+3 on 60, 3+3 on
70, 2+2 on 80, 2+2+2+2 on 90, and so on to 1+1+7+7 on 150. **The five arms are executed by an
unattended drive of the shipped campaign on nine maps** — the triggers naming them fire without
anybody playing, which is what the `reached` column falling says and what a compile-time census
cannot say.

`pipeline/check-milestone.sh`'s own census, measured directly on all 28 campaign maps of both roots
with `missionrun -mission N -trace -ticks 1`:

| opcode | authored nodes, before | after |
|---|---|---|
| 16 | 29 | 0 |
| 17 | 27 | 0 |
| 18 | 3 | 0 |
| 32 | 11 | 0 |
| 33 | 11 | 0 |
| **census total, en** | **447** | **366** |
| **census total, ru** | **447** | **366** |

The 81 authored nodes are the census's own population. `TRIG-INSTCENSUS-046` reports that 56 of
them are reachable from a trigger; the census counts every compiled node, reachable or not, so the
census falls by 81 and the reachable subset is 56 of that.

`pipeline/milestone-baseline.txt` was NOT touched. Regenerating it is the seat's step.

### The two numbers the lane brief asks for

Missions 10 and 20, `missionrun -mission N -trace -ticks 1 | grep -c UNSUPPORTED`, `en` root:

| mission | master before | this branch |
|---|---|---|
| 10 | 17 | 17 |
| 20 | 11 | 11 |

**Unchanged, and that is the correct result.** Neither map authors a node of any of the five
opcodes: mission 10's gap is 13 instant-2 nodes, 1 instant-20 and 3 group sub-commands, and mission
20's is 11 instant-2 nodes. The census falls on the twelve maps that do author them — 60, 70, 80,
90, 91, 120, 131, 140, 141 and 150, plus 121 and 130 through the sweep's wider count.

`pipeline/milestone-baseline.txt`'s recorded values for these two missions are 17 and 11, which is
what both columns above reproduce.

## What the five opcodes have to work on, off a lawful install

`cmd/presenceprobe` loads a shipped mission through the production loader and reports, per opcode,
how many nodes the mission's own compiled script holds and how many of them have every unit and
group reference resolving against the world the loader built. Run over the twelve campaign maps
that author these opcodes, on both roots:

```
mission  60  op 16   8/8   op 17   5/5   op 18   3/3
mission  70  op 16   3/3   op 17   3/3
mission  80  op 16   2/2   op 17   2/2
mission  90  op 16   2/2   op 17   2/2   op 32   2/2   op 33   2/2
mission  91  op 16   1/1   op 17   1/1
mission 120  op 16   6/6   op 17   7/7   op 32   1/1   op 33   1/1
mission 131  op 16   5/5   op 17   5/5
mission 140  op 32   1/1   op 33   1/1
mission 141  op 16   1/1   op 17   1/1
mission 150  op 16   1/1   op 17   1/1   op 32   7/7   op 33   7/7
```

81 of 81 nodes, identical between EN and RU. The per-opcode totals are 29, 27, 3, 11 and 11, which
reproduce `TRIG-INSTCENSUS-046`'s authored figures exactly through a fourth instrument — the
production loader — that shares no code with either parser the claim was measured with.

## Nothing was watched on a screen

**No windowed game was launched for this story, and nothing about map presence was seen on a
screen.** A unit disappearing from the map and coming back is a visible event and nobody watched
one.

What stands in its place is named exactly, and it is not a substitute for having looked:

- the sweep's `reached` column, which is a count of arms in triggers that FIRED during an
  unattended drive, and which fell by 35 across nine shipped campaign maps;
- the probe above, which shows all 81 nodes' references resolve against real maps on both roots;
- `TestAnOffMapUnitIsNotDrawn` (`pkg/game/presence_test.go`), which drives the production draw
  derivation — the same `entityDraws` the window tier calls — and was checked by reverting the
  filter.

The last is the dispatch a screen would exercise. It is not a witness that the picture was correct,
and it is not written up as one.

## AC accounting

Every acceptance criterion, and the test that witnesses it. All in `pkg/sim/presence_test.go`
unless named otherwise.

| AC | Witness |
|---|---|
| AC-1 | `TestTheRemovalSetsTheBitAndChangesNothingElse` — the record is compared whole, with the bit put back |
| AC-2 | `TestTheRemovalIsIdempotent` |
| AC-3 | `TestAnOffMapUnitOccupiesNoCell` — a real route, not a hand-seeded plane |
| AC-4 | `TestAnOffMapHostileIsNotAcquired` |
| AC-5 | `TestAVictimTakenOffTheMapIsDropped` |
| AC-6 | `TestAnOffMapUnitIsNotAdvanced` |
| AC-7 | `TestAGroupsCountIsWholeWithItsMembersOffTheMap` |
| AC-8 | `TestTheReturnPutsTheUnitBackOnItsRetainedCell` |
| AC-9 | `TestAReturnWithNoFreeCellChangesNothing` |
| AC-10 | `TestAReturnFallsBackToTheNeighbourhood` |
| AC-11 | `TestTheSwapRemovesTheFirstAndPlacesTheSecond` |
| AC-12 | `TestTheSwapRefusesOneUnitNamedTwice`, `TestTheRemovalRefusesEveryReferenceItCannotResolve` |
| AC-13 | `TestTheGroupArmsRemoveAndReturnEveryMember`, `TestAGroupsCountIsWholeWithItsMembersOffTheMap` |
| AC-14 | `TestTheGroupArmsRemoveAndReturnEveryMember` |
| AC-15 | `TestTheOffMapBitSurvivesTheByteForm` |
| AC-16 | `TestUnmarshalRefusesEveryVersionButTheCurrentOne` and `TestThePinIsThePreStoryPinPlusTheOffMap` (`pkg/sim/binary_test.go`) |
| AC-17 | `TestTheFivePresenceOpcodesAreNoLongerUnsupported` |
| AC-18 | the revert table below |

## FR accounting

| FR | Where it landed | Witness |
|---|---|---|
| FR-1 | `Entity.OffMap`, `pkg/sim/world.go` | `TestTheCanonicalWorldsFieldSetsArePinned` (`pkg/sim/nostate_test.go`) names the field |
| FR-2 | `takeOffMap`, `pkg/sim/presence.go`; opcode-16 arm | AC-1, AC-2 |
| FR-3.1 | `counted`, `pkg/sim/route.go` | AC-3, reverted |
| FR-3.2 | `candidates`, `pkg/sim/engage.go` | AC-4, reverted |
| FR-3.3 | `aiGroups` **and** `groupLivingMembers`, `pkg/sim/engage.go` | `TestAnOffMapMemberIsNotDecidedFor`, reverted |
| FR-3.4 | the move loop, `pkg/sim/step.go` | AC-6, reverted |
| FR-3.5 | the attacker loop (`pkg/sim/step.go`) and `advanceAttack` (`pkg/sim/combat.go`) | AC-5, reverted |
| FR-3.6 | `entityDraws`, `pkg/game/world.go` | `TestAnOffMapUnitIsNotDrawn` (`pkg/game/presence_test.go`), reverted |
| FR-4 | no code: the count check and the group hand-over already read `Group` alone | AC-7 |
| FR-5 | `returnToMap`, `placeNear`, `tryAttempts`, `trySquare`, `placeAt`, `placeFree` | AC-8, AC-9, AC-10 |
| FR-6 | opcode-18 arm, `pkg/sim/script.go` | AC-11, AC-12 |
| FR-7 | `groupMembers` and the opcode-32/33 arms | AC-13, AC-14, `TestAGroupArmNamingAnUnusedGroupChangesNothing` |
| FR-8 | `formatVersion` 48, `entityLen` 223, the +222 byte, `pkg/sim/binary.go` | AC-15, AC-16, `TestTheOffMapByteIsRefusedOutsideZeroAndOne` |
| FR-9 | `scriptInstantSupported`, `pkg/sim/script.go` | AC-17, and the census fall above |

## P accounting

| P | Witness |
|---|---|
| P-1 | Every arm resolves by `indexOfEntity` or scans the whole slice. `TestTheGroupArmsRemoveAndReturnEveryMember` places its two members at different cells and checks each returns to its own, so nothing is carried between members in slice order. |
| P-2 | `TestAReturnWithNoFreeCellChangesNothing` asserts the generator state moved on a FAILED search — the draws are consumed whatever the outcome, so the draw count is a function of world state and never of a drawn value. `internal/archtest`'s source scan continues to hold `pkg/sim` to no `time` and no `math/rand`. |
| P-3 | `TestTheRemovalRefusesEveryReferenceItCannotResolve` runs seven refusals and compares the entity records before and after. AC-9 and AC-12 do the same for the two arms with a partial outcome available. |

## DD accounting

| DD | Where it shows |
|---|---|
| DD-1 | `Entity.OffMap` is a field on the record, not a set on the world; the digest is the byte form, so AC-15's digest separation follows by construction |
| DD-2 | the one line at the top of `counted`; the revert below is what shows it reaches all three of that predicate's readers |
| DD-3 | `placeFree` is a linear scan reading `counted` and `terrainOpen`; no `routeScratch` is allocated on a script pass |
| DD-4 | `tryAttempts` draws twice per attempt at every radius, radius 0 included; P-2's assertion is the witness |
| DD-5 | `placeAt` writes the cell and the bit together, so AC-9 can be checked against the whole record |
| DD-6 | `returnToMap` is `tryAttempts(r=0)` then `placeNear`; the opcode-18 arm is `placeNear` alone |
| DD-7 | `groupMembers` reads `Group`; AC-13 puts its members in the map's own group word |
| DD-8 | `takeOffMap` writes one field; `advanceAttack` drops the victim — AC-5 removes the victim and steps, and the attacker loses it on the attacker's own tick |
| DD-9 | `TestAVersion44FormIsRefused` is now `TestASupersededFormVersionIsRefused`; the number left the name |
| DD-10 | `aiGroups` carries the gate as well as `groupLivingMembers`. **This was found by reverting**, not by reading: with the gate only in `groupLivingMembers`, `TestAnOffMapMemberIsNotDecidedFor` still failed |

## SC accounting

| SC | How it is discharged |
|---|---|
| SC-1 | Stated in `spec.md` and in `Entity.OffMap`'s own doc: the bit is canonical HERE, and this build claims nothing about the original's save format. `provenance.md` names it as EXP-0169's one open thread. |
| SC-2 | No packet layer exists in `pkg/sim`; the arms make the state change the packets announce. Stated in `presence.go`. |
| SC-3 | No log line is written. Stated in `spec.md` and in `returnToMap`'s doc. |
| SC-4 | `placeAt` commits nothing until an attempt succeeds. AC-9 is the witness that a failed return is a true no-op. |
| SC-5 | `tryAttempts` implements the claim's quoted arithmetic. AC-10 asserts the returned cell lies in `[x-1, x+2] x [y-1, y+2]`, which is the arithmetic's window and not the summary's 3x3. |
| SC-6 | `trySquare` scans x outer, y inner. Stated in its doc. |
| SC-7 | `bindParams` fills the second unit slot from the second `Target_Unit`. The probe above shows all 3 shipped instant-18 nodes resolve both references, which is the corpus half of the claim. |
| SC-8 | The section "Nothing was watched on a screen" above. |

## The revert table

Every presence gate was checked by DELETING it and re-running the test that covers it. Each was
restored immediately.

| Gate deleted | Test | Result with the gate gone |
|---|---|---|
| `counted`'s `OffMap` test (`route.go`) | `TestAnOffMapUnitOccupiesNoCell` | FAIL: "the walker stopped at (3,3)" |
| `candidates`' `OffMap` test (`engage.go`) | `TestAnOffMapHostileIsNotAcquired` | FAIL: "the hunter acquired 2" |
| `aiGroups`' `OffMap` test (`engage.go`) | `TestAnOffMapMemberIsNotDecidedFor` | FAIL: "a group decided for an off-map member" |
| the move loop's `OffMap` test (`step.go`) | `TestAnOffMapUnitIsNotAdvanced` | FAIL: "an off-map unit walked from (3,3) to (4,4)" |
| `advanceAttack`'s `OffMap` test (`combat.go`) | `TestAVictimTakenOffTheMapIsDropped` | FAIL: "the attacker still holds 2" |
| `entityDraws`' compaction (`pkg/game/world.go`) | `TestAnOffMapUnitIsNotDrawn` | FAIL: "the derivation draws 2 entities, want 1" |

`TestAnOnMapUnitStillOccupiesItsCell` is AC-18's control and stayed green through the first revert,
so the pair separates "the gate works" from "the fixture never reaches the gate".

**Two tests were rewritten because a revert showed they passed for the wrong reason.**
`TestAnOffMapHostileIsNotAcquired` as first written stayed GREEN with its gate deleted: the hunter
in its fixture never acquired the prey at any distance, so "acquires nothing" was true before the
removal as well. Both it and `TestAnOffMapMemberIsNotDecidedFor` now carry a positive control in
the same fixture and assert acquisition happens without the removal before asserting it does not
with it. This is the whole value of the revert step and it is recorded rather than quietly fixed.

## Gates

Run on a clean tree at the head of `0164-map-presence`.

```
go build ./...                       clean
go vet ./...                         clean
gofmt -l $(git ls-files '*.go')      prints nothing
go test -trimpath -count=1 ./...     all packages ok
```

`implementation/scripts/check-*.sh`, by glob:

```
check-doc-budget.sh          OK
check-hotfix-ledger.sh       OK
check-no-game-assets.sh      OK
check-sdd-audit-selftest.sh  OK
check-sdd-audit.sh           OK
```

`research/scripts/check-*.sh`, by glob, plus the research Go chain:

```
check-claim-ids.sh           OK      go build ./...   clean
check-doc-size.sh            OK      go vet ./...     clean
check-exp-budget.sh          OK      go test ./...    all ok
check-gofmt.sh               OK
check-md-tables.sh           OK
check-no-local-paths.sh      OK
check-retraction-status.sh   OK
```

`check-sdd-audit.sh`'s note/warning count is not comparable from a worktree, which has no
`builds/`. The FAIL set is empty.

## What the byte-form bump cost

`formatVersion` 46 to **48**, and the entity record 222 to 223 bytes. 47 was allocated to parallel
work at the same story boundary and is not reused here.

Re-pinned, each because the record widened by one byte and for no other reason: `pinDigest`,
`rtfDigest`, `rlxTick1Digest`, `hybTick1Digest`, two goldens in `release_test.go`, one in
`commanded_test.go`, and `gfDigest` in `pkg/mapload`. The offset table in
`TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth` gained two rows and every offset at or
past the second record moved by one.

**The peel chain is what says nothing else moved.** `strippedOfTheOffMap` removes this story's one
byte per record and puts version 46 back; `TestThePinIsThePreStoryPinPlusTheOffMap` requires the
result to hash to `0xae4c7eafa4cd51f3` and `0x85c5e4f52f660a84` — the two digests this tree carried
before the change, pasted from the pre-story tree rather than computed from the new one. Both hold.
`pkg/mapload` has the same peel and the same check. That is the assertion a re-pinned digest cannot
make about itself.

## What this story did not do

- The other unrunnable instant opcodes (2, 7, 20, 21, 24, 25, 29, 30, 34) and the three
  unimplemented group sub-commands. 425 arms remain.
- No windowed game was launched and no screen was watched.
- `pipeline/milestone-baseline.txt` was not regenerated; that is the seat's step, and the numbers
  it should carry are in the census table above.
- Whether an off-map unit regenerates health or mana is untouched and therefore unchanged: it does.
  The claims say nothing about it and this story invented no rule for it.
