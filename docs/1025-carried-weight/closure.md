# 1025 — carried weight: closure

As-built evidence at the landing of 2026-08-22, on branch `1025-carried-weight` over master
`0908a1b`. `spec.md` states the behaviour; this file states what was measured.

## The result someone can point at

The character card draws a `WEIGHT` row with a real per-character value on every surface that draws
a character sheet. Before this story no value anywhere in the build carried an item's weight.

Measured on both preserved roots through the production composer:

- `pipeline/check-scenarios.sh` selects **14** scenarios where it selected 13, and the new one,
  `scenarios/1025-mission10-weight.json`, enters campaign mission 10 with a generated hero and
  states his carried load and his carrying capacity. 14 of 14 pass on `en` and on `ru`.
- The statistics card lays out **17** rows where it laid out 16, and still fits at the production
  font on both roots (`TestReleaseStatisticsCardFitsAtTheProductionFont`,
  `TestReleaseChargenMaximumAllocationFitsTheCard`).
- The mission panel's sheet states **21** lines where it stated 20, and the generator's sheet and
  the lived sheet still agree line for line
  (`TestReleaseGeneratedCharacterLaunchSaveLoadAndCampaignContinuity`).

The milestone census did not move, and the reason is stated under "The census" below.

## The twelve aspects

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | `pkg/data/itemweight.go` reads runtime column 3 and ladder slot 3. `pkg/data/itemweight_test.go` pins the arithmetic as literals worked out from the claim. `cmd/wearcheck -weights` sweeps the shipped tables. |
| Runtime state | PASS | `sim.Entity.Load` and `sim.Entity.Capacity`. `recomputeLoad` is the only writer of the load and every item-moving producer calls it. |
| Simulation | PASS | `overloadedSpeed` is the only consumer, applied through `moverSpeed`. `TestOnlyALoadAtOrAboveCapacityCostsSpeed` (8-row table) and `TestAnOverloadedActorStepsSlower`, which reads `w.StepRate` rather than recomputing it. |
| Player input | N/A | No input path changes. Nothing refuses an action on weight. |
| AI | N/A | The AI issues the same actions. The penalty reaches it through `moverSpeed` without any decision change. |
| UI/HUD | PASS | `PanelFieldWeight` in both layouts, three `RenderCharacterPanel` call sites. `TestTheCompactCardDrawsTheWeightRowBetweenTheResistancesAndTheTotals` is registered in `pkg/ui/screenregistry.go` and was mutation-proved at the use site. |
| Triggers/scripts | PASS | A compiled script's own item literals are declared at mission start (`declareItemWeights` through `mapload.StartMissionScripted`), and the give-item and take-item instants recompute the load. `TestAScriptsOwnItemLiteralIsDeclared`. |
| Inventory/equipment | PASS | The container sum is `weight * count` over the elements; the worn sum counts each slot once. Every carry, transfer, drop, take, equip and unequip path recomputes. |
| Persistence | PASS | Byte form 55 to 56: the entity record grows 267 to 275 bytes and a new item-weight section is written. `TestThePinIsTheVersion55PinPlusCarriedWeight` and `TestTheLoadTheCapacityAndTheWeightTableAreCanonicalState`. |
| Campaign/session | PASS | `mapload.PartyLoad` answers the same law for a party member with no entity, and `TestPartyLoadIsTheSameLawOutsideAMission` pins that the two agree. The card's line set survives a save and a load in the release continuity test. |
| Shipped content | PASS | `cmd/wearcheck -weights`, every (row, shape, material) triple of the three collections, both roots. Numbers below. |
| Interactions | PASS | The penalty composes with the existing formation speed (`DIV-226`) and is applied at the mover, so pathing and stepping are unchanged in shape. The milestone drive is byte-identical to master's. |

No in-scope GAP.

## The integration witness

`scenarios/1025-mission10-weight.json`, run by `pipeline/check-scenarios.sh` on both roots:

```
[3] create_character screen=map members=1 purse=100 docs=0
    hero Aeryn  entity=35 xp=1593 defense=23 absorption=0 load=25 capacity=201 skills=Air=10(1593)
```

It is a real campaign mission on real shipped assets, and the two asserted numbers are of different
kinds on purpose.

