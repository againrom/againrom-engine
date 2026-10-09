# Room pages built by the town composer

## Intent

The ROM1 tavern, shop and school centres are built by the town composer from
the ROM1 town description, and the three rooms' tips are read from it.
Nothing the player sees, hears or saves changes. A room's centre is a scene:
art, draw sources, sound slots, clocks, actors running named page programs,
the steps one paint runs and the layers each paint group draws. A room page
is one builder, `town.Page`; each room is data for it.

Base: `af6c6a09` (game 0.105.0), merged with `dfb84f0f`. Knowledge pin: k208.

## Authority

- Owner: one town composer; one builder per kind; the composer holds no
  game's facts, so ROM1 values sit in `pkg/game/towns/rom1.json` beside the
  claim IDs and divergence rows the replaced code cited.
- ROM1 values: TOWN-015, TOWN-021, TOWN-146, TOWN-147, TOWN-149, TOWN-154,
  TOWN-188, TOWN-379..381, TOWN-407..413, TOWN-427..433, TOWN-500,
  SHOP-ANIMATION-081..084, SHOP-MERCHANT-046, SHOP-SHELF-047, SHOP-TIP-045,
  and the rows DIV-121, DIV-132, DIV-524, DIV-525, DIV-814, DIV-842..844,
  DIV-847..856, DIV-872 and DIV-873. No new ROM1 fact is asserted.
  DIV-2700..2705 are unused.

## As built

- `pkg/town` gains the room scene (`SceneSpec`) and its runtime `Page`:
  `NewPage`, `Enter`, `Reset`, `SetActive`, `Advance`, `Event`, `Paint`,
  over a `PageHost` that answers art, time, bounded draws, reseeds, sounds
  and named values. Page programs: `loop`, `alternating`, `selector`,
  `priority`, `bounce`, `cycle`, `target` and `training`. Clocks have a
  period or none, a strict or inclusive compare, an entry offset, a
  back-stamp rule and a resume rebase (`every`, `after-first`). Sound slots
  are tracked (one voice) or untracked. Art series load whole, member by
  member (`keep-missing`), with an exact size or a size envelope.
- `validatescene.go` refuses a scene that names an unknown program, phase,
  action, actor, art, slot, draw source or clock, a step group on a clock
  with no period, a zero period, a layer with both or neither of art and
  actor, a training actor whose column is no target, an endpoint or reverse
  index outside its frames. The vocabulary check covers the values and
  events a scene names; an event an actor raises itself is the scene's.
- Tavern centre (`tavern` scene): candle and cauldron `loop`, tender
  `alternating` (parity of the drawn wait chooses the drink or the pour),
  painted in group `centre`.
- Shop centre (`shop` scene): the four racks are one `selector` (selected by
  the value `shop-chosen`, event `rack`, raising `rack-changed`), the merchant
  a `priority` (idle after a drawn wait; `merchant-yes`, `merchant-no` and
  `rack-changed` raise yes or no), painted in group `interior`.
- School centre (`school` scene): the training column a `target` (endpoint by
  the value `member-class`), the training movies one `training` actor with
  two variant sides (event `class-change`), the diamond a `bounce` (event
  `train`), the idle shine a `cycle` whose count maps to a slot per class,
  painted in groups `movies`, `column` and `diamond`.
- Room tips: each room names its tip text and panel rectangle.
- `pkg/game/roompage.go` is the ROM1 adapter: `roomPage` builds each page
  once; `roomPageHost` binds art, the animation clock, the draw services
  (`tender`, `idle`, `training`), the room sound player, the values
  `shop-chosen` and `member-class`, and the events `rack`, `merchant-yes`,
  `merchant-no`, `train` and `class-change`. The hooks keep their names and
  order: `tavern-interior`, `tavern-leave`, `shop-interior`, `shop-reset`,
  `school-reset` and `school-leave` now enter, pause or reset the pages; the
  ui lifecycle seams (`TavernInteriorActive`, `ShopInteriorActive`,
  `SchoolTrainingActive`, `AdvanceShopInteriorAnimation`,
  `AdvanceTownSurfaceAnimation`) call the pages.
- `pkg/ui`: a room view carries a `RoomScene` and paints its groups at the
  places the replaced painter drew; the ui art structs hold the scene's
  pictures as `Scene` maps.
- Deleted: `pkg/game/taverninterior.go`, `taverninteriorart.go`,
  `shopinterior.go`, `schooldiamond.go`, `schoolcolumn.go`,
  `schooltraining.go`, the ui types `TavernInteriorArt`,
  `TavernInteriorFrame`, `ShopInteriorFrame`, `SchoolTrainingArt` and
  `SchoolTrainingFrame`, the per-room art fields they fed, the shop and
  school family loaders and the game's room tip path constants. There is no
  fallback path.

## Proof

- `TestReleaseTownRoomTraceIsUnchanged` (`pkg/game`), committed before the
  move on unchanged code: 2620 ticks per root of frame hash, sound and hook
  records over a scripted path through the tavern, shop and school. After
  each move (tavern, shop, school) it is identical on EN and RU; the recorded
  files are unchanged.
- `pkg/town` `TestComposerHoldsNoGameFact` passes unchanged.
- `pkg/game` `TestTownDescriptionEveryValueIsCited`: 282 objects, 494 cites,
  each a pinned claim, a ledger row or "owner".
- `pkg/town` `page_test.go`: every page program, the lifecycle and fourteen
  scene refusals on synthetic scenes over generated art, without an install.
- `pkg/game` `TestRoomTipsAreTheDescribedTextsAndRectangles` pins the
  described tips to the ui's tip rectangles.
- The tavern, shop, school, tip, town, city and shared release tests pass
  on EN and RU; the full gate results are in the lane return.

## Accepted differences, not player-visible

- A page whose room art did not load does not advance; the replaced shop
  and school code advanced their counters without art. Nothing is drawn
  either way.
- Leaving the school for the square resets the whole school page, including
  the diamond and the column; the replaced code kept them until the next
  school entry, which reset them first.
- Two tests changed what they observe for these reasons: the diamond
  leave test reads the state after leaving, and the shine view test loads
  school art.

## Open debt

- Page widgets stay page-kind code: the tavern roster, panes and buttons,
  the shop shelves, table, strips, buttons and plaques, the school skill
  cells, masks and buttons, and the static art the ui structs load.
- Room music: the ui music controller keeps its scene table (tavern, shop,
  school with the shown class first). Moving it needs a list-valued music
  request.
- The world map is not a scene. Its flag and cross counters gate arrival,
  so moving them moves a travel rule; it stays page-kind code.
- `ui.TavernTipRect`, `ui.SchoolTipRect` and `ui.ShopTipRect` remain for
  ui tests and `cmd/tippanelcheck`; a test pins them to the description.
- The trace records step hooks by name and room behaviour by its effects
  (frames, sounds); page events are witnessed through those effects.
