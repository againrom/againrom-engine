# 0135-skill-moves — verification

Environment: Windows 11, `go 1.26.1` (the toolchain `go.mod` pins), both lawful
installs present at `gameversions/en` and `gameversions/ru`. Research submodule
at pin `1e9e1a7` for the whole of the work, and at `7eed909` after the merge
with master — every claim this story rests on is in both.

## The gate, as it printed

Clean tree, final commit.

```
go build ./...                        (clean)
go vet ./...                          (clean)
gofmt -l $(git ls-files '*.go')       (no output)
go test -count=1 -trimpath ./...      exit 0, 33 packages ok, 0 FAIL
scripts/check-no-game-assets.sh       check-no-game-assets: clean (tree scan)   exit 0
scripts/check-doc-budget.sh           exit 0, 0 rows OVER
scripts/check-hotfix-ledger.sh        row length ok, Owes ok, size ok, 19 commits examined, ok   exit 0
scripts/check-sdd-audit.sh            FAIL set below
```

`check-sdd-audit.sh`'s FAIL set at the last run before this file existed was
exactly one row, and it is this file's absence:

```
FAIL 0135-skill-moves: every task in tasks.md has landed and there is no verification.md
```

Its note/warning **count** is not comparable from a lane — `builds/` is
untracked, so a worktree emits none. Only the FAIL set is stated.

Trailer bijection, read off the log rather than off a report:

```
b616dca  SDD-Task: 0135-skill-moves/T4   Co-Authored-By: (none)
7d97504  SDD-Task: 0135-skill-moves/T3   Co-Authored-By: (none)
ffd6d38  SDD-Task: 0135-skill-moves/T2   Co-Authored-By: (none)
b08f936  SDD-Task: 0135-skill-moves/T1   Co-Authored-By: (none)
```

Three further commits carry no trailer and touch only `docs/`: the artifacts
and two revisions. Deletion set against the branch point `3f8720e`: **empty**.

## The deliverable, on both lawful roots — AC-10, SC-1

`missionrun` now states each party slot's six levels before and after a drive.
The party member is walked toward the two hostiles south-west of his drop; he
engages on the way, kills one, and that blow raises Blade.

```
$ go run ./cmd/missionrun -assets .../gameversions/en -mission 10 -waypoint p0:14:53:2 -tail 400
mission 10  scenario/10.alm  80x80  36 entities
party p0 skills before: [0 10 0 0 0 0]
waypoint 1  p0 -> (14,53) r2 : reached (12,55), Chebyshev 2, after 179 ticks
outcome undecided at tick 579
party p0 skills after: [0 11 0 0 0 0]

$ go run ./cmd/missionrun -assets .../gameversions/ru -mission 10 -waypoint p0:14:53:2 -tail 400
mission 10  scenario/10.alm  80x80  36 entities
party p0 skills before: [0 10 0 0 0 0]
waypoint 1  p0 -> (14,53) r2 : reached (12,55), Chebyshev 2, after 179 ticks
outcome undecided at tick 579
party p0 skills after: [0 11 0 0 0 0]
```

The arithmetic behind that one point of movement, so the number is checked and
not merely observed: the victim's row states `XPvalue` 14 over a maximum health
of 10, so a kill removing all ten pays `14×10/(2×10) + 1 = 8`; the member's
Mind is 15, so `8 × (4×15+30)/120 = 6`; his Blade experience is exactly
`S(10) = 1593` at the mint, `1593 + 6 = 1599`, and `1599 > 1593` strictly, so
the level rises by one. It cannot rise by two: the cap is
`S(11) − S(10) = 260`, and one crossing is checked once rather than in a loop.

**`-tail` is new and defaults to 64**, which is exactly what the tool did
before, so no existing invocation moves. The fight above concludes at tick 260
and the old fixed tail ended the drive at 243 — seventeen ticks short.

### The milestone drive has not moved

Both roots, the pinned argv, unchanged flags:

```
outcome lost at tick 224
census: 4 of 36 unit(s) moved, 1 fell, over 224 tick(s)
```

Identical to the baseline taken on this branch before a line was written.
`pipeline/check-milestone.sh` extracts only lines beginning `outcome ` and
`census: `, so the two new `party p0 skills` lines cannot reach it.

