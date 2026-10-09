# ROM2 ordinary departure for every mission and the stage-30 inn

## Intent

Before this story the ROM2 campaign controller continued only missions 10,
20 and 21, played only the mission 10 movie, and wrote a mission SAV only on
those three maps. The story1368 census counted 43 maps blocked on
continuation, 4 on a movie and 43 on save. This story runs the whole
published ordinary departure for every won mission, produces each movie exit
behind its gates, and writes a mission SAV on every ordinary mission. ROM1
code paths, release tests and hashed state do not change.

The second step implements the town inn options and TALK that knowledge k205
publishes. From a new game, ordinary play reaches town 2 at stage 30, where
TALK 22 admits mission 30 and TALK 2108 admits mission 31; mission 31 departs
to 32.

Base: `71847832` (game 0.97.0). Knowledge pin: k204, moved to k205.

## Authority

| Part | Claims |
|---|---|
| dispatch: 12 ID cases (10..110), 9 stage cases on slot 768 (30..110), join for every other ID | R2-ENGINE-145 |
| adds per ID | R2-ENGINE-146, R2-ENGINE-147 |
| output DWORD and its gates | R2-ENGINE-148 |
| common prefix | R2-SESSION-047 |
| per-ID bank stores | R2-SESSION-048 |
| join on slot 768 and stage switch | R2-SESSION-049 |
| auxiliary array stores | R2-SESSION-050 |
| slot 896+ID store is unbounded | R2-SESSION-052 |
| type-2 departure keeps a later town's node | R2-ENGINE-144 |
| cutpaths row indexed by the output | R2-ENGINE-075 |
| EnterInn selects on slot 768 for any current record | R2-ENGINE-215 |
| stage-10 and stage-30 inn entries, slot 927 gate | R2-ENGINE-161, R2-ENGINE-216 |
| fifteen continuation stores, signed stage compares, slot 776/781 topic pick | R2-ENGINE-217 |
| dynamic tail of kinds 1 and 2 | R2-ENGINE-221 |
| kinds 0 and 3 are talk actors; first match; npc%dtalk%d | R2-ENGINE-220 |
| TalkTo: kind 3 admits the topic; NPC 22 topic 30 stores 533 and 553 | R2-ENGINE-219 |
| initial topic 10 stores slot 769 | R2-SESSION-110 |
| stage 30 at the first town 2 visit, 40 after mission 30 | R2-SESSION-107, R2-SESSION-109 |

## As built

- `secondCampaign.completeBank` runs, in order: slot 773 zero, the slot
  512..551 normalisation, removal of current, slot 896+ID = 1, the ID case,
  the join (slot 768 += 10 for an ID divisible by ten) and the stage switch on
  the joined slot 768. It returns the output DWORD, -1 unless a case stores
  one. Case bodies follow R2-ENGINE-146 and R2-SESSION-048: 10 adds 20; 20
  adds town 2 and 21 if slot 772; 31 adds 32; 40 adds 50 and 60 and stores
  773=23; 50 adds town 3 and stores 771, 534, 554; 60 adds 80 and stores 537,
  557, 774; 70 stores 535, 555, then 777 and 770 after its gate; 80 stores
  778 after its gate; 100 stores 538, 558.
- Outputs: 10 gives 1, 30 gives 2, 70 gives 3 when slot 777 is zero and 778
  nonzero, 80 gives 3 when 778 is zero and 777 nonzero, 110 gives 5 when 779
  is nonzero and 4 otherwise. The gates read the bank before the case stores.
- `secondContinues` admits every ordinary ID 1..127; slot 896+127 is the last
  bank slot. Bank775 restoration still refuses.
- The auxiliary array is `secondAux`, eight entries of five DWORD fields.
  Stage cases store field +4 (entries 0..3 for stages 30 and 40, 0..7 for
  50..110) and stages 90 and 100 field +0 of entries 0, 1 and 3. ID 50 stores
  fields +0, +8 and +0xC of entries 4..7. Nothing reads the array (DIV-2437);
  field +0x10 immediates are unpublished (DIV-2630). The current SAV carries
  it as `Aux`, exactly 8x5, absent when zero.
