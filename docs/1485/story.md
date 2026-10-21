# World map travel from the town catalog

## Intent

The first game's world map reads its towns from the town catalog. The party travels between open
towns, arrives in each with that town's own state, and a mission win returns it to the town it
left. Which towns are open and which missions each offers are entry data. The SAV keeps the
first game's town ID. The shipped games list the towns they listed before, so no player sees a
change: the first game's one town stands at MapObject1, offers what it offered, and writes the
SAV bytes it wrote.

## Authority

Owner direction (`pipeline/BACKLOG.md` T2): the world map reads towns from the catalog; the party
travels between open towns; a win returns to the town the party left; open towns and offered
missions are data; the first game's town ID is kept in the SAV. Owner rules: one builder per
kind (the catalog, `newTownState`, `partyPlace`); the second game's screens are its own and play
exactly as now; SAV is the only save format, written by one producer. Seat ruling: a town's map
point resolves against one existing `globalmap.reg` read, and no third world-map entry-assembly
route is added. ROM1 claims: `SAV-CAMPPOS-072` (the campaign record keeps the map position as a
relation to the first MapPoint or the selected mission's point; no field names a town).

## As built

### Entry fields

`pkg/town/catalog.go` gains three optional fields. A shipped entry gives none of them, so each
shipped catalog decodes as before.

| Field | Type | Meaning | Default |
|---|---|---|---|
| `name` | string | The town's title on its travel record and on the home scroll while it is the party's town. | The install's town word. |
| `opens-after` | int | The mission whose win opens the town to travel. Negative is an error. | 0: open from the start. |
| `missions` | list of int | The missions the town's buildings offer. Each must be positive; a mission listed by two towns is an error. | Empty: every mission no other entry lists. |

`pkg/game/towncatalog.go` adds the rules that need the descriptions (a `townBook` is one decoded
catalog with its descriptions):

- The offer source and hook set must pair (`chapter-keys` with `chapters`, `scenario-bank` with
  `destinations`), and the description must hold every room its hook set enters (`chapters`:
  tavern, shop, school, gates). Otherwise the decode fails with "has no usable description".
- A `scenario-bank` entry refuses `opens-after` and `missions`: the second game's bank opens its
  towns and holds their missions.
- Chapter towns start with town 1, open from the start, with offers from the chapter keys.

The test fixture (`travelCatalog`, `pkg/game/towntravel_test.go`) lists West (town 1, MapObject1,
no missions, so it offers every mission East does not) and East (town 2, MapObject3, description
`rom1-east`, `missions: [31]`). `TestTownOpensAfterItsMission` adds `opens-after: 30` to East.

### Town states

`Campaign.Towns` holds the campaign's catalog (`FrontEnd` sets it from the profile's catalog).
`NewTown` builds one `townState` per chapter town through `newTownState`: town 1's in `Town`'s
embedded state, the others in `Town.others`, keyed by town ID. `Town.travelTo(id)` swaps the
party's state out and town `id`'s in. With more than one town, `Offers`, `TavernRoster` and
`Take` show and accept only the missions the party's town offers (`offeredHere`). With one town
nothing filters.

### Mercenaries across towns

Seat decision on the two-town catalog; with one town each rule is the base rule.

- Unlocks are campaign-wide. `Town.mercEnabled` is one set of flags, not part of `townState`: a
  won mission's `EnableMercenary` enables the type in every chapter town, open now or opening
  later, and a cold LOAD derives it as it derived town 1's (the original record's
  `PermanentMercenaries`, the session's flags, or the won missions). Pools and capacities stay
  per town.
- A hire belongs to its town. While one town holds a type hired, no other town lists or hires it
  (`mercenaryHiredElsewhere`; `tavernMercenaries`, `mercenaryHire`), so the party carries at most
  one squad of a type. Returning the squad before a mission works in the hiring town.
- At every mission end, wherever the party is, every chapter town's tavern runs MERC-DEATH-006's
  merge (`mercenaryBoundary`): a hired type's pool becomes the live tally of the squad, which only
  that town can have hired; an unhired type grows by one up to its capacity; all hires end.

### Map, travel and return

- `townMapPoint(entry, assets)` is the one answer to "where is the town on the map": `object` reads
  the world-map assets' `globalMapData` (the first game's existing `globalmap.reg` read); `record`
  reads the map description's record. `worldMapHome` is the party's town's point; the
  `worldMapState` field `homeTown` names the town, and every former MapObject1 caller reads
  `home()`.
- `enterWorldMap` appends one record of kind `placeTown` for each open town other than the
  party's (`travelTowns`), titled by `name`, at the entry's point. The town gates open the map
  when a mission is announced or a town is open to travel.
- Clicking a town record travels the route as a mission record does. `arriveInTown` calls
  `travelTo`, sets the home to the new town, and stands the party on its square, composed from
  that town's description.
