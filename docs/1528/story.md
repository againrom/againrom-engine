# Iglez joins the second game's party after his talk in Kaarg's inn

## Intent

In the second game, the talk with Iglez (NPC 22) in the Kaarg inn at stage 30
ends with Iglez in the party. He is built from the Humans row with serverID
10220 by the one character builder and carries party identity 22. He goes into
every mission with the hero, comes back with the party, and stays through SAVE
and LOAD. A second talk adds no second Iglez. Script literals 10001+k bind to
the party member with identity 21+k, so the 10002 nodes of `30.alm` build.
Identity 23 (the Mage_Hero row, serverID 10230) joins the same way when slot
773 asks for it at a location leave after mission 40. The first game is
unchanged.

## Authority

Owner direction (seat process log): "Iglez is like Reniesta (or Brian), but in
ROM2 he seems to join only after a talk (possibly the one about the Lairisha
quest)". The owner marks the moment as uncertain. Seat: Iglez is a companion
of the first game's AddHero kind (PARTY-ADDHERO-017) and joins through the
companion path with the second game's rule as profile data; the join is at the
end of the stage-30 talk with NPC 22. The talk is `#npc22talk30` in main.res
`text/town.txt` ("meeting Iglez, mission 30 is given"). Owner rule: one
builder per kind.

The pinned claim tool (knowledge k371) read these claims:

- R2-ENGINE-105, R2-ENGINE-124 (Medium): the character producer writes actor
  WORD+0x14c=21 for the hero.
- R2-ENGINE-107 (High / Medium): literal references 10001..11000 compare
  WORD+0x14c-21 with reference-10001 in the selected owner's list; 10002
  names identity 22.
- R2-ENGINE-106 (Medium): a location leave passes a nonzero slot 773 and the
  stage to the addition producer; for identities 21..30 it constructs nobody
  when an owner actor already carries the identity.
- R2-ENGINE-379 (Medium): mission 40's departure requests identity 23 through
  slot 773. Exact join timing and the producers of identity 22 are Unknown.
- R2-ENGINE-378 (Medium): Fighter_Hero (10220) and Mage_Hero (10230) are the
  template families of identities 22 and 23; displayed names are Unknown.
- R2-ENGINE-072 (High / Medium): the Humans parser turns a serverID above
  10000 into (serverID/10)%1000, so 10220 gives 22 and 10230 gives 23.
- R2-ENGINE-216, R2-ENGINE-219 (High): the stage-30 inn offers NPC 22 at topic
  30; that TALK admits mission 30 and stores slots 533=1 and 553=2.
- R2-ENGINE-380 (Medium, branch candidate): the NPC 22 topic 30 store enables
  hire 2 rather than constructing identity 22; talk and availability changes
  create no actor.
- R2-SESSION-047 (High): every ordinary departure zeroes slot 773.

The join at the talk's end is owner direction against R2-ENGINE-380's reading
of the talk body; DIV-4186 records it.

## As built

- `pkg/base/edition.go`: `Edition.HeroIdentity` (21 for the second game, 0
  for the first) and `Edition.PartyIdentities`, one row per identity: 22 from
  serverID 10220 with the talk NPC 22, topic 30; 23 from serverID 10230 with
  no talk.
- `pkg/mapload/partyhumanrow.go`: `PartyIdentity` gives a member's identity
  (the starting hero's `HeroIdentity`, a companion's `CompanionNPC` when the
  edition has its row). `PartyHumanRow` takes the identity row's Humans row
  first, so the SAV writer and the mission start bind identity 22 to
  Fighter_Hero.
- `pkg/game/campaignreturn.go`: `carryCompanion` is the shared tail of
  `carryTownCompanion` (reuse a carried member with the same NPC, else build
  and append). `addPartyIdentity` is the addition producer: it builds the
  edition row through `townInstall.character` with
  `CharacterSpec{NPC: id, Row: <row of the serverID>}`, names him by
  npcnames.txt line id-1, normalizes worn spells as a joining companion, and
  settles the arrival with `settleCompanionArrival`. `addTalkIdentities` runs
  it for the rows whose talk matches.
- `pkg/game/secondcampaign.go`: an inn talk remembers its option while its
  dialogue is open; the dialogue's close runs `endInnTalk`, which runs
  `addTalkIdentities`. A talk with no section ends at once. A joined
  identity stays an inn talk patron: the stage-30 inn still lists TALK 22
  after Iglez joined (a second talk adds nobody), and the stage-50 inn lists
  TALK 23 with Dina in the party. The original's list for a joined identity
  is Unknown (DIV-4190).