- The movie is chosen at acknowledgement: `frontTransitions.finishWon` sets
  the live viewer's completion movie to the cutpaths row the output selects;
  output -1 plays nothing (DIV-2628). Mission entry selects no movie.
- Every town has a square (TAVERN, GATES) and an inn.
  `secondCampaign.innOptions` (`pkg/game/secondinn.go`) is EnterInn: the
  stage body selected by slot 768 (stage 10: NPC 207 topic 9 and NPC 2108
  topic 8 of kind 0, NPC 517 topic 10 of kind 3; stage 30: NPC 22 topic 30
  of kind 3, NPC 2108 topic 31 of kind 3 when slot 927 is zero, NPC 2110
  topic 39 of kind 0), then the fifteen continuation stores under their
  exact signed compares and current-ID gates, then the dynamic tail. It
  changes no state. Other stage bodies offer nothing (DIV-2634).
- The inn shows one "TALK <NPC>" row per distinct kind 0 or 3 NPC key, then
  GATES (DIV-2635). A row shows the npc%dtalk%d section of the town text and
  applies `talkTo`: kind 3 admits mission topic, topic 10 also stores slot
  769, NPC 22 topic 30 also stores 533=1 and 553=2; kind 0 stores nothing
  (DIV-2639). Kinds 1 and 2 are computed and not presented (DIV-2636).
  Admission and allocation are not modelled (DIV-2637, DIV-2638).
- GATES clears current. Town 1 also removes its node and opens only once
  mission 10 is available; a later town keeps its node. Quiet towns 1..3,
  square or inn, are save points (DIV-2631).
- A mission SAV is written for every ordinary ID 1..127. Missions 10 and 20
  keep their exact availability shapes; every other mission and towns 2 and
  3 take bounded availability: at most 130 unique entries, missions 1..127,
  towns 1..3, the current mission present, bank775 zero (DIV-2633).
- The census reads the same controller. Reach visits each available town,
  talks to every speaker, leaves, then enters each available mission. A movie
  exit counts as produced when `completeBank` stores its output with every
  gate held and with no gate inverted.
- Mission 50's bank780-gated fixed record has no type or ID and is not
  appended (DIV-2629).

## Divergences

DIV-2628 (movie output), DIV-2629 (mission 50 record), DIV-2630 (unpublished
auxiliary immediates), DIV-2631 (town 3), DIV-2632 (the stage-30 inn route,
Medium as a composed route), DIV-2633 (every ordinary mission and its SAV),
DIV-2634 (other stage bodies), DIV-2635 (talk rows and dialogue), DIV-2636
(kinds 1 and 2), DIV-2637 (EnterInn admission), DIV-2638 (allocation) and
DIV-2639 (kind 0 effects), all in `docs/divergences/rom2.md`. DIV-2437
records the stored auxiliary array. DIV-2356, DIV-2392 and DIV-2436 point to
the new rows.

## Proof

Focused tests (`pkg/game`):

- `TestSecondDepartureCaseBodies`: IDs 10, 20, 30, 31, 40, 50, 60, 70, 80, 90,
  100, 110 and the join-only IDs 25, 41, 120, 127, each under all 32
  combinations of slots 772, 777, 778, 779, 780 and three incoming stages;
  bank, auxiliary array, availability order and output against tables written
  from the claims.
- `TestSecondDepartureJoinAndStageSwitch`: the join for divisible and other
  IDs, and the stage switch on a moved and an unmoved slot 768.
- `TestSecondDepartureMovieExitsFollowTheirGates`: each movie exit with its
  gates held and each gate inverted.
- `TestSecondDepartureMovieAtAcknowledgement`: the production acknowledgement
  route requests the selected family or none.
