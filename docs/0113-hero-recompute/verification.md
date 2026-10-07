# Verification — 0113-hero-recompute

Branch `0113-hero-recompute`, base `b23915d`: stages 1 to 3, the three task commits, one untrailered
correction commit carrying FR-6a, AC-13 and DD-13 (see *The correction*, below), and this one.

## Gate

Run from the worktree on the committed tree, nothing dirty:

```
$ go build ./... && go vet ./... && go test -trimpath -count=1 ./...
ok  againrom/... (every package; no game install present)
$ gofmt -l $(git ls-files '*.go')
(no output)
$ sh scripts/check-no-game-assets.sh
$ sh scripts/check-doc-budget.sh
$ sh scripts/check-sdd-audit.sh
```

```
$ git log --format='%h %(trailers:key=Co-Authored-By)' b23915d..HEAD
(no trailer on any commit)
$ git diff --diff-filter=D --name-only master..HEAD
(no deletions)
```

## Acceptance

**AC-1 — one implementation, and the accessors compute nothing.** `Derive`, `Speed` and `Sight` are
one line each (`pkg/data/hero.go:180`, `:204`, `:229`), and each is a call into `Recompute`.
Witnessed by MUTATION rather than by reading, because a wrapper that looks thin can still hold a
term: three separate edits inside `Recompute`, each reverted after the run.

```
$ # defence = reaction/defenceDivisor  ->  + 1
--- FAIL: TestABarePersonsEightAreTheDerivationAndHisOwnCadence
--- FAIL: TestTheChargenStartWithItsStartingWeapon
--- FAIL: TestTheZeroHeroDerivesTheNumbersThePartyCarriedBefore
--- FAIL: TestTheHeadlessCheckLineIsWhatItWas          (pkg/game, through Derive)
--- FAIL: TestAnInstalledTableBecomesTheHerosBand      (pkg/game, through Derive)
$ # speed = reaction/speedDivisor + speedBranch  ->  + 1
--- FAIL: TestSpeedIsDerivedFromReactionAtABranchOfTwelve
$ # sight = (mind+reaction)/sightDivisor + sightBase  ->  + 1
--- FAIL: TestSightIsDerivedFromMindAndReactionTogether
```

Each of those tests calls only `Derive`, `Speed` or `Sight`. A body left behind in any of the three
would have absorbed the mutation and the test would have passed.

**AC-2 — an unequipped character reads one number on all five protections.**
`data.TestAC2UnequippedProtectionsAreCappedSpiritHalved`. Spirit 41 gives 20 on all five; Spirit 999
gives 25 on all five, which is the CAPPED statistic halved and not the raw one. This is
`HERO-RESIST-012`'s own published prediction and it holds.

**AC-3 — no accumulation across two recomputes.**
`data.TestAC3RecomputeIsIdempotentAndDoesNotAccumulate`, over a character with a profile, a skill and
an `EquipMod` carrying a defence term and five protection terms. `Derived` is comparable, so the
witness is one `==` over the whole set rather than a field walk that could omit the field that drifts.

**AC-4 — the combat block is unchanged, field for field.** `pkg/data/hero_test.go` is BYTE-IDENTICAL
to master:

```
$ git diff --stat master -- pkg/data/hero_test.go
(no output)
```

Its cases — bare, armed, untrained, above the cap, the zero value, the active-skill asymmetry, the
cadence guard, reach — all pass against the new implementation. Nothing in them was relaxed, and the
AC-1 mutations above show they are still live.

**AC-5 — the experience sum.** `data.TestAC5ExperienceIsAFunctionOfSkillLevels`: all-zero skills give
0 in every slot and 0 in total; one slot at level 10 gives 1593 in that slot and 0 in the other five.
`data.TestAC5SlotZeroIsInTheSum` covers the asymmetry: slot 0 at level 10 also gives 1593, even
though the restore and the clamp both skip it.

**AC-6 — the health column gates the first arm.**
`data.TestAC6HealthColumnGatesTheFirstArm`, on a character whose experience term is nonzero, so the
two arms are distinguishable rather than both collapsing to the logarithm of 1.

