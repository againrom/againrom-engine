# 1010 — contract

## What this story settles

`docs/DIVERGENCES.md` row `DIV-104` states that this build's world-map travel route is not built
the way the original builds it: this build treats `PathMap.bmp` indices 1 and 2 as one traversable
set, runs an eight-neighbour breadth-first search over pixels from the party's town point to the
nearest traversable point near the destination, and samples the result every ten pixels; a
disconnected or absent mask falls back to a straight sampled line. `EXP-0188` has since published
the mask's own contract, the graph the original builds from it, the search over that graph, and the
reveal cadence, so the correct behaviour is known.

This story rebuilds the route construction, search and reveal over `PathMap.bmp`'s own node graph
and closes `DIV-104`.

## What will work after this story

Selecting a mission's scroll builds the travel route as the original does: the least-accumulated-
length chain over `PathMap.bmp`'s node graph, expanded to that graph's own corridor pixels — not a
pixel-BFS shortest path and not a sampled straight line. Route reveal advances 8 coordinates per
tick and a `BallMap.bmp` stamp is drawn at every eighth revealed coordinate, matching the original's
own paint cadence. Cached mission markers paint before the route, matching the original's own paint
order.

## The observable result

`docs/DIVERGENCES.md`'s `DIV-104` moves to `CLOSED`: the route this build's `worldMapRoute` produces
is the node-graph least-length chain, not a pixel search, and the reveal/stamp cadence numbers match
`TOWN-120`.

## Claims this story is built on

| Claim | Confidence | What it supplies |
|---|---|---|
| `TOWN-116` | High for the operational index meanings and dimensions/counts | `PathMap.bmp` is a 640x480 indexed mask: index 1 corridor, index 2 graph node, every other sampled value impassable. Corpus: 304042 index-0, 3077 index-1, 81 index-2 pixels. All 35 `globalmap.reg` MapPoints land on an index-2 pixel. |
| `TOWN-117` | High | World-map enter unconditionally builds an all-node route-segment matrix from the mask, transfers it into the view, and destroys the bitmap before activation; the bitmap is never blitted. |
| `TOWN-119` | High | The route is the least accumulated precomputed segment length over the node graph, expanded to a pixel list by concatenating the chosen segments' point lists. Not a straight interpolation, not a per-frame pixel search. |
| `TOWN-120` | High | Paint order: `GMap.bmp`; cached mission markers; `BallMap.bmp` at every eighth revealed route coordinate; animated destination `Cross`; hovered-scroll `Flag1` after completion; current-position `Flag`; scroll cards. Route progress advances 8 coordinates per paint; `Cross`/`Flag` frame counters advance once per paint. |
| `TOWN-121` | High for ordering / Unknown for exclusive wall-clock paint cadence | Travel completes before the destination opens and is skippable. The one proved paint driver throttles to more than 99 ms between paints; whether another path also paints is Unknown. |

## Aspects that apply

| Aspect | Applies |
|---|---|
| Data | No — no format changes; `PathMap.bmp` is decoded exactly as before. |
| Runtime state | Yes — the world-map view's route list and reveal counter are rebuilt from the node graph. |
| Simulation | No — outside `pkg/sim`; the route is Client-side presentation state. |
| Player input | No — selection and click handling are unchanged. |
| AI | No. |
| UI / HUD | Yes — the drawn route, its reveal cadence and its stamp cadence. |
| Triggers / scripts | No. |
| Inventory / equipment | No. |
| Persistence / save-load | No — `TOWN-117`'s own evidence finds no save serializer for these view fields; this build's existing snapshot test already asserts the world map leaves the save unchanged. |
| Campaign / session | No — which missions are offered and their mapping are untouched. |
| Shipped content | Yes — verified against the installed `PathMap.bmp` and all 35 registry MapPoints (developer tool evidence, not a test). |
| Interactions with existing mechanics | Yes — `DIV-105`'s scroll-card selection and `DIV-128`'s marker-cache gap are read alongside this row; neither is touched. |

## Domains touched

**Client** (`pkg/ui/worldmap.go`: paint order, reveal/stamp cadence) and **Campaign & Scripts**
(`pkg/game/worldmap.go`, new `pkg/game/worldmaproute.go`: the node graph, the search, route
construction on mission selection).

## Divergence allocation

`DIV-104` closes in place. `DIV-129`, `DIV-130`, `DIV-131` are reserved for this story:

- `DIV-129` — the graph-unusable fallback (missing mask, an endpoint that is not itself a node
  pixel, or two nodes with no connecting chain) falls back to a straight sampled line. No claim
  describes this case, and it does not arise for any shipped registry MapPoint (verified: all 35
  land on a node, and the 81-node corpus graph is one connected component).
- `DIV-130` — this build still draws the traveling marker at the route's leading revealed point
  (`Hero.bmp`) instead of `TOWN-120`'s own current-position `Flag`, has no marker-cache-gated `Flag1`
  hover marker, and does not animate `Cross`/`Flag` frame counters. Out of this story's scope for the
  same reason `DIV-128` gave for the adjacent marker-cache gap: marker identity and animation are a
  separate concern from route construction.
- `DIV-131` — the least-accumulated-length search's tie-break (when two node chains have equal
  total length) is authored: research does not fix the exact per-edge weight or the original's
  tie-break order, and the shipped 81-node graph has a unique minimum for every path exercised so
  far.

## Out of scope

Marker identity and gating (`DIV-128`, `DIV-105`), arrival semantics (auto-opening the mission on
completion versus this build's explicit second click), audio (no world-map sound exists in this
build today and none is added), and the wall-clock paint cadence (`TOWN-121` leaves it Unknown; this
build calls `WorldMapTick` once per frame, unchanged by this story).
