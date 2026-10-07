# Verification

Accepted base: `92e338c3df704130989a2fecec0a91fc04cdab05` (1164).
Knowledge: `abdefd56d239c6f203889c0d77521045e3b1012a`.
The reconciliation preserved both appended divergence blocks and all inherited
simulation and persistence code. This story adds no simulation or save changes.

## Observable result

The existing read-only `review/wall-occlusion/probe.go` and `ui_probe.go`
instrument was rebuilt against this worktree using its source overlay. At
mission101 tick8, EN and RU production `Viewer.drawArt` submissions render Earth
over the tree trunk. The measured cell `(22,19)`, ground `(720,620)`, tree
rectangle `(656,524)..(784,652)` and Earth rectangle
`(704,588)..(736,652)` are unchanged. The pair has 788 changed opaque-overlap
pixels. Production crop SHA256 is
`12aa4d450f53a3a76e34ab16ef6067106ccac3617ad596bb1e42061525409a49`
in both roots; baseline was
`b751d9759456e08fb7dd172c854301d1b77bf18b5c8c13c2cd3589f9c169ba27`.
The crop was visually inspected. Artifacts remain untracked at
`review/wall-occlusion/1165/{en,ru}/` in the seat.

The authored Fire trigger still has zero painted static overlaps at tick24 in
both roots. Its positive overlap witness is declared installed art at a shared
cell, not an observed authored Fire/tree collision. These are CPU composites of
production submissions, not native-window or original-game pixel captures.

## Focused checks

The reconciled focused run passed in `pkg/ui`, `pkg/game`, `pkg/render/terrain`
and `internal/gatedtests`, with `-trimpath -count=1` and this test selector:

```text
Test(CellArt|Inspection|DepthOrder|AMovingUnit|AStationaryUnit|DrawArt|TheArtSwitches|EntityDrawCategory|LoadUnitsCarries|RetainedOverlay|OrdinaryObjectUsingRetainedArt|SpellPass|Shadow|ScanMatches)
```

`TestDepthOrderMergesEntitiesByRow`,
`TestDrawArtPutsAUnitInTheRowOrderOfTheArtPlanes`, depth-move and corpse/sack
checks remain. New production composites cover earlier/later actor rows against
both trees and buildings, both column directions, alternate and air exceptions,
and the native corpse/sack/live tie. All12 actor/scenery composites also check
inspection at their actual opaque overlap; the existing pointer, transparency
and fog tests pass. Loader/projection tests cover inherited Z, original-class
identity through composed/corpse substitutions and corpse stages. A live Fire
clock with nil or empty owned Cells emits no sprite; another effect keeps its
explicit cell even when that cell differs from the shared anchor.

Focused installed tests passed separately on absolute EN and RU roots:
`go test -trimpath -count=1 ./pkg/ui ./pkg/game -run 'TestRelease(Walls|SpellPasses|UnitDrawCategories)' -v`.
Both roots agree:

| witness | measured result |
|---|---|
| Earth/tree, active shadow | 65536 pixels compared; 1183 painted, 865 transparent wall pixels; 788 opaque-tree overlap pixels changed |
| Fire/tree positive control | 65536 pixels compared; 2549 painted, 13835 transparent wall pixels; 1805 opaque-tree overlap pixels changed |
| clipped Earth / Fire | 1120 pixels each; 627 / 940 changed opaque-tree overlap pixels; 358 / 12 transparent wall pixels |
| earlier/later row and column controls | Earth 22/901/175/156; Fire 994/897/360/361 raw overlap pixels compared |
| Freezing/Poison and synthetic bodies in three categories | 441 raw-oracle pixels per case, 6 cases per locale |
| installed unit classes | 34 classes; nonzero Z only 70/71, both 96; air at live and late-corpse stages |

The three new installed test functions are registered in
`internal/gatedtests/testdata/population.txt`. Its live scanner passes with245
entries, up from the accepted base's242. Focused logs and rebuilt production
probe receipts remain under `review/wall-occlusion/1165/` in the seat.

`missionrun -mission <10|20> -trace -ticks 1`, built from this worktree and
run on EN, prints 0/0 `UNSUPPORTED` lines. The master milestone baseline has no
unsupported nodes for either mission; this renderer slice leaves the census
unchanged. Its observable result is the production image above.

`gofmt` and `git diff --check` are clean; `check-no-game-assets.sh` prints
`clean (tree scan)`. Allocation sweeps before/after the ledger edit both report
`missing answers: 0`; the reconciled sweep agrees. With `AGAINROM_IMPL` set to
this worktree, `pipeline/check-div-claims.sh` parses all426 live rows at k13 and
reports118 advisory retracted-claim citations; the changed citations were read
with their narrowed clauses. No files were deleted and no install was written.

The seat will commission the sole independent review and run the final full-Go,
paired-release and relevant scenario gates before landing and rebuilding
`builds/current/`. This pushed candidate is not that landing.
