# 1010 — closure

Branch `1010-world-map-route`, base `c3ac3d4`, merged `master` at `857fb43` (`1012-notice-cadence`
landing, no conflict — that story touched no Go, only `docs/DIVERGENCES.md`'s `DIV-011` row and its
own doc folder). Research pin `02a1403` (unchanged; not bumped, per brief).

Gate on the merged tree: `go build ./...`, `go vet ./...`, `gofmt -l` over tracked and untracked Go
files (clean, no output), and `go test -trimpath -count=1 ./...` — 39 packages `ok`, 0 `FAIL`.
`bash scripts/check-no-game-assets.sh` reports clean. The deletion set against base is empty (`git
diff --stat master...HEAD` touches only `pkg/game/worldmap.go`, `pkg/game/worldmap_test.go`,
`pkg/game/worldmaproute.go` (new), `pkg/ui/worldmap.go`, `pkg/ui/worldmap_test.go`, and this story's
own `docs/1010-world-map-route/`).

## Result

`pkg/game/worldmap.go`'s route construction is replaced: a new `pkg/game/worldmaproute.go` builds
the `PathMap.bmp` node graph (`TOWN-116`/`TOWN-117`) and searches it for the least-accumulated-length
chain (`TOWN-119`), instead of an eight-neighbour BFS over every mask pixel sampled at 10-pixel
intervals. `pkg/ui/worldmap.go`'s `ComposeWorldMap` reveals 8 route coordinates per tick and stamps
`BallMap.bmp` at every eighth revealed coordinate (`TOWN-120`), instead of one stamp per revealed
point, and paints cached mission markers before the route to match the claim's own paint order.

`docs/DIVERGENCES.md`'s `DIV-104` moves to `DIVERGENCES-CLOSED.md`, status `CLOSED`. Three new rows
are added: `DIV-129` (Authored table — the graph-unusable fallback), `DIV-130` (Divergences table —
marker identity/animation, out of scope, same reasoning `DIV-128` already gave for the adjacent
marker-cache gap), `DIV-131` (Authored table — the search's edge weight, which `TOWN-119`'s headline and body
state differently, and its tie-break). All three ids allocated to this story are spent; none is returned.

## The twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | N/A | `PathMap.bmp` decoding (`terrain.DecodeBMP8`) is unchanged; only what this build does with the decoded mask changes. |
| Runtime state | PASS | `worldMapState.route`/`.shown` are rebuilt from the new graph route and reveal at the new cadence; `worldMapAssets.graph` is new cached view state, built once per session like `.path` beside it. |
| Simulation | N/A | Outside `pkg/sim`; no simulation file touched. |
| Player input | N/A | Selection and click handling (`WorldMapClick`, `WorldMapMove`, `WorldMapChoose`) are unchanged. |
| AI | N/A | Not touched. |
| UI / HUD | PASS | Paint order and both cadence numbers now match `TOWN-120`; witnessed by `TestWorldMapCompositionStampsBallOnlyAtEveryEighthRevealedCoordinate`, `TestWorldMapCompositionPaintsRouteOverMissionMarkers`, and the integration witness below. |
| Triggers / scripts | N/A | Not touched. |
| Inventory / equipment | N/A | Not touched. |
| Persistence / save-load | PASS | `TOWN-117`'s own evidence finds no ROM1 save serializer for these view fields, and this build serializes none either; `TestGatesShowLiveScrollTextSelectRouteAndUseTheMissionDoor` already asserts entering, selecting and returning from the world map produces byte-identical before/after saves, unaffected by this story's route change. |
| Campaign / session | N/A | Mission offer/mapping logic (`WorldMapSweep`, `Town.Available`) is untouched. |
| Shipped content | PASS | Developer-tool evidence below, both preserved roots. |
| Interactions with existing mechanics | PASS | `DIV-105` (scroll-card selection) and `DIV-128` (marker-cache gating) are read; neither is touched, and this story's own `DIV-130` names the remaining marker-identity gap in the same terms `DIV-128` already used, rather than silently reopening it. |

No in-scope GAP.

## Integration witness

`cmd/worldmapcheck` (unchanged by this story) drives the production path — `Town`/`Campaign` state,
building selection, `TownScreen.Choose`, `WorldMapClick`, `WorldMapTick`, `ComposeWorldMap` — through
`FrontEnd.WorldMapWitness` and `FrontEnd.WorldMapSweep`, both already-shipped developer checks. Run
against both preserved installs:

