# The engine writes the Druids' town file with Iglez and Dina in the party

## Intent

The second game's Druids' town (town 3) file is written after the engine's own play, with
Gildarius, Iglez (identity 22) and Dina (identity 23) as party members in groups of their own and
every member's pack where the original keeps it. A LOAD of the original's own Druids' town file
keeps each piece where the file holds it.

## Authority

- Owner, first milestone: «Хотя бы Каарг надо уметь писать, а еще лучше еще и Город друидов».
- Owner, in the brief: a field with no source is a DIV row; the town's party items stay in the
  one second-game shared-bag model; a game difference is profile data.
- Claims: `R2-SESSION-293` (town 3 enters the place list through Leave50; arrival stage 60, 70 or
  80), `R2-SESSION-301` (identity 23 first joins at the exit from mission 40), `R2-ENGINE-146`
  and `R2-ENGINE-419` (Leave50 adds town 3, then mission 52 when slot 780 is nonzero),
  `R2-ENGINE-048` (EnterLocation clears slots 752..767), `R2-ENGINE-418` (catalog positions),
  `ITEM-ARMFILL-032` (an armour's +0x48 word is magic capacity).
- Observation: the owner's EN line `gameversions/saves/2026-10-31/rom2-gog-original-3`
  (`game0025`, `game0026` mission 40, `game0029` mission 50 won, `game0030` the original's
  Druids' town).

## As built

| kind | builder | change |
|---|---|---|
| SAV LOAD | `restoreOriginal`, `decodeSavedDocument` (`savdocument.go`), through `leaveBagToCampaign` (`secondparty.go`) | a town LOAD moves only the hero's pack into the campaign's bag; each companion keeps his own |
| town party items | the second-game shared-bag model (`leaveBagToCampaign`, `secondBagUnitForm`) | no new home: a companion's pack is his Human Inventory, as at a town's arrival |
| location leave and campaign tail | `completeBank` (`secondcampaign.go`), `enterLocation` (`secondworldmap.go`), census rows `secondDepartureCases` | Leave50 adds mission 52 after town 3 when slot 780 is nonzero; entering a town from the map clears slots 752..767 |
| party member addition | `PartyIdentities` (`pkg/base/edition.go`), `addPartyIdentity` | unchanged; Leave40's slot 773 = 23 already reaches it |
| SAV document producer | `ExportCurrentSave` through `worldcarriers.go`, `savform.go` | unchanged |

- Companion packs at a town LOAD. Before, `leaveBagToCampaign` emptied every member's pack. On
  `game0030` Iglez's six carried pieces (0x81cd, 0x5149, 0x81cd, 0x514f, 0x81cd, 0x5954) were
  lost: the base's resave held 47 records against the file's 59. Now Iglez keeps them in the
  file's order, the hero keeps the shield 0xb226 and magic item 0x0e4d, which have no bag form
  (DIV-4153), and his other six units are the campaign's bag. The resave's body is 9036 bytes,
  the file's.
- Mission 52. `R2-ENGINE-419` resolves Leave50's slot-780 record as mission 52, so DIV-2629
  closes. The engine's walk now reaches all 46 maps; no map is blocked on continuation.
- Slots 752..767. The engine's Druids' town after the departure held slots 753 and 754 = 3 from
  mission 50; `game0030` holds 0. The town entry clears them, as a mission entry already did.

## Proof

Focused tests (`pkg/game`):

| test | base `db343e5dd` | candidate |
|---|---|---|
| `TestSecondTownFileLoadKeepsEachCompanionsPack` | FAIL: every pack empty | PASS |
| `TestSecondCampaignTwentyDepartureBankAndSelections` | FAIL: the town entry keeps slots 752..767 | PASS |
| `TestSecondDepartureCaseBodies` | FAIL: Leave50 with slot 780 adds town 3 alone | PASS |
| `TestSecondDepartureKeepsEveryExitAndItsGates` | FAIL: mission 50's exit is the unresolved record, a continuation blocker | PASS |
| `TestSecondControllerReachesAndContinuesOnlyItsMissions` (reach includes 52) | not run | PASS |
| `TestSecondLeaveAfterFortyAddsIdentityTwentyThree` (Leave40 adds 23 once) | PASS | PASS |

Inspection with tools built by `scripts/build-tools.sh`, EN then RU, notes in
`review/story1561-rom2-druids-town/`:

- `savecheck -party -resave load` of `game0030`: the party and bag above on both roots; the EN
  and RU resaves are byte-equal.
- The headless departure (LOAD `game0029`, victory, travel to town 3, SAVE): town 3 current at
  stage 60; places 2/2, 1/31, 1/43, 1/41, 1/42, 1/60, 2/3, 1/52 in `game0030`'s order; the bank
  equals `game0030`'s; Gildarius, Dina and Iglez in groups of their own with `game0030`'s packs.
  EN and RU files are equal field by field.

Release tests written and registered, not run in the lane:
`TestReleaseSecondTownDruidsLoadKeepsEachPack`, `TestReleaseSecondTownDruidsAfterMissionFifty`,
`TestReleaseSecondDinaJoinsAtMissionFortysLeave`, and the changed `TestReleaseSecondGameSupportCensus`
(46 entry maps, 45 ready, 15 engine exits, none unreached).

## Open debt

- DIV-4391: town record fields of the Druids' town without a source (Player Outcome, Human
  SpellsHeader, U48, U90, U154, U15C, U178, piece F47 and F48, W52 bytes, Block12 bytes 6..7,
  RuntimeIDs, Inventory1C against the file's 0).
- DIV-4392: the shield 0xb226 enters the bag at the return from mission 50 with type byte 0; the
  original keeps it in the hero's pack at type 1.
- DIV-4305: the hero's pack order (his own pieces before the bag's) and the record keys.
- DIV-4306: the tail's two words (781 against 1487) and the 16 raw bytes of places 2/3 and 1/52
  the departure adds (zero against the catalog's x, y, x, y).
- Unknown: the original's run of these files. The owner-test kit names what to check.
