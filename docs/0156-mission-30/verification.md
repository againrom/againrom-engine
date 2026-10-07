# 0156 — verification

What was run, and what it showed. With no `tasks.md`, every FR and DD is
accounted for here.

Branch `0156-mission-30`, merged with master `4b5526a` (0154, `formatVersion`
44 → 45). Research pin `a60a981`.

## The result

Mission 30's script-gap census, `pipeline/check-milestone.sh`'s counter:

| | before (`pipeline/milestone-baseline.txt`) | after |
|---|---|---|
| en m30 | `1 x instant op 12`, `2 x instant op 13`, `8 x instant op 2` | `8 x instant op 2` |
| ru m30 | the same three rows | `8 x instant op 2` |

11 unrunnable nodes → 8, on both roots. Across all 28 campaign maps the diff
against the baseline removes 13 rows per root and adds none:

```
en/ru m20   1 x instant op 12          en/ru m100  1 x op 12, 6 x op 13
en/ru m30   1 x op 12, 2 x op 13       en/ru m131  3 x op 12, 11 x op 13
en/ru m60   6 x instant op 13          en/ru m141  1 x op 12, 1 x op 13
en/ru m81   8 x instant op 13          en/ru m151  5 x instant op 13
en/ru m90   4 x instant op 13
```

7 op-12 nodes and 43 op-13 nodes per root, 50 per root and 100 over both. No
other census row changed. SC-2.

