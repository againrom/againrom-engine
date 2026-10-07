# Story 1141 — a patroller fights

## Player result

The troll standing beside the party's start in mission 80 now attacks. Before
this story it walked its two-cell patrol past a party unit standing next to it
and never stopped; the party could shoot it down at leisure or walk away. The
change is campaign-wide and not mission-80-shaped: every actor in patrol state,
on every map, now runs the per-actor guard arm before it advances its ring, and
every actor in guard state under a group order of 0 runs that arm at all.

Measured in the campaign drive
(`missionrun -mission 80 -census -ticks 4000`, EN root, base `bef68ef` against
this candidate): **1 fell** before this story, **4 fell** after it, over the
same 9 of 107 movers. The three extra deaths are the three units that walk
into the troll's post block:

```
-   moved  u112   slot 4 group 3  (100,37) -> (105,41)  125 step(s), 480 hp
-   moved  u115   slot 2 group 2  (115,28) -> (106,40)  105 step(s), 51 hp
-   moved  u116   slot 2 group 2  (107,20) -> (106,43)  187 step(s), 51 hp
-   moved  u117   slot 2 group 2  (107,16) -> (105,40)  175 step(s), 20 hp
+   moved  u112   slot 4 group 3  (100,37) -> (109,37)  133 step(s), 480 hp
+   moved  u115   slot 2 group 2  (115,28) -> (109,35)  23 step(s), -146 hp
+   moved  u116   slot 2 group 2  (107,20) -> (108,35)  31 step(s), -119 hp
+   moved  u117   slot 2 group 2  (107,16) -> (107,35)  35 step(s), -122 hp
```

Unit 112 is the patrolling troll. Before this story it wandered off its patrol
line to (105,41) and the three units that met it survived on 51, 51 and 20 hp;
now it stays on the line between (100,37) and (110,37) and all three are dead
within 35 steps. Nothing else in the census moved.

## In-scope behaviour

Four behaviours under one contract, all at the actor layer:

1. **The guard arm exists** (`armGuard`, `pkg/sim/guardarm.go`). It anchors the
   post on a zero, scans a Chebyshev block of radius 5 around the post,
   engages the first survivor it finds, walks home when the block is empty and
   it is not standing on its post, and falls through to `acquireStanding` when
   the block is empty and it is home.
2. **The actor dispatch reaches it** (`actorPass`, `pkg/sim/actor.go`). The
   guard case is the one arm with a gate of its own: it runs only for an actor
   whose group order is 0.
3. **Patrol runs guard first** (`armPatrol`). The arm clears the order,
   re-anchors the post to the actor's current cell, calls guard, and advances
   the ring only when guard left the order idle.
4. **Patrol anchors a post.** The script's sub-command 14 and the player's own
   patrol command both write the member's current cell as its post.

## Authority

Read from the pinned snapshot (`knowledge/`, `go run ./tools/claim <ID>`).

| Claim | What this story takes from it |
|---|---|
| `AI-PATROL-013` | `R0159` clears the move order, calls the guard arm at `L00345`, and advances the ring only when guard leaves the order 0 or 0xb |
| `AI-PATROL-017` | script sub-command 14 builds the two-node ring |
| `AI-PATROL-018` | `R0172` writes `order+0x00` — the actor's own cell — as the post at `L00415`; `order+0x04` is the re-anchor latch, set at `L00423` and consumed at `L00424…L00425` |
| `AI-POST-042` | patrol is guard with a moving post; `ord+0x00` is anchored on a zero at `L00327` (three clauses of this claim are retracted by EXP-0125 and are not used) |
| `AI-BREAK-041` | the block is post-to-target Chebyshev with radius `mover+0x08` = 5 (written at `L00587`); engage the first survivor at `L00586`; empty block and not home walks home at `L00334`; empty block and home falls into `R0022`; the `actor+0xa5` sight test is behaviourally dead |
| `AI-GUARD-012`, `AI-GUARD-007` | the guard arm's identity and its engage routine `R0009` |
| `AI-ORDER-010`, `AI-POST-095` | a group under order 0 hands its members to their own `actor+0x50` states; a group under order 1 or 3 never evaluates that byte |
| `AI-SCRIPTATTACK-120` | sub-command 10 writes `actor+0x50 = 3` for every non-target member |
| `AI-CMD-054` | the player's order `0x19` writes `actor+0x50 = 3` at `L00014` |
| `AI-STATE-043` | the written value set of `actor+0x50` includes 3 |