## Acceptance criteria

| AC | Evidence |
|---|---|
| AC-1 | `TestAC1ANonCarrierBlowRaisesExactlyTheCreditedSlotsLevel` (`pkg/sim`) |
| AC-2 | `TestASkillLevelRoundTripsByteIdentically`, `TestTwoWorldsDifferingOnlyInOneSkillLevelHashDifferently`, and `TestThePinIsThePreStoryPinPlusTheSkill`, which peels the 24-byte block back off and lands on the **unchanged** pre-story digest |
| AC-3 | `TestAC3ACarrierBlowChangesNoLevelAndNoExperience` |
| AC-4 | `TestAC4ANonCarrierWithWeaponSkillZeroGainsNothingFromABlow`, `TestAwardSkillCarrierTakesTheNamedSlotAndRefusesSlotZero` |
| AC-5 | `TestAwardSkillRefusesASlotAlreadyAtLevel100` |
| AC-6 | `TestAwardSkillCapsAtOneLevelsWorth` |
| AC-7 | `TestAC7ACastRaisesTheSpellsOwnSchool`, `TestACastAtAHigherSchoolLevelTakesMoreHealth` (the cast itself), `TestAC7DamageRisesWithTheSchoolsLevel` (the arithmetic under it) |
| AC-8 | Byte-identity: every `TestAwardSkillRefuses…` compares both entities before and after. No-randomness: `awardSkill` names the generator nowhere, and the blow feed's own no-draw property is `TestPayingOrRefusingExperienceDrawsNothingEither` (0125). **The sink has no draw test of its own** — that half rests on the absence of a call, not on a comparison run. |
| AC-9 | `TestARaiseEarnsOneRowAndATickWithoutOneEarnsNothing`, plus four `TestSkillRiseRows…` over the row-builder |
| AC-10 | the two drives above |

## Properties

**P-1** `TestP1TheSimTableAndTheDataCurveAgreeAtEveryLevel` compares
`sim.SkillXPFor` against `data.SkillXPFor` at all 101 levels: **they agree at
every one.** `TestP1TheThreeCheckpointsPublishedInPkgSimHold` pins `S(0) = 0`,
`S(10) = 1593`, `S(100) = 13779612` in both packages.

**P-2** `internal/archtest`'s determinism scan over `pkg/sim`'s non-test
sources is part of `go test ./...` and is green. The curve is 101 `int32`
literals; nothing added is a float, and `pkg/sim` took on no import.

**P-3** `TestP3NoSingleAwardRaisesASlotByMoreThanOne`.

## Success criteria

**SC-1** met — the two drives above.
**SC-2** met — the gate block above.
**SC-3** met, and it is the section below.

## Reverting a line to see whether it is witnessed

Each statement was removed or weakened, the tests run, then the file restored.
The failure is the witness; the assertion is not.

| Reverted | What went red |
|---|---|
| FR-5.4's `a.Skill[slot] >= 100` refusal | `TestAwardSkillRefusesASlotAlreadyAtLevel100` |
| FR-7's non-carrier `slot == 0` refusal | `TestAwardSkillNonCarrierCreditsXPSlotAndRefusesAnEmptyHand`, `TestAC4ANonCarrierWithWeaponSkillZeroGainsNothingFromABlow` |
| FR-8's strict test, `<=` weakened to `<` | `TestAwardSkillRaisesOnlyWhenStrictlyAboveTheThreshold` **and** `pkg/game`'s `TestEquippingAMaceMovesTheCreditedSkillFromBladeToBludgen` |
| FR-12's re-derive trigger | `TestRearmRecomputesToHitWhenTheCreditedSkillRisesWithNoEquipThatTick` |
| FR-13's row gathering | **nothing, at first.** See below. |
| FR-13's first-sight baseline guard | `TestARaiseEarnsOneRowAndATickWithoutOneEarnsNothing` |
| FR-10's cast feed, the whole call | `TestAC7ACastRaisesTheSpellsOwnSchool` |
| FR-11's level argument at the cast site, forced to 0 | **nothing, at first.** See below. |