- Return rule: the party's town changes only by travel, so `beginWorldMapReturn` routes a won
  mission back to the party's town, the town it left for that mission.

### Descriptions (review note N6)

`townScreen.townDescription` reads the party's town's entry from the campaign's catalog. A
non-starting description's square, tavern, school and shop art load through `townArtOf`, cached
per description on the install's shared archives. The shop book is lent by each front end. The
starting town's art stays in the install resources. `roomPage` returns no page for a nil
description instead of dereferencing it. `TownDescription(profile)` remains the starting town's
description for the install share, the default shop art and the mission tip's close click (an
edition test), and decode refuses a chapter town without a usable description.

### SAV

The town record goes in the session JSON of the `AgainromActions` supplement leaf, field
`Towns` (`currentTowns`): `Town`, the party's town ID, and `Others`, each other chapter town
whose state differs from its fresh build (ID, taken offers, tavern pool, capacity and hired
flags; a hired flag names the town that hired the squad the party carries). The campaign's
unlocks stay in the original record's `PermanentMercenaries`. The record is written only when the
party's town is not 1 or another town's state differs from its fresh build. A campaign of one
town, which is every shipped first-game campaign, writes no field. A missing field loads as town 1
with every other town fresh. LOAD refuses, with a bounded error, a record naming a town the
campaign does not hold or one town twice, a pool below zero or above its capacity, a hire of a
type with capacity 0, and one type hired in two towns. Every restore of a snapshot returns that
error (`restoreTown`). The campaign refuses a chapter town whose `opens-after` names a mission it
does not hold (`Campaign.checkTowns`). Row: DIV-3853.

### Second game

`secondCampaignScreen.placePoint` reads a town's point through its catalog entry (`record`), the
same record read as before. The town-opening rule stays in `completeBank`'s per-mission switch
(town 2 after mission 20, town 3 after mission 50, with the bank writes in the same cases), named
below as debt.

## Proof

- `pkg/town/catalog_test.go`: the three fields decode; negative opening, zero mission and a
  mission listed twice fail.
- `pkg/game/towntravel_test.go`, on the fixture map of four objects:
  `TestPartyTravelsBetweenTwoChapterTowns` covers the map showing East at its point, travel there
  and back, separate offers and hires, each town's description and home title, and a win of
  mission 31 returning to East. Its loss control returns toward a fixed home point, which ends in
  West. `TestTownOpensAfterItsMission`; `TestShippedFirstGameHoldsOneTown` (no record, no other
  state, no SAV record, home at MapObject1). `TestTownRecordCarriesThePartysTownAcrossLoad`: the
  record round-trips; its loss control drops it and the LOAD lands in town 1. Bad records are
  refused. `TestCatalogRefusesATownWithNoUsableDescription`.
- `pkg/game/townmercenary_test.go`: `TestHireEndsInItsTownWhereverTheMissionEnds` (a squad hired
  in West, a mission ended from East: East neither hires the type nor loses its pool; West's hire
  ends with the live tally as its pool; a second mission from West grows that pool by one),
  `TestMercenaryUnlockReachesEveryTown` (an unlock won in West is enabled in East, in a town that
  opens after that mission and after a restore), `TestTownRecordRefusesBadMercenaryState`,
  `TestCampaignRefusesATownOpeningAfterAnUnknownMission`.
- `TestReleaseTwoTownHireGoesHomeAcrossAMissionSave` (EN and RU): a squad hired in West, travel to
  East, whose tavern neither lists nor hires the type; mission 31 started from East, a mission
  SAVE and a cold LOAD land in East with West's hire; the mission end clears it, West's pool is the
  live tally and East keeps its pool. `TestReleaseShippedFirstGameSavesCarryNoTownRecord` (EN and
  RU): the shipped town SAVE, with a squad hired, and mission SAVE write no `Towns` field; a
  `townsRecord` that always writes a record fails it.
- `TestReleaseTwoTownSaveLoadsIntoTheSecondTown` (EN and RU): the fixture catalog over the install;
  a town SAVE in East and a cold LOAD land in East with West's hire kept and East's description.
  Its loss control is the same file without its engine-state leaves, which lands in town 1.
- Unchanged: the EN and RU release sets, the mission-stage scenarios, the second game's SAVE
  selection on `rom2-en` and `rom2-ru`, and the milestone-2 family; the lane's return lists each
  run. Headless frames of the fixture on EN are in the lane's notes, not in the repository.

## Open debt

- The first game's shop goods (`FrontEnd.Shop`) are session state, shared by every first-game
  town; a town's offers and mercenaries are its own.
- The second game's town opening is a per-mission switch (`completeBank`), not entry data.
- A first-game town other than town 1 must offer its missions from the chapter keys; a town with
  its own mission source has no offer source yet.
- T3: a mod supplies catalog entries in this format, loaded through the same decoder, with a
  fixture mod and its own description and art.
