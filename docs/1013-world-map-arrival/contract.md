# 1013 — world map arrival: contract

## Owner directive

2026-08-18, verbatim: «клик на свиток показывает анимацию движения на карте и сразу затем
запускает миссию — второго клика по свитку делать не надо.» One click on a mission scroll shows
the travel animation on the world map and then starts the mission automatically. No second click
on the scroll.

## Result

After this story: clicking a mission's scroll card once selects it, starts travel, and the mission
opens by itself once the route reveal completes. No further click on the scroll is needed. A click
that misses every scroll while travel is in progress skips the remaining animation and the mission
still opens by itself on the next paint. The paint/reveal cadence is gated on wall-clock time
rather than on the fixed-tick simulation clock. The current-position marker is the animated `Flag`
sprite at the party's own position, not the static `Hero.bmp`; a hover marker (`Flag1`) shows only
while at the town's own point with no travel in progress. A picture-bearing mission's own marker
still paints only after it has been selected at least once this session (`DIV-128`, implemented by
this story — 1009's own closure scoped the session-state seam out, and `worldSelectedOnce` is new
code in this story's diff; this story also closes the residual identity/animation gap under
`DIV-130`). A mission whose object carries no picture (`Picture` is `"nothing"` or empty) paints no
marker at all, at any time: this story removes the placeholder dot the previous build stamped for
such a mission. On the shipped campaign this is 23 of the 28 mapped missions; only 5 show a marker
once selected.

## Behaviours (six, one contract)

1. **Arrival.** A click on a mission's scroll selects it and starts travel; the destination opens
   automatically when the route reveal completes, with no second click. Closes `DIV-135`.
2. **Skip.** A click that lands on neither a scroll nor a mission region, while travel is in
   progress, fast-forwards the reveal to the route's own end; the next paint then opens the
   destination — `TOWN-121`'s own skip arm.
3. **Reveal cadence.** The call rate that advances route reveal and the `Cross`/`Flag` frame
   counters is gated on wall-clock time in `pkg/ui/app.go` (the Client domain, which already owns
   wall-clock reads for `townSurfaceAt`/`chargenChoiceAt`). `pkg/game`'s `WorldMapTick` stays a
   pure per-call step; `pkg/sim` is untouched. `TOWN-121` grades the exact cadence Unknown, so the
   value is authored (`DIV-136`).
4. **Marker identity and animation.** The current-position marker is `Flag`, not `Hero.bmp`, drawn
   at the party's own current-position field, which changes only at travel completion. `Cross` and
   `Flag` are both frame-animated, one frame per paint. `Flag1` is a hover marker at a hovered
   scroll's own map point, shown only while at the town's own point (`AtHome`) with no travel in
   progress.
5. **Marker cache gate.** A picture-bearing mission's own marker is not painted until it has been
   selected at least once this session. `DIV-128` was open before this story — 1009's own closure
   scoped the session-state seam out — and this story implements the gate (`worldSelectedOnce`) and
   closes it. This story also closes the marker identity/animation gap left under `DIV-130`, and
   records the cache's session-only lifetime as `DIV-138`. A mission with no picture paints no
   marker regardless of selection: the placeholder dot the previous build stamped for it is removed
   (23 of 28 shipped mapped missions; see Result).
6. **Per-game session state does not survive into a different game.** `townScreen` is built once
   and held for the life of the process; loading a game replaces only the model behind it, not the
   screen object. Every field that belongs to the GAME rather than to the INSTALL — the world map's
   own two fields (behaviour 5), plus the room, the open conversation, the shop and shell
   presentation, and the party-keyed composition caches — is dropped at every site that installs a
   genuinely different game: a native load, an original save (both its town and mission branches),
   and a new campaign start. An ordinary mission-to-town return WITHIN one game drops none of it
   except the current-position field, which resets on every arrival regardless of game identity.
   Found by this story's own adversarial review across two rounds, the same shape each time —
   round 1 for the world map's own fields, round 2 for the rest of the struct: `resetForNewGame`
   (`pkg/game/townscreen.go`) enumerates and resets the whole population, held to that population by
   a reflection-based test (`TestResetForNewGameDropsExactlyTheGamePopulation`,
   `pkg/game/townscreen_test.go`) a field added to the struct later must pass before landing.

One contract, not six stories: all six are one mechanism — the same `WorldMapTick`/`WorldMapClick`
pair, the same presentation snapshot (`ui.WorldMapView`), and the same long-lived screen object
whose per-game state they share — read from the one piece of owner testimony above. Splitting them
would buy six doc stacks for one result.

## Domains touched

- **Campaign & Scripts** (`pkg/game`): `townScreen.WorldMapClick`/`WorldMapTick`/`WorldMapView`,
  mission opener, the party's own current-position and selection-history state.
- **Client** (`pkg/ui`): `app.go`'s wall-clock cadence gate, `ComposeWorldMap`'s marker draw order,
  the headless pointer/dispatch surface used to drive the new one-click path from a scenario.

No other domain (`pkg/sim`, Town & Economy, Persistence) is touched. `pkg/sim`'s determinism wall
is not reached: `WorldMapTick` takes no simulation tick and posts no simulation command.

## Research claims

`TOWN-118` (High) — a mission is selected by clicking its scroll entry, never a map region.
`TOWN-119` (High) — the route is the least-accumulated-segment node-graph path (already
implemented by story 1010, unchanged here). `TOWN-120` (High) — paint draw order, route-per-paint
advance of 8, `Cross`/`Flag` frame counters. `TOWN-121` (High for ordering/skip/dispatch, Unknown
for the exact wall-clock paint cadence) — travel completes and opens the destination with no
second click; a miss-click skips. `TOWN-123` (High) — the marker cache is the persisted,
save-carried selection history. `TOWN-040`'s active clause (High, amended) — the map-rectangle
gate's two sources, cited for the marker-cache identity `TOWN-123` depends on.

## Divergences

Reserved: `DIV-136`–`DIV-139`. Spent: `DIV-136` (reveal cadence, authored value), `DIV-137`
(current position across a town arrival, authored reset policy), `DIV-138` (marker-selection cache
is session state, not the persisted save field `TOWN-123` decodes). Returned unused: `DIV-139` —
no third technical fact was found distinct from `DIV-137` and `DIV-138`. Closed by this story,
moved to `DIVERGENCES-CLOSED.md`: `DIV-128`, `DIV-130`, `DIV-135`.

## Out of scope

- Persisting `worldSelectedOnce` or `worldPosition` in the save format (`formatVersion` is not
  bumped). Both are typed UNKNOWN rows (`DIV-137`, `DIV-138`), not silent gaps.
- The exact wall-clock paint cadence `TOWN-121` leaves Unknown (`DIV-136`).
- The keyboard/Enter direct-open shortcut (`openWorldMission`, `WorldMapChoose`) — an existing
  scenario vocabulary path (`headlessActivateWorldMap`'s "walk out to mission N") kept unchanged;
  it is not the click path this story's contract is about.

## Adversarial review ceiling

Three passes: no hashed simulation state is reached, and two domains are touched (Campaign &
Scripts, Client).
