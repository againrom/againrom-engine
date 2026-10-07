# round2-orders.md — fix round 2, orders and AI slice

Branch `story/1001-r2-orders`, based on `5cee2da22d879246fb38752c95c244b911872cf3`. This document
covers review findings A1, A5, G1, G2, G3 and the order-related test reversals listed under F4. The
seat folds its proposed spec sentences and ledger rows into the canonical documents at the landing;
this lane did not edit `spec.md`, `closure.md` or `docs/DIVERGENCES.md`.

All commands below are run from the worktree root with `GOCACHE=<seat>/.gocache-impl1001`.

## What was wrong

The story introduced `actorActionBusy` as a single admission guard and applied it to movement orders
and to every attack command. The predicate returned true for an actor merely approaching an admitted
victim (`HasAttackTarget && HasTarget`) and for most of the melee cycle
(`AttackPhase != AttackReady || AttackCountdown != 0`), so the following orders were dropped without
effect:

- `pkg/sim/step.go:416`, `case KindMoveTo` — a fighting or approaching unit ignored a move order.
  `0064` FR-2 states that a move order ends the fight.
- `pkg/sim/group.go:124`, `groupOrder` — a busy member never entered `members`, so `KindGroupMoveTo`,
  `KindGroupSwarmTo`, `KindGroupStance` and `KindGroupPatrolTo` all skipped it.
- `pkg/sim/group.go:245-252`, `issueGroupDestination` — a second filter, applied before the centroid
  and the formation rate were computed, so the surviving members were sent to different cells and at
  a different speed than the same order produced on master. The same body serves the script's group
  sub-commands 4 and 5 through `cmdGroupCommandedMove`, so a shipped trigger's march left its
  fighting members standing.
- `pkg/sim/engage.go:1315-1322`, `orderAttack` — every actor, not only a caster, refused a retarget.
  A player's click on a different enemy did nothing, and the group decision's own mid-pursuit
  rescoring was dead.
- `pkg/sim/engage.go:1301-1310`, `orderAttack` — a weapon-spell carrier returned before writing
  anything when its own sight march did not reach the target, while a plain fighter took the order
  and approached (finding A5).
- `pkg/sim/engage.go:196-204`, `engagementPassObserved` — a member that began a cast was removed from
  `g.members` before `w.decide(g)`. `decide` reads that list for the guard centroid and `candidates`
  reads it for the decider and for the shared group sight stamp, so the group lost that member's
  vision for the same pass. A group whose every member cast was not decided for at all (finding G1).

Each of the five reversed test assertions listed in the review recorded one of these behaviours as
intended. None of them is in `contract.md` or `spec.md`. `contract.md:40` lists the guard's sharers
as "attack commands, weapon-spell attacks, manual casts, AI casts, offensive autocast, and idle
Heal"; movement is not among them.

## What changed

### Two predicates instead of one

`actorActionBusy` (`pkg/sim/spell.go`) is now the cast-admission guard alone. It answers "may a new
spell begin on this actor", and returns true when a physical or weapon attack cycle is loaded or when
a cast owns the actor. Its two consumers, `bookSpellRefusal` and `beginBookSpellAt`, are unchanged.
The approach arm is gone: non-overlap between the attack cycle and a cast is already total, because
the attack advance and the mover both stand down for an actor carrying a cast (`step.go`), and this
predicate refuses a cast for an actor whose cycle is loaded.

`actorCastBusy` (`pkg/sim/actionguard.go`, new) is the narrower predicate: a pending book cast, or a
non-zero `CastWait`. It is what a competing **attack** order consults.

Movement consults neither.

The resulting matrix, for one actor:

| new action | while its attack cycle is loaded | while a cast owns it |
|---|---|---|
| movement order (player, group, script, AI) | admitted; ends the fight | admitted; attached, walked after release |
| attack order (physical or weapon-spell) | admitted; a different victim resets the cycle | refused |
| cast (manual, autocast, idle Heal, AI) | refused | refused |

