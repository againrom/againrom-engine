# Verification — a group decides whom to fight, and nobody has to be told

## The gate

Each command run on its own and branched on **its own** exit code, never on a grep or a tail of its
output.

```
go build ./...                                  EXIT=0
go vet ./...                                    EXIT=0
gofmt -l $(git ls-files '*.go')                 EXIT=0   (no output)
go test -count=1 -trimpath ./...                EXIT=0   31 ok, 3 without tests, 0 FAIL
bash scripts/check-no-game-assets.sh            EXIT=0   clean (tree scan)
bash scripts/check-no-game-assets.sh --history  EXIT=0   clean (history scan)
bash scripts/check-doc-budget.sh                EXIT=0   every artifact and both chain relations ok
bash scripts/check-sdd-audit.sh                 EXIT=0   0 FAIL; 0086 appears in no note either
```

`-trimpath` is not optional here: Windows Defender quarantines one test binary without it.

`check-sdd-audit.sh` run from a worktree reports a note and warning **count** that means nothing —
its `builds/` arm is guarded by `[ -d builds ]` and a worktree has no `builds/`, that directory
being untracked rather than ignored. Only the FAIL set is enforced and only it is comparable.

**It exited 1 before this file landed, on four FAILs.** One was this file's own absence. The other
three were real and had been on the branch since `tasks.md` was written:

```
FAIL 0086-ai-engages: every task in tasks.md has landed and there is no verification.md
FAIL 0086-ai-engages: plan.md accounts for no: FR15 FR2 FR9
FAIL 0086-ai-engages: no task carries: FR10 FR11 FR12 ... FR9   (15 ids)
FAIL 0086-ai-engages: no task carries: DD10 DD11 DD12 ... DD9   (10 ids)
```

The audit reads ids, not ranges, and `tasks.md` wrote its coverage as `FR-1 … FR-5` and
`DD-5 … DD-14`, which names two ids and hides the rest. Every range is now written out and
`plan.md`'s traceability table names FR-2, FR-9 and FR-15, which it had genuinely omitted. Worth
keeping as a shape: an ellipsis reads to a human as *more* coverage and to the gate as less.

## The deletion set

```
git diff --diff-filter=D --name-only 8d34461 HEAD   -> empty (0 files)
```

Trailers: `T1` … `T5`, exactly once each, on the five code commits and nowhere else. The handoff
commit carries none, and neither does this one. No `Co-Authored-By` anywhere. The submodule sat at
`404966d7` with **no leading character** throughout.

## What was found here, and why the story grew a fifth task

The drive below is what found it. Run against a lawful root after T4, `missionrun` came back **byte
for byte identical to the run before the story existed** — and it should not have, because 760
ordered pairs of entities on that map are hostile to each other.

`StartMission` and `StartMissionScripted` each build a world and then build a **second** one out of
its parts. Neither named the relation, so it was dropped one call after the loader authored it, and
`pkg/game` starts every mission through the second of them. T4's store was correct and the world it
reached was thrown away. The failure is silent in the worst way available: the world still builds,
still hashes, still ticks, and its only symptom is that nothing starts a fight — which is exactly
what every world in this tree looked like before the relation existed, so no test written before
this story could see it go. **T5** is the fix and its fence is asked at every shape that reaches the
two rebuilds.

## AC evidence

