# round3-cone.md — fix round 3, the cone origin

Branch `story/1001-r3-cone`, based on `4519a76` (`docs(1001): fold r3-effect into spec.md and the
divergence ledger`). This document covers one defect: Acid Stream's cone did not start on the
caster's own cell. The seat folds the proposed `spec.md` text and `docs/DIVERGENCES.md` rows into
the canonical documents at the landing; this lane did not edit `spec.md`, `closure.md` or
`docs/DIVERGENCES.md`.

All commands below are run from the worktree root, `<seat>\wt-1001-cone`.

## Owner report

2026-08-15, with a screenshot: «конус кислоты правильный, но почему-то его начало не по
центральной клетке от персонажа, а смещено влево (в других проекциях тоже неверно)». The Acid
Stream cone's shape is right; its start is not the caster's own cell, and other orientations are
wrong the same way.

## The handed diagnosis, verified

The brief handed this location, with its own evidence, and asked me to check the evidence before
building on it. I did, and it holds.

`landAreaFacing` (`pkg/sim/celleffect.go`, then at line 404) built the standing record as
`cellEffect{Key: cellKey(x, y), ..., Direction: areaDirection(x-fromX, y-fromY)}`, where `(x, y)` is
the aimed cell and `(fromX, fromY)` is the caster's own cell at cast time. The caster's position
entered only through `areaDirection`, which recovers a facing byte from the SIGN of the delta and
contributes to `Direction` alone.

`ringStageCells` (`pkg/sim/celleffect.go:647`) reads `cx, cy := keyCell(e.Key)` and builds every
stage's cells as `cx+dx, cy+dy` for the row's own offset table. With `e.Key` set from the aimed
cell, stage zero — and every later stage, since the tick-advance pass at `celleffect.go:243` rereads
`keyCell(e.Key)` from the STORED record on every tick — was algebraically anchored at the aimed
cell, whatever the orientation.

`areaEffectDraws` (`pkg/game/spellbolt.go:414`) iterates `e.Cells` — the painted footprint the
simulation reports — and draws one sprite per cell, at that cell's own position. It does not read
the exported `CellEffect.X`/`.Y` at all. Confirmed by reading the function: the client draws exactly
the cell set the simulation reports, with no further transform. The defect is in the cell set, not
in the drawing.

I additionally reproduced the diagnosis's own reproduction, through the real production path, before
writing the fix: see "Witness by reverting" below — all eight orientations reddened with the record's
own cell reading the clicked cell, not the caster's.

## The fix

`landAreaFacing` now decides an anchor cell, `anchorX, anchorY`, from the rule, before building the
record:

- For every row but Acid Stream (`rule.ID == 9`), the anchor is the aimed cell, unchanged.
- For Acid Stream, the anchor is the caster's own cell (`fromX, fromY`). The aimed cell still
  supplies orientation alone, through the unchanged `areaDirection(x-fromX, y-fromY)` call.

The anchor feeds both the six-slot admission refusal asked inside `landAreaFacing`
(`areaLandingRefusal(rule, power, anchorX, anchorY)`) and `e.Key` (`cellKey(anchorX, anchorY)`).
Setting `e.Key` once at creation is sufficient: the tick-advance pass rereads `keyCell(e.Key)` from
the stored record on every later stage, so every stage of a standing Acid Stream record stays
anchored at the caster's cell without any change to that pass. `ringStageCells` itself is untouched;
its existing unit test (`TestStagedAreaProgramsUseTheirDecodedOrderedCells`) still validates the
generator's own offset arithmetic against a hand-built record, unaffected by where the caller now
sources the anchor from.

The caller does not change: `castBookAt`, `castSpell` and `landAreaCast` still pass their own aimed
cell and the caster's own position exactly as before; the anchor decision lives entirely inside
`landAreaFacing`, from `rule.ID` alone.

## Fire Sacrifice: settled from the data, not touched

The brief asked me to read Fire Sacrifice's installed row's own target kind rather than assume it.

