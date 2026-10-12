# The first game's cell planes at mission start

## Intent

Every first-game World holds the cell planes carrier (Static, Dynamic, Cost, CostKnown, Height and
the installed `map.reg` cost table) from mission start, as a World loaded from an original SAV
holds it. One terrain rule then runs after a fresh start and after a LOAD, and a SAV the engine
writes carries the planes the original would hold.

## Authority

`TERR-PASS-049`, `TERR-PASS-053`, `TERR-STRUCT-071`, `TERR-STRUCT-078`, `SAV-LOAD-057`,
`SAV-CELLLOAD-109` to `SAV-CELLLOAD-111`, `SAV-CELLENTRY-582`, `SAV-CELLLEAVE-584`, `MOVE-084` to
`MOVE-088`, `MAGIC-AREACOST-046`, `ALM-TERR-043`, and at k273 `TERR-228`, `TERR-238` to
`TERR-240`. Owner direction: one kind of World from mission start; the engine never adapts to a
crooked save.

## As built

- The cost table. `readTerrainCosts` reads the install's `world/data/map.reg` into
  `mapload.Table.TerrainCosts`; `mapload.Planes` gives the Terrain its cost plane and the
  installed constructor base. A World with no install keeps the compiled table (DIV-3093).
- One plane builder. `World.CellPlanes` (`pkg/sim/cellplanes.go`) builds the planes from a base:
  at mission start the native grid, cost and height fold over the base; at a LOAD the base takes
  the document's Block records. `newWorld` calls it through `ConstructSavedCellPlanes` when the
  Terrain carries a base; `importOriginalCellPlanes` calls it for the original and the engine SAV
  LOAD. A World with no base holds no planes (DIV-3095).
- Cell slots. At mission start every living on-map actor enters its cells by the entry rule; a
  native crossing releases and occupies by `MOVE-088`. A native actor's slot holds key 0; SAVE
  writes the actor's Identity and a LOAD returns the slot to that form (DIV-3094). A terminal
  actor releases its native slots (`clearTerminalRegistry`); a LOAD moves a native actor's stale
  key-0 slot to the cells it covers (`syncNativeActorCells`). A script that takes an actor off the
  map releases its slots, and placing it enters its cells (`takeOffMap`, `placeAt`, DIV-3099).
- Sacks. Every Sack registers on its cell by `SAV-SACKENTRY-590` at construction, after the actors
  and before the Building records; a Sack planted in play registers too, and a pick-up removes the
  record. The slot key derives from the cell (`sackCellKey`); a SAVE writes it as the Sack's
  Identity, and a Sack new since the LOAD takes the key its document cell already holds
  (`pkg/game/worldsave.go`) (DIV-3098).
- Buildings. On a World with planes each Building cell holds the record the attach writes
  (`TERR-240`): baselines, bit 5 and the Building at +12, then the recompute
  (`attachStructureRecords`). The recompute sets Static and Dynamic bits 0 and 2 on a blocking
  cell, clears them on an open one and prices an open cell (a doorway or a bridge deck) at the
  cracked-ground cost `Costs[5]` (`TERR-STRUCT-071`). A structure import on such a World takes its
  block bits from the planes and unbinds records naming no Building of the imported roster.
- Readers. Terrain openness (`terrainOpen`), movement cost (`costAt`), height (`heightAt`), the
  area cost recompute (`recomputeAreaCosts`, `RewriteImportedLayerCosts`), placement (`placementOpen`)
  and a crossing's slots (`recomputeCrossing`) read the carrier on every World that holds it. The
  native grid stays as the routing mirror the planes keep in step (`syncNativeSavedPlaneCell`);
  the native cost, height and area-cost tables remain only for a World without planes (DIV-3095).
- Second game. Second-game Worlds carry no installed base and hold no planes from creation
  (DIV-3095); their LOAD is unchanged.
- Structure flags and grid bit 3. A LOAD of a native SAV yields the native form: the grid's static
  object bit names no structure binding on a World with planes, and the World's policy Terrain
  omits that mirror. A LOAD whose policy carries no structure flag keeps the imported roster on a
  World with planes, so the cell records name the Buildings the SAVE wrote
  (`restoreCurrentPolicy`). Every restore marks the cells of standing magic-wall clouds in the grid.
