# Verification — 0109

Four tasks, four commits, one trailer each. The gate below belongs to the tip of the branch, run on
a clean tree; the two measurements against an install were taken with binaries built from it.

## The commits

```
6a74006  0109 T1: the six fields, the fault and version 25       SDD-Task: 0109-regeneration/T1
7309357  0109 T2: the regeneration pass                          SDD-Task: 0109-regeneration/T2
cd746b8  0109 T3: a placement is born with its periods and pool  SDD-Task: 0109-regeneration/T3
a350cc9  0109 T4: the panel says how much mana                   SDD-Task: 0109-regeneration/T4
```

No `Co-Authored-By` trailer on any of them. The deletion set `master..HEAD` is empty.

## The gate

```
go build ./...                     0
go vet ./...                       0
gofmt -l $(git ls-files '*.go')    (no output)
go test -trimpath -count=1 ./...   0
check-no-game-assets.sh            0
check-doc-budget.sh                0
```

`check-sdd-audit.sh` returns early for a story with an unlanded task, so it is meaningful only from
the seat after this file exists. From a worktree its note and warning **counts** are meaningless in
both directions — there is no `builds/` here — and only its FAIL set is comparable.

## The version

`formatVersion` was **24** at the branch point and this story spends **25**. `entityLen` 127 -> 145,
the eighteen bytes at the entity record's tail. Both were read off the branch point rather than
taken from the plan, which says to.

## What witnesses each id

- **AC-1** `TestAQualifyingTickAddsExactlyTheDecodedHundredths` and
  `TestTheHealthArmMovesOneFullTickInFourAndManaOnAllFour`.
- **AC-2** `TestAGainUnderOnePointStillReachesTheMaximum` — 40 to 45 at period 100, the six-tick
  sequence transcribed. It is the one assertion that separates this pass from one that heals nobody.
- **AC-3** `TestNeitherPoolExceedsItsMaximum`. **AC-4**
  `TestANonPositivePeriodRegeneratesNothingOnItsOwnArm`. **AC-5** `TestANotAliveEntityGainsNothing`.
- **AC-6** `TestADecodedRegenerationRecordCrossesTheFormWholeAtBothExtremes`,
  `TestThePreviousVersionFormIsRefused`, `TestUnmarshalRefusesEveryVersionButTheCurrentOne` and
  `TestThePinIsThePreStoryPinPlusTheRegen` — the last is the stripped form, version byte restored,
  hashing to the previous story's digest.
- **AC-7** `TestAResolvedUnitsPlacementCarriesItsRowsPeriodsAndPool`,
  `TestTheHumansArmCarriesTheConstructorsPeriodsWithTheRowsPool` and
  `TestAStartedPartyMemberCarriesTheConstructorsPeriodsAndNoPool`.
- **AC-8** `TestPanelManaRowFollowsTheHealthRowsForm`, with `TestPanelSubjectCarriesTheManaPair` and
  `TestTheSeamCarriesTheWorldsOwnManaPool` for the wiring behind it.
- **P-1** `TestRegenerationDrawsNoRandomness`, plus `internal/archtest`'s source scan over `pkg/sim`.
- **P-2** and **P-3** `TestNeitherPoolExceedsItsMaximum`,
  `TestANegativeMaximumRegeneratesNothingOnEitherArm` and `TestANegativePoolLeavesALegalRemainder`.
- **P-4** `TestAWorldWithNoPeriodMovesNoPinnedDigest`, and the whole suite: no digest outside T1's
  and T3's own re-pins moved.
- **P-5** `TestACappingTickStillStoresItsRemainder` reads the remainder the pass wrote and no other
  writer exists; the mutation sweep below is what establishes that reverting the store is visible.
- **DD-9** is witnessed by absence: no clause was added to the not-alive block, and no digest moved
  for one. **DD-11** by `TestUnmarshalRefusesEveryVersionButTheCurrentOne`'s own name.