- **capacity 201** is the formula's own value for the body statistic the scenario states,
  `20 * 10 + 1`. It depends on no shipped table and could be written down before the run.
- **load 25** is what the shipped weight column resolves to for the starting weapon the generator
  gives this character. It is an ingestion pin, which is what an `install` scenario is for. It is the
  same on both roots.

The scenario reads the live entity's own fields, not a party projection, because the member has a
live entity id on the map screen. Its assertion arm is pinned install-free by
`TestHeadlessAMemberAssertionAnswersForTheCarriedLoadAndTheCapacity`.

## The shipped-content sweep

`AGAINROM_ASSETS=<root> go run ./cmd/wearcheck -weights`, identical on `en` and `ru`:

| Collection | Rows | Codes swept | Refused | Weight range | Negative | Zero | Heaviest |
|---|---|---|---|---|---|---|---|
| Weapons | 28 | 2240 | 80 | -2..2340 | 12 | 176 | row 26, shape 3, material 14 |
| Shields | 10 | 800 | 80 | 2..260 | 0 | 0 | row 8, shape 3, material 14 |
| Armors | 31 | 2480 | 80 | 1..260 | 0 | 0 | row 19, shape 3, material 14 |

The population is every (row, shape, material) triple, walked rather than sampled. The 80 refusals
per collection are exactly the reserved unwritten zeroth row, 8 shapes by 10 materials.

Two findings the synthetic tests cannot state. Twelve Weapons triples resolve to a **negative**
weight, which is reproduced rather than clamped (`DIV-225`). 176 resolve to zero, which is the
rounding sending a small positive product below 0.5 to nothing.

## The producer population

An item code reaches a world through four construction-time doors and one door from outside a
mission. Each has a test.

| Door | Site | Test |
|---|---|---|
| equipment slots | `declareItemWeights`, `pkg/mapload/itemweight.go` | `TestAWornItemIsDeclaredAndCountsTowardTheLoad` |
| containers | same | same pass, same test |
| ground sacks | same | `TestASackOnTheGroundIsDeclared` |
| a compiled script's item literals | same, reached only through `mapload.StartMissionScripted` | `TestAScriptsOwnItemLiteralIsDeclared` |
| an outside caller replacing a stock | `mapload.DeclareCodeWeights` | `TestDeclareCodeWeightsIsTheDoorForAnOutsideCaller` |

Why the population is complete: `sim.World`'s only writers of an equipment slot or a container
element are the map load that builds the world, the item-moving commands (which move codes that are
already in the world), and `ReplaceStock`. `FromALMWith` and `StartMission` pass a nil script, so
`StartMissionScripted` is the only place a script's item literals can reach the pass. `ReplaceStock`
has two production callers, `pkg/game/originalsave.go:288` and `pkg/game/world.go:2300`, and both
call `DeclareCodeWeights` first.

What the instrument cannot see:

- A code whose class field names no equipment slot resolves to no item definition and weighs
  nothing. That is documents, keys and the money bag, and it is the correct answer, but the sweep
  cannot distinguish it from a class this build failed to dispatch.
- A producer written later without a declaration. Nothing fails; that actor's load is understated.
  This is `DIV-223`, and it is the price of resolving above the determinism wall.

## The census

`pipeline/check-milestone.sh`, before: exit 0, "the script gap and the drive are where they were
recorded, both roots". **59** unsupported script nodes per root over 28 campaign maps; the drive
lost at tick 240; census 4 of 36 units moved, 1 fell, over 240 ticks.

After: **identical, line for line**. The census script drives `builds/current/missionrun.exe`, which
is master's binary, so the after run was made with a drive built from this worktree
(`go build -trimpath -o mr1025.exe ./cmd/missionrun`) driven through the script's own mission list
and its own argv, `-mission 10 -census -waypoint u21:56:21:3 -waypoint p0:66:16:3`. The output
diffed clean against `pipeline/milestone-baseline.txt`.

**Why it did not move, and the proof that the instrument is not blind to it.** A speed change would
move the drive. No speed changed, because the penalty arm requires `load >= capacity`: the party
member's load is 25 against a capacity of 201, and every placed unit has a capacity of zero, which
`overloadedSpeed` skips (`DIV-224`).

That reasoning was checked by mutation rather than asserted.

