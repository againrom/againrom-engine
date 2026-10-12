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

To be completed with the gates.

## Open debt

- DIV-3073: the catalog's 49 points (`R2-ENGINE-418`) are not in the public
  claims; only town 1, mission 10 and missions 93 through 96 have points.
  Every other record is disabled on the map until the table is published.
- DIV-3069: palette, transparency and the sound and music cues.
- DIV-3071: the party point after a victory and at a loaded map.
