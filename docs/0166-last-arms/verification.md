# 0166-last-arms — verification

No `tasks.md` was written. The slice fitted one context, so this file accounts for every `FR`, `AC`,
`P`, `D` and `SC` directly, and every commit on the branch is against `spec.md` and `plan.md`.

## The result someone can point at

**The script-gap census falls by 34 nodes per preserved root, from 288 to 254 — 576 to 508 over the
pair. The campaign's mechanical gap is now zero.** Every remaining node is instant opcode 2, a
broadcast that touches no simulation state.

Measured with one command run against two binaries — `builds/current/missionrun.exe`, the build
master carried when this story branched, and `builds/0166-last-arms/missionrun.exe` — over all 28
campaign maps of both preserved roots:

```powershell
$roots = '<seat>\gameversions\en','<seat>\gameversions\ru'
$missions = 10,20,30,31,40,41,50,51,60,61,70,71,80,81,90,91,100,101,110,111,120,121,130,131,140,141,150,151
$n = 0
foreach ($r in $roots) { $env:AGAINROM_ASSETS = $r
  foreach ($m in $missions) { $n += (./missionrun.exe -mission $m -trace -ticks 1 | Select-String 'UNSUPPORTED instant').Count } }
$n
```

| operation | before, per root | after | fall |
|---|---|---|---|
| instant 7, set formation | 1 | 0 | 1 |
| instant 20, drop all | 3 | 0 | 3 |
| instant 25, cell-record tail | 2 | 0 | 2 |
| instant 30, attached-effect duration | 2 | 0 | 2 |
| instant 34, property setter | 5 | 0 | 5 |
| group sub-command 10, attack | 5 | 0 | 5 |
| group sub-command 11, defend | 6 | 0 | 6 |
| group sub-command 15, follow | 10 | 0 | 10 |
| instant 2, broadcast | 254 | 254 | 0 |
| **total** | **288** | **254** | **34** |

The per-operation figures are `pipeline/milestone-baseline.txt`'s own, aggregated. Both roots give
identical counts before and after. `pipeline/milestone-baseline.txt` is not updated here: it lives
outside this repository and is regenerated at the landing against `builds/current/` rebuilt from
master.

### Mission 10 and mission 20, the two the lane brief asks for

```
go build -o /tmp/mr ./cmd/missionrun
for m in 10 20; do AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission $m -trace -ticks 1 | grep -c UNSUPPORTED; done
```

| mission | master before | this branch |
|---|---|---|
| 10 | 17 | **13** |
| 20 | 11 | 11 |

Mission 10 falls by four: one Defend, two Follows and one Drop all. Mission 20 authors none of the
eight, so it is unchanged and its eleven are all opcode 2. Both roots give the same pair of numbers.

### What was watched, and what was not

**Nothing was watched on a screen.** No windowed game was launched. What was observed is
`missionrun`'s own trace on the owner's own asset roots, which is the instrument
`pipeline/check-milestone.sh` uses.

The clearest single observation is mission 110's one `Set formation` node, on the same map at the
same tick under both builds. `builds/current`:

```
script  UNSUPPORTED instant 20: instant op 7
tick 6  trigger 2 FIRED (map latch 2, once)
          slot 2  instant 20 instant op 7  [THIS BUILD DOES NOT RUN IT]
```

This branch:

```
tick 6  trigger 2 FIRED (map latch 2, once)
          slot 2  instant 20 instant op 7
```

Mission 151's four instant-34 nodes lose the same marker. Both are in
`builds/0166-last-arms/README.md`, whose commands were run the owner's way — from that directory,
through `$env:AGAINROM_ASSETS` — before they were written down.

### The mission 10 drive is unchanged

```
mission 10  scenario/10.alm  80x80  36 entities
waypoint 1  u21 -> (56,21) r3 : was STOPPED BY THE WORLD DECIDING, short of (41,43), Chebyshev 22, after 224 ticks
outcome lost at tick 224
census: 4 of 36 unit(s) moved, 1 fell, over 224 tick(s)
```

Identical to `pipeline/milestone-baseline.txt`'s recorded drive on both roots. `lost` is by design.

`scripts/campaign-sweep.sh --census` over both roots: 28 of 28 maps load, exit 0, `unsupported`
falls from 347 to 313 — the same 34 — and the `alive` and `fallen` columns are unchanged on every
one of the 28. The `hash` column moves on every map, which is the byte-form version bump and not a
behaviour change; the unchanged `alive`, `fallen` and drive lines are what says so.

