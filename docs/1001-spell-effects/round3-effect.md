# 1001-spell-effects — fix round 3, effect lane (R3-A1)

Branch `story/1001-r3-effect`, based on `efd89b8`.

This lane owns finding R3-A1 (the speed floor) and the minor finding round3-review.md names the
same family: a Protection effect that clamps at attach, then a recompute, then expiry.

Every fix below has a witness that fails when its own production line is reverted and passes with
it restored. Each revert was undone afterwards and the full suite is green.

## What was wrong

Fix round 2 gave `applyEffectDelta` a floor at `minEffectSpeed = 1` and made `attachEffect` store
what actually landed, so a single effect's own attach and expiry are exact inverses. Both halves
fail once a second effect of the same kind stands on the same target, because a clamp on ONE
record's removal is decided against the target's CURRENT value — which already carries every OTHER
active effect — not against that record's own isolated contribution.

`pkg/sim/effect.go`'s `removeAttachedAt` called `applyEffectDelta` and discarded the return.
Reproduced with the reviewer's own numbers (Haste +7, Slow -7, Freezing Cloud -7, shipped
magnitudes at power 100, on a base-10 actor): all three attach to speed 3 (10+7-7-7, no clamp
engages during attach). Haste expires first (shortest duration). Its reversal is `-7`: `3-7=-4`,
floored to `1`. The floor is correct AT THAT INSTANT — Slow and Freezing Cloud alone do put the
actor below 1 — but the 3 units the floor withheld from that removal were never recorded anywhere,
so when Slow and Freezing Cloud each expire afterward, their own full `+7` reversal is credited on
top of a value that is already 2 higher than the mathematically exact trajectory. Final speed: 15,
not the base 10.

