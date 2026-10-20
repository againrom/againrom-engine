# Player Groups and script Group selectors after a LOAD

## Intent and authority

Owner direction: after a LOAD a script selector reaches one Group by the
original's order rule, with no ambiguous outcome; a SAVE writes the Groups the
original's writer writes; a field without a source stays named debt.

| Claim | Confidence | Used for |
|---|---|---|
| `SAV-1337` | High / Medium | LOAD appends Players, then each Player's Groups, in file order |
| `AI-366` | High | the binder's id-only map keeps the last insert |
| `SAV-1338` | Medium | after LOAD the last Group carrying an id wins, Players then Groups in file order |
| `SAV-1340` | High / Medium | 164 scripted ids in 25 original mission saves bind as the map start binds; 38 to an empty Group |
| `TRIG-151` | High | 169 scripted (map, id) pairs; none authored for owner 1; 4 ids have two owners |
| `SAV-1251` | High predicate / Medium population | SAVE removes the empty Groups of a `+28`-zero Player from the live list before the file is built |
| `SAV-1234` | High | the serializers write every Group still listed, an empty one empty |
| `SAV-1341` | High | `+28`-zero Groups are never empty in the corpus; `+28`-one Players hold 51 empty Groups of 405 |
| `PARTY-047` | High / Medium | the human Player is the first Player, `+28` zero; 0 of 1,004 members carry another Player's key |
| `SAV-1339` | High / Medium | the creator of 43 of 45 unauthored human `+1c` values is unknown |

## As built

### Selector rule

`World.resolveSavedGroups` (`pkg/sim/savedgroupcommands.go`) answers the last
Group carrying the selector in the Player-then-Group walk (`savedWalkRank`):
the same Player's later Group or a later Player's Group. No carrier set is
ambiguous. A selector no Group carries still names no Group, and a carrier
with an unmaterialized member still refuses. This is the one resolver; counts,
commands and the trigger block all call it. DIV-3789 (FIDELITY-DEBT, Medium).

### SAVE pruning

- `sim.SaveKeepsGroup(participant, members)` is the one rule: a Group is
  written unless it is empty and its Player's participant word `+28` is zero.
- The prune applies only to Group lists a LOAD set. Owner direction: a run
  that never LOADs keeps every World hash and SAV digest, so the lists a map
  start or a new campaign built are written as before, emptied Groups
  included.
- Mission: `FrontEnd.playerSnapshot` (game menu, save dialog, quicksave) calls
  `World.PruneGroupsForSave` before the World half and the document are built.
  The save dialog builds its file from the pruned lists but prunes the live
  World only when the file is written: `Prepare` prunes, captures, and puts
  the lists and source bindings back (`World.PruneGroupsForSaveUndo`, the same
  prune), and the prepared `Commit` prunes again once the file is written. A
  cancelled overwrite, or any dialog path that writes nothing, leaves the
  World and its hash unchanged (review finding F1).
  A registry constructed at map start (`fresh`) is left alone. The participant
  word comes from the World's Player carrier; without one the
  human-participant slot answers zero and other slots one. An actor whose
  retained source Group is dropped takes the Group that holds it, so LOAD of
  the written file builds the World the SAVE left. The autosaves keep the plain
  `Snapshot` and change nothing: a timer must not change hashed state.
  DIV-3790 (FIDELITY-DEBT).
- Town: a LOAD marks the town's groups (`Town.cityGroupsLoaded`). A town SAVE
  the player makes then removes each group with no party member from the live
  list (`Town.pruneCityGroupsForSave`), and `writeCityGroups` applies the same
  rule to what it writes. The town's Player has `+28` zero.

### Town Groups of an engine-created campaign

The engine's town Player carries `+28` zero (`nativeCityPlayerState`).

- Never LOADed: unchanged from base. A group whose members all left (death,
  script removal, mission loss) stays listed and is written empty; a later
  hire of that type opens a new group behind it, so empty groups accumulate
  across missions until the mission-start head sweep (AI-479) reaches them.
- After a LOAD of its town file: the next town SAVE drops each emptied group
  from the live list and the file, so a LOAD resumes the town the SAVE left.
  `TestReleaseCityGroupsRepeatedHireCyclesF2SAV` now expects a lost squad's
  group to be gone at the next cycle's SAVE.
- The constructed group of a town with no readable group record is unchanged:
  80 zero AI bytes and `+1c` 0, named debt under SAV-1339 (DIV-1686).

### Party actor in another Player's Group, unauthored `+1c`

No code change. PARTY-047 narrows DIV-1438: no original-authored save holds
such an actor. The `+1c` of a Group made during play has no established
creator (SAV-1339): DIV-1685 and DIV-1686 name it.

## Proof

### Focused tests, each with a loss control

- `TestSavedSelectorOneOwnerLaterGroupWins`: one Player's two Groups carry
  selector 5; the count and a command reach the later Group, also after the
  byte form. Control: the selector on the earlier Group only. Fails on base.
- `TestSavedSelectorLaterPlayerWins`: two owners, slots chosen so a
  slot-keyed rule picks the other Group; unchanged from base. Control as above.
- `TestSavedGroupsLaterCarrierAndUnboundRefuseAtomically`,
  `TestSavedScriptSelectorResolvesToLaterOwner`: the same-owner cases the old
  resolver refused now reach the later Group.
