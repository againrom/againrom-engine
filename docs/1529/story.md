# The second game's bag goes into the mission with the hero

## Intent

In the second game, what the hero bought in a town shop vanished when a
mission started: the bag stayed with the town's shop party and never reached
the party the mission is built from. Now the bag's units enter the mission as
the hero's pack, work there as the mission pack works for found items, are in
the mission SAV and its cold LOAD, and come back to the bag at the next town
with what the mission left. The shop then lists and sells them. The first
game is unchanged.

## Authority

- Owner direction (brief): what the hero bought, worn or in the bag, goes
  into the mission with him, stays usable there (wear, take off, drop, pick
  up through the mission's existing pack), is in the mission SAV and its cold
  LOAD, and comes back to town with what the mission left him. Where no
  second-game claim states a step, the first game's rule for the same step is
  the reading, and a DIV row names it.
- `R2-ENGINE-404` (High): a buy inserts the shop's own item object into the
  hero bag.
- `R2-ENGINE-406` (High/Medium/Unknown): the read reuse path returns the
  existing hero Unit; that he keeps his item objects on it is Medium. In a
  mission, ground items reach the bag through the item-move message and
  `#pickup all`. Whether the map unload at a mission's end keeps the hero
  Unit, and the town re-entry, are Unknown.
- `R2-ENGINE-403` (High/Medium/Unknown), `R2-ENGINE-327`, `R2-ENGINE-333`,
  `R2-ENGINE-414` (High): the item's ID, type, price and effect list; the
  armour ID; types 1 and 2; effect tags 41 and 42.
- `R2-SESSION-236` (High/Medium/Unknown): each Player-1 Human of the frozen EN
  files, town-like and mission-like, holds equipment and bag.

No second-game claim reads the hero's items at a mission's end or at the next
town's entry. The first game's party carry fills that step (DIV-4191).

## As built

The hero's bag is the pack of `secondCampaign.party[0]`, the one-member shop
party (DIV-3043). The mission is built from `CampaignSession.Carried`, the
campaign party. The bag now joins that party's carry at two boundaries:

- `CampaignSession.leftLocation`, which every town leave through the gates and
  every won mission's departure calls, and the map's mission entry
  (`destinationTownMap.enterLocation`), call `packBag`: the bag's units move
  to the end of the campaign hero's pack through the shop's pack writer
  (`memberWithPack`, extracted from `setShopPackItemInstancesRefresh` in
  `pkg/game/shoproom.go`), each in its mission form,
  `secondshop.MissionUnit`: the item `mapload.ItemInstanceFromCode` builds
  for the ID, at the unit's own price and with its castspell or teachspell
  effect. A bought potion thus has type 3 and its payload, and the mission
  pack's use drinks it. The map's entry covers a bag the map still holds.
  The bag stays when the hero's carried load refuses the pack.
- `destinationTownMap.enterLocation` for a town calls
  `CampaignSession.enteredTown`: each unit of the campaign hero's pack that
  `secondshop.KeptUnit` admits goes back to the bag in its bag form, in pack
  order, while the bag holds fewer than 8192 units. `KeptUnit` takes a bag
  unit as it is; any other unit it writes with the bag's one unit writer,
  `secondshop.Unit`, checks the bag form with `secondshop.FromInstance`, and
  admits it only when `MissionUnit` turns that bag form back into the unit's
  ID, type, price and effects; a mission's weight and equipment do not count.
  A potion found in the mission comes home so. A unit bound to a saved
  object or a native record, one at a price below 0 or past the cap (0x0e4d
  at -1 on both installs), and one whose type or effects the mission form
  does not make stay in his pack. Only the hero's pack comes home; Iglez
  keeps his.
- `MissionUnit` and `KeptUnit` live in `pkg/secondshop` beside `Unit` and
  `FromInstance`, and take the decoder as a function: the bag unit's forms
  belong to the bag package, the decoder to `pkg/mapload`, and
  `pkg/secondshop` imports no install table (`internal/archtest`'s graph).
- `secondFileSession` reads a mission-like original file's hero bag only for
  its refusal test: the bag is the mission Human's inventory, which the file's
  world carries, so it is not copied into the campaign's bag and comes home
  once.

Worn items already lived on the campaign party and need no move.

### Kinds and builders

| Kind | Builder used |
|---|---|
| Party member carried into a mission with his items | the first game's carry: `CampaignSession.Carried` through `NextParty`, `mapload.MaterializePartyCarry` at the mission start, `CampaignSession.carryMissionHome` back |
| Pack writer of a party member | `memberWithPack`, the shop's writer extracted unchanged for its shop caller |
| Item instance | `sim.ItemInstance` |
| Bag unit | written by `secondshop.Unit`, read by `secondshop.FromInstance`; `secondshop.KeptUnit` beside them decides that a unit comes home without a loss |
| Item decoder | `mapload.ItemInstanceFromCode`, the one decoder of an item ID |
| Mission form of a bag unit | `secondshop.MissionUnit` over that decoder, used by `packBag` and by `KeptUnit` |
| Mission pack and its screen | the existing mission inventory; no pane added |
| Mission SAV of the Human inventory and equipment | the current SAV writer and reader of the mission World |
| Town SAV bag | `captureSecondParty` and `validateSecondParty`, unchanged |
| Original file session | `secondFileSession` |
| Game difference | none new: the code runs only where a second campaign exists (`secondOf`) |

### Hashed state, SAV and old saves

The bag's units become World items at the mission start, with object IDs, and
are hashed with the World. A mission SAV writes them as the hero's inventory;
the campaign document's bag is empty in a mission. An engine mission SAV
written before this change holds the bag in the campaign document and not in
the World: its LOAD keeps that, and the next town's arrival adds the mission
pack to that bag. A SAV at the map after a victory holds the brought-home
units in the hero's party pack; the town arrival after its LOAD moves them.

## Proof

### Focused tests (`pkg/game/secondbagcarry_test.go`)

| Test | Fails when |
|---|---|
| `TestSecondBagEntersTheMissionWithTheHero` (bought, empty, at the map) | `leftLocation` does not call `packBag` (bought); the map's mission entry does not call `packBag` (at the map). The empty bag is the loss control |
| `TestSecondBagMissionSaveAndColdLoadKeepThePacks` | `leftLocation` does not call `packBag` |
| `TestSecondBagComesHomeAsTheMissionLeftIt` | `leftLocation` does not call `packBag`; the town arrival does not call `enteredTown` |
| `TestSecondBagOfAMissionFileComesHomeOnce` | `secondFileSession` copies a mission-like file's bag into the campaign's bag |
| `TestSecondBagArrivalKeepsWhatTheBagWouldLose` | the arrival writes every unit through `secondshop.Unit` without the loss test |
| `TestSecondBagPotionIsDrunkInTheMission` | `packBag` appends the bag form (type 0, no payload): the use queues nothing |
| `TestSecondBagPotionsComeHome` | `packBag` appends the bag form; `KeptUnit` compares a unit with its bag form instead of its mission form, which leaves both potions in the hidden pack |
| `TestKeptUnitLosesNothing` (`pkg/secondshop`) | `KeptUnit` takes every `Unit` result with a bag form, or compares with the bag form |
| `TestMissionUnitTakesTheDecodersForm` (`pkg/secondshop`) | the mission form is the bag form |

The fixture hero wears a weapon with a castspell effect and an armour and
buys an armour, a weapon and two potions; Iglez (identity 22) has his own
pack and armour. In the return test the hero wears the bought armour, drops
the bought weapon and a potion, picks the potion up, and mission 10 is won by
a script instant through `secondCampaignRules.finishWon`; at town 2 the bag
holds the potions, the hero wears the armour, the weapon stays on the ground,
Iglez keeps his items, and the shop sells the potions.
`TestSecondCampaignTwentyDepartureBankAndSelections` now compares the restored
campaign with the shop's trade rules, which the capture of a nonempty bag
builds on demand.

### Witness over the installs

An untracked test, kept out of the repository, took the install path on its
command line and walked the route over `rom2-en` and `rom2-ru`: new game, the
inn talk, BUY of the first entry of each of the four shop categories, GATES,
mission 10 from the map, a bought armour put on, a bought unit dropped and
picked up through the mission's commands, a mission SAVE and its cold LOAD in
a new front end, the victory route into mission 20, mission 20 won by a
script instant, and the arrival at town 2 from the map. Each cell names a
unit as ID/type/price/effects in the bag's item form. The two installs gave
equal tables:

| Stage | Bag | Hero's pack | Worn |
|---|---|---|---|
| town 1 after BUY | 0x0e81/0/1000/[{42 0 1}] 0x1c3e/1/200 0x1c3e/1/200 0x812d/2/1051/[{41 0 196609}] | 0x0e1c/0/50000 | 0x0103/2/250 |
| mission 10 first frame | empty | 0x0e1c/0/50000 0x0e81/0/1000/[{42 0 1}] 0x1c3e/1/200 0x1c3e/1/200 0x812d/2/1051/[{41 0 196609}] | 0x0103/2/250 |
| mission 10 after wear, drop and pick-up | empty | 0x0e1c/0/50000 0x0e81/0/1000/[{42 0 1}] 0x1c3e/1/200 0x812d/2/1051/[{41 0 196609}] | 0x0103/2/250 0x1c3e/1/200 |
| mission 10 after SAVE and cold LOAD | empty | as above | as above |
| mission 20 after the victory | empty | 0x0e1c/0/50000 0x0e4d/0/-1 0x0e81/0/1000/[{42 0 1}] 0x1c3e/1/200 0x812d/2/1051/[{41 0 196609}] | 0x0103/2/250 0x1c3e/1/200 |
| town 2 after the return | 0x0e1c/0/50000 0x0e81/0/1000/[{42 0 1}] 0x1c3e/1/200 0x812d/2/1051/[{41 0 196609}] | 0x0e4d/0/-1 | 0x0103/2/250 0x1c3e/1/200 |

The witness reads the World and the campaign, not the mission inventory's
pixels, hover or click. The hero's start pack unit 0x0e1c joins the bag at
town 2. Unit 0x0e4d, at price -1, first appears in mission 20's pack; it has
no bag form and stays in the hero's pack. The town 2 sale sold the bag's
first stack.

The table names units in the bag's form, so it is the same with the mission
form. As the World holds them on mission 10's first frame, on both installs,
the bought units are 0x1c3e/1, 0x1c3e/1, 0x812d/2 with its castspell and
0x0e81/0 with its teachspell (the decoder gives book codes type 0 and no
payload), and the start unit 0x0e1c is type 4. A second witness bought town
1's healing potions with the shop's own take and buy: 0x0e46 at 1000 and
0x0e47 at 10000, type 0 and no payload in the bag, are 0x0e46/3 [{6 0 30}]
and 0x0e47/3 [{6 0 100}] in mission 10's pack, both usable; after a wound
from 145 to 73, the mission pack's use of 0x0e46 took the unit and the
health rose to 105, on both installs.

### Release tests for the release

`pkg/game/secondbagmission_release_test.go`, every root:
`TestReleaseSecondBagEntersMissionTen`,
`TestReleaseSecondBagMissionSaveAndColdLoad`,
`TestReleaseSecondBagComesHomeToTheBag`,
`TestReleaseSecondBagPotionIsDrunkInMissionTen`.

## Open debt

- A unit without the bag's item form stays in the hero's pack in town, where
  no town page shows it; on both installs unit 0x0e4d at price -1 does. Which
  units can a second-game mission give the hero, and where does the
  original's town show them?
- The hero's start pack unit 0x0e1c, price 50000, is not in the bag at town 1
  and is in it at town 2 as type 0, where the shop can sell it; in the next
  mission it is type 4 again. Is it in the original's bag at town 1, and can
  the original's shop buy it?
- What type and payload does the original give a bought potion in a mission?
- At the town's arrival the bag takes units in the hero's pack order. In what
  order does the original's bag list units brought back from a mission?
- What does the original do with the hero's items when the map unloads at a
  mission's end, and when the next town is entered?
- Does the original merge a unit brought back into an equal bag entry?