`pkg/sim/rearm.go`'s `SetDerived` carried no floor on Speed at all — `e.Speed = d.Speed +
w.effectDelta(id, EffectSpeed)`, unclamped — while the two neighbouring lines (scan range, and
`SetCombat`'s protection loop) already clamp. `SetDerived` is reached from every skill level rise
(`pkg/game/rearm.go:100`, `recomputeRaisedSkills`, called at `pkg/game/world.go:3067`), which this
story made routine: every successful spellbook use trains. A fresh base too low to sustain the
active total produced a negative `Entity.Speed`, which `moverSpeed`/`rated` (`pkg/sim/world.go`)
read as "no rate at all" — the FASTEST cadence the engine has, not the slowest — and which
`groupMinSpeed` (`pkg/sim/group.go`, not this lane's file) narrows through `int16` back to `uint8`,
turning `-4` into `252`.

The same shape recurs one layer up, in `SetCombat`'s protection loop, which the review names the
same family: a Protection effect that clamps at 100 on attach stores its landed magnitude (say
`+20` on a base of 80). A later recompute (a level rise, an armour swap — anything that calls
`SetCombat` while the effect is still active) supplies a NEW base — say 95, higher than the one the
effect originally clamped against. `SetCombat` recomputes `95 + 20 = 115`, clamps to 100 — correct
for that instant — but the effect's own stored magnitude still reads 20. Expiry then subtracts 20
from 100, landing at 80: the ATTACH-TIME base, not the recompute's own 95.

## The shape I built, and the one I rejected

Both bugs are the same fact stated twice: `Entity.Speed` (and the clamped protection slots) are
supposed to equal an effect-free base plus the sum of every active effect's own contribution, with
exactly one clamp applied to the total — not one clamp per record, decided against whatever the
OTHER records already did to the value.

**Rejected: recompute the whole value fresh at every attach and removal, from a base reconstructed
as `Entity.Speed_before − Σ(stored magnitudes active before this operation)`.** This is
mathematically identical to discarding the shortfall — traced by hand and confirmed by test against
the exact reviewer scenario: the moment ANY prior clamp has engaged, `Entity.Speed_before` no
longer equals the true unclamped value, so subtracting the (also-true) prior total from a corrupted
`Entity.Speed_before` reconstructs a corrupted base, and every later operation inherits the error.
This produces the SAME wrong 15. The missing ingredient in this shape is a persisted, never-clamped
"true base" per entity, which `pkg/sim` cannot supply for the attach/remove pair operating without
a `SetDerived` call in between (`Entity` carries no such field, and adding one is a byte-form change
— a new format-version story, not this fix).

**Built: redistribute what a clamp could not deliver onto a still-active sibling record of the same
kind and target**, so the invariant `current value == effect-free base + Σ(stored magnitudes of
active effects)` stays exactly true at every step, using only the fields `attachedEffect` and
`Entity` already carry. `pkg/sim/effect.go` adds `redistributeShortfall(target, kind, skip,
shortfall)`: it finds the first still-active record of `kind` on `target` other than `skip` and
adds `shortfall` to its stored `Magnitude`. Three call sites feed it exactly what their own clamp
discarded (`delivered − intended`):

- `removeAttachedAt` (`pkg/sim/effect.go`), for every kind except `EffectHealth`.
- `SetDerived`'s Speed line (`pkg/sim/rearm.go`), when a fresh base still cannot clear the floor.
- `SetCombat`'s protection loop (`pkg/sim/rearm.go`), when a fresh base plus the active total
  crosses 0 or 100.

`EffectHealth` is excluded everywhere. Its own "clamp" is the `MaxHP` ceiling, which ordinary
healing unrelated to any effect can also reach, so there is no effect-free base to redistribute
against the way Speed, ScanRange and Protection each have one — redistributing there would corrupt
an UNRELATED health effect's own stored magnitude on the next combat tick that happens to graze the
ceiling. Witnessed directly: see `TestHealthEffectRemovalDoesNotCorruptASiblingsOwnMagnitude` below.

**Known cost of the shape I built, disclosed rather than hidden.** A record that has absorbed a
sibling's clamp shortfall no longer holds its rule's own nominal magnitude. `SetDerived`'s own doc
comment says active spell deltas are "re-applied to the fresh base," which reads as the rule's
nominal number; a level-up recompute landing while a redistributed sibling is still active combines
the NEW base with an ADJUSTED, not nominal, total, and can differ from what a recompute against the
untouched nominal magnitudes would give. This is narrow — it needs a clamp to have already engaged
among overlapping same-kind effects, AND a base-changing recompute to land before the redistributed
sibling itself expires — and it is the trade against the alternative (leave every stored magnitude
alone and let a clamped removal simply discard what it cannot deliver), which is what R3-A1
measured: a base-10 actor permanently at 15, with no further recompute involved at all. Reversal
exactness for a stable base is the invariant `spec.md` already states and round3-review.md
measured; exactness of a LATER, DIFFERENT base's recompute against a sibling a clamp had to touch
is the second-order case this trades away. `DIV-053` below records it.

## Witnesses

All five live in `pkg/sim/round3effect1001_test.go`, new. Each revert was applied, run, and undone;
the tree is clean and the fixes are all back in place after this section.

1. **`TestSpeedEffectsReverseExactlyAcrossOverlappingClamps/positive_expires_first`** — the
   reviewer's own three-effect scenario. Revert: change `removeAttachedAt` back to
   `w.applyEffectDelta(ti, e.Kind, -e.Magnitude)` with the return discarded (no redistribution
   call). Failure: `speed after every effect expired = 15, want the base 10` — the exact number
   round3-review.md measured. The `positive_expires_last` subtest stayed green under this same
   revert: negative-first removal never needs the floor, confirming it as a true no-regression
   control rather than a second copy of the same assertion.

2. **`TestSetDerivedFloorsWhenTheFreshBaseStillCannotClearIt`** — R3-A1's own "no floor at all"
   half. Revert: remove the `if speed < minEffectSpeed { ... }` block around `SetDerived`'s Speed
   line. Failure: `speed after SetDerived is -4, want the floor 1 exactly`.

3. **`TestGroupSpeedReadingDoesNotWrapAfterASpeedFloorMiss`** — the `groupMinSpeed` symptom. Same
   revert as (2). Failure: `groupMinSpeed reads 252, want the floored member's own 1 — 252 would be
   -4's wrapped low byte` — the exact wrapped value round3-review.md measured.
   `TestTheGroupTermIsTheMinimumInTheOriginalsOwnWidths` (`formation_test.go`, unmodified) already
   documents that same `-4 -> 252` narrowing as an INTENTIONAL, decoded byte-width rule reachable by
   customisation; this test's only claim is that the floor keeps an ordinary effect stack from
   reaching it that way.

