# Campaign actors and second-game hires

## Result

One campaign actor resolver supplies character templates, first-game hire and siege squads, second-game inn subjects and selected second-game mission hires. Iglez has an inn card and figure before his talk. He still joins at the talk's end. A selected hire enters the mission once and retains identity and appearance through current SAV and cold LOAD.

## Authority

- Owner direction: ROM1 is profile data of the fuller ROM2 builder. Existing first-game instances must remain unchanged. The join time remains the end of Iglez's talk.
- `R2-ENGINE-524` (High/Medium): Humans parameter 24 before Units parameter 55, highest matching index, base key 10000+10*ID and conditional stage key 9998+10*ID+stage/10.
- `R2-ENGINE-525` (High/Medium): hire identity and row class remain distinct. `R2-ASSET-125` (High) supplies seven installed families.
- `R2-ENGINE-381` (Medium), narrowed by `R2-ENGINE-523` (High/Medium): fresh mission entry selects nonzero selection and availability. LOAD does not create a hire in the acquired routes.
- `R2-ENGINE-633`, `R2-ENGINE-635`, `R2-ENGINE-637` retain their High/Medium/Unknown grades for inn preparation, identity 22 from key 10220, town registration and the identity-to-companion join.
- `R2-ENGINE-527`, `R2-ENGINE-530`, `R2-ENGINE-532`: the campaign hire's exact cell remains Unknown. `PARTY-MERC-007` and `MERC-LEVEL-005` are ROM1-only authority for the owner's fallback, never ROM2 evidence.

## Builders

| Kind | Builder |
|---|---|
| Campaign actor and template resolution | `mapload.CampaignActor`, with named Human, named Unit, composed NPC and base/stage-key profile data |
| Inn subject and card | `innPatron` -> `partyPanelSubject`, existing room/card and member portrait composers |
| Dialogue face | `speakerResolver.SpeakerFace`, fed by the same campaign actor values |
| Party transfer | Fresh `enterMission`, then the existing party spawn and actor constructor |
| SAV | Existing current SAV producer and current party restore |

`CharacterMember`, `buildMercenarySquad`, `buildSiegeSquad` and `siegeHireMember` delegate construction. The static architecture guard prevents those entry points from decoding or directly constructing a member again. It does not claim to detect arbitrary renamed copies.

## As-built behaviour

First-game hire profiles retain the named NPC level templates, Catapult and Ballista, rotation, face, equipment, membership and empty-list shape. No first-game hire ID is repurposed for ROM2.

The inn previews an edition-controlled identity from its edition row without granting membership. After the talk it reads the carried companion and current outfit. Hire seats use the current stage's actor. Humans use their paperdoll; Units use their class portrait. The seat binds the actor's class art. The hire's own lines in its accept and reject sections draw the same actor through the existing portrait path.

Each page of an accept or reject section draws its own speaker, as every town talk does (DIV-4199): the hire's lines draw the hire actor, the lines of NPC 22 and 23 draw Iglez and Dina, and the hero's lines draw the hero. 12 of the 94 installed hire sections hold a line of NPC 22 or 23 on both second-game roots; `npc2accept30` opens with a line of NPC 22 and continues with one of NPC 2. Whether hire 2 and controlled identity 22 are one original actor stays Unknown (Q6).

Fresh mission entry owns the selected and available hire list, deduplicates existing hire membership and constructs missing members with stable `hire:<id>` identities. First-game `MercenaryType` stays zero for these members. Units use their selected Units row through the existing unit definition and party spawn. A resumed mission restores its current actors and does not run hire creation.

The current SAV preserves membership and appearance through its existing policy and ordinary actor data. Two optional policy fields store campaign hire membership and the selected collection. They default to zero/false in old saves and are omitted for existing first-game and controlled-companion members. Invalid membership combinations return a bounded error. The actor token's definition row is one byte, and every installed Humans hire template lies above it (rows 263..289 on both second-game roots). A Human hire therefore carries the first Humans row of its TypeID as its definition row; the party record, the mission actor the SAV producer writes and the member a cold LOAD restores carry that same row (DIV-4249). Its seeded dying time equals the template's for all 27 installed templates. Current actor state and portrait policy carry the template's appearance and statistics. A Units hire keeps its own Units row. Native original resave equivalence for these hires is unproved.

`DIV-3111` is narrowed from absent mission hires to remaining native placement/SAV questions and missing-template selection. `DIV-3109` records stage-resolved seats and accept portraits. `DIV-4199` records the pre-talk preview choice. `DIV-4249` records the shared formation/free-cell placement and selective Fire after a town LOAD under the owner's ROM1 fallback. IDs 4250 through 4252 are unused.