```
AGAINROM_ASSETS=.../gameversions/en  -> world map: chapter 30 accepted mission 30; 1 scroll(s);
  title 10 bytes; briefing 75 bytes; payment 1000; route 227 points; frame composed; mission opened
  world map sweep: 28 mappings, 28 title/briefing pairs, 15 main, 9 side and 4 mapping-only rows;
  26 building-offered; 5 picture and 23 fallback markers; frame composed; all consumed

AGAINROM_ASSETS=.../gameversions/ru  -> world map: chapter 30 accepted mission 30; 1 scroll(s);
  title 11 bytes; briefing 68 bytes; payment 1000; route 227 points; frame composed; mission opened
  world map sweep: (identical counts to EN)
```

227 route points on both roots is the full node-graph corridor path for mission 30's own two
endpoints, not a ten-pixel sample of the straight line between them (which this pairing's on-screen
distance would sample down to roughly 20-30 points under the previous implementation) — confirming
the graph route is live in the production path, not only in the unit tests, and that EN/RU produce
the identical route length, consistent with `TOWN-116`'s byte-identical `PathMap.bmp` on both roots.

The graph's own corpus counts are printed by `cmd/worldmapcheck`, which the landing extended for
them: the story's own measurement came from a probe built against the lane worktree and deleted, and
a number no committed command reproduces is not evidence. Identical output on both preserved roots:

```
world map graph: 3077 corridor, 81 node, 304042 impassable pixel(s); 81 node(s), 180 directed
edge(s), 0 duplicate edge(s); 81 of 81 node(s) reachable from node 0; 35 of 35 MapPoint(s) on a node
```

The pixel counts match `TOWN-116`'s own published corpus exactly. All 35 valid `globalmap.reg`
`MapPoint` values land on a node pixel. The 81 nodes form one connected component. No (source,
destination) pair repeats among the 180 directed edges, so the matrix has no ambiguous cell.
`worldMapGraphDigest` itself is witnessed by a synthetic three-node test in `pkg/game`, mutation
killed by miscounting one pixel class (golden rule 2: no test reads an install).

## Mutation kills

One reverted line per part, each confirmed to redden a test, then restored:

- **Reveal cadence** (`pkg/game/worldmap.go`, `worldMapRouteReveal`): `8` -> `1` reddened
  `TestGatesShowLiveScrollTextSelectRouteAndUseTheMissionDoor` ("route animation prefix = 1 after
  tick, want 8").
- **Stamp cadence** (`pkg/ui/worldmap.go`, `worldMapBallStride`): `8` -> `1` reddened
  `TestWorldMapCompositionStampsBallOnlyAtEveryEighthRevealedCoordinate` (route point 1 painted the
  route colour instead of background).
- **Least-accumulated-length search** (`pkg/game/worldmaproute.go`, `Route`'s edge relaxation):
  dropping `len(edge.points)` from the weight (leaving only `dist[u] + 1`, a fewest-hops search)
  reddened `TestWorldMapRoutePrefersLeastAccumulatedLengthOverFewestHops`: the mutated search took
  the direct 65-point one-hop detour instead of the 36-point two-hop chain through the shorter
  intermediate node.

## Research reconciliation

`TOWN-116`, `TOWN-117`, `TOWN-119`, `TOWN-120` are implemented as described, at their published High
confidence. `TOWN-121` is read for context (arrival/skip ordering) and left unimplemented: this
story is scoped to route construction and reveal, not arrival semantics, which stay as they were
before this story (`contract.md`, out of scope). That mismatch is `DIV-135`, written at the landing:
the original enters the mission only after both completion counters pass their ends, and this build
enters on a second click on the selected scroll with no read of route progress. It was declared out
of scope here and named in the reconciliation, but a mismatch may not live only in a story document.

No claim answers: what the original does when a route endpoint is not itself a node pixel (does not
arise for any shipped `MapPoint`, `DIV-129`); what the original does when the two endpoints are in
disconnected graph components (does not arise — the shipped graph is one connected component,
verified above; `DIV-129`); which accumulated quantity the original's
solver bounds on, `TOWN-119`'s headline and body stating it differently, or the tie-break order for a
non-unique-minimum chain (`DIV-131`; the tie-break is not exercised by the shipped 81-node graph, the
weight is exercised by 101 of its 595 MapPoint pairs); which sprite plays
which role and whether `Cross`/`Flag` are frame-animated (`DIV-130`, deferred with `DIV-128`).

## Milestone census (`pipeline/check-milestone.sh`'s underlying instrument)

```
go build -o /tmp/mr ./cmd/missionrun
AGAINROM_ASSETS=.../gameversions/en /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED  -> 0
AGAINROM_ASSETS=.../gameversions/en /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED  -> 0
```

Both match `pipeline/milestone-baseline.txt`'s state before this story: this story touches only
`pkg/game/worldmap*.go` and `pkg/ui/worldmap.go`, none of which is reachable from `pkg/sim` script
execution or `cmd/missionrun`'s trace path, so the count is structurally unchanged, not independently
re-measured against a checkout of master. This story's own observable result is the world-map route
behaviour above, not a change to this census.
