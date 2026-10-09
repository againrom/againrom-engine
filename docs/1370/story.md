# ROM2 ordinary departure for every mission

## Intent

Before this story the ROM2 campaign controller continued only missions 10,
20 and 21, played only the mission 10 movie, and wrote a mission SAV only on
those three maps. The story1368 census counted 43 maps blocked on
continuation, 4 on a movie and 43 on save. This story runs the whole
published ordinary departure for every won mission, produces each movie exit
behind its gates, and writes a mission SAV on every ordinary mission. ROM1
code paths, release tests and hashed state do not change.

Maps 31 and 32 are not made reachable. No published departure adds mission 30
or 31; ID 31 adds 32. The record that offers 31 is most likely a stage-30 inn
or town 2 TALK entry, which no claim names (DIV-2632).

Base: `71847832` (game 0.97.0). Knowledge pin: k204.

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
- Town 3 and every later town show GATES only with an unavailable-services
  line; GATES clears current and keeps the node. Quiet town 3 is a save point
  (DIV-2631).
- A mission SAV is written for every ordinary ID 1..127. Missions 10, 20 and
  21 keep their exact availability shapes; every other mission takes bounded
  availability: at most 130 unique entries, missions 1..127, towns 1..3, the
  current mission present, bank775 zero (DIV-2633).
- The census reads the same controller: a movie exit counts as produced when
  `completeBank` stores its output with every gate held and with no gate
  inverted.
- Mission 50's bank780-gated fixed record has no type or ID and is not
  appended (DIV-2629).

## Divergences

DIV-2628 (movie output), DIV-2629 (mission 50 record), DIV-2630 (unpublished
auxiliary immediates), DIV-2631 (town 3), DIV-2632 (stage-30 inn entries and
reach), DIV-2633 (every ordinary mission and its SAV), all in
`docs/divergences/rom2.md`. DIV-2437 now records the stored auxiliary array.
DIV-2356, DIV-2392 and DIV-2436 point to the new rows.

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

Installed witnesses, EN and RU ROM2 roots (`secondlaterroute_release_test.go`).
Each starts from a constructed stage-30 or stage-100 campaign position, since
no published entry reaches those maps:

- `TestReleaseSecondLaterDepartureReachesTheNextMap`: mission 31 is won by
  its own triggers after a disclosed `HeadlessKillPlayer(4)` and a
  `HeadlessPlace` beside the unit its register-74 trigger measures; the
  acknowledgement adds mission 32, and mission 32 opens with the carried party
  and the departed bank.
- `TestReleaseSecondMovieExitAtLaterDeparture`: mission 110 is won by its own
  trigger after a disclosed `HeadlessKill` of the unit it tests; slot 779 is
  set through `SetROM2ScenarioState`. Output 4 (779 zero) and output 5 (779
  nonzero) each play the installed cutpaths row.
- `TestReleaseSecondLaterMissionSaveContinuation`: mission 32, reached by that
  route, writes a named SAV carrying the stage-30 auxiliary stores; a fresh
  front end cold-loads it and 24 ticks after the same pointer command equal
  the uninterrupted run.

## Census

`TestReleaseSecondGameSupportCensus` on both ROM2 roots, against the
story1369 baseline:

| Total | Before | After |
|---|---|---|
| engine_win | 3 | 46 |
| engine_exits (of 15) | 4 | 14 |
| maps blocked on continuation | 43 | 1 (mission 50, DIV-2629) |
| maps blocked on movie | 4 | 0 |
| maps blocked on save | 43 | 0 |
| maps blocked on entry | 43 | 43 |
| engine_entry, ready | 3, 3 | 3, 3 |

The save count is the census predicate (`secondSaveMission`); the installed
SAV witness covers missions 10, 20, 21 and 32.

## Open debt

- Entry to mission 30 and every later map waits on the stage-30 inn and town
  2 TALK entries (DIV-2632).
- Mission 50's fixed record (DIV-2629); auxiliary consumers (DIV-2437) and
  field +0x10 immediates (DIV-2630).
- Catalog membership is not checked before an add (DIV-2633).
