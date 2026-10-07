# Verification

## Automated evidence

The implementation test suite contains asset-free unit coverage and explicit
environment-gated release tests. The latter establish the named result only
when their lawful asset and save roots are configured. Neither kind of test
establishes a visible desktop GUI result.

| Criteria | Automated evidence |
|---|---|
| FR-1; AC-1, AC-2 | `TestRestoredCharactersKeepTheirFigureClassAndSex`, `TestPersistentOriginalCharacterExcludesTheTemporaryMission20Roster`, `TestRestoredLoadoutKeepsSavedStackCounts`, `TestFrontEndLoadsABetweenMissionOriginalSaveIntoTown`, `TestReleaseOriginalNairaSavesKeepFemaleAppearance`, and the env-gated `TestReleaseOriginalSave666KeepsItsRestoredAppearance` |
| FR-2; AC-3, AC-4; P-1, P-9 | `TestSwitchingInventoryHeroesTransfersNoCharacterState`, `TestTemporaryPartyMembersKeepTheirOwnInventorySubjects`, `TestDollPresentRebuildsOnTheSelectedSource`, `TestTheDollBoxPrefersFigureThenPortraitThenFrame`, and `TestReleaseTemporaryPartyAndNonPartyKeepTheirOwnPresentation` |
| FR-3; AC-5; P-3 | `TestMission30AddHeroCreatesASeparatePersistentCompanion` and `TestReleaseReniestaTavernJoinAndMapKeepBothCharacters` |
| FR-4; AC-6; P-2, P-6 | `TestACompanionPickupCreditsThePrimaryPurseAndDocumentSurface` and `TestDocumentPickupWithoutAPrimaryHeroIsAtomic` |
| FR-5; AC-7 | `TestAFreshMissionStartsWithTheTownPurse`, `TestAnOriginalMissionStartsWithItsPlayersSavedPurse`, `TestFinishingAMissionCarriesItsPurseBackToTown`, `TestTheAuthoredTownTransitionRewardIsSeparateConfigurableAndPaidOnce`, and `TestReleaseReniestaTavernJoinAndMapKeepBothCharacters`, which witnesses fresh 100 becoming 600 through the disclosed +500 reward; death-gold, sack, script, and canonical-form tests in `pkg/sim` and `pkg/mapload` cover the other automated boundaries |
| FR-6; AC-8; P-5 | `TestTownNPCTextPathNamesTheInnLeaf`, `TestShopDialogueUsesThePortraitNoticeWindow`, `TestMissingOrEmptyTownDialogueDoesNotOpenOrConsumeAnOffer`, and `TestTownDialogueIsModalAndItsOKButtonAdvances` |
| FR-6; AC-9 | `TestTownShopStripScrollsFiveSeparateEightyPixelPopupCells` |
| FR-7; AC-10 | `TestMageCloakPrimaryIsBehindTheBodyAndSecondaryIsInFront` |
| FR-8; AC-11; P-7 | `TestWeaponSpellDamageReportsTheLiveReleasesDamageInterval`, `TestWeaponSpellCharacteristicsForReportsPrismaticDamageRaysAndRange`, `TestWeaponSpellCharacteristicsForReportsStoneCurseDurationInTicks`, `TestAWeaponSpellMakesDamageTheEffectiveSpellInterval`, `TestEntityDrawCarriesTheLiveWeaponSpellIntervalToTheSheet`, `TestChargenDerivedUsesTheLiveStaffSpellAsDamage`, `TestItemInfoLinesUsesTheLiveWeaponSpellInsteadOfStaffPhysicalDamage`, `TestItemInstanceInfoLinesDescribesAnAttachedStaffCastOnce`, `TestMagicStaffTooltipsDescribeStoneAndPrismaticAcrossEveryProducer`, and `TestHoveredItemInfoAtKeepsAStaffSpellDamageInterval`. The producer table requires live, stored and kind-41 Stone Curse and Prismatic Spray projections to be byte-for-byte equal at the tooltip boundary. It pins exact sixteenth-second formatting, powered damage, owner-defined ray count, raw release range and `Value N`, and rejects every raw leading `#` and `power` line. The env-gated save-666 and Reniesta release paths check the Fire Arrow control through the production presenter and headless snapshot; the ordinary-weapon control remains physical. |
| FR-8; AC-12; P-8 | `TestMissionCompleteClickIsConsumedAcrossTheTownTransition` |

`TestReleaseGeneratedMagicStaffKeepsItsSpellAndExactPriceThroughTheWholePath`
is the installed EN/RU value-line witness: it carries one generated staff from
shelf through table, purchase, equip, form-61 reload, cast, sale and back to the
weapon shelf while requiring the same `Value N` description at each visible
trade boundary.

