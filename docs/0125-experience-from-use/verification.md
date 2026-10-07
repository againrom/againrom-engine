# 0125 — experience from use: verification

Branch `0125-experience-from-use`, off `1039eea`, merged with `origin/master` three times while
it ran — at `31e95cf`, `b267c4a` and finally at **`ffc2522`**, before the final gate. Five task
commits, three verification-stage commits, no deletions.

## The gate, on the committed tree

| Check | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `gofmt -l $(git ls-files '*.go')` | no output |
| `go test -trimpath -count=1 ./...` | exit 0, 32 packages `ok`, no skips |
| `scripts/check-no-game-assets.sh` | exit 0 |
| `scripts/check-doc-budget.sh` | exit 0 |
| `scripts/check-sdd-audit.sh` | no FAIL for this story |
| `git diff --diff-filter=D --name-only ffc2522..HEAD` | empty |
| `git log --format='%(trailers:key=Co-Authored-By)'` over the branch | empty at every commit |

Trailers are a bijection with `tasks.md`: T1 `bf5b1de`, T2 `86eac67`, T3 `c525b6e`, T4
`c6c662e`, T5 `777927c`. The verification commit carries no trailer and touches no production
line.

## Witness by reverting — including the three it caught

The rule is that a line is witnessed when reverting it makes a test go red **on a value**, and
that reading an assertion proves nothing. Every clause below was probed by patching the line,
running the suite, and restoring it.

| Reverted | Result | Test that fired |
|---|---|---|
| FR-7 — the whole Mind term, `xpGain` returns the raw amount | **RED** | `TestFR7MindScalesTheGainToTheseExactNumbers`, `TestFR7TwoAttackersDifferingOnlyInMindEarnDifferentAmounts` |
| FR-6 — the `+ 1` per blow | RED | `TestFR6PaysHalfPlusOneAndSplitsAcrossBlows`, `TestXPRawMatchesTheRealValuedForm`, and two more |
| FR-6 — the halving, `2 * maxHP` becomes `maxHP` | RED | the same two |
| FR-5.2 — `resolveBlow`'s liveness read becomes a bare `true` | **RED** | `TestResolveBlowReadsTheLivenessBeforeItSubtracts` |
| FR-5.3 — the human-only refusal | RED | `TestPayExperienceRefusesAClassThatDoesNotGain` |
| FR-5.4 — the same-owner refusal | RED | `TestPayExperienceRefusesTheSameOwnerSlotAndALockedRelation` |
| FR-5.5 — the locked-relation refusal | RED | the same |
| FR-10 — the inverse shifted by one | RED | `TestP1SkillLevelForIsTheExactInverse` |
| FR-15 — the row prints five numbers instead of six | RED | `TestPanelStatesTheSkillRowForOneSlotAndForNone`, `TestAPanelStatesACharacterInTheDecodedOrder` |
| FR-17 — the live overlay disabled | RED | `TestTheReadoutsSkillNumberRisesOnABlowAndHoldsWithoutOne` |

**Three of these were GREEN when the probe first ran, and that is the finding of this stage.**

*FR-7 was not witnessed at all.* Deleting the entire Mind multiplier left `pkg/sim` green.
Every payment test asserted that the credited slot **holds more than zero** — a flag — and a
flag survives the deletion of the term that decides how much more. P-3's sweep survived it too,
by construction: it counts divergences from the real-valued form and fails only when it finds
**none**, so an arithmetic that diverges everywhere passes it. Two tests now pin the scaling to
numbers written out by hand rather than recomputed from the expression under test.

*FR-5.2's caller-side read was not witnessed.* The guard itself was exercised by handing
`payExperience` an `aliveBefore` of `false`, but nothing checked that `resolveBlow` computes
that argument from the health the target held **before** its own subtraction. The reason is
worth keeping: every path into a blow refuses a dead victim one phase earlier, so the state is
currently **unreachable through a tick**. The new test says so and calls `resolveBlow`
directly; what it pins is the wiring, for the story that later makes it reachable.

## What witnesses each acceptance criterion

