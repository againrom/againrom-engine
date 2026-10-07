# 0139 — verification

Environment: Windows 11, Go per `go.mod`, two lawful installs at
`<seat>/gameversions/{en,ru}`. Every command below was run and every block is its
output.

**Re-taken after the merge with `master`.** The branch was three landings stale when it came to
land, and the merge brought in the six-slot skill block, the extraction of the re-derivation out of
`pkg/game/world.go`, and carried stacks. The gate below is the merged, committed tree, not the
pre-merge one; the AC evidence further down was re-run on it unchanged unless a section says
otherwise.

## Gate

```
go build ./...                       clean
go vet ./...                         clean
go test -count=1 -trimpath ./...     all packages ok, 0 failures
gofmt -l $(git ls-files '*.go')      printed nothing
bash scripts/check-no-game-assets.sh check-no-game-assets: clean (tree scan)
bash scripts/check-doc-budget.sh     exit 0; every 0139 row ok, spec 13044/13312, plan <= 1.2 x spec,
                                     tasks <= 1.2 x plan
bash scripts/check-hotfix-ledger.sh  check-hotfix-ledger: ok
bash scripts/check-sdd-audit.sh      exit 0, no FAIL line for 0139-staff-casts
```

The note/warning **count** `check-sdd-audit` prints is not recorded here and is not comparable from
a worktree: `builds/` is untracked, so a lane emits none of those warnings until its own build stage
creates a directory and then emits every other story's at once. Only the FAIL set is enforced and
only it is quoted.

Deletion set, the one check a merge that builds because something vanished would still fail:

```
$ git diff --diff-filter=D --name-only origin/master...HEAD
(no output)
```

Trailer bijection, checked by hash rather than by report:

```
$ git log --format='%h %s | SDD:[%(trailers:key=SDD-Task,valueonly)] CO:[%(trailers:key=Co-Authored-By)]' origin/master..HEAD
eec3a45 0139: a staff's cast trains the school its own spell names (T4) | SDD:[0139-staff-casts/T4] CO:[]
8f60464 Merge origin/master into 0139-staff-casts                       | SDD:[] CO:[]
088d571 0139: verification, and the plan's own requirement citations    | SDD:[] CO:[]
7f998c6 0139: the wiring and a door to walk through it (T3)             | SDD:[0139-staff-casts/T3] CO:[]
e2ec2e1 0139: the attack cycle can cast, in place of a strike (T2)      | SDD:[0139-staff-casts/T2] CO:[]
8f5bc95 0139: a weapon carries the castSpell attachment as an id-shaped pair (T1) | SDD:[0139-staff-casts/T1] CO:[]
```

Four tasks, four trailered commits, each id once. The merge and the two evidence commits carry no
trailer, which is what a pipeline stage is owed. No `Co-Authored-By` on any commit in the range.

The research submodule is at `7eed909d01bcd83669a8c2e67e563ab1e37d4077`, `master`'s own pin,
unmoved by this story — read off `git ls-tree HEAD research` after the commit, not off the checked
-out directory.

## AC-1 — every shipped token resolves, measured on both roots

`cmd/wearcheck -spells` walks every trailing string of the `Humans` and `Units` collections,
resolves each `{castSpell=…}` name through `data.ResolveWeapon` and then the token through
`mapload.SpellIDByToken`, and fails if any token names no row.

```
$ go run ./cmd/wearcheck -assets .../gameversions/en -spells    (tail)
  Humans row=180  "Elven Magic Wood Staff {castSpell=Stone_Curse:30}"     token="Stone_Curse"      level=30   -> spell id 20
  Humans row=210  "Uncommon Wood Shaman Staff {castSpell=Fire_Arrow:10}"  token="Fire_Arrow"       level=10   -> spell id 1
  Units  row=26   "Boulder Thrower{castSpell=Fire_Ball:70}"               token="Fire_Ball"        level=70   -> spell id 2
  Units  row=27   "Boulder Thrower{castSpell=Fire_Ball:40}"               token="Fire_Ball"        level=40   -> spell id 2

48 of 48 token(s) resolved to a Spells row, 0 did not

$ go run ./cmd/wearcheck -assets .../gameversions/ru -spells    (tail)
48 of 48 token(s) resolved to a Spells row, 0 did not
```