- `TestPruneGroupsForSaveFollowsTheParticipantWord`: human first, human second,
  no human, carrier absent; a second prune changes nothing; the byte form
  round-trips. `TestPruneGroupsForSaveNativeDialect`: no registry and a
  map-start registry are not pruned; control: the same lists once LOADed are.
- `TestSaveDialogPrunesOnlyWhenTheFileIsWritten` (`pkg/game`): after a
  LOAD with empty human Groups, a dialog SAVE cancelled at the overwrite
  confirmation, and a `Prepare` alone, leave the Group lists and `World.Hash`
  unchanged and the old file intact; the confirmed one prunes, writes, and the
  World equals the pruned World and the LOAD of the file. Loss control: with
  the prune left in `Prepare` the test fails on the cancelled overwrite.
- `TestPruneGroupsForSaveRebindsTheSourceGroup`: control keeps the source.
- `TestMissionSaveDropsOnlyParticipantZeroEmptyGroups` (`pkg/game`): words
  0/1 drop the left Player's empty Groups 1 and 2 and keep the right Player's
  empty Group 4; control 1/0 keeps 1 and 2 and drops 4.
- `TestCitySaveDropsAnEmptiedGroupOfTheHumanPlayer`: groups a LOAD read,
  word 0, omit the emptied group; controls: word 1, and groups no LOAD read,
  write it empty. `TestCitySaveRemovesAnEmptiedGroupFromALoadedTown`: the
  live list loses it; control: an unloaded town keeps it.

Existing `pkg/game` tests, release tests among them, that LOAD a file and then
assert "a SAVE leaves the live World unchanged" take their pre-SAVE capture
after `PruneGroupsForSave`; `TestDocument1115Composed…` takes the World hash
after the menu SAVE. `TestReleaseSavedFormation1159AppContinuation` reads the
second session's Groups after its SAVE.

### M2 groups census

`scripts/check-milestone2-acceptance.sh` on EN and RU, save corpus
`gameversions/saves` (83 mission files resumed, 0 import refusals on either
root).

| File | Issues before | After |
|---|---|---|
| `2026-08-02/game0001.sav` | 4 | 0 |
| `2026-08-02/game0007.sav` | 3 | 0 |
| `2026-08-14/game0014.sav` | 2 | 0 |
| `2026-08-15/game0018.sav` | 2 | 0 |
| `2026-08-30/EXP-0278-human-runtime-en/game0001.sav` | 4 | 0 |
| `2026-08-30/EXP-0278-human-runtime-en/game0007.sav` | 3 | 0 |
| `2026-08-30/EXP-0278-human-runtime-en/game0014.sav` | 2 | 0 |
| `2026-08-30/EXP-0278-human-runtime-en/game0018.sav` | 2 | 0 |
| `2026-09-09/game0076.sav` | 2 | 0 |
| `2026-10-02/projectiles-original-en/game0022.sav` | 15 | 0 |
| `2026-10-02/projectiles-original-en/game0023.sav` | 15 | 0 |
| `2026-10-02/projectiles-original-en/game0024.sav` | 15 | 0 |
| `2026-10-06/game0007-original-m70-boltcoming.sav` | 4 | 0 |
| `2026-10-06/game0008-original-m70-dying.sav` | 4 | 0 |
| `2026-10-07/original-game0011-after-beforekargallas.sav` | 2 | 0 |

The counts are equal on EN and RU. All 79 base issues were "saved Group
selector N is ambiguous": 73 for selector 0 and 6 for selector 1667853411.
Neither is a scripted id (TRIG-151), so no script in these files read a
different Group.

### Hashes and digests against c1f8dfbf

- Mission scenarios (`pipeline/check-mission-scenarios.sh`, 38 scenarios per
  root): all 652 step hashes are equal to base on EN and RU. No mission
  scenario writes a save.
- A run that never LOADs is unchanged: its Group lists are the map start's or
  the new campaign's, which no SAVE prunes.
- After a LOAD, a player SAVE whose human Player holds an empty Group writes
  the file without it, and the World after that SAVE lacks it; its LOAD equals
  it. In the M2 writer census the engine-written
  `2026-10-07/saveorcsdontgo.sav` is such a file on both roots.
- Release set (`pipeline/check-release-tests.sh`, 1,329 gated tests, EN and
  RU): the release tests that LOAD a file, play until the human Player holds
  an empty Group, and compare a pre-SAVE capture moved; they take the capture
  after the prune. `TestReleaseOriginalControlResaveUnchanged` reports the
  resave digest `c8e96009…` against a pinned `7d1fde99…` on both roots; base
  `c1f8dfbf` gives the same `c8e96009…`, and no Group is pruned in that run.
- Second game: `constructGroups` builds no registry for it, so the mission
  prune and the selector rule do not reach its World. Its SAVE selection
  passes on `rom2-en` and `rom2-ru`.

## Open debt

- DIV-3789: the rank of a Group no listed Player holds; the lifetime of the
  pointer a script node holds after the Group is removed.
- DIV-3790: the population SAVE leaves is Medium; a run that never LOADed and
  the autosaves write the human Player's emptied Groups.
- DIV-1685: a campaign that never LOADed keeps and writes emptied town groups.
- DIV-1686, DIV-1685: the `+1c` of a Group made in play (SAV-1339).
- DIV-1438: a party actor in another Player's Group has no projection.