| Id | Witness |
|---|---|
| AC-1 | `TestAGainingAttackerEarnsInExactlyOneSlot` — one slot up, the other five untouched, and the target's own array unmoved |
| AC-2 | `TestAnEntitysExperienceRoundTripsByteIdentically`, `TestEachSkillSlotMovesTheDigestAndNoneCollide`, `TestMindXPValueXPSlotAndGainsXPEachMoveTheDigest` |
| AC-3 | `TestPayExperienceRefusesAClassThatDoesNotGain` |
| AC-4 | `TestPayExperienceRefusesTheSameOwnerSlotAndALockedRelation` — both halves |
| AC-5 | `TestPayExperienceRefusesATargetThatWasAlreadyDead` at the guard, `TestResolveBlowReadsTheLivenessBeforeItSubtracts` at the caller |
| AC-6 | `TestFR6PaysHalfPlusOneAndSplitsAcrossBlows`, `TestAKillingBlowStillPaysExperience` |
| AC-7 | `TestPanelStatesTheSkillRowForOneSlotAndForNone` — one trained slot, and none |
| AC-8 | `TestTheReadoutsSkillNumberRisesOnABlowAndHoldsWithoutOne` |

The refusals of DD-4 are witnessed from both sides — `TestNewWorldRefuses…` and
`TestUnmarshalRefuses…` for the slot range and the negative experience, and the decoder alone
for a `GainsXP` byte outside `{0,1}`, which no constructor can produce. What DD-4 deliberately
does **not** refuse is witnessed too, by
`TestMindAndXPValueAreCarriedWholeAndRefusedNowhere`.

## Where the numbers came out

**P-1 — the inverse.** Exact over the whole level range, both halves. The curve never goes
flat: no `n` in `[1,100]` has `S(n) = S(n-1)`, measured, so the inverse is injective throughout
and there is no level a gain could skip.

**P-2 — the blow amount.** Zero divergences from the truncated real-valued form
`xpValue * 0.5 * removed / maxHP + 1`, swept over experience values `{0,1,2,3,7,25,100,999,
2^20, 2^31-1}` × maximum health `{1,2,3,5,10,37,100,1000,2000}` × removals from 1 to three
times the maximum, so overkill blows are covered. The `+ 1` is exact outside the truncation
because `floor(x + 1) = floor(x) + 1` for any real `x`.

**P-3 — the Mind scaling, measured and not claimed.** The integer form
`raw * (4*mind + 30) / 120` is **not** equal to `raw * (mind/30 + 0.25)`. Over `mind` in
`[0,400]` × `raw` in `[0,2000]` — 802 401 pairs — there are **1683 divergences**. The first is
**`mind = 3`, `raw = 180`: the integer form gives 63, the real-valued form gives 62.** The
reason is legible: `3/30` is `0.1`, which is not representable in binary and rounds below, so
`180 × 0.35` falls just under 63 and truncates to 62. Every divergence is of this kind — the
integer form returns the exact value and the float form returns one less at a point where the
product is an exact integer. This build takes the integer form and the divergence is a
disclosed, measured departure from the original's own float arithmetic, not an equality claim.

**P-4 — the determinism wall.** `internal/archtest` passes. `pkg/sim`'s non-test sources gained
no float identifier, no float literal and no import at all. The two real-valued references live
in `pkg/sim`'s **test** files, which the source scan does not read, and in `pkg/data`, which is
outside the wall and already calls `math.Pow`.

**P-5 — no invented numbers.** `XPValue` is the units table's own column, already parsed and
previously unread by anything. Mind is the row's own. The slot ordering is `pkg/data`'s. The
game's compiled constants sit in one block in `pkg/sim/combat.go` with a claim id beside each —
the halving and the added one under `HERO-KILL-027`, the three numbers of the Mind scaling
under `HERO-XP-010`.

## Against a lawful install

`cmd/paneldump -assets <root> -mission 10`, run from this branch:

```
party  id=35   box=298x229  rows=12  name="Human Swordsman"
         | SKILL 0 10 0 0 0 0  XP 1593
```

**SC-2 holds.** Six numbers, in slot order, zeros included — where the row before this story
read `SKILL Blade 10` and was suppressed entirely for a hero trained in nothing. `1593` is
`S(10)` exactly, so the panel's total at tick zero is what his one trained level already
accounted for (FR-13).

**SC-1 is witnessed to the readout and not through the window.** The rise is asserted end to
end in `pkg/game` by stepping a real `sim.World` through a landed blow and reading the number
the panel would state, and that test goes red when the overlay is removed. It is **not** driven
through the GUI here, and the attempt is recorded rather than worked around. `cmd/missionrun
-attack p0:u57` on mission 10 walked for the whole 40 000-tick ceiling without closing on its
victim, which stayed at full health; the same run given a waypoint first was stopped at tick
544 by the world deciding the mission **lost**. Both are pre-existing conditions of the driver
and the map, unchanged by this story, and neither leaves a blow whose payment could be read off
a campaign run.

## The third merge, and whether the credited slot moved

