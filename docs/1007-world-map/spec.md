# 1007 — world map — behavioural specification

## 1. Entry and ownership

Choosing GATES on the town square enters the world map. The map is the graphical presentation of
the existing `roomGates` state under `ui.ScreenTown`; it is not a second campaign or a second town.
The previous text mission list is absent.

The screen takes a snapshot of `Town.Available()` on each gates entry. It uses `FrontEnd.Campaign`
for `Payment`, `FrontEnd.MissionOpener` for activation, and `NextParty` through that opener. It does
not create or modify campaign, party, purse, shop, mission or simulation state.

The separate TOWN scroll and Escape return to the town square. Re-entering the gates rebuilds the
mission scroll set from current live town state and resets hover, selection and route animation.

## 2. Global-map registry

The adapter reads `scenario/globalmap.reg`. `[General] ObjectCount` creates an indexed object slice.
`MapObject1` through `MapObject<ObjectCount>` are read without compacting invalid rows.

A valid object has `MapPoint=[x,y]` and `MapRect=[x,y,width,height]`. Width and height must be
positive. Addition must fit the platform integer. The rectangle becomes the half-open rectangle
`(x,y)-(x+width,y+height)` once. An optional `Picture` string is retained.

Every integer `[MissionObjects] Mission<n>` maps mission `n` to the stated 1-based object number.
The adapter converts it to a zero-based slice index once. Out-of-range and invalid objects disable
only the affected mission scroll. Multiple missions may map to one object.

`PictureOffset` is not read. No shipped registry at the pin produces it, and its effect is Unknown.

## 3. Asset and text manifest

The lazy manifest attempts each fixed node once per `FrontEnd`:

- `main/graphics/global.map/gmap.bmp`;
- `graphics/global.map/pathmap.bmp`, `ballmap.bmp`, `hero.bmp`;
- `graphics/global.map/flag1/sprites.16a`, `flag/sprites.16a`, `cross/sprites.16a`;
- `graphics/global.map/scroll01.bmp`, `scroll02.bmp`, `scroll03.bmp`;
- `graphics/global.map/scrollp1.bmp`, `scrollp2.bmp`, `scrollp3.bmp`.

Missing background art leaves a filled 640×480 frame. Missing route, marker or scroll art leaves the
corresponding program-drawn fallback. Failed optional reads are cached as failures and are not
retried on later frames or visits. The adapter converts every absent optional image pointer to a nil
`image.Image` before it crosses into the UI view. The compositor independently treats a nil pointer
held inside any nil-able `image.Image` implementation as absent, so another screen implementation
cannot bypass the fallback by producing a typed nil.

For an object's non-empty `Picture` other than `nothing`, the adapter attempts
`main/graphics/global.map/<Picture>.256`. Frame zero must have a palette and a positive extent. The
cache key is the mission number. Every failure uses the available-location fallback and keeps a
valid mission actionable.

Mission `n` reads `main/text/battle/m<n>/title.txt` and `briefmap.txt`. Bytes stay in the install's
code page and are drawn by the current install font. A missing or empty title becomes `Mission n`.
A missing or empty briefing becomes `Briefing unavailable`. Positive `Payment` is shown as
`Reward <value>`; zero or absent `Payment` produces no reward line.

## 4. Scrolls, markers and route

The TOWN card is index -1. Mission cards follow live campaign order. Seven mission cards form one
page in a four-column, two-row geometry table with 8-pixel outer margins, 6-pixel gaps, 150×74 cards
and the shipped normal/pressed scroll art when available. Keyboard or wheel selection moves the
visible page with the selection; the mission collection has no shipped-count bound. Every card
contains its title, one briefing line and its positive reward. The program-drawn parchment and
border keep every card visible without art.

`MapPoint` is the marker and route destination. `MapRect` is the map hit region. A valid optional
mission picture is drawn at the marker. Otherwise the available flag is drawn. The selected mission
also draws the selected flag. Card controls take hit-test priority over map regions. Overlapping map
regions resolve by ascending `MapObject` index and then campaign order.

The route starts at `MapObject1.MapPoint`. In `PathMap.bmp`, palette indices 1 and 2 are passable.
The start and destination snap to their nearest passable pixels. An eight-neighbour breadth-first
search produces the shortest pixel path. The presentation retains every tenth path point and both
endpoints. One retained point becomes visible per application update. `BallMap.bmp` stamps the
visible prefix and `hero.bmp` marks its current end. A missing, disconnected or unusable mask uses a
sampled straight line so selection remains visible.

The start object, mask indices, neighbourhood, sampling interval and one-stamp cadence are authored
under the owner's route directive. They are not stated as ROM1 facts.

## 5. Input

Pointer coordinates use the existing native-frame placement inverse. A release on TOWN returns to
the square. A first release on a mission card selects it and rebuilds the route. A second release on
the selected card opens the mission. A map-region release selects or cycles the missions sharing
that object; map-region releases do not activate a mission.

Up, wheel-up, Down and wheel-down select the previous or next enabled mission with wraparound. Enter
selects the first enabled mission when none is selected, then opens an already selected mission.
Invalid mission cards remain visible and state their reason but do not enter an opener.

Escape uses the town room's existing Back transition. It returns from the world map to the square
without advancing a mission tick. The in-game menu remains reached from the square on the next
Escape, unchanged.

## 6. Campaign, failures and persistence

Mission activation returns the existing `MissionOpener` in a `TownAction`. `flow.applyTownAction`
uses the existing `enter` path. A loader failure leaves the world map open, retains selection and
shows the loader's error. A successful opener uses the carried party, campaign purse, mission map,
script runtime, cadence and mission viewer already owned by that path.

Hover, selection, route and visible-prefix state are presentation state. No new snapshot or byte
form field exists. Entering, selecting and returning produces an equal town snapshot and equal
encoded save bytes. Loading a town reconstructs the map from the restored `Town.Available()` state
on the next gates entry.

## 7. Acceptance criteria

**AC-1** A non-shipped `ObjectCount` determines the object slice. Invalid rows retain later indices.

**AC-2** Rectangles convert width and height once. Invalid and overflowing rectangles disable only
their own mappings.

**AC-3** Dynamic mission mappings convert 1-based object numbers once and preserve duplicates.

**AC-4** The manifest and optional mission art are attempted once. Missing optional files and typed
nil image implementations take the drawn fallbacks and keep the town control and valid offers
usable.

**AC-5** Every live offer has one scroll containing title, briefing and positive `Payment`, or an
explicit fallback or disabled reason.

**AC-6** Card controls precede map regions. Map regions resolve by object order. First card release
selects; second card release or Enter activates through the existing mission opener.

**AC-7** Selection draws a flag and constructs a route from the authored start through the authored
mask algorithm. The route prefix advances by one stamp per update.

**AC-8** TOWN and Escape return to the square. Re-entry reads the current live offer set.

**AC-9** World-map entry, selection and return change no campaign, party, equipment, shop or save
bytes.

**AC-10** The EN and RU production witnesses accept a building offer, show its scroll, animate a
route, return, re-enter and open the mission.

**AC-11** The two-root sweep consumes all 28 mappings and all 28 title/briefing pairs through the
production adapter, including optional-picture and fallback markers.

**AC-12** No Sim Core, AI, inventory, equipment, digest or serialized format changes.
