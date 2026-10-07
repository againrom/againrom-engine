# 0154 — two spells complete: verification

Branch `0154-spells`, off `316bbf0`, with master merged twice mid-story
(`a35ee03` for `0155-headless`, `789edbc` for the research pin at that
boundary). Research pin `6b8970d`; `git submodule status` shows no leading
character.

There is no `tasks.md`: one lane implemented its own slice, so every `FR` and
`DD` is accounted for here.

## Gate

Run from the worktree on a clean tree, at `0154-spells` head:

| Command | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `gofmt -l $(git ls-files '*.go')` | prints nothing |
| `go test -trimpath -count=1 ./...` | all packages pass |
| `bash scripts/check-no-game-assets.sh` | `clean (tree scan)` |
| `bash scripts/check-doc-budget.sh` | exit 0; `spec.md` 14294/24576, `plan.md` 8603, `provenance.md` 4942 |
| `bash scripts/check-hotfix-ledger.sh` | `ok` (26 commits examined) |
| `bash scripts/check-sdd-audit.sh` | FAIL set empty for `0154-spells` |

`git log --format='%h %(trailers:key=Co-Authored-By)' master..HEAD` prints an
empty trailer for every commit. No commit carries an `SDD-Task:` trailer,
because no `tasks.md` was written.

One gate result moved during the story and is recorded as a correction rather
than hidden: `check-hotfix-ledger.sh` failed on the middle commit, whose subject
`game+ui(0154): …` the script's own filter does not read as naming a story — its
pattern admits `name(NNNN)` and not `name+name(NNNN)`. The commit was reworded to
`ui(0154): …` and the check passes. The script was not changed.

## The script-gap census

Measured with `cmd/missionrun` against the `en` install, before and after, and
compared with `pipeline/milestone-baseline.txt`:

```
go build -o /tmp/mr ./cmd/missionrun
AGAINROM_ASSETS=<install> /tmp/mr -mission N -trace -ticks 1 | grep -c UNSUPPORTED
```

| Mission | master (`316bbf0`) | this branch |
|---|---|---|
| 10 | 17 | 17 |
| 20 | 12 | 12 |

**Unchanged, and that is the expected result.** This story adds no script arm; it
is a rule, its state and two surfaces. The claim is that these two numbers did
not move, and both were measured rather than assumed. The story's own result is
the second kind the pipeline recognises: something visible in `builds/current/`,
described in `builds/0154-spells/README.md`.

## The drive

The slice is driven through the **shipped binary** with no install present:

```
go build -o /tmp/ar ./cmd/againrom
/tmp/ar --headless scenarios/0154-synthetic-spells.json
```

Exit 0. What the trace reports, step by step:

- tick 0 — `p0` at 60/60 with 50 mana and no autocast; `p1` wounded at 20/60;
  `u57` at 40/40 on the hostile slot;
- step 5, a `cast` of spell 6 at `p1` — `p1` rises to 34/60 and `p0`'s mana falls
  from 50 to exactly 40, once;
- step 9, a `cast` of spell 1 at `u57` — `u57` falls to 36/40 and `p0`'s mana
  falls to exactly 37;
- step 13, an `autocast` of spell 1, and **no further order at all**;
- step 15, `u57` is dead after **69 more ticks**, at −3/40, felled by unbidden
  casts alone; `p0` is alive at 60/60 and still holds autocast 1;
- step 19, an `autocast` of spell 0 — the setting reads back as 0.

The asset-root forms were exercised on the SAME binary, through the headless
route, which resolves the root the way the windowed launch does: 0155's own
`scenarios/0155-mission10-escort.json` run once with `AGAINROM_ASSETS` set and
once with `-assets`, each exit 0 and each reporting the same 36 units at the same
cells. The windowed launch itself was not run.

`reached_unsupported_at_most: 0` holds throughout, so nothing on this drive
walked into an unimplemented script arm. The same file is run under `go test` by
`TestTheShippedSpellScenarioRunsEndToEndWithNoInstallPresent`, so it cannot rot
while the grammar under it moves.

## Not seen on a screen

The game window was not brought forward on the owner's desktop and no synthetic
keystroke was sent. The autocast key, the dashed border, the popup and the ring
were driven through the production input and composition paths in `pkg/ui`'s own
tests, and the drive above reaches the rule rather than the picture. Nothing here
is a claim that the change was watched running. `builds/0154-spells/README.md`
says the same to the owner.

## Requirements

