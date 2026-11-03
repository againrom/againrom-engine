# Shared world-map page and records

## Intent

One world-map page and one record builder serve both campaigns. Each campaign keeps its existing opening, input, travel, arrival, presentation and SAVE values.

## Authority

Owner direction: one builder per kind, with the first game represented by profile data in the second game's page shape. Architecture work counts zero toward a release. The scoped brief covers audit-20 ranks R12 and R40. This is a neutral refactor of base `dae8cfe1a`, pin k412; it adds no claim about original behaviour and no divergence.

## As-built behaviour

`townScreen.mapPage` constructs only `worldMapPage`. Its `openWorldMap(from, records)` uses `worldMapEntryBuilder.open`; `WorldMapView` reads the existing `worldMapState.view`. The input contract has one production page implementation. The host forwards events through it.

`worldmaps/rom1.json` and `rom2.json` carry `WorldMapPagePolicy` rows. The rows select the open condition, build lifetime, initial mode and point, ordered record sources, font and text inputs, task animation, selection history, leave action and arrival steps. The first row retains Gates entry, mode zero, the party's retained point, mission scrolls followed by towns, document font, town home title, task frames and selected markers. The second retains the unlocated/no-talk condition, an opening keyed by campaign and opening identity, select or route mode, the departure record's point, destination order, view-time text, no cancel, and snapping to the first enabled record at the target. LOAD with no opening still starts at the first available destination in select mode.

`worldMapEntryBuilder.record` is the sole producer of `ui.WorldMapMission` and the sole caller of `townMapPoint`. It resolves place, label, point, region and admission for registry missions, catalog destinations and town entries. `open` is the sole producer of `worldMapState` and its record list. Home, restored mission points and victory departure points use the same record resolver.

The navigate hook reads the page's build row. Map art binding follows that row and keeps the first game's callback nil, preserving the retained host state that a refused LOAD leaves unchanged. `worldMapState`, its view, the route builder and the SAV producer are unchanged.

Campaign location entry remains the named boundary admitted by the brief. `worldMapLocationEntries` dispatches `town-state` to `arriveInTown`, `mission-action` to `worldMapMissionAction`, and `location` to `enterLocation`. The last handler moved from the removed page receiver to its host without changing its statements. There is no existing common campaign entry call: one campaign swaps chapter-town state and wraps mission opener recovery; the other enters its scenario location and carries the bag. The page supplies one policy-driven arrival step and does not replace those campaign operations.

## Proof

Commit `ed922d7e8` records the pre-change behaviour before production edits. `TestWorldMapPageRecords` compares 39 fixed SHA-256 records. Each covers the complete exported view, image bounds and pixel hashes, the composed synthetic frame, mode, cross counter, host location, arrival action, marker serialization order, `firstMapPoint` and selected mission. Destination records also include the campaign bank and availability. The script covers both openings, hover, move, click, choose, tick and leave; both LOAD cases; chapter victory and homeward arrival; and travel to another chapter town. No install is read.

All 39 records match after the change. A temporary test overlay setting `rom1.Page.Task` to false makes the unchanged record test fail at `chapter/tick` with exit 1. The overlay never changes production files or the recorded hashes.

Final focused checks pass:

- `pkg/game`, the brief's filter plus SecondMap, SecondVictory, SecondBag, RestoreOriginal and ResetForNewGame: 5.783 s.
- `pkg/ui`, WorldMap and Headless: 0.588 s.
- `internal/archtest`, WorldMapPage and the DAG's TestLiveTreeClean: 0.829 s.
- `internal/storyguard`, TestLiveTreeClean: 4.493 s.
- `go build ./...` and `go vet ./...`.

`CheckWorldMapPage` scans production game syntax. It rejects another page or partial page operation, another implementation of a page interface, embedded and aliased pages, another constructor, game discriminators, page type switches, records and state literals outside the builder, record list writes, and a town point call or alias outside the record resolver. It pins the three named campaign entry keys and their targets. Its planted cases exercise these shapes. It introduces no package dependency.

The five original censuses reach their targets: page methods 16/2 to 0/0, town point sites 4/2 to 2/1, retired page types 2/1 to 0/0, UI record literals 3/2 to 1/1, state literals 2/2 to 1/1. A broader census checks page-shaped types, interfaces, host routing, leave discriminators, views and record appends. Its unrelated map inspectors, mission lists and SAV records remain outside these two kinds. Exact sites and final-candidate gate receipts are returned in the lane notes.

## Open debt

Installed pixels and native SAV output equality are not proved by fixture records. No release test runs in the lane. The next release must run both-root world-map, victory return, route witness, marker SAVE/LOAD, mission handoff, mod-town travel and campaign continuation tests; the lane return names the release inventory. The existing campaign location entries remain explicit guarded boundaries. The seat pushes the candidate, reconciles main and runs the sole adversarial review. Versions, changelog and hotfix ledger do not move in this lane.
