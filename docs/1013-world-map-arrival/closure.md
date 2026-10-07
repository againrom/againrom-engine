# 1013 — world map arrival: closure

## Twelve-aspect matrix

| Aspect | Result | Notes |
|---|---|---|
| Data | PASS | `flag/sprites.16a`, `cross/sprites.16a` decoded whole (all frames), not frame 0 alone. `Hero.bmp` no longer read. |
| Runtime state | PASS | `townScreen.worldPosition`/`worldSelectedOnce` (session-scoped, `DIV-137`/`DIV-138`); `worldMapState.current`/`.frame`. |
| Simulation | N/A | `pkg/sim` untouched; `WorldMapTick` posts no simulation command and reads no simulation tick. |
| Player input | PASS | One click (`WorldMapClick`) selects and starts travel; a miss-click skips; no second click opens. |
| AI | N/A | Not reached. |
| UI/HUD | PASS | `ComposeWorldMap` marker draw order (`Flag` at `Position`, `Flag1` hover-only, `Cross`/`Flag` frame-animated). |
| Triggers/scripts | N/A | The world map is a front-end screen; no mission script is running while it is open. |
| Inventory/equipment | N/A | Not reached. |
| Persistence/save-load | GAP, disclosed | `worldPosition`/`worldSelectedOnce` are session state, not in the byte-form save. Typed `DIV-137`/`DIV-138`, `formatVersion` deliberately not bumped (out of contract scope). |
| Campaign/session | PASS | `arriveWorldMap` opens through `FrontEnd.MissionOpener`; `arriveInTown` resets the current-position field on every town arrival, but not the marker cache, which must survive an ordinary mission-to-town return within one game. `resetForNewGame` (`pkg/game/townscreen.go`) resets `townScreen`'s WHOLE per-game session population — the world map's own fields plus room, conversation, shop/shell presentation and the party-keyed composition caches — at every site that installs a genuinely different game (`installCandidate`, both `RestoreOriginal` branches, `newGameChargen`), enforced against the struct itself by `TestResetForNewGameDropsExactlyTheGamePopulation` (`pkg/game/townscreen_test.go`). |
| Shipped content | PASS | `WorldMapWitness`/`WorldMapSweep` run against both EN and RU installs (below). |
| Interactions with existing mechanics | PASS | `selectWorldMission` routes from the party's own current position (1010's `worldMapRoute`, unchanged); the keyboard/Enter direct-open path (`WorldMapChoose`) is untouched and still exercised by the pre-existing scenario `0163-mission-to-town.json`. |

The one disclosed GAP (persistence) is a typed, owner-authored divergence, not a silent omission:
`formatVersion` is deliberately not bumped this story, per brief.

