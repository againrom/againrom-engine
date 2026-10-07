# 1013 — world map arrival: spec

As-built. Canonicalized to the behaviour this story shipped.

## Scope

`pkg/game/worldmap.go` (`townScreen`'s `TownWorldMapScreen` implementation), `pkg/ui/worldmap.go`
(`ui.WorldMapView`/`ComposeWorldMap`), `pkg/ui/app.go` (`stepTownAt`'s cadence gate). Two domains:
Campaign & Scripts and Client. `pkg/sim` is untouched.

## State

`townScreen` carries two pieces of session state beyond the existing `worldMap *worldMapState`:

- `worldPosition image.Point`, `worldPositionSet bool` — the party's own current-position field,
  outside `worldMapState` because it survives leaving and re-entering the world map screen within
  one game. Reset to the town's own first `MapObject` point by `FrontEnd.arriveInTown` on every
  arrival in town (`resetWorldPosition`), whether from a mission verdict or a load path's own
  town-arrival call. Not part of the byte-form save (`DIV-137`).
- `worldSelectedOnce map[int]bool` — which mission numbers have been selected on the world map at
  least once this game. Cleared, along with `worldPosition`, by `resetWorldMapSession` at every
  site that installs a genuinely different game (Behaviour 6); not persisted in the byte-form save
  (`DIV-138`).

`worldMapState` (rebuilt fresh on every `enterWorldMap`) adds two fields: `current image.Point`,
seeded from `townScreen.worldPosition`, written back only at arrival; `frame int`, the `Cross`/
`Flag` animation counter, incremented once per `WorldMapTick` call regardless of whether the route
is revealing.

## Behaviour 1 — Arrival (no second click)

`WorldMapClick(p)`: a hit on a scroll card (`ui.WorldMapCardAt`) calls `selectWorldMission(i)`,
which sets `worldMap.selected = i`, builds the route from `worldMap.current` to the mission's own
anchor (`worldMapRoute`, unchanged since 1010), and marks the mission selected. It returns an empty
`ui.TownAction` — no open here.

`WorldMapTick()` advances `worldMap.frame` unconditionally, then, while a mission is selected and
its route is non-empty, advances `worldMap.shown` by `worldMapRouteReveal` (8), clamped to the
route's own length. The moment `shown` reaches the route's own length, it calls `arriveWorldMap()`
and returns its `ui.TownAction`. `arriveWorldMap` copies the destination anchor into both
`worldMap.current` and `townScreen.worldPosition`, clears the route and selection, and returns
`ui.TownAction{Open: f.MissionOpener(m.Number), ...}` for an enabled mission (a disabled one
returns only its `Problem` message, opening nothing). No further click is read anywhere in this
path: the open happens on the tick that finishes the reveal.

The early-return guard reads `s.selected < 0 || len(s.route) == 0` only — NOT
`s.shown >= len(s.route)`. A guard that also excluded `shown >= len(route)` would treat a route
already fully revealed (behaviour 2's own output state) as idle and never call `arriveWorldMap`,
which was this story's own first defect (see Adversarial self-check below).

## Behaviour 2 — Skip

`WorldMapClick(p)`, when `p` hits neither a card nor a mission region
(`ui.WorldMapCardAt`/`ui.WorldMapRegionAt` both miss) and a mission is selected with
`shown < len(route)`, sets `shown = len(route)` and returns an empty action. It does not itself
call `arriveWorldMap`: the very next `WorldMapTick` call sees `shown == len(route)` already and
arrives immediately, by the same code path as behaviour 1's own completing tick.

## Behaviour 3 — Reveal cadence (Client-owned wall clock)

`pkg/ui/app.go`'s `stepTownAt(in, now)` calls `world.WorldMapTick()` at most once per
`worldMapTickInterval` (100 ms) of wall-clock time: `a.worldMapTickAt` records when it last fired,
and a call is skipped unless it is zero (first call, or the next entry after leaving the world
map, since `stepTownAt` zeroes it when `!onMap`) or `now.Sub(a.worldMapTickAt) >= worldMapTickInterval`.
`pkg/game`'s `WorldMapTick` itself reads no clock and stays a pure per-call step — the gate lives
entirely in `pkg/ui`, which already owns `townSurfaceAt`'s and `chargenChoiceAt`'s wall-clock
reads. `TOWN-121` grades the paint driver's own real-time cadence Unknown, establishing only a
floor ("more than 99 ms" between successful firings of message `0x402`); 100 ms is authored as a
value at least that far apart (`DIV-136`).

## Behaviour 4 — Marker identity and animation

`worldMapAssets.flag`/`.cross` decode every frame of `flag/sprites.16a` and `cross/sprites.16a`
(`sprite16Frames`/`decodeWorldMap16Frames`, via `spr16.DecodeA(raw, true)`), not frame 0 alone.
`flagFrame(n)`/`crossFrame(n)` pick frame `n` from each sheet, wrapping modulo the sheet's own
length (`worldMapPickFrame`). `ui.WorldMapView.Flag`/`.Cross` carry the frame `worldMap.frame`
selects, recomputed on every `WorldMapView()` call. `Hero.bmp` is no longer loaded.

`ui.WorldMapView.Position` is `worldMap.current` (`townScreen.worldPosition`, effectively).
`ComposeWorldMap` stamps `Flag` once, at `Position`, never at the route's leading revealed point
and never per route coordinate. `AtHome` is `worldMap.current == worldMapHomePoint(a)` — the first
`MapObject`'s own point (fallback `(320,240)` if the registry has none). `Available` (`Flag1`) is
drawn only when `AtHome && !traveling` (no route in progress) at the hovered mission's own anchor.

## Behaviour 5 — Marker cache gate

`DIV-128` was open before this story: 1009's own closure scoped the session-state seam out rather
than closing it. This story implements the gate and closes `DIV-128`. `WorldMapView()` resolves
each mission's `Marker` field live, per call, from `worldSelectedOnce[m.Number]` — a picture-bearing
mission with no selection this session paints no marker. `worldSelectedOnce` is cleared at every
site that installs a genuinely different game into the one long-lived `townScreen`
(`resetWorldMapSession`, Behaviour 6) and survives an ordinary mission-to-town return within the
same game.

A mission whose object carries no picture (`Picture` is `"nothing"` or empty) paints no marker at
any time, regardless of selection: `ComposeWorldMap` no longer stamps a placeholder dot for it (the
previous build's `stampAt` fallback is removed). On the shipped campaign this is 23 of the 28 mapped
missions; the remaining 5 carry a picture and show a marker once selected
(`cmd/worldmapcheck`'s `WorldMapSweep`, quoted in `closure.md`'s integration witness).

This story's own addition is `DIV-138`: the cache (`worldSelectedOnce`) is in-memory only. A
save-and-reload of the same in-progress game forgets which missions were selected before the save;
every genuinely-different-game install site clears it outright (Behaviour 6).

## Behaviour 6 — Per-game session reset

`townScreen` is built once by `TownScreen()` and held for the life of the process; loading a game
replaces only the model behind it, never the screen object. `townScreen.resetForNewGame()`
(`pkg/game/townscreen.go`) drops every field of that struct which belongs to the GAME rather than
to the INSTALL, grouped:

- room and conversation: `room`, `dialogueBuilding`, `npc`, `offer`, `said`, `genericTalk`.
- shell and shop presentation: `shopChosen`, `shopShelf`, `shelfBase`, `packBase`, `shopMember`,
  `tavernCell`, `schoolCell`, `townStats`, `shopBook`.
- party-keyed composition caches: `shopFigures`, `shopFigureMasks`, `shopSuppressSlot`,
  `shopSuppressMember`, `shopSuppressFigure`, `shopSuppressMask`.
- world map (`resetWorldMapSession`, Behaviour 5): `worldMap`, `worldPosition`,
  `worldPositionSet`, `worldSelectedOnce`.

Kept, because each is the INSTALL's and not the GAME's: `f` (the back-reference to the owning
`FrontEnd`), `shopIconCache` (item pictures keyed by item code, resolved from the install's own
archives), `shopTip` (re-read from the install's own `shop1.txt` on every shop-room entry), and
`resolver` (rebuilt by `composeShopFaces` on every ROOM ENTRY that can show it, which is not the
same as every conversation: the tavern's own generic mercenary line sets `roomTalk` directly and
calls neither `composeShopFaces` nor `openTownDialogue`. That sub-path is reachable only from an
already-entered `roomTavern`, whose entry rebuilt the resolver for the current game).

`resetForNewGame` runs at every site that installs a genuinely different game: `installCandidate`
(native `.ags` load, both the town-only and mission branches), `RestoreOriginal` (original-save
load, both its between-mission/town branch and its mid-mission branch), and `newGameChargen`'s
`Begin` (a new campaign, reachable again on the same process after a lost mission's own notice
returns to `ScreenMenu`). It does not run from `arriveInTown` alone: that is the ordinary
mission-to-town return within one game, where the marker cache and every other game-scoped field
must survive (only the current-position field resets there, on every arrival, `DIV-137`).

Found across two rounds of this story's own adversarial review, the same shape both times:
round 1's P finding was `RestoreOriginal`'s town branch calling `arriveInTown` alone, which never
cleared `worldSelectedOnce` — a player who selected a picture-bearing mission, then loaded an
original save, saw that mission's marker painted in a game he never selected it in. Round 2's P
finding was the same gap one level up: three of the four install sites called
`resetWorldMapSession` (the world map's own fields) without resetting the rest of `townScreen`, so
a genuinely different game could still show a previous game's own open room, conversation or shop
state. Closed by enumerating the whole struct once, rather than a third field pair found later:
`TestResetForNewGameDropsExactlyTheGamePopulation` (`pkg/game/townscreen_test.go`) walks
`townScreen` with `reflect` on a struct seeded fully non-zero and requires every field back at its
own expected value after `resetForNewGame`, except the four kept fields above — a field added to
the struct later fails that test until it is classified. Round 1's own fields are witnessed by
`pkg/game/worldmapsession_test.go`.

## Out of scope

`selectWorldMission`'s route-construction call and `worldMapRoute`'s node-graph search are
unchanged from story 1010. The keyboard/Enter path (`WorldMapChoose` → `openWorldMission`) is
unchanged: it still opens a selected mission directly with no travel wait, is not reached by
`WorldMapClick`, and remains the vocabulary `headlessActivateWorldMap`'s "walk out to mission N"
scenario target drives — a different door onto the same `MissionOpener`, not this story's subject.

## Headless surface added

`pkg/game/headlesspointer.go`'s `HeadlessPoint` gains three forms: `world_map_mission` (a
scroll card, by mission number), `world_map_town` (the separate return-to-town card), and
`world_map_miss` (a point naming neither a card nor a region — the skip arm's own input).
`pkg/ui/headlesspointer.go` resolves each against the production hit tests
(`WorldMapCardAt`/`WorldMapRegionAt`/`WorldMapCardRect`), the same functions `WorldMapClick`
itself is dispatched through.

The headless dispatch clock (`pkg/ui/chargen_headless.go`'s `headlessAt()`) now advances by 20 ms
per call instead of staying frozen at a single fixed instant: `worldMapTickInterval` is the first
headless-reachable gate that needs elapsed time to make progress at all, rather than only to
debounce a double click within one still instant, which every earlier consumer of the headless
clock tolerated. Every headless dispatch entry point (`HeadlessStep`, `HeadlessKey`,
`HeadlessPointer`, `HeadlessType`, `headlessSelectEntityOnce`) now calls `a.headlessAt()` instead
of reading the old fixed `headlessNow` directly, so the whole headless surface shares one
per-`App` advancing clock.

## Adversarial self-check (found and fixed before push)

`WorldMapTick`'s original guard also excluded `s.shown >= len(s.route)`, which is exactly the
state the skip arm (behaviour 2) leaves the route in. Under that guard, a skip-then-tick sequence
never reached `arriveWorldMap`: the tick that should have opened the destination returned an empty
action instead, silently. Two `pkg/game` tests caught it once written
(`TestGatesShowLiveScrollTextSelectRouteAndUseTheMissionDoor`,
`TestWorldMapArrivalClearsSelectionMovesPositionAndOpensWithNoSecondClick`); the guard was
narrowed to `s.selected < 0 || len(s.route) == 0` alone.
