# Who a placed unit is, in the map editor

## Intent

The unit window sets who a placed unit is with one choice of three kinds: a creature (a Units row, by name and variant), a person (a Humans row, by name) and a named character (an NPC record, by name and portrait). It offers only the settings the game reads for that kind, keeps the raw map keys visible read-only, writes through the one field writer, and the Add window places a new unit through the same choice. The window says whether a dialogue shows the unit as a speaker. Every unit is named by one label (who it is, its number, its cell; the hero band by role), and an On map pick stores the unit standing on the clicked cell.

## Authority

- Owner story: the six results of the brief (one choice of three kinds; only what the game reads, raw keys read-only in a fold; one writer and the same choice in the Add window; a speaker line from `UnitSpeaker`; one unit label; the second game said and left).
- `MISSION-ARM-006` (High): four arms in one order. Class key +0x08 at or above 0x1a builds a Units creature; below it, flag bit 0 takes the NPC arm before the definition id +0x10 is read; a definition id other than 0 and 0xcdcdcdcd finds a Humans row by server id; else the class key finds the Humans row by type. Decides the kinds, their order and which keys each kind writes.
- `ALM-CLS-052` (High / Medium): the Units search matches the class and secondary keys against a row's type and face; the definition id walks Humans by server id. Decides that a creature is keyed by type and face and a person by server id or type.
- `ALM-FLAGPATH-109` (High / Unknown): inside the Human band bit 0 selects NPC; other flag bits are read per arm. Decides that a kind change touches only bit 0 of the flags.
- `DLG-SPEAKER-022`, `DLG-SPEAKER-023`, `REG-NPC-088`, `DLG-SYNTH-042`: which placed unit a speaker record shows. `UnitSpeaker`, which the dialogue editor's speaker pick already uses, applies them; the unit window reads it and adds no test of its own.
- The second game keys a unit by server id and the person flag (`serverKeys.resolve`); its unit records are 48 bytes and the editor's unit section is the 70-byte first-game record, so the editor edits no second-game unit.
- Rows: `DIV-4476` (who a placed unit is), `DIV-4477` (one unit label, the on-cell pick and the speaker line); `DIV-3117` and `DIV-3209` amended and moved with them to `docs/divergences/ui-map-editor-units.md`, because `ui-map-editor.md` stood 23 bytes under the area-file ceiling; `DIV-618` amended in place.

## As-built behaviour

Reading. `mapload.Identify` turns `mapload.Resolve`'s answer into an `Identity` (kind, Units or Humans row, NPC record). `mapload.Identities` lists every identity the install's key rule can key a placement to, kinds in the rule's order; the named kind appears only where the table has NPC records. Both are data of the `unitKeys` rule (`classKeys` for the first game, `serverKeys` for the second), not a branch on game identity.