`townScreen` is built once by `TownScreen()` and held for the life of the process; loading a game
replaces only the model behind it, never the screen object. `resetForNewGame` enumerates and
resets every field of that struct which belongs to the GAME rather than to the INSTALL — 25 of its
29 fields, grouped as room/conversation, shop/shell presentation, party-keyed composition caches,
and the world map (behaviour 5's own two fields plus `worldMap` itself) — at each of the four sites
that install a genuinely different game. The remaining 4 fields (`f`, `shopIconCache`, `shopTip`,
`resolver`) are the install's own and are left untouched, each for a stated reason in
`resetForNewGame`'s own doc comment. `TestResetForNewGameDropsExactlyTheGamePopulation` walks the
struct with `reflect` on a value seeded fully non-zero and requires every field back at its
expected value after the call, except the four named exceptions: a field added to the struct later
fails that test until it is classified.

## Integration witness

New headless scenario `scenarios/1013-world-map-one-click.json` (stage `frontend`, `assets:
install`): loads an original save, crosses a mission verdict into town, takes work at the tavern,
opens the gates (world map), presses and releases once on mission 30's own scroll card (`{"pointer",
"press"/"release", "at": {"world_map_mission": 30}}`), then `wait_until`s `{"screen": "map"}` with
**no further pointer step of any kind**. The mission opens on its own from the reveal ticks alone,
which is the story's own contract statement ("no second click"). Run:

```
$ AGAINROM_IMPL=<worktree> bash pipeline/check-scenarios.sh "<en root>"
check-scenarios: selected 12 scenario(s) under <worktree>/scenarios
check-scenarios: ok   scenarios/1013-world-map-one-click.json
check-scenarios: ok (12 of 12)
```

Baseline before this story was 11/11 (`pipeline/check-scenarios.sh`'s own selection count); all 11
pre-existing scenarios still pass unmodified, including `0163-mission-to-town.json`, which still
drives the pre-existing keyboard/Enter direct-open path unchanged.

`cmd/worldmapcheck`'s `WorldMapWitness` (rewritten this story for the one-click contract) and
`WorldMapSweep`, run against both preserved installs:

```
$ AGAINROM_ASSETS=<en root> worldmapcheck
world map: chapter 30 accepted mission 30; 1 scroll(s); title 10 bytes; briefing 75 bytes;
payment 1000; route 227 points; one click, skip completed travel, no second click; arrived at
(386,166); frame composed; mission opened
world map sweep: 28 mappings, 28 title/briefing pairs, 15 main, 9 side and 4 mapping-only rows;
26 building-offered; marker cache gate held before selection for all; 5 picture marker(s) painted
and 23 with no picture drawing nothing, once selected; frame composed; all consumed

