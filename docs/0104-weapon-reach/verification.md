# Verification — 0104

Three tasks landed on `story-0104-weapon-reach`, one commit each:

| Commit | Task |
|---|---|
| `5b69eae` | T1 — `data.WeaponRange`, the shared `{…}` strip, `Combat.Reach` |
| `b3808f6` | T2 — `sim.Entity.Reach`, `strikeDistance`, byte form version 23 |
| `be7a7bd` | T3 — `unitReach` in `definitionFor`, the reach onto the entity |

The gate, on a clean tree at the branch's last commit, exit codes captured directly:

```
go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -trimpath -count=1 ./...   0
bash scripts/check-no-game-assets.sh                                                                    0
bash scripts/check-doc-budget.sh                                                                        0
bash scripts/check-sdd-audit.sh                                                                         0
git diff --diff-filter=D --name-only origin/master..HEAD                                                (empty)
```

## Witnesses, by revert

Every line below was verified by REVERTING it and watching a named test go red, then
restoring. The revert is stated; the test is what it broke.

| Id | Revert applied | What went red |
|---|---|---|
| FR-1, FR-4, AC-2 | `definitionFor`: `unitReach(c.EntryStrings(r.Index), t)` → `int32(1)` | `TestAUnitsReachIsItsRowsFirstResolvingWeapon` (`a bow, alone`; `armour first, then the bow`) |
| FR-2 | `stripSuffix`: the `IndexByte(name, '{')` truncation deleted | `TestATrailingSuffixIsStrippedBeforeMatching` |
| FR-3 | `UnitDef.Combat`: `Reach: d.Reach` → `Reach: 1` | `pkg/data` no longer builds — `d.Reach` becomes unused; restored and re-run green |
| FR-5 | `HumanDef.Combat`: the `if w != nil { c.Reach = w.Range }` arm deleted | `TestAnArmedPersonsReachIsHisWeaponsRange` |
| FR-6 | — see below; witnessed forward by `TestTheAdjustmentLeavesReachAlone` | |
| FR-7, DD-3 (decoder) | `reachFault`: `if e.Reach == 0` → `if false` | `TestUnmarshalRefusesAZeroReachAndAcceptsTheTopOfItsRange` |
| FR-7, DD-3 (constructor) | `newWorld`: `if cp[i].Reach == 0 { cp[i].Reach = 1 }` → `if false` | 12+ tests across `pkg/sim`, incl. `TestMarshalRoundTripsAndReMarshalsIdentically`, `TestAPatrollingActorRoundTrips`, every byte-form pin |
| FR-8, DD-1 | `strikeDistance`: the `d <= 0x180` arm's `return 1` → `return 0` | `TestStrikeDistanceEqualsTheChebyshevFloorOverASeparationRange` |
| FR-9, FR-10, P-5 | `inReach`: `int32(a.Reach)` → `int32(t.Reach)` | `TestAReachOfFourStrikesAtFourRefusesAtFiveAndStopsItsWalkAtFour` (`in reach at exactly the boundary`; `an approach stops exactly at the boundary and then strikes`) |
| FR-11 | `formatVersion` 23 → 22 | 12+ tests, incl. `TestUnmarshalRefusesEveryVersionButTwentyThree`, `TestMarshalledBytesArePinned`, `TestHashIsPinned`, `TestTheByteFormAndItsDigestAreTheOnesAlreadyPinned` |

**One mutant was equivalent, and the reason is worth recording.** The first FR-8 revert
tried was the threshold itself — `if d <= 0x180` → `if d <= 0x0` — and `pkg/sim` stayed
**green**. That is not an unwitnessed line: at a one-cell footprint the footprint term is
zero, so the only separation for which `d` falls inside `(0, 0x180]` is one cell, where
`d = 0x100` and the general arm answers `(0x100 + 0x40) >> 8 = 1` — the same 1 the early
return gives. The early return only becomes distinguishable once a footprint term shrinks
`d`, which is DD-6's owed story. Reverting the arm's **value** instead does discriminate,
and that is the revert recorded above.