## Proof

The installed witnesses are `TestReleaseCampaignActorInnIdentity`, `TestReleaseCampaignActorHireRoute`, `TestReleaseCampaignActorFireAfterTownLoad` and `TestReleaseCampaignActorLossControls`. Each process reads one explicit root. They drive App headless selection and ordinary SAVE/LOAD routes; no window or audio is required.

Git archives of the requested base `698240cb51f50233dbd6db12040a604b0b9e0fd7` and integrated main `08613cb0b8d8bac88b44ef50103b8d9fcb111dca` receive the same witness source as the candidate. Each comparison has 178 equal files per ROM1 root and 18 per ROM2 root: 390 of 390, including raw SAV. Loss controls cover all fifteen first-game squad types at the four levels, five companion IDs across four sex/class inputs and four mission bands, 17 party portraits and production sprite frames, mission-30 World and raw SAV. The second-game control covers the existing Iglez join and the slot-773 Dina addition producer. Receipts stay outside the repository in the lane notes.

All seven hire-family mission/SAV/cold-LOAD witnesses pass on both second-game roots, with the definition row equal before and after the cold LOAD and each accept page drawing its own speaker. The ordinary `TestTownHireDialogueDrawsEachSpeakersOwnFace` drives the 12 hire sections with lines of NPC 22 or 23 through the inn's HIRE action, and `TestCampaignHireHumansRowAboveTheTokenByteTakesItsTypesFirstRow` pins the definition row rule. Fire after a town cold LOAD preserves Iglez and the other selected hire. Iglez's selected inn card retains his name, figure and end-of-talk join time. The base fails the three behaviour witnesses, including seven hire and two Fire cases. Final gate receipts are in the lane return. CPU actor images are compared; this lane does not claim full GPU mission-frame validation or original-runtime pixel equality.

## Open debt and release witnesses

Q1: the original pre-talk inn frame, town registration and identity 22's exact relation to the companion remain Unknown. Q6: native stage-specific seat/accept pixels, mission cells and original SAVE/LOAD/Fire semantics remain Unknown. Selected IDs without a valid template refuse fresh entry; the native row-0 fallback for IDs 8..20 is outside the delivered population. The original's token row for a Human hire whose template lies above 255 is Unknown.

Known engine debt the story leaves:

- Talk seats NPC 2 (topic 51) and NPC 5 (topic 79) share their low ID with hires 2 and 5. While that hire seat is present, the talk seat's card and that speaker's face are the hire actor. The original's choice is Unknown (Q6).
- The refusal for a selected ID without a template, `campaign hire %d at stage %d has no actor template`, is English on both locales. The installed data make it unreachable: the engine writes availability only for hires 1, 2 and 4.
- `reconcileCityGroups` would put a campaign hire carried in town into the hire group keyed by `MercenaryType` 0. The mission return drops every hire, so no route carries one in town.
- `missionParty` reads the literals `511+id` and `531+id` beside the named `secondHireSelection` and `secondHireAvailability` constants.
- The hire spec (`keyedCampaignActor`, identity, `bank[768]`, `Hire`) is written twice, in `townScreen.campaignActor` and in `missionParty`.
- Iglez's pre-talk inn card is built without the worn spells the join applies, so its item lines may differ before and after the talk (Q1).
- The first-game loss controls do not measure the siege restore on LOAD, the town-card numbers of `PartyDisplayWithTable`, `CharacterMember` with `Row`, `Face` or `Cells`, or a LOAD of the written first-game SAV. Each equals the base by code reading only.

At the next release, run the four new witnesses on their applicable EN/RU roots, `TestReleaseSecondInnPatrons`, `TestReleaseSecondTownInnHireSaveAndFire`, `TestReleaseSecondKaargTalkJoinsIglez`, `TestReleaseSecondKaargIglezKeepsThroughTownSave`, `TestReleaseSecondMissionThirtyStartsWithIglez`, `TestReleaseSecondIglezTalksDrawHisPortrait`, `TestReleaseSecondDinaTalksDrawHerPortrait`, `TestReleaseHiredIdentityCityAndMission`, `TestReleaseNativeTownSaveRealPartyEmitsNativeSAVAndRoundTrips`, `TestReleaseHiredSiegeCreaturesSurviveMissionSAV` and `TestReleaseSiegeEnginesCarryAFireBallWeaponSpell`. The lane runs only its own new installed witnesses.
