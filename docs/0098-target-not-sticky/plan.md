# Plan — 0098

## Shape

One package, one file of production code, one new test file. Everything the contract needs is in
`pkg/sim/engage.go`; the release calls two functions that already exist, in `combat.go` and
`step.go`, and adds nothing to either.

## Design decisions

**DD-1 — the release is one function and it is `orderAttack`'s inverse.** `orderAttack` is the ONE
writer of an attack order and says so in its own comment. Its inverse is added beside it, named for
what it does rather than for what it clears, and it is the ONE clearer reached from a decision. Two
call sites, both inside `decide`, and no other caller anywhere. Serves FR-1, FR-13, AC-13.

**DD-2 — it clears the attack fields and the order fields, through the two functions that already
own those clearings.** `Entity.clearAttack` drops victim, phase and countdown together and exists
precisely so that no site forgets one; `World.clearOrder` drops destination, stall count and route
together for the same reason. Writing seven fields out here would make a third place that has to
agree with them. Serves FR-1, AC-12.

**DD-3 — `clearOrder` is called plain, not through `restAt`.** `restAt` is the occupancy-aware form
and exists because the move loop runs with a live scratch plane. `engagementPass` runs *before* the
tick's scratch is built — the file's own comment on `engagementPass` says so, and `orderAttack`
already calls `clearOrder` plain for exactly that reason. Using `restAt` here would need a scratch
that does not exist. Serves AC-14.

**DD-4 — the release returns immediately on a member holding no victim.** This is FR-2 and it is
load-bearing twice: it keeps a walking member's destination, and it makes the release provably a
no-op on the population any commanded-unit story owns. It is a guard on `HasAttackTarget`, and it
is what AC-4 witnesses.

**DD-5 — the count test replaces the early return; it does not sit beside it.** Today `decide`
returns on `len(cands) == 0`. That return is deleted. In its place stands a test on the two counts
narrowed to bytes, whose body **releases every member and then returns**. The ordinary per-member
release below it then handles everything else, including the empty-list case that the byte test
would already have caught. Two sites, matching the law's two clearing sites, and neither is
reachable by accident. Serves FR-7, FR-8, AC-5, AC-8, AC-9.

**DD-6 — the narrowing is `uint8(len(...)) == 0` and not a comparison against 256.** The law reads a
byte; a modulus written out would be a second spelling of the same fact and would stop agreeing the
day the field's width was revisited. This is the same form `candidateCost` already uses for the
distance and `noticeBase` for its maximum. Serves FR-8, FR-9.

**DD-7 — the member-count test is written even though it cannot fire.** `aiGroups` yields a group
only when a living member was found for it, so `len(g.members) >= 1` always and the narrowed value
is zero only at exactly 256, 512, … members. It is written because the law tests both counts in the
same instruction pair and because a build that tested one and not the other would be making a claim
about which of the two is reachable. The comment says it cannot fire and why. Serves FR-8.

**DD-8 — the stance gate is `st == stanceGuard`, with the owner rule written beside it.** The code
tests the property `decide` already holds; the comment states that the law's own gate is the
member's owner, that this build has one human participant seated at the one stand-ground slot, and
that a build which separates the two must split this test. Serves FR-5, FR-6.

**DD-9 — the file's header paragraph is rewritten, not patched.** Its last three sentences state
0086 FR-20 as decoded fact and cite a clause the pin has since **retracted** (`AI-GRPGUARD-074`'s
`ord+0x00` parenthesis, refuted by `AI-POST-095`). Leaving them beside code that does the opposite
is the failure mode `WORKFLOW.md` S-7 names. The rewrite says what the arm's three outcomes are,
which of them this build has, and that the walk home and the turn are still absent. Serves FR-3,
FR-4.

**DD-10 — no format version is taken.** Clearing fields adds none: no field, no width, no ordering
changes, so the byte form, its version byte and any given world's digest are what they were.
FR-12's round-trip and digest checks are the falsification, not the absence of a literal. **If the
executor finds itself reaching for a version number, it has left the plan and must stop.**

**DD-12 — the two landed tests were found by running the change, not by reading.** A probe applied
the minimal edit and ran the suite; two tests in `pkg/sim/engage_test.go` went red and **only**
those two. Reading had predicted one. The second, `TestReEngagingTheSameVictimDoesNotRestartTheCycle`,
fails in its **control** arm — a peaceful-relation world driven by an attack command — which no
reading of the story's own FRs would have reached, because the arm is not testing this story at all.
Both are retargeted to the exempt stance, `TestAFriendlyRelationDoesNotEndAnAttackAlreadyIssued`
keeping its name true and gaining a guard-stance counterpart in `release_test.go`. Serves FR-14,
FR-15, AC-15, AC-16.

