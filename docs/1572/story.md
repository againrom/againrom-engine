# A mod mission chosen by the starting hero

## Intent

A mod can give one win several guest missions, each with its own number, map and texts, and the
starting hero the player made chooses which one opens. A `[[mission]]` table of
`data/missions.toml` names the heroes it plays for with two optional keys:

```toml
[[mission]]
key     = "mage_female"
number  = 28
map     = "maps/mage_female.alm"
after   = 20
guest   = 22
chapter = 30
class   = "mage"     # or "fighter"; absent: either class
sex     = "female"   # or "male"; absent: either sex
```

A mission with neither key plays for every hero. Two guest missions after one win whose keys
admit one hero are refused at load, naming both lines and the hero they share; missions whose
heroes do not overlap load. A hero no mission admits goes on to the town. The keys are
documented in `docs/MODS.md`, "Missions by starting hero".

The second game's inn mission talks that the original picks by the hero flags, slots 776 (mage)
and 781 (female), go through the same chooser: the NPC record 2022 families 74..77, 84..87 and
93..96, and topics 81/82. What the ROM2 player sees does not change, so the CHANGELOG has no
`[ROM2]` line.

## Authority

- Owner direction (board, «как рекомендовано»): the owner's mod plans two maps of one mission, one
  for each starting hero; one choice by the starting hero's class and sex serves the mod and the
  second game. The `data/missions.toml` keys are Againrom's own format.
- R2-ENGINE-283 (High, static route): Accept commits the hero with SetVar(776, mage) and
  SetVar(781, female).
- R2-ENGINE-217 (High, promoted): three NPC2022 families with no stage bound, one record of each
  chosen by 776/781 in the order zero/zero, zero/nonzero, nonzero/nonzero, nonzero/zero.
- R2-ENGINE-424 (Medium): the same families 74..77, 84..87, 93..96 in town 2, and 81 when 776 is
  nonzero, 82 when it is zero, in inn 80 of town 2.
- R2-ENGINE-419 (High): the campaign restoration appends one of 93..96 by the same selection
  (95 both, 96 776 only, 94 781 only, 93 neither). The restoration is not built (DIV-2392); its
  route is unreachable from a fresh campaign (R2-ENGINE-425).
- R2-ENGINE-749 (High / Medium): 776 and 781 have writers only in SetVar, NewGame and Load, so the
  slots hold the committed hero for the whole campaign.
- HERO-JOIN-120: the composed archetype selector, female +1, mage +2.

## As built

| kind | builder |
|---|---|
| chooser by the starting hero | new leaf `pkg/herokind`: `Kind` (`Of(mage, female)`, female +1, mage +2), `Set` (`Parse(class, sex)`, `Named`, `Holds`, `Overlaps`, `String`), `Choose(kind, alternatives, default)` and `Overlap(alternatives)`; `internal/archtest/dag.go` allows it under `pkg/data`, `pkg/mapload` and `pkg/mod`; `pkg/town`, which imports only `pkg/screenart` and the standard library, carries the words and the slots and reads neither |
| hero class and sex reading (party) | `pkg/mapload/placedhero.go`: `StartingHero` (was `partyPrimary`) and `StartingHeroKind`, the one reading of the party's starting hero; `placedHeroName` takes its person line as `21 + herokind.Of(...)`; `data.CampaignServerID` composes its selector with `herokind.Of` |
| mod mission list and its checks | `pkg/mod/missions.go`: `Mission.Class`, `Mission.Sex`, `Heroes`; `ParseMissions` reads `class` and `sex` (guest missions only; a replacing mission refuses them) and refuses overlapping heroes after one win within a file; `MergeMissions` → `checkChains` refuses them across mods; `AfterWin(n, kind)` chooses through `herokind.Choose`; `Follows(n)` |
| mission record writer | `pkg/mod/missionwrite.go` `missionPairs` writes `class` and `sex` when set; the map editor's `selected` clears them for a replacing mission |
| campaign next-mission (first game, mod) | `pkg/game/modmission.go` `startGuestMission` and `finishGuestMission` call `AfterWin` with `mapload.StartingHeroKind`; the waiting party's hero (`guestHero`) is kept whenever any link follows, so a later link is chosen by the waiting hero |
| second campaign inn rows | the catalog's one row type `town.OfferRow` gains `class` and `sex`; `town.Catalog.HeroSlots` (`hero-slots`) names the bank slots, and `Catalog.Validate` refuses a tavern row with a hero, a row with a hero and no slots, and negative or shared slots; `innState.admits` (`pkg/game/innoffers.go`) admits a row whose `herokind.Named` set holds the hero; `secondCampaign.innOptions` reads the hero through `bankHero`, which composes with `herokind.Of`; `checkOfferRows` (was `checkOfferBank`) also refuses a word `herokind.Parse` refuses, for layer 0 and for a mod's merged rows; `pkg/game/towncatalogs/rom2.json` moves 776/781 out of the 14 rows' `zero`/`nonzero` gates into `class`/`sex` and names `"hero-slots": {"mage": 776, "female": 781}` |
| mod layer over an inn row with a hero | `layerOffers` (`pkg/game/taverncatalog.go`), the one merge of a mod's offer records, calls `heroAsGates` (`pkg/game/innoffers.go`) before each record's `Apply`: when the record names `zero` or `nonzero` and the row names `class` or `sex`, it appends the catalog's `hero-slots` slots to the row's `zero` (fighter, male) and `nonzero` (mage, female) and clears `class` and `sex`; the record then replaces the lists it names, as at the base. A record that names neither list leaves `class` and `sex`. No slot number is in code; the offer decision stays in `innState.admits` |
| SAV of the campaign position | unchanged: the guest record `currentGuest` already names the mission by mod, key and number, so each branch, being its own mission, round-trips with no new field; the second game's slots are bank slots the SAV already carries |