**AC-7 — the pools clamp.** `data.TestAC7ClampPools`. The test first asserts that the SAME character
with the mana column set reaches a nonzero maximum, and fails loudly if he does not — without that
line the zero-column assertion would pass for a character who has no mana either way and would
witness nothing.

**AC-8 — a character with only a weapon has five zero resistances.**
`data.TestAC8ResistancesAreZeroWithOnlyAWeapon`. The five damage-kind resistances are never
re-derived; the block's clear is their only writer.

**AC-9 — the defence and damage terms are independent.**
`data.TestAC9EquipModDefenceAndDamageAreIndependent`, both directions.

**AC-10 — the protection clamp, both ends.** `data.TestAC10ProtectionClamp`. At Spirit 30 the ceiling
is `30/2 + 70 = 85`; a term of 1000 lands on 85 and a term of −1000 lands on 0.

**AC-11 — the panel states the three families.** Two witnesses, and the second was added because the
first does not discriminate. `game.TestPartyCharactersPairsTheStartsOwnSlices` compares whole
`ui.UnitCharacter` values against wants computed through `Recompute` — that pins the PAIRING, that
`partyCharacters` calls the derivation for each member, and it would pass a wrong derivation as
readily as a right one. So one independent arithmetic pin sits beside it: member 8's Spirit is 17, so
his five protections are 8, his one slot at level 10 gives experience 1593, and his five resistances
are 0, all written out rather than derived. Measured:

```
$ # protection[i] = spirit/spiritHalfDivisor  ->  + 1
BEFORE the pin was added:  ok  againrom/pkg/game   (the mutation passed)
AFTER:                     --- FAIL: TestPartyCharactersPairsTheStartsOwnSlices
```

`ui.TestPanelStatesTheRecomputesExperienceAndFamilies` witnesses the text: the number, the
space-separated five, and the `Known` gate. `ui.TestAPanelStatesACharacterInTheDecodedOrder` pins the
row list with the three new rows in position, over a fixture given nonzero and mutually distinct
values so the rows say something.

**AC-12 — a recompute lands on one live entity.** `sim.TestSetCombatWitnessesAC12`: the named
entity's nine fields move, a second entity in the same world is untouched field for field, and an
unknown id answers false and leaves `World.Hash()` identical.

**AC-13 — the skill restore.** `data.TestAC13TheSkillRestore`. A bonus of 20 on a level of 10
restores to 30; that raised level moves to-hit by exactly 60 and the damage base by exactly 4, and
moves the spread by nothing — the asymmetry `HERO-DERIVE-034` names. The same bonus on slot 0 moves
nothing and slot 0's own level of 200 is not clamped, because slot 0 is in neither loop. A bonus of
500 lands on 100 and one of −500 lands on 0.

## Derived properties

**P-1 — no simulation field, byte form, digest or version.**

```
$ git diff --stat master -- pkg/sim/binary.go
(no output)
$ grep -n 'const formatVersion' pkg/sim/binary.go
336:const formatVersion = 25
```

Version 25 is `0109`'s and is untouched; **27 was allocated to this story and was not spent**. The
reason it is not needed was checked rather than assumed: nothing in `pkg/sim` reads a protection or a
resistance — `resolveBlow`'s own doc block lists the elemental protections among what is not there —
so the new values are loader values like `ui.UnitCharacter` already is. `pkg/sim/rearm.go` adds a
type and a method and no field to `Entity`.

**P-2 — the float arithmetic stays outside the determinism wall.** `pkg/data` holds `math.Pow` and
`math.Log`; `pkg/sim` gains only int32 and bool. `internal/archtest`'s determinism scan is unchanged
and green.

**P-3 — the party's health pair is unchanged.** `pkg/mapload/start.go` is not touched by this branch
at all, so a member still spawns at `SpawnHP` on `0078`'s standing divergence. `HealthMax` and
`ManaMax` are produced and wired to nothing; the seam is named in `Derived`'s doc block with the
three inputs nobody in this tree can state.

**P-4 — truncation toward zero throughout.** Every floating intermediate goes through `ftol`, which
truncates and answers 0 for NaN and infinity. The pools truncate BETWEEN their three steps rather
than once at the end, so the truncations compound as they do in the original.