**DD-13 — `internal/archtest`'s destination-writer pin is not touched.** It matches
`.HasTarget = true` in lockstep and the release assigns `false`, so it cannot fire. If it does, the
release has grown a writer and the change is wrong — the tripwire is not to be widened. Serves
FR-13, AC-17.

**DD-11 — `decide`'s member loop is not otherwise restructured.** The self-refusal, the scorer call,
the clip and the candidate build are untouched. The diff is: one deleted return, one new test with a
release loop, one `else` on the existing `if at >= 0`, one new function, one rewritten comment
block. Serves FR-11, AC-10.

## The change, precisely

In `pkg/sim/engage.go`:

0. Two paragraphs are `0095`'s residue and false today: `aiGroup`'s doc says a group record would be
   "state no world could exercise" and that none of its three fields "has anything to be frozen
   against", which `0095`'s frozen `base` — carried on the World, in the byte form and in the digest
   — contradicts. Corrected while the file is open.
1. Rewrite the header comment's closing paragraphs per DD-9.
2. `decide`: replace `if len(cands) == 0 { return }` with the narrowed two-count test of DD-5/DD-6/
   DD-7, whose body releases every member under DD-8's gate and returns.
3. `decide`: give `if at >= 0 { w.orderAttack(...) }` an `else` that releases under DD-8's gate.
4. Rewrite `decide`'s own doc comment: it currently states 0086 FR-20 verbatim.
5. Add `releaseAttack` beside `orderAttack`, per DD-1 through DD-4, with a comment naming it as
   `orderAttack`'s inverse and as the only clearer a decision reaches.

In `pkg/sim/engage_test.go`, per DD-12: retarget the two named tests' worlds to the exempt stance,
each with a comment saying why the owner moved and what the test still witnesses. Their assertions
do not change.

Nothing outside those two files and the new `pkg/sim/release_test.go` changes.

## Verification plan

**Unit, in `pkg/sim/release_test.go`** — every one of these must fail if the line it witnesses is
removed, and the executor checks that by removing it:

| Test | Witnesses |
|---|---|
| a guarding member holding a victim that becomes unscoreable is left holding neither order, all seven fields checked | FR-1, AC-1 |
| the same world stepped on for further ticks: position and health unchanged | FR-4, AC-2 |
| a guarding member whose victim still scores keeps it, cycle untouched | FR-11, AC-3 |
| a guarding member holding a destination and no victim, scoring nothing, keeps destination, stall count and route | FR-2, AC-4 |
| a guarding group with an empty candidate list: the victim-holder is released, the walker is not | FR-7, AC-5 |
| a stand-ground member whose victim stepped past reach keeps it | FR-5, AC-6 |
| a guarding member whose only candidate is vetoed by the preference table is released; the same world with the veto absent engages | FR-10, AC-7 |
| 256 scoreable candidates release; 255 and 257 engage | FR-8, FR-9, AC-8, AC-9 |
| a released member's world round-trips through the byte form and decodes | FR-12, AC-12, AC-14 |
| a world in which every decided member scores something, stepped many ticks, matches a golden field dump taken before the change | FR-12, AC-10 |
| the byte form's version byte and a fixed world's digest are the values that already shipped | FR-12, AC-11 |
| a source-level check that the victim's setter and clearer are one function each and that the clearer's only callers are in `decide` | FR-13, AC-13 |
| a guarding unit given a victim by an attack command, nothing hostile to it, is released; the same at the exempt stance keeps it | FR-14, AC-15, AC-16 |
| the pinned destination-writer set is the one that shipped | FR-13, AC-17 |

**Evidence, a pipeline stage and not a task.** On **both roots**, before and after, from the story
head: the 28-mission / 2000-tick / no-command census (SC-1); the mission-10 clubman (SC-2); mission
10's outcome, tick and arm from `missionrun -trace` (SC-3). The census harness is built outside the
repo — no census mode is added to `cmd/missionrun`, and no test expectation or predicate is edited
(SC-3). Reverts recorded for SC-4. SC-5 names the population that keeps fighting from the same runs.

**Build, a pipeline stage and not a task.** `builds/0098-target-not-sticky/` with `againrom.exe`,
`missionrun.exe` and a `README.md` whose commands are run on both roots before they are written
down.

## Risks

- **The census may not fall.** If it does not, SC-1 says so plainly and the story ships as a
  correctness fix with a null behavioural result, as `0095` did. It does not get rescued by widening
  scope.
- **The census may fall too far**, because this build has no remembered-attacker memory and so
  releases sooner than the law does (`analysis.md`). Any large fall is reported with that caveat
  attached rather than as a faithful reproduction.
- **`0097` touches the same loop.** The merge resolution is its skip outside this release; the two
  populations are disjoint, so no clause of either needs rewording.
