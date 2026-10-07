# Hotfix `DIV-385` — verification

## Adversarial pass 1

The independent pass reviewed pushed SHA
`d1f38cffa7071b447c9cecb36e086f2d27a1cfd6` at its exact tree, based on
`dddae8cd96c892ac519a66ba2434fd69ab1efcd8` and research pin
`d7ee0c62cfa4a16083d356f24b1870035e1f0209`.

It found one class-P counterexample. A production script with instant 30, a matching attached effect
and authored duration zero left the record at `Remaining=0` after the phase-6 script pass. An
immediate `MarshalBinary` succeeded, but `UnmarshalBinary` rejected the same canonical record as
having an empty field. This was player-visible at a mid-tick save/load boundary and also prevented
the saved hashed state from being restored.

The original hotfix introduced the zero write in `pkg/sim/effect.go`; the rejecting predicate in
`pkg/sim/castbinary.go` predates the hotfix. The feature made that adjacent persistence defect
reachable. The minimal correction in `15e3ef68` permits zero only for attached `Remaining`.
Attached spell and kind remain required.

## Exhaustive class population

Searches covered every `Remaining == 0` decoder or validator and every attached-effect binary
producer and consumer. There is one attached-effect empty-field predicate. Its spell and kind arms
remain enforced by malformed-record tests. Pending casts, book casts and cell effects are different
record families whose zero-duration rejection remains required. No upgrade path duplicates the
attached predicate, and the byte-form version is unchanged.

`TestInstantThirtyZeroDurationRoundTripsBeforeNextEffectPass` supplies the production reproducer. It
authors duration zero through `NewScript` and `StepTraced`, reaches both matching target records with
stored spell ids 20 and 276, immediately proves exact form and hash equality after load, and proves
the next ordinary effect pass removes both. The attached-effect decoder test directly fixes the
malformed-record boundary.

## Correction gates

The focused simulation command passes:

`go test ./pkg/sim -run 'TestInstantThirty|TestAttachedEffectDecoder' -count=1`

The complete corrected tree passes `go build ./...`, `go vet ./...`, the repository-wide formatting
scan and `go test ./... -count=1`. The release gate selects and passes all 49 install-gated tests on
each lawful root with zero skips. The scenario gate passes all 15 scenarios on each root. The exact
campaign closure passes on EN with `maps=28 checks=680 instants=759 triggers=398 rows=52 synthetic=9`
and on RU with `maps=28 checks=678 instants=759 triggers=397 rows=52 synthetic=9`; each reports 181
natural exact instants. The preserved-install gate still measures 162 unchanged files. The game-asset
and claim-citation gates pass.

The correction still requires an independent pass 2. This document records the pass-1 finding and
its correction; it is not a self-review.
