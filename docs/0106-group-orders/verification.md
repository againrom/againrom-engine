# Verification — 0106

Three tasks landed, then two verification-stage commits that carry no trailer. The method here is
mutation, not reading: every clause below was checked by breaking the line that implements it and
watching a named test go red. Where a mutant survived it is said so and what was done about it.

## The gate

Run on a clean tree at `20f3d27`, exit codes captured directly.

```
go build ./...                              EXIT=0
go vet ./...                                EXIT=0
gofmt -l $(git ls-files '*.go')             (no output)
go test -trimpath -count=1 ./...            EXIT=0   31 packages ok
bash scripts/check-no-game-assets.sh        EXIT=0
bash scripts/check-doc-budget.sh            EXIT=0
git diff --diff-filter=D --name-only origin/master..HEAD   (empty)
git log --format='%(trailers:key=Co-Authored-By)' origin/master..HEAD   (all empty)
```

`-trimpath` is not optional locally: Windows Defender quarantines one test binary without it.

## The mutants

Each row is one line reverted in production code, the whole suite re-run, and the file restored.
`KILLED` names one of the tests that went red.

```
M1  FR-1 FR-3 DD-2  constructor's post write deleted        KILLED TestAFreshWorldsPostIsEveryEntitysOwnCell (+30 more)
M2  FR-2             guard arm's anchoring deleted           KILLED TestCmdGroupGuardAnchorsEveryLivingMembersCurrentCell
M3  FR-2             stand-ground arm's anchoring deleted    KILLED TestCmdGroupStandGroundAnchorsEveryLivingMembersCurrentCell
M4  FR-2             the living filter dropped               KILLED TestCmdGroupStandGroundAnchorsEveryLivingMembersCurrentCell
M5  FR-4             the walk home's destination write cut   KILLED TestAnOffPostGuardWithNoCandidatesWalksHome (+5)
M6  FR-5             the victim test cut                     KILLED TestAGuardHoldingAVictimGainsNoDestinationEvenOffPost
M7  FR-6             the on-post test cut                    KILLED TestAnArrivedGuardIsLeftInEveryField (+4)
M8  FR-7             the empty-candidate call site deleted   KILLED TestAnOffPostGuardWithNoCandidatesWalksHome (+5)
M9a FR-8 FR-9        guard gate dropped, empty-cand. site    KILLED TestAStandGroundMemberGainsNothingEvenOffPost
M9b FR-8 FR-9        guard gate dropped, loop-foot site      SURVIVED -> see below
M10 FR-10            PostX encoded as 0                      KILLED TestMarshalledBytesArePinned (+11)
M11 FR-10            decoded post clamped at 0               KILLED TestThePinnedBytesDecodeBackToThePinnedWorld (+5)
M12 FR-10 FR-11      formatVersion back to 23                KILLED TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth (+11)
M13 AC-11            decoder also accepts version 23         KILLED TestAVersion23FormIsRefused, ...ButTwentyFour
M14 P-1              post follows the mover in stepWorld     KILLED TestTheTickOneDigestIsTheContractsRoutesFirstCell (+11)
M15 P-2 SC-1         one w.rng draw added to walkHome        KILLED TestAnOffPostGuardWithNoCandidatesWalksHome
M16 AC-10            an out-of-map post refused on decode    KILLED TestThePinnedBytesDecodeBackToThePinnedWorld (+11)
M17 AC-10            the post zeroed on a dead entity        KILLED TestThePinnedBytesDecodeBackToThePinnedWorld (+5)
```

**M9b survived, and it was not equivalent — it was unwitnessed.** Dropping `order == orderGuard`
from the walk home's second call site left the whole suite green while dropping it from the first
was killed at once. The reason was a fixture shape: every non-guard-stance fixture in the suite
builds a member with nothing to see, so its group leaves `decide` through the empty-candidate branch
and never reaches the foot of the per-member loop. Stand Ground and Swarm 2 both reach that foot the
moment their candidate list is non-empty, and there the gate was the only thing holding the post out
of their reach. `TestNoNonGuardStanceWalksHomeOffTheLoopFoot` (commit `0418836`) gives each of them
a candidate it can see and cannot score — the preference table's absolute veto, a ground member
never auto-selecting a flier, chosen over distance because it holds whatever the two arms do about
reach. It fails on the mutant and passes on the tree.

**AC-12 is a positive check, not a kill.** `walkHome` and both call sites were deleted outright:
`go build ./...` exited 0, and the only tests that failed were this story's own six walk-home tests.
Every guard test that predates the story, and this story's own three "nothing happens" tests, stayed
green — which is the whole of AC-12's claim.

## Every id, and what witnesses it

**FR-1** `Entity.PostX, PostY int32` (`world.go`), pinned in `nostate_test.go`'s reflective field
set, which is a test that fails on any field added without it. **FR-2** and **FR-3**: M1 for the
constructor, M2/M3 for the two commands, M4 for the living filter; `TestOnlyTheTwoStancesWriteAPost`
is the negative half, sweeping swarm, move, swarm 2 and patrol and finding the post where the
constructor left it. **FR-4** M5. **FR-5** M6. **FR-6** M7. **FR-7** M8, and
`TestAnOffPostGuardWithNoCandidatesWalksHome` reaches the empty branch because nothing else exists
to see, not because a clip narrowed a list. **FR-8** and **FR-9** M9a and M9b, plus
`TestAStandGroundMemberGainsNothingEvenOffPost`, which also steps 64 ticks and finds the member
where it stood. **FR-10** M10, M11, M12, M16, M17. **FR-11** `TestThePinIsThePreStoryPinPlusThePost`
is the sharp one: it strips exactly eight bytes per record off the pinned form, puts the version byte
back, and requires the result to hash to the digest the tree carried one story ago. A pin and a
digest taken from that pin agree by construction; this one cannot be faked by re-taking both.