- Forcing every mover's capacity to 1 (`overloadedSpeed(base, e.Load, 1)`) left the drive's output
  **byte-identical**. The units that move in this drive carry nothing, so their load is zero and the
  penalty subtracts zero.
- Forcing every mover to the floor (`overloadedSpeed(base, 1000, 1)`) moved the drive a long way:
  the loss ran to tick **528** instead of 240, and all four movers took fewer steps and stopped
  elsewhere.

So the census does see this code path; nothing in mission 10 reaches the penalty's own gate.

## Mutations

Every one was applied to the production line a maintainer would actually change, run, and reverted.
`git diff` after the last revert showed only the intended edits, and no `MUTANT` marker remains
anywhere under `pkg/`, `cmd/` or `internal/`.

| # | Mutation | Result |
|---|---|---|
| M1 | `carryHalving` 2 to 1 | killed |
| M2 | the saturation compare `>=` to `>` | killed |
| M3 | `overloadFloor` 6 to 5 | killed |
| M4 | the penalty gate `load < capacity` to `<=` | killed |
| M5 | `moverSpeed` returns `base`, dropping the penalty | killed exactly one test, `TestAnOverloadedActorStepsSlower` |
| M6 | `fromalm.go` drops `declareItemWeights` | killed, 3 failures |
| M7 | `start.go` drops `declareItemWeights` | killed, `TestAScriptsOwnItemLiteralIsDeclared` |
| M8 | `capacityAddend` 1 to 0 | killed, 8 failures |
| M9 | `itemWeightColumn` 3 to 2 | killed |
| M10 | `scaleWeightSlot` 3 to 2 | killed |
| M11 | drop the `+ 0.5` | killed |
| M12 | the composer's own draw call, `ln.at.Y` to `ln.at.Y+1` | killed at the USE site: "drawn on rows 98..99, want 97..98" |
| M13 | move the WEIGHT row below XP in `CompactPanelLayout` | killed on both neighbour assertions |
| M14 | `moverSpeed` capacity forced to 1 | census unchanged; explained above |
| M15 | `moverSpeed` load 1000 capacity 1 | census moved: loss at tick 528, four movers changed |
| M16 | `capacityAddend` 1 to 0, against the new scenario | killed: `member "hero" capacity = 200, want 201` |
| M17 | `itemWeightColumn` 3 to 2, against the new scenario | killed: `member "hero" load = 973, want 25` |

M14 is the one mutation that changed nothing, and it is recorded because it is the measurement that
made M15 necessary.

## The derivation, and the comparison that came after it

The contract's first risk is fitting the arithmetic to the owner's observed card values. The
derivation was written from the claims alone, before those values were read, and is preserved in the
lane's own notes. What it produced:

- The per-item weight is `ftol(column3 * shape * material)`; the load is `ownWeight` plus half the
  container sum with a flat assignment past `0xfa00`; the capacity is `body * 10 + 1`; the penalty
  fires only at or above capacity and floors at 6.
- `actor+0x8e` is the **equipped-weight accumulator**, not a body weight. Nothing in any claim
  initialises it from a `Data.bin` column or a statistic, and every equip path adds to it while
  every inverse subtracts. **A character with an empty doll and an empty pack therefore has a load
  of zero and draws `0.0`.**
- The drawn row is the load under the sheet's own decimal-tenths convention, divisor 10, truncating.
- The magnitude prediction, written down before the comparison: "if a full kit sums to roughly
  100-500 raw units the drawn row reads roughly 10.0-50.0."

The comparison to the owner's three observed values afterwards: `FERGARD 0.0`, `DANATH 18.1`,
`BRIAN 45.7`. All three fall inside the prediction, and the zero is the case the derivation had
already explained rather than a value it had to accommodate. The same card's ATTACK row is likewise
absent for FERGARD, which is independent corroboration that his doll is empty.

**No divisor, column or ladder slot was adjusted after the comparison.** The derivation stands as
written.

## Research reconciliation

