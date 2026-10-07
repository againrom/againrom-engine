# Verification — 0055 selection count

## Gate

Run from the story's worktree at `5531d7c` — every artifact of this story in place, the commit after
it adding only the lines you are reading. One `&&`-chain, no pipes:

```
go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))"
EXIT=0
```

The FAIL set is empty and byte-identical to the pre-story baseline taken on the same worktree at
`49d9a85`. `builds/` is untracked, so `check-sdd-audit.sh`'s warning count is not comparable from a
worktree and only the FAIL set was compared. The one note this story draws is
`no plan.md yet (story in flight)`, which is the lightened process showing up in the gate rather
than a defect.

**One defect the gate caught, and it is worth the line.** The probe below began as a plain package
under the gitignored `builds/`, and `internal/archtest`'s DAG test failed it: that check walks the
whole module tree and is **fail-closed on any package not in its allow-map, gitignored or not**. The
fix is the convention `_probe-0028` already carries — a nested `go.mod` with `replace againrom
=> ../..`, which `dag.go` skips as its own module while the probe still calls the same exported
functions the shipped code calls. A gitignored file is not outside the gate.

## Where each criterion is witnessed

Every test below is in `pkg/ui`, is synthetic, and reads no game install. `selcount_test.go` is this
story's; the font, ink-layout, viewer and entity fixtures are `panel_test.go`'s and
`panel_draw_test.go`'s.

| # | Witness |
|---|---|
| AC-1 | `TestPanelCountFieldText` — 0, 1, 2, 17 through the one function that turns a field into text |
| AC-2 | `TestSingleSelectionPanelIsUnchanged` — byte-identical pictures at 1 and at 0 selected, against the authored layout with its count row **filtered out** rather than a hand-copied twin |
| AC-3 | `TestPanelStatesTheCount` — the four rows and their exact label/value pairs, plus a pixel comparison against the same unit at 1 selected, so the count is shown to reach the picture and not only the line list |
| AC-4 | `TestPanelCountsOnlyPresentUnits` — five ids, one dead, one absent: 3 stated, and the described unit is the lowest live one |
| AC-5 | `TestPanelRebuildsWhenTheCountChanges` — a death that moves only the count rebuilds once and restates 2; five following unchanged frames rebuild nothing |
| AC-6 | `TestPanelCountIsAnOrdinaryRow` — a count-less layout states no count for a four-unit subject and yields the same pixels as the authored layout at one selected; a placed row lands at its own offset under its own label, with the value at the label's pen plus the gap |
| AC-7 | `TestPanelCountNeedsASubjectAndAFont` — fontless viewer with two selected, empty selection, and a selection of ids the snapshot does not hold |
| P-1 | `TestPanelCountsOnlyPresentUnits`, and the code path: the count is the length of the slice the described unit is taken from, so no second reading of "present" exists to disagree |
| P-2 | `TestSingleSelectionPanelIsUnchanged` (composition) and `TestPanelCountIsAnOrdinaryRow`'s first subtest (the same identity reached from the layout side) |
| P-3 | `TestPanelCountIsAnOrdinaryRow`; `TestPanelIsAFunctionOfItsLayout`, which still holds and whose fresh-value assertion was made row-order independent by T1 |
| P-4 | `go test -trimpath -count=1 ./...` green across the tree with no test changed outside `pkg/ui`; the diff touches `pkg/ui` only, adds no entity field and widens no import set, so `internal/archtest`'s import check and `pkg/sim` source scan are unaffected and passed unchanged |

## Deletion check

```
git diff --diff-filter=D --name-only 49d9a85 b38eddf
(empty)
```

## Developer run against a lawful install

Both shipped roots, read-only, through `builds/_probe-0055` (a nested module under the gitignored
`builds/`; run from its own directory, asset root from `-assets`):

```
go run . -assets <root> -out <dir outside both repositories>
probe-0055: 34 named classes, lowest id 1 = "Unarmed Fighter"
probe-0055: ...\panel-selected-4.png  168x85 (x3), 4 selected
probe-0055: ...\panel-selected-1.png  168x67 (x3), 1 selected
```

Identical output from the RU root. The two boxes differ by exactly one line height plus the gap —
`85 − 67 = 18`, which is the font's 15px line and the layout's 3px gap — so the count row costs one
row and the rest of the box is where it was.

The story's own binary was run headless against both roots:

```
againrom.exe -check -assets ...\gameversions\en   ->  38 map rows, 8 of 8 buttons have a mask region
againrom.exe -check -assets ...\gameversions\ru   ->  34 map rows, 8 of 8 buttons have a mask region
```

Renders and the run note: `<seat>\review\0055-selection-count\` and
`builds/0055-selection-count/README.md`.

## What was NOT done, and is not claimed

This story ran under a lightened process by the orchestrator's instruction. There was **no separate
peer-prediction read, no separate adversarial read, and no mutation campaign**; no `plan.md` and no
`analysis.md` were written. The tests above were run, and nothing beyond them is asserted.

The windowed drag-select of the build README's step 3 was **not** performed by the implementer —
this is a headless environment. What was verified is the composition and the viewer path that feeds
it, in tests, plus the two renders through the shipped seam against both lawful installs.
