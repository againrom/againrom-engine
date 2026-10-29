# One inn offers builder for both games

## Intent

The inn roster, hire and talk rows of both games come from one builder over
town catalog data. A second-game mod edits an inn talk the way a first-game
mod edits a tavern talk. Frames, World state and SAV bytes of both games stay
equal.

## Authority

- Owner rulings in the story brief: one builder per kind; the first game is a
  reduced second game, so the rows take the second game's shape; no costly
  test, one cheap `internal/archtest` guard; no branch on game identity in the
  builder.
- Architecture audit `3d1771a7`, row 4: the inn roster, hire and talk offers
  were built two ways. The first game read `town.TavernRow` catalog data. The
  second game read the Go literal tables `secondInnStages` and
  `secondInnShared`, so a mod could not touch an inn talk.
- The admission rules are the second game's EnterInn rules as the base
  encoded them (R2-ENGINE-161, R2-ENGINE-215 to R2-ENGINE-217, R2-ENGINE-221,
  R2-ENGINE-223 to R2-ENGINE-228). The mercenary rule keeps TAVERN-ORDER-015
  and DIV-1410. No claim was re-read for new behaviour: the rows restate the
  base's tables one for one.

## As built

Kinds and their one builder:

| Kind | Builder |
| --- | --- |
| Inn offer rows (roster, talk admission), both games | `innOffers` in `pkg/game/innoffers.go` over `[]town.OfferRow` and an `innState` |
| Second-game hire options | `innHires` in `pkg/game/innoffers.go`, called only from `innOffers` |
| Offer row type, both games | `town.OfferRow` (`pkg/town/inn.go`) in `town.Catalog.Offers`, its table named by `Catalog.OfferTable` (`tavern` or `inn`); shipped in `pkg/game/towncatalogs/rom2.json` (`inn`); `rom1.json` names `tavern` and states no rows |
| Offer row validator, both games | `Catalog.validateOffers` (bounds and keys) and `offerRules.check` in `pkg/game/taverncatalog.go` (missions, towns, NPC records, talk texts) |
| A first-game tavern row of a chapter | `town.ChapterOffer`; layer 0 `scenarioTavern`, admitted by `innOffers` from `Campaign.chapter` |
| Mercenary stock walk and price rule | `town.Catalog.Mercenaries` (`town.MercenaryRule`), shipped in `rom1.json`, read as `tavernRule` |
| Mod data, both tables | `[[tavern]]` and `[[inn]]` tables in `data/towns.toml`, parsed by `offerRecordFrom` into `mod.OfferRecord`, layered by `layerOffers` through `mod.Merge`, reached through `FrontEnd.SetModTowns` |
| Tavern or inn page, town screen host, talk box | unchanged |

- `town.OfferRow` is {kind, topic, npc, town, stages, after, zero, nonzero,
  one, any}. Kind 0 is a talk, kind 3 a mission talk. Town 0 admits every
  town. `after` is a stage floor. `zero`, `nonzero` and `one` gate on bank
  slots; `any` asks one of its slots to be nonzero. The catalog validates kind,
  ranges, town, unique key and non-negative slots; `pkg/game` refuses a slot
  outside the 1024-slot bank.
- A tavern row is the reduced shape: one town of the catalog, one stage (its
  chapter), its mission the topic, kind 3 exactly when it gives a mission, no
  stage floor or bank gate, and a chapter names an NPC record once. Its key
  is (town, chapter, npc); an inn row's is (topic, npc). `[[tavern]]` keeps
  its keys `town`, `chapter`, `npc`, `mission`, `off`.
- A row a mod adds or changes is judged by its catalog's table: a tavern row
  by the scenario's main missions and mission sections, the towns offering
  them, the install's NPC table and `main/text/inn/npc/`; an inn row by the
  world map's mission records, the same NPC table and the section
  `npc<npc>talk<topic>` of `main/text/town.txt`. Each refusal names the mod,
  file, line and row.
