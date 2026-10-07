# Plan — 0097

## The shape

One predicate, one line in the partition, and two test surfaces. The engagement decision's arms are
not touched at all: the defect is in *who* is decided over, so that is the only thing that moves.

## Design decisions

**DD-1 — the state is derived, and therefore no `formatVersion` is taken.** The law's latch for
"this member is still executing its move order" is a per-order byte cleared while the member walks
and set when it arrives. This build already carries that latch under another name: a unit still
holds the destination its order wrote. Reading it rather than storing a second flag beside it means
the byte form, its version and every world's digest are untouched (FR-10), and means the two cannot
come apart. The alternative — an entity field and version 21 — buys robustness against a writer
that does not exist yet and pays for it with hashed state and a coordination cost against two other
live lanes. **This story takes no format version.**

**DD-2 — the skip goes in the partition, not in the decision and not in the order writer.** Three
places could carry it and only one is right. In `decide` it would be a per-member test inside a
group's arm, which says the group decided and then declined — the law's group never asks. In
`orderAttack` it would make the *writer* of an attack order conditional, and that writer is shared
with the attack command, which is entitled to end a walk. In `aiGroups` it is a statement about
membership, which is what it is: the unit is in the group its command built, not in the group the
map placed it in. It also lands beside the rule that already excludes an entity of slot 0 for the
same kind of reason, so the file gains a second clause rather than a second mechanism.

**DD-3 — a commanded unit's sight leaves its authored group, and that follows from DD-2 rather than
being chosen.** The candidate list is built from the members the partition returns, so a member the
partition does not return contributes nothing. FR-5 discloses it. The alternative — keep the member
in the partition for sight and skip it for scoring — would be a unit in two groups at once, which
nothing at the pin supports.

**DD-4 — on arrival the unit returns to the stance its owner slot selects.** The law hands an
arrived member to a per-unit acquisition instead; both admit only what is within reach, and this
tree has no per-unit acquisition to hand it to. Bounded and disclosed in FR-8. Building the per-unit
acquisition here would be a second story inside this one.

**DD-5 — no third `stance` value.** A value whose only behaviour is to decline is a value that says
nothing (S-7). The group order this story implements is expressed as the absence of a decision,
which is exactly what its arm does, and the type's doc says so.

**DD-6 — the exactness of FR-1 is defended by a source scan, not by a comment.** FR-1 is exact
because of a property of the package rather than of any one function: only the two command arms give
an entity a destination without also leaving it a victim. A comment saying so would go stale
silently. `internal/archtest` already owns "a property of `pkg/sim`'s source" and already has the
shape — a pure check over a map of file contents, plus a loader — so this is a second check in an
existing home rather than a new mechanism.

## Files

**`pkg/sim/engage.go`** — the whole of the behaviour change.

- A new `underCommand(e Entity) bool`, returning `e.HasTarget && !e.HasAttackTarget`. Its doc
  carries three things and nothing else: that it is the law's arrival latch read out of fields this
  tree already has; that it is exact only because the two command arms end the fight they find; and
  that a future writer of a destination on a unit holding no victim — the walk-home branch this
  file's header already names as absent, or a script-ordered move — inherits it and must decide
  what it means, with `internal/archtest` the thing that will say so.
- `aiGroups` gains one clause in the existing skip: `underCommand(e)`. Its doc gains a short
  paragraph beside the slot-0 one, saying the unit is in the group its command built, that that
  group's order is *walk and re-issue the walk* with no candidate list and no scorer, and that the
  absence of the group is therefore the whole of its behaviour — and that the unit stays a candidate
  for everyone else, which is the same asymmetry the slot-0 paragraph already states.
- The `stance` type's doc: order 4 is now reached, and is carried as an absence from the partition
  rather than as a value here.

Nothing else in the file changes. No arm, no scorer, no constant, no `orderAttack`.

**`pkg/sim/commanded_test.go`** — new. AC-1 to AC-10, AC-12, AC-13.

**`pkg/sim/partyslot_test.go`** — one test rewritten, AC-15. **Added in Phase 4**, because Stage 3
enumerated the code this story changes and not the measurement that already pinned the behaviour it
removes. The previous story's FR-6a test drives a player's unit under a single order and asserts it
is *held in contact* — the defect itself, recorded as a fact. It is rewritten to the assertion it
was already comparing against, in the same commit as the change, because a commit that made the
production change without it would leave the tree red and a commit that rewrote it first would
assert a behaviour the tree did not yet have. Its slot-0 control and its two walkers survive whole;
what changes is which side of the comparison is expected to win.

**`internal/archtest/destination.go`**, **`destination_test.go`** — new. AC-11.

## Tests

Built on the package's existing synthetic-world helpers; no test reads an install.

