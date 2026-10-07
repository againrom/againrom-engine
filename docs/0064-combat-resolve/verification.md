# Verification — an attack that resolves

Windows 11, Go 1.26.1 (the toolchain `go.mod` pins), no game install present. Every command below
was run in this worktree at the tip of `impl/0064-combat-resolve`. Research is at the pin the
submodule carries, `a93d19a`, with no leading character.

## The gate

The tree here is `impl/0064-combat-resolve` with `origin/master` merged in — `0062`, `0063` and
`0065` included — so every number below is the merged result and not this branch alone.

```
$ go build ./...        EXIT=0
$ go vet ./...          EXIT=0
$ gofmt -l $(git ls-files '*.go')
(nothing)
$ go test -count=1 -trimpath ./...
ok  againrom/cmd/terraintool  ok  againrom/cmd/texttool     ok  againrom/internal/archtest
ok  againrom/internal/notices ok  againrom/internal/synth   ok  againrom/pkg/data
ok  againrom/pkg/formats/alm  ok  againrom/pkg/formats/databin  ok  againrom/pkg/formats/pal
ok  againrom/pkg/formats/reg  ok  againrom/pkg/formats/res  ok  againrom/pkg/formats/spr16
ok  againrom/pkg/formats/spr256  ok  againrom/pkg/game      ok  againrom/pkg/mapedit
ok  againrom/pkg/mapload      ok  againrom/pkg/render/camera   ok  againrom/pkg/render/frame
ok  againrom/pkg/render/menu  ok  againrom/pkg/render/terrain  ok  againrom/pkg/render/text
ok  againrom/pkg/sim          ok  againrom/pkg/ui           ok  againrom/pkg/vfs
(no FAIL; 3 packages have no test files)

$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)                                            EXIT=0
$ sh scripts/check-doc-budget.sh
0064-combat-resolve: plan <= 1.2 x spec   13307 <= 15970 bytes  ok
0064-combat-resolve: tasks <= 1.2 x plan   5084 <= 15968 bytes  ok                 EXIT=0
$ sh scripts/check-sdd-audit.sh
FAIL set: EMPTY                                                                    EXIT=0
```

`gofmt -l .` from the root walks into the `research/` submodule and reports seventeen files there;
none is ours, and the invocation above is the repo's own (`AGENTS.md`). `check-sdd-audit`'s
note/warning **count** is meaningless from a worktree, which has no `builds/` — only its FAIL set
is comparable and it is empty.

`git diff --diff-filter=D --name-only <base> HEAD` over this branch prints nothing: **no file was
deleted.** 22 files changed, 1878 insertions, 438 deletions.

## Acceptance criteria

Every witness is a Go test in `pkg/sim`; the suite runs 279 top-level tests with no game install.

| # | Witness |
|---|---|
| AC-1 | `TestAnOrderedAttackKillsItsVictim` — 20 health, blows of 10: downed on one advance, dead on a later one, and neither unit holds an order after |
| AC-2 | `TestABlowLandsOnTheChargeThAdvanceAndThePeriodIsTheCycle` — six cadences, health traced advance by advance, first blow on the charge-th and every gap in `[c+r, c+r+3]` |
| AC-3 | `TestSpeedMovesNothingAboutTheCycle` — two worlds from one seed differing only in a speed, compared field by field for 40 advances |
| AC-4 | `TestABlowThatCannotLandRemovesNothing/a miss removes nothing…` — 400 advances, both outcomes required to occur, every landed blow in `[base−abs, base+spread−abs]` |
| AC-5 | `TestAlwaysHitsBeatsAnyDefence` — defence at the top of the range, to-hit at the bottom, the blow lands |
| AC-6 | `TestABlowThatCannotLandRemovesNothing/absorption above the damage…` — 200 advances, health never moves |
| AC-7 | `TestAnOutOfReachCycleRunsAndDrawsOnlyItsJitter` — two cells apart: one draw, no wound, and a cycle that still reached ready |
| AC-8 | `TestAnAttackOrderIsIgnoredWhereAnyOtherOrderWouldBe` — five refused shapes, each compared by **digest** against the same world advanced by no command, plus a positive control that must move |
| AC-9 | the downed-attacker and dead-attacker rows of the same test |
| AC-10 | `TestAWorldMidCycleAdvancesTheSameFromItsBytes` (60 advances, digest per tick, fixture asserted to be mid-charge) and `TestADecodedWorldStepsOnToTheSameDigests`, whose fixture now carries two units fighting |
| AC-11 | `TestUnmarshalRefusesAndLeavesTheReceiverExactlyAsItWas` — sixteen new cases, one per FR-8 refusal and one per record where a decoder checking only the first would pass. Each was run through a throwaway probe and **prints the message of the rule it names**, so none is refused incidentally |
| AC-12 | `TestWalkingAndAttackingAreOneState` — a mover holding a route and owing crossing ticks, ordered to attack then to move; residue checked in both directions, crossing checked to have advanced by exactly one |
| AC-13 | `TestTheLowerIdKillsAndTheHigherDrawsNothing` — the killing advance costs three draws, not six, and the higher id holds no order at the end of it |
| AC-14 | `TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth` (a partition of the whole form, twelve new rows per record, offsets written out by hand), `TestMarshalledBytesArePinned`, `TestHashIsPinned`, `TestThePinnedDigestIsFNV1aOfThePinnedBytes` |
| AC-15 | `TestReIssuingAnOrderDoesNotResetTheCycle` — the same victim for 40 advances lands blows; a victim alternating every advance lands none |

