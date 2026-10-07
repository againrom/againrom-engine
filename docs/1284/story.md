# Area-layer cost timing and Prismatic Spray group sight

## Intent and authority

Two follow-ups to the area-layer movement cost story. First, the cost byte of a layered cell follows the original's timing: a recompute at the tick a mover's stride crosses a cell, a layer write that lands after every actor of its tick, and the original's own first read after LOAD. Second, Prismatic Spray's secondaries are chosen from the whole group's sight.

Authority: `MOVE-084`, `MOVE-085`, `MOVE-086`, `MAGIC-AREACOST-046`, `MAGIC-SPRAY-134` to `MAGIC-SPRAY-137`, `AI-SPRAY-266`, `AI-SPRAY-267`, `AI-GROUPSEE-068`, `MAGIC-REACH-179`, `MAGIC-CASTCLOCK-171`. Owner ruling: a loaded state and a continuing state are one kind of state; where the original itself differs after LOAD, the engine follows the original.

## As built

- A stride crosses a cell on the tick whose position lies in a different cell from the next tick's. The position is a 16-bit value per axis with fraction 0x80 at the start and a signed step per tick, so a negative step crosses at a different tick from a positive one. The released and the occupied footprint are recomputed, every cell of each, in native and saved-plane worlds (`strideCrossing`, `recomputeCrossing`).
- In a saved-plane world the cost byte written by a layer event (cast, expiry, clear, blast landing) is held until every actor of the tick has run, then written in cell order. Layered flags are read as they stood at the start of the tick (`pkg/sim/layercostwindow.go`). A native world already applied layers after the actors. The block planes, record counts and slots, and the effect step itself keep their earlier order.
- An original-SAV import rewrites each layered cell of its saved plane from its record (`RewriteImportedLayerCosts`); the original keeps the ingest byte there and reads a quarter of it first. The engine's form carries the saved plane's cost, so a load of that form keeps the plane as saved. A native load lists every standing-cloud cell for the first tick at the byte its form carries (`ResetLoadedAreaCosts`). A layer cast onto a layered cell in the first tick after LOAD is read as no layer, as in a continuing world. The quarter read is not given on any load, because the SAVE and LOAD release witnesses hold restored equal to source (`DIV-1842`).
- Prismatic Spray scores the candidates of the caster's group: the saved Group, else the command or placed group, with the group's first member as diplomacy decider. List B beside a living A cannot win, because its score is at least 65536 against the 65530 sentinel. A player cast order gives the caster a command group of its own; an unbidden cast uses the placed group.

## Divergences

- `DIV-1842`: what a loaded effect does at its first tick is Unknown, and a loaded layered cell does not read at the original's quarter (DEVIATION, ACCEPTED by owner decision). Moved to `docs/divergences/magic-area-movement-cost.md`.
- `DIV-1843`: the crossing tick is exact; slot skips and a mover removed before its crossing are not modelled (FIDELITY-DEBT, OPEN). Same file.
- `DIV-1844`: the cost byte is held behind the actors; the effect step and the block planes are not (FIDELITY-DEBT, OPEN). Same file.
- `DIV-1930`: closed, moved to `docs/DIVERGENCES-CLOSED.md`.
- `DIV-1272`: narrowed; the temporary-caster group, the heading for fractions and footprints (`DIV-1631`) and the facing gate on the item, scroll and weapon arms remain (FIDELITY-DEBT, OPEN).
- `DIV-1962` to `DIV-1967` are returned unused.

## Proof

- `pkg/sim/areacrossing_test.go`, `areacost_test.go`, `areacostload_test.go`: crossing arithmetic against an independent computation, the whole footprint, a layer landing after the actors (it fails with the window disabled: transit 16 against 11), the first read after LOAD for a saved plane, and the first-tick second layer after a native load with a restore that skips the reset as the loss control, and a saved plane's decayed layered cell across SAVE and cold LOAD with a rewritten restore as the loss control (`TestSavedPlaneFormKeepsADecayedLayeredCellAcrossLoad`).
- `TestReleaseAreaLayerCostRecomputesOnTheCrossingTick` and `TestReleaseAreaLayerLandsAfterTheActorsOfItsTick` (`pkg/game`, EN and RU roots): mission 20's terrain, cost plane and spell rules; movers and the cloud are placed and advanced by ordinary ticks.
- `TestReleaseAreaLayerCostContinuesAcrossSaveAndLoad`: the continuing and the cold-loaded worlds marshal alike after the same orders; a world loaded from before the cast is the loss control. 
- `TestPrismaticSpraySecondariesComeFromTheWholeGroupsSight` (sim): with and without a group-mate, and with a command group as the control. `TestReleasePrismaticSpraySecondariesFollowTheGroupsSightAndRank` (`pkg/game`, EN and RU roots): on mission 20's terrain, an unbidden mage and a group-mate; foes beyond the mage's sight are chosen by the mate's sight and ordered by distance against list order; without the mate, or with a mate of no sight, the primary is hit alone.

## Open debt

- Question for research: after LOAD of a SAV whose effect record is standing, does the first area pass write that effect's layer onto its cells, and what is the effect's `+0x48` at that tick (`DIV-1842`)?
- Question for research: which tick of a transit calls the release and occupy recompute when the mover's slot is empty or taken (`MOVE-085`)?
- Question for research: whose group does a script-created caster belong to when it casts Prismatic Spray (`MAGIC-SPRAY-135`)?