- Prepared maps. The entry path for a prepared map with neither snapshot nor fog builds the
  Building records like a fresh start and does not take the document's planes
  (`pkg/game/missionentry.go`). The only production prepare (`originalSource.prepareMission`)
  always returns a fog, so in production that path serves a fresh start only.
- Refused maps. On a first-game install a map that `mapload.OriginalTerrainPlanes` refuses (a
  dimension outside 8..256, or a tile word in strip groups 13 to 15) does not start: `newWorld`
  fails with `installed terrain planes`. Before this story such a map started and only its SAVE
  failed. No installed map and no editor route produces such a tile; a hand-built mod map can.
- Felled bodies. A body whose transit clears (a felled actor, a headless relocation) keeps in its
  known native position block the cell centre its stride last showed (`clearTransit`), so the SAVE
  of a body that died after moving writes the position the World holds. A different route on the
  planes reached this state in the owner save `2026-10-07/saveorcsdontgo.sav` (M2 round-trip gate,
  dying route). `clearTransit` runs also where a living actor stops mid-stride: a manual cast, a
  cell-cast approach, mission return and a motion import. While the transit is nonzero
  `NativeBasisNow` interpolates, so such an actor's known block keeps a fine position that can lie
  in its source cell while `X,Y` names the destination. The SAVE writes those bytes and the LOAD
  restores them byte for byte.
- Summoning. The scratch World that builds a summoned or cheat actor carries no installed cost
  table, because its 40 by 40 map has no terrain (`mapload.CheatActor`).

## Proof

Census tests, `pkg/game/cellplanesstart_release_test.go`, run on EN and RU (`AGAINROM_ASSETS`
each root, corpus `gameversions/saves`):

| test | result on EN and RU | loss control |
|---|---|---|
| `TestReleaseEveryMissionStartHoldsTheInstalledCellPlanes` | 27 missions; carrier equals the native projection (installed constructor planes with the native grid, cost and height folded in, written in the test) except on cells a start writer changes: actor cells 2488 Static and 2488 Dynamic, Building cells 4989 Static, 4989 Dynamic and 1059 Cost, cell-tail cells 43 Static, 43 Dynamic and 1 Cost, actor cells on open Building cells 7 Cost; no unexplained cell | a mission with a living actor must show an actor-caused Dynamic difference; a planes-free control World must hold no planes |
| `TestReleaseMissionStartPlanesAgainstTheRestartSlots` | 7 restart slots (`game9999.sav`, missions 10, 10, 20, 30, 30, 41, 141): fresh Static and Dynamic differ from the original's start only on 1 to 4 cells where the original holds an actor and the fresh start does not (one a Building cell on mission 10) and on mission 30's 4 cell-tail cells holding an actor; no Sack cell differs; Cost differs on 13 to 25 open Building cells per slot (fresh cracked-ground 6, LOAD the installed cost, DIV-3096) | a cell no start writer explains fails |
| `TestReleaseMissionRoutesAndCostsMatchTheNativeReader` | 27 missions, 27 walks of up to 4 actors for 256 ticks against the planes-free World (the old native reader). Height equal on every cell; cost equal on every cell but open Building cells at `Costs[5]`. 9 walks change: 7 on an open Building cell's cracked-ground cost (missions 40, 51, 60, 91, 130, 140, 150), 2 where a domain-1 mover stops at Static bit 2 (missions 10, 71) | mission 10's planes Cost plane rewritten on 1225 cells of the native route changes the walk |
| `TestReleaseCellPlanesSurviveSaveColdLoadAndTicks` | mission 10 fresh, its engine SAV reloaded and saved again, and the corpus SAV `2026-08-02/game0000.sav` (mission 20): each cold LOAD holds the continuous World's hash and every plane byte, and 256 ticks of the same walk keep equal hashes and planes | the fresh SAV with constructor Block bytes on its occupied cells LOADs to a different World |

Focused tests for the World-form rules, each failing with its fix removed: in
`pkg/sim/cellplaneslots_test.go` `TestTerminalBodyReleasesItsNativeCellSlot`,
`TestPlacedSackEntersItsCellRecord`, `TestPlanesWorldKeepsAnImportedRosterWithoutTheCarrierFlag`,
`TestPlanesWorldRestoreMarksStandingWallClouds`, `TestNativeActorStaleSlotMovesToItsFootprint` and
`TestMapPresenceMovesTheActorsCellSlot`; `TestCheatActorBuildsUnderAnInstalledCostTable` in
`pkg/mapload`.