4. **`TestSetDerivedRecomputesExactlyAfterAClampedRemoval`** — goes past "does it floor" to "is the
   number right" for the redistribution invariant itself. After Haste's clamped removal, the active
   total (`w.effectDelta(2, EffectSpeed)`) is asserted to be exactly `-9` (Slow's stored magnitude
   redistributed from `-7` to `-2`, plus Freezing Cloud's untouched `-7`), and `SetDerived` with a
   fresh base of 20 is asserted to land on exactly `20 + (-9) = 11`. This test stayed green under
   both reverts above (it exercises neither line); it is the positive proof that the invariant holds
   exactly, not only that it clamps.

5. **`TestProtectionSurvivesARecomputeWhileClampedThenExpiresToTheNewBase`** — the minor finding's
   own fix. Revert: remove the `if hasEffect { ... }` redistribution block in `SetCombat`'s
   protection loop. Failure: `fire protection after expiry is 80, want the recompute's own base 95`
   — the actor lands on the ATTACH-TIME base rather than the recompute's, exactly as
   round3-review.md describes.

6. **`TestHealthEffectRemovalDoesNotCorruptASiblingsOwnMagnitude`** — the `EffectHealth` exclusion
   has no witness among tests that predate this lane (checked: dropping the exclusion left the
   whole suite green). Built one: two independent duration-mode damage effects on one actor, with
   ordinary combat healing (a direct `HP` write, unrelated to either effect) pushing the actor to
   the `MaxHP` ceiling before the first expires. Revert: drop `&& e.Kind != EffectHealth` from
   `removeAttachedAt`'s guard. Failure: `HP after B's own removal is 190, want 170 (150 plus B's
   nominal 20)` — the ceiling clamp on A's removal leaked 20 units into B's own stored magnitude,
   which B's own later removal then gave back on top of unrelated combat healing.

## Gate

```
go build ./... && go vet ./... && gofmt -l $(git ls-files --cached --others --exclude-standard '*.go') && go test -trimpath -count=1 ./...
```

Clean: no build or vet output, gofmt prints nothing, and every package reports `ok` (nine packages
report `[no test files]`, all pre-existing and unrelated to this lane). `go test ./internal/archtest/...`
passes explicitly: `pkg/sim` stays stdlib-only and float-free, both files this lane touched included.

```
bash scripts/check-no-game-assets.sh
```

`check-no-game-assets: clean (tree scan)`.

## Proposed `spec.md` sentences

For the "Ordinary point effects" section, immediately after the existing "Expiry reverses exactly
what the application landed..." paragraph:

> Where more than one effect of a kind stands on an actor and a clamp keeps one record's own
> removal from delivering its full reversal, the undelivered remainder moves onto a still-active
> record of the same kind rather than being discarded, so the group's combined total stays exactly
> recoverable. A fresh recompute — SetDerived's speed and scan-range lines, and a recompute's
> protection block — applies the same rule to its own clamp against the fresh base it is handed.
> Health is excluded: its ceiling is `MaxHP`, reachable by healing unrelated to any effect, so there
> is no effect-free base to redistribute a clamped health reversal against.

Immediately after the existing "An effect may not take an actor's speed below 1." sentence, no
change needed — it already states the floor; the paragraph above covers the multi-effect case.

## Proposed `docs/DIVERGENCES.md` rows

`DIV-036`'s existing "Implemented behaviour" cell describes the single-effect case only. Proposed
replacement text for that cell (same row, ID and every other column unchanged):

> An effect may not take `Entity.Speed` below 1. Reversal is exact per record in isolation. Where
> several `EffectSpeed` records stand together and a clamp keeps one record's removal from
> delivering its full reversal, the undelivered remainder is redistributed onto a still-active
> sibling record so the group's own total stays exactly reconstructible (`DIV-052`).

Two new rows, `DIV-052` and `DIV-053`:

| DIV-052 | simulation / stacked effect clamps | — | `HERO-SPEED-008` establishes only a single scalar's own destination; no claim states what the original does when several effects of one kind combine and a clamp keeps one of them from delivering its own full reversal | `removeAttachedAt`, `SetDerived`'s Speed line and `SetCombat`'s protection loop each redistribute what a clamp could not deliver onto one still-active sibling record of the same kind and target (`pkg/sim/effect.go`'s `redistributeShortfall`), rather than discarding it. A base-10 actor under Haste +7, Slow -7 and Freezing Cloud -7, expired in duration order, now returns to 10 rather than the 15 round3-review.md measured under fix round 2 | UNKNOWN | Discarding the remainder is fix round 2's own bug (R3-A1): it let a base-10 actor drift to 15. Recomputing a fresh base at every operation was tried and proven equivalent to discarding it by hand-trace and by test, because `pkg/sim` has no access to an effect-free base once any prior clamp has engaged — the determinism wall keeps it from `pkg/data`, and `Entity` carries no such field. `DIV-053` records the redistribution's own cost | A claim naming how the original resolves overlapping same-kind effect magnitudes against a shared clamp, or an owner decision to add a byte-form field carrying an effect-free base | OPEN |
| DIV-053 | simulation / stacked effect clamps (recompute cost) | — | Not addressed by any claim; see `DIV-052` | A record whose stored magnitude has absorbed a sibling's clamp shortfall (`DIV-052`) no longer holds its rule's own nominal magnitude. A later `SetDerived` or `SetCombat` recompute that combines a NEW base with that record's adjusted magnitude, while the record is still active, can differ from combining the new base with the rule's nominal magnitude | FIDELITY-DEBT | The alternative — leaving every stored magnitude at its rule's nominal value always — breaks the primary, spec-stated, review-measured invariant (expiry reverses exactly what application landed) instead. Reversal exactness against a stable base is prioritized over recompute exactness across a base change surviving a redistributed sibling. Narrow: needs a clamp already engaged among overlapping same-kind effects AND a base-changing recompute landing before the redistributed sibling expires | An `Entity` field carrying an effect-free base (a byte-form change, its own story) removes the need to choose between the two | OPEN |

## What I did not do, this pass

- **`pkg/sim/rearm.go`'s scan-range line in `SetDerived`** (`sight := int32(d.ScanRange) +
  w.effectDelta(...)`, already floored and ceiled): left untouched. It was not named by the review,
  its magnitudes are small (Light/Darkness: `power/30 + 1`, at most about 4 at power 100) against a
  0..255 range, and `removeAttachedAt`'s own redistribution (which does cover `EffectScanRange`,
  ungated by kind beyond the `EffectHealth` exclusion) already keeps its OWN attach/remove pair
  exact. Only the SAME cross-base-recompute second-order case `DIV-053` names for Speed and
  Protection is left open for ScanRange too, and it is narrower still there given the smaller
  magnitudes.
- **`pkg/sim/celleffect.go`, `pkg/sim/route.go`, `pkg/sim/step.go`, `pkg/sim/group.go`,
  `pkg/sim/spell.go`, `pkg/sim/combat.go`, `pkg/game/iteminfo.go`**: not read for editing, per scope.
- **`pkg/game/rearm.go`**: read, not changed. `recomputeRaisedSkills` is the call site the review
  names as making `SetDerived` routine; it needed no change of its own; the fix is entirely in what
  `SetDerived` does with the value it is handed.

## Files this pass changed

`pkg/sim/effect.go`, `pkg/sim/rearm.go`, and one new test file, `pkg/sim/round3effect1001_test.go`.
