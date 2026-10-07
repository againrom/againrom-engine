# Verification — 0043 the mover's domain

Toolchain `go 1.26.1`, submodule pin `e47abfb`, branch `lane-a` rebased onto `origin/master`
`1768537`. `-trimpath` throughout; every run is on the finished tree.

## The gate

```
go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))"

go list ./... | wc -l        30
git ls-files '*.go' | wc -l  292
packages reporting ok        27
```

## The criteria, and the tests that carry them

```
--- PASS: TestTheZeroValueIsTheGroundDomain                          AC-1
--- PASS: TestTheThreeDomainsAreTheirDocumentedValues                AC-1
--- PASS: TestTheConstructorRefusesAnUndefinedDomain                 AC-1
--- PASS: TestEachDomainCrossesItsOwnTerrain                         AC-2
--- PASS: TestNoDomainCrossesTheBounds                               AC-2
--- PASS: TestOnlyTheMoversOfOneLayerContend                         AC-3
--- PASS: TestALayerIsCountedSeparatelyAtEveryCell                   AC-3
--- PASS: TestTwoMovingFlyersInterpenetrate                          AC-4
--- PASS: TestAFlyerEntersItsLayerWhenItsOrderEnds                   AC-4
--- PASS: TestAFlyerCrossesARestingFlyersCell                        AC-5
--- PASS: TestTwoFlyersOrderedOntoOneCellEndDistinct                 AC-6
--- PASS: TestAFlyerOrderedOntoARestingFlyerSettlesBeside            AC-7
--- PASS: TestASettleSkipsARingCellAnotherFlyerRestsOn               AC-7
--- PASS: TestAFlyerNeverStandsOnAnAntiAirCell                       AC-8
--- PASS: TestEveryDomainRoundTripsThroughTheForm                    AC-9, P-3
--- PASS: TestUnmarshalRefusesAnUndefinedDomainAndLeavesTheReceiver  AC-9
--- PASS: TestARouteIsCheckedAgainstItsOwnMoversDomain               AC-9
--- PASS: TestAWorldOfEveryDomainIsDeterministic                     AC-10
--- PASS: TestAFlyerReOrderedOntoItsOwnCellEndsCounted               AC-11
--- PASS: TestAFlyerSettlingOnItsOwnCellEndsItsOrder                 AC-11
--- PASS: TestAFlyerWhoseSearchFailsRestsWhereItStands               AC-11
--- PASS: TestTheNonSettlingSearchRefusesAnOccupiedGoal              AC-13
--- PASS: TestTheTwoRelationsDifferExactlyOnOccupancy                P-2
```

What each **discriminates**, where that is not obvious from the name:

- **AC-1** sweeps `defined()` over the whole byte range, so a fourth domain defined anywhere above 2
  fails rather than passing quietly, and 3/4/127/128/255 are each refused with no world returned.
- **AC-2** records **nine** answers — three domains against the three bytes a derivation can produce
  — so no domain's mask is read off another's. bit 0 stops ground only; bit 1 stops ghost and air
  only; both stop all three.
- **AC-3** asks the predicate directly at a cell each domain stands on: six further answers.
- **AC-8**'s wall is bit 1 with one gap. The flyer never stands on a walled cell and still arrives;
  a ground mover on the same grid walks straight through it, which is what says the wall is bit 1.
- **AC-9** spoils the domain byte at **both** records' `+34`, so a decoder checking only the first
  is caught, and shows the three domains' forms pairwise distinct (P-3).
- **AC-13** also orders the same mover to a **free** goal and sees it route, so the refusal above is
  the rest term and not that mode refusing everything.

## AC-6's tie-break, as measured

Two flyers ordered onto one cell end distinct, one on it and one adjacent, with no tick holding two
**resting** flyers on a cell. **Which one takes it is not id order in general** — the nearer flyer
arrives first — so the fixture is run twice and the tie-break is asserted only in the equidistant
one, where the lower id wins.

`TestASettleSkipsARingCellAnotherFlyerRestsOn` is the fixture AC-7's simpler one cannot answer:
two flyers rest side by side and a third is ordered onto the further, so the cheapest ring cell by
label **is** the nearer flyer's own cell. The centre is never probed, which is why the simple
fixture leaves the settle filter's mutant alive.

## The ground contrast is not what the plan assumed

On the head-on corridor two ground units never share a cell **and never arrive**: a near search
refuses a sub-goal another unit stands on, so the pair holds, stalls and gives up. On the pass-by
fixture the ground mover **does not move at all**. Both are shipped behaviour this story does not
touch, and both make the fixtures ones a hard collision visibly fails.

## AC-12, SC-6 — nothing outside `pkg/sim` moved

Census over a synthetic 40x32 map with interior water, a mountain word and a scenery overlay, from a
program outside the repo. No game install is read.

```
map                       40x32 = 1280 cells
blocks ground             906
blocks air                896
cells in the border ring  896
cells with bit 1 set      896
  bit 1 and NOT border    0
  border and NOT bit 1    0
interior arms: water 8, mountain 1, scenery 1, impassable 0
BorderCell true           896
  margin and NOT bit 1    0
  bit 1 and NOT margin    0
entities built by FromALM 2
  ground / ghost / air    2 / 0 / 0
```

Bit 1 is the border ring exactly, 0 discrepancies either way; the margin predicate agrees cell for
cell; the ten interior blocked cells raise the ground count without touching bit 1; and both
map-built entities are **ground**. The story added no writer of bit 1.

## P-1

`restFree` reads the grid byte, the asking mover's domain and the two counts and nothing else.
`internal/archtest`'s source scan over `pkg/sim` is green, so no float, `os`, `time` or `math/rand`
entered on this path, and no map is on it.