## Touched surfaces

| File | Change |
|---|---|
| `pkg/sim/guardarm.go` | new: `armGuard`, `postEngage`, `guardWalkHome`, `actorLayerDecides`, `postRadius` |
| `pkg/sim/actor.go` | the `actorStateGuard` dispatch case; `armPatrol` rewritten around the guard call; `patrolInterrupted` |
| `pkg/sim/facing.go` | `clearTurnUnlessCasting` extracted out of `cancelTurnForTargetChange` |
| `pkg/sim/savedgroupsai.go` | `savedGuard`'s block scan is now the same `postEngage` body |
| `pkg/sim/script.go` | `cmdGroupPatrol` writes the post; `cmdGroupAttack` writes actor state 3 |
| `pkg/sim/group.go` | `commandPatrol` writes the post |
| `pkg/sim/step.go` | the player's accepted attack order writes actor state 3 |
| `pkg/sim/world.go` | `actorStateEngage` (3) exists, is a defined state, and is named in `patrolFault` |
| `pkg/sim/patrolfight1141_test.go` | new: 18 witnesses |
| `pkg/sim/stanceanchor_test.go` | corrected expectation: patrol writes a post |
| `pkg/sim/pickupcompletion1117_test.go` | corrected expectations: state 3 after an attack order; the unknown-state probe moved from 3 to 7 |
| `internal/archtest/destination_test.go` | `guardWalkHome` added to the destination-writer pin, with its triage |

### Two decisions inside the arm

**The re-anchor latch is derived, not stored.** `AI-PATROL-018`'s `order+0x04`
is a real field of the original's order record. This build does not add it to
the entity record: `patrolInterrupted` answers the same question from state the
record already carries — an actor holding an attack target, or a destination
that is not its own leg waypoint, was interrupted and does not re-anchor. The
alternative was a new persisted field, which moves `formatVersion` 85 to 86 and
touches nine dedicated fast paths in `upgrade.go`. This is 0099 D-2's own
precedent, and it has one parting case, recorded as DIV-985.

**It is read before the order is cleared, and that ordering is the leash.**
`clearOrder` zeroes `HasTarget`, so a latch read after it can see only the
victim half of the question and never a walk home. The post would then
re-anchor on the entry after every break-off, the five-cell block would travel
with the actor, and `AI-BREAK-041`'s post-relative leash would become an
actor-relative one that no pursuit ends: a target that keeps withdrawing tows
the patroller across the map. `savedPatrol` (`pkg/sim/savedgroups.go`) reads
the stored `o.Raw[4:]` at the same point, before anything clears it, so the two
readings of `R0159` in this package are one reading.

**The dead sight test is omitted.** `AI-BREAK-041` grades the `actor+0xa5` test
behaviourally dead. `savedGuard` already omitted it, so both sides of the same
routine agree.

## Actor state 3 has a writer again

Story 0099's D-10 declined to model `actor+0x50 = 3` because no arm read the
byte. The guard arm reads it now, and the byte's absence became a defect: a
member handed an attack order by script sub-command 10 was left in guard state,
so the guard arm broke its order off on the very next tick — the target is not
a hostile inside the member's own block. The paired release tests caught this
(`AI-SCRIPTATTACK-120: non-target member did not acquire the named victim`, on
`71.alm/5`, `120.alm/16` and `140.alm/14`, both roots), not `go test ./...`.

State 3 is therefore written at the two sites two High claims name — script
sub-command 10 (`AI-SCRIPTATTACK-120`) and the player's own accepted attack
order (`AI-CMD-054`) — and joins `actorStateDefined`. It has no arm of its own:
`actorPass`'s switch still has no default, so an engaging actor is left to the
combat path that owns it. That is the D-1 seam unchanged.

## Proof

### Witnesses

`pkg/sim/patrolfight1141_test.go`, 18 tests. The load-bearing ones:

- `TestAPatrollerFightsWhatStandsInsideItsBlock` — the story's result. The
  candidate stands three cells away and the patroller's reach is 1, so
  `acquireStanding` (stand ground, reach cap) cannot have produced the victim.
- `TestAPatrollerWalksOnPastANonHostileNeighbour` — a non-hostile one cell away
  is not engaged and the ring advances.