**P-5 — the zero character remains legal.** `data.TestTheZeroHeroDerivesTheNumbersThePartyCarriedBefore`
in the untouched `hero_test.go`, plus the all-zero arm of AC-5.

## Success criteria

**SC-1, SC-2** — the gate block at the top; all green on the committed tree.

**SC-3 — no derivation body left in `hero.go`.**

```
$ grep -n 'pow11\|math\.Log' pkg/data/*.go | grep -v _test
pkg/data/hero.go:264:func pow11(n int32) float64 { ... }      <- the shared helper, unchanged
pkg/data/recompute.go:167,236,256,283,349,351                  <- every USE is here
```

`hero.go` retains the helper and no expression that uses it.

**SC-4 — the byte form and its version are untouched.** Shown under P-1.

**SC-5 — the deletion set is empty.** Shown in the gate block.

## The correction

After T1, T2 and T3 had landed, the seat corrected the premise this story's experience clause was
written on. The correction is right and it moved real code, not only prose.

What was wrong: `HERO-XP-010`'s headline — experience is a function of the skill levels, not an
accumulated counter — was read as *experience has no storage*. The mechanism under it says otherwise.
There are three fields (the level word, the per-slot experience dword at `actor+0x1cc + 4i`, and the
recomputed total) and two routines between them in opposite directions: the sum, and an inverse the
loss path uses to put a LEVEL back after taking a tenth off a slot's EXPERIENCE. What is genuinely
retracted is the "nothing increments `+0x130`" enumeration; what survives is that a hero's total is
recomputed rather than accumulated.

What changed as a result:

- every sentence of the form *there is no counter* is gone from `spec.md`, `analysis.md`,
  `provenance.md`, `pkg/data/recompute.go` and `pkg/ui/panel.go`;
- `Derived.SkillXP` is documented as the PRIMARY storage and the total as the derived value, with the
  inverse named as a seam — its arithmetic is unread, so it is not written, and it is not derived
  algebraically either: a plausible closed form is not a decoded one, which is the lesson
  `HERO-ARMOUR-018`'s retraction was recorded for;
- and the correction surfaced a term this story had missed outright. `HERO-ORDER-014`'s second
  ordering point is about the SKILL RESTORE, and `HERO-SKILL-009` states it: slots 1 to 5 are
  restored from a base with a per-slot bonus added and clamped to `[0,100]`, slot 0 is in neither
  loop but is in the experience sum. That is now FR-6a, DD-13 and AC-13, with `EquipMod.SkillBonus`
  and `Derived.Skill` carrying it.

It is one commit and it carries no `SDD-Task:` trailer, because the three task ids had already
landed and the bijection allows each exactly once.

## Two things a reader should not take on trust

**The mana column gates the whole derivation, not one arm.** T1 first implemented it as health's
counterpart — first arm skipped, experience term still standing — which left a trained character with
no mana column carrying a nonzero maximum and made AC-7 pass only because its fixture had no skills.
The claim says the current mana is set to 0 *instead*, so the gate covers the whole thing. Corrected
before T1 was committed, and the AC-7 fixture now has a trained slot so the assertion discriminates.

**`SetCombat` is a second exported writer on `World`, and it is declared rather than excused.**
`pkg/sim/world_test.go` pins the exported method set and requires a writer to be named. It is named,
beside `UnmarshalBinary`, with the reason in the list: the numbers cannot be computed inside the wall,
and `Command` — the ordinary door — refuses a third argument field in its own doc block, so nine
numbers do not fit through it. It sets no position, no tick, no bounds and no generator state, which
is what that pin was built to catch. Nothing in this story calls it.

## What the owner sees

Start a mission and left-click a party member. The bottom-left information panel now carries three
rows under `ABSORB`. For the hero this front end generates — Body 43, Reaction 26, Mind 15,
Spirit 15, Blade at 10:

| row | before | after |
|---|---|---|
| `XP` | absent | `1593` |
| `PROTECT` | absent | `7 7 7 7 7` |
| `RESIST` | absent | `0 0 0 0 0` |

Seven is his capped Spirit halved, and five equal numbers on an unequipped character is the published
prediction this build now satisfies on screen. The five zeroes are correct rather than missing: a
damage-kind resistance is never re-derived for a character, and only equipment moves one.
