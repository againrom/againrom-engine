# Verification — the player's own units can fight

Four task commits, `SDD-Task: 0078-hero-numbers/T1..T4`, on `impl/0078-hero-numbers` from master
`d22630d`, research pinned at `20921e2` with no leading character.

## The gate

Run in this seat, branching on the exit code and not on a grep:

```
go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -count=1 -trimpath ./...
```

`EXIT=0`, every package `ok`, `gofmt` and `vet` silent.

```
bash scripts/check-no-game-assets.sh ; bash scripts/check-doc-budget.sh ; bash scripts/check-sdd-audit.sh
```

All three `EXIT=0`. `check-sdd-audit` reports 0078 with no note of its own; the note count it prints
is the orchestrator's figure, not a lane's, and is not quoted here.

`formatVersion` is **12** and no bump was needed: the eight fields this story fills were already on
the entity and already in the record, so nothing about the layout moved.

## Was the zero a bug or a consequence? BOTH, and they were indistinguishable

Re-measured in this seat before any code was written, over `StartMission` with a one-member party:
`dmgBase 0, dmgSpread 0, toHit 0, defence 0, absorption 0` against 100 hp. The handed observation
stands.

`HERO-BARE-037` gives `base = spread = ftol(1.1^Body / 20)`, zero below Body 32; the party stood at
the Units constructor's Body 30. **A correct model of a bare hero at Body 30 produces exactly the
numbers we were producing by accident.** The zero was arithmetically a consequence and causally a
defect — the hero should not have been bare, because `HERO-START-039` hands a new campaign hero
exactly one weapon. Recorded at length in `analysis.md`.

## What witnesses what

51 test cases were added, all passing. The mapping is per criterion.

**SC-1 — the fold (FR-2, FR-5, FR-6; AC-2, AC-3, AC-4).**
`pkg/data/hero_test.go`. `TestTheFourMeasuredBandEdgesReproduce`: at Body 15 / 32 / 39 / 43 with one
skill at 10 and the sword at `(5, 3)` the sheet bands are `7-10`, `8-12`, `9-14`, `10-16` — the
owner's four, four of four. `TestTheBandStepsAtTheStatedBodies` walks Body 15..50 and finds the band
constant except at 32, 39, 43, 46 and 49, which is his *"the band is constant between them"* turned
into a prediction. `TestABareHeroDoesNothingBelowBodyThirtyTwo`: `0-0` at 15 and at 31, `1-2` at 32,
`2-4` at 39, `3-6` at 43, cadence `8/4` throughout.
`TestTheChargenStartWithItsStartingWeapon` pins the whole eight-field value.

**SC-2 — the resolution (FR-3; AC-5, AC-6, AC-9).** `pkg/data/weapon_test.go`, against a synthetic
table. `TestAWeaponWithNoShapeWordScalesThroughIndexZero` is the case the retracted ladder got wrong:
a leading word that is a material leaves the shape at index 0, so a row shipping 23 and 40 carries
`(5, 3)` and not `(23, 17)`. `TestALeadingShapeWordIsConsumedAndItsFactorMultiplies` shows the
to-hit factor is a *different slot of the same ladder*, not the damage one.
`TestATwoWordMaterialBeatsItsOneWordSuffix` is DD-3's own case, arranged so the two readings give 8
and 2. `TestTheSecondNumberIsASpreadAndNotAMaximum` and `TestTheScaleRoundsHalfUp` each name the
rival they kill. `TestAPrefixThatIsNotAWholeWordIsNotAMatch` and
`TestAnUnmatchedWordFallsToIndexZeroOfTheTable` fix the fallback as *index 0*, which is the item
constructor's own byte and not a name this tree goes looking for.
`pkg/formats/databin/doubles_test.go` pins the ladder itself: slot j at `8j`, a short record padded
rather than shifted, nil for a collection that writes none, and a copy rather than a view.

**SC-3 — the refusals (FR-4; AC-7, AC-8, P-3).** `TestARangedRowIsRefusedByName` (attack type 11,
the zero value, and the error naming both the row and the type), `TestARemainderNamingNoRowIsRefused`
and `TestAShortRowIsRefused`.