Master landed the starting-equipment hotfix (`c9d0a2c`) after the second merge. It touches this
story's own files, and it left **four conflict hunks, all in `pkg/mapload/fromalm.go`** — both
sides add fields at the same four construction sites. Every hunk is additive on both sides and
was resolved by keeping both. `pkg/sim/binary.go`, `pkg/sim/world.go` and `pkg/mapload/start.go`
auto-merged.

**`formatVersion` stays 35 and `entityLen` stays 183.** The hotfix fills `0124`'s existing
equipment section rather than widening the record, so this story's block is still the tail.

**No landed test moved, and that is measured rather than asserted.** The set of `func Test…`
names on `origin/master` is a strict subset of the set on this branch: the difference in one
direction is **empty** and in the other is the **34** this story adds. The full suite passed on
the first run after the merge — including the one auto-merged site worth suspecting, the
unresolved arm of `blockFor`, which took only this story's fields. That is correct: master's own
version of that arm names no weapon either, because a placement resolving to nothing has none.

**The credited slot did not move.** The hotfix puts a resolved weapon's item **code** into
`sim.Stock.Equipped`; this story's `XPSlot` rides on `Combat.SkillSlot`, folded from the
`*data.Weapon` the hotfix left untouched. Measured on the EN install over missions 10, 20 and
30, the two channels are provably disjoint in **both** directions:

| Mission | credits a weapon slot, wears nothing | wears a weapon, credits `General` |
|---|---|---|
| 10 | 3 | 0 |
| 20 | 3 | 24 |
| 30 | 4 | 14 |

The first column is the hotfix withholding an `"NPC"`-templated row's item while the row's own
weapon still folded into its numbers — so `XPSlot` cannot be reading the loadout. The second is
a **creature** wearing a code while crediting slot 0, because `UnitDef.Combat` never calls the
recompute and so leaves `SkillSlot` at zero. That costs nothing today, since a creature does not
gain at all; it is written down because a later story that lets one gain would credit `General`
for every blow, and this table is where that is already visible.

After this merge the whole revert probe was **re-run again** and all ten clauses were still red,
and P-2 and P-3 were **re-measured**, not carried: still zero divergences for P-2, and still
1683 for P-3 with the same first case at `mind = 3`, `raw = 180`.

## Divergences, named

1. **Per-slot experience is authoritative state, not a cache.** The original re-derives the
   per-slot value from the level, which discards progress short of a level. This build stores
   the experience and derives the level, so the first blow moves the number.
2. **The credited slot is fixed at load.** The original re-reads its class flag at the blow.
   This tree carries no fighter/mage flag on either definition tier, so both arms collapse to
   one expression — the weapon's own skill slot, which is `General` for anyone holding no melee
   weapon, i.e. exactly what the mage arm would credit. It is one expression, in one function.
3. **The second refusal is transcribed positionally.** The original refuses on the diplomacy
   byte's bit 1. This build's relation byte carries bit 1 as *permanently allied*, which is the
   meaning that refusal would want, but no claim binds the two bit numberings together.
4. **A person's experience value is the constructor's** — the humans table carries no such
   column, so a placed person is worth only the per-blow one. Same precedent as the two
   regeneration periods.
5. **A raised level does not re-derive a blow's numbers.** Nothing in this story recomputes.

## Cut, each in one line

The 10 % loss path; the type-id range restriction, which `GainsXP` already carries for a
person; a class bit in the simulation. All three are stated in `spec.md`'s own out-of-scope
section with their reasons.

## The byte form, and what the second merge cost

Version **35**, `entityLen` **183**. Master moved twice under this branch: version 32 at the
first merge, **34** at the second, and the entity record itself stayed at 149 both times —
`0124`'s version 34 is a world-level section, not an entity field — so the block still sits at
the tail with no offset in front of it moved.

The second merge left **14 conflicted files and about 45 hunks**, and it contained exactly the
failure a sibling lane had warned of: this story's own peel was still pinned at **32** in three
files, and **two tests carried no conflict marker at all** while peeling a field straight off a
buffer that had since gained the experience tail. Git had nothing to mark and reading the diff
would not have shown it; the suite going red is what found them. Ten digests moved, each taken
from a failing run's reported value. Two are their own cross-check: the pre-experience pin and
routed digests came out **equal to master's own pre-merge values**, which is what a correctly
composed peel chain must produce and a wrong one almost certainly would not.

`pkg/sim/corpseloot_test.go`'s version tripwire and `sightrange_test.go`'s `entityLen` tripwire
were **re-pinned with a note naming this story, never deleted**.