**P-1** `TestADoubledAttackOrderIsIdempotentAndADoubledDamageCommandIsNot`, twelve run lengths.
**P-2** the zero and negative rows of AC-2's table, which are what every loader-built world carries.
**P-3** AC-4, AC-6, AC-7 and the no-health-system sub-test.
**P-4** AC-8 measured on the digest.
**P-5** `TestTheCanonicalWorldsFieldSetsArePinned` (twelve new rows) with AC-10 and AC-14.
**P-6** AC-2 and AC-3 together.

## Success criteria

SC-1 → AC-2. SC-2 → AC-3. SC-3 → AC-1. SC-4 → AC-11. SC-5 → AC-10. SC-6 → P-1. SC-7 → AC-14.
SC-8 → AC-7. SC-9 → the gate above. SC-10 → AC-13. SC-11 → AC-15.

## Mutants, run in this seat

Each is applied to the tree, `go test -count=1 -trimpath ./pkg/sim` decides, the tree is restored.

The whole set was RE-RUN on the merged tree, which is what this step exists for: a merge that
silently un-kills a mutant is invisible to a green suite. Every kill held and two counts rose (M5
2→3, K1 7→10), so the merged suite is strictly stronger than either half. The last two are new and
are the merge's own boundary — the order of `0063`'s script phase and this story's attack phase
inside one advance, which no test could have pinned before both existed.

```
M1   the attack loop walked in DESCENDING id                     KILLED   (1 test)
M2   the reach test moved AFTER the two draws                    KILLED   (1 test)
M3   the charge floor weakened to a non-negative one             SURVIVED (argued equivalent)
M3a  the charge floor removed entirely                           SURVIVED (argued equivalent)
M4   an attack order always restarts the cycle                   KILLED   (1 test)
M5   uniform returns early at a bound of zero                    KILLED   (3 tests)
M6   the auto-hit band read as > rather than >=                  KILLED   (1 test)  [was SURVIVED]
M7   the relax floor removed                                     KILLED   (1 test)  [was SURVIVED]
M8   the hit test taken in int32                                 KILLED   (1 test)  [was SURVIVED]
M9   a dead victim no longer ends its attacker's order           KILLED   (2 tests)
M10  the damage sum taken in int32                               KILLED   (1 test)  [was SURVIVED]
M11  health wraps rather than saturating                         KILLED   (1 test)  [was SURVIVED]
M12  the third loop placed BEFORE the move loop                  KILLED   (4 tests)
K1   the blow subtracts nothing (a control)                      KILLED   (10 tests)
M13  the attack loop moved BEFORE the script switch              KILLED   (4 tests)
M14  the script switch removed from the advance                  KILLED   (16 tests)
```

Five survivors of the first round were real gaps and are closed by four tests added in the evidence
stage — the band at its own edge on four seeds chosen for the roll they produce, the relax floor as
an equality between two worlds rather than as a range, and both sums at the top of the `int32`
range, where a narrow one turns a certain hit into a certain miss and a killing blow into a
resurrection. That commit carries no task trailer: the green-but-hollow audit is part of Verify.

## What no test sees

- **The charge floor has no behavioural witness and cannot have one.** M3 and M3a survive because
  the decrement is guarded on owing something, so a charge of nought, one, or any negative load
  counts that fire on the same advance. The floor buys **representability** — no site can store a
  negative countdown, which the byte form refuses — and `chargeTicks`' own comment claimed
  behaviour it does not buy until this stage corrected it. FR-3's "a charge below one counts as
  one" is therefore true and unobservable; it becomes observable the day a site returns between the
  load and the blow.
