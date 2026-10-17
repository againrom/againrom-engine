# 1454 — Town services along the campaign chain

## Result

`TestReleaseCampaignChainFromMission10` now uses the tavern, the shop and the
school in each of the 13 towns of the Main line (the towns before missions 30,
40, ..., 150). Each service runs in a `services_<n>` subtest from its own cold
LOAD of the town SAV the chain writes, so the Main line, its purse ledger and
the side missions are unchanged. The tavern hires every squad the purse
allows, the shop buys one item and sells one, and the school trains one skill
step. Each result is checked after a SAVE and a cold LOAD. Hired squads are
also checked in the party of the chapter's mission, entered by the world map.
It passes on EN and RU. No engine defect was found. The story adds no
divergence row and changes no engine behaviour.

## Intent

Owner direction (backlog: wider campaign regression). Stories 1340 and 1452
check the chain's offers and purse but buy, hire and train nothing. Tavern,
shop and school had release tests only in a few towns. A defect in a later
chapter's roster, shelf or school would reach the owner first.

## Authority

No ROM1 behaviour is claimed beyond the installed data and these claims.

- Tavern roster: `MERC-SHELF-002`. The chapter's `[Mission<n>] Mercenaries`
  ∩ the types unlocked by an `EnableMercenary` of a won main mission ∩ a
  non-empty pool. The chain hires nothing and wins no side mission on the Main
  line, so each pool is `[General] MercenaryCount` (`MERC-DEATH-006`) and
  only `[Mission10]`'s unlocks apply. The test reads all three keys from
  `scenario.reg` with `pkg/formats/reg`, not through the campaign decoder.
- Squad price: `MERC-PRICE-004`, `(PriceA + n·PriceB) × unitPrice(mission)`.
  The test reads `PriceA` and `PriceB` of `npc<type>` from `npc.reg` with
  `pkg/formats/reg`, not through `pkg/data`'s NPC decoder. It holds the
  claim's ladder `10 15 20 40 60 80 100 600 800 1000 6000 8000 10000` as its
  own literal. A hire takes the whole squad (`MERC-HIRE-003`).
- Shop ceiling: `[Mission<n>] ShopMaxPrice` from `scenario.reg`, read with
  `pkg/formats/reg` (`SHOP-MISSION-018`). The floor is 0
  (`SHOP-MISSION-019`).
- Shelf contents: `SHOP-GEN-005`, `SHOP-POOL-006`. The armour and weapons
  shelves hold 100 draws each and the magic shelf at most 20. Item prices come
  from `Data.bin` through `data.ShopPool`. That production decoder is the only
  reader of item prices; the shelf loss controls carry this check.
- Trade: `SHOP-BUY-009` (the purse falls by the unit price) and
  `SHOP-SELL-010` (the purse rises by `ftol(0.5 × price + 0.5)`). The sold
  unit's price is the one the table shows, from the same production decoder.
- School: `HERO-SKILLBUY-076`, price `ftol(1.1^base × 200)` for slots 1..5.
  One step raises the skill by one. The test computes the price itself. The
  base is the member's maintained level as the loaded SAV gives it.

## As built

`pkg/game/campaignservices_release_test.go`, called from
`pkg/game/campaignchain_release_test.go`. Rooms open through the App
(`TAVERN`, `SHOP`, `SCHOOL`; offer conversations are answered). Squads are
hired through the App's `Mercenary <type>` cell and Hire button. Training uses
the school's cell and the App's Train button, and the shop uses the town
screen's `ShopClick` and `ShopScroll` doors. A school member step that turns
the class column is settled by App paints on a test clock.

| service | assertions per town |
|---|---|
| tavern | roster (type, head count, price) equals the installed roster; each squad the purse allows, in cell order, hired: purse falls by its price, hire flag set, its members carried; after SAVE and cold LOAD the purse, flags and carried members; world-map entry of mission `<n>`: mission purse equals the town purse, the mission party holds each hired squad, one entity per member |
| shop | ceiling equals `ShopMaxPrice`; every armour and weapons element is a candidate at its price, 100 units each; magic shelf 1..20 units of the unfiltered pool; the cheapest affordable weapons item bought (armour if none): purse and the shown member's pack move by its price; one unit of the first priced pack stack sold: purse rises by half its price rounded up, pack falls by one; after SAVE and cold LOAD the purse, the pack and the regenerated shelf |
| school | the dearest affordable step over the whole party (member stepped to with the picker), the base from the loaded member; the cell shows level and price; Train: purse falls by the price, skill + 1; after SAVE and cold LOAD the same |