Five distinct tokens appear — `Fire_Arrow`, `Fire_Ball`, `Lightning`, `Prismatic_Spray`,
`Stone_Curse` — at levels 1 to 99, across 46 `Humans` rows and 2 `Units` rows. The
underscore-to-space substitution is what carries all five; without it none resolves.

## AC-13, SC-1 — a generated mage in a real mission, on both roots

`missionrun -mage` opens mission 10 with the party's hero generated as a caster and drives him into
an attack. Same map, same victim, same tick ceiling; only the class axis differs.

```
$ go run ./cmd/missionrun -assets .../gameversions/<root> [-mage] -mission 10 -attack p0:u19 -ticks 2000

en   fighter  : attack 1  p0 -> u19 : FELLED it after 156 ticks, victim at 0 hp, attacker facing N
en   -mage    : attack 1  p0 -> u19 : FELLED it after 104 ticks, victim at -2 hp, attacker facing NE
ru   fighter  : attack 1  p0 -> u19 : FELLED it after 156 ticks, victim at 0 hp, attacker facing N
ru   -mage    : attack 1  p0 -> u19 : FELLED it after 104 ticks, victim at -2 hp, attacker facing NE
```

Identical on both roots. The mage's overshoot to −2 is the shape of the change: his last release
removed more than the health that was left, which a 2-2 swing cannot do.

**What the entity actually holds**, read straight off the started world before any tick — the mage's
staff, resolved and folded and minted:

```
party member: mage=true class=24 body="mage_st"
  weapon: "Wood Staff {castSpell=Fire_Arrow:10}" spell="Fire_Arrow" power=10 dmg=0+0 reach=5 attackType=3
entity 35: HP=72 MaxMana=36 Mana=36 WeaponSpell=1 WeaponSpellLevel=10 Reach=5 dmg=3+3 toHit=14
spell 1: range=7 dmg=4..8 targetsUnit=true damaging=true mana=3
```

**Tick by tick**, the same started world with the attack issued by hand and the attacker's own cycle
printed every tick. `phase=3` is the casting phase, `tgt` the entity the order names:

```
t=  0 me(35,54) phase=3 cd=6 tgt=2 tgtHP=31 tgtCheb=2
t=  5 me(35,54) phase=3 cd=1 tgt=2 tgtHP=31 tgtCheb=2
t=  6 me(35,54) phase=2 cd=5 tgt=2 tgtHP=22 tgtCheb=2      <- release, 9 removed
t= 19 me(35,54) phase=2 cd=6 tgt=2 tgtHP=13 tgtCheb=2      <- release, 9 removed
t= 24 me(35,54) phase=3 cd=7 tgt=0 tgtHP=10 tgtCheb=1      <- the engagement pass retargets
t= 31 me(35,54) phase=2 cd=6 tgt=0 tgtHP= 4 tgtCheb=1      <- release, 6 removed
```

Three releases, three landings, **no miss and no refusal**. 9, 9 and 6 are exactly `5 + U[0,5]`:
Fire Arrow's columns are 4 and 8, and at the staff's own level 10 the arithmetic gives base
`4*(10+30)/30 = 5` and spread `8*40/30 − 5 = 5`. **That is the owner's reported 5-10**, arrived at
from the level in the shipped item string and arithmetic that landed in 0127 — nothing was fitted to
it.

The retarget at t=24 is the engagement pass choosing a nearer candidate and is **not this story's**:
the same retarget happens with `approach`'s stop test reverted to the melee `inReach`, measured by
reverting that one line and re-running. It is why `-attack p0:u21` reports the named victim
unfelled — the caster moves on to the nearer enemy and the first one regenerates.

The probe that produced the two blocks above was a throwaway `main` inside the module, deleted
after the run; the tool-driven `missionrun` and `wearcheck` lines above are the committed,
reproducible evidence.

## SC-2 — the non-caster is untouched

The same command, same map, same victim, run against **`origin/master`** and against this branch:

```
origin/master (70b1c2b):  attack 1  p0 -> u19 : FELLED it after 156 ticks, victim at 0 hp, attacker facing N
this branch (en):         attack 1  p0 -> u19 : FELLED it after 156 ticks, victim at 0 hp, attacker facing N
this branch (ru):         attack 1  p0 -> u19 : FELLED it after 156 ticks, victim at 0 hp, attacker facing N
```

Tick for tick and hit point for hit point. A fighter's whole fight — the pathing, the cadence, the
to-hit rolls, the damage draws and the kill tick — is what it was, which also means the generator's
stream past every strike site is what it was. The milestone below is the second reading of the same
thing over a whole mission, and `AC-5`'s test is the third, on a synthetic world where the two
actors differ only in a mana pool.

## The milestone — unmoved

```
$ go run ./cmd/missionrun -assets .../gameversions/<root> -mission 10 -census \
      -waypoint u21:56:21:3 -waypoint p0:66:16:3

before this story (en):  outcome lost at tick 224   census: 4 of 36 moved, 1 fell, over 224 ticks
after  this story (en):  outcome lost at tick 224   census: 4 of 36 moved, 1 fell, over 224 ticks
after  this story (ru):  outcome lost at tick 224   census: 4 of 36 moved, 1 fell, over 224 ticks
```

No unit on that map takes the new branch: the mission's own placements carry no spell-bearing staff
on an actor with a mana pool. So the story neither advanced nor regressed the milestone, and both
roots agree.

## AC-2 to AC-12, AC-14, AC-15 — the unit evidence

Each is a test in the package that owns the behaviour, and each was proven load-bearing by
reverting the line that satisfies it, watching it go red, and restoring it.

| AC | Test | Line reverted |
|---|---|---|
| AC-2 | `data.TestResolveWeaponParsesTheCastSpellAttachment`, `…AgreeOnEveryOtherField` | the `SpellName:`/`SpellPower:` assignment in `ResolveWeapon`'s return |
| AC-3, AC-15 | `data.TestTakeCastSpellRefusesEveryMalformedShape` (7 subtests) | the colon guard, then the digit-range check, separately |
| AC-4 | `sim.TestAWeaponBorneCastReplacesTheStrike` | the caster test in `advanceAttack`'s ready arm |
| AC-5 | `sim.TestANonCasterOrANoSpellWeaponStrikesNormally` | `weaponSpell`'s `!isMage(e)` half |
| AC-6 | `sim.TestAWeaponSpellNamingNoLoadedRowStrikesNormally` | `weaponSpell`'s `findSpell` call |
| AC-7 | `sim.TestACasterStopsAtSpellRangeNotMeleeReach` | `closedOn`'s caster branch |
| AC-8 | `sim.TestRepeatedReleasesKeepTheWeaponAndCastAtTheSamePower` | the ready arm's caster test |
| AC-9 | `sim.TestAnEmptyManaPoolAndAnUnknownSpellStillCast` | **proven by insertion**, not reversion — a mana charge and a `knowsSpell` gate were each added to `releaseWeaponSpell` in turn, each turned the test red, and both were removed |
| AC-10 | `sim.TestFR3aBothDirectionsOfEligibilityChangingMidWindUp` | the divert comparison |
| AC-11 | `sim.TestFR10sFiveRefusalsLeaveTheVictimUntouched` (5 subtests), `…TouchesNothingButTheAttackersOwnCycle` | each of the release's five guards, one at a time |
| AC-11a | `sim.TestFR3bTheSitesOwnDrawIsIndependentOfEligibility` | the discard draw inside the divert |
| AC-12 | `sim.TestAReleasedCastReproducesTheCommandedCastsArithmetic` | the level passed to `applySpellDamage` |
| AC-14 | `sim.TestAWeaponBorneCastRoundTripsThroughTheByteForm`, `…ACastingCycleNoTickCanLeaveIsRefused` | the encode and decode lines; `attackFault`'s phase case |
| FR-13 | `mapload.TestAStartedPartyMembersWeaponSpellReachesHisEntity`, `…APlacedPersonsWeaponSpell…`, `…APlacedCreaturesWeaponSpell…` | the mint line in `start.go`, in `fromalm.go`, and the carry in `spawn.go`, each separately |
| D-14 | `game.TestRearmKeepsTheStartWeaponsSpellWhenTheSlotsCodeIsItsOwn` | the `w.Code != code` half of `rearm`'s guard |
| FR-14 | `game.TestMissionPartyAsResolvesTheMageArmsOwnWeapon` | the `resolveWeaponForSlot(true, …)` call |