| AC | Test |
|---|---|
| AC-1 | a slot-1 unit holding a destination with a hostile adjacent: after a decision it still holds the destination and holds no victim; stepped on, its cell advances |
| AC-2 | the same world, from the hostile's side: the hostile's group is given the commanded unit |
| AC-3 | the same world with the destination removed: the unit is given the hostile — the control that says the world itself would have produced a fight |
| AC-4 | a unit holding both a victim and a destination: after a decision it is still scored, shown by its victim being re-chosen when a nearer candidate appears |
| AC-5 | a unit one step from its destination beside a hostile: stepped until it arrives, then decided — it holds the hostile |
| AC-6 | a commanded unit struck by a command: it keeps its destination and takes no victim |
| AC-7 | two units of one group given one group move order, only one of which can see a hostile: neither is given a victim, and a third group whose only line of sight is through the commanded member gets nothing |
| AC-8 | a second move order onto a commanded unit: still commanded, new cell |
| AC-9 | an attack command onto a commanded unit: no longer commanded, victim held |
| AC-10 | a commanded unit felled: neither destination nor victim |
| AC-12 | a world holding a commanded unit: round trip byte-identical, version byte equal to the constant, and the digest equal to a literal pinned in the test — so a version bump made later cannot pass silently |
| AC-13 | two worlds with no commanded unit stepped 64 ticks: entity slices equal |

AC-11 is `internal/archtest`: parse `pkg/sim`'s non-test files, collect every assignment whose
left-hand side selects the destination flag and whose right-hand side is `true`, and report the
enclosing function of each. The expected set is pinned by name. A finding names file, line and
function, in file order, exactly as the determinism check does. Its own tests drive the check over
literal source strings — an unexpected writer, an expected one removed, an empty set — so the check
is tested without the tree having to be broken.

## Measurements

All on both roots, from the orchestrator's own installs, before and after.

**SC-1** `missionrun -mission 10 -waypoint p0:25:55:1 -ticks 400`. Before: `STOPPED SHORT of
(24,57), Chebyshev 2, after 400 ticks`, outcome undecided at 464, identical EN and RU.

**SC-2** The same drive, reported as a pair: the tick at which the commanded unit passes the hostile
without engaging, and the tick at which — after arrival — it engages something within reach. If it
arrives with nothing in reach, drive it again to a cell beside a hostile and report that instead.
The claim is *a unit not under command fights*, and the measurement must show the fight.

**SC-3** Over every mission both roots ship, count the placements at the player's own roster slot
and how many hold a destination at load. Expected 9 and 0, on maps 41, 71, 150, 151 — measured
already, before the design was fixed, and to be re-measured after. Zero holding a destination is
what makes "still Stand Ground" a fact about the maps rather than about the change.

**SC-4** `missionrun -mission 10 -waypoint u21:56:21:3 -waypoint p0:66:16:3`. Before, EN: waypoint 1
`was STOPPED BY THE WORLD DECIDING, short of (43,46), Chebyshev 25, after 272 ticks`, `outcome lost
at tick 272`. This drive commands a script-owned unit as well as the party, so its first waypoint is
squarely inside this story's change and the outcome is expected to move. Record it, on both roots,
and if it moves report the tick and which arm ends the mission — `missionrun -trace` is on master
and prints it. **`TestTheTenthMissionIsDrivenToAWin` is not edited, and no predicate or threshold
anywhere is tuned.**

## Order, and what the gate is

The two tasks are independent and are done in the order given only because the second is a tripwire
for the first.

Local gate before anything is pushed: `go build ./...`, `go vet ./...`, `gofmt -l` over the tree's
own files, `go test -count=1 -trimpath ./...` with no install present, then again with
`AGAINROM_ASSETS` set for the drives, and `scripts/check-no-game-assets.sh`,
`check-doc-budget.sh`, `check-sdd-audit.sh`.

## Success criteria

- **FR-1** and **FR-2** -> AC-11 for the writer set, AC-4 and AC-9 for the "no victim" half, AC-12
  for the absence of a field. DD-1 and DD-6 are what make them defensible rather than asserted.
- **FR-3** -> AC-1, AC-3, AC-5. DD-2 places it. **FR-4** -> AC-2. **FR-5** -> AC-7, from DD-3.
- **FR-6** -> AC-6. **FR-7** -> AC-5, AC-8, AC-9, AC-10. **FR-8** -> AC-5, bounded by DD-4.
- **FR-9** -> AC-13, AC-3. **FR-10** -> AC-12. **FR-11** -> AC-15.
- **DD-5** is a negative and is witnessed by the absence it names: the `stance` type still has two
  values, which AC-13's field-for-field equality would not survive if a third had been added and
  reached.
- **SC-1** to **SC-4** are measured from the tools and the corpus and recorded in `verification.md`;
  none is asserted in the suite, and SC-4 explicitly must not be.

## Risks

**The diff overlaps two live lanes.** `0095` and `0096` both hold `pkg/sim/engage.go`. The change is
kept to one new function and one clause inside `aiGroups` for that reason as much as for S-7: a
conflict against either is then a conflict in one hunk.

**Mission 10's outcome is expected to move**, because its first waypoint drives a unit that is under
command for most of the run. That is a measurement, not a regression, and SC-4 forbids reaching for
the test to make it come out.