| id | where it landed | what witnesses it |
|---|---|---|
| FR-1 | `pkg/data/spell.go`, `pkg/mapload/spell.go`, `pkg/sim/spell.go` | `TestLoadSpellsRestorativeIsHealAloneAndNeverAlsoDamaging`, `TestAHealRowWithNoDamagePairIsNeitherFlag` |
| FR-2 | `castSpell` steps 4, 5, 5a | `TestAHealAcrossAHostileRelationOrAtACorpseLeavesTheWorldByteIdentical`, `TestAHealAtOneselfIsAppliedAndADamageCastAtOneselfIsNot` |
| FR-3 | `applySpellHealing` | `TestAHealSpendsManaOnceAndRaisesHealthInsideTheBand`, `TestAHealNeverRaisesHealthAboveTheMaximum`, `TestAFullyHealthyTargetTakesTheWholeCast` |
| FR-4 | `Entity.SpellFX`/`SpellFXSpell`, `decaySpellEffects`, `binary.go` | `TestAnAppliedCastMarksBothActorsAndTheMarkExpiresToNothing` |
| FR-5 | `markSpellEffect`, called at `castSpell`'s tail | same, plus `TestARefusedCastLeavesNoMark` |
| FR-6 | `MapEntity.SpellFX*`, `mapWorld.spellSchool`, `spellEffectPasses` | `TestOnlyAMarkedUnitContributesASpellEffectPass`, `TestASchoolOutsideThePaletteStillDrawsAMark`, `TestTheMarkingSpellsSchoolIsResolvedOffTheWorldsOwnTable` |
| FR-7 | `Entity.AutoSpell`/`AutoCastWait`, `binary.go` | `TestTheAutocastAndTheMarkRoundTripAndReachTheDigest` |
| FR-8 | `stepAutoCasts`, `autoCast`, `stepWorld` | `TestADamagingAutocastFiresWithNoCommandAndThenWaits`, `TestARefusedAutocastDoesNotResetTheWait` |
| FR-9 | `autoCastTarget` | `TestARestorativeAutocastPicksTheMostHurtAllyAndThenTheLowerID`, `TestADamagingAutocastWithItsEnemyOutOfRangeSpendsNothing` |
| FR-10 | `KindAutocast`, `mapWorld.setAutocast`, `Viewer.toggleAutocast`, `appInput.Autocast` | `TestCtrlAWithAUnitAndASpellSelectedSetsClearsAndReplacesTheAutocast`, `TestTheAutocastSeamAppendsOneCommandToTheQueueTheOrdersUse`, `TestTheAutocastCommandStoresTheIDAndZeroClearsIt` |
| FR-11 | `drawAutocastBorder`, `composeSpellBar` | `TestOnlyAnAutocastingCellCarriesTheDashedBorderAndItsDashesTravel`, `TestTheDashedBorderIsDashedAndNotSolid` |
| FR-12 | `spellInfoLines`, `spellPopupPresent` | `TestTheSpellPopupStatesTheHoveredSpellAndNothingElse`, `TestASpellTheTableCouldNotNameStillStatesItsNumbers` |
| FR-13 | `pkg/game/spell.go`'s `spellbookOf` | `TestTheBookCarriesTheAutocastFlagAndTheSpellsOwnLines` |
| FR-14 | `pkg/game/scenario.go`, `headless.go`, `scenarios/0154-synthetic-spells.json` | `TestTheShippedSpellScenarioRunsEndToEndWithNoInstallPresent`, `TestTheSpellVocabularyRefusesTheFormsItCannotMean`, `TestAUnitAuthoredHurtStaysHurt` |

## Acceptance criteria

AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-7, AC-8, AC-9, AC-10, AC-11, AC-12 and
AC-12a are each witnessed by the tests named against their own FR row above, in
the same order the spec states them: the loader's two flags (AC-1); the heal's
band, its ceiling and its two refusals (AC-2, AC-3); self as a target (AC-4); the
mark's birth, decay and absence (AC-5); the round trip, four distinct digests and
the version-44 refusal (AC-6, `TestAVersion44FormIsRefused`); the autocast's
fire-and-wait and its out-of-range silence (AC-7); its two target rules and the
full-health case (AC-8); its three refusals with the wait unmoved (AC-9); the key
over five states (AC-10); the border on the autocasting cell alone and its travel
(AC-11); the popup and its three refusals (AC-12); and the shipped scenario with
its six refused forms (AC-12a).

## Properties

**P-1** `TestAnAutocastTouchesOnlyItsCasterItsTargetAndTheGenerator` — a
bystander in reach of both, compared field by field.