**FR-13 was unwitnessed and this is how it was found.** Deleting the whole of
the announcement's gathering — so that a raise reached the screen never — left
`pkg/game` green: the pure row-builder had four tests and the path from a level
moving in the simulation to a row being earned had none. `pkg/ui`'s `Viewer`
exposes no accessor for what was posted, which is why the executor tested the
pure half and stopped. The repair is that the detector now **returns** what it
posted, and one test drives a real `mapWorld` over a real world across a real
raise and reads it; only the single `PostPickup` call is now uncovered. It was
folded into T4's own commit, which is what the one-task-one-commit rule asks.

**FR-11 was unwitnessed at its call site, and the shape is the same.** Forcing
the level `castSpell` hands the power to a constant 0 left `pkg/sim` green:
`TestAC7DamageRisesWithTheSchoolsLevel` calls `spellPower` and `spellDamage`
directly and never reaches `castSpell`, so it witnessed the arithmetic and not
the argument. `TestACastAtAHigherSchoolLevelTakesMoreHealth` now casts twice
through `Step` into two worlds differing only in the caster's own level in the
spell's school, with the two damage columns equal so the roll cannot enter,
and requires the higher level to take strictly more health.

**Both holes are one habit:** a test written against the pure function a
statement calls does not witness the statement. Four of the reverts that did go
red were caught by tests of the same shape — the difference is that those four
*reverted the pure function's own body*.

## The merge with master, and what it moved

Master had advanced by `0131`, `0132`, `0134`, `0136`, `0137` and two research
pin bumps; the merge is in this branch and the pin reads **`7eed909`**, no
leading character. Seven conflicted hunks, five in `pkg/game/world.go` and two
in `cmd/missionrun/main.go`. Three were mechanical unions — two `drive`
signatures and one `mapWorld` field block, each side having added its own.
**Two were design interactions:**

- **`0136` moved this story's own door.** The resolve, the fold, the recompute
  and the `SetCombat` left `mw.rearm` for an exported `Rearm` in a new
  `pkg/game/rearm.go`. `Rearm` takes its hero **by value**, so `mw.rearm` keeps
  the level trigger and seeds a local copy with the live levels before handing
  it over — `Rearm` needs nothing from this story. A raise now re-folds the
  **whole worn set** through `Rearm`'s `FoldWear` rather than the weapon alone,
  which is `0136`'s widening arriving here free.
- **`0137` added a band guard on the exact line FR-14 changed.** A creature's
  `Skills` row holds columns rather than levels, so `0137` withholds that one
  write for the creature band. The two compose without either giving way:
  `0137` decides **whether** the line runs, this story decides **what** it
  writes when it does — the entity's stored level rather than
  `data.SkillLevelFor` of its experience.

Two tests came over red and were corrected in place, keeping their names:
`wear_test.go` called `drive` at the pre-`-tail` arity, and
`TestAStepDoesNotMoveACreaturesSkillPositions` asserted
`person.Skills[slot] == SkillLevelFor(person.Experience)` — the retired model.
It now compares against the entity's own stored level, read back from the world
rather than pasted, fails if nothing moved at all so it cannot pass vacuously,
and keeps untouched the creature half that measures `0137`'s guard.

The whole gate was re-run on the **merged** result, not on the branch:

```
go build ./...                    ok
go vet ./...                      rc 0
gofmt -l $(git ls-files '*.go')   empty output   (read, not exited on)
go test -count=1 -trimpath ./...  rc 0, 34 packages ok
check-no-game-assets.sh           clean (tree scan)
check-doc-budget.sh               rc 0, 0 rows OVER
check-hotfix-ledger.sh            ok
check-sdd-audit.sh                rc 0, FAIL rows = 0
```

`gofmt` was **dirty on both resolved files** until `gofmt -w`, and `go vet`
reported a real arity error through a pipe whose exit code read 0. Same trap
twice; both caught by reading output rather than exit codes.

The three witnesses touching the moved code were re-run against the merged tree
rather than carried over — the FR-12 trigger, the live-level seeding into
`Rearm`, and the FR-13 gathering each still reddens the test it reddened
before.