| Claim | Used for | State after this story |
|---|---|---|
| `ITEM-WEAPCOL-021` | the weight is runtime column 3 | reproduced |
| `ITEM-LADDER-019` | slot 3 of the two ladders scales it | reproduced |
| `HERO-EQUIP-017` | the item's own `+0x4a`, filled at construction, added by every equip | reproduced, with `DIV-223` for the mechanism difference |
| `ITEM-STACK-003` | the container sum is `weight * count` | reproduced; walked rather than accumulated |
| `ITEM-LOAD-005` | the load derive, the halving, the saturation, and that the penalty is its only consumer | reproduced |
| `HERO-SIGHT-007` | `capacity = body * 10 + 1` | reproduced; the population it runs for is `DIV-224` |
| `HERO-SPEED-008` | the penalty and its floor | reproduced; its composition with the group term is `DIV-226` |
| `HERO-104` | the sheet's fractional convention, and that neither identified site prints weight | the format is authored, `DIV-222` |

`DIV-209`, which recorded that this build had no weight value to draw, is **CLOSED** and moved to
`docs/DIVERGENCES-CLOSED.md`. Its residual question, the sheet's own source field and divisor, moved
to `DIV-222` rather than closing with it.

Ids spent: `DIV-222`, `DIV-223`, `DIV-224`, `DIV-225`, `DIV-226`. Returned unused and retired:
`DIV-227`, `DIV-228`, `DIV-229`. `DIV-191` was amended in place for the row's position in the card's
order.

## Gates

Run on the branch tip, on a clean tree.

| Gate | Result | Exit |
|---|---|---|
| `go build ./...`, `go vet ./...` (implementation) | clean | 0 |
| `gofmt -l` over tracked and untracked Go files | prints nothing | 0 |
| `go test -count=1 -trimpath ./...` | all packages ok | 0 |
| `scripts/check-claim-citations.sh` | ok, 1191 distinct citations resolve against 1400 claims and 209 experiments | 0 |
| `scripts/check-no-game-assets.sh` | clean (tree scan) | 0 |
| research `go build`, `go vet`, `go test` at pin `4d2df83` | clean | 0 |
| `research/scripts/check-claim-ids.sh` | ok, 1400 ids, 31 ledgers | 0 |
| `research/scripts/check-retraction-status.sh` | ok, 229 overturned ids, every one marked | 0 |
| `pipeline/check-release-tests.sh` en | selected 38, 38 of 38 ran and passed, 0 skipped | 0 |
| `pipeline/check-release-tests.sh` ru | selected 38, 38 of 38 ran and passed, 0 skipped | 0 |
| `pipeline/check-scenarios.sh` en | selected 14, ok 14 of 14 | 0 |
| `pipeline/check-scenarios.sh` ru | selected 14, ok 14 of 14 | 0 |
| `pipeline/check-milestone.sh` (master's drive, before) | ok, both roots where recorded | 0 |
| the same census with this branch's own drive (after) | identical to `pipeline/milestone-baseline.txt` | 0 |

Deletion set `git diff --diff-filter=D --name-only origin/master HEAD`: **empty**.

Commit trailers over `origin/master..HEAD`: **none**, on five commits.

The three install-gated tests that failed on the first run of the release gate were failing on
expectations the card's new row moved, not on behaviour: 16 rows to 17, `300x273` to `300x286`, and
20 sheet lines to 21. Each was updated with the reason recorded beside it, and the card's own fit
assertions, which had never run before because the row-count check is a `Fatalf`, pass at the
production font on both roots.

## Open items

- **The card has no room for an eighteenth row at the production font.** Its seventeen rows now end
  at card-local y=204, and the pane's three persistent controls begin at y=205. This is recorded in
  `pkg/game/towncard_release_test.go`'s own header. It is not a defect of this story; it is the
  budget the next story that wants a row has to work in.
- `DIV-222` stays open until a claim traces the sheet's `WEIGHT` row to the field it reads and the
  divisor it prints with. `HERO-104` names the untraced writer at `L04535`.
- `DIV-224` stays open until a claim names the entity kinds the actor recompute runs for. Until
  then a creature carrying loot is never slowed by it.
- `DIV-225` and `DIV-226` are research questions, not implementation debt.

## What was not done

- No refusal on weight anywhere, by exclusion.
- No encumbrance category, stamina or fatigue.
- Money is not in the load.
- The `SIGHT` row's own fractional convention is untouched.
- `builds/1025-carried-weight/` was not created; the brief excluded it.
- The card's row order, fonts, rectangles and chrome were not changed. The `WEIGHT` row was placed
  in the space `1022` left for it.