AC-9's evidence is weaker than the rest and is recorded as such: an absent charge cannot be
reverted, so it was witnessed by adding the charge and watching the test fail.

## AC-16, FR-16 — accepted item effects train through the common sink

`TestStaffStoneCurseRaisesEarthFromPseudoDamageOnly` drives the whole attack cycle and checks the
exact Earth delta from the 3%-health pseudo-damage producer. `TestItemAreaTrainsPerDamageEventWithoutHalfManaAward`
separates two damage events from the much larger mana-cost rival.

`TestAWeaponBorneReleaseTrainsFromDamageNotMana` pins a caster's actual school and exact damage
delta. `TestTheFightersRiderTrainsTheCurrentWeaponSkill` isolates the fighter's spell contribution
from the later physical award. `TestARefusedReleaseTrainsNothing` holds the refusal side unchanged.

`TestPoisonTicksCarrySignedAwardsAndClearOnlyBelowZeroCasterHealth` pins the signed amount, zero and
negative caster-health boundary, and signed save/load round trip. `TestLowTypeHumanCannotTrainFromStaffDamage`
pins the recipient TypeID gate. Fighter rider coverage compares otherwise identical physical blows
with and without the spell event.

`TestDirectItemSpellKillUsesActualSpellForDelayedCredit` separates the direct damage award from the
same-tick half-XP death award and reads the actual point spell id after the resolver's temporary
kind. `TestDelayedDeathConsumesSurvivingAttributionOnce` crosses the death boundary on a later tick
and proves it cannot pay twice. `TestLaterPointAttributionRedirectsKillAndFighterUsesCurrentWeaponSkill`,
`TestAttributionClearAndRetainWritersAreShapeSpecific`, `TestDrainAndRefusedEffectsDoNotRewritePriorAttribution`
and `TestPoisonTickDoesNotRefreshOverwrittenAreaAttribution` cover the overwrite, fighter, clear,
retain, Drain, refusal and stale-Poison histories independently.

`TestKillAttributionRoundTripsHashesAndRejectsInvalidTails` reads the source, presence and signed
spell byte back from form 55, distinguishes a world without them in both bytes and digest, and
refuses a non-Boolean presence, an absent source with pointer residue and a present source naming no
entity. `TestThePinIsTheVersion54PinPlusItemAttribution` peels exactly those six bytes from every
record and reproduces both form-54 pinned digests.

## Digests

Simulation form 55 widens each entity record from 261 to 267 bytes: a four-byte source id, a
one-byte presence and a one-byte signed spell id at the record tail. The two independent pinned-form
builders write the tail explicitly, and the primary and routed pins match the live encoder byte for
byte. Peeling those six bytes from every record, restoring byte 0 to 54, reproduces the two exact
form-54 digests. The ordinary round-trip, malformed-form table, route builder and all secondary
digest pins then pass at the new width. `hash_test` still compares `World.Hash()` with FNV-1a of the
hand-transcribed form, so neither side learns its expected bytes from the encoder it checks.

## What is not verified

- **The `ru` root's own item strings were not read out one by one.** `wearcheck -spells` counted 48
  of 48 on both roots and the two tallies agree, which is the claim made; whether the two roots
  carry the *same* 48 strings was not compared string by string.
- **A staff taken from the ground casts nothing** (spec D-5). Verified as a limit, by the test that
  asserts it, not as a behaviour.
- **The caster's retarget after a kill-in-progress was not pinned by a test.** It was measured by
  reverting one line, and it belongs to the engagement pass rather than to this contract.
- **The install-facing evidence above was not re-run after the merge.** AC-1's token count, AC-13's
  mission drive and the milestone were measured on the pre-merge tree and are recorded as they were
  taken. Nothing the merge brought in touches the parse, the lookup or the trigger, and the whole
  synthetic suite is green on the merged tree — but that is an argument, not a re-measurement, and
  it is named here rather than left to look like one.