**P-2** `TestTwoWorldsAutocastingFromTheSameStateStayByteIdentical` — forty ticks,
two worlds, byte-compared each tick, with a guard that says the fixture actually
cast. `internal/archtest`'s source scan still holds `pkg/sim` to stdlib-only with
no float, unchanged by this story.

**P-3** `TestARefusedCastLeavesNoMark` and the control-world comparisons in
`TestAHealAcrossAHostileRelationOrAtACorpseLeavesTheWorldByteIdentical` and
`TestADamagingAutocastWithItsEnemyOutOfRangeSpendsNothing`. The comparison is
against a world stepped the same tick with no command, not against the world
before the step: a tick changes a corpse's decay ladder whatever the command was,
so the weaker form would have passed for the wrong reason.

**P-4** `TestTheArmIsChosenByTheRowsOwnFlagAndNeverByItsID` — Heal and Fire Arrow
are given each other's ids and each still takes the arm its own flag names. A
source scan would have proved no literal appears; this proves the thing the
property is for.

**P-5** `TestTheBookCarriesTheAutocastFlagAndTheSpellsOwnLines` composes the lines
on the `pkg/game` side and compares them there; `pkg/ui`'s own popup test drives
`SpellEntry.Info` it was handed. `internal/archtest`'s import graph is what makes
`pkg/ui` unable to name a spell row at all.

## Design decisions

**DD-1** Taken: the autocast is simulation state. `formatVersion` 44 → 45 and the
entity record 217 → 222 bytes. **Allocation 45 was used.**

**DD-2** Authored, and disclosed in the reader's units in
`builds/0154-spells/README.md` and in `provenance.md`.

**DD-3** Authored. `Ctrl+A`, the acting-on-the-selected-spell shape, FR-9's target
rules and the sixteen-tick wait are all ours; the divergence is disclosed in the
build README.

**DD-4** Authored. `autocastDashOn`/`Off`/`Step` and `autocastDashColor` name the
numbers; the colour reuses the bar's own selected-cell yellow.

**DD-5** Held: the mark is per-entity. Witnessed by there being no map-object list
anywhere in `pkg/sim` and by `Entity`'s two fields being the whole of it —
`TestTheCanonicalWorldsFieldSetsArePinned` pins that field list.

**DD-6** `TestARefusedAutocastDoesNotResetTheWait`, three refusals, wait 0 in each.

**DD-7** Held: `spellInfoLines` returns `[]string` and `SpellEntry` gained one
slice, not five scalars.

**DD-8** `TestTheAutocastKeyIsNotAlsoAPan` reads `readInput`'s four guarded
letters. It is a source scan because `readInput` calls the engine directly and
cannot be driven with no window; removing a guard fails it.

## Success criteria

**SC-1** Met. The gate table at the top of this file is the result, run on a
clean tree at this branch's head.

**SC-2** Met. `go test -trimpath -count=1 ./...` passes with no game install
present. Every fixture this story adds is authored in test code or in the
`synthetic` scenario; nothing added here reads an archive.

**SC-3** Met. The drive section above is that run, invoked without `-assets` and
without an install, and its trace is quoted step by step.

**SC-4** Met. The census table above carries both sides. The numbers are 17 and
12 before and after, and the fact that they did not move is stated as the claim
rather than passed over.

**SC-5** Met, by reverting rather than by reading. Deleting the
`rule.Restorative && w.hostileTo(...)` clause makes
`TestAHealAcrossAHostileRelationOrAtACorpseLeavesTheWorldByteIdentical` fail on
its hostile subtest; restoring `readInput`'s `ebiten.IsKeyPressed(ebiten.KeyA)`
without the `letters &&` guard makes `TestTheAutocastKeyIsNotAlsoAPan` fail on
that key. Both reverts were applied, observed and undone.

**SC-6** Met, and each cut is stated in `spec.md`'s own "Out of scope". No third
cast arm exists — a row that is neither `Damaging` nor `Restorative` is refused
at `castSpell` step 4, which 0127's own
`TestAKnownSpellNamingNoRowOrANonDamagingOrNonTargetingRowIsANoOp` still covers.
`2*id + 9` appears nowhere, so nothing is put in flight and no projectile is
drawn. No resistance, area shape, `Effects` parse, learning or item cast was
added. `text/spell.txt` is read nowhere in this tree. And nothing sets
`AutoSpell` on a unit the local player does not own: `toggleAutocast` is gated on
`canArmAttack`, witnessed by the fifth subtest of
`TestCtrlAWithAUnitAndASpellSelectedSetsClearsAndReplacesTheAutocast`.