### The decision about a cast in progress

**A pending cast is not interruptible, and a movement order is not dropped.** The order attaches on
the tick it arrives — destination written, fight ended, command group allocated — and the mover
declines to advance a body while its cast winds up, so the walk begins when the cast releases. The
cast then applies or refuses on its own terms. Nothing is spent and nothing is refunded by the move.

The alternative was to cancel the pending cast without paying. It was rejected because it adds a
second removal path for `bookCasts` and a refund question, for the same visible outcome delayed by
the wind-up, which is floored at eight ticks. What ROM1 does is not established: `AI-ORDER-039`
decodes the per-actor order machine `ord+0x08` and gives arm 8 as "cast `ord+0x30` at the actor
`ord+0x28`, `actor+0x54 = 0xd`", but the claim is about the AI's own order byte and says nothing
about what writing a new order does to an actor already at `actor+0x54 = 0xd`. This is proposed as a
ledger row rather than treated as settled.

### A5, the weapon-spell attack order

The perception gate is removed from `orderAttack`. An attack order on a weapon-spell carrier is an
ordinary attack order. Actor-local perception stays at the weapon cast's own release
(`pkg/sim/spell.go:898`), which returns without applying, spending or training when the actor cannot
currently see the target. `spec.md:67-70` scopes actor-local perception to cast admission and
release; an attack order is neither.

### G1, the group decision

`engagementPassObserved` runs `aiCast` over every member and then calls `decide(g)` with the member
list intact. A member that began a cast is protected by `orderAttack`'s own cast guard instead, and a
destination written to it stands until the cast releases.

### A commanded autocaster walks: the cast owns the body through wind-up, not through recovery

Attaching the destination is not obeying the order if the actor never takes a step, and it did not.

`step.go`'s mover stood down for an actor with a pending cast **or** a non-zero `CastWait`. An armed
offensive row with a target in range re-arms on the tick recovery reaches zero, and `stepAutoCasts`
runs before the mover in the tick, so the actor never got a tick in which to walk. Measured on a
synthetic fixture — a mage at (2,2), `AttackCharge 12`, `AttackRelax 4`, Fire Arrow armed, a hostile
at (4,4) it cannot kill, ordered to (2,9), stepped 201 ticks:

| revision | armed | disarmed |
|---|---|---|
| `b51b439` (master) | walks; there was no wind-up or recovery at all | walks |
| `5cee2da` | ends at (2,3): one cell in 201 ticks | ends at (2,9) |
| this branch, before this fix | ends at (2,2): no cell in 201 ticks | ends at (2,9) |
| this branch | arrives at (2,9) on tick 29, 36 mana spent | arrives on tick 5 |

The mover now stands down for the wind-up alone. `spec.md`'s own wording supports it: release "enters
the actor's equipped recovery interval" and "a later action may begin only after simulation
recovery" — a statement about admission, which `actorActionBusy` makes, and not about the walk. The
caster stands while it channels, which is the wind-up, and walks while it recovers.

The owner named this symptom in his ruling of 2026-08-15 deferring the route-following Teleport
autocast: «сейчас маг просто стоит на месте и тратит ману впустую». Removing the Teleport row from
autocast, which the point lane is doing, does not reach it: any armed offensive row with a target in
range produced the same standstill.

Witnessed by `TestACommandedAutocasterWalksAndShoots`, which fails at `5cee2da` and against this
branch's own first commit.

### A movement order and an armed autocast coexist

`autoCastOrder` (`spell.go:1402`) gates only the unbidden, unarmed Heal on `underCommand`; an armed
row is appended unconditionally. With the movement arm's `claim` removed, the sweep reaches a
commanded actor on the tick its order arrives and on every tick after it, the release does not
consume the order, and a place-targeted row (`KindCastAt`, `beginBookSpellAt`) is admitted on the
same terms as a unit-targeted one. Witnessed by
`TestAnArmedAutocastReachesAMovingActorOnTheTickItIsOrdered`.

