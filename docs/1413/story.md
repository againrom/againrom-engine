# The second game's global map

## Intent

In the second game the campaign moves between its towns and missions on the
global map instead of the destination list (DIV-2390, DIV-2966). Leaving a
town or winning a mission opens the map; the pointer targets the nearest
available record, a press sends the party along the road to it, and the
travel's end enters the record. The first game's world map is built by the
same builder and is unchanged. Base: public main `d5f37ccd`, merged with
main `b2af390a`; knowledge pin k326.

## Authority

Second-game evidence only. The map's opening, modes, input and end
`R2-ENGINE-428`; art and sound `R2-ENGINE-429`, `R2-ASSET-093`; markers and
party sprite `R2-ENGINE-430`; the nearest pick `R2-ENGINE-431`; travel, the
Cross and the skip `R2-ENGINE-432`; route images and the route search
`R2-ENGINE-433`; record text `R2-ENGINE-434`, `R2-ENGINE-435` and the
substring lookup `R2-ENGINE-462`; sound ordering `R2-ENGINE-463`; the idle
paint gate `R2-ENGINE-464`; draw order and sprite shading `R2-ENGINE-465`;
the party point stores `R2-SESSION-185` (amending `R2-SESSION-177`);
the town-leave arm and the 0x468 entry
`R2-ENGINE-272`; the victory arm `R2-ENGINE-422`; the record catalog
`R2-ENGINE-418` and its table of the 49 records' points; `R2-ENGINE-073` as
partially retracted. Remaining presentation debt and choices where a value
is Unknown are rows DIV-3069 through DIV-3072 in
`docs/divergences/rom2-global-map.md`; DIV-3073, the
catalog points, is closed in `docs/DIVERGENCES-CLOSED.md`. DIV-2390 and
DIV-2966 in `docs/divergences/rom2-town.md` are amended.

## As built

### One builder

The world map is one kind with one builder. Each edition names a world-map
description (`base.Edition.WorldMap`): `pkg/game/worldmaps/rom1.json` and
`rom2.json`, decoded and validated by `pkg/game/worldmapdescription.go`. A
description holds the art, where records and points come from (`registry` or
`catalog`), the modes (background, route image, party sprite and placement,
whether markers and text show, whether the mode opens travelling to its first
record), the pick rule (`scroll` or `nearest`), preview, skip at start, the
Cross and its hold, the task sheet, scroll cards, marker rules and the text
block. `worldMapState` (`pkg/game/worldmapstate.go`) is the one state machine
and `worldmapinput.go` the one input path, both following the description;
`ui.ComposeWorldMap` the one compositor, with sprite placements and a text
block as view data. A campaign supplies only its records, its view input and
what choosing, leaving and arriving do (`worldMapCampaign`).

The first game's description reproduces its map: registry records, scroll
pick, card art, task flag and record art. Its gates art moved from
`towns/rom1.json` into `worldmaps/rom1.json`; its state, input rules and
picture are unchanged.

### The second game's map

`secondCampaign.openMap` records an opening when a town is left
(`leaveTown`) or a mission is won (`completeBank`): the record the party
left, whose catalog point is the party point (`R2-SESSION-185`), and route mode when
the record is town 1 (`R2-ENGINE-272`). The campaign screen is at the map
while no location is current. Its records are the available list in order,
each at its catalog point: the description's catalog holds the 49 points of
`R2-ENGINE-418`. A record outside the catalog would be offered disabled with
a notice; no claimed writer of the available list produces one. Select mode draws GMap with PathMap routes, YourFlag at the
party, MissionFlag at the target and FlagOnMap at other records not at the
party, and the target's `globalmap.txt` name and description (DIV-3070).
The name and description stay drawn after the travel starts; the record
markers alone disappear (`R2-ENGINE-434`, `R2-ENGINE-465`).
Route mode draws Umoir with UmoirPath routes and Flag, no markers, and opens
travelling to the first record. A press starts the travel or skips it; a key
skips. Each tick reveals 8 route points and advances the Cross; the travel
ends when both are past their ends (DIV-3072). The end enters the first
available record at the target: a town becomes current on its square, a
mission opens.