**SC-4 — the stat census (FR-2; AC-10, AC-11, AC-12).**
`TestMindAndSpiritReachNothing` moves each over `0..100` and finds the eight numbers unmoved — the
implementation side of `HERO-STATDMG-036`. `TestTheSkillMovesTheBaseAndTheToHitAndNotTheSpread` is
the asymmetry a fold adding the skill to both ends would pass at one Body and fail here.
`TestTheActiveSkillIsTheWeaponsKind`: an axe-trained hero holding a blade derives the *untrained*
numbers exactly. `TestAStatAboveFiftyDerivesAsFifty` binds the cap and checks it does not bind at 49.
`TestAbsorptionIsAlwaysZero` sweeps three weapons and three stats, and then shows a weapon's defence
column *does* reach defence — so the zero is a fact about absorption and not about the fold.

**SC-5 — the party (FR-1, FR-7, FR-8; AC-1, AC-13, AC-14, AC-18).** `pkg/mapload/hero_test.go`.
`TestAStartedPartyMemberCarriesItsHerosNumbers` measures at the seam the player reaches.
`TestAStartedPartyMemberKeepsTheHealthRateAndDomainItHad` pins the three disclosed divergences, so a
later story deriving one has to come through it. `TestAZeroValueMemberCarriesTheNumbersThePartyHadBefore`
is DD-8: the zero member reaches the pre-0078 numbers through the same arithmetic, no fallback.
`TestABarePartyMemberSwingsForNothingAtTheChargenStart` shows he is not inert — to-hit and defence
come off the stats alone. `TestAWorldHoldingAnArmedPartyRoundTrips` asserts the encoded version byte
is **12 by number** and round-trips to equal bytes and an equal hash. AC-18 is the whole suite,
green: every existing placement, spawn, digest and byte-form test is unchanged in outcome.

**P-2 — `pkg/sim` is not edited.** `git diff --stat d22630d..HEAD -- pkg/sim` is empty: no rule of
combat, no field, no version. The determinism wall is untouched from both sides —
`internal/archtest`'s source scan over `pkg/sim` is green above, and every float this story
introduces lives in `pkg/data`, which is BELOW the wall, so only truncated integers cross it.
**P-5 — the import graph is unchanged in direction.** `internal/archtest`'s allow-map test is green:
`pkg/data` gained `math` and `strings`, both stdlib and neither an intra-module edge, and
`pkg/mapload`, `pkg/game` and `cmd/missionrun` gained no import at all — `pkg/game` already imported
`pkg/data`, `pkg/formats/databin` and `pkg/mapload`.

**SC-6 — the front end (FR-9; AC-15, P-4).** `pkg/game/hero_test.go`:
`TestLoadDefinitionsResolvesTheStartingWeapon` (one walk, both outputs),
`TestATableWithNoStartingWeaponIsNotFatal` (the reason carried, the table still built, `LoadTable`
unaffected), `TestTheStartingWeaponLiterals`, `TestMissionPartyCarriesTheChargenHero` (including that
writing the returned slice cannot reach the next party), and
`TestAnInstalledTableBecomesTheHerosBand`, which runs the whole chain from a table's bytes to `7-10`
with nothing stubbed. `pkg/game/frontend_test.go`'s character-pin now asserts BOTH check lines, bare
and armed; `cmd/againrom/main_test.go` asserts them again over a whole synthetic install, with
`omitStartingWeapon` as the switch that makes them two cases.

**SC-7 — the truncation margin (P-1; AC-16).**
`TestNoTermOfTheDeriveSitsOnATruncationBoundary` sweeps Body `0..100` and every `(Body, Reaction)`
pair in `0..100`, and requires `1e-6`. Measured: damage term **8.976603e-3** at Body 46, to-hit term
**2.0484977e-4** at Body 47 / Reaction 59 — of order `1e12` ULPs, so `math.Pow` differing from the
CRT's by an ULP cannot move an integer this tree hashes.

**SC-8 — the instrument (FR-10; AC-17).** `cmd/missionrun/main_test.go`:
`TestAnAttackIsAttackerColonVictim`, `TestAMalformedAttackIsRefused` (eight forms), and the malformed
flag added to `TestTheRefusalsHappenBeforeAnArchiveIsOpened`, which is what keeps that file synthetic.
A run with no `-attack` prints exactly what it printed before — the campaign drive
`TestTheTenthMissionIsDrivenToAWin` is unchanged and green.

## Measured against the lawful install — BOTH ROOTS

`gameversions/en` and `gameversions/ru`, 2026-08-03. G1 needs both; both were run and they agree.

**The check line.** EN: `againrom: 38 map rows, 8 of 8 buttons have a mask region; hero Iron Short
Sword 7-10, to-hit 39, defence 8`. RU: the same clause after `34 map rows`.

**The five ordinary chargen literals resolve 5/5 on each root**, which is DD-3's falsifier discharged
— and every published figure reproduces from the install through this tree's own resolver, not from
the claim:

| literal | ours | `ITEM-DMGFACT-020` |
|---|---|---|
| `Iron Short Sword` | (5, 3), `+0x52` 5, `+0x6a` 0, `+0x50` 1 | (5, 3), 5, 0, 1 |
| `Uncommon Bronze Axe` | (5, 6) | (5, 6) |
| `Uncommon Bronze Mace` | (4, 5) | (4, 5) |
| `Bronze Pike` | (4, 4) | (4, 4) |
| `Uncommon Wood Short Bow` | (3, 2) | (3, 2) |

Five different `(shape, material, row)` triples, all agreeing, on two roots whose `Data.bin` files
differ byte for byte. Under the retracted ladder none of them would: the sword alone would read
`(23, 17)`.

**The four measured band edges, re-derived from the install** rather than from a fixture — Body 15 →
`7-10`, 32 → `8-12`, 39 → `9-14`, 43 → `10-16`, identical on both roots. The item's own sheet line is
`5-8`, which is what the owner reads for his sword.

**The kill.**
`missionrun -mission 10 -waypoint p0:13:49:2 -attack p0:u30 -ticks 30000`, identical on both roots:

```
waypoint 1  p0 -> (13,49) r2 : reached (11,50), Chebyshev 2, after 2194 ticks
attack 1  p0 -> u30 : FELLED it after 68 ticks, victim at -6 hp
```

**The player's own hero fells a mission 10 monster in 68 ticks** — about six of his own attack
cycles, which is what a 7-10 blow against 10 health, defence 10 and a to-hit of 39 comes to. He is
untouched at 100 health.

Owner-review artifact: `builds/0078-hero-numbers/` — `againrom.exe`, `missionrun.exe` and a README
giving the three invocations above and what to look for.

## What does NOT work, stated plainly

**An attack order does not carry the hero across mission 10's terrain.** Ordered from the party's own
drop cell onto `u30`, he walks from (17,66) to (14,56), stops on tick **267**, and does not move
again in 20 000 ticks.

**It is not this story's, and that was measured rather than argued.** The identical order run against
the party this tree placed BEFORE 0078 — a zero hero, no weapon — stops at the **same cell on the
same tick**: `start (17,66) -> stops (14,56), last move tick 267` for both. Nothing on the approach
path reads a combat number.

Two further measurements about it, so a movement story starts with facts and not with my guess. The
victim's own cell (13,49) is **closed** on the ground plane the world routes on, so a pursuit aimed
at it can never arrive; and the open ground between the stall and the victim is a **71-step detour**
for a Chebyshev gap of 7 — `(14,54)` is 2 steps from the stall and `(11,50)`, where the ordinary move
order got him, is 71. Monster-on-monster attacks over comparable distances complete (102, 148 and 165
ticks over three pairs), so the pursuit works and this case does not.

**A hypothesis, flagged as one because this story did not read the search:** that is what
`MOVE-COST-002` describes — the Chebyshev distance only sizes the route budget, so a detour needing
more slack fails and a substitute goal replaces it — which would make the behaviour **faithful**
rather than broken. The evidence above is what made me believe it; whoever takes it should verify
that evidence before building on it. `pkg/sim` movement is `impl/0076-terrain-cost`'s.

## What was NOT verified, and why

- **A hero's health, speed and reach.** Each is derived from these same stats by the original and
  none is among the eight numbers this story fills (`spec.md` FR-7). The party keeps `SpawnHP`, the
  constructor's rate and the simulation's one-cell reach, and `pkg/mapload/hero_test.go` pins all
  three so a later story cannot move one silently.
- **That a shipped campaign takes the ordinary chargen arm.** Research grades it **Medium** and this
  story carries the grade rather than resolving it. The owner's sheet listing his sword at `5-8` is
  this arm's blade weapon, which is corroboration from the game and is recorded as such.
- **The auto-hit bit's second effect** (`HERO-AUTOHIT-031`): the bit also skips absorption, and
  `pkg/sim`'s resolver applies absorption unconditionally. No party number is wrong — a hero never
  carries the bit — but eight monster classes are over-protected against. Read, not fixed; it is
  `pkg/sim`'s.
- **The mage arm's staff and the high arm's five literals.** The high arm is written down beside the
  ordinary one so the Medium above is one constant away from being tested; the staff needs a class
  axis and a spell-carrying weapon, and this tree has neither.
- **The point-buy cost table.** There is no screen to spend points on. The chargen *start* is
  implemented; the cost function is not.