- **No world this tree builds carries a combat number.** `FromALM` fills health, domain, speed and
  class and nothing else, so an attack ordered on a loaded map today runs a one-advance cycle and
  removes nothing. Every witness above is a synthetic world. **Filling those numbers from the class
  is out of scope and named as such**; `pkg/data`'s `UnitDef` already decodes all seven.
- **No end-to-end witness exists** — no front-end can issue the order, and nothing walks an
  attacker into reach. The deliverable is verified at the simulation's own boundary and nowhere
  above it.
- **The generator is ours.** Every draw's *shape* is decoded — the ranges, the inclusivity, the
  order — and every *value* is this package's SplitMix64, which no source claims is the engine's.
  No test here says a shipped fight would go this way, and none could.
- **The reach test is exercised at 1 and 2 cells only**, in a straight line. The diagonal is covered
  by the Chebyshev arithmetic and by no case.
- **`spec.md` and `plan.md` both sit within six bytes of their ceilings.** That is the gate's own
  signal that a story is large, and it is recorded rather than argued away: the contract carries
  eight requirements, fifteen criteria and a record layout, and the two artefacts were cut twice to
  fit rather than the ceiling being raised.

## Disclosures

**The debug commands are unchanged.** `KindKill` and `KindDamage` keep exactly the behaviour and
the doc they had, and the front-end seam that issues them is untouched.

**The doubled-apply hole is unchanged in size and is now witnessed rather than described.**
`0019`'s disclosure said a frame's commands applied twice inside one `Step` is undetectable, and
that this was safe only while no command accumulated. Two corrections. First, the premise was
already false when this story began: `KindDamage` landed in `0033` and accumulates, and `0033` did
not re-disclose it. Second, **this story does not widen the hole**: the attack order sets state, so
doubling it is the identity, and the damage is the cycle's, which runs once per advance by
construction. Both halves are now one passing test —
`TestADoubledAttackOrderIsIdempotentAndADoubledDamageCommandIsNot` — which states that a doubled
attack order changes nothing and that two damage commands remove twice the health. **Named
closer:** a story that narrows `KindDamage` to a set-health form, which that test will fail loudly
on rather than pass silently.

**The serialized form moved.** `formatVersion` → **10**, `entityLen` 44 → 83, the twelve new fields
at the record's tail from +44. **Version 9 is `0063`'s** — its mission-script section, which closes
the whole form after the routes — and this story took the next number free at merge rather than
choosing one in the lane. Every earlier version is refused and there is no migration. Every
hand-written width, offset, version byte and pinned digest in `pkg/sim` and `pkg/mapload` was
re-derived rather than regenerated, each keeping its superseded value in the note beside it;
`pkg/game` holds no pinned digest.

**The two newest sections are independent, and that is now a checked fact rather than a hope.**
`0063` appends after the routes; this story appends inside each entity record. Neither moved an
offset the other names. The merged `pkg/mapload` digests were re-derived by `0063`'s own recipe run
twice: lift the thirty-nine bytes out of every record, confirm each run is zero, put byte 0 back to
9, and the result hashes to `0063`'s externally-derived number **exactly**; strip that form's 1421
zeros, put byte 0 back to 8, and it hashes to `0059`'s externally-derived number **exactly**. So
the version-10 number is the externally-derived version-8 number plus two version bumps and two
runs of zeros, checked against two literals neither story's own code produced.

**A latent defect found in a file this story re-pinned — and found twice.** Three route-refusal
cases in `routeform_test.go` built their fixture at version 7 while the valid form was built at 8,
so each was refused by the version byte rather than by the rule it names. `0063` found and fixed
the same three sites independently while this lane did; the merged tree keeps the fix, and two
lanes arriving at it separately is the strongest evidence available that it was real.

**One thing the merge revealed that neither story could see alone.** `0063`'s
`TestTheScriptSectionRefusesTheBytesNoTickCanLeave` computes where the script section begins from a
hand-written `34 + 80*80 + 44 + 4` — and that `44` is the entity record's width, which this story
moves. Nothing but running the merged suite would have caught it: each story's own tree was green.
It is corrected to 83, with the number's meaning named beside it so the next record change moves it
in one place.

**Divergences, each named in `provenance.md` under *ours by choice*:** a blow whose damage after
absorption is not positive removes nothing (the engine's clamp on the physical component was not
read); the damage-kind reduction is not modelled, being identity for every unit this build can
construct; an order whose victim is dead ends rather than re-acquiring, the order machine being out
of scope; reach is a constant of one cell.