$ AGAINROM_ASSETS=<ru root> worldmapcheck
world map: chapter 30 accepted mission 30; ...; route 227 points; one click, skip completed
travel, no second click; arrived at (386,166); frame composed; mission opened
world map sweep: 28 mappings, ...; marker cache gate held before selection for all; 5 picture
marker(s) painted and 23 with no picture drawing nothing, once selected; frame composed;
all consumed
```

Of the 28 shipped mapped missions, 23 carry no picture and paint no marker at all, on either root;
the placeholder dot the previous build drew for such a mission is removed this story (`DIV-107`).
Only 5 missions show a marker, and only once selected (behaviour 5).

Both roots agree on route length (227 points) and arrival point (386,166); `WorldMapGraphWitness`
(unchanged this story, run alongside) still reports 81/81 nodes reachable and 35/35 registry
MapPoints on a node on both roots.

## Milestone census

`pipeline/milestone-baseline.txt` records no `cannot run` rows for mission 10 or mission 20 on
either root. This story is Client/Campaign-only and does not touch script-node support; measured
directly:

```
$ go build -o missionrun ./cmd/missionrun
$ AGAINROM_ASSETS=<en root> missionrun -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED
0
$ AGAINROM_ASSETS=<en root> missionrun -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED
0
```

Both unchanged from before this story (baseline also carries no `cannot run` row for either
mission).

## Added at the landing

The round-3 adversarial review returned MERGE with no P finding and two others, both fixed here
rather than carried:

- **D.** `resetForNewGame`'s own doc and this story's `spec.md` both stated that every path into
  `roomTalk` calls `composeShopFaces` immediately before `openTownDialogue`. That is not universal:
  the tavern's generic mercenary line (`pkg/game/townshell.go`, the squad row) sets `roomTalk`
  directly and calls neither. `resolver` is still never read stale, for a different reason: the
  sub-path is reachable only from an already-entered `roomTavern`, and room entry is what rebuilds
  the resolver. Both texts now state the per-room-entry invariant.
- **W.** `resetWorldMapSession` covered the world-map four and had one production caller. With two
  whole-session resets callable, a later install site reaching for the narrower one would have left
  the other 21 fields stale while every test stayed green, because the four per-site tests in
  `pkg/game/worldmapsession_test.go` asserted world-map fields only. The helper is folded into
  `resetForNewGame`, so one reset is callable; `seedWorldSession` now seeds `room` and
  `assertWorldSessionCleared` asserts it, so a site calling any narrower reset fails those four
  tests. Production behaviour is unchanged by both edits.

## Research reconciliation

`TOWN-118` (High) — implemented unchanged since 1007: mission selection is card-click-only.
`TOWN-119` (High) — implemented unchanged since 1010: node-graph least-accumulated-length route.
`TOWN-120` (High) — implemented this story for the residual identity/animation gap left under
`DIV-130`: `Flag` (not `Hero.bmp`) at the party's own position, `Cross`/`Flag` frame counters
advancing once per paint. `TOWN-121` (High for ordering/skip/dispatch) — implemented this story:
arrival with no second click, the skip arm. `TOWN-121`'s own Unknown clause (exact wall-clock
cadence) is carried forward as `DIV-136`, authored at 100 ms. `TOWN-123` (High) and `TOWN-040`'s
active clause — the marker-cache identity (`DIV-128`) is implemented by this story:
`worldSelectedOnce` is new code in this diff, and 1009's own closure scoped this session-state seam
out rather than closing it. This story's own addition is recording the cache's session-only
lifetime as `DIV-138`, since `TOWN-123` decodes a persisted, save-carried cache and this build's is
not.

No claim read for this story states what ROM1 does with the current-position field across a town
return or a save/load (`DIV-137`, UNKNOWN, authored: reset to town start on every town arrival).

Every mismatch found is either fixed (marker identity/animation, arrival/skip mechanism) or
recorded as a typed divergence row (`DIV-136` reveal cadence, `DIV-137` position across a town
return, `DIV-138` marker cache persistence) — none is a silent gap.

## Divergence ledger

Reserved: `DIV-136`–`DIV-139`. Spent: `DIV-136`, `DIV-137`, `DIV-138` (all UNKNOWN, `docs/DIVERGENCES.md`'s "Authored where research is silent" table, except `DIV-138` which is FIDELITY-DEBT in the main table). Returned unused: `DIV-139` — no third technical fact distinct from `DIV-137`/`DIV-138` was found. Closed, moved to `docs/DIVERGENCES-CLOSED.md`: `DIV-128`, `DIV-130`, `DIV-135`.

## Gate

```
go build ./...                                                          clean
go vet ./...                                                            clean
gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')    clean (no output)
go test -trimpath -count=1 ./...                                        ok, all packages
bash scripts/check-no-game-assets.sh                                    check-no-game-assets: clean (tree scan)
AGAINROM_IMPL=<worktree> bash pipeline/check-scenarios.sh "<en root>"   ok (12 of 12)
```

## Open items

- `DIV-136` (reveal cadence exact value), `DIV-137` (position across a town return/save), `DIV-138`
  (marker cache persistence) are all open UNKNOWN/FIDELITY-DEBT rows, each naming what would close
  it (a claim, or a persistence story).
- The keyboard/Enter direct-open shortcut (`WorldMapChoose`/`openWorldMission`) is unchanged and
  out of scope; it remains a second, un-timed way to reach a mission from the world map, distinct
  from this story's click contract.
- Round-1 adversarial review, at `ab8a9f6`, returned the story: one P finding (the marker cache
  survived a load through `RestoreOriginal` that installed a genuinely different game) and three D
  findings (the DIV-128 attribution, the `WorldMapSweep` wording, and DIV-107's cell), all fixed in
  place on this branch.
- Round-2 adversarial review, at `d71aa66`, returned the story again: one P finding of the SAME
  CLASS as round 1's, in a different field pair — `room` and `worldMap` (and 23 other fields)
  survived into a genuinely different game at three of the four install sites, because those sites
  called `resetWorldMapSession` (the world map's own two fields) without also resetting the rest of
  `townScreen`'s per-game state. Closed by enumeration rather than by a third field pair: `Behaviour
  6` (spec.md) and `resetForNewGame` (`pkg/game/townscreen.go`) now cover the whole population, held
  to it by `TestResetForNewGameDropsExactlyTheGamePopulation`.
- `DIV-139` remains returned unused across both rounds: no finding in either round named a technical
  fact distinct from `DIV-137`/`DIV-138` — both rounds' P findings are defects fixed in this build's
  own code, not divergences from ROM1.
