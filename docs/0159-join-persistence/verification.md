# 0159-join-persistence — verification

## Gate

Run in the lane worktree `wt-0159-join` on a clean tree:

```
go build ./...                          clean
go vet ./...                            clean
gofmt -l $(git ls-files '*.go')         printed nothing
go test -trimpath -count=1 ./...        ok, every package
scripts/check-no-game-assets.sh         PASS
scripts/check-doc-budget.sh             PASS
scripts/check-sdd-audit.sh              FAIL set empty
scripts/check-hotfix-ledger.sh          PASS
research/scripts/check-*.sh (glob)      PASS
```

No `tasks.md` was written. The slice was implemented in one lane context, so every `FR`, `AC`, `P`
and `DD` is accounted for below instead.

## The pointable result

The result is in the game, not in the census. `pipeline/check-milestone.sh`'s script-gap numbers are
unchanged, which is a claim and was measured:

| Drive | Baseline | This branch |
|---|---|---|
| mission 10, `missionrun -trace -ticks 1`, `UNSUPPORTED` lines | 17 | 17 |
| mission 20, same | 11 | 11 |

`bash pipeline/check-milestone.sh` over both preserved roots differs from
`pipeline/milestone-baseline.txt` by nothing. This story implements no new script opcode: instants 19
and 22 were already runnable, which was checked before any code was written.

What changed is what those instants do, and it is visible in the shipped mission 40. The new scenario
`scenarios/0159-mission40-join.json` drives that mission's own script: the player walks to the
paladin, the map's `Paladin` trigger fires, and the paladin changes owner and group.

```
$env:AGAINROM_ASSETS='<seat>\gameversions\en'
go run ./cmd/againrom --headless scenarios\0159-mission40-join.json
```

| Build | Step 6 assertion | Result |
|---|---|---|
| master (`e436271`) | `u6` owner 1, group 17 | FAIL: `unit u6 group = 14, want 17` |
| this branch | same | PASS |

Both builds reach the hand-over at tick 2034 with the same tick count and the same alive/fallen
counts. Master leaves the paladin in the group the map placed him in; this branch gives him a group
of his own, which is `PARTY-JOIN-025`'s third write.

The world digest at tick 0 differs between the two builds (`0cf57d8137675843` on master,
`268a1bbb14597102` here). That is FR-1 reaching the byte form: a person now carries a band server
type id where he carried zero. It is expected and disclosed, not a regression.

Mission 40's script was read out of the shipped map to confirm which trigger issues the hand-over.
Action id 4, `TransferPaladin`, is opcode 22 over group 14 to player 1. It is named by trigger
`Paladin` — condition `NearPaladin`, opcode 15, the least distance over player 1's living units to
cell (77,112), compared against the constant 6 — and by trigger `Win` beside the win declaration.
The scenario fires the first of the two. The two checks this build cannot evaluate on mission 40,
both opcode 16, belong to other triggers (`Item to point`, `TreaTaken`) and do not gate the
hand-over.

## FR-1 — construction mode decides the server type id

`pkg/mapload/fromalm.go` preserves the Humans row's TypeID and overwrites it with
`sim.HumanTypeID` only for `ArmNPC` when `NPCDefs.Hero` finds the exact token.
`pkg/mapload/start.go` separately assigns `sim.HumanTypeID` to a party mint.

Witnessed by `TestNPCDefsPreservesTheExactHeroConstructorFlag`,
`TestAPlacedPersonAndAPlacedCreatureCarryDifferentTypeIDs`,
`TestAHandedOverLowTypeHumanIsMissionOnly` and `TestAMintedPartyMemberCarriesABandTypeID`.

The death-gold clause is witnessed by `TestABandTypeIDPaysNoDeathGold`, which sets the gold chance to
100 and both treasure bounds to 500 and reads zero for the band values and non-zero for the value
above `0x40`.

## FR-2 — a hand-over moves the actor and gives it a group of its own

`pkg/sim/script.go`'s new `handOver`, called by both arms.

Witnessed by `TestTheGroupArmGivesEachMemberAGroupOfItsOwn` (three members, three distinct groups,
none of them the old one, the two non-members untouched, and the owner write still made),
`TestAHandOverClearsTheCommandGroup`, `TestAnAbsentUnitReferenceWritesNoGroupEither` over four
refusal shapes, and `TestAFreshGroupIsAboveEveryGroupTheScriptNames`.

The clause about not becoming a primary character is witnessed by FR-4's own membership assertions:
nothing in the hand-over path writes a starting-hero flag, and
`TestTheJoinerIsAPersistentMemberAndNotAPrimaryCharacter` reads it false.

## FR-3 — the boundary keeps the human participant's band survivors

`pkg/sim/persist.go`'s `World.BoundarySurvivors`.

Witnessed by `TestTheBoundaryKeepsOnlyTheLiveBandActorsOfTheParticipant`, over a world holding a
live band actor of the participant, a dead one, a live out-of-band one, a live band actor of another
slot, and a live zero-type-id one. Exactly the first is kept, and the other slot's own band actor is
kept under that slot.

## FR-4 — a band survivor that was not a roster member becomes one

`pkg/mapload/carry.go`'s `CarryRoster`, over `pkg/mapload/fromalm.go`'s `FromALMRoster` templates
carried on `Start.Roster`, reached from `pkg/game/frontend.go`'s `FinishMissionWithRoster`.

Witnessed by `TestAHandedOverPersonEntersThePartyWithWhatHeCarried` — the fixture person wears a
resolved weapon and the script puts one item in his pack, and both clauses fail the test if either is
empty — `TestTheJoinerIsMintedFromHisOwnRowOnTheNextMap`,
`TestTheJoinerIsAPersistentMemberAndNotAPrimaryCharacter`,
`TestAHandedOverCreatureDoesNotEnterTheParty`, and at the front-end tier
`TestAWonMissionCarriesTheJoinedCompanion`, which reads the companion out of `NextParty` and not
only out of a field.

