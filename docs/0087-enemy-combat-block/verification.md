# Verification — 0087

Worktree `wt-0087`, branch `impl/0087-enemy-combat-block`, **rebased onto master `a41ad33`**
(0086's merge plus its pin bump); originally forked from `8d34461`. Research submodule at the new
pin `96f0b15`, and every claim this story cites was re-witnessed there as still active —
`HERO-COMBAT-011` remains partially retracted in the clause `provenance.md` already excluded, and
none of the others is retracted or amended against this story's use. Its status shows `git submodule status` showing no leading
character. Roots: `gameversions/en` and `gameversions/ru`.

## The byte form was not touched, at all

```
$ git diff --name-only a41ad33 HEAD -- pkg/sim
$ git diff --diff-filter=D --name-only a41ad33 HEAD
$
```

Both empty against the **new** merge base, re-run after the rebase rather than carried over from the
old one.

**No file under `pkg/sim` is in this story's diff.** That is FR-8 and AC-13, and it is a stronger
statement than a test asserting the version literal — which only says the number is still the one the
story was written beside. **No `formatVersion` was taken: not 15, not 17, none.** The re-allocation
to 17 arrived mid-story and does not apply, because this change adds no field: every number it fills
was already declared, already carried in the canonical form at its own offset and width, and already
read by a tick. What moves is the digest of any world holding a person a map placed.

## Gates, each branched on its own exit code

| Gate | Exit |
|---|---|
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l $(git ls-files '*.go')` | 0, and the list is empty |
| `go test -count=1 -trimpath ./...` | 0 |
| `bash scripts/check-no-game-assets.sh` | 0 |
| `bash scripts/check-no-game-assets.sh --history` | 0 |
| `bash scripts/check-doc-budget.sh` | 0 |
| `bash scripts/check-sdd-audit.sh` | 0 |

The audit was **1** until this file existed, on `FAIL 0087-enemy-combat-block: every task in tasks.md
has landed and there is no verification.md` — the only FAIL it reported, and the one this stage
closes. Its note and warning counts are not quoted: a worktree has no `builds/`, so that part of the
script is guarded off and the count is meaningless from here.

```
$ git diff --diff-filter=D --name-only 8d34461 HEAD
$
```

The deletion set is **empty**.

## What the lawful installs give, and it is the whole story

`classdump -databin <root>/world.res 10.alm 2`, mission 1, EN and RU, **identical on both roots**:

```
BEFORE (master 8d34461)
      0 0x000a/0x0000       8      4      0        0       0        0          0   false  none - the server-id arm reached no unit definition
      1 0x000a/0x0000       8      4      0        0       0        0          0   false  none - the server-id arm reached no unit definition
      0 key 0x000a/0x0000  arm server-id entry  203  healthMax   100  speed   10  domain ground

AFTER
      0 0x000a/0x0000       7      4      3        6       0        2          1   false  "M10_Brigands"
      1 0x000a/0x0000       7      4      3        6       0        2          1   false  "M10_Brigands"
      0 key 0x000a/0x0000  arm server-id entry  203  healthMax    15  speed   16  domain ground
```

Columns are charge, relax, to-hit, defence, absorption, damage base, damage spread, always-hits.

**The two brigands who dealt `0-0` now deal `2-3`.** Their health falls from the provisional 100 to
their row's own **15**, their rate rises from the constructor's 10 to their row's **16**, their
cadence becomes their row's **7/4**, their to-hit **3** and their defence **6**. They are the pair
mission 1's second win trigger requires dead, and until this story nothing on the map could hurt the
player at all.

The arithmetic is checkable by hand and was derived before it was run. Body 5, Reaction 20: bare
damage is `ftol(1.1^5 / 20) = ftol(0.0805) = 0` into both halves — `0-0`, and the handed hypothesis'
mechanism arriving at the right row by a different road. To-hit is `ftol((1.1^5 + 1.1^20)/5) =
ftol(1.667) = 1`; defence is `20/3 = 6`. The `Wood Club` his row names then adds base 2, spread 1 and
to-hit 2, and states no cadence of its own, so his template's 7/4 stands. **The whole of the
difference between 0-0 and 2-3 is one equipment cell.**

Both `speed 16` and `healthMax 15` are the row's own cells; nothing here is derived that has a
column, and nothing here has a column that is derived.

**A limitation, stated rather than glossed:** AC-14's second half — driving `missionrun` so a party
member fells a brigand — was **not run**. The tool addresses a unit by the map script's own
identifier and the brigand placements carry none, so the drive cannot name them without a facility
this story did not build. What is witnessed is the numbers, on both roots, off the built world rather
than off a second reading of the table.

## The mission-1 milestone drive, measured across three tips

`TestTheTenthMissionIsDrivenToAWin`, `cmd/missionrun`. It guards on `AGAINROM_ASSETS` and SKIPS
without it, which is why a gate can be green over it.

| Tip | Outcome | Where the escorted unit stopped |
|---|---|---|
| `8d34461` (this story's old base, before 0086) | **won at tick 2608** | reached its waypoint |
| `a41ad33` (0086 on master, before this story) | **lost at tick 480** | (43,46), Chebyshev 25 short |
| this branch rebased onto `a41ad33` | **lost at tick 272** | (44,46), Chebyshev 25 short |

Identical on EN and RU. The first two rows were run in this seat before rebasing, so the baseline is
held here and not taken on report: my own pre-rebase tip still WON, which places the break in 0086
and not in this story.

**This story makes it fail faster and not differently.** The interception point is the same 25 cells
short; only the clock moves, from 480 ticks to 272. That is the expected direction: this branch arms
21 hostiles on that map who carried `0-0` and could not previously hurt anything, so a unit that was
already being intercepted now dies at it. **It is a measurement, not a regression to chase** — the
test is not fixed and not edited here. Its likely cause is a divergence 0086 disclosed: with no
line-of-sight term a group sees through walls and acquires strictly more than the original, and
research is open on exactly that.

## Mutation sweep — twelve, all run against `./...`, all killed

Never against the package under change: two of these are killed **only** with the help of a package
two tiers away, which is exactly what a narrow run would have hidden.

| # | Mutation | Killed by |
|---|---|---|
| M1 | humans slot 9 skipped, shifting every slot after it | `pkg/data`, `pkg/mapload` |
| M2 | slot 18 not consumed, so the cursor drifts one | `pkg/data`, `pkg/mapload` |
| M3 | health does not follow the maximum | `pkg/data` |
| M4 | the template cadence not written, so the bare pair leaks through | `pkg/data`, `pkg/mapload` |
| M5 | the weapon's empty cadence cell assigned onward | `pkg/data`, `pkg/mapload` |
| M6 | the same, in the generated character's derive | `pkg/data` |
| M7 | the weapon search takes the first cell, not the first that resolves | `pkg/mapload` |
| M8 | a table with only some item collections still arms | `pkg/mapload` |
| M9 | the person arm takes the unresolved arm's health | **`cmd/classdump`**, `pkg/mapload` |
| M10 | the person arm is asked before the creature arm | **`cmd/classdump`**, `pkg/mapload` |
| M11 | the creature accessor returns the damage pair crossed | `cmd/classdump`, `pkg/data`, `pkg/mapload` |
| M12 | the trailing-strings accessor returns the wrong entry | `pkg/formats/databin` |

Each was applied, `go test -count=1 -trimpath ./...` run, the exit code branched on, and the file
restored; the tree was `git status --porcelain` clean afterwards.

## The two judgment gates, and what they cost

Both were run from a separate context and both found real defects, folded back in
`86677f8` before any of the affected code was written.

The **peer-prediction** read found the refusal bound off by one (twenty-three cells is every slot
present and must LOAD, not be refused); a self-contradictory sentence saying the streamed skill array
is overwritten when the derivation actually READS five of its six; and a direct contradiction between
the out-of-scope list and AC-7 over what happens to a ranged weapon.

The **adversarial** read found the one that would have shipped: the weapon resolver stores an empty
cadence cell verbatim and the generated character's derive assigned it onward, which the simulation's
own floor reads as the fastest possible attacker — so a party member and a scenario human holding the
same weapon row would have come out with different cadences. That is T6, and M6 is its witness. It
also named the four existing tests that encode the behaviour this story reverses, three of them in a
file the task fence had half forbidden.

## The handed hypothesis, and where it was wrong

It said the tree spawns a placed actor without applying its combat columns, and named a `Man_Club`
row at Body 19. The mechanism is right and the row is not on this map: `Man_Club` is index **10 of
the Humans collection**, a row index rather than a mission number, and mission 1 is `10.alm`. Nineteen
of its thirty-five placements — every creature — already carried their whole template before this
story; sixteen carried nothing. The defect was the humans band alone, and it could not be fixed by
reading columns, because that collection ships **no damage column at all**.

## What was cut, and where each is recorded

Armour and shields, the creature collection's own equipment cell, ranged weapons, and the auto-hit
mark's absorption skip — each named in `spec.md`'s out-of-scope list with its reason. The one
question this story would put to research is in `provenance.md`: whether the recompute runs at spawn
on this arm, which if settled would move health off its column and onto a Body-and-experience derive
needing a class bit no scenario person has.

The build folder is `builds/0087-enemy-combat-block/`, with the invocation that reproduces the two
reports above.

## Every criterion, and what witnessed it

Named individually because a summary cannot be checked. The unit witnesses are all in
`go test -count=1 -trimpath ./...`, exit 0.

| Criterion | Witness |
|---|---|
| FR-1, AC-1, AC-2, AC-3, P-1, P-4 | `pkg/data/humandef_test.go` — the per-slot table walks all twenty-three, one written cell at a time; the all-empty row; the short-row refusal; the sentinel scan. M1, M2, M3 |
| FR-2, AC-5, AC-6, P-2, P-3 | `TestABarePersonsEightAreTheDerivationAndHisOwnCadence`, `TestAnArmedPersonDiffersByExactlyHisWeapon`, `TestAPersonsEightAreAFunctionOfTheRowAlone`, `TestAUnitsEightAreItsOwnColumns`. M11 |
| FR-3, AC-7 | `TestThePersonsWeaponIsTheFirstCellThatResolves`, seven equipment shapes. M7 |
| FR-3, AC-8 | `TestATableMissingAnyItemCollectionArmsNobody`, each of the three withheld alone. M8 |
| FR-4, AC-9 | `TestAnEmptyWeaponCadenceCellLeavesTheTemplatesStanding`, `TestAWeaponsEmptyCadenceCellLeavesTheBarePairStanding`, and the four cadence cases in `pkg/data`. M4, M5, M6 |
| FR-5, AC-10 | `TestAPlacedPersonCarriesHisRowsHealthRateAndCadence`. M9 |
| FR-6, AC-11 | `TestTheDifficultyDoesNotReachAPerson`; and the repaired health test, where id 0 reads 30 at all three settings |
| FR-7, AC-4, AC-12 | `TestAnUnresolvedPlacementCarriesTheConstructorsEight` re-aimed; `TestOnlyAMatchedUnitsEntryCanYieldANonGroundMover`, two rungs one entry one health; the pinned pre-story digest and form length, untouched and green. M10 |
| AC-12a | `TestArmingAPersonMovesTheDigest` — a difference, then an equality once the weapon's own four are put back |
| FR-8, AC-13 | the empty `git diff` over `pkg/sim` above, plus `TestAWorldHoldingAnArmedPersonRoundTrips` |
| AC-14 | the two `classdump` reports above, EN and RU. Its drive half was **not run** — see the limitation stated there |
| P-5 | the gate table above, every exit code 0 |
| FR-3, FR-7 in the tool (T7) | the report's band split, witnessed on EN: `M10_Brigands` reads `person`, `Ghost` and `Bee` read `creature`; the check that pinned the blanket sentence now asserts the split |
| DD-1, R-2 | T1: the interface widened, four compile errors, three of them the test fakes. M12 |
| DD-2, DD-3 | T2: the separate type, the shared cursor, the hand-written defaults expectation |
| DD-4 | T3: `HumanDef.Combat` defined in terms of `Hero.Derive` |
| DD-5, DD-6, DD-7 | T4: the three table fields, `spawnBlock`/`blockFor`, `firstWeapon` |
| DD-8 | T5: `TestLoadTableReadsTheTwoSearchedCollections` asserts all three present |
| DD-9 | the empty `pkg/sim` diff |
| DD-10 | T6, and M6 |
| R-1 | the pre-story digest pin, green and unedited |
| R-3 | the before/after report: health 100 to 15 on both roots, deliberately |
| R-4 | the search runs once per world; the mission-1 report builds in well under a second |
| R-5 | the four repaired tests, each keeping its own question |
| R-6 | recorded, not reachable: the one producer passes named in-range collection ids |
| SC-1 | the AFTER report — the row's health, rate, cadence and a derived band |
| SC-2 | the weapon reached the placement by resolving; a table missing any collection arms nobody |
| SC-3 | the creature and unresolved arms unmoved at all three difficulties; the table-less pin green |
| SC-4 | `git diff --name-only` names no file under `pkg/sim`; AC-12a derives the digest that did move |
| SC-5 | both lawful roots place two brigands at their own entry's health and a band whose top is 3 |