## SC-8 — the mutants

Each applied alone to production code, the whole tree run, then reverted.

```
1 the ghost's mask set to bit 0              KILLED  TestEachDomainCrossesItsOwnTerrain,
                                                     TestARouteIsCheckedAgainstItsOwnMoversDomain
2 the air seed made unconditional            KILLED  TestAFlyerEntersItsLayerWhenItsOrderEnds
3 near search given the unit relation, air   KILLED  TestAFlyerCrossesARestingFlyersCell
4 moved called for an air mover too          KILLED  TestTwoMovingFlyersInterpenetrate
5 restAt reduced to a plain order-clear      KILLED  TestTwoFlyersOrderedOntoOneCellEndDistinct
6 the settle candidate filter dropped        KILLED  TestASettleSkipsARingCellAnotherFlyerRestsOn
7 the fourth staleness test removed          KILLED  TestTwoFlyersOrderedOntoOneCellEndDistinct
8 the non-settling search's rest term gone   KILLED  TestTheNonSettlingSearchRefusesAnOccupiedGoal
9 decodeRoutes holding every domain to bit 0 KILLED  TestARouteIsCheckedAgainstItsOwnMoversDomain
```

**Three survived a first run and the record says so.** 4 and 5 were run at the soft-collision slice,
where nothing yet READ the air plane: a corrupted count changed no outcome, so they were moved to
the slice that added the rest test and killed there. 6 survived against AC-7's simpler fixture and
was killed once the fixture above existed.

## SC-1 … SC-7

SC-1 is AC-1 and AC-9; SC-2 is AC-2; SC-3 is AC-3; SC-4 is AC-4 and AC-5; SC-5 is AC-6, AC-7 and
AC-11 together, the settle onto the mover's own cell included; SC-5a is AC-13; SC-6 is AC-12 above;
SC-7 is AC-10.

## The digests, re-derived

The version and the record width moved every pinned digest and byte string. None was read off the
encoder: each was assembled from the documented layout by a program outside the tree and hashed
there, and each agrees with what this build produces.

```
pkg/sim/hash_test.go        pinDigest       0x4835b2e24f792300 -> 0x0b5bd2c163debaa0
pkg/sim/routeform_test.go   rtfDigest       0x6fc8c538db18c544 -> 0x695327a5e8b998a9
pkg/sim/relaxation_test.go  rlxTick1Digest  0xf34baf1219fd7838 -> 0x17798c4a42be56d3
pkg/sim/relaxation_test.go  hybTick1Digest  0x1ca4f71915ecc6de -> 0xeefafea1ba8826c7
pkg/mapload/gridform_test.go gfDigest       0x578ed5f38a8547b1 -> 0x435d47015db7490a
```

`pinBytes` and `rtfBytes` gained a domain byte per record and their version byte; the two relaxation
forms and the map-built form were assembled field by field from the byte-form table. Exactly one of
the two hand-written 34s moved: the header stays 34, the record is 35.

One landed criterion is superseded and amended where it is written rather than left to be found:
`binary_test.go`'s version-4 stream no longer has a length this build's arithmetic accepts, so its
case can no longer isolate the version byte. The note says so, a check asserts the coincidence is
gone, and a body well formed for **this** version with only its first byte spoiled took over the
isolating job.

## What is NOT witnessed, and why

- **A flyer on a shipped map.** The column that decides a domain is `Data.bin`'s `movementType`,
  which this tree has no reader for, so `FromALM` builds ground movers only — pinned above as
  `2 / 0 / 0`. Every flyer in this evidence is a hand-built entity.
- **A ghost stopped by an object.** This plane derives no bit 2; `provenance.md` records it open
  with the engine's own 229 092-cell figure beside it.
- **The disclosed overlap on a map.** It needs a flyer terrain-sealed in the air domain, which on a
  derived plane is the border alone, so the fixture is synthetic by necessity.

## Scope reconciliation

Two corrections were cascaded upstream rather than absorbed here. The adversarial plan read found
that the search which does not settle had no rest term, so **FR-6 grew a fourth site** and AC-13 was
added. And `pkg/mapload/gridform_test.go` had to move, so **DD-14 now says no PRODUCTION file
outside `pkg/sim` changes** and names the one test file that does.

AC-8, AC-10 and AC-7's discriminating fixture were written at this stage rather than inside a task,
against behaviour already built and unchanged by them. They land in this stage's commit, which
carries no trailer.

## The audit's FAIL set at push

Full sweep at this tip, unscoped:

```
check-sdd-audit: ok (77 note(s)/warning(s), none enforced)
```

**EMPTY** — no FAIL line, mine or anyone else's, which is the recorded baseline. The notes are the
standing ones: absent `builds/` READMEs, that tree being gitignored, and the pre-`0014` stories'
unenforced witness debt.

## Appended 2026-08-01 — the engine figure quoted above has moved

*What is NOT witnessed* quotes "the engine's own 229 092-cell figure". At research `130bb79` that
census is re-scoped: 229 092 counts the **ingest** block plane, and after the structure pass the
`0x44` mover is blocked on **242 179** of 880 704 (`MOVE-DOM-026`; `TERR-PASS-051` in
`retracted.md`, classed **SUPERSEDED** — the plane was named later, the measurement was not wrong).

Nothing witnessed in this file moves with it: no evidence here was measured against that number,
which is quoted only to size the gap this tree carries. The gap is larger than recorded —
**83 203** cells between the ghost mask and the air mask, not 70 116. `provenance.md` carries the
correction in full.
