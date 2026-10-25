# A route witness of the second game from the main menu to the first mission

## Intent

One release test walks the second game on one install through the App's headless input, from the main menu to the first mission's first ticks, and records every step: screen, control, its rectangle, action, result, the text the step shows, a frame and the state the game holds. The record is evidence to compare against the original's own route; the test itself asserts that each step reaches its screen.

## Authority

Owner direction is the authority for the route. The behaviour each step should show is the original's; this story claims no original behaviour and adds no divergence. The screens are the ones the existing second-game release tests already cite (for example R2-ENGINE-437 for the window, R2-ENGINE-249 and DIV-3046 for the shop page). DIV-4057 and DIV-4058 are unused.

## As built

`TestReleaseSecondGameRouteWitness` (`pkg/game/routewitness_release_test.go`) holds one step table, `secondRouteStages`: menu, generator pre-create page, generator detail page, town square, town statue, tavern, shop, way out, world map, mission dialogue, mission screen. Each stage names the screen it opens on and a walk. Controls come from each screen's own hit test or layout: the menu mask's `ButtonAt` and binding names; `ui.PreCreateControlAt` and `ui.DetailedControlAt` for the generator; the square scene's `ControlAt` and `TipAt`; the inn page's `Seats.SeatAt` and `town.ButtonAt`; the shop view's layout and `HeadlessShopPoint`; `ui.WorldMapCardAt` and `ui.WorldMapRegionAt`; on the mission screen the target `HeadlessTooltip` reports over the whole frame at an 8-pixel step.

A hover rests the pointer past the hover box delay on the 20 ms headless clock. A click is hover, press and release with a frame composed after each edge. The generator walk clicks every control, then returns to the opening values through Back and Forward and asserts them before Play. The tavern walk talks to every seat and pages each dialogue by its own button; the first town's inn seats no hire actor, which is recorded as a row. The shop walk opens each shelf, hovers its first item, buys the cheapest item and sells it back, asserting the gold each way. The town has no hero information or inventory screen (the second game's shop page has no character panel, DIV-3046); the hero and his pack are recorded on the mission screen's character pane, whose pack corner is pressed twice. In route mode the world map answers no card or site and travels by itself; the travel is recorded every 40 steps. The mission is paused with key `0` while its controls are hovered; the hero's worn, doll and pack cells share the target `item/<id>` and are told apart by their hover lines, one row each (`item/29#1` to `#6` in the first mission). The town screen reads the App's headless clock, so two runs write byte-identical frames.

With `AGAINROM_ROUTE_WITNESS_DIR` set, the test writes one PNG per step through the release tests' existing `writeTipRender` and one `route.tsv` under `<dir>/<edition>/`; the directory is refused inside the install. Without it the test writes nothing and still asserts every step. Frames off the mission are the App's 640x480 frame. The mission screen has no composed CPU frame, so its rows carry the part that shows: the notice window, the hover box, the character pane, or a 640x480 map window at the hero.

Two read-only hooks in `pkg/ui`, shared with the first game: `DetailedControlAt` names the detail page's control under a point through the same name function as `PreCreateControlAt`, whose names are unchanged; `(*App).HeadlessTooltipLines` returns the text of the visible hover box. Neither draws nor changes state.

## Proof

On `rom2-ru` and `rom2-en` the witness passes headless with 225 steps and 225 frames per edition, in about 17 s per root. Steps per stage on both: menu 10, generator pre-create 26, generator detail 38, town square 6, town statue 2, tavern 34, shop 45, way out 2, world map 5, mission dialogue 3, mission screen 54. The route ends in the first mission with one notice page paged and the hero selected.

Loss control and the release reruns are listed in the story's return.

## Open debt

- The statue's game menu rows are recorded as text only; the game menu exports no row rectangles for a hover walk.
- The mission screen is witnessed by parts, not one composed frame.
- The world map in route mode offers no hover target; a later town's world map with cards is not walked.
- The first town's inn offers no hire; the hire offer step is a recorded absence.
- Which menu, inn and shop button hovers show no text in the original is Unknown; this record shows none.
