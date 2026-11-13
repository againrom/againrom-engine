# The mod window shows the campaign as a graph, read-only

## Intent

The map editor's mod window (W) gains a third page, the campaign graph. It shows the start, every mission, the
town of every chapter and every hand-out of a town's tavern, shop and school, with an arrow for what follows what.
With no mod open the graph is the install's campaign; with a mod open it is the campaign with that mod's overrides,
and the missions and arrows the mod changes carry a mark that the page states in words. A click on a mission opens
its map by the route the editor already has. The page shows and changes nothing. Editing the campaign from the
graph is a later story.

## Authority

Owner direction in the story brief: a read-only graph page in W; one builder per kind; the graph reads the one
`Campaign` value with the mod's overrides as the game's launch builds it and parses neither `scenario.reg` nor
`data/missions.toml` itself; a mission's map opens by the editor's existing route; every string through the word
book in EN and RU; positions from the data, none placed by hand. What a town's buildings hand out is `REG-SCN-062`
and `REG-SCN-064`: `InnNPC[i]` pairs with `InnMission[i]`, a zero `InnMission` is a visitor with nothing to give,
and `ShopMission` and `TCMission` are the shop's and the school's lists. The editor and mods are Againrom
surfaces; no divergence row changes.

## As-built behaviour

The page. W's pages are the list `modPages` (`pkg/ui/mapeditorworkspace.go`): the mod's fields and missions, the
overview when the host gives one, the campaign graph when the host draws one. Tabs, the window's key and a click
on a tab all go through `turnModPage` and `showModPage`. W's key (`W`) with W open turns to the next page whenever
W has more than one. A first-game install has a campaign, so W shows two tabs with no mod open, Missions and
Campaign graph, and three with a mod open; before, W had tabs only with a mod open.

The canvas. The popup builder (`popupSpec`, `layoutPopup`, `drawPopup`, `popupLayout.hit` in
`pkg/ui/editorpopup.go`) gained one element, `popupSpec.Canvas` (`pkg/ui/editorcanvas.go`): a drawing as data,
boxes of text with marks and lines with arrow heads on a sheet that may be larger than its view. The builder lays
the view under the rows, clips the sheet to it, draws a scroll bar when the sheet is taller, and answers three
hit parts: a box, the ground, the bar. A box that takes a press is drawn as a push button; a box has a hover tip.
Each box's place in the window is a `controlLayout` row that the draw reads and the hit test asks, and `drawPopup`
itself pushes the boxes' buttons. Any window may set a canvas; W's graph page is its first user.

The graph's data. `campaignGraphOf` (`pkg/game/campaigngraph.go`) builds the nodes and edges from a `Campaign`
value and the mod's mission rows. With no mod the value is the install's campaign with the install's town book, as
the game's launch reads it. With a mod open it is the campaign of the mod's admitted fork, which `openModContext`
now keeps beside the fork's other data (`admittedFork`, `FrontEnd.SetModData`), with the fork's mission rows. A
mod the game refuses for its missions, and a mod whose fork was not built for another reason, each show the
install's graph and a line that says which. The graph of an open mod is the mod as its folder holds it; an
unsaved edit of a mission field shows after the mod is saved.

What follows a win. The graph states no route of its own. It asks the readers the game asks:

- `Campaign.winRoute(n, main)` (`pkg/game/campaign.go`) walks the win of mission `n` while the campaign stands on
  main mission `main`: the missions the sections lead through (`AutoAdvance`), the main mission the campaign then
  stands on, and the offer that follows (`NextMission`), which opens the town where the last win is set to
  (`TownAfter`). `withModMissions` fills `missionTowns` and `TownBoundary` from this walk, as it did from its own
  loop before.
- `Campaign.advances(completed, main)` says whether a win moves the town. `Town.advanceMain` asks it. The graph
  asks it for a mission a town hands out: a win that does not move the town returns to that town.
- `Campaign.holds(n)` says whether the campaign names a mission. `withModMissions` asks it to tell a mission a mod
  adds; the graph asks it of the install's campaign for the `new` mark.
- `mod.MissionData.AfterWin(n, kind)` is the guest mission a starting hero of that kind plays after a win. The
  graph asks it for each of the four starting heroes and draws the union.

Nodes and edges. The start leads to the first mission. A mission with `AutoGetMission` leads straight to the
mission it names. A mission whose win opens a town leads to that town, for each hero through the guest missions
the mod plays for that hero. Where the arrows out of one mission hold for different heroes, each carries the
heroes' word (`Man`, `Woman`, `Mage`, `Fighter` or one hero; the message editor's words). A town leads to each of
its hand-outs, in the scenario's order: the tavern's visitors, the shop, the school. A hand-out leads to the
mission it gives; a tavern visitor with nothing to give is a node with no mission. A town that a win set to open
the town leads to (`TownAfter`) offers its main mission itself, and leads straight to it unless a building of the
town hands it out. A main mission that no building hands out stands straight after the mission before it, as the
game offers it. A mission nothing reaches stands alone under the towns.

Layout. Three columns: the start and the towns, the hand-outs, the missions. A node takes the next free row when
the walk reaches it, so the rows follow the campaign, and a hand-out's mission stands on the hand-out's row. Each
column is as wide as its longest text between two bounds. A line runs straight to the node on its right or under
it, down a trunk from a town to its hand-outs, and otherwise along a lane right of the columns into its target's
top. A line with heroes' words always takes a lane, leaves its mission at a level of its own and carries the
words there, between the mission column and the lanes. No coordinate is stated per mission; two builds of the
same campaign are equal.