## Required real-data and live acceptance checklist

These checks remain required because a synthetic fixture cannot establish them.
They must use the lawful local installation and the exact source invocation
below; a substitute invocation is not equivalent evidence.

1. For each lawful GOG installation, EN and RU, set that installation as the
   configured asset root and run:

   ```text
   go run ./cmd/againrom
   ```

2. On each installation, load original save `666`. Record the displayed primary identity and party
   membership: Naira remains female when applicable; the mission-20 temporary
   actors are not permanent party heroes; the decoded purse is shown; and the
   incomplete town-progress boundary is disclosed.

3. Load the preserved Naira inputs before general GUI exploration:
   `game0013.sav` (`naira`, mission 10,
   `B211B9AD621A2CEC38FF5A1D1F3F632542A58E1E377C8F562B25BA48CA2AA5EA`)
   and `game0014.sav` (`naira 2`, mission 20,
   `4C37A3D122EA849CE989F054C82483E32BE9450886E7E3B60B641936A0BF0AFC`)
   from `gameversions/saves/2026-08-14`. The recorded real-data probe for both
   reports `name=Naira`, `FigureDir=ffighter`, `face=1`, saved body `archer`,
   `BodyDir=heroes_l`, worn codes `[33076 0 0 0 0 0 46895 0 0 0 0 44060]`, a
   nonnil composed doll with no unread inputs, Human Archer world art at tick
   zero and first tick, and purse `100`.

4. Select each permanent hero, a temporary ally, a mercenary, and an enemy or other
   non-party actor. Confirm every selected unit shows its own nonblank doll-box
   picture, never the previously selected hero's art. Confirm every party
   member exposes only its own pack and worn-equipment boxes; an enemy or other
   non-party selection receives no party inventory. Confirm campaign gold and
   Quest Documents remain primary-only. Return to the loaded primary and
   confirm its appearance remains unchanged.

5. On both EN and RU, reach mission 30 from each primary-sex path. Confirm the
   companion is added once. For NPC22, confirm the same identity is used in
   tavern dialogue, junction, map, and equipment: female mage, staff, dress,
   and cloak. Confirm EN displays Reniesta and RU preserves the lawful localized
   NPC22 display name rather than forcing the English string.

6. On both EN and RU, start a fresh campaign and confirm the purse is 100. Complete mission 20
   and confirm the owner-authored configurable +500 campaign-transition reward
   and resulting balance are visible; do not label it the shipped mission
   payment, whose scenario value is zero. Pick up gold with a companion and
   confirm its quantity notice, EXP-0008 gold sprite in the primary inventory,
   the primary purse, and Quest Documents routing; ordinary items remain on the
   companion.

7. On both EN and RU, open TAVERN, SHOP, and SCHOOL. Confirm their shipped dialogue text and
   matching per-part portraits, modal overlay behavior, visible room buttons,
   and no underlying hover or control activation. Confirm the shop displays a
   scrollable, read-only inventory line with arrows and item popups.

8. On both EN and RU, inspect a mage wearing a cloak in map and hero inventory views. Confirm the
   primary cloak lies behind the body and the secondary layer is foreground.
   Inspect a staff user in character generation and the map sheet: the one
   `Damage` value is live spell damage, without a physical row or `0-0`/`2-2`
   placeholder. Inspect Stone Curse and Prismatic Spray staffs: Stone Curse
   states its duration in seconds and range; Prismatic Spray states damage,
   rays and range; neither shows level, power, or raw leading-`#` lines. With
   lawful save `666` loaded, inspect Reniesta's equipped Wood Staff tooltip. It
   shows `Magic`, `Casts Fire Arrow`, `Damage 5-10` and Fire Arrow's `Range`,
   not `Damage 0-0` or physical to-hit/defence. Confirm an ordinary weapon
   tooltip still shows its physical interval and fields.

9. On each exact source launch, click Mission Complete while the pointer ends
   over a town control. Confirm the click's release does not activate that
   control. Confirm a subsequent independent click does.

## Evidence boundaries

The env-gated release tests directly exercise preserved Naira inputs and save
`666` when their lawful asset/save roots are configured. They check the
save-666 headless worn-item tooltip against the live staff spell, interval and
range. They do
not certify the visible GUI, EN/RU localized rendering, or checklist
interactions by themselves. No recorded result in this folder substitutes for
the remaining live checklist steps, including the gold sprite and pointer
behavior.

## Mission census evidence

The current mission census reports mission 10 at 17, unchanged from the
baseline. Mission 20 reports 12 against the baseline 13 because instant
operation 23 is now supported. These figures witness that one census change
only; they are not a playability verdict or evidence that the live GUI checklist
is complete.
