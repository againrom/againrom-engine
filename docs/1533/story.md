# A mission Shop opens its Shop page in the first game

## Intent

When a hero uses a Shop structure (class 34 or 35) on a first-game mission
map, the town's Shop page opens over the mission. Buy and Sell act on the
hero's pack and purse in the mission's World. Exit returns to the mission at
the frame it left. What the hero bought or sold survives SAVE and LOAD.

## Authority

- Story brief (owner): the stock is what claims establish, else the town's
  current stock, with one DIV row; the mission is the second caller of the
  town's one Shop page builder; one builder per kind.
- Seat direction: one trade call per deal unless that breaks a town-graph
  invariant; the `campaign+0x3dc` gate and the contested status of
  `SHOP-ENTRY-016` go to open debt.
- Claims read for this story (`knowledge/`, `go run ./tools/claim <ID>`):
  - `UNIT-STRUCTUSE-137` (High): an actor's use of a Shop posts `0x83`, then
    `0x42a`.
  - `TOWN-475` (High): `0x42a` reaches the shop open `R1315` while
    `campaign+0x3dc` is 0, or 1 with `+0x6bc` at most 1.
  - `SESS-SCREEN-003` (High): the idle handler runs a simulation tick only
    while bit 0 of `campaign+0x3dc` is set.
  - `TOWN-373` (High): `R1315` sets bit 1 only.
  - `SHOP-ENTRY-016` (High, amended, contested): in single player the shop
    resolver returns the static town shop for every placed Shop id; the claim
    reads the mission placement as unreachable.
- Census at base `05289791` (instrument in the story notes): the one shipped
  mission Shop is `scenario/120.alm` record 71, id 72, class 35, owner slot 2,
  tile (69,109), on EN and on RU.

## As built

Kinds and their one builder:

| Kind | Builder | Change |
| --- | --- | --- |
| Structure use | `sim.World.applyStructureUse` | Classes 34 and 35 in the first game record a Shop use in the step report (`Report.ShopUses`); no hashed state changes |
| Shop page | `shopController` (town Shop room) | The mission is its second caller, by data: the `bag`, `settle` and `wornFixed` hooks and a session copy |
| Shop stock | `Town.newShop` | Extracted from `arriveInTown`, its first caller; the mission visit is its second |
| Town install with the launch multiplier | `runtimeSwitches.townInstall` | `FrontEnd.townInstall` and the mission visit both call it |
| Room steps | `town.Scene` | `ExitRoom` runs a room's exit steps; `LeaveRoom` is `ExitRoom` then `EnterSquare` |
| Mission screen host | `pkg/ui` flow screens | `ScreenTown` is armed over the armed mission map (`flow.shopOverMap`) |
| Pack and purse change | `sim.World.TradeCarried` | New: one atomic change of one actor's pack and purse |
| SAV writer | unchanged | The World carries the pack and purse |

Route:

1. The hero's use finishes in `applyStructureUse`. The step reports the user
   and the structure. `mapWorld.observeShopUses` accepts a member of the party
   the mission opened with and raises the Viewer's shop request.
2. The map arm of `App.step` takes the request on the same gate as the
   documents panel and calls `townScreen.OpenMissionShop`. The town screen
   keeps a copy of itself and runs the Shop room over a session copy: the
   town's state with the hero's purse as its gold, the hero as its party, no
   city graph, the town not open, and the mission's stock as its deal.
3. Each committed page gesture runs through `settleMissionDeal`. The page's
   pack and its table units are matched unit by unit against the World's
   pack (`shopDealDiff`), and the difference is one `World.TradeCarried`
   call. A deal the World refuses restores the page as it stood.
4. Exit or Escape calls `Back`, which closes the visit: the room's exit steps
   return the table, the deal is committed, the town screen is restored, and
   the flow returns to `ScreenMap`. The map's pace restarts from the frame it
   resumes on.