Mod marks. A mission node carries a word in its own ink for each change: `map` (the mod replaces the mission's
map), `flow` (the mod states what follows the mission), `new` (the mod adds the mission), `party` (a guest party
of the mod plays it). An arrow the mod sets has the ink of `flow`: it leaves a mission whose successor the mod
states, or it ends at or leaves an added mission. Those arrows are drawn over the others, so one keeps its ink
where another shares its lane. With a mod open four legend lines above the graph state each mark in words.

Opening a map. A click on a mission the open mod holds the map of calls W's own row action,
`modAction("source", row)`, which is `SetWorkspaceField("package", "source", row)` and `selectWorkspaceMission`,
the route of the Open button of W's mission list. A click on any other mission calls `MapEditor.Open` with the
key the map list uses for the install's map. Both routes ask the editor's Save / Discard / Cancel question when
there are unsaved changes. A mission the install lists no map of opens nothing and its tip says so.

Moving the graph. The arrows, PgUp, PgDn, Home, End, the wheel and a drag of the ground move the sheet under the
view; the scroll bar shows the place. The modes table holds three new rows: `workspace/mod-graph`,
`workspace/mod-graph-pan` and `workspace/mod-graph-node`.

The second game. Its towns hand out missions from the scenario bank, which the `Campaign` value's chapters do not
hold. On a second-game install the page shows one line that says the graph is not read for this game.

Unchanged. No mod file, manifest key, SAV byte or simulation state changes. Showing the page, moving the graph
and leaving the page make no draft change and no undo step.

## Proof

- `pkg/game/campaigngraph_test.go`, on a synthetic campaign and a synthetic mod in `t.TempDir()`: the exact node
  and edge sets of a campaign with a flow edge, two side missions, a tavern visitor with nothing and each hand-out
  kind; each mod mark (replaced map, stated flow, added mission, guest party) and the marked arrows; the layout
  equal over nine builds with no two nodes in one cell; a click on a mission opens the install's map, or the mod's
  map when the mod holds it, and a mission with no map opens nothing; a refused mod shows the install's graph;
  `TestEditorGraphPageChangesNothing` shows the page, presses every moving key, draws, turns on and closes, and
  finds the mod folder, the profile folder, the document, the map bytes, the mod's draft and the undo history as
  they were.
- `pkg/game/campaigngraphroute_test.go`: three mods on a six-chapter campaign (a side mission sent to a later
  main mission; a main mission whose lead the mod ends; two guest missions that between them play for every hero)
  and a guest mission for mages only, each mission's arrows and heroes' words stated; and over ten fixture
  campaigns and mods, every mission node's arrows equal what `AutoAdvance`, `missionTowns`, `NextMission`,
  `TownAfter` and `AfterWin` give for that mission, with the same heroes.
- `pkg/game/campaigngraphpage_test.go`: the page in EN and RU, of the install and of a mod, has its caption, its
  legend and every box drawn inside the window.
- `pkg/ui/mapeditorgraph_test.go`: the canvas is laid, hit and painted; W's key and tabs turn to the graph; the
  legend states each mark in EN and RU; keys, wheel, drag and the bar move the sheet inside its bounds; a node
  click opens by the existing routes and asks about unsaved changes; the layout comes from the data alone.
- The modes table (`pkg/ui/testdata/mapeditor_modes.golden`): three rows added, none moved.
- Release witness `TestReleaseCampaignGraphEqualsTheHandoutCensus` (`pkg/game`), for EN and RU: the graph of the
  first-game install has 13 towns whose buildings hand out 26 distinct missions, each by one building, tavern 19,
  shop 4, school 3, with three tavern visitors that give nothing, in chapters 50, 110 and 130 (`REG-SCN-064`).
- Release witnesses that draw W on an install, for the release: `TestReleaseMapEditorOwnerFrames` and
  `TestReleaseEditorInteractionFrames`. W now has tabs with no mod open, so their frames of W may move. Neither
  ran in this lane.
- Owner frames from synthetic content, drawn headless with the installed fonts: the page in EN and RU, and with a
  mod's marks in EN and RU. They stay in the review notes; no frame is committed.

## Open debt

- The second game has no graph: nothing reads its town catalog's scenario-bank offers as hand-outs.
- The graph of an open mod does not follow an unsaved edit of a mission field; it follows the saved mod.
- A refused mod shows the install's graph, not the part of its campaign the game would accept.
- The game's live win route (`Town.complete`, `CampaignSession.recordWin`, `campaignProgress`) still reads
  `AutoAdvance`, `TownAfter` and `NextMission` in its own order; only `withModMissions` and `Town.advanceMain`
  share `winRoute` and `advances` with the graph. A side mission that a main mission's section leads to is drawn
  as `missionTowns` gives it, the next chapter's town.
- A town reached before a later win set to open the town leads to it gets its own mission's arrow late, on a row
  under the town's hand-outs.
- Four labelled arrows out of one mission leave it closer than a text line apart, so their words touch; a label
  covers the piece of another line under it.
- A mission node names the building that hands the mission out, not the gates of a mod's tavern row (class, sex,
  bank slots).
- Editing from the graph (adding a mission, changing what follows) is not built. It needs a selection on the
  canvas, an edit route from a node to the mod's mission rows, and a graph rebuilt from the draft.