On the merged result the demonstration prints `[0 10 0 0 0 0]` →
`[0 11 0 0 0 0]` on both roots, and **the milestone drive is unmoved on both**:
`outcome lost at tick 224`, `census: 4 of 36 unit(s) moved, 1 fell, over 224
tick(s)`.

## What is not carried, and why

**FR-12 reaches ten numbers and no more.** A raise re-runs the whole derive,
and the damage pair, to-hit, defence, absorption, the two attack times,
always-hits, reach and the trained slot all move with it. **Health, mana,
sight, speed and the five protections do not** — the same derive computes them
and there is no writer for them onto a unit already in a world. They move at
the next mission boundary, where the carry rebuilds the member from his levels.
And only the character whose pack is open is refreshed; another member waits
for the same boundary. Both limits are this tree's, and widening either is its
own story.

**The kill feed is not built.** Its amount and its slot rule are decoded, but
no claim names its call site, so there is no trigger to build without authoring
one. The blow feed and the cast feed are both built.

**The killing blow's exemption is a disclosed divergence.** The original
applies the two source refusals only while the source is still alive, and a
killing blow arrives with its victim already below zero — so there, killing an
ally or a treaty-protected unit pays. Here it does not. Recorded in the spec's
Out of scope with the reason.

**The more-than-one-participant fifth of the cap** has no session object to
read in this tree, and the flag's meaning is graded Medium rather than High.

## Divergences and choices that are ours, not the game's

- FR-6's cap reads the **credited** slot's level. The claim orders the cap
  before the slot resolution, which would read the *named* slot; it also states
  the cap's purpose as "no single event can carry a slot up two levels", which
  only holds on the credited slot. The second reading has an argument behind
  it, the first only an ordering.
- FR-10 passes the **victim** as the cast's source. Nothing states what the
  original's cast feed passes.
- `S(n)` exists twice, once per side of the determinism wall. P-1 is what keeps
  the two from parting.
- The announcement's text, dwell and geometry are authored; the original draws
  no such row.

## Revisions, and two rules bent

**A revision at `spec.md`.** FR-5.2 originally gated the two source refusals on
the source being alive — which *is* the killing-blow exemption the same
document's Out of scope said was not built, since an award is made after the
victim's health has already fallen. Found by reading the two sections against
each other before T3 was briefed; the contract was revised and the divergence
recorded. Commit `7cb052c`.

**A revision at `tasks.md`.** T4 needed two files its entry did not name: a
`pkg/game` test that T3's own fence forbade it from repairing, and the drive
tool's tail. An unplanned file starts a revision at the earliest affected
stage, which for a task boundary is `tasks.md`. Commit `51fdffc`.

**Rule bent, first: T3's commit was not green everywhere.** Its brief fenced it
out of `pkg/game` to protect two parallel lanes, and the raise it landed made a
`pkg/game` test false in the same instant. One implementation task should be
one *test-passing* commit; this one was not, and the branch was red between
`7d97504` and `b616dca`. The fence should have carved out the one test file it
was going to break — a fence written to protect other lanes' files silently
also forbade repairing the damage the commit itself did.

**Rule bent, second: T4's commit was amended.** The FR-13 witness repair above
belongs to T4 and to no other task, and the branch was unpushed, so amending
kept one task to one commit rather than adding a second commit for the same
task. The FR-11 repair belongs to T3, which is no longer the branch tip, so it
landed instead as an **untrailered** commit — the shape this repo already uses
for a layered-audit correction, and it adds no second trailer for one task.

**A design decision deliberately carries no `DD-` number.** The reconciliation
of `0125` and `0127` is settled in `plan.md` under its own heading and not as
`DD-9`, because a `DD-` must be named by a task, a task is a commit, and a
trailered commit must touch a file outside `docs/` — a document-only
reconciliation cannot be one of those without lying about what it changed.

**And a trap re-measured.** Writing this story's `FR-11` into `0127`'s own
`spec.md` minted a phantom `FR-11` for `0127`: `check-sdd-audit.sh` extracts
ids per folder, and it produced two FAILs — `plan.md accounts for no: FR11`,
`no task carries: FR11`. The cross-reference now names the story and not the
id. This is `0130`'s trap in a new place, and the lesson is narrower than
"cite carefully": **never spell another story's `FR-n` inside that story's own
files.**