The remaining ids are witnessed by tests that assert them forward, named here so the audit
and a reader see the same set:

- **AC-3** `TestAReachOfFourStrikesAtFourRefusesAtFiveAndStopsItsWalkAtFour`. Its four cases
  are the whole clause: in reach at exactly 4, refused at 5, an ordered approach that rests
  at (10,0) against a victim at (14,0) — four cells short — lands a blow there and does not
  step again, and a reach-1 attacker in the **identical** situation walking all the way to
  (13,0) before it strikes. **The contrast case was added in this stage.** The three tasks
  witnessed only the bow, and "stopped at 4" is consistent with a build that stopped everyone
  at 4; without the spear the audit was right that AC-3 was not witnessed.
- **AC-4** `TestAWorldHoldingSeveralReachesRoundTrips` (a world of reaches 1, 4, 100 and 255,
  byte for byte and digest for digest); `TestUnmarshalRefusesAZeroReachAndAcceptsTheTopOfItsRange`
  (0 refused, 255 accepted); `TestAVersion22FormIsRefused` (the previous version refused),
  with `TestUnmarshalRefusesEveryVersionButTwentyThree` sweeping all 256 first bytes behind
  it.

- **P-1** `pkg/sim` imports no definition table; `internal/archtest` holds the package to
  stdlib-only, tests included. `TestStrikeDistanceEqualsTheChebyshevFloorOverASeparationRange`
  reads two entities and nothing else.
- **P-2** same test: the answer is a function of the two cells at every separation 0..24,
  independent of reach, class and health.
- **P-3** `TestTheConstructorFoldsAZeroReachToOneAndLeavesOthersAlone` (construction),
  `TestUnmarshalRefusesAZeroReachAndAcceptsTheTopOfItsRange` (decode),
  `TestAUnitsReachIsItsRowsFirstResolvingWeapon` (load, floor case),
  `TestTheAdjustmentLeavesReachAlone` (difficulty).
- **P-4** `TestABareUnitDefinitionsCombatReachIsOne`, `TestABarePersonsReachIsOne`.
- **P-6** integer throughout; `internal/archtest`'s source scan over `pkg/sim` fails on a
  float identifier, a float literal, and on `os`/`time`/`math/rand`.
- **P-7** `TestARangedRowWeaponRangeAnswersAndResolveWeaponStillRefuses`, and the
  unresolvable-string case of `TestAUnitsReachIsItsRowsFirstResolvingWeapon` (reach 1, no
  load error).
- **P-8** `TestAWorldHoldingSeveralReachesRoundTrips` decodes from bytes alone.
- **DD-2** the assignment is `d.Reach = unitReach(...)`, not `1 + (range − 1)`; the two agree
  over the whole shipped table because no unit row names two equipment strings (measured
  below: 26 rows, 26 resolving, none with a second).
- **DD-4** `groupScorerReach = 1` in `engage.go` is a literal and is not wired to
  `Entity.Reach`; `engage_test.go`, `groupcmd_test.go`, `partyslot_test.go` and
  `release_test.go` are unchanged in behaviour across the story.
- **DD-5** no cadence value moves. `git diff origin/master..HEAD -U0` mentions the four
  cadence names on 12 changed lines, and every one is incidental — a composite literal
  reformatted, or a test entity literal rewritten to carry `Reach: 1`. Each name's values
  appear the same number of times on the `-` side as on the `+` side and are identical:
  `AttackChargeTime: d.AttackChargeTime` 1 and 1, `AttackRelaxTime: d.AttackRelaxTime` 1 and
  1, `AttackCharge: 8` 5 and 5, `AttackRelax: 4` 5 and 5.
- **DD-6, DD-7** disclosed, not implemented: no footprint field, no projectile.

## AC-1 — the data measurement, both roots