The trade is not a `sim.Command`: a command cannot carry a bought item's
instance. It is one call per committed gesture, and the visit's town copy has
no city graph, so no town-graph invariant is touched.

Divergences: `DIV-4218` (the stock is a per-mission copy),
`DIV-4219` (the mission clock stands while the page shows),
`DIV-4220` (customer, doll and pack order), all in
`docs/divergences/mission-screen.md`. `DIV-284` in
`docs/divergences/client-mission-screen.md` is amended to the remaining
`+0x6bc` gate.

## Proof

Base `05289791cc3d281a8ed0dbd59e132d06aab6d593`; head is this branch after
merging engine main `3c0dbfc3`. Frames were captured headlessly on the EN and
RU roots from a `git archive` of the base and from the head. The same captures
from a `git archive` of `3c0dbfc3` equal the base captures byte for byte.

- Mission states: a scratch test in `pkg/game`, not committed, finishes the
  campaign's first town-offering mission (`FinishMission`), opens mission 120
  through `MissionOpener`, places the hero four cells west of the Shop
  (record 71, class 35, tile 69,109) with god mode on, dismisses the opening
  notices, queues the Shop use and steps until world tick 110. It writes the
  CPU-composed parts of each state: the character pane and the minimap of the
  mission (the map itself has no CPU composite) and `HeadlessFrame` of the
  page. A second run buys the first shelf unit before Exit.
- Equal states: `cmd/screenshot`, every screen it composes (14 on EN, 13 on
  RU), built with `scripts/build-tools.sh` at head and with the same flags
  from the base archive. `documents` and RU `town-tavern-talk` are not reached
  and `picker`, `gameplay` and `game-menu` are refused at base and at head
  alike.

The use finishes at world tick 80 on both trees. At head the page shows at
tick 80 and the World stays at tick 80 while the page shows; after Exit the
mission runs to tick 110. At base no page opens and the mission runs to tick
110. World hashes at tick 110 without a deal: EN `74c97c4db6faf2a1` at base and
head, RU `a118147f0f9a5729` at base and head. With the purchase: purse 600 to
597, pack `[3612]` to `[3612 41475]`, EN hash `9fb3389cbde73295`, RU
`f2dbd735ce154a7d`.

39 of 43 outputs are byte-equal. The 4 that differ are the page frames, which
do not exist at base.

