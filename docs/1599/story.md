# One win finisher and one session mission identity

## Intent

What follows a won mission runs through one finisher over rows in the
second game's departure shape, for both games and a mod guest chain. Every
caller that asks which mission the session plays, or where its return
starts, reads one session mission identity, and restoring a save builds that
identity. No player change: the next mission, the town, the world-map
return, the payment, the joins, the guest stash and its return stay as they
were, and every SAV keeps its bytes, the guest record included.

## Authority

Owner direction: one builder per kind, and the multi-way allowance falls
from 50 to 20 (audit-20 plan, ranks R10 and R42, wave 6). Owner ruling: the
first game is a reduced second game, one builder per kind in the second
game's shape with the first game as profile data; architecture work counts 0
toward the release. No claim changes; no DIV row is added.

## As-built behaviour

- `finishWin` (`pkg/game/winfinish.go`) is the one finisher. `winRowSets`
  declares two row sets, tried in order. A set has a guard, rows
  `{guard, writes, exits}`, a route and a return to town. The first row
  whose guard holds runs. Its writes run in order, and a write that refuses
  ends the win at -1. Its first exit whose gate holds answers the next place.
- The bank set belongs to `base.CampaignDestinations`, the second game's
  profile. It holds only for a win through the mission seam on a town the
  bank stands for. Its guard calls the campaign service's `bankWin`: the
  first game's answers false, the second game's tests the seam and the
  town's bank, so `winRows` takes the first set whose guard holds. Writes: `bankFinish`, `carryWin`, `bankComplete`. Exit:
  `enteredWithoutMap`. Route: below zero stay, above zero the mission, zero
  the town. Return: the live map is ended.
- The chapter set holds for every other win: the first game, a second-game
  town with no bank, and the public `FinishMission` and
  `FinishMissionWithRoster`, which enter with no seam as before.
  - The guest-link row is guarded by the identity's `guest`. Writes:
    `carryWin`, `guestJoiner`. Exits: `guestFollows` (the chain's `AfterWin`)
    to `guestNext`, else `guestHome`, the stash restore.
  - The chapter row exits to `finishMission`. That reads the campaign
    registry for the successor, the payment and the AutoGetMission travel, and
    starts a guest chain at the town arrival.
  - Route: `routeAfterWin`. Return: `beginWorldMapReturn` from the identity's
    `back`.
- `frontTransitions.finishWon` is the one `finishWon`. The campaign service
  no longer declares `finishWon`, `route` or `returnToTown`. The second
  game's `beforeAdvance` is unchanged.
- `missionIdentity` (`pkg/game/sessionmission.go`) is
  `{played, record, guest, own, replaced, back}`, built only by
  `newMissionIdentity(n, guest record, mod missions)`. Its readers are:
  - `activateLive`, the mission entry's selection, `leaveMission` and
    `MissionOpener` (through `entry`);
  - the campaign record overlay and the player colours (through `modMap`);
  - the chapter return to town, and the guest-link row.
- `restoreMissionIdentity` is the restore of a save's identity. It reads the
  guest record (`restoreGuest`, unchanged), then refuses a mod's own mission
  that has no guest record and a replaced map that has no mod mark, in the
  order and with the words the restore used before. `originalrestore.go`
  calls it and resolves nothing itself. `refuseUnmarkedReplaced` reads the
  same identity.
- The accessors of the guest record are `guestMission`, `snapshotGuest` and
  `takeGuestJoining`. They live in the identity file with `playing`,
  `returnMission` and `guestEntry`. The guest record type, `currentGuest`,
  `record` and `restoreGuest` keep their fields and bytes.
- `internal/archtest` `CheckWinRoute` reports:
  - a call of `guestMission()`, or a one-argument call of `playing`,
    `returnMission`, `guestEntry`, `ByNumber` or `Replacing`, in a production
    file of `pkg/game` other than `sessionmission.go`;
  - a `finishWon` on any receiver but `frontTransitions`, or a count of
    `finishWon` other than one;
  - a `finishWin` outside `winfinish.go`, or a count of `finishWin` other
    than one;
  - a win step declared by `campaignService`.

  The guard has no named entries: no part of either kind stayed outside
  them. No import was added, so no `dag` row was added.

## Proof

- `TestWinRouteRecordIsTheBaseRecord` (`pkg/game/winroute_record_test.go`)
  was committed alone before the change and holds after it. It uses
  synthetic fixtures with no install. For each win it records the next
  place, the line, the payment, the party, the joins, the guest record, the
  stash, the world-map return, the SAV saved right after the win, and a
  restore of that SAV. It covers:
  - the first game's wins 10, 20, 30 and 40, and 30 and 40 again after a
    load;
  - a two-link mod chain, 25 then 27;
  - the second game's bank wins 10, 20 and 40, and the direct entry.

  A planted early stash return fails it on 36 lines.
- Focused tests: `./pkg/game -run
  'Guest|ModMission|Campaign|Win|Successor|Transition|Restore|Resume|MissionEntry|MissionLeave|SecondCampaign|SecondBag|SetModMissions|UnmarkedSave|ContinueMission|CampaignRecord|PlayerColour|ModChain|ModHero'
  -skip '^TestRelease'` (273 pass, 1 skip), `./internal/archtest -run
  'WinRoute|^TestLiveTreeClean$|CheckNamesOffendingEdge|Composition'`,
  `./internal/storyguard -run '^TestLiveTreeClean$'`.
- `TestCheckWinRoute` plants one instance per shape.

## Open debt

- No release test ran. The next release runs, on both games, EN and RU, the
  release tests named in the lane result that win a mission, play a mod
  chain, or save and restore a guest or a campaign position.
- `recordWin` keeps its branches over the campaign registry inside the
  chapter row's exit. Splitting them into exits is a later step.
