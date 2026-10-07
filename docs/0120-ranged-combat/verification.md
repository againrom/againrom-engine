# Verification — 0120-ranged-combat

Branch `0120-ranged-combat` off master `67d2e57`. Six commits: stages 1–3, T1, T2, T3, T4 and
two untrailered formatting repairs.

## What the gate printed

Run from the lane worktree on the committed tree, `git status --porcelain` empty:

```
go build ./...                      clean
go vet ./...                        clean
gofmt -l $(git ls-files '*.go')     nothing
go test -trimpath -count=1 ./...    every package ok, exit 0
scripts/check-no-game-assets.sh     clean (tree scan)
scripts/check-doc-budget.sh         every 0120 row ok, largest 91%
scripts/check-sdd-audit.sh          no FAIL for 0120
git diff --diff-filter=D --name-only 67d2e57..HEAD    empty
git log --format='%(trailers:key=Co-Authored-By)'     empty on every commit
git diff --name-only 67d2e57..HEAD -- pkg/sim         empty
```

`gofmt` was **not** clean when the seat first ran it, twice, on files two task agents had each
reported clean. Both were whitespace only — a struct-literal alignment in T2's test file and a
trailing blank line in T4's — and both were repaired in untrailered commits. Recorded because
the pattern is the point: a task agent's gate claim was wrong twice out of four, and the seat's
own run on the committed tree is what caught it.

## The finding that reshaped the story

The brief arrived with three suspected walls. **Two of them were false**, and the truth is worth
recording because it is the opposite of what a reader would guess from the owner's report.

**Ranged combat in the simulation has been standing since `0104`.** `approach` stops the walk at
`inReach`, which is `strikeDistance <= Entity.Reach`; both target scorers already fold to
preference row 0 and rewrite the distance term the moment reach passes 1; and
`pkg/sim/reach_test.go` already carries
`TestAReachOfFourStrikesAtFourRefusesAtFiveAndStopsItsWalkAtFour`. Nothing in `pkg/sim` needed
to change and nothing did.

**Every bow in the shipped game is attack type 5**, which is below the ranged threshold of 10,
so `ResolveWeapon` was already accepting all of them. Census of the shipped `Weapons`
collection, read with this tree's own parser against a lawful install: Short Bow 5/range 4, Long
Bow 5/range 5, Crossbow 5/range 6, Sonic Beam 5/range 4, Boulder Thrower 5/range 20, and
**exactly one** ranged row in the whole table — Flame Thrower, attack type 11, range 8, carried
by the four `Dragon` classes. So the refusal at `weapon.go:203` was real code that, in shipped
data, cost one creature.

`groupScorerReach` was **not** reversed. `0104` DD-4's reason survives contact with
`AI-REACH-072`: the ceiling is the law's own reach, and a member of reach above 1 passes the
refusal anyway because the distance rewrite immediately above it has already turned that
member's distance term into 1. Widening the ceiling would be a group-behaviour story with its
own measurement, exactly as `0104` said.

## Acceptance, each witnessed by a revert

Every revert below was performed by the orchestrator seat, not taken on a task agent's report,
and the tree was confirmed clean again afterwards.

**AC-1 — a ranged row resolves.** Witnessed against the shipped table rather than only a
fixture: resolving all 26 equipment strings the shipped unit classes carry gives **26 resolved,
0 unresolved, 4 reporting `Ranged()`** — the four Dragons. Before this story the four were
errors.

**AC-2 — the fold's two arms.** Replacing `if !w.Ranged()` with `if true` in
`pkg/data/foldweapon.go`:

```
--- FAIL: TestARangedWeaponFoldsItsReachAndCadenceAndNoneOfItsDamage
    a ranged fold moved damage/to-hit/defence: got {DamageBase:105 DamageSpread:63 ToHit:47
    Defence:22 ... Reach:6}, want the bearer's own {DamageBase:5 DamageSpread:3 ToHit:7
    Defence:2 ... Reach:1}
```

**AC-3 — the creature fold adds, and does not assign.** This is the story's riskiest line: an
assignment there would replace 26 shipped classes' numbers instead of raising them and would
look entirely plausible. Turning the four `+=` into `=`:

```
--- FAIL: TestACreatureNamingAMeleeWeaponArrivesWithTheSumOfBoth
    DamageBase = 5, want 8 · DamageSpread = 4, want 10 · ToHit = 7, want 17 · Defence = 2, want 6
```

The direction is settled in `provenance.md`: the row is streamed first and the equip runs after
it, through a routine read as an addition at every site.

**AC-4 — no shipped reach moved.** `pkg/mapload/reach_test.go`'s two existing tests passed
unmodified, and T2 added a table-driven case over six row shapes asserting every reach is what
`unitReach` alone produced. The corpus agrees: the fold reaches all 26 armed classes, so the
fallback is never taken in shipped data, and the reach it assigns comes off the same range slot
`unitReach` reads.

**AC-5 — the hero holds a bow.** Witnessed **on the real install**, not only in a fixture:

```
againrom -assets <root> -check
  hero ... Blade 10, Iron Short Sword 10-16, to-hit 49, defence 8
againrom -assets <root> -skill shooting -check
  hero ... Shooting 10, Uncommon Wood Short Bow 8-13, to-hit 54, defence 8
againrom -assets <root> -skill nonsense -check
  unknown -skill "nonsense" ... (one of: blade, axe, bludgen, pike, shooting)  exit 2
```

The bow's range column is 4, and `Derive` assigns `Reach = Weapon.Range`, so that hero reaches
four cells. Reverting the setter's store:

```
--- FAIL: TestSetPartySkillGeneratesAnArmedHero
    StartWeaponErr = data: weapon "Iron Short Sword": no Weapons entry named "Iron Short Sword"
```

**AC-6 — the shot.** Relaxing the Chebyshev gate from `>= 2` to `>= 0`:

```
--- FAIL: TestAnAdjacentAttackerDrawsNoShotAtAnyReach
    an adjacent reach-3 attacker carries a Shot: (1120,1024)
```

**AC-7 — nothing hashed moved.** `git diff --name-only 67d2e57..HEAD -- pkg/sim` is empty, so
no simulated entity gained a field, the byte-form version did not move and no pinned digest was
touched. **Byte-form version 30 was allocated to this story and is returned unused.**

## What this changes in the game

The 26 shipped classes that carry equipment now carry its numbers. The four bow classes gain
their weapon's to-hit and, more visibly, its **cadence**: every bow in the table states a charge
of 20 ticks against 6 or 7 for a blade, and that number was being read and thrown away. So an
archer now draws slowly and strikes hard-ish from four or five cells, where before it inherited
its row's melee cadence.

`Goblin_Sling` and `Orc_Bow` gain between 2 and 7 damage and between 5 and 19 to-hit. `Dragon`
gains reach 8 and nothing else, which is correct: its damage belongs to a component this tree
does not build.

## Divergences, and one that is larger than it looks

**D-1, the third damage component, is the one to read.** A ranged weapon's damage does not join
the ordinary pair in the original — it feeds a third component with its own protection selector.
The offset arithmetic makes this certain rather than likely: the equip arm's destination triple
is one fixed displacement from the live triple that third component rolls, the same displacement
that separates the modifier copy from the live copy everywhere else in that block. That also
explains why the claim's own census of the selector byte found no displacement-addressed writer
— the only writer reaches it through the wholesale modifier fold the claim names as its blind
spot. Building the component was declined under the MVP rule; a ranged weapon therefore
contributes no damage, and for every shipped class that is exactly the behaviour the tree
already had.

D-2 (a ranged weapon's to-hit is not reassigned from a field this tree does not model), D-3 (no
active-skill field), D-4 (the registry's projectile class, shoot delay and shoot offset are
still unread, so the drawn mark is this project's own diagnostic design) and D-5 (a brace clause
on an equipment name is still stripped) all stand as written.

**One consequence not foreseen in the spec.** Removing the resolver's refusal also removed
`firstWeapon`'s "is this a weapon" predicate for ranged rows on the **humans** arm, which the
spec's FR-3 and FR-4 only cover for creatures. A humans-band row whose first resolving
equipment cell is ranged now equips it — reach and cadence, no damage — instead of coming out
bare. No shipped humans row carries a ranged weapon, so nothing shipped moves; two `pkg/mapload`
test cases that pinned the old behaviour were converted to assert the new one, and the two doc
blocks in `spawn.go` that stated the old reason were rewritten. Recorded here because the spec
did not name it and a reader of the spec alone would not find it.

**One refinement of a plan decision.** DD-8 said the seam would carry "a position and nothing
else", and it does: `MapEntity.Shot` is one optional point in 1/256 of a cell, with no target
id, no reach and no attack phase. DD-10 said the new colour would sit beside the existing
marker colours; those live in `pkg/render/terrain`, which is not in T4's file list, so it sits
beside the marquee colour in `pkg/ui/viewer.go` instead — the same kind of declaration, one
package over.

## The derived properties and the success criteria

**P-1** — the fold assigns reach and adds damage, so folding twice would double the damage and
leave the reach right. Nothing folds twice: the person path folds once inside `Recompute`, the
creature path once inside `definitionFor`, and both write into a fresh `Combat`. Stated so the
next caller knows which half it would corrupt.

**P-2** — reach is never 0. Witnessed by a test that folds a weapon whose range cell is the
empty sentinel and gets reach 1, and by the corpus above, where the lowest range any shipped
armed class resolves to is 1.

**P-3** — FR-4 held without a special case, and the corpus is why: the fold and the old
range-only read take the same slot the same way, so removing the refusal changed only whether
the resolution returned. Nothing had to be excepted.

**P-4** — a world with no reach above 1 draws no shot at all, so every existing mission looks
exactly as it did. Witnessed by its own test and by mission 1 having no ranged hostile.

**SC-1** — AC-1 to AC-6 each hold and each was reverted by the seat, above. **SC-2** — the gate
is green on the committed tree and the three repo scripts pass with no new FAIL. **SC-3** —
`git diff --name-only 67d2e57..HEAD -- pkg/sim` is empty, so the version literal and every
pinned digest are byte-identical to master. **SC-4** — the deletion set is empty.

## What was not built

- **The third damage component** (D-1). It is a story of its own: three components, a five-way
  protection selector and a writer switch, all decoded, all reaching hashed state.
- **A real projectile.** The mark drawn between shooter and victim is a diagnostic square on the
  attacker's own charge clock. The registry names a projectile class, a shoot delay and a
  per-direction shoot offset for every unit class; none is read, and resolving them is what
  would turn the mark into an arrow.
- **Withdrawal.** Every class that withdraws is ranged, so the two populations are the same, but
  nothing here touches it and it remains absent from the tree.
- **Anything in `pkg/sim`.** Deliberate, and it is what kept the byte form still.
- **No `builds/` folder and no owner-review artifact**, per the standing rule that an
  implementation story's deliverable is the build.