The owner confirmed walking-and-casting as correct default behaviour for the offensive rows and for
idle Heal. The route-following Teleport autocast is deferred and none of it is built here.

### G3, the AI arms

`armSwarm` reaches `orderAttack`, which now refuses a casting actor, so the AI cannot retarget one.
`escortClose` writes a destination and clears the attack directly; that write is now exactly the
write a player's move order performs, and it does not cancel the cast. The asymmetry the finding
names — the AI able to interrupt an action the player cannot — is removed in both directions.

### The movement order no longer claims its actor against the autocast sweep

`stepWorld`'s `KindMoveTo` arm called `claim(e.ID)`, which put the actor into the set the autocast
sweep skips. That made "this unit has somewhere to go" mean "this unit does not cast" for the tick
the order arrived on, and for every tick for a client that emits a move command while the button is
held. The call is removed.

Nothing is lost by removing it. The guarantee the story's own comment gave for it — an explicit move
keeps idle Heal from detaching the order — is kept by `underCommand` (`engage.go:269`,
`HasTarget && !HasAttackTarget`), which `autoCastOrder` (`spell.go:1402`) already consults for the
unbidden, unarmed Heal alone. An **armed** autocast is the player's own instruction and is meant to
reach a moving unit.

This is also what the owner's Teleport-autocast ruling of 2026-08-15 requires; see the section on it
below.

### Two claim sets in `stepWorld`

`stepWorld` used one map for two purposes: the casts that released this tick, which suppress the
autocast sweep **and** hold recovery still for that tick, and the actors an explicit order claimed,
which suppress only the sweep. `step.go` now keeps `released` and `commanded` apart, and
`unionOfClaims` (`actionguard.go`) builds the sweep's skip set without copying either map on the
ordinary tick.

With the movement arm no longer claiming, no arm can claim a recovering actor: both remaining
claimers run after a guard that refuses a non-zero `CastWait`, so the recovery freeze the split
prevents is currently unreachable. The split is kept because `ageCastRecovery`'s argument is meant to
be the set of casts that released and not the set of actors somebody gave an order to. The Teleport
story will add a producer where a movement order does begin a cast, and it should not have to
re-derive the distinction.

## Witnesses

Every test below fails at `5cee2da` and passes on this branch, except the two marked as guards. The
falsification was run by checking out `5cee2da`'s `pkg/sim/{spell,step,group,engage}.go` over this
branch's test files, moving `pkg/sim/actionguard.go` aside, and running the package. Fourteen tests
failed; the tree was then restored and the package is green.

Reproduce the whole set:

```
go test -trimpath -count=1 ./pkg/sim/
```

Reproduce the falsification:

```
git checkout 5cee2da -- pkg/sim/spell.go pkg/sim/step.go pkg/sim/group.go pkg/sim/engage.go
mv pkg/sim/actionguard.go /tmp/actionguard.go.bak
go test -trimpath -count=1 ./pkg/sim/
git checkout HEAD -- pkg/sim/spell.go pkg/sim/step.go pkg/sim/group.go pkg/sim/engage.go
mv /tmp/actionguard.go.bak pkg/sim/actionguard.go
```