## What a reader should be sceptical of

- **Heal's magnitude is authored**, and this document does not claim otherwise.
  The band it uses is the Heal row's own damage columns; no published claim states
  what the original restores.
- **The ring is not the original's picture.** The original draws no map object for
  this kind of cast at all, and what stands in its place here is ours.
- **`autoCastPeriod` is 16 and `spellFXLife` is 4** because they read well, not
  because anything decoded says so.
- **The two census numbers did not move**, which is a real claim about this branch
  and was measured on both sides rather than assumed.

## The second round

The owner rejected the story from his screen on 2026-08-14. What a rebuilt
`builds/current/` will show: a cast animates its caster and draws a bolt and a
burst, a staff cast too, and a party loaded from a save has a spellbook at all.
No build was made from this branch.

### What the first round shipped, read against his report

Three of the four are the specification carried out faithfully:

| His words | Cause | Where it was written down |
|---|---|---|
| no graphics | a cast drew a four-tick ring on the two actors and nothing else | `spec.md` FR-5, FR-6, SC-2 |
| no fire arrow animation when the mage fights with his staff | the weapon-borne release set no mark and was reported nowhere, so no tier above had anything to draw | `pkg/sim/spell.go`, `releaseWeaponSpell` |
| the staff animation does not correspond to the moments the spell is applied | the drawn attack run restarted only on `AttackCharging`; a spell-carrying weapon loads `AttackCasting` and never `AttackCharging`, so the counter ran free from the tick the caster acquired a victim | `pkg/game/world.go`, `advanceSwings` |
| no autocast at all | a party member restored from a save had `KnownSpells` hardcoded to 0 | `pkg/game/originalparty.go`, `restoredMember` |

### The autocast chain, read hop by hop

The belief handed to this lane was that the mechanism is wired and something
between the key and the screen is broken. The first half is right; the second is
not. Every hop connects:

`Ctrl`+`A` → `readInput` sets `Input.Autocast` from
`inpututil.IsKeyJustPressed(KeyA) && ctrlHeld()` and drops `A` from the pan keys
while Ctrl is held → `app.go` calls `v.toggleAutocast()` → that reads
`v.selectedSpell` and `v.SelectedUnit()`, gates on `canArmAttack` and calls
`v.autocastSink` → the sink is `mw.setAutocast`, installed by `SetAutocastSink`
on both doors that open a map screen → it queues `KindAutocast` → `stepWorld`
stores `AutoSpell` → `spellbookOf` sets `SpellEntry.Autocast` → `composeSpellBar`
draws `drawAutocastBorder`.

The break is upstream. `toggleAutocast` returns at once when
`v.selectedSpell == 0`; a spell is selected only by clicking a spellbook cell;
and `spellbookBar` returns no bar for a viewer whose book is empty. For a party
restored from a save every book was empty, because `restoredMember` set
`KnownSpells: 0`. So he never saw a bar, never selected a spell, and the key was
a no-op — and for the same reason he could never cast from the book.

One further defect was found reading that chain and is fixed here (FR-21):
`setAutocast` also set `mw.commanded[id]`, the permanent "the player has taken
this unit over", which drops that unit's scripted command track.

### The requirements of the second round

| Id | Where | Witnessed by |
|---|---|---|
| FR-15 (AC-13) | `restoredDefinition` returns the row's `KnownSpells` | `TestARestoredCharacterKnowsTheSpellsHisOwnRowNames` |
| FR-16 (AC-14) | `pkg/sim/castevent.go`, `StepObserved` | `TestAnObservedStepReportsTheCasterTheTargetAndTheCellsOfAnAppliedCast`, `TestARefusedCastIsObservedAsNothing`, `TestObservingAStepChangesNoByteOfTheWorld` |
| FR-17 (AC-15) | `releaseWeaponSpell` marks and records | `TestAWeaponBorneReleaseMarks...AndIsObserved` |
| FR-18 (AC-16) | `autoCastOrder`, `knownRestorative`, `inCombat`, `affordsAutoHeal` | five tests, one per clause, plus `TestAHigherPriorityRowWithNoTargetFallsThrough...` |
| FR-19 (AC-17) | `pkg/game/spellbolt.go`, `pkg/ui/spellbolt.go` | four tests in `pkg/game`, five in `pkg/ui` |
| FR-20 (AC-18) | `windUp`, read by the run restart and the voicing | `TestBothWindUpPhasesAreASwing`, `TestACastingWindUpStartsAFreshSwingRun` |
| FR-21 (AC-19) | `setAutocast` appends and marks nothing | `TestArmingAnAutocastQueuesTheCommandWithoutTakingTheUnitOver`, and the amended `TestTheAutocastSeamAppendsOneCommandToTheQueueTheOrdersUse` |
| FR-22 (AC-20) | `startCastRun`, `casting`, `advanceCastRuns`; the swing gate asks `casting` | `TestABookCastPlaysTheCastersOwnAttackRun`, `TestAWeaponBorneCastStartsNoSecondRun` |
| FR-23 (AC-21) | `scaleRun`, `castSwingSpan`; `entityDraws` scales a cast and not a blow | `TestARunIsScaledToTheIntervalItsProjectileCrossesIn`, `TestOnlyACastScalesItsRun` |
| FR-24 (AC-22) | `castPeriod`, `castRunFallbackTicks`; `castSpell` refuses on a standing `CastWait` | `TestTheCadenceFloorHoldsASecondCastOffForCastPeriodTicks`, `TestABookCastRunIsNeverShorterThanTheFloor` |