The second game reads the hero from the bank, not from the party, because the original's records
read the slots and the slots are the campaign state a SAV carries; `bankHero` and
`StartingHeroKind` both end in `herokind.Of`, and the choice itself is `Set.Holds`, the predicate
`Choose` takes.

## Proof

Focused ordinary tests; each new test file fails to build at `a92003639` (`herokind`,
`StartingHeroKind`, `HeroSlots`, `Mission.Class` and the two-argument `AfterWin` do not exist
there).

- `pkg/herokind/herokind_test.go`: the selector order, `Parse` over every word pair and its
  refusals, `Choose` over the four alternatives and a subset with a default, `Overlap`.
- `pkg/mod/missionheroes_test.go`: three missions after one win (fighter, male mage, female mage)
  choose by hero and merge; a mage-only mission leaves a fighter none; overlaps within a file are
  refused naming both lines and the shared hero; across two mods the refusal names both mods'
  files and lines; missions that do not overlap merge; bad words and replacing missions are
  refused; the writer keeps both keys.
- `pkg/game/modheromission_test.go` over the fixture mod `pkg/game/testdata/hero-mission`
  (three synthetic maps written by the test, no install bytes), read through `mod.Resolve` and
  `modrt.Load` and set with `SetModMissions`: each of the four heroes opens its own mission and
  that mission's map is served at its number; a mission SAV's guest record and the town SAV's
  record after the win (the waiting guest) cross JSON and `restoreGuest` into the same mission
  and map; the saved waiting hero chooses as before the LOAD; an overlap in the fixture is refused
  by the loader.
- `pkg/game/heroinn_test.go`: the catalog's hero slots equal the generator's commit slots
  (776, 781); each 2022 family's four rows do not overlap; for seven 776/781 values (including
  -1, -2, 3 and 7) `herokind.Choose` takes and `innOptions` offers the R2-ENGINE-217 record; 81
  and 82 split by class alone; `TestInnRowHeroesAreReadByTheChooser`: `bankHero` over four banks
  and no slots, `admits` by hero, and `checkOfferRows` refusing a bad word by the row's key. The
  existing `TestSecondInnStageGates` and `TestSecondInnContinuationGates` pass unchanged on the
  converted rows.
- `pkg/game/heroinnmod_test.go` `TestModGateOnAHeroInnRowReplacesItsSlotGate`, through
  `writeInnMod` → `runTownModsFor` → `layerTownBook` → `secondCampaign.innOptions`, for seven
  776/781 values (0/0, 1/0, 0/1, 1/1, -1/7, 2/0, 0/-2): a mod that sets `zero = [978]` on topic 82
  and `zero = [970, 971, 972, 973]` on topic 74 offers the base's lists (inn 80: [82], [81 82],
  [82], [81 82]; family 2022: [74], [74 77], [74 75], [74 76] for the first four values); a mod
  that sets only `zero` on topic 75 keeps 781 in its `nonzero` ([74], [77], [75], [75 76]); a mod
  that names neither list on topic 76 keeps its `class` and `sex`. It fails at `d425740d6` (12
  wrong lists) and passes unchanged on a `git archive 80d409b65` extract, the base, whose rows
  gate on 776/781 in `zero` and `nonzero`.
- `pkg/town/heroslots_test.go`: rows decode `class` and `sex` and `Equal` reads them; refusals of
  a tavern row with a hero, a row with a hero and no slots, and negative or shared slots.
- `pkg/mapload/startinghero_test.go`: the marked member, else the first, over four figures.
- `pkg/modrt/mergeddata_test.go` reads mission records without the two new fields, so the
  recorded digests of the fixture mods, which name no hero, stay.

Release test, not run in the lane: `TestReleaseModMissionAfterAWinIsTheStartingHerosMap`
(`pkg/game/modheromission_release_test.go`, row in `internal/gatedtests/testdata/population.txt`)
plays missions 10 and 20 with each of the four generated heroes on the first game's install, opens
the chosen mission, checks its built map by an altitude byte per branch, and SAVEs and cold-LOADs
inside it.

## Open debt

- A mod's `[[inn]]` table cannot state `class` or `sex`; a row that names neither `zero` nor
  `nonzero` keeps a layer 0 row's, and one that names either turns them back into slot gates.
- A replacing mission plays for every starting hero; a per-hero replacement of a campaign map is
  not offered.
- The map editor's mission form shows neither key; it keeps them on save and drops them when a
  mission becomes a replacing one. Whether its test play takes the chosen branch for the hero it
  plays with is not checked.
- The second game's campaign restoration (R2-ENGINE-419), which would append one of 93..96, is not
  built (DIV-2392); built, it takes the catalog's hero slots and the same chooser.
- DIV-4409 to DIV-4411 are unused: no second-game mismatch remains in this result.