| Finding | Test | File |
|---|---|---|
| A1, single move order | `TestAMoveOrderStillEndsAnApproach` (restored) | `pkg/sim/pursuit_test.go:141` |
| A1, single move order | `TestWalkingAndAttackingAreOneState` (restored) | `pkg/sim/combat_test.go:250` |
| A1, group move | `TestAGroupMoveOrderEndsAFight` (restored) | `pkg/sim/combat_test.go:262` |
| A1, group membership and centroid | `TestAGroupMoveTakesEveryAddressedMemberAndKeepsTheCentroid` | `pkg/sim/orderguard1001_test.go` |
| A1, Guard/Stand-Ground and Patrol | `TestGroupStanceAndPatrolReachAFightingMember` | `pkg/sim/orderguard1001_test.go` |
| A1, the shipped script's march | `TestAScriptedGroupMarchTakesItsFightingMembers` | `pkg/sim/orderguard1001_test.go` |
| A1, retarget and the cycle reset | `TestReIssuingAnOrderDoesNotResetTheCycle` (restored) | `pkg/sim/combat_test.go:705` |
| A1, the AI's mid-pursuit rescoring | `TestACommandedFighterStillDecidesForItsGroup` (restored) | `pkg/sim/commanded_test.go:104` |
| cast contention, and the move decision | `TestACastOwnsItsActorWhileAMoveOrderStillLands` | `pkg/sim/orderguard1001_test.go` |
| cast contention, the attack-cycle side | `TestAnAttackCycleRefusesACastAndAdmitsAnotherAttackOrder` | `pkg/sim/orderguard1001_test.go` |
| A5 | `TestTargetDirectedWeaponSpellUsesVisibilityAtItsRelease` | `pkg/sim/spell_visibility_test.go` |
| G1 | `TestACastingMemberStillSuppliesItsGroupsSight` | `pkg/sim/orderguard1001_test.go` |
| a commanded actor is still reached by its armed autocast | `TestAnArmedAutocastReachesAMovingActorOnTheTickItIsOrdered` | `pkg/sim/orderguard1001_test.go` |
| a commanded autocaster actually walks | `TestACommandedAutocasterWalksAndShoots` | `pkg/sim/orderguard1001_test.go` |
| G3 (guard; passes at `5cee2da`) | `TestTheAIArmsCannotInterruptACastTheyDoNotOwn` | `pkg/sim/orderguard1001_test.go` |
| recovery freeze (guard; passes at `5cee2da`) | `TestAMoveOrderDoesNotHoldCastRecoveryStill` | `pkg/sim/orderguard1001_test.go` |

`TestTargetDirectedWeaponSpellUsesVisibilityAtAdmissionAndRelease` was renamed and rewritten. Its
first half asserted that the attack order gained no side effect on a hidden target, which is the A5
defect stated as an assertion. It now asserts that the order is taken, that the release lands nothing
while the target stays hidden, and that the same action releases once the target becomes visible.
That file was not in this lane's list; it is the only test outside the list that this lane touched,
and it was touched because it recorded the reversal.

## Campaign sweep

`bash scripts/campaign-sweep.sh` over all 28 EN campaign maps, `AGAINROM_ASSETS` at the preserved EN
root, 2000 ticks:

```
AGAINROM_ASSETS=<againrom>/gameversions/en bash scripts/campaign-sweep.sh
```

Every map loads and drives on both revisions. The script-gap census is **59 unsupported / 7375
reached** on both, unchanged. Ignoring the hash column, which necessarily differs because the story
adds hashed state, exactly one row differs between `b51b439` (master) and this branch:

| mission | `b51b439` alive/fallen | `5cee2da` alive/fallen | this branch alive/fallen |
|---|---|---|---|
| 80 | 106 / 1 | 105 / 2 | **106 / 1** |
| 90 | 153 / 1 | 150 / 4 | 150 / 4 |

Mission 80 is restored to master's figure. Mission 90 is unchanged from `5cee2da`.

Mission 90's cause was measured by ablation rather than inferred. With the body of
`engagementPassObserved`'s `aiCast` loop made unreachable and nothing else changed, mission 90
returns to `153 / 1` and mission 80 stays at `106 / 1`:

```
mission  tick  outcome     unsupported  reached  alive  fallen
     80  2000  undecided             0        0    106       1
     90  2000  undecided             3      375    153       1
```