The two numbers the lane brief asks for, `missionrun -mission N -trace -ticks 1 |
grep -c UNSUPPORTED` over the EN root, this tree against `builds/current`
(master's binary):

| | master | this branch |
|---|---|---|
| mission 10 | 17 | 17 |
| mission 20 | 12 | 11 |

Mission 10 authors no item instant, so 17 is unchanged and that is a real claim.
Mission 20's single op-12 node now runs. SC-4.

## The win

**Reached and asserted, over `30.alm`'s own trigger shape, in
`TestACarriesExpectationReadsTheUnitsOwnContainer`** (`pkg/game/scenario_test.go`,
runs under `go test` with no install): an always-true start trigger running
instant 12 with code `0x0e1e`, and an arrival trigger whose one condition is a
distance test and whose action list is instant 13, instant 13 against an
unresolved second carrier, then WIN. The scenario walks the unit to the cell,
waits for `outcome: won`, and asserts the code is gone from the container. The
outcome is asserted because the scenario's own order is what makes it
deterministic. AC-9's mechanism.

**Not reached on the shipped map, and this is the story's one honest gap.**
`scenarios/0156-mission30-cure.json` runs against a lawful install and passes on
both roots with identical digests (`a23331232fcefa29`): the map loads, the
compiled census is 8, the hero stands at the drop cell (15,69) carrying nothing,
and by tick 20 the map's own instant-12 node has put **code 3614 (`0x0e1e`),
count 1** in his container. That is instant 12 end to end on shipped bytes. The
file stops there because the walk to unit 56 at (65,15) cannot be completed:

- A headless mission start gives a party of **one**. `[Mission30] AddHero=22`
  is implemented (`pkg/game/frontend.go`), but it runs when the **town view**
  activates, and `missionrun` and the mission stage both start a mission
  directly. Real play arrives with two heroes; a headless drive arrives with one.
- That lone 145-HP hero loses the crossing. Driven at the destination he reaches
  (25,25), kills four owner-3 units, and is finished by owner 4 (u38..u42,
  50+50+15+15+15 HP). Six routes were tried — direct, two eastern, two northern,
  one western — and every one ends the same way; the south-east is unroutable
  from the start pocket, so the middle cannot be avoided.
- Driven the other way, unit 56 walks from (65,15) to (60,63) and is killed by
  owner 5 before it reaches the hero. Its only route out of the north-east
  pocket runs down the east side past u54.

So the map's win **condition** is not what was missing — it is a distance test
and it already worked. What was missing were the two operations on its action
list, and those now run. Whether a party can survive the crossing is a combat
and party-composition question, outside this story's contract, and it is
reported rather than claimed either way.

## FR and DD, and what witnesses each

| id | witness |
|---|---|
| FR-1 | `TestATargetItemParameterCompilesToAPackedItemCode`, `TestOnlyTheFirstTargetItemSlotIsTaken` (`pkg/mapload`) — value 6 compiles to `0x0e1e`, the win node's flag stays clear, `Args` stays empty |
| FR-2 | `TestTheAddArmCreatesOneUnitInTheNamedUnitsContainer`, `TestTwoAddsOfOneCodeLeaveOneElementAtCountTwo`, `TestAnAddOntoAHeldCodeKeepsThatElementsPlace` |
| FR-3 | `TestTheTakeArmRemovesExactlyOneUnit` (five cases: 2→1, 1→gone, 5→4, code not held, empty container) |
| FR-4 | `TestTheFourRefusalsLeaveTheWorldExactlyAsFound`, eight sub-cases (four refusals × two arms) |
| FR-5 | `TestNeitherItemArmTouchesAnythingButTheContainer`, `TestNeitherItemArmArmsATrigger` |
| FR-6 | `TestAScriptCarryingItemNodesRoundTripsThroughTheByteForm`; versions 44 and 45 refused |
| FR-7 | `TestACarriesExpectationReadsTheUnitsOwnContainer`, `TestACarriesExpectationFailsWhenTheCodeIsNotThere` |
| AC-1 | `TestTheAddArmCreatesOneUnitInTheNamedUnitsContainer` |
| AC-2 | `TestTwoAddsOfOneCodeLeaveOneElementAtCountTwo`, `TestAnAddOntoAHeldCodeKeepsThatElementsPlace` |
| AC-3 | `TestTheTakeArmRemovesExactlyOneUnit` |
| AC-4 | `TestTheTakeArmLeavesEveryOtherElementWhereItWas`, `TestTheTakeArmWritesOnlyTheNamedUnitsContainer` |
| AC-5 | `TestNeitherItemArmTouchesAnythingButTheContainer` |
| AC-6 | `TestTheFourRefusalsLeaveTheWorldExactlyAsFound` |
| AC-7 | `TestAScriptCarryingItemNodesRoundTripsThroughTheByteForm` |
| P-1 | `TestTheContainerInvariantSurvivesBothArms` |
| P-2 | `TestWhichContainerAnItemArmWritesIsTheIdAndNotThePlace` |
| AC-8 | the census table at the top: no op-12 or op-13 row on m30, either root |
| AC-9 | the win paragraph above — mechanism reached, shipped map partially |
| P-3 | the install scenario passes on EN and RU with the same digest |
| DD-1, DD-2, DD-3 | FR-1's two binder tests: the code is `0x0e18 + V` as a `uint16`, and the flag is separately observable on a node naming no item |
| DD-4 | the version bump; see the paragraph below |
| DD-5 | `TestTwoAddsOfOneCodeLeaveOneElementAtCountTwo` — mutation M4 below |
| DD-6 | `TestTheTakeArmRemovesExactlyOneUnit` — mutations M2 and M3 below |
| DD-7 | `TestNeitherItemArmArmsATrigger` asserts check 17 still reports unsupported |
| DD-8 | `TestAWorldWithADeathRoundTripsAndKeepsTheFormVersion` — the tripwire's name carries no number and its literal was re-pinned with a sentence |
| DD-9 | `scenarios/README.md` and the `carries` clause take a numeric `code` |
| SC-1 | the gate block below: build, vet, gofmt and the suite, all green in both repos |
| SC-2 | the census table at the top |
| SC-3 | `scenarios/0156-mission30-cure.json` passes over the lawful EN root, and over RU with the same digest |
| SC-4 | the mission 10 and 20 table at the top |

## The version bump, and the proof that nothing else moved

`formatVersion` 45 → **46**. The instant record grows three bytes at its tail:
the code at +64, the flag at +66, 64 bytes to 67.

The lane had no allocation and could not reach the orchestrator mid-run
(`AskUserQuestion` is unavailable inside a subagent). It took 46 from the
observed state of every branch and worktree on the machine — master 44,
`0154-spells` 45, nothing above — and 0154 has since landed at 45, so 46 is now
45's plain successor and skips nothing. **This is the one decision in the story
that is the seat's to ratify.** Renumbering costs one constant and the literals
listed below.

Every moved pin was re-derived by measurement, not merged:

> With `formatVersion` reverted to 45 and every other change of this story in
> place, `go test ./...` passes whole. So no byte but the version moved in any
> fixture in either package.

Re-pinned on that basis: `pinDigest`, `rtfDigest`, `rlxTick1Digest`,
`hybTick1Digest`, `gfDigest`, two `release_test.go` digests, one
`commanded_test.go` digest, and the version literals in `binary_test.go`,
`routeform_test.go`, `commanded_test.go`, `corpseloot_test.go` and
`fromalm_test.go`. `strippedOfTheScriptItem` is the new outermost peel and
restores 45; 0154's `strippedOfTheSpellState` then restores 44, so both stories'
newest links survive in both chains.

## Mutation testing

Each mutation applied alone to `pkg/sim/script.go`, `go test ./pkg/sim/` counted:

| mutation | failures |
|---|---|
| M1 drop the zero-code guard | 2 |
| M2 always unlink the element (`Count <= 1` → `true`) | 4 |
| M3 never unlink, only decrement (`→ false`) | 4 |
| M4 add appends without folding | 2 |
| M5 drop the `HasUnit` guard | 3 |
| M6 drop the `HasItem` guard | 3 |

M5 and M6 **survived at first**, because a wholly zero record is refused by the
id lookup and the zero code anyway. The two refusal cases were changed to carry a
value with the flag clear, which is what makes them witness the flags rather than
the lookup. Both kill now.

## Gate

Run from the worktree on the merged tree, clean:

```
go build ./...                              ok
go vet ./...                                ok
gofmt -l $(git ls-files '*.go')             (no output)
go test -trimpath -count=1 ./...            ok, no FAIL
bash scripts/check-no-game-assets.sh        ok
bash scripts/check-doc-budget.sh            ok
bash scripts/check-hotfix-ledger.sh         ok
bash scripts/check-sdd-audit.sh --story 0156-mission-30   ok
```

`check-sdd-audit.sh` was run **scoped to this story**; the cross-story half was
not run here and is the seat's at the landing.

## Found and not built

- **Check opcode 17, `Item in inventory`** (`TRIG-ITEMTEST-040`): 23 authored
  nodes over the campaign, 0 in `30.alm`. Building it needs the same reference on
  the **check** record, which grows that record too and changes trigger inertness
  on maps this story never looked at. `TestNeitherItemArmArmsATrigger` pins that
  it still reports unsupported.
- **Instant opcode 11**, the transfer (`TRIG-XFERITEM-039`): 0 authored nodes on
  either root, so building it moves no census number.
- **Instant opcode 2**, the broadcast: mission 30's remaining 8 unrunnable nodes
  are all this one. `TRIG-MSG-023` reads it as writing no simulation state, so it
  does not stand between the map and its win.
- **Hero ordinal 2 does not resolve.** `resolveUnit` binds only ordinal 1, so
  mission 30's second instant-13 node (unit `10002`) compiles with no unit and
  does nothing. That is correct for a one-hero party and is the same absence
  `AddHero` above names; the map's pair of takes exists to cover both carriers
  and one of them is enough.
- **The headless mission start bypasses `AddHero`**, so a drive is one hero short
  of real play. Not fixed here: it reaches party assembly and the town view.