- **AC-9** and **AC-10** are the two measurements below.

## AC-9 — the milestone did not move, and the reason is measured rather than assumed

The drive is `-mission 10 -census -waypoint u21:56:21:3 -waypoint p0:66:16:3`, the milestone's own
argv, against both installed roots:

```
en  outcome lost at tick 272
en  census: 4 of 36 unit(s) moved, 1 fell, over 272 tick(s)
ru  outcome lost at tick 272
ru  census: 4 of 36 unit(s) moved, 1 fell, over 272 tick(s)
```

AC-9 predicts `lost` at a tick no earlier than 272, identically on both roots. It holds: 272, both
roots. The whole per-unit block is byte-identical to the same drive built from `master`, u21's
closing `-3 hp` included.

**That identity is not evidence that the pass is dead, and it was not taken on trust.** A throwaway
`println` inside `regenerate`, built into `missionrun` and reverted afterwards, counted the drive's
own dispatches: the pass dispatches **17** times (272/16 full ticks), of which 5 carry the health
arm, and exactly **one** call gets past the three gates — at tick 204, on a unit standing at 16 of
45 with a period of 100. Its gain is 45*2*100/100 = 90 hundredths. Nine tenths of a point is not a
point: the quotient is still 16 and only the remainder byte moves, so no observable value changes
and the drive prints what it printed before.

Two facts explain the rest, and both are measured below: nothing on mission 10 is born wounded, so
the `cur >= max` gate holds until a blow lands; and no placement on it carries a mana pool, so the
mana arm never gets past its own maximum gate at all.

The seat's `pipeline/milestone-baseline.txt` therefore does not need to move. It was not touched.

## AC-10 — the census, both roots

`classdump -regen <root> 10.alm`, a new verb written for this measurement (below). Identical on `en`
and `ru`:

```
parameterised rows         56        10.alm                     35
  mana maximum above zero    5         mana maximum above zero    0
  health period <= 0         2         health period <= 0         0
  mana period <= 0          50         mana period <= 0          19
  health gain truncates to 0 0         health gain truncates to 0 0
  mana gain truncates to 0   0         mana gain truncates to 0   0
  born below health max      0         born below health max      0
  a mana pool and NO period  0         a mana pool and NO period  0
                                       one qualifying tick adds 216
                                       hundredths to its widest pool
```

**Five of the 56 parameterised Units rows carry a mana maximum above zero, and mission 10 places
none of them.** The mana arm is therefore unobservable on the mission this project drives — it is
witnessed by the package's own tests and by nothing on the shipped corpus's tenth map. That is the
answer to the question the contract left open, and it is a fact about the corpus rather than a gap
in the work.

Three further figures the census settled:

- **DD-2's premise is measured, not assumed.** Zero rows pair a mana maximum above zero with a mana
  period at or below zero. The unreachable-divide argument rests on exactly this count.
- **No row and no placement has a gain that truncates to zero.** The clause under FR-4 about a
  period above a hundred times the doubled maximum describes a case the corpus does not contain.
- **Nothing is born wounded.** Every placement starts at its own maximum, so regeneration is
  invisible on a map until something takes a blow — which is why an undriven mission shows nothing.

Two rows carry no health period and never regenerate health; 19 of mission 10's 35 placements carry
no mana period, which matters to nothing because none of them carries a pool. `missionrun` counts 36
entities against the census's 35 placements: the extra one is the party, which the map does not
place.

## Mutation testing

Twenty-two reverts, each applied to the committed tree and undone afterwards. **Twenty were killed
on the first pass; two survived and are now killed by tests added in this stage.**

Killed by the task tests: the `max <= 0` gate (DD-10), the remainder store, the remainder read,
cancelling the two hundreds, the `Alive()` gate, the `period <= 0` gate, the cap, the health arm's
doubling, the health cadence, `regenPhase`, the call's position before `decayPass`, `regenFault` on
decode, the constructor's fold, the unresolved arm's periods, the party mint's periods, the creature
arm's mana pair, the panel's `MaxMana > 0`, the seam's mana pair, one tail byte on encode, and
`formatVersion`.

