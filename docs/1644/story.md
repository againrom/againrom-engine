# A town save loads after its mod changed

## Result

A town save loads after one of its mods was edited, updated or set otherwise, if the same mods are active in order. What
the changed mods no longer hold goes: an item they no longer define, a skill above their cap (to the cap), a layer,
a guest or campaign mission they no longer declare; an item they still define stays, even when its code moved. The
Load page lists what went, in English or Russian, and the town opens when the list closes. A mission save, or one
made under other mods, is still refused.

## Authority

- Owner: «добавить возможность загружать измененный мод в городе -- имеется в виду, что мод поменялся и мы просто
  разрешаем загрузить сохранение которое из города».
- Owner: «решение верное» on the rule: an item the mod no longer defines is removed and the load lists it; a skill
  the mod no longer raises falls to the ordinary cap.
- Owner: «Да, записывать» on writing each held mod item's mod and key into the save's mod mark.
- Owner: «Грузить и при другой настройке»: a changed mod setting loads as a changed mod does.

## Rules

1. A town save whose mods are the same in the same order and differ only in files, versions or settings loads.
2. A mission save, other mods, another order or another base is refused as before, same line.
3. An item the active mods no longer define leaves its holder's pack, or comes off its worn slot, which stays empty.
4. A skill above the active cap falls to the cap; a clothing layer the mods no longer define comes off.
5. A waiting guest whose mission is still declared takes its number now; one whose mission is gone is dropped.
6. A campaign mission no longer declared leaves the town's lists; a selection of it falls to the main mission.
7. A load or quick load lists what went, EN and RU, in the Load page's mod window; a load that lost nothing shows none.
8. The next save writes the active mods' mark and loads with no list; the file that was loaded is not changed.
9. Unchanged mods and no mod leave the same state, and the same next save but for `mod` and `key` in the mark.
10. The mark names each held mod item and layer by mod and key; under changed mods each keeps the item its key names.
11. A removed item or layer is listed by its key and mod, beside its holder.
12. A save from before item keys loads under changed mods only without mod items or layers; else it is refused.

## Code

| kind | its one builder | change |
|---|---|---|
| mod admission | `applyModMark` over `mod.Compare` | `townAdmits`, a rule inside it; no second comparison |
| skill, item, layer readers | `restoreModDomain`, `restoreModItems`, `applyModLayers` | a changed flag: held to the cap, gone, left off |
| party item removal | `withCityShopMutation` with `cityShopTakePack` | `cityShopTakeWorn` split out of `cityShopUnequip`; a hire is a holder |
| detached shop room | `detachedShop`, shared with `resolveShopSnapshot` | its three lines moved out, both call it |
| guest and campaign readers | `restoreGuest`, `campaignProgressFromSAV` | inputs held first: `adoptChangedGuests`, `pruneUndeclaredMissions` |
| load seam | `RestoreOriginal` | answers `*ui.ModLoadNotice` over a completed town load |
| notice window | `modMismatchWindow`, opened by `openLoadNotice` | reason `ModMismatchChanged`, six row kinds, words in both tables; quick load opens it too |
| save producer | `markDocumentForMods` | stamps `mod` and `key` on each item and layer (format 1, omitted when empty) |

## Proof

| test | rules |
|---|---|
| `TestATownSaveLoadsWithoutThePackItemItsModNoLongerDefines` | 1, 3, 7, 8 |
| `TestATownSaveWhoseModChangedOnlyItsFilesLoadsWithNoList` | 1, 7 |
| `TestATownSaveLoadsWithoutTheWornItemItsModNoLongerDefines`, `TestARemovedItemComesOffATavernHireToo` | 3, 8, 11 |
| `TestATownSaveLoadsWithItsSkillsHeldToTheLoweredCap`, `TestTheModMarkCarriesLayersAndRefusesBadOnes` | 1, 4 |
| `TestATownSaveUnderChangedModsHoldsItsGuestsToTheDeclaredMissions` | 5 |
| `TestATownSaveUnderChangedModsDropsTheMissionsItsCampaignLost` | 6 |
| `TestChangedModsStillRefuseAMissionSaveAndAnotherSet`, `TestLoadRefusalsCarryTheirDifferencesAsData` | 1, 2 |
| `TestATownLoadUnderChangedModsListsWhatWentThenEntersTheTown`, `TestAQuickLoadUnderChangedModsNamesWhatWent`, `TestALoadWithoutANoticeOpensNoWindow` | 7, 11 |
| `TestATownLoadUnderChangedModsWithoutTheFontSaysItOnTheMessageLine`, `TestTablesEqualTheReplacedWords` | 7 |
| `TestAnUnchangedModTownLoadLeavesTheSameState`, `TestAnUnchangedModTownSaveAddsOnlyTheItemKeys` (base hashes) | 9 |
| `TestATownSaveKeepsItsModItemsWhenTheirCodesShift`, `TestTheModMarkHoldsAModItemsStandInAndRestoresIt` | 10 |
| `TestATownSaveWrittenBeforeItemKeys`, `TestASaveWrittenBeforeItemKeysNamesWhyItIsRefused` | 12 |

Release tests at the release: `TestReleaseLoadPageListsEveryModMismatch`,
`TestReleaseModSkillCapRaisesAHeroPastTheOriginalCapAndTheSaveCarriesIt`,
`TestReleaseModSkillCapLeavesAnUnmoddedSaveUntouched`, `TestReleaseModInstallFilesTextModIsMarkedAndItsSaveNeedsTheMod`,
`TestReleaseModAbandonReturnsToTheTownAsEntered`, `TestReleaseModMissionGuestPlaysAndJoins`, under `AGAINROM_ASSETS`.

## Open debt

- A removed item is named by its key, not a translated name: the active tables no longer hold it.
- A layer whose pack unit was removed is listed twice, as the item and as the layer.
- Mod missions declare their own numbers, so guests and campaign missions match by declaration, not position.
- A gone item outside a party pack or worn slot, as in a second-game town bag the city graph does not hold, refuses.
- Changed files or settings are reconciled for items, skills, layers, guests and missions only; nothing else is listed.
