# 1336 — FrontEnd original restore

## Result

The restore of an original SAV no longer copies the FrontEnd. The file decodes
from plain values, the between-mission game is built on a bare
`CampaignSession`, and the mission half is planned and prepared from plain
values. The won-mission routing, the leave-mission campaign writes and the
viewer's art resolution are named operations on the components that own their
state. Player-visible behaviour, world hashes, SAV bytes and scenario hashes
are unchanged; this is a refactor.

## Intent

Owner direction: FrontEnd's operations move to the components that own their
state. Coordination stays where a transition spans systems, and the top level
calls named operations instead of editing each system's internals. Done when
campaign logic is testable without building a whole FrontEnd, and a new audio
service needs no change to mission-transition logic. Passing the whole
`*FrontEnd` into a component does not meet it.

This story continues 1332 and 1333. Their open debt named `restoreOriginal`
(`draft := *f`, `markerFront := *f`), the coordinators that edit systems'
internals and the ports that are method values of `*FrontEnd`.

## As built

### Original SAV restore (`pkg/game/originalrestore.go`, `originalsave.go`)

| operation | replaces |
|---|---|
| `decodeOriginalSource(saved, mods, campaign)` | the decode half of `restoreOriginal`: repair, mod mark, every optional record, the campaign decode and, for a mission save, the actor graph, pools, profiles, holdings, books, dead, buildings and ground |
| `(*InstallResources).originalInstall()` | the install reads of the restore: table, bodies, start weapon, archives, text selector |
| `(*CampaignSession).restoreOriginalTown(src, in, campaign)` | the town branch, which ran on `draft := *f` |
| `(*originalSource).planMission(in, units, selected)` | the mission candidate, the restored party and the purse |
| `(*originalSource).prepareMission(in, plan)` | the `prepare` closure, byte for byte over locals unpacked from the source |
| `(*CampaignSession).reportTown`, `originalSource.offered`, `worldMapReturn` | the stderr line and two reads of the candidate |

`FrontEnd.restoreOriginal` is the coordinator: decode, the world-map marker
population, then `restoreOriginalTown` or `restoreOriginalMission`. The town
coordinator hands a zero `CampaignSession` to the operation, then clones the
unit bundle, installs the candidate and reports. The mission coordinator plans,
builds the opener with `missionOpenerMode` and returns the commit.

`restoreHiredMercenaries` is a method of `CampaignSession` over the table.
`buildMercenarySquad` and `buildSiegeSquad` are functions of the table and the
town; the town screen's methods call them. They read only those two values.

### World-map marker population (`worldmap.go`)

`loadWorldMapAssets(archives)` is the read; `worldMapAssets()` is the
first-use cache around it. `coldWorldMapData(cache, archives)` takes the cache
by value, as the `markerFront := *f` copy did: a cache that was not tried is
read for that call and is not stored. `worldMapMarkerMissions(town, data)` is
the filter, shared with `frontWorldMapFilteredMarkers`.

### Ports and operations

- `viewerArt` resolves through `viewerArtSource{in, pr}` over
  `*InstallResources` and `*Presentation`. The first-use art caches
  (`gameMenuArt`, `characterPanes`, `characterPaneCorners`,
  `characterPaneFiller`) are methods of `Presentation` taking the install;
  `tipFont` is a method of `InstallResources`. The FrontEnd methods of the same
  names remain as one-line calls for their other callers.
- `CampaignSession.routeAfterWin(campaign, n, successor)` returns where a won
  mission leads (`winStay`, `winEnding`, `winMission`, `winTown`, `winList`) and
  forgets the live map at the end of the campaign. `continuity` switches on it.
- `CampaignSession.leaveLive(in, n, ms)` is the campaign half of
  `leaveMission`: the resumed-mission carry home, the return-city retention and
  the quick-spell restore.

## Differences from the base

Statement order, error strings, nil guards and hashed state are unchanged,
except where listed.

- The detached copy's town screen (built by `draft.TownScreen()`, which loads
  the square's tip text) is gone. Nothing read it; the squad builder read only
  the table and the town. The detached audio scope stays: the town coordinator
  creates it before the operation and destroys it on return, as the base did.
- The text selector and the install reads are taken when the restore starts,
  not when `prepare` runs. Nothing between them writes either.

## Proof

- `originalrestore_test.go` decodes a synthetic original SAV and builds its
  town on a bare `CampaignSession` with an empty install value; refuses bytes
  that are not a SAV; refuses an oversized hired pool on a session alone; runs
  `coldWorldMapData`, `worldMapMarkerMissions`, `viewerArtSource.resolve`,
  `routeAfterWin` and `leaveLive` with no FrontEnd.
- Unchanged EN and RU release tests, the milestone-2 acceptance family over
  the original save corpus, and the scenario hashes carry the no-behaviour
  claim. Gate verdicts are in the lane return.

## Measures

Measured with `internal/archtest/cmd/composition`.

| measure | before | after |
|---|---|---|
| component fields | 82 | 82 |
| coordinating functions (three or more components) | 16 | 15 |

`restoreOriginal` leaves the coordinating set.

## Open debt

- `openCityBase` and `continuity` remain method values of the front end. The
  first projects the whole game state as a SAVE would write it; the second
  opens screens and the next mission's opener. Neither has a narrower service
  to name without a behaviour change.
- `loadMap` (non-mission rows) keeps its own art setters and edits the live
  triple through `liveDriver`. Its set of setters differs from `viewerArt.apply`,
  so moving it onto `viewerArt` would change what the picker's map shows.
- `restoreOriginalMission` keeps the opener and the install commit on FrontEnd:
  `missionOpenerMode` builds the ports from the whole front end.
- `townScreen` holds a `*FrontEnd`; screen switching is a coordinator call, not
  a port.