- `pkg/game/secondparty.go`, `pkg/game/campaignsecond.go`: `leftLocation`
  runs the producer for a nonzero slot 773, after a won mission's departure
  stores and after a town leave through the gates (DIV-4188).
- `pkg/game/mission.go`: `campaignScriptPartyRefs` binds by identity when the
  edition has a hero identity: member with identity 21+k binds 10001+k, first
  match first. The first game keeps the hero ordinal role test
  (TRIG-HEROORD-075).
- `pkg/game/secondheropane.go`, `pkg/game/shopview.go`,
  `pkg/game/shopcontroller.go`, `pkg/game/secondshoproom.go`: the inn pane's
  corners and the shop's PREVIOUS and NEXT step over the party the pane
  shows (the campaign party) through `stepMemberOver`, the body of
  `shopStepMember`. The shop host's `picker` hook names that party; the
  second game's shop sets it to `secondPaneParty`, and the first game's
  shop leaves it nil, so its picker steps its customers as before. A
  purchase still lands with the bag member: `shopMemberIndex` clamps the
  shown index through `townPickerIndex` over the one-member bag party
  (DIV-3043).
- `pkg/game/world.go`, `pkg/game/residuecarrier.go`,
  `pkg/game/campaignsecond.go`: owner direction ("Yes, his own notice",
  DIV-4189). The guarded loss (DIV-219) names the character it found
  fallen; `guardedLossReason` reads his party identity's edition row, and
  identity 22's row carries failure reason 3. With no script reason of 2 or
  more, the second game's loss notice shows that reason's section, so
  Iglez's fall in mission 30 shows `failure3`. The hero has no row and keeps
  the generic body.
- `pkg/base/edition.go`: `Edition.BindsByIdentity` is the one test of the
  binding rule; `mapload.PartyIdentity` and `campaignScriptPartyRefs` both
  ask it. An edition that binds by identity has no role fallback
  (R2-ENGINE-107).
- `pkg/game/secondcensus.go`: the census party is the hero plus every
  edition identity, built by the same producer; a new `unbound_heroes` column
  lists hero-band values the party leaves unbound.

### Kinds and builders