`TargetsUnit` (`pkg/data/spell.go`) is slot 4, "Spell Target", true only when the cell is exactly 1.
Fire Sacrifice's row carries `Spell Target = 2` (`research/experiments/EXP-0065-magic/evidence/
spell-table.txt`, row 4), so `TargetsUnit` is false: it is a point/cell-aimed row, not a
unit-targeted one, by that column alone.

The column that settles self-targeting is a different one: `MaxRange`, slot 6, "Max Range". Fire
Sacrifice's row carries `Max Range = 0` — the same value, and the same column, Shield carries
(`spell-table.txt` row 18; Shield is an unambiguous self-buff). `spellRange` (`pkg/sim/spell.go:662`)
returns 0 whenever `rule.MaxRange == 0`, with a doc comment naming exactly these two rows: "a zero
maximum means no range bonus is computed at all. Shield and Fire Sacrifice ship on this arm."

Every admission gate that reaches a Fire Sacrifice cast tests the caster-to-aim Chebyshev distance
against `spellRange(rule, power)`: `castBookAt` at `spell.go:971` and `castSpell` at `spell.go:837`.
With `spellRange` forced to 0, the aimed cell must be at distance 0 from the caster — the SAME
cell — or the cast is refused before it reaches `landAreaFacing`. So the aimed cell Fire Sacrifice's
row can ever reach `landAreaFacing` with already equals the caster's own cell, through every
reachable book-cast path.

I independently cross-checked this against the live EN install rather than the research evidence
file alone: `go run ./cmd/spelleffectcheck` against `gameversions/en` prints Fire Sacrifice's row
params with a `0` at the same index Shield's own row carries `0` and Acid Stream's own row carries
`3`, over the actual shipped `Data.bin`.

**Conclusion: Fire Sacrifice is self-targeted by its own row's columns (`Spell Target = 2` and
`Max Range = 0`, matching Shield's own `Max Range`), and its anchor needs no change.** I made none;
`rule.ID == 4` takes the unchanged branch. The owner's separate report — «отрисовка огненной жертвы
неправильная, взрыв должен растекаться от центра, где стоит персонаж» — is therefore about the
DRAWING (the sprite/animation placement for the two fixed shells), not about the cell set
`landAreaFacing` builds, which already lands at the caster's cell. That belongs to a later
presentation story and is left alone here, per the brief's own instruction.

## Research position, verified

Read directly rather than by ledger: `cd research && go run ./tools/claim MAGIC-RING-048`.

The claim states the three ring generators' offsets are relative to "the target cell," in the sense
`ringStageCells` itself already implements — an offset table walked from one fixed anchor. It says
nothing about what ROM1's own cast dispatch resolves as that cell for a directional aim; it is a
claim about the generator, not about the UI feeding it. This matches the brief's characterization
exactly: absent, not contradicted. No conflict found.

## Divergence rows

The seat's mid-task correction reassigned my ids from `DIV-056`/`DIV-057` (already spent by the
weapon lane) to **`DIV-063`** and **`DIV-064`**. Both are used; none returned. Code comments in
`celleffect.go` were written against the corrected ids from the start of the visible diff (an
earlier draft cited the old ids and was corrected before this commit).

Ready-to-paste rows, in the ledger's own column order:

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-063 | magic / area effects, staged ring anchor | The Acid Stream cone must start on the caster's own cell, not the aimed cell (2026-08-15, screenshot) | `MAGIC-RING-048` states the three ring generators' offsets are relative to the target cell; it does not state what ROM1's own cast dispatch resolves as that cell for a directional aim | `landAreaFacing` anchors Acid Stream's (id 9) ring program at the caster's own cell; the aimed cell supplies orientation alone through `areaDirection`. Fire Sacrifice and Meteor Storm are unchanged, anchored at the aimed cell as before | UNKNOWN | Owner-authored per the pipeline-v2 ruling that owner directive outweighs research absence for what to build; no claim resolves the UI's own target-cell argument for a directional aim | A claim naming what ROM1's own cast dispatch resolves as the ring generator's target-cell argument for a directional, non-self-targeted area cast | OPEN |
| DIV-064 | magic / area effects, six-slot admission pre-check | — | No claim addresses this build's own two-phase admission design (a pre-check before mana is paid, a structural refusal at the actual landing) | `castBookAt`'s own pre-check and three further `areaLandingRefusal` call sites in `pkg/sim/spell.go` still test the AIMED cell's six-slot fullness before mana is paid. `landAreaFacing`'s own internal refusal and the final `addAreaEffect` insertion now test Acid Stream's real anchor, the caster's own cell, so the two stay aligned inside `celleffect.go`. The outer pre-check in `spell.go` does not, and can in principle disagree with the real landing when the caster's own cell and the aimed cell hold a different count of standing area effects — a state that needs six records already stacked on one of the two cells, which no shipped map or script approaches (`cellEffectSlots`'s own comment: no shipped cell receives the same spell twice) | FIDELITY-DEBT | Closing the gap needs threading the caster's own position into `pkg/sim/spell.go`'s four `areaLandingRefusal` call sites, a file owned by another lane this round | A round that reaches `pkg/sim/spell.go`'s cast admission call sites to align the pre-check's cell with `landAreaFacing`'s own anchor decision for Acid Stream | OPEN |

## Ready-to-paste `spec.md` text

`spec.md` currently reads (lines 278–293, unedited by this lane):

> Ring mode executes stage zero at landing and later stages at ticks 3, 6, 9 and so on. Fire
> Sacrifice has two fixed orientation-independent stages. Relative to the target, stage zero visits
> `(-1,1) (-1,0) (-1,-1) (0,1) (0,-1) (1,1) (1,0) (1,-1)` and stage one visits `(-2,1) (-2,0) (-2,-1)
> (-1,2) (0,2) (1,2) (-1,-2) (0,-2) (1,-2) (2,1) (2,0) (2,-1)`.
>
> Acid Stream has six stages and orientation `o = facing >> 5`. [...]

Proposed replacement, inserted before the "Fire Sacrifice has two fixed..." sentence:

> Every ring row's stage offsets are relative to its own anchor cell, and the anchor is not always
> the aimed cell. Fire Sacrifice's own row carries `Max Range 0`, the column Shield also carries 0
> on, so the admission gate accepts only an aim at the caster's own cell: the anchor and the aimed
> cell coincide for every reachable book cast, and its two stages below are relative to that shared
> cell. Acid Stream's own row carries `Max Range 3`, a real reach, and is a cone the caster throws:
> its anchor is the caster's own cell, and the aimed cell supplies orientation alone. Meteor Storm's
> anchor is the aimed cell.

And at line 329 ("...when the target cell already holds the six records one cell carries"), append:

> Acid Stream's own admission pre-check, in `castBookAt` and three sibling call sites, still tests
> the AIMED cell rather than its anchor (`DIV-064`).

## Witness by reverting

New test file, owned by this lane: `pkg/sim/celleffect_cone1001_test.go`.

`TestAcidStreamConeOriginatesAtTheCastersCell` casts Acid Stream through the real production path —
`KindCastAt` → `step.go` → `beginBookSpellAt`/`castBookAt` → `landArea` → `landAreaFacing` — at eight
click offsets around a caster fixed at `(20, 20)`, one per compass direction, so both the even and
odd orientation families (`MAGIC-RING-048`'s two transforms) are exercised. It asserts the landed
record's own cell (`CellEffects()[0].X/Y`) and its stage-zero footprint sit at the caster's cell, not
the clicked one. `TestMeteorStormStillAnchorsAtTheAimedCell` is the negative control: Meteor Storm
cast away from the caster must still land at the clicked cell.

With the fix reverted (`git stash push -- pkg/sim/celleffect.go`, tests left in place, then
`git stash pop` to restore), all eight orientations reddened, and the negative control stayed green.
Verbatim:

```
=== RUN   TestAcidStreamConeOriginatesAtTheCastersCell/north
    celleffect_cone1001_test.go:59: Acid Stream's own cell = (20,18), want the caster's cell (20,20) — it landed on the clicked cell (20,18) instead
    celleffect_cone1001_test.go:74: Acid Stream stage-zero cells = [4628], want [5140] (offsets from the caster's cell (20,20), not the clicked cell (20,18))
=== RUN   TestAcidStreamConeOriginatesAtTheCastersCell/north-east
    celleffect_cone1001_test.go:59: Acid Stream's own cell = (22,18), want the caster's cell (20,20) — it landed on the clicked cell (22,18) instead
    celleffect_cone1001_test.go:74: Acid Stream stage-zero cells = [4630], want [5140] (offsets from the caster's cell (20,20), not the clicked cell (22,18))
=== RUN   TestAcidStreamConeOriginatesAtTheCastersCell/east
    celleffect_cone1001_test.go:59: Acid Stream's own cell = (22,20), want the caster's cell (20,20) — it landed on the clicked cell (22,20) instead
    celleffect_cone1001_test.go:74: Acid Stream stage-zero cells = [5142], want [5140] (offsets from the caster's cell (20,20), not the clicked cell (22,20))
=== RUN   TestAcidStreamConeOriginatesAtTheCastersCell/south-east
    celleffect_cone1001_test.go:59: Acid Stream's own cell = (22,22), want the caster's cell (20,20) — it landed on the clicked cell (22,22) instead
    celleffect_cone1001_test.go:74: Acid Stream stage-zero cells = [5654], want [5140] (offsets from the caster's cell (20,20), not the clicked cell (22,22))
=== RUN   TestAcidStreamConeOriginatesAtTheCastersCell/south
    celleffect_cone1001_test.go:59: Acid Stream's own cell = (20,22), want the caster's cell (20,20) — it landed on the clicked cell (20,22) instead
    celleffect_cone1001_test.go:74: Acid Stream stage-zero cells = [5652], want [5140] (offsets from the caster's cell (20,20), not the clicked cell (20,22))
=== RUN   TestAcidStreamConeOriginatesAtTheCastersCell/south-west
    celleffect_cone1001_test.go:59: Acid Stream's own cell = (18,22), want the caster's cell (20,20) — it landed on the clicked cell (18,22) instead
    celleffect_cone1001_test.go:74: Acid Stream stage-zero cells = [5650], want [5140] (offsets from the caster's cell (20,20), not the clicked cell (18,22))
=== RUN   TestAcidStreamConeOriginatesAtTheCastersCell/west
    celleffect_cone1001_test.go:59: Acid Stream's own cell = (18,20), want the caster's cell (20,20) — it landed on the clicked cell (18,20) instead
    celleffect_cone1001_test.go:74: Acid Stream stage-zero cells = [5138], want [5140] (offsets from the caster's cell (20,20), not the clicked cell (18,20))
=== RUN   TestAcidStreamConeOriginatesAtTheCastersCell/north-west
    celleffect_cone1001_test.go:59: Acid Stream's own cell = (18,18), want the caster's cell (20,20) — it landed on the clicked cell (18,18) instead
    celleffect_cone1001_test.go:74: Acid Stream stage-zero cells = [4626], want [5140] (offsets from the caster's cell (20,20), not the clicked cell (18,18))
--- FAIL: TestAcidStreamConeOriginatesAtTheCastersCell (0.00s)
=== RUN   TestMeteorStormStillAnchorsAtTheAimedCell
--- PASS: TestMeteorStormStillAnchorsAtTheAimedCell (0.00s)
```

Restored (`git stash pop`) and reran: `go test ./pkg/sim/...` → `ok`, all tests including the new
ones green.

## What was checked for regressions

- **Full suite:** `go test -trimpath -count=1 ./...` — every package `ok`, with the fix in place.
- **Meteor Storm and blast/wall/diamond rows:** unaffected by construction — `anchorX, anchorY`
  defaults to the aimed cell and is overridden only for `rule.ID == 9`; `TestMeteorStormStillAnchorsAtTheAimedCell`
  witnesses it through the production path, and every pre-existing area/ring test
  (`TestStagedAreaProgramsUseTheirDecodedOrderedCells`, `TestACloudPulsesOnItsOwnCounterAndNotOnElapsedTicks`,
  `TestAreaBookApplyTrainsOncePerHostileUnitReached`, the `areacost1001_test.go` and
  `scriptcast_test.go` suites) still passes unmodified.
- **`bash scripts/campaign-sweep.sh`, 28 EN maps, 2000-tick unattended drive:** `59 unsupported /
  7375 reached`, matching the recorded baseline in `round3-orders.md`. Run with the fix in place and
  again with `pkg/sim/celleffect.go` reverted (tree otherwise identical): `diff` between the two full
  28-row tables, hash column included, is empty (`diff` exit 0). `spelleffectcheck`'s own embedded
  campaign census (`go run ./cmd/spelleffectcheck` against the EN root) shows no shipped script
  instant casts Acid Stream at a cell or at a unit across any of the 28 maps (`cast-at-cell` and
  `cast-at-unit` list ids 3, 5, 8, 10, 16, 17, 19, 20, 21, 22, 23 — never 9), which is consistent
  with the byte-identical sweep: no shipped script-driven landing exercises this fix. A hero or
  AI-owned mage that KNOWS Acid Stream through its own spellbook could still land one during the
  sweep's unattended AI decision pass; the byte-identical result says none did on this seed and root
  over 2000 ticks per map, not that none ever could.
- **`go run ./cmd/spelleffectcheck` against `gameversions/en`:** ran clean, printed the full 28-row
  table unchanged from its known form, and (see "Fire Sacrifice" above) cross-checked `Max Range`
  for Fire Sacrifice, Shield and Acid Stream against the live install.
- **Milestone census** (`go build -o mr ./cmd/missionrun`, then
  `AGAINROM_ASSETS=<againrom>/gameversions/en ./mr -mission $m -trace -ticks 1 | grep -c
  UNSUPPORTED`, for `$m` in 10 and 20): **0 and 0**, identical with the fix in place and with
  `pkg/sim/celleffect.go` reverted. This story does not move that census; the claim is that it is
  unchanged, and both figures were re-measured, not assumed.

## Gate, at the pushed sha

From a clean tree:

```
go build ./...
go vet ./...
gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')
go test -trimpath -count=1 ./...
bash scripts/check-no-game-assets.sh
```

All clean: `go build` silent, `go vet` silent, `gofmt -l` printed nothing, every package `ok` under
`go test -trimpath -count=1 ./...`, and `check-no-game-assets: clean (tree scan)`.

## Files touched

- `pkg/sim/celleffect.go` — the anchor decision inside `landAreaFacing`.
- `pkg/sim/celleffect_cone1001_test.go` — new, the production-path witness and the Meteor Storm
  negative control.
- `docs/1001-spell-effects/round3-cone.md` — this document.

No other file was touched. `pkg/sim/spell.go` was read but not edited — `DIV-064` above records
the one place its own call sites would need to change to fully close the residual pre-check gap.