A victory with the stage below the credits and no town available enters the
first available mission without the map (`R2-ENGINE-422`).

### Session and SAV

The opening is not persisted; the map is not a save point. A document loaded
with no current location opens the map in select mode with the party at the
first available record (DIV-3071). SAV content is unchanged.

## Proof

Fixture tests in `pkg/game/secondworldmap_test.go` hold the rules with
synthetic sheets: the nearest pick and its tie, markers by mode and phase,
the reveal and Cross end, the skip, route mode's opening travel, arrival at
the first record at the target, a disabled record never targeted,
the victory without the map and the text by record kind.
`secondcampaign_test.go` and `seconddeparture_test.go` drive the campaign
through the map with fixture points.

Release tests on `rom2-en` and `rom2-ru`
(`pkg/game/secondworldmap_release_test.go`):

- `TestReleaseSecondCampaignWorldMapRouteMode`: leaving town 1 opens Umoir
  with Flag at town 1's point, one record (mission 10 at (302,158)), no
  marker or text, a route from (215,234) to (302,158); the Flag's pixels
  change only inside its placement (loss control: a transparent sheet of
  the same size); the travel ends in mission 10.
- `TestReleaseSecondCampaignWorldMapSkip`: a press during the travel enters
  on the next paint.
- `TestReleaseSecondCampaignWorldMapSelectMode`: two targets (town 1,
  mission 93) move MissionFlag and FlagOnMap, preview their own routes and
  draw different frames; mission 93's `globalmap.txt` name and description
  draw only inside the text block; town 1 has no key and draws no text;
  mission 21 stands at its catalog point (423,367) with FlagOnMap; the
  travel's end makes town 1 current on its square.
- `TestReleaseSecondCampaignWorldMapTravelKeepsText`: a new campaign
  completes missions 10 and 20 through the installed headless victory
  instruments, then targets town 2 at (370,305). Its name and description
  survive the click and ticks 1 and 2 while record markers stay hidden.
  Compositions show text only inside its block: EN 4333 pixels, RU 4141.
  A temporary production overlay restoring the old travel predicate fails
  this test by name on both roots; the overlay is removed after the run.

The second game's campaign release tests reach towns 2 and 3 and missions
21, 102 and 110 through the map at their catalog points. The ROM2 subject
manifest declares 105 subjects, including the travel-text witness. Its
groups run as `pipeline/check-rom2-save-tests.sh` runs them on `rom2-en`
and `rom2-ru`; per-head gate receipts are recorded in the lane handoff.

The ROM2 profile witnesses changed in one frame hash, the map at the gate;
mission 10's world and save hashes after the travel equal the destination
list's. The first game's world-map, gate, map-point and SAV release tests
and its profile witness pass on EN and RU after the merge with main; its
witness files are main's.

## Open debt

- DIV-3069: omitted map sound requests, frame-count cycles, remaining
  device output and cursor identity. The mode-4 shade table and Point2-first
  ordering are settled by `R2-ENGINE-465` and `R2-ENGINE-463`.
- DIV-3070: font identities, ink and wrapping. First-substring lookup is
  settled by `R2-ENGINE-462`; select-mode text remains during travel.
- DIV-3071: point persistence in SAV and LOAD, other writers and off-node
  routing. The town-leave and victory stores are settled by `R2-SESSION-185`.
- DIV-3072: press/key dispatch and paint/counter ordering; unread
  route-count helpers and additional paint sources. `R2-ENGINE-464` settles
  the 100-ms idle gate without catch-up, not every paint source or duration.
- `R2-ENGINE-418` leaves Unknown whether code outside the rebuild appends
  to the catalog list and the role of the three type-1 records built
  outside it; the engine uses the 49 records of the rebuild.