- `TestAPatrollerWalksOnPastAHostileOutsideTheBlock` — the boundary pair, five
  cells engaged and six cells not, which is what pins radius 5.
- `TestAPatrollersPursuitBreaksOffWhenTheTargetLeavesTheBlock` and
  `TestAGuardAtHomeDropsAVictimThatLeftItsBlock` — `AI-BREAK-041`'s leash from
  both positions.
- `TestAPatrollersWalkHomeSurvivesTheNextPass` — one actor pass further than
  the test above, which is where the leash is kept or lost: the post must not
  move on the entry that follows a break-off, and the walk home must be
  re-issued rather than dropped.
- `TestAKitedPatrollerStaysOnItsLeash` — the same law as a bound. A hostile
  withdrawing one cell every second pass over 120 passes must not tow the
  patroller past its own ring tail. Measured on this tree:
  `actor=(15,10) post=(10,10) victim-held=false engaged=6/120`.
- `TestTheRingStillAdvancesWhenGuardFindsNothing` — the ring still runs.
- `TestTheGuardCaseRunsOnlyForAGroupUnderOrderZero` — the dispatch gate.
- `TestTheMission80TrollLeavesPatrolWhenAPartyUnitComesInside` — the mission's
  own shape, stated as coordinates (unit 112, group 3, patrol (100,37) to
  (110,37), a party unit inside the block). It reads no install byte.
- `TestTheSavedGuardScansTheSameBlock` — the same block from the
  original-runtime side.
- `TestAScriptAttackSurvivesTheNextActorPass` — the release-test defect above,
  as a unit witness.

### Mutation table

Every production edit reverted alone, `go test -count=1 ./pkg/sim/
./internal/archtest/` run against the otherwise complete tree, the file
restored. Twenty edits, twenty failures; the first failing test and its own message:

| # | Reverted edit | Failing witness | Message |
|---|---|---|---|
| P1 | the `actorStateGuard` dispatch case | `TestAGuardAtHomeDropsAVictimThatLeftItsBlock` | a guard standing on its post kept victim 2, ten cells away |
| P2 | `armPatrol`'s guard call | `TestTheMission80TrollLeavesPatrolWhenAPartyUnitComesInside` | the troll holds victim 0 (present false), want the party unit 1 |
| P3 | `armPatrol`'s 0/0xb gate | `TestTheMission80TrollLeavesPatrolWhenAPartyUnitComesInside`, and `TestAKitedPatrollerStaysOnItsLeash` with it | the troll also holds destination (110,37): it is still walking its ring while fighting — and the ring advance overwrites the walk home, so the kited patroller is towed to x=72 |
| P4 | `armPatrol`'s re-anchor condition (re-anchor on every entry) | `TestAPatrollersWalkHomeSurvivesTheNextPass` | fixture: pass 2 did not produce a walk home (victim true, destination false) — the post travels with the actor and the pursuit never breaks off |
| P4b | `armPatrol`'s re-anchor write itself (never re-anchor) | `TestThePostFollowsAPatrollerAndStopsWhereGuardInterruptsIt` | post is (10,10) after a patrolling step, want the actor's cell (11,10) |
| P5 | `armPatrol`'s deferred turn cancel | `TestAPatrollerEndsAnUnownedTurnUnlessItIsCasting` | the turn remaining is 2 after the ring issued a destination the actor did not already hold, want 0 |
| P6 | `armGuard`'s block scan | `TestAPatrollerFightsWhatStandsInsideItsBlock` | the patroller holds victim 0 (present false), want entity 2 — it walked past a hostile three cells from its post |
| P7 | `armGuard`'s zero-post anchor | `TestGuardAnchorsAZeroPostOnItsOwnCell` | post is (0,0) after the first guard pass, want the actor's own cell (10,10) |
| P8 | `armGuard`'s walk home | `TestAPatrollersPursuitBreaksOffWhenTheTargetLeavesTheBlock` | the patroller holds destination (20,10) present true, want the post (10,10) |
| P9 | `armGuard`'s standing acquisition at home | `TestAGuardAtHomeDropsAVictimThatLeftItsBlock` | a guard standing on its post kept victim 2, ten cells away |
| P10 | `armGuard`'s three refusals | `TestAnOffMapGuardIsLeftInEveryField` | an off-map guard was changed by its own arm |
| P11 | the dispatch gate `actorLayerDecides` | `TestASettledOrderPointsAtTheCellItSettledFor` (`approach_test.go`, pre-existing) | the unit is … — an ungated arm decides the whole map a second time |
| P12 | `clearTurnUnlessCasting`'s casting exception | `TestAPatrollerEndsAnUnownedTurnUnlessItIsCasting` | the turn remaining is 0 …, want 2 |
| P13 | `savedGuard`'s reuse of `postEngage` | `TestTheSavedGuardScansTheSameBlock` | the saved guard holds victim 0 (present false), want engaging=true |
| P14 | `cmdGroupPatrol` writes the post | `TestPatrolCommandAnchorsThePost` | post is (5,5), want the cell the member stands on (12,13) |
| P15 | `cmdGroupAttack` writes state 3 | `TestAScriptAttackSurvivesTheNextActorPass` | member 2 is in actor state 11, want engage (3) |
| P16 | `commandPatrol` writes the post | `TestPatrolCommandAnchorsThePost` | post is (5,5), want the cell the member stands on (12,13) |
| P17 | the player attack order writes state 3 | `TestPickup1117ReplacementOrdersWinBeforeCompletion` | replacement lost to completion: state=11 want 3 |
| P18 | `actorStateEngage` joins the defined set | `TestAWorldMidCycleAdvancesTheSameFromItsBytes` | UnmarshalBinary: sim: entity record 0: actor state is 3, which is not one of guard (11), … |
| P19 | the latch is read before `clearOrder` | `TestAPatrollersWalkHomeSurvivesTheNextPass` | the post moved to (12,10) on the entry after a break-off, want the post it broke off from (10,10) — the leash is post-relative and this makes it actor-relative |