- `SetModTowns` takes the profile's book. It refuses a missing scenario only
  for a catalog that takes town layers (the first game's); the second game,
  whose install holds no `scenario/scenario.reg`, layers its inn rows and
  keeps the campaign's read error as it was.
- The 46 second-game rows replace the two literal tables and the
  continuation, town 2 family and topic 62 branches of the base, in the base's
  order. The second campaign binds the rows of a layered book when a mod
  layers one, and the shipped rows otherwise.
- The first game's chapter rows reach `innOffers` with `every` set, so town
  and bank gates do not apply; its roster predicates (`tavernHere`,
  `tavernVisibleOffers`) are unchanged.
- `internal/archtest/innoffers.go` (`CheckInnOffers`, live test
  `TestInnOffersHaveOneBuilder`): no production composite literal whose
  elements are `innOffer` or `town.OfferRow` outside `innOffers`, `innHires`
  and `town.ChapterOffer`, and no type named `TavernRow` or `InnRow` under
  `pkg/`.
- `docs/MODS.md` names the `[[inn]]` keys and refusals; the CHANGELOG has one
  `[ROM2]` line.

## Proof

Base `05289791cc3d281a8ed0dbd59e132d06aab6d593`, head of this branch. Both
games were captured headlessly with `againrom --headless` from a base build
(`git archive` of the base, `scripts/build-tools.sh`) and a head build. The
table was measured again on the head of the review correction, with the
first-game `tavern-rows` fixture mod (`pkg/modrt/testdata/mods`) added on
EN and RU.

- First game EN and RU: kit save 9714 (chapter 60), tavern candidates, a
  selected candidate (Mercenary 10), a hire (EN; purse 5000 to 2080), the
  tavern after a reload of that SAV, a talk (NPC 25) and the page after it;
  SAV after the hire (EN) and after the talk. The RU root loads the EN kit
  save; its Hire and Exit labels are CP866 bytes a JSON scenario cannot name,
  so RU has no headless hire.
- Second game EN and RU: owner save copy `game0002.sav` (stage 10, town 1)
  for the stage 10 rows and a talk (TALK 207); a copy of `game0000.sav`
  edited in the notes directory to stage 30, town 2, hire 1 available, for the
  stage 30 rows and a hire (HIRE 1; bank slot 512 becomes 1 in the written
  SAV); SAV after the talk and after the hire.
- First game with `tavern-rows` (EN and RU, `-mods-accept-unmarked`): the
  same steps. The mod changes chapter 50; a LOAD keeps the save's chapter 60
  offers, so the frames equal the frames without the mod; the SAV bytes
  differ from the run without the mod, which writes no mod mark.
- A town save holds no live `sim.World`; the headless runner prints no World
  hash. The SAV bytes carry the campaign state the tavern and inn write.

46 of 46 outputs are byte-equal:

| State | Root | Base SHA-256 | Head SHA-256 | Equal |
| --- | --- | --- | --- | --- |
| rom1 saves/after hire.sav | en | 343338d99f3e80c299774ccebbe96222e5eb95cf34f41e1163199e28d3bc4f61 | 343338d99f3e80c299774ccebbe96222e5eb95cf34f41e1163199e28d3bc4f61 | yes |
| rom1 saves/after talk.sav | en | 0192308a86d97d7a38c8e5e905bad6f3be9285a96cef968e18b7038856d6243d | 0192308a86d97d7a38c8e5e905bad6f3be9285a96cef968e18b7038856d6243d | yes |
| rom1 tavern-after-talk.png | en | be7d22915c2bb7dba07c76964e614b673bfd3949f389d688f6c9f280ea902641 | be7d22915c2bb7dba07c76964e614b673bfd3949f389d688f6c9f280ea902641 | yes |
| rom1 tavern-hire.png | en | 88a9a0c3c9dfcae0b014e862f644196725c5ca69233bd7d9a79bab05797f3411 | 88a9a0c3c9dfcae0b014e862f644196725c5ca69233bd7d9a79bab05797f3411 | yes |
| rom1 tavern-reloaded.png | en | 0781fd1df0a5a8d2711002951b3ec137338c466c9ec4b19e49eb5fbc32de3ce6 | 0781fd1df0a5a8d2711002951b3ec137338c466c9ec4b19e49eb5fbc32de3ce6 | yes |
| rom1 tavern-selected.png | en | c7f1c62fd9f837d09f8685aa9103509749fe99294d7bb34a8176e9aea35dff62 | c7f1c62fd9f837d09f8685aa9103509749fe99294d7bb34a8176e9aea35dff62 | yes |
| rom1 tavern-talk.png | en | 7e0aebabca6bc8645390347745194e336509f66aeab6984d10db03efb9670294 | 7e0aebabca6bc8645390347745194e336509f66aeab6984d10db03efb9670294 | yes |
| rom1 tavern.png | en | 4f776cf2ee66cc48d309badb764f5a7804bcf903855e32704772a07c059129e3 | 4f776cf2ee66cc48d309badb764f5a7804bcf903855e32704772a07c059129e3 | yes |
| rom1 saves/after talk.sav | ru | 96a4eceac2196ed27906ab8697baad278d496b4e8b853069de894b112f7c7338 | 96a4eceac2196ed27906ab8697baad278d496b4e8b853069de894b112f7c7338 | yes |
| rom1 tavern-after-talk.png | ru | 94d5cd63ebb4105e3e86723f664933d69e6493349ff11eed7010b47b26438856 | 94d5cd63ebb4105e3e86723f664933d69e6493349ff11eed7010b47b26438856 | yes |
| rom1 tavern-selected.png | ru | 6ca6facf9765431b7890c20f11fdf81f37be47c8ff7f95f1f3af053065228b40 | 6ca6facf9765431b7890c20f11fdf81f37be47c8ff7f95f1f3af053065228b40 | yes |
| rom1 tavern-talk.png | ru | acb89c2c46ba7b9486cf69923796935da2bf4006dc01e5673208dcb650ba6893 | acb89c2c46ba7b9486cf69923796935da2bf4006dc01e5673208dcb650ba6893 | yes |
| rom1 tavern.png | ru | 2da8cef2dd0704b67df791a2c4661ece9fcbb3901655a35fa455886e427c9e85 | 2da8cef2dd0704b67df791a2c4661ece9fcbb3901655a35fa455886e427c9e85 | yes |
| rom1 tavern-rows saves/after hire.sav | en | 90e1449b1ab35e41dcc900d0b76205c8e022574d7a775adf816577c3e84c7f74 | 90e1449b1ab35e41dcc900d0b76205c8e022574d7a775adf816577c3e84c7f74 | yes |
| rom1 tavern-rows saves/after talk.sav | en | c00acdad09d39d990e6f89130204b1d326e5fa65dba8cc1fb93ad7a7d555e6a2 | c00acdad09d39d990e6f89130204b1d326e5fa65dba8cc1fb93ad7a7d555e6a2 | yes |
| rom1 tavern-rows tavern-after-talk.png | en | be7d22915c2bb7dba07c76964e614b673bfd3949f389d688f6c9f280ea902641 | be7d22915c2bb7dba07c76964e614b673bfd3949f389d688f6c9f280ea902641 | yes |
| rom1 tavern-rows tavern-hire.png | en | 88a9a0c3c9dfcae0b014e862f644196725c5ca69233bd7d9a79bab05797f3411 | 88a9a0c3c9dfcae0b014e862f644196725c5ca69233bd7d9a79bab05797f3411 | yes |
| rom1 tavern-rows tavern-reloaded.png | en | 0781fd1df0a5a8d2711002951b3ec137338c466c9ec4b19e49eb5fbc32de3ce6 | 0781fd1df0a5a8d2711002951b3ec137338c466c9ec4b19e49eb5fbc32de3ce6 | yes |
| rom1 tavern-rows tavern-selected.png | en | c7f1c62fd9f837d09f8685aa9103509749fe99294d7bb34a8176e9aea35dff62 | c7f1c62fd9f837d09f8685aa9103509749fe99294d7bb34a8176e9aea35dff62 | yes |
| rom1 tavern-rows tavern-talk.png | en | 7e0aebabca6bc8645390347745194e336509f66aeab6984d10db03efb9670294 | 7e0aebabca6bc8645390347745194e336509f66aeab6984d10db03efb9670294 | yes |
| rom1 tavern-rows tavern.png | en | 4f776cf2ee66cc48d309badb764f5a7804bcf903855e32704772a07c059129e3 | 4f776cf2ee66cc48d309badb764f5a7804bcf903855e32704772a07c059129e3 | yes |
| rom1 tavern-rows saves/after talk.sav | ru | b4df91d4d980c79002c680ec2cd26d649a01650cf9bd74b57631575a0f976787 | b4df91d4d980c79002c680ec2cd26d649a01650cf9bd74b57631575a0f976787 | yes |
| rom1 tavern-rows tavern-after-talk.png | ru | 94d5cd63ebb4105e3e86723f664933d69e6493349ff11eed7010b47b26438856 | 94d5cd63ebb4105e3e86723f664933d69e6493349ff11eed7010b47b26438856 | yes |
| rom1 tavern-rows tavern-selected.png | ru | 6ca6facf9765431b7890c20f11fdf81f37be47c8ff7f95f1f3af053065228b40 | 6ca6facf9765431b7890c20f11fdf81f37be47c8ff7f95f1f3af053065228b40 | yes |
| rom1 tavern-rows tavern-talk.png | ru | acb89c2c46ba7b9486cf69923796935da2bf4006dc01e5673208dcb650ba6893 | acb89c2c46ba7b9486cf69923796935da2bf4006dc01e5673208dcb650ba6893 | yes |
| rom1 tavern-rows tavern.png | ru | 2da8cef2dd0704b67df791a2c4661ece9fcbb3901655a35fa455886e427c9e85 | 2da8cef2dd0704b67df791a2c4661ece9fcbb3901655a35fa455886e427c9e85 | yes |
| rom2 inn-after-hire.png | rom2-en | 7f5bbde92ef3a9c17a6de37850371578710956be1f9e514d5cf2504773756b5d | 7f5bbde92ef3a9c17a6de37850371578710956be1f9e514d5cf2504773756b5d | yes |
| rom2 inn-after-talk.png | rom2-en | d2030a8267550420077be2f902fa44fe748bd607c826708ed4745dd78d78d597 | d2030a8267550420077be2f902fa44fe748bd607c826708ed4745dd78d78d597 | yes |
| rom2 inn-hire.png | rom2-en | 619daeb54d2c8cba2cdf52d28286ea966f18a37745b9eb242df9969640e4f154 | 619daeb54d2c8cba2cdf52d28286ea966f18a37745b9eb242df9969640e4f154 | yes |
| rom2 inn-stage10.png | rom2-en | 4a4b8c3d01dc1123bfd51161d31300a8a98c777ad37e8c136fbfc44fbf9b5a4f | 4a4b8c3d01dc1123bfd51161d31300a8a98c777ad37e8c136fbfc44fbf9b5a4f | yes |
| rom2 inn-stage30.png | rom2-en | ab45dfc3e8ae0b4af6308acc171fa6e7e3fc8cc5aad903d803e10ea57cdff118 | ab45dfc3e8ae0b4af6308acc171fa6e7e3fc8cc5aad903d803e10ea57cdff118 | yes |
| rom2 inn-talk.png | rom2-en | b777d7dd9c134e078970300f92684e6923a016d49b8498f05d74815c60f31d61 | b777d7dd9c134e078970300f92684e6923a016d49b8498f05d74815c60f31d61 | yes |
| rom2 saves/game0000.sav | rom2-en | 070a9c3d68bc455b17bc2c353ffe93d45581b7d30c7dddf31b2b88b494841ad9 | 070a9c3d68bc455b17bc2c353ffe93d45581b7d30c7dddf31b2b88b494841ad9 | yes |
| rom2 saves/game0001.sav | rom2-en | 0b10acf651c289189991f10ee379bb0ea1146bc6fbf5a701bb527c92977337f0 | 0b10acf651c289189991f10ee379bb0ea1146bc6fbf5a701bb527c92977337f0 | yes |
| rom2 saves/game0002.sav | rom2-en | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | yes |
| rom2 saves/game0003.sav | rom2-en | aefad0ab526ae4c85967b869b8785a782f499b6e684bee9305bbdd2daa06f6ff | aefad0ab526ae4c85967b869b8785a782f499b6e684bee9305bbdd2daa06f6ff | yes |
| rom2 inn-after-hire.png | rom2-ru | 42a2260afd4b331ed13a34f26cb1e6ed5cd847872aeee0574b027832d7b531d2 | 42a2260afd4b331ed13a34f26cb1e6ed5cd847872aeee0574b027832d7b531d2 | yes |
| rom2 inn-after-talk.png | rom2-ru | a180f04b55c24de4de80b11b972eb9dc558a5618a119a66cf91dec028f7177b5 | a180f04b55c24de4de80b11b972eb9dc558a5618a119a66cf91dec028f7177b5 | yes |
| rom2 inn-hire.png | rom2-ru | 1d1ff8797b95686aeeaef94d68d1421b55a63a80c3bd0b54108ae11df481dc7a | 1d1ff8797b95686aeeaef94d68d1421b55a63a80c3bd0b54108ae11df481dc7a | yes |
| rom2 inn-stage10.png | rom2-ru | 03ec45e3eaa0aebdb069c126f2e5057ad48e2823088522696a2e51da1b2915e0 | 03ec45e3eaa0aebdb069c126f2e5057ad48e2823088522696a2e51da1b2915e0 | yes |
| rom2 inn-stage30.png | rom2-ru | 5e53d6bd70ff89fe99879e91bbe3aff625081592d9038dee1838268af944fea9 | 5e53d6bd70ff89fe99879e91bbe3aff625081592d9038dee1838268af944fea9 | yes |
| rom2 inn-talk.png | rom2-ru | cbf73b16016dd55b03ab4af682174001f2ee8663a41ac79695d22142c402f7f1 | cbf73b16016dd55b03ab4af682174001f2ee8663a41ac79695d22142c402f7f1 | yes |
| rom2 saves/game0000.sav | rom2-ru | 070a9c3d68bc455b17bc2c353ffe93d45581b7d30c7dddf31b2b88b494841ad9 | 070a9c3d68bc455b17bc2c353ffe93d45581b7d30c7dddf31b2b88b494841ad9 | yes |
| rom2 saves/game0001.sav | rom2-ru | 0b10acf651c289189991f10ee379bb0ea1146bc6fbf5a701bb527c92977337f0 | 0b10acf651c289189991f10ee379bb0ea1146bc6fbf5a701bb527c92977337f0 | yes |
| rom2 saves/game0002.sav | rom2-ru | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | yes |
| rom2 saves/game0003.sav | rom2-ru | aefad0ab526ae4c85967b869b8785a782f499b6e684bee9305bbdd2daa06f6ff | aefad0ab526ae4c85967b869b8785a782f499b6e684bee9305bbdd2daa06f6ff | yes |

A second-game mod on the installs, head build only: the base has no
`[[inn]]` table and refuses the mod (`unknown table "inn"`), and the review's
candidate refused it at start (`the install's campaign is unavailable for its
towns: read scenario/scenario.reg`). Fixture mod `innedit`: an `[[inn]]` row
topic 7 npc 520 kind 0 town 2 stage 30 (`npc520talk7` is a section of the
English town text no shipped row names; the Russian text lacks it, so the
mod carries `files/ru/main/text/town.txt`, the Russian text with that
section added) and topic 39 npc 2110 switched off. On the stage 30 copy of
`game0000.sav` in town 2 the inn lists `HIRE 1`, `TALK 22`, `TALK 2108`,
`TALK 520`, `EXIT` on both roots; `TALK 520` opens the English section on
rom2-en and the mod's section on rom2-ru; `TALK 2110` is no row.

| State | Root | Head SHA-256 |
| --- | --- | --- |
| innedit inn-mod-stage30.png | rom2-en | 11dabd9891c44cfc22bc8ed1c968c1d5ace8a118a7e299e6dc2cd0230c3b0385 |
| innedit inn-mod-talk520.png | rom2-en | 26ac9afb22176382b3026fa9d7a010af647edd2e62dc2ee23350a8aa54dc6c43 |
| innedit inn-mod-talk520-part2.png | rom2-en | 0dc303fee5d6dc489a67e9595664873ad93845671b1e19e06adbdd14b54805ad |
| innedit inn-mod-after-talk.png | rom2-en | 2e3f909122a6f1f7d35298499a169678183903d70c3afdb36e7bacc852710539 |
| innedit saves/game0001.sav | rom2-en | e3e829583ae5a18a7b8f67394a628e4b5287e43de192223d3262b6b739ef1d43 |
| innedit inn-mod-stage30.png | rom2-ru | 64c4d26b1767e93a45bfcd858cd63e402f25653caf9d9ae32b4e5d5f5a0578da |
| innedit inn-mod-talk520.png | rom2-ru | 1ea7ae515adeb536b5b26db151c26ee48e4a782606b158f7ca586fa6a6e22732 |
| innedit inn-mod-after-talk.png | rom2-ru | 370537e47048bba20b23763ef5c4b2bd0058ed8ce921cda5967127db3f84fb7c |
| innedit saves/game0001.sav | rom2-ru | 4e2a81781ceddb0fd783bad90af96e65609f5c40d49ef8eb79cd2e2e0cf76d9c |

`againrom -check` on rom2-en and rom2-ru, head build: `innedit` and a mod
changing the shipped mission talk topic 31 npc 2108 load; each mod naming one
missing reference is refused on both roots at `data/towns.toml:1`, with
`inn row topic 7 npc 9999: npc 9999 is not a record of the install's NPC
table`, `inn row topic 98 npc 2110: its talk text section npc2110talk98 of
main/text/town.txt is neither in the install nor served by a loaded mod` and
`inn row topic 999 npc 2110: mission 999 is not a mission record of the world
map`; `innedit` without its Russian town text is refused on rom2-ru alone,
for `npc520talk7`. On en and ru, `tavern-rows` loads with output equal to the
base's, and an `[[inn]]` table is refused with `catalog rom1 takes [[tavern]]
offer rows, not [[inn]] ones`.

Focused tests with synthetic data: `TestInnOffersFollowTheRows`,
`TestSecondInnReadsItsBookRows`, `TestTavernRowsAreInnRows`,
`TestSecondGameModEditsAnInnTalk`, `TestSecondGameModLoadsWithoutAScenario`
(a second-game front end with no scenario takes an `[[inn]]` mod through
`SetModTowns`; the inn lists and opens the added row and drops the one
switched off), `TestSecondGameModRefusesMissingReferences`,
`TestTavernLayerRefusals` (the first game's missing chapter, mission, NPC
record and talk text), `TestShippedMercenaryRule` (`pkg/game`);
`TestInnRowsDecodeAndRefuse`, `TestTavernRowsAreTheReducedShape`,
`TestMercenaryRuleIsData` (`pkg/town`); `TestTownsReadTavernRows`,
`TestTownsReadInnRows`, `TestTownsReadOfferRowsOfBothTables` (`pkg/mod`);
the archtest cases in `internal/archtest/innoffers_test.go`.

Loss control: a `secondInnStages` map literal of `innOffer` rows appended to
`pkg/game/secondinn.go` made `TestInnOffersHaveOneBuilder` fail at that line;
the file was restored from this branch's commit. A `type TavernRow struct`
appended to `pkg/town/inn.go` made it fail at that line ("type TavernRow is a
second offer row type"); the line was removed.

Release tests for the next release: en and ru `tavernrows_release_test.go`,
`tavernroster_release_test.go`, `taverninspection_release_test.go`,
`taverncompleted_release_test.go`, `tavernavailability_release_test.go`;
rom2-en and rom2-ru `secondinnhire_release_test.go`,
`secondinnpatron_release_test.go`.

## Open debt

- The hire producer and the talk opener are still one per game. The first
  game hires a mercenary squad for gold (`tavernMercenaries`,
  `buildMercenarySquad`) and opens a talk through `openOfferDialogue`; the
  second game selects a hire in the bank (`innHires`, `hireOrFire`) and opens
  a talk through `talkTo`. Their state and text sources differ; joining them
  needs its own story.
- The first game's roster predicates stay first-game code over the shared
  rows.
- The second-game base dump writes no `[[inn]]` rows.
- A second-game mod that adds a talk on a Russian install whose town text
  lacks the section carries the whole town text; there is no per-section
  text file.
- The first-game `tavern-rows` rows of chapter 50 are not shown headlessly:
  the kit saves load mid-chapter, and a LOAD keeps the save's offers. The
  release tests below show them.
- Neither the first-game RU hire nor a World hash at the tavern is captured
  headlessly.
