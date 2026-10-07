# 1007 — world map

## Result

The town gates open a visible install-backed world map instead of a mission list. The map shows one
separate town scroll and a paged scroll for every live accepted mission. Each mission scroll shows
its shipped title, briefing and `Payment`. Selecting a destination places its flag and animates the
party route. A second release on the selected mission scroll, or Enter, starts the mission through
the existing campaign mission opener.

The result works with both lawful installs. It reads each root's own background, route mask,
markers, scrolls, mission text and registry. Missing optional art uses a stable fallback. Invalid
registry data leaves the affected mission visible with a reason and never retargets another
mission. The town scroll and Escape return to the town without changing the accepted offers, purse,
party, equipment, completed missions, shop state or next save bytes.

The observable result is the world-map frame in the production campaign and the non-windowed
`cmd/worldmapcheck` drive. The drive accepts a shipped building offer, enters the map, selects and
animates its route, returns to the town, re-enters, and opens the shipped mission.

## Claims and owner-directed behaviour

The research pin is `4aae01f`. The listed claims are active at that pin.

| Claims | Contract use |
|---|---|
| `TOWN-036`, `TOWN-037` | Background and fixed global-map asset addresses |
| `TOWN-038`, `TOWN-039` | `MapPoint`, width-and-height `MapRect`, ordered rectangular hit testing |
| `TOWN-040` | Live offered missions are one availability source; the separate ROM1 per-region flag remains Unknown |
| `TOWN-041` | Optional mission `Picture`, `nothing`, mission-number cache identity and root-specific marker bytes |
| `TOWN-042` | `PictureOffset` has no shipped producer and its effect remains Unknown |
| `TOWN-043` | `PathMap.bmp` is loaded conditionally; this pin does not establish its consumer |
| `TOWN-044` | Mission-number `title.txt` and `briefmap.txt` paths and the `Payment` display path |
| `TOWN-045` | No progress indicator was found in the searched paths; the result is scoped to those paths |
| `REG-SCN-059`, `REG-GMAP-066` | Campaign registries, map objects and mission-to-object mappings |
| `REG-SCN-065`, `REG-SCN-067` | Listed versus live offered missions; `Payment` as mission cash reward |
| `MISSION-TEXT-005` | Mission-number text family and shipped missing-file behaviour |
| `SPR256-EXC-017` | Secondary data in the relevant `.256` pictures is runtime-inert |

The owner directed the screen to use `PathMap.bmp` as the route topology, to place mission scrolls
at the top, to provide a separate town scroll, and to show a flag and a constructed party route when
a destination is selected. This directive supersedes the prepared contract's earlier omission of
the route. The pinned research does not establish the route-mask values, starting object, path
search, sampling cadence, scroll layout or activation gesture. These are implemented as authored
behaviour and recorded in `DIV-104` and `DIV-105`. The missing ROM1 per-region availability flag is
recorded in `DIV-106`. The omitted `PictureOffset` effect is recorded in `DIV-107`.

## Customisation boundary

`ObjectCount`, `MapObject<n>`, `Mission<n>` mappings, coordinates, rectangles, optional picture
names, mission numbers and mission text are data. The shipped counts are not engine bounds. Invalid
objects retain their declared indices. Every live offer receives a scroll even when its mapping is
invalid.

The fixed world-map asset names are engine literals held by one lazy manifest. Replacing bytes at
those addresses changes no shipped file format. This story changes no shipped byte and adds no
serialized field. Supporting another fixed manifest is a Client/Assets configuration seam.

## Aspects and domains

| Aspect | Scope |
|---|---|
| Data | Global-map registry, background, route mask, markers, scroll art and mission text |
| Runtime state | Hovered mission, selected mission, route and visible route prefix |
| Simulation | N/A; no Sim Core state, command, digest or tick rule changes |
| Player input | Pointer selection, scroll/card priority, arrows, wheel, Enter, town scroll and Escape |
| AI | N/A |
| UI / HUD | Native 640×480 map composition and frame placement |
| Triggers / scripts | Selected missions enter the existing production mission and script path |
| Inventory / equipment | N/A; state is preserved and not inspected by the map |
| Persistence / save-load | Map-only state is not serialized; town and party snapshots remain byte-identical |
| Campaign / session | Town gates, live accepted offers, carried party and mission opener |
| Shipped content | Both roots, all mappings and all title/briefing pairs |
| Interactions | Town acceptance, mission start, language-specific font conversion and larger windows |

Touched domains: **Assets**, **Client**, **Campaign & Scripts**, **Town & Economy** and
**Persistence** at its unchanged snapshot boundary.

## Out of scope

The main-menu picker, loose-map browsing, mission-map rendering, combat HUD, campaign progression,
offer lifetime, reward application, town-room redesign, map editing and a campaign-progress
indicator. Publishing or consuming the unfinished EXP-0188 research branch is outside this story.