| State | Root | Base SHA-256 | Head SHA-256 | Equal |
| --- | --- | --- | --- | --- |
| mission 120 before use, pane | en | 6892890241056f220fa353a6395f3baec683e01025bc8d6349fa411047de0b53 | 6892890241056f220fa353a6395f3baec683e01025bc8d6349fa411047de0b53 | yes |
| mission 120 before use, minimap | en | 217f18e033703bf0f384c477fefe52fcd36e5afc56ea10c4c125ce9182c3b151 | 217f18e033703bf0f384c477fefe52fcd36e5afc56ea10c4c125ce9182c3b151 | yes |
| mission 120 Shop page | en | no page | 63e27fdccab2ceeb565030fb086ceb1393d96c3d1f525aa44ac2708a944b9d59 | no |
| mission 120 after Exit, pane | en | 6892890241056f220fa353a6395f3baec683e01025bc8d6349fa411047de0b53 | 6892890241056f220fa353a6395f3baec683e01025bc8d6349fa411047de0b53 | yes |
| mission 120 after Exit, minimap | en | 07e4206dec76c2e6b9d81b2e0023721c55ade41545231d4f23ea134da4c17bc7 | 07e4206dec76c2e6b9d81b2e0023721c55ade41545231d4f23ea134da4c17bc7 | yes |
| mission 120 Shop page after a purchase | en | no page | b3edc58f1968fa25ee04919c15b98c3a720cab0f3237134a88be7ac9842dc8e6 | no |
| mission 120 after a purchase and Exit, pane | en | 6892890241056f220fa353a6395f3baec683e01025bc8d6349fa411047de0b53 | 6892890241056f220fa353a6395f3baec683e01025bc8d6349fa411047de0b53 | yes |
| mission 120 after a purchase and Exit, minimap | en | 07e4206dec76c2e6b9d81b2e0023721c55ade41545231d4f23ea134da4c17bc7 | 07e4206dec76c2e6b9d81b2e0023721c55ade41545231d4f23ea134da4c17bc7 | yes |
| chargen-detailed | en | d15f181e2270152fb368f6ada8b16e318259b78a7ba3000fb53fa2b7182c4672 | d15f181e2270152fb368f6ada8b16e318259b78a7ba3000fb53fa2b7182c4672 | yes |
| chargen-precreate | en | b1559d8a7774c20a4d3ceaa69d73889bdc46ae75f79754f86582a9900e8ec4b1 | b1559d8a7774c20a4d3ceaa69d73889bdc46ae75f79754f86582a9900e8ec4b1 | yes |
| load | en | 9c4f66d6eb4b9977463773c827d45e92dd4c539eae4f6a9baa1358dc1b566d02 | 9c4f66d6eb4b9977463773c827d45e92dd4c539eae4f6a9baa1358dc1b566d02 | yes |
| menu | en | 9877a515580f6759e93279dcecf2745b697d7912bfe61411b3dd7e3aab7cc050 | 9877a515580f6759e93279dcecf2745b697d7912bfe61411b3dd7e3aab7cc050 | yes |
| mission-column-card | en | faf3b041f7ca6e2e47e8db01e54994285446aee8f3595a3fc17e30b113984a8c | faf3b041f7ca6e2e47e8db01e54994285446aee8f3595a3fc17e30b113984a8c | yes |
| mission-pane-doll | en | 90f320e7a4b94bdc8b85f58c6286f6a0d139b0c2db4c60dac8de03c57f321ab2 | 90f320e7a4b94bdc8b85f58c6286f6a0d139b0c2db4c60dac8de03c57f321ab2 | yes |
| mission-pane-stats | en | faf3b041f7ca6e2e47e8db01e54994285446aee8f3595a3fc17e30b113984a8c | faf3b041f7ca6e2e47e8db01e54994285446aee8f3595a3fc17e30b113984a8c | yes |
| town-shop-stats | en | 3efb6e057a0564d4df0d6bf41c7055356febaa0ab18ff811bbfa63c44cfb0e0c | 3efb6e057a0564d4df0d6bf41c7055356febaa0ab18ff811bbfa63c44cfb0e0c | yes |
| town-shop | en | 4f21e93ab62ff5b5716667e013cb64fb839cc17d2c5b3efaa829270780a88ae7 | 4f21e93ab62ff5b5716667e013cb64fb839cc17d2c5b3efaa829270780a88ae7 | yes |
| town-square | en | 5ecd27ad36d8669d5962492dbae3c756c156a05b1a61ef2821d4c9c5bce3612f | 5ecd27ad36d8669d5962492dbae3c756c156a05b1a61ef2821d4c9c5bce3612f | yes |
| town-tavern-next-frame | en | 157f7e4d0611b02250c71ae509f6b964ce26f075ceadb6da281187a0bf1a1bb9 | 157f7e4d0611b02250c71ae509f6b964ce26f075ceadb6da281187a0bf1a1bb9 | yes |
| town-tavern-selected | en | 8be774cf4dace6d2b08ecdd991dc4896aa1ece9a11a8c3c543a374685ebc88f9 | 8be774cf4dace6d2b08ecdd991dc4896aa1ece9a11a8c3c543a374685ebc88f9 | yes |
| town-tavern-talk | en | c410357dda48f7dc5098f59b1c9a9f428e2c87b1b5d6a3f80e1fcad393fdb4cd | c410357dda48f7dc5098f59b1c9a9f428e2c87b1b5d6a3f80e1fcad393fdb4cd | yes |
| town-tavern | en | 8be774cf4dace6d2b08ecdd991dc4896aa1ece9a11a8c3c543a374685ebc88f9 | 8be774cf4dace6d2b08ecdd991dc4896aa1ece9a11a8c3c543a374685ebc88f9 | yes |
| mission 120 before use, pane | ru | eef9f3972cd2cfd413941d17405779a7a45b8b393ff003c2b38f8a60b43f8171 | eef9f3972cd2cfd413941d17405779a7a45b8b393ff003c2b38f8a60b43f8171 | yes |
| mission 120 before use, minimap | ru | 217f18e033703bf0f384c477fefe52fcd36e5afc56ea10c4c125ce9182c3b151 | 217f18e033703bf0f384c477fefe52fcd36e5afc56ea10c4c125ce9182c3b151 | yes |
| mission 120 Shop page | ru | no page | 3e9c2c8f4a46907c924429ee5249a2a74f3a614d252e86e9f293d15cb3f0c0ff | no |
| mission 120 after Exit, pane | ru | eef9f3972cd2cfd413941d17405779a7a45b8b393ff003c2b38f8a60b43f8171 | eef9f3972cd2cfd413941d17405779a7a45b8b393ff003c2b38f8a60b43f8171 | yes |
| mission 120 after Exit, minimap | ru | 07e4206dec76c2e6b9d81b2e0023721c55ade41545231d4f23ea134da4c17bc7 | 07e4206dec76c2e6b9d81b2e0023721c55ade41545231d4f23ea134da4c17bc7 | yes |
| mission 120 Shop page after a purchase | ru | no page | be93ded945187def5f5baf6901aab97cfd86dc378e7d1e08791a8854311f132f | no |
| mission 120 after a purchase and Exit, pane | ru | eef9f3972cd2cfd413941d17405779a7a45b8b393ff003c2b38f8a60b43f8171 | eef9f3972cd2cfd413941d17405779a7a45b8b393ff003c2b38f8a60b43f8171 | yes |
| mission 120 after a purchase and Exit, minimap | ru | 07e4206dec76c2e6b9d81b2e0023721c55ade41545231d4f23ea134da4c17bc7 | 07e4206dec76c2e6b9d81b2e0023721c55ade41545231d4f23ea134da4c17bc7 | yes |
| chargen-detailed | ru | a1e6027caf9e5bfc013b0371c84ca68c8a71e6ab4413e4f4d703301155cd7b4e | a1e6027caf9e5bfc013b0371c84ca68c8a71e6ab4413e4f4d703301155cd7b4e | yes |
| chargen-precreate | ru | dfbf9919d2f2fb9841d8d12b99b459fb71a227e3b548b4e649bfe8b96dc88839 | dfbf9919d2f2fb9841d8d12b99b459fb71a227e3b548b4e649bfe8b96dc88839 | yes |
| load | ru | 4044e1b5df9d1051ba8b64bb15a1c847e6faec11a046ddb7b0532e44279c6f7d | 4044e1b5df9d1051ba8b64bb15a1c847e6faec11a046ddb7b0532e44279c6f7d | yes |
| menu | ru | e2b8f75dd3e29de69fbc9115a7e2b5abe2c44608eb0ba4acc82b7759a3c750e3 | e2b8f75dd3e29de69fbc9115a7e2b5abe2c44608eb0ba4acc82b7759a3c750e3 | yes |
| mission-column-card | ru | 4e33850792a6b56244cd68c425cb90d93afcad94bdeab8ec3e937394c1c113a2 | 4e33850792a6b56244cd68c425cb90d93afcad94bdeab8ec3e937394c1c113a2 | yes |
| mission-pane-doll | ru | 90f320e7a4b94bdc8b85f58c6286f6a0d139b0c2db4c60dac8de03c57f321ab2 | 90f320e7a4b94bdc8b85f58c6286f6a0d139b0c2db4c60dac8de03c57f321ab2 | yes |
| mission-pane-stats | ru | 4e33850792a6b56244cd68c425cb90d93afcad94bdeab8ec3e937394c1c113a2 | 4e33850792a6b56244cd68c425cb90d93afcad94bdeab8ec3e937394c1c113a2 | yes |
| town-shop-stats | ru | cbe12bd5dc223aad844336b33b68687964dcbd3b5b9a04bea9e40103a5c7f658 | cbe12bd5dc223aad844336b33b68687964dcbd3b5b9a04bea9e40103a5c7f658 | yes |
| town-shop | ru | 129e9b9e5bf03643a0f1829b56d0ece917b40da3dbb86828530d5d6f060d8d6f | 129e9b9e5bf03643a0f1829b56d0ece917b40da3dbb86828530d5d6f060d8d6f | yes |
| town-square | ru | d54a0f6af72ac10bd2fc8f1ab287080e958c7d64d234d04e7c8ea6c74edcbeea | d54a0f6af72ac10bd2fc8f1ab287080e958c7d64d234d04e7c8ea6c74edcbeea | yes |
| town-tavern-next-frame | ru | 8097176bf1bec32f6087eb0748f003106f35208bbc83992411214790106a2711 | 8097176bf1bec32f6087eb0748f003106f35208bbc83992411214790106a2711 | yes |
| town-tavern-selected | ru | 3c57df7729ebfc0cd559b31ffd2432eafd488ee0b3ac2095d9ef2d5d606a7e17 | 3c57df7729ebfc0cd559b31ffd2432eafd488ee0b3ac2095d9ef2d5d606a7e17 | yes |
| town-tavern | ru | 3c57df7729ebfc0cd559b31ffd2432eafd488ee0b3ac2095d9ef2d5d606a7e17 | 3c57df7729ebfc0cd559b31ffd2432eafd488ee0b3ac2095d9ef2d5d606a7e17 | yes |

