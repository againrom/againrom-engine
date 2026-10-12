# The second game's global map

## Intent

In the second game the campaign moves between its towns and missions on the
global map instead of the destination list (DIV-2390, DIV-2966). Leaving a
town or winning a mission opens the map; the pointer targets the nearest
available record, a press sends the party along the road to it, and the
travel's end enters the record. The first game's world map is built by the
same builder and is unchanged. Base: public main `d5f37ccd`; knowledge pin
k267.

## Authority

Second-game evidence only. The map's opening, modes, input and end
`R2-ENGINE-428`; art and sound `R2-ENGINE-429`, `R2-ASSET-093`; markers and
party sprite `R2-ENGINE-430`; the nearest pick `R2-ENGINE-431`; travel, the
Cross and the skip `R2-ENGINE-432`; route images and the route search
`R2-ENGINE-433`; record text `R2-ENGINE-434`, `R2-ENGINE-435`; the party
point `R2-SESSION-177`; the town-leave arm and the 0x468 entry
`R2-ENGINE-272`; the victory arm `R2-ENGINE-422`; the record catalog
`R2-ENGINE-418`; `R2-ENGINE-073` as partially retracted. Choices where a value
is Unknown are rows DIV-3069 through DIV-3073 in `docs/divergences/rom2.md`;
DIV-2390 and DIV-2966 in `docs/divergences/rom2-town.md` are amended.

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
left, whose catalog point is the party point (DIV-3071), and route mode when
the record is town 1 (`R2-ENGINE-272`). The campaign screen is at the map
while no location is current. Its records are the available list in order,
each at its catalog point; a record with no point is offered disabled
(DIV-3073). Select mode draws GMap with PathMap routes, YourFlag at the
party, MissionFlag at the target and FlagOnMap at other records not at the
party, and the target's `globalmap.txt` name and description (DIV-3070).
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
the first record at the target, the victory without the map and the text by
record kind. `secondcampaign_test.go` and `seconddeparture_test.go` drive
the campaign through the map with fixture points.

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
  mission 21 has no public point and is disabled; the travel's end makes
  town 1 current on its square.

The ROM2 profile witnesses changed in one frame hash, the map at the gate;
mission 10's world and save hashes after the travel equal the destination
list's. The first game's world-map tests and the whole release set pass on
EN and RU unchanged.

### Blocked by the catalog

The ROM2 subject groups (as `pipeline/check-rom2-save-tests.sh` runs them)
fail 24 tests on each locale, every one because a record it travels to has
no public point: towns 2 and 3, mission 21 after mission 20, and constructed
positions at missions 102 and 110. They pass once the catalog's points
fill `worldmaps/rom2.json` (DIV-3073).

## Open debt

- DIV-3073: the catalog's 49 points (`R2-ENGINE-418`) are not in the public
  claims; only town 1, mission 10 and missions 93 through 96 have points.
  Every other record is disabled on the map until the table is published.
- DIV-3069: palette, transparency and the sound and music cues.
- DIV-3071: the party point after a victory and at a loaded map.