Re-pin list. Expected values moved:
- `TestNativeCellSlots`: the native slot form after a LOAD bound key 77, and its actor now stands
  off the stale cell (DIV-3094).
- The profile witness `pkg/game/testdata/profilewitness/rom1-en.txt` and `rom1-ru.txt`: world
  hashes and SAV digests only, because a fresh World now holds the planes, the Sack records and the
  Building records; no frame moved.
- `generatedmissionsavfields_test.go`: cell-tail cells count as record cells.
- Sack tests whose premise was a Sack with no cell record: `newsackcurrent_release_test.go` now
  requires the cell present; `currentcarrierabsence_test.go` (a death Sack carries no absent cell
  Sack, and the Sack subtest finds the key in the World cell); `currentitems_test.go` (the motion
  malformed modes synthesize an absent row; the cell-edit test takes the Sack cell from the World).
- `structureuse1150_release_test.go` and `loadedcycle_release_test.go`: the hand-built witness
  World takes the mission planes and the Building records.
- `TestMilestone2WriterLossControls`: the older writer's SAV no longer loses its Buildings on a
  World with planes, which keeps the roster the file lists; the control now requires the Buildings
  instrument to pass on that file and the other ten to fail. The Buildings instrument's own
  controls (`milestone2_buildings_controls_test.go`) still prove it finds a changed roster.

Every other test edit follows a fixture or a field move, not a value: 16 edited `pkg/sim` test
files read the per-cell records from `cellRecords` instead of `savedMotion.Cells`; 11 `pkg/game`
fixtures give a hand-built World the mission planes (`withMissionCellPlanes`,
`withLoadedCellPlanes`) or give a hand-built Table the install's cost table;
`TestCurrentBuildingConstructorOwnsFreshFieldsBeforeFirstSave` also requires equal fresh and cold
hashes. No other witness digest or pinned byte form moved.

Gates on `3a0be61b` (main `165270ed` merged, pin k275; the commit that adds this list changes
only this document):

| gate | result |
|---|---|
| `check-release-tests.sh`, EN | ok, 1016 of 1118 ran, 102 lacked a subject, no failing name |
| `check-release-tests.sh`, RU | ok, 1016 of 1118 ran, 102 lacked a subject, no failing name |
| `check-milestone2-acceptance.sh` EN and RU | ok, 50 instruments per root; `TestMilestone2TerrainBlockPlane` 78 world-half files, 214211 delta records, 0 differences |
| second-game groups as `check-rom2-save-tests.sh` runs them | `rom2-en` 99 and `rom2-ru` 99 declared subjects pass, no skip or fail |
| `gofmt -l`, `git diff --check` | clean |
| `go vet` of `pkg/mapload`, `pkg/sim`, `pkg/game` (also with `sessioncorpusaudit`), `cmd/againrom` | clean |
| `go test -trimpath -count=1 ./...` | ok, including `internal/storyguard`, `internal/archtest`, `internal/gatedtests`, `internal/divledger`, `cmd/againrom` |
| `scripts/check-no-game-assets.sh` | clean |
| `check-div-claims.sh` with `AGAINROM_IMPL` set to the lane | exit 0 |

## Open debt

- DIV-3096: the engine keeps a fifth plane, `CostKnown`, which the original does not build
  (`TERR-238`) and does not save with the cost plane (`TERR-PASS-053`). Closure: a LOAD that builds
  cost and height from the install.
- DIV-3097: the mission-start writers do not run in `TERR-228`'s order: actors enter at the end
  of `newWorld` in entity order, Sacks register after them, Building records attach last, and
  casters are unordered. The order matters only where two writers meet one cell (`TERR-239`).
- DIV-3098: a Sack's slot holds a key derived from its cell, not the original's pointer; no claim
  states the Identity the original's writer gives a Sack the World created.
- DIV-3099: the slot writes at the off-map and placement instants are unresearched.
- Unresearched: whether the original starts a map whose terrain the mission build refuses, and
  the original's position bytes when a living actor's stride is cut.
- The rows DIV-1843, DIV-1844, DIV-2029, DIV-1842, DIV-1413, DIV-2476 and DIV-1199 are narrowed to
  a World without planes or to their remaining Unknowns; DIV-932 is unchanged.