## FR-5 — persistence is unbounded

No code. Witnessed by `TestTheJoinerSurvivesASecondBoundaryExactlyOnce`, which wins a second mission
with no hand-over in it and reads the same identity and the same carry on the other side.

## Acceptance

| Id | Witness |
|---|---|
| AC-1 | `TestNPCDefsPreservesTheExactHeroConstructorFlag`, `TestAPlacedPersonAndAPlacedCreatureCarryDifferentTypeIDs`, `TestAHandedOverLowTypeHumanIsMissionOnly` |
| AC-2 | `TestAMintedPartyMemberCarriesABandTypeID`, `TestABandTypeIDPaysNoDeathGold` |
| AC-3 | `TestTheGroupArmGivesEachMemberAGroupOfItsOwn` |
| AC-4 | `TestAHandOverClearsTheCommandGroup` |
| AC-5 | `TestAnAbsentUnitReferenceWritesNoGroupEither` |
| AC-6 | `TestTheBoundaryKeepsOnlyTheLiveBandActorsOfTheParticipant` |
| AC-7 | `TestAHandedOverPersonEntersThePartyWithWhatHeCarried` |
| AC-8 | `TestTheJoinerIsMintedFromHisOwnRowOnTheNextMap` |
| AC-9 | `TestTheJoinerIsAPersistentMemberAndNotAPrimaryCharacter` |
| AC-10 | `TestTheJoinerSurvivesASecondBoundaryExactlyOnce` |
| AC-11 | `TestAHandedOverCreatureDoesNotEnterTheParty` |
| AC-12 | `scenarios/0159-mission40-join.json`, run above against the preserved `en` root |
| AC-13 | `TestAHandedOverLowTypeHumanIsMissionOnly`; `scenarios/0163-mission-to-town.json` exact two-member assertions at town and mission 30 |

## Properties

| Id | Witness |
|---|---|
| P-1 | `TestTheBoundaryAnswerDoesNotDependOnStorageOrder`, over the same six actors in both storage orders |
| P-2 | `TestASecondHandOverKeepsTheSlotAndTakesASecondGroup` |
| P-3 | `TestCarryRosterWithNoRosterIsCarryParty`, `TestFinishMissionWithNoRosterCarriesOnlyTheMembers`, and the eleven existing `CarryParty` test files, which are unchanged and pass |
| P-4 | `TestALostMissionCarriesNoJoinedCompanion`, over a lost mission whose world holds the band survivor and whose start holds his template |

## Plan decisions

| Id | Where it landed |
|---|---|
| D-1 | Table TypeID for zero-mode Humans; `sim.HumanTypeID` only for player-character mode |
| D-2 | The existing `Entity.TypeID`; no new field |
| D-3 | No format version consumed. **Version 47 is returned unused**: the byte form's layout did not move |
| D-4 | `handOver` calls `freeCommandGroup` |
| D-5 | `ScriptInstantGiveGroup` collects member indices before the first write |
| D-6 | `handOver` clears `CommandGroup` before taking the fresh id |
| D-7 | `BoundarySurvivors` is a `pkg/sim` method |
| D-8 | `FromALMRoster`; `FromALMWith` is a wrapper |
| D-9 | `Start.Roster` |
| D-10 | `CarryRoster` beside an unchanged `CarryParty` |
| D-11 | `rosterTemplate` writes `join:<entity id>` |
| D-12 | `CarryRoster` skips a survivor with no template |
| D-13 | No mark; FR-5's witness is the second boundary |

## Divergences as built

- **DIV-1 retired** — zero-mode Humans now preserve their row TypeID; the `0x21` simplification is
  limited to the admitted player-character constructor mode.
- **DIV-2** — The boundary re-mints rather than resetting eight fields in place.
  `TestTheJoinerIsMintedFromHisOwnRowOnTheNextMap` reads the joiner at full health on the next map,
  which is what the original's own reset of each pool from its maximum produces.
- **DIV-3** — The joiner's name is his placement's row name, or an npc subscript where the placement
  took the scenario npc arm.

## Not done

- The joined companion is not read out of an owner-produced save. Out of scope in `spec.md`.
- Mission 40 was not driven to a win. The hand-over's other issuing trigger, `Win`, needs variables
  77 and 78 both set, which needs the orc group killed and the treasure taken; the party is one hero
  against six creatures and an unattended drive does not achieve it. The trigger that was fired is
  the one the owner's own play reaches first.

## Mission 20 boundary correction

`TestNPCDefsPreservesTheExactHeroConstructorFlag` pins the constructor discriminator rather than a
template-name guess. `TestAHandedOverLowTypeHumanIsMissionOnly` transfers a zero-mode Human, reads
its unchanged low TypeID and `GainsXP=false`, then proves `CarryRoster` removes it. The mission-20
production data witness reads all three group-16 `NPC14_1` Clubmen through their canonical Humans
rows with the authored mace and shield still equipped while alive; corpse suppression remains a
separate death-time rule.

The original-save scenarios now assert an exact party size of two after mission 20 reaches town and
again after mission 30 starts: Danath survives, fresh `AddHero=22` Reniesta is created at the town
boundary, and Sarindar plus the three transferred guards are absent. `0152-save666.json` repeats the
same exact count after the implementation save/load round trip. Mission 30's type-14 tavern stock is
therefore a fresh roster offer reusing `NPC14_1`, not one of mission 20's actor identities.
- No windowed game was launched from this lane.