Taken through this tree's own loaders (`game.LoadDefinitions` → `data.WeaponRange`) against
`gameversions/en` and `gameversions/ru`. **The two roots agree in every figure below.**

```
units rows: 119 total (56 parameterised), 26 carry an equipment string,
            26 of those resolve, 0 unresolved
rows with derived reach > 1: 18
  reach  4  Bat_Sonic .. Bat_Sonic.4        Sonic Beam                     (4 rows)
  reach  4  Goblin_Sling, Goblin_Sling.2    (Uncommon) Wood Short Bow
  reach  4  Orc_Bow, Orc_Bow.4              (Uncommon [Magic]) Wood Short Bow
  reach  5  Goblin_Sling.3, .4              (Uncommon) Wood Long Bow
  reach  5  Orc_Bow.2, Orc_Bow.3            (Uncommon) Wood Long Bow
  reach  8  Dragon .. Dragon.4              Flame Thrower                  (4 rows)
  reach 20  Ballista                        Boulder Thrower{castSpell=Fire_Ball:40}
  reach 20  Catapult                        Boulder Thrower{castSpell=Fire_Ball:70}
distinct reaches: 1(x8) 4(x8) 5(x4) 8(x4) 20(x2)
```

This re-derives the stage-1–3 lane's numbers exactly: 26 rows carry a string, 26 of 26
resolve, 18 exceed 1, and the five distinct weapons are Short Bow 4, Long Bow 5, Sonic Beam
4, Flame Thrower 8, Boulder Thrower 20.

**Two of those rows are why FR-1 exists as a second entry point, and both are measured
rather than argued.** The four `Dragon` rows are the only ones the existing `ResolveWeapon`
refuses — `attack type 11 takes the ranged equip arm, which is not modelled` — so a story
that had reused it as the range resolver would have silently lost the Dragon's 8. And the
`Ballista`/`Catapult` rows are the only two shipped names carrying a `{…}` suffix, so FR-2's
strip is load bearing against a real install and not only against a fixture.

The party hero's start weapon reads `range=1 attackType=1` on both roots.

## AC-2 — a placement, through the loader, against an installed root

`StartMission(N)` addresses `scenario/N.alm`, and the shipped campaign names its maps `10`,
`20`, `30`, `31`, `40`, `41`, `50`, `100`, `101`, `110`, `111`, `120`… — so of `N` in 1..28
exactly two resolve to a file, on either install. The other twenty-six campaign maps are
reachable by name rather than by number and this measurement says nothing about them.

| Mission | Entities | Reach 1 | Reach 4 |
|---|---|---|---|
| 10 | 36 | 32 | 4 — `u44` (62,24), `u45` (50,16), `u46` (56,12), `u47` (70,12), all owner slot 2 |
| 20 | 57 | 34 | 23 |

Identical on both roots. The four on mission 10 are the map's bow and sonic placements; on
mission 20 nearly half the roster carries reach 4.

## AC-5 — the re-pin