**AC-1** `TestAnOffPostGuardWithNoCandidatesWalksHome`. **AC-2**
`TestAWalkedHomeMemberDoesNotReIssueOnTheNextDecision` and `TestAnArrivedGuardIsLeftInEveryField`.
**AC-3** `TestAGuardHoldingAVictimGainsNoDestinationEvenOffPost`, whose candidate is placed adjacent
to where the member *stands* and not to its post, so the fixture cannot pass by coincidence.
**AC-4** `TestAReleasedOffPostGuardIsWalkedHomeInTheSameTick`. **AC-5**
`TestAGuardsOwnOffPostDestinationIsReplacedByItsPost`. **AC-6**
`TestAStandGroundMemberGainsNothingEvenOffPost`. **AC-7** `TestAFreshWorldsPostIsEveryEntitysOwnCell`,
over an entity in no group, one not alive, and one under a non-stance order. **AC-8** M2, M3, M4.
**AC-9** `TestAMidCrossingEntitysPostIsItsOwnCellAtConstruction` and
`TestAMidCrossingMembersPostFollowsTheCellItHolds`. **AC-10** M16 and M17 — both clauses, the
out-of-map post and the post on a dead entity, are carried by fixtures that already exist and are
compared whole. **AC-11** `TestAVersion23FormIsRefused` (below). **AC-12** the deletion above.

**P-1** M14. **P-2** M15, and every walk-home fixture asserts `w.rng` unchanged across the decision.
**P-3** no writer reads a group's order: M1 shows the constructor writing every entity whatever its
order, and `TestOnlyTheTwoStancesWriteAPost` shows the four non-stances writing none. **P-4** and
**SC-2** are discussed below. **P-5** the digest pins: every world in the suite holding no guard
group hashes to what it hashed before, which is what makes M14's twelve failures meaningful.

**DD-1** the pair is on `Entity`, not on the group record — the field set pin says where.
**DD-2** `TestTheConstructorOverwritesWhateverPostACallerNamed`. **DD-3** the gate reads the cell
alone (M7); the divergence it discloses — a member at its post still owing crossing ticks is
re-ordered in the law and left alone here — is an absence, disclosed rather than witnessed.
**DD-4** and **DD-5** likewise absences: M15 shows no draw was added, and this tree has no
regeneration for the heal to be inconsistent with. **DD-6** is a claim about `0099`'s D-3 and is
carried by `engage.go`'s own header. **DD-7** M12, plus
`TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth`, which reads every offset by hand: no
existing one moved.

**SC-1** M15. **SC-3** the deletion under AC-12. **SC-4** no test added here reads an install; there
is no `AGAINROM_ASSETS`, no `os.Getenv` and no path in any of the three new files. **SC-5** the gate.

## What is wrong, and left standing

**SC-2 is false as written and cannot be made true.** It says "No file outside `pkg/sim` changes."
Three files outside it did:

- `pkg/mapload/fromalm_test.go` and `pkg/mapload/gridform_test.go` — both hold hand-transcribed
  `sim.Entity` literals and hand-transcribed byte forms over `FromALM`'s output. A field added to
  the record and eight bytes added to each record break them by construction. This is what the reach
  field's own story did one version ago.
- `internal/archtest/destination_test.go` — the pinned list of functions that may write a
  destination. `walkHome` is a new one, so the list gains its name. That file exists to be edited
  exactly when this happens; leaving it alone would have failed the gate.

**P-4, which SC-2 exists to enforce, is intact**: no non-test file outside `pkg/sim` changed, and no
package outside the simulation gained a field, an import or a call. The clause is left standing with
this note rather than rewritten, because the contract is `master`'s and the honest record is that it
was stated more tightly than the change could be made.

**AC-11 had no witness of its own until this stage.** The version sweep refuses 23 along with every
other non-current byte, so the refusal was covered — but it refuses a form whose version byte was
*spoiled*, and it reads no message. 23 is the one number for which a real, well-formed stream
already exists in the suite: `strippedOfThePost` builds the exact form this tree emitted one story
ago, byte-for-byte current up to `+119` in every record, so nothing but the version check stands
between it and a decode that would read the next record's first eight bytes as a post.
`TestAVersion23FormIsRefused` asserts both arms and the message on each (commit `20f3d27`).

**`TestUnmarshalRefusesEveryVersionButTwentyThree` was still called that at version 24.** Its own
doc block records that this staleness has had to be repaired after the fact twice already and asks
the next bump to move the name; the next bump did not. Renamed to `...ButTwentyFour`, and the tally
now records the third occurrence. The paragraph has failed as often as it has been written, which is
an argument for a check that reads the name — not made here.

**Where `0098`'s release lives under order 3 is still not established.** `decide`'s generic
per-member release runs under `orderStandGround` as well as guard, keyed on the owner, and that is
`0098`'s FR-23/DD-17 with a test pinning it. No published row establishes order 3's tail either way.
This story does not touch that path and does not resolve it; it is disclosed here and `0098`'s
contract is unchanged.

## The milestone

`cmd/missionrun`'s `TestTheTenthMissionIsDrivenToAWin` was run on both lawful roots before and after
the change. All four results are the same: `outcome lost at tick 272`.

```
en before: outcome lost at tick 272      en after: outcome lost at tick 272
ru before: outcome lost at tick 272      ru after: outcome lost at tick 272
```

That is what stages 1–3 predicted and it is the expected result, not a shortfall: a hostile that has
never engaged is already standing on its post, so the walk home is a no-op for it. The outcome could
only move where something had already pushed a guard off its post. Nothing was tuned to make it
move; a predicate adjusted until mission 10 wins would break the other 27 maps.