**Survivor M8 — the DD-7 non-negative remainder — was unwitnessed.** Reverting the three-line
adjustment left the whole suite green. It is not equivalent: Go's `%` takes its left operand's sign,
so an accumulator below zero answers a negative remainder, `uint8(-55)` is 201, and 201 is a value
`regenFault` refuses — the pass could drive a world into a state its own decoder rejects.
`TestANegativePoolLeavesALegalRemainder` now witnesses it, and asserts the round trip rather than
only the number.

**Survivor M12 — the remainder stored on a capping tick — was unwitnessed.** Storing only on the
uncapped path left the suite green. Also not equivalent: the remainder is hashed and serialised, so
a skipped store leaves a stale fraction under the pool. `TestACappingTickStillStoresItsRemainder`
witnesses it.

Both mutations were re-applied after the two tests landed and both are now killed. The shape they
share is the one the seat asked to look for: a criterion whose effect is covered for by a gate later
in the same call, so nothing downstream ever reads what it wrote.

One mutation is worth recording for a different reason. Removing the `period <= 0` gate **panics**
on a fixture elsewhere in the package, which aborts the test binary before the intended witness
runs; the witness had to be confirmed by re-running that one revert under `-run`. A revert that
kills the run is not the same as a revert the assertion caught.

## What the task tests did not look at, and the two things that came of it

**T1's fence was too narrow and the tree was left red.** `pkg/sim/{actorform,reach}_test.go` each
spell the current version as a literal `"24"` inside a refusal assertion — DD-11's disease in a
place DD-11 did not name, since it is in a body rather than in a name. Both were repaired in T1's
own commit by reading `formatVersion` through `strconv` while leaving the refused version, which is
that fixture's own, a literal.

**T3's change moved a landed test in a package T3 may not touch.** Every world the loader builds now
carries a health period, so `pkg/game`'s `TestOneCommandStreamReachesOneDigestWhateverIsDrawnBetween
ItsTicks` stopped observing its `Downed` life state: its fixture damages entity 1 by 60 and then 40
against a maximum of 100, and the unit now regenerates four qualifying ticks' worth — 8 whole points
— in between, so the second blow no longer lands it on exactly zero. The second damage moved to
**48**, with the derivation written beside it. What moved is the fixture, not the seam. The task
entry's file list could not contain that change; it was folded into T3 rather than widened silently,
and this paragraph is the record.

Every other landed test that moved moved as a **pin**: the digests re-taken in T1 and T3, listed in
those two commit bodies.

## The census tool

`cmd/classdump` gained a `-regen <asset-root> [<map name>]` verb, and `openCampaign` gained the four
collections `FromALMWith` reads. It prints counts and nothing else — no name, no parameter, no
per-row value (golden rule 2). It may not name `pkg/sim`: `cmd/classdump -> pkg/sim` is a DAG
violation and `internal/archtest` failed on the first draft, which named `*sim.World` in a helper's
signature; the entities are now ranged over with their type inferred.

## Still open

- **The idle bonus (FR-5) is cut and the corpus cannot say what it costs.** `regenRate` is 1 for
  every actor on every tick. Nothing measures the divergence because nothing in this tree writes an
  action's due end.
- **The mana arm has no witness on either startable map.** `20.alm` was censused too — 56
  placements, **0** with a mana maximum, both roots — so neither mission 10 nor mission 20 places
  one of the five rows that carry a pool. The panel's MANA row therefore never draws in play, and
  the arm and the row are witnessed by `pkg/sim`, `pkg/ui` and `pkg/game` tests alone. Which of the
  other 36 shipped maps places one is unmeasured; the verb answers it one map at a time.