Two edits were reported without a witness on the first pass and are now
covered: `savedGuard`'s shared body (P13) and the script attack's state write
(P15, which only the paired release tests had caught). Two more were
compile-witnessed only and were re-cut as behaviour mutations (P5, P12).

### What the hash pin can and cannot say

`TestHashIsPinned`'s `pinDigest` is `0x8a439cba352d8ae0` before and after this
story, and it could not have been anything else: `pinWorld` is constructed and
never stepped, and `Hash()` is FNV-1a over `w.encode()`, so the pin measures
the encoder and not the rules. This story adds no field to the byte form and
changes no encoder. **The pin is therefore incapable of falsifying this
change** and its stillness is not evidence that hashed state held still — it
did not. Real hashed state moves: the mission-80 census differs in four
entities' end cells, step counts and health, and `PostX/PostY` now moves on
every patrol command and at every patroller's entry. The instrument that can
see this story is the campaign drive, and it is the one reported above.

`formatVersion` stays 85. That has a compatibility consequence of its own, in
Open debt below and in DIV-987.

### Gates

Run from this worktree on the exact candidate commit, with `AGAINROM_IMPL`
pointed at it and `AGAINROM_MILESTONE_DRIVE` at a `missionrun` built from it.

| Gate | Result |
|---|---|
| `gofmt -l .` | clean |
| `go test -trimpath -count=1 ./...` | clean, 49 `ok`, 0 FAIL |
| `go vet ./...` | clean except eleven pre-existing `BookSpell` unkeyed-field warnings in files this branch does not touch |
| `scripts/check-no-game-assets.sh` | `clean (tree scan)` |
| `pipeline/check-release-tests.sh` EN + RU | 8 packages, 208 gated tests, 2 roots; `ok (208 of 208 ran, 0 lacked a subject)` on EN and again on RU |
| `pipeline/check-milestone.sh` | exit 0, `ok the script gap and the drive are where they were recorded, both roots` |
| `pipeline/check-div-claims.sh` | exit 0, `selected 406 live row(s) of 406 (0 closed)` |
| `pipeline/check-scenarios.sh` EN and RU | 49 of 50 on each root, and the identical 49 of 50 on the base commit on each root |
| missions 10 and 20, `-trace -ticks 1`, EN | 0 and 0 UNSUPPORTED nodes, unchanged from `pipeline/milestone-baseline.txt` |

**The one scenario failure is pre-existing and it is not code.** Measured four
ways — base and candidate, EN and RU — `1119-quick-spells.json` fails
identically every time, at `headless select entity 35: frame pixel (304,559)
has no window pixel at this placement`. Nothing in this story reaches the front
end. A checkout that lacks `review/headless-saves/` and a seeded `saves/`
fails six more at `headless activate "666 - mission 20 [original]": no row on
screen load`; those six pass here because the seat supplies both, and they are
environment rather than code (measured in this story's adversarial pass, in
scratch clones).