| | Witness |
|---|---|
| **AC-1** | `TestARelationSurvivesTheFormAndReachesTheDigest` |
| **AC-2** | `TestNoRelationAndAnAllZeroRelationAreOneWorld` |
| **AC-3** | `TestTheRelationDecidesAcquisitionInOneDirection` |
| **AC-4** | `TestAHostileGroupStartsAFightNobodyOrdered` — two units, no command on any tick, one dies |
| **AC-5** | `TestNoDecisionIsTakenOffTheDecisionPhase` |
| **AC-6** | `TestSightBoundsTheCandidatePopulation` |
| **AC-7** | `TestTheNoticeRadiusIsTheArithmeticTheLawNames` — as arithmetic, because D-3 shows no world can present it one |
| **AC-8** | `TestAGroupStandingItsGroundTakesOnlyWhatItCanStrike` |
| **AC-9** | `TestAGroundMemberNeverTakesAFlier` |
| **AC-10** | `TestTheCheapestCandidateWinsAndTheFirstBreaksATie` |
| **AC-11** | `TestAGroupWithOnlyCorpsesInSightAttacksACorpse` |
| **AC-12** | `TestReEngagingTheSameVictimDoesNotRestartTheCycle` |
| **AC-13** | `TestAFriendlyRelationDoesNotEndAnAttackAlreadyIssued` |
| **AC-14** | `TestASlotTheRosterDoesNotNameNeitherAcquiresNorIsAcquired`, `TestASlotTheMatrixDoesNotHoldIsHostileToNothing` |
| **AC-15** | `TestType5Roster/the relation row decodes in order and at full width` and its absent-record case (the file half); `TestTheRosterAuthorsTheWorldsRelation`, `TestNoRosterAuthorsNothing`, `TestARosterSlotOutsideTheMatrixContributesNothing`, `TestARosterWordReachesTheDigest` (the store); `TestAStartCarriesTheMapsRelationThroughEveryRebuild` (the mission's world) |

Two more carry no AC of their own and are the story's point end to end:
`TestALoadedMapFightsWithoutBeingTold` and `TestAStartedMissionFightsWithoutBeingTold` — a map's own
roster row is the whole of what starts the fight, on the plain load and on the path a mission takes.

**Where the relation is read from matters.** Wherever a row's *untouched* half is the point it is
read out of the **byte form**, not through `Relations`: the accessor answers zero for a cell the
matrix does not hold, so it cannot tell "column 0 was never written" from "column 0 cannot be
addressed". The form's 2500 bytes at stride 50 are the store itself and a write anywhere in them
shows.

**The rows in every store test are deliberately not corpus-shaped.** All 3056 words of all 38
shipped maps are `{0, 1, 2}`, so a loader that dropped the narrowing, took the high byte, or moved
the word into the byte some other way would agree with every shipped map. The fixtures carry
`0xffff`, a word whose low byte is 2 under a high byte of 1, and one whose low byte is 0 under a
nonzero word.

## Success criteria

**SC-1** is AC-4's witness. **SC-2** is AC-3 and AC-6 … AC-11, each asserted on both sides of its
boundary. **SC-3** is AC-1 and AC-2. **SC-4** is the drive below. **SC-5** is AC-5's. **SC-6** is
AC-15's, now at all three tiers: the file, the load, and the started mission.

## Properties

**P-1** — `TestTheDecisionIsDeterministic`, plus the structural half: no map, no sort, no draw
anywhere on the decision's path, and `internal/archtest`'s source scan already refuses a clock, a
generator or a float in `pkg/sim`. The drive below adds the measured half.

**P-2** — `TestARelationRefusesEveryLengthButItsOwn`, `TestTheRelationIsDirectionalAndReadsOnlyBitZero`,
`TestARelationSurvivesTheFormAndReachesTheDigest`, `TestAFormTooShortForItsRelationIsRefused`,
`TestTheRelationADecoderIsHandedIsCopiedNotAliased`. No byte is masked on the way in or out, so two
matrices the form distinguishes stay two worlds.

**P-3** and **P-4** — `TestADecisionIsIdempotentOnAWorldNothingElseMoved`: the same assignment twice,
and a member with no candidate untouched in every field.

**P-5** — `TestNoCandidateAGroupCanSeeIsEverClipped`, swept over group geometries and candidate
placements. It is the measured form of D-3 and it is **the test that should fail the day a group
record lands**, which is when the freeze arrives and the clip becomes a rule.

## Mutation testing

Seventeen mutations, each applied, compiled, run against **`./...`** and reverted. **All seventeen
were killed**; none was rejected by the compiler, and a mutation the compiler rejects would not be
evidence.

The whole suite is the target and not the package under change, and one mutation shows why:
**transposing `relationIndex`** — a defect entirely inside `pkg/sim` — was killed by
`TestTheRosterAuthorsTheWorldsRelation`, two packages away in `pkg/mapload`, and by nothing in
`pkg/sim` itself.

- **`pkg/formats/alm` (3).** The row read big-endian; the word narrowed to its low byte in the format
  tier; the words read at a fixed offset from the payload rather than from the record. Each killed by
  `TestType5Roster`.
- **`pkg/mapload` (7).** The relation dropped from the constructor call entirely — the whole-call
  class — killed by four tests; the forced diagonal deleted; the diagonal forced to 0; the row stored
  into 0-based columns; the roster index taken as the slot; the word's high byte stored instead of
  its low; the store transposed.
- **`pkg/sim` (7).** `relationIndex` transposed; hostility read off bit 1; `Hostile` inverted; the
  corpse park deleted; `orderAttack`'s "already holds this victim" test forced true; the decision
  pass deleted from `Step`; the decision moved out of its phase arm so it runs every tick — the last
  killed by `TestNoDecisionIsTakenOffTheDecisionPhase` and by nothing else in `pkg/sim`.

## The drive, on both lawful roots

Mission 10, `scenario/10.alm`, EN and RU, each run twice. **All eight runs of each drive are byte
identical** — across roots and across runs of one root, which is SC-4 as far as this instrument
measures it.

```
$ missionrun -assets ..\gameversions\{en,ru} -mission 10 -ticks 4000 \
    -waypoint u28:79:79:200 -waypoint u29:79:79:200 -attack u28:u29     EXIT=0
mission 10  scenario/10.alm  80x80  36 entities
waypoint 1  u28 -> (79,79) r200 : reached (69,51), Chebyshev 28, after 1 ticks
waypoint 2  u29 -> (79,79) r200 : reached (64,46), Chebyshev 33, after 1 ticks
attack 1  u28 -> u29 : FELLED it after 128 ticks, victim at -1 hp, attacker facing NW
outcome undecided at tick 194
```

**0081's fixture is unmoved**, and that is the honest result rather than a disappointing one: both
waypoints are satisfied where the units already stand, so the drive runs 194 ticks with nothing
walking and nothing coming within a sight radius of a hostile. It is also the run that hid the T5
defect, because it looks the same either way.

A drive that actually crosses the map separates them. Against `8d34461` and against `HEAD`, both
roots:

```
$ missionrun -assets ..\gameversions\{en,ru} -mission 10 -ticks 4000 -waypoint u28:40:40:2
  before   waypoint 1 : reached (42,42), Chebyshev 2, after 540 ticks   outcome undecided at tick 604
  after    waypoint 1 : reached (39,51), Chebyshev 11, after 864 ticks  outcome LOST at tick 864
```

The walk is intercepted and **the mission is lost** — a shipped map now fights back, on a drive that
issues nothing but move orders. Both roots agree byte for byte in both builds.

The map's own roster is five records — `Self`, `Villagers`, `Rogues`, `Beasts`, `Nocturnal` — whose
rows reach the started world as 23 nonzero cells and 760 ordered hostile entity pairs, identical on
EN and RU. That was read with a throwaway probe outside any commit, and it is the measurement that
told the two identical drives apart.

**Per-tick digests were not compared.** No shipped tool emits one and this story did not add one, so
what is claimed is what the instrument shows: identical printed drives across roots and across runs.

The runnable build is `builds/0086-ai-engages/` — `againrom.exe` and `missionrun.exe`, with the
exact invocations and the before/after differential in its `README.md`. It is a snapshot of this
story; `builds/current/` is the one rebuilt from master at every landing.

## What is diverged, and what is missing

Each is named at the site that would otherwise imply otherwise, and each is in `spec.md` in full.

- **D-1, no line of sight.** The candidate population is the Chebyshev disk of the sight radius with
  no occlusion. `AI-LOS-081` reads the law's predicate whole and states in its own Unknown clause
  that neither the step grid at `fog+0x22000` nor the cost grid at `fog+0x28000` was traced to a
  writer, so the region's *shape* cannot be reproduced — only its bound, since the law's region is a
  subset of this one. **A group here sees through walls and acquires strictly more.**
- **D-2, no turn cost.** `R0115` appears in the ledger only at its call sites. It is the low
  byte under a distance term shifted eight bits, so it can only reorder equidistant candidates, and
  that order becomes list order. Written as `turnCost` returning zero, as the seam.
- **D-3, and the finding it carries: the notice clip is inert in this build.** A freshly computed
  radius is at least `max over members (distance + sight)`, and Chebyshev obeys the triangle
  inequality, so every candidate FR-10 admits is already inside the circle. The law's radius is
  *frozen* at the last guard issue (`AI-RADFREEZE-075`), and that freeze is the whole of what makes
  the clip a rule. Implemented and pinned as arithmetic (AC-7) and swept (P-5), so the story that
  adds the freeze changes where the number comes from and nothing else.
- **D-8, and its finding: an engage on a *dead* body ends inside the tick that issued it**, because
  0064's attack cycle already drops an order whose victim is dead. The corpse **selection** is the
  part this story owns and is witnessed directly; the law's "keeps attacking a corpse" holds here
  for a **downed** body and not for a dead one. Lifting the rest is a change to the cycle.
- **D-4 … D-7** are unreachable or unrepresentable here — the radius jitter's latch, the attacker
  memory, invisibility and spells, and the two reach-dependent arms — and are written out so the
  table can be checked against its source rather than only against what fires.

Named as **missing** rather than diverged: the line-of-sight predicate (D-1); the turn cost (D-2);
**which roster slot a human participant holds**, inherited rather than answered and already recorded
at `pkg/sim`'s owner field and at `SetLocalOwner(0)`; and **the post at `ord+0x00`**, since
`AI-GRPGUARD-074` states that under a load-time guard order nothing has ever written it, so a
faithful walk home would send every idle guard toward cell 0. The walk half of the guard arm is out
of scope for that reason and R-2 is its visible cost: a group that acquires and then loses sight
pursues its victim across the map, because there is no break-off under group orders 1/2/3/5 at all.

## What is not claimed

- **Nothing here was looked at.** No screenshot, no window, no statement that it looks right. What
  is established is what the code does. The looking is the owner's.
- **The byte-form version literal is asserted nowhere**, on the rule this tree already applies: the
  version is a fact about the encoding, not a contract this story holds, and a pin taxes every later
  bump. The field-set pin and the round trip are what stand in its place.
- **No fixture reads a game install.** Every test in this story is synthetic; the two lawful roots
  appear only in the drive above, whose whole output is the lines it prints.
- **`pkg/data` and `pkg/mapload/spawn.go` were not touched**, being another lane's ground for the
  duration.
