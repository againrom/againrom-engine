# Town catalog in the base

## Intent

Towns become one base kind. Each game lists its towns in one town catalog in one entry format; the
town screens, the town state and the party's place read the catalog's town ID. The first game
lists its one town, the second game its three. A later story can add a town by supplying an entry
in the same format. No player-visible change.

## Authority

Owner direction (backlog items T1 to T3): towns are a base concept, not wired per game; the first
game gives one town, the second three, and a mod supplies towns in the right format. Owner rule:
one builder per kind, so the list of towns and the party's place are one kind each. Each game
keeps its own town descriptions and art; only the catalog, the entry format, the composer and the
place model are shared.

## As built

### Entry format

`pkg/town/catalog.go`. A catalog is a JSON object decoded strictly (unknown fields and trailing
data are errors):

| Field | Type | Meaning |
|---|---|---|
| `catalog` | string | The catalog's name; an edition names its catalog by it. |
| `cite` | list of strings | Claims or rows behind the catalog. |
| `towns` | list of entries | The towns, in catalog order. At least one. |

Each entry:

| Field | Type | Meaning |
|---|---|---|
| `id` | int | Town ID, positive, unique in the catalog. The key of the town's state and of the place "town ID". |
| `description` | string | Name of the town description (the composer JSON) the town screen is composed from. |
| `hooks` | string | Hook set the description's hooks, conditions and values resolve in: `chapters` (the first game's town screen tables `townHooks`, `townConditions`, `roomValues`) or `destinations` (the second game's `secondTownHooks`, `secondTownConditions`, `secondTownValues`). |
| `offers` | string | Source of the tavern, shop and school offers: `chapter-keys` (`scenario.reg` chapter keys `InnMission`, `ShopMission`, `TCMission`) or `scenario-bank` (the second game's Scenario bank). |
| `map-point` | object | World-map point source, exactly one of `object` (the world-map registry's MapObject number, from 1) and `record: true` (the world-map description's catalog record of kind town and this ID). |
| `cite` | list of strings | Claims or rows behind the entry. |

Every value is plain data. Hook sets and offer sources are names a campaign answers
(`town.CatalogVocabulary`); an unknown name fails the decode.

### Catalogs

- `pkg/game/towncatalogs/rom1.json`: town 1, description `rom1`, hooks `chapters`, offers
  `chapter-keys`, map point object 1.
- `pkg/game/towncatalogs/rom2.json`: towns 1, 2 and 3, descriptions `rom2-town1`, `rom2-town2`,
  `rom2-town3`, hooks `destinations`, offers `scenario-bank`, map point the world-map record.

`Edition.Towns` names the catalog (`rom1`, `rom2`). It replaces `Edition.Town` and `Edition.Rooms`;
`secondTownDescriptions` is removed. `pkg/game/towncatalog.go` decodes every catalog at start and
each description a catalog names under its entry's hook set; a description named under two hook
sets, an unshipped description, or an unknown hook set or offer source stops the process.
`TownCatalog(profile)` answers the catalog. The first game's town screen and room pages read
`TownDescription(profile)`, the catalog's town under the `chapters` hook set at the party's town
ID; the second game's town screen reads the current town's entry under the `destinations` hook
set. `RoomDescription` is removed: in the first game it named the same description.

### Town state

`townState` (`pkg/game/town.go`) is one town's own state: its ID, its consumed offers and its
tavern's mercenary pools, capacities and flags. `newTownState(entry, campaign)` is its one
builder. The first game's `Town` embeds the state of town 1, so every reader keeps its field names.
A `scenario-bank` town's state holds its ID and no pool: the second game keeps its inn in the
Scenario bank and its shop assortment in the shop records, both indexed by the same town ID.

### Place model

`partyPlace{kind, id}` (`pkg/game/place.go`), kind 1 mission and kind 2 town, is the second game's
former `secondLocation` and now the place of both games. `campaignService.place(session)` answers
it: the second game reads its campaign's current place; the first game stands in its live mission
while one is open and otherwise, once the campaign reached its town, in town `Town.townID()`, which
is 1.

### SAV

No byte moves. The second game's leaf (`currentSecondCampaign`, fields `Current`, `Available`)
keeps its layout. The first game writes no town ID.

## Proof

- `pkg/town/catalog_test.go`: the entry format decodes field by field; fourteen broken forms each
  fail with a named error.
- `pkg/game/towncatalog_test.go`: `TestFirstGameCatalogListsItsOneTown` and
  `TestSecondGameCatalogListsItsThreeTowns` pin each catalog's towns, descriptions, hook sets,
  offer sources and map points on `rom1-en`, `rom1-ru`, `rom2-en`, `rom2-ru`; each second-game
  town has a world-map record and a shop record. `TestTwoChapterTownsKeepTwoStates`: a fixture
  catalog of two first-game towns builds two states; town 2's accepted offer, hire and mercenary
  loss do not reach town 1. Loss controls: one `Town` for both IDs leaks the hire, the offers and
  the mercenaries; two towns sharing one offer record leak the offers.
  `TestOnePlaceModelForBothGames` pins the place in both games.
- Unchanged: the EN and RU release sets, the mission-stage scenarios on EN and RU, and the second
  game's SAVE selection on `rom2-en` and `rom2-ru`; the lane's return lists each run.

## Open debt

- T2: world-map travel between catalog towns; the home point still reads MapObject1
  (`worldMapHomePoint`) and the second game's map reads its own record, not the entry's
  `map-point`; return to the town the party left; open towns and offered missions as data; the
  session holds one first-game town state, and holding one state per catalog town needs the town
  ID in the SAV (a missing field meaning town 1).
- T3: a mod supplies catalog entries in this format, loaded through the same decoder, with a
  fixture mod.
