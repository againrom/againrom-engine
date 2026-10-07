# Verification — 0098

Story head `a195149` on `story/0098-target-not-sticky`, research pin `4920d6d`. Every figure below
was re-run from the orchestrator seat on a **clean tree**, not taken from the executor's report.

## The gate

`go build ./...`, `go vet ./...`, `gofmt -l $(git ls-files '*.go')`, `go test -trimpath -count=1
./...`, `scripts/check-no-game-assets.sh` and `check-doc-budget.sh`: all clean at `a195149`.

One task, one trailered commit, `SDD-Task: 0098-target-not-sticky/T1`, touching exactly
`pkg/sim/engage.go`, `pkg/sim/engage_test.go` and the new `pkg/sim/release_test.go`.

## SC-1 — the no-command census, and it falls

28 campaign missions, each started and stepped **2000 ticks with no commands at all**. The harness
is a separate module outside the repo (plan's verification section), so it adds nothing to
`go build ./...`; the before figures come from the same harness built against `pkg/sim/engage.go`
at `8a35a4d`, which is this branch's tree without T1.

| | before | after |
|---|---|---|
| missions | 28 | 28 |
| entities | 2361 | 2361 |
| moved at all | **79** | **76** |
| cells travelled | **760** | **564** |
| died | **39** | **33** |

**Identical on the EN and the RU root**, before and after. The drift falls by **196 cells, 26 %**,
three fewer entities move at all, and six fewer die. So this *is* the mechanism `0095` went looking
for and did not find in the radius.

Two figures differ from `0095`'s published 79 / 501 / 35 and the difference is in the **instrument**,
not the tree. `moved` agrees exactly at 79, which is what says the two harnesses are measuring the
same thing. `cells` is summed per tick here — a unit that walks out and back travelled the whole
way — where 501 is consistent with an endpoint-to-endpoint sum; and `died` counts entities the decay
ladder has since removed from the world as well as bodies still in it. Both are stated so the before
column is a measurement of this harness rather than a quotation of another.

## SC-2 — the clubman, and it does not move

Mission 10, the party driven to (25,55) and then to (40,50), 400 ticks. Commands are written
directly rather than aimed, so the waypoint instrument's occupancy blindness cannot reach this run.

| id | owner | before | after |
|---|---|---|---|
| 1 | 3 | (32,62) → (33,53), acquired tick 135, never released | **identical** |
| 0 | 3 | (24,54) → (34,54), acquired tick 135 | identical |
| 2 | 2 | (36,51) → (34,52), acquired tick 359 | identical |
| 35 | 1 | (17,66) → (35,54), acquired 199, released 201 | identical |

Identical on both roots. **This story does not change the clubman, and the reason is the contract
working rather than failing**: a guarding member releases only when its victim stops scoring, and
the party stays inside the clubman's frozen notice circle for the whole 400 ticks, so it keeps
scoring and the chase is faithful. Entity 35's release at tick 201 is the attack cycle dropping a
dead victim, not FR-1 — it is at the exempt stance.

The figure disagrees with two earlier readings — `0094`'s (32,62)→(31,53) and `0095`'s
(32,62)→(27,57) — and that disagreement is about the drive, not the tree: what is load-bearing here
is that **before and after are the same on one harness**.

## SC-3 — mission 10's outcome, unchanged

`missionrun -mission 10 -waypoint u21:56:21:3 -waypoint p0:66:16:3`, EN and RU identical:

| | Waypoint 1 | Outcome |
|---|---|---|
| before | `STOPPED BY THE WORLD DECIDING, short of (39,41), Chebyshev 20, after 272 ticks` | `lost at tick 272` |
| after | *the same line* | `lost at tick 272` |

`-trace` names the same arm at the same tick in both: `tick 262 check 12 vip(u21) COUNTED A LOSS:
its unit is not alive (-2 hp)`. **No test expectation, threshold or predicate was edited** —
`TestTheTenthMissionIsDrivenToAWin` is untouched and the story's diff shows it.

Both `0097`'s figures reproduce here exactly, which is the check that the rebase carried its change
intact rather than that this story did nothing to it.

## SC-4 — witnessed by reverting, from this seat

Three reverts run here on the committed tree and restored, beyond the executor's own:

- **The stance gate** (FR-5). `else if st == stanceGuard` → `else if true`:
  `TestAStandGroundMemberKeepsAVictimThatStepsPastReach` fails with *"a stand-ground member lost its
  victim (0/false) once it stepped past reach"*.
- **The byte narrowing** (FR-8, DD-6). `uint8(len(cands))` → `len(cands)`:
  `TestTheCandidateCountNarrowsToAByte/256` fails and **255 and 257 stay green**, which is the shape
  that says the test is measuring the narrowing and not the release.
- **Both release sites neutralised.** `TestTheAlwaysScoringPopulationMatchesTheGoldenDigest` and
  `TestTheByteFormsVersionAndAFixedWorldsDigestAreUnchanged` both **pass**, which is what makes them
  genuine before-values (FR-12, AC-10, AC-11) rather than a self-consistency check.

The executor reports nine further reverts, each naming the test it broke. Those are its claim; the
three above are this seat's.

## SC-5 — the population that keeps fighting, named

It is not empty and it is the majority. **33 of the 39 deaths survive the change** across the
28-mission corpus, so the release removes six unordered kills and leaves the rest; and the clubman
above is a named, measured guarding member that keeps its victim for 400 ticks because that victim
never stops scoring. FR-11 is therefore witnessed on real content and not only in a fixture.

## The two landed tests that changed, and what they still witness

Found by **running** a throwaway probe of the minimal change before Phase 4, not by reading — reading
had predicted one of the two. Both are in `pkg/sim/engage_test.go` and both moved their world to the
exempt stance with **no assertion altered** (FR-15):

- `TestAFriendlyRelationDoesNotEndAnAttackAlreadyIssued` (0086 AC-13). Its claim is now false under
  guard — that reversal *is* this story — and still true under stand ground, where it now stands.
  The guard case is added with the opposite expectation in `release_test.go`.
- `TestReEngagingTheSameVictimDoesNotRestartTheCycle` (0086 AC-12). Only its **control** arm moved;
  the decided arm is still a guarding unit or the comparison would test nothing.

No third test went red at any point, here or in the executor's run.

## What is disclosed rather than fixed

- **A released member stands where it stopped.** The original puts it on a walk home or an idle
  turn; this build has neither the post nor the turn. FR-4.
- **A player's attack command is not privileged.** Under guard it ends at the first decision that
  scores nothing, because nothing distinguishes a commanded victim from a decided one. FR-14 —
  unobservable on shipped content, since every unit a player can command stands at the exempt slot.
- **This build releases sooner than the original does.** It has no remembered-attacker memory
  (`AI-GROUPSEE-068` phase (b): a struck AI-owned member keeps its assailant's *cell* in the sight
  stamp for 20 group ticks). Part of the 196-cell fall is that omission rather than the fix, and no
  measurement here separates the two.
- **`AI-CANDBYTE-110` is Medium and its reachability unmeasured.** FR-8 is carried as a limit.
- **`R0015` is unread**, so whether the original rewrites a human participant's order on the
  clear path is open. FR-5 takes the conservative side.

## The instrument's own blind spot

`missionrun`'s `aim` picks the open cell nearest the mover, *open by terrain alone*. At a small
radius it can pick a cell a unit stands on, and the drive then prints an identical line whether or
not a change worked — which is how `0097`'s fix hid from its own measurement. SC-2 here avoids it by
writing commands directly instead of aiming; SC-3 keeps `0097`'s exact invocation so the two stories
are comparable. Teaching `aim` to read occupancy is a tool change and no story has made it.

## Manual

`builds/0098-target-not-sticky/` holds `againrom.exe` and `missionrun.exe`, both `go build -trimpath`
from the story head, and a `README.md` whose every command was run on both roots before it was
written down. `againrom -check` reports 38 map rows on EN and 34 on RU with the same generated hero;
the `missionrun` invocation reproduces SC-3. Neither binary writes a file.

## Every criterion, and what witnesses it

Each row is a test that fails if the line it names is removed. All in `pkg/sim`, all green at
`a195149`, all re-run from this seat.

| | Witness |
|---|---|
| AC-1 | `TestAReleasedMemberHoldsNoneOfItsSevenFields` — all seven fields checked, not just the victim |
| AC-2 | `TestAReleasedMemberStandsWhereItStoppedForeverAfter` |
| AC-3 | `TestAVictimThatStillScoresSurvivesASecondDecision` |
| AC-4 | `TestAWalkerWithNoVictimIsUntouchedByADecision` |
| AC-5 | `TestAnEmptyCandidateListReleasesOnlyTheVictimHolders` |
| AC-6 | `TestAStandGroundMemberKeepsAVictimThatStepsPastReach` — also SC-4's first revert |
| AC-7 | `TestAVetoedCandidateReleasesAHeldVictim`, both arms: veto present releases, veto absent engages |
| AC-8, AC-9 | `TestTheCandidateCountNarrowsToAByte`, subtests 256 / 255 / 257 — SC-4's second revert |
| AC-10 | `TestTheAlwaysScoringPopulationMatchesTheGoldenDigest` — golden value taken from the tree before T1 |
| AC-11 | `TestTheByteFormsVersionAndAFixedWorldsDigestAreUnchanged` |
| AC-12, AC-14 | `TestAReleasedMemberRoundTripsThroughTheByteForm` — a partial clear is refused by the decoder's own message |
| AC-13 | `TestOrderAttackIsTheOnlySetterAndDecidesOnlyClearerIsTheRelease` — an AST scan, so it fails on a second setter and on `decide` reaching `clearAttack` directly |
| AC-15, AC-16 | `TestACommandedAttackIsNotPrivilegedUnderGuard`, and its exempt-stance half is `TestAFriendlyRelationDoesNotEndAnAttackAlreadyIssued` in `engage_test.go` |
| AC-17 | `internal/archtest`'s own suite, **unmodified** and green: the pin matches `.HasTarget = true` in lockstep and the release assigns `false`, so it cannot fire. No test was added here — duplicating that scan in `pkg/sim` would be the second place DD-13 exists to avoid |

AC-17 is the one row with no new code behind it, and that is deliberate rather than a gap.