Focused tests: `pkg/sim/shoptrade_test.go` (Shop use reports the user for
classes 34 and 35 and changes no hashed state; the trade is atomic; refusals
change nothing; a saved pack reloads) and `pkg/game/missionshop_test.go` (the
page opens and trades with the hero, the clock stands, Escape returns and the
town's shop and gold are unchanged; the Exit button returns and the doll is
fixed; a class 28 use opens no page; SAVE after a purchase and a fresh LOAD
give the same World hash, pack and purse). With the Shop arm in
`applyStructureUse` switched off, 2 sim and 3 game tests fail.

Release tests to run at a release, each with `AGAINROM_ASSETS` on the EN root
and on the RU root (`pkg/game`):

- Structure use: `TestReleaseStructureUse1150` (includes the mission 120
  class 35 placement) and
  `TestReleaseALoadedAttackCycleFinishesBeforeAStructureUse`.
- Shop page: `TestReleaseShopClicksTransferOnTheReleaseFrame`,
  `TestReleaseCheckshopStaffStockSurvivesPurchaseAndCurrentSAV`,
  `TestReleaseShopBookPurchaseReadAndMissionCarry`,
  `TestReleaseRoomTipsAtEveryEnterAndShopSecondText`.
- Mission pack and SAVE/LOAD:
  `TestReleaseMissionPackAndWornBoxDoubleClicksSaveAndColdLoad`,
  `TestReleaseEnginePurseGoldDropSurvivesQueuedSaveLoadAndPickup`.

## Open debt

- The `campaign+0x3dc` and `+0x6bc` gate of `TOWN-475` is not modelled: the
  page opens whenever a party member's use finishes (`DIV-284`).
- `SHOP-ENTRY-016` is contested and reads the mission placement as
  unreachable in single player; the owner's brief opens it.
- Whether the shop's entry offer speaks on a mission visit is Unknown; the
  visit's town is not open, so no offer speaks.
- An actor who joined the party after the mission opened opens no page
  (`DIV-4220`).
- The mission's stock copy is not written to the mission SAV; a LOAD draws it
  again (`DIV-4218`).
- The mission's pace restarts on close, so no catch-up ticks run
  (`DIV-4219`).
