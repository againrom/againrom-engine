# One World transaction picks up a sack

## Intent

A sack pickup (the order to walk to a sack's cell, the arrival test, the take, the quest documents moved to the starting hero, the completion on the next Step) has one builder, a World transaction in `pkg/sim/sackpickup.go`. Every driver calls it: the window's click order, its after-Step settle, its pickup key, the cheat `#pickup all`, the headless `pick_item` step and `Mission.PickUpUnderfoot` with `cmd/missionrun`. `pkg/game` keeps only presentation: it turns the transaction's result into the ordinary pickup lines. Nothing changes for a player in either game.

## Authority

- Owner direction for audit-20 wave 4: one builder per kind; the first game is a reduced second game, its difference profile data; architecture work changes no player behaviour, hashed state or SAV byte.
- The audit-20 plan, rank R17 (the sack pickup lifecycle): the instances, the census and the builder.
- The rules the moved code already applied and still cites: `ITEM-PICK-016` (the order is admitted for a sack at the cell and a living taker on the map), `AI-356` (the order rides beside physical progress), `AI-645` (a sack gone on arrival completes without a transfer), `MENU-108` and `R2-ENGINE-301` (`#pickup all`), `SAV-1114` (the reached stacks the lines name), `AI-CMD-032` and `AI-STATE-011` (a later order overwrites a standing pickup), `DIV-292` (the key acts for the window's subject).

## As-built behaviour

The transaction's entries:
- `World.OrderSackPickup(taker, cell)` answers the `PickUp` command when a sack stands at the cell and the taker is alive and on the map. The window queues it through `queueGroup`, so a later order for the same entity still overwrites it in the World.
- `World.PickUpSackUnderfoot(taker, docs)` is the key, the `pick_item` step and `Mission.PickUpUnderfoot`. A taker mid-strike gets `SackPickupOrder` with the pickup order for its own cell, which the window queues; otherwise a taker that has arrived takes the sack on its cell.
- `World.SettleSackPickup(docs)` is the after-Step settle, one call the window makes after each Step, as before. It settles the first actor in entity order holding a pending pickup: a dead or off-map taker's order is cancelled; on arrival at the order's cell a sack gone completes the pickup without a transfer, otherwise the sack is taken and the order ends.
- `World.CheatPickupAll(hero, docs, told)` takes every ground sack, first to last, as the hero's pickup, and `told` hears each take.

Each take is `pickUpSack`: refuse a sack holding quest documents when no holder can carry them, then `TakeSack`, `MoveCarried` of the documents to the starting hero, `CompleteSackPickup`. It returns `SackPickup`: the taker, the sack, each holder's stacks before and after, and the local player's purse before and after. The window's `announcePickup` turns that into `pickupLinesForTake` and `announce`, unchanged.

`SackDocuments` carries the quest document code and the starting hero. `pkg/game.questDocuments` builds it from `startingHero`, the one lookup over the party; `startingHeroIndex` is the one loop over `primaryPlayerHero`, also used by `HeroAudience` and the speaker direction.

Removed from `pkg/game`: the latch `mapWorld.pickup` (`pickupIntent`), `cancelPickup` and its five calls, `takeSackFor`, `takeSackUnderfoot`, `takeSack`, `announceTake`, the window's arrival test `sackPickupArrived`, and the scenario's own starting-hero loop. `CancelSackPickup` became the transaction's `cancelSackPickup`; `CheatPickupAll`'s old walk is `cheatPickupEach`. The arrival test is `World.SackPickupArrived` for tests; in production only the transaction calls it.

`cmd/missionrun`'s `pickUpAt` keeps its own walk stop (crossing done and the cell's centre). The arrival test also waits for the strike cycle and the order progress, so calling it there would change how many ticks the tool spends and what it prints. Its take is `Mission.PickUpUnderfoot`, which applies the arrival test.

`internal/archtest/sackpickup.go` guards the kind. Outside `pkg/sim`, no production function:
- calls `TakeSack`, `MoveCarried`, `CompleteSackPickup`, `CancelSackPickup` or the arrival test;
- declares a `takeSack…` or `completeSackPickup…` function or a `sackPickupArrived`;
- builds `sim.PickUp`;
- writes the sack list through `ReplaceGroundSacks`;
- tests arrival by hand (an `ActorFinePosition` read, a comparison with 128 and a `Transit` read);
- names a `sim.SackDocuments` outside `questDocuments`;
- calls `primaryPlayerHero` in a loop outside `startingHeroIndex`.

Inside `pkg/sim`, each internal is called only by its transaction functions in `sackpickup.go`, and exactly one `pickUpSack` exists. The named entries are `applyOriginalGround`'s ground import and `pickUpAt`'s walk stop.

## Proof

- `TestSackPickupRecord` (`pkg/game/sackpickuprecord_test.go`, golden `pkg/game/testdata/sackpickup_record.txt`, 2927 lines) was committed alone at `cc6781129` on base `e8bc8f34c`. It is equal on the candidate. On synthetic fixtures and for each game's World profile, it records each tick's World hash and byte-form digest, each actor's order, state and pack, the purse, the sacks, the window's queue and the posted lines. It covers 22 scenarios: the click order (with quest documents, refused documents, an owner past the relation slots, no sack), a sack gone before arrival, a later order, a LOAD with a pending pickup, two pending pickups (one blocked), the key underfoot, behind queued orders, mid-strike, mid-crossing and refused, the cheat (with no starting hero and refused), the `pick_item` step and refused, and `Mission.PickUpUnderfoot` and refused.
- Planted failure: with the documents' move skipped in `pickUpSack`, the record differs at line 153 (`first click-documents`, tick 32 hash `63ebfe3aadc95469` recorded, `06bb488746759e6f` now).
- `TestCheckSackPickupNamesEachShape` plants one instance per shape (19 cases, including the clean forms and both named entries). `TestSackPickupHasOneBuilder` scans the live `pkg` and `cmd` trees.
- The census before and after (`git grep` over production files) is in the lane's notes. The pickup's internals are now called only in `pkg/sim/sackpickup.go`. `func (mw *mapWorld) takeSack` is gone (1 → 0). `sackPickupArrived` has no caller outside `pkg/sim`. `primaryPlayerHero` sits in one loop.

## Open debt

- `DIV-337` in `docs/divergences/client-presentation-other.md` still names `mapWorld.pickup` and `cancelPickup` as the driver-tier latch. The pickup state is now the World's pending order alone. The row's text needs the owner's edit; this story adds no DIV row. The `pickup` row of `docs/0143-save-and-load/spec.md` FR-5 likewise rules a field `mapWorld` no longer has; `TestEveryMapWorldFieldIsRuled` no longer lists it.
- `TakeSack`, `CompleteSackPickup` and `MoveCarried` stay exported because tests use them as primitives. The guard, not the type system, keeps production callers to the transaction.
- The after-Step settle stays a driver call after `Step`; moving it into `Step` was not tried.
- `cmd/missionrun`'s walk stop remains a named entry (see above).
