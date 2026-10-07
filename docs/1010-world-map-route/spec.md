# 1010 — spec (as-built)

## Subject

How this build constructs and reveals the world-map travel route between the party's current point
and a selected mission's map point, and how that matches `PathMap.bmp`'s own graph, search and paint
cadence as `EXP-0188` describes them.

## The node graph (`pkg/game/worldmaproute.go`)

`buildWorldMapNodeGraph(mask *image.Paletted) *worldMapNodeGraph` builds the graph unconditionally
from the decoded `PathMap.bmp` mask (`TOWN-117`, High: the original's own construction is
unconditional at world-map enter, before the first-position branch that may build mission scrolls).
This build constructs it once per session, when `worldMapAssets()` first loads the mask, and caches
it on `worldMapAssets.graph`.

Construction (`TOWN-116`/`TOWN-117`, High):

1. Scan every pixel of the mask for index 2. Each is a node, in raster scan order (`nodes[]`,
   indexed 0..80 for the shipped mask).
2. From every node, walk eight-neighbour index-1 corridor pixels outward (depth-first, marking each
   visited within that node's own walk). A branch stops when it reaches a different node pixel
   (index 2): that corridor becomes one directed edge, carrying the corridor pixels strictly between
   the two nodes, in travel order away from the source. A branch that reaches only dead-end index-1
   pixels or the map boundary records nothing.

Verified against the installed `graphics/global.map/pathmap.bmp` (both preserved roots share one
SHA-256 per `TOWN-116`): 81 node pixels, 3077 corridor pixels, matching the claim's own corpus
counts exactly. All 35 valid `globalmap.reg` `MapPoint` values land exactly on a node pixel. The 81
nodes form one connected component (BFS from node 0 reaches all 81). 8 corridor pixels have more
than two passable neighbours (genuine branch points in the shipped mask); no two of the 180
discovered directed edges collide on the same (source, destination) pair, so this build's matrix has
no ambiguous cell to resolve. (Developer-tool evidence; not a test — golden rule 2 forbids a test
from reading an install.)

## The search (`(*worldMapNodeGraph).Route`)

`Route(start, end image.Point) (path []image.Point, ok bool)` maps both endpoints to node indices by
exact pixel-coordinate match against the graph's node list, then runs a Dijkstra shortest-path search
over the node graph, weighting each edge by its own corridor pixel count plus one (for the
destination node pixel), and returns the winning chain's nodes and edges concatenated into one pixel
list: `[node0] + edge(0,1).points + [node1] + edge(1,2).points + [node2] + ...`.

This produces the least accumulated segment length over the graph (`TOWN-119`, High), matching the
original's own stated result. `TOWN-119` describes the original's search as a recursive
branch-and-bound rather than Dijkstra; the two produce the identical winning chain for any graph
with a unique minimum under the same edge weight.

`TOWN-119` does not state one edge weight. Its headline gives the accumulated segment LENGTH and its
own body gives the best accumulated segment COUNT. This build takes the headline reading. The two
readings select different chains for 101 of the 595 unordered MapPoint pairs on the shipped graph,
and for 4 of the 34 routes from the town start point, so the choice is not cosmetic (`DIV-131`,
which carries the measurement). No claim fixes the tie-break order for a graph with more than one
minimum chain; the shipped graph has a unique minimum for every MapPoint pair under the implemented
weight.

`ok` is false, and the caller falls back to a ten-pixel-sampled straight line, when either endpoint
is not itself a node pixel or when no chain connects the two nodes (`DIV-129`: authored, does not
arise for shipped content — see above).

## Wiring (`pkg/game/worldmap.go`)

`selectWorldMission` calls `worldMapRoute(assets.graph, start, m.Anchor, worldMapFallbackStep)` where
`start` is the party's town point (registry `MapObject1`, `Objects[0].Point`, unchanged from before
this story) and `m.Anchor` is the selected mission's own registry point. `worldMapRoute` prefers the
graph route and falls back to the sampled line only when the graph returns `ok=false`.

`WorldMapTick` advances `worldMapState.shown` by `worldMapRouteReveal` (8) per call, clamped to the
route's own length (`TOWN-120`, High: route progress advances 8 coordinates per paint). This build
still calls `WorldMapTick` once per rendered frame; `TOWN-121` leaves the original's own wall-clock
paint cadence Unknown beyond "more than 99 ms between paints on the one proved driver", so this
story makes no claim about matching real time and changes nothing about the call site.

## Paint (`pkg/ui/worldmap.go`, `ComposeWorldMap`)

Draw order now follows `TOWN-120` (High) for the route-relevant part of its own sequence: background,
then cached mission markers, then route stamps, then the selected mission's destination cross and
current-position marker, then scroll cards. `BallMap.bmp` is stamped only at indices that are
multiples of `worldMapBallStride` (8) and below the revealed count — "every eighth revealed
coordinate" — not at every revealed point as before this story.

This build does not reorder or rename the markers themselves: the traveling marker remains
`Hero.bmp` drawn at the route's leading revealed point, the destination marker at the selected
mission's anchor remains `Cross` and `Flag` together, and the picture-less-mission fallback remains
`Available` (`Flag1`'s asset). `TOWN-120` names a different assignment — a static current-position
`Flag`, a hover-driven `Flag1`, animated `Cross`/`Flag` frame counters — which this story leaves
alone (`DIV-130`, out of scope for the reason `DIV-128` already gave for the adjacent marker-cache
gap: marker identity and animation are a separate concern from route construction).

## Fallback and fixture coverage

`sampledLine` (unchanged) remains the fallback for a missing mask, a missing/absent graph, or the two
`DIV-129` cases above. Synthetic tests build small hand-designed masks and assert literal expected
node lists, edge point lists, and route point lists — never a value derived by calling the router
itself:

- `TestWorldMapNodeGraphFindsEveryNodeAndItsDirectEdges` — a 3-node, 2-edge mask; asserts each edge's
  exact point list.
- `TestWorldMapRoutePrefersLeastAccumulatedLengthOverFewestHops` — a mask where the direct one-hop
  edge is a 63-pixel detour (weight 64) and a two-hop chain is 33 corridor pixels total (weight 35);
  asserts the route takes the two-hop chain.
- `TestWorldMapRouteFallsBackWhenEndpointIsNotANodeOrMaskIsMissing` — `DIV-129`'s two fallback cases.
- `TestGatesShowLiveScrollTextSelectRouteAndUseTheMissionDoor` (integration-level, `pkg/game`) — a
  640x480 fixture with two nodes joined by an L-shaped corridor; asserts the route is exactly the
  corridor's 51 pixels, not a 10-pixel sample of the straight line between the endpoints, and that
  one `WorldMapTick` call advances `RouteShown` by 8.
- `TestWorldMapCompositionStampsBallOnlyAtEveryEighthRevealedCoordinate` and
  `TestWorldMapCompositionPaintsRouteOverMissionMarkers` (`pkg/ui`) — the stamp cadence and the
  marker-before-route paint order, each against a hand-built `WorldMapView`.

All of `go test ./...` runs with no game install present (golden rule 2); the developer-tool
evidence above is separate, from `cmd/worldmapcheck` and the corpus figures quoted in this spec, run
against the installed EN root only for this story's own verification.