The three additional casualties therefore come from `aiCast`, the story's new AI spell arm, and not
from the action guard. This lane did not remove them. An AI mage that can now cast offensive spells
killing three more units in an unattended 2000-tick drive is the new arm acting, and
`scripts/campaign-sweep.sh`'s own header states that the outcome columns are an instrument and not a
verdict. What is a defect is that the arm has no authority: `spec.md` contains no sentence about it,
`contract.md` names "AI casts" only as a sharer of the admission guard, and the story folder cites no
claim for its filter, its uniform choice or its Mind-59 hand-back. A proposed spec sentence and a
ledger row are below.

Mission 10 and mission 20 unsupported script nodes, `cmd/missionrun -trace -ticks 1`, EN root:
**0 and 0**, on this branch and on `b51b439` alike. `pipeline/milestone-baseline.txt` records no
per-mission unsupported count, so the sweep's own census column is quoted above instead. This story's
slice was not expected to move that census and does not.

## Proposed `spec.md` sentences

Ready to paste. The seat decides placement.

Under "Target selection and autocast", replacing the paragraph that ends "Action admission remains
authoritative, so this precedence cannot overlap a staff attack, a weapon rider or another cast":

> The actor-action admission guard governs the start of a spell. A manual cast, an armed autocast, an
> idle Heal, an AI cast and a weapon-borne release cannot begin while another cast owns the actor or
> while a physical or weapon attack cycle is loaded. A competing attack order is refused only while a
> cast owns the actor; against another attack order the established cycle rule applies, so a
> different victim resets the cycle and the same victim keeps its timeline.

New paragraph in the same section:

> A movement order is never refused for busyness. A move from the player, from a group command, from
> a script or from the AI ends whatever fight the ordered unit was in and attaches its destination,
> whether the unit is approaching, charging, relaxing or carrying a book cast. A pending cast is not
> interruptible: the body does not advance while the cast winds up, the cast then applies or refuses
> on its own terms, and the walk begins. The move spends nothing and refunds nothing.

Added to the same new paragraph, or standing beside it:

> A movement order does not suppress unbidden casting on the actor it orders. An armed autocast is
> the player's own instruction and reaches a unit that is walking, on the tick the order arrives and
> on every tick after it. The unbidden, unarmed Heal is the one row a movement order does keep off
> the actor, because a unit holding a destination and no victim is under command.

New paragraph in the same section:

> A group order addresses every living member the command names. The centroid, the formation test,
> the per-member offsets and the group rate term are computed over that addressed set. A member that
> is fighting, approaching, charging, relaxing or casting is neither dropped from the set nor
> excluded from the computation. The same rule holds for the script's own group march.

Replacing the last sentence of the "Current perception" paragraph at `spec.md:67-70`:

> An attack order on a weapon-spell carrier is an ordinary attack order: the victim is written and
> the approach begins on the same terms as for a plain fighter. Actor-local perception gates the
> weapon-borne release, not the order. A release whose target the actor cannot currently see applies
> nothing, spends nothing and leaves the attack cycle to turn over.

New paragraph covering the group decision (G1):

> A member that begins a cast on a decision pass stays in its group's member list. The group's shared
> sight stamp, its guard centroid and its notice circle are computed over every member the group
> holds, and a group in which every member casts is still decided for.

New paragraph covering the AI spell arm, which `spec.md` does not describe at all:

> A map-owned mage may begin one cast per decision pass. The candidate rows are those it knows, that
> are applicable, that are not defensive and whose mana cost it can pay; one is chosen uniformly and
> aimed by the ordinary automatic target selection. A caster of Mind above 59 hands the decision back
> to the ordinary group arm on about three passes in ten. The cast is admitted through the same guard
> as every other, so it cannot overlap an attack cycle or another cast. The player's own and
> unowned actors do not use this arm.

## Proposed `docs/DIVERGENCES.md` rows

Column order is the ledger's own:
`ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status`.

The ledger stands at `DIV-027`. Three lanes are proposing rows in parallel and this lane was not
allocated ids, so the ids below are placeholders for the seat to assign.