**DD-7** and **DD-11** cite six `MAGIC-*` claims from the pin; nothing was
re-derived, and what is authored is stated in `spec.md` and repeated in the doc
comments of `pkg/ui/spellbolt.go`, `pkg/game/spellbolt.go` and `pkg/sim/spell.go`.
**DD-8** the restored book comes from the `Humans` row. **DD-9** "in battle" is a
reading over `HasAttackTarget` and `ScanRange` floored at `minimalGuardRange`.
**DD-10** the reserve is a share, one quarter, authored.

### EXP-0163, and the pin

The first round cut the picture on `MAGIC-PIC-027`'s Unknown. `EXP-0163` settled
it mid-round: the even branch every cast takes animates the caster
(`MAGIC-CASTANIM-029`), and the burst picture belongs to the 10 `AreaEffect`
spells, of which Fire Arrow is not one (`MAGIC-BURST-031`). FR-22 is the first,
built; the burst drawn here is therefore ours, and `spec.md` DD-7 says so.

The research pin moves from `a60a981` to `e885699` **inside** the story rather
than at its boundary — a deliberate exception, because those claims are what this
round was told to build against.

`spec.md` FR-23 and FR-24 are the owner's, dated 2026-08-14, and both are
recorded as divergences rather than as fidelity. What makes them divergences is
`MAGIC-CASTTICK-030` itself, which establishes that the original runs the picture
and the effect on two unrelated clocks. Nothing in it is retracted or doubted.

### What did not change

The serialized form is unchanged: no field was added to `Entity` or to `World`,
`AutoCastWait` was renamed `CastWait` at the same offset and width, and
`binary_test.go`'s pins and `nostate_test.go`'s literal field tables are
untouched, which is the direct statement that neither struct grew.

Six sim tests were changed and each change is a behaviour change, not a
weakening. Four heal tests build their worlds through the new `hlAtWar`, which
places a living hostile inside the guard range: without it the caster is out of
battle, knows the heal and now heals unbidden on the same tick as the commanded
cast, so the tests would measure two casts as one. Their assertions are
unchanged.

`TestTheArmIsChosenByTheRowsOwnFlagAndNeverByItsID` steps `castPeriod` ticks
between its two casts, for the cadence floor.
`TestCastingTheSameCommandTwiceCostsTheCasterTwice` is replaced by
`TestTheCadenceFloorHoldsASecondCastOffForCastPeriodTicks`: 0127 FR-5's claim
that one slice carrying a cast twice pays twice no longer holds, and `spec.md`
FR-24 records the supersession. The shipped scenario waits eight ticks between
its two casts for the same reason.

### What is still not delivered

- **The shipped art.** Nothing here loads `projectiles.reg` or the sheets behind
  it: a bolt is a coloured square and a burst a coloured ring, so two spells of
  one school look identical.
- **A book cast's picture trails its damage**, because the simulation resolves a
  commanded cast inside the tick it arrives on. A staff cast does not.
- **A weapon-borne release is not held to the eight-tick floor.** A weapon whose
  charge is faster than eight releases faster than eight and its swing restarts
  rather than completes. No shipped weapon was measured against that bound.
- **0127's remaining cuts** — resistance, area shape, learning, the item cast.
- **This round was not seen on a screen.** The owner ruled on 2026-08-14 that the
  code is self-sufficient and the game is not to be launched to establish that
  something works. The evidence above is the code path read end to end and the
  tests; it is not a photograph and this document does not claim one.