Every literal digest and byte-form offset/length pin the version bump moved was re-pinned
from a run in T2's commit, and the complete `old -> new` list is that commit's body
(`git show b3808f6`). It covers 31 `_test.go` files across `pkg/sim` and `pkg/mapload`;
the headline pins are `pinDigest 0xbc1dc5d5eef60d85 -> 0xc400c725bab385c3`, `rtfDigest
0x5a8a000b43c044bc -> 0x030da70bda8dd508`, and `pkg/mapload`'s `gfDigest
0xa59d2f08ce28ff9a -> 0x9697af552efac26e`. T3 moved no pin: every pinned fixture either
carries no full item-collection set or has no unit row naming an equipment string.

## AC-6 — the tenth mission, before and after, both roots

`cmd/missionrun`'s `TestTheTenthMissionIsDrivenToAWin`, with the same two waypoints it
carries, run at `origin/master` (47bd279) and at this branch's head:

| | EN | RU |
|---|---|---|
| before | `outcome lost at tick 272` | `outcome lost at tick 272` |
| after | `outcome lost at tick 272` | `outcome lost at tick 272` |

**The drive did not move, in either direction, on either root.** Nothing was changed to make
it move, and the loss is the known recorded state rather than anything this story
introduced. Two further A/B measurements say why, both run at both commits:

- `missionrun -mission 10 -census -ticks 600` — identical at both: two movers (`u57`, `u58`),
  0 fell.
- `missionrun -mission 20 -census -ticks 2000` — identical at both: 0 of 57 moved.

Nothing on either map engages on its own inside those budgets, so the four (and twenty-three)
entities that now carry reach 4 never take an engagement decision during the drive. The
reach reaches the entities — AC-2 measures that — but on these two maps no hostile ever gets
as far as choosing whether to close.

## What was found wrong, and what was done about it

**1. `spec.md` FR-7 contradicts `plan.md` FR-7, DD-3 and `tasks.md` T2.** FR-7 says zero
reach is *"refused by the constructor and by the decoder alike, in the same words"*; the
plan and DD-3 say the constructor **normalises** and only the decoder **refuses**. The plan
is right and the clause is left standing: refusing at construction would turn every entity
literal in `pkg/sim`'s tests red for nothing a reader learns from, which is the very reason
DD-3 exists. What shipped is the plan's split, and both halves are witnessed above.

**2. The version test's name was left one bump behind — again.** T2 moved `formatVersion` to
23 and left `TestUnmarshalRefusesEveryVersionButTwentyTwo` named for 22. That function's own
doc block warns about precisely this (*"the name moves with the number so the two cannot come
apart"*) and records having been left stale once before, through 0099's bump. Renamed to
`TestUnmarshalRefusesEveryVersionButTwentyThree` in this stage's commit, with the second
lapse written into the comment. This is a test rename in an untrailered commit; it touches
no production line and moves no pin.

**3. `tasks.md` T2 misplaces one sentence.** It asks for the rewrite of *"the sentence in
`approach`'s comment calling the stop distance a Chebyshev distance of one cell"*. That
sentence is in `inReach`'s own doc comment; `approach` refers to the stop distance and never
says Chebyshev. `inReach`'s comment was rewritten, which is where the sentence was.

**4. A gap the story leaves open, disclosed rather than closed.** `pkg/mapload/start.go:261`
builds each **party member's** combat block through `data.Hero.Derive(p.Weapon)`, not through
`HumanDef.Combat`, and `Derive` deliberately sets no reach — so an armed party hero's reach
comes out of the constructor's floor at 1 rather than off his weapon. FR-4 and FR-5 are
written about a **placed** unit and a **placed** person, and a party member is generated
rather than placed, so this is outside the contract as written; but it is a third population
and the spec does not name it. It is invisible today and measurably so: the start weapon
reads `range=1` on both roots, so the floor and the weapon agree. It becomes visible the
first time character generation hands the hero a bow. **Owed by whichever story gives the
party its equipment channel.**

**5. `const reach` had a second reader.** Deleting it (T2, as instructed) broke
`candidateCost`'s group-order scorer in `engage.go`, which used the same constant as its own
fixed veto distance. Replaced by a locally named `groupScorerReach = 1`, documented as
deliberately not wired to `Entity.Reach` — which is DD-4 held, not a new decision. Five files
outside T2's named list moved for it, all named in that commit.

**6. `missionrun` cannot show this story's behavioural change on the shipped campaign.** Six
`-attack` and `-waypoint` constructions were tried across missions 10 and 20; each either
landed no blow at all in **both** trees or ended in mission 10's own pre-existing loss, so
none discriminates. The behaviour is witnessed by `pkg/sim`'s tests instead, and the
loader-side effect by the AC-2 census above. This is a limit of the instrument on these two
maps, not a claim that the behaviour is absent.