## Every FR

| FR | Where | Witness |
|---|---|---|
| FR-1 | `runInstant` case `ScriptInstantFormation`, `setFormationMode` (`formation.go`) | `TestInstantSevenWritesTheNamedPlayersFormationMode`, `TestInstantSevenRefusesWhatItCannotName` |
| FR-2 | `formationDefault`, `resetFormations`, called by `newWorld` | `TestInstantSevenWritesTheNamedPlayersFormationMode` reads all 50 slots before the node |
| FR-3 | `groupInFormation` (`formation.go`), read by `issueGroupDestination` | `TestTheFormationModeGatesTheGroupMoveDistribution`, four modes |
| FR-4 | `dropAll` (`carry.go`) | `TestInstantTwentyDropsTheWholeContainerAtTheUnitsFeet` |
| FR-5 | `dropAll` through `pourSack` | `TestInstantTwentyMergesIntoASackAlreadyThere` |
| FR-6 | `dropAll`'s two guards | `TestInstantTwentyRefusesWhatItCannotDrop` (three cases over the whole byte form); the gold is asserted 0 in AC-4's test |
| FR-7 | `tailKey`, `setCellTail` (`celltail.go`) | `TestInstantTwentyFiveWritesTheCellTail`, `TestTheTailKeyIsNotTheAreaEffectKey` |
| FR-8 | `setCellTail`'s byte literal | `TestInstantTwentyFiveWritesTheCellTail`, `TestTwoNodesOnOneCellLeaveOneTail` |
| FR-9 | `tailbinary.go`, `World.cellTails` | `TestEveryKindOfStateThisStoryAddsIsCanonical` |
| FR-10 | `setUnitEffectTime` (`script.go`) | `TestInstantThirtyRetimesTheAttachedEffect`, three cases |
| FR-11 | the store has no guard in front of it | `TestInstantThirtyWritesAZeroDuration`, including the decay pass that removes it |
| FR-12 | `Entity.SpellFX` widened to `uint16` | the same test writes and reads 60000 |
| FR-13 | `setUnitProperty` (`script.go`) | `TestInstantThirtyFourWritesTheSelectedProperty`, three selectors and a fourth that stores nothing |
| FR-14 | no clamp in `setUnitProperty` | `TestInstantThirtyFourWritesAHealthAboveTheMaximum` |
| FR-15 | `w.clearFelled(i)` after the health store | `TestInstantThirtyFourFellsAUnitItWritesToZero`, which round-trips the world it produces |
| FR-16 | `stopGroupMembers` | `TestSubCommandTenEngagesEveryOtherMember`, on the NAMED unit's destination |
| FR-17 | `cmdGroupAttack`, `targetVetoed`, `acquireInPlace` | `TestSubCommandTenEngagesEveryOtherMember`, `TestSubCommandTenHoldsAVetoedMemberInPlace` |
| FR-18 | the two guards in front of the stop | `TestTheLastThreeSubCommandsRefuseWhatTheyCannotName`, all three sub-commands × three refusals |
| FR-19 | `cmdGroupEscort` | `TestSubCommandsElevenAndFifteenAreOneShapeAndTwoStates` |
| FR-20 | `escortRangeDefault` and the byte narrowing | `TestAnEscortRangeOfZeroBecomesThree`, at 0 and at 256 |
| FR-21 | the same two guards | the same refusal test covers 11 and 15 |
| FR-22 | `formatVersion` 50, `entityLen` 230, `tailbinary.go` | `TestEveryKindOfStateThisStoryAddsIsCanonical`, `TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth`, `TestMarshalledBytesArePinned`, `TestThePinIsThePreStoryPinPlusTheLastArms` |
| FR-23 | `scriptInstantSupported`, `groupOrderSupported` | `TestTheLastEightOperationsAreNoLongerUnsupported`, which also asserts instant 2 is still reported |

## Every AC

AC-1 to AC-17 are each named in the table above beside the FR they belong to, and every one of them
is a named test in `pkg/sim/lastarms_test.go`. The mapping is one to one:

AC-1 `TestInstantSevenWritesTheNamedPlayersFormationMode`; AC-2 and AC-3
`TestTheFormationModeGatesTheGroupMoveDistribution`; AC-4
`TestInstantTwentyDropsTheWholeContainerAtTheUnitsFeet`; AC-5
`TestInstantTwentyMergesIntoASackAlreadyThere`; AC-6 `TestInstantTwentyFiveWritesTheCellTail`; AC-7
`TestTwoNodesOnOneCellLeaveOneTail`; AC-8 `TestInstantThirtyRetimesTheAttachedEffect`; AC-9
`TestInstantThirtyFourWritesTheSelectedProperty`; AC-10
`TestInstantThirtyFourFellsAUnitItWritesToZero`; AC-11 `TestSubCommandTenEngagesEveryOtherMember`;
AC-12 `TestSubCommandTenHoldsAVetoedMemberInPlace`; AC-13 and AC-14
`TestSubCommandsElevenAndFifteenAreOneShapeAndTwoStates`; AC-15
`TestAnEscortRangeOfZeroBecomesThree`; AC-16 `TestEveryKindOfStateThisStoryAddsIsCanonical`; AC-17
`TestTheLastEightOperationsAreNoLongerUnsupported` and the census table above.

## Every P

- **P-1.** `internal/archtest`'s source scan over `pkg/sim` is unchanged and green: no `os`, `time`
  or `math/rand` import, no float identifier or literal. Nothing this story adds reads a map.
- **P-2.** Every unit reference goes through `indexOfEntity`. The refusal tests reach the
  "a reference this world does not hold" case on instants 20, 30 and 34 and on all three
  sub-commands.
- **P-3.** `TestTheLastEightOperationsAreNoLongerUnsupported` asserts the report and the dispatch
  agree for all eight, and that instant 2 is still reported unsupported.
- **P-4.** Every refusal test compares the **whole byte form** before and after, not a field.
- **P-5.** `setCellTail` inserts at the sorted position and `decodeScriptState` refuses a form whose
  tails are not strictly ascending. `TestTwoNodesOnOneCellLeaveOneTail` witnesses the one-per-cell
  half.

## Every D

D-1 `World.formations` is `[relationSlots]uint8`; D-2 `resetFormations` in `newWorld`; D-3
`issueGroupDestination` reads `w.entities[members[0]].Owner`; D-4 `dropAll` calls `pourSack`,
`expandContainer` and `sackFault` and does not call `deathGold`; D-5 `celltail.go` is its own file;
D-6 `tailKey` beside `cellKey`, witnessed by `TestTheTailKeyIsNotTheAreaEffectKey`; D-7 `SpellFX` is
`uint16`; D-8 the guard is `e.SpellFX == 0 || e.SpellFXSpell != uint8(spell)`, witnessed by
`TestInstantThirtyDoesNotMatchAnEntityCarryingNothing`; D-9 `w.clearFelled(i)`; D-10 `orderAttack`
and no `actorStateEngage` constant; D-11 `targetVetoed` passes `orderNone`; D-12 three states and
three fields, and `actorPass`'s switch gained no case; D-13 `patrolFault`'s three new rules,
`clearEscort` at the constructor and in `clearFelled`; D-14 `stopGroupMembers` has three callers;
D-15 `formatVersion` 50 and `entityLen` 230, witnessed by the offset table; D-16 done — see below.

### D-16, the version test

Two tests spelled the live version number in their bodies and had to be re-spelled by every story
that bumped it. Both are version-free now: `TestAWorldWithADeathRoundTripsAndKeepsTheFormVersion`
(`corpseloot_test.go`) and `TestACommandedWorldRoundTripsAndHashesToTheSameDigest`
(`commanded_test.go`) assert against the constant. The version's own tripwire stays the pinned form
and digest in `binary_test.go` and `hash_test.go`, which an unexamined bump cannot satisfy.

`sightrange_test.go`'s `entityLen` tripwire is re-pinned to 230 rather than removed: it is a
statement about the record's width and not about the version, and its own doc block says which
stories moved it.

## Every SC

- **SC-1**, the per-tick behaviour of the three new actor states, is **not built**. `actorPass`'s
  switch gained no case, which is checkable by deletion exactly as that file's own SC-1 states.
  The state, the escort target and the escort range are written, carried and hashed.
- **SC-2**, one attached effect per entity rather than a list, is stated in `setUnitEffectTime`'s
  own doc. No shipped node can distinguish the two readings: the campaign's two instant-30 nodes
  name one unit and one effect id.