Observed on EN and RU (identical):

| town | roster types | hired | shop ceiling | trained |
|---|---|---|---|---|
| 30 | 14 | 14 | 1000 | hero slot 4, 10 → 11, 518 |
| 40 | 6 14 | 6 14 | 3000 | hero slot 4, 10 → 11, 518 |
| 50 | 6 13 14 | 6 14 | 7000 | npc25 slot 1, 25 → 26, 2166 |
| 60 | 4 6 13 14 | 4 | 10000 | npc25 slot 1, 25 → 26, 2166 |
| 70 | 4 6 13 14 | none affordable | 30000 | npc25 slot 1, 25 → 26, 2166 |
| 80 | 4 6 7 13 14 | none affordable | 70000 | npc25 slot 1, 25 → 26, 2166 |
| 90 | 3 4 6 7 13 14 | none affordable | 100000 | npc25 slot 1, 25 → 26, 2166 |
| 100 | 2 3 4 6 7 13 14 | 7 | 300000 | npc25 slot 1, 25 → 26, 2166 |
| 110 | 2 3 4 6 7 9 13 14 | 9 7 4 | 700000 | npc24 (mage) slot 2, 60 → 61, 60896 |
| 120 | 2 3 4 6 7 9 13 14 | 9 7 | 1000000 | npc24 (mage) slot 2, 60 → 61, 60896 |
| 130 | 2 3 4 6 7 9 12 13 14 | none affordable | 3000000 | npc24 (mage) slot 2, 60 → 61, 60896 |
| 140 | 2 3 4 6 7 9 12 13 14 | none affordable | 7000000 | npc24 (mage) slot 2, 60 → 61, 60896 |
| 150 | 2 3 4 6 7 9 12 13 14 | none affordable | 10000000 | npc26 (mage) slot 1, 70 → 71, 157949 |

Each shop bought a 4, 7 or 16 gold weapon and sold the hero's code `0xe06`
(price 50, paid 25). The carried document `0xe1c` has no price and is not
offered for sale.

Loss controls run in town 50. Each must make its check fail:

| subtest | carrier removed |
|---|---|
| `tavern/loss_controls/roster_unlock_removed` | the unlock of the first roster type in a cold town |
| `tavern/loss_controls/hire_purse_changed` | the loaded purse after the hires, one gold either way |
| `tavern/loss_controls/hired_unit_dropped_from_mission` | the first hired squad from the mission party |
| `shop/loss_controls/trade_purse_changed` | the loaded purse after the trade, one gold either way |
| `shop/loss_controls/bought_item_removed` | one unit of the bought item from the expected pack |
| `shop/loss_controls/shelf_ceiling_changed` | the shop's ceiling, raised by one |
| `shop/loss_controls/shelf_price_changed` | a weapons element's price, raised by one |
| `school/loss_controls/trained_skill_removed` | the trained step from the loaded member |

## Proof

On the merge of current main (0.125.1):

- Services alone, `go test -run 'TestReleaseCampaignChainFromMission10$' -v`
  per root: the `services_<n>` subtests sum to 14.9 s on EN and 13.7 s on RU.
- Inside the whole release set (12 workers): services 19.2 s on EN and 18.7 s
  on RU. The chain test as a whole took 45.7 s and 44.9 s.
- `check-release-tests.sh`: EN 1142 and RU 1140 of 1246 ran, no failure.
- `go test -trimpath -count=1 ./...`, `gofmt -l`, `go vet` of the touched
  packages, `git diff --check` and `scripts/check-no-game-assets.sh` are
  clean.

## Open debt

- The purse along the Main line affords no squad in towns 70, 80, 90, 130, 140
  and 150. There the roster is checked but nothing is hired. Types 1, 2, 3,
  12 and 13 are never hired, and type 1 (Catapult) is never offered on the
  Main line.
- The shop buys the cheapest affordable item. Magic, book and potion shelves
  are checked only for draw count (magic) or not at all (books, potions).
- Item prices and the members' skill bases come only from production decoders.
  Their correctness rests on the loss controls and the focused shop and school
  tests, not on an independent reader.
- Side-mission unlocks (types 1, 5, 8, 10) are not seen in a Main line town;
  the side subtests do not run the services.
