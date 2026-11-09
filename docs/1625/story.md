# Story 1625: one party item home

## Intent

One home holds what a campaign party carries between missions and answers every crossing. The second
game's bag and a member's pack are views of it. The games differ only in profile data: the first game
has no bag and is member packs only. No item moves differently; no SAV byte, RNG step, World hash,
frame, hit area or error text changes on any input.

## Authority

Owner direction on architecture audit `5d474c65`, kind R2 (town party and its item home), plan row 4
of `review/audit-20/PLAN.md`; earlier owner item on audit `a5c60f20`: the party's items live in one
place, first game a pocket per member, second game the shared bag, following the campaign.

## As-built behaviour

- `pkg/game/partyitemhome.go` is the home. `partyItemHome{sess, table, profile}` is a view over a
  session; `partyItemProfile` is the game's row: `bagUnits` (0 is no bag), the forms `kept`, `mission`,
  `shopUnit`, `bagForm`. `secondPartyItems` is the second game; `firstPartyItems` is the zero row.
- Operations: `leaveLocation`, `arriveTown`, `takeLoadedPack` (a town LOAD, a profile operation over
  the file's party), `shopView` (read as the shop shows), `writeShopView` (write as the shop leaves),
  `writePack` (the first game's shop write, units as they are), `commitPack` (the one point a
  crossing's pack replaces the party's and the town's item graph follows). `followHeroPack` has one
  production caller, `commitPack`. `memberWithPack`, `secondHeroBag`, `secondBagUnitForm` and
  `secondMissionItem` moved to the home with their code unchanged; `savedBagView` is the bag a town
  SAVE writes, read from the saved party.
- Row selection: `CampaignSession.partyItems` takes the second row when the session holds the second
  campaign's record (the bag's only store), else the first; it is the one place a row is picked
  from a session. A second-game shop row and a loaded second-game file name `secondPartyItems`
  directly, as they did before the file's move: they are the second game's own routes.
- The bag's store stays in `secondCampaign.party[0].CarriedItems` and the SAV shape is unchanged:
  `captureSecondParty`, `validateSecondParty`, `restoreSecondParty`, `adoptLegacyHero`,
  `maxSecondBagUnits` and `writeSecondTownBag` keep their places.
- The first game: no crossing between town and mission reaches any home function. Departure is
  `StartMission` with the party; return is `currentMissionCityParty` in `campaignreturn.go`. The
  first game's shop write (`setShopPackItemInstancesRefresh` without a `packWrite` row) is the one
  first-game route through the home, `writePack`, the same `memberWithPack` call as before.
  `leftLocation` runs in both games from `finishWin` and moves nothing where no bag exists.
- Left apart: the mission Shop commit (`commitMissionDeal`, `shopDealDiff`) writes the live World's
  pack, which no campaign store holds; `settleSecondTable` and `leaveSecondShop` end a deal into the
  bag's store through `heroBag`; `bankShopRow`'s `bagUnit` and `setShopPackItemInstancesRefresh`'s bag
  write stay, as the brief keeps the shop row's form unchanged.

## Proof

- `pkg/game/partyitemcrossings_test.go` runs each crossing over a synthetic fixture and compares
  `testdata/partyitemcrossings.golden`, numbers only, recorded at the base commit `78a23278b` with the
  test added at `7cb792764`. Crossings: bag leave, arrival with and without a bag form, arrival over
  the bound, a graph that cannot follow, a town LOAD, the shop's chosen member read and write, a table
  settle, a mission Shop commit, a SAVE of the bag, the first game's pack writes. A planted fault
  (the bound `<` as `<=`) fails it. The record is equal at the candidate.
- `internal/archtest` `CheckPartyItemHome`: `secondshop.KeptUnit`, `secondshop.MissionUnit` and
  `memberWithPack(` appear in production only in the home's file; `followHeroPack(` appears there as
  one declaration and one call.
- Census at `pkg` and `cmd`, production lines (base to candidate): `followHeroPack(` 3 to 2,
  `leaveBagToCampaign(` 2 to 0, `packBag(` 3 to 0, `memberUnits(` 2 to 0, `setMemberUnits(` 2 to 0,
  `secondshop.KeptUnit(` 3 to 0 (a profile value in the home), `secondshop.MissionUnit(` 3 to 0 (a
  profile value in the home), `memberWithPack(` 6 outside to 0 outside; `secondHeroBag(`, in the
  home; `commitMissionDeal(` 3, `settleSecondTable(` 3 and `writeSecondTownBag(` 2 unchanged.

## Open debt

- `partyItems` picks the row by the second campaign's record being present in the session, not by
  game identity: an unset record picks the first row, as every operation before answered nothing on
  it. A second-game session always holds the record.
- The first game's shop still reads its pack through `shopPackItemInstances` and its town graph
  through `cityGraph`: its row has no forms to convert, so those routes stay in the shop code.