Composed companions. The game composes npc records 21 to 24 from the starting hero (`REG-SCN-098`); the install's table alone keys them to no row, so two shipped placements (EN and RU `70.alm` unit 105 on record 23, `100.alm` unit 112 on record 24) read Nobody before the correction. `tableForPartyNPCs` now calls `tableForHero(t, kind, mission)`, the one composition, and `mapload.IdentifyAt(u, t, mission)` answers `Identify` or, for a named placement that reaches no row, the npc record with the Humans row `tableForHero` gives it for each `herokind.Kind`. `Identities` and `Place` take the mission and offer and place these records the same way. The editor's mission is the one test play starts the map as (`testPlayNumber`: the package's number, else the guest mission `testPlayGuest` picks); on both installs a shipped map opened alone plays as guest mission 21, so the rows named are the tier-0 heroes (record 24: Man fighter PC_Fergard, Woman fighter PC_Reniesta, Man mage PC_Danath, Woman mage PC_Naira; record 23: PC_Naira or PC_Danath).

The choice. The unit window's identity section opens with Who: one choice whose entries are grouped under a heading per kind, each with its picture from the editor's existing builders (the Add window's class sprites, the NPC portraits). The choice shown is `IdentifyAt`'s; a composed companion shows its record as "companion by the starting hero", with a meaning line that names the mission and each hero kind's row, and a unit whose keys reach no row under any hero reads Nobody and asks for a choice. A note under the choice says the list does not mark speakers. Then owner, group and the Dialogue speaker line, and the fields the unit's arm reads as before (face, sex and the person setup of `DIV-4206` for a person, the stats and overrides of `DIV-3117`). The class key, secondary key, npc flag and definition id are no longer fields. A collapsed Map keys line lists each read-only with what it does on the unit's arm or that it is not read. A set of any raw key through the field writer, the commands or the plain view is refused with "the unit's ... key is read-only".

Writing. A choice runs `mapload.Place`, which writes the keys of the chosen rung so that `Identify` answers the choice and keeps every other byte, then `setIdentity` writes the changed keys as one `SetFields` edit (one Undo step):

| Kind chosen | Class key | Secondary key | Flag bit 0 | Definition id |
|---|---|---|---|---|
| creature | the row's type | the row's face | kept | kept |
| person | the row's type | kept if the unit was a person, else 0 | cleared | the row's server id, or 0 |
| named character | the Humans row's type | the NPC record | set | kept |

Choosing the identity the unit already has writes nothing. A choice the install's tables cannot key is refused.

Add window. The unit kind's steps are Who (the same entries), owner, group and cell. `AddUnitIdentity` places `placeNewUnit` with the keys `Place` gives an empty record, then the same `setIdentity`, then the existing placement check. The `NewUnitRecord` preview builds the same record. It is the one way to place a unit: `AddUnit` and `AddUnitTier` (place by raw class key and tier) are deleted, their five release callers place by identity, and the unreached `setField` checks `CheckClassKey`, `CheckDefinitionID`, `CheckUnitTier` and `checkUnitField` went with them.

Speaker line. The window's Dialogue speaker line reads `UnitSpeaker`: the speaker record that shows the unit at mission start, or the cause it is not shown, in the words of the speaker pick (`ui.SpeakerMissText`, now shared). Where another unit takes the record, the line names that unit by its label. For a composed companion `UnitSpeaker` answers `ui.SpeakerByHero`: the speaker depends on the starting hero.

One label. `MapInspector.unitLabel` builds `<kind> <name>, unit <n> at <x>,<y>` (a unit whose keys reach no row: `Nobody (class key k), unit n at x,y`). It is the record's name in the tree and the popup title, and every unit choice of a script parameter, the trigger wizard and the node popup (`InspectionScriptTarget.ChoiceText`). The hero band 10001 and above keeps its role labels from `editorHeroLabels`.

The pick. `MapEditor.recordHitsAt` lists the unit standing on the clicked cell before units whose outline only covers the cell; `unitValueAt` stores the first with a target.

Second game. Its units are not editable, so the window offers no Who choice and no Map keys there. `Identities` already lists the second game's two kinds from `serverKeys`; they wait for a second-game unit section.

Commands. The map tools' unit steps are identity, owner, group and cell (`docs/MAPEDIT-MCP.md`), which also gives the identity value (kind in the high 32 bits, row or record in the low 32) and the `set_field` refusal of `class`, `tier`, `definition`, `npc` and `npcflag`. A command record's `Name` is converted to UTF-8 like every other text (`plainRecord`).

## Proof

Failing at `568973b60` (tests only, on base `6ab9e349e`) and passing on the candidate, in `pkg/game/mapinspectionunitidentity_test.go`, on planted maps with no install byte: `TestEditorUnitIdentityShowsTheResolvedKindAndOnlyItsFields`, `TestEditorUnitIdentityChoiceWritesTheChosenRow`, `TestEditorAddWindowCreatesEachIdentity`, `TestEditorUnitPopupStatesItsSpeaker`, `TestEditorUnitChoicesTellUnitsOfOneKindApart`. In `pkg/ui/mapeditorparampick_test.go`, `TestTriggerUnitPickTakesTheUnitOnTheClickedCell` fails with the on-cell branch of `recordHitsAt` removed (stores 102, wants 103) and passes with it.

Census over the shipped maps, read-only (a temporary test in `pkg/game`, not committed, opening each map through `MapInspector.Editor().Open` and replaying the pick at each unit's cell; EN `gameversions/en` 28 scenario and 10 loose maps, RU 28 and 6):

| Measure | EN before | EN after | RU before | RU after |
|---|---|---|---|---|
| placed units, each with a pick target | 8094 | 8094 | 3991 | 3991 |
| pick on the unit's cell stores that unit | 8093 | 8094 | 3990 | 3991 |
| pick stores another unit | 1 | 0 | 1 | 0 |
| units whose shown choice equals another's but for the number | 7743 | 0 | 3736 | 0 |

Before, a unit choice was `Unit <n>: <class name>[  Tier t][; definition: Humans row r, <name>]`, without a cell; the stored-other case was scenario `50.alm` in both installs, a squirrel at 117,35 inside a dragon's outline. Arms by `mapload.Resolve`: EN 6672 units, 1405 server id, 15 NPC, 2 by type; RU 3366, 609, 15, 1.

Correction, failing at `3a17221b0` (tests only, on `1e209ce4a`) and passing on the corrected candidate: `TestEditorUnitIdentityNamesAComposedCompanion` (`pkg/game/mapinspectioncomposed_test.go`: a planted record 23 composed from the hero names the record, each hero kind's row, the byhero label and speaker cause, is offered by Who, and an unchanged Apply writes byte-equal), `TestEditorCommandRecordNameIsUTF8` (`pkg/ui`: a planted Cyrillic name). `TestEditorAddUnitIdentityWritesTheClassAndTierRecord` (`pkg/game/mapinspectionaddidentity_test.go`) pinned, before `AddUnitTier` was deleted, that both paths write the same 70-byte record for the pairs the moved release tests use (class 69 tier 0, type 10 tier 0, class 64 tier 3); a temporary probe over both installs found the two paths' records equal for those pairs. The `pkg/mapedit` output digest is re-pinned; with the old check numbering the same session gives the old digest, so only the numbering moved. The modes golden is re-recorded (`MAPEDITOR_MODES_WRITE=1`); its `ui:click-unit` gesture, whose cell 10,10 now falls below the shorter Add unit window's list, clicks that list's second entry when the point misses the list, so the `list`/`unit` row again moves the choice from "1: One" to "2: Two" and the `unit-owner` row moves the owner choice the same way.

Census after the correction (the review's `TestReviewIdentityShippedMaps`, a temporary test in `pkg/game` under an overlay, not committed; guest mission 21): EN 38 maps, 8094 units; RU 34 maps, 3991 units. Units whose identity reads none: 2 before, 0 after, in each install. Unchanged Apply byte-equal: EN 8094 of 8094, RU 3991 of 3991; no value outside the Who choices, no change that moves another unit or bytes outside the record.

Release witnesses for the next release, EN and RU (none ran in the lane): the five moved to `AddUnitIdentity`, `TestReleaseMapEditorNewMapIsWonAndLostByItsArms`, `TestReleaseMapEditorNewMapPlaysAsModMission`, `TestReleaseTestPlayOfANewMapStartsAtOnce`, `TestReleaseMapEditorPlayerNamesKeepSAVBytes`, `TestReleaseMapEditorPlayersPlayWithAuthoredColoursAndRelations`; then `TestReleaseMapEditorUnitTierPopupWritesTheSetFieldBytes`, `TestReleaseMapEditorNewUnitTakesTheChosenTier`, `TestReleaseMapEditorUnitValuesEqualTheGame`, `TestReleaseMapEditorUnitTierCensus`, `TestReleaseMapEditorUnitTierFrames`, `TestReleaseMapEditorAddWindowWritesTheRouteBytes`, `TestReleaseMapEditorAddWindowFrames`, `TestReleaseMapEditorOverridePopupWritesTheSetFieldBytes`, `TestReleaseMapEditorOverrideFrames`, `TestReleaseMapEditorUnitAuthoringPlaysAsModMission`, `TestReleaseMapEditorCloneNeverTakesADeletedScriptedUnitID`, `TestReleaseMapEditorFreshRecordAuthoringSAVAndGoals`, `TestReleaseMapEditorUnitStockCloneHasIndependentInventory`, `TestReleaseMapEditorNamedRecordChoicesUseExistingPrompts`, `TestReleaseMapEditorSpeakerPickNamesWhyEveryRefusedUnit`, `TestReleaseDialogueSpeakerIsNamedByItsRecordsLine`, `TestReleaseDialogueHeroSpeakersHaveTheFaceOfAMaleHero`, `TestReleaseTriggerWizardOpensShippedTriggers`, `TestReleaseTriggerWizardFrames`, `TestReleaseTriggerWizardReadsEveryShippedRowAsAnItem`, `TestReleaseMapEditorWizardApplyWithoutEditKeepsEveryTrigger`. Proposed: a release test that opens the popup of every shipped placement, applies it unchanged and checks the saved map byte-equal to the source.

## Open debt

- A unit with id 0, a shared id or an id in the hero band has no pick target, so a pick on it stores nothing; none is on a shipped map.
- The identity list does not mark which entries a speaker record would show; the speaker line answers after the choice.
- The second game's units are not editable; its two kinds wait for a second-game unit section.
- A composed companion's rows are those of the mission test play starts the map as (guest mission 21 for a shipped map opened alone), not those of campaign missions 70 and 100, where the game gives the higher-tier heroes; the meaning line names the mission it used.
- `serverKeys.identities` (the second game's two kinds) has no test of its own.
- The Add window and the identity list read the install's tables, not a mod's added rows.