| Kind | Builder used |
|---|---|
| Party member construction | `mapload.CharacterMember` through `townInstall.character` |
| Companion carry and arrival | `CampaignSession.carryCompanion` (tail of `carryTownCompanion`) and `settleCompanionArrival` |
| Addition of a member by identity | `CampaignSession.addPartyIdentity` over `Edition.PartyIdentities` |
| Game difference as profile data | `base.Edition` (`HeroIdentity`, `PartyIdentities`, `BindsByIdentity`) |
| A member's Humans row (SAV, dying time) | `mapload.PartyHumanRow` |
| Script hero-band binding | `campaignScriptPartyRefs` with `bindHeroOrdinal` |
| Guarded loss and its notice | `guardedCharacterLost` (now naming the fallen character), the edition row's `FailureReason`, `secondCampaignRules.outcomeText` |
| Inn talk and its end | the inn talk route (`talk`, `sectionTownTalk.turnTownTalk`) |
| Town leave | `destinationTownCampaign.leave`, `secondCampaignRules.finishWon` |
| Picker step | `shopController.stepMemberOver` (`shopStepMember` calls it over the shop host's `picker` party, else the customers) |
| Speaker cast | `newSpeakerCast`, unchanged; it reads `CompanionNPC` |
| SAV party record | `currentPartyPolicy`, unchanged; it carries `CompanionNPC` |
| Census | `SecondGameCensus`, `secondScript` |

Hires 1..20 would be further rows of the same producer: a row per hire ID h
with its template serverID and a selection gate (slots 511+h and 531+h,
R2-ENGINE-380, R2-ENGINE-381) in place of a talk, run at the mission entry,
built by `addPartyIdentity` and carried by `carryCompanion`. Not built here
(DIV-3111).

### Hashed state, SAV and old saves

No new persisted field. Iglez is a carried party member; the SAV party policy
writes his `CompanionNPC` and the Humans row `PartyHumanRow` gives, and LOAD
restores both. An old second-game SAV carries no identity member; its party
loads as before. It gains Iglez only through the stage-30 talk with NPC 22:
an old SAV at stage 40 or later never gains Iglez, because the stage-40
TALK 22 has topic 40 and no edition row names it, and slot 773 only ever
asks for identity 23; its maps keep 10002 unbound, as at base. An old SAV
written after mission 40's departure still holds slot 773=23 until the next
won mission's departure zeroes it (R2-SESSION-047): it gains Dina at its
next town leave through the gates, as a new campaign does. One written after
that later departure holds 773=0 and never gains Dina, as at base (an
untracked diagnostic over the SAV document form, both cases). The first
game's edition has no hero identity, so its binding, party and SAV are
unchanged.

## Proof

### Focused tests (`pkg/game`)

| Test | States | Fails on base or under the mutation |
|---|---|---|
| `TestSecondInnTalkAddsIdentityTwentyTwoOnce` | TALK 2108 adds nobody; the open TALK 22 keeps one member; its end adds identity 22 named Igles from row 30; a second talk adds nobody; the speaker cast gives npc 22 to him | `endInnTalk` emptied |
| `TestSecondInnTalkWithoutSectionAddsAtOnce` | a talk with no section adds 22 at once | `endInnTalk` emptied |
| `TestSecondIdentityKeepsThroughPolicyAndIsReused` | the party policy record keeps identity 22; the producer does not build him again | base: no producer (compile failure) |
| `TestSecondLeaveAfterFortyAddsIdentityTwentyThree` | slot 773=23 after mission 40's stores; the leave adds 23 once, row 31 | `identityRow` disabled: row 0 |
| `TestSecondHeroBandBindsByIdentity` | 10001 the hero, 10002 identity 22, 10003 identity 23, by identity, not order | identity branch of the binder disabled: 10002 unbound |
| `TestFirstHeroBandKeepsTheRoleTest` | the first game binds no literal by identity | guards the first game |
| `TestSecondPanePickerStepsToIdentityTwentyTwo` | the pane counts 2 members; NEXT shows Iglez, PREVIOUS the hero | `TownSquarePane` back on `shopStepMember`: member 0 of 2 |
| `TestSecondGuardedLossOfIdentityTwentyTwoShowsFailureThree` | identity 22's fall shows the mission text's `failure3`; the hero's fall shows the generic body, not `failure2` | identity 22's `FailureReason` 0, the candidate's behaviour: the generic body |
| `TestSecondShopPanePickerStepsToIdentityTwentyTwo` | in the shop, NEXT shows Iglez; a purchase then lands in the bag member and leaves Iglez's items; PREVIOUS shows the hero | the shop host's `picker` nil (the candidate before the correction): member 0 of 2 |

### Census

`campaigncensus` over each second-game root, base binary (3149253d) and
candidate. The fixture column is unbound references, omitted checks and
omitted actions from the census party's compile. EN and RU are equal map by
map; their totals differ only in placements (4212, 4234) and triggers (715,
714).

| Map | Fixture before | Blockers before | Unbound before | Fixture after | Blockers after |
|---|---|---|---|---|---|
| 30 | 11/10/0 | party | 10002 | 0/0/0 | - |
| 40 | 10/7/3 | party | 10002 | 0/0/0 | - |
| 41 | 1/1/0 | party | 10002 | 0/0/0 | - |
| 42 | 4/6/0 | party | 10002 | 0/0/0 | - |
| 43 | 1/2/0 | party | 10002 | 0/0/0 | - |
| 50 | 10/12/0 | party, continuation | 10002, 10003 | 0/0/0 | continuation |
| 51 | 2/2/0 | party | 10002, 10003 | 0/0/0 | - |
| 52 | 6/6/0 | party, entry | 10002, 10003 | 0/0/0 | entry |
| 53 | 10/10/0 | party | 10002, 10003 | 0/0/0 | - |
| 60 | 3/2/1 | party | 10003 | 0/0/0 | - |
| 62 | 4/4/0 | party | 10002, 10003 | 0/0/0 | - |
| 63 | 4/7/0 | party | 10002, 10003 | 0/0/0 | - |
| 70 | 17/6/4 | party | 10002, 10003 | 0/0/0 | - |
| 71 | 4/6/0 | party | 10002, 10003 | 0/0/0 | - |
| 72 | 12/8/0 | party | 10002, 10003 | 0/0/0 | - |
| 73 | 4/4/0 | party | 10002, 10003 | 0/0/0 | - |
| 74 | 2/2/0 | party | 10002, 10003 | 0/0/0 | - |
| 75 | 6/9/0 | party | 10002, 10003 | 0/0/0 | - |
| 76 | 6/8/0 | party | 10002, 10003 | 0/0/0 | - |
| 77 | 4/6/0 | party | 10002, 10003 | 0/0/0 | - |
| 80 | 4/4/0 | party | 10002, 10003 | 0/0/0 | - |
| 84 | 2/2/0 | party | 10002, 10003 | 0/0/0 | - |
| 85 | 2/4/0 | party | 10002, 10003 | 0/0/0 | - |
| 86 | 4/10/0 | party | 10002, 10003 | 0/0/0 | - |
| 87 | 6/10/0 | party | 10002, 10003 | 0/0/0 | - |
| 90 | 4/4/0 | party | 10002, 10003 | 0/0/0 | - |
| 91 | 6/6/0 | party | 10002, 10003 | 0/0/0 | - |
| 92 | 6/6/0 | party | 10002, 10003 | 0/0/0 | - |
| 100 | 6/6/0 | party | 10002, 10003 | 0/0/0 | - |
| 101 | 4/4/0 | headless-loss, party | 10002, 10003 | 0/0/0 | headless-loss |
| 102 | 8/10/0 | party | 10002, 10003 | 0/0/0 | - |
| 103 | 16/8/0 | party | 10002, 10003 | 0/0/0 | - |

The other 14 maps had no fixture omission before or after. Totals: fixture
189/192/8 to 0/0/0, fixture maps 32 to 0, ready 14 to 43. No other total
moved. The census party carries identities 21, 22 and 23 on every map; a
campaign carries 22 only after the talk and 23 only after slot 773.

### Witness over the installs

A headless test, untracked (`pkg/game/zz_iglezwitness_test.go`, its copy in
the lane's notes), on rom2-en and rom2-ru from a new game at the constructed
stage 30 in Kaarg (the `secondTownTwo` position). EN and RU are equal except
the names (RU in cp866).

| Step | EN | RU |
|---|---|---|
| inn rows at stage 30 | TALK 22, TALK 2108, TALK 2110, EXIT | same |
| party before the talk | 1: hero, identity 21, Aldor | 1: hero, identity 21, Альдор |
| after TALK 2108 | 1 | 1 |
| during TALK 22 | 1 | 1 |
| after TALK 22 | 2: hero; npc:22, identity 22, Igles, row 30 Fighter_Hero | 2: hero; npc:22, identity 22, Иглез, row 30 Fighter_Hero |
| after a second TALK 22 | 2 | 2 |
| mission 30 first frame (tick 0) | hero entity 124 at (12,14), 145/145; Iglez entity 125 at (11,16), 158/158; both owner 1; world hash 68ef5083afc42afa | same |
| 30.alm binding | 10001 to entity 124, 10002 to entity 125 | same |
| 30.alm with Iglez | 0 unbound hero-band references, 0 omitted checks, 0 omitted actions | same |
| 30.alm, hero alone | 11 unbound references to 10002 (2 actions, 9 checks), 9 omitted checks, 2 omitted actions | same |

### Release tests for the release

Written, not run in this lane:

- New, on rom2-en and rom2-ru: `TestReleaseSecondKaargTalkJoinsIglez`,
  `TestReleaseSecondKaargIglezKeepsThroughTownSave`,
  `TestReleaseSecondKaargShopPaneStepsToIglez` (the shop pane's NEXT and
  PREVIOUS by pointer, a purchase with Iglez shown),
  `TestReleaseSecondMissionThirtyStartsWithIglez` (mission 30 with Iglez,
  its SAV and cold LOAD), `TestReleaseSecondMissionThirtyWithoutTheTalk` (the
  loss control).
- `TestReleaseSecondGameSupportCensus`: ready 43, fixture maps 0.
- On the changed route (the stage-30 talk now adds Iglez):
  `TestReleaseSecondStageThirtyInnOpensMissionThirty`,
  `TestReleaseSecondLaterDepartureReachesTheNextMap`,
  `TestReleaseSecondLaterMissionSaveContinuation`,
  `TestReleaseSecondStageFortyInnOffersItsMissions`,
  `TestReleaseSecondTownInnHireSaveAndFire`, the `TestReleaseSecondTownKaarg*`
  tests and the second-game town and mission SAV tests.

## Open debt

- DIV-4186: which producer creates identity 22 in the original, and at which
  moment of the campaign?
- R2-ENGINE-380: is hire 2, which the NPC 22 talk enables, the original's
  Iglez, and what does its selection create at the mission entry?
- DIV-4187: which template does the producer select for identity 22 and 23 at
  each stage, and which name does the original display?
- DIV-4188: which location leaves reach the slot 773 consumer, and is a
  mission's end one of them?
- DIV-4189: what does the original show when identity 22 falls in mission
  30, and when the hero falls, and what stores the failure reason on such a
  loss?
- DIV-4190: which inn patrons does the original list for an identity the
  party already carries (TALK 22 at stage 30 after Iglez joined, TALK 23 at
  stage 50 with Dina in the party)?
- How does an original second-game save record identity 22 (actor
  WORD+0x14c), and should the original-save reader map it to the companion?
- Does the original gate literal binding on Session +0x74 at mission load, and
  what does its alternate global list bind?