- `TestCurrentSecondAuxiliaryArrayPersists`, `TestSecondCampaignThirdTown`,
  `TestCurrentSecondOrdinaryMissionsKeepTheirCampaign`,
  `TestCurrentSecondLaterMissionKeepsEveryAvailableRecordInOrder`,
  `TestCurrentSecondLaterMissionRefusesMalformedAvailabilityAtomically`,
  `TestSecondCompletionUsesTheSelectedOutputTable`.
- `TestSecondInnStageOptions`, `TestSecondInnContinuationGates`,
  `TestSecondInnDynamicTail`, `TestSecondInnTalkEffects`: every stage, slot
  927, the signed stage and slot compares, the 776/781 pick for each family,
  topic 62, the tail's kind and high bit, and each TalkTo effect against
  tables written from the claims.
- `TestSecondInnScreenTalksToEachSpeaker`: rows, dialogue, a missing
  section, a kind 0 talk, a save point in the inn and GATES through the
  screen.

Installed witnesses, EN and RU ROM2 roots (`secondlaterroute_release_test.go`).
`secondStageThirtyVisit` plays a new game: TALK 517, mission 10 won by a
pointer move, mission 20 won with disclosed `HeadlessPlace` steps, town 2
entered at stage 30, TAVERN, then TALK 22 (16 dialogue pages on EN; admits
30), TALK 2108 (8 pages; admits 31) and TALK 2110 (1 page; no change), then
GATES.

- `TestReleaseSecondStageThirtyInnOpensMissionThirty`: mission 30 opens from
  that campaign with slots 533=1, 553=2 and stage 30.
- `TestReleaseSecondLaterDepartureReachesTheNextMap`: from that campaign
  mission 31 is won by its own triggers after a disclosed
  `HeadlessKillPlayer(4)` and a `HeadlessPlace` beside the unit its
  register-74 trigger measures; the acknowledgement adds mission 32 beside
  the still-available 30, and mission 32 opens with the carried party and the
  departed bank.
- `TestReleaseSecondLaterMissionSaveContinuation`: mission 32, reached by that
  route, writes a named SAV carrying the stage-30 auxiliary stores; a fresh
  front end cold-loads it and 24 ticks after the same pointer command equal
  the uninterrupted run.
- `TestReleaseSecondMovieExitAtLaterDeparture` starts from a constructed
  stage-100 position, since no published entry reaches mission 110. The
  mission is won by its own trigger after a disclosed `HeadlessKill` of the
  unit it tests; slot 779 is set through `SetROM2ScenarioState`. Output 4
  (779 zero) and output 5 (779 nonzero) each play the installed cutpaths row.

## Census

`TestReleaseSecondGameSupportCensus` and `cmd/campaigncensus` on both ROM2
roots (46 maps), against the story1369 baseline. EN and RU agree.

| Total | Baseline | Departure step | Inn step |
|---|---|---|---|
| engine_win | 3 | 46 | 46 |
| engine_exits (of 15) | 4 | 14 | 14 |
| maps blocked on continuation | 43 | 1 (mission 50, DIV-2629) | 1 |
| maps blocked on movie | 4 | 0 | 0 |
| maps blocked on save | 43 | 0 | 0 |
| maps blocked on entry | 43 | 43 | 40 |
| engine_entry (reach) | 3 | 3 | 6: 10, 20, 21, 30, 31, 32 |
| ready | 3 | 3 | 5: 10, 20, 21, 31, 32 |

Mission 30 is reached and keeps its census party blocker. The save count is
the census predicate (`secondSaveMission`); the installed SAV witness covers
missions 10, 20, 21 and 32.

## Open debt

- Entry to every map after 32 waits on unpublished stage bodies and
  continuation writers (DIV-2634); the composed route is Medium (DIV-2632).
- Inn rendering and dialogue choice, kinds 1 and 2, admission, allocation
  and kind 0 effects (DIV-2635..DIV-2639).
- Mission 50's fixed record (DIV-2629); auxiliary consumers (DIV-2437) and
  field +0x10 immediates (DIV-2630).
- Catalog membership is not checked before an add (DIV-2633).