| DIV-0xx | sim / orders during a cast | Spells must have their effects, and a unit must obey a click | `AI-ORDER-039` decodes the per-actor order machine `ord+0x08` and gives arm 8 as "cast `ord+0x30` at the actor `ord+0x28`, `actor+0x54 = 0xd`". No claim states what writing a new order arm does to an actor already at `actor+0x54 = 0xd`, so whether a movement order aborts a cast in progress is undecoded | A movement order is always admitted and always ends the fight. It does not cancel a pending book cast: the destination is attached, the body does not advance until the cast releases, and the walk begins then. Nothing is spent and nothing is refunded | UNKNOWN | Authored at `1001` fix round 2. The alternative — the move cancels the pending cast without paying — needs a second removal path for the cast record and a refund rule, for the same visible outcome delayed by the wind-up floor of eight ticks | A claim naming what a new order does to an actor mid-cast | OPEN |
| DIV-0xx | sim / AI spell arm | Spells must have their effects, including for the AI's own mages | No claim is cited anywhere in `docs/1001-spell-effects/` for the AI's spell decision. `AI-ORDER-039` gives the order machine's cast arms 8 and 9 but not the choice that fills them | A map-owned mage begins at most one cast per decision pass, chosen uniformly among the known, applicable, non-defensive rows it can afford, aimed by the ordinary automatic target selection, with a hand-back of about three passes in ten above Mind 59 | UNKNOWN | Built at `1001` with no cited authority and no `spec.md` sentence. Measurable on shipped data: mission 90 drives to 150 alive / 4 fallen with the arm and 153 / 1 without it | A claim naming the AI's spell selection | OPEN |
| DIV-0xx | sim / body during cast recovery | A unit must obey a click; «сейчас маг просто стоит на месте и тратит ману впустую» (2026-08-15) | No claim states whether the original's actor advances during a cast's recovery interval | The mover stands down for a pending cast's wind-up alone. Recovery gates the next action, through the admission guard, and not the walk, so a commanded mage with an armed row walks and casts | UNKNOWN | Authored at `1001` fix round 2. Standing down for recovery as well left a commanded autocaster covering no ground in 201 ticks, because the sweep re-arms before the mover runs | A claim naming what the original's actor does with its walk during cast recovery | OPEN |
| DIV-0xx | sim / weapon-spell attack order | A staff-armed hero must obey a click on an enemy the player can see | `spec.md`'s actor-local perception is the story's own authored seam; no claim states that an attack ORDER on a weapon-spell carrier is refused for the carrier's own sight | The attack order is admitted and the approach begins as for a plain fighter. Actor-local perception gates the weapon-borne release alone | DEVIATION | Authored at `1001` fix round 2. Gating the order made a staff-armed hero ignore a click on any enemy the player could see through another unit, which the plain fighter obeys | A claim on the original's attack-order admission for a spell-bearing weapon | OPEN |

## Gate

Run on the clean tree at commit `b8cfc70`:

```
go build ./...
go vet ./...
gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')
go test -trimpath -count=1 ./...
PATH="/usr/bin:/bin:$(dirname "$(command -v git)")" bash scripts/check-no-game-assets.sh
```

All green; `gofmt` printed nothing; `check-no-game-assets` printed `clean (tree scan)`.

## Not done in this lane

- Mission 90's three extra casualties are `aiCast`'s, and the arm was left in place. Removing or
  re-deriving it is a scope and authority question, not an order-guard defect.
- `pkg/sim/binary.go`'s `formatVersion` comment. The seat writes that paragraph at the landing.
- Every finding outside A1, A5, G1, G2, G3 and the order-related F4 reversals. In particular
  `pkg/sim/spell_test.go:384`, `pkg/render/terrain/overlay_test.go:1108`, `pkg/ui/fogindicators_test.go:32`,
  `pkg/sim/autocast_test.go:114`, `pkg/sim/heal_test.go:135`, `pkg/sim/weaponspell_test.go:660`,
  `pkg/sim/skill_test.go:514` and `scenarios/0154-synthetic-spells.json` were left untouched.