`check-div-claims.sh` lists DIV-985, DIV-986, DIV-989 and DIV-990 in its
advisory for citing claims that carry a retraction row. Each was read with
`go run ./tools/claim`: `AI-POST-042`'s retracted clauses are the writer-set
count, the fixed-for-the-mission clause and the order-1 initialiser;
`AI-GUARD-012`'s are the Roam and no-wander clauses, with the post, the leash
and both walk-back stores explicitly retained; `AI-SCRIPTATTACK-120`'s are its
headline's universality and the dispatcher-completeness clause, with the
helper's field writes explicitly untouched. None of the three retracted
clauses is one these rows lean on.

## Open debt

**0099 D-3 closes.** The guard post is modelled and now has a reader; the
re-anchor latch is modelled as behaviour rather than as a field. What remains
of D-3's own wording is only its second half — the record, the byte form and
the digest still carry no latch — and that is DIV-985 rather than a story seam.

**0099 D-10 is overturned in part.** Actor state 3 has two writers and a reader
that cares about it. It still has no arm of its own.

**A save this build writes is refused by the previous build at the same
version** (DIV-987). `actorStateDefined` widened to accept 3 while
`formatVersion` stayed 85, so one version number names two readers. The
decision here is to leave the version at 85: a move to 86 would not let a
pre-1141 build read such a save, it would only change which field the refusal
names, and it costs a tenth dedicated fast path in `upgrade.go`, an
`upgradeSteps` row and a hash-pin move. The direction this project's format
guarantee runs — old saves in a new build — is untouched, and no save written
before this story can hold the value. DIV-987 names the affected population
exactly and the condition that would revisit it.

**A player unit that casts while retreating or completing a pickup now walks
home to its spawn cell** (DIV-988). `stepWorld`'s `KindCast` and `KindCastAt`
call `commandGroup(..., orderNone, ...)` and write no actor state, and
`commandGroup` leaves the unit in guard; from this story the guard arm decides
it every tick. No shipped map reaches it: an instrumented build reports 0
entities entering the new dispatch case over missions 10, 20, 50, 60, 80, 90,
100, 131 and 141 at 1500 ticks each.

**The two writers of actor state 3 disagree with each other and with
`AI-CMD-054`** (DIV-989). The script side writes it even where `orderAttack`
refused; the player side writes it only on acceptance, implements none of the
claim's `0xc` veto branch, and also writes it for `KindAttackStructure`, which
no cited claim covers. Nothing observable follows while state 3 reaches no arm.

**The block picker has no corpse preference** where `acquireStanding`'s
`actorCandidates` parks bodies behind the living, so a body whose health sits
between -10 and 0 with a lower index can be taken ahead of a living hostile.
Recorded inside DIV-986, which already holds the picker's ordering as Unknown;
`AI-GUARD-012` gives the filter and no corpse ordering, so no claim is
contradicted.

**The order-0 gate is on the guard case alone, and that is not an invariant**
(DIV-990). A script `Group Command: Guard` against a group an earlier node put
on patrol raises the group's order to 1 without rewriting member states, so
patrol-state members still run their arm — and now its block scan — under order
1, against `AI-ORDER-010`. The ungated `armPatrol` predates this story; the
scan is new. `guardarm.go`'s own doc comment stated the invariant and has been
corrected. Not reached on shipped data: all fourteen entities that run the
patrol arm over the eight patrol maps run it under group order 0.

**No drive in this tree covers the leash.** The eight patrol-mission censuses
are identical with and without the F-1 correction, because a headless census
issues no orders and no unit walks out of a patroller's block in one. The input
that reaches the leash is a player kiting a patroller, which no drive produces,
so the two new witnesses are the only instrument that sees it.

New divergence rows: DIV-985 (the derived re-anchor latch), DIV-986 (the
block's traversal order and its corpse ordering), DIV-987 (the widened actor
state set at a fixed format version), DIV-988 (a caster left guarding its spawn
cell), DIV-989 (the two writers of state 3) and DIV-990 (the order-0 gate).
None of the six reserved numbers is retired.

Not addressed, and out of this story's scope: the group-order AI in
`engage.go`, the escort and follow arms, relation loading in `pkg/mapload`, and
any arm for actor state 3 itself.
