# Verification — the buildings are on the screen

Seven tasks landed, in order, one commit each. The gate ran from the orchestrator
seat at every landing: `go build`, `go vet`, `go test -trimpath -count=1`,
`check-no-game-assets.sh`, `check-doc-budget.sh`, `check-sdd-audit.sh` and
`gofmt -l` over the tracked Go files, chained with `&&` and no pipe. **EXIT=0 and
an empty FAIL set at every one of the seven**, and the ten `DECLARED OVERRUN`
notices are the nine pre-existing ones plus this story's own.

Every test in this story is synthetic and headless: no test reads an install, no
test opens a window, and the corpus figures below come from developer tools run
by hand against a lawful install. No game byte is committed anywhere.

## The criteria

| ID | Where it is witnessed | Result |
|---|---|---|
| AC-1 | `pkg/game/structures_test.go` — `TestLoadStructuresExclusions`, `TestLoadStructuresErrorsOnlyOnTheRegistry` | seven classes: the whole one draws; the absent sheet, the palette-less sheet, the 32x31 frame, the zero extent and `VariableSize` each resolve to a class that draws nothing, counted apart; the id above 255 is reachable by no key. Only an unreadable or unparseable registry errors |
| AC-2 | `pkg/game/structureanim_test.go` — `TestStructureAnimationGateIsAConjunction`; `pkg/game/structures_test.go` — `TestStructureAnimationGate` | ten precondition shapes, only the one meeting all three carrying a timeline. **The wrong-length mask arm moved one tier down** — see *Findings* |
| AC-3 | `pkg/render/terrain/structures_test.go` — `TestStructureStripIsCellForCellAndFrameForFrame`, `TestStructureFullHeightEqualToTileHeightIsOneFramePerCell`, `TestStructureOutOfExtentCellsAreDroppedAndCounted`, `TestStructureAnchoredWhollyOutsideYieldsNothing` | the 3x2 of `FullHeight` 5 matches the contract's strip cell for cell and frame for frame against a hand-written table; the degenerate class is one frame per cell; the overrunning rectangle drops its off-map cells and touches no cell of the next row; the outside placement yields nothing |
| AC-4 | `pkg/render/terrain/structures_test.go` — `TestStructureSelectsTheThreeBlocks`, `TestStructureRuinBlockIsAddressedFromTheEnd`, `TestStructureSelectionNeverLeavesTheSheet` | every arm at every counter of two whole cycles; the ruin block addressed from the end; `Indestructible` returning to base; short sheets landing at the base frame |
| AC-5 | `pkg/render/terrain/structures_test.go` — `TestStructureLiftAtEveryParity`; `pkg/ui/structures_test.go` — `TestStructureLiftTakesCornerHeightsAndNotCellHeights` | all three parity combinations; the odd/odd case equal to the centre cell's four-corner mean, asserted against an independent mean; every entry of one structure at one lift; `destX` the flat geometry's |
| AC-6 | retired by `spec.md` — folded into AC-3 | not reused, and no test claims it |
| AC-7 | `pkg/render/terrain/structures_test.go` — `TestStructurePlanesMergeByRow`, `TestStructureMergeComparesRowsAndLeadsAtATie`; `pkg/ui/structures_test.go` — `TestDrawArtPutsTheFlatPassFirstAndMergesTheRest` | the flat one before every static object; the other in rectangle-row order among them; columns descending inside a row; equal cells in record order |
| AC-8 | `pkg/game/structures_test.go` — `TestStructureContractIsIndifferentToTheOmittedKeyDefault` | eight scalar keys, each omitted and each spelled at the registry's own decoded default: identical bundle, list and census. **Its id is listed as retired and also carries a row in the criteria table** — see *Findings* |
| AC-9 | `pkg/game/structures_test.go` — `TestLoadMapViewerWithoutAStructureBundle`; `pkg/ui/structures_test.go` — `TestStructureViewerWithoutABundle`, `TestDrawArtWithoutStructuresIsTheObjectLayerAlone`; `pkg/render/terrain/structures_test.go` — `TestStructureCensusLinesNameEveryCounter` | all three no-bundle shapes place nothing and draw the object layer alone, call for call; the census reports both levels with no install |
| AC-10 | corpus run below | ten loose maps of the EN install and six of the RU one censused with no error; both levels recorded; the three observations reported and reconciled with nothing |

## The derived properties