- **SC-3**, the dynamic-plane bit-0 rejection and the recomputation mark, have no counterpart in
  this tree and are not built. `setCellTail`'s doc says so.
- **SC-4**, the ordinary scorer rather than the stand-ground variant, is `targetVetoed`'s
  `orderNone`.
- **SC-5**, no notification. Nothing this story adds sends anything.

**None of the five cuts removes anything the story was asked for.** The eight operations all run and
all write what their claims say they write; what is cut is a per-tick arm for a state, a list this
build has never had, two fields of a record this build does not model, one of two scorer variants,
and a packet layer that does not exist.

## The gate

Run on a clean tree at `cf2c73c`, from the worktree:

```
go build ./...                                                        clean
go vet ./...                                                          clean
gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')  prints nothing
go test -trimpath -count=1 ./...                                      all packages ok
bash scripts/check-doc-budget.sh                                      ok
bash scripts/check-hotfix-ledger.sh                                   ok
bash scripts/check-no-game-assets.sh                                  ok
bash scripts/check-sdd-audit-selftest.sh                              ok
bash scripts/check-sdd-audit.sh                                       no FAIL for 0166
bash scripts/check-sdd-audit.sh (research submodule glob)             every check-*.sh ok
```

`git log --format='%h %(trailers:key=Co-Authored-By)' 982a7d1..HEAD` prints three commits and no
trailer value on any of them.

The `check-sdd-audit.sh` note and warning **count** is not comparable from a worktree, which has no
`builds/`. Only the FAIL set is enforced here and 0166 is not in it.

### What was reverted to learn whether it is witnessed

Nine lines were deleted or inverted one at a time and the naming test re-run. Eight failed, which is
what makes them witnessed:

| reverted | test that went red |
|---|---|
| `targetVetoed` dropped from `cmdGroupAttack`'s condition | `TestSubCommandTenHoldsAVetoedMemberInPlace` |
| the 0-to-3 escort range coercion | `TestAnEscortRangeOfZeroBecomesThree` |
| `groupInFormation` replaced by bare `inFormation` | `TestTheFormationModeGatesTheGroupMoveDistribution` |
| `w.clearFelled(i)` after the health store | `TestInstantThirtyFourFellsAUnitItWritesToZero` |
| the tail's sixth byte changed from `y` to `x` | `TestInstantTwentyFiveWritesTheCellTail` |
| `w.carried[i] = nil` after the pour | `TestInstantTwentyDropsTheWholeContainerAtTheUnitsFeet` |
| the `SpellFX == 0` half of instant 30's guard | `TestInstantThirtyDoesNotMatchAnEntityCarryingNothing` |
| `w.clearOrder(mi)` in `stopGroupMembers` | `TestSubCommandTenEngagesEveryOtherMember` — **after a correction** |

**The ninth was not witnessed and is corrected in the branch.** The stop's `clearOrder` was asserted
on an ENGAGING member, whose destination `orderAttack` clears whether or not the stop ran, so
deleting `clearOrder` left the test green. The assertion is on the NAMED unit now, which takes no
attack order at all, and deleting the line then goes red. It was found by reverting the line rather
than by reading the assertion.

## What is authored rather than decoded

- **The escort range's own default of 3** is decoded (`AI-FOLLOWRANGE-115`), not authored. What is
  authored is nothing in this story's arms: every value, width, order and refusal comes from a
  claim.
- **The refusal of a self-naming pair** is not reached by any arm here, so the give-all's own
  authored refusal is not extended.
- **Instant 34's health-to-zero felling** is this build's own coupling and not the original's: the
  original stores the word with no death path behind it. It is a disclosed divergence, stated in
  `spec.md` FR-15 and in `setUnitProperty`'s doc, and it is authored-unreachable on shipped data —
  the campaign's one selector-6 node writes 1.

## What is not done

- The per-tick behaviour of defend, follow and acquire (SC-1). That is the next story on this
  contract: `AI-DEFEND-111`'s approach and cover engagement, `AI-FOLLOWTAB-113`'s inner table and
  `AI-FOLLOWGAP-114`'s crowding check.
- Instant 2, the broadcast. It touches no simulation state and is the whole of what is left of the
  campaign's script gap.
- `pipeline/milestone-baseline.txt` is not regenerated here — it is outside this repository.
