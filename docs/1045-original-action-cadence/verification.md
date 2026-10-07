# Story `1045` — verification

This record accounts for every behaviour and design decision in `spec.md`. Commands ran from the
story worktree. The production result is the two lawful-install drives below. No game or GUI process
was launched.

## Contract-to-evidence map

| Requirement | Evidence | Verdict |
|---|---|---|
| B1, explicit Humanoid input | `TestTheHumansArmCarriesTheConstructorsPeriodsAndTheGraphsOwnManaMaximum`, `TestAStartedPartyMemberCarriesItsHerosNumbers`, `TestADecodedNonPrimaryCompanionSurvivesTheWholeConstructionBoundary` and the raised-actor regression population cover fresh-map, party, original-save and raised producers. Form 62 and the hash witnesses cover its canonical consumers. | PASS |
| B2, Spell Complication input | `TestASpellTableReachesTheWorldFieldForField` varies Complication independently through the data and mapload seams. The form, hash, interval and release witnesses consume it in simulation without reopening an install. | PASS |
| B3, weapon identity and weight | The physical matrix varies equipped state and declared runtime weight independently. The EN/RU mission-90 drive obtains both through the production loader and item table. | PASS |
| B4, physical interval | `TestCadence1045PhysicalDistanceAndSchedulerTerms` and `TestCadence1045PhysicalIntervalCarriesEveryTerm` measure charge, ranged extra, relax, every jitter result and both scheduler boundaries. | PASS |
| B5, Humanoid penalty | `TestCadence1045HumanoidPenaltyUsesTheRuntimeWeaponWeight` covers the Humanoid and equipped gates, signed division, both clamp ends, Reaction and runtime weight. | PASS |
| B6, weapon Spell routes | `TestCadence1045MageWeaponDiversionOmitsFloorRangeAndComplication` and `TestCadence1045FighterWeaponSpellIsARiderOnThePhysicalCycle` separate diversion from the physical rider and exclude a second recovery. | PASS |
| B7, one retained book record | `TestRepeatedExplicitCastsPreserveTheCompleteRetainedInterval` drives unit and cell commands on every tick and proves the retained target survives recovery, both boundaries and re-arm. Existing duplicate-caster and ordering refusal tests remain green. | PASS |
| B8, ordinary-book interval | `TestCadence1045RetainedBookIntervalIncludesComplicationAndBoundaries` and `TestRepeatedExplicitCastsPreserveTheCompleteRetainedInterval` measure the eight-tick floor, relax, all four jitter values, Humanoid penalty, Complication and both boundary ticks. The owner-authored floor is isolated to this route and recorded in the divergence ledger. | PASS |
| B9, insufficient-mana retry | `TestCadence1045InsufficientManaRetainsRetryState` covers both retained completion states. `TestCadence1045InsufficientManaSeparatesOneShotAndRetainedProducers` covers explicit retention, offensive and restorative one-shot reselection, idle-Heal affordability and defensive disposal of one-shot pending residue. `TestCadence1045ManaLostDuringChargeSeparatesRetainedAndOneShot` covers release-time mana loss for both record classes. | PASS |
| B10, existing producers | The player command, offensive/restorative autocast, idle-Heal, script and weapon-Spell regression populations pass through their production dispatch seams. One-shot refusal returns to selection; retained refusal keeps its order. | PASS |
| B11, death cancellation | The original four cadence death tests cover charge and both recovery forms. `TestCadence1045EquipmentHealthLossNormalizesEveryProducer` covers replacement, unequip and worn-item drop. `TestCadence1045SetDerivedNormalizesHealthDeathAtomically` covers derived health. `TestCadence1045DeathAfterBookAdvanceRemovesTheRecordBeforeSave` covers command and later-ID combat after the book sweep. `TestCadence1045SelfKillingApplicationsDoNotRestoreRecovery` and `TestCadence1045BookApplyWritersRefuseDeadCasters` cover retained unit/cell book, one-shot autocast, mage diversion and fighter rider. `TestCadence1045AreaReleaseRebindsTheBookSweepAfterEveryRemovalShape` covers lower, releasing and higher records, multiple removals, surviving and dead releasing casters, retained and one-shot release, one transition per survivor, the exact RNG position and canonical form-62/hash/save-load state. | PASS |
| B12, physical revalidation | `TestCadence1045PhysicalApplicationRechecksReach`, `TestCadence1045PhysicalApplicationCancelsWhenTheTargetIsRemoved`, `TestCadence1045TargetHealthAloneDoesNotCancelTheSecondStrike` and `TestCadence1045NegativeLinkedTargetSurvivesUntilTeardown` cover reach, presence, same-tick ordering, negative-health physical/fighter-rider application and next-tick teardown. | PASS |
| B13, book refusal bounds | Existing target-form, applicability, visibility, range, terrain and busy refusal tests remain green. External progress replacement and the general Spell target-loss rule remain the two research Unknowns, not authored behaviour. | PASS within stated bounds |
| B14, form 62 validation | `TestCadenceForm62RoundTripsEveryLifecycleBoundary` and `TestCadenceForm62RefusesEveryMalformedLifecycleClass` cover Humanoid, Complication, phase, countdown, progress, completion, retention, non-retained pending, unit/cell target consistency and the full Complication byte range. `TestCadenceForm62NormalizesAndRefusesDeadCastRecoveryResidue` covers dead `CastWait`; `TestCadenceForm62RefusesBookStateWithoutALiveCaster` covers absent and dead book casters. | PASS |
| B15, hash and resume | `TestCadenceInputsAndLifecycleEachChangeTheCanonicalHash` separates each later-affecting input. `TestCadenceForm62NormalizesAndRefusesDeadCastRecoveryResidue` proves constructor normalization is form- and hash-identical to zero. `TestCadence1045DeathAfterBookAdvanceRemovesTheRecordBeforeSave` proves command and combat death remove the record before the immediate form and hash. `TestCadenceForm62ResumeHasTheSameNextTicks` compares events and bytes for 24 ticks across lawful boundaries. | PASS |
| B16, legacy migration | `TestForm61MigrationDisclosesAndDoesNotGuessCadence` checks form 61 directly. The all-readable-version upgrade population rebuilds every earlier form through the ordered table, preserves the form-61 item section and requires one action-cadence loss sentence. | PASS |
| DD1, separate state shapes | Physical and book tests observe their own records while sharing transition ordering; neither encoder assigns irrelevant union fields. | PASS |
| DD2, one lifecycle story | The retry witness consumes completion written by successful release, and form 62 persists both sides in one record. | PASS |
| DD3, isolated book floor | Mage diversion and ordinary-book tests use unequal decoded and floored charge values, so a leaked floor fails one side. | PASS |
| DD4, preserve before default | The form-61 migration witness compares the carried item section byte-for-byte before checking the three cadence defaults. | PASS |
| DD5, headless shipped seam | `TestReleaseActionCadenceUsesShippedHumanoidWeaponAndSpellInputs` starts production missions, loads typed archives and tables, then isolates only the actors whose cadence it drives. | PASS |