| ID | Where it is witnessed | Result |
|---|---|---|
| P-1 | `TestStructureBuilderIsPureAndTotal` | six grids including a zero one, a negative extent and a saturated key: two builds agree, nothing is mutated, a nil bundle is an empty one |
| P-2 | `TestStructurePlacementCarriesNoAnchor`, `TestStructureDestXDependsOnTheColumnAlone` | the placement type's own field set is read by reflection, so an `Anchor` added and left zero fails; `destX` is its column times `CellSize` at every class, geometry and origin |
| P-3 | `TestStructureLiftIsOnePerStructureOverRandomFields`, `TestStructureDisplacedListIsTheFlatListTranslated` | 32 generated height fields, two structures each: one lift per structure, and the two lists differ by a vertical translation alone |
| P-4 | `TestStructureSelectionNeverLeavesTheSheet` | eight frame counts x twelve counters x every grid index: no selection outside the sheet, and every fallback is that index's own base frame |
| P-5 | `TestStructureMergeIsTheObjectListWithoutStructures`, `TestDrawArtWithoutStructuresIsTheObjectLayerAlone` | with no structures the merged order is the object list ref for ref, and the draw pass submits exactly what the object layer alone submits |
| P-6 | `TestStructureCensusIsComplete`, `TestStructureOverhangIsCountedAsAFrame` | 64 generated maps: the four per-placement counters sum to the record count, and strips plus out-of-extent drops equal the drawn rectangles' cells |
| P-7 | `pkg/ui/structures_test.go` — `TestViewerPlacesFromTheBuildersOwnList` | the viewer's two lists, its two animated subsets, its census and its plane order are the builder's and the merge's own output, entry for entry |
| P-8 | `TestStructureContractIsIndifferentToTheOmittedKeyDefault` | see AC-8 |

## The success criteria

- **SC-1** — every shipped map censuses with no error and both levels are
  recorded; the per-placement counters sum to each map's placement count. Ten EN
  maps and six RU, zero failures. See the corpus block below.
- **SC-2** — a rendered map carrying buildings, from a lawful install, shipped as
  an owner-review artifact. `review/0054-structure-art/`, outside both repos.
- **SC-3** — with no bundle the rendered bytes, placement lists and counts are the
  pre-story ones: AC-9 and P-5 above, plus the whole pre-existing suite of
  `pkg/ui` and `cmd/mapview`, which builds no structure bundle and is unedited in
  every assertion about what it draws.
- **SC-4** — the window and the raster harness produce the same structure
  geometry for one map, checked as the LIST they both place from: P-7 above. The
  harness calls the same builder with the same two terms and the same merge.
- **SC-5** — every mutant each task names was applied where that task wrote it and
  killed by that task's own tests. **34 of 34 killed**; the run is below.
- **SC-6** — the three observations are recorded as figures with the corpus they
  came from, below, and none is reconciled into a contract clause or a default.

## The corpus (AC-10, SC-1, SC-6)

`terraintool structures` over every loose `.alm` of both lawful installs, and
`mapview -structures -check` over the same. No game bytes are reproduced here —
figures only.

```
EN, 10 maps, 0 failed
  placements  2183 drawn, 0 no class, 0 undrawable, 7 variable-size
  cells      11419 strips, 14279 frames (2860 overhang), 0 outside the map
RU, 6 maps, 0 failed
  placements   730 drawn, 0 no class, 0 undrawable, 0 variable-size
  cells       4176 strips, 5270 frames (1094 overhang), 0 outside the map

per map (EN)
  Forester   207 drawn / 1252 strips 1566 frames (314 overhang)
  Horror     411 drawn / 2168 strips 2722 frames (554 overhang), 4 variable-size
  Islands    287 drawn / 1599 strips 2030 frames (431 overhang)
  Kids        16 drawn /   92 strips  127 frames ( 35 overhang)
  LuMoir     107 drawn /  522 strips  691 frames (169 overhang)
  Waters     113 drawn /  711 strips  856 frames (145 overhang)
  Beast      478 drawn / 1911 strips 2394 frames (483 overhang)
  Cross      289 drawn / 1622 strips 2037 frames (415 overhang), 3 variable-size
  Kids2       19 drawn /  112 strips  159 frames ( 47 overhang)
  Tomb       256 drawn / 1430 strips 1697 frames (267 overhang)
```

**Nothing is unresolved and nothing is undrawable on either install.** Every
type-4 key on every loose map names a loaded class, and every class those keys
name decoded to a sheet of `CellSize`-square frames — so the five exclusions are
exercised by fixtures alone and by no shipped byte.

The three observations, identical on both installs, reported and not reconciled:

1. **`VariableSize` class ids: 33 and 37.** The two wooden bridges.
2. **Placements carrying an eight-byte extension: 7 across the EN corpus** — 4 on
   `Horror`, 3 on `Cross` — and 0 on the RU one. The extension count equals the
   variable-size skip count on every map, which is what `spec.md`'s "no placement
   drawn here carries one" predicts and does not assert.
3. **Rectangle disagreement: 2 of 66 classes**, ids 14 and 57, where the
   registry's `TileWidth x TileHeight` and the definition table's `sizeX`/`sizeY`
   differ. This contract reads the registry alone, so the figure decides nothing
   here; whichever story holds both tables owns the reconciliation.

**The corpus is the LOOSE maps of both installs.** The campaign maps inside
`scenario.res` are not reached by this harness, which takes a host path; extending
it to enumerate an archive is not this story's and the gap is stated rather than
papered over.

## Mutation (SC-5)

Each task's named mutants, applied to the lines that task wrote and reverted
there. Six for T1, five for T2, six for T3, five for T4, five for T5, four for T6,
four for T7 — **34 applied, 34 killed**. Three needed a test the first pass did
not have and each is recorded as a finding below rather than as a pass.

Two mutants beyond the named set were applied and killed as well, both at seams a
named one could not reach: `Projection.AnchorHeight` substituted for
`Projection.Altitude` at the viewer's own call site (the per-cell accessor for the
corner one, DD-6's whole distinction), and a `-ruins` flag added to the game
front-end.

## Findings against the landed papers

Three, each recorded in the code beside what it changed.

**`plan.md` DD-4's "the mask's length is validated here and nowhere else" is
wrong about this tree.** `pkg/data`'s own validation carries the identical
`TileWidth x FullHeight` rule and makes a wrong-length `AnimMask` a LOAD ERROR for
the whole registry — stricter than this story's gate, and reached first. So AC-2's
second shape cannot arrive through `LoadStructureClasses` at all: the registry is
refused and no bundle is produced. That is a stronger form of "refused whole", not
a weaker one. The gate's own arm stays, because it is what makes the loader total
over any resolved class handed to it, and it is exercised directly by an internal
test, since no registry can deliver one.

**AC-1's "the two default values give identical bundles" could not hold for the
three extents as the loader first carried them.** An omitted `TileWidth` resolves
to 0 today and to -1 at the registry's own decoded default; both make a class
undrawable, and the two bundles still differed in a field neither draws with. The
loader now does not carry extents that are not a rectangle at all — such a class
draws nothing and its extents mean nothing — so the criterion holds as written for
every scalar this contract reads.

**`spec.md` lists AC-8 in its criteria table and also declares its id retired**
("AC-6 and AC-8 were folded into AC-3 and AC-1; their ids are retired, not
reused"). AC-6 is retired and has no row; AC-8 has a row. Both are witnessed above
— the row's own clause by the P-8 test, the retirement by saying so — rather than
one of the two statements being picked and the other ignored.

## What was deliberately done beyond the papers

**The game front-end draws structures.** No task names `pkg/game/frontend.go`, so
on the papers alone `cmd/againrom` would still open every map with no building on
it. The owner asked to open the game and see one, and a placed structure is the
map's own content — there is no flag on the object or unit bundles' path either —
so the bundle is loaded at startup beside those two and every map opens with its
art on. An unreadable structure registry now fails startup, as an unreadable
object or unit registry already does. T2's fence — "no flag turns this on by
default in the game front-end" — is honoured: no flag was added anywhere near it,
and the developer viewer keeps its `-structures` switch.

**`cmd/againrom` gained a flag-surface pin.** It had none, and "the diagnostic is
not reachable from the front-end" was otherwise a claim no test made. It records
what the game must NOT offer as well as what it does.

## Build and review artifacts

`builds/0054-structure-art/` holds `againrom.exe`, `mapview.exe` and
`terraintool.exe` with a README giving the exact invocations, the map to open
first and what each switch shows. `builds/` is gitignored; nothing in it is
committed.

`review/0054-structure-art/`, outside both repositories, holds three PNGs
rendered from a lawful install — a whole map and two crops:

```
kids-village.png     760x570    md5 5e00588e345bdbbbd2d44b27fb067560
beast-town.png       900x700    md5 3e334fb5b2f271cc3309ce772e497af1
kids-structures.png  2560x2571  md5 d92cf5d2628d53f0c711c4bd47b9ad29
```

The crops are the picture, not the count: a stone tower, a red-roofed house and a
fountain in one clearing; and a whole town of houses, a watermill with its wheel,
a temple and a windmill, every one standing on its own ground with the object
layer's trees correctly in front of and behind it.