## Lawful-install discriminator

On each EN and RU root, the release witness starts mission 90 and finds UnitID 42 classified as
Humanoid with an equipped weapon whose code has a declared runtime weight. It issues one physical
attack command and measures three applications. Each gap lies inside

`charge + relax + humanoidPenalty + U[0,3] + 2`.

The same drive starts mission 10 with the production generated mage, chooses an admissible
positive-cost unit Spell from the installed normalized table and issues one cast command. The first
release exhausts mana. With no further command, insufficient-mana retries retain the same Spell and
target until ordinary regeneration supplies its cost; that later release is observed on both roots.

## Repository and release gates

- `go build ./...`, `go vet ./...`, gofmt and
  `go test -p 1 -trimpath -count=1 ./...`: pass on the frozen candidate. The package suite runs
  serially to stay within the host's memory limit.
- `scripts/check-claim-citations.sh`: 1,356 distinct citations resolve against 1,580 claims and 233
  experiments under 829 prefixes.
- `scripts/check-no-game-assets.sh`: clean.
- Research at exact pin `23daf74f`: its prescribed build and complete checker glob pass;
  `check-claim-ids` reads 1,580 ids across 31 ledgers and reads 30 back through the claim tool,
  `check-regen-out` checks 24 scripts, and `check-retraction-status` reads 251 retraction entries.
- `pipeline/check-pin-forward.sh`: the story and current implementation master both pin exact
  research `23daf74f`.
- `pipeline/check-div-claims.sh`: 249 live rows have nine cells and cite 320 distinct claim ids; the
  census includes accepted `DIV-425` and the merged `DIV-429` debt.
- `pipeline/check-scenarios.sh`: 15 of 15 pass on EN and 15 of 15 on RU.
- `pipeline/check-release-tests.sh`: 63 of 63 pass, zero skipped, on each root. The population is 60
  asset-only tests, one original-save test and two save-666 tests.
- `pipeline/check-preserved-installs.sh`: expected external RED from the owner's known EN save-set
  drift. EN `game0000.sav`, `game0001.sav` and `game9999.sav` differ in size from the recorded
  standard, and `game0003.sav` through `game0021.sav` are additional. No install was mutated or
  re-recorded by this story.

## Milestone result

`pipeline/check-milestone.sh`, pointed at `missionrun.exe` built from this worktree, preserves every
script-support count across all 28 maps and both roots. The expected simulation result changes the
unattended mission-10 defeat from tick 240 to tick 256 on both roots; the census remains 4 of 36
units moved and 1 fallen. The shared parent-tree baseline therefore needs its seat-owned update at
landing. This lane does not edit orchestration state from the implementation worktree.

## Remaining surface

The remaining-surface list is empty across every canonical input; physical, ordinary-book,
mage-diverted and fighter-rider formulas; repeated explicit input; retained and one-shot retry states
at admission and release; actor death through command, later-ID combat, equipment and derived-health
producers, including self-kill during every application route; target presence, linkage, reach and
non-positive health through approach and teardown; lower, releasing and higher book-record removal
during one area application, including multiple removals, a surviving or dead releasing caster,
retained continuation and one-shot removal; each form-62 field and malformed
class; every readable migration; hash separation; save/resume; command and AI producers; event
consumers; and both shipped roots. `DIV-425` records the intentional eight-tick floor conflict.
External progress replacement and Spell-specific target loss remain expressly outside the contract
under `HERO-CADENCE-115` and `MAGIC-CADENCE-127`. No in-scope production or administrative surface
remains.
